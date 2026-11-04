// Package torrents serves the native torrent API (/api/torrents, /api/stream,
// /api/engine/settings; docs/en/31-rest-api.md §1.8–1.10) on top of the engine in
// internal/torrserver.
package torrents

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/gin-gonic/gin"

	tslog "plenka/internal/torrserver/log"
	sets "plenka/internal/torrserver/settings"
	"plenka/internal/torrserver/torr"
	"plenka/internal/torrserver/torr/state"
	tsutils "plenka/internal/torrserver/utils"
)

// infoTimeout is how long a stream or playlist request waits for torrent metadata.
const infoTimeout = 60 * time.Second

// Engine settings are applied through these, so tests can stub the engine restart.
var (
	applySettings = torr.SetSettings
	applyDefaults = torr.SetDefSettings
)

// API is the torrents module.
type API struct{}

// New returns the module.
func New() *API { return &API{} }

// RegisterRoutes mounts the routes.
func (a *API) RegisterRoutes(r gin.IRouter) {
	r.GET("/api/torrents", listTorrents)
	r.POST("/api/torrents", addTorrent)
	r.POST("/api/torrents/upload", uploadTorrent)
	r.GET("/api/torrents/:hash", getTorrent)
	r.DELETE("/api/torrents/:hash", removeTorrent)
	r.POST("/api/torrents/:hash/drop", dropTorrent)
	r.GET("/api/torrents/:hash/cache", getCache)
	r.POST("/api/torrents/:hash/preload", preloadTorrent)
	r.POST("/api/torrents/:hash/download", downloadTorrent)
	r.DELETE("/api/torrents/:hash/download", cancelDownload)
	r.GET("/api/torrents/:hash/playlist.m3u", playlist)

	for _, p := range []string{"/api/stream/:hash/:id", "/api/stream/:hash/:id/*name"} {
		r.GET(p, stream)
		r.HEAD(p, stream)
	}

	r.GET("/api/engine/settings", getSettings)
	r.PUT("/api/engine/settings", putSettings)
	r.DELETE("/api/engine/settings", resetSettings)
}

func fail(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(code, gin.H{"error": msg})
}

// find returns the torrent named by the :hash parameter or answers 404.
func find(c *gin.Context) *torr.Torrent {
	tor := torr.FindTorrent(c.Param("hash"))
	if tor == nil {
		fail(c, http.StatusNotFound, "torrent not found")
	}
	return tor
}

func listTorrents(c *gin.Context) {
	out := []Torrent{}
	for _, t := range torr.ListTorrent() {
		out = append(out, toTorrent(t))
	}
	c.JSON(http.StatusOK, out)
}

func getTorrent(c *gin.Context) {
	if tor := find(c); tor != nil {
		c.JSON(http.StatusOK, toTorrent(tor))
	}
}

type addRequest struct {
	Link   string `json:"link"`
	Title  string `json:"title"`
	Poster string `json:"poster"`
	Data   string `json:"data"`
}

func addTorrent(c *gin.Context) {
	var req addRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "bad request: "+err.Error())
		return
	}
	spec, err := parseLink(req.Link)
	if err != nil {
		fail(c, http.StatusBadRequest, "bad link: "+err.Error())
		return
	}
	tslog.TLogln("add torrent", spec.InfoHash.HexString())
	add(c, spec, req.Title, req.Poster, req.Data)
}

func uploadTorrent(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "file is required")
		return
	}
	f, err := fh.Open()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	spec, err := fromTorrentFile(f)
	f.Close()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	tslog.TLogln("add .torrent", fh.Filename)
	add(c, spec, c.PostForm("title"), c.PostForm("poster"), c.PostForm("data"))
}

// add loads the torrent, applies the given title/poster/data (also to an existing
// torrent), answers with its state and saves it; metadata arrives in the background.
func add(c *gin.Context, spec *torrent.TorrentSpec, title, poster, data string) {
	tor, err := torr.AddTorrent(spec, title, poster, data)
	if err != nil {
		fail(c, http.StatusInternalServerError, "add torrent: "+err.Error())
		return
	}
	if title != "" {
		tor.Title = title
	}
	if poster != "" {
		tor.Poster = poster
	}
	if data != "" {
		tor.Data = data
	}
	if tor.Title == "" {
		tor.Title = spec.DisplayName
	}
	c.JSON(http.StatusOK, toTorrent(tor))

	go func() {
		torr.SaveTorrentToDB(tor)
		if !tor.GotInfo() {
			tslog.TLogln("add torrent: no metadata for", tor.Hash().HexString())
			return
		}
		if tor.Title == "" {
			tor.Title = tor.Name()
		}
		// Saved again with the metadata, so it loads without peers next time.
		torr.SaveTorrentToDB(tor)
	}()
}

func removeTorrent(c *gin.Context) {
	if tor := find(c); tor != nil {
		torr.RemTorrent(tor.Hash().HexString())
		c.Status(http.StatusNoContent)
	}
}

func dropTorrent(c *gin.Context) {
	if tor := find(c); tor != nil {
		torr.DropTorrent(tor.Hash().HexString())
		c.Status(http.StatusNoContent)
	}
}

func getCache(c *gin.Context) {
	if tor := find(c); tor != nil {
		c.JSON(http.StatusOK, toCache(tor.Hash().HexString(), tor.CacheState()))
	}
}

type preloadRequest struct {
	FileID int `json:"fileId"`
}

