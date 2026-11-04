// Package indexers is the tracker search module: an own implementation of the
// Cardigann definition format (YAML tracker definitions downloaded at runtime), the
// own definitions embedded in the binary, the native Knaben indexer, indexer management and the native search API.
package indexers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"plenka/definitions"
	"plenka/internal/appconfig"
	"plenka/internal/store"
)

const (
	bucketState   = "indexers"
	bucketCookies = "indexers_cookies"
	bucketMeta    = "indexers_meta"

	keyTokenKey = "tokenKey"
)

// Options configures the Manager.
type Options struct {
	Store  *store.Store
	Config func() appconfig.Config
	// DataDir holds the downloaded definitions ($DATA_DIR/definitions).
	DataDir string
	// DefinitionsURL is the .tar.gz to download definitions from.
	DefinitionsURL string
	// HTTPClient downloads the definitions archive (nil: default client).
	HTTPClient *http.Client
}

// indexerState is persisted per indexer id in the "indexers" bucket.
type indexerState struct {
	Enabled       bool              `json:"enabled"`
	Config        map[string]string `json:"config,omitempty"`
	LastError     string            `json:"lastError"`
	LastCheckedAt *time.Time        `json:"lastCheckedAt"`
}

// indexerMeta is what is kept in memory for every known indexer; the full definition
// is parsed again only when the indexer is used.
type indexerMeta struct {
	ID, Name, Description, Language, Type string
	Links, LegacyLinks, Replaces          []string
	Settings                              []Setting
	cats                                  []int    // Newznab categories it maps
	file                                  string   // "" for native indexers
	issues                                []string // unsupported features (indexer cannot run)
}

// engine is implemented by Cardigann indexers and the native Knaben.
type engine interface {
	Search(ctx context.Context, q *Query) ([]*Release, error)
	Download(ctx context.Context, link string) ([]byte, string, error)
}

type entry struct {
	meta *indexerMeta

	mu    sync.Mutex
	state indexerState
	eng   engine
	cg    *cardigann // non-nil for definition-based engines (session persistence)
}

// Manager owns all indexers. Safe for concurrent use.
type Manager struct {
	st       *store.Store
	cfg      func() appconfig.Config
	tokenKey []byte
	src      *defSource
	reloadMu sync.Mutex

	mu       sync.RWMutex
	entries  map[string]*entry
	order    []*entry
	defCount int
}

// New loads the definitions present on disk and the stored indexer state. It does not
// download anything; call StartDefinitionsUpdate for that.
func New(opts Options) (*Manager, error) {
	if opts.Store == nil {
		return nil, errors.New("indexers: store is required")
	}
	if opts.DataDir == "" {
		return nil, errors.New("indexers: data dir is required")
	}
	if opts.Config == nil {
		opts.Config = func() appconfig.Config { return appconfig.Config{SearchTimeoutSec: 30} }
	}
	if opts.DefinitionsURL == "" {
		opts.DefinitionsURL = DefaultDefinitionsURL
	}
	m := &Manager{st: opts.Store, cfg: opts.Config, entries: map[string]*entry{}}
	m.src = &defSource{dataDir: opts.DataDir, url: opts.DefinitionsURL, client: opts.HTTPClient, onDone: m.finishUpdate}
	m.src.recover()
	if err := m.loadTokenKey(); err != nil {
		return nil, err
	}
	m.reload()
	return m, nil
}

func (m *Manager) loadTokenKey() error {
	var k string
	if _, err := m.st.Get(bucketMeta, keyTokenKey, &k); err != nil {
		return err
	}
	if b, err := hex.DecodeString(k); err == nil && len(b) >= 32 {
		m.tokenKey = b
		return nil
	}
	m.tokenKey = make([]byte, 32)
	if _, err := rand.Read(m.tokenKey); err != nil {
		return err
	}
	return m.st.Put(bucketMeta, keyTokenKey, hex.EncodeToString(m.tokenKey))
}

