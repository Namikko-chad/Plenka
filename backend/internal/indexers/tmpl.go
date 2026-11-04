package indexers

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// A small template language compatible with what Cardigann definitions use:
//   {{ .Var.Path }}  {{ . }}  {{ $i }}
//   {{ if X }}…{{ else if Y }}…{{ else }}…{{ end }}
//   {{ range X }}…{{ end }}  {{ range $i, $e := X }}…{{ end }}
//   functions: or, and, not, eq, ne, join, re_replace (Go-template-like calling style,
//   arguments may be parenthesised sub-expressions or "quoted strings").
// Values are nil, string or []string. Empty string, nil and empty list are false.
// Variable path segments may contain '-' (setting names like cat-id are common).

type tmplVars map[string]any

type tmpl struct {
	nodes []tnode
	plain bool
	src   string
}

type tnode interface{}

type (
	textNode  string
	printNode struct{ e texpr }
	ifNode    struct {
		cond      texpr
		then, els []tnode
	}
	rangeNode struct {
		idxVar, elemVar string
		list            texpr
		body            []tnode
	}
)

type texpr interface{}

type (
	varExpr   []string // segments after the dot; empty means "."
	localExpr string   // $name
	litExpr   string
	callExpr  struct {
		fn   string
		args []texpr
	}
)

var knownFuncs = map[string]bool{"or": true, "and": true, "not": true, "eq": true, "ne": true, "join": true, "re_replace": true}

// ---- cache ----

// boundedCache memoises compiled artefacts (templates, regexes, selectors) by source
// text. Sources come from definitions, so the set is finite; the bound is a safety net.
type boundedCache struct {
	mu sync.RWMutex
	m  map[string]any
}

const cacheLimit = 20000

func (c *boundedCache) get(k string) (any, bool) {
	c.mu.RLock()
	v, ok := c.m[k]
	c.mu.RUnlock()
	return v, ok
}

func (c *boundedCache) put(k string, v any) {
	c.mu.Lock()
	if c.m == nil || len(c.m) >= cacheLimit {
		c.m = map[string]any{}
	}
	c.m[k] = v
	c.mu.Unlock()
}

var tmplCache boundedCache

func compileTmpl(src string) (*tmpl, error) {
	if !strings.Contains(src, "{{") {
		return &tmpl{plain: true, src: src}, nil
	}
	if v, ok := tmplCache.get(src); ok {
		if err, isErr := v.(error); isErr {
			return nil, err
		}
		return v.(*tmpl), nil
	}
	t, err := parseTmpl(src)
	if err != nil {
		err = fmt.Errorf("template %q: %w", abbrev(src, 80), err)
		tmplCache.put(src, err)
		return nil, err
	}
	tmplCache.put(src, t)
	return t, nil
}

// expand compiles and runs a template. escape (optional) is applied to every value the
// template prints (not to its literal text).
func expand(src string, vars tmplVars, escape func(string) string) (string, error) {
	t, err := compileTmpl(src)
	if err != nil {
		return "", err
	}
	if t.plain {
		return src, nil
	}
	var b strings.Builder
	st := &tstate{vars: vars, escape: escape}
	if err := st.run(&b, t.nodes); err != nil {
		return "", fmt.Errorf("template %q: %w", abbrev(src, 80), err)
	}
	return b.String(), nil
}

