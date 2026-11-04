# REST API

One Go binary (`backend/`) on port **8095** serves everything: the Vue web app, the
API used by the web app and the Flutter app, and the media streams. No auth (LAN
appliance). All times are UTC ISO 8601 (RFC 3339).
CORS: all origins allowed (incl. Private Network Access preflight).

Errors on `/api/*` routes: HTTP 4xx/5xx with `{"error": "<human message>"}`.

---

## 1. API (`/api/*`, camelCase JSON)

### 1.1 Server info

`GET /api/info`
```json
{
  "service": "plenka-server",
  "version": "1.0.0",
  "tmdb": true,                 // TMDB key configured
  "torrents": true,             // torrent engine running
  "indexers": { "total": 120, "enabled": 6 }
}
```
Clients identify the server by `service == "plenka-server"`.

`GET /health` → `204` (container health check).

### 1.2 App config (server-side settings, not engine settings)

`GET /api/config` →
```json
{
  "tmdbApiKey": "",          // always "" in responses; see tmdbApiKeySet
  "tmdbApiKeySet": true,
  "searchTimeoutSec": 30,
  "flareSolverrUrl": ""       // FlareSolverr address, "" = off
}
```
`PUT /api/config` body: any subset of the fields above. `tmdbApiKey: ""` leaves the key
unchanged; to clear it send `"tmdbApiKey": "-"`. `flareSolverrUrl` must be an http(s) URL
or `""` (`400` otherwise); a trailing `/` is dropped. Returns the same shape as GET.
Env `TMDB_KEY` (if set) wins over the stored key and makes `tmdbApiKeySet` true.

### 1.3 TMDB proxy

- `GET /api/tmdb/{path}?{query}` → proxied to `https://api.themoviedb.org/{path}?{query}`
  with the API key added (v3 key as `api_key`, or a v4 token as `Authorization: Bearer`).
  The server adds no `language`: each client sends its own interface language
  (`language=en-US` / `ru-RU`); without it TMDB answers in English. Response is TMDB's JSON
  verbatim. Example: `/api/tmdb/3/movie/popular?page=1&language=en-US`,
  `/api/tmdb/3/search/multi?query=matrix&include_adult=false`,
  `/api/tmdb/3/movie/603?append_to_response=credits`.
  Responses are cached in memory for 10 minutes. No key → `503 {"error":"TMDB key is not configured"}`.
- `GET /img/{tmdb image path}` → proxied to `https://image.tmdb.org/{path}`,
  e.g. `/img/t/p/w342/abc.jpg`. Long `Cache-Control` (immutable). Sizes in use:
  `w342` (list posters), `w500` (details poster), `w780` (16:9 cards), `w1280` (backdrop).

### 1.4 Tracker search

`GET /api/search?q=<text>&indexers=<id,id>&cat=<torznab cat,…>`
- `indexers` optional — default: all enabled indexers.
- `cat` optional Torznab category ids (2000 movies, 5000 TV, …).
- Every indexer is queried in parallel with the configured timeout.
```json
{
  "results": [
    {
      "title": "The Matrix (1999) BDRip 1080p",
      "tracker": "Rutor",               // indexer display name
      "trackerId": "rutor",
      "size": 12345678901,              // bytes
      "seeders": 120,
      "peers": 15,                      // leechers
      "link": "magnet:?xt=urn:btih:…",  // what to give to the torrent engine (magnet preferred,
                                        // else "/api/indexers/{id}/download?token=…" — relative URL)
      "magnet": "magnet:?…",            // may be ""
      "infoHash": "abcdef…",            // lowercase hex, may be ""
      "details": "https://rutor.info/torrent/…",
      "publishDate": "2026-10-01T00:00:00Z",   // may be null
      "category": [2000, 2040],
      "categoryDesc": "Movies/HD"
    }
  ],
  "indexers": [
    { "id": "rutor", "name": "Rutor", "status": "ok", "results": 42, "error": "", "elapsedMs": 812 }
  ]
}
```
`status`: `ok` | `error`. Results are de-duplicated by infoHash (else title+size) and
sorted by seeders desc.