// Own definitions are embedded (package plenka/definitions) and named "builtin:<file>".
const builtinPrefix = "builtin:"

func isBuiltin(file string) bool { return strings.HasPrefix(file, builtinPrefix) }

func readDefinitionFile(file string) ([]byte, error) {
	if name, ok := strings.CutPrefix(file, builtinPrefix); ok {
		return fs.ReadFile(definitions.FS, name)
	}
	return os.ReadFile(file)
}

// definitionFiles lists the own definitions first, then the downloaded ones, so an own
// definition overrides a downloaded one with the same id.
func definitionFiles(downloaded []string) []string {
	own, _ := fs.Glob(definitions.FS, "*.yml")
	sort.Strings(own)
	sort.Strings(downloaded)
	files := make([]string, 0, len(own)+len(downloaded))
	for _, f := range own {
		files = append(files, builtinPrefix+f)
	}
	return append(files, downloaded...)
}

// loadMetas parses and checks every definition file in order. Broken files are
// skipped; for a repeated id the first file wins.
func loadMetas(files []string) (metas []*indexerMeta, skipped int) {
	seen := map[string]string{}
	for _, f := range files {
		data, err := readDefinitionFile(f)
		if err != nil {
			slog.Warn("indexers: cannot read definition", "file", filepath.Base(f), "err", err)
			skipped++
			continue
		}
		d, err := parseDefinition(data)
		if err != nil {
			slog.Warn("indexers: skipping definition", "file", filepath.Base(f), "err", err)
			skipped++
			continue
		}
		if first, dup := seen[d.ID]; dup {
			if isBuiltin(first) {
				slog.Info("indexers: own definition overrides a downloaded one", "id", d.ID, "file", filepath.Base(f))
			} else {
				slog.Warn("indexers: duplicate definition id", "id", d.ID, "file", filepath.Base(f))
				skipped++
			}
			continue
		}
		seen[d.ID] = f
		md := &indexerMeta{
			ID: d.ID, Name: d.Name, Description: d.Description, Language: d.Language, Type: d.Type,
			Links: d.Links, LegacyLinks: d.LegacyLinks, Replaces: d.Replaces, Settings: d.settings(),
			cats: newCatMap(d.Caps).newznab(), file: f, issues: checkDefinition(d),
		}
		if len(md.issues) > 0 {
			slog.Info("indexers: definition uses unsupported features", "id", d.ID, "issues", strings.Join(md.issues, "; "))
		}
		metas = append(metas, md)
	}
	return metas, skipped
}

// reload rebuilds the indexer list from the definitions on disk. Enabled state and
// settings are keyed by id in the store, so they carry over.
func (m *Manager) reload() {
	m.reloadMu.Lock()
	defer m.reloadMu.Unlock()
	t0 := time.Now()
	metas, skipped := loadMetas(definitionFiles(m.src.files()))
	all := append([]*indexerMeta{knabenMeta()}, metas...)
	downloaded := 0
	for _, md := range metas {
		if !isBuiltin(md.file) {
			downloaded++
		}
	}
	entries := map[string]*entry{}
	var order []*entry
	for _, md := range all {
		if entries[md.ID] != nil {
			continue
		}
		e := &entry{meta: md}
		for _, key := range append([]string{md.ID}, md.Replaces...) {
			var s indexerState
			found, err := m.st.Get(bucketState, key, &s)
			if err != nil {
				slog.Warn("indexers: bad stored state", "indexer", key, "err", err)
				continue
			}
			if found {
				e.state = s
				break
			}
		}
		entries[md.ID] = e
		order = append(order, e)
	}
	sort.Slice(order, func(i, j int) bool {
		return strings.ToLower(order[i].meta.Name) < strings.ToLower(order[j].meta.Name)
	})
	m.mu.Lock()
	m.entries, m.order, m.defCount = entries, order, downloaded
	m.mu.Unlock()
	slog.Info("indexers: loaded", "definitions", len(metas), "skipped", skipped, "elapsed", time.Since(t0).Round(time.Millisecond))
}

