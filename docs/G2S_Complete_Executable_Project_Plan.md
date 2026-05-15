# G2S Muting Controller Complete Executable Project Plan

Source set compared:

- `G2S_Raspberry_Pi_Project_Plan.md`
- `Ai-o1-G2S_mute_architecture_spec.md`
- `Ai-G1.md`

Additional source folder reviewed:

- `other info/G2S_XML_Format_Reference_Sheet.md`
- `other info/G2S_Session_Authentication_Guide.md`
- `other info/G2S_Certificate_Side_Guide.md`
- `other info/G2S_Certificate_Bringup_Checklist.md`
- `other info/g2s_compliance_framework.txt`
- `other info/G2S_Lab_Runbook_Template.md`
- `other info/post_2018_gaming_mfg_list.csv`

## 1. Executive Direction

Build a Raspberry Pi-based G2S muting controller as a small appliance:

- Raspberry Pi 5 or Pi-based industrial form factor
- Go service as the core runtime
- JSON configuration for site-specific values
- SQLite local audit ledger
- HTTPS/SOAP G2S host listener plus outbound G2S client/callback communication as required by the session
- GPIO or isolated I/O for security line, PSU, and buzzer signals
- Lightweight local web dashboard using Go templates and HTMX
- Systemd-managed deployment on Linux ARM64

The fastest executable path is to build a vertical slice first: config load, state machine, local event log, simulated trigger, fake EGM fan-out, and dashboard. Real GPIO, real certificates, and real cabinet payloads can then plug into stable interfaces without blocking the software team.

Important correction from the added material: do not assume there is one universal cross-vendor `mute` command. Treat the first real cabinet work as a G2S discovery and control-mapping task. The leading candidate for audio, master volume, and jurisdictional audio-disable behavior is `OPTIONCONFIG`, with `CABINET` kept in scope if the control is part of a broader machine or regulatory operating state.

## 2. Comparison Of The Three Source Sets

| Source | Strongest Contribution | Keep | Adjust For Fast Execution |
| --- | --- | --- | --- |
| `G2S_Raspberry_Pi_Project_Plan.md` | Business goal, Raspberry Pi baseline, deployment phases, slide traceability | Parallel integration with current SAS/progressive paths, mute/restore/status/history, compatibility matrix | Add software module detail, dependency map, and a sharper critical path |
| `Ai-o1-G2S_mute_architecture_spec.md` | Release architecture and module breakdown | Go, config-driven runtime, mTLS G2S client, SQLite ledger, state machine, web UI, per-EGM status model | Build simulator and fake EGM harness first so unknown G2S/hardware items do not stop progress |
| `Ai-G1.md` | Concrete config fields, async Go shape, GPIO events, PSU/buzzer behavior | 250 ms security line threshold, concurrent fan-out, GREY threshold buzzer escalation, HTMX dashboard idea | Replace sample global state mutation with engine-owned state and explicit interfaces |

Unified decisions:

- Core language: Go
- Target OS: Linux ARM64 on Raspberry Pi OS or another minimal Debian-style image
- Storage: SQLite local file database
- UI: embedded Go web server with server-rendered pages and HTMX polling
- Config: one local JSON file under `/etc/g2s-mute/config.json`
- Protocol: outbound HTTPS/mTLS G2S client to configured EGM endpoints
- Session direction: expose a host listener for EGM-initiated `commsOnLine`, then use outbound calls where the startup algorithm or control workflow requires host-to-EGM requests
- Runtime ownership: one controller engine owns mutable state; all other goroutines publish events
- Hardware design: GPIO and power inputs are behind interfaces so software can run before final wiring is locked

## 3. MVP Scope

The first complete build should include only the parts required to prove the muting controller path end to end.

In scope for MVP:

