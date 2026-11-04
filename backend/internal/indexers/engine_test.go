package indexers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// All sites, titles and hashes below are fictional.

const (
	hashA = "0123456789abcdef0123456789abcdef01234567"
	hashB = "89abcdef0123456789abcdef0123456789abcdef"
)

// fakeTorrent is a minimal bencoded torrent.
var fakeTorrent = []byte("d8:announce25:http://tracker.test/announce4:infod6:lengthi1e4:name10:sample.bin12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaaee")

func mustDef(t *testing.T, y string) *Definition {
	t.Helper()
	d, err := parseDefinition([]byte(y))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if issues := checkDefinition(d); len(issues) > 0 {
		t.Fatalf("issues: %v", issues)
	}
	return d
}

func newTestEngine(t *testing.T, y, site string, cfg map[string]string) *cardigann {
	t.Helper()
	if cfg == nil {
		cfg = map[string]string{}
	}
	cfg["sitelink"] = site + "/"
	c, err := newCardigann(mustDef(t, y), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

const publicDef = `---
id: example-tracker
name: Example Tracker
description: A fictional public tracker
language: en-US
type: public
encoding: UTF-8
links:
  - https://example-tracker.test/
caps:
  categorymappings:
    - {id: 1, cat: Movies/HD, desc: "HD Movies"}
    - {id: 2, cat: TV/HD, desc: "HD TV"}
    - {id: 3, cat: Audio, desc: "Music", default: true}
  modes:
    search: [q]
settings:
  - name: sort
    type: select
    label: Sort
    default: seeders
    options:
      seeders: Seeders
      date: Date
search:
  paths:
    - path: "{{ if .Keywords }}search/{{ .Keywords }}/{{ else }}latest/{{ end }}"
  inputs:
    $raw: "{{ range .Categories }}c[]={{.}}&{{end}}"
    sort: "{{ .Config.sort }}"
    empty: ""
  keywordsfilters:
    - name: re_replace
      args: ["\\s+", " "]
  rows:
    selector: table.results > tbody > tr
    after: 1
    filters:
      - name: andmatch
  fields:
    category:
      selector: a[href*="cat="]
      attribute: href
      filters:
        - name: querystring
          args: cat
    title:
      selector: a.title
    details:
      selector: a.title
      attribute: href
    download:
      selector: a.dl
      attribute: href
    infohash:
      selector: a[href^="magnet:"]
      attribute: href
      optional: true
      filters:
        - name: regexp
          args: "([a-fA-F0-9]{40})"
    size:
      selector: td.size
    seeders:
      selector: td.seed
    leechers:
      selector: td.leech
    date:
      selector: td.date
      filters:
        - name: append
          args: " +02:00"
        - name: dateparse
          args: "dd.MM.yyyy HH:mm zzz"
    downloadvolumefactor:
      case:
        img.free: 0
        "*": 1
    uploadvolumefactor:
      text: 1
    _note:
      selector: span.note
      optional: true
    description:
      text: "{{ if .Result._note }}{{ .Result._note }}{{ else }}no note{{ end }}"
`

const publicPage = `<html><body><table class="results"><tbody>
<tr class="main">
  <td><a href="/browse?cat=1">HD</a></td>
  <td><a class="title" href="/details/1">Sample Movie (2001) 1080p</a> <img class="free"></td>
  <td><a class="dl" href="/get/1.torrent">dl</a> <a href="magnet:?xt=urn:btih:%s&dn=x">m</a></td>
</tr>
<tr class="extra"><td class="size">1.5 GB</td><td class="seed">120</td><td class="leech">7</td><td class="date">05.06.2026 10:00</td><td><span class="note">great</span></td></tr>
<tr class="main">
  <td><a href="/browse?cat=2">TV</a></td>
  <td><a class="title" href="/details/2">Sample Movie Extras Show S01E01</a></td>
  <td><a class="dl" href="/get/2.torrent">dl</a></td>
</tr>
<tr class="extra"><td class="size">350 MB</td><td class="seed">3</td><td class="leech">1</td><td class="date">06.06.2026 11:30</td></tr>
<tr class="main">
  <td><a href="/browse?cat=1">HD</a></td>
  <td><a class="title" href="/details/3">Unrelated Film (1999)</a></td>
  <td><a class="dl" href="/get/3.torrent">dl</a></td>
</tr>
<tr class="extra"><td class="size">2 GB</td><td class="seed">50</td><td class="leech">2</td><td class="date">07.06.2026 12:00</td></tr>
</tbody></table></body></html>`

func TestPublicHTMLSearch(t *testing.T) {
	fixedNow(t)
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/get/") {
			w.Write(fakeTorrent)
			return
		}
		gotPath, gotQuery = r.URL.EscapedPath(), r.URL.RawQuery
		fmt.Fprintf(w, publicPage, strings.ToUpper(hashA))
	}))
	defer srv.Close()
	c := newTestEngine(t, publicDef, srv.URL, nil)

	rels, err := c.Search(context.Background(), &Query{Q: "Sample  Movie", Categories: []int{2000}})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/search/Sample%20Movie/" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "c[]=1&sort=seeders" {
		t.Errorf("query = %q", gotQuery)
	}
	// andmatch drops "Unrelated Film"; the category filter (Movies) drops the TV row.
	if len(rels) != 1 {
		t.Fatalf("got %d releases: %+v", len(rels), rels)
	}
	r := rels[0]
	if r.Title != "Sample Movie (2001) 1080p" || r.Size != 1610612736 || r.Seeders != 120 || r.Leechers != 7 {
		t.Errorf("release = %+v", r)
	}
	if r.Details != srv.URL+"/details/1" || r.Link != srv.URL+"/get/1.torrent" {
		t.Errorf("links = %q %q", r.Details, r.Link)
	}
	if r.InfoHash != hashA || !strings.HasPrefix(r.Magnet, "magnet:?xt=urn:btih:"+hashA) {
		t.Errorf("hash/magnet = %q %q", r.InfoHash, r.Magnet)
	}
	if want := time.Date(2026, 6, 5, 8, 0, 0, 0, time.UTC); !r.PublishDate.Equal(want) {
		t.Errorf("date = %v", r.PublishDate)
	}
	if r.DownloadVolumeFactor != 0 || r.Description != "great" {
		t.Errorf("dvf/description = %v %q", r.DownloadVolumeFactor, r.Description)
	}
	if fmt.Sprint(r.Category) != "[2000 2040]" {
		t.Errorf("category = %v", r.Category)
	}

	// browse (no keywords): default categories, latest path, all rows
	rels, err = c.Search(context.Background(), &Query{})
	if err != nil || len(rels) != 3 || gotPath != "/latest/" || gotQuery != "c[]=3&sort=seeders" {
		t.Fatalf("browse: %d %v path=%q query=%q", len(rels), err, gotPath, gotQuery)
	}
	data, magnet, err := c.Download(context.Background(), srv.URL+"/get/2.torrent")
	if err != nil || magnet != "" || string(data) != string(fakeTorrent) {
		t.Fatalf("download: %v %q", err, magnet)
	}
}

