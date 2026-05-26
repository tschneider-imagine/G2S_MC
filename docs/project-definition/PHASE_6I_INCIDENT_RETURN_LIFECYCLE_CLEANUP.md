# Phase 6I: Incident Return Lifecycle Cleanup

This pass hardens incident/action lifecycle cleanup after return-to-normal success.

- When a return-to-normal action run succeeds, older unresolved incident runs are marked `SUPERSEDED`.
- Superseded is not success and not failure.
- Related unresolved pending-delivery message rows are marked `SUPERSEDED` and retained as evidence.
- Confirmed return evidence remains unchanged.
- Pending views stop treating superseded work as active waiting work.
- No new Operator top-level page is added.
- No outbound payload behavior is expanded by this cleanup.
- No fake or simulator behavior is added.
