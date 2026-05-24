package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

func (s *SQLiteStore) GetInputRuntimeState(ctx context.Context, inputID string) (*inputruntime.InputRuntimeState, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT input_id, stable_raw_state, derived_state, stable_since,
		        COALESCE(last_observed_raw_state, ''), last_observed_at,
		        COALESCE(pending_raw_state, ''), pending_since,
		        COALESCE(last_transition_id, 0), updated_at
		   FROM input_runtime_states
		  WHERE input_id = ?`,
		inputID,
	)

	var state inputruntime.InputRuntimeState
	var lastObservedAt sql.NullTime
	var pendingSince sql.NullTime
	if err := row.Scan(
		&state.InputID,
		&state.StableRawState,
		&state.DerivedState,
		&state.StableSince,
		&state.LastObservedRawState,
		&lastObservedAt,
		&state.PendingRawState,
		&pendingSince,
		&state.LastTransitionID,
		&state.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if lastObservedAt.Valid {
		state.LastObservedAt = lastObservedAt.Time
	}
	if pendingSince.Valid {
		value := pendingSince.Time
		state.PendingSince = &value
	}
	return &state, nil
}

func (s *SQLiteStore) UpsertInputRuntimeState(ctx context.Context, state inputruntime.InputRuntimeState) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO input_runtime_states (
		    input_id, stable_raw_state, derived_state, stable_since,
		    last_observed_raw_state, last_observed_at, pending_raw_state, pending_since,
		    last_transition_id, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(input_id) DO UPDATE SET
		    stable_raw_state = excluded.stable_raw_state,
		    derived_state = excluded.derived_state,
		    stable_since = excluded.stable_since,
		    last_observed_raw_state = excluded.last_observed_raw_state,
		    last_observed_at = excluded.last_observed_at,
		    pending_raw_state = excluded.pending_raw_state,
		    pending_since = excluded.pending_since,
		    last_transition_id = excluded.last_transition_id,
		    updated_at = excluded.updated_at`,
		state.InputID,
		state.StableRawState,
		state.DerivedState,
		state.StableSince,
		nullableTrimmed(string(state.LastObservedRawState)),
		nullableNonZeroTime(state.LastObservedAt),
		nullableTrimmed(string(state.PendingRawState)),
		nullableTime(state.PendingSince),
		nullableInt64(state.LastTransitionID),
		state.UpdatedAt,
	)
	return err
}

func (s *SQLiteStore) RecordInputTransition(ctx context.Context, transition inputs.InputTransition) (int64, error) {
	if err := transition.Validate(); err != nil {
		return 0, err
	}
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO input_transitions (
		    input_channel_id, previous_derived_state, new_derived_state, transition_at, reason, action_run_id
		) VALUES (?, ?, ?, ?, ?, ?)`,
		transition.InputChannelID,
		transition.PreviousDerived,
		transition.NewDerived,
		transition.TransitionAt,
		nullableTrimmed(transition.Reason),
		nullableTrimmed(transition.ActionRunID),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *SQLiteStore) ListInputTransitions(ctx context.Context, limit int) ([]inputs.InputTransition, error) {
	limit = normalizeLimit(limit)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, input_channel_id, previous_derived_state, new_derived_state, transition_at,
		        COALESCE(reason, ''), COALESCE(action_run_id, '')
		   FROM input_transitions
		   ORDER BY id DESC
		   LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []inputs.InputTransition{}
	for rows.Next() {
		var transition inputs.InputTransition
		if err := rows.Scan(
			&transition.ID,
			&transition.InputChannelID,
			&transition.PreviousDerived,
			&transition.NewDerived,
			&transition.TransitionAt,
			&transition.Reason,
			&transition.ActionRunID,
		); err != nil {
			return nil, err
		}
		result = append(result, transition)
	}
	return result, rows.Err()
}

func nullableNonZeroTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
