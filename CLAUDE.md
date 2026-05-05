# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is XBVR

XBVR is a VR video library manager. It scrapes metadata from adult VR sites, matches local video files to that metadata, and serves content to VR players (DeoVR, HereSphere) via REST API and DLNA. This fork adds PMV (Porn Music Video) metadata matching and PMVHaven import support.

## Development Commands

**Prerequisites:** Go 1.24, Node.js 22.x, Yarn, Air (`go install github.com/cosmtrek/air@latest`)

```bash
yarn dev              # Full dev stack: Go backend (Air hot-reload) + React UI (Vite watch)
yarn build            # Production build of React UI (tsc + vite build)
yarn serve            # Run React UI dev server only (no Go backend)
go test ./...         # Run all Go tests
go test ./pkg/tasks/  # Run tests in a specific package
go build -o dist/xbvr -tags="json1" main.go  # Manual Go build (Air does this automatically)
```

**Running with a persistent database (WSL):**
```bash
go run . -app_dir /path/to/data/dir
```
Without `-app_dir`, data is stored in the default config directory. Do not use `--enableLocalStorage` for persistent workflows (it creates a temporary DB).

## Architecture

### Backend (Go)

- **Entry point:** `main.go` → `pkg/server/server.go` (HTTP + WebSocket setup)
- **REST API:** `pkg/api/` — go-restful v3 handlers for scenes, actors, files, tasks, config, DeoVR/HereSphere player integrations
- **Models:** `pkg/models/` — GORM models (Scene, Actor, File, Playlist, etc.) backed by SQLite or MySQL
- **Scrapers:** `pkg/scrape/` — 50+ site-specific scrapers using Colly/Goquery. Each file handles one site/studio
- **Tasks:** `pkg/tasks/` — background job engine: volume scanning, metadata scraping, file matching (`pmv_match.go`), heatmap/preview generation, cron scheduling
- **DLNA:** `pkg/dms/` — UPnP/DLNA media server for streaming to VR players
- **Real-time:** `pkg/session/` — WAMP protocol over WebSocket (`ws://localhost:9998`) for live task status and remote control
- **Config:** `pkg/config/` — app settings, scraper enable/disable state

### Frontend

- **`ui-new/`** — Primary UI: React 18 + TypeScript + Vite + TailwindCSS + Zustand state management
- **`ui/`** — Legacy Vue 2 UI (deprecated, still buildable via `yarn build:old-ui`)

## Key Patterns

- **Build tag:** Go builds require `-tags="json1"` for SQLite JSON support (handled by Air config)
- **CGO:** Required (`CGO_ENABLED=1`) for SQLite — set in `.air.toml`
- **Embedded assets:** `go generate` runs before dev to embed UI assets
- **Web UI:** served at `http://localhost:9999`, WebSocket at `localhost:9998`
- **API base:** all REST endpoints under `/api/`
- **Tests:** test files live alongside implementation (e.g., `pkg/tasks/pmv_match_test.go`)
