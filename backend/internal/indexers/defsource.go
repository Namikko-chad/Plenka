package indexers

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// The server does not ship tracker definitions: they are downloaded as a .tar.gz
// (by default the Jackett repository snapshot) and every "*.yml" whose parent
// directory is named "Definitions" is kept in $DATA_DIR/definitions/.

// DefaultDefinitionsURL is used when DEFINITIONS_URL is not set.
const DefaultDefinitionsURL = "https://codeload.github.com/Jackett/Jackett/tar.gz/refs/heads/master"

const (
	defsDirName       = "definitions"
	maxDefinitionSize = 2 << 20
	maxDefinitions    = 5000
	maxArchiveBytes   = 256 << 20 // uncompressed bytes read from the archive
	downloadTimeout   = 2 * time.Minute
	keyDefsState      = "definitions"
)

// defsState is persisted in the indexers_meta bucket.
type defsState struct {
	UpdatedAt *time.Time `json:"updatedAt"`
	LastError string     `json:"lastError"`
}

// DefinitionsStatus is the GET /api/definitions payload.
type DefinitionsStatus struct {
	Source    string     `json:"source"`
	Count     int        `json:"count"`
	UpdatedAt *time.Time `json:"updatedAt"`
	Updating  bool       `json:"updating"`
	LastError string     `json:"lastError"`
}

type updateCall struct {
	done chan struct{}
	err  error
}

type defSource struct {
	dataDir string
	url     string
	client  *http.Client
	onDone  func(error) // called once per run, before waiters are released

	mu      sync.Mutex
	running *updateCall
}

func (s *defSource) dir() string { return filepath.Join(s.dataDir, defsDirName) }

// recover finishes an interrupted swap (the new set is moved in after the old one
// was moved away).
func (s *defSource) recover() {
	dir, old := s.dir(), s.dir()+".old"
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if _, err := os.Stat(old); err == nil {
			_ = os.Rename(old, dir)
		}
	}
	_ = os.RemoveAll(old)
}

// files lists the definition files currently on disk.
func (s *defSource) files() []string {
	files, _ := filepath.Glob(filepath.Join(s.dir(), "*.yml"))
	return files
}

// update downloads and installs a new definition set; concurrent callers share one run.
// The download itself is not bound to ctx (a client hanging up must not cancel it).
func (s *defSource) update(ctx context.Context) error {
	s.mu.Lock()
	call := s.running
	if call == nil {
		call = &updateCall{done: make(chan struct{})}
		s.running = call
		go func() {
			call.err = s.download()
			if s.onDone != nil {
				s.onDone(call.err)
			}
			s.mu.Lock()
			s.running = nil
			s.mu.Unlock()
			close(call.done)
		}()
	}
	s.mu.Unlock()
	select {
	case <-call.done:
		return call.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *defSource) updating() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running != nil
}

func (s *defSource) download() error {
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return fmt.Errorf("bad DEFINITIONS_URL: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	client := s.client
	if client == nil {
		client = &http.Client{Transport: sharedTransport}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(s.dataDir, ".definitions-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	n, err := extractDefinitions(resp.Body, tmp)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("the archive contains no definitions (no *.yml in a Definitions directory)")
	}
	return s.install(tmp)
}

// install swaps the extracted directory in place of the current set.
func (s *defSource) install(tmp string) error {
	dir, old := s.dir(), s.dir()+".old"
	_ = os.RemoveAll(old)
	if _, err := os.Stat(dir); err == nil {
		if err := os.Rename(dir, old); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp, dir); err != nil {
		_ = os.Rename(old, dir)
		return err
	}
	_ = os.RemoveAll(old)
	return nil
}

// extractDefinitions streams a .tar.gz and writes matching files into dst.
func extractDefinitions(r io.Reader, dst string) (int, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return 0, fmt.Errorf("not a gzip archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(io.LimitReader(gz, maxArchiveBytes))
	count := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, fmt.Errorf("bad archive: %w", err)
		}
		if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeRegA {
			continue
		}
		name := path.Clean(strings.TrimPrefix(h.Name, "./"))
		base := path.Base(name)
		if path.Base(path.Dir(name)) != "Definitions" || !strings.HasSuffix(strings.ToLower(base), ".yml") {
			continue
		}
		if strings.HasPrefix(base, ".") || strings.ContainsAny(base, `/\`) || h.Size > maxDefinitionSize {
			continue
		}
		if count >= maxDefinitions {
			return count, errors.New("the archive contains too many definitions")
		}
		f, err := os.OpenFile(filepath.Join(dst, base), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return count, err
		}
		_, cerr := io.Copy(f, io.LimitReader(tr, maxDefinitionSize))
		if err := f.Close(); cerr == nil {
			cerr = err
		}
		if cerr != nil {
			return count, cerr
		}
		count++
	}
	return count, nil
}

// ---- manager integration ----

// DefinitionsStatus reports the loaded set and the update state.
func (m *Manager) DefinitionsStatus() DefinitionsStatus {
	var st defsState
	if _, err := m.st.Get(bucketMeta, keyDefsState, &st); err != nil {
		slog.Warn("indexers: cannot read definitions state", "err", err)
	}
	m.mu.RLock()
	count := m.defCount
	m.mu.RUnlock()
	return DefinitionsStatus{
		Source: m.src.url, Count: count, UpdatedAt: st.UpdatedAt,
		Updating: m.src.updating(), LastError: st.LastError,
	}
}

// UpdateDefinitions downloads the definitions (joining a running update), reloads the
// indexers on success and records the outcome.
// If ctx ends first the update keeps running in the background.
func (m *Manager) UpdateDefinitions(ctx context.Context) error {
	return m.src.update(ctx)
}

func (m *Manager) finishUpdate(err error) {
	var st defsState
	_, _ = m.st.Get(bucketMeta, keyDefsState, &st)
	if err != nil {
		st.LastError = err.Error()
		slog.Warn("indexers: definitions update failed", "err", err)
	} else {
		now := time.Now().UTC()
		st.UpdatedAt, st.LastError = &now, ""
		m.reload()
	}
	if perr := m.st.Put(bucketMeta, keyDefsState, st); perr != nil {
		slog.Warn("indexers: cannot save definitions state", "err", perr)
	}
}

// StartDefinitionsUpdate runs an update in the background (used at startup).
func (m *Manager) StartDefinitionsUpdate() {
	go func() {
		if err := m.UpdateDefinitions(context.Background()); err == nil {
			slog.Info("indexers: definitions updated", "count", m.DefinitionsStatus().Count)
		}
	}()
}
