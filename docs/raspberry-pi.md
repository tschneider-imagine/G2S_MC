# Raspberry Pi Bring-Up

This guide assumes the target Pi already has Git, Go, and Codex available.

## Install

From the repo on the Pi:

```bash
sudo bash ./scripts/pi-install.sh
```

To install and immediately start the systemd service:

```bash
sudo bash ./scripts/pi-install.sh --start
```

The installer builds the Go binaries, installs the service, creates the service user, and creates these paths:

- `/usr/local/bin/g2s-mute`
- `/usr/local/bin/g2s-fake-egm`
- `/usr/local/bin/g2s-dev-certs`
- `/etc/g2s-mute/config.json`
- `/etc/g2s-mute/certs`
- `/var/lib/g2s-mute`
- `/var/log/g2s-mute`

## First Local Smoke Test

Stop the service if it is already running, then run the foreground smoke test:

```bash
sudo systemctl stop g2s-mute.service
sudo CONFIG_PATH=/etc/g2s-mute/config.json bash ./scripts/pi-smoke.sh
```

Expected result:

```text
commsOnLine -> HTTP 200
keepAlive 1 -> HTTP 200
Pi smoke test passed
```

## Multi-Fake-EGM Lab Run

To drive several fake EGMs from the configured roster at once:

```bash
CONFIG_PATH=./configs/config.pi.multi-fake.example.json KEEPALIVE_COUNT=-1 bash ./scripts/multi-fake-egm.sh
```

Useful overrides:

```bash
HOST_URL=http://127.0.0.1:8444/g2s \
EGM_IDS=EGM-01,EGM-03 \
KEEPALIVE_INTERVAL=3s \
bash ./scripts/multi-fake-egm.sh
```

Notes:

- `KEEPALIVE_COUNT=-1` keeps each fake EGM sending keepAlive traffic until `Ctrl+C`
- `EGM_IDS` limits the run to a subset of the configured roster
- `CA_PATH`, `CERT_PATH`, and `KEY_PATH` can be set for TLS/mTLS lab runs

## Service Commands

```bash
sudo systemctl restart g2s-mute.service
systemctl status g2s-mute.service --no-pager --full
journalctl -u g2s-mute.service -n 80 --no-pager
```

The dashboard should be available at:

```text
http://<pi-ip>:8444/dashboard
```

## Appliance Verification

After installation, restart the service and verify the local operator surface:

```bash
sudo bash ./scripts/pi-install.sh
sudo systemctl restart g2s-mute.service
systemctl status g2s-mute.service --no-pager --full
curl -fsS http://127.0.0.1:8444/healthz
curl -fsS -i http://127.0.0.1:8444/readyz
curl -fsS http://127.0.0.1:8444/api/status
curl -fsS http://127.0.0.1:8444/api/certificates
journalctl -u g2s-mute.service -n 80 --no-pager
```

Expected key lines:

```text
Active: active (running)
ok
HTTP/1.1 200 OK
"overall":"READY_LAB"
"controller_id":"G2S-MC-PI-001"
"overall":"READY_LAB"
"input_mode":"SIMULATED_SOFTWARE_ONLY"
"certificate_summary"
service ready protocol=http bind_address=0.0.0.0:8444 health=/healthz ready=/readyz status=/api/status dashboard=/dashboard
```

Operator Console v2 checks on `/dashboard`:

- top alert strip is visible and reflects readiness/incident/EGM/certificate priority
- `/readyz` appears as the primary readiness badge and should match `curl -i /readyz`
- stale data badge starts as fresh and moves to warning after 10s, critical after 30s without a successful poll
- EGM table supports `All` / `Healthy` / `Unhealthy` filters and client-side sorting on EGM, status, and last seen
- certificate inventory rows show blocking vs non-blocking state explanations; `NOT_CONFIGURED` is shown as lab-expected

In the default Pi lab config, `/api/certificates` reports the local certificate paths as `MISSING` or `NOT_CONFIGURED` until TLS lab mode is configured. That is expected while `g2s.require_tls` and `g2s.require_client_cert` are false.

