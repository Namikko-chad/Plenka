package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPAFallbackAndAPI(t *testing.T) {
	web := fstest.MapFS{
		"index.html":    {Data: []byte("<html>app</html>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
	r := New(InfoSource{
		TMDBConfigured: func() bool { return true },
		IndexerStats:   func() (int, int) { return 10, 3 },
	}, web)

	cases := []struct {
		path, wantBody string
		wantCode       int
	}{
		{"/", "app", 200},
		{"/movie/603", "app", 200},
		{"/assets/app.js", "console.log", 200},
		{"/assets/missing.js", "", 404},
		{"/api/nope", `"error"`, 404},
		{"/api/info", `"service":"plenka-server"`, 200},
		{"/echo", "app", 200}, // no TorrServer routes: SPA fallback
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if w.Code != tc.wantCode || !strings.Contains(w.Body.String(), tc.wantBody) {
			t.Errorf("%s: got %d %q", tc.path, w.Code, w.Body.String())
		}
	}
}
