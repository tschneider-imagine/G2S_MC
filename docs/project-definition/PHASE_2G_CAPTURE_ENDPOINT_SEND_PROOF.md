# Phase 2G: Capture Endpoint Send Proof

## Scope

Phase 2G proves outbound HTTP send behavior against a capture endpoint while preserving normal runtime send-attempt behavior.

This phase adds:

- a local capture sink command for recording outbound payloads,
- endpoint override support for observability testing,
- message journal evidence for send attempt and terminal outcome.

## Why This Is Not an EGM Simulator

The capture sink:

- accepts and records raw HTTP payloads,
- returns a generic `202 Accepted`,
- does not parse G2S semantics,
- does not emulate cabinet behavior,
- does not emit simulated acknowledgements.

It is an observability endpoint for outbound delivery verification.

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
  -queue-actions \
  -dispatch-dry-run \
  -send-prepared \
  -transport http \
  -duration 60s \
  -interval 100ms
```

Expected on trigger:

- queued run created,
- dispatch prepares rendered messages,
- send-prepared reports attempted/succeeded/failed counts,
- capture JSONL contains rendered payloads,
- action run state remains matcher/confirmation driven.

## Intentionally Not Included

- no fake cabinet workflow,
- no response-confirmation protocol change,
- no retry/escalation redesign,
- no UI changes.

## Definition Of Done

- `cmd/g2s-capture-sink` records raw payloads.
- prepared messages can be sent to the configured capture endpoint.
- message journal records send metadata and outcomes.
- `go test ./...` passes.
