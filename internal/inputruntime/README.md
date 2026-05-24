# internal/inputruntime

Runtime evaluation for configured input channels using observed binary HIGH/LOW samples.

This package owns:
- debounced stable raw-state tracking,
- derived NORMAL/TRIGGERED state transitions,
- queued action reference selection from channel bindings,
- runtime transition/audit record intent.

This package does not own:
- hardware GPIO reading,
- action execution,
- G2S message sending,
- UI behavior.
