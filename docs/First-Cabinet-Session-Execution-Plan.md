# First Cabinet Session Execution Plan

Purpose:

Provide an operator-ready, step-by-step first cabinet session plan based on the persisted `cabinet_profile` implementation and current Pi runtime behavior.

Scope guardrails:

- This plan is for session startup, identity/TLS/registration verification, and evidence capture.
- This plan does not include cabinet control logic execution.
- This plan does not include GPIO behavior changes.

## A) Session Objective And Success Criteria

Objective:

Run one controlled first cabinet session that proves identity, endpoint, TLS, registered-host setup, and baseline G2S session readiness while preserving full audit evidence.

Success criteria:

1. Service is healthy and `readyz` reports `READY` or `READY_LAB`.
2. Effective `cabinet_profile` values match the approved on-wire identity for the session.
3. `profile_source` is explicitly known (`file`, `override`, or `mixed`) and documented before starting cabinet exchange.
4. Certificate SANs match the exact DNS/IP identity used on wire.
5. EGM registered-host entry and role are confirmed.
6. First session startup messages (at minimum `commsOnLine` and ACK path) are captured in logs/transcripts.
7. Go/No-Go gate passes before any control operation.

## B) Preconditions Checklist (Network, Certs, Host Registration, Config)

- [ ] Pi service is installed and running (`g2s-mute.service` active).
- [ ] Cabinet VLAN can route to Pi listener (`8444/tcp`) without TLS interception.
- [ ] `/etc/g2s-mute/config.json` contains approved `cabinet_profile` defaults for the session window.
- [ ] Certificate files exist and ownership is correct:
  - `/etc/g2s-mute/certs/ca.crt`
  - `/etc/g2s-mute/certs/host.crt`
  - `/etc/g2s-mute/certs/host.key`
  - `/etc/g2s-mute/certs/client.crt` (if mTLS)
  - `/etc/g2s-mute/certs/client.key` (if mTLS)
- [ ] Host certificate SAN includes exact cabinet-facing DNS/IP.
- [ ] EGM registered-host entry exists and uses approved role/permissions.
- [ ] Vendor/site support contact and rollback window are active.

## C) Exact Runtime Values To Confirm (From Cabinet Profile Fields)

Confirm from `GET /api/cabinet-profile` and `GET /api/status` immediately before session start.

Current observed baseline on Pi (2026-05-19):

| Field | Required Meaning | Observed Value |
| --- | --- | --- |
| `cabinet_profile.wire_host_url` | Exact URL EGM must use on wire | `https://pi-cabinet-host.example:8444/g2s` |
| `cabinet_profile.listener_dns_name` | DNS identity for certificate/SAN checks | `pi-cabinet-host.example` |
| `cabinet_profile.listener_ip` | IP identity for certificate/SAN checks | `192.168.50.40` |
| `cabinet_profile.required_san_dns[]` | Required DNS SAN set | `["pi-cabinet-host.example"]` |
| `cabinet_profile.required_san_ips[]` | Required IP SAN set | `["192.168.50.40"]` |
| `cabinet_profile.host_id` | Host identity expected by EGM | `HOST-PI-001` |
| `cabinet_profile.first_test_egm_ids[]` | First cabinet set for session | `["EGM-01"]` |
| `profile_source` | Runtime source of effective profile | `file` (baseline) |
| `profile_last_updated_at` | Timestamp when override exists | unset at baseline |

Stop condition:

- If any value above differs from approved session sheet, stop and resolve before cabinet connection.

## D) Exact Command Sequence To Run On Pi During First Session

Run in order. Keep all output artifacts.

