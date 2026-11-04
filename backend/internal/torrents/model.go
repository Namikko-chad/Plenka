package torrents

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"

	sets "plenka/internal/torrserver/settings"
	"plenka/internal/torrserver/torr"
	"plenka/internal/torrserver/torr/state"
	cachestate "plenka/internal/torrserver/torr/storage/state"
	tsutils "plenka/internal/torrserver/utils"
)

// Torrent is the API view of a torrent (docs/en/31-rest-api.md §1.8).
type Torrent struct {
	Hash             string `json:"hash"`
	Title            string `json:"title"`
	Poster           string `json:"poster"`
	Data             string `json:"data"`
	Name             string `json:"name"`
	Stat             int    `json:"stat"`
	StatString       string `json:"statString"`
	AddedAt          string `json:"addedAt"`
	TorrentSize      int64  `json:"torrentSize"`
	LoadedSize       int64  `json:"loadedSize"`
	PreloadedBytes   int64  `json:"preloadedBytes"`
	PreloadSize      int64  `json:"preloadSize"`
	FullDownload     bool   `json:"fullDownload"`
	DownloadSpeed    int64  `json:"downloadSpeed"`
	UploadSpeed      int64  `json:"uploadSpeed"`
	TotalPeers       int    `json:"totalPeers"`
	PendingPeers     int    `json:"pendingPeers"`
	ActivePeers      int    `json:"activePeers"`
	ConnectedSeeders int    `json:"connectedSeeders"`
	HalfOpenPeers    int    `json:"halfOpenPeers"`
	BytesRead        int64  `json:"bytesRead"`
	Files            []File `json:"files"`
}

// File is one file of a torrent. Ids start at 1, in path order.
type File struct {
	ID     int    `json:"id"`
	Path   string `json:"path"`
	Length int64  `json:"length"`
}

// Cache is the piece cache state of a torrent.
type Cache struct {
	Hash         string           `json:"hash"`
	Capacity     int64            `json:"capacity"`
	Filled       int64            `json:"filled"`
	PiecesLength int64            `json:"piecesLength"`
	PiecesCount  int              `json:"piecesCount"`
	Pieces       map[string]Piece `json:"pieces"`
	Readers      []Reader         `json:"readers"`
}

// Piece is a cached piece.
type Piece struct {
	ID        int   `json:"id"`
	Length    int64 `json:"length"`
	Size      int64 `json:"size"`
	Completed bool  `json:"completed"`
	Priority  int   `json:"priority"`
}

// Reader is an active reader: the piece range it keeps cached and its position.
type Reader struct {
	Start  int `json:"start"`
	End    int `json:"end"`
	Reader int `json:"reader"`
}

// toTorrent builds the API view of an engine torrent.
func toTorrent(t *torr.Torrent) Torrent {
	var infoBytes []byte
	if t.TorrentSpec != nil {
		infoBytes = t.TorrentSpec.InfoBytes
	}
	return torrentFromStatus(t.Status(), infoBytes)
}

// torrentFromStatus converts an engine status. A torrent that is not loaded has no
// file list in its status; it is then read from the saved metadata (infoBytes).
func torrentFromStatus(st *state.TorrentStatus, infoBytes []byte) Torrent {
	out := Torrent{
		Hash:             strings.ToLower(st.Hash),
		Title:            st.Title,
		Poster:           st.Poster,
		Data:             clientData(st.Data),
		Name:             st.Name,
		Stat:             int(st.Stat),
		StatString:       st.StatString,
		AddedAt:          time.Unix(st.Timestamp, 0).UTC().Format(time.RFC3339),
		TorrentSize:      st.TorrentSize,
		LoadedSize:       st.LoadedSize,
		PreloadedBytes:   st.PreloadedBytes,
		PreloadSize:      st.PreloadSize,
		FullDownload:     st.FullDownload,
		DownloadSpeed:    speed(st.DownloadSpeed),
		UploadSpeed:      speed(st.UploadSpeed),
		TotalPeers:       st.TotalPeers,
		PendingPeers:     st.PendingPeers,
		ActivePeers:      st.ActivePeers,
		ConnectedSeeders: st.ConnectedSeeders,
		HalfOpenPeers:    st.HalfOpenPeers,
		BytesRead:        st.BytesRead,
		Files:            []File{},
	}
	for _, f := range st.FileStats {
		out.Files = append(out.Files, File{ID: f.Id, Path: f.Path, Length: f.Length})
	}
	if len(out.Files) == 0 {
		// Without metadata the engine reports the magnet dn or "infohash:…" as the name.
		out.Name = ""
	}
	if len(out.Files) == 0 && len(infoBytes) > 0 {
		if name, files, size, ok := filesFromInfo(infoBytes); ok {
			out.Name = name
			out.Files = files
			if out.TorrentSize == 0 {
				out.TorrentSize = size
			}
		}
	}
	return out
}

// clientData drops the file list TorrServer used to store in "data" when the client
// gave none, so clients only ever see their own JSON there.
func clientData(data string) string {
	if strings.HasPrefix(data, `{"TorrServer":`) {
		return ""
	}
	return data
}

func speed(v float64) int64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return int64(math.Round(v))
}

// filesFromInfo lists the files of bencoded torrent info with the same paths, order
// and ids as the engine's status of a loaded torrent.
func filesFromInfo(infoBytes []byte) (name string, files []File, size int64, ok bool) {
	var info metainfo.Info
	if err := bencode.Unmarshal(infoBytes, &info); err != nil {
		return "", nil, 0, false
	}
	name = info.BestName()
	for _, f := range info.UpvertedFiles() {
		files = append(files, File{
			Path:   strings.Join(append([]string{name}, f.BestPath()...), "/"),
			Length: f.Length,
		})
		size += f.Length
	}
	sort.Slice(files, func(i, j int) bool {
		return tsutils.CompareStrings(files[i].Path, files[j].Path)
	})
	for i := range files {
		files[i].ID = i + 1
	}
	return name, files, size, true
}

