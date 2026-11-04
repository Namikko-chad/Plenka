package indexers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// knownFilters lists the filter names the engine implements. "andmatch" is a row filter.
var knownFilters = map[string]bool{
	"querystring": true, "prepend": true, "append": true, "replace": true, "re_replace": true,
	"split": true, "trim": true, "regexp": true, "dateparse": true, "timeparse": true,
	"timeago": true, "reltime": true, "fuzzytime": true, "urldecode": true, "urlencode": true,
	"htmldecode": true, "htmlencode": true, "tolower": true, "toupper": true, "diacritics": true,
	"jsonjoinarray": true, "hexdump": true, "strdump": true, "validfilename": true,
	"validate": true, "base64decode": true, "base64encode": true, "andmatch": true,
}

// applyFilters runs a chain. Template expressions in arguments are expanded with vars.
func applyFilters(value string, chain []FilterDef, vars tmplVars, logID string) (string, error) {
	for _, f := range chain {
		args := make([]string, len(f.Args))
		for i, a := range f.Args {
			x, err := expand(a, vars, nil)
			if err != nil {
				return "", err
			}
			args[i] = x
		}
		var err error
		value, err = applyFilter(value, strings.ToLower(f.Name), args, logID)
		if err != nil {
			return "", fmt.Errorf("filter %s: %w", f.Name, err)
		}
	}
	return value, nil
}

func arg(a []string, i int) string {
	if i < len(a) {
		return a[i]
	}
	return ""
}

func applyFilter(v, name string, args []string, logID string) (string, error) {
	switch name {
	case "querystring":
		return queryParam(v, arg(args, 0)), nil
	case "prepend":
		return arg(args, 0) + v, nil
	case "append":
		return v + arg(args, 0), nil
	case "replace":
		if len(args) < 1 || args[0] == "" {
			return v, nil
		}
		return strings.ReplaceAll(v, args[0], arg(args, 1)), nil
	case "re_replace":
		re, err := compileRegex(arg(args, 0))
		if err != nil {
			return "", err
		}
		return re.Replace(v, arg(args, 1), -1, -1)
	case "regexp":
		re, err := compileRegex(arg(args, 0))
		if err != nil {
			return "", err
		}
		m, err := re.FindStringMatch(v)
		if err != nil || m == nil {
			return "", err
		}
		if g := m.GroupByNumber(1); g != nil && len(m.Groups()) > 1 {
			return g.String(), nil
		}
		return m.String(), nil
	case "split":
		sep := arg(args, 0)
		idx, err := strconv.Atoi(strings.TrimSpace(arg(args, 1)))
		if err != nil {
			return "", fmt.Errorf("bad index %q", arg(args, 1))
		}
		parts := strings.Split(v, sep)
		if idx < 0 {
			idx += len(parts)
		}
		if idx < 0 || idx >= len(parts) {
			return "", nil
		}
		return parts[idx], nil
	case "trim":
		if len(args) == 0 || args[0] == "" {
			return strings.TrimFunc(v, unicode.IsSpace), nil
		}
		return strings.Trim(v, args[0]), nil
	case "dateparse", "timeparse":
		t, err := parseDotNetDate(strings.TrimSpace(v), arg(args, 0))
		if err != nil {
			// leave the value for the final date conversion, which is more lenient
			if t2, err2 := parseFuzzy(v, false); err2 == nil {
				return t2.Format(dateOut), nil
			}
			return v, nil
		}
		return t.Format(dateOut), nil
	case "timeago", "reltime":
		t, err := parseTimeAgo(v)
		if err != nil {
			return v, nil
		}
		return t.Format(dateOut), nil
	case "fuzzytime":
		t, err := parseFuzzy(v, strings.Contains(strings.ToUpper(arg(args, 0)), "UK"))
		if err != nil {
			return v, nil
		}
		return t.Format(dateOut), nil
	case "urldecode":
		if s, err := url.QueryUnescape(v); err == nil {
			return s, nil
		}
		return v, nil
	case "urlencode":
		return url.QueryEscape(v), nil
	case "htmldecode":
		return html.UnescapeString(v), nil
	case "htmlencode":
		return html.EscapeString(v), nil
	case "tolower":
		return strings.ToLower(v), nil
	case "toupper":
		return strings.ToUpper(v), nil
	case "diacritics":
		return stripDiacritics(v), nil
	case "jsonjoinarray":
		return jsonJoin(v, arg(args, 0), arg(args, 1))
	case "hexdump":
		slog.Debug("indexers: hexdump", "indexer", logID, "value", fmt.Sprintf("% x", v))
		return v, nil
	case "strdump":
		slog.Debug("indexers: strdump", "indexer", logID, "tag", arg(args, 0), "value", v)
		return v, nil
	case "validfilename":
		return strings.Map(func(r rune) rune {
			if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
				return -1
			}
			return r
		}, v), nil
	case "validate":
		return validateWords(v, arg(args, 0)), nil
	case "base64decode":
		for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
			if b, err := enc.DecodeString(strings.TrimSpace(v)); err == nil {
				return string(b), nil
			}
		}
		return "", fmt.Errorf("invalid base64")
	case "base64encode":
		return base64.StdEncoding.EncodeToString([]byte(v)), nil
	case "andmatch":
		return v, nil // row filter, applied by the search runner
	}
	return "", fmt.Errorf("unsupported filter %q", name)
}

// queryParam extracts a query-string argument from a (possibly relative) URL.
func queryParam(raw, key string) string {
	q := raw
	if i := strings.IndexByte(raw, '?'); i >= 0 {
		q = raw[i+1:]
	}
	if i := strings.IndexByte(q, '#'); i >= 0 {
		q = q[:i]
	}
	for _, part := range strings.FieldsFunc(q, func(r rune) bool { return r == '&' || r == ';' }) {
		k, val, _ := strings.Cut(part, "=")
		if uk, err := url.QueryUnescape(k); err == nil {
			k = uk
		}
		if k == key {
			if uv, err := url.QueryUnescape(val); err == nil {
				return uv
			}
			return val
		}
	}
	return ""
}

func stripDiacritics(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		return s
	}
	return out
}

// validateWords keeps the words of v that appear in the allowed list (comma separated,
// '_' standing for a space), returned lower-case and comma separated.
func validateWords(v, allowed string) string {
	allow := map[string]bool{}
	for _, w := range strings.Split(allowed, ",") {
		w = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(w, "_", " ")))
		if w != "" {
			allow[w] = true
		}
	}
	words := strings.FieldsFunc(v, func(r rune) bool { return strings.ContainsRune(", /.)(;[]\"|:", r) })
	var out []string
	seen := map[string]bool{}
	for _, w := range words {
		w = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(w, "_", " ")))
		if allow[w] && !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	return strings.Join(out, ", ")
}

// jsonJoin evaluates a simple JSONPath ("$.a.b", "$.list[0]") and joins an array result.
func jsonJoin(v, path, sep string) (string, error) {
	var root any
	if err := json.Unmarshal([]byte(v), &root); err != nil {
		return "", fmt.Errorf("input is not JSON: %w", err)
	}
	p := strings.TrimPrefix(strings.TrimPrefix(path, "$"), ".")
	sel, err := parseJSONSel(p)
	if err != nil {
		return "", err
	}
	x, ok := sel.first(root, nil)
	if !ok {
		return "", nil
	}
	if arr, isArr := x.([]any); isArr {
		parts := make([]string, 0, len(arr))
		for _, e := range arr {
			parts = append(parts, jsonString(e))
		}
		return strings.Join(parts, sep), nil
	}
	return jsonString(x), nil
}
