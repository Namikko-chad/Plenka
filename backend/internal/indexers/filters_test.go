package indexers

import (
	"testing"
	"time"
)

// fixedNow pins "now" for date tests: Wednesday 2026-06-17 12:00:00 UTC.
func fixedNow(t *testing.T) time.Time {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	old := nowFunc
	nowFunc = func() time.Time { return now }
	t.Cleanup(func() { nowFunc = old })
	return now
}

func run(t *testing.T, v string, name string, args ...string) string {
	t.Helper()
	out, err := applyFilters(v, []FilterDef{{Name: name, Args: args}}, testVars(), "test")
	if err != nil {
		t.Fatalf("%s(%q): %v", name, v, err)
	}
	return out
}

func TestStringFilters(t *testing.T) {
	cases := []struct {
		in, name string
		args     []string
		want     string
	}{
		{"browse.php?cat=12&page=2", "querystring", []string{"cat"}, "12"},
		{"/t.php?id=5&name=a%20b", "querystring", []string{"name"}, "a b"},
		{"abc", "prepend", []string{"x-"}, "x-abc"},
		{"abc", "append", []string{"-{{ .Today.Year }}"}, "abc-2026"},
		{"Y-day 12:27", "replace", []string{"Y-day", "yesterday"}, "yesterday 12:27"},
		{"12x45", "re_replace", []string{`(\d{2})x(\d{2})`, "S$1E$2"}, "S12E45"},
		{"Uploaded 09-14 02:31, Size 1 GiB", "regexp", []string{"Uploaded (.+?),"}, "09-14 02:31"},
		{"no match", "regexp", []string{`(\d+)`}, ""},
		{"sub/45/0", "split", []string{"/", "1"}, "45"},
		{"a|b|c", "split", []string{"|", "-1"}, "c"},
		{"  padded  ", "trim", nil, "padded"},
		{"xxTitlexx", "trim", []string{"x"}, "Title"},
		{"a%20b%2Bc", "urldecode", nil, "a b+c"},
		{"a b&c", "urlencode", nil, "a+b%26c"},
		{"Fish &amp; Chips &#39;", "htmldecode", nil, "Fish & Chips '"},
		{"<b>", "htmlencode", nil, "&lt;b&gt;"},
		{"MiXeD", "tolower", nil, "mixed"},
		{"MiXeD", "toupper", nil, "MIXED"},
		{"ŠĐĆŽšđčćž", "diacritics", []string{"replace"}, "SĐCZsđccz"},
		{`a?File<Name>*`, "validfilename", nil, "aFileName"},
		{"crime, x264, (music), Comedy, Science_Fiction", "validate", []string{"Action, Crime, Comedy, Science_Fiction"}, "crime, comedy, science fiction"},
		{`{"genres":["Drama","Comedy"]}`, "jsonjoinarray", []string{"$.genres", ", "}, "Drama, Comedy"},
		{"c2FtcGxl", "base64decode", nil, "sample"},
		{"sample", "base64encode", nil, "c2FtcGxl"},
		{"Тест", "re_replace", []string{`[\p{IsCyrillic}]+`, "cyr"}, "cyr"},
	}
	for _, c := range cases {
		if got := run(t, c.in, c.name, c.args...); got != c.want {
			t.Errorf("%s(%q, %v) = %q, want %q", c.name, c.in, c.args, got, c.want)
		}
	}
}

