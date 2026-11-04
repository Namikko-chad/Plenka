package indexers

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Query is one search request.
type Query struct {
	Q          string
	Categories []int
	Season     string
	Ep         string
	Limit      int
}

// episodeString is "S01E02" / "S01" for TV queries ("" otherwise).
func (q *Query) episodeString() string {
	s, err := strconv.Atoi(strings.TrimSpace(q.Season))
	if err != nil || s <= 0 {
		return ""
	}
	out := fmt.Sprintf("S%02d", s)
	if e, err := strconv.Atoi(strings.TrimSpace(q.Ep)); err == nil && e > 0 {
		out += fmt.Sprintf("E%02d", e)
	}
	return out
}

// Release is one search result.
type Release struct {
	Title       string
	Details     string
	Link        string // .torrent URL (or details page when the definition has a download block)
	Magnet      string
	InfoHash    string // lower-case hex
	Size        int64  // bytes, 0 = unknown
	Seeders     int64
	Leechers    int64
	Grabs       int64
	Category    []int
	PublishDate time.Time
	Description string
	Poster      string
	IMDBID      string

	DownloadVolumeFactor float64
	UploadVolumeFactor   float64
}

var (
	sizeRe = regexp.MustCompile(`(?i)([0-9][0-9.,\s]*)\s*([kmgtp]?i?b|[kmgtp]|bytes?|octets?|[кмгт]?б)?`)
	numRe  = regexp.MustCompile(`-?[0-9][0-9.,\s]*`)
)

// parseSize reads "1.5 GB", "1,024.4MB", "1.024,4 MiB", "700 МБ" or plain bytes.
func parseSize(s string) int64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, " ", " "))
	if s == "" {
		return 0
	}
	m := sizeRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	n, ok := parseLooseFloat(m[1])
	if !ok {
		return 0
	}
	unit := strings.ToLower(m[2])
	mult := 1.0
	switch {
	case unit == "" || strings.HasPrefix(unit, "byte") || strings.HasPrefix(unit, "octet") || unit == "b" || unit == "б":
	case strings.HasPrefix(unit, "k") || strings.HasPrefix(unit, "к"):
		mult = 1 << 10
	case strings.HasPrefix(unit, "m") || strings.HasPrefix(unit, "м"):
		mult = 1 << 20
	case strings.HasPrefix(unit, "g") || strings.HasPrefix(unit, "г"):
		mult = 1 << 30
	case strings.HasPrefix(unit, "t") || strings.HasPrefix(unit, "т"):
		mult = 1 << 40
	case strings.HasPrefix(unit, "p"):
		mult = 1 << 50
	}
	v := n * mult
	if v > math.MaxInt64/2 {
		return 0
	}
	return int64(math.Round(v))
}

// parseLooseFloat accepts both decimal conventions: "1,024.5", "1.024,5", "1 024,5".
func parseLooseFloat(s string) (float64, bool) {
	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, strings.TrimSpace(s))
	s = strings.TrimRight(s, ".,")
	if s == "" {
		return 0, false
	}
	lastDot, lastComma := strings.LastIndexByte(s, '.'), strings.LastIndexByte(s, ',')
	switch {
	case lastDot >= 0 && lastComma >= 0:
		if lastComma > lastDot { // 1.024,5
			s = strings.ReplaceAll(s, ".", "")
			s = strings.Replace(s, ",", ".", 1)
		} else { // 1,024.5
			s = strings.ReplaceAll(s, ",", "")
		}
	case lastComma >= 0:
		if strings.Count(s, ",") > 1 || len(s)-lastComma-1 == 3 {
			s = strings.ReplaceAll(s, ",", "")
		} else {
			s = strings.Replace(s, ",", ".", 1)
		}
	case lastDot >= 0 && strings.Count(s, ".") > 1:
		s = strings.ReplaceAll(s, ".", "")
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// parseCount reads the first number of s ("1,234", "12 seeders"); 0 when none.
func parseCount(s string) int64 {
	m := numRe.FindString(strings.ReplaceAll(s, " ", " "))
	if m == "" {
		return 0
	}
	f, ok := parseLooseFloat(m)
	if !ok {
		return 0
	}
	return int64(f)
}

// parseFactor reads a volume factor; def when empty or not a number.
func parseFactor(s string, def float64) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	if f, ok := parseLooseFloat(s); ok {
		return f
	}
	return def
}

var hashRe = regexp.MustCompile(`(?i)\b([0-9a-f]{40})\b`)

// normalizeInfoHash extracts a 40-hex infohash (from a bare hash or a magnet).
func normalizeInfoHash(s string) string {
	if m := hashRe.FindStringSubmatch(s); m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

// magnetHash returns the infohash of a magnet URI.
func magnetHash(magnet string) string {
	u, err := url.Parse(magnet)
	if err != nil {
		return ""
	}
	for _, xt := range u.Query()["xt"] {
		if h := strings.TrimPrefix(strings.ToLower(xt), "urn:btih:"); h != strings.ToLower(xt) {
			return normalizeInfoHash(h)
		}
	}
	return ""
}

// Public trackers appended to magnets built from a bare infohash.
var publicTrackers = []string{
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://open.demonii.com:1337/announce",
	"udp://open.stealth.si:80/announce",
	"udp://tracker.torrent.eu.org:451/announce",
	"udp://exodus.desync.com:6969/announce",
	"udp://tracker.openbittorrent.com:6969/announce",
}

func publicMagnet(hash, title string) string {
	var b strings.Builder
	b.WriteString("magnet:?xt=urn:btih:")
	b.WriteString(strings.ToLower(hash))
	if title != "" {
		b.WriteString("&dn=")
		b.WriteString(url.QueryEscape(title))
	}
	for _, tr := range publicTrackers {
		b.WriteString("&tr=")
		b.WriteString(url.QueryEscape(tr))
	}
	return b.String()
}

var spaceRun = regexp.MustCompile(`\s+`)

func cleanTitle(s string) string {
	return strings.TrimSpace(spaceRun.ReplaceAllString(s, " "))
}

var imdbRe = regexp.MustCompile(`(?i)(?:tt)?(\d{4,9})`)

func normalizeIMDB(s string) string {
	if m := imdbRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		if n > 0 {
			return fmt.Sprintf("tt%07d", n)
		}
	}
	return ""
}
