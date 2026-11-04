SHELL := bash

VERSION := $(shell sed -n 's/^## \[\([0-9][0-9.]*\)\].*/\1/p' CHANGELOG.md | head -1)
# Android versionCode: 1.2.3 -> 10203, so every release installs over the previous one.
BUILD_NUMBER := $(shell echo $(VERSION) | awk -F. '{print $$1*10000 + $$2*100 + $$3}')
DIST := dist

ALL_ARCHS := amd64 arm64 armv7
UNAME_M   := $(shell uname -m)
HOST_ARCH := $(if $(filter x86_64 amd64,$(UNAME_M)),amd64,$(if $(filter aarch64 arm64,$(UNAME_M)),arm64,$(if $(filter armv7%,$(UNAME_M)),armv7)))

empty :=
space := $(empty) $(empty)
comma := ,

# ARCH: "all", or one or more of $(ALL_ARCHS) separated by commas or spaces; empty = host.
ARCHS = $(if $(filter all,$(ARCH)),$(ALL_ARCHS),$(or $(strip $(subst $(comma),$(space),$(ARCH))),$(HOST_ARCH)))
check_arch = $(if $(filter-out $(ALL_ARCHS),$(ARCHS)),$(error unknown ARCH "$(filter-out $(ALL_ARCHS),$(ARCHS))", use: $(ALL_ARCHS) or all))
join_comma = $(subst $(space),$(comma),$(strip $(1)))

# Architecture names in each toolchain.
goarch     = GOARCH=$(if $(filter armv7,$(1)),arm GOARM=7,$(1))
platform   = linux/$(if $(filter armv7,$(1)),arm/v7,$(1))
android    = android-$(if $(filter amd64,$(1)),x64,$(if $(filter armv7,$(1)),arm,arm64))
apk_abi    = $(if $(filter amd64,$(1)),x86_64,$(if $(filter armv7,$(1)),armeabi-v7a,arm64-v8a))
flutter_os = $(if $(filter amd64,$(1)),x64,$(1))

.PHONY: backend frontend web test build build-server build-image build-push build-apk build-linux

# Run the backend (Go) on :8095.
backend:
	$(MAKE) -C backend development

# Run the frontend dev server (Vite) on :5173, proxying the API to :8095.
frontend:
	cd frontend && pnpm dev

# Build the frontend and embed it into the backend.
web:
	$(MAKE) -C backend web

test:
	$(MAKE) -C backend test
	cd frontend && pnpm test

# Interactive release builder: asks what to build and for which architectures, then runs
# the matching build-* target. Skip the questions with WHAT=... and ARCH=..., for example
#   make build WHAT=image ARCH=arm64
#   make build WHAT=push ARCH=all IMAGE=ghcr.io/<user>/plenka-server
build:
	@what='$(WHAT)'; arch='$(ARCH)'; \
	if [[ -z $$what ]]; then \
	  echo "What to build? (Plenka $(VERSION))"; \
	  echo "  1) server  plenka-server binary with the web app  -> $(DIST)/plenka-server-linux-<arch>"; \
	  echo "  2) image   Docker image saved as a tar (MikroTik, docker load)  -> $(DIST)/plenka-server-<arch>.tar"; \
	  echo "  3) push    multi-arch Docker image pushed to a registry"; \
	  echo "  4) apk     Android / Android TV app  -> $(DIST)/plenka-$(VERSION)-android-<arch>.apk"; \
	  echo "  5) linux   Linux desktop app, host architecture only  -> $(DIST)/plenka-$(VERSION)-linux-<arch>.tar.gz"; \
	  read -rp '> ' what; \
	fi; \
	case $$what in \
	  1|server) what=server;; 2|image) what=image;; 3|push) what=push;; \
	  4|apk) what=apk;; 5|linux) what=linux;; \
	  *) echo "unknown choice: '$$what'" >&2; exit 1;; \
	esac; \
	if [[ $$what != linux && -z $$arch ]]; then \
	  echo "Architecture? (several: separate with spaces or commas; Enter = $(HOST_ARCH))"; \
	  echo "  1) amd64   x86-64: PC, server, MikroTik CHR, Android emulator"; \
	  echo "  2) arm64   MikroTik hAP ax2/ax3, RB5009, Raspberry Pi 4/5, most phones and TV boxes"; \
	  echo "  3) armv7   32-bit ARM: older MikroTik models and TV boxes"; \
	  echo "  4) all"; \
	  read -rp '> ' arch; \
	fi; \
	list=; \
	for a in $${arch//,/ }; do \
	  case $$a in \
	    1|amd64) list+=" amd64";; 2|arm64) list+=" arm64";; 3|armv7) list+=" armv7";; \
	    4|all) list=" $(ALL_ARCHS)"; break;; \
	    *) echo "unknown architecture: '$$a'" >&2; exit 1;; \
	  esac; \
	done; \
	$(MAKE) --no-print-directory build-$$what ARCH="$${list# }"