// Stats returns the number of known and enabled indexers.
func (m *Manager) Stats() (total, enabled int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.order {
		e.mu.Lock()
		if e.state.Enabled {
			enabled++
		}
		e.mu.Unlock()
	}
	return len(m.order), enabled
}

func (m *Manager) get(id string) *entry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.entries[id]
}

func (m *Manager) list() []*entry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]*entry(nil), m.order...)
}

// unsupportedError describes why a definition cannot run.
func unsupportedError(md *indexerMeta) error {
	return fmt.Errorf("unsupported definition: %s", strings.Join(md.issues, "; "))
}

// engine builds (once) the runnable indexer for the entry.
func (m *Manager) engine(e *entry) (engine, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.eng != nil {
		return e.eng, nil
	}
	var cookies map[string]string
	if _, err := m.st.Get(bucketCookies, e.meta.ID, &cookies); err != nil {
		slog.Warn("indexers: cannot read cookies", "indexer", e.meta.ID, "err", err)
	}
	cfg := make(map[string]string, len(e.state.Config))
	for k, v := range e.state.Config {
		cfg[k] = v
	}
	if e.meta.file == "" {
		e.eng = newKnaben()
		return e.eng, nil
	}
	if len(e.meta.issues) > 0 {
		return nil, unsupportedError(e.meta)
	}
	data, err := readDefinitionFile(e.meta.file)
	if err != nil {
		return nil, fmt.Errorf("definition %s: %w", e.meta.ID, err)
	}
	def, err := parseDefinition(data)
	if err != nil {
		return nil, fmt.Errorf("definition %s: %w", e.meta.ID, err)
	}
	if def.ID != e.meta.ID {
		return nil, fmt.Errorf("definition %s changed on disk", e.meta.ID)
	}
	cg, err := newCardigann(def, cfg, cookies)
	if err != nil {
		return nil, err
	}
	cg.sess.solverURL = func() string { return m.cfg().FlareSolverrURL }
	e.eng, e.cg = cg, cg
	return cg, nil
}

// persistSession saves session cookies changed during a request.
func (m *Manager) persistSession(e *entry) {
	e.mu.Lock()
	cg := e.cg
	e.mu.Unlock()
	if cg == nil {
		return
	}
	if c, dirty := cg.cookieSnapshot(); dirty {
		if err := m.st.Put(bucketCookies, e.meta.ID, c); err != nil {
			slog.Warn("indexers: cannot save cookies", "indexer", e.meta.ID, "err", err)
		}
	}
}

// recordResult stores lastError/lastCheckedAt.
func (m *Manager) recordResult(e *entry, err error) {
	now := time.Now().UTC()
	e.mu.Lock()
	e.state.LastCheckedAt = &now
	e.state.LastError = ""
	if err != nil {
		e.state.LastError = err.Error()
	}
	st := e.state
	e.mu.Unlock()
	if perr := m.st.Put(bucketState, e.meta.ID, st); perr != nil {
		slog.Warn("indexers: cannot save state", "indexer", e.meta.ID, "err", perr)
	}
}

// ---- search ----

// hit is one release found by one indexer.
type hit struct {
	rel *Release
	e   *entry
}

// indexerStatus is the per-indexer outcome of a search.
type indexerStatus struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Results   int    `json:"results"`
	Error     string `json:"error"`
	ElapsedMs int64  `json:"elapsedMs"`
}

// targets resolves the indexers for a search: explicit ids, or all enabled ones.
func (m *Manager) targets(ids []string) []*entry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*entry
	if len(ids) > 0 {
		seen := map[string]bool{}
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if e := m.entries[id]; e != nil && !seen[id] {
				seen[id] = true
				out = append(out, e)
			}
		}
		return out
	}
	for _, e := range m.order {
		e.mu.Lock()
		on := e.state.Enabled
		e.mu.Unlock()
		if on {
			out = append(out, e)
		}
	}
	return out
}

