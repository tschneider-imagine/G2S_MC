# Pi Run Report - Readiness Precedence Fix

Date: 2026-05-18

## Commands Run
- `go test ./...`
- `sudo bash ./scripts/pi-install.sh`
- `sudo systemctl restart g2s-mute.service`
- `curl -fsS http://127.0.0.1:8444/api/status`
- `curl -fsS -i http://127.0.0.1:8444/readyz`

## Test Result
- `go test ./...` passed.

## /readyz Result
- Response: `HTTP/1.1 200 OK`
- Body: `{"overall":"READY_LAB","issues":[]}`

## Files Changed
- `cmd/g2s-mute/main.go`
- `cmd/g2s-mute/main_test.go`
- `docs/pi-runs/2026-05-18-pi-readyz-precedence-fix.md`
