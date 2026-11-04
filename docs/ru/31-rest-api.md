# REST API

Один Go-бинарник (`backend/`) на порту **8095** отдаёт всё: веб-приложение Vue,
API для веб-приложения и Flutter-приложения, и медиапотоки. Авторизации нет (устройство для локальной сети). Всё время
— UTC ISO 8601 (RFC 3339). CORS: разрешены любые источники (включая preflight Private
Network Access).

Ошибки маршрутов `/api/*`: HTTP 4xx/5xx с `{"error": "<сообщение>"}`.

---

## 1. API (`/api/*`, JSON в camelCase)

### 1.1 Информация о сервере

`GET /api/info`
```json
{
  "service": "plenka-server",
  "version": "1.0.0",
  "tmdb": true,                 // ключ TMDB задан
  "torrents": true,             // торрент-движок работает
  "indexers": { "total": 120, "enabled": 6 }
}
```
Клиенты узнают сервер по `service == "plenka-server"`.

`GET /health` → `204` (проверка здоровья контейнера).

### 1.2 Настройки приложения (настройки сервера, не движка)

`GET /api/config` →
```json
{
  "tmdbApiKey": "",          // в ответах всегда ""; см. tmdbApiKeySet
  "tmdbApiKeySet": true,
  "searchTimeoutSec": 30,
  "flareSolverrUrl": ""       // адрес FlareSolverr, "" — выключено
}
```
`PUT /api/config` — тело с любым подмножеством этих полей. `tmdbApiKey: ""` оставляет
ключ как есть; чтобы удалить его, отправьте `"tmdbApiKey": "-"`. `flareSolverrUrl` —
http(s)-адрес или `""` (иначе `400`); завершающий `/` отбрасывается. Ответ в том же
виде, что и GET. Переменная окружения `TMDB_KEY` (если задана) перекрывает сохранённый ключ и
делает `tmdbApiKeySet` истинным.

### 1.3 Прокси TMDB

- `GET /api/tmdb/{path}?{query}` → проксируется на `https://api.themoviedb.org/{path}?{query}`
  с подставленным ключом (ключ v3 как `api_key` или токен v4 как `Authorization: Bearer`).
  Сервер не добавляет `language`: каждый клиент передаёт язык своего интерфейса
  (`language=en-US` / `ru-RU`); без него TMDB отвечает по-английски. Ответ — JSON TMDB
  без изменений. Примеры: `/api/tmdb/3/movie/popular?page=1&language=en-US`,
  `/api/tmdb/3/search/multi?query=matrix&include_adult=false`,
  `/api/tmdb/3/movie/603?append_to_response=credits`.
  Ответы кэшируются в памяти на 10 минут. Нет ключа → `503 {"error":"TMDB key is not configured"}`.
- `GET /img/{путь картинки TMDB}` → проксируется на `https://image.tmdb.org/{path}`,
  например `/img/t/p/w342/abc.jpg`. Долгий `Cache-Control` (immutable). Используемые
  размеры: `w342` (постеры в списках), `w500` (постер в карточке), `w780` (карточки
  16:9), `w1280` (фон).

### 1.4 Поиск по трекерам

`GET /api/search?q=<текст>&indexers=<id,id>&cat=<категория torznab,…>`
- `indexers` необязателен — по умолчанию все включённые трекеры.
- `cat` необязателен — id категорий Torznab (2000 фильмы, 5000 сериалы, …).
- Все трекеры опрашиваются параллельно с заданным таймаутом.
```json
{
  "results": [
    {
      "title": "Матрица / The Matrix (1999) BDRip 1080p",
      "tracker": "Rutor",               // название трекера
      "trackerId": "rutor",
      "size": 12345678901,              // байты
      "seeders": 120,
      "peers": 15,                      // личи
      "link": "magnet:?xt=urn:btih:…",  // что передать торрент-движку (предпочтительно magnet,
                                        // иначе "/api/indexers/{id}/download?token=…" — относительный URL)
      "magnet": "magnet:?…",            // может быть ""
      "infoHash": "abcdef…",            // hex в нижнем регистре, может быть ""
      "details": "https://rutor.info/torrent/…",
      "publishDate": "2026-10-01T00:00:00Z",   // может быть null
      "category": [2000, 2040],
      "categoryDesc": "Movies/HD"
    }
  ],
  "indexers": [
    { "id": "rutor", "name": "Rutor", "status": "ok", "results": 42, "error": "", "elapsedMs": 812 }
  ]
}
```
`status`: `ok` | `error`. Дубли убираются по infoHash (иначе по названию+размеру),
результаты сортируются по сидам по убыванию.

