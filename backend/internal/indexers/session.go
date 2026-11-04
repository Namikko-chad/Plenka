package indexers

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/unicode"
)

const (
	userAgent    = "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"
	maxBody      = 16 << 20
	maxRedirects = 10
)

var (
	errCloudflare        = errors.New("the site is behind a Cloudflare challenge: set the FlareSolverr address in the settings")
	errCloudflarePersist = errors.New("the site still shows a Cloudflare challenge after FlareSolverr solved it")
)

var sharedTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	DialContext:           (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	TLSHandshakeTimeout:   15 * time.Second,
	ResponseHeaderTimeout: 45 * time.Second,
	MaxIdleConns:          32,
	MaxIdleConnsPerHost:   2,
	IdleConnTimeout:       60 * time.Second,
	ForceAttemptHTTP2:     true,
}

// defaultClient is used by native indexers; it does not follow redirects itself.
var defaultClient = &http.Client{
	Transport:     sharedTransport,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// siteJar is a flat cookie jar (name -> value) scoped to the tracker's domain. It is
// what gets persisted between restarts.
type siteJar struct {
	mu      sync.Mutex
	site    string // registrable-ish domain of the tracker ("example.org")
	cookies map[string]string
	dirty   bool
}

func baseDomain(host string) string {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if net.ParseIP(host) != nil {
		return host
	}
	parts := strings.Split(host, ".")
	if len(parts) <= 2 {
		return host
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

func (j *siteJar) related(u *url.URL) bool {
	return j.site != "" && baseDomain(u.Host) == j.site
}

func (j *siteJar) SetCookies(u *url.URL, cs []*http.Cookie) {
	if !j.related(u) {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, c := range cs {
		if c.MaxAge < 0 || (!c.Expires.IsZero() && c.Expires.Before(time.Now())) || c.Value == "deleted" {
			if _, ok := j.cookies[c.Name]; ok {
				delete(j.cookies, c.Name)
				j.dirty = true
			}
			continue
		}
		if j.cookies[c.Name] != c.Value {
			j.cookies[c.Name] = c.Value
			j.dirty = true
		}
	}
}

func (j *siteJar) Cookies(u *url.URL) []*http.Cookie {
	if !j.related(u) {
		return nil
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	out := make([]*http.Cookie, 0, len(j.cookies))
	for k, v := range j.cookies {
		out = append(out, &http.Cookie{Name: k, Value: v})
	}
	return out
}

// setRaw parses "a=b; c=d" (a browser cookie header) into the jar.
func (j *siteJar) setRaw(raw string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, part := range strings.Split(raw, ";") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && k != "" {
			j.cookies[strings.TrimSpace(k)] = strings.TrimSpace(v)
			j.dirty = true
		}
	}
}

func (j *siteJar) set(name, value string) {
	j.mu.Lock()
	if j.cookies[name] != value {
		j.cookies[name] = value
		j.dirty = true
	}
	j.mu.Unlock()
}

func (j *siteJar) snapshot() (map[string]string, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	c := make(map[string]string, len(j.cookies))
	for k, v := range j.cookies {
		c[k] = v
	}
	d := j.dirty
	j.dirty = false
	return c, d
}

func (j *siteJar) clear() {
	j.mu.Lock()
	if len(j.cookies) > 0 {
		j.cookies = map[string]string{}
		j.dirty = true
	}
	j.mu.Unlock()
}

// session performs HTTP requests for one indexer.
type session struct {
	id      string
	client  *http.Client
	jar     *siteJar
	enc     encoding.Encoding
	ua      string
	delay   time.Duration
	mu      sync.Mutex // guards lastReq and ua
	lastReq time.Time
	// solverURL returns the FlareSolverr address ("" = off); nil = off.
	solverURL func() string
}

func (s *session) userAgent() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ua
}

func newSession(def *Definition, base *url.URL, cookies map[string]string, ua string) *session {
	jar := &siteJar{site: baseDomain(base.Host), cookies: map[string]string{}}
	for k, v := range cookies {
		jar.cookies[k] = v
	}
	tr := http.RoundTripper(sharedTransport)
	if len(def.Certificates) > 0 {
		tr = pinnedTransport(def.Certificates)
	}
	enc, err := htmlindex.Get(def.Encoding)
	if err != nil || enc == unicode.UTF8 {
		enc = nil
	}
	if ua == "" {
		ua = userAgent
	}
	return &session{
		id:  def.ID,
		jar: jar,
		client: &http.Client{
			Transport:     tr,
			Jar:           jar,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		enc:   enc,
		ua:    ua,
		delay: time.Duration(def.RequestDelay * float64(time.Second)),
	}
}

// pinnedTransport accepts certificates that fail normal verification when their
// SHA-1 fingerprint is listed in the definition.
func pinnedTransport(fps []string) http.RoundTripper {
	allowed := map[string]bool{}
	for _, f := range fps {
		allowed[strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(f), ":", ""))] = true
	}
	t := sharedTransport.Clone()
	t.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true, // verified below
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 {
				return errors.New("no certificate")
			}
			opts := x509.VerifyOptions{DNSName: cs.ServerName, Intermediates: x509.NewCertPool()}
			for _, c := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(c)
			}
			if _, err := cs.PeerCertificates[0].Verify(opts); err == nil {
				return nil
			}
			sum := sha1.Sum(cs.PeerCertificates[0].Raw)
			if allowed[strings.ToUpper(hex.EncodeToString(sum[:]))] {
				return nil
			}
			return errors.New("certificate is not trusted")
		},
	}
	return t
}

