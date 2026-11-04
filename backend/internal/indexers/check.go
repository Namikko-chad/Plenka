package indexers

import (
	"fmt"
	"strings"

	"github.com/andybalholm/cascadia"
	"github.com/dlclark/regexp2"
)

// checkDefinition compiles everything that can be compiled ahead of time (templates,
// selectors, regular expressions, filter names) and returns the problems found. It
// does not use the compile caches, so checking every definition retains nothing. A
// definition with problems is listed but reports them instead of searching.
func checkDefinition(d *Definition) []string {
	ck := &checker{seen: map[string]bool{}}
	typ := ""
	if r := d.Search.Paths[0].Response; r != nil {
		typ = strings.ToLower(r.Type)
	}
	switch typ {
	case "", "html", "json", "xml":
	default:
		ck.add("response type %q is not supported", typ)
	}

	if lb := d.Login; lb != nil {
		switch strings.ToLower(strings.TrimSpace(lb.Method)) {
		case "", "post", "get", "form", "cookie", "oneurl":
		default:
			ck.add("login method %q is not supported", lb.Method)
		}
		ck.tmpl(lb.Path, lb.SubmitPath)
		ck.pairs(lb.Inputs)
		ck.headers(lb.Headers)
		if lb.Form != "" {
			ck.css(lb.Form)
		}
		if lb.Selectors {
			for _, in := range lb.Inputs {
				ck.css(in.Key)
			}
		}
		for _, nb := range lb.SelectorInputs {
			ck.block(nb.Block, "")
		}
		for _, nb := range lb.GetSelectorInputs {
			ck.block(nb.Block, "")
		}
		ck.errors(lb.Error)
		if lb.Test != nil {
			ck.tmpl(lb.Test.Path)
			ck.css(lb.Test.Selector)
		}
		if lb.Captcha != nil {
			ck.css(lb.Captcha.Selector)
		}
	}

	s := &d.Search
	for _, p := range s.Paths {
		ck.tmpl(p.Path, p.Method)
		ck.pairs(p.Inputs)
	}
	ck.pairs(s.Inputs)
	ck.headers(s.Headers)
	ck.filters(s.KeywordsFilters)
	ck.filters(s.PreprocessingFilters)
	ck.errors(s.Error)
	ck.tmpl(s.Rows.Selector)
	if typ == "json" {
		ck.jsonSel(s.Rows.Selector)
		if s.Rows.Attribute != "" {
			ck.jsonSel(s.Rows.Attribute)
		}
		if s.Rows.Count != nil {
			ck.jsonSel(s.Rows.Count.Selector)
		}
	} else {
		ck.css(s.Rows.Selector)
		if s.Rows.DateHeaders != nil {
			ck.css(s.Rows.DateHeaders.Selector)
			ck.filters(s.Rows.DateHeaders.Filters)
		}
	}
	ck.filters(s.Rows.Filters)
	hasTitle := false
	for _, f := range s.Fields {
		if strings.HasPrefix(f.Name, "title") {
			hasTitle = true
		}
		ck.block(f.Block, typ)
	}
	if !hasTitle {
		ck.add("no title field")
	}

	if dl := d.Download; dl != nil {
		ck.headers(dl.Headers)
		if b := dl.Before; b != nil {
			ck.tmpl(b.Path)
			ck.pairs(b.Inputs)
			if b.PathSelector != nil {
				ck.block(*b.PathSelector, "")
			}
		}
		for _, sb := range dl.Selectors {
			ck.block(sb, "")
		}
		if ih := dl.Infohash; ih != nil {
			ck.block(ih.Hash, "")
			ck.block(ih.Title, "")
		}
	}
	return ck.issues
}

type checker struct {
	issues []string
	seen   map[string]bool
}

func (c *checker) add(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if !c.seen[msg] && len(c.issues) < 10 {
		c.seen[msg] = true
		c.issues = append(c.issues, msg)
	}
}

func (c *checker) tmpl(srcs ...string) {
	for _, s := range srcs {
		if !strings.Contains(s, "{{") {
			continue
		}
		if _, err := parseTmpl(s); err != nil {
			c.add("template %q: %v", abbrev(s, 80), err)
		}
	}
}

func (c *checker) pairs(ps pairs) {
	for _, p := range ps {
		c.tmpl(p.Value)
	}
}

func (c *checker) headers(h headers) {
	for _, x := range h {
		c.tmpl(x.Values...)
	}
}

func (c *checker) css(sel string) {
	c.tmpl(sel)
	if sel == "" || strings.Contains(sel, "{{") {
		return
	}
	if _, err := cascadia.ParseGroup(normalizeCSS(sel)); err != nil {
		c.add("selector %q: %v", abbrev(sel, 80), err)
	}
}

func (c *checker) jsonSel(sel string) {
	c.tmpl(sel)
	if sel == "" || strings.Contains(sel, "{{") {
		return
	}
	if _, err := parseJSONSel(sel); err != nil {
		c.add("JSON selector %q: %v", abbrev(sel, 80), err)
	}
}

func (c *checker) errors(eb []ErrorBlock) {
	for _, e := range eb {
		c.css(e.Selector)
		if e.Message != nil {
			c.block(*e.Message, "")
		}
	}
}

func (c *checker) block(b SelectorBlock, typ string) {
	if b.Text != nil {
		c.tmpl(*b.Text)
	}
	if b.Default != nil {
		c.tmpl(*b.Default)
	}
	if b.Selector != "" {
		if typ == "json" {
			c.jsonSel(b.Selector)
		} else {
			c.css(b.Selector)
		}
	}
	if b.Remove != "" {
		c.css(b.Remove)
	}
	for _, cs := range b.Case {
		c.tmpl(cs.Value)
		if typ != "json" && cs.Key != "*" {
			c.css(cs.Key)
		}
	}
	c.filters(b.Filters)
}

func (c *checker) filters(fs []FilterDef) {
	for _, f := range fs {
		name := strings.ToLower(f.Name)
		if !knownFilters[name] {
			c.add("filter %q is not supported", f.Name)
			continue
		}
		for _, a := range f.Args {
			c.tmpl(a)
		}
		if (name == "re_replace" || name == "regexp") && len(f.Args) > 0 && !strings.Contains(f.Args[0], "{{") {
			if _, err := regexp2.Compile(translateBlocks(f.Args[0]), regexp2.None); err != nil {
				c.add("regex %q: %v", abbrev(f.Args[0], 60), err)
			}
		}
	}
}
