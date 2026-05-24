# Phase 2A: Pi GPIO Reader Probe

## Scope

Phase 2A adds a Raspberry Pi-focused GPIO environment probe, a real digital input reader adapter, and a local smoke-test command path.

Implemented in this phase:
- `internal/gpioinput` package for probe + read operations,
- Linux GPIO character-device reader path (v2 ioctl),
- pull-up bias request on input reads,
- local CLI smoke-test command: `cmd/g2s-gpio-probe`.

## Pi GPIO Assumptions

- GPIO channel numbers are **BCM GPIO numbers**, not physical header pin numbers.
- Default smoke-test channels are:
  - `GPIO16`
  - `GPIO20`
  - `GPIO21`
  - `GPIO26`
- Reads are binary only: `HIGH` or `LOW`.
- Trigger semantics (NORMAL/TRIGGERED) remain owned by input configuration and `internal/inputruntime`.

## Pull-Up Requirement

The Linux reader requests pull-up bias on requested input lines using the GPIO character-device v2 line request flags.

If pull-up bias cannot be requested on the active platform/kernel/driver, probe/read paths return clear warnings/errors so operators know the electrical bias request was not applied.

## Smoke Test Steps

Default channels:

```bash
go run ./cmd/g2s-gpio-probe
```

Explicit channels:

```bash
go run ./cmd/g2s-gpio-probe -channels GPIO16,GPIO20,GPIO21,GPIO26
```

Expected behavior:
- print environment probe report,
- print per-channel read result (`HIGH`/`LOW`) or explicit error per channel,
- no action engine execution,
- no UI requirement.

## Dependency Choice

Phase 2A uses the Linux GPIO character-device ioctl path directly (via standard Linux syscalls) rather than adding a large hardware framework dependency.

Reason:
- minimal dependency footprint,
- direct control of pull-up bias request flags,
- deterministic mapping from raw values to `HIGH`/`LOW`.

## Intentionally Not Included

Not implemented in Phase 2A:
- GPIO-to-transition runtime polling loop,
- action execution,
- G2S message sending,
- frontend/UI rebuild,
- fake or virtual EGM product features.

## Definition Of Done

Phase 2A is done when:
- probe command compiles and runs,
- default channels are `GPIO16`, `GPIO20`, `GPIO21`, `GPIO26`,
- pull-up request is attempted and support/failure is explicitly reported,
- channel reads return `HIGH`/`LOW` or clear GPIO access errors,
- tests pass (`go test ./...`),
- no action execution, G2S sending, or UI rebuild work is introduced.

