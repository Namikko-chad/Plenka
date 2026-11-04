// Package tmdb proxies the TMDB API (/api/tmdb/*) and its image CDN (/img/*) so that
// clients never see the API key and images load from the server's origin.
package tmdb

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"plenka/internal/appconfig"
)

const (
	apiBase   = "https://api.themoviedb.org"
	imageBase = "https://image.tmdb.org"
	cacheTTL  = 10 * time.Minute
	cacheMax  = 500
)

// Proxy serves the TMDB API and image proxies.
type Proxy struct {
	cfg     func() appconfig.Config
	client  *http.Client
	apiBase string
	imgBase string
	mu      sync.Mutex
	cache   map[string]cached
}

type cached struct {
	body    []byte
	status  int
	expires time.Time
}

// New creates a proxy reading the key and language from cfg on every request.
func New(cfg func() appconfig.Config) *Proxy {
	return &Proxy{
		cfg:     cfg,
		client:  &http.Client{Timeout: 20 * time.Second},
		apiBase: apiBase,
		imgBase: imageBase,
		cache:   map[string]cached{},
	}
}

// RegisterRoutes mounts /api/tmdb/*path and /img/*path.
func (p *Proxy) RegisterRoutes(r gin.IRouter) {
	r.GET("/api/tmdb/*path", p.api)
	r.GET("/img/*path", p.image)
}

// Configured reports whether a TMDB key is set.
func (p *Proxy) Configured() bool { return p.cfg().TMDBApiKey != "" }

func (p *Proxy) api(c *gin.Context) {
	cfg := p.cfg()
	if cfg.TMDBApiKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "TMDB key is not configured"})
		return
	}
	path := c.Param("path")
	if !strings.HasPrefix(path, "/3/") && !strings.HasPrefix(path, "/4/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path must start with /3/ or /4/"})
		return
	}
	q := c.Request.URL.Query()
	q.Del("api_key")
	// `language` comes from the client: each one asks in its own interface language.
	// v4 read tokens are JWTs (long, dotted); v3 keys are 32 hex chars.
	bearer := strings.Count(cfg.TMDBApiKey, ".") == 2
	cacheKey := path + "?" + q.Encode()
	if !bearer {
		q.Set("api_key", cfg.TMDBApiKey)
	}

	if hit, ok := p.fromCache(cacheKey); ok {
		c.Data(hit.status, "application/json; charset=utf-8", hit.body)
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, p.apiBase+path+"?"+q.Encode(), nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid TMDB path"})
		return
	}
	req.Header.Set("Accept", "application/json")
	if bearer {
		req.Header.Set("Authorization", "Bearer "+cfg.TMDBApiKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "TMDB is unreachable"})
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "TMDB response was interrupted"})
		return
	}
	if resp.StatusCode == http.StatusOK {
		p.toCache(cacheKey, cached{body: body, status: resp.StatusCode, expires: time.Now().Add(cacheTTL)})
	}
	c.Data(resp.StatusCode, "application/json; charset=utf-8", body)
}

func (p *Proxy) image(c *gin.Context) {
	path := c.Param("path")
	if !strings.HasPrefix(path, "/t/p/") || strings.Contains(path, "..") {
		c.Status(http.StatusNotFound)
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, p.imgBase+(&url.URL{Path: path}).EscapedPath(), nil)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	resp, err := p.client.Do(req)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.Status(resp.StatusCode)
		return
	}
	h := c.Writer.Header()
	h.Set("Cache-Control", "public, max-age=2592000, immutable")
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		h.Set("Content-Type", ct)
	}
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		h.Set("Content-Length", cl)
	}
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, resp.Body)
}

func (p *Proxy) fromCache(key string) (cached, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.cache[key]
	if !ok || time.Now().After(v.expires) {
		delete(p.cache, key)
		return cached{}, false
	}
	return v, true
}

func (p *Proxy) toCache(key string, v cached) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.cache) >= cacheMax {
		now := time.Now()
		for k, e := range p.cache {
			if now.After(e.expires) {
				delete(p.cache, k)
			}
		}
		if len(p.cache) >= cacheMax {
			p.cache = map[string]cached{}
		}
	}
	p.cache[key] = v
}
