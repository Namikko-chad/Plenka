package indexers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

// cardigann runs one definition: it logs in, searches and resolves download links.
type cardigann struct {
	def  *Definition
	cats *catMap
	cfg  map[string]string
	base *url.URL
	sess *session
	resp string // "", "json" or "xml" (from the first search path)

	loginMu  sync.Mutex
	loggedIn bool
}

var (
	errLoggedOut = errors.New("the tracker session has expired")
	errNotFound  = errors.New("no match")
)

// Fields that are dropped (not fatal for the row) when their selector finds nothing.
var lenientFields = map[string]bool{
	"poster": true, "imdb": true, "imdbid": true, "rageid": true, "tmdbid": true, "tvmazeid": true,
	"tvdbid": true, "traktid": true, "doubanid": true, "genre": true, "description": true,
}

func newCardigann(def *Definition, cfg map[string]string, cookies map[string]string) (*cardigann, error) {
	if cfg == nil {
		cfg = map[string]string{}
	}
	base, err := url.Parse(siteLink(def, cfg))
	if err != nil || base.Host == "" {
		return nil, fmt.Errorf("invalid site link %q", siteLink(def, cfg))
	}
	c := &cardigann{def: def, cats: newCatMap(def.Caps), cfg: cfg, base: base}
	c.sess = newSession(def, base, cookies, strings.TrimSpace(cfg["useragent"]))
	if r := def.Search.Paths[0].Response; r != nil {
		c.resp = strings.ToLower(r.Type)
	}
	return c, nil
}

// siteLink is the configured base URL (legacy links fall back to the first link).
func siteLink(def *Definition, cfg map[string]string) string {
	l := withSlash(strings.TrimSpace(cfg["sitelink"]))
	if l == "" {
		return def.Links[0]
	}
	for _, old := range def.LegacyLinks {
		if strings.EqualFold(old, l) {
			return def.Links[0]
		}
	}
	return l
}

// ---- template variables ----

func (c *cardigann) baseVars() tmplVars {
	conf := map[string]any{}
	for _, s := range c.def.settings() {
		if strings.HasPrefix(strings.ToLower(s.Type), "info") {
			continue
		}
		conf[s.Name] = configValue(s, c.cfg)
	}
	conf["sitelink"] = c.base.String()
	return tmplVars{
		"Config": conf,
		"True":   "True",
		"False":  nil,
		"Today":  map[string]any{"Year": strconv.Itoa(nowFunc().Year())},
		"Query":  map[string]any{},
		"Result": map[string]any{},
	}
}

// configValue is the template value of a setting: checkboxes are "True" or empty,
// multi-selects are lists.
func configValue(s Setting, cfg map[string]string) any {
	v, ok := cfg[s.Name]
	if !ok {
		v = s.Default
		if strings.EqualFold(s.Type, "multi-select") && len(s.Defaults) > 0 {
			v = strings.Join(s.Defaults, ",")
		}
	}
	switch strings.ToLower(s.Type) {
	case "checkbox":
		if isTrue(v) {
			return "True"
		}
		return nil
	case "multi-select":
		var out []string
		for _, x := range strings.Split(v, ",") {
			if x = strings.TrimSpace(x); x != "" {
				out = append(out, x)
			}
		}
		return out
	}
	return v
}

func isTrue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "on", "yes":
		return true
	}
	return false
}

