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