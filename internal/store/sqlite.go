package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

const operatorAuditRetentionLimit = 2000

type CabinetProfileOverride struct {
	Profile   config.CabinetProfile
	UpdatedAt time.Time
	UpdatedBy string
}

type HeartbeatPolicyOverride struct {
	IntervalMS         int
	WarningAfterMissed int
	BlockAfterMissed   int
	UpdatedAt          time.Time
	UpdatedBy          string
}

type BlockerPolicyOverride struct {
	ApprovedBlockerIDs []string
	UpdatedAt          time.Time
	UpdatedBy          string
}

func Open(ctx context.Context, path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("database path is required")
	}
	if path != ":memory:" && !strings.HasPrefix(path, "file:") {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &SQLiteStore{db: db}
	if err := store.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `PRAGMA journal_mode = WAL`); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, InitMigration)
	if err != nil {
		return err
	}
	return s.ensureHeartbeatPolicyOverrideSchema(ctx)
}

func (s *SQLiteStore) ensureHeartbeatPolicyOverrideSchema(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(heartbeat_policy_overrides)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasInterval := false
	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if strings.EqualFold(strings.TrimSpace(name), "interval_ms") {
			hasInterval = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if hasInterval {
		return nil
	}
	_, err = s.db.ExecContext(ctx, `ALTER TABLE heartbeat_policy_overrides ADD COLUMN interval_ms INTEGER`)
	return err
}

func (s *SQLiteStore) RecordIncident(ctx context.Context, incident model.Incident) (int64, error) {
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO incident_records (created_at, trigger_type, trigger_source, resolved_at, final_state)
		 VALUES (?, ?, ?, ?, ?)`,
		incident.CreatedAt,
		incident.TriggerType,
		incident.TriggerSource,
		incident.ResolvedAt,
		incident.FinalState,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *SQLiteStore) RecordEGMStatus(ctx context.Context, snapshot model.EGMStatusSnapshot) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO egm_status_snapshots (created_at, egm_id, status, event_type, detail, last_error)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		snapshot.CreatedAt,
		snapshot.EGMID,
		snapshot.Status,
		snapshot.EventType,
		snapshot.Detail,
		snapshot.LastError,
	)
	return err
}

func (s *SQLiteStore) RecordEGMComplianceLog(ctx context.Context, entry model.EGMComplianceLog) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO egm_compliance_logs (
			incident_id, egm_id, ip_address, action_sent, status_result,
			http_status_code, latency_ms, response_excerpt, created_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.IncidentID,
		entry.EGMID,
		entry.IPAddress,
		entry.ActionSent,
		entry.StatusResult,
		entry.HTTPStatusCode,
		entry.LatencyMS,
		entry.ResponseExcerpt,
		entry.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) RecordStateChange(ctx context.Context, change model.StateChange) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO controller_state_history (created_at, old_state, new_state, reason)
		 VALUES (?, ?, ?, ?)`,
		change.CreatedAt,
		change.OldState,
		change.NewState,
		change.Reason,
	)
	return err
}

