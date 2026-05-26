# Phase 6L: Operator Pending Delivery Controls

This pass adds safe pending-delivery lifecycle controls inside existing Operator Console pages.

What this advances:
- Operators can reprepare, expire, or supersede eligible pending delivery rows from Comms.
- Controls are limited to pending lifecycle states (`PREPARED`, `PENDING_DELIVERY`, `OFFERED`).
- Confirmed and failed evidence cannot be rewritten by these controls.
- Lifecycle actions write Audit Timeline evidence.
- Operators can run pending-delivery sweep on demand from existing Actions page context.

Scope guardrails:
- No new top-level page.
- No outbound payload send added.
- `PREPARED` and `OFFERED` are not treated as success.
- Existing inbound confirmation/failure semantics remain unchanged.
