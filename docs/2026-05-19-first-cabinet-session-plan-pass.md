# Pi Run Report: First Cabinet Session Plan Pass (2026-05-19)

## Commands Run

1. `git status --short`
2. `git branch -vv`
3. `go test ./...`
4. `curl -fsS http://127.0.0.1:8444/api/status`
5. `curl -fsS http://127.0.0.1:8444/api/cabinet-profile`
6. `curl -fsS -i http://127.0.0.1:8444/readyz`
7. `curl -fsS -X PUT http://127.0.0.1:8444/api/cabinet-profile -H 'Content-Type: application/json' --data '{"wire_host_url":"https://override-host.example:9443/g2s","listener_dns_name":"override-host.example","listener_ip":"192.168.50.41","required_san_dns":["override-host.example"],"required_san_ips":["192.168.50.41"],"host_id":"HOST-OVERRIDE-001","first_test_egm_ids":["EGM-99"]}'`
8. `curl -fsS http://127.0.0.1:8444/api/cabinet-profile`
9. `curl -fsS http://127.0.0.1:8444/api/status`
10. `curl -fsS -X DELETE http://127.0.0.1:8444/api/cabinet-profile`
11. `curl -fsS http://127.0.0.1:8444/api/cabinet-profile`
12. `curl -fsS http://127.0.0.1:8444/api/status`
13. `sed -n '1,220p' docs/First-Cabinet-Prep-Checklist.md`
14. `sed -n '1,260p' docs/raspberry-pi.md`
15. `sed -n '1,260p' docs/pi-runs/2026-05-18-cabinet-profile-persistence-pass.md`

## Files Changed

- `docs/First-Cabinet-Session-Execution-Plan.md` (new)
- `docs/raspberry-pi.md` (link to execution plan)
- `docs/2026-05-19-first-cabinet-session-plan-pass.md` (new; fallback path)

No content update was required in `docs/First-Cabinet-Prep-Checklist.md` for this pass.

## Key Runtime Observations

- Baseline state was clean and test suite passed (`go test ./...`).
- `GET /api/status` baseline:
  - `profile_source=file`
  - `profile_differs_from_file=false`
  - `cabinet_profile` values present and complete.
- `GET /api/cabinet-profile` baseline:
  - `override_present=false`
  - effective profile matched file values.
- `GET /readyz` returned `HTTP/1.1 200 OK` with `{"overall":"READY_LAB","issues":[]}`.
- Override persistence behavior verified:
  - PUT set `profile_source=override` and `profile_last_updated_at`.
  - DELETE reverted to `profile_source=file`.
  - Status and cabinet-profile endpoints stayed consistent across transitions.

## Remaining Unknowns Blocking Live Cabinet Session

1. Final production cabinet-facing DNS/IP and `wire_host_url` are still placeholders.
2. Final certificate issuance/trust chain ownership and approved SAN set remain site/vendor dependent.
3. Registered-host role/class permission confirmation on the real EGM is pending.
4. Real cabinet session evidence for `commsOnLine`/ACK/keepalive under production network conditions is not yet captured.
5. Authn/authz hardening for override endpoints is still pending for production use.

## Notes

- Requested path `docs/pi-runs/2026-05-19-first-cabinet-session-plan-pass.md` could not be created in this session because `docs/pi-runs` is owned by `nobody:nogroup` and not writable by `ts`.
