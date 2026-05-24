# G2S_MC Rebuild Project Definition, Repo Handling, and Guardrails

**Recommended repo path:** `docs/project-definition/G2S_MC_REBUILD_PROJECT_DEFINITION_AND_GUARDRAILS.md`  
**Status:** Draft for project direction and implementation planning  
**Primary decision:** Hybrid rebuild. Keep useful backend foundations, rebuild the front end, and re-orient the backend around hardwire inputs, actions, G2S templates, communications history, and audit.

---

## 0. First Step: Repo Handling

The repo must be stabilized before implementation planning starts. The current project contains useful work, but the active branch has signs of unsafe merge state and architectural drift. We should not start the rebuild by layering more code on top of the current shape.

### 0.1 Repo decision

Use the **same GitHub repository**, not a new repo, but create a clean rebuild branch and protect the old work.

Reason:

- We keep history, docs, and learned behavior.
- We avoid wasting time recreating project setup.
- We can salvage backend packages and tests.
- We can clearly separate legacy/scaffold code from the new product direction.

### 0.2 Immediate repo steps

1. **Freeze direct work on `main`.**
   - No new feature commits directly to `main`.
   - Treat current `main` as a historical snapshot until the build is proven clean.

2. **Tag the current state.**
   - Suggested tag: `archive/broken-main-before-rebuild-YYYYMMDD`
   - Purpose: preserve the exact state before cleanup.

3. **Create a build-recovery branch.**
   - Suggested branch: `cleanup/restore-build`
   - Purpose: remove conflict markers, restore compilation, and run tests.

4. **Resolve or revert the dirty merge commit.**
   - Remove all `<<<<<<<`, `=======`, and `>>>>>>>` markers.
   - Prefer the simpler API direction:
     - Use `issues` consistently for preflight/readiness.
     - Do not reintroduce `blockers` unless the new action engine has a clear need for it.
   - If the latest dirty commit cannot be trusted, reset to the latest compiling commit and cherry-pick only clean work.

5. **Run verification.**
   - Required local command:
     ```bash
     go test ./...
     ```
   - Any check script can wrap this, but the source of truth is still `go test ./...`.

6. **Create the rebuild branch from a green commit.**
   - Suggested branch: `rebuild/input-action-engine`
   - This becomes the branch where the new project definition is implemented.

7. **Protect `main`.**
   - Require pull requests.
   - Require CI passing.
   - Require conflict-marker scan passing.
   - Disallow direct pushes.
   - Require at least one review for architectural changes.

8. **Add this document to the repo.**
   - Path:
     ```text
     docs/project-definition/G2S_MC_REBUILD_PROJECT_DEFINITION_AND_GUARDRAILS.md
     ```

### 0.3 Required CI guardrails before new work

Add these checks immediately:

```bash
go test ./...
```

```bash
! grep -R "<<<<<<<\|=======\|>>>>>>>" .
```

Recommended CI jobs:

- Go tests
- Conflict marker scan
- Go formatting check
- Static package boundary check
- Frontend build check once new frontend exists
- Generated asset check once frontend bundling exists

### 0.4 Legacy handling

Do not delete everything immediately. Move intentionally.

Recommended legacy strategy:

```text
legacy/
  ui-operator-v2/
    README.md
    notes.md
```

But only move legacy UI there if it is useful for reference. Do not keep the old UI as the active product shell.

Do not allow old project plans, research notes, XSD staging notes, or internal development docs to be embedded into the running product. Product help is allowed only when intentionally curated as operator-facing help.

---

## 1. Product Definition

The product is:

> A four-input hardwire-driven emergency action appliance that uses G2S as a configurable, vendor-tolerant action transport, with full message history, action tracking, escalation, return-to-normal behavior, and emergency audit evidence.

The product is **not** primarily:

- a G2S standards compliance checker,
- an EGM simulator,
- a generic dashboard,
- a lab-only fake EGM host,
- a project-document viewer,
- or a compliance-first blocker that refuses to act during a life-safety event.

### 1.1 Prime directive

The appliance exists to support life safety.

When a configured emergency signal is triggered, the system should:

1. detect the hardwire input change,
2. determine the configured action,
3. send the configured G2S message sequence to the intended EGMs,
4. monitor the result,
5. retry or escalate when needed,
6. record everything,
7. restore/return to normal when the signal returns to normal or when an authorized operator clears it.