func (s *SQLiteStore) ReplaceCertificateInventory(ctx context.Context, records []model.CertificateInventory) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM certificate_inventory`); err != nil {
		return err
	}
	for _, record := range records {
		var notBefore any
		var notAfter any
		if record.NotBefore != nil {
			notBefore = *record.NotBefore
		}
		if record.NotAfter != nil {
			notAfter = *record.NotAfter
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO certificate_inventory (
				cert_role, path, subject, issuer, not_before, not_after,
				sha256_fingerprint, last_checked_at, status
			 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			record.Role,
			record.Path,
			record.Subject,
			record.Issuer,
			notBefore,
			notAfter,
			record.SHA256Fingerprint,
			record.LastCheckedAt,
			statusWithError(record),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) ListIncidents(ctx context.Context, limit int) ([]model.Incident, error) {
	limit = normalizeLimit(limit)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT incident_id, created_at, trigger_type, trigger_source, resolved_at, final_state
		 FROM incident_records
		 ORDER BY incident_id DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	incidents := []model.Incident{}
	for rows.Next() {
		var incident model.Incident
		var resolvedAt sql.NullTime
		if err := rows.Scan(
			&incident.ID,
			&incident.CreatedAt,
			&incident.TriggerType,
			&incident.TriggerSource,
			&resolvedAt,
			&incident.FinalState,
		); err != nil {
			return nil, err
		}
		if resolvedAt.Valid {
			incident.ResolvedAt = &resolvedAt.Time
		}
		incidents = append(incidents, incident)
	}
	return incidents, rows.Err()
}

func (s *SQLiteStore) ListEGMStatus(ctx context.Context, limits model.HistoryLimits) ([]model.EGMStatusSnapshot, error) {
	limit := normalizeLimit(limits.Limit)
	query := `SELECT created_at, egm_id, status, event_type, detail, last_error
		FROM egm_status_snapshots`
	args := []any{}
	if limits.EGMID != "" {
		query += ` WHERE egm_id = ?`
		args = append(args, limits.EGMID)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snapshots := []model.EGMStatusSnapshot{}
	for rows.Next() {
		var snapshot model.EGMStatusSnapshot
		if err := rows.Scan(
			&snapshot.CreatedAt,
			&snapshot.EGMID,
			&snapshot.Status,
			&snapshot.EventType,
			&snapshot.Detail,
			&snapshot.LastError,
		); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (s *SQLiteStore) ListEGMComplianceLogs(ctx context.Context, limit int) ([]model.EGMComplianceLog, error) {
	limit = normalizeLimit(limit)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT incident_id, egm_id, ip_address, action_sent, status_result,
			http_status_code, latency_ms, response_excerpt, created_at
		 FROM egm_compliance_logs
		 ORDER BY log_id DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []model.EGMComplianceLog{}
	for rows.Next() {
		var entry model.EGMComplianceLog
		if err := rows.Scan(
			&entry.IncidentID,
			&entry.EGMID,
			&entry.IPAddress,
			&entry.ActionSent,
			&entry.StatusResult,
			&entry.HTTPStatusCode,
			&entry.LatencyMS,
			&entry.ResponseExcerpt,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, entry)
	}
	return logs, rows.Err()
}

func (s *SQLiteStore) ListStateChanges(ctx context.Context, limit int) ([]model.StateChange, error) {
	limit = normalizeLimit(limit)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT created_at, old_state, new_state, reason
		 FROM controller_state_history
		 ORDER BY id DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	changes := []model.StateChange{}
	for rows.Next() {
		var change model.StateChange
		if err := rows.Scan(
			&change.CreatedAt,
			&change.OldState,
			&change.NewState,
			&change.Reason,
		); err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, rows.Err()
}

func (s *SQLiteStore) ListCertificateInventory(ctx context.Context) ([]model.CertificateInventory, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT cert_role, path, subject, issuer, not_before, not_after,
			sha256_fingerprint, last_checked_at, status
		 FROM certificate_inventory
		 ORDER BY cert_role`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []model.CertificateInventory{}
	for rows.Next() {
		var record model.CertificateInventory
		var notBefore sql.NullTime
		var notAfter sql.NullTime
		if err := rows.Scan(
			&record.Role,
			&record.Path,
			&record.Subject,
			&record.Issuer,
			&notBefore,
			&notAfter,
			&record.SHA256Fingerprint,
			&record.LastCheckedAt,
			&record.Status,
		); err != nil {
			return nil, err
		}
		if notBefore.Valid {
			record.NotBefore = &notBefore.Time
		}
		if notAfter.Valid {
			record.NotAfter = &notAfter.Time
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *SQLiteStore) GetCabinetProfileOverride(ctx context.Context) (*CabinetProfileOverride, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT wire_host_url, listener_dns_name, listener_ip, required_san_dns_json,
		        required_san_ips_json, host_id, first_test_egm_ids_json, updated_at, COALESCE(updated_by, '')
		   FROM cabinet_profile_overrides
		  WHERE id = 1`,
	)

	var wireHostURL string
	var listenerDNSName string
	var listenerIP string
	var requiredSANDNSJSON string
	var requiredSANIPsJSON string
	var hostID string
	var firstTestEGMIDsJSON string
	var updatedAt time.Time
	var updatedBy string
	if err := row.Scan(
		&wireHostURL,
		&listenerDNSName,
		&listenerIP,
		&requiredSANDNSJSON,
		&requiredSANIPsJSON,
		&hostID,
		&firstTestEGMIDsJSON,
		&updatedAt,
		&updatedBy,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	requiredSANDNS := []string{}
	if err := decodeJSONStringSlice(requiredSANDNSJSON, &requiredSANDNS); err != nil {
		return nil, fmt.Errorf("decode required_san_dns_json: %w", err)
	}
	requiredSANIPs := []string{}
	if err := decodeJSONStringSlice(requiredSANIPsJSON, &requiredSANIPs); err != nil {
		return nil, fmt.Errorf("decode required_san_ips_json: %w", err)
	}
	firstTestEGMIDs := []string{}
	if err := decodeJSONStringSlice(firstTestEGMIDsJSON, &firstTestEGMIDs); err != nil {
		return nil, fmt.Errorf("decode first_test_egm_ids_json: %w", err)
	}

	return &CabinetProfileOverride{
		Profile: config.CabinetProfile{
			WireHostURL:     wireHostURL,
			ListenerDNSName: listenerDNSName,
			ListenerIP:      listenerIP,
			RequiredSANDNS:  requiredSANDNS,
			RequiredSANIPs:  requiredSANIPs,
			HostID:          hostID,
			FirstTestEGMIDs: firstTestEGMIDs,
		},
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
	}, nil
}

func (s *SQLiteStore) UpsertCabinetProfileOverride(ctx context.Context, profile config.CabinetProfile, updatedBy string) error {
	requiredSANDNSJSON, err := encodeJSONStringSlice(profile.RequiredSANDNS)
	if err != nil {
		return err
	}
	requiredSANIPsJSON, err := encodeJSONStringSlice(profile.RequiredSANIPs)
	if err != nil {
		return err
	}
	firstTestEGMIDsJSON, err := encodeJSONStringSlice(profile.FirstTestEGMIDs)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO cabinet_profile_overrides (
		    id, wire_host_url, listener_dns_name, listener_ip, required_san_dns_json,
		    required_san_ips_json, host_id, first_test_egm_ids_json, updated_at, updated_by
		 ) VALUES (1, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?)
		 ON CONFLICT(id) DO UPDATE SET
		    wire_host_url = excluded.wire_host_url,
		    listener_dns_name = excluded.listener_dns_name,
		    listener_ip = excluded.listener_ip,
		    required_san_dns_json = excluded.required_san_dns_json,
		    required_san_ips_json = excluded.required_san_ips_json,
		    host_id = excluded.host_id,
		    first_test_egm_ids_json = excluded.first_test_egm_ids_json,
		    updated_at = CURRENT_TIMESTAMP,
		    updated_by = excluded.updated_by`,
		profile.WireHostURL,
		profile.ListenerDNSName,
		profile.ListenerIP,
		requiredSANDNSJSON,
		requiredSANIPsJSON,
		profile.HostID,
		firstTestEGMIDsJSON,
		updatedBy,
	)
	return err
}

