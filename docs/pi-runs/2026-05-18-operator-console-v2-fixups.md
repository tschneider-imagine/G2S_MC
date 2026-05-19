# Pi Run Report - Operator Console v2 Fixups

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

## Pass/Fail
- `go test ./...`: PASS
- `pi-install.sh`: PASS
- `systemctl restart g2s-mute.service`: PASS
- `/healthz`: PASS (`ok`)
- `/readyz`: PASS (`HTTP/1.1 200 OK`, `{"overall":"READY_LAB","issues":[]}`)
- `/api/status`: PASS (`state=HEALTHY`, `readiness.overall=READY_LAB`)
- `/dashboard`: PASS
- `systemctl status`: PASS (`active (running)`)

## Files Changed
- `internal/ui/assets_operator_v2.go`
- `internal/ui/server_test.go`
- `docs/pi-runs/2026-05-18-operator-console-v2-fixups.md`

## Fix Summary
1. Added synthetic degraded `/readyz` fallback when `/readyz` fetch fails (`overall=DEGRADED`, `issues=["readyz unavailable"]`) and ensured degraded visual state is applied.
2. Added explicit API failure banner (`api-failure-banner`) that appears when any API call fails in a poll and hides on fully successful polls.
3. Replaced non-ASCII sort arrows with ASCII-safe sort labels: `EGM (asc/desc)`, `Status (asc/desc)`, and `Last seen (asc/desc)`.
