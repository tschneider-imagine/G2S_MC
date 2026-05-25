package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/incidents"
)

func (s *SQLiteStore) ensureIncidentLifecycleSchema(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(incident_records)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	existing := map[string]bool{}
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
		existing[strings.ToLower(strings.TrimSpace(name))] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	type columnDef struct {
		name string
		def  string
	}
	defs := []columnDef{
		{name: "status", def: "TEXT NOT NULL DEFAULT 'OPEN'"},
		{name: "severity", def: "TEXT"},
		{name: "primary_input_id", def: "TEXT"},
		{name: "primary_action_run_id", def: "TEXT"},
		{name: "opened_by_transition_id", def: "INTEGER"},
		{name: "closed_by_transition_id", def: "INTEGER"},
		{name: "close_reason", def: "TEXT"},
		{name: "summary", def: "TEXT"},
		{name: "detail_json", def: "TEXT"},
	}
	for _, def := range defs {
		if existing[def.name] {
			continue
		}
		if _, err := s.db.ExecContext(ctx, "ALTER TABLE incident_records ADD COLUMN "+def.name+" "+def.def); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) CreateIncidentRecord(ctx context.Context, record incidents.IncidentRecord) (incidents.IncidentRecord, error) {
	if record.OpenedAt.IsZero() {
		record.OpenedAt = time.Now().UTC()
	}
	if strings.TrimSpace(string(record.Status)) == "" {
		record.Status = incidents.StatusOpen
	}
	finalState := string(record.Status)
	if finalState == "" {
		finalState = "OPEN"
	}
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO incident_records (
			created_at, trigger_type, trigger_source, resolved_at, final_state,
			status, severity, primary_input_id, primary_action_run_id, opened_by_transition_id,
			closed_by_transition_id, close_reason, summary, detail_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.OpenedAt,
		"INPUT_TRIGGERED",
		nullableTrimmed(record.PrimaryInputID),
		nullableTime(record.ClosedAt),
		finalState,
		record.Status,
		nullableTrimmed(record.Severity),
		nullableTrimmed(record.PrimaryInputID),
		nullableTrimmed(record.PrimaryActionRunID),
		nullableInt64(record.OpenedByTransitionID),
		nullableInt64(record.ClosedByTransitionID),
		nullableTrimmed(record.CloseReason),
		nullableTrimmed(record.Summary),
		nullableTrimmed(record.DetailJSON),
	)
	if err != nil {
		return incidents.IncidentRecord{}, err
	}
	record.ID, err = result.LastInsertId()
	if err != nil {
		return incidents.IncidentRecord{}, err
	}
	return record, nil
}

func (s *SQLiteStore) GetIncidentRecord(ctx context.Context, id int64) (*incidents.IncidentRecord, error) {
	if id <= 0 {
		return nil, nil
	}
	row := s.db.QueryRowContext(
		ctx,
		`SELECT incident_id, created_at, resolved_at, COALESCE(status, ''), COALESCE(severity, ''),
		        COALESCE(primary_input_id, ''), COALESCE(primary_action_run_id, ''), COALESCE(opened_by_transition_id, 0),
		        COALESCE(closed_by_transition_id, 0), COALESCE(close_reason, ''), COALESCE(summary, ''), COALESCE(detail_json, '')
		   FROM incident_records
		  WHERE incident_id = ?`,
		id,
	)
	return scanIncidentRecord(row)
}

func (s *SQLiteStore) GetOpenIncidentByInput(ctx context.Context, inputID string) (*incidents.IncidentRecord, error) {
	trimmed := strings.TrimSpace(inputID)
	if trimmed == "" {
		return nil, nil
	}
	row := s.db.QueryRowContext(
		ctx,
		`SELECT incident_id, created_at, resolved_at, COALESCE(status, ''), COALESCE(severity, ''),
		        COALESCE(primary_input_id, ''), COALESCE(primary_action_run_id, ''), COALESCE(opened_by_transition_id, 0),
		        COALESCE(closed_by_transition_id, 0), COALESCE(close_reason, ''), COALESCE(summary, ''), COALESCE(detail_json, '')
		   FROM incident_records
		  WHERE primary_input_id = ?
		    AND (UPPER(COALESCE(status, 'OPEN')) = 'OPEN')
		  ORDER BY incident_id DESC
		  LIMIT 1`,
		trimmed,
	)
	return scanIncidentRecord(row)
}

func (s *SQLiteStore) ListOpenIncidentRecords(ctx context.Context, limit int) ([]incidents.IncidentRecord, error) {
	limit = normalizeLimit(limit)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT incident_id, created_at, resolved_at, COALESCE(status, ''), COALESCE(severity, ''),
		        COALESCE(primary_input_id, ''), COALESCE(primary_action_run_id, ''), COALESCE(opened_by_transition_id, 0),
		        COALESCE(closed_by_transition_id, 0), COALESCE(close_reason, ''), COALESCE(summary, ''), COALESCE(detail_json, '')
		   FROM incident_records
		  WHERE UPPER(COALESCE(status, 'OPEN')) = 'OPEN'
		  ORDER BY incident_id DESC
		  LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]incidents.IncidentRecord, 0, limit)
	for rows.Next() {
		row, scanErr := scanIncidentRecordRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) CloseIncidentRecord(ctx context.Context, id int64, closedAt time.Time, closedByTransitionID int64, closeReason string) (*incidents.IncidentRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("incident id is required")
	}
	if closedAt.IsZero() {
		closedAt = time.Now().UTC()
	}
	record, err := s.GetIncidentRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}
	record.Status = incidents.StatusClosed
	record.ClosedAt = &closedAt
	record.ClosedByTransitionID = closedByTransitionID
	record.CloseReason = strings.TrimSpace(closeReason)

	durationMS := int64(0)
	if !record.OpenedAt.IsZero() {
		durationMS = closedAt.Sub(record.OpenedAt).Milliseconds()
		if durationMS < 0 {
			durationMS = 0
		}
	}
	_, err = s.db.ExecContext(
		ctx,
		`UPDATE incident_records
		    SET resolved_at = ?,
		        final_state = ?,
		        duration_ms = ?,
		        status = ?,
		        closed_by_transition_id = ?,
		        close_reason = ?
		  WHERE incident_id = ?`,
		closedAt,
		string(incidents.StatusClosed),
		durationMS,
		incidents.StatusClosed,
		nullableInt64(closedByTransitionID),
		nullableTrimmed(record.CloseReason),
		id,
	)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (s *SQLiteStore) UpdateIncidentPrimaryActionRun(ctx context.Context, id int64, actionRunID string) error {
	if id <= 0 {
		return fmt.Errorf("incident id is required")
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE incident_records
		    SET primary_action_run_id = ?
		  WHERE incident_id = ?`,
		nullableTrimmed(actionRunID),
		id,
	)
	return err
}

func (s *SQLiteStore) ListActionRunsByIncident(ctx context.Context, incidentID string, limit int) ([]string, error) {
	trimmed := strings.TrimSpace(incidentID)
	if trimmed == "" {
		return nil, nil
	}
	limit = normalizeLimit(limit)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id
		   FROM action_runs
		  WHERE incident_id = ?
		  ORDER BY started_at DESC, id DESC
		  LIMIT ?`,
		trimmed,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, strings.TrimSpace(id))
	}
	return ids, rows.Err()
}

