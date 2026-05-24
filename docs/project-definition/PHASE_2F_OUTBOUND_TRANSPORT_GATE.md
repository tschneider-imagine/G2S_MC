# Phase 2F: Outbound Transport Safety Gate

## Scope
- Introduce an outbound transport abstraction for prepared action messages.
- Add a hard safety gate so real send cannot happen accidentally.
- Keep default runtime behavior no-send.
- Record blocked/attempted/succeeded/failed send outcomes in message journal metadata.

## Safety Gate Rules
Real network send is allowed only when both are true:
1. `transport_mode=HTTP`
2. `allow_real_send=true`

If either condition is missing, send is blocked and no network call is made.

## Transport Modes
- `DISABLED`: always blocked, no network calls.
- `DRY_RUN`: blocked at transport boundary, no network calls.
- `HTTP`: uses guarded HTTP sender, still blocked unless `allow_real_send=true`.

## No-Send Default
- Existing monitor flows remain no-send unless explicit send flags are provided.
- `-dispatch-dry-run` still only prepares outbound journal rows.
- `-send-prepared` with default transport (`disabled`) records blocked behavior.

## HTTP Send Boundary
- Uses `net/http` with safe timeout defaults.
- Defaults method to `POST` when unspecified.
- Defaults content type to `application/soap+xml`.
- Captures HTTP status, latency, and response excerpt (capped).
- No retries/escalation in this phase.

## Message Journal Behavior
- Added result values:
  - `SEND_BLOCKED`
  - `SEND_ATTEMPTED`
  - `SEND_FAILED`
  - `SEND_SUCCEEDED`
- Added send metadata fields:
  - `http_status_code`
  - `latency_ms`
  - `response_excerpt`
  - `sent_at`
  - `completed_at`
  - `transport_mode`

## Smoke Test Commands
No-send blocked path:
```bash
go run ./cmd/g2s-input-monitor \
  -db ./data/phase2f-gate.db \
  -init-defaults \
  -seed-demo-actions \
  -seed-demo-egms \
  -queue-actions \
  -dispatch-dry-run \
  -send-prepared \
  -duration 30s \
  -interval 100ms
```

Explicit HTTP mode without allow flag (still blocked):
```bash
go run ./cmd/g2s-input-monitor \
  -db ./data/phase2f-gate.db \
  -queue-actions \
  -dispatch-dry-run \
  -send-prepared \
  -transport http
```

## Intentionally Not Included
- Automatic send on GPIO-triggered transitions.
- Retry strategy/escalation strategy.
- Run completion/success workflow.
- UI/frontend work.
- G2S schema validation as emergency-blocking logic.

## Definition of Done
- Transport abstraction exists with disabled + guarded HTTP senders.
- Safety gate enforced by both mode and allow flag.
- Default command behavior remains no-send.
- Blocked sends proven with tests to avoid network calls.
- Message journal update path records send metadata.

## Next Expected Phase
- Controlled, operator-driven send orchestration with confirmation and retry policy integration.
