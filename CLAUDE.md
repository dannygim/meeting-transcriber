# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Meeting Transcriber — a macOS desktop app that records meeting audio via PortAudio and transcribes on-device using whisper-cpp. Built with Wails v3 (Go backend + React/TypeScript frontend).

## Build & Development Commands

```bash
# Development (hot-reload for both Go and frontend)
task dev

# Production build (creates .app bundle in bin/)
task build && task package

# Frontend only
cd frontend && pnpm install && pnpm run build

# Regenerate TypeScript bindings after changing Go service methods
wails3 generate bindings

# Go module maintenance
go mod tidy
```

System dependencies: `brew install portaudio whisper-cpp`

## Architecture

### Wails v3 Service Pattern

Go structs registered as Wails Services expose their exported methods to the frontend via auto-generated TypeScript bindings. Services must implement the correct lifecycle signatures:

```go
// CORRECT — Wails v3 requires these exact signatures
func (s *MyService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error
func (s *MyService) ServiceShutdown() error
func (s *MyService) ServiceName() string
```

A zero-parameter `ServiceStartup() error` compiles but **silently never gets called** — this was a past bug.

### Backend Services (`services/`)

**AudioService** (`audio.go`): PortAudio recording at device native sample rate (typically 48kHz), downsampled to 16kHz WAV for whisper-cpp. Maintains a ring buffer (`specBuf`) for real-time spectrum data. Thread-safe via `sync.Mutex`.

**TranscribeService** (`transcriber.go`): Invokes `whisper-cpp` CLI as subprocess. Searches Homebrew paths directly (`/opt/homebrew/bin/`, `/usr/local/bin/`) because macOS GUI apps don't inherit shell PATH.

### Frontend (`frontend/src/`)

Hooks (`hooks/`) own state and call Go bindings. Components (`components/`) are pure presentational. Bindings are auto-generated in `frontend/bindings/` — don't edit them manually.

The binding import path mirrors the Go module path:
```typescript
import { AudioService } from '../../bindings/github.com/dannygim/meeting-transcriber/services'
```

### Build System

Task runner (`Taskfile.yml`) orchestrates everything. The root Taskfile includes `build/Taskfile.yml` (common tasks) and `build/darwin/Taskfile.yml` (macOS-specific). Frontend uses pnpm (not npm). `build/config.yml` holds Wails v3 app metadata.

### macOS Permissions

Both `build/darwin/Info.plist` (production) and `Info.dev.plist` (development) must include `NSMicrophoneUsageDescription`. Running via `go run` requires the terminal app itself to have microphone permission in System Preferences; the `.app` bundle gets its own permission dialog.

## Key Gotchas

- **Sample rate**: PortAudio must open at the device's native rate (usually 48kHz). Opening at 16kHz returns all-zero samples on macOS. The service downsamples to 16kHz when writing the WAV.
- **Bindings regeneration**: After changing any Go service method signature, run `wails3 generate bindings` and rebuild frontend.
- **pnpm required**: The frontend uses pnpm (lockfile is `pnpm-lock.yaml`). npm will fail due to workspace protocol.
- **CGO required**: PortAudio binding needs `CGO_ENABLED=1`. This is set automatically by the Taskfile.
