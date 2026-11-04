# Plenka

A self-hosted home cinema that fits on a router. One Go binary (or one small container)
browses the TMDB catalog, searches torrent trackers and streams the chosen file straight to a
browser, phone or TV box on your local network. It runs on a MikroTik router (RouterOS 7
container) or on any Linux host or Docker setup.

The interface (web and app) is in English and Russian: it follows the browser or device
language and can be switched in Settings.

## What's inside

- **`backend/` — plenka-server.** One pure-Go binary on port 8095:
  - a torrent streaming engine (derived from [TorrServer](https://github.com/YouROK/TorrServer)
    MatriX) with an in-memory or disk cache;
  - tracker search with its own engine for Cardigann YAML definitions, which it downloads
    from [Jackett](https://github.com/Jackett/Jackett) on start (no Jackett or .NET
    required), plus the built-in Knaben indexer and our own definitions in
    `backend/definitions/` (RuTracker.org) that override downloaded ones with the same id;
    every tracker starts disabled until you enable it;
  - a caching TMDB proxy for the catalog, posters and details;
  - watch history shared by every client ("Continue watching");
  - the embedded web app.
- **`frontend/`** is a Vue 3 web app that looks and behaves like a mobile app. It covers
  the catalog, search, release picker, player, and management screens for torrents,
  trackers and engine settings.
- **`app/`** is a Flutter client for Android phones, Android TV boxes and Linux. It has a
  native player (libmpv) that plays almost any format and finds the server on the LAN by
  itself.

## Quick start

### Docker

```bash
TMDB_KEY=<your TMDB key> docker compose up --build
```

Then open http://localhost:8095.

### From source

You need Go, Node.js with pnpm, and Flutter for the mobile app.

```bash
make build WHAT=server     # builds the web app, embeds it, produces dist/plenka-server-linux-<arch>
DATA_DIR=./data TMDB_KEY=<key> dist/plenka-server-linux-amd64
```

`make build` without arguments is an interactive builder: it asks what to build (server
binary, Docker image, Android APK, Linux app) and for which architectures. See
[Release builds](docs/en/12-deployment.md#release-builds).

For development, run the server and the Vite dev server side by side:

```bash
cd backend && make development              # http://localhost:8095
cd frontend && pnpm install && pnpm dev     # http://localhost:5173, proxies the API to :8095
```

### MikroTik router

Run `make build WHAT=image ARCH=arm64` to build `dist/plenka-server-arm64.tar`, upload it
to the router's disk and run the setup script from
[Deployment → MikroTik router](docs/en/12-deployment.md#mikrotik-router-routeros-7-container-package).

### TMDB key

The catalog needs a free [TMDB API key](https://www.themoviedb.org/settings/api), either a
v3 key or a v4 read token. Set it in the web app under **Settings** or pass
`TMDB_KEY` to the server.

### Tracker definitions

On every start the server downloads the tracker definitions (about 5 MB from GitHub) into
`$DATA_DIR/definitions/` in the background; until the first download finishes only the
built-in Knaben and RuTracker.org indexers are available. **Settings → Trackers → Update
definitions** in the web app updates them on demand. To use a mirror or a pinned version, point `DEFINITIONS_URL` at
any `.tar.gz` archive that has the YAML files in a `Definitions/` directory.

## Documentation

The full documentation is in [`docs/en`](docs/en/02-table-of-contents.md), with a Russian
version in [`docs/ru`](docs/ru/02-table-of-contents.md). The HTTP contract shared by the
server and both clients is [REST API](docs/en/31-rest-api.md).

## Security

Plenka is built for a trusted home network. It has no user accounts and no
authentication, so **do not expose port 8095 to the internet**.

## Disclaimer

Plenka is a tool for playing torrents that you are entitled to download. You are
responsible for what you search for, download and stream, and for complying with the laws
of your country. The project does not host, index or distribute any content.

This product uses the TMDB API but is not endorsed or certified by
[TMDB](https://www.themoviedb.org/).

## License

[GPL-3.0](LICENSE).

Plenka includes code derived from other projects:

- [TorrServer](https://github.com/YouROK/TorrServer) (GPL-3.0): the torrent engine in
  `backend/internal/torrserver/`, modified for Plenka;
- [anacrolix/torrent](https://github.com/anacrolix/torrent) (MPL-2.0) through the
  [tsynik/torrent](https://github.com/tsynik/torrent) fork.

The tracker definitions are not part of this repository or the binary. The server
downloads [Jackett](https://github.com/Jackett/Jackett)'s definitions (GPL-2.0) at run
time from `DEFINITIONS_URL`; they stay under their own license.
