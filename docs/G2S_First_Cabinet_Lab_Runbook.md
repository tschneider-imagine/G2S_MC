# G2S First Cabinet Lab Runbook

Purpose:

Use this runbook for the first real cabinet session. It is designed to isolate failures in the right order: identity, TLS, registered-host permission, G2S session startup, discovery, then audio control.

## 1. Identity Freeze

Fill this before generating certificates or configuring the EGM.

| Field | Value |
| --- | --- |
| Run ID |  |
| Date |  |
| Operator |  |
| EGM vendor/model |  |
| Cabinet / platform family |  |
| Game title |  |
| Firmware / software version |  |
| Host system name |  |
| `hostId` |  |
| `egmId` |  |
| Host URL configured in EGM |  |
| Host IP |  |
| Host port |  |
| EGM URL, if host connects outbound |  |
| TLS identity used on wire: DNS or IP |  |
| Certificate SAN value matching that identity |  |
| Required registered-host role | owner / configurator / guest / other |

Go/no-go:

- Do not generate certificates until the host URL and SAN value match the exact DNS name or IP that will be used on the wire.

## 2. Certificate And Trust Setup

| Item | Value / Result |
| --- | --- |
| CA type | self-signed lab / private CA / production CA |
| Root CA file |  |
| Intermediate CA file |  |
| Host certificate file |  |
| Host private key file |  |
| Host certificate EKU | serverAuth / clientAuth / both |
| EGM client certificate file, if mutual TLS |  |
| EGM private key file, if mutual TLS |  |
| OCSP enabled | no for first pass / yes after TLS is proven |
| SCEP used | no for first pass / yes |

First-pass rule:

- Prove one-way TLS first.
- Add mutual TLS second.
- Add OCSP last.

## 3. TLS Proof Before G2S

Run these checks before debugging SOAP or XML.

```bash
openssl x509 -in <host-cert.pem> -noout -subject -issuer -dates -ext subjectAltName
openssl verify -CAfile <ca-bundle.pem> <host-cert.pem>
openssl s_client -connect <host>:<port> -servername <dns-name> -CAfile <ca-bundle.pem> -verify_hostname <dns-name>
```

If testing by IP:

```bash
openssl s_client -connect <ip>:<port> -CAfile <ca-bundle.pem> -verify_ip <ip>
```

For mutual TLS:

```bash
openssl s_client -connect <host>:<port> -servername <dns-name> -CAfile <ca-bundle.pem> -cert <client-cert.pem> -key <client-key.pem> -verify_hostname <dns-name>
```

Record:

| Check | Pass / Fail | Notes |
| --- | --- | --- |
| TCP port reachable |  |  |
| Certificate chain validates |  |  |
| SAN matches DNS/IP used |  |  |
| Host presents expected cert |  |  |
| Client certificate required intentionally |  |  |
| Mutual TLS handshake succeeds, if enabled |  |  |

## 4. EGM Registered-Host Setup

Record the EGM communications configuration.

| Item | Value / Result |
| --- | --- |
| Host appears in registered-host list |  |
| Host URL exactly matches planned URL |  |
| Host role |  |
| Device/class permissions |  |
| Any local allowlist or trust setting |  |
| Screenshot/export saved |  |

Go/no-go:

- Do not attempt audio control until the host is registered and has the required class/device permissions.

## 5. G2S Session Startup

Prove session lifecycle before attempting control.

| Step | Pass / Fail | Evidence File / Notes |
| --- | --- | --- |
| `commsOnLine` observed |  |  |
| `commsOnLineAck` sent/received |  |  |
| `hostId` matches expected value |  |  |
| `egmId` matches expected value |  |  |
| `sessionId` correlation works |  |  |
| `g2sAck` behavior observed |  |  |
| `getDescriptor` / `descriptorList` complete |  |  |
| keepalive configured |  |  |
| keepalive stable for target window |  |  |

Minimum target:

- Keepalive stable for at least 10 minutes before the first control attempt.

## 6. Discovery Before Control

Capture these outputs before choosing a mute/audio workflow.

| Evidence | Captured | Notes |
| --- | --- | --- |
| `descriptorList` |  |  |
| class support matrix |  |  |
| option list or option catalog |  |  |
| `getOptionSeries` or equivalent |  |  |
| `commHostList` / registered-host output |  |  |
| vendor extension namespaces |  |  |
| current cabinet status |  |  |
| current option values |  |  |

Expected investigation order:

1. `OPTIONCONFIG` for game audio, master volume, or jurisdictional audio-disable.
2. `CABINET` if the control is part of a broader machine state or regulatory operating mode.
3. `MEDIADISPLAY`, PUI, or EMDI only if the vendor confirms the audio is hosted-content or media-window audio.

## 7. Control Attempt

Do not treat ACK alone as completion. Record final state evidence.

| Field | Value |
| --- | --- |
| Control class used | OPTIONCONFIG / CABINET / vendor extension / other |
| Option identifier or cabinet state |  |
| Payload fixture or transcript file |  |
| Request timestamp |  |
| ACK received | yes / no |
| Completion evidence | status / option value / event / log / observed device state |
| Result | success / denied / unsupported / timeout / other |
| Restore/recovery command used |  |
| Restore completion evidence |  |

## 8. Failure Buckets

| Symptom | Likely Area | First Checks |
| --- | --- | --- |
| TLS fails before HTTP | certificate trust | CA chain, key pair, client-cert requirement, port |
| TLS works but identity fails | SAN mismatch | DNS/IP used on wire, certificate SAN |
| TLS works but G2S never starts | registration/session | host URL, registered-host role, `hostId`, `egmId`, endpoint path |
| Session starts then drops | liveness | ACK behavior, keepalive timing, `sessionId`, OCSP if enabled |
| Control ACKs but does not complete | class workflow | status query, option value, completion event, permission/state gating |
| Control denied | authorization | owner/configurator/guest role, class/device permission, jurisdiction state |

## 9. Evidence To Save

- certificate fingerprints
- full server certificate chain
- OpenSSL output
- EGM registered-host screenshot/export
- raw first HTTPS request/response
- raw `commsOnLine` and `commsOnLineAck`
- descriptor exchange transcript
- option discovery transcript
- keepalive transcript
- control request and response
- completion evidence
- restore request and response
- final pass/fail notes

## 10. Final Result

| Field | Value |
| --- | --- |
| Cabinet control path found | yes / no |
| Best class owner | OPTIONCONFIG / CABINET / vendor extension / unknown |
| Ready-now title | yes / no |
| Requires firmware/software update | yes / no |
| Requires manufacturer confirmation | yes / no |
| Next action |  |
