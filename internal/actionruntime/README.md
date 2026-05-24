# internal/actionruntime

Action run queue skeleton for transition-driven action IDs.

This package owns:
- converting a queued action ID into persisted `ActionRun` and `ActionTargetResult` records,
- using the action planner to resolve target previews before creating run skeleton rows,
- recording audit timeline entries for queue events.

This package does not own:
- action execution,
- G2S rendering or sending,
- GPIO polling,
- UI behavior.

Phase 2C behavior is persistence-only. It must never execute action steps.
