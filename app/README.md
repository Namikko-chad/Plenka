# Plenka (Flutter app)

Native Plenka client (Android / Android TV / Linux) for **plenka-server**: TMDB catalog,
torrent search across trackers, streaming torrent playback and "Continue watching", shared
with the server's web UI.

## Server

The app needs `plenka-server` (a single Go binary, port **8095**) on the same network — the API
contract is [`../docs/en/31-rest-api.md`](../docs/en/31-rest-api.md). On launch the app discovers
the server by itself (`GET /api/info`, `service == "plenka-server"`) at the router address of each
device network (`x.y.z.1`), `192.168.88.1`, `10.77.0.2`, `10.0.2.2`, `127.0.0.1`; the found address
is remembered. The address can be entered manually on the connection screen or in Settings →
"Change server"; such an address is pinned: on launch the app connects only to it (with retries)
and never replaces it with an auto-discovered one.

Everything is configured from the app (Settings → Server settings): trackers (enabling, logins,
testing, updating definitions), the TMDB key, the search timeout, FlareSolverr and the torrent
engine.

## `--dart-define`

| Key | Default | Purpose |
|---|---|---|
| `SERVER_URL` | — | Explicit server address (`http://192.168.88.1:8095`); tried first |
| `SERVER_PORT` | `8095` | Port for auto-discovery and for addresses without a port |

## Building

```sh
flutter pub get
flutter analyze && flutter test
flutter run -d linux --dart-define=SERVER_URL=http://127.0.0.1:8095
flutter build apk --release        # Android / Android TV
flutter build linux --release
```

Per-architecture release APKs and the Linux archive are built by `make build` at the repository
root (`apk` and `linux` variants), see `docs/en/12-deployment.md`.