- config loader and validator
- EGM roster from config
- controller state machine
- simulated trigger source
- optional real GPIO security line adapter if wiring is ready
- incident creation in SQLite
- per-EGM fan-out result logging
- G2S client interface with fake payload support
- mTLS-capable HTTP transport
- G2S session startup model: `commsOnLine`, ACK, descriptor exchange, keepalive
- protocol discovery placeholders for descriptors, option catalog, and host permissions
- fake EGM test server
- dashboard showing controller state, EGM statuses, incidents, and certificate health
- systemd service file
- deployment notes for Raspberry Pi setup

Out of scope for MVP:

- player-facing features
- picture-in-picture, watermark, PUI, or vendor extension control
- broad machine health scraping
- final casino deployment package
- fully automated title compatibility database
- battery critical handling unless the UPS interface is known immediately
- production OCSP/SCEP automation unless required for the first cabinet test

## 4. Critical Dependency Map

| Dependency | Blocks | Fast Path | Final Requirement |
| --- | --- | --- | --- |
| IGSA G2S and XTP source packages | Normative XML, WSDL/XSD, SOAP transport rules | Use public Quick Start, fixtures, and fake EGM tests | IGSA G2S Message Protocol package, XTP/SOAP HTTPS package, and Quick Start ZIP schema artifacts |
| Exact audio control mapping | Real cabinet mute/restore | Build discovery interfaces and fixture-based `OPTIONCONFIG`/`CABINET` workflows | Vendor-confirmed class, option identifier, command sequence, state preconditions, and completion evidence |
| Exact G2S XML/SOAP payloads | Real cabinet control | Create payload builder interfaces and fixture XML for fake EGM tests | Vendor-validated XML, namespace, SOAP action, endpoint path, and response parser |
| Session startup and liveness rules | Reliable production session | Simulate `commsOnLine`, ACK, descriptor exchange, and keepalive in fake EGM | Real startup sequence, `hostId`, `egmId`, `sessionId`, `g2sAck`, keepalive timers, and descriptor flow |
| Host listener URL and firewall path | EGM can start the session | Use localhost fake EGM and host listener in dev | Stable host URL, DNS/IP, port, firewall rules, and EGM commConfig pointing at the Pi host |
| Registered host and permissions | Authorized control | Model host role in config and lab runbook | EGM commConfig host registration, owner/configurator/guest role, and class/device permissions |
| Endpoint identity and certificate SAN | TLS handshake and identity validation | Freeze lab host URL and generate dev certs with matching SAN | CA-issued host/EGM certs whose SAN matches the exact DNS name or IP used on the wire |
| Client certificate, key, and CA | Real mTLS cabinet testing | Generate dev CA and certs for fake EGM server | Site/client certificates with trusted CA chain, expiry tracking, and optional OCSP |
| EGM roster and network addresses | Real fan-out and compatibility tracking | Use sample config, fake local endpoints, and manufacturer CSV as compatibility seed data | Cabinet ID, IP, port, bank/zone, vendor, cabinet family, title, firmware, and expected control path |
| Security line wiring and polarity | Real trigger input | Start with simulated trigger and configurable active-low setting | Opto-isolated input wiring, pin number, voltage level, debounce confirmation |
| PSU input wiring | Warning state and buzzer behavior | Stub adapter until pins are available | Two confirmed input pins and polarity rules |
| UPS or battery interface | Battery critical emergency | Leave adapter stub in MVP | NUT, USB, serial, I2C, GPIO, or vendor-specific integration |
| Buzzer output hardware | Audible alert | Log alert pattern in software until hardware is ready | Confirm pin, voltage/current driver, acknowledgement behavior |
| SQLite driver choice | Build and deployment | Use the pinned pure-Go `modernc.org/sqlite` driver | Revisit only if field performance requires another driver |
| Web authentication policy | Operator dashboard release | Start with local login and config-controlled auth | Final decision on password, client cert, or both |
| Raspberry Pi storage and power | Field reliability | Use SSD and supervised shutdown for test unit | Industrial storage, protected power, UPS behavior, and recovery test |

