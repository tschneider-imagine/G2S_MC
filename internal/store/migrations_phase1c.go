package store

const phase1CInputRuntimeMigration = `
CREATE TABLE IF NOT EXISTS input_runtime_states (
    input_id TEXT PRIMARY KEY,
    stable_raw_state TEXT NOT NULL,
    derived_state TEXT NOT NULL,
    stable_since DATETIME NOT NULL,
    latch_active INTEGER NOT NULL DEFAULT 0,
    latch_cleared_at DATETIME,
    last_observed_raw_state TEXT,
    last_observed_at DATETIME,
    pending_raw_state TEXT,
    pending_since DATETIME,
    last_transition_id INTEGER,
    last_transition_at DATETIME,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_input_runtime_updated_at ON input_runtime_states (updated_at DESC);
`
