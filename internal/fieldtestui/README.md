# internal/fieldtestui

Minimal field-test operator configuration and review shell.

Responsibilities:

- provide a small `/field-test` web surface for operators,
- expose current field-test configuration and runtime review data,
- support minimal configuration mutation forms with existing mutation auth hooks.

Non-responsibilities:

- no legacy operator bundle ownership,
- no real-send transport expansion,
- no action execution logic,
- no GPIO polling or runtime evaluator ownership.

