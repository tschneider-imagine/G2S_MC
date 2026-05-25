# G2S Muting Controller

This repository is the working codebase for the Raspberry Pi-based G2S muting controller.

## Current Status

The repo now contains the first Go MVP scaffold:

- config loading and validation
- controller runtime state model
- in-memory engine event loop
- G2S host listener for fake `commsOnLine` and `keepAlive` traffic
- fake EGM command for exercising the host listener
- development CA/server/client certificate generator
- HTTPS and mutual TLS smoke-test path for fake EGM traffic
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
go run ./cmd/g2s-fake-egm -host-url http://127.0.0.1:8444/g2s -egm-id EGM-01
```

For the TLS-shaped local flow:

```powershell
go run ./cmd/g2s-dev-certs -out .\certs
go run ./cmd/g2s-mute -config .\configs\config.tls.example.json
```

In a second terminal:

```powershell
go run ./cmd/g2s-fake-egm -host-url https://localhost:8444/g2s -egm-id EGM-01 -ca .\certs\ca.crt -cert .\certs\client.crt -key .\certs\client.key
```

Once running, the development listener exposes:

- `GET /`
- `GET /dashboard`
- `GET /healthz`
- `GET /api/status`
- `GET /api/incidents`
- `GET /api/egms/history`
- `GET /api/compliance`
- `GET /api/state-history`
- `GET /api/certificates`
- `POST /g2s`

Active planning entry point:

- `docs/project-definition/README.md`

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
