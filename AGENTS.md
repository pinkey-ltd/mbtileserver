# AGENTS.md

Guidance for AI coding agents working in this repository.

## Project overview

`mbtileserver` is a Go server for map tiles stored in [MBTiles](https://github.com/mapbox/mbtiles-spec) format (SQLite databases). It serves tiles in the XYZ/Web Mercator scheme and supports `png`, `jpg`, `webp`, and `pbf` (vector) tilesets per MBTiles spec 1.0. It is developed by Conservation Biology Institute (`github.com/consbio/mbtileserver`) and is designed to be fast and run on small cloud VMs.

In addition to tile access, it provides per-tileset TileJSON 2.1.0 endpoints, an embedded HTML map preview (Leaflet-based, `handlers/templates/map.html`), a services list endpoint, and an optional minimal ArcGIS MapServer-compatible API. UTF8 grids are no longer supported.

The module targets Go >= 1.21 (see `go.mod`, currently `go 1.26.0`).

## Technology stack

- **Language:** Go (go modules; no vendor directory)
- **Key dependencies** (`go.mod`):
  - `github.com/labstack/echo/v4` — HTTP framework
  - `github.com/brendan-ward/mbtiles-go` — MBTiles reading/SQLite access (via `crawshaw.io/sqlite`, which requires **CGO and a C compiler**)
  - `github.com/spf13/cobra` — CLI flags (most flags also configurable via env vars like `HOST`, `PORT`, `TILE_DIR`, `TLS_CERT`, `HMAC_SECRET_KEY`, etc.)
  - `github.com/sirupsen/logrus` + `logrus_sentry` — logging / Sentry
  - `github.com/fsnotify/fsnotify` — filesystem watching for tileset reloads
  - `golang.org/x/crypto` — Auto TLS (Let's Encrypt / autocert)
- **Static assets:** embedded via `//go:embed` (templates in `handlers/templates/`, static preview assets in `handlers/templates/static/`)

## Repository layout

- `main.go` — CLI entrypoint (`cobra`), flag/env parsing, server startup (HTTP/HTTPS/AutoTLS), graceful-reload supervisor (`--enable-reload-signal` forks a child that inherits the listener via file descriptor 3; SIGHUP triggers reload), Sentry hook wiring
- `watch.go` — filesystem watcher (`--enable-fs-watch`) that reloads tilesets when `.mbtiles` files change, with debouncing
- `handlers/` — the core HTTP logic:
  - `serviceset.go` — `ServiceSet` holding all registered tilesets; routes service list / TileJSON / preview / tile requests
  - `tile.go`, `tileset.go` — per-tileset HTTP handlers and tileset model
  - `arcgis.go` — optional ArcGIS MapServer endpoints
  - `middleware.go` — HMAC request-authentication middleware (`--secret-key`)
  - `id.go` — tileset ID generation (`SHA1ID`, `RelativePathID`)
  - `blankpng.go` — blank PNG fallback for missing image tiles (unless `--missing-image-tile-404`)
  - `templates.go`, `templates/` — embedded preview page
  - `*_test.go` — tests (see below)
- `testdata/` — sample `.mbtiles` files used by tests (valid PNG/JPG/WebP, missing bounds, invalid files, missing center)
- `.github/workflows/` — CI: `test.yml` (tests on Ubuntu/macOS, Go 1.21–1.23, incl. linux/arm64 cross-build with CGO), `codeql-analysis.yml`, `docker.yml` (multi-arch images to ghcr.io), `release.yml`
- `Dockerfile` — two-stage build (golang alpine → plain alpine) with musl libc symlink workaround; `docker-compose.yml` for local runs
- `CHANGELOG.md`, `README.md` — user-facing docs

## Build and test commands

CGO must be enabled and gcc/clang available (needed by `crawshaw.io/sqlite`):

```sh
go build .          # builds the ./mbtileserver binary
go vet ./...
CGO_ENABLED=1 go test -v ./...
```

CI sets `CGO_ENABLED=1` and installs gcc (including `gcc-multilib` on Ubuntu) before testing. Tests run on Ubuntu and macOS.

## Testing strategy

- Tests are plain Go tests alongside sources (`handlers/*_test.go`), using the real sample MBTiles files in `testdata/` (loaded with relative paths, so tests run from the module root / respective package dirs).
- There is no separate test framework, no generated mocks; `serviceset_test.go`-style tests construct a `ServiceSet` and issue real `httptest` requests.
- Keep tests table-driven and offline (no network) to match existing style.

## Runtime / deployment notes

- Default port 8000 (443 with TLS); default tile directory `./tilesets` (`--dir`, comma-separated list allowed).
- Deployment is via Docker images published to `ghcr.io/consbio/mbtileserver` on pushes to `main` and `v*` tags, or `go install github.com/consbio/mbtileserver@latest`.
- When `--enable-reload-signal` is used, the parent process supervises a child (`MBTS_IS_CHILD` env var); SIGHUP gracefully restarts the child while keeping the listener.

## Code style guidelines

- Standard Go formatting (`gofmt`); idiomatic Go with explicit error handling (`if err != nil`, wrap with `fmt.Errorf("...: %w", err)`).
- Logging via `logrus` (`log "github.com/sirupsen/logrus"`); comments and docs are in English.
- Keep CLI flags paired with environment-variable equivalents when adding configuration.
- Match existing handler structure: add endpoints on the `ServiceSet` and register them in `serviceset.go`.