func abbrev(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// ---- parsing ----

type actionTok struct {
	kind string // "ident", "var", "local", "str", "(", ")", ",", ":="
	val  string
}

func parseTmpl(src string) (*tmpl, error) {
	// split into text and action chunks
	type chunk struct {
		action bool
		s      string
	}
	var chunks []chunk
	rest := src
	for {
		i := strings.Index(rest, "{{")
		if i < 0 {
			if rest != "" {
				chunks = append(chunks, chunk{false, rest})
			}
			break
		}
		if i > 0 {
			chunks = append(chunks, chunk{false, rest[:i]})
		}
		end, err := actionEnd(rest[i+2:])
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk{true, rest[i+2 : i+2+end]})
		rest = rest[i+2+end+2:]
	}

	// build the tree with an explicit stack
	type frame struct {
		node   tnode // *ifNode / *rangeNode / nil for the root
		inElse bool
		list   *[]tnode
	}
	var root []tnode
	stack := []*frame{{list: &root}}
	top := func() *frame { return stack[len(stack)-1] }
	for _, c := range chunks {
		if !c.action {
			*top().list = append(*top().list, textNode(c.s))
			continue
		}
		body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(c.s, "-"), "-"))
		toks, err := lexAction(body)
		if err != nil {
			return nil, err
		}
		if len(toks) == 0 {
			return nil, errors.New("empty action")
		}
		kw := ""
		if toks[0].kind == "ident" {
			kw = toks[0].val
		}
		switch kw {
		case "if":
			e, err := parseExpr(toks[1:])
			if err != nil {
				return nil, err
			}
			n := &ifNode{cond: e}
			*top().list = append(*top().list, n)
			stack = append(stack, &frame{node: n, list: &n.then})
		case "else":
			f := top()
			n, ok := f.node.(*ifNode)
			if !ok || f.inElse {
				return nil, errors.New("unexpected else")
			}
			f.inElse = true
			f.list = &n.els
			if len(toks) > 1 && toks[1].kind == "ident" && toks[1].val == "if" {
				e, err := parseExpr(toks[2:])
				if err != nil {
					return nil, err
				}
				inner := &ifNode{cond: e}
				n.els = append(n.els, inner)
				f.node, f.inElse, f.list = inner, false, &inner.then
			} else if len(toks) > 1 {
				return nil, errors.New("unexpected tokens after else")
			}
		case "end":
			if len(stack) == 1 {
				return nil, errors.New("unexpected end")
			}
			stack = stack[:len(stack)-1]
		case "range":
			n := &rangeNode{}
			rt := toks[1:]
			if i := indexTok(rt, ":="); i >= 0 {
				decl := rt[:i]
				switch {
				case len(decl) == 1 && decl[0].kind == "local":
					n.elemVar = decl[0].val
				case len(decl) == 3 && decl[0].kind == "local" && decl[1].kind == "," && decl[2].kind == "local":
					n.idxVar, n.elemVar = decl[0].val, decl[2].val
				default:
					return nil, errors.New("bad range declaration")
				}
				rt = rt[i+1:]
			}
			e, err := parseExpr(rt)
			if err != nil {
				return nil, err
			}
			n.list = e
			*top().list = append(*top().list, n)
			stack = append(stack, &frame{node: n, list: &n.body})
		default:
			e, err := parseExpr(toks)
			if err != nil {
				return nil, err
			}
			*top().list = append(*top().list, printNode{e})
		}
	}
	if len(stack) != 1 {
		return nil, errors.New("missing end")
	}
	return &tmpl{nodes: root, src: src}, nil
}

// actionEnd finds the "}}" closing an action, skipping quoted strings.
func actionEnd(s string) (int, error) {
	inStr, escStr := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escStr { // a \"string\" (backslash-escaped quotes left in a plain scalar)
			if c == '\\' && i+1 < len(s) && s[i+1] == '"' {
				escStr = false
				i++
			}
			continue
		}
		if c == '\\' && i+1 < len(s) && s[i+1] == '"' && !inStr {
			escStr = true
			i++
			continue
		}
		if inStr {
			if c == '\\' {
				i++
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			continue
		}
		if c == '}' && i+1 < len(s) && s[i+1] == '}' {
			return i, nil
		}
	}
	return 0, errors.New("unclosed action")
}

func indexTok(t []actionTok, kind string) int {
	for i, x := range t {
		if x.kind == kind {
			return i
		}
	}
	return -1
}

