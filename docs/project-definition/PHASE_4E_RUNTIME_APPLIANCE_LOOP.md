# Phase 4E: Runtime Appliance Loop Lite

## Scope

Phase 4E adds an optional runtime loop to `cmd/g2s-mute` so one service can run
input polling, transition-driven action queueing, and optional execution.

## Included

- new `internal/appliance` orchestration package,
- opt-in runtime loop flags in `g2s-mute`,
- optional default input seeding for the four configured channels,
- transition-to-queue flow inside the main appliance service,
- optional execution of newly queued runs from the current process only,
- explicit delivery settings passed to the executor with default delivery disabled.

## Intentionally Not Included

- no fake EGM behavior,
- no new Operator Console module,
- no hidden fallback endpoints,
- no automatic execution of historical pending runs,
- no automatic execution of escalation runs.

## Notes

- runtime loop is disabled by default and must be explicitly enabled,
- delivery remains explicitly configured and disabled by default.