func scanIncidentRecord(row *sql.Row) (*incidents.IncidentRecord, error) {
	record, err := scanIncidentRecordScanner(row.Scan)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func scanIncidentRecordRows(rows *sql.Rows) (incidents.IncidentRecord, error) {
	return scanIncidentRecordScanner(rows.Scan)
}

func scanIncidentRecordScanner(scan func(dest ...any) error) (incidents.IncidentRecord, error) {
	var record incidents.IncidentRecord
	var resolvedAt sql.NullTime
	var statusRaw string
	if err := scan(
		&record.ID,
		&record.OpenedAt,
		&resolvedAt,
		&statusRaw,
		&record.Severity,
		&record.PrimaryInputID,
		&record.PrimaryActionRunID,
		&record.OpenedByTransitionID,
		&record.ClosedByTransitionID,
		&record.CloseReason,
		&record.Summary,
		&record.DetailJSON,
	); err != nil {
		return incidents.IncidentRecord{}, err
	}
	if resolvedAt.Valid {
		value := resolvedAt.Time
		record.ClosedAt = &value
	}
	switch strings.ToUpper(strings.TrimSpace(statusRaw)) {
	case "CLOSED":
		record.Status = incidents.StatusClosed
	default:
		record.Status = incidents.StatusOpen
	}
	return record, nil
}

func parseIncidentID(raw string) (int64, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