### 1.2 Compliance posture

G2S compliance is useful for diagnostics, interoperability, and template creation. It must not be the core runtime blocker.

Correct posture:

> Validation should diagnose problems, help create safer templates/actions, and warn loudly. It should not block emergency operation merely because a message is non-standard if that message is the configured working path for the target EGM/template.

### 1.3 Safety posture

“Work no matter what” does not mean “send random unsafe messages.” Safety is created by:

- explicit operator configuration,
- template versioning,
- preview of targets and generated messages,
- confirmation rules,
- retry limits,
- escalation rules,
- audit logging,
- reversible return-to-normal actions,
- emergency-priority behavior,
- and clear indication when the system is operating with warnings.

Runtime should be tolerant. Configuration should be guarded.

---

## 2. Core Operating Flow

```text
GPIO Digital Input High/Low
    ↓
Input Normal/Triggered Evaluator
    ↓
Event State and Priority Resolver
    ↓
Configured Action
    ↓
Target EGM Selection
    ↓
Template-Based G2S Message Rendering
    ↓
Message Send / Observe / Retry
    ↓
Confirmation / Failure / Escalation
    ↓
Emergency Audit Timeline
    ↓
Return-to-Normal Action when signal clears
```

The software layer sees each hardwire input as a binary digital GPIO state only:

```text
High
Low
```

The EE input-stage flexibility, including wide voltage support, is intentionally hidden from the software layer. Software should not care whether the source signal originated as 3.3 VDC, 24 VDC, 120 VAC, 277 VAC, or any other supported electrical source. The normalized GPIO state is the contract.

---

## 3. Top-Level Product Areas

The product should be organized around these top-level UI/API areas:

```text
Live
Inputs
Actions
Comms
EGMs
Templates
Audit
Settings
```

### 3.1 Live

Purpose:

- show current appliance state,
- show current active input modes,
- show active actions,
- show failed or escalating EGMs,
- show whether floor state is normal, broadcast, emergency, or restoring.

The Live page is for fast operator understanding, not detailed configuration.

### 3.2 Inputs

Purpose:

- monitor and configure the four physical input channels.

Each input channel has:

| Field | Description |
|---|---|
| Enabled | Whether this input is active. |
| Name | Operator-facing label. Example: Emergency Broadcast. |
| GPIO channel | Hardware channel identity. |
| Current state | High or Low. |
| Normal state | Normal High or Normal Low. |
| Derived state | Normal or Triggered. |
| Debounce / hold time | Prevents bounce/chatter from creating repeated events. |
| Priority | Determines which signal wins if multiple inputs are active. |
| On-trigger action | Action run when Normal changes to Triggered. |
| On-normal action | Action run when Triggered changes to Normal. |
| Latching mode | Auto clear or require authorized manual clear. |
| Last transition | Last transition timestamp and audit link. |

Expected named modes:

- Regular Operation
- General Broadcast
- Emergency Broadcast
- Local Notice

These should be configurable names, not hardcoded assumptions.

### 3.3 Actions

Purpose:

- build the workflows that run when inputs change state.

An Action is a configured workflow, not just a single message.

Action fields:

| Field | Description |
|---|---|
| Action ID | Stable identifier. |
| Name | Operator-facing name. |
| Severity | Notice, broadcast, emergency, restore, maintenance. |
| Trigger source | Input transition or manual operator action. |
| Target selector | Which EGMs receive the action. |
| Template selector | Which G2S template or per-EGM template binding to use. |
| Message sequence | One or more message steps. |
| Confirmation rule | What counts as success. |
| Retry policy | Attempts, delay, backoff. |
| Escalation policy | What to do if confirmation fails. |
| Return action | Paired action for return to normal. |
| Audit policy | What must be captured. |
| Operator permissions | Who can edit, run, pause, or clear it. |

Example action:

```text
Action: Emergency Broadcast Silence
Triggered by: Input 3 Normal → Triggered
Targets: All emergency-enabled EGMs
Primary behavior: Send template-defined silence/mute command
Confirmation: Template-defined success response or observed state
Retries: 2 attempts
Escalation: Cabinet lock command if silence not confirmed
Return action: Restore Floor Audio
Audit severity: Emergency
```

