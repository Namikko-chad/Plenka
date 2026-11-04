package indexers

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/andybalholm/cascadia"
	"github.com/dlclark/regexp2"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ---- regular expressions (.NET flavour, as written in the definitions) ----

var regexCache boundedCache

func compileRegex(pattern string) (*regexp2.Regexp, error) {
	if v, ok := regexCache.get(pattern); ok {
		if err, isErr := v.(error); isErr {
			return nil, err
		}
		return v.(*regexp2.Regexp), nil
	}
	re, err := regexp2.Compile(translateBlocks(pattern), regexp2.None)
	if err != nil {
		err = fmt.Errorf("regex %q: %w", abbrev(pattern, 60), err)
		regexCache.put(pattern, err)
		return nil, err
	}
	re.MatchTimeout = 2 * time.Second
	regexCache.put(pattern, re)
	return re, nil
}

// .NET named Unicode blocks (\p{IsCyrillic}) are not known to regexp2; they are
// rewritten as code point ranges.
var unicodeBlocks = map[string]string{
	"IsBasicLatin": `\u0000-\u007F`, "IsLatin1Supplement": `\u0080-ÿ`,
	"IsLatinExtended-A": `Ā-ſ`, "IsLatinExtended-B": `ƀ-ɏ`,
	"IsGreek": `Ͱ-Ͽ`, "IsGreekandCoptic": `Ͱ-Ͽ`,
	"IsCyrillic": `Ѐ-ӿ`, "IsCyrillicSupplement": `Ԁ-ԯ`,
	"IsArmenian": `԰-֏`, "IsHebrew": `֐-׿`, "IsArabic": `؀-ۿ`,
	"IsThai": `฀-๿`, "IsHiragana": `぀-ゟ`, "IsKatakana": `゠-ヿ`,
	"IsCJKUnifiedIdeographs": `一-鿿`, "IsHangulSyllables": `가-힯`,
}

func translateBlocks(p string) string {
	if !strings.Contains(p, `{Is`) {
		return p
	}
	var b strings.Builder
	inClass := false
	for i := 0; i < len(p); i++ {
		c := p[i]
		if c == '\\' && i+1 < len(p) {
			if (p[i+1] == 'p' || p[i+1] == 'P') && i+2 < len(p) && p[i+2] == '{' {
				if end := strings.IndexByte(p[i+3:], '}'); end >= 0 {
					name := p[i+3 : i+3+end]
					if r, ok := unicodeBlocks[name]; ok {
						neg := p[i+1] == 'P'
						switch {
						case inClass && !neg:
							b.WriteString(r)
							i += 3 + end
							continue
						case !inClass && neg:
							b.WriteString("[^" + r + "]")
							i += 3 + end
							continue
						case !inClass:
							b.WriteString("[" + r + "]")
							i += 3 + end
							continue
						}
					}
				}
			}
			b.WriteByte(c)
			b.WriteByte(p[i+1])
			i++
			continue
		}
		if c == '[' && !inClass {
			inClass = true
		} else if c == ']' && inClass {
			inClass = false
		}
		b.WriteByte(c)
	}
	return b.String()
}

// ---- CSS selectors ----

var cssCache boundedCache

// compileCSS compiles a selector group. Definitions are written for a browser-grade
// selector engine; a few spellings are normalised for cascadia first.
func compileCSS(sel string) (cascadia.SelectorGroup, error) {
	if v, ok := cssCache.get(sel); ok {
		if err, isErr := v.(error); isErr {
			return nil, err
		}
		return v.(cascadia.SelectorGroup), nil
	}
	g, err := cascadia.ParseGroup(normalizeCSS(sel))
	if err != nil {
		err = fmt.Errorf("selector %q: %w", abbrev(sel, 80), err)
		cssCache.put(sel, err)
		return nil, err
	}
	cssCache.put(sel, g)
	return g, nil
}

// normalizeCSS rewrites constructs cascadia does not accept:
//   - a leading combinator inside :has(> a) is accepted by cascadia already;
//   - ":contains(text with spaces)" without quotes is quoted;
//   - ":scope" (the row or page the selector is applied to) becomes an attribute
//     selector; the context node carries that attribute while it is queried;
//   - unquoted attribute values that are not identifiers ([border=1]) are quoted.
var unquotedAttr = regexp.MustCompile(`\[\s*([\w:-]+)\s*([~|^$*]?=)\s*([^\]"'\s]+)\s*\]`)

