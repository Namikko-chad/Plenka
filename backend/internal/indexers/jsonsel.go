package indexers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// JSON selectors as used by definitions with "response: type: json":
//   a.b.c        nested keys          files[0].name   array index
//   $            the document root    $.x / $[0].id  path from the root
//   ..title      read from the parent row (rows with "attribute")
//   :has(sel)    keep when sel finds a value        :not(sel)  the opposite
//   :contains(t) keep when the value's text contains t
// Applied to an array (rows), the predicates filter its elements.

type jsonStep struct {
	key   string
	idx   int
	isIdx bool
}

type jsonPred struct {
	kind string // has | not | contains
	sub  *jsonSel
	text string
}

type jsonSel struct {
	up, root bool
	steps    []jsonStep
	preds    []jsonPred
}

type jsonCtx struct {
	root, parent any
}

var jsonSelCache boundedCache

func compileJSONSel(s string) (*jsonSel, error) {
	if v, ok := jsonSelCache.get(s); ok {
		if err, isErr := v.(error); isErr {
			return nil, err
		}
		return v.(*jsonSel), nil
	}
	sel, err := parseJSONSel(s)
	if err != nil {
		err = fmt.Errorf("JSON selector %q: %w", abbrev(s, 80), err)
		jsonSelCache.put(s, err)
		return nil, err
	}
	jsonSelCache.put(s, sel)
	return sel, nil
}

func parseJSONSel(s string) (*jsonSel, error) {
	s = strings.TrimSpace(s)
	sel := &jsonSel{}
	switch {
	case strings.HasPrefix(s, ".."):
		sel.up = true
		s = s[2:]
	case strings.HasPrefix(s, "$"):
		sel.root = true
		s = strings.TrimPrefix(s[1:], ".")
	}
	// the path runs up to the first ':' outside brackets
	end := len(s)
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
		case ':':
			if depth == 0 {
				end = i
				i = len(s)
			}
		}
	}
	path, preds := s[:end], s[end:]
	if path != "" {
		for _, seg := range strings.Split(path, ".") {
			name := seg
			var idxs []int
			for {
				o := strings.IndexByte(name, '[')
				if o < 0 {
					break
				}
				c := strings.IndexByte(name[o:], ']')
				if c < 0 {
					return nil, fmt.Errorf("unclosed [ in %q", seg)
				}
				n, err := strconv.Atoi(strings.TrimSpace(name[o+1 : o+c]))
				if err != nil {
					return nil, fmt.Errorf("bad index in %q", seg)
				}
				idxs = append(idxs, n)
				name = name[:o] + name[o+c+1:]
			}
			if name != "" {
				sel.steps = append(sel.steps, jsonStep{key: name})
			}
			for _, n := range idxs {
				sel.steps = append(sel.steps, jsonStep{idx: n, isIdx: true})
			}
		}
	}
	for preds != "" {
		if preds[0] != ':' {
			return nil, fmt.Errorf("unexpected %q", preds)
		}
		open := strings.IndexByte(preds, '(')
		if open < 0 {
			return nil, fmt.Errorf("pseudo-class without argument in %q", preds)
		}
		kind := strings.ToLower(preds[1:open])
		closeAt := matchParen(preds, open)
		if closeAt < 0 {
			return nil, fmt.Errorf("unbalanced parentheses")
		}
		argText := strings.TrimSpace(preds[open+1 : closeAt])
		preds = preds[closeAt+1:]
		switch kind {
		case "contains":
			sel.preds = append(sel.preds, jsonPred{kind: "contains", text: unquote(argText)})
		case "has", "not":
			sub, err := parseJSONSel(argText)
			if err != nil {
				return nil, err
			}
			sel.preds = append(sel.preds, jsonPred{kind: kind, sub: sub})
		default:
			return nil, fmt.Errorf("unsupported pseudo-class :%s", kind)
		}
	}
	return sel, nil
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

func (s *jsonSel) start(v any, ctx *jsonCtx) any {
	switch {
	case s.root && ctx != nil:
		return ctx.root
	case s.up && ctx != nil && ctx.parent != nil:
		return ctx.parent
	}
	return v
}

func (s *jsonSel) resolve(v any, ctx *jsonCtx) (any, bool) {
	cur := s.start(v, ctx)
	for _, st := range s.steps {
		switch x := cur.(type) {
		case map[string]any:
			if st.isIdx {
				return nil, false
			}
			val, ok := x[st.key]
			if !ok {
				return nil, false
			}
			cur = val
		case []any:
			if !st.isIdx || st.idx < 0 || st.idx >= len(x) {
				return nil, false
			}
			cur = x[st.idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

func (s *jsonSel) passes(e any, ctx *jsonCtx) bool {
	for _, p := range s.preds {
		switch p.kind {
		case "contains":
			if !strings.Contains(jsonString(e), p.text) {
				return false
			}
		case "has":
			if _, ok := p.sub.first(e, ctx); !ok {
				return false
			}
		case "not":
			if _, ok := p.sub.first(e, ctx); ok {
				return false
			}
		}
	}
	return true
}

// first returns the selected value (for fields); null counts as not found.
func (s *jsonSel) first(v any, ctx *jsonCtx) (any, bool) {
	x, ok := s.resolve(v, ctx)
	if !ok || x == nil {
		return nil, false
	}
	if !s.passes(x, ctx) {
		return nil, false
	}
	return x, true
}

// all returns the selected rows: array elements passing the predicates, or the single
// object itself.
func (s *jsonSel) all(v any, ctx *jsonCtx) []any {
	x, ok := s.resolve(v, ctx)
	if !ok || x == nil {
		return nil
	}
	if arr, isArr := x.([]any); isArr {
		var out []any
		for _, e := range arr {
			if s.passes(e, ctx) {
				out = append(out, e)
			}
		}
		return out
	}
	if s.passes(x, ctx) {
		return []any{x}
	}
	return nil
}

func decodeJSON(body []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// jsonString renders a JSON value as text the way definitions expect
// (booleans as True/False, numbers verbatim, objects as compact JSON).
func jsonString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		if x {
			return "True"
		}
		return "False"
	}
	b, _ := json.Marshal(v)
	return string(b)
}
