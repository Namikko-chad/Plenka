# Torrent Engine

`backend/internal/torrserver/` is the torrent engine. It is derived from
[TorrServer MatriX](https://github.com/YouROK/TorrServer) (GPL-3.0) and modified for
Plenka. Only the engine is kept; the HTTP API on top of it is our own native one in
`backend/internal/torrents/` (`/api/torrents`, `/api/stream`, `/api/engine/settings`, see
[REST API](31-rest-api.md) §1.8–1.10). There are no TorrServer-compatible routes.

## Packages

| Package | Role |
|---|---|
| `torrserver/torr/` | `BTServer` (anacrolix client configuration), `Torrent` (spec, status, speed, idle disconnect), preload, `Stream` (HTTP range serving), the API helper facade (`AddTorrent`, `FindTorrent`, `ListTorrent`, `RemTorrent`, `DropTorrent`, `SetSettings`, …) |
| `torrserver/torr/storage/torrstor/` | the RAM/disk piece cache: per-torrent `Cache`, readers with readahead windows, piece eviction outside reader ranges |
| `torrserver/settings/` | `BTSets` engine settings and the `config.db` bbolt store (settings, saved torrents) |
| `torrserver/log/`, `torrserver/utils/`, `torrserver/mimetype/` | logging, name sorting and media-type helpers |
| `torrents/` | the native HTTP API: link parsing, Torrent/cache/settings JSON, playlists, streaming |

`FindTorrent` returns a loaded torrent or its saved copy (`stat` 5) without loading it;
the file list of a saved torrent comes from its stored metadata. Streams and playlists
load a saved torrent on demand and wait up to 60 s for metadata (`504` otherwise).

## Storage

`config.db` (bbolt, in `DATA_DIR`) holds two buckets: `Settings` (key `BitTorr`, the
`BTSets` JSON) and `Torrents` (one JSON entry per saved torrent: spec with metadata,
title, poster, client `data`, timestamp, size). On start, data left by older builds is
migrated: `settings.json` (where the settings used to live by default) is copied into
`config.db` and removed; the unused `Viewed` bucket and `viewed.json` are deleted. Saved
torrents are kept as they are. `trackers.txt` and `blocklist` in the data dir are still
read when present.

## Settings

The API exposes the fields listed in §1.10 (camelCase). `PUT` clamps out-of-range values,
saves them and restarts the BitTorrent client (loaded torrents are dropped and reconnect
on the next request); `DELETE` saves and applies the defaults. The tracker list fields
(`TrackersListURL`, `DefaultTrackers`) and local peer discovery (`EnableLPD`, `LPDIPv6`)
keep working with their defaults but are not exposed.

Defaults: 64 MB RAM cache, 95 % readahead, 50 % preload, 25 connections per torrent,
idle torrents disconnected after 30 s.

**Cache and kept pieces.** The cache (`cacheSize`, in RAM or on disk with `useDisk`) only
holds the window around the playback position and evicts the rest. Preload and full
download do not use it: their pieces are *kept* on disk, always, whatever `useDisk` says,
in `<torrentsSavePath>/<hash>/keep/` (empty `torrentsSavePath` → `DATA_DIR/torrents`). A
piece is written to `<id>.part` and renamed to `<id>` once the client has verified it;
on load the cache picks the kept pieces up again. Kept pieces are not counted against
`cacheSize` and are never evicted; neither the idle unload nor `removeCacheOnDrop`
deletes them, only removing the torrent (or cancelling a full download, below) does.
A piece already in the cache when it becomes kept moves to disk with its data. Wanted
kept pieces keep their priority (`Normal`, `High` for the preloaded file) whatever the
readers' windows do; a watchdog re-arms it every 5 s.

**Preload** (`POST /api/torrents/{hash}/preload`, `Torrent.Preload`) downloads the first
`preloadCache` % of **every** file plus its last 8/16 MB (the container index), the asked
file first: with 50 % and a 16 GB file, 8 GB go to disk while the 64 MB cache follows
playback. The torrent is in the preload state (`stat` 2, never idle) until all of it is
in or the torrent is closed; `preloadedBytes` / `preloadSize` count the kept pieces.
`preloadCache` 0 turns preload off.

**Full download** (`POST /api/torrents/{hash}/download`) keeps every piece the same way,
and `Torrent.expired` ignores the idle timeout. Cancelling deletes the pieces it fetched
except those kept before it started and those preloads asked for, and re-arms the idle
timeout. Both handlers refuse (`507`) what does not fit in the free disk space under the
keep dir (with what other preloads and full downloads still have to fetch, plus a 512 MB
reserve). On a router with little RAM keep the cache at 64–128 MB and put
`torrentsSavePath` on a USB/NVMe drive.

## Indexer download links

Search results that have no magnet carry a `link` of the form
`/api/indexers/{id}/download?token=…`. `POST /api/torrents` recognises that path
(relative, or absolute to any host) and calls `torrents.IndexerLinkResolver` (set in
`main.go` to the indexer manager's `ResolveLink`), which logs in to the tracker if needed
and returns either the `.torrent` bytes or a magnet URI. Other accepted links: magnet,
http(s) `.torrent` URL, hex or base32 info hash.

## Removed from TorrServer

The TorrServer HTTP API (`/echo`, `/torrents`, `/stream`, `/play`, `/playlist*`,
`/viewed`, `/cache`, `/settings`, `/torrent/upload`, `/ffp`) and its web pages, DLNA,
Bonjour, GStreamer, FUSE/WebDAV, the Telegram bot, MCP, MSX, rutor search and the Torznab
client (replaced by `internal/indexers`), TMDB settings, HTTPS/SSL and WAF, ffprobe,
viewed marks, torrshash links, categories, `file://` links, the JSON settings store and
its migrations, read-only mode, the proxy and public-IP flags.

## Dependency pin

`go.mod` replaces `github.com/anacrolix/torrent` with `github.com/tsynik/torrent v1.2.34`
(and `anacrolix/upnp` with `tsynik/upnp`), exactly as TorrServer does: the cache code
targets that fork's storage API. Moving to upstream anacrolix needs a `torrstor` rewrite.
