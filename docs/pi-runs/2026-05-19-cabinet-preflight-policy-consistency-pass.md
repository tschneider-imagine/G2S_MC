# 2026-05-19 Cabinet Preflight Policy Consistency

## Gates
- `GATES: SUDO_OK=0 PASS_PHRASE_NEEDED=NO PASS_PHRASE_READY=N/A`
- `SUDO_DETAIL`:
  - `sudo: /etc/sudo.conf is owned by uid 65534, should be 0`
  - `sudo: The "no new privileges" flag is set, which prevents sudo from running as root.`
  - `sudo: If sudo is running in a container, you may need to adjust the container configuration to disable the flag.`

## Policy Fix Implemented
Updated preflight SAN policy in `cmd/g2s-mute/cabinet_preflight.go`:
- When runtime mode is certificate-optional (`g2s.require_tls=false` and `g2s.require_client_cert=false`), `certificate_san_wire_identity` now returns `PASS` (skipped) if:
  - `web_server_cert` inventory record is missing, or
  - `web_server_cert` path is empty.
- When runtime mode requires certificates (`g2s.require_tls=true` or `g2s.require_client_cert=true`), strict behavior is preserved:
  - missing inventory record => `FAIL`
  - parse/read errors => `FAIL`
  - SAN mismatch => `FAIL`

## Tests Added/Updated
In `cmd/g2s-mute/cabinet_preflight_test.go`:
- Added coverage that certificate-optional mode with empty `web_server_cert_path` produces SAN check `PASS` (skipped).
- Added coverage that certificate-required mode with empty `web_server_cert_path` still fails SAN check.
- Existing failure-path assertions remain intact.

## Validation
Commands run with:
- `export GOMODCACHE=/tmp/g2s-go/pkg/mod`
- `export GOCACHE=/tmp/g2s-go/build`

Results:
- `go test ./...` => pass.
- Sandbox namespace API checks:
  - `curl -fsS http://127.0.0.1:8444/api/cabinet-preflight || true` => connection failed.
  - `bash ./scripts/cabinet-preflight.sh || true` => explicit API-unavailable FAIL output.
- Live host API payload collected outside sandbox namespace:
  - current running service still reports `certificate_san_wire_identity=FAIL` with `web_server_cert path is empty` (existing binary/config state).

## Note
This pass delivers and verifies the code+test policy change. Live runtime behavior at `:8444` remains unchanged until the service runs the updated build.
