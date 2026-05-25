# Phase 5F: Comms Handler Rule Lite

## Scope
- Advances Message Journal handling under the existing `Comms` area.
- Adds operator-defined Handler Rules from observed messages without source-code changes.
- Supports rule preview, storage, update, and disable flows.
- Optionally applies matching inbound handler rules to correlated target confirmation/failure outcomes.

## Included
- Handler rule model fields for direction, related template/message/action metadata, match JSON, and outcome.
- Store support for upsert/get/list/enabled-only/disable.
- Comms UI routes under `/operator/comms/handler-rules` and message-linked rule creation from `/operator/comms`.
- Rule preview against selected message payload/summary.
- Audit events for rule create/update/disable and inbound rule match use.
- Message journal linkage via `handler_rule_id`.

## Intentionally Not Included
- No new top-level Operator navigation area.
- No in-place mutation of active template versions.
- No new delivery behavior or transport behavior.
- No action execution behavior changes beyond optional inbound handler-rule classification.

## Notes
- Handler rules are separate operator/runtime controls.
- Template matcher behavior remains available; handler rules can classify inbound messages and, when correlated, apply configured confirmation/failure outcomes.

## Definition Of Done
- Handler rules can be created from observed messages and listed under Comms.
- Handler rule preview works with message payload and summary.
- Rule create/update/disable is audited.
- Active templates are not silently modified.
- Inbound rule matching is integrated for confirmation/failure/ignore/note outcomes.
