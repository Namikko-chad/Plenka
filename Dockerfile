# plenka-server: one image with the Go server and the embedded Vue app.
# Multi-arch: every build stage runs on the build host ($BUILDPLATFORM) and Go
# cross-compiles, the final stage only copies files — no qemu emulation needed.
# Per-architecture tars (MikroTik) and multi-arch pushes: make build (image / push).

FROM --platform=$BUILDPLATFORM node:24-alpine AS web
WORKDIR /src/frontend
RUN corepack enable
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS server
ARG TARGETOS TARGETARCH TARGETVARIANT VERSION=dev
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN find web/dist -mindepth 1 -delete
COPY --from=web /src/frontend/dist/ web/dist/
RUN GOARM=${TARGETVARIANT#v} CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
      -ldflags "-s -w -checklinkname=0 -X plenka/internal/buildinfo.Version=${VERSION}" \
      -o /out/plenka-server .

# CA certificates and zoneinfo are architecture-independent files.
FROM --platform=$BUILDPLATFORM alpine:3.22 AS rootfs
RUN apk add --no-cache ca-certificates tzdata

FROM alpine:3.22
COPY --from=rootfs /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=rootfs /usr/share/zoneinfo/ /usr/share/zoneinfo/
COPY --from=server /out/plenka-server /usr/local/bin/plenka-server
ENV PORT=8095 DATA_DIR=/data
VOLUME /data
EXPOSE 8095 8095/udp
HEALTHCHECK --interval=60s --timeout=5s CMD wget -q -O /dev/null http://127.0.0.1:8095/health || exit 1
ENTRYPOINT ["/usr/local/bin/plenka-server"]
