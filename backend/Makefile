VERSION ?= $(shell sed -n 's/^## \[\([0-9][0-9.]*\)\].*/\1/p' ../CHANGELOG.md | head -1)
LDFLAGS := -s -w -checklinkname=0 -X plenka/internal/buildinfo.Version=$(VERSION)
BIN     := bin/plenka-server

.PHONY: development build web test vet tidy clean

# Run the server locally (data in ./data, port 8095). Serves whatever is in web/dist.
development:
	DATA_DIR=./data go run -ldflags "$(LDFLAGS)" .

# Build the frontend and copy it into web/dist so it is embedded in the binary.
web:
	cd ../frontend && pnpm install --frozen-lockfile && pnpm build
	find web/dist -mindepth 1 ! -name .gitkeep -delete
	cp -r ../frontend/dist/. web/dist/

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) .

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin
