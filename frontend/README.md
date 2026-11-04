# Plenka web

Mobile-app-style web client for plenka-server (torrent engine + tracker search + TMDB
proxy in one Go binary), including its torrent, engine and tracker admin screens. Pure Vue 3 + Vite +
vue-router + Pinia, TypeScript strict, hand-written CSS. Russian UI, dark theme.

## Scripts (pnpm only)

| Command          | What it does                                          |
| ---------------- | ----------------------------------------------------- |
| `pnpm install`   | install dependencies                                  |
| `pnpm dev`       | dev server on http://localhost:5173                   |
| `pnpm build`     | `vue-tsc` type check + production build → `dist/`     |
| `pnpm preview`   | serve the built `dist/`                               |
| `pnpm typecheck` | type check only                                       |
| `pnpm test`      | unit tests (vitest + happy-dom)                       |

## Dev proxy

`pnpm dev` proxies server paths to `http://localhost:8095` (override with
`PLENKA_SERVER=http://host:port pnpm dev`): `/api/*`, `/img/*`, `/health`. The proxy
rules are exact regexes, and SPA routes never reuse a server path, so reloading any SPA
page works
both in dev and behind the Go server.

## How the build is served

`pnpm build` writes `dist/` with base `/`. The Go server embeds `dist` and serves it at
the same origin with an SPA fallback (unknown GET paths → `index.html`), so every API
call in the app is a same-origin relative URL. Fonts are self-hosted in `public/fonts/`;
`public/manifest.webmanifest` + `public/icon.svg` make the app installable to the home
screen.

API contract: `../docs/en/31-rest-api.md`.
