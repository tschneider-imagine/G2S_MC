# Phase 1D: Action Planning and Target Preview

## Scope

Phase 1D introduces a non-executing action planner that resolves action target selectors against current EGM registry data and template assignments to produce deterministic preview plans.

Implemented in this phase:
- `internal/actionplanner` package,
- selector resolution for emergency, explicit IDs, group, and template selectors,
- plan warnings for empty target sets, missing templates, and unsupported group membership details,
- read-only API preview endpoint.

## Intentionally Not Included

Not implemented in Phase 1D:
- action execution,
- action run creation,
- G2S rendering/sending,
- GPIO hardware reading,
- UI/frontend changes,
- fake EGM product features.

## Planning Behavior

Input: `ActionDefinition` + current EGM records + template catalog.

Planner output:
- action identity/version,
- deterministic target list sorted by EGM ID,
- action steps,
- planning warnings.

### Selector Rules

- `ALL_EMERGENCY_ENABLED`: selects EGMs that are enabled and emergency-enabled.
- `EGM_IDS:<id1,id2,...>`: selects explicit enabled EGMs by ID.
- `TEMPLATE:<template-id>`: selects enabled EGMs assigned to a template.
- `GROUP:<group-id>`: uses group metadata and zone-based fallback (`EGMRecord.zone == group-id`) with warning.

Disabled EGMs are excluded in all selectors during this phase.

### Warning Rules

Warnings are generated when:
- selected EGMs do not have a template,
- selected EGMs reference unknown templates,
- selector resolves to no eligible targets,
- group selector uses fallback behavior.

## API Addition

Read-only preview endpoint:
- `GET /api/v2/actions/{id}/preview`

Response includes:
- action ID/name/version,
- target count,
- targets (EGM ID, display name, template ID, endpoint/IP),
- steps,
- warnings.

No action execution is performed.

## Definition Of Done

Phase 1D is done when:
- action plans can be previewed without execution,
- selectors resolve deterministically,
- warnings cover planning gaps,
- preview API endpoint is implemented and tested,
- full `go test ./...` passes,
- no G2S messages are sent and no action run is started.

## Next Expected Phase

Phase 2A: Pi GPIO reader adapter.

Phase 2A baseline GPIO channel expectations:
- default channels: `GPIO16`, `GPIO20`, `GPIO21`, `GPIO26` (BCM numbering),
- request pull-up bias on all four lines when supported,
- report clear warning/error when pull-up request is not supported,
- keep trigger semantics configurable by input normal-state settings.

Planned smoke-test command examples for Phase 2A:
- `go run ./cmd/g2s-gpio-probe`
- `go run ./cmd/g2s-gpio-probe -channels GPIO16,GPIO20,GPIO21,GPIO26`
