# Phase 6A: Certificate and Configured Delivery Readiness

## Scope

- Make certificate and trust material readiness explicit in existing Settings.
- Ensure configured delivery uses explicit EGM/template endpoint data.
- Ensure TLS/mTLS client construction uses configured CA trust and client certificate material.
- Keep delivery diagnostics visible through message journal and audit evidence.

## Included

- Certificate/trust readiness reflects configured, missing, invalid, and valid states.
- Configured delivery endpoint resolution is explicit and has no fallback endpoint.
- Template endpoint quirks can control method, content type, headers, timeout, and endpoint path.
- Missing endpoint or certificate/trust errors fail clearly and are recorded as delivery evidence.

## Not Included

- No new Operator top-level area.
- No fake EGM behavior.
- No runtime schema/compliance blocker added.
- No private key material exposure.

## Notes

- Capture-only restrictions remain explicitly capture-only behavior and are not the configured product delivery path.
- Action success/failure still depends on matcher outcomes, not send-attempt status alone.
