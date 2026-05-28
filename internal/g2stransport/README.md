# internal/g2stransport

Outbound transport boundary for G2S action messages.

This package owns:
- outbound HTTP send execution,
- configured endpoint resolution from EGM/template data,
- configured TLS/mTLS outbound client construction.

This package does not own:
- action planning,
- action queueing,
- template rendering,
- GPIO polling,
- UI behavior.

Delivery policy:
- runtime uses HTTP outbound send attempts for configured messages,
- delivery success/failure is determined by real endpoint/network response,
- policy flags must not silently block emergency/test send attempts.
