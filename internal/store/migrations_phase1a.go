package store

const phase1ADomainMigration = `
CREATE TABLE IF NOT EXISTS input_channels (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    gpio_channel TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    normal_state TEXT NOT NULL,
    current_state TEXT NOT NULL,
    derived_state TEXT NOT NULL,
    debounce_ms INTEGER NOT NULL DEFAULT 0,
    priority INTEGER NOT NULL DEFAULT 0,
    on_trigger_action_id TEXT,
    on_normal_action_id TEXT,
    latching_mode TEXT NOT NULL,
    last_transition_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS input_transitions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    input_channel_id TEXT NOT NULL,
    previous_derived_state TEXT NOT NULL,
    new_derived_state TEXT NOT NULL,
    transition_at DATETIME NOT NULL,
    reason TEXT,
    action_run_id TEXT
);

CREATE TABLE IF NOT EXISTS action_definitions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    severity TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    target_selector TEXT NOT NULL,
    template_selector TEXT NOT NULL,
    steps_json TEXT NOT NULL,
    retry_policy_json TEXT,
    escalation_policy_json TEXT,
    return_action_id TEXT,
    audit_policy_json TEXT,
    version INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS action_runs (
    id TEXT PRIMARY KEY,
    action_definition_id TEXT NOT NULL,
    incident_id TEXT,
    input_transition_id INTEGER,
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    status TEXT NOT NULL,
    trigger_reason TEXT,
    target_count INTEGER NOT NULL DEFAULT 0,
    confirmed_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    escalated_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS action_target_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action_run_id TEXT NOT NULL,
    target_egm_id TEXT NOT NULL,
    status TEXT NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    last_result_at DATETIME
);

CREATE TABLE IF NOT EXISTS g2s_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    vendor TEXT NOT NULL,
    cabinet_family TEXT,
    software_version_match TEXT,
    status TEXT NOT NULL,
    current_version_id TEXT,
    notes TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS g2s_template_versions (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    version_label TEXT NOT NULL,
    endpoint_quirks_json TEXT,
    actions_json TEXT NOT NULL,
    confirmation_rules_json TEXT,
    failure_rules_json TEXT,
    heartbeat_profile_json TEXT,
    variables_json TEXT,
    notes TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS message_journal (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    direction TEXT NOT NULL,
    from_endpoint TEXT,
    to_endpoint TEXT,
    egm_id TEXT,
    action_run_id TEXT,
    action_step_id TEXT,
    input_transition_id INTEGER,
    template_id TEXT,
    template_version TEXT,
    handler_rule_id TEXT,
    message_type TEXT,
    raw_payload TEXT NOT NULL,
    parsed_summary_json TEXT,
    result TEXT NOT NULL,
    error TEXT
);

CREATE TABLE IF NOT EXISTS handler_rules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    direction TEXT NOT NULL DEFAULT 'ANY',
    template_id TEXT,
    message_type TEXT,
    egm_id TEXT,
    action_id TEXT,
    action_step_id TEXT,
    match_json TEXT NOT NULL,
    outcome TEXT NOT NULL DEFAULT 'NOTE',
    handle_json TEXT NOT NULL,
    notes TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS egm_records (
    egm_id TEXT PRIMARY KEY,
    display_name TEXT,
    ip_address TEXT,
    endpoint_path TEXT,
    vendor TEXT,
    cabinet_family TEXT,
    game_title TEXT,
    software_version TEXT,
    zone TEXT,
    enabled INTEGER NOT NULL DEFAULT 1,
    emergency_enabled INTEGER NOT NULL DEFAULT 1,
    template_id TEXT,
    heartbeat_override_json TEXT,
    last_seen_at DATETIME,
    current_action_state TEXT NOT NULL,
    notes TEXT
);

CREATE TABLE IF NOT EXISTS egm_groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    egm_ids_json TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS audit_timeline (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    occurred_at DATETIME NOT NULL,
    severity TEXT NOT NULL,
    event_type TEXT NOT NULL,
    summary TEXT NOT NULL,
    detail_json TEXT,
    action_run_id TEXT,
    input_transition_id INTEGER,
    message_journal_id INTEGER,
    operator TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_input_transitions_channel_time ON input_transitions (input_channel_id, transition_at DESC);
CREATE INDEX IF NOT EXISTS idx_action_runs_definition_time ON action_runs (action_definition_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_message_journal_time ON message_journal (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_timeline_time ON audit_timeline (occurred_at DESC);
`
