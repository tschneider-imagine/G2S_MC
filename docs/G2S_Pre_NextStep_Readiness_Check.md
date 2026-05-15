# G2S Pre-Next-Step Readiness Check

Date checked: 2026-05-15

Purpose:

This is the missing-info and double-check list before starting implementation. It separates what we can build immediately from what must be confirmed before real cabinet testing.

## 1. Local Project Check

Files reviewed:

- `docs/G2S_Complete_Executable_Project_Plan.md`
- `docs/G2S_First_Cabinet_Lab_Runbook.md`

Local environment findings:

- `go` is not currently available in `PATH` on this machine.
- `openssl` is not currently available in `PATH` on this machine.
- `C:\Users\SnowM\Documents\GitHub\G2S_MC` is a Git repository and is the correct place to start implementation.

Impact:

- We can keep planning and editing docs now.
- Before coding the Go service locally, install Go or choose a dev environment where Go is available.
- Before certificate bring-up testing, install OpenSSL or use an equivalent TLS inspection tool.
- Code and docs should be created in this repo so changes are tracked from the start.

## 2. Public Source Double-Check

Confirmed from current public sources:

- IGSA lists G2S and XTP as land-based standards, and the standards page says IGSA standards are available for members only. This means the normative G2S package and XTP transport package still need to be obtained through the proper IGSA/member channel.
- The IGSA G2S committee scope is protocol messaging and implementation/certification documentation. The public charter explicitly excludes transport-specific means and methods from the G2S committee scope, so transport details must be checked in XTP/transport material.
- The IGSA Quick Start guide confirms the starter flow and command set we planned around: `commsOnLine`, `commsOnlineAck`, `getDescriptor`, `descriptorList`, `setKeepAlive`, `keepAlive`, `getCabinetStatus`, and `setCabinetState`.
- The Quick Start guide also confirms the G2S stack shape: HTTP 1.1, SOAP 1.1, WSDL, XSD/JAXB-style schema handling, dispatching, and schema/WSDL files in the Quick Start ZIP.
- RadBlue's RGS/RST guide confirms a key architecture correction: the EGM initiates `commsOnline` by connecting to the host web server, then the host uses the descriptor exchange to learn classes, devices, and permissions.
- IGSA's regulator guide confirms required/core classes include Communications, Cabinet, Events, Meters, Game Play, CommConfig, and OptionConfig. That supports keeping `OPTIONCONFIG` and `CABINET` central in the control-mapping work.
- RadBlue CVT material confirms `optionConfig` is one of the G2S classes covered by CVT test scripts, and RGS/RPA are useful tools for crafting messages and capturing transcripts.
- Public RadBlue security material supports SSL/TLS, valid certificates, SCEP, OCSP, and each EGM having a list of registered hosts.
- IGSA technical update material supports certificate identity/role checking: CN should carry an application-level identifier such as EGM ID or host ID, and OU should carry the role such as `G2S_egm` or `G2S_host`.
- IGSA's April 2026 newsletter says the Transport Committee is being looked at again because SOAP is old/deprecated and G2S/S2S transport updates are being considered. This does not block the MVP, but it means we should not overfit the code to one transport implementation.

## 3. Architecture Corrections Before Coding

Make these corrections part of the implementation:

- The Pi controller must expose a G2S host listener URL, not only send outbound POSTs.
- The fake EGM harness should initiate `commsOnLine` into the host listener.
- The G2S package should model both host listener handling and outbound host-to-EGM requests.
- The control workflow should be discovery-first: session startup, descriptor exchange, option/class discovery, permission check, then control attempt.
- The first real audio control should not be named `mute` internally until the actual class/option/cabinet-state mapping is known.

## 4. Must-Have Before Coding

These are needed before efficient local implementation:

- Go installed and available in `PATH`, or a chosen Go-capable dev environment.
- OpenSSL or equivalent TLS/certificate inspection tooling.
- Decision on whether we build on Windows first, WSL, a Raspberry Pi, or a container/dev VM.
- Initial Go package layout approval from `docs/G2S_Complete_Executable_Project_Plan.md`.
- SQLite driver choice for Go. Fast recommendation: use a pure-Go driver first to avoid CGO friction on Windows and Raspberry Pi.
- XML/SOAP strategy for the fake harness. Fast recommendation: start with `encoding/xml` fixtures and interface boundaries, then replace with generated/schema-aware bindings once the official WSDL/XSD package is available.
- Exact XML command spelling/case from WSDL/XSD. Public docs use both `commsOnLine` and `commsOnline` styling, so generated code and fixtures should follow the official schema once obtained.

