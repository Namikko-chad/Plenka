# Backend Structure

Go module `plenka` in `backend/`. Build and run with `make` (see `backend/Makefile`):

| Target | What it does |
|---|---|
| `make development` | `go run .` with `DATA_DIR=./data` on port 8095 |
| `make web` | builds `frontend/` with pnpm and copies `frontend/dist` into `web/dist` (embedded) |
| `make build` | static binary (`CGO_ENABLED=0`) in `bin/`; release builds for other architectures come from the root `make build` |
| `make test` / `make vet` | `go test ./...` / `go vet ./...` |

The version comes from the newest release heading in `CHANGELOG.md` and is injected into
`internal/buildinfo.Version`.

## Layout

```
backend/
  main.go                  startup and shutdown wiring
  definitions/             own Cardigann definitions (rutracker.yml), embedded; override downloaded ones
  web/                     embed.go + dist/ (the built Vue app; placeholder index.html in git)
  internal/
    config/                env → Config (PORT, DATA_DIR, TMDB_KEY, LOG_PATH, DEFINITIONS_URL)
    buildinfo/             Version (ldflags)
    store/                 bbolt wrapper for plenka.db: Get/Put/Delete/ForEach with JSON values
    appconfig/             app settings (TMDB, search timeout, FlareSolverr address) + GET/PUT /api/config
    tmdb/                  /api/tmdb/* and /img/* proxies (key injection, 10-min JSON cache)
    history/               watch history + /api/history
    discovery/             UDP LAN discovery responder (port = PORT, udp)
    indexers/              tracker search: own Cardigann engine, definitions download, manager, /api/search, /api/indexers/*, /api/definitions*
    torrents/              /api/torrents/*, /api/stream/*, /api/engine/settings on top of the engine
    httpserver/            gin router: middleware, /api/info, /health, modules, SPA fallback
    torrserver/            the torrent engine, derived from TorrServer MatriX (see 08-torrent-engine)
```

## Environment

| Variable | Default | Meaning |
|---|---|---|
| `PORT` | `8095` | HTTP port |
| `DATA_DIR` | `./data` | databases, downloaded definitions, and `torrents/` (preloads and full downloads) when `torrentsSavePath` is empty |
| `TMDB_KEY` | — | TMDB v3 key or v4 read token; overrides the key saved in settings |
| `LOG_PATH` | — | write the log to this file instead of stdout |
| `DEFINITIONS_URL` | `https://codeload.github.com/Jackett/Jackett/tar.gz/refs/heads/master` | `.tar.gz` archive with the tracker definitions (every `*.yml` in a `Definitions/` directory) |

## Middleware chain

`WebLogger` (engine logger, debug-only request log) → `gin.Recovery` → CORS (all origins,
Private Network Access allowed, `Range` allowed, `Content-Range` exposed).

## Router composition

`httpserver.New(info, webFS, modules...)` mounts `/api/info` and `/health`, then each
module's `RegisterRoutes` (appconfig, tmdb, history, indexers, torrents). `NoRoute`
serves the SPA: existing files from the bundle (`assets/*` cached for a year), `404` for
unknown `/api/*` paths and missing files with an extension, `index.html` otherwise.

## Error responses

`/api/*` routes answer errors as `{"error": "<message>"}`: `400` for invalid input
(including a torrent link that cannot be resolved), `404` for unknown ids, `503` when TMDB
is not configured, `502` when an upstream (TMDB, a tracker, the definitions download)
fails, `504` when a stream gets no torrent metadata in time.

## Late binding

`torrents.IndexerLinkResolver` is a package variable set in `main.go` to
`indexers.Manager.ResolveLink`. The torrent API calls it for links whose path is
`/api/indexers/{id}/download`, so it never imports the indexers package.
