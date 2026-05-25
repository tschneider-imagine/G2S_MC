# internal/actionexecutor

Action Execution Lite for queued action runs.

This package owns:
- executing a specific queued action run against configured targets,
- rendering configured template action payloads per target,
- attempting delivery through an injected sender interface,
- classifying responses with matcher rules,
- applying retry policy and optional escalation queueing,
- persisting message journal, target result, action run status, and audit updates.

This package does not own:
- GPIO polling,
- input transition generation,
- UI behavior,
- transport mode policy defaults.
