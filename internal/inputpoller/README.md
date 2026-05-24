# inputpoller

`internal/inputpoller` owns polling configured input channels through a digital reader and feeding samples into `internal/inputruntime.Evaluator`.

Owned by this package:
- reading enabled input channel GPIO sources (`DigitalReader`),
- per-poll sample processing and error collection,
- active triggered input resolution using persisted runtime state,
- optional default Pi 4 input channel seeding helpers.

Not owned by this package:
- GPIO hardware implementation,
- action execution,
- G2S message sending,
- UI behavior,
- template behavior.
