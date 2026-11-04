package indexers

import (
	"net/url"
	"testing"
)

func testVars() tmplVars {
	return tmplVars{
		"Config": map[string]any{
			"sitelink": "https://example-tracker.test/", "sort": "2", "cat-id": "7",
			"freeleech": "True", "off": nil, "langs": []string{"en", "fr"},
		},
		"Keywords":   "Sample Movie",
		"Categories": []string{"101", "202"},
		"Query":      map[string]any{"IMDBID": nil, "Q": "Sample Movie", "Season": "1"},
		"Result":     map[string]any{"_cat": "series", "title": "Sample Movie (2001)", "empty": nil},
		"True":       "True",
		"False":      nil,
		"Today":      map[string]any{"Year": "2026"},
	}
}

func TestTemplates(t *testing.T) {
	cases := []struct{ src, want string }{
		{"plain text", "plain text"},
		{"{{ .Config.sitelink }}browse", "https://example-tracker.test/browse"},
		{"{{ .Config.cat-id }}", "7"},
		{"{{ if .Keywords }}search/{{ .Keywords }}{{ else }}latest{{ end }}", "search/Sample Movie"},
		{"{{ if .Query.IMDBID }}imdb{{ else }}kw{{ end }}", "kw"},
		{"{{ if .Config.off }}a{{ else if .Config.freeleech }}b{{ else }}c{{ end }}", "b"},
		{`{{ if eq .Result._cat "series" }}512 MB{{ else }}2 GB{{ end }}`, "512 MB"},
		{`{{ if ne .Result._cat "series" }}x{{ else }}y{{ end }}`, "y"},
		{`{{ if or (eq .Result._cat "movie") (or (eq .Result._cat "x") (eq .Result._cat "series")) }}hit{{ else }}miss{{ end }}`, "hit"},
		{"{{ if and .Config.freeleech .Config.off }}a{{ else }}b{{ end }}", "b"},
		{"{{ if and .Query.Season (eq .Query.IMDBID .False) }}tv{{ else }}no{{ end }}", "tv"},
		{"{{ or .Result.empty .Result.title }}", "Sample Movie (2001)"},
		{"{{ range .Categories }}cat[]={{.}}&{{end}}", "cat[]=101&cat[]=202&"},
		{"{{ range $i, $e := .Categories }}&c[{{$i}}]={{$e}}{{end}}", "&c[0]=101&c[1]=202"},
		{`{{ join .Categories "," }}`, "101,202"},
		{`{{ re_replace .Keywords "\\s+" "." }}`, "Sample.Movie"},
		{`{{ re_replace .Keywords "(\\w+) (\\w+)" "$2-$1" }}`, "Movie-Sample"},
		{"{{ .Today.Year }}", "2026"},
		{"{{ .Missing.Path }}x", "x"},
		{"{{ .True }}|{{ .False }}", "True|"},
		{`{{ if eq .Result._cat \"series\" }}esc{{ end }}`, "esc"},
		{"{{if .Keywords}}nospace{{end}}", "nospace"},
	}
	for _, c := range cases {
		got, err := expand(c.src, testVars(), nil)
		if err != nil {
			t.Errorf("%q: %v", c.src, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %q, want %q", c.src, got, c.want)
		}
	}
}

func TestTemplateEscape(t *testing.T) {
	got, err := expand("search/{{ .Keywords }}/", testVars(), url.PathEscape)
	if err != nil || got != "search/Sample%20Movie/" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestTemplateErrors(t *testing.T) {
	for _, src := range []string{
		"{{ if .Keywords }}open",
		"{{ end }}",
		"{{ printf .Keywords }}",
		"{{ .Keywords ",
		`{{ re_replace .Keywords "[" "" }}`,
	} {
		if _, err := expand(src, testVars(), nil); err == nil {
			t.Errorf("%q: expected an error", src)
		}
	}
}