```bash
cd ~/projects/G2S_MC
RUN_ID="first-cabinet-$(date -u +%Y%m%dT%H%M%SZ)"
ARTIFACT_DIR="$HOME/g2s-session-artifacts/$RUN_ID"
mkdir -p "$ARTIFACT_DIR"

go test ./... | tee "$ARTIFACT_DIR/go-test.txt"
systemctl --no-pager --full status g2s-mute.service | tee "$ARTIFACT_DIR/systemctl-status-pre.txt"
journalctl -u g2s-mute.service -n 200 --no-pager | tee "$ARTIFACT_DIR/journal-pre.txt"

curl -fsS -i http://127.0.0.1:8444/readyz | tee "$ARTIFACT_DIR/readyz-pre.txt"
curl -fsS http://127.0.0.1:8444/api/status | tee "$ARTIFACT_DIR/status-pre.json"
curl -fsS http://127.0.0.1:8444/api/cabinet-profile | tee "$ARTIFACT_DIR/cabinet-profile-pre.json"
curl -fsS http://127.0.0.1:8444/api/certificates | tee "$ARTIFACT_DIR/certificates-pre.json"

openssl x509 -in /etc/g2s-mute/certs/host.crt -noout -subject -issuer -dates -ext subjectAltName \
  | tee "$ARTIFACT_DIR/host-cert-summary.txt"
openssl x509 -in /etc/g2s-mute/certs/host.crt -noout -fingerprint -sha256 \
  | tee "$ARTIFACT_DIR/host-cert-fingerprint.txt"
openssl x509 -in /etc/g2s-mute/certs/client.crt -noout -fingerprint -sha256 \
  | tee "$ARTIFACT_DIR/client-cert-fingerprint.txt"

# Optional: only if identity correction is required and approved.
curl -fsS -X PUT http://127.0.0.1:8444/api/cabinet-profile \
  -H 'Content-Type: application/json' \
  --data '{"wire_host_url":"https://override-host.example:9443/g2s","listener_dns_name":"override-host.example","listener_ip":"192.168.50.41","required_san_dns":["override-host.example"],"required_san_ips":["192.168.50.41"],"host_id":"HOST-OVERRIDE-001","first_test_egm_ids":["EGM-99"]}' \
  | tee "$ARTIFACT_DIR/cabinet-profile-put.json"
curl -fsS http://127.0.0.1:8444/api/status | tee "$ARTIFACT_DIR/status-post-put.json"
curl -fsS http://127.0.0.1:8444/api/cabinet-profile | tee "$ARTIFACT_DIR/cabinet-profile-post-put.json"

# Revert temporary override unless approved for production use.
curl -fsS -X DELETE http://127.0.0.1:8444/api/cabinet-profile | tee "$ARTIFACT_DIR/cabinet-profile-delete.json"
curl -fsS http://127.0.0.1:8444/api/status | tee "$ARTIFACT_DIR/status-post-delete.json"
curl -fsS http://127.0.0.1:8444/api/cabinet-profile | tee "$ARTIFACT_DIR/cabinet-profile-post-delete.json"

journalctl -u g2s-mute.service -n 300 --no-pager | tee "$ARTIFACT_DIR/journal-post.txt"
```

## E) Evidence Capture Checklist

- [ ] `readyz` snapshot with HTTP status and body.
- [ ] `api/status` pre-session snapshot.
- [ ] `api/cabinet-profile` pre-session snapshot.
- [ ] `api/certificates` snapshot.
- [ ] Host and client certificate SHA-256 fingerprints.
- [ ] Host certificate SAN dump (`subjectAltName` output).
- [ ] `systemctl` pre-session service status.
- [ ] `journalctl` pre/post session logs with timestamps.
- [ ] If override used: PUT response, post-PUT status/profile, DELETE response, post-DELETE status/profile.
- [ ] Registered-host UI evidence (screenshot/export).
- [ ] Raw `commsOnLine`/ACK evidence in logs or packet capture transcript.

## F) Failure Triage Matrix

| Symptom | Likely Cause | First Checks | Immediate Action |
| --- | --- | --- | --- |
| TLS handshake failure | SAN mismatch or trust chain issue | Compare `wire_host_url`, SAN dump, CA chain | Stop session; regenerate/install matching certs |
| Host registration rejected | Missing/incorrect registered-host role | EGM host list, role, permission class | Stop session; fix registration before retry |
| Endpoint mismatch / 404 | Wrong path or URL in EGM | `wire_host_url`, `/g2s` path, EGM config screen | Correct endpoint and retry smoke checks |
| Session timeout / keepalive drop | Network instability or liveness mismatch | VLAN ACLs, keepalive logs, `journalctl` timestamps | Hold control actions; stabilize session first |
| No ACK after `commsOnLine` | Identity mismatch or protocol gating | `host_id`, `egm_id`, registration status | Stop; capture transcript and vendor escalation |
| `profile_source` unexpected | Stale DB override row | `GET /api/cabinet-profile`, `override_present` | Use DELETE override, re-check source `file` |
| `readyz` not ready | Service degraded or blocking issue | `GET /readyz`, `/api/status.readiness.issues` | No-Go until issues clear |

## G) Go/No-Go Gate With Explicit Stop Conditions

Go only if all are true:

- [ ] `readyz` returns `200` and readiness is `READY` or `READY_LAB`.
- [ ] `profile_source` is explicitly expected for this run and documented.
- [ ] Effective `cabinet_profile` exactly matches approved session identity sheet.
- [ ] Certificate SAN matches on-wire DNS/IP.
- [ ] Registered-host role/permissions are confirmed on the EGM.
- [ ] Evidence capture path is active and writable.
- [ ] `journalctl` shows no unresolved startup/security errors.

Immediate No-Go stop conditions:

1. Any mismatch between approved identity sheet and effective `cabinet_profile`.
2. Any TLS chain/SAN verification failure.
3. Any `readyz` non-200 response or non-ready overall state.
4. Missing registered-host permission for required session startup classes.
5. Any unexplained session timeout or ACK loss during startup.

## H) Post-Session Closeout Checklist And Artifacts To Archive

- [ ] Final `api/status` and `api/cabinet-profile` snapshots stored.
- [ ] Final `systemctl` and `journalctl` captures stored.
- [ ] Any temporary override removed (`profile_source=file` confirmed unless override intentionally retained).
- [ ] Session summary filled: pass/fail by gate item, timestamped operator notes, escalation items.
- [ ] Artifact bundle archived under run ID and linked from `docs/pi-runs/`.
- [ ] Follow-up tickets created for unresolved blockers (identity, cert, registration, protocol).