### 3.4 Comms

Purpose:

- generic communications listener and message journal.

Every inbound and outbound message should be visible.

Message journal fields:

| Field | Description |
|---|---|
| Timestamp | Message time. |
| Direction | Inbound or outbound. |
| From | Source IP / endpoint / EGM. |
| To | Destination IP / endpoint / controller. |
| EGM ID | Parsed or assigned. |
| Message type | Best-effort classification. |
| Raw payload | Original message exactly as observed/sent. |
| Parsed summary | Best-effort extracted fields. |
| Action link | Action that caused the message, if any. |
| Input link | Input event that caused the action, if any. |
| Template used | Template and version. |
| Handler rule | Rule that interpreted the message. |
| Result | Sent, received, acked, confirmed, failed, ignored, escalated. |
| Operator notes | Optional manual notes. |

The operator should be able to select a message and define or refine how similar messages are handled in the future.

This should create handler rules, not source-code changes.

Example handler rule:

```text
Match:
  direction = inbound
  template = Aristocrat observed lab v1
  payload contains <vendorMuteAccepted>

Handle as:
  confirmation for action step Emergency Silence primary command
  mark target EGM as confirmed
  stop retry timer for this EGM
```

### 3.5 EGMs

Purpose:

- manage the EGM registry and template assignment.

EGM fields:

| Field | Description |
|---|---|
| EGM ID | Identity used in messages. |
| Display name | Operator-friendly label. |
| IP / endpoint | Target endpoint or observed endpoint. |
| Vendor | Manufacturer. |
| Cabinet family | Optional. |
| Game / title | Optional. |
| Software version | Optional. |
| Zone / group | For targeting. |
| Emergency enabled | Whether included in emergency actions. |
| Applied template | Default template for this EGM. |
| Heartbeat settings | Expected frequency / miss tolerance overrides. |
| Last communication | Last inbound/outbound message. |
| Current action state | Normal, pending, silenced, failed, escalating, restoring. |
| Notes | Field notes and quirks. |

### 3.6 Templates

Purpose:

- manage vendor/version behavior, raw G2S messages, parsing rules, confirmation rules, heartbeat quirks, and escalation instructions.

Template fields:

| Field | Description |
|---|---|
| Template ID | Stable ID. |
| Name | Example: IGT observed lab v1. |
| Vendor | Manufacturer or generic. |
| Version | Template version. |
| Applies to | Vendor, cabinet, software, or manually assigned EGMs. |
| Endpoint quirks | Path, SOAPAction, namespaces, headers, timeout behavior. |
| Primary actions | Message templates for silence/broadcast/local notice/etc. |
| Return actions | Restore/return-to-normal message templates. |
| Confirmation rules | What inbound/outbound response proves success. |
| Failure rules | What response means rejection or failure. |
| Escalation actions | Stronger fallback commands. |
| Heartbeat profile | Expected frequency and template-specific tolerance. |
| Variables | Host ID, EGM ID, timestamp, action ID, etc. |
| Notes/evidence | What real machine/analyzer evidence supports it. |

Template changes must be versioned and audited.

Do not silently edit active templates in place during emergency use. Clone/version first.

### 3.7 Audit

Purpose:

- provide emergency timeline, message linkage, and exportable evidence.

Audit timeline should include:

| Event | Example |
|---|---|
| Input transition | Input 3 changed Normal → Triggered. |
| Event priority decision | Emergency Broadcast became active mode. |
| Action start | Emergency Broadcast Silence started. |
| Target selection | 112 EGMs selected. |
| Message render | Template generated outbound command for EGM-001. |
| Message send | Outbound message sent. |
| Response receive | Inbound ACK or vendor response received. |
| Confirmation | EGM-001 confirmed silenced. |
| Retry | Attempt 2 sent after timeout. |
| Escalation | Cabinet lock fallback sent. |
| Failure | EGM-009 not confirmed after escalation. |
| Return to normal | Input cleared; restore action started. |
| Operator action | Operator acknowledged / exported / annotated. |

Audit exports should be available per incident.

### 3.8 Settings

Settings should be organized into these sections:

#### Appliance / Network / Certificates

