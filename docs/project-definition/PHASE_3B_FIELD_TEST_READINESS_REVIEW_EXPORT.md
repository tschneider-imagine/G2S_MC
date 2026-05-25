# Phase 3B: Field-Test Readiness Review + Export

## Scope

Phase 3B extends the `/field-test` operator shell with a structured readiness review and evidence export flow.

This phase advances these field-test must-have sections:

- Action Builder Lite
- Template Manager Lite
- G2S Comms Journal
- Emergency Audit Timeline
- Network/Cert Settings

## Field-Test Readiness Review Contents

Readiness adds a checklist view (`/field-test/readiness`) and JSON representation (`/field-test/readiness.json`) that evaluates:

- Inputs: required channels, GPIO/normal/debounce/latch visibility, action bindings, emergency `MANUAL_CLEAR`.
- Actions: definition presence, severity/selectors/steps, return action references, retry/escalation config visibility.
- EGMs: registry count, emergency-enabled/disabled counts, missing template assignment warnings.
- Templates: active versions, renderable action keys, actions JSON presence/validity, matcher placeholders.
- Comms: journal presence, latest outbound result, latest send-gate result.
- Audit: latest transition/action queued/manual clear/send proof entries.
- Settings/Safety: real-send gated default, transport-gate summary, capture policy summary, db/bind metadata, cert status summary.

## Export Behavior

Phase 3B adds evidence/reporting exports only:

- `GET /field-test/export`: full JSON evidence package with readiness report, config snapshots, action previews, comms, audit, and safety summary.
- `GET /field-test/comms/export`: message journal JSON export.
- `GET /field-test/audit/export`: audit timeline JSON export.

Export data is read-only evidence and must not include private key material.

## Intentionally Not Included

- no real EGM send capability expansion,
- no LAN capture allowlist,
- no retry/escalation execution engine changes,
- no fake EGM tooling or simulator behavior,
- no UI chart/dashboard polish work,
- no action run success marking changes,
- no project-definition markdown embedded as runtime page content.

## Definition Of Done

- `/field-test/readiness` exists and renders checklist output.
- `/field-test/readiness.json` exists and returns structured JSON.
- `/field-test/export` exists and returns evidence JSON.
- `/field-test/comms/export` and `/field-test/audit/export` exist.
- Readiness checks cover inputs/actions/EGMs/templates/comms/audit/settings safety.
- Action Builder Lite pages show retry/escalation/return configuration fields.
- Template Manager Lite pages show render preview and matcher placeholder fields.
- Real send remains gated with no new transport behavior.
- `go test ./...` passes.

## Next Expected Phase

Phase 3C: focused operator workflow polish and field-test ergonomics improvements while preserving all send-safety guardrails.