func supportsCats(have, want []int) bool {
	if len(want) == 0 || len(have) == 0 {
		return true
	}
	return anyCatMatches(have, want)
}

func (m *Manager) timeout() time.Duration {
	t := time.Duration(m.cfg().SearchTimeoutSec) * time.Second
	if t <= 0 {
		t = 30 * time.Second
	}
	return t
}

// search queries the indexers in parallel with the configured timeout and returns
// hits sorted by seeders (desc) and de-duplicated, plus per-indexer statuses.
func (m *Manager) search(ctx context.Context, q *Query, ids []string) ([]hit, []indexerStatus) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout())
	defer cancel()

	var targets []*entry
	for _, e := range m.targets(ids) {
		if supportsCats(e.meta.cats, q.Categories) {
			targets = append(targets, e)
		}
	}
	statuses := make([]indexerStatus, len(targets))
	results := make([][]*Release, len(targets))
	sem := make(chan struct{}, 16)
	var wg sync.WaitGroup
	for i, e := range targets {
		wg.Add(1)
		go func(i int, e *entry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			start := time.Now()
			rels, err := m.searchOne(ctx, e, q)
			st := indexerStatus{ID: e.meta.ID, Name: e.meta.Name, Status: "ok", Results: len(rels), ElapsedMs: time.Since(start).Milliseconds()}
			if err != nil {
				st.Status, st.Error, st.Results = "error", err.Error(), 0
				rels = nil
			}
			statuses[i], results[i] = st, rels
		}(i, e)
	}
	wg.Wait()

	var hits []hit
	for i, rels := range results {
		for _, r := range rels {
			hits = append(hits, hit{r, targets[i]})
		}
	}
	return dedupeSort(hits), statuses
}

func (m *Manager) searchOne(ctx context.Context, e *entry, q *Query) (rels []*Release, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error: %v", r)
			slog.Error("indexers: panic during search", "indexer", e.meta.ID, "panic", r)
		}
		if ctx.Err() != nil && err != nil {
			err = fmt.Errorf("timed out: %w", err)
		}
		m.persistSession(e)
		m.recordResult(e, err)
	}()
	if err := m.checkConfigured(e); err != nil {
		return nil, err
	}
	eng, err := m.engine(e)
	if err != nil {
		return nil, err
	}
	qc := *q
	return eng.Search(ctx, &qc)
}

func (m *Manager) checkConfigured(e *entry) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if missing := missingSettings(e.meta, e.state.Config); len(missing) > 0 {
		return fmt.Errorf("indexer is not configured: %s required", strings.Join(missing, ", "))
	}
	return nil
}

// dedupeSort sorts by seeders desc and drops duplicates (infohash, else title+size).
func dedupeSort(hits []hit) []hit {
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].rel.Seeders > hits[j].rel.Seeders })
	seen := map[string]bool{}
	out := hits[:0]
	for _, h := range hits {
		key := "h:" + strings.ToLower(h.rel.InfoHash)
		if h.rel.InfoHash == "" {
			key = fmt.Sprintf("t:%s|%d", strings.ToLower(strings.TrimSpace(h.rel.Title)), h.rel.Size)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, h)
	}
	return out
}

// downloadLink returns the magnet or a relative tokenized download URL for a hit.
func (m *Manager) downloadLink(h hit) string {
	if h.rel.Magnet != "" {
		return h.rel.Magnet
	}
	return m.tokenPath(h)
}

// tokenPath is "/api/indexers/{id}/download?token=…" ("" if nothing to resolve).
func (m *Manager) tokenPath(h hit) string {
	link := h.rel.Link
	if link == "" {
		link = h.rel.Details
	}
	if link == "" {
		return ""
	}
	return "/api/indexers/" + h.e.meta.ID + "/download?token=" + signToken(m.tokenKey, h.e.meta.ID, link)
}

