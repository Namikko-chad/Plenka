# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- **Torrents tab in the Flutter app** ([app/lib/presentation/torrents/](app/lib/presentation/torrents/), [app/lib/presentation/shell/shell.dart](app/lib/presentation/shell/shell.dart), [docs/en/11-mobile-app.md](docs/en/11-mobile-app.md), [docs/ru/11-mobile-app.md](docs/ru/11-mobile-app.md)): the app now has the web UI's Torrents section — the list of the server engine's torrents (`GET /api/torrents`) with status, speed, peers and preload / full-download progress; add by magnet, `.torrent` URL or hash; open files, unload from memory, delete; and a details screen with stats, the cache piece map, preload and full download.

### Changed

- **Release picker renamed** ([app/lib/presentation/releases/releases_screen.dart](app/lib/presentation/releases/releases_screen.dart)): the app's release search screen is now `ReleasesScreen` at `/releases` (was `TorrentsScreen` at `/torrents`), matching the web `ReleasesView`; `/torrents` is the new Torrents tab.

## [1.0.2] - 2026-11-04

### Fixed

- **Built-in player gave up on slow-starting streams** ([app/lib/presentation/player/player_screen.dart](app/lib/presentation/player/player_screen.dart), [docs/en/11-mobile-app.md](docs/en/11-mobile-app.md), [docs/ru/11-mobile-app.md](docs/ru/11-mobile-app.md)): libmpv's network timeout is raised from media_kit's 5 s to 90 s, so a torrent the server first has to load back into the engine (e.g. a fully downloaded one) now plays in the built-in player instead of leaving a black screen at 00:00, as it already did in an external player.

## [1.0.1] - 2026-11-04

### Fixed

- **English-only sources and docs** ([app/](app/), [frontend/src/](frontend/src/), [backend/internal/torrserver/settings/dbreadcache.go](backend/internal/torrserver/settings/dbreadcache.go), [app/README.md](app/README.md), [README.md](README.md), [docker-compose.yml](docker-compose.yml), [docs/en/](docs/en/02-table-of-contents.md)): code comments and the app README are now in English, and the English docs name the English UI labels (e.g. **Settings → Trackers → Update definitions**) instead of the Russian ones. Russian text remains only in the translations (`docs/ru/`, the `ru` locale files) and in data the code matches (tracker categories, size units, test fixtures).
- **Tracker settings location in the docs** ([docs/en/07-indexers.md](docs/en/07-indexers.md), [docs/en/12-deployment.md](docs/en/12-deployment.md), [docs/ru/07-indexers.md](docs/ru/07-indexers.md), [docs/ru/12-deployment.md](docs/ru/12-deployment.md)): trackers are enabled and their definitions updated under **Settings → Trackers**, not on a separate Trackers screen.

## [1.0.0] - 2026-11-04

First public release.

### Added

- **plenka-server** ([backend/](backend/)): one pure-Go binary on port 8095 with bbolt storage in `DATA_DIR`; no external services. `GET /api/info` identifies it, `GET /health` answers 204, and it answers LAN discovery requests on `8095/udp`.
- **Torrent engine** ([backend/internal/torrserver/](backend/internal/torrserver/), [backend/internal/torrents/](backend/internal/torrents/)): derived from TorrServer MatriX; streams any file of a torrent with an in-memory or disk cache. Native API under `/api/torrents`, `/api/stream` and `/api/engine/settings`: add, list, drop, cache piece map, preload, full download to disk. Preloaded and downloaded pieces are kept on disk across restarts.
- **Tracker search** ([backend/internal/indexers/](backend/internal/indexers/)): an own engine for Cardigann YAML definitions, downloaded at run time from `DEFINITIONS_URL` (Jackett's definitions by default), plus the built-in Knaben indexer and own definitions in [backend/definitions/](backend/definitions/) (RuTracker.org). `GET /api/search` queries enabled trackers in parallel; FlareSolverr is supported for protected sites. Every tracker starts disabled.
- **TMDB proxy** ([backend/internal/tmdb/](backend/internal/tmdb/)): `/api/tmdb/*` and `/img/*` with caching; the catalog language follows the client's interface language.
- **Shared watch history** ([backend/internal/history/](backend/internal/history/)): `/api/history` — "Continue watching" is the same in every client.
- **Settings** ([backend/internal/appconfig/](backend/internal/appconfig/)): `GET/PUT /api/config` — TMDB key (never returned), search timeout, FlareSolverr address.
- **Web app** ([frontend/](frontend/)): Vue 3 app embedded into the server, styled as a mobile app: catalog, search, details with trailers, release picker, player with audio and subtitle choice, torrents, and Settings in tabs (general, catalog and search, trackers, torrent engine). English and Russian.
- **Flutter app** ([app/](app/)): Android, Android TV and Linux client with a libmpv player (external subtitle and audio files, video output setting), LAN server discovery, server switching, server settings and TV-remote navigation. English and Russian.
- **Builds and deployment** ([Makefile](Makefile), [Dockerfile](Dockerfile), [docker-compose.yml](docker-compose.yml)): interactive `make build` for the server binary, Docker images (amd64, arm64, armv7; saved as tar or pushed as one multi-arch tag), Android APKs and the Linux app; a MikroTik RouterOS 7 container setup ([docs/en/12-deployment.md](docs/en/12-deployment.md)).
- **CI** ([.github/workflows/ci.yml](.github/workflows/ci.yml)): vet, tests and builds of the backend, the web app and the Flutter app on every push and pull request.
- **Documentation** ([docs/en/](docs/en/02-table-of-contents.md), [docs/ru/](docs/ru/02-table-of-contents.md)) in English and Russian, including the REST API contract.
