-- Phase 1C additive runtime input-state table.
CREATE TABLE IF NOT EXISTS input_runtime_states (
    input_id TEXT PRIMARY KEY,
    stable_raw_state TEXT NOT NULL,
    derived_state TEXT NOT NULL,
    stable_since DATETIME NOT NULL,
    last_observed_raw_state TEXT,
    last_observed_at DATETIME,
    pending_raw_state TEXT,
    pending_since DATETIME,
    last_transition_id INTEGER,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_input_runtime_updated_at ON input_runtime_states (updated_at DESC);