// encodeValue percent-encodes s in the site's charset (space as '+' when form is true,
// "%20" otherwise). Unreserved characters and keep are left as they are.
func (s *session) encodeValue(v string, form bool, keep string) string {
	b := []byte(v)
	if s.enc != nil {
		if eb, err := s.enc.NewEncoder().Bytes(b); err == nil {
			b = eb
		}
	}
	var out strings.Builder
	for _, c := range b {
		switch {
		case c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.' || c == '~':
			out.WriteByte(c)
		case c == ' ' && form:
			out.WriteByte('+')
		case c < 0x80 && strings.IndexByte(keep, c) >= 0:
			out.WriteByte(c)
		default:
			fmt.Fprintf(&out, "%%%02X", c)
		}
	}
	return out.String()
}

// encodePairs builds "k=v&k2=v2". Pairs with key "$raw" are appended verbatim.
func (s *session) encodePairs(ps []pair, sep string) string {
	if sep == "" {
		sep = "&"
	}
	var parts []string
	for _, p := range ps {
		if p.Key == "$raw" {
			if r := strings.Trim(p.Value, "&"); r != "" {
				parts = append(parts, r)
			}
			continue
		}
		parts = append(parts, s.encodeValue(p.Key, true, "[]")+"="+s.encodeValue(p.Value, true, ""))
	}
	return strings.Join(parts, sep)
}

// decode converts a body from the site's charset to UTF-8.
func (s *session) decode(b []byte) []byte {
	if s.enc == nil || isTorrent(b) {
		return b
	}
	if d, err := s.enc.NewDecoder().Bytes(b); err == nil {
		return d
	}
	return b
}

type request struct {
	method   string
	url      string
	body     string // form-encoded body for POST
	ctype    string
	headers  []pair
	follow   bool
	referer  string
	noDecode bool
}

type response struct {
	url        *url.URL // final URL
	status     int
	header     http.Header
	body       []byte // UTF-8 (unless noDecode)
	redirected bool   // at least one redirect happened
	location   string // set when a redirect was not followed (or pointed to a magnet)
}