- appliance controller ID,
- host name,
- bind address,
- G2S listener URL,
- network interface status,
- server certificates,
- client certificates,
- CA/trust material,
- certificate expiration status,
- time sync status,
- production/lab mode,
- key export policy.

#### Inputs / Events

- input channel defaults,
- normal state behavior,
- debounce defaults,
- event priorities,
- latching defaults.

#### Actions / Escalation / Buzzer

- global retry defaults,
- emergency escalation defaults,
- global heartbeat miss tolerance modifier,
- buzzer rules,
- alert thresholds,
- manual acknowledge behavior,
- emergency timeout policy.

#### EGMs

- default EGM settings,
- registry import/export,
- group/zone defaults,
- emergency inclusion defaults,
- template assignment defaults.

#### Templates

- template storage,
- template import/export,
- raw message editing policy,
- versioning policy,
- validation profile.

#### History / Timeline

- message retention,
- audit retention,
- raw payload retention,
- export package format,
- timeline filtering defaults,
- storage limits.

---

## 4. Backend Architecture Direction

The backend should be reorganized around the product flow, not around the old dashboard/scaffold.

Recommended package layout:

```text
cmd/g2s-mute/
  main.go

internal/api/
  routes.go
  inputs_handler.go
  actions_handler.go
  comms_handler.go
  egms_handler.go
  templates_handler.go
  audit_handler.go
  settings_handler.go

internal/inputs/
  channel.go
  gpio_reader.go
  evaluator.go
  debounce.go
  transitions.go

internal/actions/
  definition.go
  executor.go
  target_selector.go
  retry_policy.go
  escalation_policy.go
  return_to_normal.go

internal/g2sengine/
  message.go
  template_renderer.go
  sender.go
  listener.go
  loose_parser.go
  handler_rules.go
  confirmation.go
  vendor_quirks.go

internal/egms/
  registry.go
  groups.go
  template_assignment.go
  state.go

internal/templates/
  template.go
  versioning.go
  validation.go
  import_export.go

internal/audit/
  timeline.go
  message_journal.go
  incident.go
  export.go

internal/store/
  sqlite.go
  migrations.go
  inputs_store.go
  actions_store.go
  messages_store.go
  templates_store.go
  audit_store.go

internal/config/
  keep, but reorganize as needed

internal/model/
  keep, but move toward focused domain models

internal/ui/
  new server shell only

web/
  frontend source if using TypeScript/Vite
```

### 4.1 `cmd/g2s-mute/main.go` rule

`main.go` should only:

- load config,
- open the database,
- construct services,
- wire routes,
- start the HTTP/G2S servers,
- handle shutdown.

It should not own business logic.

Guardrail:

- If `main.go` exceeds roughly 250 lines, review architecture before adding more.

### 4.2 G2S engine rule

The G2S layer should not be compliance-first. It should be action-first and template-driven.

It must support:

- raw outbound templates,
- variable substitution,
- loose inbound parsing,
- handler rules,
- vendor quirks,
- confirmation detection,
- failure detection,
- retry/escalation integration,
- raw message journal.

Strict validation can exist as a diagnostic tool, not as the emergency runtime gate.

### 4.3 Input engine rule

Inputs are authoritative physical state.

The input engine owns:

- reading high/low state,
- normal/triggered derivation,
- debounce,
- priority evaluation,
- event generation,
- audit of transitions.

The action engine decides what to do with events.

### 4.4 Action engine rule

The action engine owns:

- action execution,
- target selection,
- per-EGM execution state,
- retries,
- escalation,
- return-to-normal,
- audit linkage.

The action engine should not contain vendor-specific XML. It asks the template/G2S engine to render and send.

---

## 5. Frontend Direction

Rebuild the frontend.

Do not continue with a giant embedded JavaScript blob as the source of truth.

Recommended options:

### Option 1: TypeScript frontend

Use:

```text
web/
  package.json
  src/
```

Build into static assets embedded by Go.

Pros:

- better structure,
- safer refactoring,
- typed API models,
- easier complex UI like action builder and template editor.

Cons:

- adds frontend build step.

### Option 2: Server-rendered UI with minimal JS

Use Go templates plus small JS modules.

Pros:

- simple appliance deployment,
- less tooling.

Cons:

- action builder/template editor may become messy again if not carefully controlled.

### Recommendation