func TestUnknownFilter(t *testing.T) {
	if _, err := applyFilters("x", []FilterDef{{Name: "nosuchfilter"}}, testVars(), "t"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestDateParse(t *testing.T) {
	fixedNow(t)
	cases := []struct{ in, format, want string }{
		{"2017-09-18 19:17:24 +00:00", "yyyy-MM-dd HH:mm:ss zzz", "2017-09-18T19:17:24Z"},
		{"18 Sep 17 00:00:00 +03:00", "dd MMM yy HH:mm:ss zzz", "2017-09-17T21:00:00Z"},
		{"Tue, 19 Sep 2017 21:21:52 +0200", "ddd, dd MMM yyyy HH:mm:ss zzz", "2017-09-19T19:21:52Z"},
		{"5 June 2026 3:04 PM", "d MMMM yyyy h:mm tt", "2026-06-05T15:04:00Z"},
		{"05.06.2026 7:04", "dd.MM.yyyy H:mm", "2026-06-05T07:04:00Z"},
		{"06/05 07:04", "MM/dd HH:mm", "2026-06-05T07:04:00Z"}, // no year: this year
		{"12/25 07:04", "MM/dd HH:mm", "2025-12-25T07:04:00Z"}, // would be in the future
		{"14:30 +02:00", "HH:mm zzz", "2026-06-17T12:30:00Z"},  // time only: today
		{"2026-06-05T07:04:00", "yyyy-MM-dd'T'HH:mm:ss", "2026-06-05T07:04:00Z"},
	}
	for _, c := range cases {
		out := run(t, c.in, "dateparse", c.format)
		got, ok := parseResultDate(out)
		if !ok || got.Format(time.RFC3339) != c.want {
			t.Errorf("dateparse(%q, %q) = %q (%v), want %s", c.in, c.format, out, got, c.want)
		}
	}
}

func TestRelativeDates(t *testing.T) {
	now := fixedNow(t)
	cases := []struct {
		in   string
		fn   string
		want time.Time
	}{
		{"2 hours and 1 day ago", "timeago", now.Add(-26 * time.Hour)},
		{"9hr,12m,39s", "timeago", now.Add(-(9*time.Hour + 12*time.Minute + 39*time.Second))},
		{"1 week", "reltime", now.AddDate(0, 0, -7)},
		{"5 months", "timeago", now.AddDate(0, -5, 0)},
		{"an hour ago", "timeago", now.Add(-time.Hour)},
		{"now", "fuzzytime", now},
		{"Yesterday 10:15", "fuzzytime", time.Date(2026, 6, 16, 10, 15, 0, 0, time.UTC)},
		{"Today", "fuzzytime", now},
		{"1505788002", "fuzzytime", time.Unix(1505788002, 0).UTC()},
		{"Monday at 15:30", "fuzzytime", time.Date(2026, 6, 15, 15, 30, 0, 0, time.UTC)},
		{"1 Jan", "fuzzytime", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"03-04-2025", "fuzzytime", time.Date(2025, 3, 4, 0, 0, 0, 0, time.UTC)},
		{"Jun 3rd, 2024", "fuzzytime", time.Date(2024, 6, 3, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		out := run(t, c.in, c.fn)
		got, ok := parseResultDate(out)
		if !ok || !got.Equal(c.want) {
			t.Errorf("%s(%q) = %q, want %s", c.fn, c.in, out, c.want)
		}
	}
	out := run(t, "03-04-2025", "fuzzytime", "UK")
	if got, _ := parseResultDate(out); !got.Equal(time.Date(2025, 4, 3, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("fuzzytime UK = %q", out)
	}
}

func TestParseSize(t *testing.T) {
	cases := map[string]int64{
		"1.5 GB":     1610612736,
		"1,024.4 MB": 1074161254,
		"1.024,4 MB": 1074161254,
		"700 MiB":    734003200,
		"512 KB":     524288,
		"123456":     123456,
		"2,5 ГБ":     2684354560,
		"1 TB":       1 << 40,
		"0 B":        0,
		"":           0,
		"1 234 MB":   1293942784,
	}
	for in, want := range cases {
		if got := parseSize(in); got != want {
			t.Errorf("parseSize(%q) = %d, want %d", in, got, want)
		}
	}
	if parseCount("1,234") != 1234 || parseCount("12 seeders") != 12 || parseCount("-") != 0 {
		t.Error("parseCount")
	}
}

func TestJSONSelectors(t *testing.T) {
	root, err := decodeJSON([]byte(`{"data":[
		{"id":1,"attributes":{"name":"Sample Movie 1080p","size":10,"free":true},"files":[{"name":"a.mkv"}]},
		{"id":2,"attributes":{"name":"Other Show 720p","size":20,"free":false}},
		{"id":3,"attributes":{"name":"Sample Clip 1080p"}}
	],"total":3}`))
	if err != nil {
		t.Fatal(err)
	}
	ctx := &jsonCtx{root: root}
	sel, _ := parseJSONSel("data:has(attributes.size):not(attributes.name:contains(720p))")
	rows := sel.all(root, ctx)
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	get := func(s string, v any, c *jsonCtx) string {
		js, err := parseJSONSel(s)
		if err != nil {
			t.Fatal(err)
		}
		x, ok := js.first(v, c)
		if !ok {
			return "<none>"
		}
		return jsonString(x)
	}
	row := rows[0]
	if got := get("attributes.name", row, ctx); got != "Sample Movie 1080p" {
		t.Error(got)
	}
	if got := get("files[0].name", row, ctx); got != "a.mkv" {
		t.Error(got)
	}
	if got := get("attributes.free", row, ctx); got != "True" {
		t.Error(got)
	}
	if got := get("$.total", row, ctx); got != "3" {
		t.Error(got)
	}
	if got := get("name:contains(1080)", row.(map[string]any)["attributes"], ctx); got != "Sample Movie 1080p" {
		t.Error(got)
	}
	if got := get("name:not(:contains(1080))", row.(map[string]any)["attributes"], ctx); got != "<none>" {
		t.Error(got)
	}
	child := &jsonCtx{root: root, parent: row}
	if got := get("..id", map[string]any{}, child); got != "1" {
		t.Error(got)
	}
}

func TestNormalizeCSS(t *testing.T) {
	cases := map[string]string{
		`table[width="100%"][border=1] > tr`: `table[width="100%"][border="1"] > tr`,
		`td:contains(\00a0GB)`:               "td:contains(\" GB\")",
		`a:contains("x y")`:                  `a:contains("x y")`,
		`:scope > div`:                       `[data-plenka-scope] > div`,
	}
	for in, want := range cases {
		if got := normalizeCSS(in); got != want {
			t.Errorf("normalizeCSS(%q) = %q, want %q", in, got, want)
		}
		if _, err := compileCSS(in); err != nil {
			t.Errorf("compile %q: %v", in, err)
		}
	}
}