`GET /api/indexers/{id}/download?token=…` — разрешает результат без magnet: при
необходимости входит на трекер, выполняет блок `download` определения и отвечает либо
`302` на `magnet:`, либо файлом `.torrent` (`application/x-bittorrent`). Торрент-движок
разрешает такие относительные ссылки внутри процесса (`POST /api/torrents` принимает их
в `link`), без HTTP-запроса.

### 1.5 Управление трекерами

`GET /api/indexers?q=&type=&enabled=` →
```json
[
  {
    "id": "rutor",
    "name": "Rutor",
    "description": "RuTor is a RUSSIAN Public site for MOVIES / TV / GENERAL",
    "language": "ru-RU",
    "type": "public",            // public | semi-private | private
    "links": ["https://rutor.info/"],
    "enabled": true,
    "configured": true,          // все обязательные настройки заполнены (у публичных без настроек — всегда true)
    "lastError": "",             // последняя ошибка поиска/проверки, "" если нет
    "lastCheckedAt": "2026-11-03T06:00:00Z" // или null
  }
]
```

`GET /api/indexers/{id}` → элемент списка плюс:
```json
{
  "settings": [
    { "name": "username", "type": "text", "label": "Username", "default": "", "options": null },
    { "name": "password", "type": "password", "label": "Password" },
    { "name": "sort", "type": "select", "label": "Sort requested from site", "default": "0",
      "options": [{ "value": "0", "label": "created" }] },
    { "name": "info", "type": "info", "label": "About", "default": "<html text>" }
  ],
  "config": { "username": "me", "password": "", "sitelink": "https://rutor.info/" },
  "categories": [{ "id": 2000, "name": "Movies" }]
}
```
- `type` настройки: `text` | `password` | `checkbox` | `select` | `multi-select` | `info*`.
  `config` содержит сохранённые значения. Пароли не возвращаются никогда: значение
  `""`, а `config["<name>__set"] = "1"` отмечает, что пароль задан.
- `config.sitelink` — выбранный адрес сайта (один из `links` или свой).

`PUT /api/indexers/{id}` с телом `{ "enabled": true, "config": { "username": "…" } }` →
ответ в виде карточки. Пароль, отправленный как `""`, сохраняет прежнее значение.

`POST /api/indexers/{id}/test` → `{ "ok": true, "results": 50, "error": "", "elapsedMs": 900 }`
(выполняет пустой поиск «последних раздач»; при необходимости сначала входит на трекер).

В списке есть встроенный мета-поиск Knaben, собственные определения сервера
(RuTracker.org, id `rutracker`, нужны `username`/`password`) и по одному трекеру на каждое
скачанное определение Cardigann (см. 1.6); собственное определение перекрывает скачанное
с тем же id. Все индексаторы изначально выключены. Включённость и сохранённые настройки хранятся по id
трекера, поэтому переживают обновление определений; трекер, чей id пропал из определений,
скрывается и возвращается со своими настройками, если следующее обновление его вернёт.

### 1.6 Определения трекеров

Сервер не поставляется с определениями трекеров. Он скачивает YAML-определения Cardigann
(по умолчанию — из Jackett, `src/Jackett.Common/Definitions/*.yml`) архивом `.tar.gz` с
`DEFINITIONS_URL` и хранит их в `$DATA_DIR/definitions/`. При запуске он загружает то, что
лежит на диске, и затем обновляет их в фоне; неудачное обновление оставляет прежний набор.

`GET /api/definitions` →
```json
{
  "source": "https://codeload.github.com/Jackett/Jackett/tar.gz/refs/heads/master",
  "count": 586,                          // сколько определений загружено сейчас
  "updatedAt": "2026-11-03T06:00:00Z",   // последнее успешное обновление или null
  "updating": false,                     // идёт обновление
  "lastError": ""                        // ошибка последнего неудачного обновления, "" если оно прошло
}
```

`POST /api/definitions/update` → запускает обновление (или присоединяется к уже идущему),
ждёт его окончания и возвращает тот же объект. Не удалось скачать или в архиве нет
определений → `502 {"error": "…"}`; прежние определения остаются загруженными.

### 1.7 История просмотров (общая для веба и приложения)

`GET /api/history?limit=300` → новые первыми:
```json
[
  {
    "hash": "abc…", "fileId": 3, "fileName": "S01E03.mkv", "torrentTitle": "…",
    "media": { "id": 603, "type": "movie", "title": "Матрица", "originalTitle": "The Matrix",
               "year": 1999, "posterPath": "/x.jpg", "backdropPath": "/y.jpg",
               "overview": "…", "rating": 8.2 },          // или null
    "positionMs": 120000, "durationMs": 7200000,
    "updatedAt": "2026-11-03T06:00:00Z"
  }
]
```
`PUT /api/history` — тело с одной записью (без `updatedAt`) → сохранённая запись
(`updatedAt` ставит сервер). Ключ = `hash` + `fileId`; хранится не больше 300 записей.
`DELETE /api/history/{hash}/{fileId}` → 204. `DELETE /api/history/{hash}` → 204 (все
файлы). Удаление несуществующей записи тоже возвращает 204.