Fast execution is realistic if the software team does not wait for all external dependencies before building. The only items that truly block real cabinet testing are the G2S control mapping, G2S payload/endpoint contract, certificates, registered-host permissions, roster, network access, and at least one test cabinet.

## 5. Target Architecture

```text
+------------------------+
| Web UI / API          |
| Dashboard + actions   |
+-----------+------------+
            |
            v
+------------------------+       +----------------------+
| Controller Engine      |<----->| SQLite Audit Ledger |
| State machine + rules  |       | Incidents + results |
+---+--------+-------+---+       +----------------------+
    |        |       |
    |        |       +----------> Buzzer / Alert Adapter
    |        |
    |        +------------------> G2S SOAP Listener + mTLS Client
    |
    +---------------------------> Hardware Input Adapters
                                  Security line, PSU, battery
```

Primary Go packages:

- `cmd/g2s-mute`: process entrypoint
- `internal/config`: JSON load, validation, checksum
- `internal/model`: states, events, EGM status, incidents
- `internal/engine`: state transitions and event handling
- `internal/io`: trigger, PSU, battery, and buzzer interfaces
- `internal/gpio`: Raspberry Pi GPIO implementation
- `internal/g2s`: HTTPS/SOAP listener, session startup, discovery, payload builder, mTLS client, fan-out, response classifier
- `internal/compliance`: class/option mapping, evidence capture, compatibility notes
- `internal/store`: SQLite connection, migrations, queries
- `internal/ui`: HTTP server, routes, auth, templates
- `internal/health`: heartbeat scheduler and subsystem checks
- `packaging/systemd`: service unit
- `configs`: example and development config
- `migrations`: SQL schema
- `docs`: operations and architecture notes

## 6. State Model

Core controller states:

- `BOOTING`: config, certs, database, and adapters are initializing
- `HEALTHY`: controller is ready and EGMs are within expected status
- `WARNING`: one PSU fault, partial EGM degradation, or non-critical issue
- `EMERGENCY_PENDING`: valid trigger received and incident is being opened
- `EMERGENCY_ACTIVE`: mute fan-out has been issued and results are visible
- `RECOVERY_PENDING`: restore/unmute sequence is in progress
- `DEGRADED`: required subsystem failed while the service is still running

Per-EGM states:

- `GREEN`: reachable and normal
- `YELLOW`: warning or missed heartbeat
- `RED`: mute accepted or emergency-muted
- `GREY`: unreachable, timeout, or transport failure

Important rule:

Only the controller engine mutates authoritative state. GPIO watchers, web handlers, heartbeat jobs, and G2S workers send typed events or results into the engine.

## 7. Protocol And Compliance Findings

Useful findings from `other info` that change the execution plan:

- G2S session authentication is certificate and endpoint-registration oriented, not a separate username/password XML login.
- A trustworthy session means transport identity succeeds, host/EGM identity matches, registered-host permissions are correct, `commsOnLine`/ACK succeeds, descriptors are exchanged, and keepalive remains healthy.
- ACK is not enough to prove a control completed. For configuration-style controls, completion must be proven through class-specific status, resulting device state, logs, or option-change evidence.
- Audio, master volume, and jurisdictional audio-disable should be investigated through `OPTIONCONFIG` first. `CABINET` remains a secondary path when the behavior is a machine-level regulatory state.
- `MEDIADISPLAY`, PUI, and EMDI are not primary for base game audio unless a vendor confirms the audio is part of hosted content or media-window behavior.
- The correct cabinet workflow is discovery-first: descriptor output, option output, host permissions, then control attempt.
- The controller must be ready to act as the host-side listener because public RadBlue startup flow shows the EGM initiates `commsOnLine` by connecting to the host URL.
- Certificate bring-up must freeze the exact host URL before generating certs. The DNS name or IP used on the wire must appear in the certificate SAN.
- OCSP should be enabled only after basic TLS, mutual TLS, host registration, and first G2S traffic are working.
- The manufacturer CSV has 113 rows and should seed the cabinet/title compatibility tracker, but it is not proof of support for the mute/audio control.

