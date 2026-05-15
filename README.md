# G2S Muting Controller

This repository is the working codebase for the Raspberry Pi-based G2S muting controller.

## Current Status

The repo now contains the first dependency-light Go scaffold for the MVP:

- config loading and validation
- controller runtime state model
- in-memory engine event loop
- G2S host listener for fake `commsOnLine` and `keepAlive` traffic
- SQLite migration files for the future ledger
- Raspberry Pi/systemd packaging starter

The local machine used to create this scaffold did not have `go` in `PATH`, so compile/test verification still needs to run after Go is installed.

## Quick Start

```powershell
go test ./...
go run ./cmd/g2s-mute -config .\configs\config.example.json
```

Once running, the development listener exposes:

- `GET /healthz`
- `GET /api/status`
- `POST /g2s`

Current project documentation lives in:

- `docs/G2S_Complete_Executable_Project_Plan.md`
- `docs/G2S_First_Cabinet_Lab_Runbook.md`
- `docs/G2S_Pre_NextStep_Readiness_Check.md`
- `docs/development.md`
- `docs/setup-windows.md`

Recommended first implementation target:

1. Create the Go module and package layout.
2. Build the host listener plus fake EGM `commsOnLine` flow.
3. Add config loading, SQLite, and the controller engine.

## Verification

After Go is installed:

```powershell
.\scripts\check.ps1
```
