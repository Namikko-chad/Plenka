package indexers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts the native search, indexer and definitions routes.
func (m *Manager) RegisterRoutes(r gin.IRouter) {
	r.GET("/api/search", m.handleSearch)
	r.GET("/api/indexers", m.handleList)
	r.GET("/api/indexers/:id", m.handleGet)
	r.PUT("/api/indexers/:id", m.handlePut)
	r.POST("/api/indexers/:id/test", m.handleTest)
	r.GET("/api/indexers/:id/download", m.handleDownload)
	r.GET("/api/definitions", m.handleDefinitions)
	r.POST("/api/definitions/update", m.handleDefinitionsUpdate)
}

func apiError(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

// ---- search ----

type resultJSON struct {
	Title        string     `json:"title"`
	Tracker      string     `json:"tracker"`
	TrackerID    string     `json:"trackerId"`
	Size         int64      `json:"size"`
	Seeders      int64      `json:"seeders"`
	Peers        int64      `json:"peers"`
	Link         string     `json:"link"`
	Magnet       string     `json:"magnet"`
	InfoHash     string     `json:"infoHash"`
	Details      string     `json:"details"`
	PublishDate  *time.Time `json:"publishDate"`
	Category     []int      `json:"category"`
	CategoryDesc string     `json:"categoryDesc"`
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (m *Manager) handleSearch(c *gin.Context) {
	var cats []int
	for _, p := range splitList(c.Query("cat")) {
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 {
			apiError(c, http.StatusBadRequest, "cat must be a comma-separated list of category ids")
			return
		}
		cats = append(cats, n)
	}
	q := &Query{Q: strings.TrimSpace(c.Query("q")), Categories: cats}
	hits, statuses := m.search(c.Request.Context(), q, splitList(c.Query("indexers")))
	results := make([]resultJSON, 0, len(hits))
	for _, h := range hits {
		r := h.rel
		cat := r.Category
		if cat == nil {
			cat = []int{}
		}
		var pd *time.Time
		if !r.PublishDate.IsZero() {
			t := r.PublishDate.UTC()
			pd = &t
		}
		results = append(results, resultJSON{
			Title: r.Title, Tracker: h.e.meta.Name, TrackerID: h.e.meta.ID, Size: r.Size,
			Seeders: r.Seeders, Peers: r.Leechers, Link: m.downloadLink(h), Magnet: r.Magnet,
			InfoHash: r.InfoHash, Details: r.Details, PublishDate: pd, Category: cat,
			CategoryDesc: categoryDesc(cat),
		})
	}
	if statuses == nil {
		statuses = []indexerStatus{}
	}
	c.JSON(http.StatusOK, gin.H{"results": results, "indexers": statuses})
}

// categoryDesc names the most specific category of a result.
func categoryDesc(cats []int) string {
	best := 0
	for _, id := range cats {
		if best == 0 || (id%1000 != 0 && best%1000 == 0) {
			best = id
		}
	}
	return newznabByID[best]
}

// ---- indexers ----

type indexerJSON struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	Language      string     `json:"language"`
	Type          string     `json:"type"`
	Links         []string   `json:"links"`
	Enabled       bool       `json:"enabled"`
	Configured    bool       `json:"configured"`
	LastError     string     `json:"lastError"`
	LastCheckedAt *time.Time `json:"lastCheckedAt"`
}

type optionJSON struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type settingJSON struct {
	Name    string       `json:"name"`
	Type    string       `json:"type"`
	Label   string       `json:"label"`
	Default string       `json:"default"`
	Options []optionJSON `json:"options"`
}

type categoryJSON struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type indexerDetailJSON struct {
	indexerJSON
	Settings   []settingJSON     `json:"settings"`
	Config     map[string]string `json:"config"`
	Categories []categoryJSON    `json:"categories"`
}

func (m *Manager) itemJSON(e *entry) indexerJSON {
	e.mu.Lock()
	st := e.state
	e.mu.Unlock()
	links := e.meta.Links
	if links == nil {
		links = []string{}
	}
	lastErr := st.LastError
	if lastErr == "" && len(e.meta.issues) > 0 {
		lastErr = unsupportedError(e.meta).Error()
	}
	return indexerJSON{
		ID: e.meta.ID, Name: e.meta.Name, Description: e.meta.Description, Language: e.meta.Language,
		Type: e.meta.Type, Links: links, Enabled: st.Enabled,
		Configured: len(missingSettings(e.meta, st.Config)) == 0, LastError: lastErr,
		LastCheckedAt: st.LastCheckedAt,
	}
}

func (m *Manager) detailJSON(e *entry) indexerDetailJSON {
	out := indexerDetailJSON{indexerJSON: m.itemJSON(e), Settings: []settingJSON{}, Config: map[string]string{}, Categories: []categoryJSON{}}
	e.mu.Lock()
	cfg := e.state.Config
	for _, s := range e.meta.Settings {
		sj := settingJSON{Name: s.Name, Type: s.Type, Label: s.Label, Default: s.Default}
		if strings.EqualFold(s.Type, "multi-select") && sj.Default == "" {
			sj.Default = strings.Join(s.Defaults, ",")
		}
		if strings.HasPrefix(strings.ToLower(s.Type), "info") && sj.Label == "" {
			sj.Label = s.Name
		}
		for _, o := range s.Options {
			sj.Options = append(sj.Options, optionJSON{Value: o.Key, Label: o.Value})
		}
		out.Settings = append(out.Settings, sj)
		if strings.HasPrefix(strings.ToLower(s.Type), "info") {
			continue
		}
		v, ok := cfg[s.Name]
		if strings.EqualFold(s.Type, "password") {
			if ok && v != "" {
				out.Config[s.Name+"__set"] = "1"
			}
			out.Config[s.Name] = ""
			continue
		}
		if ok {
			out.Config[s.Name] = v
		}
	}
	e.mu.Unlock()
	if len(e.meta.Links) > 0 {
		def := &Definition{Links: e.meta.Links, LegacyLinks: e.meta.LegacyLinks}
		out.Config["sitelink"] = siteLink(def, cfg)
	}
	for _, id := range e.meta.cats {
		out.Categories = append(out.Categories, categoryJSON{ID: id, Name: newznabByID[id]})
	}
	return out
}

func (m *Manager) handleList(c *gin.Context) {
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	typ := strings.ToLower(strings.TrimSpace(c.Query("type")))
	var enabled *bool
	if s := strings.TrimSpace(c.Query("enabled")); s != "" {
		b, err := strconv.ParseBool(s)
		if err != nil {
			apiError(c, http.StatusBadRequest, "enabled must be true or false")
			return
		}
		enabled = &b
	}
	out := []indexerJSON{}
	for _, e := range m.list() {
		it := m.itemJSON(e)
		if typ != "" && it.Type != typ {
			continue
		}
		if enabled != nil && it.Enabled != *enabled {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(it.ID+"\n"+it.Name+"\n"+it.Description), q) {
			continue
		}
		out = append(out, it)
	}
	c.JSON(http.StatusOK, out)
}

func (m *Manager) entryOr404(c *gin.Context) *entry {
	e := m.get(c.Param("id"))
	if e == nil {
		apiError(c, http.StatusNotFound, "unknown indexer")
	}
	return e
}

func (m *Manager) handleGet(c *gin.Context) {
	if e := m.entryOr404(c); e != nil {
		c.JSON(http.StatusOK, m.detailJSON(e))
	}
}

func (m *Manager) handlePut(c *gin.Context) {
	e := m.entryOr404(c)
	if e == nil {
		return
	}
	var body struct {
		Enabled *bool          `json:"enabled"`
		Config  map[string]any `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		apiError(c, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := m.update(e, body.Enabled, body.Config); err != nil {
		if errors.Is(err, errBadSiteLink) {
			apiError(c, http.StatusBadRequest, err.Error())
			return
		}
		apiError(c, http.StatusInternalServerError, "could not save the indexer settings")
		return
	}
	c.JSON(http.StatusOK, m.detailJSON(e))
}

func (m *Manager) handleTest(c *gin.Context) {
	e := m.entryOr404(c)
	if e == nil {
		return
	}
	n, elapsed, err := m.test(c.Request.Context(), e)
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	c.JSON(http.StatusOK, gin.H{"ok": err == nil, "results": n, "error": msg, "elapsedMs": elapsed.Milliseconds()})
}

func (m *Manager) handleDownload(c *gin.Context) {
	id := c.Param("id")
	if m.get(id) == nil {
		apiError(c, http.StatusNotFound, "unknown indexer")
		return
	}
	data, magnet, err := m.resolveLink(c.Request.Context(), id, c.Query("token"))
	switch {
	case errors.Is(err, errBadToken):
		apiError(c, http.StatusBadRequest, err.Error())
	case err != nil:
		apiError(c, http.StatusBadGateway, err.Error())
	case magnet != "":
		c.Redirect(http.StatusFound, magnet)
	default:
		c.Header("Content-Disposition", `attachment; filename="`+id+`.torrent"`)
		c.Data(http.StatusOK, "application/x-bittorrent", data)
	}
}

// ---- definitions ----

func (m *Manager) handleDefinitions(c *gin.Context) {
	c.JSON(http.StatusOK, m.DefinitionsStatus())
}

func (m *Manager) handleDefinitionsUpdate(c *gin.Context) {
	if err := m.UpdateDefinitions(c.Request.Context()); err != nil {
		apiError(c, http.StatusBadGateway, err.Error())
		return
	}
	c.JSON(http.StatusOK, m.DefinitionsStatus())
}
