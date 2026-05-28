# Phase 2F: Outbound Transport Attempt Policy

## Scope
- Introduce outbound transport wiring for prepared action messages.
- Require runtime to attempt configured outbound delivery and record actual outcomes.
- Keep evidence in message journal metadata for operator verification.

## Runtime Policy
- Outbound delivery uses HTTP transport.
- When a prepared message has a resolved endpoint, runtime attempts a network send.
- Runtime records `SEND_ATTEMPTED`, then terminal outcome (`SEND_SUCCEEDED` or `SEND_FAILED`).
- Runtime does not suppress configured emergency/test messages with local policy gates.

## HTTP Send Boundary
- Uses `net/http` with timeout controls.
- Defaults method to `POST` when unspecified.
- Defaults content type to `application/soap+xml`.
- Captures status code, latency, and a capped response excerpt.

## Message Journal Evidence
- Result values:
  - `SEND_ATTEMPTED`
  - `SEND_FAILED`
  - `SEND_SUCCEEDED`
- Send metadata:
  - `http_status_code`
  - `latency_ms`
  - `response_excerpt`
  - `sent_at`
  - `completed_at`
  - `transport_mode`

## Smoke Test Commands
Prepare and send:
```bash
go run ./cmd/g2s-input-monitor \
  -db ./data/phase2f-send.db \
  -init-defaults \
  -seed-demo-actions \
  -seed-demo-egms \
  -queue-actions \
  -dispatch-dry-run \
  -send-prepared \
  -transport http \
  -duration 30s \
  -interval 100ms
```

## Intentionally Not Included
- Automatic completion semantics from send attempt alone.
- Advanced retry orchestration updates.
- New UI modules.

## Definition of Done
- Outbound sender is active in runtime send path.
- Configured targets receive attempted sends.
- Journal/audit evidence captures real outcomes.
