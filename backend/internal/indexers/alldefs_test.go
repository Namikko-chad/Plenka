package indexers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestAllDefinitions loads every definition in $PLENKA_DEFINITIONS_DIR (for example a
// downloaded $DATA_DIR/definitions) and reports how many load and pass the checks. It
// then runs a search with each one against a local stub site, to catch runtime errors
// in templates and selectors. Skipped when the variable is not set.
type stubTransport struct{ target, kind string }

func (s stubTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	u, _ := url.Parse(s.target)
	r2 := r.Clone(r.Context())
	r2.URL.Scheme, r2.URL.Host, r2.Host = u.Scheme, u.Host, u.Host
	r2.Header.Set("X-Stub-Kind", s.kind)
	return http.DefaultTransport.RoundTrip(r2)
}

func TestAllDefinitions(t *testing.T) {
	dir := os.Getenv("PLENKA_DEFINITIONS_DIR")
	if dir == "" {
		t.Skip("set PLENKA_DEFINITIONS_DIR to a directory of *.yml definitions")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.yml"))
	if len(files) == 0 {
		t.Skipf("no definitions in %s", dir)
	}
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-Stub-Kind") {
		case "json":
			fmt.Fprint(w, `{}`)
		case "xml":
			fmt.Fprint(w, `<?xml version="1.0"?><rss><channel></channel></rss>`)
		default:
			fmt.Fprint(w, `<html><body><a class="logout" href="logout.php">x</a><table><tr><td>-</td></tr></table></body></html>`)
		}
	}))
	defer stub.Close()

	loaded, supported := 0, 0
	var problems []string
	runtimeErrs := map[string]int{}
	t0 := time.Now()
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		d, err := parseDefinition(data)
		if err != nil {
			problems = append(problems, fmt.Sprintf("SKIPPED %s: %v", filepath.Base(f), err))
			continue
		}
		loaded++
		if issues := checkDefinition(d); len(issues) > 0 {
			problems = append(problems, fmt.Sprintf("UNSUPPORTED %s: %s", d.ID, strings.Join(issues, "; ")))
			continue
		}
		supported++

		cfg := map[string]string{"sitelink": stub.URL + "/"}
		for _, s := range d.settings() {
			if s.Default == "" && !strings.EqualFold(s.Type, "checkbox") {
				cfg[s.Name] = "x"
			}
		}
		c, err := newCardigann(d, cfg, nil)
		if err != nil {
			runtimeErrs["engine: "+err.Error()]++
			continue
		}
		// every request (also absolute API URLs) goes to the stub; no delays
		c.sess.client.Transport = stubTransport{target: stub.URL, kind: c.resp}
		c.sess.delay = 0
		for _, q := range []*Query{{Q: "Sample Movie"}, {}} {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("%s: panic: %v", d.ID, r)
					}
				}()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if _, err := c.Search(ctx, q); err != nil && (strings.Contains(err.Error(), "template") || os.Getenv("PLENKA_SMOKE_ALL") != "") {
					runtimeErrs[d.ID+": "+err.Error()]++
				}
			}()
		}
	}
	sort.Strings(problems)
	for _, p := range problems {
		t.Log(p)
	}
	for e := range runtimeErrs {
		t.Errorf("runtime template error: %s", e)
	}
	t.Logf("%d files: %d loaded, %d fully supported (%s)", len(files), loaded, supported, time.Since(t0).Round(time.Millisecond))
}
