package indexers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

var errCaptcha = errors.New("the login form requires a CAPTCHA, which is not supported; use a cookie-based setup if the tracker offers one")

// ensureLogin makes sure the session is authenticated (no-op without a login block).
func (c *cardigann) ensureLogin(ctx context.Context) error {
	lb := c.def.Login
	if lb == nil {
		return nil
	}
	c.loginMu.Lock()
	defer c.loginMu.Unlock()
	if c.loggedIn {
		return nil
	}
	method := strings.ToLower(strings.TrimSpace(lb.Method))
	if method == "cookie" {
		if err := c.applyCookieConfig(); err != nil {
			return err
		}
	} else if snap, _ := c.sess.jar.snapshotPeek(); len(snap) > 0 && lb.Test != nil && lb.Test.Path != "" {
		// stored session cookies: reuse them if they still pass the login test
		if ok, _ := c.testLogin(ctx); ok {
			c.loggedIn = true
			return nil
		}
		c.sess.jar.clear()
	}
	if err := c.login(ctx, method); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	c.loggedIn = true
	return nil
}

func (c *cardigann) relogin(ctx context.Context) error {
	c.loginMu.Lock()
	c.loggedIn = false
	if !strings.EqualFold(c.def.Login.Method, "cookie") {
		c.sess.jar.clear()
	}
	c.loginMu.Unlock()
	return c.ensureLogin(ctx)
}

func (c *cardigann) applyCookieConfig() error {
	vars := c.baseVars()
	raw := ""
	if v, ok := c.def.Login.Inputs.get("cookie"); ok {
		x, err := expand(v, vars, nil)
		if err != nil {
			return err
		}
		raw = x
	}
	if strings.TrimSpace(raw) == "" {
		raw = c.cfg["cookie"]
	}
	if strings.TrimSpace(raw) == "" {
		return errors.New("no cookie configured")
	}
	c.sess.jar.setRaw(raw)
	return nil
}

func (c *cardigann) login(ctx context.Context, method string) error {
	lb := c.def.Login
	vars := c.baseVars()
	for _, ck := range lb.Cookies {
		c.sess.jar.setRaw(ck)
	}
	hdrs, err := c.expandHeaders(lb.Headers, vars)
	if err != nil {
		return err
	}
	var resp *response
	switch method {
	case "cookie":
		if lb.Test == nil || lb.Test.Path == "" {
			return nil
		}
		if ok, err := c.testLogin(ctx); err != nil {
			return err
		} else if !ok {
			return errors.New("the configured cookie is invalid or has expired")
		}
		return nil
	case "", "post", "get":
		resp, err = c.loginDirect(ctx, method, hdrs, vars)
	case "form":
		resp, err = c.loginForm(ctx, hdrs, vars)
	case "oneurl":
		resp, err = c.loginOneURL(ctx, hdrs, vars)
	default:
		return fmt.Errorf("login method %q is not supported", lb.Method)
	}
	if err != nil {
		return err
	}
	if resp != nil && len(lb.Error) > 0 {
		if doc, err := parseHTML(resp.body); err == nil {
			if msg, bad := c.checkPageErrors(doc, lb.Error, vars); bad {
				return errors.New(msg)
			}
		}
	}
	if lb.Test != nil && lb.Test.Path != "" {
		ok, err := c.testLogin(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("the login check did not pass (wrong credentials?)")
		}
	}
	return nil
}

// selectorValues extracts selectorinputs/getselectorinputs from a page.
func (c *cardigann) selectorValues(doc *html.Node, blocks namedBlocks, vars tmplVars) ([]pair, error) {
	var out []pair
	for _, nb := range blocks {
		v, found, err := c.domField(doc, nb.Block, vars)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", nb.Name, err)
		}
		if found {
			out = append(out, pair{nb.Name, v})
		}
	}
	return out, nil
}

