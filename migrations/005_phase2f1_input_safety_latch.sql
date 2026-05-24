-- Phase 2F.1 additive manual-clear latch fields for input runtime state.
ALTER TABLE input_runtime_states ADD COLUMN latch_active INTEGER NOT NULL DEFAULT 0;
ALTER TABLE input_runtime_states ADD COLUMN latch_cleared_at DATETIME;
ALTER TABLE input_runtime_states ADD COLUMN last_transition_at DATETIME;