Real cabinet discovery evidence to capture:

- `descriptorList`
- option list or equivalent option catalog
- `getOptionSeries` or vendor equivalent
- `commHostList` / registered-host output
- class support and conformance matrix
- raw first request/response transcript
- `commsOnLine` / `commsOnLineAck`
- keepalive transcript
- option-change or cabinet-state completion evidence

## 8. Implementation Order

### Phase 0: Repo And Contracts

Goal:

Create the skeleton that lets multiple pieces be built in parallel.

Tasks:

- create Go module
- create package layout
- define model types for controller state, EGM status, events, incidents, and results
- define interfaces for trigger input, PSU input, battery input, buzzer output, payload builder, and EGM client
- define interfaces for G2S session startup, descriptor discovery, option discovery, and control completion evidence
- add `configs/config.example.json`
- add first README run instructions

Dependencies:

- no external hardware dependency
- no real G2S dependency

Exit criteria:

- project builds
- empty service starts and exits cleanly
- config example documents all required fields

### Phase 1: Config, Logging, And SQLite

Goal:

Make the appliance boot deterministically and fail clearly when config is wrong.

Tasks:

- implement JSON config loader
- validate controller ID, GPIO config, cert paths, database path, web bind address, and EGM roster
- log config checksum on boot
- open SQLite database
- run migrations
- create incident, per-EGM log, state history, cert inventory, and operator action tables

Dependencies:

- chosen SQLite driver
- local writable database path

Exit criteria:

- valid config boots
- missing roster, bad cert path, or invalid GPIO value creates a clear startup error
- migrations run idempotently

### Phase 2: Controller Engine With Simulated I/O

Goal:

Prove state transitions and emergency flow before touching real hardware.

Tasks:

- implement engine goroutine with event channel
- implement `BOOTING -> HEALTHY`
- implement simulated `SECURITY_LINE_DROP` after 250 ms
- implement incident creation on emergency
- implement GREY threshold calculation
- implement buzzer event publishing
- add unit tests for state transitions, debounce, and grey threshold

Dependencies:

- config package
- store package

Exit criteria:

- simulated trigger creates exactly one incident
- emergency state is visible in memory
- transient drops shorter than 250 ms are ignored
- grey threshold alert is deterministic

### Phase 3: Fake EGM Harness, Session Startup, And G2S Client

Goal:

Prove session startup, concurrent fan-out, and result logging without needing a real cabinet.

Tasks:

- implement fake EGM HTTP server
- implement G2S host listener endpoint for fake EGM `commsOnLine`
- implement payload builder with fixture XML
- implement fake `commsOnLine`, `commsOnLineAck`, descriptor exchange, and keepalive fixtures
- implement fixture descriptor/option discovery responses
- implement HTTP client with timeout
- implement per-EGM concurrent fan-out
- classify success, timeout, connection refused, and unexpected HTTP status
- persist one compliance row per EGM per incident

Dependencies:

- fixture payload
- fake endpoints from config

Exit criteria:

- fake bank receives parallel mute request
- fake session startup and keepalive can be exercised before control messages
- fake EGM can initiate `commsOnLine` into the host listener
- unreachable fake endpoint becomes `GREY`
- successful fake endpoint becomes `RED`
- logs are written per EGM

### Phase 4: mTLS Transport And Certificate Inventory

Goal:

Make the protocol path production-shaped while still testable locally.

Tasks:

- load client certificate, client key, and CA bundle
- build mTLS `http.Transport`
- add dev certificate generation instructions or script
- require SAN values to match the exact configured DNS name or IP used in lab tests
- record certificate subject, issuer, fingerprint, and expiry
- expose certificate health to engine and UI
- document one-way TLS first, mutual TLS second, OCSP last

