# ── Config ────────────────────────────────────────────────────────────────────
APP     := projdocs-api
CMD     := ./cmd
OUT     := ./bin/$(APP)
MODE    := debug
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOOS := darwin
GOARCH := arm64
EXT := $(if $(filter windows,$(GOOS)),.exe,)
LDFLAGS      = -ldflags="-s -w -extldflags '-static' -X 'github.com/projdocs/api/config.Mode=$(MODE)' -X 'github.com/projdocs/api/config.Version=$(VERSION)'"
BUILDFLAGS   = -trimpath $(LDFLAGS)

# ── Targets ───────────────────────────────────────────────────────────────────
.PHONY: build run clean tidy vet

build: tidy
	CGO_ENABLED=0 go build $(BUILDFLAGS) -o $(OUT) $(CMD)

prod: MODE = release
prod: build
	mkdir -p ./dist
	cp $(BINARY) ./dist/$(APP)-$(VERSION)-$(GOOS)-$(GOARCH)$(EXT)

run: build
	$(OUT)

clean:
	rm -rf ./bin

tidy:
	go mod tidy

vet:
	go vet ./...