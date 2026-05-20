# 2026-05-19 Auth Enforcement Fix (Pass)

## Gates
- `GATES: SUDO_OK=0 PASS_PHRASE_NEEDED=NO PASS_PHRASE_READY=N/A`
- `SUDO_DETAIL: sudo: /etc/sudo.conf is owned by uid 65534, should be 0`
- `sudo: The "no new privileges" flag is set, which prevents sudo from running as root.`
- `sudo: If sudo is running in a container, you may need to adjust the container configuration to disable the flag.`

## Scope
- Enforce auth on mutating cabinet profile endpoints when `api.auth_token` is configured.
- Keep read endpoints open.

## Code Changes
1. Added method-scoped route middleware:
   - `requireMutationAuthForMethods(...)` in `cmd/g2s-mute/api_auth.go`.
2. Applied route-level guard to cabinet profile mutating methods only:
   - Wrapped `/api/cabinet-profile` in `cmd/g2s-mute/main.go` for `PUT` and `DELETE`.
3. Kept read endpoint behavior open:
   - `GET /api/cabinet-profile` remains unguarded.
4. Updated auth regression coverage:
   - `TestCabinetProfileRouteMutationAuthTokenGuard` in `cmd/g2s-mute/main_test.go`.
   - Verifies no-token `PUT`/`DELETE` deny with `401/403`.
   - Verifies with-token `PUT`/`DELETE` succeed.

## Validation
- `export GOMODCACHE=/tmp/g2s-go/pkg/mod`
- `export GOCACHE=/tmp/g2s-go/build`
- `go test ./...` -> PASS
- `API_BASE=http://127.0.0.1:8444 API_TOKEN="$(cat ~/.g2s_api_token)" bash ./scripts/release-gate.sh` -> FAIL
  - `healthz`, `readyz`, `status`: PASS
  - `cabinet-preflight`: FAIL (`profile_source=mixed`)
  - `api-auth-smoke`: FAIL (`PUT/DELETE /api/cabinet-profile` no-token returned `200`)

## Runtime Note
- Host process owning `127.0.0.1:8444` during validation:
  - `/usr/local/bin/g2s-mute -config /etc/g2s-mute/config.json`
- Non-sudo user cannot signal that process:
  - `kill -0 <pid>` -> `Operation not permitted`
- Because of this, live gate on `8444` exercised the host-managed service binary/config, not a replaceable repo-run instance.
