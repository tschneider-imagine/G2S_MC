# Phase 6F: Return-to-Normal Manual Clear Action

This pass fixes manual clear return-to-normal action queueing from the Operator Console.

- On `POST /operator/inputs/{id}/clear-latch`, a configured on-normal action now creates an ActionRun linked to the clear transition.
- The queued run carries incident linkage when incident context is available during clear.
- When runtime execute-actions is enabled, the queued return run executes through the same HOST_LISTENER prepared/pending path.
- Prepared/pending evidence is recorded in Message Journal and Audit Timeline.
- Prepared is not success; inbound confirmation/failure still determines final target/run outcomes.
- No outbound payload is sent by the HOST_LISTENER prepared path.
- No new top-level Operator Console page is added.
