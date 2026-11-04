# Развёртывание

## Локальная разработка

```bash
cd backend && make development        # сервер на :8095, данные в backend/data
cd frontend && pnpm install && pnpm dev   # Vite на :5173, проксирует API на :8095
```

Откройте http://localhost:5173 (горячая перезагрузка) или http://localhost:8095
(встроенная сборка; сначала выполните `make web` в `backend/`). Ключ TMDB задаётся в
Настройках или переменной `TMDB_KEY=…` при запуске бэкенда.

## Сборка релизов

`make build` в корне репозитория — интерактивный сборщик. Он спрашивает, что собрать и
под какие архитектуры, и складывает результат в `dist/`:

| Вариант | Результат | Что нужно |
|---|---|---|
| `server` | статический бинарник со встроенным веб-приложением → `dist/plenka-server-linux-<arch>` | Go, pnpm |
| `image` | Docker-образ под каждую архитектуру в `dist/plenka-server-<arch>.tar` (для MikroTik или `docker load`) | Docker с buildx |
| `push` | один мульти-арх тег (`:VERSION` и `:latest`) в registry | Docker с buildx |
| `apk` | приложение для Android / Android TV → `dist/plenka-<version>-android-<arch>.apk` | Flutter, Android SDK |
| `linux` | приложение для Linux → `dist/plenka-<version>-linux-<arch>.tar.gz` (только архитектура хоста) | Flutter, инструменты сборки Linux desktop |

Архитектуры: `amd64` (x86-64 компьютеры, MikroTik CHR, эмулятор Android), `arm64`
(MikroTik hAP ax²/ax³ и RB5009, Raspberry Pi 4/5, большинство телефонов и ТВ-приставок) и
`armv7` (32-битный ARM). Можно выбрать несколько сразу или `all`; Enter — архитектура
хоста.

Для скриптов и CI ответы передаются переменными, и вопросы не задаются:

```bash
make build WHAT=server ARCH=arm64
make build WHAT=image ARCH=arm64,amd64
make build WHAT=push ARCH=all IMAGE=ghcr.io/<user>/plenka-server
make build WHAT=apk ARCH=arm64,armv7
make build WHAT=linux
```

Версия берётся из последнего заголовка релиза в `CHANGELOG.md`. Из неё же считается
`versionCode` для Android (`1.2.3` → `10203`), поэтому каждый релиз ставится поверх
предыдущего. APK подписывается отладочным ключом. При первом запуске `push` создаётся
buildx-сборщик `plenka-multiarch` (драйвер docker-container).

## Один бинарник

`make build WHAT=server` собирает фронтенд, встраивает его и кросс-компилирует сервер.
Веб-приложение встраивается из `backend/web/dist/`. Этот каталог исключён из git, кроме
заглушки `.gitkeep`; заполняет его `make web`. Бинарник, собранный без него (просто
`make build` в `backend/`), запускается, но вместо приложения показывает короткую страницу
о том, что веб-приложение не встроено.

## Docker

```bash
docker compose up --build             # http://localhost:8095, данные в томе plenka-data
```

Корневой `Dockerfile` собирает Vue-приложение (node), Go-бинарник со встроенным
приложением (кросс-компиляция под `TARGETARCH`) и кладёт его в `alpine` с
`ca-certificates` и `tzdata`. Окружение: `PORT` (8095), `DATA_DIR` (`/data`, том),
`TMDB_KEY`, `DEFINITIONS_URL` (необязательно, см. 06-backend-structure).