func (s *SQLiteStore) ClearCabinetProfileOverride(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM cabinet_profile_overrides WHERE id = 1`)
	return err
}

func (s *SQLiteStore) GetHeartbeatPolicyOverride(ctx context.Context) (*HeartbeatPolicyOverride, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT COALESCE(interval_ms, 0), warning_after_missed, block_after_missed, updated_at, COALESCE(updated_by, '')
		   FROM heartbeat_policy_overrides
		  WHERE id = 1`,
	)

	var intervalMS int
	var warningAfterMissed int
	var blockAfterMissed int
	var updatedAt time.Time
	var updatedBy string
	if err := row.Scan(&intervalMS, &warningAfterMissed, &blockAfterMissed, &updatedAt, &updatedBy); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &HeartbeatPolicyOverride{
		IntervalMS:         intervalMS,
		WarningAfterMissed: warningAfterMissed,
		BlockAfterMissed:   blockAfterMissed,
		UpdatedAt:          updatedAt,
		UpdatedBy:          updatedBy,
	}, nil
}

func (s *SQLiteStore) UpsertHeartbeatPolicyOverride(ctx context.Context, intervalMS int, warningAfterMissed int, blockAfterMissed int, updatedBy string) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO heartbeat_policy_overrides (
		    id, interval_ms, warning_after_missed, block_after_missed, updated_at, updated_by
		 ) VALUES (1, ?, ?, ?, CURRENT_TIMESTAMP, ?)
		 ON CONFLICT(id) DO UPDATE SET
		    interval_ms = excluded.interval_ms,
		    warning_after_missed = excluded.warning_after_missed,
		    block_after_missed = excluded.block_after_missed,
		    updated_at = CURRENT_TIMESTAMP,
		    updated_by = excluded.updated_by`,
		intervalMS,
		warningAfterMissed,
		blockAfterMissed,
		updatedBy,
	)
	return err
}

func (s *SQLiteStore) ClearHeartbeatPolicyOverride(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM heartbeat_policy_overrides WHERE id = 1`)
	return err
}

