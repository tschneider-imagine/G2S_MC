# 2026-05-18 Pi Appliance Web Pass

## Run Metadata

- Date/time: 2026-05-18T11:43:45-06:00
- Pi hostname: TSPI4
- Git commit before changes: 04d4efa
- Git commit after verification before commit: 04d4efa with local working tree changes pending
- Preserved stash: `stash@{0}: On main: pi-local-before-origin-main-2026-05-18`

## Commands Run

```bash
git status --short
git stash list --max-count=3
git rev-parse --short HEAD
hostname
date -Is
go test ./...
sed -n '1,260p' cmd/g2s-mute/main.go
find internal/ui -maxdepth 3 -type f -print
sed -n '1,320p' internal/ui/server.go
sed -n '1,360p' internal/ui/assets.go
sed -n '1,260p' internal/ui/server_test.go
sed -n '1,220p' packaging/systemd/g2s-mute.service
sed -n '1,320p' docs/raspberry-pi.md
grep -R "HandleFunc\|/api/\|healthz\|status\|certificates" -n cmd internal packaging docs
sudo bash ./scripts/pi-install.sh
sudo systemctl restart g2s-mute.service
systemctl status g2s-mute.service --no-pager --full
curl -fsS http://127.0.0.1:8444/healthz
curl -fsS http://127.0.0.1:8444/api/status
curl -fsS http://127.0.0.1:8444/api/certificates
curl -fsS http://127.0.0.1:8444/dashboard
journalctl -u g2s-mute.service -n 80 --no-pager
gofmt -w cmd/g2s-mute/main.go cmd/g2s-mute/main_test.go internal/ui/server_test.go
sudo systemctl stop g2s-mute.service
sudo CONFIG_PATH=/etc/g2s-mute/config.json bash ./scripts/pi-smoke.sh
sudo systemctl restart g2s-mute.service
git diff --stat
```

## Pass/Fail Results

- `go test ./...`: pass
- `sudo bash ./scripts/pi-install.sh`: pass, existing `/etc/g2s-mute/config.json` preserved
- `sudo systemctl restart g2s-mute.service`: pass
- `curl -fsS http://127.0.0.1:8444/healthz`: pass, returned `ok`
- `curl -fsS http://127.0.0.1:8444/api/status`: pass
- `curl -fsS http://127.0.0.1:8444/api/certificates`: pass
- `curl -fsS http://127.0.0.1:8444/dashboard`: pass, returned dashboard HTML
- `sudo CONFIG_PATH=/etc/g2s-mute/config.json bash ./scripts/pi-smoke.sh`: pass
  - `commsOnLine -> HTTP 200`
  - `keepAlive 1 -> HTTP 200`
  - `Pi smoke test passed`

## Service Status Summary

Final systemd status:

```text
Loaded: loaded (/etc/systemd/system/g2s-mute.service; enabled; preset: enabled)
Active: active (running)
ExecStart: /usr/local/bin/g2s-mute -config /etc/g2s-mute/config.json
```

Useful final journal lines:

```text
loaded config controller_id=G2S-MC-PI-001 site="Pi Lab" checksum=05bb3ba7b20fadd0581b0abe128e7ab88904d33ced3ccfcb45dacf18747dd58e
runtime config_path=/etc/g2s-mute/config.json database_path=/var/lib/g2s-mute/controller.db bind_address=0.0.0.0:8444 dashboard_path=/dashboard g2s_host_url=http://127.0.0.1:8444/g2s g2s_endpoint_path=/g2s egm_count=1
security mode tls_required=false client_cert_required=false web_login_required=false admin_client_cert_required=false simulated_trigger=false
certificate inventory ok=0 expiring_soon=0 missing=2 not_configured=1 invalid=0
service ready protocol=http bind_address=0.0.0.0:8444 health=/healthz status=/api/status dashboard=/dashboard
```

No important journal warnings or errors were observed during this pass.

## Curl Results Summary

- `/healthz`: `ok`
- `/api/status`:
  - `controller_id`: `G2S-MC-PI-001`
  - `state`: `HEALTHY`
  - `runtime.input_mode`: `SIMULATED_SOFTWARE_ONLY`
  - `runtime.database_path`: `/var/lib/g2s-mute/controller.db`
  - `runtime.bind_address`: `0.0.0.0:8444`
  - `readiness.overall`: `READY_LAB`
  - `readiness.egm_count`: `1`
  - `readiness.certificate_summary`: `MISSING=2`, `NOT_CONFIGURED=1`
- `/api/certificates`:
  - `g2s_ca_cert`: `MISSING`
  - `g2s_client_cert`: `MISSING`
  - `web_server_cert`: `NOT_CONFIGURED`

The missing certs are expected in the default Pi lab config because TLS and client certificate enforcement are disabled.

## Dashboard/API Improvements Made

- `/api/status` now includes runtime metadata:
  - start time and uptime
  - config path
  - database path
  - bind address
  - dashboard and health paths
  - G2S host URL and endpoint path
  - TLS/client-cert/login mode flags
  - explicit `SIMULATED_SOFTWARE_ONLY` input mode
- `/api/status` now includes readiness metadata:
  - overall readiness, currently `READY_LAB`
  - readiness warnings
  - EGM count
  - certificate status summary
- Dashboard now shows:
  - controller state
  - appliance readiness
  - simulated input mode
  - service bind/database/G2S/TLS details
  - certificate summary
  - existing EGM roster, incidents, EGM history, state history, and certificate inventory
- Startup logs now show config, database, bind address, TLS/client cert mode, certificate summary, and service-ready paths.

## Files Changed

- `cmd/g2s-mute/main.go`
- `cmd/g2s-mute/main_test.go`
- `docs/raspberry-pi.md`
- `docs/pi-runs/2026-05-18-pi-appliance-web-pass.md`
- `internal/ui/assets.go`
- `internal/ui/server_test.go`

## Final Git Status

```text
 M cmd/g2s-mute/main.go
 M docs/raspberry-pi.md
 M internal/ui/assets.go
 M internal/ui/server_test.go
?? cmd/g2s-mute/main_test.go
?? docs/pi-runs/
```

## Remaining Risks And Next Steps

- The service is running in HTTP lab mode. Configure TLS lab certificates before any real cabinet-facing test.
- `/etc/g2s-mute/certs` is currently empty, so certificate inventory reports missing G2S CA/client certs as expected.
- Dashboard JavaScript is intentionally simple and should be hardened before exposing beyond the local lab network.
- Next recommended step: add an explicit `/readyz` endpoint or status check script if systemd watchdog/readiness integration is desired.
- Next recommended step: add TLS lab verification with `g2s-dev-certs` and curl checks against HTTPS.
- No GPIO paths were inspected or accessed.
- No real GPIO behavior, cabinet mute behavior, or cabinet restore behavior was modified or exercised.
