// Package httpserver assembles the gin router: middleware, the native /api routes and
// the embedded web app with SPA fallback.
package httpserver

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"plenka/internal/buildinfo"
	tslog "plenka/internal/torrserver/log"
)

// Module is a feature package that mounts its own routes.
type Module interface {
	RegisterRoutes(r gin.IRouter)
}

// InfoSource supplies the dynamic parts of GET /api/info.
type InfoSource struct {
	TMDBConfigured func() bool
	IndexerStats   func() (total, enabled int)
}

// New builds the router. web is the embedded file system rooted at the SPA's dist dir.
func New(info InfoSource, web fs.FS, modules ...Module) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	corsCfg := cors.DefaultConfig()
	corsCfg.AllowAllOrigins = true
	corsCfg.AllowPrivateNetwork = true
	corsCfg.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "HEAD", "OPTIONS", "DELETE"}
	corsCfg.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "X-Requested-With", "Accept", "Authorization", "Range"}
	corsCfg.ExposeHeaders = []string{"Content-Length", "Content-Range", "Accept-Ranges"}

	r.Use(tslog.WebLogger(), gin.Recovery(), cors.New(corsCfg))

	r.GET("/api/info", func(c *gin.Context) {
		total, enabled := 0, 0
		if info.IndexerStats != nil {
			total, enabled = info.IndexerStats()
		}
		c.JSON(http.StatusOK, gin.H{
			"service":  "plenka-server",
			"version":  buildinfo.Version,
			"tmdb":     info.TMDBConfigured != nil && info.TMDBConfigured(),
			"torrents": true,
			"indexers": gin.H{"total": total, "enabled": enabled},
		})
	})
	r.GET("/health", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for _, m := range modules {
		m.RegisterRoutes(r)
	}

	spa := newSPA(web)
	r.NoRoute(spa.serve)
	return r
}

// noWebApp is served when the binary was built without the web app (web/dist is empty
// in a fresh clone until `make web` runs).
const noWebApp = `<!doctype html><meta charset="utf-8"><title>Plenka</title>` +
	`<p>The web app is not embedded in this build. Run <code>make build WHAT=server</code> from the repository root.</p>`

type spaHandler struct {
	files fs.FS
	fsrv  http.Handler
	index []byte
}

func newSPA(files fs.FS) *spaHandler {
	index, _ := fs.ReadFile(files, "index.html")
	if len(index) == 0 {
		index = []byte(noWebApp)
	}
	return &spaHandler{files: files, fsrv: http.FileServer(http.FS(files)), index: index}
}

// serve returns static files from the bundle and index.html for client-side routes.
// Unknown API paths stay 404 JSON so clients get a proper error.
func (s *spaHandler) serve(c *gin.Context) {
	p := c.Request.URL.Path
	if strings.HasPrefix(p, "/api/") {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	name := strings.TrimPrefix(path.Clean(p), "/")
	if name != "" && name != "index.html" {
		if st, err := fs.Stat(s.files, name); err == nil && !st.IsDir() {
			if strings.HasPrefix(name, "assets/") {
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			}
			s.fsrv.ServeHTTP(c.Writer, c.Request)
			return
		}
		// A missing file with an extension is a real 404, not a client route.
		if path.Ext(name) != "" {
			c.Status(http.StatusNotFound)
			return
		}
	}
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", s.index)
}
