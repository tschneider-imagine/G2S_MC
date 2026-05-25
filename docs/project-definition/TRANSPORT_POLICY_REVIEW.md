# Transport Policy Review

## Scope

This note reviews current outbound transport enforcement after removal of unapproved runtime UI/export surfaces.

Prime directive: the appliance should attempt configured emergency downstream delivery when operator-approved endpoint, action, and template configuration is present. Safety controls should diagnose, warn, audit, and require explicit configuration; they must not become hidden blockers.

## Current Blockers Classification

### A. Keep (explicit safety and anti-accidental-send controls)

- Default no-send posture (`ModeDisabled`/`ModeDryRun`).
- Explicit real-send opt-in (`AllowRealSend=true`).
- Explicit transport mode selection (`ModeHTTP` required for HTTP send path).
- Audit + message journal result recording for blocked/failed/succeeded attempts.

These are explicit operator configuration gates and should remain.

### B. Rework before production send

- `CaptureOnlySend` hard requirement in HTTP sender.
- Loopback-only endpoint restriction enforced by `CaptureEndpointAllowed`.

These are valid for capture proofing but would become hidden product blockers if left as permanent behavior for real EGM delivery.

### C. Capture-proof-only behavior (must stay isolated from product send policy)

- Requirement that endpoint host must be loopback (`localhost`, `127.0.0.1`, `::1`) when capture-only mode is active.
- Capture-endpoint override usage for send-prepared smoke/capture flows.

This behavior is phase-specific proof instrumentation and not a long-term endpoint policy.

## Required Design Decision Before Real EGM Send

Before enabling product real-send flow:

1. Define explicit operator-approved endpoint policy for production (not implicit loopback-only).
2. Separate capture-proof mode from production mode in config and runtime behavior.
3. Keep explicit opt-in + audit trail for all send attempts.
4. Ensure emergency-triggered configured actions attempt delivery once approved settings exist.

## Current Status

- No real send is enabled by default.
- Capture-proof guardrails remain in place.
- Runtime UI no longer exposes unapproved transport/readiness framing.
