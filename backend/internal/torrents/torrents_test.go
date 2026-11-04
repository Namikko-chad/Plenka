package torrents

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/gin-gonic/gin"

	sets "plenka/internal/torrserver/settings"
	"plenka/internal/torrserver/torr"
	"plenka/internal/torrserver/torr/state"
)

// The engine keeps global state (settings, config.db, the BT server), so every test
// shares one data dir. The BT client is never connected: no network.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "torrents-test")
	if err != nil {
		panic(err)
	}
	sets.Path = dir
	if err := sets.InitSets(); err != nil {
		panic(err)
	}
	torr.InitApiHelper(torr.NewBTS())
	code := m.Run()
	sets.CloseDB()
	os.RemoveAll(dir)
	os.Exit(code)
}

func router() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New().RegisterRoutes(r)
	return r
}

func do(r http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// buildTorrent makes .torrent metainfo of a small multi-file torrent.
func buildTorrent(t *testing.T) *metainfo.MetaInfo {
	t.Helper()
	root := filepath.Join(t.TempDir(), "The.Matrix.1999")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"movie.mkv": "video bytes", "info.txt": "text", "Movie RU.srt": "1"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	info := metainfo.Info{PieceLength: 16 << 10}
	if err := info.BuildFromFilePath(root); err != nil {
		t.Fatal(err)
	}
	ib, err := bencode.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	return &metainfo.MetaInfo{InfoBytes: ib}
}

// seed saves a torrent with metadata as if it had been added earlier.
func seed(t *testing.T, title, data string) string {
	t.Helper()
	mi := buildTorrent(t)
	hash := mi.HashInfoBytes()
	sets.AddTorrent(&sets.TorrentDB{
		TorrentSpec: &torrent.TorrentSpec{InfoHash: hash, InfoBytes: mi.InfoBytes, DisplayName: "The.Matrix.1999"},
		Title:       title,
		Data:        data,
		Timestamp:   1793685600, // 2026-11-03T06:00:00Z
	})
	t.Cleanup(func() { sets.RemTorrent(hash) })
	return hash.HexString()
}

func TestTorrentJSONShape(t *testing.T) {
	hash := seed(t, "Матрица (1999)", `{"TorrServer":{"Files":[]}}`)
	r := router()

	w := do(r, httptest.NewRequest(http.MethodGet, "/api/torrents/"+hash, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d %s", w.Code, w.Body)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"hash", "title", "poster", "data", "name", "stat", "statString", "addedAt",
		"torrentSize", "loadedSize", "preloadedBytes", "preloadSize", "fullDownload", "downloadSpeed", "uploadSpeed",
		"totalPeers", "pendingPeers", "activePeers", "connectedSeeders", "halfOpenPeers", "bytesRead", "files"} {
		if _, ok := raw[k]; !ok {
			t.Errorf("missing field %q", k)
		}
	}
	var got Torrent
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Hash != hash || got.Stat != 5 || got.AddedAt != "2026-11-03T06:00:00Z" || got.Data != "" ||
		got.Name != "The.Matrix.1999" || got.Title != "Матрица (1999)" {
		t.Errorf("torrent: %+v", got)
	}
	wantPaths := []string{"The.Matrix.1999/info.txt", "The.Matrix.1999/movie.mkv", "The.Matrix.1999/Movie RU.srt"}
	if len(got.Files) != 3 {
		t.Fatalf("files: %+v", got.Files)
	}
	var size int64
	for i, f := range got.Files {
		size += f.Length
		if f.ID != i+1 {
			t.Errorf("file %d id %d", i, f.ID)
		}
		if !contains(wantPaths, f.Path) {
			t.Errorf("unexpected path %q", f.Path)
		}
	}
	if got.TorrentSize != size {
		t.Errorf("size %d, files %d", got.TorrentSize, size)
	}

	w = do(r, httptest.NewRequest(http.MethodGet, "/api/torrents", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), hash) || !strings.HasPrefix(w.Body.String(), "[") {
		t.Errorf("list: %d %s", w.Code, w.Body)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestEmptyStatusHasEmptyFiles(t *testing.T) {
	got := torrentFromStatus(&state.TorrentStatus{Hash: "AB"}, nil)
	buf, _ := json.Marshal(got)
	if !strings.Contains(string(buf), `"files":[]`) {
		t.Errorf("files not []: %s", buf)
	}
}

func TestNotFound(t *testing.T) {
	r := router()
	unknown := strings.Repeat("ab", 20)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/torrents/" + unknown},
		{http.MethodGet, "/api/torrents/not-a-hash"},
		{http.MethodDelete, "/api/torrents/" + unknown},
		{http.MethodPost, "/api/torrents/" + unknown + "/drop"},
		{http.MethodGet, "/api/torrents/" + unknown + "/cache"},
		{http.MethodPost, "/api/torrents/" + unknown + "/preload"},
		{http.MethodPost, "/api/torrents/" + unknown + "/download"},
		{http.MethodDelete, "/api/torrents/" + unknown + "/download"},
		{http.MethodGet, "/api/torrents/" + unknown + "/playlist.m3u"},
		{http.MethodGet, "/api/stream/" + unknown + "/1"},
		{http.MethodHead, "/api/stream/" + unknown + "/1/movie.mkv"},
		{http.MethodGet, "/api/stream/" + unknown + "/1/The%20Matrix.mkv"},
	} {
		w := do(r, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s %s: %d", tc.method, tc.path, w.Code)
		}
		if tc.method != http.MethodHead && !strings.Contains(w.Body.String(), `"error"`) {
			t.Errorf("%s %s: body %s", tc.method, tc.path, w.Body)
		}
	}

	hash := seed(t, "x", "")
	for _, p := range []string{"/api/stream/" + hash + "/99/x.mkv", "/api/stream/" + hash + "/abc", "/api/torrents/" + hash + "/playlist.m3u?file=99"} {
		if w := do(r, httptest.NewRequest(http.MethodGet, p, nil)); w.Code != http.StatusNotFound {
			t.Errorf("%s: %d", p, w.Code)
		}
	}
}

func TestPreloadBadInput(t *testing.T) {
	hash := seed(t, "x", "")
	r := router()
	for body, want := range map[string]int{
		``:              http.StatusBadRequest,
		`{}`:            http.StatusBadRequest,
		`{"fileId":0}`:  http.StatusBadRequest,
		`{"fileId":99}`: http.StatusNotFound,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/torrents/"+hash+"/preload", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if w := do(r, req); w.Code != want || !strings.Contains(w.Body.String(), `"error"`) {
			t.Errorf("body %q: %d %s", body, w.Code, w.Body)
		}
	}
}

func TestCancelDownloadOfUnloadedTorrent(t *testing.T) {
	hash := seed(t, "x", "")
	w := do(router(), httptest.NewRequest(http.MethodDelete, "/api/torrents/"+hash+"/download", nil))
	if w.Code != http.StatusNoContent {
		t.Errorf("cancel: %d %s", w.Code, w.Body)
	}
	w = do(router(), httptest.NewRequest(http.MethodGet, "/api/torrents/"+hash, nil))
	if !strings.Contains(w.Body.String(), `"fullDownload":false`) {
		t.Errorf("get: %s", w.Body)
	}
}

func TestCacheOfUnloadedTorrent(t *testing.T) {
	hash := seed(t, "x", "")
	w := do(router(), httptest.NewRequest(http.MethodGet, "/api/torrents/"+hash+"/cache", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"pieces":{}`) || !strings.Contains(w.Body.String(), `"readers":[]`) {
		t.Errorf("cache: %d %s", w.Code, w.Body)
	}
}

func TestRemoveAndDrop(t *testing.T) {
	hash := seed(t, "x", "")
	r := router()
	if w := do(r, httptest.NewRequest(http.MethodPost, "/api/torrents/"+hash+"/drop", nil)); w.Code != http.StatusNoContent {
		t.Errorf("drop: %d", w.Code)
	}
	if w := do(r, httptest.NewRequest(http.MethodDelete, "/api/torrents/"+hash, nil)); w.Code != http.StatusNoContent {
		t.Errorf("delete: %d", w.Code)
	}
	if w := do(r, httptest.NewRequest(http.MethodGet, "/api/torrents/"+hash, nil)); w.Code != http.StatusNotFound {
		t.Errorf("get after delete: %d", w.Code)
	}
}

func TestPlaylist(t *testing.T) {
	hash := seed(t, "Матрица: 1999", "")
	r := router()
	req := httptest.NewRequest(http.MethodGet, "/api/torrents/"+hash+"/playlist.m3u", nil)
	req.Host = "box.lan:8095"
	req.Header.Set("X-Forwarded-Proto", "https")
	w := do(r, req)
	if w.Code != http.StatusOK {
		t.Fatalf("playlist: %d %s", w.Code, w.Body)
	}
	body := w.Body.String()
	if !strings.HasPrefix(body, "#EXTM3U\n") || !strings.Contains(body, "#EXTINF:-1,movie.mkv\n") {
		t.Errorf("body: %s", body)
	}
	if !strings.Contains(body, "https://box.lan:8095/api/stream/"+hash+"/3/movie.mkv\n") {
		t.Errorf("stream url: %s", body)
	}
	if strings.Contains(body, "info.txt") || strings.Contains(body, ".srt") {
		t.Errorf("non-media files listed: %s", body)
	}
	if ct := w.Header().Get("Content-Type"); ct != "audio/x-mpegurl" {
		t.Errorf("content type %q", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "attachment;") || !strings.Contains(cd, "filename*=utf-8''") || strings.Contains(cd, ":") {
		t.Errorf("content disposition %q", cd)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/torrents/"+hash+"/playlist.m3u?file=1", nil)
	req.Host = "10.0.0.1:8095"
	body = do(r, req).Body.String()
	if !strings.Contains(body, "http://10.0.0.1:8095/api/stream/"+hash+"/1/Movie%20RU.srt\n") || strings.Contains(body, "movie.mkv") {
		t.Errorf("single file: %s", body)
	}
}

func TestBuildM3UAndNames(t *testing.T) {
	got := buildM3U("http://h", "abc", []File{{ID: 1, Path: "d/a b#1.mkv"}})
	want := "#EXTM3U\n#EXTINF:-1,a b#1.mkv\nhttp://h/api/stream/abc/1/a%20b%231.mkv\n"
	if got != want {
		t.Errorf("m3u:\n%q\nwant\n%q", got, want)
	}
	if got := sanitizeFileName(` a/b\c:d*e?"f<g>h|i. `); got != "a_b_c_d_e__f_g_h_i" {
		t.Errorf("sanitize: %q", got)
	}
	if got := attachmentName("Plain"); got != `attachment; filename=Plain.m3u` {
		t.Errorf("attachment: %q", got)
	}
	if got := attachmentName("  "); got != `attachment; filename=playlist.m3u` {
		t.Errorf("attachment fallback: %q", got)
	}
}

func TestAddBadInput(t *testing.T) {
	r := router()
	for _, body := range []string{`{`, `{"link":""}`, `{"link":"not a link"}`, `{"link":"file:///etc/passwd"}`, `{"link":"magnet:?xt=urn:btih:zz"}`} {
		req := httptest.NewRequest(http.MethodPost, "/api/torrents", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if w := do(r, req); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `"error"`) {
			t.Errorf("%s: %d %s", body, w.Code, w.Body)
		}
	}

	if w := do(r, httptest.NewRequest(http.MethodPost, "/api/torrents/upload", nil)); w.Code != http.StatusBadRequest {
		t.Errorf("upload without file: %d", w.Code)
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "x.torrent")
	fw.Write([]byte("not bencode"))
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/torrents/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if w := do(r, req); w.Code != http.StatusBadRequest {
		t.Errorf("upload garbage: %d %s", w.Code, w.Body)
	}
}

func TestParseLink(t *testing.T) {
	mi := buildTorrent(t)
	hash := mi.HashInfoBytes()
	var tf bytes.Buffer
	if err := mi.Write(&tf); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/t.torrent":
			w.Write(tf.Bytes())
		case "/magnet":
			http.Redirect(w, r, "magnet:?xt=urn:btih:"+hash.HexString(), http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	defer func(old func(string, string) ([]byte, string, error)) { IndexerLinkResolver = old }(IndexerLinkResolver)
	IndexerLinkResolver = func(id, token string) ([]byte, string, error) {
		switch {
		case id == "file" && token == "t1":
			return tf.Bytes(), "", nil
		case id == "mag" && token == "t2":
			return nil, "magnet:?xt=urn:btih:" + hash.HexString() + "&dn=Matrix", nil
		}
		return nil, "", errors.New("bad token")
	}

	for _, link := range []string{
		"magnet:?xt=urn:btih:" + hash.HexString() + "&amp;dn=x",
		hash.HexString(),
		strings.ToUpper(hash.HexString()),
		srv.URL + "/t.torrent",
		srv.URL + "/magnet",
		"/api/indexers/file/download?token=t1",
		"http://other.host:8095/api/indexers/mag/download?token=t2",
	} {
		spec, err := parseLink(link)
		if err != nil {
			t.Errorf("%s: %v", link, err)
			continue
		}
		if spec.InfoHash != hash {
			t.Errorf("%s: hash %s", link, spec.InfoHash.HexString())
		}
	}
	if spec, _ := parseLink(srv.URL + "/t.torrent"); spec == nil || len(spec.InfoBytes) == 0 || spec.DisplayName != "The.Matrix.1999" {
		t.Errorf(".torrent spec: %+v", spec)
	}

	for _, link := range []string{"", "file:///etc/passwd", "torrs://abc", "hello", srv.URL + "/missing",
		"/api/indexers/file/download?token=bad"} {
		if _, err := parseLink(link); err == nil {
			t.Errorf("%q: no error", link)
		}
	}
}

func TestEngineSettings(t *testing.T) {
	oldSet, oldDef := applySettings, applyDefaults
	defer func() { applySettings, applyDefaults = oldSet, oldDef }()
	applySettings = sets.SetBTSets
	applyDefaults = sets.SetDefaultConfig
	r := router()

	w := do(r, httptest.NewRequest(http.MethodGet, "/api/engine/settings", nil))
	var raw map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil || w.Code != http.StatusOK {
		t.Fatalf("get: %d %s", w.Code, w.Body)
	}
	if len(raw) != 22 || raw["cacheSize"] == nil || raw["readerReadAhead"] == nil || raw["responsiveMode"] == nil {
		t.Errorf("shape: %v", raw)
	}
	if _, ok := raw["defaultTrackers"]; ok {
		t.Errorf("hidden field exposed: %v", raw)
	}

	trackers := sets.BTsets.DefaultTrackers
	put := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPut, "/api/engine/settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		return do(r, req)
	}
	w = put(`{"cacheSize": 134217728, "readerReadAhead": 500, "preloadCache": -3, "retrackersMode": 9, "useDisk": true, "torrentsSavePath": "", "connectionsLimit": 50}`)
	var got EngineSettings
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != http.StatusOK {
		t.Fatalf("put: %d %s", w.Code, w.Body)
	}
	if got.CacheSize != 134217728 || got.ReaderReadAhead != 100 || got.PreloadCache != 0 || got.RetrackersMode != 3 ||
		got.UseDisk || got.ConnectionsLimit != 50 || !got.ResponsiveMode {
		t.Errorf("put result: %+v", got)
	}
	if sets.BTsets.DefaultTrackers != trackers || sets.BTsets.CacheSize != 134217728 {
		t.Errorf("engine settings: %+v", sets.BTsets)
	}
	if w := put(`{"cacheSize": "big"}`); w.Code != http.StatusBadRequest {
		t.Errorf("bad put: %d", w.Code)
	}

	w = do(r, httptest.NewRequest(http.MethodDelete, "/api/engine/settings", nil))
	got = EngineSettings{}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if w.Code != http.StatusOK || got != fromBTSets(sets.DefaultBTSets()) {
		t.Errorf("reset: %d %+v", w.Code, got)
	}
}
