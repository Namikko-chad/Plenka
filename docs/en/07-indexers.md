# Tracker Search (Indexers)

`backend/internal/indexers/` is the tracker search module: our own implementation of the
Cardigann YAML definition format, own definitions, a native Knaben meta-search indexer, indexer
management and the native search API (`/api/search`, `/api/indexers/*`,
`/api/definitions*`, see [31-rest-api](31-rest-api.md) §1.4–1.6). There is no
Jackett-compatible API.

## Downloaded definitions

The binary contains no third-party tracker definitions (only our own, see
[Own definitions](#own-definitions)). They are downloaded at runtime:

- Source: env `DEFINITIONS_URL` (`internal/config`), default
  `https://codeload.github.com/Jackett/Jackett/tar.gz/refs/heads/master` (~5 MB).
- `defsource.go` stream-decompresses the `.tar.gz` (nothing is buffered in RAM) and
  keeps every regular `*.yml` whose parent directory is named `Definitions` (in Jackett
  that is `src/Jackett.Common/Definitions/`; `Indexers/Definitions/` holds only `.cs`
  files). Hidden names, files over 2 MB and more than 5000 files are rejected; the
  download times out after 2 minutes.
- Files go to a temporary directory under `$DATA_DIR`, which then replaces
  `$DATA_DIR/definitions/` (rename; an interrupted swap is repaired at the next start).
- Outcome is stored in `plenka.db`, bucket `indexers_meta`, key `definitions`:
  `updatedAt` (UTC, last success) and `lastError` (last failure, cleared on success).
- At startup the manager loads what is on disk (zero definitions is fine — Knaben still
  works) and starts an update in the background (`StartDefinitionsUpdate`, wired in
  `main.go`). `POST /api/definitions/update` runs one on demand; concurrent requests join
  the running update. A failed download or an archive without definitions keeps the
  previous set and records `lastError` (the endpoint answers `502`).
- After a successful update all indexers are reloaded. Enabled state and settings are
  stored by indexer id, so they carry over; an id that disappears is hidden and comes back
  with its settings. A definition's `replaces` ids are used to pick up state stored under
  an old id.
- Each file is decoded and checked when loaded (`check.go`): templates, CSS/JSON
  selectors, regular expressions and filter names are compiled. Files that are not valid
  YAML or lack id/links/search are skipped and logged. Definitions with problems are
  listed but cannot run: their `lastError` reads `unsupported definition: …`. Only the
  metadata (name, type, links, settings, category ids) stays in memory (about 1.5 MB for
  586 definitions); the full definition is parsed again when the indexer is first used.

Every indexer starts disabled, the built-in ones included: the user enables trackers in
the tracker settings (**Settings → Trackers** in the web app). Until then a search answers
with no indexers and the clients say so.

With Jackett's definitions (586 files) all 586 load; 584 pass the checks. The two others
use selectors outside CSS (`:has(~ …)` and two compound selectors glued together).

## Own definitions

`backend/definitions/` holds Plenka's own definitions in the same YAML format. They are
embedded in the binary (`definitions.FS`), so they are listed even before anything is
downloaded and need no network to load. The manager reads them first and the downloaded
set after: an own definition overrides a downloaded one with the same id (logged as
`own definition overrides a downloaded one`). They are not counted in
`GET /api/definitions` → `count`.

- **RuTracker.org** (`rutracker.yml`, id `rutracker`; the downloaded set only has the
  unrelated `rutracker-ru` for rutracker.ru). Needs an account (`username`, `password`);
  the site link can be switched to the `rutracker.net` / `rutracker.nl` mirrors. Login:
  POST `forum/login.php` (windows-1251), checked by `#logged-in-username` on
  `forum/index.php`; a CAPTCHA (`cap_sid`) or the site's error message is reported as the
  error. Search: `forum/tracker.php?nm=…`; rows without a download link (closed or
  unchecked releases) are skipped; `.torrent` files come from `forum/dl.php?t=…` through
  the session. The site has hundreds of forums, so the category is derived from words in
  the forum name with `re_replace` filters (film/movie, series, anime, HD/UHD/DVD, music,
  audiobooks, games, …, matched by their Russian stems).

The native **Knaben** indexer (`knaben.go`, a public meta-search with a JSON API) lives in
code.

## Engine

| Area | Files | What it does |
|---|---|---|
| Model | `def.go` | YAML model; order-preserving maps for inputs, fields, `case` and select options; `\/` escapes accepted; a missing `settings` block means username + password |
| Templates | `tmpl.go` | `{{ .Config.x }}` (names may contain `-`), `.Keywords`, `.Query.*`, `.Categories`, `.Result.*`, `.DownloadUri.*`, `.Today.Year`, `.True`/`.False`; `if`/`else if`/`else`/`end`, `range` (incl. `$i, $e :=`), `and`, `or`, `not`, `eq`, `ne`, `join`, `re_replace`; values printed into paths and `$raw` inputs are URL-encoded in the site charset |
| Login | `auth.go` | methods `post`, `get`, `form` (fills the page's form: hidden fields, checked boxes, first submit, `selectors: true`), `cookie`, `oneurl`; `inputs`, `selectorinputs`, `getselectorinputs`, `cookies`, `headers`, `error` (with `message`), `test` (a redirect or a missing selector = not logged in) |
| Search | `cardigann.go` | `paths` (method, inputs with `inheritinputs`, `categories` incl. `!`, `followredirect`, `response` html/json/xml, `noResultsMessage`), `keywordsfilters`, `preprocessingfilters`, `headers`, `error`, `allowEmptyInputs`; rows with `after`, `dateheaders`, `andmatch`, `count`, `attribute`/`multiple`/`missingAttributeEqualsNoResults`; fields in definition order with `selector`/`attribute`/`text`/`remove`/`case`/`optional`/`default`/`filters` |
| Selectors | `dom.go`, `jsonsel.go` | HTML via `x/net/html` + cascadia (unquoted `:contains(…)` args and attribute values are quoted, `:scope` is supported); XML (RSS/Torznab feeds) is turned into the same node tree; JSON paths `a.b`, `x[0]`, `$`, parent `..`, `:has()`, `:not()`, `:contains()` |
| Filters | `filterchain.go`, `timeparse.go` | querystring, prepend, append, replace, re_replace, regexp (.NET syntax via regexp2; `\p{IsCyrillic}`-style blocks rewritten to ranges), split, trim, tolower, toupper, urlencode/urldecode, htmlencode/htmldecode, base64encode/base64decode, diacritics, validfilename, validate, jsonjoinarray, dateparse/timeparse (.NET format strings matched token by token), timeago/reltime, fuzzytime (incl. `UK`), strdump/hexdump (debug log); arguments may be templates |
| Values | `values.go`, `newznab.go` | sizes (`1.5 GB`, `1.024,4 MB`, `700 МБ`), counts, dates, IMDb ids; Newznab category table and mapping (a request for 2000 matches 2040; results get the parent category too); a magnet is built from `infohash` for public and semi-private trackers |
| Download | `fetch.go` | the `download` block: `before` (path or `pathselector`, method, inputs, `queryseparator`), `selectors` tried in order (`usebeforeresponse`), `infohash` → magnet; otherwise the link itself; redirects to `magnet:` are followed |
| HTTP | `session.go` | per-indexer session: cookie jar scoped to the tracker's domain (persisted in `indexers_cookies`), site charset (`windows-1251` …) for requests and responses, `requestDelay`, manual redirects, 16 MB response limit, `certificates` (SHA-1 pinning for broken certificates), `useragent` setting |

Ignored keys: the top-level `followredirect` (automatic site link update) and
`testlinktorrent`. Info, poster and id fields (`poster`, `imdbid`, `genre`,
`description`, …) are dropped when their selector finds nothing, as the format
specifies; a missing required field skips that row.

Cloudflare checks go through [FlareSolverr](https://github.com/FlareSolverr/FlareSolverr)
when its address is set (`flareSolverrUrl`, **Settings → FlareSolverr address**), as in
Jackett and Prowlarr. When a response is a Cloudflare challenge (`Cf-Mitigated:
challenge`, or a 403/503/429 from `Server: cloudflare` with the challenge page), the
session sends `request.get` for that URL to `<flareSolverrUrl>/v1` (`flaresolverr.go`,
up to 60 s, still bound by the search timeout), takes over the returned cookies of the
tracker's domain (`cf_clearance`, saved with the session) and the browser's User-Agent,
and repeats the request once. Without FlareSolverr the error is `the site is behind a
Cloudflare challenge: set the FlareSolverr address…`; a challenge after solving is
reported as such. The User-Agent is kept in memory only, so after a restart the first
request is solved again. FlareSolverr can take 10–30 s; raise the search timeout if
checks time out.

Not supported, reported as a clear error instead of crashing: login forms that show a
CAPTCHA (a cookie login can be used instead when the tracker allows it).

## Manager

`manager.go` keeps per-tracker state in `plenka.db` (`indexers`: enabled, settings, last
error and check time; `indexers_cookies`; `indexers_meta`). A search queries every
selected tracker in parallel (16 at a time) with the configured timeout
(`searchTimeoutSec`, default 30 s), skips trackers without a matching category, records
each tracker's status, merges results by infoHash (else title+size) and sorts by
seeders. Trackers with unfilled required settings (password fields and settings such as
`username`, `cookie`, `apikey`) answer `indexer is not configured`. Changing settings
rebuilds the engine and drops the stored session.

## Download tokens

Results without a magnet get `link = /api/indexers/{id}/download?token=…`. The token is
base64url JSON with the target URL, signed with HMAC-SHA256 over the indexer id and a
random key kept in `indexers_meta` (`tokenKey`). It cannot be forged into an arbitrary
URL, and it only resolves through the tracker that issued it. `Manager.ResolveLink`
(link → `.torrent` bytes or magnet) serves both the HTTP endpoint and the torrent engine,
which gets it through a late-bound hook set in `main.go` (see
[08-torrent-engine](08-torrent-engine.md)).

## Tests

`go test ./internal/indexers/` runs without network: `httptest` servers serve fictional
tracker pages, JSON and RSS for inline definitions (login forms, sessions, download
blocks, charsets, the definitions download/update flow and the HTTP API).

- `PLENKA_DEFINITIONS_DIR=<dir> go test -run TestAllDefinitions -v ./internal/indexers/`
  loads every `*.yml` in a directory (e.g. a downloaded `$DATA_DIR/definitions`),
  reports how many load and pass the checks, and runs a search with each one against a
  local stub site (no network) to catch runtime template errors.
- `go test -tags live -run TestLive -v ./internal/indexers/` downloads the definitions
  and searches real trackers (env `PLENKA_LIVE_INDEXERS`, `PLENKA_LIVE_QUERY`,
  `DEFINITIONS_URL`).