func (c *cardigann) queryVars(q *Query, vars tmplVars) (string, error) {
	raw := strings.TrimSpace(strings.TrimSpace(q.Q) + " " + q.episodeString())
	kw, err := applyFilters(raw, c.def.Search.KeywordsFilters, vars, c.def.ID)
	if err != nil {
		return "", fmt.Errorf("keywordsfilters: %w", err)
	}
	kw = strings.TrimSpace(kw)
	var season, ep any
	if q.Season != "" {
		season = q.Season
	}
	if q.Ep != "" {
		ep = q.Ep
	}
	cats := c.cats.toTracker(q.Categories)
	qv := map[string]any{
		"Type": "search", "Q": nilIfEmpty(strings.TrimSpace(q.Q)), "Keywords": nilIfEmpty(raw),
		"Season": season, "Ep": ep, "Limit": strconv.Itoa(q.Limit), "Offset": "0",
		"Categories": cats, "IsSearch": "True",
	}
	if raw == "" {
		qv["IsRssSearch"] = "True"
	}
	vars["Query"] = qv
	vars["Keywords"] = nilIfEmpty(kw)
	vars["Categories"] = cats
	return kw, nil
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func cloneVars(v tmplVars) tmplVars {
	out := make(tmplVars, len(v)+1)
	for k, x := range v {
		out[k] = x
	}
	out["Result"] = map[string]any{}
	return out
}

// expandPairs evaluates input templates. "$raw" values have their printed variables
// URL-encoded; empty values are dropped unless allowEmpty.
func (c *cardigann) expandPairs(ps pairs, vars tmplVars, allowEmpty bool) ([]pair, error) {
	var out []pair
	for _, p := range ps {
		if p.Key == "$raw" {
			v, err := expand(p.Value, vars, func(s string) string { return c.sess.encodeValue(s, true, "") })
			if err != nil {
				return nil, err
			}
			out = append(out, pair{p.Key, v})
			continue
		}
		v, err := expand(p.Value, vars, nil)
		if err != nil {
			return nil, err
		}
		if v == "" && !allowEmpty {
			continue
		}
		out = append(out, pair{p.Key, v})
	}
	return out, nil
}

func (c *cardigann) expandHeaders(h headers, vars tmplVars) ([]pair, error) {
	var out []pair
	for _, x := range h {
		for _, v := range x.Values {
			s, err := expand(v, vars, nil)
			if err != nil {
				return nil, err
			}
			out = append(out, pair{x.Name, s})
		}
	}
	return out, nil
}

// pathURL expands a path template (printed values are path-escaped) and resolves it
// against the site link.
func (c *cardigann) pathURL(tpl string, vars tmplVars) (*url.URL, error) {
	p, err := expand(tpl, vars, func(s string) string { return c.sess.encodeValue(s, false, "/:,+=&?") })
	if err != nil {
		return nil, err
	}
	return c.resolve(c.base, strings.TrimSpace(p))
}

func (c *cardigann) resolve(base *url.URL, ref string) (*url.URL, error) {
	ref = strings.TrimSpace(ref)
	if strings.HasPrefix(ref, "//") {
		ref = base.Scheme + ":" + ref
	}
	u, err := url.Parse(ref)
	if err != nil {
		// tolerate unescaped characters in hrefs
		u, err = url.Parse(strings.ReplaceAll(url.PathEscape(ref), "%2F", "/"))
		if err != nil {
			return nil, fmt.Errorf("bad URL %q", ref)
		}
	}
	return base.ResolveReference(u), nil
}

func withQuery(u *url.URL, q string) string {
	if q == "" {
		return u.String()
	}
	s := u.String()
	if strings.Contains(s, "?") {
		if strings.HasSuffix(s, "?") || strings.HasSuffix(s, "&") {
			return s + q
		}
		return s + "&" + q
	}
	return s + "?" + q
}

// ---- search ----

// Search runs every applicable search path and returns the combined releases.
func (c *cardigann) Search(ctx context.Context, q *Query) ([]*Release, error) {
	if err := c.ensureLogin(ctx); err != nil {
		return nil, err
	}
	vars := c.baseVars()
	if _, err := c.queryVars(q, vars); err != nil {
		return nil, err
	}
	trackerCats, _ := vars["Categories"].([]string)
	var all []*Release
	var firstErr error
	ran := 0
	for _, p := range c.def.Search.Paths {
		if !pathApplies(p.Categories, trackerCats) {
			continue
		}
		ran++
		rels, err := c.searchPath(ctx, p, vars, true)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		all = append(all, rels...)
	}
	if ran > 0 && len(all) == 0 && firstErr != nil {
		return nil, firstErr
	}
	all = c.postFilter(all, q, vars)
	if q.Limit > 0 && len(all) > q.Limit {
		all = all[:q.Limit]
	}
	return all, nil
}

// pathApplies implements search path "categories" (a leading "!" negates the list).
func pathApplies(pathCats, queryCats []string) bool {
	if len(pathCats) == 0 || len(queryCats) == 0 {
		return true
	}
	negate := pathCats[0] == "!"
	list := pathCats
	if negate {
		list = pathCats[1:]
	}
	hit := false
	for _, pc := range list {
		for _, qc := range queryCats {
			if pc == qc {
				hit = true
			}
		}
	}
	if negate {
		// used unless every requested category is in the excluded list
		for _, qc := range queryCats {
			in := false
			for _, pc := range list {
				if pc == qc {
					in = true
				}
			}
			if !in {
				return true
			}
		}
		return false
	}
	return hit
}

func (c *cardigann) postFilter(rels []*Release, q *Query, vars tmplVars) []*Release {
	var andWords []string
	for _, f := range c.def.Search.Rows.Filters {
		if strings.EqualFold(f.Name, "andmatch") {
			kw := valueString(vars["Keywords"])
			if n, err := strconv.Atoi(f.Args.arg(0)); err == nil && n > 0 && len([]rune(kw)) > n {
				kw = string([]rune(kw)[:n])
			}
			andWords = strings.Fields(strings.ToLower(kw))
		}
	}
	out := rels[:0]
	for _, r := range rels {
		if len(q.Categories) > 0 && len(r.Category) > 0 && !anyCatMatches(r.Category, q.Categories) {
			continue
		}
		if len(andWords) > 0 {
			t := strings.ToLower(r.Title)
			ok := true
			for _, w := range andWords {
				if !strings.Contains(t, w) {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

func anyCatMatches(have, want []int) bool {
	for _, h := range have {
		for _, w := range want {
			if catMatches(h, w) {
				return true
			}
		}
	}
	return false
}

func (c *cardigann) searchPath(ctx context.Context, p SearchPath, vars tmplVars, retry bool) ([]*Release, error) {
	s := &c.def.Search
	var inputs pairs
	if p.InheritInputs == nil || *p.InheritInputs {
		inputs = append(inputs, s.Inputs...)
	}
	for _, kv := range p.Inputs { // path inputs override inherited ones
		replaced := false
		for i := range inputs {
			if inputs[i].Key == kv.Key && kv.Key != "$raw" {
				inputs[i].Value, replaced = kv.Value, true
			}
		}
		if !replaced {
			inputs = append(inputs, kv)
		}
	}
	ps, err := c.expandPairs(inputs, vars, s.AllowEmptyInputs)
	if err != nil {
		return nil, err
	}
	u, err := c.pathURL(p.Path, vars)
	if err != nil {
		return nil, err
	}
	method, err := expand(p.Method, vars, nil)
	if err != nil {
		return nil, err
	}
	hdrs, err := c.expandHeaders(s.Headers, vars)
	if err != nil {
		return nil, err
	}
	req := request{url: u.String(), headers: hdrs, follow: p.FollowRedirect, referer: c.base.String()}
	enc := c.sess.encodePairs(ps, p.QuerySeparator)
	if strings.EqualFold(strings.TrimSpace(method), "post") {
		req.method, req.body = http.MethodPost, enc
	} else {
		req.url = withQuery(u, enc)
	}
	resp, err := c.sess.do(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.location != "" {
		if retry && c.def.Login != nil {
			if err := c.relogin(ctx); err != nil {
				return nil, err
			}
			return c.searchPath(ctx, p, vars, false)
		}
		return nil, fmt.Errorf("search was redirected to %s", resp.location)
	}
	// some sites answer "nothing found" with a 404 page: parse it like any other page
	if resp.status >= 400 && !(resp.status == http.StatusNotFound && len(bytes.TrimSpace(resp.body)) > 0) {
		return nil, fmt.Errorf("unexpected HTTP status %d", resp.status)
	}
	body := resp.body
	if len(s.PreprocessingFilters) > 0 {
		v, err := applyFilters(string(body), s.PreprocessingFilters, vars, c.def.ID)
		if err != nil {
			return nil, fmt.Errorf("preprocessingfilters: %w", err)
		}
		body = []byte(v)
	}
	if p.Response != nil && p.Response.NoResultsMessage != nil {
		msg := *p.Response.NoResultsMessage
		if (msg == "" && len(bytes.TrimSpace(body)) == 0) || (msg != "" && bytes.Contains(body, []byte(msg))) {
			return nil, nil
		}
	}
	typ := c.resp
	if p.Response != nil {
		typ = strings.ToLower(p.Response.Type)
	}
	switch typ {
	case "json":
		return c.parseJSONRows(body, resp.url, vars)
	case "xml":
		doc, err := parseXML(body)
		if err != nil {
			return nil, err
		}
		return c.parseDOMRows(doc, resp.url, vars)
	default:
		doc, err := parseHTML(body)
		if err != nil {
			return nil, err
		}
		if c.def.Login != nil && retry && c.looksLoggedOut(doc) {
			if err := c.relogin(ctx); err != nil {
				return nil, err
			}
			return c.searchPath(ctx, p, vars, false)
		}
		return c.parseDOMRows(doc, resp.url, vars)
	}
}

// looksLoggedOut: the page shows the login form instead of results (login test
// selector missing while a password field is present).
func (c *cardigann) looksLoggedOut(doc *html.Node) bool {
	t := c.def.Login.Test
	if t == nil || t.Selector == "" || strings.Contains(t.Selector, "{{") {
		return false
	}
	if n, err := queryFirst(doc, t.Selector); err != nil || n != nil {
		return false
	}
	pw, _ := queryFirst(doc, `input[type="password"]`)
	return pw != nil
}

// checkPageErrors applies error blocks to a page; it returns the reported message.
func (c *cardigann) checkPageErrors(doc *html.Node, blocks []ErrorBlock, vars tmplVars) (string, bool) {
	for _, eb := range blocks {
		if eb.Selector == "" {
			continue
		}
		n, err := queryFirst(doc, eb.Selector)
		if err != nil || n == nil {
			continue
		}
		msg := cleanTitle(nodeText(n))
		if eb.Message != nil {
			if v, found, err := c.domField(doc, *eb.Message, vars); err == nil && found && strings.TrimSpace(v) != "" {
				msg = cleanTitle(v)
			}
		}
		if msg == "" {
			msg = "the site reported an error"
		}
		return msg, true
	}
	return "", false
}

func (c *cardigann) parseDOMRows(doc *html.Node, pageURL *url.URL, vars tmplVars) ([]*Release, error) {
	s := &c.def.Search
	if msg, bad := c.checkPageErrors(doc, s.Error, vars); bad {
		return nil, fmt.Errorf("tracker error: %s", msg)
	}
	sel, err := expand(s.Rows.Selector, vars, nil)
	if err != nil {
		return nil, err
	}
	rows, err := queryAll(doc, sel)
	if err != nil {
		return nil, err
	}
	if s.Rows.After > 0 {
		var merged []*html.Node
		for i := 0; i < len(rows); i += 1 + s.Rows.After {
			r := rows[i]
			for k := 1; k <= s.Rows.After && i+k < len(rows); k++ {
				extra := rows[i+k]
				for ch := extra.FirstChild; ch != nil; {
					next := ch.NextSibling
					extra.RemoveChild(ch)
					r.AppendChild(ch)
					ch = next
				}
			}
			merged = append(merged, r)
		}
		rows = merged
	}
	var out []*Release
	var firstErr error
	for _, row := range rows {
		rv := cloneVars(vars)
		var headerDate string
		if dh := s.Rows.DateHeaders; dh != nil && dh.Selector != "" {
			headerDate = c.dateHeader(row, *dh, rv)
		}
		rel, err := c.buildRow(rv, pageURL, headerDate, func(b SelectorBlock) (string, bool, error) {
			return c.domField(row, b, rv)
		})
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			slog.Debug("indexers: row skipped", "indexer", c.def.ID, "err", err)
			continue
		}
		if rel != nil {
			out = append(out, rel)
		}
	}
	if len(out) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

// dateHeader finds the closest previous sibling of row matching the dateheaders
// selector and returns its filtered text.
func (c *cardigann) dateHeader(row *html.Node, dh SelectorBlock, vars tmplVars) string {
	g, err := compileCSS(dh.Selector)
	if err != nil {
		return ""
	}
	for n := row.PrevSibling; n != nil; n = n.PrevSibling {
		if n.Type != html.ElementNode || !g.Match(n) {
			continue
		}
		v, err := applyFilters(strings.TrimSpace(nodeText(n)), dh.Filters, vars, c.def.ID)
		if err != nil {
			return ""
		}
		return v
	}
	return ""
}

// domField extracts one field from an HTML/XML node.
func (c *cardigann) domField(row *html.Node, b SelectorBlock, vars tmplVars) (string, bool, error) {
	var v string
	found := false
	if b.Text != nil {
		x, err := expand(*b.Text, vars, nil)
		if err != nil {
			return "", false, err
		}
		v, found = x, true
	} else {
		el := row
		if b.Selector != "" {
			sel, err := expand(b.Selector, vars, nil)
			if err != nil {
				return "", false, err
			}
			el, err = queryFirst(row, sel)
			if err != nil {
				return "", false, err
			}
		}
		if el != nil {
			if b.Remove != "" {
				if err := removeMatches(el, b.Remove); err != nil {
					return "", false, err
				}
			}
			switch {
			case len(b.Case) > 0:
				for _, cs := range b.Case {
					ok := cs.Key == "*"
					if !ok {
						m, err := matchesOrContains(el, cs.Key)
						if err != nil {
							return "", false, err
						}
						ok = m
					}
					if ok {
						x, err := expand(cs.Value, vars, nil)
						if err != nil {
							return "", false, err
						}
						v, found = x, true
						break
					}
				}
			case b.Attribute != "":
				v, found = attr(el, b.Attribute)
			default:
				v, found = nodeText(el), true
			}
			v = strings.TrimSpace(v)
		}
	}
	return c.finishField(v, found, b, vars)
}

// finishField applies defaults/optional handling and the filter chain.
func (c *cardigann) finishField(v string, found bool, b SelectorBlock, vars tmplVars) (string, bool, error) {
	if !found {
		if b.Default != nil {
			d, err := expand(*b.Default, vars, nil)
			return d, err == nil, err
		}
		if b.Optional {
			return "", false, nil
		}
		return "", false, fmt.Errorf("selector %q found nothing: %w", abbrev(b.Selector, 80), errNotFound)
	}
	out, err := applyFilters(v, b.Filters, vars, c.def.ID)
	if err != nil {
		return "", false, err
	}
	return out, true, nil
}

func (c *cardigann) parseJSONRows(body []byte, pageURL *url.URL, vars tmplVars) ([]*Release, error) {
	s := &c.def.Search
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, nil
	}
	root, err := decodeJSON(body)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %w", err)
	}
	ctx := &jsonCtx{root: root}
	if cnt := s.Rows.Count; cnt != nil && cnt.Selector != "" {
		cs, err := compileJSONSel(cnt.Selector)
		if err != nil {
			return nil, err
		}
		if v, ok := cs.first(root, ctx); ok && parseCount(jsonString(v)) == 0 {
			return nil, nil
		}
	}
	selText, err := expand(s.Rows.Selector, vars, nil)
	if err != nil {
		return nil, err
	}
	sel, err := compileJSONSel(selText)
	if err != nil {
		return nil, err
	}
	type jrow struct{ v, parent any }
	var rows []jrow
	items := sel.all(root, ctx)
	if s.Rows.Attribute != "" {
		as, err := compileJSONSel(s.Rows.Attribute)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			sub, ok := as.first(it, ctx)
			if !ok {
				if s.Rows.MissingAttributeEqualsNoResults {
					continue
				}
				return nil, fmt.Errorf("row attribute %q is missing", s.Rows.Attribute)
			}
			if arr, isArr := sub.([]any); isArr {
				for _, e := range arr {
					rows = append(rows, jrow{e, it})
				}
			} else {
				rows = append(rows, jrow{sub, it})
			}
		}
	} else {
		for _, it := range items {
			rows = append(rows, jrow{it, nil})
		}
	}
	var out []*Release
	var firstErr error
	for _, r := range rows {
		rv := cloneVars(vars)
		fctx := &jsonCtx{root: root, parent: r.parent}
		row := r.v
		rel, err := c.buildRow(rv, pageURL, "", func(b SelectorBlock) (string, bool, error) {
			return c.jsonField(row, fctx, b, rv)
		})
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			slog.Debug("indexers: row skipped", "indexer", c.def.ID, "err", err)
			continue
		}
		if rel != nil {
			out = append(out, rel)
		}
	}
	if len(out) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

func (c *cardigann) jsonField(row any, ctx *jsonCtx, b SelectorBlock, vars tmplVars) (string, bool, error) {
	var v string
	found := false
	if b.Text != nil {
		x, err := expand(*b.Text, vars, nil)
		if err != nil {
			return "", false, err
		}
		v, found = x, true
	} else if b.Selector != "" {
		selText, err := expand(b.Selector, vars, nil)
		if err != nil {
			return "", false, err
		}
		sel, err := compileJSONSel(selText)
		if err != nil {
			return "", false, err
		}
		if x, ok := sel.first(row, ctx); ok {
			v, found = strings.TrimSpace(jsonString(x)), true
		}
	}
	if found && len(b.Case) > 0 {
		matched := false
		for _, cs := range b.Case {
			if cs.Key == "*" || cs.Key == v {
				x, err := expand(cs.Value, vars, nil)
				if err != nil {
					return "", false, err
				}
				v, matched = x, true
				break
			}
		}
		found = matched
	}
	return c.finishField(v, found, b, vars)
}

// buildRow extracts all fields of one row (in definition order) and builds a release.
func (c *cardigann) buildRow(vars tmplVars, pageURL *url.URL, headerDate string, get func(SelectorBlock) (string, bool, error)) (*Release, error) {
	res := vars["Result"].(map[string]any)
	vals := map[string]string{}
	var cats []int
	for _, f := range c.def.Search.Fields {
		name, _, _ := strings.Cut(f.Name, "|")
		v, found, err := get(f.Block)
		if err != nil && lenientFields[name] && errors.Is(err, errNotFound) {
			v, found, err = "", false, nil
		}
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", name, err)
		}
		if !found {
			res[name] = nil
			continue
		}
		res[name] = v
		vals[name] = v
		switch name {
		case "category":
			for _, id := range strings.Split(v, ",") {
				cats = unionInts(cats, c.cats.fromTracker(id))
			}
		case "categorydesc":
			cats = unionInts(cats, c.cats.fromDesc(v))
		}
	}
	r := &Release{Title: cleanTitle(vals["title"]), Category: cats, DownloadVolumeFactor: 1, UploadVolumeFactor: 1}
	if r.Title == "" {
		return nil, errors.New("empty title")
	}
	link := func(s string) string {
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, "magnet:") {
			return s
		}
		u, err := c.resolve(pageURL, s)
		if err != nil {
			return ""
		}
		return u.String()
	}
	r.Details = link(vals["details"])
	if d := link(vals["download"]); strings.HasPrefix(d, "magnet:") {
		r.Magnet = d
	} else {
		r.Link = d
	}
	if m := strings.TrimSpace(vals["magnet"]); strings.HasPrefix(m, "magnet:") {
		r.Magnet = m
	}
	r.InfoHash = normalizeInfoHash(vals["infohash"])
	if r.InfoHash == "" && r.Magnet != "" {
		r.InfoHash = magnetHash(r.Magnet)
	}
	if r.Magnet == "" && r.InfoHash != "" && c.def.Type != "private" {
		r.Magnet = publicMagnet(r.InfoHash, r.Title)
	}
	if r.Details == "" {
		r.Details = c.base.String()
	}
	if r.Link == "" && r.Magnet == "" && c.def.Download != nil {
		r.Link = r.Details // the download block resolves it from the details page
	}
	if r.Link == "" && r.Magnet == "" {
		return nil, errors.New("no download link")
	}
	r.Size = parseSize(vals["size"])
	r.Seeders = parseCount(vals["seeders"])
	r.Leechers = parseCount(vals["leechers"])
	r.Grabs = parseCount(vals["grabs"])
	date := vals["date"]
	if date == "" {
		date = headerDate
	}
	if t, ok := parseResultDate(date); ok {
		r.PublishDate = t
	}
	r.DownloadVolumeFactor = parseFactor(vals["downloadvolumefactor"], 1)
	r.UploadVolumeFactor = parseFactor(vals["uploadvolumefactor"], 1)
	r.Description = strings.TrimSpace(vals["description"])
	r.Poster = link(vals["poster"])
	r.IMDBID = normalizeIMDB(firstNonEmpty(vals["imdbid"], vals["imdb"]))
	return r, nil
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if strings.TrimSpace(x) != "" {
			return x
		}
	}
	return ""
}

// cookieSnapshot returns the session cookies and whether they changed.
func (c *cardigann) cookieSnapshot() (map[string]string, bool) { return c.sess.jar.snapshot() }
