# Phase 4C: Input-to-Action Execution Integration

## Scope

Phase 4C connects queued runs from the input monitor flow to Action Execution Lite when execution is explicitly enabled at runtime.

## Included

- `cmd/g2s-input-monitor` execution flags:
  - `-execute-actions`
  - `-delivery-mode disabled|http`
  - `-allow-delivery`
  - `-capture-only`
  - `-delivery-timeout-ms`
- execution of only newly queued runs created in the current monitor process,
- explicit delivery settings passed into the executor,
- queue-only behavior preserved when execution is not enabled,
- execution evidence written through existing message journal and audit paths.

## Intentionally Not Included

- no fake EGM behavior,
- no new Operator Console module,
- no fallback endpoint behavior,
- no automatic execution of historical pending runs,
- no automatic escalation execution (escalation is queued only).

## Notes

- input-triggered return/restore runs use the same execution path when queued,
- disabled or unconfigured delivery records failure evidence and does not report success,
- matcher-driven confirmation/failure behavior remains unchanged.
