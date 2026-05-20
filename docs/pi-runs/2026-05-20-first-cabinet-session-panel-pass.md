# 2026-05-20 First Cabinet Session Panel (Pass - Code/Test)

## Gates
- `pwd`: `/home/ts/projects/G2S_MC`
- `whoami`: `ts`
- `GATES: SUDO_NEEDED=YES PASS_PHRASE_NEEDED=ONLY_FOR_PUSH`
- `SUDO_OK=0`
- `SUDO_DETAIL: sudo: /etc/sudo.conf is owned by uid 65534, should be 0`
- `SUDO_DETAIL: sudo: The "no new privileges" flag is set, which prevents sudo from running as root.`

## Scope
Added a new Operator Console v2 panel titled **First Cabinet Session** that aggregates:
- `/readyz`
- `/api/status`
- `/api/cabinet-preflight`
- `/api/certificates`
- `/api/cabinet-profile`

The panel shows:
- overall session state
- last checked time
- readyz state
- cabinet preflight state
- cabinet profile source
- wire host URL
- host ID
- first test EGM IDs
- certificate blocking count
- lab optional certificate count
- API auth required/disabled state
- concise operator blocker list

If all checks are clear, the panel message is:
- `Ready for first cabinet lab session`

## Implementation Notes
- No new backend endpoints were added.
- Added `runtime.api_mutation_auth_required` in `/api/status` payload to expose auth state for the session panel.
- Added operator-friendly blocker mapping for preflight failures (for example, cabinet profile incomplete, required certificate missing, preflight API unavailable, token required).

## Test Coverage
- `internal/ui/server_test.go` now validates:
  - First Cabinet Session HTML panel marker
  - JS runbook/session rendering marker
  - CSS runbook/session styling marker
- `cmd/g2s-mute/main_test.go` includes runtime auth-state field coverage.

## Local Validation
- `go test ./...` -> PASS

## Pi Runtime Validation
- Blocked in this environment due unavailable sudo (`SUDO_OK=0`), so the following were not runnable:
  - `sudo bash ./scripts/pi-install.sh`
  - `sudo systemctl restart g2s-mute.service`
  - Pi-side health/release-gate validation sequence
