# Phase 3A: Field-Test Operator Configuration Shell

> Correction note: Field-test is the milestone name only. The runtime product surface is the Operator Console and must not use field-test as product identity.

Runtime product navigation must remain product/operator language only:

- Live
- Inputs
- Actions
- Comms
- EGMs
- Templates
- Audit
- Settings

System Check and Evidence Export are operational views, not top-level project-management modules.

## Scope

Phase 3A adds a minimal operator-facing Operator Console shell under `/operator` to review and configure the backend spine that already exists.

This phase advances these field-test must-have sections:

- Electrical Input Monitor
- Action Builder Lite
- EGM Registry
- Template Manager Lite
- G2S Comms Journal
- Emergency Audit Timeline
- Network/Cert Settings

## Routes / Pages

- `GET /operator`
- `GET /operator/inputs`
- `GET /operator/actions`
- `GET /operator/egms`
- `GET /operator/templates`
- `GET /operator/comms`
- `GET /operator/audit`
- `GET /operator/settings`
- `GET /operator/settings/system-check`
- `GET /operator/settings/system-check.json`

Minimal mutation routes (server-rendered forms):

- `POST /operator/inputs/{id}`
- `POST /operator/actions`
- `POST /operator/actions/{id}`
- `POST /operator/egms`
- `POST /operator/egms/{id}`
- `POST /operator/templates`
- `POST /operator/templates/{id}`
- `POST /operator/templates/{id}/versions`
- `POST /operator/templates/{id}/active-version`
- `POST /operator/templates/render-preview` (render-only, no network send)

## Mutation / Auth Rules

- All configuration mutations in `/operator` must use existing mutation auth.
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

- Separate `internal/operatorui` package exists and is routed.
- `/operator` pages expose operator review/config surfaces for inputs/actions/EGMs/templates/comms/audit/settings.
- Minimal mutation forms exist for inputs/actions/EGMs/templates with auth enforcement.
- Real-send safety state remains gated/disabled by current transport gate behavior.
- No legacy UI bundle edits are required.
- `go test ./...` passes.

## Next Expected Phase

Phase 3B: targeted operator workflow refinements (validation UX, export improvements, and controlled field-test ergonomics) without changing send safety guardrails.