func (c *cardigann) fetchPage(ctx context.Context, u string, hdrs []pair) (*response, *html.Node, error) {
	resp, err := c.sess.do(ctx, request{url: u, headers: hdrs, follow: true, referer: c.base.String()})
	if err != nil {
		return nil, nil, err
	}
	if resp.status >= 400 {
		return nil, nil, fmt.Errorf("unexpected HTTP status %d from %s", resp.status, u)
	}
	doc, err := parseHTML(resp.body)
	if err != nil {
		return nil, nil, err
	}
	return resp, doc, nil
}

// loginDirect sends the inputs as a POST body or GET query.
func (c *cardigann) loginDirect(ctx context.Context, method string, hdrs []pair, vars tmplVars) (*response, error) {
	lb := c.def.Login
	u, err := c.pathURL(lb.Path, vars)
	if err != nil {
		return nil, err
	}
	inputs, err := c.expandPairs(lb.Inputs, vars, true)
	if err != nil {
		return nil, err
	}
	var query []pair
	if len(lb.SelectorInputs) > 0 || len(lb.GetSelectorInputs) > 0 {
		_, doc, err := c.fetchPage(ctx, u.String(), hdrs)
		if err != nil {
			return nil, err
		}
		si, err := c.selectorValues(doc, lb.SelectorInputs, vars)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, si...)
		if query, err = c.selectorValues(doc, lb.GetSelectorInputs, vars); err != nil {
			return nil, err
		}
	}
	req := request{headers: hdrs, follow: true, referer: u.String()}
	if method == "get" {
		req.url = withQuery(u, c.sess.encodePairs(append(inputs, query...), ""))
	} else {
		req.method = http.MethodPost
		req.url = withQuery(u, c.sess.encodePairs(query, ""))
		req.body = c.sess.encodePairs(inputs, "")
	}
	return c.sess.do(ctx, req)
}

// loginOneURL opens a single URL (the inputs are part of the path).
func (c *cardigann) loginOneURL(ctx context.Context, hdrs []pair, vars tmplVars) (*response, error) {
	u, err := c.pathURL(c.def.Login.Path, vars)
	if err != nil {
		return nil, err
	}
	return c.sess.do(ctx, request{url: u.String(), headers: hdrs, follow: true})
}

// loginForm loads the login page, fills its form and submits it.
func (c *cardigann) loginForm(ctx context.Context, hdrs []pair, vars tmplVars) (*response, error) {
	lb := c.def.Login
	u, err := c.pathURL(lb.Path, vars)
	if err != nil {
		return nil, err
	}
	landing, doc, err := c.fetchPage(ctx, u.String(), hdrs)
	if err != nil {
		return nil, err
	}
	formSel := lb.Form
	if formSel == "" {
		formSel = "form"
	}
	form, err := queryFirst(doc, formSel)
	if err != nil {
		return nil, err
	}
	if form == nil {
		return nil, fmt.Errorf("login form %q not found on %s", formSel, landing.url)
	}
	if lb.Captcha != nil && lb.Captcha.Selector != "" {
		if n, _ := queryFirst(doc, lb.Captcha.Selector); n != nil {
			return nil, errCaptcha
		}
	}
	fields := formFields(form)
	set := func(name, value string) {
		for i := range fields {
			if fields[i].Key == name {
				fields[i].Value = value
				return
			}
		}
		fields = append(fields, pair{name, value})
	}
	inputs, err := c.expandPairs(lb.Inputs, vars, true)
	if err != nil {
		return nil, err
	}
	for _, in := range inputs {
		name := in.Key
		if lb.Selectors {
			n, err := queryFirst(doc, in.Key)
			if err != nil {
				return nil, err
			}
			if n == nil {
				return nil, fmt.Errorf("login input %q not found", in.Key)
			}
			name, _ = attr(n, "name")
		}
		if name != "" {
			set(name, in.Value)
		}
	}
	si, err := c.selectorValues(doc, lb.SelectorInputs, vars)
	if err != nil {
		return nil, err
	}
	for _, p := range si {
		set(p.Key, p.Value)
	}
	query, err := c.selectorValues(doc, lb.GetSelectorInputs, vars)
	if err != nil {
		return nil, err
	}

	var submit *url.URL
	if lb.SubmitPath != "" {
		submit, err = c.pathURL(lb.SubmitPath, vars)
	} else if action, ok := attr(form, "action"); ok && strings.TrimSpace(action) != "" {
		submit, err = c.resolve(landing.url, action)
	} else {
		submit = landing.url
	}
	if err != nil {
		return nil, err
	}
	method, _ := attr(form, "method")
	req := request{headers: hdrs, follow: true, referer: landing.url.String()}
	if strings.EqualFold(strings.TrimSpace(method), "get") {
		req.url = withQuery(submit, c.sess.encodePairs(append(fields, query...), ""))
	} else {
		req.method = http.MethodPost
		req.url = withQuery(submit, c.sess.encodePairs(query, ""))
		req.body = c.sess.encodePairs(fields, "")
	}
	return c.sess.do(ctx, req)
}

