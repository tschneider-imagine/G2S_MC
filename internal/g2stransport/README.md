# internal/g2stransport

Outbound transport boundary for G2S action messages.

This package owns:
- transport modes and send gating,
- disabled/no-send behavior,
- guarded HTTP transport behavior.

This package does not own:
- action planning,
- action queueing,
- template rendering,
- GPIO polling,
- UI behavior.

Safety gate:
- real network send requires `TransportMode=HTTP` and `AllowRealSend=true`,
- all other combinations are blocked with no network call.
