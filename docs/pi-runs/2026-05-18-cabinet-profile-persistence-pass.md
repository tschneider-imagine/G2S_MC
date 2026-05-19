# Pi Run Report: Cabinet Profile Persistence Pass (2026-05-18)

## Commands Run
- `go test ./...`
- `bash ./scripts/pi-install.sh`
- `sudo systemctl restart g2s-mute.service`
- `curl -fsS http://127.0.0.1:8444/api/status`
- `curl -fsS http://127.0.0.1:8444/api/cabinet-profile`
- `curl -sS -i -X PUT http://127.0.0.1:8444/api/cabinet-profile -H 'Content-Type: application/json' -d '{"host_id":"HOST-PI-OVERRIDE-01","first_test_egm_ids":["EGM-01","EGM-02"]}'`
- `curl -fsS -X PUT http://127.0.0.1:8444/api/cabinet-profile -H 'Content-Type: application/json' -d '{"wire_host_url":"https://pi-override.example:8444/g2s","listener_dns_name":"pi-override.example","listener_ip":"192.168.50.41","required_san_dns":["pi-override.example"],"required_san_ips":["192.168.50.41"],"host_id":"HOST-PI-OVERRIDE-01","first_test_egm_ids":["EGM-01","EGM-02"]}'`
- `curl -fsS http://127.0.0.1:8444/api/status`
- `curl -fsS http://127.0.0.1:8444/api/cabinet-profile`
- `curl -fsS -X DELETE http://127.0.0.1:8444/api/cabinet-profile`
- `curl -fsS http://127.0.0.1:8444/api/status`
- `curl -fsS http://127.0.0.1:8444/api/cabinet-profile`
- `systemctl status g2s-mute.service --no-pager --full`

## Pass/Fail Summary
- `go test ./...`: PASS
- Pi install script: PASS
- Service restart: PASS
- `/api/status` baseline (`profile_source=file`): PASS
- `/api/cabinet-profile` baseline (`override_present=false`): PASS
- PUT partial override validation rejection: PASS (expected 400 with required-field details)
- PUT full override persistence: PASS (`profile_source=override`, `profile_last_updated_at` set)
- DELETE override fallback: PASS (returns to `profile_source=file`)
- Final service health: PASS (`active (running)`)

## Files Changed (Code + Docs)
- `cmd/g2s-mute/main.go`
- `cmd/g2s-mute/main_test.go`
- `internal/config/config.go`
- `internal/config/load.go`
- `internal/config/config_test.go`
- `internal/store/migrations.go`
- `internal/store/sqlite.go`
- `internal/store/sqlite_test.go`
- `internal/ui/assets_operator_v2.go`
- `internal/ui/server_test.go`
- `configs/config.example.json`
- `configs/config.pi.example.json`
- `configs/config.tls.example.json`
- `docs/First-Cabinet-Prep-Checklist.md`
- `docs/raspberry-pi.md`
- `docs/pi-runs/2026-05-18-cabinet-profile-persistence-pass.md`

## Migration/Table Added
- New SQLite table: `cabinet_profile_overrides`
- Purpose: persist operator-level cabinet profile overrides (single-row pattern) with `updated_at` and `updated_by`.

## Persistence Behavior Verified
- Source-of-truth file profile loaded from `/etc/g2s-mute/config.json`.
- Override stored in DB and merged as effective runtime profile.
- Status metadata correctly emitted:
  - `profile_source` (`file`/`override`)
  - `profile_last_updated_at` (when override exists)
  - `profile_differs_from_file`
- Clearing override reverts effective runtime profile to file values.

## Runtime Verification Results
- `/api/status` returned cabinet profile fields and source metadata.
- `/api/cabinet-profile` GET/PUT/DELETE behaved as expected.
- `g2s-mute.service` logs include cabinet profile source and wire host URL.

## Remaining Unknowns
- Final cabinet network values (production DNS/IP and route approval) are still site-dependent.
- Final production cert authority/issuance ownership is still external.
- Confirm whether override API should support partial updates without full required-field payload.
- Authn/authz for `/api/cabinet-profile` is lab-only and needs production hardening plan.
- First-cabinet EGM ID set may expand beyond current placeholders.