// formFields collects the successful controls of a form as a browser would submit them.
func formFields(form *html.Node) []pair {
	var out []pair
	submitSeen := false
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			name, hasName := attr(n, "name")
			_, disabled := attr(n, "disabled")
			if hasName && name != "" && !disabled {
				switch n.Data {
				case "input":
					typ, _ := attr(n, "type")
					val, _ := attr(n, "value")
					switch strings.ToLower(typ) {
					case "checkbox", "radio":
						if _, checked := attr(n, "checked"); checked {
							if val == "" {
								val = "on"
							}
							out = append(out, pair{name, val})
						}
					case "submit":
						if !submitSeen {
							submitSeen = true
							out = append(out, pair{name, val})
						}
					case "image", "button", "file", "reset":
					default:
						out = append(out, pair{name, val})
					}
				case "select":
					val, first := "", ""
					found := false
					for _, o := range descendants(n, "option") {
						v, ok := attr(o, "value")
						if !ok {
							v = strings.TrimSpace(nodeText(o))
						}
						if first == "" {
							first = v
						}
						if _, sel := attr(o, "selected"); sel {
							val, found = v, true
							break
						}
					}
					if !found {
						val = first
					}
					out = append(out, pair{name, val})
				case "textarea":
					out = append(out, pair{name, nodeText(n)})
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(form)
	return out
}

func descendants(n *html.Node, tag string) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		for ch := x.FirstChild; ch != nil; ch = ch.NextSibling {
			if ch.Type == html.ElementNode && ch.Data == tag {
				out = append(out, ch)
			}
			walk(ch)
		}
	}
	walk(n)
	return out
}

// testLogin requests the login test page: a redirect or a missing selector means the
// session is not valid.
func (c *cardigann) testLogin(ctx context.Context) (bool, error) {
	t := c.def.Login.Test
	vars := c.baseVars()
	u, err := c.pathURL(t.Path, vars)
	if err != nil {
		return false, err
	}
	resp, err := c.sess.do(ctx, request{url: u.String(), referer: c.base.String()})
	if err != nil {
		return false, err
	}
	if resp.location != "" || resp.status >= 400 {
		return false, nil
	}
	if t.Selector == "" {
		return true, nil
	}
	sel, err := expand(t.Selector, vars, nil)
	if err != nil {
		return false, err
	}
	var doc *html.Node
	if c.resp == "json" || c.resp == "xml" {
		// API trackers: a JSON/XML body is a successful check unless it is an error page
		if _, jerr := decodeJSON(resp.body); jerr == nil {
			return true, nil
		}
	}
	if doc, err = parseHTML(resp.body); err != nil {
		return false, err
	}
	n, err := queryFirst(doc, sel)
	if err != nil {
		return false, err
	}
	return n != nil, nil
}