Dependencies:

- dev certificates for fake EGM test
- real certificates for cabinet test
- frozen host URL, host ID, EGM ID, and endpoint identity values

Exit criteria:

- fake EGM server can require client cert
- client succeeds with valid cert
- client fails clearly with missing or invalid cert
- certificate expiry is visible in status

### Phase 5: Dashboard And Operator View

Goal:

Give the team and operators a visible system immediately.

Tasks:

- create embedded HTTP server
- add dashboard page
- add EGM status grid
- add incident history page
- add certificate status page
- add JSON or fragment endpoints for HTMX polling
- add manual test trigger in dev mode only
- log operator actions

Dependencies:

- engine read model
- store queries

Exit criteria:

- dashboard refreshes every 2 seconds
- active incident and EGM state changes are visible
- manual dev trigger writes an operator action

### Phase 6: Raspberry Pi Hardware Adapters

Goal:

Connect the same engine to real Pi I/O.

Tasks:

- implement GPIO input adapter for security line
- implement configurable active-low behavior
- implement PSU input watchers
- implement buzzer output adapter
- implement battery adapter only after interface is confirmed
- add hardware loopback test mode

Dependencies:

- confirmed pin map
- electrical isolation design
- hardware access on Raspberry Pi

Exit criteria:

- real security line drop of at least 250 ms creates emergency event
- short drop is ignored
- one PSU fault creates warning
- two PSU faults create emergency or configured critical event
- buzzer pattern can be tested safely

### Phase 7: Packaging And Pi Deployment

Goal:

Make the controller easy to install, restart, and diagnose.

Tasks:

- write `g2s-mute.service`
- define `/etc/g2s-mute` layout
- define `/var/lib/g2s-mute` database path
- define `/var/log/g2s-mute` or journald usage
- add install checklist
- add backup/export command for incident logs
- add health endpoint

Dependencies:

- final binary path
- final config path
- Pi OS image choice

Exit criteria:

- service starts on boot
- service restarts after failure
- logs show startup dependencies and current state
- database persists after restart

### Phase 8: Bench And Cabinet PoC

Goal:

Move from fake EGMs to real cabinet proof using discovery-first validation.

Tasks:

- load real cabinet roster
- install real certs
- configure real endpoint path and payload adapter
- expose the Pi host listener on the exact URL configured in EGM commConfig
- register the host in EGM commConfig with the required role and permissions
- prove TLS with OpenSSL before debugging SOAP or XML
- prove `commsOnLine`, descriptor exchange, and keepalive before attempting control
- capture descriptor, option, and permission outputs
- map the audio control through `OPTIONCONFIG` first, then `CABINET` if needed
- test one cabinet first
- test small bank
- record compatibility matrix
- document title, firmware, result, and observed response

Dependencies:

- real G2S payload
- real certificates
- registered-host permissions
- test cabinet access
- network routing
- operator permission to trigger mute

Exit criteria:

- one real cabinet receives mute and restore successfully
- control path is mapped to a discovered class/option or cabinet state, not guessed
- per-EGM result is logged
- status grid reflects real outcome
- compatibility matrix has evidence for each tested title

## 9. Quick Timeline

Assuming one software lead plus one hardware/support person, the first usable software slice can be built quickly because hardware and cabinet dependencies are isolated.

| Day | Target | Deliverable |
| --- | --- | --- |
| 1 | Repo skeleton, config, model contracts | Service boots with config validation |
| 2 | SQLite and engine | Simulated emergency creates incident |
| 3 | Fake EGM and fan-out | Parallel mute results logged per EGM |
| 4 | Session startup and mTLS | Fake `commsOnLine`, descriptor exchange, keepalive, and mTLS test pass |
| 5 | Dashboard | Live state, incidents, EGM grid, cert status |
| 6 | Pi adapters | Security line, PSU, and buzzer bench tests |
| 7 | Packaging | Systemd service and repeatable Pi install |
| 8+ | Real cabinet PoC | TLS, registered host, session startup, discovery, one-cabinet control, then small bank |