const privateDef = `---
id: private-example
name: Private Example
description: A fictional private tracker with a login form
language: en-US
type: private
encoding: windows-1251
links:
  - https://private-example.test/
caps:
  categorymappings:
    - {id: movies, cat: Movies}
  modes:
    search: [q]
settings:
  - name: username
    type: text
    label: Username
  - name: password
    type: password
    label: Password
login:
  path: login.php
  method: form
  form: form#login
  inputs:
    user: "{{ .Config.username }}"
    pass: "{{ .Config.password }}"
  error:
    - selector: div.error
  test:
    path: index.php
    selector: a.logout
search:
  paths:
    - path: browse.php
      method: post
  inputs:
    q: "{{ .Keywords }}"
  rows:
    selector: div.torrent
  fields:
    title:
      selector: a.name
    details:
      selector: a.name
      attribute: href
    size:
      text: "512 MB"
    seeders:
      selector: b.s
    leechers:
      text: 0
    category:
      text: movies
    date:
      text: now
download:
  before:
    path: thanks.php
    method: post
    inputs:
      id: "{{ .DownloadUri.Query.id }}"
  selectors:
    - selector: a.missing
      attribute: href
    - selector: a.torrent
      attribute: href
      usebeforeresponse: true
`

func TestPrivateFormLoginSearchAndDownload(t *testing.T) {
	fixedNow(t)
	var logins, thanks int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authed := false
		if ck, err := r.Cookie("sid"); err == nil && ck.Value == "s3cret" {
			authed = true
		}
		switch r.URL.Path {
		case "/login.php":
			if r.Method == http.MethodGet {
				fmt.Fprint(w, `<form id="search"><input name="q"></form>
					<form id="login" action="takelogin.php" method="post">
					<input type="hidden" name="csrf" value="tok123"><input name="user"><input type="password" name="pass">
					<input type="checkbox" name="remember"><input type="submit" name="go" value="Login"></form>`)
				return
			}
		case "/takelogin.php":
			atomic.AddInt32(&logins, 1)
			_ = r.ParseForm()
			if r.PostForm.Get("csrf") != "tok123" || r.PostForm.Get("go") != "Login" || r.PostForm.Has("remember") {
				http.Error(w, "bad form", 400)
				return
			}
			if r.PostForm.Get("user") != "alice" || r.PostForm.Get("pass") != "pw" {
				fmt.Fprint(w, `<div class="error">Wrong password, try again</div>`)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: "s3cret"})
			http.Redirect(w, r, "/index.php", http.StatusFound)
			return
		case "/index.php":
			if !authed {
				http.Redirect(w, r, "/login.php", http.StatusFound)
				return
			}
			fmt.Fprint(w, `<a class="logout" href="/logout">out</a>`)
			return
		case "/browse.php":
			if !authed {
				http.Redirect(w, r, "/login.php", http.StatusFound)
				return
			}
			_ = r.ParseForm()
			// keywords arrive in the site charset (windows-1251)
			if got := r.PostForm.Get("q"); got != "\xd2\xe5\xf1\xf2 Sample" { // "Тест Sample"
				http.Error(w, "bad query "+got, 400)
				return
			}
			fmt.Fprint(w, "<div class=\"torrent\"><a class=\"name\" href=\"details.php?id=42\">\xd2\xe5\xf1\xf2 Sample Movie (2001)</a><b class=\"s\">9</b></div>")
			return
		case "/details.php":
			fmt.Fprint(w, `<p>details</p>`)
			return
		case "/thanks.php":
			_ = r.ParseForm()
			if !authed || r.PostForm.Get("id") != "42" {
				http.Error(w, "no", 403)
				return
			}
			atomic.AddInt32(&thanks, 1)
			fmt.Fprint(w, `<a class="torrent" href="/dl.php?id=42">get</a>`)
			return
		case "/dl.php":
			if !authed {
				http.Redirect(w, r, "/login.php", http.StatusFound)
				return
			}
			w.Write(fakeTorrent)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	bad := newTestEngine(t, privateDef, srv.URL, map[string]string{"username": "alice", "password": "nope"})
	if _, err := bad.Search(context.Background(), &Query{Q: "Тест Sample"}); err == nil || !strings.Contains(err.Error(), "Wrong password") {
		t.Fatalf("expected login error, got %v", err)
	}

	c := newTestEngine(t, privateDef, srv.URL, map[string]string{"username": "alice", "password": "pw"})
	rels, err := c.Search(context.Background(), &Query{Q: "Тест Sample"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rels) != 1 || rels[0].Title != "Тест Sample Movie (2001)" || rels[0].Seeders != 9 || rels[0].Size != 512<<20 {
		t.Fatalf("releases: %+v", rels)
	}
	if rels[0].Magnet != "" || rels[0].Link != srv.URL+"/details.php?id=42" {
		t.Fatalf("private releases must resolve through the download block: %+v", rels[0])
	}
	data, magnet, err := c.Download(context.Background(), rels[0].Link)
	if err != nil || magnet != "" || string(data) != string(fakeTorrent) || thanks != 1 {
		t.Fatalf("download: %v %q thanks=%d", err, magnet, thanks)
	}

	// session cookies are reused by a new engine (as after a restart): no new login
	cookies, _ := c.cookieSnapshot()
	n := atomic.LoadInt32(&logins)
	c2, _ := newCardigann(mustDef(t, privateDef), map[string]string{"sitelink": srv.URL + "/", "username": "alice", "password": "pw"}, cookies)
	if _, err := c2.Search(context.Background(), &Query{Q: "Тест Sample"}); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&logins) != n {
		t.Fatal("stored session was not reused")
	}
	// an expired session (redirect on search) triggers one re-login
	c2.sess.jar.clear()
	if _, err := c2.Search(context.Background(), &Query{Q: "Тест Sample"}); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&logins) != n+1 {
		t.Fatalf("expected a re-login, logins=%d", logins)
	}
}