func (s *SQLiteStore) GetBlockerPolicyOverride(ctx context.Context) (*BlockerPolicyOverride, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT approved_blocker_ids_json, updated_at, COALESCE(updated_by, '')
		   FROM blocker_policy_overrides
		  WHERE id = 1`,
	)

	var approvedJSON string
	var updatedAt time.Time
	var updatedBy string
	if err := row.Scan(&approvedJSON, &updatedAt, &updatedBy); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	approvedIDs := []string{}
	if err := decodeJSONStringSlice(approvedJSON, &approvedIDs); err != nil {
		return nil, fmt.Errorf("decode approved_blocker_ids_json: %w", err)
	}

	return &BlockerPolicyOverride{
		ApprovedBlockerIDs: approvedIDs,
		UpdatedAt:          updatedAt,
		UpdatedBy:          updatedBy,
	}, nil
}

func (s *SQLiteStore) UpsertBlockerPolicyOverride(ctx context.Context, approvedBlockerIDs []string, updatedBy string) error {
	approvedJSON, err := encodeJSONStringSlice(approvedBlockerIDs)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO blocker_policy_overrides (
		    id, approved_blocker_ids_json, updated_at, updated_by
		 ) VALUES (1, ?, CURRENT_TIMESTAMP, ?)
		 ON CONFLICT(id) DO UPDATE SET
		    approved_blocker_ids_json = excluded.approved_blocker_ids_json,
		    updated_at = CURRENT_TIMESTAMP,
		    updated_by = excluded.updated_by`,
		approvedJSON,
		updatedBy,
	)
	return err
}

func (s *SQLiteStore) ClearBlockerPolicyOverride(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM blocker_policy_overrides WHERE id = 1`)
	return err
}

func (s *SQLiteStore) GetSessionWorkflowProgress(ctx context.Context) (*model.SessionWorkflowProgress, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT current_phase, completed_steps_json, COALESCE(operator_notes, ''), updated_at
		   FROM session_workflow_progress
		  WHERE id = 1`,
	)

	var currentPhase string
	var completedStepsJSON string
	var operatorNotes string
	var updatedAt time.Time
	if err := row.Scan(&currentPhase, &completedStepsJSON, &operatorNotes, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	completedSteps := []string{}
	if err := decodeJSONStringSlice(completedStepsJSON, &completedSteps); err != nil {
		return nil, fmt.Errorf("decode completed_steps_json: %w", err)
	}

	return &model.SessionWorkflowProgress{
		CurrentPhase:   currentPhase,
		CompletedSteps: completedSteps,
		OperatorNotes:  operatorNotes,
		LastUpdatedAt:  updatedAt,
	}, nil
}

