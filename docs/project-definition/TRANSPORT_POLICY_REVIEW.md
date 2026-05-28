# Transport Policy Review

## Scope

This note reviews current outbound transport behavior after runtime simplification.

Prime directive: the appliance should attempt configured emergency downstream delivery when operator-approved endpoint, action, and template configuration is present. Safety controls should diagnose, warn, audit, and require explicit configuration; they must not become hidden blockers.

## Current Policy

- Delivery mode is HTTP in runtime send flow.
- Configured prepared messages are sent to resolved endpoints.
- Outcomes are evidence-first: attempted, succeeded, failed.
- Transport/network/TLS failures are recorded as failures, not suppressed by policy gates.

## Observability Controls

- Message journal captures send metadata (`http_status_code`, latency, response excerpt).
- Audit timeline captures dispatch/send lifecycle entries.
- Operator surfaces remain evidence-oriented for diagnosis.

## Production Requirements

1. Endpoint configuration must remain explicit and operator-reviewed.
2. Certificate and trust material policy must remain explicit and auditable.
3. Retry/escalation policy must remain transparent and deterministic.
4. Emergency/test runtime path must continue to attempt configured outbound delivery.

## Current Status

- Runtime send path is configured for outbound HTTP attempts.
- Legacy local send-block policy controls have been removed from runtime behavior.
- Evidence paths remain in place for operational verification.
