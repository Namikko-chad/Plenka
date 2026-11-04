package indexers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Knaben is a public meta-search API with a JSON interface; it is implemented natively
// (no definition), so it works even before any definitions are downloaded.

var knabenAPI = "https://api.knaben.org/v1"

var knabenCats = []CategoryMapping{
	{ID: "1000000", Cat: "Audio", Desc: "Audio"}, {ID: "1001000", Cat: "Audio/MP3", Desc: "MP3"}, {ID: "1002000", Cat: "Audio/Lossless", Desc: "Lossless"},
	{ID: "1003000", Cat: "Audio/Audiobook", Desc: "Audiobook"}, {ID: "1004000", Cat: "Audio/Video", Desc: "Audio Video"}, {ID: "1005000", Cat: "Audio/Other", Desc: "Radio"},
	{ID: "1006000", Cat: "Audio/Other", Desc: "Audio Other"}, {ID: "2000000", Cat: "TV", Desc: "TV"}, {ID: "2001000", Cat: "TV/HD", Desc: "TV HD"},
	{ID: "2002000", Cat: "TV/SD", Desc: "TV SD"}, {ID: "2003000", Cat: "TV/UHD", Desc: "TV UHD"}, {ID: "2004000", Cat: "TV/Documentary", Desc: "Documentary"},
	{ID: "2005000", Cat: "TV/Foreign", Desc: "TV Foreign"}, {ID: "2006000", Cat: "TV/Sport", Desc: "Sport"}, {ID: "2007000", Cat: "TV/Other", Desc: "Cartoon"},
	{ID: "2008000", Cat: "TV/Other", Desc: "TV Other"}, {ID: "3000000", Cat: "Movies", Desc: "Movies"}, {ID: "3001000", Cat: "Movies/HD", Desc: "Movies HD"},
	{ID: "3002000", Cat: "Movies/SD", Desc: "Movies SD"}, {ID: "3003000", Cat: "Movies/UHD", Desc: "Movies UHD"}, {ID: "3004000", Cat: "Movies/DVD", Desc: "Movies DVD"},
	{ID: "3005000", Cat: "Movies/Foreign", Desc: "Movies Foreign"}, {ID: "3006000", Cat: "Movies/Foreign", Desc: "Movies Bollywood"}, {ID: "3007000", Cat: "Movies/3D", Desc: "Movies 3D"},
	{ID: "3008000", Cat: "Movies/Other", Desc: "Movies Other"}, {ID: "4000000", Cat: "PC", Desc: "PC"}, {ID: "4001000", Cat: "PC/Games", Desc: "Games"},
	{ID: "4002000", Cat: "PC/0day", Desc: "Software"}, {ID: "4003000", Cat: "PC/Mac", Desc: "Mac"}, {ID: "4004000", Cat: "PC/ISO", Desc: "Unix"},
	{ID: "5000000", Cat: "XXX", Desc: "XXX"}, {ID: "5001000", Cat: "XXX/x264", Desc: "XXX Video"}, {ID: "5002000", Cat: "XXX/ImageSet", Desc: "XXX ImageSet"},
	{ID: "5003000", Cat: "XXX/Other", Desc: "XXX Games"}, {ID: "5004000", Cat: "XXX/Other", Desc: "XXX Hentai"}, {ID: "5005000", Cat: "XXX/Other", Desc: "XXX Other"},
	{ID: "6000000", Cat: "TV/Anime", Desc: "Anime"}, {ID: "6001000", Cat: "TV/Anime", Desc: "Anime Subbed"}, {ID: "6002000", Cat: "TV/Anime", Desc: "Anime Dubbed"},
	{ID: "6003000", Cat: "TV/Anime", Desc: "Anime Dual audio"}, {ID: "6004000", Cat: "TV/Anime", Desc: "Anime Raw"}, {ID: "6005000", Cat: "Audio/Video", Desc: "Music Video"},
	{ID: "6006000", Cat: "Books/Other", Desc: "Literature"}, {ID: "6007000", Cat: "Audio/Other", Desc: "Music"}, {ID: "6008000", Cat: "TV/Anime", Desc: "Anime non-english translated"},
	{ID: "7000000", Cat: "Console", Desc: "Console"}, {ID: "7001000", Cat: "Console/PS4", Desc: "PS4"}, {ID: "7002000", Cat: "Console/PS3", Desc: "PS3"},
	{ID: "7003000", Cat: "Console/PS3", Desc: "PS2"}, {ID: "7004000", Cat: "Console/PS3", Desc: "PS1"}, {ID: "7005000", Cat: "Console/PS Vita", Desc: "PS Vita"},
	{ID: "7006000", Cat: "Console/PSP", Desc: "PSP"}, {ID: "7007000", Cat: "Console/XBox 360", Desc: "Xbox 360"}, {ID: "7008000", Cat: "Console/XBox", Desc: "Xbox"},
	{ID: "7009000", Cat: "Console/NDS", Desc: "Switch"}, {ID: "7010000", Cat: "Console/NDS", Desc: "NDS"}, {ID: "7011000", Cat: "Console/Wii", Desc: "Wii"},
	{ID: "7012000", Cat: "Console/WiiU", Desc: "WiiU"}, {ID: "7013000", Cat: "Console/3DS", Desc: "3DS"}, {ID: "7014000", Cat: "Console/Wii", Desc: "GameCube"},
	{ID: "7015000", Cat: "Console/Other", Desc: "Other"}, {ID: "8000000", Cat: "PC/Mobile-Other", Desc: "Mobile"}, {ID: "8001000", Cat: "PC/Mobile-Android", Desc: "Android"},
	{ID: "8002000", Cat: "PC/Mobile-iOS", Desc: "IOS"}, {ID: "8003000", Cat: "PC/Mobile-Other", Desc: "PC Other"}, {ID: "9000000", Cat: "Books", Desc: "Books"},
	{ID: "9001000", Cat: "Books/EBook", Desc: "EBooks"}, {ID: "9002000", Cat: "Books/Comics", Desc: "Comics"}, {ID: "9003000", Cat: "Books/Mags", Desc: "Magazines"},
	{ID: "9004000", Cat: "Books/Technical", Desc: "Technical"}, {ID: "9005000", Cat: "Books/Other", Desc: "Books Other"}, {ID: "10000000", Cat: "Other", Desc: "Other"},
	{ID: "10001000", Cat: "Other/Misc", Desc: "Other Misc"},
}

