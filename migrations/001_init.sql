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