Use **TypeScript with a minimal build** if time allows. The product now needs a real action builder, message journal, template management, and rules editor. Those are complex enough that raw embedded JS will likely go haywire again.

Guardrails:

- no giant generated UI file committed as hand-edited source,
- no product logic trapped inside static asset blobs,
- API response models documented,
- UI talks to backend through stable APIs,
- operator help is curated separately from project docs.

---

## 6. Data Model Direction

Core entities:

```text
InputChannel
InputTransition
EventMode
ActionDefinition
ActionRun
ActionStep
ActionTargetResult
G2STemplate
G2STemplateVersion
MessageJournalEntry
HandlerRule
EGMRecord
EGMGroup
Incident
AuditTimelineEntry
SettingsProfile
```

### 6.1 InputChannel

```text
id
name
gpio_channel
enabled
normal_state_high_low
debounce_ms
priority
on_trigger_action_id
on_normal_action_id
latching_mode
current_state
derived_state
last_transition_at
```

### 6.2 ActionDefinition

```text
id
name
severity
enabled
target_selector
template_selector
steps
retry_policy
escalation_policy
return_action_id
audit_policy
created_at
updated_at
version
```

### 6.3 ActionRun

```text
id
action_definition_id
incident_id
input_transition_id
started_at
completed_at
status
trigger_reason
target_count
confirmed_count
failed_count
escalated_count
```

### 6.4 MessageJournalEntry

```text
id
timestamp
direction
from_endpoint
to_endpoint
egm_id
action_run_id
action_step_id
input_transition_id
template_id
template_version
handler_rule_id
message_type
raw_payload
parsed_summary_json
result
error
```

### 6.5 G2STemplate

```text
id
name
vendor
cabinet_family
software_version_match
version
status
endpoint_quirks_json
actions_json
confirmation_rules_json
failure_rules_json
heartbeat_profile_json
notes
created_at
updated_at
```

### 6.6 EGMRecord

```text
egm_id
display_name
ip_address
endpoint_path
vendor
cabinet_family
game_title
software_version
zone
enabled
emergency_enabled
template_id
heartbeat_override_json
last_seen_at
current_action_state
notes
```

---

## 7. Guardrails: Where the Old Project Went Haywire

### 7.1 No project docs inside the product

Project docs belong in:

```text
docs/project-definition/
docs/architecture/
docs/runbooks/
docs/research/
```

Operator-facing help belongs in:

```text
docs/operator/
```

The running product should not embed project plans, research dumps, or development notes unless a deliberate curated help feature is created.

Rule:

- Project docs are for builders.
- Operator docs are for users.
- Runtime product screens are for operation.

### 7.2 No giant UI asset as source of truth

Generated assets are acceptable only as build output. They should not be manually edited as the main frontend.

Rule:

- Frontend source must live in `web/src` or structured Go templates.
- Generated bundled assets must be clearly marked.
- Product behavior should not be hidden in a 4,000+ line embedded JS file.

### 7.3 No God `main.go`

`main.go` is composition only.

Business logic goes into packages.

### 7.4 No direct merges to `main`

All work through PRs after repo recovery.

### 7.5 No conflict markers

CI must fail on Git conflict markers:
- seven less-than signs at the start of a line,
- seven equals signs on a separator line,
- seven greater-than signs at the start of a line.

### 7.6 No simulator as product goal

Drop fake EGM virtualization from product scope.

Allowed:

- unit tests,
- parser tests,
- template rendering tests,
- recorded-message fixtures,
- external analyzer import/export,
- real-machine testing.

Not allowed as core product work:

- building a virtual EGM floor,
- optimizing loopback workflows,
- UI flows built around fake EGMs.

### 7.7 No compliance blocker in emergency path

Compliance/validation can warn, diagnose, and improve template creation. It should not block a configured emergency action simply because it is non-standard.

### 7.8 No silent template changes

Template edits must be versioned and audited.

Active emergency templates should not be mutated in place without a new version.

### 7.9 No unaudited emergency changes

These must be audited:

- input mapping changes,
- action changes,
- template changes,
- EGM emergency enable/disable,
- manual action runs,
- manual clears,
- handler rule changes,
- certificate/network changes affecting message delivery.

### 7.10 No hidden target selection

Before saving an action, show what targets it would affect.

Before running a manual action, show what targets it will affect.