`GET /api/indexers/{id}/download?token=…` — resolves a non-magnet result: logs in if the
indexer needs it, follows the definition's `download` block, then either responds `302`
to a `magnet:` URI or streams the `.torrent` file (`application/x-bittorrent`).
The torrent engine resolves such relative links internally (`POST /api/torrents` accepts
them as `link`; no HTTP round trip needed).

### 1.5 Indexers management

`GET /api/indexers?q=&type=&enabled=` →
```json
[
  {
    "id": "rutor",
    "name": "Rutor",
    "description": "RuTor is a RUSSIAN Public site for MOVIES / TV / GENERAL",
    "language": "ru-RU",
    "type": "public",            // public | semi-private | private
    "links": ["https://rutor.info/"],
    "enabled": true,
    "configured": true,          // all required settings filled (always true for public w/o settings)
    "lastError": "",             // last search/test error, "" if none
    "lastCheckedAt": "2026-11-03T06:00:00Z" // or null
  }
]
```

`GET /api/indexers/{id}` → the list item plus:
```json
{
  "settings": [
    { "name": "username", "type": "text", "label": "Username", "default": "", "options": null },
    { "name": "password", "type": "password", "label": "Password" },
    { "name": "sort", "type": "select", "label": "Sort requested from site", "default": "0",
      "options": [{ "value": "0", "label": "created" }] },
    { "name": "info", "type": "info", "label": "About", "default": "<html text>" }
  ],
  "config": { "username": "me", "password": "", "sitelink": "https://rutor.info/" },
  "categories": [{ "id": 2000, "name": "Movies" }]
}
```
- Setting `type`: `text` | `password` | `checkbox` | `select` | `multi-select` | `info*`.
  `config` holds the saved values. Passwords are never returned: the value is `""` and
  `config["<name>__set"] = "1"` marks a password that is set.
- `config.sitelink` — chosen base URL (one of `links`, or custom).

`PUT /api/indexers/{id}` body `{ "enabled": true, "config": { "username": "…" } }` →
the detail shape. A password field sent as `""` keeps the stored value.

`POST /api/indexers/{id}/test` → `{ "ok": true, "results": 50, "error": "", "elapsedMs": 900 }`
(runs an empty/latest search; logs in first if needed).

The list has the built-in Knaben meta-search indexer, the server's own definitions
(RuTracker.org, id `rutracker`, needs `username`/`password`) and one indexer per
downloaded Cardigann definition (see 1.6); an own definition overrides a downloaded one
with the same id. Every indexer starts disabled. Enabled state and saved settings are kept by indexer id,
so they survive a definitions update; an id that disappears from the definitions is
hidden, and comes back with its settings if a later update brings it back.

### 1.6 Tracker definitions