«Продолжить просмотр» = самая новая запись на каждый фильм (или на hash, если фильма
нет), пропускается, если её прогресс > 0.95. Считается на клиенте.

---

### 1.8 Торренты

`Torrent`:
```json
{
  "hash": "abcdef…",             // infohash в hex, строчными
  "title": "Матрица (1999)",
  "poster": "https://…",         // может быть ""
  "data": "{…}",                 // непрозрачный JSON клиента (веб и приложение хранят карточку фильма), может быть ""
  "name": "The.Matrix.1999.1080p", // имя торрента из метаданных, "" пока они не пришли
  "stat": 3,                     // 0 добавлен, 1 получает метаданные, 2 предзагрузка, 3 работает, 4 закрыт, 5 в базе (не загружен)
  "statString": "Torrent working",
  "addedAt": "2026-11-03T06:00:00Z",
  "torrentSize": 12345678901,    // байты; 0 пока нет метаданных
  "loadedSize": 0,
  "preloadedBytes": 0,
  "preloadSize": 0,
  "fullDownload": false,         // вся раздача качается и держится в кэше; прогресс = loadedSize / torrentSize
  "downloadSpeed": 0,            // байт/с
  "uploadSpeed": 0,
  "totalPeers": 0,
  "pendingPeers": 0,
  "activePeers": 0,
  "connectedSeeders": 0,
  "halfOpenPeers": 0,
  "bytesRead": 0,
  "files": [{ "id": 1, "path": "The.Matrix.1999.1080p/movie.mkv", "length": 123456789 }]
}
```
Все поля присутствуют всегда; `files` — `[]`, пока нет метаданных. id файлов начинаются с 1.

- `GET /api/torrents` → `[Torrent]`: все сохранённые торренты, загруженные и нет.
- `POST /api/torrents` с телом `{"link": "…", "title": "", "poster": "", "data": ""}` →
  `Torrent` сразу (метаданные приходят в фоне; опрашивайте `GET /api/torrents/{hash}`,
  пока `files` не станет непустым). `link`: magnet, http(s)-ссылка на `.torrent`, infohash
  в hex или относительная `/api/indexers/{id}/download?token=…` из выдачи поиска. Торрент
  сохраняется и остаётся в списке после перезапуска. Повторное добавление существующего
  торрента обновляет его title, poster и data. Ссылка неверная или её не удалось скачать
  или разрешить → `400`.
- `POST /api/torrents/upload` multipart: `file` (`.torrent`), необязательные `title`,
  `poster`, `data` → `Torrent`.
- `GET /api/torrents/{hash}` → `Torrent`; неизвестный hash → `404`.
- `DELETE /api/torrents/{hash}` → `204`: удаляет торрент и его кэш.
- `POST /api/torrents/{hash}/drop` → `204`: выгружает торрент из памяти (освобождает пиров
  и кэш); он остаётся в списке со `stat` 5 и снова загружается при следующем потоке.
  Простаивающие торренты выгружаются так же через `torrentDisconnectTimeout` секунд.
- `POST /api/torrents/{hash}/preload`, тело `{"fileId": 3}` → `202`: загружает торрент, если
  нужно, ждёт метаданные (`504`, если их нет за минуту) и в фоне качает на диск первые
  `preloadCache` % и последние 8/16 МБ **каждого** файла, этот файл первым, чтобы
  воспроизведение и перемотка шли без буферизации. Эти данные хранятся на диске вне кэша
  (`<torrentsSavePath>/<hash>/keep/`, при пустом пути — `DATA_DIR/torrents/…`, независимо от
  `useDisk`): не вытесняются, переживают выгрузку и перезапуск сервера (`removeCacheOnDrop`
  их не трогает) и удаляются вместе с торрентом. Пока она идёт, у торрента `stat` 2, он не
  отключается по простою, а `preloadedBytes` / `preloadSize` показывают прогресс. Если
  недостающее плюс то, что ещё докачивают другие предзагрузки и загрузки целиком, плюс запас
  512 МБ больше свободного места там → `507` `{"error": "not enough disk space: needs …, …
  free"}`. Нет `fileId` или он неверный → `400`; неизвестный хеш или id файла, которого нет
  в сохранённых метаданных → `404`. При `preloadCache` 0 ничего не делает.
