# Phase 1A: Domain Foundation and API Contracts

**Branch:** `rebuild/input-action-engine`  
**Scope:** one small implementation slice / one PR  
**Intent:** establish the new product foundation without changing runtime behavior.

Phase 1A establishes the core product flow:

```text
Inputs → Actions → Templates → Messages → Audit
```

This phase should create compile-safe packages, domain models, validation, additive SQLite migrations, minimal store methods, and API contract stubs. It should not rebuild the UI or execute real actions yet.

---

## Guardrails

Do **not**:

- expand `cmd/g2s-mute/main.go` beyond route/service wiring,
- build or modify the operator UI,
- add fake or virtual EGM product features,
- embed project docs into runtime UI,
- make G2S validation block emergency behavior,
- remove existing passing tests,
- break `go test ./...`.

Do:

- add small packages with clear ownership,
- keep migrations additive,
- keep models explicit and boring,
- add validation tests,
- add store tests where practical,
- document package boundaries with README files.

---

## Deliverables

### 1. Package skeleton

Create:

```text
internal/inputs
internal/actions
internal/g2sengine
internal/egms
internal/templates
internal/audit
internal/api
```

Each package should include:

```text
README.md
doc.go
```

Ownership:

| Package | Owns | Does not own |
| --- | --- | --- |
| `inputs` | GPIO input domain, normal/triggered state, transitions | G2S messages, action execution |
| `actions` | action definitions, action runs, steps, target results | raw XML templates, GPIO reads |
| `g2sengine` | rendered messages, handler rules, loose parsing contracts | EGM registry ownership, UI |
| `egms` | EGM registry/domain grouping | action execution |
| `templates` | template metadata/version/domain validation | runtime sending |
| `audit` | audit timeline and message journal domain | business decision logic |
| `api` | request/response contracts and route stubs | business logic |

---

## 2. Input channel domain model

File: `internal/inputs/model.go`

Required concepts:

- `DigitalState`: `HIGH`, `LOW`
- `DerivedInputState`: `NORMAL`, `TRIGGERED`
- `InputLatchMode`: `AUTO_CLEAR`, `MANUAL_CLEAR`
- `InputChannel`
- `InputTransition`

Required helpers:

```go
func (c InputChannel) Validate() error
func DeriveState(raw DigitalState, normal DigitalState) (DerivedInputState, error)
```

Validation rules:

- `ID` required.
- `Name` required.
- `GPIOChannel` required.
- `NormalState` must be `HIGH` or `LOW`.
- `DebounceMS >= 0`.
- `Priority >= 0`.
- `LatchMode` must be known.

Tests:

- High normal + high raw = normal.
- High normal + low raw = triggered.
- Low normal + low raw = normal.
- Invalid digital state errors.
- Missing ID/name/GPIO rejected.

---

## 3. Action definition domain model

File: `internal/actions/model.go`

Required concepts:

- `ActionSeverity`: `NOTICE`, `BROADCAST`, `EMERGENCY`, `RESTORE`, `MAINTENANCE`
- `ActionStatus`: `PENDING`, `RUNNING`, `SUCCEEDED`, `FAILED`, `ESCALATING`, `CANCELLED`
- `TargetSelectorType`: `ALL_EMERGENCY_ENABLED`, `EGM_IDS`, `GROUP`, `TEMPLATE`
- `RetryPolicy`
- `EscalationPolicy`
- `ActionDefinition`
- `TargetSelector`
- `ActionStep`
- `ActionRun`
- `ActionTargetResult`

Required helpers:

```go
func (a ActionDefinition) Validate() error
func (p RetryPolicy) Normalize() RetryPolicy
```

Validation rules:

- `ID` required.
- `Name` required.
- `Severity` must be known.
- `TargetSelector.Type` must be known.
- `RetryPolicy.MaxAttempts >= 1` after normalization.
- `RetryPolicy.DelayMS >= 0`.
- Step IDs must be unique inside an action.
- Step ordering must be deterministic.

---

## 4. G2S template and engine domain models

Files:

