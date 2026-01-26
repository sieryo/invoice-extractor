# AGENTS.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Quick commands

### Build
- Build the main binary:
  - `go build ./cmd/invoice-extractor`
- Build a runnable exe into `tmp/` (matches `.air.toml`):
  - `go build -o ./tmp/app.exe ./cmd/invoice-extractor`

### Run
- Run in dev (runs migrations on startup, listens on `:8080`):
  - `go run ./cmd/invoice-extractor`

### Hot reload (Air)
This repo includes `.air.toml` configured to build `./cmd/invoice-extractor` into `tmp/app.exe`.
- Run:
  - `air`

### Tests
- Run all tests:
  - `go test ./...`
- Run a single package:
  - `go test ./internal/app/auth`
- Run a single test (example):
  - `go test ./internal/app/auth -run TestLogin`

### Basic lint / checks (built-in Go tooling)
- Format:
  - `gofmt -w .`
- Vet:
  - `go vet ./...`

## Application overview (big picture)

### Entry point and runtime locations
- Program entry is `cmd/invoice-extractor/main.go`.
- On startup it computes an app directory under `os.UserConfigDir()/invoice-extractor` and uses it for:
  - SQLite DB: `app.db`
  - Log file: `app.log`
  - File storage: `<appDir>/storage` (see `internal/app/app.go` and `internal/infra/filestore/local.go`)
- Migrations are applied automatically at boot from the repo’s `migrations/` directory.

### HTTP API layer
- HTTP server: `internal/transport/http` (Fiber).
- Routes are registered in `internal/transport/http/route.go`.
- Current primary endpoint:
  - `POST /api/extract_invoice`
    - Handler: `internal/transport/http/handler/invoice_extract_handler.go`
    - Expects multipart form field `files` (multiple uploads).
    - Saves uploads via `FileStore` under `storage/jobs/<jobID>/input/...`.
    - Creates a background job of type `INVOICE_EXTRACT`.

### App container (wiring)
- Dependency wiring happens in `internal/app/app.go`.
- Key responsibilities:
  - Construct repositories (SQLite)
  - Construct domain services (auth, jobs, invoice extraction)
  - Create a dispatcher and register job handlers
  - Start a worker pool (`JobQueueRunner`) for async jobs

### Job system
- Domain service and interfaces: `internal/app/job/*`.
- In-memory queue + worker pool: `internal/infra/jobrunner/*`.
  - `Dispatcher` routes jobs by `job.Type` to a `job.JobHandler`.
  - `JobQueueRunner` enqueues jobs and executes them on worker goroutines.
- Persistence: `internal/infra/persistence/sqlite/*`.
  - Jobs are persisted to the `jobs` table (see `migrations/001_init.sql`).

### Invoice extraction flow
- Job handler: `internal/app/invoice/extract/job.go`.
  - Reads input payload JSON (currently `{ "pdf_paths": [...] }`).
  - Calls the extraction service.
  - Currently writes a stub/dummy output payload after extraction.
- Extraction service: `internal/app/invoice/extract/service.go`.
  - Concurrently processes PDFs and calls the PDF adapter.
- PDF adapter: `internal/infra/adapter/pdftool/*`.
  - Uses `pdftotext` via `exec.CommandContext`.

### `pdftotext` dependency (important for running extraction)
`internal/infra/adapter/pdftool/resolver.go` resolves `pdftotext` in this order:
1. `bin/pdftotext.exe` next to the built executable (for “bundled/production” layout)
2. `pdftotext` available on `PATH`

Notes:
- This repo includes Poppler binaries under `tools/poppler/bin` (including `pdftotext.exe`), but the resolver does not look there automatically.
- For local dev, either add `tools/poppler/bin` to your `PATH`, or copy/link `pdftotext.exe` into `bin/` next to the app executable when testing the “bundled” resolution path.

## Where to look when changing behavior
- Add/modify HTTP endpoints: `internal/transport/http/route.go` + `internal/transport/http/handler/*`.
- Add a new async job type:
  - Define handler implementing `job.JobHandler`
  - Register it in `internal/app/app.go` via `dispatcher.Register("TYPE", handler)`
- Change DB schema: add a new `migrations/*.sql` file (files are applied in sorted filename order).