During automatic emergency action, log what targets were selected and why.

### 7.11 No irreversible action without return plan

Every emergency action should have an explicit return-to-normal action or a documented reason why manual restoration is required.

### 7.12 No ambiguous priority behavior

If multiple inputs are triggered, priority must be deterministic.

Example default priority:

```text
Emergency Broadcast
General Broadcast
Local Notice
Regular Operation
```

This should be configurable but always deterministic.

---

## 8. Minimum Viable Rebuild Scope

Because time to deployment matters, the first field-testable version should be narrow.

### Must have

1. Four digital GPIO input channels
2. Normal High / Normal Low configuration
3. Input transition audit
4. Action builder lite
5. EGM registry
6. Template assignment per EGM
7. Raw G2S message template rendering
8. Message send and receive journal
9. Confirmation/failure rule matching
10. Retry and escalation
11. Return-to-normal action
12. Emergency audit timeline
13. Network/certificate settings
14. Exportable emergency evidence

### Can wait

- polished dashboards,
- virtual EGM simulation,
- full schema validation,
- advanced analytics,
- role-heavy user management,
- template marketplace,
- broad compatibility database,
- automated vendor detection,
- fancy charts.

---

## 9. Implementation Phases

### Phase 0: Repo Recovery and Project Definition

Goal:

- make the repo safe,
- add this document,
- create a green rebuild base.

Deliverables:

- protected `main`,
- clean rebuild branch,
- conflict marker CI,
- `go test ./...` passing,
- project definition doc committed.

Exit criteria:

- no conflict markers,
- tests pass,
- old UI identified as legacy,
- rebuild branch created.

### Phase 1: Domain Models and Storage

Goal:

- create the new data model without rebuilding the whole UI yet.

Deliverables:

- input channel model,
- action definition model,
- EGM registry model,
- template model,
- message journal model,
- audit timeline model,
- SQLite migrations,
- CRUD APIs.

Exit criteria:

- APIs can create/read/update core definitions,
- audit records are written for changes,
- no UI dependency yet.

### Phase 2: Input Engine

Goal:

- read four binary GPIO channels and generate transition events.

Deliverables:

- GPIO abstraction,
- high/low state reader,
- normal/triggered evaluator,
- debounce,
- priority resolver,
- input transition audit.

Exit criteria:

- physical input change creates a durable audit transition,
- normal/triggered state is visible through API,
- action is not required yet.

### Phase 3: Action Engine Lite

Goal:

- map input transitions to actions.

Deliverables:

- action executor,
- target selector,
- retry policy,
- escalation policy,
- return-to-normal link,
- action run audit.

Exit criteria:

- input trigger starts configured action run,
- input clear starts configured return action,
- all runs are visible in audit timeline.

### Phase 4: Template-Based G2S Engine

Goal:

- send configurable messages to EGMs using templates.

Deliverables:

- raw message template renderer,
- variable substitution,
- outbound sender,
- inbound listener/journal,
- loose parser,
- confirmation/failure matcher,
- handler rules.

Exit criteria:

- action can render and send template-defined messages,
- inbound/outbound traffic is journaled,
- confirmation can update target result,
- failure can trigger retry/escalation.

### Phase 5: Minimal Operator UI

Goal:

- provide the essential screens for field testing.

Deliverables:

- Live page,
- Inputs page,
- Action Builder Lite,
- EGM registry,
- Template editor lite,
- Comms journal,
- Audit timeline,
- Settings page.

Exit criteria:

- field operator can configure inputs/actions/templates/EGMs,
- trigger an emergency action,
- observe comms,
- export evidence,
- return to normal.

### Phase 6: Field Hardening

Goal:

- make it reliable with real machines and analyzers.

Deliverables:

- template import/export,
- handler rule refinement,
- message replay from captured analyzer logs,
- stronger certificate diagnostics,
- deployment scripts,
- backup/restore,
- operator runbook.

Exit criteria:

- real-machine test produces a complete audit package,
- at least one emergency and return-to-normal cycle is captured,
- template changes from field data are versioned and reproducible.

---

## 10. Testing Strategy

Drop product-level EGM simulation, but keep automated testing where it matters.

### Keep

