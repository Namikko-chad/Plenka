package torrents

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
)

// IndexerLinkResolver resolves links that point at this server's own tracker download
// endpoint (/api/indexers/{id}/download?token=…) in-process, without an HTTP round trip.
// It returns either the .torrent bytes or a magnet URI. Set by main at startup (late
// binding, so this package does not import the indexers).
var IndexerLinkResolver func(indexerID, token string) (torrentData []byte, magnet string, err error)

// maxTorrentFile caps a downloaded or uploaded .torrent file.
const maxTorrentFile = 16 << 20

// httpClient fetches http(s) .torrent links.
var httpClient = &http.Client{Timeout: 60 * time.Second}

var infoHashRe = regexp.MustCompile(`^(?i:[0-9a-f]{40}|[a-z2-7]{32})$`)

// parseLink turns a link from POST /api/torrents into a torrent spec: a magnet URI, an
// http(s) .torrent URL, a bare info hash or an indexer download link.
func parseLink(link string) (*torrent.TorrentSpec, error) {
	link = strings.TrimSpace(strings.ReplaceAll(link, "&amp;", "&"))
	if link == "" {
		return nil, errors.New("link is empty")
	}
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	if spec, ok, err := fromIndexerLink(u); ok {
		return spec, err
	}
	switch strings.ToLower(u.Scheme) {
	case "magnet":
		return fromMagnet(link)
	case "http", "https":
		return fromHTTP(link)
	case "":
		if !infoHashRe.MatchString(link) {
			return nil, fmt.Errorf("not a magnet, URL or info hash: %q", link)
		}
		return fromMagnet("magnet:?xt=urn:btih:" + link)
	default:
		return nil, fmt.Errorf("unsupported link scheme %q", u.Scheme)
	}
}

func fromMagnet(link string) (*torrent.TorrentSpec, error) {
	mag, err := metainfo.ParseMagnetUri(link)
	if err != nil {
		return nil, err
	}
	var trackers [][]string
	if len(mag.Trackers) > 0 {
		trackers = [][]string{mag.Trackers}
	}
	return &torrent.TorrentSpec{
		Trackers:    trackers,
		DisplayName: mag.DisplayName,
		InfoHash:    mag.InfoHash,
	}, nil
}

// fromTorrentFile parses .torrent bytes.
func fromTorrentFile(r io.Reader) (*torrent.TorrentSpec, error) {
	mi, err := metainfo.Load(io.LimitReader(r, maxTorrentFile))
	if err != nil {
		return nil, fmt.Errorf("bad .torrent file: %w", err)
	}
	info, err := mi.UnmarshalInfo()
	if err != nil {
		return nil, fmt.Errorf("bad .torrent file: %w", err)
	}
	mag := mi.Magnet(nil, &info)
	var trackers [][]string
	if len(mag.Trackers) > 0 {
		trackers = [][]string{mag.Trackers}
	}
	return &torrent.TorrentSpec{
		InfoBytes:   mi.InfoBytes,
		Trackers:    trackers,
		DisplayName: info.BestName(),
		InfoHash:    mi.HashInfoBytes(),
	}, nil
}

func fromHTTP(link string) (*torrent.TorrentSpec, error) {
	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DWL/1.1.1 (Torrent)")
	resp, err := httpClient.Do(req)
	// Trackers often redirect a download link to a magnet URI.
	var uerr *url.Error
	if errors.As(err, &uerr) && strings.HasPrefix(uerr.URL, "magnet:") {
		return fromMagnet(uerr.URL)
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch .torrent: %s", resp.Status)
	}
	return fromTorrentFile(resp.Body)
}

// fromIndexerLink handles /api/indexers/{id}/download?token=… links (relative, or
// absolute to any host).
func fromIndexerLink(u *url.URL) (*torrent.TorrentSpec, bool, error) {
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "indexers" || parts[3] != "download" {
		return nil, false, nil
	}
	if IndexerLinkResolver == nil {
		return nil, true, errors.New("indexer links are not supported")
	}
	data, magnet, err := IndexerLinkResolver(parts[2], u.Query().Get("token"))
	if err != nil {
		return nil, true, err
	}
	if magnet != "" {
		spec, err := fromMagnet(magnet)
		return spec, true, err
	}
	spec, err := fromTorrentFile(bytes.NewReader(data))
	return spec, true, err
}