const jsonDef = `---
id: example-api
name: Example API
description: A fictional JSON API tracker
language: en-US
type: semi-private
encoding: UTF-8
links:
  - https://api.example.test/
caps:
  categorymappings:
    - {id: 45, cat: Movies/HD}
    - {id: 46, cat: Movies/UHD}
settings:
  - name: apikey
    type: text
    label: API key
search:
  headers:
    x-api-key: ["{{ .Config.apikey }}"]
  paths:
    - path: api/list.json
      response:
        type: json
  inputs:
    query_term: "{{ .Keywords }}"
  rows:
    selector: data.movies
    attribute: torrents
    multiple: true
    missingAttributeEqualsNoResults: true
    count:
      selector: data.movie_count
  fields:
    _quality:
      selector: quality
    category:
      selector: quality
      case:
        "2160p": 46
        "*": 45
    year:
      selector: ..year
    _free:
      selector: free
      optional: true
    title:
      selector: ..title
      filters:
        - name: append
          args: " ({{ .Result.year }}) {{ .Result._quality }}{{ if eq .Result._free \"True\" }} FREE{{ end }}"
    details:
      selector: ..url
    download:
      selector: url
    infohash:
      selector: hash
    date:
      selector: ..uploaded_unix
    size:
      selector: size_bytes
    seeders:
      selector: seeds
    leechers:
      selector: peers
    downloadvolumefactor:
      selector: free
      optional: true
      default: 1
      case:
        "True": 0
        "False": 1
`