- `POST /api/torrents/{hash}/download` (без тела) → `200` с торрентом: качает всю раздачу на
  диск так же, как предзагрузка, и держит её, пока загрузку не отменят или торрент не удалят;
  всё это время он не отключается по простою, а `fullDownload` равен `true`. Загружает
  торрент и ждёт метаданные (`504`, если их нет за минуту); не хватает места на диске
  (считается как для предзагрузки) → `507`. Повторный вызов ничего не меняет. Неизвестный
  хеш → `404`. Сама загрузка не возобновляется после выгрузки торрента или перезапуска
  сервера; скачанное остаётся на диске.
- `DELETE /api/torrents/{hash}/download` → `204`: отменяет загрузку целиком и удаляет
  скачанное ею, кроме сохранённого до её начала и предзагруженных частей; торрент снова
  отключается по простою. Неизвестный хеш → `404`.
- `GET /api/torrents/{hash}/cache` →
  ```json
  {
    "hash": "abcdef…", "capacity": 67108864, "filled": 1048576,
    "piecesLength": 4194304, "piecesCount": 2944,
    "pieces": { "12": { "id": 12, "length": 4194304, "size": 1048576, "completed": false, "priority": 1 } },
    "readers": [{ "start": 10, "end": 30, "reader": 12 }]
  }
  ```
  В `pieces` — куски с данными (в кэше или сохранённые на диск предзагрузкой / загрузкой
  целиком) и все куски в очереди на загрузку — в окне читателя, предзагрузки или загрузки
  целиком, — по которым ещё ничего не пришло (`size: 0`, `priority > 0`); `{}`, если их нет.
  `filled` считает только кэш, без сохранённых кусков. `readers` — активные
  читатели (`[]`, если их нет).
- `GET /api/torrents/{hash}/playlist.m3u[?file={id}]` → плейлист `audio/x-mpegurl` с
  абсолютными ссылками на потоки: один файл или все видео- и аудиофайлы торрента по
  порядку путей. Отдаётся вложением с именем торрента.

### 1.9 Потоки

`GET|HEAD /api/stream/{hash}/{fileId}[/{name}]` — содержимое файла с поддержкой `Range` и
MIME-типом файла. `name` сервер игнорирует; клиенты кладут туда URL-кодированное имя файла,
чтобы плееры его показывали. Торрент из списка, который не загружен, сначала загружается.
Неизвестный или неверный hash или id файла → `404`; метаданные не пришли за 60 с → `504`.

Плееры используют адрес напрямую: `/api/stream/{hash}/{fileId}/{имя файла в URL-кодировке}`.

### 1.10 Настройки торрент-движка

`GET /api/engine/settings` →
```json
{
  "cacheSize": 67108864,          // байты
  "readerReadAhead": 95,          // 5..100 %
  "preloadCache": 50,             // 0..100 % каждого файла, качается на диск
  "useDisk": false,               // кэш воспроизведения на диске, а не в памяти
  "torrentsSavePath": "",         // предзагрузка, загрузка целиком, кэш на диске; "" = DATA_DIR/torrents
  "removeCacheOnDrop": false,
  "forceEncrypt": false,
  "retrackersMode": 1,            // 0 нет, 1 добавлять, 2 удалять, 3 заменять
  "torrentDisconnectTimeout": 30, // с
  "enableDebug": false,
  "enableIPv6": false,
  "disableTCP": false,
  "disableUTP": false,
  "disableUPNP": false,
  "disableDHT": false,
  "disablePEX": false,
  "disableUpload": false,
  "downloadRateLimit": 0,         // КБ/с, 0 = без ограничения
  "uploadRateLimit": 0,
  "connectionsLimit": 25,
  "peersListenPort": 0,           // 0 = случайный
  "responsiveMode": true
}
```
- `PUT /api/engine/settings` с полным объектом → сохранённый объект (значения вне
  диапазона приводятся к границам). Движок перезапускается с новыми настройками;
  загруженные торренты переподключаются.
- `DELETE /api/engine/settings` → настройки по умолчанию, уже сохранённые.

---

## 2. Поиск сервера в сети (UDP)

Сервер слушает UDP на том же номере порта, что и HTTP (`PORT`, по умолчанию 8095).
Клиент рассылает broadcast-датаграмму ASCII `plenka-server:discover` на этот порт; сервер
отвечает отправителю (unicast):

```json
{"service":"plenka-server","version":"4.2.0","port":8095}
```

`port` — HTTP-порт. Адрес сервера клиент берёт из IP отправителя ответа и проверяет
через `GET /api/info`. Остальные датаграммы игнорируются.