func (s *session) wait(ctx context.Context) error {
	if s.delay <= 0 {
		return nil
	}
	s.mu.Lock()
	next := s.lastReq.Add(s.delay)
	now := time.Now()
	if next.Before(now) {
		next = now
	}
	s.lastReq = next
	s.mu.Unlock()
	if d := time.Until(next); d > 0 {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
	return nil
}

// do runs a request, following redirects manually when req.follow is set.
func (s *session) do(ctx context.Context, req request) (*response, error) {
	method := strings.ToUpper(req.method)
	if method == "" {
		method = http.MethodGet
	}
	target, body := req.url, req.body
	redirected, solved := false, false
	for hop := 0; ; hop++ {
		if err := s.wait(ctx); err != nil {
			return nil, err
		}
		var rd io.Reader
		if method == http.MethodPost {
			rd = strings.NewReader(body)
		}
		hr, err := http.NewRequestWithContext(ctx, method, target, rd)
		if err != nil {
			return nil, err
		}
		hr.Header.Set("User-Agent", s.userAgent())
		hr.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,application/json;q=0.8,*/*;q=0.7")
		hr.Header.Set("Accept-Language", "en-US,en;q=0.8")
		if method == http.MethodPost {
			ct := req.ctype
			if ct == "" {
				ct = "application/x-www-form-urlencoded"
			}
			hr.Header.Set("Content-Type", ct)
		}
		if req.referer != "" {
			hr.Header.Set("Referer", req.referer)
		}
		for _, h := range req.headers {
			if strings.EqualFold(h.Key, "cookie") {
				s.jar.setRaw(h.Value)
				continue
			}
			hr.Header.Set(h.Key, h.Value)
		}
		resp, err := s.client.Do(hr)
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if len(data) > maxBody {
			return nil, errors.New("response is too large")
		}
		if isCloudflareChallenge(resp, data) {
			if solved {
				return nil, errCloudflarePersist
			}
			solver := ""
			if s.solverURL != nil {
				solver = s.solverURL()
			}
			if solver == "" {
				return nil, errCloudflare
			}
			if err := s.solveChallenge(ctx, solver, target); err != nil {
				return nil, err
			}
			solved = true
			hop-- // the retry is not a redirect
			continue
		}
		loc := resp.Header.Get("Location")
		if resp.StatusCode >= 300 && resp.StatusCode < 400 && loc != "" {
			redirected = true
			if strings.HasPrefix(loc, "magnet:") {
				return &response{url: resp.Request.URL, status: resp.StatusCode, header: resp.Header, redirected: true, location: loc}, nil
			}
			next, err := resp.Request.URL.Parse(loc)
			if err != nil {
				return nil, fmt.Errorf("bad redirect %q", loc)
			}
			if !req.follow || hop >= maxRedirects {
				return &response{url: resp.Request.URL, status: resp.StatusCode, header: resp.Header, body: s.decode(data), redirected: true, location: next.String()}, nil
			}
			if resp.StatusCode != http.StatusTemporaryRedirect && resp.StatusCode != http.StatusPermanentRedirect {
				method, body = http.MethodGet, ""
			}
			target = next.String()
			continue
		}
		out := &response{url: resp.Request.URL, status: resp.StatusCode, header: resp.Header, redirected: redirected}
		if req.noDecode {
			out.body = data
		} else {
			out.body = s.decode(data)
		}
		return out, nil
	}
}

func isCloudflareChallenge(resp *http.Response, body []byte) bool {
	if resp.Header.Get("Cf-Mitigated") == "challenge" {
		return true
	}
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusServiceUnavailable && resp.StatusCode != http.StatusTooManyRequests {
		return false
	}
	if !strings.EqualFold(resp.Header.Get("Server"), "cloudflare") {
		return false
	}
	return bytes.Contains(body, []byte("challenge-platform")) || bytes.Contains(body, []byte("cf_chl_")) ||
		bytes.Contains(body, []byte("<title>Just a moment"))
}

// isTorrent reports whether b looks like a bencoded torrent (a dictionary with "info").
func isTorrent(b []byte) bool {
	return len(b) > 10 && b[0] == 'd' && bytes.Contains(b, []byte("4:info"))
}

// snapshotPeek returns the cookies without resetting the dirty flag.
func (j *siteJar) snapshotPeek() (map[string]string, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	c := make(map[string]string, len(j.cookies))
	for k, v := range j.cookies {
		c[k] = v
	}
	return c, j.dirty
}