The server does not ship tracker definitions. It downloads the Cardigann YAML definitions
(by default Jackett's, `src/Jackett.Common/Definitions/*.yml`) as a `.tar.gz` archive from
`DEFINITIONS_URL` and keeps them in `$DATA_DIR/definitions/`. On start it loads what is on
disk and then updates in the background; a failed update keeps the previous set.

`GET /api/definitions` →
```json
{
  "source": "https://codeload.github.com/Jackett/Jackett/tar.gz/refs/heads/master",
  "count": 586,                          // definitions loaded now
  "updatedAt": "2026-11-03T06:00:00Z",   // last successful update, or null
  "updating": false,                     // an update is running
  "lastError": ""                        // error of the last failed update, "" if it succeeded
}
```

`POST /api/definitions/update` → runs an update (or joins the one already running) and
waits for it, then returns the same shape. A failed download or an archive with no
definitions → `502 {"error": "…"}`; the previous definitions stay loaded.

### 1.7 Watch history (shared between web and app)

`GET /api/history?limit=300` → newest first:
```json
[
  {
    "hash": "abc…", "fileId": 3, "fileName": "S01E03.mkv", "torrentTitle": "…",
    "media": { "id": 603, "type": "movie", "title": "The Matrix", "originalTitle": "The Matrix",
               "year": 1999, "posterPath": "/x.jpg", "backdropPath": "/y.jpg",
               "overview": "…", "rating": 8.2 },          // or null
    "positionMs": 120000, "durationMs": 7200000,
    "updatedAt": "2026-11-03T06:00:00Z"
  }
]
```
`PUT /api/history` body: one entry (without `updatedAt`) → saved entry (server sets
`updatedAt`). Key = `hash` + `fileId`; at most 300 entries kept.
`DELETE /api/history/{hash}/{fileId}` → 204. `DELETE /api/history/{hash}` → 204 (all files).
Deleting an entry that does not exist also returns 204.

"Continue watching" = newest entry per media (or per hash if no media), skipped if that
entry has progress > 0.95. Computed on the client.

---

### 1.8 Torrents

`Torrent`:
```json
{
  "hash": "abcdef…",             // lowercase hex infohash
  "title": "The Matrix (1999)",
  "poster": "https://…",         // may be ""
  "data": "{…}",                 // opaque client JSON (the web and app store the media card), may be ""
  "name": "The.Matrix.1999.1080p", // torrent name from metadata, "" until it arrives
  "stat": 3,                     // 0 added, 1 getting info, 2 preload, 3 working, 4 closed, 5 in db (not loaded)
  "statString": "Torrent working",
  "addedAt": "2026-11-03T06:00:00Z",
  "torrentSize": 12345678901,    // bytes; 0 until metadata arrives
  "loadedSize": 0,
  "preloadedBytes": 0,
  "preloadSize": 0,
  "fullDownload": false,         // whole torrent downloaded and kept; progress = loadedSize / torrentSize
  "downloadSpeed": 0,            // bytes/s
  "uploadSpeed": 0,
  "totalPeers": 0,
  "pendingPeers": 0,
  "activePeers": 0,
  "connectedSeeders": 0,
  "halfOpenPeers": 0,
  "bytesRead": 0,
  "files": [{ "id": 1, "path": "The.Matrix.1999.1080p/movie.mkv", "length": 123456789 }]
}
```
All fields are always present; `files` is `[]` until metadata arrives. File ids start at 1.

- `GET /api/torrents` → `[Torrent]`: every saved torrent, loaded or not.
- `POST /api/torrents` body `{"link": "…", "title": "", "poster": "", "data": ""}` →
  `Torrent` right away (metadata arrives in the background; poll `GET /api/torrents/{hash}`
  until `files` is not empty). `link`: magnet, http(s) `.torrent` URL, hex infohash or a
  relative `/api/indexers/{id}/download?token=…` from the search results. The torrent is
  saved and stays in the list across restarts. Adding a torrent that exists updates its
  title, poster and data. A link that is malformed or cannot be fetched or resolved →
  `400`.
- `POST /api/torrents/upload` multipart: `file` (`.torrent`), optional `title`, `poster`,
  `data` → `Torrent`.
- `GET /api/torrents/{hash}` → `Torrent`; unknown hash → `404`.
- `DELETE /api/torrents/{hash}` → `204`: removes the torrent and its cache.
- `POST /api/torrents/{hash}/drop` → `204`: unloads the torrent from memory (frees peers
  and cache); it stays in the list with `stat` 5 and loads again on the next stream.
  Idle torrents are unloaded the same way after `torrentDisconnectTimeout` seconds.
- `POST /api/torrents/{hash}/preload` body `{"fileId": 3}` → `202`: loads the torrent if
  needed, waits for its metadata (`504` if it does not arrive within a minute) and, in the
  background, downloads to disk the first `preloadCache` % plus the last 8/16 MB of
  **every** file, that file first, so playback starts and seeks without buffering. That
  data is kept on disk outside the cache (`<torrentsSavePath>/<hash>/keep/`, or
  `DATA_DIR/torrents/…` when the path is empty, whatever `useDisk` says): it is not
  evicted, survives unloading and a server restart (`removeCacheOnDrop` does not touch it)
  and is deleted with the torrent. While it runs the torrent has `stat` 2, never goes
  idle, and `preloadedBytes` / `preloadSize` show the progress. If what is still missing,
  plus what other preloads and full downloads still have to fetch, plus a 512 MB reserve,
  exceeds the free disk space there → `507` `{"error": "not enough disk space: needs …, …
  free"}`. A missing or bad `fileId` → `400`; unknown hash, or a file id the saved
  metadata does not have → `404`. With `preloadCache` 0 it does nothing.
- `POST /api/torrents/{hash}/download` (no body) → `200` with the torrent: downloads the
  whole torrent to disk the same way as a preload and keeps it until it is cancelled or
  the torrent is removed; meanwhile it never goes idle and `fullDownload` is `true`. It
  loads the torrent and waits for its metadata (`504` if it does not arrive within a
  minute); no room on disk (counted as for a preload) → `507`. Calling it again is a
  no-op. Unknown hash → `404`. The download itself is not resumed after the torrent is
  unloaded or the server restarts; what it fetched stays on disk.
- `DELETE /api/torrents/{hash}/download` → `204`: stops a full download and deletes what it
  fetched, except what was kept before it started and the preloaded parts; the torrent
  goes idle as usual. Unknown hash → `404`.
- `GET /api/torrents/{hash}/cache` →
  ```json
  {
    "hash": "abcdef…", "capacity": 67108864, "filled": 1048576,
    "piecesLength": 4194304, "piecesCount": 2944,
    "pieces": { "12": { "id": 12, "length": 4194304, "size": 1048576, "completed": false, "priority": 1 } },
    "readers": [{ "start": 10, "end": 30, "reader": 12 }]
  }
  ```
  `pieces` holds the pieces that have data (in the cache, or kept on disk by a preload or
  full download) plus every piece queued for download — in a reader's window, a preload or
  a full download — with nothing received yet (`size: 0`, `priority > 0`); `{}` when none.
  `filled` counts only the cache, not the kept pieces. `readers` are the active
  readers (`[]` when none).
- `GET /api/torrents/{hash}/playlist.m3u[?file={id}]` → `audio/x-mpegurl` playlist with
  absolute stream URLs: the one file, or every video/audio file of the torrent in path
  order. Sent as an attachment named after the torrent.

### 1.9 Streams

`GET|HEAD /api/stream/{hash}/{fileId}[/{name}]` — the file's bytes, with `Range` support
and the file's MIME type. `name` is ignored by the server; clients put the URL-encoded file
name there so players show it. A torrent in the list that is not loaded is loaded first.
Unknown or malformed hash or file id → `404`; no metadata within 60 s → `504`.

Players use it directly: `/api/stream/{hash}/{fileId}/{urlencoded file name}`.

### 1.10 Torrent engine settings

`GET /api/engine/settings` →
```json
{
  "cacheSize": 67108864,          // bytes
  "readerReadAhead": 95,          // 5..100 %
  "preloadCache": 50,             // 0..100 % of every file, downloaded to disk
  "useDisk": false,               // the playback cache on disk instead of RAM
  "torrentsSavePath": "",         // preloads, full downloads, disk cache; "" = DATA_DIR/torrents
  "removeCacheOnDrop": false,
  "forceEncrypt": false,
  "retrackersMode": 1,            // 0 none, 1 add, 2 remove, 3 replace
  "torrentDisconnectTimeout": 30, // s
  "enableDebug": false,
  "enableIPv6": false,
  "disableTCP": false,
  "disableUTP": false,
  "disableUPNP": false,
  "disableDHT": false,
  "disablePEX": false,
  "disableUpload": false,
  "downloadRateLimit": 0,         // KB/s, 0 = unlimited
  "uploadRateLimit": 0,
  "connectionsLimit": 25,
  "peersListenPort": 0,           // 0 = random
  "responsiveMode": true
}
```
- `PUT /api/engine/settings` body: the full object → the saved object (out-of-range values
  are clamped). The engine restarts with the new settings; loaded torrents reconnect.
- `DELETE /api/engine/settings` → the default settings, now saved.

---

## 2. LAN discovery (UDP)

The server listens on UDP with the same port number as HTTP (`PORT`, default 8095).
A client broadcasts the ASCII datagram `plenka-server:discover` to that port; the server
answers the sender (unicast) with:

```json
{"service":"plenka-server","version":"4.2.0","port":8095}
```

`port` is the HTTP port. The client takes the server's address from the reply's source
IP and verifies it with `GET /api/info`. Any other datagram is ignored.