func knabenMeta() *indexerMeta {
	return &indexerMeta{
		ID:          "knaben",
		Name:        "Knaben",
		Description: "Knaben is a Public torrent meta-search engine",
		Language:    "en-US",
		Type:        "public",
		Links:       []string{"https://knaben.org/"},
		Settings:    []Setting{},
		cats:        newCatMap(Caps{CategoryMappings: knabenCats}).newznab(),
	}
}

type knaben struct {
	cats   *catMap
	client *http.Client
	api    string
}

func newKnaben() *knaben {
	return &knaben{cats: newCatMap(Caps{CategoryMappings: knabenCats}), client: defaultClient, api: knabenAPI}
}

type knabenHit struct {
	Title       string `json:"title"`
	CategoryIDs []int  `json:"categoryId"`
	Hash        string `json:"hash"`
	Details     string `json:"details"`
	Link        string `json:"link"`
	MagnetURL   string `json:"magnetUrl"`
	Bytes       int64  `json:"bytes"`
	Seeders     int64  `json:"seeders"`
	Peers       int64  `json:"peers"`
	Date        string `json:"date"`
	TrackerID   string `json:"trackerId"`
	Tracker     string `json:"tracker"`
}

var knabenTZ = regexp.MustCompile(`[+-]\d{2}:\d{2}$|Z$`)

func (k *knaben) Search(ctx context.Context, q *Query) ([]*Release, error) {
	body := map[string]any{
		"order_by": "date", "order_direction": "desc", "from": 0, "size": 100, "hide_unsafe": true,
	}
	if s := strings.TrimSpace(strings.TrimSpace(q.Q) + " " + q.episodeString()); s != "" {
		body["search_type"] = "100%"
		body["search_field"] = "title"
		body["query"] = s
	}
	if cats := k.cats.toTracker(q.Categories); len(cats) > 0 {
		var ids []int
		for _, c := range cats {
			if n, err := strconv.Atoi(c); err == nil {
				ids = append(ids, n)
			}
		}
		body["categories"] = ids
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, k.api, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := k.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}
	var out struct {
		Hits []knabenHit `json:"hits"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&out); err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %w", err)
	}
	var rels []*Release
	for _, h := range out.Hits {
		if h.Seeders <= 0 {
			continue
		}
		link := h.Link
		if strings.Contains(h.TrackerID, "limetorrents") { // its .torrent links do not work
			if h.MagnetURL == "" {
				continue
			}
			link = ""
		}
		r := &Release{
			Title: h.Title, Details: h.Details, Link: link, Magnet: h.MagnetURL,
			InfoHash: normalizeInfoHash(h.Hash), Size: h.Bytes, Seeders: h.Seeders,
			Leechers: h.Peers, Description: h.Tracker,
			DownloadVolumeFactor: 0, UploadVolumeFactor: 1,
		}
		for _, c := range h.CategoryIDs {
			r.Category = unionInts(r.Category, k.cats.fromTracker(strconv.Itoa(c)))
		}
		if d := h.Date; d != "" {
			if !knabenTZ.MatchString(d) {
				d += "+01:00"
			}
			if t, err := time.Parse(time.RFC3339, d); err == nil {
				r.PublishDate = t.UTC()
			}
		}
		if r.Magnet == "" && r.InfoHash != "" {
			r.Magnet = publicMagnet(r.InfoHash, r.Title)
		}
		rels = append(rels, r)
	}
	if q.Limit > 0 && len(rels) > q.Limit {
		rels = rels[:q.Limit]
	}
	return rels, nil
}

// Download fetches a .torrent link returned by Knaben (magnets need no resolving).
func (k *knaben) Download(ctx context.Context, link string) ([]byte, string, error) {
	for i := 0; i <= maxRedirects; i++ {
		if strings.HasPrefix(link, "magnet:") {
			return nil, link, nil
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
		if err != nil {
			return nil, "", err
		}
		req.Header.Set("User-Agent", userAgent)
		resp, err := k.client.Do(req) // does not follow redirects
		if err != nil {
			return nil, "", err
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
		resp.Body.Close()
		if err != nil {
			return nil, "", err
		}
		if loc := resp.Header.Get("Location"); loc != "" && resp.StatusCode >= 300 && resp.StatusCode < 400 {
			u, err := req.URL.Parse(loc)
			if err != nil {
				return nil, "", err
			}
			link = u.String()
			continue
		}
		if !isTorrent(data) {
			return nil, "", fmt.Errorf("the tracker did not return a torrent file (HTTP %d)", resp.StatusCode)
		}
		return data, "", nil
	}
	return nil, "", errors.New("too many redirects")
}
