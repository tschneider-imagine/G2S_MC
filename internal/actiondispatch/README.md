# internal/actiondispatch

Controlled dispatch workflow for prepared message rows.

This package owns:
- advancing pending action runs into dispatch-prepared state,
- creating dry-run outbound message journal entries,
- optional send-prepared boundary integration with transport safety gating,
- recording dispatch/send audit timeline entries.

This package does not own:
- transport implementation details,
- action planning/queueing,
- GPIO polling,
- UI behavior.

Default behavior remains no-send unless explicit transport settings are provided.
