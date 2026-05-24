# internal/actiondispatch

Controlled action dispatch skeleton for dry-run/no-send behavior.

This package owns:
- advancing pending action runs into dispatch-prepared state,
- creating dry-run outbound message journal entries,
- recording dispatch audit timeline entries.

This package does not own:
- network send/transport,
- real G2S command delivery,
- GPIO polling,
- UI behavior.

Phase 2D behavior is explicitly no-send.
