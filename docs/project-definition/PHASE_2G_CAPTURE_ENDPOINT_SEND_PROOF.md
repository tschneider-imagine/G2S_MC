# Phase 2G: Capture Endpoint Send Proof

## Scope

Phase 2G proves controlled HTTP send behavior against an explicit capture endpoint while keeping all EGM behavior disabled.

This phase adds:

- a local capture sink command for recording outbound payloads,
- stricter HTTP transport safety checks,
- monitor flags for explicit capture-only send proof,
- message journal updates for capture send success/failure.

## Why This Is Not an EGM Simulator

The capture sink:

- accepts and records raw HTTP payloads,
- returns a generic `202 Accepted`,
- does not parse G2S semantics,
- does not emulate cabinet behavior,
- does not emit simulated acknowledgements.

It is only an analyzer/capture endpoint for outbound transport verification.

## Capture Endpoint Safety Rules

Real HTTP send in Phase 2G is allowed only when all conditions are true:

1. `transport_mode` is `HTTP`.
2. `allow_real_send` is `true`.
3. `capture_only_send` is `true`.
4. endpoint host is loopback-only:
   - `localhost`
   - `127.0.0.1`
   - `::1`

If any condition is missing, send is blocked with a clear reason, including
`endpoint_not_allowed_for_capture_phase` for non-loopback hosts.

## Smoke Test Commands

Terminal 1:

```bash
go run ./cmd/g2s-capture-sink -bind 127.0.0.1:18080 -path /capture -out ./data/capture.jsonl
```

Terminal 2:

```bash
go run ./cmd/g2s-input-monitor \
  -db ./data/phase2g-capture.db \
  -init-defaults \
  -seed-demo-actions \
  -seed-demo-egms \
  -capture-endpoint http://127.0.0.1:18080/capture \
  -capture-only-send \
  -queue-actions \
  -dispatch-dry-run \
  -send-prepared \
  -transport http \
  -allow-real-send \
  -duration 60s \
  -interval 100ms
```

Expected on trigger:

- queued run created,
- dry-run dispatch prepares rendered messages,
- send-prepared reports sent count for capture endpoint,
- capture JSONL contains rendered payloads,
- action run is not marked `SUCCEEDED`.

## Intentionally Not Included

- no real EGM command workflow,
- no response-confirmation protocol,
- no retry/escalation policy,
- no run completion semantics,
- no UI changes.

## Definition Of Done

- `cmd/g2s-capture-sink` exists and records raw payloads.
- HTTP send remains blocked unless explicit capture-only gate is satisfied.
- Non-loopback endpoints are blocked in Phase 2G.
- Prepared messages can be sent to loopback capture endpoint only.
- Message journal records send metadata and results.
- Action runs are not marked `SUCCEEDED`.
- `go test ./...` passes.

## Next Expected Phase

Phase 2H: controlled production endpoint policy and send lifecycle progression (still safety-first, with explicit operational controls).

