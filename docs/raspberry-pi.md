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
- `PUT /api/cabinet-profile` writes/updates override values (lab-only endpoint until auth is added)
- `DELETE /api/cabinet-profile` clears override and reverts to file values

Reset/recovery path:

1. Confirm file defaults in `/etc/g2s-mute/config.json`
2. Clear override via `DELETE /api/cabinet-profile`
3. Verify `profile_source=file` in `/api/status`

## First Cabinet Session Plan

For the operator-grade first live cabinet sequence (preconditions, command order, evidence, triage, and Go/No-Go gates), use:

- [First Cabinet Session Execution Plan](./First-Cabinet-Session-Execution-Plan.md)

## Next Hardware Checks

Before enabling real GPIO behavior, confirm:

- security line GPIO pin, voltage level, polarity, and isolation
- PSU 1 and PSU 2 GPIO pins and polarity
- buzzer driver circuit, GPIO pin, and safe current limits
- whether the Pi will use SSD storage and a UPS/power-loss strategy
