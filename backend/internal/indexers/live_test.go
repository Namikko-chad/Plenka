//go:build live

package indexers

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"plenka/internal/store"
)

// TestLive downloads the definitions (DEFINITIONS_URL or the default) and searches the
// public indexers given in PLENKA_LIVE_INDEXERS (default: a few well-known ones).
// Run with: go test -tags live -run TestLive -v ./internal/indexers/
func TestLive(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m, err := New(Options{Store: st, DataDir: t.TempDir(), DefinitionsURL: os.Getenv("DEFINITIONS_URL")})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := m.UpdateDefinitions(ctx); err != nil {
		t.Fatalf("definitions update: %v", err)
	}
	t.Logf("definitions: %+v", m.DefinitionsStatus())

	ids := []string{"knaben", "rutor", "yts", "thepiratebay"}
	if s := os.Getenv("PLENKA_LIVE_INDEXERS"); s != "" {
		ids = strings.Split(s, ",")
	}
	query := os.Getenv("PLENKA_LIVE_QUERY")
	if query == "" {
		query = "matrix"
	}
	hits, statuses := m.search(ctx, &Query{Q: query}, ids)
	for _, s := range statuses {
		t.Logf("%-14s %-5s %4d results %5d ms %s", s.ID, s.Status, s.Results, s.ElapsedMs, s.Error)
	}
	for i, h := range hits {
		if i >= 5 {
			break
		}
		t.Logf("%s | %s | %d seeders | %d bytes | %s", h.e.meta.ID, h.rel.Title, h.rel.Seeders, h.rel.Size, m.downloadLink(h))
	}
	if len(hits) == 0 {
		t.Fatal("no results from any indexer")
	}
	// resolve the first result that needs the download endpoint
	for _, h := range hits {
		link := m.downloadLink(h)
		if !strings.HasPrefix(link, "/api/indexers/") {
			continue
		}
		u, _ := url.Parse(link)
		data, magnet, err := m.ResolveLink(h.e.meta.ID, u.Query().Get("token"))
		t.Logf("resolve %s: %d bytes, magnet=%.60q, err=%v", h.e.meta.ID, len(data), magnet, err)
		break
	}
}
