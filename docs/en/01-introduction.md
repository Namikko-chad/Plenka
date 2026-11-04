# Introduction

## What is Plenka?

A self-hosted "watch movies from torrents" system for a home network, sized to run as a
container on a MikroTik router. It has three parts:

- **`backend/` — plenka-server**, one Go binary: torrent streaming over HTTP, search
  across torrent trackers (its own engine for Cardigann YAML definitions, which it
  downloads from Jackett), a TMDB catalog proxy and shared watch history. It also embeds the
  web app, so the router serves everything from one port (8095).
- **`frontend/`** — a Vue 3 web app that looks and behaves like a mobile app: catalog,
  search, release picking, player, and the management screens (torrents, trackers,
  engine settings).
- **`app/`** — a Flutter client (Android phones and TV boxes, Linux desktop) with the same
  flows and a native player (libmpv), which finds the server on the LAN automatically.

## How watching works

1. Pick a title from the TMDB catalog (or search trackers directly).
2. The server searches the enabled trackers in parallel and returns merged releases.
3. Choosing a release adds it to the torrent engine; the file list arrives from peers.
4. The player streams the file from `/api/stream/...` while the engine downloads the needed
   pieces into a RAM (or disk) cache. Playback position is saved to the server, so
   "Continue watching" is shared between the web app and the mobile app.

## Scope

- LAN appliance: no user accounts or authentication. Do not expose port 8095 to the
  internet.
- No transcoding: the built-in browser player plays what the browser supports (MP4/H.264,
  WebM, often HEVC on Safari/Android); other files open in an external player (VLC, MX
  Player) via the stream link or an M3U playlist. The Flutter app plays nearly everything.
- Only the bundled web app and Flutter app are clients: the server does not speak the
  TorrServer or Jackett/Torznab APIs.
- Trackers behind Cloudflare challenges (FlareSolverr) are not supported.

## License

GPL-3.0 (`LICENSE` at the repository root). The torrent engine is derived from
TorrServer MatriX (GPL-3.0). The tracker definitions are not part of the repository or
the binary: the server downloads Jackett's definitions (GPL-2.0) at run time, and its
engine for them is an independent implementation of the Cardigann format.