This timeline depends on having the Go environment ready and avoiding delays on real cabinet access. If G2S payloads or certs are late, the team can still complete Days 1 through 7 using fixture payloads and fake EGMs.

## 10. Test Plan

Unit tests:

- config validation
- state transition table
- 250 ms debounce behavior
- GREY threshold calculation
- certificate metadata parsing
- payload builder selection
- session ID and ACK correlation
- descriptor and option discovery parsing
- SQLite migration idempotency

Integration tests:

- simulated security trigger creates incident
- fake EGM success becomes `RED`
- fake EGM timeout becomes `GREY`
- fan-out writes one row per EGM
- mTLS succeeds with valid cert and fails with invalid cert
- certificate SAN mismatch fails in the expected way
- fake `commsOnLine` and keepalive run before control messages
- dashboard APIs return current engine read model

Hardware tests:

- GPIO security input active-low and active-high modes
- PSU 1 and PSU 2 warning behavior
- dual PSU emergency behavior if configured
- buzzer output pattern test
- service restart while inputs are normal
- service restart while a fault is active

PoC tests:

- one cabinet mute
- one cabinet restore
- registered-host permissions confirmed
- descriptor and option outputs captured
- unreachable cabinet handling
- mixed reachable/unreachable bank
- title-by-title compatibility capture
- power interruption and recovery

## 11. Definition Of Done For The Fast MVP

The MVP is complete when:

- Raspberry Pi service boots under `systemd`
- valid config loads and invalid config fails clearly
- simulated trigger can drive full emergency flow
- real or fake 250 ms security-line trigger can be tested
- incident is written once per emergency
- one compliance row is written per EGM target
- fake session startup and keepalive are modeled
- descriptor/option discovery placeholders exist
- concurrent fan-out handles success and timeout
- dashboard shows state, EGM statuses, incident history, and cert health
- operator actions are logged
- restart preserves audit history
- real cabinet PoC blockers are documented in the dependency register

## 12. Immediate Build Backlog

Start these in order:

1. Create Go module and package layout.
2. Add `configs/config.example.json` matching the agreed schema.
3. Add config loader and validation tests.
4. Add Raspberry Pi GPIO adapter once pinout is confirmed.
5. Convert `other info/post_2018_gaming_mfg_list.csv` into the first compatibility tracker seed if a tracker file is needed.
6. Keep real-cabinet readiness moving: host registration values, payload mapping, endpoint path, and production certificate source.

## 13. Decisions Needed Soon

These are the decisions most likely to slow execution if left open:

- whether first real control maps to `OPTIONCONFIG`, `CABINET`, or a vendor extension
- exact G2S control payload and restore/recovery payload
- endpoint path and response success criteria
- exact `hostId`, `egmId`, host URL, EGM URL, and certificate SAN values
- inbound firewall and routing from EGM network to the Pi host listener
- required registered-host role and device/class permissions
- real certificate source and deployment process
- cabinet roster format and network addressing
- security line voltage, isolation, pin, and polarity
- PSU input pin and polarity
- buzzer electrical driver and acknowledgement policy
- web admin authentication policy
- whether battery critical handling is required in the first PoC

## 14. Recommended First Technical Cut

The first technical cut should be a working software appliance with fake hardware and fake EGMs:

```text
config.example.json
-> service boot
-> simulated 250 ms emergency trigger
-> incident row created
-> fake EGM connects to host listener
-> fake comms session established
-> fake descriptor and option discovery returned
-> fixture G2S payload generated
-> fake EGM fan-out
-> per-EGM results logged
-> dashboard refresh shows state and results
```

That cut gives the team a useful demo, a stable architecture, and a path to plug in real hardware and real cabinets as soon as the dependencies arrive.