`/readyz` readiness policy:

- returns `200` when readiness is `READY` or `READY_LAB`
- returns `503` when readiness is `DEGRADED` or readiness cannot be computed

## TLS Lab Mode

For a local certificate test on the Pi:

```bash
sudo g2s-dev-certs -out /etc/g2s-mute/certs
sudo chown -R g2s-mute:g2s-mute /etc/g2s-mute/certs
```

Then edit `/etc/g2s-mute/config.json` to use the certificate paths from `configs/config.tls.example.json`, update `g2s.host_url` to the real Pi DNS name or IP, and ensure that value exists in the host certificate SAN before expecting real cabinet TLS to work.

## Certificate Import/Export API (Lab-Only)

Certificate material can now be managed through local-only endpoints:

- `POST /api/certificates/import`
- `GET /api/certificates/export?role=<role>&include_key=<true|false>`

Supported roles:

- `g2s_ca_cert` (certificate only)
- `g2s_client_cert` (certificate + key)
- `web_server_cert` (certificate + key)

Security limits:

- Import/export endpoints only allow loopback callers (`127.0.0.1` or `::1`).
- `include_key=true` is blocked by default.
- Private-key export requires `web_ui.allow_private_key_export=true` in `/etc/g2s-mute/config.json`.
- When `api.auth_token` is set, mutating operations require `Authorization: Bearer <token>`.

Example import (client cert/key):

```bash
CERT_JSON="$(sed ':a;N;$!ba;s/\n/\\n/g' /etc/g2s-mute/certs/client.crt)"
KEY_JSON="$(sed ':a;N;$!ba;s/\n/\\n/g' /etc/g2s-mute/certs/client.key)"
printf '{"role":"g2s_client_cert","certificate_pem":"%s","private_key_pem":"%s"}' "$CERT_JSON" "$KEY_JSON" >/tmp/g2s-cert-import.json
curl -fsS -X POST http://127.0.0.1:8444/api/certificates/import \
  -H 'Content-Type: application/json' \
  --data-binary @/tmp/g2s-cert-import.json
```

Example export (certificate only):

```bash
curl -fsS 'http://127.0.0.1:8444/api/certificates/export?role=g2s_client_cert'
```

## API Auth Token For Mutating Endpoints

To keep lab mode simple, API auth is optional. If `api.auth_token` is empty or omitted, mutating endpoints keep existing behavior.
If `api.auth_token` is set, callers must present `Authorization: Bearer <token>` for:

- `PUT /api/cabinet-profile`
- `DELETE /api/cabinet-profile`
- `POST /api/certificates/import`
- `GET /api/certificates/export?...&include_key=true`

Config example:

```json
{
  "api": {
    "auth_token": "replace-with-long-random-token"
  }
}
```

Authenticated write examples:

```bash
TOKEN='replace-with-long-random-token'

curl -fsS -X PUT http://127.0.0.1:8444/api/cabinet-profile \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  --data '{"wire_host_url":"https://pi-host.local:8444/g2s","listener_dns_name":"pi-host.local","listener_ip":"192.168.50.40","required_san_dns":["pi-host.local"],"required_san_ips":["192.168.50.40"],"host_id":"HOST-PI-001","first_test_egm_ids":["EGM-01"]}'

curl -fsS -X POST http://127.0.0.1:8444/api/certificates/import \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  --data-binary @/tmp/g2s-cert-import.json

curl -fsS "http://127.0.0.1:8444/api/certificates/export?role=g2s_client_cert&include_key=true" \
  -H "Authorization: Bearer ${TOKEN}"
```

For the Pi appliance, prefer the runtime configuration helper instead of hand-editing the active config:

```bash
bash ./scripts/pi-configure-runtime.sh
```

The helper:

