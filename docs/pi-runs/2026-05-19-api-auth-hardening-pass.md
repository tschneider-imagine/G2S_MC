# 2026-05-19 API Auth Hardening Pass

## Gates
- `GATES: SUDO_OK=0 PASS_PHRASE_NEEDED=NO PASS_PHRASE_READY=N/A`
- `SUDO_DETAIL:`
  - `sudo: /etc/sudo.conf is owned by uid 65534, should be 0`
  - `sudo: The "no new privileges" flag is set, which prevents sudo from running as root.`
  - `sudo: If sudo is running in a container, you may need to adjust the container configuration to disable the flag.`

## What Changed
1. Added optional config support for API mutation auth token:
   - `api.auth_token` in `internal/config/config.go`
2. Added shared bearer-token guard:
   - `cmd/g2s-mute/api_auth.go`
   - behavior:
     - token unset: mutating calls remain open (backward-compatible lab mode)
     - token set: require `Authorization: Bearer <token>`
     - missing/invalid token -> `401 unauthorized`
3. Protected mutating endpoints only:
   - `PUT /api/cabinet-profile`
   - `DELETE /api/cabinet-profile`
   - `POST /api/certificates/import`
   - `GET /api/certificates/export?...&include_key=true`
   - read endpoints remain open
4. Updated docs with config + curl examples:
   - `docs/raspberry-pi.md`

## Test Coverage
- Added cabinet profile auth guard tests:
  - GET open without token
  - PUT/DELETE denied without or with bad token when token configured
  - PUT/DELETE succeed with valid token
- Added certificate API auth guard tests:
  - certificate import requires token when configured
  - certificate-only export remains open
  - `include_key=true` export requires token when configured
- Existing no-token flows continue to pass.

## Validation
- `export GOMODCACHE=/tmp/g2s-go/pkg/mod`
- `export GOCACHE=/tmp/g2s-go/build`
- `go test ./...` => PASS
