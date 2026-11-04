package torrents

import (
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode"

	tsutils "plenka/internal/torrserver/utils"
)

// baseURL is the scheme and host the client used to reach this server, for absolute
// links handed to media players. X-Forwarded-Proto from a reverse proxy wins.
func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fp := r.Header.Get("X-Forwarded-Proto"); fp != "" {
		fp = strings.ToLower(strings.TrimSpace(strings.Split(fp, ",")[0]))
		if fp == "http" || fp == "https" {
			scheme = fp
		}
	}
	return scheme + "://" + r.Host
}

// streamURL is the absolute stream link of one file.
func streamURL(base, hash string, f File) string {
	return base + "/api/stream/" + hash + "/" + strconv.Itoa(f.ID) + "/" + url.PathEscape(path.Base(f.Path))
}

// playlistFiles picks the files of a playlist: the one with id fileID (0 = not given),
// or every video/audio file in path order. ok is false for an unknown id.
func playlistFiles(files []File, fileID int) (out []File, ok bool) {
	if fileID != 0 {
		for _, f := range files {
			if f.ID == fileID {
				return []File{f}, true
			}
		}
		return nil, false
	}
	for _, f := range files {
		if tsutils.GetMimeType(f.Path) != "*/*" {
			out = append(out, f)
		}
	}
	return out, true
}

// buildM3U renders an extended M3U playlist of absolute stream links.
func buildM3U(base, hash string, files []File) string {
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	for _, f := range files {
		name := path.Base(f.Path)
		b.WriteString("#EXTINF:-1," + strings.NewReplacer("\r", " ", "\n", " ").Replace(name) + "\n")
		b.WriteString(streamURL(base, hash, f) + "\n")
	}
	return b.String()
}

// attachmentName is the Content-Disposition header value for a playlist file named
// after the torrent. Non-ASCII names are sent RFC 2231-encoded.
func attachmentName(title string) string {
	name := sanitizeFileName(title)
	if name == "" {
		name = "playlist"
	}
	return mime.FormatMediaType("attachment", map[string]string{"filename": name + ".m3u"})
}

// sanitizeFileName replaces characters that are not allowed in file names on common
// systems and trims the result to a sane length.
func sanitizeFileName(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= 150 {
			break
		}
		switch {
		case unicode.IsControl(r) || strings.ContainsRune(`/\:*?"<>|`, r):
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
		n++
	}
	return strings.Trim(b.String(), " .")
}
