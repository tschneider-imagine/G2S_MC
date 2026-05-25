# PHASE 5C - Audit Evidence Package Lite

This phase advances the Operator Console Audit area for field review and exportable emergency evidence.

Scope completed in this phase:

- Audit Timeline filtering by `action_run_id`, `input_transition_id`, `egm_id`, and `limit`.
- Related evidence sections under Audit for:
  - Related Action Runs
  - Related Targets
  - Related Messages
  - Related Input Transition
- Operator Note submission under Audit (`POST /operator/audit/notes`) using existing mutation authorization.
- Audit evidence package export (`GET /operator/audit/evidence-export`) with linked:
  - input transitions
  - action runs
  - target results
  - messages
  - audit timeline rows
  - EGM metadata
  - template metadata

Guardrail notes:

- No new top-level page was added.
- Runtime navigation remains: Live, Inputs, Actions, Comms, EGMs, Templates, Audit, Settings.
- No delivery behavior change.
- No action execution behavior change.
- No project/test/milestone wording introduced to runtime UI.