// ResolveLink resolves a download token from a search result into .torrent bytes or a
// magnet URI. The torrent engine calls it for "/api/indexers/{id}/download?token=…"
// links, without an HTTP round trip.
func (m *Manager) ResolveLink(indexerID, token string) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	return m.resolveLink(ctx, indexerID, token)
}

var errUnknownIndexer = errors.New("unknown indexer")

func (m *Manager) resolveLink(ctx context.Context, indexerID, token string) (data []byte, magnet string, err error) {
	e := m.get(indexerID)
	if e == nil {
		return nil, "", fmt.Errorf("%w %q", errUnknownIndexer, indexerID)
	}
	link, err := verifyToken(m.tokenKey, indexerID, token)
	if err != nil {
		return nil, "", err
	}
	eng, err := m.engine(e)
	if err != nil {
		return nil, "", err
	}
	defer m.persistSession(e)
	return eng.Download(ctx, link)
}

// test runs a keyword-less search (the configuration check).
func (m *Manager) test(ctx context.Context, e *entry) (int, time.Duration, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, m.timeout())
	defer cancel()
	rels, err := m.searchOne(ctx, e, &Query{})
	if err == nil && len(rels) == 0 {
		err = errors.New("found no results while trying to browse this tracker")
		m.recordResult(e, err)
	}
	return len(rels), time.Since(start), err
}

// ---- configuration ----

// Settings that must be filled for an indexer to work (the format has no "required").
var requiredSettingNames = map[string]bool{
	"username": true, "password": true, "cookie": true, "apikey": true, "passkey": true,
	"rsskey": true, "token": true, "pid": true, "email": true, "uid": true,
}

// missingSettings lists required settings without a value.
func missingSettings(md *indexerMeta, cfg map[string]string) []string {
	var out []string
	for _, s := range md.Settings {
		if strings.HasPrefix(strings.ToLower(s.Type), "info") || s.Default != "" {
			continue
		}
		if strings.EqualFold(s.Type, "password") || requiredSettingNames[strings.ToLower(s.Name)] {
			if strings.TrimSpace(cfg[s.Name]) == "" {
				out = append(out, s.Name)
			}
		}
	}
	return out
}

var errBadSiteLink = errors.New("sitelink must be an http(s) URL")

// update applies a PUT /api/indexers/{id} body.
func (m *Manager) update(e *entry, enabled *bool, cfg map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	next := e.state
	next.Config = map[string]string{}
	for k, v := range e.state.Config {
		next.Config[k] = v
	}
	changed := false
	if cfg != nil {
		for _, s := range e.meta.Settings {
			raw, ok := cfg[s.Name]
			if !ok || strings.HasPrefix(strings.ToLower(s.Type), "info") {
				continue
			}
			val := configString(raw)
			if strings.EqualFold(s.Type, "password") && val == "" {
				continue // keep the stored secret
			}
			if next.Config[s.Name] != val {
				next.Config[s.Name] = val
				changed = true
			}
		}
		if raw, ok := cfg["sitelink"]; ok {
			link := strings.TrimSpace(configString(raw))
			if link != "" && !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
				return errBadSiteLink
			}
			link = withSlash(link)
			if next.Config["sitelink"] != link {
				next.Config["sitelink"] = link
				changed = true
			}
		}
	}
	if enabled != nil {
		next.Enabled = *enabled
	}
	if changed {
		next.LastError = "" // it was about the old settings
	}
	if err := m.st.Put(bucketState, e.meta.ID, next); err != nil {
		return err
	}
	e.state = next
	if changed {
		// rebuild with the new settings and force a fresh login
		e.eng, e.cg = nil, nil
		_ = m.st.Delete(bucketCookies, e.meta.ID)
	}
	return nil
}

func configString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%f", t), "0"), ".")
	case json.Number:
		return t.String()
	case []any:
		parts := make([]string, 0, len(t))
		for _, x := range t {
			parts = append(parts, configString(x))
		}
		return strings.Join(parts, ",")
	}
	return fmt.Sprint(v)
}
