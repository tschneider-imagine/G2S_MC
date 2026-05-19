# First Cabinet Prep Checklist

Purpose:

Freeze the first real-cabinet integration assumptions before booking cabinet time, while clearly separating what is already known from what must be confirmed.

Related docs:

- [G2S First Cabinet Lab Runbook](./G2S_First_Cabinet_Lab_Runbook.md)
- [G2S Pre-Next-Step Readiness Check](./G2S_Pre_NextStep_Readiness_Check.md)
- [Raspberry Pi Bring-Up](./raspberry-pi.md)

## A) Frozen Identity Fields

| Field | Current Value (Frozen Today) | To Confirm Before First Cabinet Session |
| --- | --- | --- |
| `host_id` | `HOST-PI-001` (Pi lab config) | Final production/lab host ID approved by vendor/site (`HOST-CAB-###` pattern or site standard). |
| Host URL used on wire | Lab defaults: `http://127.0.0.1:8444/g2s` and TLS example `https://localhost:8444/g2s` | Final cabinet-facing URL reachable from EGM VLAN, likely `https://<cabinet-reachable-dns-or-ip>:8444/g2s`. |
| Endpoint path | `/g2s` | Confirm vendor requires same path and no alternate SOAP endpoint path. |
| DNS/IP expected by EGM | Lab uses loopback (`127.0.0.1` / `localhost`) | Real DNS name or static IP the EGM will dial on the wire. |
| SAN values required in host cert | Must include exact on-wire identity (DNS and/or IP) | Final SAN set including cabinet-facing DNS name and/or IP; no loopback-only SAN in cabinet test cert. |
| EGM IDs for first test set | `EGM-01` | Final test set of real cabinet IDs (minimum one primary + one backup cabinet). |

## B) Certificate Plan

### Current Lab Certificate Flow (Already In Use)

1. Generate local dev certs:
   `sudo g2s-dev-certs -out /etc/g2s-mute/certs`
2. Set service ownership:
   `sudo chown -R g2s-mute:g2s-mute /etc/g2s-mute/certs`
3. Use certs in config (`g2s.require_tls=true`, `g2s.require_client_cert=true`) for TLS+mTLS lab validation.

### Production / Source-of-Truth Cert Ownership

- Owner (to confirm): site PKI/security owner and/or manufacturer-provided PKI process.
- Dependency: certificate issuance workflow (manual CSR vs internal CA vs vendor process) must be approved before cabinet day.

### Required Files And Install Paths On Pi

| File | Path | Required For |
| --- | --- | --- |
| CA certificate | `/etc/g2s-mute/certs/ca.crt` | Server trust validation and client-cert chain validation |
| Host certificate | `/etc/g2s-mute/certs/host.crt` | HTTPS listener identity |
| Host private key | `/etc/g2s-mute/certs/host.key` | HTTPS listener key material |
| Client certificate | `/etc/g2s-mute/certs/client.crt` | EGM mTLS client authentication |
| Client private key | `/etc/g2s-mute/certs/client.key` | EGM mTLS client key material |

### Rotation / Expiry Checks

- Capture `notBefore`/`notAfter` and SHA-256 fingerprints for all installed certs before session.
- Confirm expiry window is valid for planned session plus contingency.
- Verify `/api/certificates` reports expected states and no blocking certificate issues.

## C) Network + Registration Prerequisites

### Firewall / Routing

- Confirm EGM network path to Pi listener on `8444/tcp`.
- Confirm reverse path (if required by site controls) and any ACL rules.
- Confirm no NAT/proxy/TLS interception alters certificate identity.

### Registered-Host Role / Permissions On EGM

- Confirm host is present in EGM registered-host list.
- Confirm required role (`owner` / `configurator` or vendor equivalent) for first control and descriptor/option access.
- Confirm class/device permissions include communications startup and required discovery/control classes.

### Site Approval Dependencies

- Change window approved for cabinet test.
- Vendor support contact and escalation path available during session.
- Site operational approval for test commands and restore path.

## D) Endpoint / Payload Contract Capture

Capture these exact examples in first session evidence:

1. `commsOnLine` request from EGM to host and host response.
2. `commsOnLineAck` exchange details.
3. `keepAlive` request/response sequence.
4. `getDescriptor` request and `descriptorList` response.
5. First option/discovery request and response (`optionList` / `getOptionSeries` or vendor equivalent).

First-session success criteria:

- TLS identity validates with expected SAN for on-wire DNS/IP.
- Registered-host role is accepted by EGM.
- `commsOnLine` and keepalive return successful HTTP/G2S responses.
- Descriptor/discovery payloads are captured and parseable.
- No unresolved protocol authorization or permission errors for intended next control step.

## E) First-Session Evidence Checklist

- [ ] TLS handshake proof (openssl/curl trace with certificate chain details).
- [ ] Server cert fingerprint and client cert fingerprint captured.
- [ ] `commsOnLine` request/response transcript.
- [ ] `keepAlive` request/response transcript.
- [ ] Descriptor/option outputs captured and archived.
- [ ] Registered-host role/permission screen export or photo evidence.
- [ ] Per-EGM outcomes logged (success/failure/reason/timestamp).
- [ ] Any rejection/deny payload captured verbatim.
- [ ] Restore/recovery command path documented (if any control is attempted).
- [ ] End-of-session summary with next actions and blockers.

## F) Go / No-Go Gate (10 Items)

All items must be **YES** before first real cabinet attempt.

1. [ ] YES / NO: Final host URL and endpoint path are frozen and match EGM config.
2. [ ] YES / NO: Certificate SAN includes exact DNS/IP used by the EGM on wire.
3. [ ] YES / NO: Certificate files are installed at expected Pi paths with correct ownership.
4. [ ] YES / NO: TLS handshake is validated using site-intended trust model.
5. [ ] YES / NO: EGM registered-host entry exists and is enabled.
6. [ ] YES / NO: Required role/permission set is confirmed for target classes.
7. [ ] YES / NO: Routing/firewall path from cabinet network to Pi listener is open.
8. [ ] YES / NO: At least one real EGM ID is confirmed and mapped to test plan.
9. [ ] YES / NO: Evidence capture method is ready (transcripts, fingerprints, screenshots).
10. [ ] YES / NO: Site/vendor approval window and rollback/restore plan are in place.

