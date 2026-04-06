# AGENTS.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Quick commands

### Build
- Build the main binary:
  - `go build ./cmd/document-workspace`
- Build a runnable exe into `tmp/` (matches `.air.toml`):
  - `go build -o ./tmp/app.exe ./cmd/document-workspace`

### Run
- Run in dev (runs migrations on startup, listens on `:8080`):
  - `go run ./cmd/document-workspace`

### Hot reload (Air)
This repo includes `.air.toml` configured to build `./cmd/document-workspace` into `tmp/app.exe`.
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
