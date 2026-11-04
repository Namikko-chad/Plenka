// Package history keeps the watch history shared by the web app and the mobile app
// (/api/history). One entry per torrent file, newest first, capped at maxEntries.
package history

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"plenka/internal/store"
)

const (
	bucket     = "history"
	maxEntries = 300
)

// Media is the TMDB title an entry belongs to.
type Media struct {
	ID            int     `json:"id"`
	Type          string  `json:"type"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"originalTitle"`
	Year          *int    `json:"year"`
	PosterPath    *string `json:"posterPath"`
	BackdropPath  *string `json:"backdropPath"`
	Overview      string  `json:"overview"`
	Rating        float64 `json:"rating"`
}

// Entry is the playback position of one file.
type Entry struct {
	Hash         string    `json:"hash"`
	FileID       int       `json:"fileId"`
	FileName     string    `json:"fileName"`
	TorrentTitle string    `json:"torrentTitle"`
	Media        *Media    `json:"media"`
	PositionMs   int64     `json:"positionMs"`
	DurationMs   int64     `json:"durationMs"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Service stores entries in the history bucket.
type Service struct {
	mu sync.Mutex
	st *store.Store
}

// New creates the service.
func New(st *store.Store) *Service { return &Service{st: st} }

func entryKey(hash string, fileID int) string { return hash + ":" + strconv.Itoa(fileID) }

// List returns up to limit entries, newest first.
func (s *Service) List(limit int) ([]Entry, error) {
	var out []Entry
	err := s.st.ForEach(bucket, func(_ string, raw []byte) error {
		var e Entry
		if json.Unmarshal(raw, &e) == nil {
			out = append(out, e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	if out == nil {
		out = []Entry{}
	}
	return out, nil
}

// Save upserts e (UpdatedAt is set to now) and trims the oldest entries over the cap.
func (s *Service) Save(e Entry) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e.Hash = strings.ToLower(e.Hash)
	e.UpdatedAt = time.Now().UTC()
	if err := s.st.Put(bucket, entryKey(e.Hash, e.FileID), e); err != nil {
		return Entry{}, err
	}
	all, err := s.List(0)
	if err != nil {
		return e, nil
	}
	for _, old := range all[min(len(all), maxEntries):] {
		_ = s.st.Delete(bucket, entryKey(old.Hash, old.FileID))
	}
	return e, nil
}

// Delete removes one file's entry, or every entry of the torrent when fileID < 0.
func (s *Service) Delete(hash string, fileID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hash = strings.ToLower(hash)
	if fileID >= 0 {
		return s.st.Delete(bucket, entryKey(hash, fileID))
	}
	var keys []string
	_ = s.st.ForEach(bucket, func(k string, _ []byte) error {
		if strings.HasPrefix(k, hash+":") {
			keys = append(keys, k)
		}
		return nil
	})
	for _, k := range keys {
		if err := s.st.Delete(bucket, k); err != nil {
			return err
		}
	}
	return nil
}

// RegisterRoutes mounts /api/history.
func (s *Service) RegisterRoutes(r gin.IRouter) {
	r.GET("/api/history", s.list)
	r.PUT("/api/history", s.save)
	r.DELETE("/api/history/:hash", s.remove)
	r.DELETE("/api/history/:hash/:fileId", s.remove)
}

func (s *Service) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(maxEntries)))
	out, err := s.List(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read history"})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Service) save(c *gin.Context) {
	var e Entry
	if err := c.ShouldBindJSON(&e); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	if len(e.Hash) != 40 || e.FileID < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hash must be a 40-character infohash and fileId >= 0"})
		return
	}
	if e.PositionMs < 0 || e.DurationMs < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "positionMs and durationMs must be >= 0"})
		return
	}
	saved, err := s.Save(e)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save history"})
		return
	}
	c.JSON(http.StatusOK, saved)
}

func (s *Service) remove(c *gin.Context) {
	fileID := -1
	if v := c.Param("fileId"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "fileId must be a non-negative integer"})
			return
		}
		fileID = n
	}
	if err := s.Delete(c.Param("hash"), fileID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete history"})
		return
	}
	c.Status(http.StatusNoContent)
}
