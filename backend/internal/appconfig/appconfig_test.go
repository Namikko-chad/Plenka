package appconfig

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"plenka/internal/store"
)

func newService(t *testing.T, env string) (*Service, *gin.Engine) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	s, err := New(st, env)
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	s.RegisterRoutes(r)
	return s, r
}

func put(r *gin.Engine, body string) (*httptest.ResponseRecorder, response) {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(body)))
	var out response
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func TestDefaults(t *testing.T) {
	s, r := newService(t, "")
	c := s.Get()
	if c.SearchTimeoutSec != 30 {
		t.Fatalf("unexpected defaults: %+v", c)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	if strings.Contains(w.Body.String(), "tmdbLanguage") {
		t.Fatalf("the catalog language belongs to the clients, not the config: %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "jackett") {
		t.Fatalf("config response must not have a Jackett key: %s", w.Body.String())
	}
}

func TestTMDBKeyNeverReturnedAndClearable(t *testing.T) {
	s, r := newService(t, "")
	w, out := put(r, `{"tmdbApiKey":"secret"}`)
	if w.Code != 200 || out.TMDBApiKey != "" || !out.TMDBApiKeySet {
		t.Fatalf("set key: %d %+v", w.Code, out)
	}
	if s.Get().TMDBApiKey != "secret" {
		t.Fatal("key not stored")
	}
	_, out = put(r, `{"tmdbApiKey":""}`)
	if !out.TMDBApiKeySet {
		t.Fatal("empty string must keep the key")
	}
	_, out = put(r, `{"tmdbApiKey":"-"}`)
	if out.TMDBApiKeySet {
		t.Fatal(`"-" must clear the key`)
	}
}

func TestEnvKeyWins(t *testing.T) {
	s, _ := newService(t, "fromenv")
	if s.Get().TMDBApiKey != "fromenv" {
		t.Fatal("env key must override")
	}
}

func TestTimeoutValidation(t *testing.T) {
	_, r := newService(t, "")
	if w, _ := put(r, `{"searchTimeoutSec":1}`); w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestFlareSolverrURL(t *testing.T) {
	s, r := newService(t, "")
	if w, out := put(r, `{"flareSolverrUrl":" http://solver.test:8191/ "}`); w.Code != 200 || out.FlareSolverrURL != "http://solver.test:8191" {
		t.Fatalf("set: %d %+v", w.Code, out)
	}
	if s.Get().FlareSolverrURL != "http://solver.test:8191" {
		t.Fatal("not stored")
	}
	if w, _ := put(r, `{"flareSolverrUrl":"solver.test"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("bad URL: %d", w.Code)
	}
	if _, out := put(r, `{"flareSolverrUrl":""}`); out.FlareSolverrURL != "" {
		t.Fatal(`"" must turn it off`)
	}
}
