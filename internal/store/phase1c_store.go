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
		        COALESCE(latch_active, 0), latch_cleared_at,
		        COALESCE(last_observed_raw_state, ''), last_observed_at,
		        COALESCE(pending_raw_state, ''), pending_since,
		        COALESCE(last_transition_id, 0), last_transition_at, updated_at
		   FROM input_runtime_states
		  WHERE input_id = ?`,
		inputID,
	)

	var state inputruntime.InputRuntimeState
	var latchActive int
	var latchClearedAt sql.NullTime
	var lastObservedAt sql.NullTime
	var pendingSince sql.NullTime
	var lastTransitionAt sql.NullTime
	if err := row.Scan(
		&state.InputID,
		&state.StableRawState,
		&state.DerivedState,
		&state.StableSince,
		&latchActive,
		&latchClearedAt,
		&state.LastObservedRawState,
		&lastObservedAt,
		&state.PendingRawState,
		&pendingSince,
		&state.LastTransitionID,
		&lastTransitionAt,
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
	state.LatchActive = latchActive != 0
	if latchClearedAt.Valid {
		value := latchClearedAt.Time
		state.LatchClearedAt = &value
	}
	if pendingSince.Valid {
		value := pendingSince.Time
		state.PendingSince = &value
	}
	if lastTransitionAt.Valid {
		value := lastTransitionAt.Time
		state.LastTransitionAt = &value
	}
	return &state, nil
}

func (s *SQLiteStore) UpsertInputRuntimeState(ctx context.Context, state inputruntime.InputRuntimeState) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO input_runtime_states (
		    input_id, stable_raw_state, derived_state, stable_since,
		    latch_active, latch_cleared_at,
		    last_observed_raw_state, last_observed_at, pending_raw_state, pending_since,
		    last_transition_id, last_transition_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(input_id) DO UPDATE SET
		    stable_raw_state = excluded.stable_raw_state,
		    derived_state = excluded.derived_state,
		    stable_since = excluded.stable_since,
		    latch_active = excluded.latch_active,
		    latch_cleared_at = excluded.latch_cleared_at,
		    last_observed_raw_state = excluded.last_observed_raw_state,
		    last_observed_at = excluded.last_observed_at,
		    pending_raw_state = excluded.pending_raw_state,
		    pending_since = excluded.pending_since,
		    last_transition_id = excluded.last_transition_id,
		    last_transition_at = excluded.last_transition_at,
		    updated_at = excluded.updated_at`,
		state.InputID,
		state.StableRawState,
		state.DerivedState,
		state.StableSince,
		boolToInt(state.LatchActive),
		nullableTime(state.LatchClearedAt),
		nullableTrimmed(string(state.LastObservedRawState)),
		nullableNonZeroTime(state.LastObservedAt),
		nullableTrimmed(string(state.PendingRawState)),
		nullableTime(state.PendingSince),
		nullableInt64(state.LastTransitionID),
		nullableTime(state.LastTransitionAt),
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

func (s *SQLiteStore) GetInputTransition(ctx context.Context, id int64) (*inputs.InputTransition, error) {
	if id <= 0 {
		return nil, nil
	}
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, input_channel_id, previous_derived_state, new_derived_state, transition_at,
		        COALESCE(reason, ''), COALESCE(action_run_id, '')
		   FROM input_transitions
		  WHERE id = ?`,
		id,
	)
	var transition inputs.InputTransition
	if err := row.Scan(
		&transition.ID,
		&transition.InputChannelID,
		&transition.PreviousDerived,
		&transition.NewDerived,
		&transition.TransitionAt,
		&transition.Reason,
		&transition.ActionRunID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &transition, nil
}

func nullableNonZeroTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