func isIdentByte(c byte) bool {
	return c == '_' || c == '-' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func lexAction(s string) ([]actionTok, error) {
	var out []actionTok
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '(' || c == ')' || c == ',':
			out = append(out, actionTok{kind: string(c)})
			i++
		case c == ':' && i+1 < len(s) && s[i+1] == '=':
			out = append(out, actionTok{kind: ":="})
			i += 2
		case c == '"' || (c == '\\' && i+1 < len(s) && s[i+1] == '"'):
			escapedQuotes := c == '\\'
			if escapedQuotes {
				i++
			}
			i++
			var b strings.Builder
			closed := false
			for i < len(s) {
				ch := s[i]
				if escapedQuotes && ch == '\\' && i+1 < len(s) && s[i+1] == '"' {
					i += 2
					closed = true
					break
				}
				if !escapedQuotes && ch == '"' {
					i++
					closed = true
					break
				}
				if !escapedQuotes && ch == '\\' && i+1 < len(s) {
					i++
					switch s[i] {
					case 'n':
						b.WriteByte('\n')
					case 't':
						b.WriteByte('\t')
					default:
						b.WriteByte(s[i])
					}
					i++
					continue
				}
				b.WriteByte(ch)
				i++
			}
			if !closed {
				return nil, errors.New("unterminated string")
			}
			out = append(out, actionTok{kind: "str", val: b.String()})
		case c == '.':
			j := i + 1
			for j < len(s) && (isIdentByte(s[j]) || s[j] == '.') {
				j++
			}
			out = append(out, actionTok{kind: "var", val: s[i+1 : j]})
			i = j
		case c == '$':
			j := i + 1
			for j < len(s) && isIdentByte(s[j]) {
				j++
			}
			out = append(out, actionTok{kind: "local", val: s[i:j]})
			i = j
		case isIdentByte(c):
			j := i
			for j < len(s) && (isIdentByte(s[j]) || s[j] == '.') {
				j++
			}
			w := s[i:j]
			if (c >= '0' && c <= '9') || c == '-' {
				out = append(out, actionTok{kind: "str", val: w})
			} else {
				out = append(out, actionTok{kind: "ident", val: w})
			}
			i = j
		default:
			return nil, fmt.Errorf("unexpected character %q", c)
		}
	}
	return out, nil
}

// parseExpr parses a full command: either a single operand or "fn arg…".
func parseExpr(toks []actionTok) (texpr, error) {
	e, rest, err := parseCommand(toks)
	if err != nil {
		return nil, err
	}
	if len(rest) > 0 {
		return nil, errors.New("unexpected tokens")
	}
	return e, nil
}

func parseCommand(toks []actionTok) (texpr, []actionTok, error) {
	if len(toks) == 0 {
		return nil, nil, errors.New("missing expression")
	}
	if toks[0].kind == "ident" {
		fn := toks[0].val
		if !knownFuncs[fn] {
			return nil, nil, fmt.Errorf("unsupported function %q", fn)
		}
		var args []texpr
		rest := toks[1:]
		for len(rest) > 0 && rest[0].kind != ")" {
			a, r, err := parseOperand(rest)
			if err != nil {
				return nil, nil, err
			}
			args = append(args, a)
			rest = r
		}
		return callExpr{fn, args}, rest, nil
	}
	return parseOperand(toks)
}

func parseOperand(toks []actionTok) (texpr, []actionTok, error) {
	t := toks[0]
	switch t.kind {
	case "var":
		if t.val == "" {
			return varExpr(nil), toks[1:], nil
		}
		return varExpr(strings.Split(t.val, ".")), toks[1:], nil
	case "local":
		return localExpr(t.val), toks[1:], nil
	case "str":
		return litExpr(t.val), toks[1:], nil
	case "(":
		e, rest, err := parseCommand(toks[1:])
		if err != nil {
			return nil, nil, err
		}
		if len(rest) == 0 || rest[0].kind != ")" {
			return nil, nil, errors.New("missing )")
		}
		return e, rest[1:], nil
	case "ident":
		if knownFuncs[t.val] { // function call without parentheses as an argument: not allowed
			return nil, nil, fmt.Errorf("function %q used as a value", t.val)
		}
		return nil, nil, fmt.Errorf("unknown identifier %q", t.val)
	}
	return nil, nil, fmt.Errorf("unexpected %q", t.kind)
}

// ---- execution ----

type tstate struct {
	vars   tmplVars
	dot    any
	locals map[string]any
	escape func(string) string
}

