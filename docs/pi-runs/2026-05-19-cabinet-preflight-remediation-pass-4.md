# 2026-05-19 Cabinet Preflight Remediation (Pass 4)

## Environment
- Repo: `/home/ts/projects/G2S_MC`
- User: `ts`
- API base: `http://127.0.0.1:8444`
- `sudo -n true`: failed (`SUDO_OK=0` in sandbox)
- Effective runtime for API validation: existing host `systemd` service (`g2s-mute.service`) running outside sandbox

## Guard + Baseline
- `go test ./...` passed with:
  - `GOMODCACHE=/tmp/g2s-go/pkg/mod`
  - `GOCACHE=/tmp/g2s-go/build`
- Baseline health:
  - `GET /healthz` => `200 ok`
  - `GET /readyz` => `{"overall":"READY_LAB","issues":[]}`
- Baseline preflight (`GET /api/cabinet-preflight`) was:
  - `overall=FAIL`
  - blocker 1: `Cabinet profile has missing or placeholder values`
  - blocker 2: `Web server certificate could not be parsed for SAN checks` (`web_server_cert path is empty`)

## Runtime Notes
- Attempted sandboxed user-mode process with `configs/config.pi.example.json` failed on DB open:
  - `open audit store: unable to open database file (14)`
- Created user-mode config template `configs/config.pi.user.json` with local DB/cert paths for non-root runs.
- Existing host service remained healthy and reachable; remediation proceeded through live API.

## Remediation Applied

### 1) Cleared placeholder cabinet profile blocker
Applied override via API:

```bash
curl -fsS -X PUT http://127.0.0.1:8444/api/cabinet-profile \
  -H 'Content-Type: application/json' \
  --data '{"wire_host_url":"https://tspi4.local:8444/g2s","listener_dns_name":"tspi4.local","listener_ip":"192.168.10.25","required_san_dns":["tspi4.local"],"required_san_ips":["192.168.10.25"],"host_id":"HOST-TSPI4-001","first_test_egm_ids":["EGM-A100"]}'
```

Result:
- `cabinet_profile` check moved to `PASS`
- `profile_source=override`

### 2) Cert SAN parse blocker remediation (code + tests)
Observed remaining live blocker:
- `certificate_san_wire_identity`: `FAIL`
- detail: `web_server_cert path is empty`

Code changes made (strict checks preserved):
- Added precise actionable parse-failure detail builder in preflight cert SAN check.
- Empty-path failures now direct operator to set:
  - `crypto.web_server_cert_path`
  - `crypto.web_server_key_path`
  - then restart `g2s-mute`
- Added tests covering empty path, permission error, and missing-file parse failures.

## Validation
- `go test ./...` passed after changes.
- Live API validation (running systemd service) after profile override:
  - `GET /api/cabinet-preflight` => `overall=FAIL` with single blocker:
    - `Web server certificate could not be parsed for SAN checks`
  - `bash ./scripts/cabinet-preflight.sh` prints same single blocker.

## Remaining External Requirement
To fully clear preflight to `PASS` on this Pi runtime:
1. Set non-empty `crypto.web_server_cert_path` and `crypto.web_server_key_path` in `/etc/g2s-mute/config.json`.
2. Ensure cert file is readable PEM `CERTIFICATE` and SAN covers `wire_host_url` host (`tspi4.local`).
3. Restart `g2s-mute.service`.

These are root-managed runtime config/service operations and were not executable from the current non-sudo sandbox context.
