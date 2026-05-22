# ── Config ────────────────────────────────────────────────────────────────────
APP     := projdocs-api
CMD     := ./cmd
OUT     := ./bin/$(APP)
MODE    := debug

# ── Build flags ───────────────────────────────────────────────────────────────
# CGO_ENABLED=0  → pure Go, no libc dependency
# -trimpath      → strip local file system paths from the binary
# -ldflags:
#   -s            strip symbol table
#   -w            strip DWARF debug info
#   extldflags    tell the external linker to produce a static binary
CGO_ENABLED := 0
LDFLAGS      = -ldflags="-s -w -extldflags '-static' -X 'github.com/projdocs/api/config.Mode=$(MODE)'"
GCFLAGS     :=
BUILDFLAGS   = -trimpath $(LDFLAGS)

# ── Targets ───────────────────────────────────────────────────────────────────
.PHONY: build run clean tidy vet

build: tidy
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILDFLAGS) -o $(OUT) $(CMD)

prod: MODE = release
prod: build

run: build
	$(OUT)

clean:
	rm -rf ./bin

tidy:
	go mod tidy

vet:
	go vet ./...