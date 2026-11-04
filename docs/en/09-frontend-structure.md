# Frontend Structure

`frontend/` is a Vue 3 single-page app that looks and behaves like a mobile app. It is
built with Vite and embedded into the server binary, so in production it is served from
the same origin as the API and every request uses a relative URL.

Stack: Vue 3 (`<script setup lang="ts">`), vue-router, Pinia, TypeScript strict, plain
CSS with design tokens. No UI kit, no CSS framework. Package manager: **pnpm**.

| Script | What it does |
|---|---|
| `pnpm dev` | Vite on :5173; proxies the server paths `/api/*`, `/img/*` and `/health` to :8095 (`PLENKA_SERVER` overrides the target) |
| `pnpm build` | `vue-tsc` + `vite build` → `frontend/dist` (copied into `backend/web/dist` by `make web`) |
| `pnpm test` | Vitest (happy-dom) unit tests next to the code (`*.test.ts`) |
| `pnpm typecheck` | `vue-tsc --noEmit` |

## Layout

```
src/
  api/          http.ts (JSON fetch, {error} handling, timeouts, abort) + one file per server area
                (info, tmdb, search, indexers, definitions, history, torrents, engine);
                camelCase wire types in types.ts
  stores/       Pinia: server (info + config), catalog, history, torrents, indexers (+ tracker
                definitions state), engine settings, prefs (player mode), playback, toast, dialog
  composables/  useAsync, usePolling, useAddRelease, useClipboard, useMediaQuery,
                usePlayerTracks (audio / subtitle choice of the player)
  utils/        format, files, media, releases, history, external player, sanitize,
                indexer form, subtitles (SRT/VTT/ASS parser), trailers, errors — pure functions with tests
  components/
    layout/     AppShell (KeepAlive for tabs, slide transitions), NavBar (bottom tabs / left rail), AppBar
    ui/         UiButton, IconButton, Badge, StatusDot, Spinner, Skeleton, EmptyState,
                ErrorBanner, Segmented, Chip, Toggle, Sheet, ConfirmDialog, ToastHost,
                ProgressLine, CopyField, Icon (inline SVG paths in icons.ts)
    media/      PosterCard, MediaRow, MediaGrid, BackdropCard, NetImage, ReleaseList,
                ReleaseTile, TorrentCard, PieceMap, ExternalSheet, PlayChooser, TracksPanel
    prefs/      Settings tabs: PrefsGeneral, PrefsCatalog, PrefsTrackers, PrefsEngine
  views/        one component per screen
  styles/       tokens.css, fonts.css, base.css
  i18n/         every user-facing string: en.ts (template), ru.ts (same keys), index.ts —
                `msg` in the current language; browser language by default, English
                fallback; Settings → Language saves `plenka.language` and reloads
```

Data flows `api/` → `stores/` → views and components. Views never call `fetch`; after a
mutation the store updates its state from the response instead of reloading the list.

## Screens

