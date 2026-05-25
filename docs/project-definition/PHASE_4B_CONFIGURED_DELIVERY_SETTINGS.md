# Phase 4B: Configured Delivery Settings and Executor Sender Wiring

## Scope

Phase 4B wires action execution to explicit delivery settings:

- execution API now accepts delivery configuration per run execution request,
- executor uses configured delivery mode/allow/capture/timeout settings,
- sender wiring is injected through API server and main runtime wiring,
- operator settings shows read-only delivery defaults in product language.

## Included

- `POST /api/v2/actions/runs/{id}/execute` delivery request fields:
  - `delivery_mode`
  - `allow_delivery`
  - `capture_only`
  - `timeout_ms`
- default delivery remains disabled unless explicitly enabled in execute request,
- capture-only host restriction applies only when `capture_only=true`,
- configured HTTP delivery can attempt the configured target endpoint.

## Intentionally Not Included

- no new top-level operator module,
- no fake EGM behavior,
- no transport policy expansion beyond existing sender constraints,
- no action execution automation across historical runs.

## Notes

- success classification remains matcher-driven,
- message journal and audit timeline remain the execution evidence source,
- certificate and mTLS production policy hardening remains a follow-on decision.