# Static server binaries (web app embedded), cross-compiled by Go.
build-server: web
	$(check_arch)
	@mkdir -p $(DIST)
	@$(foreach a,$(ARCHS),echo ">> $(DIST)/plenka-server-linux-$(a)" && \
	  GOOS=linux $(call goarch,$(a)) $(MAKE) --no-print-directory -C backend build \
	    VERSION=$(VERSION) BIN=$(CURDIR)/$(DIST)/plenka-server-linux-$(a) &&) true
	@ls -lh $(DIST)/plenka-server-linux-*

# One Docker image per architecture, saved as a tar: RouterOS imports a single-arch image.
# All build stages run on the host and Go cross-compiles, so no qemu is needed.
build-image:
	$(check_arch)
	@mkdir -p $(DIST)
	@$(foreach a,$(ARCHS),echo ">> plenka-server:$(a) $(VERSION) for $(call platform,$(a))" && \
	  docker buildx build --platform $(call platform,$(a)) --build-arg VERSION=$(VERSION) \
	    -t plenka-server:$(a) --load . && \
	  docker save plenka-server:$(a) -o $(DIST)/plenka-server-$(a).tar &&) true
	@ls -lh $(DIST)/plenka-server-*.tar

# One multi-arch tag (:VERSION and :latest) pushed to a registry.
build-push:
	$(check_arch)
	@image='$(IMAGE)'; \
	[[ -n $$image ]] || read -rp 'Image name (e.g. ghcr.io/<user>/plenka-server): ' image; \
	[[ -n $$image ]] || { echo 'an image name is required' >&2; exit 1; }; \
	docker buildx inspect plenka-multiarch >/dev/null 2>&1 \
	  || docker buildx create --name plenka-multiarch --driver docker-container >/dev/null; \
	echo ">> $$image:$(VERSION) for $(call join_comma,$(foreach a,$(ARCHS),$(call platform,$(a))))"; \
	docker buildx build --builder plenka-multiarch \
	  --platform $(call join_comma,$(foreach a,$(ARCHS),$(call platform,$(a)))) \
	  --build-arg VERSION=$(VERSION) -t "$$image:$(VERSION)" -t "$$image:latest" --push .

# Android APKs, one per architecture.
build-apk:
	$(check_arch)
	@mkdir -p $(DIST)
	cd app && flutter build apk --release --split-per-abi \
	  --target-platform $(call join_comma,$(foreach a,$(ARCHS),$(call android,$(a)))) \
	  --build-name=$(VERSION) --build-number=$(BUILD_NUMBER)
	@$(foreach a,$(ARCHS),cp app/build/app/outputs/flutter-apk/app-$(call apk_abi,$(a))-release.apk \
	  $(DIST)/plenka-$(VERSION)-android-$(a).apk &&) true
	@ls -lh $(DIST)/plenka-$(VERSION)-android-*.apk

# Linux desktop app. Flutter does not cross-compile Linux builds, so only the host arch.
build-linux:
	$(if $(filter-out $(HOST_ARCH),$(ARCHS)),$(error the Linux app builds only for the host architecture ($(HOST_ARCH))))
	@mkdir -p $(DIST)
	cd app && flutter build linux --release --build-name=$(VERSION) --build-number=$(BUILD_NUMBER)
	tar -C app/build/linux/$(call flutter_os,$(HOST_ARCH))/release -czf $(DIST)/plenka-$(VERSION)-linux-$(HOST_ARCH).tar.gz \
	  --transform 's,^bundle,plenka,' bundle
	@ls -lh $(DIST)/plenka-$(VERSION)-linux-$(HOST_ARCH).tar.gz
