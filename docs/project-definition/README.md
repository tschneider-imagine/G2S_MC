# Project Definition Index

Use this folder as the active planning entry point for G2S_MC rebuild work.

Field-test is the milestone name only. The runtime product surface is the Operator Console and must not use field-test as product identity.
Runtime UI scope is limited to: Live, Inputs, Actions, Comms, EGMs, Templates, Audit, Settings.
Electrical Input Monitor behavior is implemented on `/operator/inputs` as a live state display with periodic in-page updates; no new top-level page is added.
Readiness/System Check runtime pages are not approved replacements and remain out of runtime scope.
MVP runtime exports are Comms export and Audit/Evidence export only.
Legacy dashboard routes are removed from active runtime serving.
Fake EGM tooling is not active product scope.

Runtime Operator Console navigation contract:

- Live
- Inputs
- Actions
- Comms
- EGMs
- Templates
- Audit
- Settings

No standalone readiness/system-check or bundled operator report export is in runtime scope.
Visible seed data in Operator Console must remain product-neutral (for example cabinet/template/vendor naming).

## Active Project Definition / Guardrails

- `G2S_MC_REBUILD_PROJECT_DEFINITION_AND_GUARDRAILS.md`

## Field-Test Must-Have Plan

- `G2S_MC_REBUILD_PROJECT_DEFINITION_AND_GUARDRAILS.md`
- `../First-Cabinet-Session-Execution-Plan.md`
- `../First-Cabinet-Prep-Checklist.md`

## Global Chat Prompt Additions

- `GLOBAL_CHAT_PROMPT_ADDITIONS.md`

## Current Phase Docs

- `PHASE_1A_DOMAIN_FOUNDATION.md`
- `PHASE_1B_API_CONFIGURATION_SURFACE.md`
- `PHASE_1C_INPUT_RUNTIME_EVALUATOR.md`
- `PHASE_1D_ACTION_PLANNING_PREVIEW.md`
- `PHASE_2A_PI_GPIO_READER.md`
- `PHASE_2B_GPIO_INPUT_MONITOR.md`
- `PHASE_2C_ACTION_QUEUE_RUN_SKELETON.md`
- `PHASE_2D_ACTION_DISPATCH_DRY_RUN.md`
- `PHASE_2E_TEMPLATE_RENDER_DRY_RUN.md`
- `PHASE_2F_OUTBOUND_TRANSPORT_GATE.md`
- `PHASE_2F1_INPUT_SAFETY_HARDENING.md`
- `PHASE_2G_CAPTURE_ENDPOINT_SEND_PROOF.md`
- `PHASE_3A_FIELD_TEST_OPERATOR_CONFIGURATION_SHELL.md`
- `PHASE_3B_FIELD_TEST_READINESS_REVIEW_EXPORT.md`
- `PHASE_4A_ACTION_EXECUTION_LITE.md`
- `PHASE_4B_CONFIGURED_DELIVERY_SETTINGS.md`
- `PHASE_4C_INPUT_TO_ACTION_EXECUTION.md`
- `PHASE_4D_INBOUND_MESSAGE_CONFIRMATION_LITE.md`
- `PHASE_4E_RUNTIME_APPLIANCE_LOOP.md`
- `PHASE_5D_OPERATOR_LIVE_VIEW.md`
- `PHASE_5E_EMERGENCY_INCIDENT_LIFECYCLE.md`
- `PHASE_5F_COMMS_HANDLER_RULE_LITE.md`
- `PHASE_5G_EGM_GROUPS_REGISTRY_IMPORT_EXPORT.md`
- `PHASE_5C_AUDIT_EVIDENCE_PACKAGE_LITE.md`
- `PHASE_6A_CONFIGURED_G2S_DELIVERY_CLIENT.md`
- `PHASE_6A_CERTIFICATE_DELIVERY_READINESS.md`
- `PHASE_6B_MESSAGE_DELIVERY_CHECK_LITE.md`
- `PHASE_6C_RUNTIME_VERSION_DELIVERY_CHECK_API.md`
- `TRANSPORT_POLICY_REVIEW.md`

## Archived Old Plans

- `../archive/old-plans/`

## Runbooks

- `../runbooks/PI_FIELD_TEST_SERVICE_RUNBOOK.md`
