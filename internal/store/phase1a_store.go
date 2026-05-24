package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	Limit     int
	EGMID     string
	Direction g2sengine.MessageDirection
}

type AuditTimelineListQuery struct {
	Limit     int
	EventType string
	Severity  audit.AuditSeverity
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

func (s *SQLiteStore) RecordMessageJournalEntry(ctx context.Context, entry g2sengine.MessageJournalEntry) (int64, error) {
	if err := entry.Validate(); err != nil {
		return 0, err
	}
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO message_journal (
		    timestamp, direction, from_endpoint, to_endpoint, egm_id, action_run_id, action_step_id,
		    input_transition_id, template_id, template_version, handler_rule_id, message_type,
		    raw_payload, parsed_summary_json, result, error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *SQLiteStore) ListMessageJournalEntries(ctx context.Context, query MessageJournalListQuery) ([]g2sengine.MessageJournalEntry, error) {
	limit := normalizeLimit(query.Limit)
	where := []string{}
	args := []any{}
	if id := strings.TrimSpace(query.EGMID); id != "" {
		where = append(where, "egm_id = ?")
		args = append(args, id)
	}
	if query.Direction != "" {
		where = append(where, "direction = ?")
		args = append(args, query.Direction)
	}
	sqlBuilder := strings.Builder{}
	sqlBuilder.WriteString(`SELECT id, timestamp, direction, COALESCE(from_endpoint, ''), COALESCE(to_endpoint, ''), COALESCE(egm_id, ''),
		        COALESCE(action_run_id, ''), COALESCE(action_step_id, ''), input_transition_id,
		        COALESCE(template_id, ''), COALESCE(template_version, ''), COALESCE(handler_rule_id, ''), COALESCE(message_type, ''),
		        raw_payload, COALESCE(parsed_summary_json, ''), result, COALESCE(error, '')
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
		); err != nil {
			return nil, err
		}
		if inputTransitionID.Valid {
			entry.InputTransitionID = inputTransitionID.Int64
		}
		result = append(result, entry)
	}
	return result, rows.Err()
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
