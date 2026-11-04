package indexers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// A made-up tracker behind a challenge: it answers only with the clearance cookie and the
// browser's User-Agent that the fake FlareSolverr hands out.
func TestFlareSolverr(t *testing.T) {
	const ua = "SolverBrowser/1.0"
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("cf_clearance")
		if err != nil || c.Value != "cleared" || r.UserAgent() != ua {
			w.Header().Set("Server", "cloudflare")
			w.Header().Set("Cf-Mitigated", "challenge")
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "<title>Just a moment...</title>")
			return
		}
		fmt.Fprintf(w, publicPage, hashB)
	}))
	t.Cleanup(site.Close)
	var solves int32
	var asked atomic.Value
	solver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req flareSolverrRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if r.URL.Path != "/v1" || req.Cmd != "request.get" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		atomic.AddInt32(&solves, 1)
		asked.Store(req.URL)
		fmt.Fprintf(w, `{"status":"ok","solution":{"status":200,"userAgent":%q,"cookies":[{"name":"cf_clearance","value":"cleared","domain":"127.0.0.1"},{"name":"foreign","value":"x","domain":".elsewhere.test"}]}}`, ua)
	}))
	t.Cleanup(solver.Close)

	def, err := parseDefinition([]byte(publicDefFor(site.URL)))
	if err != nil {
		t.Fatal(err)
	}
	newEngine := func(solverURL string) *cardigann {
		cg, err := newCardigann(def, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		cg.sess.solverURL = func() string { return solverURL }
		return cg
	}
	ctx := context.Background()

	if _, err := newEngine("").Search(ctx, &Query{Q: "Sample"}); !errors.Is(err, errCloudflare) {
		t.Fatalf("without FlareSolverr: %v", err)
	}

	cg := newEngine(solver.URL + "/")
	rels, err := cg.Search(ctx, &Query{Q: "Sample"})
	if err != nil || len(rels) == 0 {
		t.Fatalf("with FlareSolverr: %v %d", err, len(rels))
	}
	if !strings.HasPrefix(asked.Load().(string), site.URL) {
		t.Fatalf("solver asked for %v", asked.Load())
	}
	c, _ := cg.cookieSnapshot()
	if c["cf_clearance"] != "cleared" || c["foreign"] != "" {
		t.Fatalf("cookies: %v", c)
	}
	if _, err := cg.Search(ctx, &Query{Q: "Sample"}); err != nil || atomic.LoadInt32(&solves) != 1 {
		t.Fatalf("the clearance must be reused: %v, %d solves", err, solves)
	}

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"error","message":"Challenge not solved"}`)
	}))
	t.Cleanup(failing.Close)
	if _, err := newEngine(failing.URL).Search(ctx, &Query{Q: "Sample"}); err == nil || !strings.Contains(err.Error(), "Challenge not solved") {
		t.Fatalf("solver failure: %v", err)
	}
}