func (st *tstate) run(b *strings.Builder, nodes []tnode) error {
	for _, n := range nodes {
		switch n := n.(type) {
		case textNode:
			b.WriteString(string(n))
		case printNode:
			v, err := st.eval(n.e)
			if err != nil {
				return err
			}
			s := valueString(v)
			if st.escape != nil {
				s = st.escape(s)
			}
			b.WriteString(s)
		case *ifNode:
			v, err := st.eval(n.cond)
			if err != nil {
				return err
			}
			branch := n.els
			if truthy(v) {
				branch = n.then
			}
			if err := st.run(b, branch); err != nil {
				return err
			}
		case *rangeNode:
			v, err := st.eval(n.list)
			if err != nil {
				return err
			}
			var items []string
			switch x := v.(type) {
			case []string:
				items = x
			case string:
				if x != "" {
					items = []string{x}
				}
			}
			saved, savedLocals := st.dot, st.locals
			for i, it := range items {
				st.dot = it
				if n.idxVar != "" || n.elemVar != "" {
					l := map[string]any{}
					for k, v := range savedLocals {
						l[k] = v
					}
					if n.idxVar != "" {
						l[n.idxVar] = fmt.Sprint(i)
					}
					if n.elemVar != "" {
						l[n.elemVar] = it
					}
					st.locals = l
				}
				if err := st.run(b, n.body); err != nil {
					return err
				}
			}
			st.dot, st.locals = saved, savedLocals
		}
	}
	return nil
}

func (st *tstate) eval(e texpr) (any, error) {
	switch e := e.(type) {
	case litExpr:
		return string(e), nil
	case localExpr:
		return st.locals[string(e)], nil
	case varExpr:
		if len(e) == 0 {
			return st.dot, nil
		}
		var cur any = map[string]any(st.vars)
		for _, seg := range e {
			m, ok := cur.(map[string]any)
			if !ok {
				return nil, nil
			}
			cur = m[seg]
		}
		return cur, nil
	case callExpr:
		args := make([]any, len(e.args))
		for i, a := range e.args {
			v, err := st.eval(a)
			if err != nil {
				return nil, err
			}
			args[i] = v
		}
		return callFunc(e.fn, args)
	}
	return nil, errors.New("bad expression")
}

func boolVal(b bool) any {
	if b {
		return "True"
	}
	return nil
}

func callFunc(fn string, args []any) (any, error) {
	switch fn {
	case "or":
		var last any
		for _, a := range args {
			if truthy(a) {
				return a, nil
			}
			last = a
		}
		return last, nil
	case "and":
		var last any
		for _, a := range args {
			if !truthy(a) {
				return a, nil
			}
			last = a
		}
		return last, nil
	case "not":
		if len(args) != 1 {
			return nil, errors.New("not takes one argument")
		}
		return boolVal(!truthy(args[0])), nil
	case "eq", "ne":
		if len(args) < 2 {
			return nil, fmt.Errorf("%s takes two arguments", fn)
		}
		a := valueString(args[0])
		match := false
		for _, b := range args[1:] {
			if a == valueString(b) {
				match = true
				break
			}
		}
		if fn == "ne" {
			match = !match
		}
		return boolVal(match), nil
	case "join":
		if len(args) != 2 {
			return nil, errors.New("join takes two arguments")
		}
		sep := valueString(args[1])
		switch x := args[0].(type) {
		case []string:
			return strings.Join(x, sep), nil
		default:
			return valueString(x), nil
		}
	case "re_replace":
		if len(args) != 3 {
			return nil, errors.New("re_replace takes three arguments")
		}
		re, err := compileRegex(valueString(args[1]))
		if err != nil {
			return nil, err
		}
		out, err := re.Replace(valueString(args[0]), valueString(args[2]), -1, -1)
		if err != nil {
			return nil, err
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported function %q", fn)
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case string:
		return x != ""
	case []string:
		return len(x) > 0
	case bool:
		return x
	}
	return true
}

func valueString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []string:
		return strings.Join(x, ",")
	case bool:
		if x {
			return "True"
		}
		return ""
	}
	return fmt.Sprint(v)
}
