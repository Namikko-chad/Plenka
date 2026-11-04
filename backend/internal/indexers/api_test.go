package indexers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"plenka/internal/appconfig"
	"plenka/internal/store"
)

func init() { gin.SetMode(gin.TestMode) }

// makeArchive builds a .tar.gz shaped like a repository snapshot.
func makeArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	add := func(name, body string) {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(body))
	}
	_ = tw.WriteHeader(&tar.Header{Name: "Example-master/src/Example.Common/Definitions/", Typeflag: tar.TypeDir, Mode: 0o755})
	for name, body := range files {
		add("Example-master/src/Example.Common/Definitions/"+name, body)
	}
	// decoys that must be ignored
	add("Example-master/src/Example.Common/Indexers/Definitions/Native.cs", "class X {}")
	add("Example-master/src/Example.Common/Definitions/README.md", "readme")
	add("Example-master/docs/sample.yml", "id: not-a-definition")
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

// defsServer serves the current archive; status != 200 simulates an outage.
type defsServer struct {
	*httptest.Server
	mu       sync.Mutex
	archive  []byte
	status   int
	requests int32
	gate     chan struct{} // when set, requests wait for it
}

func newDefsServer(t *testing.T) *defsServer {
	ds := &defsServer{status: 200}
	ds.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&ds.requests, 1)
		ds.mu.Lock()
		gate, status, data := ds.gate, ds.status, ds.archive
		ds.mu.Unlock()
		if gate != nil {
			<-gate
		}
		if status != 200 {
			http.Error(w, "down", status)
			return
		}
		w.Write(data)
	}))
	t.Cleanup(ds.Close)
	return ds
}

func (ds *defsServer) set(archive []byte, status int) {
	ds.mu.Lock()
	ds.archive, ds.status = archive, status
	ds.mu.Unlock()
}

