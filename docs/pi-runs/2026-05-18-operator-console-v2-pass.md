# Pi Run Report - Operator Console v2

Date: 2026-05-18

## Commands Run
- `go test ./...`
- `sudo bash ./scripts/pi-install.sh`
- `sudo systemctl restart g2s-mute.service`
- `curl -fsS http://127.0.0.1:8444/healthz`
- `curl -fsS -i http://127.0.0.1:8444/readyz`
- `curl -fsS http://127.0.0.1:8444/api/status`
- `curl -fsS http://127.0.0.1:8444/dashboard >/dev/null`
- `systemctl status g2s-mute.service --no-pager --full`

## Results
- `go test ./...` passed for all packages.
- `/healthz` returned `ok`.
- `/readyz` returned `HTTP/1.1 200 OK` with `{"overall":"READY_LAB","issues":[]}`.
- `/api/status` returned `state=HEALTHY`, `readiness.overall=READY_LAB`, and readiness warnings for lab mode.
- `/dashboard` loaded successfully.
- `g2s-mute.service` was `active (running)` after restart.

## What Changed
- Operator Console v2 dashboard implementation:
  - resilient polling state model with overlap prevention and backoff (`3s -> 6s -> 10s`)
  - stale data badge (`>10s warning`, `>30s critical`)
  - prioritized operator alert strip with explicit `/readyz` primary status
  - EGM table sorting/filtering and last-seen age display
  - certificate severity grouping and short state explanations, including lab-expected `NOT_CONFIGURED`
- UI test coverage updated for new dashboard markers and asset behavior markers.
- Raspberry Pi runbook updated with operator console checks.

## Files Changed
- `internal/ui/assets.go` (legacy constants renamed)
- `internal/ui/assets_operator_v2.go`
- `internal/ui/server_test.go`
- `docs/raspberry-pi.md`
- `docs/pi-runs/2026-05-18-operator-console-v2-pass.md`

## Remaining Risks
- Client-side rendering/parsing is string-template based and can become harder to maintain as dashboard complexity grows.
- Readiness/alert behavior depends on successful polling of multiple endpoints; partial endpoint outages will show cached data with alert/stale indicators, which is intentional but operationally important to understand.
