# Phase 2D: Action Dispatch Dry Run

## Scope

Phase 2D adds a controlled, no-send dispatch skeleton for queued action runs.

Implemented in this phase:
- `internal/actiondispatch` dispatcher package,
- explicit run transition from `PENDING` to `DISPATCH_PREPARED` in dry-run mode,
- outbound message journal entries marked dry-run/no-send,
- API dispatch trigger endpoint with mutation auth.

## Run State Flow

Phase 2D run progression:

1. `PENDING` (from Phase 2C queue path)
2. dry-run dispatch request received
3. dispatcher prepares per-target outbound placeholders
4. run updated to `DISPATCH_PREPARED`
5. audit timeline records `ACTION_DISPATCH_PREPARED`

No success/failure confirmation flow is executed in this phase.

## Dry-Run Dispatch Behavior

- Only `DRY_RUN` mode is supported.
- Only runs in `PENDING` status may be dispatched.
- For each target row:
  - resolve EGM record and template assignment,
  - create outbound message journal entry with payload marker:
    - `DRY_RUN_NO_SEND action=<id> egm=<id> step=<key>`
  - set parsed summary metadata with `dry_run=true` and `no_send=true`.
- No sockets are opened.
- No HTTP clients are used for EGM traffic.
- No command is transmitted.

## Message Journal Behavior

Dry-run message rows are recorded as outbound entries with:
- `direction=OUTBOUND`
- `result=DRY_RUN`
- linked `action_run_id`
- linked `egm_id`
- dry-run placeholder payload and summary JSON.

## Intentionally Not Included

Phase 2D does not include:
- real G2S transport send,
- action step execution against EGMs,
- confirmation matching,
- retry/escalation execution logic,
- UI/frontend rebuild.

## Smoke Test Commands

Pi smoke path:

```bash
go run ./cmd/g2s-input-monitor \
  -db ./data/phase2d-dryrun.db \
  -init-defaults \
  -seed-demo-actions \
  -seed-demo-egms \
  -queue-actions \
  -dispatch-dry-run \
  -duration 60s \
  -interval 100ms
```

Expected when `GPIO21` goes LOW:
- transition to `TRIGGERED`,
- `action_queued=emergency-broadcast-trigger`,
- `queued_run run_id=...`,
- `dry_run_dispatch run_id=... messages=...`,
- message journal contains `DRY_RUN_NO_SEND` outbound entries,
- no network traffic to EGMs.

## Definition Of Done

Phase 2D is done when:
- pending runs can be dry-run dispatched,
- message journal captures outbound dry-run/no-send entries,
- run status transitions to dispatch-prepared state,
- API endpoint can trigger dry-run dispatch with mutation auth,
- monitor can optionally queue and dry-run dispatch runs created in-process,
- `go test ./...` passes,
- no real G2S send path exists.

## Next Expected Phase

Phase 2E: controlled confirmation/retry state-machine scaffolding (still with transport safety gates before any real send enablement).

