# Design System

Both clients follow the dark "monitoring dashboard" visual language of
[`DESIGN_MANIFEST.md`](../../DESIGN_MANIFEST.md). Project-local facts (where tokens live,
which component implements what) are in [`DESIGN_SPECIFICATION.md`](../../DESIGN_SPECIFICATION.md).

## Tokens

| Group | Values |
|---|---|
| Palette | bg `#080C10`, surface `#0D1117`, border `#1C2333`, accent `#00D4FF`, text `#E6EDF3`, muted `#8B949E`, on-accent `#0A0E17` |
| Status | ok `#10B981`, error `#EF4444`, active `#3B82F6`, neutral `#6B7280`, warning `#F59E0B`, purple `#7C3AED` / `#A78BFA` |
| Alpha | grid .03, ghost .10 (hover .20), badge .15, segment .12, line .30 |
| Radii | badge 4, segment 6, control 8, card 12, dialog 14 |
| Fonts | JetBrains Mono (body), Syne (headings; Latin only, Cyrillic falls back to mono) — bundled, never loaded from the network |

Web: CSS custom properties in `frontend/src/styles/tokens.css` (`--c-*`, `--rgb-*` for
alpha layering, `--a-*`, `--r-*`, sizes, z-layers). Components use tokens only — no hex
literals. Flutter: `app/lib/core/theme/colors.dart` and `app_theme.dart`.

## Rules

- The page background is bg with a fixed 40 px cyan grid at 3 % alpha.
- One primary (solid cyan) button per screen; other actions are ghost (cyan 10 % fill,
  30 % border) or neutral.
- Cards: surface, 1 px border, radius 12, no shadow. Poster cards scale slightly with a
  cyan border on hover and focus.
- The pulsing status dot is the only element with a glow.
- Every screen has loading (spinner or skeleton), empty (icon, one line, at most one
  button) and error (banner + "Retry") states.
- Mobile ergonomics: touch targets ≥ 44 px, safe-area insets, bottom tab bar on phones and
  a left rail at ≥ 840 px; cyan `:focus-visible` outlines so TV remotes and keyboards work.

## Web components (`frontend/src/components/`)

`ui/`: UiButton, IconButton, Badge, StatusDot, Spinner, Skeleton, EmptyState,
ErrorBanner, Segmented (`block` stretches it to the row, `scroll` scrolls labels that do not fit inside the frame and keeps the active one in view), Chip, Toggle, Sheet (bottom sheet; `dialog` centres it, `wide` widens
it to 960 px for video), ConfirmDialog, ToastHost,
ProgressLine, CopyField, Icon. `layout/`: AppShell, NavBar, AppBar. `media/`: PosterCard,
MediaRow, MediaGrid, BackdropCard, NetImage, ReleaseList, ReleaseTile, TorrentCard,
PieceMap, ExternalSheet, PlayChooser, TracksPanel. `prefs/`: the Settings tab panels
(PrefsGeneral, PrefsCatalog, PrefsTrackers, PrefsEngine), switched by a `block scroll`
Segmented under the app bar. UiButton takes `to` (router link) or
`href` (external link, new tab). Add a variant to a shared component instead of
restyling it in a page.
