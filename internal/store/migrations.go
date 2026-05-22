package store

const InitMigration = `
CREATE TABLE IF NOT EXISTS incident_records (
    incident_id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    trigger_type TEXT NOT NULL,
    trigger_source TEXT,
    duration_ms INTEGER,
    resolved_at DATETIME,
    final_state TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS egm_compliance_logs (
    log_id INTEGER PRIMARY KEY AUTOINCREMENT,
    incident_id INTEGER NOT NULL,
    egm_id TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    action_sent TEXT NOT NULL,
    status_result TEXT NOT NULL,
    http_status_code INTEGER,
    latency_ms INTEGER,
    response_excerpt TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(incident_id) REFERENCES incident_records(incident_id)
);

CREATE TABLE IF NOT EXISTS egm_status_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    egm_id TEXT NOT NULL,
    status TEXT NOT NULL,
    event_type TEXT NOT NULL,
    detail TEXT,
    last_error TEXT
);

CREATE TABLE IF NOT EXISTS controller_state_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    old_state TEXT NOT NULL,
    new_state TEXT NOT NULL,
    reason TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS certificate_inventory (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cert_role TEXT NOT NULL,
    path TEXT NOT NULL,
    subject TEXT,
    issuer TEXT,
    not_before DATETIME,
    not_after DATETIME,
    sha256_fingerprint TEXT,
    last_checked_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS operator_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    username TEXT NOT NULL,
    action TEXT NOT NULL,
    target TEXT,
    result TEXT NOT NULL,
    remote_addr TEXT
);

CREATE TABLE IF NOT EXISTS cabinet_profile_overrides (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    wire_host_url TEXT,
    listener_dns_name TEXT,
    listener_ip TEXT,
    required_san_dns_json TEXT,
    required_san_ips_json TEXT,
    host_id TEXT,
    first_test_egm_ids_json TEXT,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT
);

CREATE TABLE IF NOT EXISTS session_evidence_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    overall_state TEXT NOT NULL,
    readyz_state TEXT NOT NULL,
    preflight_state TEXT NOT NULL,
    host_id TEXT NOT NULL,
    wire_host_url TEXT NOT NULL,
    operator_notes TEXT,
    payload_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS run_markers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    marker_type TEXT NOT NULL,
    title TEXT NOT NULL,
    notes TEXT,
    host_id TEXT,
    wire_host_url TEXT,
    operator_name TEXT
);

CREATE TABLE IF NOT EXISTS heartbeat_policy_overrides (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    interval_ms INTEGER,
    warning_after_missed INTEGER NOT NULL,
    block_after_missed INTEGER NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT
);

CREATE TABLE IF NOT EXISTS blocker_policy_overrides (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    approved_blocker_ids_json TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT,
    last_change_action TEXT,
    last_change_rationale TEXT,
    last_change_actor_scope TEXT
);

CREATE TABLE IF NOT EXISTS blocker_policy_escalation_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    action TEXT NOT NULL,
    finding_id TEXT NOT NULL,
    rationale TEXT,
    actor_scope TEXT NOT NULL,
    egm_focus TEXT,
    updated_by TEXT
);

CREATE TABLE IF NOT EXISTS session_workflow_progress (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    current_phase TEXT NOT NULL,
    completed_steps_json TEXT NOT NULL,
    operator_notes TEXT,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS operator_audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    action TEXT NOT NULL,
    result TEXT NOT NULL,
    actor_scope TEXT NOT NULL,
    egm_focus TEXT,
    summary TEXT NOT NULL,
    detail TEXT
);

CREATE TABLE IF NOT EXISTS endpoint_integrity_alert_states (
    alert_id TEXT PRIMARY KEY,
    acked_at DATETIME,
    acked_by_scope TEXT,
    snoozed_until DATETIME,
    snooze_reason TEXT,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT
);

CREATE TABLE IF NOT EXISTS egm_registry_overrides (
    egm_id TEXT PRIMARY KEY,
    display_name TEXT,
    vendor TEXT,
    cabinet_family TEXT,
    game_title TEXT,
    software_version TEXT,
    notes TEXT,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT
);
`
