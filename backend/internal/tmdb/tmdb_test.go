package tmdb

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"

	"plenka/internal/appconfig"
)

func setup(t *testing.T, key string) (*gin.Engine, *atomic.Int32, *http.Request) {
	t.Helper()
	var calls atomic.Int32
	var last http.Request
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		last = *r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	t.Cleanup(up.Close)
	p := New(func() appconfig.Config { return appconfig.Config{TMDBApiKey: key} })
	p.apiBase, p.imgBase = up.URL, up.URL
	gin.SetMode(gin.TestMode)
	r := gin.New()
	p.RegisterRoutes(r)
	return r, &calls, &last
}

func get(r *gin.Engine, url string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	return w
}

func TestAddsKeyKeepsLanguageAndCaches(t *testing.T) {
	r, calls, last := setup(t, "v3key")
	if w := get(r, "/api/tmdb/3/movie/popular?page=1&language=ru-RU"); w.Code != 200 {
		t.Fatalf("want 200, got %d", w.Code)
	}
	q := last.URL.Query()
	if q.Get("api_key") != "v3key" || q.Get("language") != "ru-RU" || q.Get("page") != "1" {
		t.Fatalf("unexpected upstream query %v", q)
	}
	get(r, "/api/tmdb/3/movie/popular?page=1&language=ru-RU")
	if calls.Load() != 1 {
		t.Fatalf("second identical request should be cached, upstream calls=%d", calls.Load())
	}
}

func TestBearerToken(t *testing.T) {
	r, _, last := setup(t, "aaa.bbb.ccc")
	get(r, "/api/tmdb/3/movie/1")
	if last.Header.Get("Authorization") != "Bearer aaa.bbb.ccc" || last.URL.Query().Get("api_key") != "" {
		t.Fatalf("v4 token must go in the Authorization header")
	}
}

func TestNoKeyAndBadPath(t *testing.T) {
	r, _, _ := setup(t, "")
	if w := get(r, "/api/tmdb/3/movie/1"); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no key: want 503, got %d", w.Code)
	}
	r, _, _ = setup(t, "k")
	if w := get(r, "/api/tmdb/evil"); w.Code != http.StatusBadRequest {
		t.Fatalf("bad path: want 400, got %d", w.Code)
	}
	if w := get(r, "/img/etc/passwd"); w.Code != http.StatusNotFound {
		t.Fatalf("image outside /t/p/: want 404, got %d", w.Code)
	}
}

func TestNoLanguageIsAdded(t *testing.T) {
	r, _, last := setup(t, "v3key")
	get(r, "/api/tmdb/3/movie/popular")
	if _, ok := last.URL.Query()["language"]; ok {
		t.Fatalf("the server must not pick a language: %v", last.URL.Query())
	}
}
