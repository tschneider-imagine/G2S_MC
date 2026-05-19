# 2026-05-19 API Auth Live Smoke (Pass)

## Scope
Validate live API auth hardening behavior on user-mode TLS instance:
- Read endpoints remain open.
- Mutating endpoints require `Authorization: Bearer <token>` when `api.auth_token` is configured.

## Gates
- `GATES: SUDO_OK=0 PASS_PHRASE_NEEDED=NO PASS_PHRASE_READY=N/A`
- `SUDO_DETAIL: sudo: /etc/sudo.conf is owned by uid 65534, should be 0`
- `sudo: The "no new privileges" flag is set, which prevents sudo from running as root.`
- `sudo: If sudo is running in a container, you may need to adjust the container configuration to disable the flag.`

## Runtime / Config
- User-mode TLS config: `configs/config.tls.user.json`
- Bind: `127.0.0.1:9443`
- API token: `lab-token-9443`
- Cert source for import smoke payload: `certs/tls-lab/host.crt` + `certs/tls-lab/host.key`

## Test + Live Validation
- `go test ./...` passed.
- Live curl auth matrix against `https://127.0.0.1:9443`:
  - `GET /healthz` (no token) -> `200`
  - `GET /api/status` (no token) -> `200`
  - `PUT /api/cabinet-profile` (no token) -> `401`
  - `DELETE /api/cabinet-profile` (no token) -> `401`
  - `POST /api/certificates/import` (no token) -> `401`
  - `PUT /api/cabinet-profile` (with token) -> `200`
  - `DELETE /api/cabinet-profile` (with token) -> `200`
  - `POST /api/certificates/import` (with token) -> `200`

## Artifacts
- Added reusable smoke script: `scripts/api-auth-smoke.sh`
  - Supports `API_BASE` and `API_TOKEN` env vars.
  - Also supports `CERT_PATH` and `KEY_PATH` overrides for import payload source.
