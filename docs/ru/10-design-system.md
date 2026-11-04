# Дизайн-система

Оба клиента следуют тёмному визуальному языку «дашборда мониторинга» из
[`DESIGN_MANIFEST.md`](../../DESIGN_MANIFEST.md). Факты, специфичные для проекта (где
лежат токены, какой компонент что реализует), — в
[`DESIGN_SPECIFICATION.md`](../../DESIGN_SPECIFICATION.md).

## Токены

| Группа | Значения |
|---|---|
| Палитра | bg `#080C10`, surface `#0D1117`, border `#1C2333`, accent `#00D4FF`, text `#E6EDF3`, muted `#8B949E`, on-accent `#0A0E17` |
| Статусы | ok `#10B981`, error `#EF4444`, active `#3B82F6`, neutral `#6B7280`, warning `#F59E0B`, purple `#7C3AED` / `#A78BFA` |
| Прозрачность | сетка .03, ghost .10 (наведение .20), бейдж .15, сегмент .12, линия .30 |
| Скругления | бейдж 4, сегмент 6, контрол 8, карточка 12, диалог 14 |
| Шрифты | JetBrains Mono (текст), Syne (заголовки; только латиница, кириллица берётся из mono) — лежат в проекте, из сети не загружаются |

Веб: CSS-переменные в `frontend/src/styles/tokens.css` (`--c-*`, `--rgb-*` для слоёв с
прозрачностью, `--a-*`, `--r-*`, размеры, z-слои). Компоненты используют только токены —
без hex-литералов. Flutter: `app/lib/core/theme/colors.dart` и `app_theme.dart`.

## Правила

- Фон страницы — bg с фиксированной голубой сеткой 40 px при прозрачности 3 %.
- Одна основная (сплошная голубая) кнопка на экран; остальные действия — ghost (заливка
  голубым 10 %, рамка 30 %) или нейтральные.
- Карточки: surface, рамка 1 px, скругление 12, без тени. Постеры слегка увеличиваются с
  голубой рамкой при наведении и фокусе.
- Пульсирующая точка статуса — единственный элемент со свечением.
- У каждого экрана есть состояния загрузки (спиннер или скелетон), пустое (иконка, одна
  строка, не больше одной кнопки) и ошибки (баннер + «Повторить»).
- Мобильная эргономика: зоны нажатия ≥ 44 px, отступы safe-area, нижние вкладки на
  телефоне и боковая панель от 840 px; голубой контур `:focus-visible`, чтобы работали
  пульт ТВ и клавиатура.

## Веб-компоненты (`frontend/src/components/`)

`ui/`: UiButton, IconButton, Badge, StatusDot, Spinner, Skeleton, EmptyState,
ErrorBanner, Segmented (`block` растягивает его на всю строку, `scroll` прокручивает не поместившиеся подписи внутри рамки и держит активную в поле зрения), Chip, Toggle, Sheet (нижняя шторка; `dialog` — по центру, `wide` —
шириной до 960 px для видео), ConfirmDialog, ToastHost,
ProgressLine, CopyField, Icon. `layout/`: AppShell, NavBar, AppBar. `media/`:
PosterCard, MediaRow, MediaGrid, BackdropCard, NetImage, ReleaseList, ReleaseTile,
TorrentCard, PieceMap, ExternalSheet, PlayChooser, TracksPanel. `prefs/`: панели вкладок
настроек (PrefsGeneral, PrefsCatalog, PrefsTrackers, PrefsEngine), переключаемые
Segmented с `block scroll` под шапкой. UiButton принимает `to` (ссылка роутера)
или `href` (внешняя ссылка в новой вкладке). Нужен другой вид — добавьте вариант в
общий компонент, а не переопределяйте стили на странице.
