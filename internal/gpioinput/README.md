# internal/gpioinput

GPIO environment probe and digital input reading for Raspberry Pi/Linux targets.

This package owns:
- probing the GPIO userspace environment,
- reading binary GPIO input states as `HIGH`/`LOW`,
- requesting pull-up bias for input reads when supported.

This package does not own:
- NORMAL/TRIGGERED input evaluation,
- debounce/transition rules,
- action execution,
- G2S send behavior,
- UI behavior.

Phase 2A defaults use BCM channels:
- `GPIO16`
- `GPIO20`
- `GPIO21`
- `GPIO26`

The Linux implementation uses the GPIO character device API (v2 ioctl path) and requests pull-up bias as part of the line request.
