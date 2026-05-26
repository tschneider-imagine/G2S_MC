# Phase 6K: Config-Backed Pending-Delivery Sweep

This pass adds config-backed runtime scheduling for pending-delivery sweep using the existing pendingdelivery service.

- Runtime config can enable periodic pending-delivery sweep.
- Sweep interval is configured through runtime settings.
- The scheduler runs the existing waiting-confirmation lifecycle logic.
- No outbound send is added in HOST_LISTENER mode.
- No fake or simulator behavior is added.
- No new top-level Operator page is added.
