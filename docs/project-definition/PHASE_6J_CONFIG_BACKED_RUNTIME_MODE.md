# Phase 6J: Config-Backed Runtime Mode

This pass moves appliance runtime mode into config so service behavior is explicit and repeatable.

- Runtime mode now comes from `runtime.*` in config.
- `g2s-mute.service` starts with only `-config /etc/g2s-mute/config.json`.
- CLI flags still work and override config only when explicitly supplied.
- Default delivery topology is `HOST_LISTENER`.
- Outbound delivery remains disabled by default.
- No new top-level Operator page and no runtime behavior expansion.
