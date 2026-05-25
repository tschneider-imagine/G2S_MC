# G2S Muting Controller

This repository is the working codebase for the Raspberry Pi-based G2S muting controller.

## Current Status

The repo now contains the first Go MVP scaffold:

- config loading and validation
- controller runtime state model
- in-memory engine event loop
- G2S host listener for development traffic
- development CA/server/client certificate generator
- HTTPS and mutual TLS development path
- SQLite-backed audit store for incidents, EGM status snapshots, compliance logs, and state history
- Raspberry Pi/systemd packaging starter

The scaffold verifies with `.\scripts\check.ps1`.

## Quick Start

```powershell
go test ./...
go run ./cmd/g2s-mute -config .\configs\config.example.json
```

In a second terminal, simulate an EGM starting a G2S session:

```powershell
go run ./cmd/g2s-capture-sink
```

For the TLS-shaped local flow:

```powershell
go run ./cmd/g2s-dev-certs -out .\certs
go run ./cmd/g2s-mute -config .\configs\config.tls.example.json
```

In a second terminal:

```powershell
go run ./cmd/g2s-gpio-probe
```

Once running, the development listener exposes:

- `GET /` (redirects to `/operator`)
- `GET /operator`
- `GET /healthz`
- `GET /api/status`
- `GET /api/incidents`
- `GET /api/egms/history`
- `GET /api/compliance`
- `GET /api/state-history`
- `GET /api/certificates`
- `POST /g2s`

Current documentation:

- `docs/project-definition/G2S_MC_REBUILD_PROJECT_DEFINITION_AND_GUARDRAILS.md`
- `docs/project-definition/GLOBAL_CHAT_PROMPT_ADDITIONS.md`
- `docs/development.md`
- `docs/raspberry-pi.md`
- `docs/setup-windows.md`

Archived historical planning docs:

- `docs/archive/old-plans/`

## Verification

After Go is installed:

```powershell
.\scripts\check.ps1
```
