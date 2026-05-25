# Phase 6C: Runtime Version and Delivery Check API

## Scope

- Expose running service build identity for runtime verification.
- Expose Message Delivery Check as JSON for script-friendly verification.

## Included

- Runtime fingerprint fields (version, revision, build time, Go version) in service runtime APIs.
- Runtime build information shown on `/operator/settings`.
- `POST /api/v2/settings/message-delivery-check` using service-side store/config/certificate context.
- Pi verify script checks running revision against local repo head and warns on mismatch.

## Not Included

- No new top-level Operator page.
- No G2S payload send from delivery check.
- No action execution behavior changes.
- No fake tooling.

## Notes

- Network/TLS checks remain explicit and auth-gated.
- Read-only checks remain non-mutating.
- Private key material is never exposed in runtime UI or JSON output.
