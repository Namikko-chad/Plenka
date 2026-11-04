// Package appconfig holds the server-wide settings edited from the app's settings screen
// (TMDB key and language, the search timeout, the FlareSolverr address) and serves
// GET/PUT /api/config.
package appconfig

import (
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"plenka/internal/store"
)

const (
	bucket = "config"
	key    = "app"
)

// Config is the stored settings.
type Config struct {
	TMDBApiKey       string `json:"tmdbApiKey"`
	SearchTimeoutSec int    `json:"searchTimeoutSec"`
	// FlareSolverrURL is the optional FlareSolverr service (e.g. http://localhost:8191)
	// used to pass Cloudflare checks of trackers; "" = off.
	FlareSolverrURL string `json:"flareSolverrUrl"`
}

// Service reads and updates Config. Safe for concurrent use.
type Service struct {
	mu      sync.RWMutex
	st      *store.Store
	cfg     Config
	envTMDB string
}

// New loads the stored config, filling defaults on first start.
// envTMDBKey (TMDB_KEY), when non-empty, overrides the stored TMDB key.
func New(st *store.Store, envTMDBKey string) (*Service, error) {
	s := &Service{st: st, envTMDB: strings.TrimSpace(envTMDBKey)}
	if _, err := st.Get(bucket, key, &s.cfg); err != nil {
		return nil, err
	}
	changed := false
	if s.cfg.SearchTimeoutSec <= 0 {
		s.cfg.SearchTimeoutSec = 30
		changed = true
	}
	if changed {
		if err := st.Put(bucket, key, s.cfg); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Get returns a copy of the config with the effective TMDB key.
func (s *Service) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := s.cfg
	if s.envTMDB != "" {
		c.TMDBApiKey = s.envTMDB
	}
	return c
}

type response struct {
	TMDBApiKey       string `json:"tmdbApiKey"`
	TMDBApiKeySet    bool   `json:"tmdbApiKeySet"`
	SearchTimeoutSec int    `json:"searchTimeoutSec"`
	FlareSolverrURL  string `json:"flareSolverrUrl"`
}

type updateRequest struct {
	TMDBApiKey       *string `json:"tmdbApiKey"`
	SearchTimeoutSec *int    `json:"searchTimeoutSec"`
	FlareSolverrURL  *string `json:"flareSolverrUrl"`
}

func (s *Service) toResponse() response {
	c := s.Get()
	return response{
		TMDBApiKeySet:    c.TMDBApiKey != "",
		SearchTimeoutSec: c.SearchTimeoutSec,
		FlareSolverrURL:  c.FlareSolverrURL,
	}
}

// RegisterRoutes mounts GET/PUT /api/config.
func (s *Service) RegisterRoutes(r gin.IRouter) {
	r.GET("/api/config", func(c *gin.Context) { c.JSON(http.StatusOK, s.toResponse()) })
	r.PUT("/api/config", s.update)
}

func (s *Service) update(c *gin.Context) {
	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	s.mu.Lock()
	next := s.cfg
	if req.TMDBApiKey != nil {
		switch v := strings.TrimSpace(*req.TMDBApiKey); v {
		case "":
		case "-":
			next.TMDBApiKey = ""
		default:
			next.TMDBApiKey = v
		}
	}
	if req.SearchTimeoutSec != nil {
		if *req.SearchTimeoutSec < 5 || *req.SearchTimeoutSec > 300 {
			s.mu.Unlock()
			c.JSON(http.StatusBadRequest, gin.H{"error": "searchTimeoutSec must be between 5 and 300"})
			return
		}
		next.SearchTimeoutSec = *req.SearchTimeoutSec
	}
	if req.FlareSolverrURL != nil {
		v := strings.TrimRight(strings.TrimSpace(*req.FlareSolverrURL), "/")
		if u, err := url.Parse(v); v != "" && (err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "") {
			s.mu.Unlock()
			c.JSON(http.StatusBadRequest, gin.H{"error": "flareSolverrUrl must be an http(s) URL"})
			return
		}
		next.FlareSolverrURL = v
	}
	if err := s.st.Put(bucket, key, next); err != nil {
		s.mu.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save settings"})
		return
	}
	s.cfg = next
	s.mu.Unlock()
	c.JSON(http.StatusOK, s.toResponse())
}