// preloadTorrent downloads to disk, in the background, the start (preloadCache % of
// the length) and the end of every file, the asked one first, so playback starts
// without buffering. It waits for the metadata and refuses with 507 when that does not
// fit on disk.
func preloadTorrent(c *gin.Context) {
	tor := find(c)
	if tor == nil {
		return
	}
	var req preloadRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.FileID < 1 {
		fail(c, http.StatusBadRequest, "fileId is required")
		return
	}
	if known := toTorrent(tor).Files; len(known) > 0 {
		if _, ok := playlistFiles(known, req.FileID); !ok {
			fail(c, http.StatusNotFound, "file not found")
			return
		}
	}
	loaded, err := load(c.Request.Context(), tor)
	if err != nil {
		loadFailed(c, err)
		return
	}
	if !roomFor(c, loaded, loaded.PreloadNeed(req.FileID)) {
		return
	}
	go torr.Preload(loaded, req.FileID)
	c.Status(http.StatusAccepted)
}

// roomFor answers 507 and returns false when need bytes, plus what other preloads and
// full downloads still have to fetch, plus a reserve, do not fit on disk.
func roomFor(c *gin.Context, tor *torr.Torrent, need int64) bool {
	if need <= 0 {
		return true
	}
	need += pendingKeeps(tor.Hash().HexString()) + keepReserve
	if free, ok := freeSpace(); ok && need > free {
		fail(c, http.StatusInsufficientStorage, fmt.Sprintf("not enough disk space: needs %s, %s free",
			tsutils.Format(float64(need)), tsutils.Format(float64(free))))
		return false
	}
	return true
}

// downloadTorrent downloads the whole torrent to disk and keeps it until cancelled or
// removed. It waits for the metadata and refuses with 507 when it does not fit.
func downloadTorrent(c *gin.Context) {
	tor := find(c)
	if tor == nil {
		return
	}
	loaded, err := load(c.Request.Context(), tor)
	if err != nil {
		loadFailed(c, err)
		return
	}
	if !loaded.FullDownload() {
		if !roomFor(c, loaded, loaded.DownloadNeed()) {
			return
		}
		if !loaded.DownloadAll() {
			loadFailed(c, errNoInfo)
			return
		}
	}
	c.JSON(http.StatusOK, toTorrent(loaded))
}

// cancelDownload stops a full download and deletes what it fetched, except the
// preloaded parts.
func cancelDownload(c *gin.Context) {
	if tor := find(c); tor != nil {
		tor.CancelDownloadAll()
		c.Status(http.StatusNoContent)
	}
}

var errNoInfo = errors.New("no torrent metadata")

// load makes sure tor is loaded in the engine and has its metadata, waiting at most
// infoTimeout. It returns the loaded torrent.
func load(ctx context.Context, tor *torr.Torrent) (*torr.Torrent, error) {
	if tor.Stat == state.TorrentInDB {
		if tor.TorrentSpec == nil {
			return nil, errNoInfo
		}
		loaded, err := torr.AddTorrent(tor.TorrentSpec, tor.Title, tor.Poster, tor.Data)
		if err != nil {
			return nil, err
		}
		tor = loaded
	}
	done := make(chan bool, 1)
	go func() { done <- tor.GotInfo() }()
	timer := time.NewTimer(infoTimeout)
	defer timer.Stop()
	select {
	case ok := <-done:
		if !ok {
			return nil, errNoInfo
		}
		return tor, nil
	case <-timer.C:
		return nil, errNoInfo
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func loadFailed(c *gin.Context, err error) {
	if errors.Is(err, errNoInfo) || errors.Is(err, context.Canceled) {
		fail(c, http.StatusGatewayTimeout, "torrent metadata did not arrive")
		return
	}
	fail(c, http.StatusInternalServerError, err.Error())
}

func stream(c *gin.Context) {
	tor := find(c)
	if tor == nil {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		fail(c, http.StatusNotFound, "file not found")
		return
	}
	// A saved torrent's metadata answers an unknown id without loading it.
	if known := toTorrent(tor).Files; len(known) > 0 {
		if _, ok := playlistFiles(known, id); !ok {
			fail(c, http.StatusNotFound, "file not found")
			return
		}
	}
	tor, err = load(c.Request.Context(), tor)
	if err != nil {
		loadFailed(c, err)
		return
	}
	// Stream answers 404 itself for an unknown file id.
	_ = tor.Stream(id, c.Request, c.Writer)
}

func playlist(c *gin.Context) {
	tor := find(c)
	if tor == nil {
		return
	}
	fileID := 0
	if v := c.Query("file"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil || id < 1 {
			fail(c, http.StatusNotFound, "file not found")
			return
		}
		fileID = id
	}
	t := toTorrent(tor)
	if len(t.Files) == 0 {
		loaded, err := load(c.Request.Context(), tor)
		if err != nil {
			loadFailed(c, err)
			return
		}
		t = toTorrent(loaded)
	}
	files, ok := playlistFiles(t.Files, fileID)
	if !ok {
		fail(c, http.StatusNotFound, "file not found")
		return
	}
	name := t.Title
	if strings.TrimSpace(name) == "" {
		name = t.Name
	}
	if strings.TrimSpace(name) == "" {
		name = t.Hash
	}
	c.Header("Content-Disposition", attachmentName(name))
	c.Data(http.StatusOK, "audio/x-mpegurl", []byte(buildM3U(baseURL(c.Request), t.Hash, files)))
}

func getSettings(c *gin.Context) {
	c.JSON(http.StatusOK, fromBTSets(sets.BTsets))
}

func putSettings(c *gin.Context) {
	// Fields missing from the body keep their current values.
	e := fromBTSets(sets.BTsets)
	if err := c.ShouldBindJSON(&e); err != nil {
		fail(c, http.StatusBadRequest, "bad settings: "+err.Error())
		return
	}
	applySettings(toBTSets(e, sets.BTsets))
	c.JSON(http.StatusOK, fromBTSets(sets.BTsets))
}

func resetSettings(c *gin.Context) {
	applyDefaults()
	c.JSON(http.StatusOK, fromBTSets(sets.BTsets))
}