func TestJSONSearch(t *testing.T) {
	var key string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key = r.Header.Get("X-Api-Key")
		if r.URL.Query().Get("query_term") == "nothing" {
			fmt.Fprint(w, `{"data":{"movie_count":0}}`)
			return
		}
		fmt.Fprintf(w, `{"data":{"movie_count":2,"movies":[
			{"title":"Sample Movie","year":2001,"url":"https://api.example.test/m/1","uploaded_unix":1780000000,
			 "torrents":[
			   {"quality":"1080p","url":"https://api.example.test/t/1a","hash":"%s","size_bytes":1500000000,"seeds":40,"peers":4,"free":true},
			   {"quality":"2160p","url":"https://api.example.test/t/1b","hash":"%s","size_bytes":9000000000,"seeds":12,"peers":1}
			 ]},
			{"title":"No Torrents Yet","year":2002,"url":"https://api.example.test/m/2"}
		]}}`, strings.ToUpper(hashA), hashB)
	}))
	defer srv.Close()
	c := newTestEngine(t, jsonDef, srv.URL, map[string]string{"apikey": "k-123"})
	rels, err := c.Search(context.Background(), &Query{Q: "Sample"})
	if err != nil {
		t.Fatal(err)
	}
	if key != "k-123" || len(rels) != 2 {
		t.Fatalf("key=%q releases=%+v", key, rels)
	}
	a, b := rels[0], rels[1]
	if a.Title != "Sample Movie (2001) 1080p FREE" || a.InfoHash != hashA || a.Seeders != 40 || a.Size != 1500000000 || a.DownloadVolumeFactor != 0 {
		t.Errorf("a = %+v", a)
	}
	if b.Title != "Sample Movie (2001) 2160p" || fmt.Sprint(b.Category) != "[2000 2045]" || b.DownloadVolumeFactor != 1 {
		t.Errorf("b = %+v", b)
	}
	if a.Details != "https://api.example.test/m/1" || !a.PublishDate.Equal(time.Unix(1780000000, 0)) {
		t.Errorf("details/date = %q %v", a.Details, a.PublishDate)
	}
	if !strings.HasPrefix(a.Magnet, "magnet:?xt=urn:btih:"+hashA) {
		t.Errorf("semi-private infohash must produce a magnet: %q", a.Magnet)
	}
	rels, err = c.Search(context.Background(), &Query{Q: "nothing"})
	if err != nil || len(rels) != 0 {
		t.Fatalf("count=0: %v %v", rels, err)
	}
}