Трекерам за проверкой Cloudflare (RuTracker.org из многих сетей) нужен
[FlareSolverr](https://github.com/FlareSolverr/FlareSolverr): раскомментируйте сервис
`flaresolverr` в `docker-compose.yml` и укажите `http://flaresolverr:8191` в
**Настройки → Адрес FlareSolverr**. Это имя работает только между сервисами одного
compose-проекта; для контейнера из `docker run`, `network_mode: host`, бинарника или
роутера опубликуйте порт 8191 и укажите LAN-адрес машины (`http://<IP машины>:8191`). Он запускает Chromium без окна (несколько сотен МБ
памяти), поэтому на роутере его обычно держат на другой машине.

Контейнеру нужен исходящий HTTPS до GitHub: при каждом запуске он в фоне скачивает
определения трекеров (около 5 МБ) в `/data/definitions/`. Без доступа в интернет он
работает с последним скачанным набором; при первом запуске без него доступны только
встроенные индексаторы Knaben и RuTracker.org, пока не пройдёт
**Настройки → Трекеры → Обновить определения**.

Образ мульти-архитектурный: все стадии сборки идут на хосте (`$BUILDPLATFORM`), а
финальная стадия только копирует файлы, поэтому эмуляция qemu не нужна. Образы под
другие архитектуры собирает `make build` (`image` или `push`).

Поиск сервера в сети (UDP-broadcast на порт 8095) работает, только если сервер получает
broadcast. Контейнер в стандартной сети Docker (bridge) их не получает, поэтому
мобильное приложение само его не найдёт — введите адрес вручную или на Linux запустите
контейнер с `network_mode: host` (закомментировано в `docker-compose.yml`). Бинарник на
компьютере их получает; если включён файрвол, откройте порт 8095 для TCP и UDP
(например, `sudo ufw allow 8095`). На роутере приложение находит сервер по адресу
`x.y.z.1:8095` и без broadcast.

## Роутер MikroTik (RouterOS 7, пакет container)

1. На роутере один раз включите контейнеры: `/system/device-mode/update container=yes`
   (подтверждается кнопкой reset или перезагрузкой по питанию).
2. На компьютере: `make build WHAT=image ARCH=arm64` → `dist/plenka-server-arm64.tar` для
   роутеров на ARM64. Для x86 CHR — `ARCH=amd64`, для 32-битных ARM-моделей — `ARCH=armv7`.
3. Загрузите tar на USB/NVMe-диск роутера.
4. Сохраните скрипт ниже как `setup.rsc` и поправьте переменные в начале (имя диска,
   LAN-адрес, ключ TMDB по желанию). Для роутера не на ARM64 поменяйте и имя tar в
   `/container/add`. Загрузите файл и выполните `/import file-name=setup.rsc`.

```routeros
# plenka-server on a MikroTik router (RouterOS 7.x with the container package).
# Edit the variables, then: /import file-name=setup.rsc

:local disk "usb1"
:local lanIP "192.168.88.1"
:local tmdbKey ""

# --- network: a bridge for containers, the container gets 10.77.0.2 -----------------
/interface/veth/add name=veth-plenka address=10.77.0.2/24 gateway=10.77.0.1
/interface/bridge/add name=containers
/ip/address/add address=10.77.0.1/24 interface=containers
/interface/bridge/port/add bridge=containers interface=veth-plenka
/ip/firewall/nat/add chain=srcnat action=masquerade src-address=10.77.0.0/24 comment="plenka: container internet"

# Web app + API on the router's own LAN address (port 8095).
/ip/firewall/nat/add chain=dstnat action=dst-nat protocol=tcp dst-address=$lanIP dst-port=8095 \
    to-addresses=10.77.0.2 to-ports=8095 comment="plenka: web/api"

# Optional: incoming BitTorrent peers from the internet (set PeersListenPort=6881 in the
# web app: Настройки -> Торрент-движок). Remove if you do not want to expose a port.
/ip/firewall/nat/add chain=dstnat action=dst-nat protocol=tcp in-interface-list=WAN dst-port=6881 \
    to-addresses=10.77.0.2 to-ports=6881 comment="plenka: bt peers tcp"
/ip/firewall/nat/add chain=dstnat action=dst-nat protocol=udp in-interface-list=WAN dst-port=6881 \
    to-addresses=10.77.0.2 to-ports=6881 comment="plenka: bt peers udp"

# --- container ---------------------------------------------------------------------
# RouterOS < 7.20 uses "name=" for envs and mounts; 7.20+ uses "list=" for envs.
/container/envs/add name=plenka key=DATA_DIR value=/data
:if ([:len $tmdbKey] > 0) do={ /container/envs/add name=plenka key=TMDB_KEY value=$tmdbKey }
/container/mounts/add name=plenka-data src=("/" . $disk . "/plenka-data") dst=/data

/container/config/set ram-high=400M
/container/add file=($disk . "/plenka-server-arm64.tar") interface=veth-plenka \
    root-dir=($disk . "/plenka-root") mounts=plenka-data envlist=plenka \
    logging=yes start-on-boot=yes comment="plenka-server"

:delay 15s
/container/start [find comment="plenka-server"]
:put "plenka-server: open http://$lanIP:8095"
```

Скрипт создаёт мост `containers` с контейнером на `10.77.0.2`, выпускает контейнер в
интернет через NAT, пробрасывает `<LAN-адрес>:8095` в контейнер (там сервер ищет
мобильное приложение), пробрасывает порт BitTorrent 6881 с WAN (удалите эти два правила,
если это не нужно), монтирует `<диск>/plenka-data` как `/data` и включает автозапуск.

Обновление: соберите новый tar, загрузите его поверх старого и пересоздайте контейнер
(каталог данных сохранится):

```routeros
/container/stop [find comment="plenka-server"]
/container/remove [find comment="plenka-server"]
/container/add file=usb1/plenka-server-arm64.tar interface=veth-plenka \
    root-dir=usb1/plenka-root mounts=plenka-data envlist=plenka \
    logging=yes start-on-boot=yes comment="plenka-server"
/container/start [find comment="plenka-server"]
```

Держите `/data` на диске: флеш-память роутера мала и изнашивается. Кэш в памяти держите
небольшим (64–128 МБ) или включите кэш на диске в Настройки → Торрент-движок.
