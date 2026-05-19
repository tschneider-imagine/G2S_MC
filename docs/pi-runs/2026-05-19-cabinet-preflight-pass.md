# Pi Run Report: Cabinet Preflight Endpoint + Script (2026-05-19)

## Commands Run

1. `go test ./...`
2. `sudo bash ./scripts/pi-install.sh`
3. `sudo systemctl restart g2s-mute.service`
4. `curl -fsS http://127.0.0.1:8444/api/cabinet-preflight`
5. `bash ./scripts/cabinet-preflight.sh`
6. `systemctl status g2s-mute.service --no-pager --full`

## Results

- `go test ./...`: PASS
- install + restart: PASS
- service status: PASS (`active (running)`)

Preflight endpoint result:

- `overall=FAIL`
- blockers:
  - `Cabinet profile has missing or placeholder values`
  - `Web server certificate could not be parsed for SAN checks`

Wrapper script result:

- output:
  - `Cabinet preflight: FAIL`
  - ` - Cabinet profile has missing or placeholder values`
  - ` - Web server certificate could not be parsed for SAN checks`
- exit code: `1` (expected for FAIL)

## Notes

- FAIL is expected with current default Pi lab runtime:
  - `cabinet_profile` still uses placeholder-style identity values.
  - `crypto.web_server_cert_path` is empty in HTTP lab mode, so SAN compatibility cannot pass.
- The endpoint/checks are actionable and identify exact blockers to clear before first real cabinet session.

## Files Changed in This Slice

- `cmd/g2s-mute/cabinet_preflight.go`
- `cmd/g2s-mute/cabinet_preflight_test.go`
- `cmd/g2s-mute/main.go`
- `docs/raspberry-pi.md`
- `docs/pi-runs/2026-05-19-cabinet-preflight-pass.md`
- `scripts/cabinet-preflight.sh`