- input evaluator tests,
- debounce tests,
- action executor tests,
- target selector tests,
- template renderer tests,
- confirmation matcher tests,
- retry/escalation tests,
- audit write/read tests,
- API contract tests,
- recorded-message fixture tests.

### Drop

- virtual EGM floor as a product goal,
- loopback-driven UI workflows,
- fake EGM features presented as real operator tools.

### Add

Recorded message fixtures from external analyzers and real machines:

```text
testdata/messages/
  vendor_a/
    emergency_silence_ack.xml
    restore_ack.xml
  vendor_b/
    mute_rejected.xml
    cabinet_lock_success.xml
```

These are not “simulated EGMs.” They are captured evidence used to test parsers and handler rules.

---

## 11. First PR Breakdown

### PR 1: Repo recovery

- remove conflict markers,
- restore `go test ./...`,
- add CI conflict-marker scan,
- add this document.

### PR 2: Package boundary skeleton

- create new packages,
- move no logic yet,
- add README files explaining package ownership.

### PR 3: Core data models and migrations

- inputs,
- actions,
- templates,
- message journal,
- audit timeline.

### PR 4: Input engine

- GPIO abstraction,
- input state API,
- transitions and audit.

### PR 5: Action engine lite

- action definitions,
- action run state,
- input-to-action binding.

### PR 6: Template renderer and message journal

- render raw messages,
- journal outbound/inbound,
- no full UI yet.

### PR 7: Minimal UI shell

- new UI structure,
- Live / Inputs / Actions / Comms / EGMs / Templates / Audit / Settings.

### PR 8: End-to-end field-test path

- input trigger,
- action run,
- template send,
- message journal,
- confirmation/failure,
- return-to-normal,
- audit export.

---

## 12. Acceptance Criteria for First Field-Test Build

A first field-test build is acceptable when:

1. Four GPIO inputs show live High/Low state.
2. Each input can be configured Normal High or Normal Low.
3. Normal → Triggered creates an audit event.
4. Triggered → Normal creates an audit event.
5. Each input can run a configured on-trigger action.
6. Each input can run a configured on-normal action.
7. EGMs can be registered and assigned templates.
8. A template can render an outbound G2S message using EGM ID, host ID, timestamp, and action ID.
9. Outbound messages are recorded with raw payload.
10. Inbound messages are recorded with raw payload.
11. Confirmation/failure rules can update action target result.
12. Retry and escalation can be configured.
13. Emergency action run produces a timeline.
14. Return-to-normal produces a timeline.
15. Audit evidence can be exported.
16. Project docs are not embedded in the operator product.
17. No fake EGM workflow is required for field operation.
18. `go test ./...` passes.
19. Conflict-marker scan passes.
20. Active branch is protected through PR/CI process.

---

## 13. Open Decisions

These should be resolved before detailed implementation begins:

1. Frontend technology:
   - TypeScript/Vite or server-rendered Go templates?

2. GPIO library:
   - Which Linux GPIO interface will be used on the Pi target?

3. Message sending direction:
   - For each target EGM/vendor, does the appliance initiate outbound HTTP/SOAP, respond as a host listener, or both?

4. Emergency latching:
   - Should Emergency Broadcast require manual clear even after the input returns normal?

5. Priority defaults:
   - Confirm default priority among Regular, Local Notice, General Broadcast, Emergency Broadcast.

6. Cabinet lock escalation:
   - Which templates should be allowed to include lock escalation?
   - Is operator approval needed to configure this?

7. Audit retention:
   - How long should raw payloads and emergency evidence be retained?

8. Operator access:
   - What is the minimum field-test auth model?

9. Template editing:
   - Who can edit active templates?
   - Should production mode require clone/version before edit?

10. Return-to-normal safety:
   - What should happen if restore fails after emergency clears?

---

## 14. Summary Direction

Use the existing repo, but do not continue the current architecture blindly.

The rebuild should be centered on:

```text
Inputs → Actions → Templates → G2S Messages → Confirmation/Escalation → Audit
```

The front end should be rebuilt.

The backend should be salvaged selectively and reorganized.

G2S should be treated as a configurable transport to accomplish the life-safety action, not as a standards-compliance gate.

The project docs should live in the repo, but project docs must not become product runtime content.

The first implementation step is not coding the UI. The first implementation step is making the repository safe, green, protected, and aligned around this product definition.
