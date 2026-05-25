# Phase 3A: Field-Test Operator Configuration Shell

## Scope

Phase 3A adds a minimal operator-facing field-test shell under `/field-test` to review and configure the backend spine that already exists.

This phase advances these field-test must-have sections:

- Electrical Input Monitor
- Action Builder Lite
- EGM Registry
- Template Manager Lite
- G2S Comms Journal
- Emergency Audit Timeline
- Network/Cert Settings

## Routes / Pages

- `GET /field-test`
- `GET /field-test/inputs`
- `GET /field-test/actions`
- `GET /field-test/egms`
- `GET /field-test/templates`
- `GET /field-test/comms`
- `GET /field-test/audit`
- `GET /field-test/settings`

Minimal mutation routes (server-rendered forms):

- `POST /field-test/inputs/{id}`
- `POST /field-test/actions`
- `POST /field-test/actions/{id}`
- `POST /field-test/egms`
- `POST /field-test/egms/{id}`
- `POST /field-test/templates`
- `POST /field-test/templates/{id}`
- `POST /field-test/templates/{id}/versions`
- `POST /field-test/templates/{id}/active-version`
- `POST /field-test/templates/render-preview` (render-only, no network send)

## Mutation / Auth Rules

- All configuration mutations in `/field-test` must use existing mutation auth.
- Unauthorized mutation requests fail with `401`.
- Read-only pages do not require mutation auth.

## Intentionally Not Included

- no real G2S send expansion,
- no LAN allowlist or transport capability broadening,
- no retry/escalation execution changes,
- no fake EGM tooling or simulated floor behavior,
- no new charts/dashboard polish work,
- no project docs embedded as runtime product content.

## Definition Of Done

- Separate `internal/fieldtestui` package exists and is routed.
- `/field-test` pages expose operator review/config surfaces for inputs/actions/EGMs/templates/comms/audit/settings.
- Minimal mutation forms exist for inputs/actions/EGMs/templates with auth enforcement.
- Real-send safety state remains gated/disabled by current transport gate behavior.
- No legacy UI bundle edits are required.
- `go test ./...` passes.

## Next Expected Phase

Phase 3B: targeted operator workflow refinements (validation UX, export improvements, and controlled field-test ergonomics) without changing send safety guardrails.

