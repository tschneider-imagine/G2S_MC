# Phase 2F.1: Input Safety Hardening

## Issue Observed on Pi
During smoke testing, emergency input behavior showed two safety risks:
- `MANUAL_CLEAR` emergency input returned to HIGH and auto-queued `...-normal`.
- Rapid HIGH/LOW changes could create repeated transition-driven action runs.

## Scope
- Enforce strict manual-clear latch semantics in input runtime.
- Add explicit latch-clear operation (API + CLI path).
- Add transition repeat suppression for manual-clear inputs.
- Keep all send behavior unchanged (still no real send by default).

## Manual-Clear Behavior
- For `LatchingMode=MANUAL_CLEAR`, once runtime state is `TRIGGERED` and latched:
  - electrical return to normal does not auto-transition derived state to `NORMAL`,
  - `OnNormalActionID` is not auto-queued,
  - derived state remains latched until explicit clear.

## Manual Clear Operation
- Added runtime clear operation:
  - `ClearLatchedInput(ctx, inputID, actor, reason)`
- Clear succeeds only when:
  - channel is `MANUAL_CLEAR`,
  - state is latched and `TRIGGERED`,
  - stable electrical state is back to configured normal.
- If electrical state is still triggered, clear is refused.
- Clear success creates a transition `TRIGGERED -> NORMAL` with manual-clear reason and returns queued `OnNormalActionID` when configured.
- Audit records are written for clear attempted/succeeded/failed outcomes.

## Cooldown / Repeat Suppression
- Added per-runtime suppression rule for manual-clear channels:
  - minimum transition interval of 1000ms.
- Debounced transitions inside cooldown are suppressed (no transition/action queue).

## API and CLI Additions
- API:
  - `POST /api/v2/inputs/{id}/clear-latch` (mutation auth required)
- CLI:
  - `-clear-latch INPUT_ID` in `cmd/g2s-input-monitor`
  - optional chaining with `-queue-actions` and `-dispatch-dry-run`
  - no real send behavior added.

## Smoke Test Commands
Manual clear only:
```bash
go run ./cmd/g2s-input-monitor \
  -db ./data/gpio-smoke.db \
  -clear-latch emergency-broadcast
```

Manual clear + queue + dry-run dispatch:
```bash
go run ./cmd/g2s-input-monitor \
  -db ./data/gpio-smoke.db \
  -clear-latch emergency-broadcast \
  -queue-actions \
  -dispatch-dry-run
```

## Definition of Done
- Manual-clear emergency input no longer auto-normalizes on HIGH.
- `OnNormalActionID` for manual-clear only appears after explicit clear.
- Rapid transitions are suppressed for manual-clear path.
- `clear-latch` API and CLI paths exist and are tested.
- No real send behavior introduced.