// toCache converts the engine cache state; st is nil for a torrent that is not loaded.
func toCache(hash string, st *cachestate.CacheState) Cache {
	out := Cache{Hash: strings.ToLower(hash), Pieces: map[string]Piece{}, Readers: []Reader{}}
	if st == nil {
		if sets.BTsets != nil {
			out.Capacity = sets.BTsets.CacheSize
		}
		return out
	}
	out.Capacity = st.Capacity
	out.Filled = st.Filled
	out.PiecesLength = st.PiecesLength
	out.PiecesCount = st.PiecesCount
	for id, p := range st.Pieces {
		out.Pieces[strconv.Itoa(id)] = Piece{ID: p.Id, Length: p.Length, Size: p.Size, Completed: p.Completed, Priority: p.Priority}
	}
	for _, r := range st.Readers {
		if r != nil {
			out.Readers = append(out.Readers, Reader{Start: r.Start, End: r.End, Reader: r.Reader})
		}
	}
	return out
}

// EngineSettings is the API view of the engine settings (§1.10).
type EngineSettings struct {
	CacheSize                int64  `json:"cacheSize"`
	ReaderReadAhead          int    `json:"readerReadAhead"`
	PreloadCache             int    `json:"preloadCache"`
	UseDisk                  bool   `json:"useDisk"`
	TorrentsSavePath         string `json:"torrentsSavePath"`
	RemoveCacheOnDrop        bool   `json:"removeCacheOnDrop"`
	ForceEncrypt             bool   `json:"forceEncrypt"`
	RetrackersMode           int    `json:"retrackersMode"`
	TorrentDisconnectTimeout int    `json:"torrentDisconnectTimeout"`
	EnableDebug              bool   `json:"enableDebug"`
	EnableIPv6               bool   `json:"enableIPv6"`
	DisableTCP               bool   `json:"disableTCP"`
	DisableUTP               bool   `json:"disableUTP"`
	DisableUPNP              bool   `json:"disableUPNP"`
	DisableDHT               bool   `json:"disableDHT"`
	DisablePEX               bool   `json:"disablePEX"`
	DisableUpload            bool   `json:"disableUpload"`
	DownloadRateLimit        int    `json:"downloadRateLimit"`
	UploadRateLimit          int    `json:"uploadRateLimit"`
	ConnectionsLimit         int    `json:"connectionsLimit"`
	PeersListenPort          int    `json:"peersListenPort"`
	ResponsiveMode           bool   `json:"responsiveMode"`
}

func fromBTSets(s *sets.BTSets) EngineSettings {
	if s == nil {
		s = sets.DefaultBTSets()
	}
	return EngineSettings{
		CacheSize:                s.CacheSize,
		ReaderReadAhead:          s.ReaderReadAHead,
		PreloadCache:             s.PreloadCache,
		UseDisk:                  s.UseDisk,
		TorrentsSavePath:         s.TorrentsSavePath,
		RemoveCacheOnDrop:        s.RemoveCacheOnDrop,
		ForceEncrypt:             s.ForceEncrypt,
		RetrackersMode:           s.RetrackersMode,
		TorrentDisconnectTimeout: s.TorrentDisconnectTimeout,
		EnableDebug:              s.EnableDebug,
		EnableIPv6:               s.EnableIPv6,
		DisableTCP:               s.DisableTCP,
		DisableUTP:               s.DisableUTP,
		DisableUPNP:              s.DisableUPNP,
		DisableDHT:               s.DisableDHT,
		DisablePEX:               s.DisablePEX,
		DisableUpload:            s.DisableUpload,
		DownloadRateLimit:        s.DownloadRateLimit,
		UploadRateLimit:          s.UploadRateLimit,
		ConnectionsLimit:         s.ConnectionsLimit,
		PeersListenPort:          s.PeersListenPort,
		ResponsiveMode:           s.ResponsiveMode,
	}
}

// toBTSets applies e onto a copy of cur, keeping the engine fields the API does not
// expose (tracker lists, LPD).
func toBTSets(e EngineSettings, cur *sets.BTSets) *sets.BTSets {
	var s sets.BTSets
	if cur != nil {
		s = *cur
	} else {
		s = *sets.DefaultBTSets()
	}
	s.CacheSize = e.CacheSize
	s.ReaderReadAHead = e.ReaderReadAhead
	s.PreloadCache = e.PreloadCache
	s.UseDisk = e.UseDisk
	s.TorrentsSavePath = e.TorrentsSavePath
	s.RemoveCacheOnDrop = e.RemoveCacheOnDrop
	s.ForceEncrypt = e.ForceEncrypt
	s.RetrackersMode = e.RetrackersMode
	s.TorrentDisconnectTimeout = e.TorrentDisconnectTimeout
	s.EnableDebug = e.EnableDebug
	s.EnableIPv6 = e.EnableIPv6
	s.DisableTCP = e.DisableTCP
	s.DisableUTP = e.DisableUTP
	s.DisableUPNP = e.DisableUPNP
	s.DisableDHT = e.DisableDHT
	s.DisablePEX = e.DisablePEX
	s.DisableUpload = e.DisableUpload
	s.DownloadRateLimit = e.DownloadRateLimit
	s.UploadRateLimit = e.UploadRateLimit
	s.ConnectionsLimit = e.ConnectionsLimit
	s.PeersListenPort = e.PeersListenPort
	s.ResponsiveMode = e.ResponsiveMode
	return &s
}
