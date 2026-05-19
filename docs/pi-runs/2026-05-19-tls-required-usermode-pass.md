# 2026-05-19 TLS Required User-Mode Validation

## Freshness Proof
- `date -u`: `Tue May 19 20:03:03 UTC 2026`
- `git rev-parse --short HEAD`: `a0ecbd2` (run start)
- `git log --oneline -3`:
  - `a0ecbd2 Make SAN preflight policy consistent for certificate-optional mode`
  - `7e34613 Align user-mode Pi config and record pass-8 preflight remediation`
  - `7dfd6bc Improve preflight cert-path remediation and Pi user-mode runbook`

## Gates
- `GATES: SUDO_OK=0 PASS_PHRASE_NEEDED=NO PASS_PHRASE_READY=N/A`
- `SUDO_DETAIL`:
  - `sudo: /etc/sudo.conf is owned by uid 65534, should be 0`
  - `sudo: The "no new privileges" flag is set, which prevents sudo from running as root.`
  - `sudo: If sudo is running in a container, you may need to adjust the container configuration to disable the flag.`

## Changes Made
1. Added TLS user config at `configs/config.tls.user.json`:
   - `web_ui.bind_address=127.0.0.1:9443`
   - `database.path=/tmp/g2s-mute-tls-user.db`
   - `g2s.require_tls=true`, `g2s.require_client_cert=false`
   - cabinet profile uses non-placeholder local values for `localhost`/`127.0.0.1`
   - cert paths under `./certs/tls-lab`
2. Generated cert set with existing tooling:
   - `go run ./cmd/g2s-dev-certs -out ./certs/tls-lab`
   - includes SAN for `localhost` and `127.0.0.1`
3. Updated `scripts/cabinet-preflight.sh` to support `API_BASE` override while preserving `PREFLIGHT_URL` precedence:
   - `API_BASE="${API_BASE:-http://127.0.0.1:8444}"`
   - `PREFLIGHT_URL="${PREFLIGHT_URL:-${API_BASE%/}/api/cabinet-preflight}"`

## Validation
- `export GOMODCACHE=/tmp/g2s-go/pkg/mod`
- `export GOCACHE=/tmp/g2s-go/build`
- `go test ./...` passed (before and after changes).

TLS runtime validation (user mode, TLS config):
- `curl -k -fsS https://127.0.0.1:9443/healthz` => `ok`
- `curl -k -fsS https://127.0.0.1:9443/readyz` => `{"overall":"READY","issues":[]}`
- `curl -k -fsS https://127.0.0.1:9443/api/cabinet-preflight` => `overall=PASS`, all checks `PASS`, no blockers.

## TLS Preflight JSON
```json
{"overall":"PASS","checks":[{"id":"service_readiness","result":"PASS","message":"Readiness check is healthy for /readyz policy","detail":"overall=READY"},{"id":"cabinet_profile","result":"PASS","message":"Cabinet profile is complete","detail":"wire_host_url=https://localhost:9443/g2s; host_id=HOST-LOCAL-9443"},{"id":"profile_source","result":"PASS","message":"Profile source is explicit","detail":"profile_source=file"},{"id":"certificate_mode_requirements","result":"PASS","message":"Required certificates for configured runtime mode are valid","detail":"roles=web_server_cert"},{"id":"certificate_san_wire_identity","result":"PASS","message":"Web server certificate SAN matches configured wire identity","detail":"wire_identity=localhost"}],"blockers":[],"timestamp":"2026-05-19T20:40:09.155902756Z"}
```
