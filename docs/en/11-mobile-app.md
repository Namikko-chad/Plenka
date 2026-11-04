# Mobile App

`app/` is a Flutter client (package `plenka`) for Android phones and TV boxes and for Linux
desktop. It talks only to plenka-server and plays video with a native player (media_kit /
libmpv), so it handles MKV, AC3, DTS and embedded subtitles that a browser cannot.

Stack: flutter_bloc (cubits), go_router, get_it, dio, shared_preferences, media_kit,
android_intent_plus. Gate: `flutter analyze` (zero issues) and `flutter test`.

## Finding the server

The app accepts only servers whose `GET /api/info` answers `service == "plenka-server"`.

**A server set by hand** (connect screen or Settings → Change server) is pinned
(`server_manual`): on start the app probes only that address — three attempts 2 s apart
with long timeouts (6 s connect, 8 s response), because a TV's network comes up late and a
remote server answers slowly. If it still does not answer, the connect screen names it and
offers Retry, Search automatically and another address; auto-discovery never replaces it.
Settings → Find the server automatically drops the pin.

**Otherwise** it probes the build-time `SERVER_URL`, then the last found `server_url` (long
timeouts), then in parallel:

- a UDP broadcast of `plenka-server:discover` to `255.255.255.255` and `x.y.z.255` of every
  local IPv4 network on port 8095 (sent twice, replies collected for 2.5 s); a reply gives
  the server's address (sender IP + `port` from the JSON), which is then checked with
  `/api/info` — this finds a server on any LAN address, e.g. a PC;
- `x.y.z.1:8095` for every local IPv4 network, `192.168.88.1`, `10.77.0.2`, `10.0.2.2`
  and `127.0.0.1`.

The first server to answer wins and is saved as `server_url`. If nothing answers, the
connect screen lets the user enter an address.

Addresses: `192.168.88.1` → `http://192.168.88.1:8095`; with a scheme and no port
(`https://kino.example.com`) the scheme's own port is kept, for a server behind a proxy.

Build options (`--dart-define`): `SERVER_URL`, `SERVER_PORT` (default 8095).

## Layout

```
lib/
  l10n/          app_en.arb, app_ru.arb → generated AppLocalizations (flutter gen-l10n)
  core/          config (dart-defines), di (get_it), theme (design tokens), utils (format, natural sort)
  data/server/   discovery + ServerClient (shared Dio, absolute URLs for player and images)
  data/api/      tmdb_api, search_api (/api/search), torrent_api (/api/torrents, /api/stream URLs), history_api,
                 config_api (/api/config), indexers_api (/api/indexers, /api/definitions), engine_api (/api/engine/settings)
  data/models/   media, torrent, watch_entry, server_settings
  data/storage/  settings_store (server URL and whether it is pinned, player mode, video output, language), history_store (local cache + sync)
  presentation/  connect, shell (tabs / nav rail), home, search, details, trailers, releases,
                 torrents (Torrents tab and torrent details), files, player, settings, server_settings (trackers, tracker, catalog, engine), common widgets, router
```

## Flows

- **Releases:** queries `"{title} {year}"` and `"{originalTitle} {year}"` (year only for
  movies) in parallel, merges by infoHash or title+size, sorts by seeders. Shows how many
  trackers answered and which failed; with none enabled it points to `{server}/indexers`.
  `magnet` is preferred; otherwise `link` (possibly a relative indexer download link) is
  passed to the engine unchanged.
- **Torrent add:** `POST /api/torrents` with `{link, title, poster, data}` (the media card
  as a JSON string in `data`; the server always saves the torrent); the app then polls
  `GET /api/torrents/{hash}` once a second until `files` is not empty (a 404 right after
  adding is retried; gives up after 90 s).
- **Player:** stream URL `{server}/api/stream/{hash}/{fileId}/{URL-encoded file name}`
  (file ids start at 1; the name is only for the player); internal,
  external (Android intent) or "ask" mode. The built-in player's Audio and Subtitles
  pickers list the file's own tracks plus the torrent's separate subtitle files (`srt`,
  `ass`, `ssa`, `vtt`, `sub`) and audio files (`mka`, `ac3`, `eac3`, `dts`, `aac`, `flac`,
  `mp3`, `m4a`, `opus`, `ogg`, `wav`); a separate file is added to libmpv only when picked.
  Files named after the video come first; the buttons show the current choice. libmpv's
  network timeout is raised from media_kit's 5 s to 90 s: the first byte can take a while
  when the server loads a saved torrent back into the engine.
