# Phase 1C: Input Runtime Evaluator + Transition Audit

## Scope

Phase 1C adds a non-hardware input runtime layer that evaluates configured binary input channels and observed HIGH/LOW samples into stable runtime state, debounced transitions, queued action references, and audit timeline records.

Implemented in this phase:
- `internal/inputruntime` evaluator and deterministic priority resolver,
- runtime-state persistence migration/table,
- store methods for runtime state and transitions,
- read-only runtime input APIs for state and transitions.

## Intentionally Not Included

Not implemented in Phase 1C:
- real GPIO hardware reading,
- action execution,
- G2S message sending,
- frontend/UI changes,
- fake or virtual EGM simulation features.

## Input Runtime Flow

1. Receive `InputSample` (`input_id`, `raw_state`, `observed_at`).
2. Resolve configured `InputChannel`.
3. Load existing runtime state or initialize from first sample.
4. Apply debounce logic to detect stable raw-state commits.
5. Derive `NORMAL/TRIGGERED` from stable raw state and channel normal state.
6. When derived state changes:
- record `InputTransition`,
- capture `ActionQueuedID` from channel bindings,
- record `AuditTimelineEntry` (`INPUT_TRANSITION`) with metadata.
7. Persist runtime state updates.

## Debounce Behavior

- First differing sample creates/updates pending state only.
- Transition commits only when pending state remains consistent for at least `debounce_ms`.
- If sample returns to stable raw state, pending state is cleared.
- First observation initializes runtime state and does not create transitions.

## Audit Behavior

On committed derived-state transition:
- an input transition row is recorded,
- an audit timeline entry is recorded,
- entry includes transition link and metadata JSON containing raw/derived states, queued action ID, and debounce information,
- severity is elevated to `EMERGENCY` when channel/action appears emergency-like; otherwise `INFO/WARNING`.

## API Additions

Read-only endpoints added:
- `GET /api/v2/inputs/state`
- `GET /api/v2/inputs/transitions`

`/api/v2/inputs/state` returns configured channels with runtime state when available.

`/api/v2/inputs/transitions` returns recent transitions, newest first.

No sample-injection or runtime-mutation endpoint is added in this phase.

## Definition Of Done

Phase 1C is done when:
- `internal/inputruntime` exists with evaluator and priority resolver,
- debounced transition logic is implemented and tested,
- runtime state migration and store methods exist and are tested,
- input transition store methods exist and are tested,
- read-only runtime state/transition APIs exist and are tested,
- `go test ./...` passes,
- no GPIO hardware reader, action execution, G2S send path, or UI rebuild is introduced.

## Next Expected Phase

Phase 1D: Pi GPIO reader adapter integration that feeds `internal/inputruntime` with real hardware observations.
