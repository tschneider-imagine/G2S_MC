package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type MessageJournalListQuery struct {
	Limit             int
	EGMID             string
	ActionRunID       string
	IncidentID        string
	InputTransitionID int64
	Direction         g2sengine.MessageDirection
	Results           []g2sengine.MessageResult
}

type AuditTimelineListQuery struct {
	Limit             int
	EventType         string
	Severity          audit.AuditSeverity
	ActionRunID       string
	IncidentID        string
	InputTransitionID int64
}

type ActionRunListQuery struct {
	Limit              int
	Status             actions.ActionRunStatus
	ActionDefinitionID string
	IncidentID         string
	InputTransitionID  int64
}

type HandlerRuleListQuery struct {
	EnabledOnly bool
	Limit       int
}

func (s *SQLiteStore) UpsertInputChannel(ctx context.Context, channel inputs.InputChannel) error {
	if err := channel.Validate(); err != nil {
		return err
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO input_channels (
		    id, name, gpio_channel, enabled, normal_state, current_state, derived_state,
		    debounce_ms, priority, on_trigger_action_id, on_normal_action_id, latching_mode,
		    last_transition_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
		    name = excluded.name,
		    gpio_channel = excluded.gpio_channel,
		    enabled = excluded.enabled,
		    normal_state = excluded.normal_state,
		    current_state = excluded.current_state,
		    derived_state = excluded.derived_state,
		    debounce_ms = excluded.debounce_ms,
		    priority = excluded.priority,
		    on_trigger_action_id = excluded.on_trigger_action_id,
		    on_normal_action_id = excluded.on_normal_action_id,
		    latching_mode = excluded.latching_mode,
		    last_transition_at = excluded.last_transition_at,
		    updated_at = CURRENT_TIMESTAMP`,
		strings.TrimSpace(channel.ID),
		strings.TrimSpace(channel.Name),
		strings.TrimSpace(channel.GPIOChannel),
		boolToInt(channel.Enabled),
		channel.NormalState,
		channel.CurrentState,
		channel.DerivedState,
		channel.DebounceMS,
		channel.Priority,
		nullableTrimmed(channel.OnTriggerActionID),
		nullableTrimmed(channel.OnNormalActionID),
		channel.LatchingMode,
		nullableTime(channel.LastTransitionAt),
	)
	return err
}

func (s *SQLiteStore) ListInputChannels(ctx context.Context) ([]inputs.InputChannel, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, name, gpio_channel, enabled, normal_state, current_state, derived_state,
		        debounce_ms, priority, COALESCE(on_trigger_action_id, ''), COALESCE(on_normal_action_id, ''),
		        latching_mode, last_transition_at, created_at, updated_at
		   FROM input_channels
		   ORDER BY priority DESC, id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []inputs.InputChannel{}
	for rows.Next() {
		var channel inputs.InputChannel
		var enabled int
		var lastTransitionAt sql.NullTime
		if err := rows.Scan(
			&channel.ID,
			&channel.Name,
			&channel.GPIOChannel,
			&enabled,
			&channel.NormalState,
			&channel.CurrentState,
			&channel.DerivedState,
			&channel.DebounceMS,
			&channel.Priority,
			&channel.OnTriggerActionID,
			&channel.OnNormalActionID,
			&channel.LatchingMode,
			&lastTransitionAt,
			&channel.CreatedAt,
			&channel.UpdatedAt,
		); err != nil {
			return nil, err
		}
		channel.Enabled = enabled != 0
		if lastTransitionAt.Valid {
			value := lastTransitionAt.Time
			channel.LastTransitionAt = &value
		}
		result = append(result, channel)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) GetInputChannel(ctx context.Context, id string) (*inputs.InputChannel, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, gpio_channel, enabled, normal_state, current_state, derived_state,
		        debounce_ms, priority, COALESCE(on_trigger_action_id, ''), COALESCE(on_normal_action_id, ''),
		        latching_mode, last_transition_at, created_at, updated_at
		   FROM input_channels
		  WHERE id = ?`,
		strings.TrimSpace(id),
	)
	var channel inputs.InputChannel
	var enabled int
	var lastTransitionAt sql.NullTime
	if err := row.Scan(
		&channel.ID,
		&channel.Name,
		&channel.GPIOChannel,
		&enabled,
		&channel.NormalState,
		&channel.CurrentState,
		&channel.DerivedState,
		&channel.DebounceMS,
		&channel.Priority,
		&channel.OnTriggerActionID,
		&channel.OnNormalActionID,
		&channel.LatchingMode,
		&lastTransitionAt,
		&channel.CreatedAt,
		&channel.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	channel.Enabled = enabled != 0
	if lastTransitionAt.Valid {
		value := lastTransitionAt.Time
		channel.LastTransitionAt = &value
	}
	return &channel, nil
}

func (s *SQLiteStore) UpsertActionDefinition(ctx context.Context, definition actions.ActionDefinition) error {
	if err := definition.Validate(); err != nil {
		return err
	}
	stepsJSON, err := json.Marshal(definition.Steps)
	if err != nil {
		return fmt.Errorf("marshal steps: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO action_definitions (
		    id, name, severity, enabled, target_selector, template_selector, steps_json,
		    retry_policy_json, escalation_policy_json, return_action_id, audit_policy_json,
		    version, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
		    name = excluded.name,
		    severity = excluded.severity,
		    enabled = excluded.enabled,
		    target_selector = excluded.target_selector,
		    template_selector = excluded.template_selector,
		    steps_json = excluded.steps_json,
		    retry_policy_json = excluded.retry_policy_json,
		    escalation_policy_json = excluded.escalation_policy_json,
		    return_action_id = excluded.return_action_id,
		    audit_policy_json = excluded.audit_policy_json,
		    version = excluded.version,
		    updated_at = CURRENT_TIMESTAMP`,
		strings.TrimSpace(definition.ID),
		strings.TrimSpace(definition.Name),
		definition.Severity,
		boolToInt(definition.Enabled),
		strings.TrimSpace(definition.TargetSelector),
		strings.TrimSpace(definition.TemplateSelector),
		string(stepsJSON),
		nullableTrimmed(definition.RetryPolicyJSON),
		nullableTrimmed(definition.EscalationJSON),
		nullableTrimmed(definition.ReturnActionID),
		nullableTrimmed(definition.AuditPolicyJSON),
		definition.Version,
	)
	return err
}

func (s *SQLiteStore) ListActionDefinitions(ctx context.Context) ([]actions.ActionDefinition, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, name, severity, enabled, target_selector, template_selector, steps_json,
		        COALESCE(retry_policy_json, ''), COALESCE(escalation_policy_json, ''), COALESCE(return_action_id, ''),
		        COALESCE(audit_policy_json, ''), version, created_at, updated_at
		   FROM action_definitions
		   ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	definitions := []actions.ActionDefinition{}
	for rows.Next() {
		var definition actions.ActionDefinition
		var enabled int
		var stepsJSON string
		if err := rows.Scan(
			&definition.ID,
			&definition.Name,
			&definition.Severity,
			&enabled,
			&definition.TargetSelector,
			&definition.TemplateSelector,
			&stepsJSON,
			&definition.RetryPolicyJSON,
			&definition.EscalationJSON,
			&definition.ReturnActionID,
			&definition.AuditPolicyJSON,
			&definition.Version,
			&definition.CreatedAt,
			&definition.UpdatedAt,
		); err != nil {
			return nil, err
		}
		definition.Enabled = enabled != 0
		if err := json.Unmarshal([]byte(stepsJSON), &definition.Steps); err != nil {
			return nil, fmt.Errorf("unmarshal steps_json for action %q: %w", definition.ID, err)
		}
		definitions = append(definitions, definition)
	}
	return definitions, rows.Err()
}

func (s *SQLiteStore) GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, severity, enabled, target_selector, template_selector, steps_json,
		        COALESCE(retry_policy_json, ''), COALESCE(escalation_policy_json, ''), COALESCE(return_action_id, ''),
		        COALESCE(audit_policy_json, ''), version, created_at, updated_at
		   FROM action_definitions
		  WHERE id = ?`,
		strings.TrimSpace(id),
	)
	var definition actions.ActionDefinition
	var enabled int
	var stepsJSON string
	if err := row.Scan(
		&definition.ID,
		&definition.Name,
		&definition.Severity,
		&enabled,
		&definition.TargetSelector,
		&definition.TemplateSelector,
		&stepsJSON,
		&definition.RetryPolicyJSON,
		&definition.EscalationJSON,
		&definition.ReturnActionID,
		&definition.AuditPolicyJSON,
		&definition.Version,
		&definition.CreatedAt,
		&definition.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	definition.Enabled = enabled != 0
	if err := json.Unmarshal([]byte(stepsJSON), &definition.Steps); err != nil {
		return nil, fmt.Errorf("unmarshal steps_json for action %q: %w", definition.ID, err)
	}
	return &definition, nil
}

func (s *SQLiteStore) CreateActionRun(ctx context.Context, run actions.ActionRun) (actions.ActionRun, error) {
	if err := run.Validate(); err != nil {
		return actions.ActionRun{}, err
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO action_runs (
		    id, action_definition_id, incident_id, input_transition_id, started_at, completed_at, status,
		    trigger_reason, target_count, confirmed_count, failed_count, escalated_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(run.ID),
		strings.TrimSpace(run.ActionDefinitionID),
		nullableTrimmed(run.IncidentID),
		nullableInt64(run.InputTransitionID),
		run.StartedAt,
		nullableTime(run.CompletedAt),
		run.Status,
		nullableTrimmed(run.TriggerReason),
		run.TargetCount,
		run.ConfirmedCount,
		run.FailedCount,
		run.EscalatedCount,
	)
	if err != nil {
		return actions.ActionRun{}, err
	}
	return run, nil
}

func (s *SQLiteStore) GetActionRun(ctx context.Context, id string) (*actions.ActionRun, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, action_definition_id, COALESCE(incident_id, ''), COALESCE(input_transition_id, 0), started_at,
		        completed_at, status, COALESCE(trigger_reason, ''), target_count, confirmed_count, failed_count, escalated_count
		   FROM action_runs
		  WHERE id = ?`,
		strings.TrimSpace(id),
	)
	var run actions.ActionRun
	var completedAt sql.NullTime
	if err := row.Scan(
		&run.ID,
		&run.ActionDefinitionID,
		&run.IncidentID,
		&run.InputTransitionID,
		&run.StartedAt,
		&completedAt,
		&run.Status,
		&run.TriggerReason,
		&run.TargetCount,
		&run.ConfirmedCount,
		&run.FailedCount,
		&run.EscalatedCount,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if completedAt.Valid {
		value := completedAt.Time
		run.CompletedAt = &value
	}
	return &run, nil
}

func (s *SQLiteStore) UpdateActionRun(ctx context.Context, run actions.ActionRun) error {
	if err := run.Validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE action_runs
		    SET action_definition_id = ?,
		        incident_id = ?,
		        input_transition_id = ?,
		        started_at = ?,
		        completed_at = ?,
		        status = ?,
		        trigger_reason = ?,
		        target_count = ?,
		        confirmed_count = ?,
		        failed_count = ?,
		        escalated_count = ?
		  WHERE id = ?`,
		strings.TrimSpace(run.ActionDefinitionID),
		nullableTrimmed(run.IncidentID),
		nullableInt64(run.InputTransitionID),
		run.StartedAt,
		nullableTime(run.CompletedAt),
		run.Status,
		nullableTrimmed(run.TriggerReason),
		run.TargetCount,
		run.ConfirmedCount,
		run.FailedCount,
		run.EscalatedCount,
		strings.TrimSpace(run.ID),
	)
	return err
}

func (s *SQLiteStore) ListActionRuns(ctx context.Context, query ActionRunListQuery) ([]actions.ActionRun, error) {
	limit := normalizeLimit(query.Limit)
	where := []string{}
	args := []any{}
	if query.Status != "" {
		where = append(where, "status = ?")
		args = append(args, query.Status)
	}
	if id := strings.TrimSpace(query.ActionDefinitionID); id != "" {
		where = append(where, "action_definition_id = ?")
		args = append(args, id)
	}
	if id := strings.TrimSpace(query.IncidentID); id != "" {
		where = append(where, "incident_id = ?")
		args = append(args, id)
	}
	if query.InputTransitionID > 0 {
		where = append(where, "input_transition_id = ?")
		args = append(args, query.InputTransitionID)
	}

	sqlBuilder := strings.Builder{}
	sqlBuilder.WriteString(`SELECT id, action_definition_id, COALESCE(incident_id, ''), COALESCE(input_transition_id, 0), started_at,
		        completed_at, status, COALESCE(trigger_reason, ''), target_count, confirmed_count, failed_count, escalated_count
		   FROM action_runs`)
	if len(where) > 0 {
		sqlBuilder.WriteString(" WHERE ")
		sqlBuilder.WriteString(strings.Join(where, " AND "))
	}
	sqlBuilder.WriteString(" ORDER BY started_at DESC, id DESC LIMIT ?")
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, sqlBuilder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []actions.ActionRun{}
	for rows.Next() {
		var run actions.ActionRun
		var completedAt sql.NullTime
		if err := rows.Scan(
			&run.ID,
			&run.ActionDefinitionID,
			&run.IncidentID,
			&run.InputTransitionID,
			&run.StartedAt,
			&completedAt,
			&run.Status,
			&run.TriggerReason,
			&run.TargetCount,
			&run.ConfirmedCount,
			&run.FailedCount,
			&run.EscalatedCount,
		); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			value := completedAt.Time
			run.CompletedAt = &value
		}
		result = append(result, run)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) CreateActionTargetResult(ctx context.Context, row actions.ActionTargetResult) (actions.ActionTargetResult, error) {
	if err := row.Validate(); err != nil {
		return actions.ActionTargetResult{}, err
	}
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO action_target_results (
		    action_run_id, target_egm_id, status, attempt_count, last_error, last_result_at
		) VALUES (?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(row.ActionRunID),
		strings.TrimSpace(row.TargetEGMID),
		row.Status,
		row.AttemptCount,
		nullableTrimmed(row.LastError),
		nullableTime(row.LastResultAt),
	)
	if err != nil {
		return actions.ActionTargetResult{}, err
	}
	row.ID, err = result.LastInsertId()
	if err != nil {
		return actions.ActionTargetResult{}, err
	}
	return row, nil
}

func (s *SQLiteStore) ListActionTargetResults(ctx context.Context, actionRunID string) ([]actions.ActionTargetResult, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, action_run_id, target_egm_id, status, attempt_count, COALESCE(last_error, ''), last_result_at
		   FROM action_target_results
		  WHERE action_run_id = ?
		  ORDER BY id ASC`,
		strings.TrimSpace(actionRunID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []actions.ActionTargetResult{}
	for rows.Next() {
		var target actions.ActionTargetResult
		var lastResultAt sql.NullTime
		if err := rows.Scan(
			&target.ID,
			&target.ActionRunID,
			&target.TargetEGMID,
			&target.Status,
			&target.AttemptCount,
			&target.LastError,
			&lastResultAt,
		); err != nil {
			return nil, err
		}
		if lastResultAt.Valid {
			value := lastResultAt.Time
			target.LastResultAt = &value
		}
		result = append(result, target)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) UpdateActionTargetResult(ctx context.Context, row actions.ActionTargetResult) error {
	if row.ID <= 0 {
		return fmt.Errorf("id is required")
	}
	if err := row.Validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE action_target_results
		    SET action_run_id = ?,
		        target_egm_id = ?,
		        status = ?,
		        attempt_count = ?,
		        last_error = ?,
		        last_result_at = ?
		  WHERE id = ?`,
		strings.TrimSpace(row.ActionRunID),
		strings.TrimSpace(row.TargetEGMID),
		row.Status,
		row.AttemptCount,
		nullableTrimmed(row.LastError),
		nullableTime(row.LastResultAt),
		row.ID,
	)
	return err
}

func (s *SQLiteStore) UpsertG2STemplate(ctx context.Context, tpl templates.G2STemplate) error {
	if err := tpl.Validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO g2s_templates (
		    id, name, vendor, cabinet_family, software_version_match, status,
		    current_version_id, notes, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
		    name = excluded.name,
		    vendor = excluded.vendor,
		    cabinet_family = excluded.cabinet_family,
		    software_version_match = excluded.software_version_match,
		    status = excluded.status,
		    current_version_id = excluded.current_version_id,
		    notes = excluded.notes,
		    updated_at = CURRENT_TIMESTAMP`,
		strings.TrimSpace(tpl.ID),
		strings.TrimSpace(tpl.Name),
		strings.TrimSpace(tpl.Vendor),
		nullableTrimmed(tpl.CabinetFamily),
		nullableTrimmed(tpl.SoftwareVersionMatch),
		tpl.Status,
		nullableTrimmed(tpl.CurrentVersionID),
		nullableTrimmed(tpl.Notes),
	)
	return err
}

func (s *SQLiteStore) ListG2STemplates(ctx context.Context) ([]templates.G2STemplate, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, name, vendor, COALESCE(cabinet_family, ''), COALESCE(software_version_match, ''), status,
		        COALESCE(current_version_id, ''), COALESCE(notes, ''), created_at, updated_at
		   FROM g2s_templates
		   ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []templates.G2STemplate{}
	for rows.Next() {
		var tpl templates.G2STemplate
		if err := rows.Scan(
			&tpl.ID,
			&tpl.Name,
			&tpl.Vendor,
			&tpl.CabinetFamily,
			&tpl.SoftwareVersionMatch,
			&tpl.Status,
			&tpl.CurrentVersionID,
			&tpl.Notes,
			&tpl.CreatedAt,
			&tpl.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, tpl)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) GetG2STemplate(ctx context.Context, id string) (*templates.G2STemplate, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, vendor, COALESCE(cabinet_family, ''), COALESCE(software_version_match, ''), status,
		        COALESCE(current_version_id, ''), COALESCE(notes, ''), created_at, updated_at
		   FROM g2s_templates
		  WHERE id = ?`,
		strings.TrimSpace(id),
	)
	var tpl templates.G2STemplate
	if err := row.Scan(
		&tpl.ID,
		&tpl.Name,
		&tpl.Vendor,
		&tpl.CabinetFamily,
		&tpl.SoftwareVersionMatch,
		&tpl.Status,
		&tpl.CurrentVersionID,
		&tpl.Notes,
		&tpl.CreatedAt,
		&tpl.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &tpl, nil
}

func (s *SQLiteStore) UpsertG2STemplateVersion(ctx context.Context, version templates.G2STemplateVersion) error {
	if err := version.Validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO g2s_template_versions (
		    id, template_id, version_label, endpoint_quirks_json, actions_json, confirmation_rules_json,
		    failure_rules_json, heartbeat_profile_json, variables_json, notes, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
		    template_id = excluded.template_id,
		    version_label = excluded.version_label,
		    endpoint_quirks_json = excluded.endpoint_quirks_json,
		    actions_json = excluded.actions_json,
		    confirmation_rules_json = excluded.confirmation_rules_json,
		    failure_rules_json = excluded.failure_rules_json,
		    heartbeat_profile_json = excluded.heartbeat_profile_json,
		    variables_json = excluded.variables_json,
		    notes = excluded.notes`,
		strings.TrimSpace(version.ID),
		strings.TrimSpace(version.TemplateID),
		strings.TrimSpace(version.VersionLabel),
		nullableTrimmed(version.EndpointQuirksJSON),
		strings.TrimSpace(version.ActionsJSON),
		nullableTrimmed(version.ConfirmationRulesJSON),
		nullableTrimmed(version.FailureRulesJSON),
		nullableTrimmed(version.HeartbeatProfileJSON),
		nullableTrimmed(version.VariablesJSON),
		nullableTrimmed(version.Notes),
	)
	return err
}

func (s *SQLiteStore) GetG2STemplateVersion(ctx context.Context, templateID string, version int) (*templates.G2STemplateVersion, error) {
	versionLabel := strconv.Itoa(version)
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, template_id, version_label, COALESCE(endpoint_quirks_json, ''), actions_json,
		        COALESCE(confirmation_rules_json, ''), COALESCE(failure_rules_json, ''), COALESCE(heartbeat_profile_json, ''),
		        COALESCE(variables_json, ''), COALESCE(notes, ''), created_at
		   FROM g2s_template_versions
		  WHERE template_id = ? AND version_label = ?`,
		strings.TrimSpace(templateID),
		versionLabel,
	)
	record, err := scanTemplateVersion(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (s *SQLiteStore) GetActiveG2STemplateVersion(ctx context.Context, templateID string) (*templates.G2STemplateVersion, error) {
	templateRow, err := s.GetG2STemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if templateRow == nil {
		return nil, nil
	}
	active := strings.TrimSpace(templateRow.CurrentVersionID)
	if active == "" {
		return nil, nil
	}

	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, template_id, version_label, COALESCE(endpoint_quirks_json, ''), actions_json,
		        COALESCE(confirmation_rules_json, ''), COALESCE(failure_rules_json, ''), COALESCE(heartbeat_profile_json, ''),
		        COALESCE(variables_json, ''), COALESCE(notes, ''), created_at
		   FROM g2s_template_versions
		  WHERE template_id = ? AND (version_label = ? OR id = ?)
		  ORDER BY created_at DESC
		  LIMIT 1`,
		strings.TrimSpace(templateID),
		active,
		active,
	)
	record, err := scanTemplateVersion(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (s *SQLiteStore) ListG2STemplateVersions(ctx context.Context, templateID string) ([]templates.G2STemplateVersion, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, template_id, version_label, COALESCE(endpoint_quirks_json, ''), actions_json,
		        COALESCE(confirmation_rules_json, ''), COALESCE(failure_rules_json, ''), COALESCE(heartbeat_profile_json, ''),
		        COALESCE(variables_json, ''), COALESCE(notes, ''), created_at
		   FROM g2s_template_versions
		  WHERE template_id = ?
		  ORDER BY created_at ASC, id ASC`,
		strings.TrimSpace(templateID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []templates.G2STemplateVersion{}
	for rows.Next() {
		record := templates.G2STemplateVersion{}
		if err := rows.Scan(
			&record.ID,
			&record.TemplateID,
			&record.VersionLabel,
			&record.EndpointQuirksJSON,
			&record.ActionsJSON,
			&record.ConfirmationRulesJSON,
			&record.FailureRulesJSON,
			&record.HeartbeatProfileJSON,
			&record.VariablesJSON,
			&record.Notes,
			&record.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) SetActiveG2STemplateVersion(ctx context.Context, templateID string, version int) error {
	versionLabel := strconv.Itoa(version)
	record, err := s.GetG2STemplateVersion(ctx, templateID, version)
	if err != nil {
		return err
	}
	if record == nil {
		return fmt.Errorf("template version %s/%d not found", strings.TrimSpace(templateID), version)
	}
	_, err = s.db.ExecContext(
		ctx,
		`UPDATE g2s_templates
		    SET current_version_id = ?, updated_at = CURRENT_TIMESTAMP
		  WHERE id = ?`,
		versionLabel,
		strings.TrimSpace(templateID),
	)
	return err
}

func scanTemplateVersion(scanner interface {
	Scan(dest ...any) error
}) (*templates.G2STemplateVersion, error) {
	record := templates.G2STemplateVersion{}
	if err := scanner.Scan(
		&record.ID,
		&record.TemplateID,
		&record.VersionLabel,
		&record.EndpointQuirksJSON,
		&record.ActionsJSON,
		&record.ConfirmationRulesJSON,
		&record.FailureRulesJSON,
		&record.HeartbeatProfileJSON,
		&record.VariablesJSON,
		&record.Notes,
		&record.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *SQLiteStore) UpsertEGMRecord(ctx context.Context, record egms.EGMRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO egm_records (
		    egm_id, display_name, ip_address, endpoint_path, vendor, cabinet_family, game_title,
		    software_version, zone, enabled, emergency_enabled, template_id, heartbeat_override_json,
		    last_seen_at, current_action_state, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(egm_id) DO UPDATE SET
		    display_name = excluded.display_name,
		    ip_address = excluded.ip_address,
		    endpoint_path = excluded.endpoint_path,
		    vendor = excluded.vendor,
		    cabinet_family = excluded.cabinet_family,
		    game_title = excluded.game_title,
		    software_version = excluded.software_version,
		    zone = excluded.zone,
		    enabled = excluded.enabled,
		    emergency_enabled = excluded.emergency_enabled,
		    template_id = excluded.template_id,
		    heartbeat_override_json = excluded.heartbeat_override_json,
		    last_seen_at = excluded.last_seen_at,
		    current_action_state = excluded.current_action_state,
		    notes = excluded.notes`,
		strings.TrimSpace(record.EGMID),
		nullableTrimmed(record.DisplayName),
		nullableTrimmed(record.IPAddress),
		nullableTrimmed(record.EndpointPath),
		nullableTrimmed(record.Vendor),
		nullableTrimmed(record.CabinetFamily),
		nullableTrimmed(record.GameTitle),
		nullableTrimmed(record.SoftwareVersion),
		nullableTrimmed(record.Zone),
		boolToInt(record.Enabled),
		boolToInt(record.EmergencyEnabled),
		nullableTrimmed(record.TemplateID),
		nullableTrimmed(record.HeartbeatOverrideJSON),
		nullableTime(record.LastSeenAt),
		record.CurrentActionState,
		nullableTrimmed(record.Notes),
	)
	return err
}

func (s *SQLiteStore) GetEGMRecord(ctx context.Context, egmID string) (*egms.EGMRecord, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT egm_id, COALESCE(display_name, ''), COALESCE(ip_address, ''), COALESCE(endpoint_path, ''),
		        COALESCE(vendor, ''), COALESCE(cabinet_family, ''), COALESCE(game_title, ''), COALESCE(software_version, ''),
		        COALESCE(zone, ''), enabled, emergency_enabled, COALESCE(template_id, ''), COALESCE(heartbeat_override_json, ''),
		        last_seen_at, current_action_state, COALESCE(notes, '')
		   FROM egm_records
		  WHERE egm_id = ?`,
		strings.TrimSpace(egmID),
	)
	var record egms.EGMRecord
	var enabled int
	var emergencyEnabled int
	var lastSeenAt sql.NullTime
	if err := row.Scan(
		&record.EGMID,
		&record.DisplayName,
		&record.IPAddress,
		&record.EndpointPath,
		&record.Vendor,
		&record.CabinetFamily,
		&record.GameTitle,
		&record.SoftwareVersion,
		&record.Zone,
		&enabled,
		&emergencyEnabled,
		&record.TemplateID,
		&record.HeartbeatOverrideJSON,
		&lastSeenAt,
		&record.CurrentActionState,
		&record.Notes,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	record.Enabled = enabled != 0
	record.EmergencyEnabled = emergencyEnabled != 0
	if lastSeenAt.Valid {
		value := lastSeenAt.Time
		record.LastSeenAt = &value
	}
	return &record, nil
}

func (s *SQLiteStore) ListEGMRecords(ctx context.Context) ([]egms.EGMRecord, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT egm_id, COALESCE(display_name, ''), COALESCE(ip_address, ''), COALESCE(endpoint_path, ''),
		        COALESCE(vendor, ''), COALESCE(cabinet_family, ''), COALESCE(game_title, ''), COALESCE(software_version, ''),
		        COALESCE(zone, ''), enabled, emergency_enabled, COALESCE(template_id, ''), COALESCE(heartbeat_override_json, ''),
		        last_seen_at, current_action_state, COALESCE(notes, '')
		   FROM egm_records
		   ORDER BY egm_id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []egms.EGMRecord{}
	for rows.Next() {
		var record egms.EGMRecord
		var enabled int
		var emergencyEnabled int
		var lastSeenAt sql.NullTime
		if err := rows.Scan(
			&record.EGMID,
			&record.DisplayName,
			&record.IPAddress,
			&record.EndpointPath,
			&record.Vendor,
			&record.CabinetFamily,
			&record.GameTitle,
			&record.SoftwareVersion,
			&record.Zone,
			&enabled,
			&emergencyEnabled,
			&record.TemplateID,
			&record.HeartbeatOverrideJSON,
			&lastSeenAt,
			&record.CurrentActionState,
			&record.Notes,
		); err != nil {
			return nil, err
		}
		record.Enabled = enabled != 0
		record.EmergencyEnabled = emergencyEnabled != 0
		if lastSeenAt.Valid {
			value := lastSeenAt.Time
			record.LastSeenAt = &value
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) GetEGMGroup(ctx context.Context, id string) (*egms.EGMGroup, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, COALESCE(description, ''), COALESCE(egm_ids_json, '[]'), created_at, updated_at
		   FROM egm_groups
		  WHERE id = ?`,
		strings.TrimSpace(id),
	)
	var group egms.EGMGroup
	var egmIDsJSON string
	if err := row.Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&egmIDsJSON,
		&group.CreatedAt,
		&group.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if err := decodeJSONStringSlice(egmIDsJSON, &group.EGMIDs); err != nil {
		return nil, fmt.Errorf("decode egm_ids_json for group %q: %w", group.ID, err)
	}
	return &group, nil
}

func (s *SQLiteStore) UpsertEGMGroup(ctx context.Context, group egms.EGMGroup) error {
	if err := group.Validate(); err != nil {
		return err
	}
	egmIDsJSON, err := encodeJSONStringSlice(group.EGMIDs)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO egm_groups (
		    id, name, description, egm_ids_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
		    name = excluded.name,
		    description = excluded.description,
		    egm_ids_json = excluded.egm_ids_json,
		    updated_at = CURRENT_TIMESTAMP`,
		strings.TrimSpace(group.ID),
		strings.TrimSpace(group.Name),
		nullableTrimmed(group.Description),
		egmIDsJSON,
	)
	return err
}

func (s *SQLiteStore) ListEGMGroups(ctx context.Context) ([]egms.EGMGroup, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, name, COALESCE(description, ''), COALESCE(egm_ids_json, '[]'), created_at, updated_at
		   FROM egm_groups
		   ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []egms.EGMGroup{}
	for rows.Next() {
		var group egms.EGMGroup
		var egmIDsJSON string
		if err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.Description,
			&egmIDsJSON,
			&group.CreatedAt,
			&group.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if err := decodeJSONStringSlice(egmIDsJSON, &group.EGMIDs); err != nil {
			return nil, fmt.Errorf("decode egm_ids_json for group %q: %w", group.ID, err)
		}
		result = append(result, group)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) RecordMessageJournalEntry(ctx context.Context, entry g2sengine.MessageJournalEntry) (int64, error) {
	if err := entry.Validate(); err != nil {
		return 0, err
	}
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO message_journal (
		    timestamp, direction, from_endpoint, to_endpoint, egm_id, action_run_id, action_step_id,
		    input_transition_id, template_id, template_version, handler_rule_id, message_type,
		    raw_payload, parsed_summary_json, result, error, http_status_code, latency_ms,
		    response_excerpt, offered_at, offer_count, sent_at, completed_at, transport_mode
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.Timestamp,
		entry.Direction,
		nullableTrimmed(entry.FromEndpoint),
		nullableTrimmed(entry.ToEndpoint),
		nullableTrimmed(entry.EGMID),
		nullableTrimmed(entry.ActionRunID),
		nullableTrimmed(entry.ActionStepID),
		nullableInt64(entry.InputTransitionID),
		nullableTrimmed(entry.TemplateID),
		nullableTrimmed(entry.TemplateVersion),
		nullableTrimmed(entry.HandlerRuleID),
		nullableTrimmed(entry.MessageType),
		entry.RawPayload,
		nullableTrimmed(entry.ParsedSummaryJSON),
		entry.Result,
		nullableTrimmed(entry.Error),
		nullablePositiveInt(entry.HTTPStatusCode),
		nullablePositiveInt(entry.LatencyMS),
		nullableTrimmed(entry.ResponseExcerpt),
		nullableTime(entry.OfferedAt),
		nullablePositiveInt(entry.OfferCount),
		nullableTime(entry.SentAt),
		nullableTime(entry.CompletedAt),
		nullableTrimmed(entry.TransportMode),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *SQLiteStore) GetMessageJournalEntry(ctx context.Context, id int64) (*g2sengine.MessageJournalEntry, error) {
	if id <= 0 {
		return nil, fmt.Errorf("message journal id is required")
	}
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, timestamp, direction, COALESCE(from_endpoint, ''), COALESCE(to_endpoint, ''), COALESCE(egm_id, ''),
		        COALESCE(action_run_id, ''), COALESCE(action_step_id, ''), input_transition_id,
		        COALESCE(template_id, ''), COALESCE(template_version, ''), COALESCE(handler_rule_id, ''), COALESCE(message_type, ''),
		        raw_payload, COALESCE(parsed_summary_json, ''), result, COALESCE(error, ''),
		        COALESCE(http_status_code, 0), COALESCE(latency_ms, 0), COALESCE(response_excerpt, ''),
		        offered_at, COALESCE(offer_count, 0), sent_at, completed_at, COALESCE(transport_mode, '')
		   FROM message_journal
		  WHERE id = ?`,
		id,
	)
	var entry g2sengine.MessageJournalEntry
	var inputTransitionID sql.NullInt64
	var sentAt sql.NullTime
	var completedAt sql.NullTime
	var offeredAt sql.NullTime
	if err := row.Scan(
		&entry.ID,
		&entry.Timestamp,
		&entry.Direction,
		&entry.FromEndpoint,
		&entry.ToEndpoint,
		&entry.EGMID,
		&entry.ActionRunID,
		&entry.ActionStepID,
		&inputTransitionID,
		&entry.TemplateID,
		&entry.TemplateVersion,
		&entry.HandlerRuleID,
		&entry.MessageType,
		&entry.RawPayload,
		&entry.ParsedSummaryJSON,
		&entry.Result,
		&entry.Error,
		&entry.HTTPStatusCode,
		&entry.LatencyMS,
		&entry.ResponseExcerpt,
		&offeredAt,
		&entry.OfferCount,
		&sentAt,
		&completedAt,
		&entry.TransportMode,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if inputTransitionID.Valid {
		entry.InputTransitionID = inputTransitionID.Int64
	}
	if sentAt.Valid {
		value := sentAt.Time
		entry.SentAt = &value
	}
	if offeredAt.Valid {
		value := offeredAt.Time
		entry.OfferedAt = &value
	}
	if completedAt.Valid {
		value := completedAt.Time
		entry.CompletedAt = &value
	}
	return &entry, nil
}

func (s *SQLiteStore) UpdateMessageJournalHandlerRule(ctx context.Context, id int64, handlerRuleID string) error {
	if id <= 0 {
		return fmt.Errorf("message journal id is required")
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE message_journal
		    SET handler_rule_id = ?
		  WHERE id = ?`,
		nullableTrimmed(handlerRuleID),
		id,
	)
	return err
}

func (s *SQLiteStore) UpdateMessageJournalResult(
	ctx context.Context,
	id int64,
	result g2sengine.MessageResult,
	errText string,
	responseExcerpt string,
	httpStatusCode int,
	latencyMS int,
	transportMode string,
	sentAt *time.Time,
	completedAt *time.Time,
) error {
	switch result {
	case g2sengine.MessageResultSent, g2sengine.MessageResultReceived, g2sengine.MessageResultAcked, g2sengine.MessageResultConfirmed, g2sengine.MessageResultFailed, g2sengine.MessageResultIgnored, g2sengine.MessageResultEscalated, g2sengine.MessageResultDryRun, g2sengine.MessageResultPrepared, g2sengine.MessageResultPending, g2sengine.MessageResultOffered, g2sengine.MessageResultDelivered, g2sengine.MessageResultExpired, g2sengine.MessageResultSuperseded, g2sengine.MessageResultSendAttempted, g2sengine.MessageResultSendFailed, g2sengine.MessageResultSendSucceeded:
	default:
		return fmt.Errorf("result is invalid")
	}
	if id <= 0 {
		return fmt.Errorf("message journal id is required")
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE message_journal
		    SET result = ?,
		        error = ?,
		        response_excerpt = ?,
		        http_status_code = ?,
		        latency_ms = ?,
		        transport_mode = ?,
		        sent_at = ?,
		        completed_at = ?
		  WHERE id = ?`,
		result,
		nullableTrimmed(errText),
		nullableTrimmed(responseExcerpt),
		nullablePositiveInt(httpStatusCode),
		nullablePositiveInt(latencyMS),
		nullableTrimmed(transportMode),
		nullableTime(sentAt),
		nullableTime(completedAt),
		id,
	)
	return err
}

func (s *SQLiteStore) UpdateMessageJournalOffer(ctx context.Context, id int64, offeredAt time.Time, result g2sengine.MessageResult) (bool, error) {
	switch result {
	case g2sengine.MessageResultOffered, g2sengine.MessageResultDelivered:
	default:
		return false, fmt.Errorf("offer result is invalid")
	}
	if id <= 0 {
		return false, fmt.Errorf("message journal id is required")
	}
	if offeredAt.IsZero() {
		offeredAt = time.Now().UTC()
	}
	res, err := s.db.ExecContext(
		ctx,
		`UPDATE message_journal
		    SET result = ?,
		        offered_at = ?,
		        offer_count = COALESCE(offer_count, 0) + 1
		  WHERE id = ?
		    AND result IN (?, ?, ?)`,
		result,
		offeredAt.UTC(),
		id,
		g2sengine.MessageResultPrepared,
		g2sengine.MessageResultPending,
		g2sengine.MessageResultOffered,
	)
	if err != nil {
		return false, err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLiteStore) ListMessageJournalEntries(ctx context.Context, query MessageJournalListQuery) ([]g2sengine.MessageJournalEntry, error) {
	limit := normalizeLimit(query.Limit)
	where := []string{}
	args := []any{}
	if id := strings.TrimSpace(query.EGMID); id != "" {
		where = append(where, "egm_id = ?")
		args = append(args, id)
	}
	if id := strings.TrimSpace(query.ActionRunID); id != "" {
		where = append(where, "action_run_id = ?")
		args = append(args, id)
	}
	if id := strings.TrimSpace(query.IncidentID); id != "" {
		where = append(where, "action_run_id IN (SELECT id FROM action_runs WHERE incident_id = ?)")
		args = append(args, id)
	}
	if query.InputTransitionID > 0 {
		where = append(where, "input_transition_id = ?")
		args = append(args, query.InputTransitionID)
	}
	if query.Direction != "" {
		where = append(where, "direction = ?")
		args = append(args, query.Direction)
	}
	if len(query.Results) > 0 {
		placeholders := make([]string, 0, len(query.Results))
		for _, value := range query.Results {
			if strings.TrimSpace(string(value)) == "" {
				continue
			}
			placeholders = append(placeholders, "?")
			args = append(args, value)
		}
		if len(placeholders) > 0 {
			where = append(where, "result IN ("+strings.Join(placeholders, ", ")+")")
		}
	}
	sqlBuilder := strings.Builder{}
	sqlBuilder.WriteString(`SELECT id, timestamp, direction, COALESCE(from_endpoint, ''), COALESCE(to_endpoint, ''), COALESCE(egm_id, ''),
		        COALESCE(action_run_id, ''), COALESCE(action_step_id, ''), input_transition_id,
		        COALESCE(template_id, ''), COALESCE(template_version, ''), COALESCE(handler_rule_id, ''), COALESCE(message_type, ''),
		        raw_payload, COALESCE(parsed_summary_json, ''), result, COALESCE(error, ''),
		        COALESCE(http_status_code, 0), COALESCE(latency_ms, 0), COALESCE(response_excerpt, ''),
		        offered_at, COALESCE(offer_count, 0), sent_at, completed_at, COALESCE(transport_mode, '')
		   FROM message_journal`)
	if len(where) > 0 {
		sqlBuilder.WriteString(" WHERE ")
		sqlBuilder.WriteString(strings.Join(where, " AND "))
	}
	sqlBuilder.WriteString(" ORDER BY id DESC LIMIT ?")
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, sqlBuilder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []g2sengine.MessageJournalEntry{}
	for rows.Next() {
		var entry g2sengine.MessageJournalEntry
		var inputTransitionID sql.NullInt64
		var sentAt sql.NullTime
		var completedAt sql.NullTime
		var offeredAt sql.NullTime
		if err := rows.Scan(
			&entry.ID,
			&entry.Timestamp,
			&entry.Direction,
			&entry.FromEndpoint,
			&entry.ToEndpoint,
			&entry.EGMID,
			&entry.ActionRunID,
			&entry.ActionStepID,
			&inputTransitionID,
			&entry.TemplateID,
			&entry.TemplateVersion,
			&entry.HandlerRuleID,
			&entry.MessageType,
			&entry.RawPayload,
			&entry.ParsedSummaryJSON,
			&entry.Result,
			&entry.Error,
			&entry.HTTPStatusCode,
			&entry.LatencyMS,
			&entry.ResponseExcerpt,
			&offeredAt,
			&entry.OfferCount,
			&sentAt,
			&completedAt,
			&entry.TransportMode,
		); err != nil {
			return nil, err
		}
		if inputTransitionID.Valid {
			entry.InputTransitionID = inputTransitionID.Int64
		}
		if sentAt.Valid {
			value := sentAt.Time
			entry.SentAt = &value
		}
		if offeredAt.Valid {
			value := offeredAt.Time
			entry.OfferedAt = &value
		}
		if completedAt.Valid {
			value := completedAt.Time
			entry.CompletedAt = &value
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) UpsertHandlerRule(ctx context.Context, rule g2sengine.HandlerRule) error {
	rule.Direction = g2sengine.HandlerRuleDirection(strings.ToUpper(strings.TrimSpace(string(rule.Direction))))
	if strings.TrimSpace(string(rule.Direction)) == "" {
		rule.Direction = g2sengine.HandlerRuleDirectionAny
	}
	rule.Outcome = g2sengine.HandlerRuleOutcome(strings.ToUpper(strings.TrimSpace(string(rule.Outcome))))
	if strings.TrimSpace(string(rule.Outcome)) == "" {
		rule.Outcome = g2sengine.HandlerRuleOutcomeNote
	}
	if strings.TrimSpace(rule.HandleJSON) == "" {
		payload, err := json.Marshal(map[string]string{"outcome": string(rule.Outcome)})
		if err == nil {
			rule.HandleJSON = string(payload)
		}
	}
	if err := rule.Validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO handler_rules (
		    id, name, enabled, direction, template_id, message_type, egm_id, action_id, action_step_id,
		    match_json, outcome, handle_json, notes, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
		    name = excluded.name,
		    enabled = excluded.enabled,
		    direction = excluded.direction,
		    template_id = excluded.template_id,
		    message_type = excluded.message_type,
		    egm_id = excluded.egm_id,
		    action_id = excluded.action_id,
		    action_step_id = excluded.action_step_id,
		    match_json = excluded.match_json,
		    outcome = excluded.outcome,
		    handle_json = excluded.handle_json,
		    notes = excluded.notes,
		    updated_at = CURRENT_TIMESTAMP`,
		strings.TrimSpace(rule.ID),
		strings.TrimSpace(rule.Name),
		boolToInt(rule.Enabled),
		strings.ToUpper(strings.TrimSpace(string(rule.Direction))),
		nullableTrimmed(rule.TemplateID),
		nullableTrimmed(rule.MessageType),
		nullableTrimmed(rule.EGMID),
		nullableTrimmed(rule.ActionID),
		nullableTrimmed(rule.ActionStepID),
		strings.TrimSpace(rule.MatchJSON),
		strings.ToUpper(strings.TrimSpace(string(rule.Outcome))),
		nullableTrimmed(rule.HandleJSON),
		nullableTrimmed(rule.Notes),
	)
	return err
}

func (s *SQLiteStore) GetHandlerRule(ctx context.Context, id string) (*g2sengine.HandlerRule, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, enabled, COALESCE(direction, 'ANY'), COALESCE(template_id, ''), COALESCE(message_type, ''), COALESCE(egm_id, ''),
		        COALESCE(action_id, ''), COALESCE(action_step_id, ''), COALESCE(match_json, ''), COALESCE(outcome, 'NOTE'),
		        COALESCE(handle_json, ''), COALESCE(notes, ''), created_at, updated_at
		   FROM handler_rules
		  WHERE id = ?`,
		strings.TrimSpace(id),
	)
	var result g2sengine.HandlerRule
	var enabledInt int
	if err := row.Scan(
		&result.ID,
		&result.Name,
		&enabledInt,
		&result.Direction,
		&result.TemplateID,
		&result.MessageType,
		&result.EGMID,
		&result.ActionID,
		&result.ActionStepID,
		&result.MatchJSON,
		&result.Outcome,
		&result.HandleJSON,
		&result.Notes,
		&result.CreatedAt,
		&result.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	result.Enabled = enabledInt == 1
	return &result, nil
}

func (s *SQLiteStore) ListHandlerRules(ctx context.Context, query HandlerRuleListQuery) ([]g2sengine.HandlerRule, error) {
	limit := normalizeLimit(query.Limit)
	where := ""
	args := []any{}
	if query.EnabledOnly {
		where = " WHERE enabled = 1"
	}
	args = append(args, limit)

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, name, enabled, COALESCE(direction, 'ANY'), COALESCE(template_id, ''), COALESCE(message_type, ''), COALESCE(egm_id, ''),
		        COALESCE(action_id, ''), COALESCE(action_step_id, ''), COALESCE(match_json, ''), COALESCE(outcome, 'NOTE'),
		        COALESCE(handle_json, ''), COALESCE(notes, ''), created_at, updated_at
		   FROM handler_rules`+where+`
		  ORDER BY updated_at DESC, id ASC
		  LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []g2sengine.HandlerRule{}
	for rows.Next() {
		var row g2sengine.HandlerRule
		var enabledInt int
		if err := rows.Scan(
			&row.ID,
			&row.Name,
			&enabledInt,
			&row.Direction,
			&row.TemplateID,
			&row.MessageType,
			&row.EGMID,
			&row.ActionID,
			&row.ActionStepID,
			&row.MatchJSON,
			&row.Outcome,
			&row.HandleJSON,
			&row.Notes,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		row.Enabled = enabledInt == 1
		result = append(result, row)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) ListEnabledHandlerRules(ctx context.Context, limit int) ([]g2sengine.HandlerRule, error) {
	return s.ListHandlerRules(ctx, HandlerRuleListQuery{EnabledOnly: true, Limit: limit})
}

func (s *SQLiteStore) DisableHandlerRule(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("handler rule id is required")
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE handler_rules
		    SET enabled = 0,
		        updated_at = CURRENT_TIMESTAMP
		  WHERE id = ?`,
		strings.TrimSpace(id),
	)
	return err
}

func (s *SQLiteStore) RecordAuditTimelineEntry(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error) {
	if err := entry.Validate(); err != nil {
		return 0, err
	}
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO audit_timeline (
		    occurred_at, severity, event_type, summary, detail_json,
		    action_run_id, input_transition_id, message_journal_id, operator
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.OccurredAt,
		entry.Severity,
		entry.EventType,
		entry.Summary,
		nullableTrimmed(entry.DetailJSON),
		nullableTrimmed(entry.ActionRunID),
		nullableInt64(entry.InputTransitionID),
		nullableInt64(entry.MessageJournalID),
		nullableTrimmed(entry.Operator),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *SQLiteStore) ListAuditTimelineEntries(ctx context.Context, query AuditTimelineListQuery) ([]audit.AuditTimelineEntry, error) {
	limit := normalizeLimit(query.Limit)
	where := []string{}
	args := []any{}
	if eventType := strings.TrimSpace(query.EventType); eventType != "" {
		where = append(where, "event_type = ?")
		args = append(args, eventType)
	}
	if query.Severity != "" {
		where = append(where, "severity = ?")
		args = append(args, query.Severity)
	}
	if id := strings.TrimSpace(query.ActionRunID); id != "" {
		where = append(where, "action_run_id = ?")
		args = append(args, id)
	}
	if id := strings.TrimSpace(query.IncidentID); id != "" {
		where = append(where, "(action_run_id IN (SELECT id FROM action_runs WHERE incident_id = ?) OR input_transition_id IN (SELECT opened_by_transition_id FROM incident_records WHERE CAST(incident_id AS TEXT) = ?) OR input_transition_id IN (SELECT closed_by_transition_id FROM incident_records WHERE CAST(incident_id AS TEXT) = ?))")
		args = append(args, id, id, id)
	}
	if query.InputTransitionID > 0 {
		where = append(where, "input_transition_id = ?")
		args = append(args, query.InputTransitionID)
	}
	sqlBuilder := strings.Builder{}
	sqlBuilder.WriteString(`SELECT id, occurred_at, severity, event_type, summary, COALESCE(detail_json, ''),
		        COALESCE(action_run_id, ''), input_transition_id, message_journal_id, COALESCE(operator, '')
		   FROM audit_timeline`)
	if len(where) > 0 {
		sqlBuilder.WriteString(" WHERE ")
		sqlBuilder.WriteString(strings.Join(where, " AND "))
	}
	sqlBuilder.WriteString(" ORDER BY id DESC LIMIT ?")
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, sqlBuilder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []audit.AuditTimelineEntry{}
	for rows.Next() {
		var entry audit.AuditTimelineEntry
		var inputTransitionID sql.NullInt64
		var messageJournalID sql.NullInt64
		if err := rows.Scan(
			&entry.ID,
			&entry.OccurredAt,
			&entry.Severity,
			&entry.EventType,
			&entry.Summary,
			&entry.DetailJSON,
			&entry.ActionRunID,
			&inputTransitionID,
			&messageJournalID,
			&entry.Operator,
		); err != nil {
			return nil, err
		}
		if inputTransitionID.Valid {
			entry.InputTransitionID = inputTransitionID.Int64
		}
		if messageJournalID.Valid {
			entry.MessageJournalID = messageJournalID.Int64
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullableTrimmed(raw string) any {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	return value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt64(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullablePositiveInt(value int) any {
	if value <= 0 {
		return nil
	}
	return value
}