```text
internal/templates/model.go
internal/g2sengine/model.go
```

Template concepts:

- `TemplateStatus`: `DRAFT`, `ACTIVE`, `ARCHIVED`
- `G2STemplate`
- `G2STemplateVersion`

Engine concepts:

- `MessageDirection`: `INBOUND`, `OUTBOUND`
- `MessageResult`: `RECEIVED`, `SENT`, `CONFIRMED`, `FAILED`, `IGNORED`, `ESCALATED`
- `HandlerRule`

Validation rules:

- Template ID/name required.
- Template version must be positive.
- Status must be known.
- JSON fields may be empty, but must be valid JSON if present.
- Handler rule ID/name required.
- Handler rule priority non-negative.

---

## 5. Message journal domain model

File: `internal/audit/message_journal.go`

Required concept:

- `MessageJournalEntry`

Fields should include:

- timestamp,
- direction,
- from endpoint,
- to endpoint,
- EGM ID,
- action run ID,
- action step ID,
- input transition ID,
- template ID,
- template version,
- handler rule ID,
- message type,
- raw payload,
- parsed summary JSON,
- result,
- error.

Validation rules:

- Timestamp required.
- Direction required.
- Result must be known if present.
- Template version non-negative.
- Parsed summary JSON must be valid if non-empty.

---

## 6. Audit timeline domain model

File: `internal/audit/timeline.go`

Required concepts:

- `TimelineEventType`
- `AuditSeverity`
- `AuditTimelineEntry`

Timeline event types:

```text
INPUT_TRANSITION
ACTION_STARTED
ACTION_STEP
MESSAGE_SENT
MESSAGE_RECEIVED
CONFIRMATION
RETRY
ESCALATION
RETURN_TO_NORMAL
OPERATOR_ACTION
SYSTEM_WARNING
```

Validation rules:

- Timestamp required.
- Type known.
- Severity known.
- Summary required.
- Metadata JSON valid if non-empty.

---

## 7. EGM registry domain model

File: `internal/egms/model.go`

Required concepts:

- `EGMRecord`
- `EGMGroup`

Validation rules:

- `EGMID` required.
- Emergency-enabled EGM should also be enabled.
- Heartbeat override JSON valid if non-empty.
- Group ID/name required.
- Group EGM IDs unique.

---

## 8. SQLite migrations

Add additive migrations only. Do not remove or rewrite existing tables.

Preferred approach:

```go
const RebuildPhase1AMigration = `...`
```

Call it from `SQLiteStore.Migrate` after the existing migration.

New tables:

```text
input_channels
input_transitions
action_definitions
action_runs
action_target_results
g2s_templates
g2s_template_versions
message_journal
handler_rules
egm_records
egm_groups
audit_timeline
```

Migration tests must verify:

- migration runs on empty DB,
- migration is idempotent,
- all new tables exist,
- basic insert/select works for at least input channel, action definition, G2S template, message journal entry, and audit timeline entry.

---

## 9. Minimal store methods

Suggested file: `internal/store/rebuild_phase1a.go`

Add minimal methods:

```go
func (s *SQLiteStore) UpsertInputChannel(ctx context.Context, channel inputs.InputChannel) error
func (s *SQLiteStore) ListInputChannels(ctx context.Context) ([]inputs.InputChannel, error)

func (s *SQLiteStore) UpsertActionDefinition(ctx context.Context, action actions.ActionDefinition) error
func (s *SQLiteStore) ListActionDefinitions(ctx context.Context) ([]actions.ActionDefinition, error)

func (s *SQLiteStore) UpsertG2STemplate(ctx context.Context, template templates.G2STemplate) error
func (s *SQLiteStore) ListG2STemplates(ctx context.Context) ([]templates.G2STemplate, error)

func (s *SQLiteStore) RecordMessageJournalEntry(ctx context.Context, entry audit.MessageJournalEntry) (int64, error)
func (s *SQLiteStore) ListMessageJournalEntries(ctx context.Context, limit int) ([]audit.MessageJournalEntry, error)

func (s *SQLiteStore) RecordAuditTimelineEntry(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error)
func (s *SQLiteStore) ListAuditTimelineEntries(ctx context.Context, limit int) ([]audit.AuditTimelineEntry, error)
```

