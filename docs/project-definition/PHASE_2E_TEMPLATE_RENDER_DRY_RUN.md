# Phase 2E: Template-Rendered Dry-Run Dispatch

## Scope
- Keep dispatch in strict no-send mode.
- Replace placeholder dry-run payloads with rendered payloads from active G2S template versions.
- Persist rendered dry-run journal rows for operator inspection.
- Keep run state progression in the existing dry-run dispatch flow.

## Template Action JSON Shape
`g2s_template_versions.actions_json` uses:

```json
{
  "actions": {
    "queue_only_no_send": {
      "message_type": "DRY_RUN_NO_SEND",
      "content_type": "application/xml",
      "payload_template": "<dryRunG2SMessage noSend=\"true\" action=\"{{.ActionID}}\" run=\"{{.ActionRunID}}\" egm=\"{{.EGMID}}\" step=\"{{.TemplateActionKey}}\" timestamp=\"{{.TimestampRFC3339}}\"/>",
      "headers": {
        "X-Dry-Run": "true"
      }
    }
  }
}
```

## Supported Render Variables
- `ActionID`
- `ActionRunID`
- `ActionStepID`
- `TemplateActionKey`
- `EGMID`
- `HostID`
- `TimestampRFC3339`
- `IPAddress`
- `EndpointPath`
- `TemplateID`
- `TemplateVersion`
- extra caller-supplied `Variables` map keys

## Dry-Run Rendering Behavior
- Dispatch loads:
  - action run,
  - action definition,
  - target rows,
  - target EGM record/template assignment,
  - active template version.
- For each target, renderer resolves the action step `TemplateActionKey` in `actions_json`.
- Render success records message journal with rendered payload.
- Missing template/template-version/action-key/render errors do not abort the whole run:
  - warning is recorded,
  - dry-run no-send journal row is still recorded with error details.
- No network transport is invoked.

## Message Journal Behavior
- Direction: `OUTBOUND`
- Result: `DRY_RUN`
- Payload: rendered when available, otherwise explicit dry-run render-unavailable placeholder.
- Summary JSON includes:
  - `dry_run=true`
  - `no_send=true`
  - `rendered` true/false
  - action/run/template/action-key/message metadata

## Smoke Test Commands
```bash
go run ./cmd/g2s-input-monitor \
  -db ./data/phase2e-render.db \
  -init-defaults \
  -seed-demo-actions \
  -seed-demo-egms \
  -queue-actions \
  -dispatch-dry-run \
  -duration 60s \
  -interval 100ms
```

## Intentionally Not Included
- Real G2S schema validation as a dispatch blocker.
- Any outbound network send.
- Action execution/confirmation loop.
- GPIO hardware behavior changes.
- UI/frontend rebuild.

## Definition of Done
- Active template versions are persisted and retrievable.
- Dry-run dispatch renders from `actions_json` when available.
- Journal rows contain rendered no-send payloads and diagnostics.
- Demo seed data includes active no-send template version.
- No network calls are added.

## Next Expected Phase
- Real transport integration and controlled send path with explicit safeguards.
