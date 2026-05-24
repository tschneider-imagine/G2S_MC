# Phase 2C: Action Queue Run Skeleton

## Scope

Phase 2C connects queued action IDs from input transitions to persisted `ActionRun` and `ActionTargetResult` rows without executing actions.

Implemented in this phase:
- `internal/actionruntime` queue package,
- persistence methods for action runs and target results,
- read-only API endpoints for queued runs and targets,
- optional `cmd/g2s-input-monitor` queue helpers and demo action seeding.

## Transition-To-Action-Run Flow

1. Input runtime records a transition and resolves `ActionQueuedID`.
2. Action queue path receives `QueueRequest` (`ActionID`, transition ID, reason, actor, timestamp).
3. Queuer loads the `ActionDefinition`.
4. Queuer builds a preview plan through `internal/actionplanner`.
5. Queuer persists:
   - one `ActionRun` with `status=PENDING`,
   - zero or more `ActionTargetResult` rows (`status=PENDING`, `attempt_count=0`).
6. Queuer records an `ACTION_QUEUED` audit timeline entry with metadata (`action_id`, `target_count`, target IDs, plan warnings).

## Intentionally Not Included

Phase 2C does not include:
- action execution,
- G2S rendering,
- G2S sending,
- action step dispatch,
- cabinet lock/mute command execution,
- UI/frontend rebuild.

## Smoke Test

Queue smoke path on Pi:

```bash
go run ./cmd/g2s-input-monitor -db ./data/gpio-smoke.db -init-defaults -seed-demo-actions -queue-actions -duration 60s -interval 100ms
```

Expected behavior:
- input transitions still occur through Phase 2B pipeline,
- queued action IDs create `ActionRun` rows with `PENDING` status,
- target rows are created when planner resolves targets,
- no G2S messages are sent,
- no action steps are executed.

## Definition Of Done

Phase 2C is done when:
- transition `ActionQueuedID` can create pending action-run skeleton rows,
- target skeleton rows are persisted from planner targets,
- audit timeline records `ACTION_QUEUED`,
- monitor CLI can optionally seed/bind demo action IDs and queue runs,
- `go test ./...` passes,
- no execution or G2S send behavior is added.

## Next Expected Phase

Phase 2D: controlled action-dispatch skeleton (still no real G2S send), with explicit run-state transitions and operator-safe execution gates.