func normalizeCSS(sel string) string {
	s := strings.TrimSpace(sel)
	var b strings.Builder
	for i := 0; i < len(s); {
		if hasPrefixFold(s[i:], ":contains(") || hasPrefixFold(s[i:], ":containsown(") {
			open := strings.IndexByte(s[i:], '(') + i
			end := matchParen(s, open)
			if end < 0 {
				b.WriteString(s[i:])
				break
			}
			arg := strings.TrimSpace(s[open+1 : end])
			b.WriteString(s[i : open+1])
			if len(arg) >= 2 && (arg[0] == '"' || arg[0] == '\'') && arg[len(arg)-1] == arg[0] {
				b.WriteString(arg)
			} else {
				b.WriteString(quoteCSS(unescapeCSS(arg)))
			}
			b.WriteByte(')')
			i = end + 1
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	out := unquotedAttr.ReplaceAllString(b.String(), `[$1$2"$3"]`)
	out = strings.ReplaceAll(out, ":scope", "["+scopeAttr+"]")
	return out
}

func hasPrefixFold(s, p string) bool {
	return len(s) >= len(p) && strings.EqualFold(s[:len(p)], p)
}

// matchParen returns the index of the ')' matching s[open] == '(' (quote-aware).
func matchParen(s string, open int) int {
	depth := 0
	var q byte
	for i := open; i < len(s); i++ {
		c := s[i]
		if q != 0 {
			if c == '\\' {
				i++
			} else if c == q {
				q = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			q = c
		case '\\':
			i++
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// unescapeCSS resolves CSS escapes (\00a0, \:) in an unquoted argument.
func unescapeCSS(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		j := i + 1
		for j < len(s) && j < i+7 && isHex(s[j]) {
			j++
		}
		if j > i+1 {
			var r rune
			fmt.Sscanf(s[i+1:j], "%x", &r)
			b.WriteRune(r)
			if j < len(s) && s[j] == ' ' {
				j++
			}
			i = j - 1
			continue
		}
		b.WriteByte(s[i+1])
		i++
	}
	return b.String()
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func quoteCSS(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

const scopeAttr = "data-plenka-scope"

// withScope marks n as the :scope element while fn runs (documents are per request).
func withScope(n *html.Node, sel string, fn func()) {
	if !strings.Contains(sel, ":scope") || n.Type != html.ElementNode {
		fn()
		return
	}
	n.Attr = append(n.Attr, html.Attribute{Key: scopeAttr})
	fn()
	for i, a := range n.Attr {
		if a.Key == scopeAttr {
			n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
			break
		}
	}
}

// queryFirst returns the first match of sel inside n (descendants only), or nil.
func queryFirst(n *html.Node, sel string) (*html.Node, error) {
	g, err := compileCSS(sel)
	if err != nil {
		return nil, err
	}
	var out *html.Node
	withScope(n, sel, func() { out = cascadia.Query(n, g) })
	return out, nil
}

func queryAll(n *html.Node, sel string) ([]*html.Node, error) {
	g, err := compileCSS(sel)
	if err != nil {
		return nil, err
	}
	var out []*html.Node
	withScope(n, sel, func() { out = cascadia.QueryAll(n, g) })
	return out, nil
}

// matchesOrContains reports whether n itself matches sel or has a matching descendant.
func matchesOrContains(n *html.Node, sel string) (bool, error) {
	g, err := compileCSS(sel)
	if err != nil {
		return false, err
	}
	found := false
	withScope(n, sel, func() { found = g.Match(n) || cascadia.Query(n, g) != nil })
	return found, nil
}

// ---- documents ----

func parseHTML(body []byte) (*html.Node, error) {
	return html.Parse(bytes.NewReader(body))
}

// parseXML builds an html.Node tree from XML so that the same CSS selectors work.
// Element and attribute names are lower-cased (selectors are case-insensitive) and
// namespace prefixes are kept as "prefix:name" for elements, dropped for attributes.
func parseXML(body []byte) (*html.Node, error) {
	doc := &html.Node{Type: html.DocumentNode}
	cur := doc
	dec := xml.NewDecoder(bytes.NewReader(body))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity
	dec.CharsetReader = func(_ string, r io.Reader) (io.Reader, error) { return r, nil }
	for {
		tok, err := dec.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			if cur != doc || doc.FirstChild != nil {
				break // keep what was parsed
			}
			return nil, fmt.Errorf("invalid XML: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := strings.ToLower(t.Name.Local)
			if t.Name.Space != "" {
				name = strings.ToLower(t.Name.Space) + ":" + name
			}
			n := &html.Node{Type: html.ElementNode, Data: name, DataAtom: atom.Lookup([]byte(name))}
			for _, a := range t.Attr {
				n.Attr = append(n.Attr, html.Attribute{Key: strings.ToLower(a.Name.Local), Val: a.Value})
			}
			cur.AppendChild(n)
			cur = n
		case xml.EndElement:
			if cur.Parent != nil {
				cur = cur.Parent
			}
		case xml.CharData:
			cur.AppendChild(&html.Node{Type: html.TextNode, Data: string(t)})
		}
	}
	return doc, nil
}

// nodeText is the concatenated text content of n.
func nodeText(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		switch x.Type {
		case html.TextNode:
			b.WriteString(x.Data)
		case html.ElementNode:
			if x.DataAtom == atom.Br {
				b.WriteByte('\n')
			}
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// attr returns the attribute value (case-insensitive name) and whether it exists.
func attr(n *html.Node, name string) (string, bool) {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val, true
		}
	}
	return "", false
}

// removeMatches detaches every descendant of n matching sel.
func removeMatches(n *html.Node, sel string) error {
	nodes, err := queryAll(n, sel)
	if err != nil {
		return err
	}
	for _, x := range nodes {
		if x.Parent != nil {
			x.Parent.RemoveChild(x)
		}
	}
	return nil
}
