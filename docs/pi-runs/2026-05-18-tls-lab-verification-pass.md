# Pi Run Report - TLS Lab Verification

Date: 2026-05-18

## Commands Run
1. `cd ~/projects/G2S_MC`
2. `git status --short`
3. `go test ./...`
4. `sudo g2s-dev-certs -out /etc/g2s-mute/certs`
5. `sudo chown -R g2s-mute:g2s-mute /etc/g2s-mute/certs`
6. `ls -la /etc/g2s-mute/certs`
7. `sudo cp /etc/g2s-mute/config.json /etc/g2s-mute/config.json.bak`
8. `sudo cp ./configs/config.tls.example.json /etc/g2s-mute/config.json`
9. `sudo chown g2s-mute:g2s-mute /etc/g2s-mute/config.json`
10. `sudo systemctl restart g2s-mute.service`
11. `systemctl status g2s-mute.service --no-pager --full`
12. `curl -k -fsS -i https://127.0.0.1:8444/healthz`
13. `curl -k -fsS -i https://127.0.0.1:8444/readyz`
14. `curl -k -fsS https://127.0.0.1:8444/api/status`
15. `curl -k -fsS https://127.0.0.1:8444/api/certificates`
16. `go run ./cmd/g2s-fake-egm -host-url https://localhost:8444/g2s -egm-id EGM-01 -ca /etc/g2s-mute/certs/ca.crt -cert /etc/g2s-mute/certs/client.crt -key /etc/g2s-mute/certs/client.key -keepalive-count 1 -keepalive-interval 0s`
17. `journalctl -u g2s-mute.service -n 120 --no-pager`
18. `sudo cp /etc/g2s-mute/config.json.bak /etc/g2s-mute/config.json`
19. `sudo chown g2s-mute:g2s-mute /etc/g2s-mute/config.json`
20. `sudo systemctl restart g2s-mute.service`
21. `systemctl status g2s-mute.service --no-pager --full`

## Pass/Fail by Step
- Step 1 baseline checks: PASS (`git status` clean; `go test ./...` passed).
- Step 2 cert generation: PASS (CA/host/client cert+key generated under `/etc/g2s-mute/certs`).
- Step 3 config switch: PASS (backup + TLS config copy + ownership complete).
- Step 4 TLS start: FAIL initially, then PASS after minimal runtime-only config fix.
  - Failure cause: TLS example config used relative paths (`./data`, `./certs`) under systemd and failed with `open audit store: mkdir data: permission denied`.
  - Minimal fix applied in `/etc/g2s-mute/config.json` only:
    - `./data/controller-tls.db` -> `/var/lib/g2s-mute/controller-tls.db`
    - `./certs/*` -> `/etc/g2s-mute/certs/*`
  - After fix: service started with `protocol=https`.
- Step 5 HTTPS endpoint checks with provided `curl -k` commands: FAIL.
  - Error: `tlsv13 alert certificate required`.
  - Reason: TLS lab config sets `g2s.require_client_cert=true`, and server enforces client cert at TLS handshake.
- Step 6 fake EGM mTLS: PASS.
  - `commsOnLine -> HTTP 200`
  - `keepAlive 1 -> HTTP 200`
- Step 7 logs collection: PASS (`journalctl` captured startup/failure/success and TLS handshake entries).
- Step 8 restore non-TLS config: PASS.
  - Service restarted on original config and returned to `protocol=http`.

## Key Endpoint Results
- In TLS+mTLS mode:
  - `curl -k` to `/healthz`, `/readyz`, `/api/status`, `/api/certificates`: failed due client cert requirement.
  - Service log confirmed HTTPS startup: `service ready protocol=https ...`.
- After restore:
  - Service log confirmed HTTP startup: `service ready protocol=http ...`.

## mTLS Fake EGM Result
- PASS:
  - `commsOnLine -> HTTP 200 in 22ms`
  - `keepAlive 1 -> HTTP 200 in 1ms`

## Code Change Status
- Repo code changes required: No.
- Runtime-only operational adjustment in `/etc/g2s-mute/config.json` was applied during verification and then original config was restored from backup.

## Final Git Status
- `?? docs/pi-runs/2026-05-18-tls-lab-verification-pass.md`
