package indexers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

// Tests for the own definition backend/definitions/rutracker.yml against a made-up
// site that uses RuTracker's markup (windows-1251 on the wire).

const rutrackerPage = `<html><body>
<a id="logged-in-username" href="profile.php">alice</a>
<table id="tor-tbl"><tbody>
<tr class="tCenter hl-tr">
  <td class="row1 f-name-col"><div class="f-name"><a class="gen f ts-text" href="tracker.php?f=101">Зарубежные фильмы (HD Video)</a></div></td>
  <td class="row4 med tLeft t-title-col tt"><div class="wbr t-title"><a data-topic_id="1001" class="med tLink tt-text ts-text" href="viewtopic.php?t=1001">Пример фильма / Sample Movie (2001) BDRip 1080p</a></div></td>
  <td class="row4 small nowrap tor-size" data-ts_text="2147483648"><a class="small tr-dl dl-stub" href="dl.php?t=1001">2 GB</a></td>
  <td class="row4 nowrap" data-ts_text="42"><b class="seedmed">42</b></td>
  <td class="row4 leechmed bold">3</td>
  <td class="row4 small number-format">1234</td>
  <td class="row4 small nowrap" data-ts_text="1700000000"><p>14-Ноя-23</p></td>
</tr>
<tr class="tCenter hl-tr">
  <td class="row1 f-name-col"><div class="f-name"><a class="gen f ts-text" href="tracker.php?f=202">Отечественные сериалы</a></div></td>
  <td class="row4 med tLeft t-title-col tt"><div class="wbr t-title"><a data-topic_id="1002" class="med tLink tt-text ts-text" href="viewtopic.php?t=1002">Пример сериала (Сезон 1)</a></div></td>
  <td class="row4 small nowrap tor-size" data-ts_text="1073741824"><a class="small tr-dl dl-stub" href="dl.php?t=1002">1 GB</a></td>
  <td class="row4 nowrap" data-ts_text="-3"><span class="seedmed">-3</span></td>
  <td class="row4 leechmed bold">0</td>
  <td class="row4 small number-format">10</td>
  <td class="row4 small nowrap" data-ts_text="1700100000"><p>16-Ноя-23</p></td>
</tr>
<tr class="tCenter hl-tr">
  <td class="row1 f-name-col"><div class="f-name"><a class="gen f ts-text" href="tracker.php?f=303">Аудиокниги</a></div></td>
  <td class="row4 med tLeft t-title-col tt"><div class="wbr t-title"><a data-topic_id="1003" class="med tLink" href="viewtopic.php?t=1003">Закрытая раздача</a></div></td>
  <td class="row4 small nowrap tor-size" data-ts_text="0">—</td>
  <td class="row4 small nowrap" data-ts_text="1700200000"><p>17-Ноя-23</p></td>
</tr>
</tbody></table></body></html>`

const rutrackerLoginPage = `<html><body><form action="login.php" method="post">%s
<input name="login_username"><input name="login_password" type="password">
</form></body></html>`

