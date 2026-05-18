# 2026-05-18 Pi Readyz Pass

## Scope

Add and validate `/readyz` appliance readiness signaling on the Pi runtime only.

## Commands Run

```bash
git status --short
go test ./...
sudo bash ./scripts/pi-install.sh
sudo systemctl restart g2s-mute.service
curl -fsS -i http://127.0.0.1:8444/healthz
curl -fsS -i http://127.0.0.1:8444/readyz
curl -fsS http://127.0.0.1:8444/api/status
systemctl status g2s-mute.service --no-pager --full
```

## Results

- `/readyz` implemented and wired into service routes.
- `/readyz` policy implemented:
  - `200` when readiness is `READY` or `READY_LAB`
  - `503` when readiness is `DEGRADED` or readiness cannot be computed
- Go tests passed after adding `/readyz` tests.
- Pi install passed.
- Systemd restart passed.
- Service status: `active (running)`.

## Curl Summary

- `GET /healthz`: `HTTP/1.1 200 OK`, body `ok`
- `GET /readyz`: `HTTP/1.1 200 OK`, body `{"overall":"READY_LAB","issues":[]}`
- `GET /api/status`:
  - `readiness.overall = READY_LAB`
  - `readiness.issues = []`
  - `runtime.input_mode = SIMULATED_SOFTWARE_ONLY`

## Files Changed

- `cmd/g2s-mute/main.go`
- `cmd/g2s-mute/main_test.go`
- `docs/raspberry-pi.md`
- `docs/pi-runs/2026-05-18-pi-readyz-pass.md`

## Pass/Fail

- Pass

## Notes

- No GPIO behavior was modified.
- No GPIO device paths were inspected.
- No cabinet mute/restore behavior was touched.
