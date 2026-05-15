package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tschneider-imagine/G2S_MC/internal/model"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
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
	_, err := s.db.ExecContext(ctx, InitMigration)
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

func (s *SQLiteStore) Count(ctx context.Context, table string) (int, error) {
	switch table {
	case "incident_records", "egm_status_snapshots", "egm_compliance_logs", "controller_state_history":
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