// fakeKnaben points the native Knaben indexer at a local fictional API.
func fakeKnaben(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"hits":[{"title":"Sample Movie (2001) 1080p","categoryId":[3001000],"hash":"%s",
			"details":"https://meta.example.test/d/1","link":"","magnetUrl":"","bytes":1610612736,"seeders":5,"peers":1,
			"date":"2026-06-01T10:00:00","trackerId":"x","tracker":"Example Upstream"}]}`, strings.ToUpper(hashA))
	}))
	old := knabenAPI
	knabenAPI = srv.URL
	t.Cleanup(func() { knabenAPI = old; srv.Close() })
}

type testEnv struct {
	m      *Manager
	r      *gin.Engine
	ds     *defsServer
	site   *httptest.Server
	dir    string
	config func() appconfig.Config
}

func newEnv(t *testing.T) *testEnv {
	t.Helper()
	fakeKnaben(t)
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	env := &testEnv{ds: newDefsServer(t), dir: t.TempDir()}
	env.config = func() appconfig.Config { return appconfig.Config{SearchTimeoutSec: 10} }
	env.site = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/get/"):
			w.Write(fakeTorrent)
		case r.URL.Path == "/slow/":
			time.Sleep(1500 * time.Millisecond)
		default:
			fmt.Fprintf(w, publicPage, hashB)
		}
	}))
	t.Cleanup(env.site.Close)
	m, err := New(Options{Store: st, Config: func() appconfig.Config { return env.config() }, DataDir: env.dir, DefinitionsURL: env.ds.URL})
	if err != nil {
		t.Fatal(err)
	}
	env.m = m
	env.r = gin.New()
	m.RegisterRoutes(env.r)
	return env
}

func (env *testEnv) do(t *testing.T, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	var rd *strings.Reader
	if body != "" {
		rd = strings.NewReader(body)
	} else {
		rd = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	env.r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

// enable switches the given indexers on (everything starts disabled).
func (env *testEnv) enable(t *testing.T, ids ...string) {
	t.Helper()
	for _, id := range ids {
		if w, _ := env.do(t, "PUT", "/api/indexers/"+id, `{"enabled":true}`); w.Code != 200 {
			t.Fatalf("enable %s: %d", id, w.Code)
		}
	}
}

// publicDefFor points the public example definition at the local site.
func publicDefFor(site string) string {
	return strings.Replace(publicDef, "https://example-tracker.test/", site+"/", 1)
}

const unsupportedDef = `---
id: odd-example
name: Odd Example
description: Uses a filter that does not exist
language: en-US
type: public
encoding: UTF-8
links: [https://odd.example.test/]
caps:
  categorymappings:
    - {id: 1, cat: Other}
settings: []
search:
  paths:
    - path: /
  rows:
    selector: tr
  fields:
    title:
      selector: td
      filters:
        - name: frobnicate
`

func TestDefinitionsLifecycleAndAPI(t *testing.T) {
	env := newEnv(t)

	// nothing downloaded yet: only the native indexer
	_, st := env.do(t, "GET", "/api/definitions", "")
	if st["count"].(float64) != 0 || st["updatedAt"] != nil || st["source"] != env.ds.URL || st["updating"] != false {
		t.Fatalf("initial status: %v", st)
	}
	if total, enabled := env.m.Stats(); total != 2 || enabled != 0 {
		t.Fatalf("stats before update: %d %d", total, enabled)
	}

	env.ds.set(makeArchive(t, map[string]string{
		"example-tracker.yml": publicDefFor(env.site.URL),
		"private-example.yml": privateDef,
		"odd-example.yml":     unsupportedDef,
		"broken.yml":          "id: [unclosed",
	}), 200)
	w, st := env.do(t, "POST", "/api/definitions/update", "")
	if w.Code != 200 || st["count"].(float64) != 3 || st["updatedAt"] == nil || st["lastError"] != "" {
		t.Fatalf("update: %d %v", w.Code, st)
	}
	if files, _ := filepath.Glob(filepath.Join(env.dir, "definitions", "*")); len(files) != 4 {
		t.Fatalf("files on disk: %v", files) // broken.yml is stored but skipped when loading
	}
	if total, enabled := env.m.Stats(); total != 5 || enabled != 0 {
		t.Fatalf("stats after update: %d %d (everything starts disabled)", total, enabled)
	}
	env.enable(t, "example-tracker", "knaben")

	// list + filters
	w = httptest.NewRecorder()
	env.r.ServeHTTP(w, httptest.NewRequest("GET", "/api/indexers?type=private", nil))
	var list []map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 || list[0]["id"] != "private-example" || list[0]["configured"] != false {
		t.Fatalf("private list: %s", w.Body.String())
	}
	_, odd := env.do(t, "GET", "/api/indexers/odd-example", "")
	if !strings.Contains(odd["lastError"].(string), `filter "frobnicate" is not supported`) {
		t.Fatalf("unsupported definition must say why: %v", odd["lastError"])
	}

	// settings: password never returned
	w, det := env.do(t, "PUT", "/api/indexers/private-example", `{"enabled":true,"config":{"username":"alice","password":"pw"}}`)
	cfg := det["config"].(map[string]any)
	if w.Code != 200 || cfg["password"] != "" || cfg["password__set"] != "1" || cfg["username"] != "alice" || det["configured"] != true {
		t.Fatalf("put: %d %v", w.Code, det)
	}
	if cfg["sitelink"] != "https://private-example.test/" {
		t.Fatalf("sitelink: %v", cfg["sitelink"])
	}
	_, det = env.do(t, "PUT", "/api/indexers/private-example", `{"config":{"password":""}}`)
	if det["config"].(map[string]any)["password__set"] != "1" {
		t.Fatal("empty password must keep the stored one")
	}
	if w, _ := env.do(t, "PUT", "/api/indexers/private-example", `{"config":{"sitelink":"ftp://x"}}`); w.Code != 400 {
		t.Fatalf("bad sitelink: %d", w.Code)
	}
	_, det = env.do(t, "PUT", "/api/indexers/private-example", `{"enabled":false}`)
	settings := det["settings"].([]any)
	if len(settings) != 2 || det["categories"].([]any)[0].(map[string]any)["name"] != "Movies" {
		t.Fatalf("detail: %v", det)
	}
	_, det = env.do(t, "GET", "/api/indexers/example-tracker", "")
	opts := det["settings"].([]any)[0].(map[string]any)["options"].([]any)
	if opts[0].(map[string]any)["value"] != "seeders" || opts[1].(map[string]any)["label"] != "Date" {
		t.Fatalf("options keep their order: %v", opts)
	}

	// search over the enabled indexers (example-tracker + knaben)
	w, res := env.do(t, "GET", "/api/search?q=Sample+Movie", "")
	if w.Code != 200 {
		t.Fatalf("search: %d %s", w.Code, w.Body.String())
	}
	results := res["results"].([]any)
	statuses := res["indexers"].([]any)
	if len(statuses) != 2 {
		t.Fatalf("statuses: %v", statuses)
	}
	// example-tracker returns 2 matching rows (hashB on row 1), knaben 1 (hashA): 3 results
	if len(results) != 3 {
		t.Fatalf("results: %s", w.Body.String())
	}
	first := results[0].(map[string]any)
	if first["seeders"].(float64) != 120 || first["trackerId"] != "example-tracker" || !strings.HasPrefix(first["link"].(string), "magnet:") ||
		first["infoHash"] != hashB || first["peers"].(float64) != 7 || first["categoryDesc"] != "Movies/HD" {
		t.Fatalf("first result: %v", first)
	}
	var tokenLink string
	for _, x := range results {
		r := x.(map[string]any)
		if strings.HasPrefix(r["link"].(string), "/api/indexers/") {
			tokenLink = r["link"].(string)
		}
		if r["publishDate"] == nil {
			t.Errorf("publishDate missing: %v", r)
		}
	}
	if !strings.HasPrefix(tokenLink, "/api/indexers/example-tracker/download?token=") {
		t.Fatalf("expected a token link for the row without infohash: %v", results)
	}

	// token download over HTTP and in-process
	w = httptest.NewRecorder()
	env.r.ServeHTTP(w, httptest.NewRequest("GET", tokenLink, nil))
	if w.Code != 200 || w.Header().Get("Content-Type") != "application/x-bittorrent" || !bytes.Equal(w.Body.Bytes(), fakeTorrent) {
		t.Fatalf("download: %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	u, _ := url.Parse(tokenLink)
	data, magnet, err := env.m.ResolveLink("example-tracker", u.Query().Get("token"))
	if err != nil || magnet != "" || !bytes.Equal(data, fakeTorrent) {
		t.Fatalf("ResolveLink: %v %q", err, magnet)
	}
	if _, _, err := env.m.ResolveLink("private-example", u.Query().Get("token")); err == nil {
		t.Fatal("a token must only resolve through its own indexer")
	}
	if w, _ := env.do(t, "GET", "/api/indexers/example-tracker/download?token=forged.sig", ""); w.Code != 400 {
		t.Fatalf("bad token: %d", w.Code)
	}
	if w, _ := env.do(t, "GET", "/api/indexers/nope/download?token=x", ""); w.Code != 404 {
		t.Fatalf("unknown indexer: %d", w.Code)
	}

	// category filter and explicit indexer list
	_, res = env.do(t, "GET", "/api/search?q=Sample+Movie&cat=5000&indexers=example-tracker", "")
	if n := len(res["results"].([]any)); n != 1 {
		t.Fatalf("cat=5000: %d results", n)
	}
	if w, _ := env.do(t, "GET", "/api/search?q=x&cat=abc", ""); w.Code != 400 {
		t.Fatal("bad cat must be 400")
	}

	// test endpoint
	_, tr := env.do(t, "POST", "/api/indexers/example-tracker/test", "")
	if tr["ok"] != true || tr["results"].(float64) != 3 {
		t.Fatalf("test: %v", tr)
	}
	env.do(t, "PUT", "/api/indexers/private-example", `{"config":{"sitelink":"http://127.0.0.1:1/"}}`) // nothing listens there
	_, tr = env.do(t, "POST", "/api/indexers/private-example/test", "")
	if tr["ok"] != false || tr["error"] == "" {
		t.Fatalf("test of an unreachable tracker must fail: %v", tr)
	}

	// failed update keeps the set and records the error
	env.ds.set(nil, 500)
	w, body := env.do(t, "POST", "/api/definitions/update", "")
	if w.Code != 502 || !strings.Contains(body["error"].(string), "HTTP 500") {
		t.Fatalf("failed update: %d %v", w.Code, body)
	}
	_, st = env.do(t, "GET", "/api/definitions", "")
	if st["count"].(float64) != 3 || !strings.Contains(st["lastError"].(string), "HTTP 500") || st["updatedAt"] == nil {
		t.Fatalf("status after failure: %v", st)
	}
	env.ds.set(makeArchive(t, map[string]string{}), 200)
	if w, body := env.do(t, "POST", "/api/definitions/update", ""); w.Code != 502 || !strings.Contains(body["error"].(string), "no definitions") {
		t.Fatalf("empty archive: %d %v", w.Code, body)
	}

	// an id that disappears is hidden; it comes back with its settings
	env.ds.set(makeArchive(t, map[string]string{"example-tracker.yml": publicDefFor(env.site.URL)}), 200)
	if w, _ := env.do(t, "POST", "/api/definitions/update", ""); w.Code != 200 {
		t.Fatal("update")
	}
	if w, _ := env.do(t, "GET", "/api/indexers/private-example", ""); w.Code != 404 {
		t.Fatalf("removed definition must be hidden: %d", w.Code)
	}
	env.ds.set(makeArchive(t, map[string]string{"example-tracker.yml": publicDefFor(env.site.URL), "private-example.yml": privateDef}), 200)
	env.do(t, "POST", "/api/definitions/update", "")
	_, det = env.do(t, "GET", "/api/indexers/private-example", "")
	if c := det["config"].(map[string]any); c["username"] != "alice" || c["password__set"] != "1" || c["sitelink"] != "http://127.0.0.1:1/" {
		t.Fatalf("settings must survive: %v", det)
	}
	_, ex := env.do(t, "GET", "/api/indexers/example-tracker", "")
	if ex["enabled"] != true {
		t.Fatal("enabled state must survive updates")
	}
}

func TestConcurrentUpdatesJoin(t *testing.T) {
	env := newEnv(t)
	gate := make(chan struct{})
	env.ds.mu.Lock()
	env.ds.archive, env.ds.gate = makeArchive(t, map[string]string{"example-tracker.yml": publicDefFor(env.site.URL)}), gate
	env.ds.mu.Unlock()
	var wg sync.WaitGroup
	codes := make([]int, 3)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w, _ := env.do(t, "POST", "/api/definitions/update", "")
			codes[i] = w.Code
		}(i)
	}
	for !env.m.DefinitionsStatus().Updating {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	close(gate)
	wg.Wait()
	if n := atomic.LoadInt32(&env.ds.requests); n != 1 {
		t.Fatalf("expected one download, got %d", n)
	}
	for _, c := range codes {
		if c != 200 {
			t.Fatalf("codes: %v", codes)
		}
	}
	if env.m.DefinitionsStatus().Count != 1 {
		t.Fatal("count")
	}
}

func TestStartupLoadsFromDiskAndBackgroundUpdate(t *testing.T) {
	env := newEnv(t)
	env.ds.set(makeArchive(t, map[string]string{"example-tracker.yml": publicDefFor(env.site.URL)}), 200)
	env.m.StartDefinitionsUpdate()
	deadline := time.Now().Add(5 * time.Second)
	for env.m.DefinitionsStatus().Count != 1 {
		if time.Now().After(deadline) {
			t.Fatal("background update did not finish")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// a new manager on the same data dir loads the stored set without downloading
	env.ds.set(nil, 500)
	before := atomic.LoadInt32(&env.ds.requests)
	m2, err := New(Options{Store: env.m.st, DataDir: env.dir, DefinitionsURL: env.ds.URL})
	if err != nil {
		t.Fatal(err)
	}
	if m2.DefinitionsStatus().Count != 1 || atomic.LoadInt32(&env.ds.requests) != before {
		t.Fatal("startup must load from disk only")
	}
	// an interrupted swap is recovered
	dir := filepath.Join(env.dir, "definitions")
	if err := os.Rename(dir, dir+".old"); err != nil {
		t.Fatal(err)
	}
	m3, _ := New(Options{Store: env.m.st, DataDir: env.dir, DefinitionsURL: env.ds.URL})
	if m3.DefinitionsStatus().Count != 1 {
		t.Fatal("interrupted swap not recovered")
	}
}

func TestSearchTimeout(t *testing.T) {
	env := newEnv(t)
	slow := strings.Replace(publicDefFor(env.site.URL), `"{{ if .Keywords }}search/{{ .Keywords }}/{{ else }}latest/{{ end }}"`, "slow/", 1)
	env.ds.set(makeArchive(t, map[string]string{"example-tracker.yml": slow}), 200)
	env.do(t, "POST", "/api/definitions/update", "")
	env.enable(t, "example-tracker", "knaben")
	env.config = func() appconfig.Config { return appconfig.Config{SearchTimeoutSec: 1} }
	start := time.Now()
	_, res := env.do(t, "GET", "/api/search?q=Sample", "")
	if time.Since(start) > 2500*time.Millisecond {
		t.Fatal("search did not respect the timeout")
	}
	var slowStatus map[string]any
	for _, s := range res["indexers"].([]any) {
		if s.(map[string]any)["id"] == "example-tracker" {
			slowStatus = s.(map[string]any)
		}
	}
	if slowStatus["status"] != "error" || !strings.Contains(slowStatus["error"].(string), "timed out") {
		t.Fatalf("status: %v", slowStatus)
	}
	if len(res["results"].([]any)) != 1 { // knaben still answers
		t.Fatalf("results: %v", res["results"])
	}
}

func TestDedupe(t *testing.T) {
	e := &entry{meta: &indexerMeta{ID: "a"}}
	hits := []hit{
		{&Release{Title: "X", InfoHash: hashA, Seeders: 1}, e},
		{&Release{Title: "Y", InfoHash: strings.ToUpper(hashA), Seeders: 9}, e},
		{&Release{Title: "Same", Size: 10, Seeders: 3}, e},
		{&Release{Title: " same ", Size: 10, Seeders: 2}, e},
		{&Release{Title: "Same", Size: 11, Seeders: 4}, e},
	}
	out := dedupeSort(hits)
	if len(out) != 3 || out[0].rel.Title != "Y" || out[1].rel.Size != 11 {
		t.Fatalf("dedupe: %+v", out)
	}
}

func TestExtractRejectsUnsafeNames(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, n := range []string{"x/Definitions/../../evil.yml", "x/Definitions/.hidden.yml", "x/Definitions/ok.yml"} {
		tw.WriteHeader(&tar.Header{Name: n, Mode: 0o644, Size: 2, Typeflag: tar.TypeReg})
		tw.Write([]byte("id"))
	}
	tw.Close()
	gz.Close()
	dst := t.TempDir()
	n, err := extractDefinitions(&buf, dst)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if _, err := os.Stat(filepath.Join(dst, "ok.yml")); err != nil {
		t.Fatal(err)
	}
}
