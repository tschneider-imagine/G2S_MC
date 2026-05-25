# Phase 4D: Inbound Message Confirmation Lite

## Scope

Phase 4D adds inbound message capture and matcher-based confirmation correlation.

## Included

- inbound payloads are journaled as `INBOUND` message rows,
- best-effort parsing for EGM ID, action run ID, and message type from query/header/body,
- correlation to action run targets when unambiguous,
- matcher-based expected/failure evaluation using configured template version rules,
- target/run status updates only when matcher evidence supports it,
- audit evidence for receive, correlation, confirmation/failure, and no-match warnings.

## Intentionally Not Included

- no schema-blocking validation,
- no fake EGM behavior,
- no new Operator Console module,
- no fallback endpoint/message generation,
- ambiguous inbound messages are journaled but do not mutate target status.

## Notes

- inbound capture is non-blocking to listener responses,
- existing Comms and Audit views surface inbound evidence from shared data sources.
