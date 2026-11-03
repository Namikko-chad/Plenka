# Plenka

Watch movies and series from torrents at home, served by one container on a MikroTik
router.

- **`backend/` — plenka-server**: one Go binary that streams torrents (TorrServer-compatible
  engine), searches trackers (Jackett-compatible, runs Jackett's Cardigann definitions),
  proxies the TMDB catalog, keeps shared watch history and serves the web app — all on
  port 8095.
- **`frontend/`** — Vue 3 web app styled as a mobile app: catalog, search, releases,
  player, plus torrent, tracker and engine management.
- **`app/`** — Flutter client for Android phones/TV and Linux with a native player; finds
  the server on the LAN by itself.

Original Lampa and other TorrServer/Jackett clients can point at the same server.

## Quick start

```bash
cd backend && make development              # http://localhost:8095
cd frontend && pnpm install && pnpm dev     # http://localhost:5173 (dev, proxies to :8095)
```

Set a TMDB key in the web app (Настройки) or start the backend with `TMDB_KEY=…`.

Docker: `docker compose up --build`. Router: `./build.sh` (arm64 + amd64 images), then `routeros/setup.rsc` —
see [docs/en/12-deployment.md](docs/en/12-deployment.md).

## Layout

```
backend/     Go server (see backend/CLAUDE.md)
frontend/    Vue 3 web app (see frontend/CLAUDE.md)
app/         Flutter client (see app/CLAUDE.md)
routeros/    RouterOS container setup script
docs/        documentation (docs/en, docs/ru — kept in sync)
jackett/, torrserver/   upstream sources kept as reference (not built)
```

## Conventions

- Backend: `make`. Frontend: `pnpm`. Flutter: `flutter analyze`, `flutter test`.
- Every file-modifying change adds a `CHANGELOG.md` entry (Keep a Changelog).
- Documentation changes are applied to all language dirs under `docs/` together.
# Plenka