Keep serialization simple. JSON fields may be stored as text.

---

## 10. API contract stubs

Package: `internal/api`

Files:

```text
routes.go
contracts.go
```

Route contracts:

```text
GET    /api/v2/inputs
PUT    /api/v2/inputs/{id}
GET    /api/v2/actions
PUT    /api/v2/actions/{id}
GET    /api/v2/templates
PUT    /api/v2/templates/{id}
GET    /api/v2/egms
PUT    /api/v2/egms/{id}
GET    /api/v2/comms/messages
GET    /api/v2/audit/timeline
```

For Phase 1A, stubs may return `501 Not Implemented` or empty store-backed results if that remains clean.

Prefer not to wire these into `cmd/g2s-mute/main.go` yet unless the wiring is tiny and clean.

---

## 11. Required tests

Phase 1A must pass:

```bash
go test ./...
```

New tests should include:

```text
internal/inputs/model_test.go
internal/actions/model_test.go
internal/templates/model_test.go
internal/g2sengine/model_test.go
internal/egms/model_test.go
internal/audit/model_test.go
internal/store/rebuild_phase1a_test.go
```

Minimum expectations:

- validation rejects bad required fields,
- validation accepts minimal valid objects,
- migration runs twice successfully,
- new tables exist,
- one insert/list cycle works for key store methods.

---

## Definition of done

Phase 1A is done when:

1. New packages exist and compile.
2. Domain models are defined.
3. Domain validation exists and is tested.
4. SQLite migration adds the new rebuild tables.
5. Migration is idempotent.
6. Minimal store methods exist for key entities.
7. API contract structs and route stubs exist.
8. No UI rebuild has started.
9. No fake EGM simulation work has been added.
10. `go test ./...` passes.
11. Conflict-marker scan passes.
12. PR description clearly says: `Domain foundation only; no runtime behavior change intended.`

---

## Codex handoff prompt

```text
You are working in repo tschneider-imagine/G2S_MC on branch rebuild/input-action-engine.

Implement Phase 1A only: domain foundation and API contracts.

Read:
docs/project-definition/G2S_MC_REBUILD_PROJECT_DEFINITION_AND_GUARDRAILS.md
docs/project-definition/PHASE_1A_DOMAIN_FOUNDATION.md

Do not modify legacy UI except if required for compilation.
Do not add fake or virtual EGM product features.
Do not embed project docs into runtime UI.
Do not expand cmd/g2s-mute/main.go beyond route/service wiring.
Do not change runtime behavior intentionally.
Do not remove existing tests.

Create package skeletons:
internal/inputs
internal/actions
internal/g2sengine
internal/egms
internal/templates
internal/audit
internal/api

Each package should have README.md and doc.go.

Add domain models and validation for:
- InputChannel
- InputTransition
- ActionDefinition
- ActionRun
- ActionStep
- ActionTargetResult
- G2STemplate
- G2STemplateVersion
- HandlerRule
- EGMRecord
- EGMGroup
- MessageJournalEntry
- AuditTimelineEntry

Add additive SQLite migrations for:
- input_channels
- input_transitions
- action_definitions
- action_runs
- action_target_results
- g2s_templates
- g2s_template_versions
- message_journal
- handler_rules
- egm_records
- egm_groups
- audit_timeline

Add minimal store methods for:
- Upsert/List input channels
- Upsert/List action definitions
- Upsert/List G2S templates
- Record/List message journal entries
- Record/List audit timeline entries

Add internal/api contract structs and route stubs for:
- /api/v2/inputs
- /api/v2/actions
- /api/v2/templates
- /api/v2/egms
- /api/v2/comms/messages
- /api/v2/audit/timeline

Prefer not to wire these into cmd/g2s-mute/main.go in Phase 1A unless it is very small and clean.

Add tests for validation and migration idempotence.
Run gofmt.
Run go test ./...
Keep the PR small and focused.
```
