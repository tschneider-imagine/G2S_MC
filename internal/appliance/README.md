# internal/appliance

`internal/appliance` hosts the optional runtime loop used by `cmd/g2s-mute`.

It polls configured digital inputs, evaluates transitions through `inputruntime`,
queues action runs for transition-bound action IDs, and can optionally execute
newly queued runs with explicit delivery settings.

The package does not own GPIO access internals, action planning internals,
template rendering internals, sender implementation details, or Operator Console
rendering.

