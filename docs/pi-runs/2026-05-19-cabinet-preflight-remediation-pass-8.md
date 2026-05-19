# 2026-05-19 Cabinet Preflight Remediation (Pass 8)

## Gates
- `GATES: SUDO_OK=0 PASS_PHRASE_NEEDED=NO PASS_PHRASE_READY=N/A`
- `SUDO_DETAIL`:
  - `sudo: /etc/sudo.conf is owned by uid 65534, should be 0`
  - `sudo: The "no new privileges" flag is set, which prevents sudo from running as root.`
  - `sudo: If sudo is running in a container, you may need to adjust the container configuration to disable the flag.`
- Mode selected: `user`

## Phase 1 Baseline
Commands run:
- `go test ./...` with `GOMODCACHE=/tmp/g2s-go/pkg/mod` and `GOCACHE=/tmp/g2s-go/build` (passed).
- `curl -fsS http://127.0.0.1:8444/api/cabinet-preflight || true`
- `bash ./scripts/cabinet-preflight.sh || true`

Baseline behavior from sandbox namespace:
- local API at `127.0.0.1:8444` unreachable (`curl: (7) Failed to connect...`).
- script returned explicit FAIL with API-unavailable guidance.

## Phase 2 Runtime by Mode
`MODE=user` branch executed.
- Unsandboxed local probe confirmed API already reachable and skipped user-mode start:
  - `user-mode start skipped: API already reachable`

## Phase 3 Preflight Remediation
- Verified active cabinet profile override is non-placeholder via `GET /api/cabinet-profile`.
- Re-ran:
  - `GET /api/cabinet-preflight`
  - `bash ./scripts/cabinet-preflight.sh`
  - `go test ./...`

Result:
- `cabinet_profile`: `PASS`
- `profile_source`: `PASS` (`override`)
- remaining blocker: `certificate_san_wire_identity` with detail `web_server_cert path is empty` on active service config.

## Repo Changes in This Pass
Updated user-mode config for repeatable local runs:
- `configs/config.pi.user.json`
  - switched to non-placeholder Pi identity values (`tspi4.local`, `192.168.10.25`, `HOST-TSPI4-001`, `EGM-A100`)
  - aligned web cert paths to Phase-2 generated files:
    - `./certs/web_server_cert.pem`
    - `./certs/web_server_key.pem`

## Remaining External Dependency
Active runtime at `127.0.0.1:8444` is the host service using `/etc/g2s-mute/config.json`; without sudo, this pass cannot update root-managed `crypto.web_server_cert_path`/`crypto.web_server_key_path` there or restart service.
