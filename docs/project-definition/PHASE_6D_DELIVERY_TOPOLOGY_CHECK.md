# Phase 6D: Delivery Topology Check

This pass makes Message Delivery Check topology-aware for first field testing.

- `HOST_LISTENER` is the default appliance topology.
- `HOST_LISTENER` does not require an outbound EGM endpoint.
- `OUTBOUND_ENDPOINT` requires a configured endpoint.
- `CAPTURE_ENDPOINT` is explicit capture/analyzer mode and requires a configured capture endpoint.
- Message Delivery Check now reports delivery topology and whether an endpoint is required.
- Message Delivery Check remains read-only and does not send a G2S payload.
- Network and TLS checks remain explicit and mutation-auth gated.