- **Video output (Android):** Settings → Video output. "Hardware — MediaCodec" (default)
  renders with `vo=mediacodec_embed`, `hwdec=mediacodec`, which shows 10-bit HEVC,
  HDR10+ and Dolby Vision that the OpenGL path renders black; "Compatible — OpenGL" keeps
  media_kit's defaults (`vo=gpu`, `hwdec=auto-safe`). After a playback error in hardware
  mode the player suggests the compatible mode once. Linux always uses the defaults.
- **Watch → Torrents / Trailers:** the details screen's Watch button asks what to watch.
  Trailers (`/trailers/{type}/{id}`) lists the title's YouTube videos from TMDB `/videos`
  (catalog language, then `en-US`; trailers first, newest first); a video opens in YouTube
  (Android `VIEW` intent, `xdg-open` on Linux).
- **Files and preload:** each release has a Files button that opens the file list in
  browse mode (every file, non-video files dimmed, nothing starts by itself); tapping the
  release still autoplays a single video. Each video row has a Preload button
  (`POST /api/torrents/{hash}/preload`); while `stat` is 2 the screen shows
  "Preload: X / Y" (the preload goes to the server's disk). A button in the app bar downloads
  the whole torrent and keeps it on the server's disk (`POST`/`DELETE /api/torrents/{hash}/download`, `507` → "does not fit");
  the screen shows "In full: X / Y".
- **Torrents tab** (`/torrents`, between Search and Settings, like the web Torrents tab):
  every torrent saved in the server's engine (`GET /api/torrents`, polled every 3 s while
  the tab is open; pull to refresh). A card shows the poster, title, engine status,
  size, full-download badge, speed and peers of a loaded torrent, and a progress line for a
  running preload or full download. Card buttons: Files (the file list in browse mode,
  nothing starts by itself), Unload from memory (`POST /api/torrents/{hash}/drop`, only
  for a loaded torrent) and Delete (`DELETE /api/torrents/{hash}`, after a confirmation).
  **Add** takes a magnet, `.torrent` URL or hash plus an optional title
  (`POST /api/torrents`); there is no `.torrent` file upload in the app. Tapping a card
  opens its details (`/torrent`): stats (status, size, loaded, speeds, peers, seeders,
  full download and preload progress, time added in the device's timezone, the hash —
  tap to copy), the cache piece map (`GET /api/torrents/{hash}/cache`; loaded, partial,
  queued, being read) and buttons Files, Preload (the last watched file of this torrent,
  else the first video), Download in full / Cancel the download and Delete; the torrent
  and its cache are polled every 2 s.
- **Watch history:** stored on the server (`/api/history`) with a local copy, so Home
  shows instantly and works offline. Saves go local first; network writes run in the
  background, and failed ones wait in a persisted queue. Sync runs on connect, on opening
  Home and on pull-to-refresh: the newer `updatedAt` wins, entries deleted on the server
  are dropped locally, and old local-only history is uploaded once.
- **Language:** English and Russian. Texts live in `lib/l10n/app_*.arb` (English is the
  template); widgets read them with `context.l10n`, code without a `BuildContext` (error
  texts, size units, TMDB requests) with the global `tr`. By default the app follows the
  device language and falls back to English when it is not supported; Settings →
  Language switches between "Device language", English and Russian (saved as
  `app_language`). TMDB requests send `language` matching the interface (`en-US` /
  `ru-RU`); the catalog language cannot be set separately.
- **TV remote:** each tab lives in a nested navigator with its own focus scope, which
  stops arrow keys at its edge; the shell catches "left" that finds nothing inside the tab
  and moves focus to the current item of the side rail. "Right" from the rail goes back
  into the tab. Up/down always move focus, also out of a text field (Flutter would move
  the caret instead and the user could not leave a form field); the app has no multi-line
  fields.
- **Settings:** server address and version, whether it was set by hand or found; Change
  server; Find the server automatically; status of torrents, trackers ("enabled N of M")
  and TMDB; interface language and player mode. **Server settings** are edited in the app,
  with no trip to the web UI:
  - *Trackers* (`/server/indexers`): definitions status and update, search and filter
    (all / enabled / public / private), on/off switch per tracker (switching on a tracker
    that still needs a login opens its settings). A tracker screen
    (`/server/indexers/{id}`) has the on/off switch, the site address (one of `links` or a
    custom one), every definition field (text, password, checkbox, select, multi-select;
    `info` HTML shown as text) and Test, which saves the form first. Values go back in
    the shape the server sent them; an empty password keeps the stored one.
  - *Catalog and search* (`/server/catalog`): TMDB key (set, replace, remove), tracker
    search timeout, FlareSolverr address.
  - *Torrent engine* (`/server/engine`): cache (MB), read-ahead, preload, limits, ports,
    retrackers, downloads folder (always shown), disk cache, protocol switches; Reset to defaults. The full settings object
    is sent back, so fields the app does not know are kept.
