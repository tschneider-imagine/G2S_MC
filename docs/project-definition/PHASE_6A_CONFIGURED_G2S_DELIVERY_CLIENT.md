# Phase 6A: Configured G2S Delivery Client

## Scope

- Build outbound HTTP delivery client behavior from configured certificate and trust material.
- Resolve delivery endpoint/method/headers/content type from configured EGM and template data.
- Keep configured delivery controls (`delivery_mode`, timeout) in effect.
- Record delivery attempt evidence through existing message journal and audit paths.

## Included Behavior

- Endpoint resolution uses configured EGM/template data only.
- Missing or invalid endpoint configuration fails clearly.
- Template endpoint quirks can override method/content type/headers/timeout.
- TLS/mTLS client setup uses configured CA trust and client certificate/key paths.
- Runtime attempts outbound delivery for configured emergency/test actions and records terminal outcome.

## Not Included

- No fake EGM or simulator behavior.
- No new Operator Console module.
- No automatic fallback endpoint guessing.
- No transport retry policy changes (retry remains in action execution logic).

## Remaining Design Decisions

- Final production endpoint registration/policy for cabinet rollout still requires explicit approval.
- Production certificate lifecycle and rotation policy remains a separate operational decision.
