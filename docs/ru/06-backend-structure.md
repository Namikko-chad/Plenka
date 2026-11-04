# Структура бэкенда

Go-модуль `plenka` в `backend/`. Сборка и запуск — через `make` (см. `backend/Makefile`):

| Цель | Что делает |
|---|---|
| `make development` | `go run .` с `DATA_DIR=./data` на порту 8095 |
| `make web` | собирает `frontend/` через pnpm и копирует `frontend/dist` в `web/dist` (встраивается) |
| `make build` | статический бинарник (`CGO_ENABLED=0`) в `bin/`; релизные сборки под другие архитектуры — корневой `make build` |
| `make test` / `make vet` | `go test ./...` / `go vet ./...` |

Версия берётся из последнего заголовка релиза в `CHANGELOG.md` и попадает в
`internal/buildinfo.Version`.

## Структура каталогов

```
backend/
  main.go                  запуск и остановка
  definitions/             собственные определения Cardigann (rutracker.yml), встроены; перекрывают скачанные
  web/                     embed.go + dist/ (собранное Vue-приложение; в git — заглушка index.html)
  internal/
    config/                окружение → Config (PORT, DATA_DIR, TMDB_KEY, LOG_PATH, DEFINITIONS_URL)
    buildinfo/             Version (ldflags)
    store/                 обёртка bbolt для plenka.db: Get/Put/Delete/ForEach со значениями в JSON
    appconfig/             настройки приложения (TMDB, таймаут поиска, адрес FlareSolverr) + GET/PUT /api/config
    tmdb/                  прокси /api/tmdb/* и /img/* (подстановка ключа, кэш JSON на 10 минут)
    history/               история просмотров + /api/history
    discovery/             ответчик поиска сервера в сети по UDP (порт = PORT, udp)
    indexers/              поиск по трекерам: свой движок Cardigann, скачивание определений, менеджер, /api/search, /api/indexers/*, /api/definitions*
    torrents/              /api/torrents/*, /api/stream/*, /api/engine/settings поверх движка
    httpserver/            роутер gin: middleware, /api/info, /health, модули, SPA
    torrserver/            торрент-движок на основе TorrServer MatriX (см. 08-torrent-engine)
```

## Переменные окружения

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `PORT` | `8095` | HTTP-порт |
| `DATA_DIR` | `./data` | базы, скачанные определения и `torrents/` (предзагрузка и загрузка целиком), если `torrentsSavePath` пуст |
| `TMDB_KEY` | — | ключ TMDB v3 или токен чтения v4; перекрывает ключ из настроек |
| `LOG_PATH` | — | писать лог в этот файл, а не в stdout |
| `DEFINITIONS_URL` | `https://codeload.github.com/Jackett/Jackett/tar.gz/refs/heads/master` | архив `.tar.gz` с определениями трекеров (все `*.yml` в каталоге `Definitions/`) |

## Цепочка middleware

`WebLogger` (логгер движка, журнал запросов только в режиме отладки) → `gin.Recovery` →
CORS (любые источники, разрешён Private Network Access, разрешён `Range`, отдаётся
`Content-Range`).

## Сборка роутера

`httpserver.New(info, webFS, modules...)` подключает `/api/info` и `/health`, затем
`RegisterRoutes` каждого модуля (appconfig, tmdb, history, indexers, torrents). `NoRoute`
отдаёт SPA: существующие файлы сборки (`assets/*` кэшируются на год), `404` для
неизвестных путей `/api/*` и отсутствующих файлов с расширением, иначе `index.html`.

## Ответы с ошибками

Маршруты `/api/*` отвечают ошибкой `{"error": "<сообщение>"}`: `400` — неверные данные
(в том числе ссылка на торрент, которую не удалось разрешить), `404` — неизвестный id,
`503` — не настроен TMDB, `502` — сбой внешнего сервиса (TMDB, трекер, скачивание
определений), `504` — поток не получил метаданные торрента вовремя.

## Позднее связывание

`torrents.IndexerLinkResolver` — переменная пакета, которой `main.go` присваивает
`indexers.Manager.ResolveLink`. API торрентов вызывает её для ссылок с путём
`/api/indexers/{id}/download`, поэтому не импортирует пакет трекеров.
