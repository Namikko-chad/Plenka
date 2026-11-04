package indexers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// Download resolves a result link into .torrent bytes or a magnet URI, logging in first
// when needed and following the definition's download block.
func (c *cardigann) Download(ctx context.Context, link string) ([]byte, string, error) {
	if strings.HasPrefix(link, "magnet:") {
		return nil, link, nil
	}
	if err := c.ensureLogin(ctx); err != nil {
		return nil, "", err
	}
	data, magnet, err := c.download(ctx, link)
	if errors.Is(err, errLoggedOut) && c.def.Login != nil {
		if err := c.relogin(ctx); err != nil {
			return nil, "", err
		}
		data, magnet, err = c.download(ctx, link)
	}
	return data, magnet, err
}

func downloadURIVars(u *url.URL) map[string]any {
	q := map[string]any{}
	for k, v := range u.Query() {
		if len(v) > 0 {
			q[k] = v[0]
		}
	}
	port := u.Port()
	if port == "" {
		port = map[string]string{"https": "443", "http": "80"}[u.Scheme]
	}
	return map[string]any{
		"AbsoluteUri": u.String(), "AbsolutePath": u.EscapedPath(), "Scheme": u.Scheme,
		"Host": u.Hostname(), "Port": port, "PathAndQuery": u.RequestURI(), "Query": q,
	}
}

func (c *cardigann) download(ctx context.Context, link string) ([]byte, string, error) {
	lu, err := url.Parse(link)
	if err != nil {
		return nil, "", fmt.Errorf("bad link %q", link)
	}
	dl := c.def.Download
	if dl == nil {
		return c.fetchTorrent(ctx, link, "", nil)
	}
	vars := c.baseVars()
	vars["DownloadUri"] = downloadURIVars(lu)
	hdrs, err := c.expandHeaders(dl.Headers, vars)
	if err != nil {
		return nil, "", err
	}

	var linkPage *response
	var linkDoc *html.Node
	getLinkPage := func() (*response, *html.Node, error) {
		if linkPage == nil {
			r, d, err := c.fetchPage(ctx, link, hdrs)
			if err != nil {
				return nil, nil, err
			}
			if r.redirected && c.def.Login != nil && c.looksLoggedOut(d) {
				return nil, nil, errLoggedOut
			}
			linkPage, linkDoc = r, d
		}
		return linkPage, linkDoc, nil
	}

	var before *response
	var beforeDoc *html.Node
	if b := dl.Before; b != nil && (b.Path != "" || b.PathSelector != nil) {
		var target *url.URL
		if b.PathSelector != nil {
			page, doc, err := getLinkPage()
			if err != nil {
				return nil, "", err
			}
			v, found, err := c.domField(doc, *b.PathSelector, vars)
			if err != nil {
				return nil, "", err
			}
			if !found || v == "" {
				return nil, "", errors.New("download: before.pathselector found nothing")
			}
			if target, err = c.resolve(page.url, v); err != nil {
				return nil, "", err
			}
		} else if target, err = c.pathURL(b.Path, vars); err != nil {
			return nil, "", err
		}
		inputs, err := c.expandPairs(b.Inputs, vars, true)
		if err != nil {
			return nil, "", err
		}
		req := request{headers: hdrs, follow: true, referer: link}
		enc := c.sess.encodePairs(inputs, b.QuerySeparator)
		if strings.EqualFold(b.Method, "post") {
			req.method, req.url, req.body = http.MethodPost, target.String(), enc
		} else {
			req.url = withQuery(target, enc)
		}
		if before, err = c.sess.do(ctx, req); err != nil {
			return nil, "", fmt.Errorf("download: before request: %w", err)
		}
		if beforeDoc, err = parseHTML(before.body); err != nil {
			return nil, "", err
		}
	}

	pick := func(useBefore bool) (*response, *html.Node, error) {
		if useBefore && before != nil {
			return before, beforeDoc, nil
		}
		return getLinkPage()
	}

	if len(dl.Selectors) > 0 {
		var lastErr error
		for _, sb := range dl.Selectors {
			page, doc, err := pick(sb.UseBeforeResponse)
			if err != nil {
				return nil, "", err
			}
			v, found, err := c.domField(doc, sb, vars)
			if err != nil {
				lastErr = err
				continue
			}
			if !found || strings.TrimSpace(v) == "" {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(v), "magnet:") {
				return nil, strings.TrimSpace(v), nil
			}
			u, err := c.resolve(page.url, v)
			if err != nil {
				lastErr = err
				continue
			}
			data, magnet, err := c.fetchTorrent(ctx, u.String(), dl.Method, hdrs)
			if err == nil {
				return data, magnet, nil
			}
			lastErr = err
		}
		if lastErr != nil {
			return nil, "", lastErr
		}
		return nil, "", errors.New("download link not found on the details page")
	}

	if ih := dl.Infohash; ih != nil {
		_, doc, err := pick(ih.UseBeforeResponse)
		if err != nil {
			return nil, "", err
		}
		hash, found, err := c.domField(doc, ih.Hash, vars)
		if err != nil || !found {
			return nil, "", fmt.Errorf("download: infohash not found: %v", err)
		}
		h := normalizeInfoHash(hash)
		if h == "" {
			return nil, "", fmt.Errorf("download: %q is not an infohash", abbrev(hash, 60))
		}
		title, _, _ := c.domField(doc, ih.Title, vars)
		return nil, publicMagnet(h, cleanTitle(title)), nil
	}

	if before != nil && isTorrent(before.body) {
		return before.body, "", nil
	}
	return c.fetchTorrent(ctx, link, dl.Method, hdrs)
}

// fetchTorrent downloads a .torrent (or follows a redirect to a magnet).
func (c *cardigann) fetchTorrent(ctx context.Context, link, method string, hdrs []pair) ([]byte, string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, "", fmt.Errorf("bad link %q", link)
	}
	req := request{url: link, headers: hdrs, follow: true, noDecode: true, referer: c.base.String()}
	if strings.EqualFold(method, "post") {
		q := u.RawQuery
		u.RawQuery = ""
		req.method, req.url, req.body = http.MethodPost, u.String(), q
	}
	resp, err := c.sess.do(ctx, req)
	if err != nil {
		return nil, "", err
	}
	if strings.HasPrefix(resp.location, "magnet:") {
		return nil, resp.location, nil
	}
	if isTorrent(resp.body) {
		return resp.body, "", nil
	}
	if b := bytes.TrimSpace(resp.body); bytes.HasPrefix(b, []byte("magnet:")) {
		return nil, string(b), nil
	}
	if c.def.Login != nil && (resp.redirected || resp.status == http.StatusUnauthorized || resp.status == http.StatusForbidden) {
		return nil, "", errLoggedOut
	}
	if resp.status >= 400 {
		return nil, "", fmt.Errorf("download failed: HTTP %d", resp.status)
	}
	return nil, "", fmt.Errorf("the tracker did not return a torrent file (HTTP %d)", resp.status)
}
