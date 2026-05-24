package store

const phase2F1InputSafetyMigration = `
ALTER TABLE input_runtime_states ADD COLUMN latch_active INTEGER NOT NULL DEFAULT 0;
ALTER TABLE input_runtime_states ADD COLUMN latch_cleared_at DATETIME;
ALTER TABLE input_runtime_states ADD COLUMN last_transition_at DATETIME;
`
