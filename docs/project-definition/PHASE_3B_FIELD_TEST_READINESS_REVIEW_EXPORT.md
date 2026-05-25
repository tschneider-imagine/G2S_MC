# Phase 3B: Field-Test Readiness Review + Export

> Correction note: Field-test is the milestone name only. The runtime product surface is the Operator Console and must not use field-test as product identity.
>
> Runtime scope correction: Readiness pages/endpoints are superseded and removed from runtime scope. System Check is not an approved replacement.
>
> Runtime scope correction: legacy dashboard routes are removed from active serving. Fake EGM tooling is outside product runtime scope.

## Scope

Phase 3B now records export-focused runtime scope only.

This phase advances these field-test must-have sections:

- Action Builder Lite
- Template Manager Lite
- G2S Comms Journal
- Emergency Audit Timeline
- Network/Cert Settings

## Runtime Scope

- Runtime navigation is limited to: `Live | Inputs | Actions | Comms | EGMs | Templates | Audit | Settings`.
- Readiness runtime pages and JSON endpoints are removed.
- System Check runtime pages and JSON endpoints are not approved.
- Runtime UI must not show project/test language or "Gate" wording.
- Legacy dashboard routes are not served.

## Export Behavior

Phase 3B approved runtime exports:

- `GET /operator/comms/export`: message journal JSON export.
- `GET /operator/audit/export`: audit timeline JSON export.
- Visible seeded runtime records must use product-neutral names (`EGM-001`, `EGM-002`, `template-generic-g2s-action`).

Export data is read-only evidence and must not include private key material.

## Intentionally Not Included

- no real EGM send capability expansion,
- no LAN capture allowlist,
- no retry/escalation execution engine changes,
- no Readiness runtime UI/API/export,
- no System Check runtime UI/API/export,
- no fake EGM tooling or simulator behavior,
- no UI chart/dashboard polish work,
- no action run success marking changes,
- no project-definition markdown embedded as runtime page content.

## Definition Of Done

- `/operator/comms/export` and `/operator/audit/export` exist.
- Action Builder Lite pages show retry/escalation/return configuration fields.
- Template Manager Lite pages show render preview and matcher placeholder fields.
- Runtime sending remains disabled with no new transport behavior.
- `go test ./...` passes.

## Next Expected Phase

Phase 3C: focused operator workflow polish and field-test ergonomics improvements while preserving all send-safety guardrails.
