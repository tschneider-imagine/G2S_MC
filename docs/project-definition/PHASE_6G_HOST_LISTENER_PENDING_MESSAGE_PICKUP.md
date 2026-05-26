# Phase 6G: Host Listener Pending Message Pickup

This phase advances host listener pending delivery behavior using existing runtime paths.

- In `HOST_LISTENER` topology, prepared outbound messages can be offered during `/g2s` client contact for the matching EGM.
- When `/g2s` contact supplies `action_run_id`, pickup is scoped to that action run and EGM; unrelated backlog messages are not offered.
- Without `action_run_id`, pickup keeps oldest-pending-by-EGM behavior.
- Message lifecycle is explicit: `PREPARED` -> `OFFERED` -> `CONFIRMED` / `FAILED` / `EXPIRED` / `SUPERSEDED`.
- Offered delivery is not treated as success; target/run success still requires inbound matcher or handler-rule confirmation.
- Waiting-confirmation sweep support is provided through service functions without introducing a new runtime module.
- No fake or simulator behavior is added.
- No outbound endpoint is required for host listener pickup behavior.
- No new top-level Operator page is added.