## 5. Must-Have Before First Real Cabinet Test

These items should be gathered before booking cabinet time:

- IGSA G2S package, XTP/Point-to-Point SOAP/HTTPS transport package, and Quick Start ZIP with WSDL/XSD.
- Vendor/platform G2S version and supported class matrix.
- Exact `hostId`, `egmId`, host URL, EGM URL, host port, and transport directionality.
- EGM registered-host setup process, required role, and device/class permissions.
- Certificate source: self-signed lab, private CA, production CA, manual CSR, or SCEP.
- Certificate profile: CN, OU, SAN DNS/IP, EKU, key algorithm, key size, validity period, and revocation policy.
- Network route and firewall rules from EGM network to the Pi host listener.
- Cabinet roster: cabinet ID, IP, port, vendor, cabinet family, game title, firmware/software version, bank/zone.
- Descriptor and option discovery outputs from the test cabinet.
- Vendor confirmation of whether audio/master volume/jurisdictional audio-disable maps to `OPTIONCONFIG`, `CABINET`, or a vendor extension.
- Restore/recovery workflow and completion evidence.
- Permission to perform a real mute/restore test and the operational state required for testing.

## 6. Can Wait Until After The First Software Slice

These should not block the first vertical slice:

- OCSP hard-fail behavior.
- SCEP automation.
- Full title compatibility database.
- Production enclosure and DIN-rail packaging.
- UPS/battery adapter if the interface is not yet selected.
- Player-facing PUI, mediaDisplay, picture-in-picture, watermark, or marketing display features.
- Full compliance/certification test pass.

## 7. Vendor / Standards Request Packet

Ask the manufacturer or standards owner for:

- G2S and XTP version used by the platform.
- WSDL/XSD/schema bundle.
- Endpoint paths and SOAP action requirements.
- Startup sequence expectations and timeout values.
- `commConfig` registered-host process.
- Required host role and permissions for option/configuration changes.
- `descriptorList` sample for the target platform.
- `optionList`, `getOptionSeries`, or equivalent option discovery output.
- Exact option or cabinet-state mapping for audio, master volume, and jurisdictional audio-disable.
- Allowed machine states and jurisdictional preconditions for the change.
- Completion evidence: status, event, option value, cabinet state, or audit log.
- Certificate profile requirements and trust/enrollment process.
- Revocation policy: none, OCSP, CRL, soft-fail, or hard-fail.

## 8. Recommended Next Action

Set up the dev foundation before writing application code:

1. Install or select a Go-capable dev environment.
2. Install OpenSSL or choose equivalent certificate tooling.
3. Start the MVP with a G2S host listener plus fake EGM `commsOnLine` flow.

That keeps the first implementation aligned with the real protocol shape while still letting us move fast without cabinet access.

## 9. Public Links Checked

- IGSA Standards: https://igsa.org/standards/
- IGSA G2S Committee: https://igsa.org/committees/g2s-game-to-system-committee/
- IGSA Quick Start for G2S Implementation Guide: https://igsa.org/wp-content/uploads/2025/02/article-2020-05-quick-start-for-g2s-implementation-guide.pdf
- IGSA Regulatory Guide to IGSA Standards: https://igsa.org/wp-content/uploads/2025/02/guide-2020-09-regulators-guide-to-igsa-standards.pdf
- IGSA April 2026 Newsletter: https://igsa.org/igsa-newsletter-the-voice-of-standards-for-the-gaming-industry-april-2026/
- RadBlue CVT: https://www.radblue.com/products/cvt/
- RadBlue RGS/RST/RPA Quick Start: https://www.radblue.com/docs/quickStart_Guide_r2.pdf
- RadBlue RPA: https://www.radblue.com/products/rpa/
- OpenG2S Architecture: https://openg2s.sourceforge.net/architecture.html