func cp1251(t *testing.T, s string) []byte {
	t.Helper()
	b, err := charmap.Windows1251.NewEncoder().Bytes([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

type fakeRutracker struct {
	srv     *httptest.Server
	logins  int32
	captcha bool
	lastQ   atomic.Value
}

func newFakeRutracker(t *testing.T) *fakeRutracker {
	f := &fakeRutracker{}
	loginPage := func(w http.ResponseWriter, extra string) {
		w.Write(cp1251(t, fmt.Sprintf(rutrackerLoginPage, extra)))
	}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("bb_session")
		loggedIn := err == nil && c.Value == "s3cr3t"
		switch r.URL.Path {
		case "/forum/login.php":
			atomic.AddInt32(&f.logins, 1)
			_ = r.ParseForm()
			login, _ := charmap.Windows1251.NewDecoder().String(r.PostForm.Get("login"))
			switch {
			case f.captcha:
				loginPage(w, `<input type="hidden" name="cap_sid" value="x">`)
			case r.PostForm.Get("login_username") == "alice" && r.PostForm.Get("login_password") == "pw" && login == "вход":
				http.SetCookie(w, &http.Cookie{Name: "bb_session", Value: "s3cr3t", Path: "/"})
				http.Redirect(w, r, "/forum/index.php", http.StatusFound)
			default:
				loginPage(w, `<h4 class="warnColor1 tCenter mrg_16">неверный пароль</h4>`)
			}
		case "/forum/index.php":
			if !loggedIn {
				loginPage(w, "")
				return
			}
			w.Write(cp1251(t, `<a id="logged-in-username" href="profile.php">alice</a>`))
		case "/forum/tracker.php":
			if !loggedIn {
				loginPage(w, "")
				return
			}
			q, _ := charmap.Windows1251.NewDecoder().String(r.URL.Query().Get("nm"))
			f.lastQ.Store(q)
			w.Write(cp1251(t, rutrackerPage))
		case "/forum/dl.php":
			if !loggedIn {
				loginPage(w, "")
				return
			}
			w.Write(fakeTorrent)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func rutrackerDef(t *testing.T) *Definition {
	t.Helper()
	data, err := readDefinitionFile(builtinPrefix + "rutracker.yml")
	if err != nil {
		t.Fatal(err)
	}
	def, err := parseDefinition(data)
	if err != nil {
		t.Fatal(err)
	}
	if issues := checkDefinition(def); len(issues) > 0 {
		t.Fatalf("own definition has issues: %v", issues)
	}
	return def
}

func newTestRutracker(t *testing.T, f *fakeRutracker, password string) *cardigann {
	t.Helper()
	cg, err := newCardigann(rutrackerDef(t), map[string]string{
		"username": "alice", "password": password, "sitelink": f.srv.URL + "/",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return cg
}

func TestRutrackerDefinition(t *testing.T) {
	f := newFakeRutracker(t)
	cg := newTestRutracker(t, f, "pw")
	ctx := context.Background()

	rels, err := cg.Search(ctx, &Query{Q: "Пример"})
	if err != nil {
		t.Fatal(err)
	}
	if f.lastQ.Load() != "Пример" || atomic.LoadInt32(&f.logins) != 1 {
		t.Fatalf("query %v, logins %d", f.lastQ.Load(), f.logins)
	}
	if len(rels) != 2 {
		t.Fatalf("rows without a download link are skipped: %d results", len(rels))
	}
	r := rels[0]
	if r.Title != "Пример фильма / Sample Movie (2001) BDRip 1080p" || r.Size != 2147483648 || r.Seeders != 42 ||
		r.Leechers != 3 || r.Grabs != 1234 || r.PublishDate.Unix() != 1700000000 ||
		r.Link != f.srv.URL+"/forum/dl.php?t=1001" || r.Details != f.srv.URL+"/forum/viewtopic.php?t=1001" ||
		r.Description != "Зарубежные фильмы (HD Video)" {
		t.Fatalf("release: %+v", r)
	}
	if !anyCatMatches(r.Category, []int{2040}) || !anyCatMatches(rels[1].Category, []int{5000}) || rels[1].Seeders != 0 {
		t.Fatalf("second release / categories: %+v %v", rels[1], r.Category)
	}
	if c, dirty := cg.cookieSnapshot(); !dirty || c["bb_session"] != "s3cr3t" {
		t.Fatalf("session cookie must be kept: %v %v", c, dirty)
	}

	data, magnet, err := cg.Download(ctx, r.Link)
	if err != nil || magnet != "" || !isTorrent(data) {
		t.Fatalf("download: %v %q", err, magnet)
	}
	if atomic.LoadInt32(&f.logins) != 1 {
		t.Fatal("the session must be reused")
	}
}

func TestRutrackerLoginErrors(t *testing.T) {
	f := newFakeRutracker(t)
	ctx := context.Background()
	if _, err := newTestRutracker(t, f, "nope").Search(ctx, &Query{}); err == nil || !strings.Contains(err.Error(), "неверный пароль") {
		t.Fatalf("wrong password: %v", err)
	}
	f.captcha = true
	if _, err := newTestRutracker(t, f, "pw").Search(ctx, &Query{}); err == nil || !strings.Contains(err.Error(), "captcha") {
		t.Fatalf("captcha: %v", err)
	}
}

func TestRutrackerCategories(t *testing.T) {
	def := rutrackerDef(t)
	var filters []FilterDef
	for _, f := range def.Search.Fields {
		if f.Name == "category" {
			filters = f.Block.Filters
		}
	}
	for forum, want := range map[string]string{
		"Зарубежные фильмы (HD Video)":    "movies-hd",
		"Фильмы 2021-2023":                "movies",
		"Зарубежные сериалы (UHD Video)":  "tv-uhd",
		"Аниме (HD Video)":                "anime",
		"Документальные фильмы":           "documentary",
		"Аудиокниги: фантастика":          "audiobook",
		"Классическая музыка (lossless)":  "lossless",
		"Игры для Windows":                "games",
		"Что-то непонятное":               "other",
		"Мультсериалы":                    "tv",
		"Arthouse и авторское кино (DVD)": "movies-dvd",
	} {
		got, err := applyFilters(forum, filters, tmplVars{}, def.ID)
		if err != nil || got != want {
			t.Errorf("%q: %q %v, want %s", forum, got, err, want)
		}
	}
}

// An own definition wins over a downloaded one with the same id.
func TestOwnDefinitionOverridesDownloaded(t *testing.T) {
	dir := t.TempDir()
	dup := filepath.Join(dir, "rutracker.yml")
	other := strings.Replace(publicDef, "id: example-tracker", "id: rutracker", 1)
	if err := os.WriteFile(dup, []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}
	metas, skipped := loadMetas(definitionFiles([]string{dup}))
	if skipped != 0 || len(metas) != 1 || !isBuiltin(metas[0].file) || metas[0].Name != "RuTracker.org" {
		t.Fatalf("metas: %+v skipped %d", metas, skipped)
	}
}
