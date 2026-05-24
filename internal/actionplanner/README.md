# internal/actionplanner

Non-executing action planning and target preview.

This package owns:
- deterministic target selection from action definitions,
- template attachment for selected targets,
- warning generation for planning gaps.

This package does not own:
- action execution,
- G2S message rendering/sending,
- runtime action-run state transitions.
