# Architecture

```
 phone / TV / browser                         MikroTik router (container, 10.77.0.2)
 ┌────────────────────┐     http :8095      ┌──────────────────────────────────────────┐
 │ Vue web app (SPA)  │ ──────────────────► │ plenka-server (one Go process)           │
 │ Flutter app        │                     │  ├─ embedded web app (SPA fallback)      │
 └────────────────────┘                     │  ├─ /api/info, /api/config               │
                                            │  ├─ /api/tmdb/*, /img/*  ──► TMDB        │
                                            │  ├─ /api/search, /api/indexers/* ─► trackers
                                            │  ├─ /api/definitions ──► GitHub (Jackett)│
                                            │  ├─ /api/history                         │
                                            │  ├─ /api/torrents, /api/engine/settings  │
                                            │  └─ /api/stream/*                        │
                                            │        torrent engine ◄──► BitTorrent peers
                                            │  data dir: plenka.db, config.db,         │
                                            │            definitions/, cache           │
                                            └──────────────────────────────────────────┘
```

## Process wiring (`backend/main.go`)

Startup order:

1. `config.Load()` reads env (`PORT`, `DATA_DIR`, `TMDB_KEY`, `LOG_PATH`,
   `DEFINITIONS_URL`) and creates the data dir.
2. The torrent engine's settings package opens its own bbolt file `config.db` in the data
   dir (`tssettings.InitSets`), holding engine settings and saved torrents.
3. `store.Open` opens `plenka.db` for the server's own data: app config, indexer state and
   cookies, definitions update state, watch history.
4. Feature services are built: `appconfig`, `tmdb`, `history`, `indexers`. The indexer
   manager loads the definitions found in `$DATA_DIR/definitions/` and
   `StartDefinitionsUpdate` downloads a fresh set in the background.
5. `torrents.IndexerLinkResolver = idx.ResolveLink` — late binding that lets the torrent
   API resolve `/api/indexers/{id}/download?token=…` links in-process, without importing
   the indexers package.
6. The BitTorrent client starts (`torr.NewBTS().Connect()`).
7. `httpserver.New` assembles the gin router with the feature modules (appconfig, tmdb,
   history, indexers, torrents) and the HTTP server starts. SIGINT/SIGTERM shut it down
   gracefully, then the engine and both databases are closed.
8. `discovery.Listen` answers LAN discovery probes on UDP with the same port number
   (see 31-rest-api, section 2). A failure to bind is logged and does not stop the server.

## Data flow: watching a title

1. Client → `GET /api/tmdb/3/movie/{id}` (details) — proxied with the key added, cached 10 min.
2. Client → `GET /api/search?q=…` — the indexer manager queries every enabled tracker in
   parallel with the configured timeout, de-duplicates and sorts by seeders.
3. Client → `POST /api/torrents {"link":…}` — the link (magnet, .torrent URL, infohash or
   an indexer download link) is resolved, the torrent is saved and connects to peers.
4. Client polls `GET /api/torrents/{hash}` until `files` arrive.
5. Player → `GET /api/stream/{hash}/{fileId}/{name}` — Range requests are served from the
   engine's reader; pieces ahead of the reader get priority (readahead).
6. Client → `PUT /api/history` every 5 s with the position.

## Storage

| File | Owner | Content |
|---|---|---|
| `plenka.db` | `internal/store` | buckets `config` (app config), `indexers` (enabled flag, settings, last error), `indexers_cookies`, `indexers_meta` (download-token key, defaults applied per id, definitions update state), `history` |
| `config.db` | `internal/torrserver/settings` | `Settings/BitTorr` (engine settings), `Torrents/<hash>` |
| `definitions/` | `internal/indexers` | downloaded Cardigann YAML definitions, replaced as a whole on each successful update |
| torrents dir | engine | `TorrentsSavePath` (empty → `DATA_DIR/torrents`): `<hash>/keep/` holds preloaded and fully downloaded pieces (always on disk, removed with the torrent); `<hash>/<piece>` is the playback cache when "use disk" is on |

All timestamps are stored and served in UTC.
