# Phase 4A: Action Execution Lite

## Scope

Phase 4A adds controlled execution for queued action runs:

- execute a specific queued run,
- render configured template actions per target,
- attempt configured delivery through an injected sender,
- classify responses with matcher rules,
- apply retry count and delay from stored action policy,
- queue escalation action runs when configured failures exhaust attempts,
- persist execution evidence to message journal and audit timeline.

## Included

- `internal/actionexecutor` package (executor, models, tests),
- target result update persistence,
- `POST /api/v2/actions/runs/{id}/execute` (mutation-auth required),
- execution audit and message journal evidence updates.

## Intentionally Not Included

- no new top-level Operator Console pages,
- no fake EGM behavior,
- no transport policy expansion,
- no automatic batch execution of historical runs,
- no unconditional success marking based on send alone.

## Execution Rules

- only explicitly requested run IDs are executed,
- run must be `PENDING`,
- run is set `RUNNING` before attempts begin,
- run is `SUCCEEDED` only when target confirmation is matched,
- failures follow retry policy, then mark target/run failed,
- configured escalation actions are queued (not auto-executed in this phase).

## Transport Boundary Note

Executor does not hardcode transport policy defaults; it uses the injected sender.
Any capture-proof restrictions remain inside sender implementation until a later transport policy phase.