func (s *SQLiteStore) UpsertSessionWorkflowProgress(ctx context.Context, currentPhase string, completedSteps []string, operatorNotes string) error {
	completedStepsJSON, err := encodeJSONStringSlice(completedSteps)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO session_workflow_progress (
		    id, current_phase, completed_steps_json, operator_notes, updated_at
		 ) VALUES (1, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(id) DO UPDATE SET
		    current_phase = excluded.current_phase,
		    completed_steps_json = excluded.completed_steps_json,
		    operator_notes = excluded.operator_notes,
		    updated_at = CURRENT_TIMESTAMP`,
		currentPhase,
		completedStepsJSON,
		operatorNotes,
	)
	return err
}

func (s *SQLiteStore) ClearSessionWorkflowProgress(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM session_workflow_progress WHERE id = 1`)
	return err
}

func (s *SQLiteStore) RecordSessionEvidence(ctx context.Context, record model.SessionEvidenceRecord) (int64, error) {
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO session_evidence_records (
			created_at, overall_state, readyz_state, preflight_state, host_id, wire_host_url, operator_notes, payload_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		record.CreatedAt,
		record.OverallState,
		record.ReadyzState,
		record.PreflightState,
		record.HostID,
		record.WireHostURL,
		record.OperatorNotes,
		record.PayloadJSON,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *SQLiteStore) ListSessionEvidence(ctx context.Context, limit int) ([]model.SessionEvidenceRecord, error) {
	limit = normalizeLimit(limit)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, created_at, overall_state, readyz_state, preflight_state, host_id, wire_host_url, COALESCE(operator_notes, ''), payload_json
		 FROM session_evidence_records
		 ORDER BY id DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []model.SessionEvidenceRecord{}
	for rows.Next() {
		var record model.SessionEvidenceRecord
		if err := rows.Scan(
			&record.ID,
			&record.CreatedAt,
			&record.OverallState,
			&record.ReadyzState,
			&record.PreflightState,
			&record.HostID,
			&record.WireHostURL,
			&record.OperatorNotes,
			&record.PayloadJSON,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *SQLiteStore) DeleteSessionEvidence(ctx context.Context, id int64) error {
	_, err := s.DeleteSessionEvidenceByID(ctx, id)
	return err
}

func (s *SQLiteStore) DeleteSessionEvidenceByID(ctx context.Context, id int64) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM session_evidence_records WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return deleted > 0, nil
}

func (s *SQLiteStore) ListAllSessionEvidence(ctx context.Context) ([]model.SessionEvidenceRecord, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, created_at, overall_state, readyz_state, preflight_state, host_id, wire_host_url, COALESCE(operator_notes, ''), payload_json
		 FROM session_evidence_records
		 ORDER BY id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []model.SessionEvidenceRecord{}
	for rows.Next() {
		var record model.SessionEvidenceRecord
		if err := rows.Scan(
			&record.ID,
			&record.CreatedAt,
			&record.OverallState,
			&record.ReadyzState,
			&record.PreflightState,
			&record.HostID,
			&record.WireHostURL,
			&record.OperatorNotes,
			&record.PayloadJSON,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *SQLiteStore) RecordOperatorAuditEvent(ctx context.Context, event model.OperatorAuditEvent) (int64, error) {
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO operator_audit_events (
			created_at, action, result, actor_scope, egm_focus, summary, detail
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.Timestamp,
		event.Action,
		event.Result,
		event.ActorScope,
		strings.TrimSpace(event.EGMFocus),
		event.Summary,
		event.Detail,
	)
	if err != nil {
		return 0, err
	}
	if _, err := s.db.ExecContext(
		ctx,
		`DELETE FROM operator_audit_events
		  WHERE id NOT IN (
		      SELECT id FROM operator_audit_events
		      ORDER BY id DESC
		      LIMIT ?
		  )`,
		operatorAuditRetentionLimit,
	); err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *SQLiteStore) ListOperatorAuditEvents(ctx context.Context, query model.OperatorAuditQuery) ([]model.OperatorAuditEvent, error) {
	limit := normalizeLimit(query.Limit)
	where := []string{}
	args := []any{}

	if action := strings.TrimSpace(query.Action); action != "" {
		where = append(where, "action = ?")
		args = append(args, action)
	}
	if result := strings.ToLower(strings.TrimSpace(query.Result)); result != "" {
		where = append(where, "result = ?")
		args = append(args, result)
	}
	if search := strings.TrimSpace(query.Search); search != "" {
		searchLike := "%" + search + "%"
		where = append(where, `(action LIKE ? OR result LIKE ? OR actor_scope LIKE ? OR COALESCE(egm_focus, '') LIKE ? OR summary LIKE ? OR COALESCE(detail, '') LIKE ?)`)
		args = append(args, searchLike, searchLike, searchLike, searchLike, searchLike, searchLike)
	}

	sqlBuilder := strings.Builder{}
	sqlBuilder.WriteString(`SELECT id, created_at, action, result, actor_scope, COALESCE(egm_focus, ''), summary, COALESCE(detail, '')
		  FROM operator_audit_events`)
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

	events := []model.OperatorAuditEvent{}
	for rows.Next() {
		var event model.OperatorAuditEvent
		if err := rows.Scan(
			&event.ID,
			&event.Timestamp,
			&event.Action,
			&event.Result,
			&event.ActorScope,
			&event.EGMFocus,
			&event.Summary,
			&event.Detail,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *SQLiteStore) RecordRunMarker(ctx context.Context, marker model.RunMarker) (int64, error) {
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO run_markers (
			created_at, marker_type, title, notes, host_id, wire_host_url, operator_name
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		marker.CreatedAt,
		marker.MarkerType,
		marker.Title,
		marker.Notes,
		marker.HostID,
		marker.WireHostURL,
		marker.Operator,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *SQLiteStore) ListRunMarkers(ctx context.Context, limit int) ([]model.RunMarker, error) {
	limit = normalizeLimit(limit)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, created_at, marker_type, title, COALESCE(notes, ''), COALESCE(host_id, ''), COALESCE(wire_host_url, ''), COALESCE(operator_name, '')
		 FROM run_markers
		 ORDER BY id DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []model.RunMarker{}
	for rows.Next() {
		var record model.RunMarker
		if err := rows.Scan(
			&record.ID,
			&record.CreatedAt,
			&record.MarkerType,
			&record.Title,
			&record.Notes,
			&record.HostID,
			&record.WireHostURL,
			&record.Operator,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *SQLiteStore) Count(ctx context.Context, table string) (int, error) {
	switch table {
	case "incident_records", "egm_status_snapshots", "egm_compliance_logs", "controller_state_history", "certificate_inventory", "cabinet_profile_overrides", "session_evidence_records", "run_markers", "heartbeat_policy_overrides", "session_workflow_progress", "operator_audit_events", "blocker_policy_overrides":
	default:
		return 0, fmt.Errorf("unsupported count table %q", table)
	}
	row := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func statusWithError(record model.CertificateInventory) string {
	if record.Error == "" {
		return record.Status
	}
	return record.Status + ": " + record.Error
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func encodeJSONStringSlice(values []string) (string, error) {
	raw, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeJSONStringSlice(raw string, out *[]string) error {
	if strings.TrimSpace(raw) == "" {
		*out = []string{}
		return nil
	}
	return json.Unmarshal([]byte(raw), out)
}
