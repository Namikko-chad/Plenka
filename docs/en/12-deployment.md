# Deployment

## Local development

```bash
cd backend && make development        # server on :8095, data in backend/data
cd frontend && pnpm install && pnpm dev   # Vite on :5173, proxies the API to :8095
```

Open http://localhost:5173 (hot reload) or http://localhost:8095 (the embedded build;
run `make web` in `backend/` first). Set a TMDB key in Settings or start the backend with
`TMDB_KEY=…`.

## Release builds

`make build` in the repository root is an interactive builder. It asks what to build and
for which architectures, and puts the result into `dist/`:

| Choice | Result | Needs |
|---|---|---|
| `server` | static binary with the web app embedded → `dist/plenka-server-linux-<arch>` | Go, pnpm |
| `image` | Docker image per architecture, saved as `dist/plenka-server-<arch>.tar` (for MikroTik or `docker load`) | Docker with buildx |
| `push` | one multi-arch tag (`:VERSION` and `:latest`) pushed to a registry | Docker with buildx |
| `apk` | Android / Android TV app → `dist/plenka-<version>-android-<arch>.apk` | Flutter, Android SDK |
| `linux` | Linux desktop app → `dist/plenka-<version>-linux-<arch>.tar.gz` (host architecture only) | Flutter, Linux desktop toolchain |

Architectures are `amd64` (x86-64 PCs, MikroTik CHR, the Android emulator), `arm64`
(MikroTik hAP ax²/ax³ and RB5009, Raspberry Pi 4/5, most phones and TV boxes) and
`armv7` (32-bit ARM). Several can be picked at once, or `all`; Enter takes the host
architecture.

The questions are skipped when the answers are passed as variables, for scripts and CI:

```bash
make build WHAT=server ARCH=arm64
make build WHAT=image ARCH=arm64,amd64
make build WHAT=push ARCH=all IMAGE=ghcr.io/<user>/plenka-server
make build WHAT=apk ARCH=arm64,armv7
make build WHAT=linux
```

The version is the newest release heading in `CHANGELOG.md`. The Android `versionCode` is
derived from it (`1.2.3` → `10203`), so each release installs over the previous one. The
APK is signed with the debug key. `push` creates a `plenka-multiarch` buildx builder
(docker-container driver) on first use.

## Single binary

`make build WHAT=server` builds the frontend, embeds it and cross-compiles the server. The
web app is embedded from `backend/web/dist/`, which is git-ignored except for a `.gitkeep`
placeholder. `make web` fills it. A binary built without it (plain `make build` in
`backend/`) still runs, but serves a short page that says the web app is not embedded.

## Docker

```bash
docker compose up --build             # http://localhost:8095, data in the plenka-data volume
```

The root `Dockerfile` builds the Vue app (node), the Go binary with the app embedded
(cross-compiled for `TARGETARCH`), and ships it on `alpine` with `ca-certificates` and
`tzdata`. Environment: `PORT` (8095), `DATA_DIR` (`/data`, a volume), `TMDB_KEY`,
`DEFINITIONS_URL` (optional, see 06-backend-structure).

Trackers behind a Cloudflare check (RuTracker.org from many networks) need
[FlareSolverr](https://github.com/FlareSolverr/FlareSolverr): uncomment the `flaresolverr`
service in `docker-compose.yml` and set `http://flaresolverr:8191` in **Settings → FlareSolverr
address**. That name resolves only between services of the same compose project; for
a container started with `docker run`, `network_mode: host`, a binary or a router, publish
port 8191 and use the host's LAN address (`http://<host IP>:8191`).
It runs a headless Chromium (several hundred MB of RAM), so on a router it usually lives
on another machine.

The container needs outbound HTTPS to GitHub: on every start it downloads the tracker
definitions (about 5 MB) into `/data/definitions/` in the background. Without internet
access it keeps the last downloaded set; on a first start without it only the built-in
Knaben and RuTracker.org indexers are available until
**Settings → Trackers → Update definitions** succeeds.

The image is multi-arch: all build stages run on the build host (`$BUILDPLATFORM`) and
the final stage only copies files, so no qemu emulation is needed. Images for other
architectures come from `make build` (`image` or `push`).

LAN discovery (UDP broadcast on port 8095) needs the server to see broadcasts. A container
on Docker's default bridge network doesn't receive them, so the mobile app won't find it by
itself — enter the address by hand, or on Linux run the container with
`network_mode: host` (commented out in `docker-compose.yml`). A native binary on a PC
receives them; with a host firewall, allow port 8095 for TCP and UDP (e.g.
`sudo ufw allow 8095`). On the router the app finds the server through `x.y.z.1:8095`
without discovery.

## MikroTik router (RouterOS 7, container package)

1. On the router, enable containers once: `/system/device-mode/update container=yes`
   (confirm with the reset button or a power cycle).
2. On a PC: `make build WHAT=image ARCH=arm64` → `dist/plenka-server-arm64.tar` for ARM64
   routers. Use `ARCH=amd64` for x86 CHR and `ARCH=armv7` for 32-bit ARM models.
3. Upload the tar to the router's USB/NVMe disk.
4. Save the script below as `setup.rsc` and edit the variables at the top (disk name, LAN
   IP, optional TMDB key). For a non-ARM64 router, also change the tar name in
   `/container/add`. Upload the file and run `/import file-name=setup.rsc`.

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
# web app: Settings -> Torrent engine). Remove if you do not want to expose a port.
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
/container/add file=usb1/plenka-server-arm64.tar interface=veth-plenka \
    root-dir=usb1/plenka-root mountlists=plenka-data envlists=plenka \
    logging=yes start-on-boot=yes comment="plenka-server"

:delay 15s
/container/start [find comment="plenka-server"]
:put "plenka-server: open http://$lanIP:8095"
```

The script creates the `containers` bridge with the container at `10.77.0.2`, NATs the
container to the internet, forwards `<LAN IP>:8095` to it (where the mobile app looks for
the server), forwards BitTorrent port 6881 from WAN (delete those two rules if you don't
want it), mounts `<disk>/plenka-data` as `/data` and starts the container on boot.

To update, build a new tar, upload it over the old one and recreate the container (the
data dir survives):

```routeros
/container/stop [find comment="plenka-server"]
/container/remove [find comment="plenka-server"]
/container/add file=usb1/plenka-server-arm64.tar interface=veth-plenka \
    root-dir=usb1/plenka-root mounts=plenka-data envlist=plenka \
    logging=yes start-on-boot=yes comment="plenka-server"
/container/start [find comment="plenka-server"]
```

Use a disk for `/data`: router flash is small and wears out. Keep the RAM cache modest
(64–128 MB) or enable the disk cache in Settings → Torrent engine.
