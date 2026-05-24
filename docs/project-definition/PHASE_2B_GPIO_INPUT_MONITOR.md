# Phase 2B: GPIO Input Monitor (Runtime Evaluator Integration)

## Scope

Phase 2B connects real GPIO sample reads to the existing `internal/inputruntime` evaluator so the appliance can produce stable input runtime state, debounced transitions, and audit timeline records from hardware observations.

Implemented in this phase:
- `internal/inputpoller` package for polling enabled input channels via a `DigitalReader`,
- default Pi 4 input-channel seeding helpers,
- `cmd/g2s-input-monitor` CLI to run one-shot or interval polling and print runtime observations.

## Default GPIO Channels

Default BCM channels for first smoke path:
- `GPIO16` (`regular-operation`)
- `GPIO20` (`general-broadcast`)
- `GPIO21` (`emergency-broadcast`)
- `GPIO26` (`local-notice`)

## Default Normal State

Phase 2B default channels use:
- `normal_state = HIGH`

This means with pull-ups enabled and no active pull-down, channels derive `NORMAL`.

Trigger semantics are still per-channel and come from `InputChannel.NormalState`; this phase does not hardcode LOW or HIGH as universally triggered.

## Smoke Test Commands

Initialize defaults and poll once:

```bash
go run ./cmd/g2s-input-monitor -init-defaults -once
```

Run interval polling for 30 seconds:

```bash
go run ./cmd/g2s-input-monitor -init-defaults -duration 30s -interval 100ms
```

Use explicit DB path:

```bash
go run ./cmd/g2s-input-monitor -db ./data/gpio-smoke.db -init-defaults -duration 60s -interval 100ms
```

## Observing Transitions

The monitor prints, per poll, each enabled input with:
- GPIO channel,
- raw state (`HIGH`/`LOW`),
- derived state (`NORMAL`/`TRIGGERED`),
- transition flag and transition ID,
- queued action ID (if configured by channel binding),
- per-input read/evaluation errors.

Runtime transitions are persisted through the evaluator store path:
- `input_runtime_states` updates,
- `input_transitions` records,
- `audit_timeline` records (`INPUT_TRANSITION`).

## Safety Note

If testing directly on Pi header pins, do not apply external voltage to GPIO pins.
Use the EE input stage or a safe pull-up/pull-down jumper test path.

## Intentionally Not Included

Phase 2B does not include:
- action execution,
- G2S message sending,
- UI rebuild or UI runtime polling integration,
- template/action behavior changes.

## Definition Of Done

Phase 2B is done when:
- GPIO samples feed `internal/inputruntime.Evaluator` through polling,
- default four input channels can be seeded,
- runtime state persists,
- debounced transitions are recorded,
- audit timeline entries are recorded for transitions,
- highest-priority active triggered input can be resolved,
- CLI monitor works for one-shot and interval runs,
- no action execution or G2S sending is introduced.

## Next Expected Phase

Phase 2C: input action runner wiring (controlled execution path) with strict guardrails and no UI rebuild until backend runtime behavior is stable.