| Route | Screen |
|---|---|
| `/` (tab) | Home: "Continue watching" from `/api/history`, six TMDB rows |
| `/search` (tab) | TMDB multi-search; `?mode=trackers` searches `/api/search` directly |
| `/downloads` (tab) | Torrents in the engine (`/api/torrents`): status, speed, peers, added time, drop/remove, add by link or `.torrent` upload, cache piece map, preload of the last watched (else the first) video, "Download in full" / cancel in the details sheet (progress on the card) |
| `/prefs/:section?` (tab) | Settings, split into tabs (segmented control; the tab is in the URL, switching replaces the history entry): `/prefs` General — server status, interface language, player mode, app version; `/prefs/catalog` Catalog and search — TMDB key (the catalog is requested in the interface language: `language=en-US` / `ru-RU` on every `/api/tmdb` call); `/prefs/trackers` Trackers — tracker search timeout and FlareSolverr address, "Tracker definitions" block (count, last update time, source, last update error, "Update definitions" button), tracker search, filter, enable toggle; `/prefs/engine` Torrent engine — `/api/engine/settings`, reset to defaults. Old `/indexers` and `/indexers/:id` links redirect here |
| `/movie/:id`, `/tv/:id` | Details; "Watch" asks what to watch: torrents (releases) or trailers |
| `/trailers/:type/:id` | The title's YouTube videos from TMDB (`/videos` in the configured language, then `en-US`; trailers first, newest first); a tap plays the video in an embedded YouTube player with an "Open on YouTube" link |
| `/releases/:type/:id` | Tracker releases for a title: query editing, sort, quality chips, per-tracker status |
| `/files/:hash?resume=&browse=1` | Waits for the file list, filters video files, autoplay for one file or resume. With `browse=1` (the "Files" buttons of the Torrents tab) it lists every file of the torrent and starts nothing. Each video has a preload button (`POST /api/torrents/{hash}/preload`); the preload progress shows under the status line |
| `/player/:hash/:fileId` | HTML5 player with custom controls, resume, progress saving, next episode, audio / subtitle panel, external player fallback |
| `/prefs/trackers/:id` | Tracker settings form generated from its definition, save and test |

SPA paths must not collide with server paths (`/api`, `/img`, `/health`). The server
returns `index.html` for any other unknown GET path.

## Behaviour notes

- Phones get a bottom tab bar; at ≥ 840 px it becomes a left navigation rail. Pushed
  screens slide in; tab switches do not animate; tab screens keep state and scroll.
- Browsers cannot play many MKV/AC3/DTS files. On a playback error, or by the
  "external player" button, the player offers: copy the stream link, an M3U playlist, an
  Android `intent:` for VLC or any video app, and `vlc-x-callback` on iOS. The `<video>`
  source and the links are `/api/stream/{hash}/{fileId}/{encoded file name}`; the playlists
  are `/api/torrents/{hash}/playlist.m3u?file={id}` (one file) and
  `/api/torrents/{hash}/playlist.m3u` (whole torrent).
- Audio and subtitles: the player's panel lists the audio tracks the browser exposes
  inside the file (`HTMLMediaElement.audioTracks`: Safari; other browsers play only the
  default track, and the panel says so), the text tracks it exposes, and the torrent's
  separate files: subtitles (`srt`, `vtt`, `ass`, `ssa`; decoded as UTF-8, else
  Windows-1251, and added as a `TextTrack`) and audio (`mka`, `ac3`, `eac3`, `dts`, `aac`,
  `flac`, `mp3`, `m4a`, `opus`, `ogg`, `wav`). A separate audio file plays in a hidden
  `<audio>` kept in sync with the muted video; files named after the video come first.
- No picture: when a video file has sound but a 0×0 picture (the browser cannot decode
  HEVC 10-bit, HDR10+ or Dolby Vision video), the player pauses and offers the external
  player or "sound only".
- Tracker definitions: the server downloads them on start and the list fills in when that
  finishes. While the server reports `updating`, the indexers screen polls
  `GET /api/definitions` every 3 s and reloads the list when the update ends. The
  "Update definitions" button calls `POST /api/definitions/update` (150 s timeout, spinner
  while it runs) and then reloads the list; a failed update shows its error and keeps the
  previous set. With no definitions and only the built-in indexers (Knaben and the own
  RuTracker.org definition), the screen shows an
  empty state explaining that definitions are being downloaded, or offering the button.
- The torrent `data` field holds the TMDB media snapshot as JSON (the same shape as a
  history entry's `media`), so the files and player screens survive a reload.
- Removing a title from "Continue watching" deletes the history of every torrent of
  that title (`DELETE /api/history/{hash}` per torrent), so an older episode does not
  take its place.
- Times from the server are UTC and are shown in the browser's time zone as
  `YYYY-MM-DD HH:MM:SS`.
- PWA manifest and icon allow "Add to home screen"; there is no service worker, so it
  never caches API calls or video.
