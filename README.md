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

Current project documentation lives in:

- `docs/G2S_Complete_Executable_Project_Plan.md`
- `docs/G2S_First_Cabinet_Lab_Runbook.md`
- `docs/G2S_Pre_NextStep_Readiness_Check.md`
- `docs/development.md`
- `docs/raspberry-pi.md`
- `docs/setup-windows.md`

Recommended first implementation target:

1. Run the Pi installer and smoke test on the target Pi.
2. Add Raspberry Pi GPIO adapters once the pinout is confirmed.
3. Continue real-cabinet readiness around host registration, payload mapping, and certificate source.

## Verification

After Go is installed:

```powershell
.\scripts\check.ps1
```