const xmlDef = `---
id: example-feed
name: Example Feed
description: A fictional torznab-style feed
language: en-US
type: public
encoding: UTF-8
links:
  - https://feed.example.test/
caps:
  categorymappings:
    - {id: 2000, cat: Movies}
settings: []
search:
  paths:
    - path: api
      response:
        type: xml
  inputs:
    t: search
    q: "{{ .Keywords }}"
  rows:
    selector: rss > channel > item
  fields:
    title:
      selector: title
    details:
      selector: comments
    download:
      selector: enclosure
      attribute: url
    date:
      selector: pubDate
    size:
      selector: "[name=size]"
      attribute: value
    seeders:
      selector: "[name=seeders]"
      attribute: value
    leechers:
      selector: "[name=peers]"
      attribute: value
    infohash:
      selector: "[name=infohash]"
      attribute: value
    category:
      selector: "[name=category]"
      attribute: value
`

func TestXMLSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><title>feed</title>
<item><title>Sample Movie (2001) 720p</title><link>https://feed.example.test/x</link>
<comments>https://feed.example.test/details/9</comments><pubDate>Fri, 05 Jun 2026 10:00:00 +0000</pubDate>
<enclosure url="https://feed.example.test/dl/9.torrent" length="1" type="application/x-bittorrent"/>
<torznab:attr name="size" value="734003200"/><torznab:attr name="seeders" value="15"/>
<torznab:attr name="peers" value="20"/><torznab:attr name="infohash" value="%s"/><torznab:attr name="category" value="2000"/>
</item></channel></rss>`, hashB)
	}))
	defer srv.Close()
	c := newTestEngine(t, xmlDef, srv.URL, nil)
	rels, err := c.Search(context.Background(), &Query{Q: "Sample"})
	if err != nil || len(rels) != 1 {
		t.Fatalf("%v %+v", err, rels)
	}
	r := rels[0]
	if r.Title != "Sample Movie (2001) 720p" || r.Size != 734003200 || r.Seeders != 15 || r.Leechers != 20 ||
		r.InfoHash != hashB || r.Link != "https://feed.example.test/dl/9.torrent" ||
		!r.PublishDate.Equal(time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)) || fmt.Sprint(r.Category) != "[2000]" {
		t.Fatalf("release = %+v", r)
	}
}

const dateHeaderDef = `---
id: dated-example
name: Dated Example
description: Rows take their date from header rows
language: en-US
type: public
encoding: UTF-8
links: [https://dated.example.test/]
caps:
  categorymappings:
    - {id: 1, cat: Other}
settings: []
login:
  method: cookie
  inputs:
    cookie: "{{ .Config.cookie }}"
  test:
    path: /
    selector: span.user
search:
  paths:
    - path: /
  rows:
    selector: tr.t
    dateheaders:
      selector: tr.day
      filters:
        - name: dateparse
          args: "yyyy-MM-dd"
  fields:
    title:
      selector: td
    download:
      text: "/dl/{{ .Result.title }}"
    size:
      text: "1 KB"
    seeders:
      text: 1
    leechers:
      text: 1
    category:
      text: 1
`

func TestDateHeadersAndCookieLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ck, err := r.Cookie("uid"); err != nil || ck.Value != "77" {
			fmt.Fprint(w, `<p>please log in</p>`)
			return
		}
		fmt.Fprint(w, `<span class="user">me</span><table>
			<tr class="day"><td>2026-06-02</td></tr><tr class="t"><td>first</td></tr><tr class="t"><td>second</td></tr>
			<tr class="day"><td>2026-06-01</td></tr><tr class="t"><td>third</td></tr></table>`)
	}))
	defer srv.Close()

	bad := newTestEngine(t, dateHeaderDef, srv.URL, map[string]string{"cookie": "uid=1"})
	if _, err := bad.Search(context.Background(), &Query{}); err == nil || !strings.Contains(err.Error(), "cookie") {
		t.Fatalf("expected a cookie error, got %v", err)
	}
	c := newTestEngine(t, dateHeaderDef, srv.URL, map[string]string{"cookie": "uid=77; other=x"})
	rels, err := c.Search(context.Background(), &Query{})
	if err != nil || len(rels) != 3 {
		t.Fatalf("%v %+v", err, rels)
	}
	want := []string{"2026-06-02", "2026-06-02", "2026-06-01"}
	for i, r := range rels {
		if r.PublishDate.Format("2006-01-02") != want[i] {
			t.Errorf("row %d date %v", i, r.PublishDate)
		}
	}
}

func TestCaptchaIsReported(t *testing.T) {
	y := strings.Replace(privateDef, "  error:\n", "  captcha:\n    type: image\n    selector: img.captcha\n    input: code\n  error:\n", 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<form id="login"><input name="user"><img class="captcha" src="/c.png"></form>`)
	}))
	defer srv.Close()
	c := newTestEngine(t, y, srv.URL, map[string]string{"username": "a", "password": "b"})
	if _, err := c.Search(context.Background(), &Query{Q: "x"}); err == nil || !strings.Contains(err.Error(), "CAPTCHA") {
		t.Fatalf("expected captcha error, got %v", err)
	}
}

func TestCloudflareIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "cloudflare")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `<html><head><title>Just a moment...</title></head></html>`)
	}))
	defer srv.Close()
	c := newTestEngine(t, publicDef, srv.URL, nil)
	if _, err := c.Search(context.Background(), &Query{Q: "x"}); err == nil || !strings.Contains(err.Error(), "Cloudflare") {
		t.Fatalf("expected a Cloudflare error, got %v", err)
	}
}

func TestPathCategories(t *testing.T) {
	if !pathApplies(nil, []string{"1"}) || !pathApplies([]string{"1"}, nil) {
		t.Error("empty lists always apply")
	}
	if !pathApplies([]string{"1", "2"}, []string{"2"}) || pathApplies([]string{"1"}, []string{"3"}) {
		t.Error("positive list")
	}
	if pathApplies([]string{"!", "9"}, []string{"9"}) || !pathApplies([]string{"!", "9"}, []string{"9", "1"}) {
		t.Error("negated list")
	}
}

func TestScopeSelector(t *testing.T) {
	doc, _ := parseHTML([]byte(`<div id="row"><span><a>direct</a></span><p><span><a>nested</a></span></p></div>`))
	row, _ := queryFirst(doc, "#row")
	n, err := queryAll(row, ":scope > span > a")
	if err != nil || len(n) != 1 || nodeText(n[0]) != "direct" {
		t.Fatalf("%v %v", n, err)
	}
	if _, ok := attr(row, scopeAttr); ok {
		t.Fatal("scope marker left behind")
	}
}