- creates or reuses `~/.g2s_api_token`
- updates `/etc/g2s-mute/config.json` with `api.auth_token`
- writes non-placeholder `cabinet_profile` defaults using the Pi hostname/IP
- restarts `g2s-mute.service`
- clears stale cabinet profile overrides through the API
- runs `scripts/release-gate.sh` against `http://127.0.0.1:8444`

Useful overrides:

```bash
WIRE_HOST_URL=https://tspi4.local:8444/g2s \
LISTENER_DNS_NAME=tspi4.local \
LISTENER_IP=192.168.10.25 \
HOST_ID=HOST-TSPI4-001 \
FIRST_TEST_EGM_IDS=EGM-001 \
bash ./scripts/pi-configure-runtime.sh
```

Write and backup behavior on import:

- writes to configured runtime paths in `crypto.*_path`
- safe write pattern: temp file + rename
- replaced files are backed up with timestamp suffix: `*.bak-YYYYMMDDTHHMMSSZ`
- file permissions:
  - certificate files: `0644`
  - private keys: `0600`
- certificate inventory is refreshed after import (`GET /api/certificates`)

## Cabinet Profile Persistence

Cabinet-facing identity values are now explicit in `cabinet_profile` and have two persistence layers:

- file defaults in `/etc/g2s-mute/config.json`
- optional DB override row in `cabinet_profile_overrides` (inside the configured SQLite database)

Effective runtime behavior:

- file profile only: `profile_source=file`
- full override row: `profile_source=override`
- partial override row with file fallback: `profile_source=mixed`

Runtime endpoints:

- `GET /api/status` includes `cabinet_profile`, `profile_source`, `profile_last_updated_at`, and `profile_differs_from_file`
- `GET /api/cabinet-profile` returns effective profile and override metadata
- `GET /api/cabinet-preflight` runs actionable cabinet readiness checks and returns `overall`, `checks`, and `blockers`
- `PUT /api/cabinet-profile` writes/updates override values (requires bearer token when `api.auth_token` is set)
- `DELETE /api/cabinet-profile` clears override and reverts to file values (requires bearer token when `api.auth_token` is set)

Reset/recovery path:

1. Confirm file defaults in `/etc/g2s-mute/config.json`
2. Clear override via `DELETE /api/cabinet-profile`
3. Verify `profile_source=file` in `/api/status`

Run cabinet preflight:

```bash
curl -fsS http://127.0.0.1:8444/api/cabinet-preflight
bash ./scripts/cabinet-preflight.sh
```

Interpretation:

- `overall=PASS`: the current runtime state satisfies all required cabinet preflight checks.
- `overall=FAIL`: one or more blockers were detected; use `blockers` and per-check details to resolve them before a real cabinet session.

## One-Command Release Gate

Use the combined first-cabinet gate script to validate runtime health, readiness, preflight policy, and API auth behavior in one run.

Default (service-mode HTTP on port 8444):

```bash
bash ./scripts/release-gate.sh
```

TLS/user-mode example:

```bash
API_BASE=https://127.0.0.1:9443 API_TOKEN=replace-with-token bash ./scripts/release-gate.sh
```

Gate behavior:

- checks `/healthz`, `/readyz`, and `/api/status` (HTTP 200 required)
- checks `/api/cabinet-preflight` and requires `overall=PASS`
- runs `scripts/api-auth-smoke.sh` using the same `API_BASE`/`API_TOKEN`
- exits non-zero on any required failure
- writes a markdown run report to `docs/pi-runs/<UTC-date>-release-gate-run.md`

## First Cabinet Session Plan

For the operator-grade first live cabinet sequence (preconditions, command order, evidence, triage, and Go/No-Go gates), use:

- [First Cabinet Session Execution Plan](./First-Cabinet-Session-Execution-Plan.md)

## Next Hardware Checks

Before enabling real GPIO behavior, confirm:

- security line GPIO pin, voltage level, polarity, and isolation
- PSU 1 and PSU 2 GPIO pins and polarity
- buzzer driver circuit, GPIO pin, and safe current limits
- whether the Pi will use SSD storage and a UPS/power-loss strategy
