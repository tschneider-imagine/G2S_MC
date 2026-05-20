package ui

const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <link rel="stylesheet" href="/static/dashboard.css">
</head>
<body>
  <header class="topbar">
    <div>
      <p class="eyebrow">Local appliance</p>
      <h1>G2S Muting Controller</h1>
    </div>
    <div class="top-actions">
      <span id="last-refresh">Waiting for telemetry</span>
      <span id="stale-badge" class="stale-badge stale-critical">No successful snapshot</span>
      <button id="refresh-button" type="button">Refresh</button>
    </div>
  </header>

  <section id="operator-alert" class="alert-strip alert-info">
    <strong id="alert-title">Waiting for readiness</strong>
    <span id="alert-detail">The console is collecting first telemetry snapshots.</span>
  </section>
  <section id="api-failure-banner" class="api-banner api-banner-hidden">
    <strong>API polling issue</strong>
    <span id="api-failure-detail">One or more API requests failed; showing cached data where available.</span>
  </section>

  <main class="shell">
    <section class="status-band">
      <div>
        <p class="label">Controller</p>
        <strong id="controller-id">-</strong>
      </div>
      <div>
        <p class="label">State</p>
        <strong id="controller-state" class="state-pill">-</strong>
      </div>
      <div>
        <p class="label">Status Readiness</p>
        <strong id="readiness-state" class="state-pill">-</strong>
      </div>
      <div>
        <p class="label">/readyz Primary</p>
        <strong id="readyz-state" class="state-pill">-</strong>
      </div>
      <div>
        <p class="label">Inputs</p>
        <strong id="input-mode">-</strong>
      </div>
      <div>
        <p class="label">Last event</p>
        <strong id="last-event">-</strong>
      </div>
      <div>
        <p class="label">Active incident</p>
        <strong id="active-incident">None</strong>
      </div>
    </section>

    <section class="grid two">
      <div class="panel">
        <div class="panel-head">
          <h2>Appliance Readiness</h2>
          <span id="uptime">-</span>
        </div>
        <dl class="kv-list">
          <div><dt>Bind</dt><dd id="bind-address">-</dd></div>
          <div><dt>Database</dt><dd id="database-path">-</dd></div>
          <div><dt>G2S endpoint</dt><dd id="g2s-endpoint">-</dd></div>
          <div><dt>TLS mode</dt><dd id="tls-mode">-</dd></div>
          <div><dt>/readyz HTTP</dt><dd id="readyz-http">-</dd></div>
          <div><dt>/readyz issues</dt><dd id="readyz-issues">-</dd></div>
          <div><dt>Warnings</dt><dd id="readiness-warnings">-</dd></div>
        </dl>
      </div>

      <div class="panel">
        <div class="panel-head">
          <h2>Certificate Summary</h2>
        </div>
        <div id="certificate-summary" class="summary-grid"></div>
      </div>
    </section>

    <section class="grid">
      <div class="panel">
        <div class="panel-head">
          <h2>Cabinet Identity Profile</h2>
          <span id="cabinet-profile-source" class="source-pill source-file">file</span>
        </div>
        <dl class="kv-list">
          <div><dt>Wire Host URL</dt><dd id="cabinet-wire-host-url">-</dd></div>
          <div><dt>Listener DNS</dt><dd id="cabinet-listener-dns">-</dd></div>
          <div><dt>Listener IP</dt><dd id="cabinet-listener-ip">-</dd></div>
          <div><dt>Required SAN DNS</dt><dd id="cabinet-required-san-dns">-</dd></div>
          <div><dt>Required SAN IPs</dt><dd id="cabinet-required-san-ips">-</dd></div>
          <div><dt>Host ID</dt><dd id="cabinet-host-id">-</dd></div>
          <div><dt>First Test EGM IDs</dt><dd id="cabinet-first-test-egm-ids">-</dd></div>
          <div><dt>Profile Updated</dt><dd id="cabinet-profile-updated-at">-</dd></div>
          <div><dt>Profile Warning</dt><dd id="cabinet-profile-warning">None</dd></div>
        </dl>
      </div>
    </section>

    <section class="grid two">
      <div class="panel first-cabinet-session-panel">
        <div class="panel-head panel-head-stack">
          <div class="panel-title-row">
            <h2>First Cabinet Session</h2>
            <span id="first-cabinet-session-state" class="source-pill source-file">checking</span>
          </div>
          <span id="first-cabinet-session-message" class="muted-text">Waiting for first cabinet readiness telemetry.</span>
        </div>
        <dl class="kv-list">
          <div><dt>Overall Session State</dt><dd id="first-cabinet-overall">-</dd></div>
          <div><dt>Last Checked</dt><dd id="first-cabinet-last-checked">-</dd></div>
          <div><dt>Readyz State</dt><dd id="first-cabinet-readyz">-</dd></div>
          <div><dt>Cabinet Preflight State</dt><dd id="first-cabinet-preflight">-</dd></div>
          <div><dt>Cabinet Profile Source</dt><dd id="first-cabinet-profile-source">-</dd></div>
          <div><dt>Wire Host URL</dt><dd id="first-cabinet-wire-host-url">-</dd></div>
          <div><dt>Host ID</dt><dd id="first-cabinet-host-id">-</dd></div>
          <div><dt>First Test EGM IDs</dt><dd id="first-cabinet-egm-ids">-</dd></div>
          <div><dt>Certificate Blocking Count</dt><dd id="first-cabinet-cert-blocking">-</dd></div>
          <div><dt>Lab Optional Certificate Count</dt><dd id="first-cabinet-cert-lab-optional">-</dd></div>
          <div><dt>API Auth State</dt><dd id="first-cabinet-auth-state">-</dd></div>
        </dl>
        <div class="first-cabinet-session-blockers-wrap">
          <p class="label">Session Blockers</p>
          <div id="first-cabinet-session-blockers" class="first-cabinet-session-blockers"></div>
        </div>
      </div>

      <div class="panel evidence-capture-panel">
        <div class="panel-head panel-head-stack">
          <div class="panel-title-row">
            <h2>Session Evidence Capture</h2>
            <span id="session-evidence-state" class="source-pill source-file">ready</span>
          </div>
          <span id="session-evidence-message" class="muted-text">Capture the current cabinet session state as JSON or Markdown.</span>
        </div>
        <dl class="kv-list">
          <div><dt>Current Session State</dt><dd id="session-evidence-overall">-</dd></div>
          <div><dt>Snapshot Timestamp</dt><dd id="session-evidence-timestamp">-</dd></div>
          <div><dt>Incidents in Snapshot</dt><dd id="session-evidence-incident-count">0</dd></div>
          <div><dt>State History Rows</dt><dd id="session-evidence-state-count">0</dd></div>
        </dl>
        <label class="cert-textarea-label evidence-notes-label">Operator Notes
          <textarea id="session-evidence-notes" rows="5" placeholder="Optional test notes, cabinet observations, or follow-up context."></textarea>
        </label>
        <div class="setup-actions evidence-actions">
          <button id="session-evidence-save-button" type="button">Save to Appliance History</button>
          <button id="session-evidence-json-button" type="button">Download JSON Evidence</button>
          <button id="session-evidence-markdown-button" type="button" class="secondary-button">Download Markdown Evidence</button>
          <button id="session-evidence-export-all-button" type="button" class="secondary-button">Download All Saved Evidence</button>
        </div>
        <div class="first-cabinet-session-blockers-wrap">
          <p class="label">Recent Captures</p>
          <div id="session-evidence-history" class="timeline"></div>
        </div>
        <div class="first-cabinet-session-blockers-wrap">
          <p class="label">Selected Saved Capture</p>
          <div id="session-evidence-selected" class="timeline"></div>
        </div>
      </div>
    </section>

    <section class="grid">
      <div class="panel">
        <div class="panel-head panel-head-stack">
          <div class="panel-title-row">
            <h2>Cabinet Setup</h2>
            <span id="cabinet-setup-state" class="source-pill source-file">ready</span>
          </div>
          <span id="cabinet-setup-message" class="muted-text">Current values loaded from the appliance.</span>
        </div>
        <form id="cabinet-setup-form" class="setup-form">
          <div class="form-grid">
            <label>Wire Host URL<input id="setup-wire-host-url" name="wire_host_url" autocomplete="off"></label>
            <label>Listener DNS<input id="setup-listener-dns" name="listener_dns_name" autocomplete="off"></label>
            <label>Listener IP<input id="setup-listener-ip" name="listener_ip" autocomplete="off"></label>
            <label>Host ID<input id="setup-host-id" name="host_id" autocomplete="off"></label>
            <label>Required SAN DNS<input id="setup-required-san-dns" name="required_san_dns" autocomplete="off"></label>
            <label>Required SAN IPs<input id="setup-required-san-ips" name="required_san_ips" autocomplete="off"></label>
            <label>First Test EGM IDs<input id="setup-first-test-egm-ids" name="first_test_egm_ids" autocomplete="off"></label>
            <label id="setup-api-token-wrapper"><span id="setup-api-token-label">API Token Required for Save/Clear</span><input id="setup-api-token" name="api_token" type="password" autocomplete="off"></label>
          </div>
          <div id="setup-token-controls" class="token-help">
            <span id="setup-token-help-text">Enter the appliance API token to save or clear cabinet setup overrides.</span>
            <button id="setup-copy-token-button" type="button" class="secondary-button" disabled>Copy Entered Token</button>
          </div>
          <div class="setup-details">
            <div>
              <p class="label">SAN Expectation</p>
              <strong id="setup-san-summary">-</strong>
            </div>
            <div>
              <p class="label">Validation</p>
              <strong id="setup-validation-summary">-</strong>
            </div>
          </div>
          <div id="setup-validation-list" class="validation-list"></div>
          <div class="setup-actions">
            <button id="setup-save-button" type="submit" disabled>Save Override</button>
            <button id="setup-reset-button" type="button" class="secondary-button" disabled>Clear Override</button>
            <button id="setup-reload-button" type="button" class="secondary-button">Reload</button>
          </div>
        </form>
      </div>
    </section>

    <section class="grid two">
      <div class="panel wide">
        <div class="panel-head panel-head-stack">
          <div class="panel-title-row">
            <h2>EGM Roster</h2>
            <span id="egm-count">0 EGMs</span>
          </div>
          <div class="toolbar-row">
            <div class="filter-tabs" role="tablist" aria-label="EGM filter tabs">
              <button type="button" class="filter-tab is-active" data-filter="all">All</button>
              <button type="button" class="filter-tab" data-filter="healthy">Healthy</button>
              <button type="button" class="filter-tab" data-filter="unhealthy">Unhealthy</button>
            </div>
            <span id="egm-sort-label" class="muted-text">Sort: EGM ID asc</span>
          </div>
        </div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th><button type="button" class="sort-button" data-sort-key="egm_id">EGM</button></th>
                <th><button type="button" class="sort-button" data-sort-key="status">Status</button></th>
                <th>Address</th>
                <th>Game</th>
                <th><button type="button" class="sort-button" data-sort-key="last_seen">Last seen</button></th>
              </tr>
            </thead>
            <tbody id="egm-table">
              <tr><td colspan="5">Loading...</td></tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <h2>Recent Incidents</h2>
        </div>
        <div id="incident-list" class="timeline"></div>
      </div>
    </section>

    <section class="grid two">
      <div class="panel">
        <div class="panel-head">
          <h2>EGM History</h2>
        </div>
        <div id="egm-history" class="timeline"></div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <h2>State History</h2>
        </div>
        <div id="state-history" class="timeline"></div>
      </div>
    </section>

    <section class="grid two">
      <div class="panel">
        <div class="panel-head panel-head-stack">
          <div class="panel-title-row">
            <h2>Certificate Manager</h2>
            <span id="cert-manager-state" class="source-pill source-file">ready</span>
          </div>
          <span id="cert-manager-message" class="muted-text">Import and export certificate materials for configured roles.</span>
        </div>
        <form id="cert-manager-form" class="setup-form cert-manager-form">
          <div class="form-grid cert-form-grid">
            <label>Role
              <select id="cert-role-select" name="role">
                <option value="g2s_ca_cert">CA Certificate</option>
                <option value="g2s_client_cert">Client Certificate + Key</option>
                <option value="web_server_cert">Web Server Certificate + Key</option>
              </select>
            </label>
            <label id="cert-api-token-wrapper"><span id="cert-api-token-label">API Token (required for import/export key)</span><input id="cert-api-token" name="api_token" type="password" autocomplete="off"></label>
          </div>
          <div id="cert-token-controls" class="token-help">
            <span id="cert-token-help-text">Use API token for import and private-key export actions.</span>
            <button id="cert-copy-token-button" type="button" class="secondary-button" disabled>Copy Entered Token</button>
          </div>
          <div id="cert-role-summary" class="cert-role-summary">Select a role to review support and status.</div>
          <div class="setup-details cert-manager-details">
            <div>
              <p class="label">Role Rules</p>
              <strong id="cert-role-rules">-</strong>
            </div>
            <div>
              <p class="label">Current Role Status</p>
              <strong id="cert-role-current-status">-</strong>
            </div>
            <div>
              <p class="label">Export Key Access</p>
              <strong id="cert-role-export-policy">-</strong>
            </div>
            <div>
              <p class="label">Import Validation</p>
              <strong id="cert-validation-summary">-</strong>
            </div>
          </div>
          <div id="cert-validation-list" class="validation-list"></div>
          <label class="cert-textarea-label">Certificate PEM
            <textarea id="cert-certificate-pem" name="certificate_pem" rows="8" placeholder="-----BEGIN CERTIFICATE-----"></textarea>
          </label>
          <label id="cert-private-key-wrapper" class="cert-textarea-label">Private Key PEM
            <textarea id="cert-private-key-pem" name="private_key_pem" rows="8" placeholder="-----BEGIN PRIVATE KEY-----"></textarea>
          </label>
          <div class="setup-actions cert-actions">
            <button id="cert-import-button" type="submit">Import Role Certificate</button>
            <button id="cert-export-cert-button" type="button" class="secondary-button">Export Selected Public Cert</button>
            <button id="cert-export-key-button" type="button" class="secondary-button">Export Selected Cert + Key</button>
            <button id="cert-clear-form-button" type="button" class="secondary-button">Clear Form</button>
          </div>
        </form>
        <div class="cert-exports">
          <p class="label">Quick Public Cert Exports</p>
          <div class="cert-export-buttons">
            <button type="button" class="secondary-button cert-export-role-button" data-export-role="g2s_ca_cert">Export CA Cert</button>
            <button type="button" class="secondary-button cert-export-role-button" data-export-role="g2s_client_cert">Export Client Cert</button>
            <button type="button" class="secondary-button cert-export-role-button" data-export-role="web_server_cert">Export Web Server Cert</button>
          </div>
        </div>
        <div id="cert-manager-detail" class="cert-manager-detail muted-text"></div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <h2>Certificate Inventory</h2>
        </div>
        <div id="certificate-list" class="timeline"></div>
      </div>
    </section>
  </main>

  <script src="/static/dashboard.js"></script>
</body>
</html>`

const dashboardCSS = `:root {
  --ink: #17201b;
  --muted: #65736b;
  --line: #cad7ce;
  --paper: #f4f7f2;
  --panel: #ffffff;
  --green: #208a5b;
  --yellow: #a67a00;
  --red: #b93632;
  --grey: #6b7280;
  --blue: #245f91;
  --amber-bg: #fdf3d2;
  --red-bg: #fde4e3;
  --green-bg: #e9f6ef;
  --info-bg: #eaf3fc;
}

* { box-sizing: border-box; }

body {
  margin: 0;
  color: var(--ink);
  background:
    linear-gradient(135deg, rgba(36, 95, 145, 0.10), transparent 34%),
    linear-gradient(315deg, rgba(32, 138, 91, 0.12), transparent 42%),
    var(--paper);
  font-family: "Aptos", "Segoe UI", sans-serif;
}

body.console-degraded .topbar {
  border-bottom-color: #d7a3a1;
  background: rgba(253, 228, 227, 0.9);
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 28px 36px 20px;
  border-bottom: 1px solid var(--line);
  background: rgba(244, 247, 242, 0.88);
}

.top-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--muted);
  font-size: 14px;
}

button {
  border: 1px solid var(--ink);
  background: var(--ink);
  color: white;
  min-height: 36px;
  padding: 0 14px;
  border-radius: 6px;
  font-weight: 700;
  cursor: pointer;
}

button:disabled {
  opacity: 0.55;
  cursor: wait;
}

.eyebrow,
.label {
  margin: 0 0 5px;
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
}

h1,
h2 {
  margin: 0;
}

h1 { font-size: 30px; }
h2 { font-size: 18px; }

.alert-strip {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 36px;
  border-bottom: 1px solid var(--line);
  font-size: 14px;
}

.alert-strip strong {
  min-width: 180px;
}

.alert-critical {
  background: var(--red-bg);
  color: #82221f;
}

.alert-warning {
  background: var(--amber-bg);
  color: #734f00;
}

.alert-info {
  background: var(--info-bg);
  color: #16456f;
}

.alert-healthy {
  background: var(--green-bg);
  color: #18683f;
}

.api-banner {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 36px;
  border-bottom: 1px solid #d7a3a1;
  background: var(--red-bg);
  color: #82221f;
  font-size: 13px;
}

.api-banner-hidden {
  display: none;
}

.stale-badge {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
}

.stale-fresh {
  color: #18683f;
  background: var(--green-bg);
}

.stale-warning {
  color: #735200;
  background: var(--amber-bg);
}

.stale-critical {
  color: #842926;
  background: var(--red-bg);
}

.shell {
  width: min(1440px, 100%);
  margin: 0 auto;
  padding: 24px 36px 40px;
}

.status-band {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 1px;
  border: 1px solid var(--line);
  background: var(--line);
}

.status-band > div,
.panel {
  background: rgba(255, 255, 255, 0.92);
}

.status-band > div {
  padding: 16px;
  min-width: 0;
}

.status-band strong {
  display: block;
  overflow-wrap: anywhere;
  font-size: 17px;
}

.state-pill,
.status-pill {
  display: inline-flex;
  align-items: center;
  min-height: 26px;
  padding: 3px 9px;
  border-radius: 999px;
  color: white;
  background: var(--grey);
  font-size: 13px;
}

.source-pill {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 2px 8px;
  border-radius: 999px;
  color: #fff;
  font-size: 12px;
  font-weight: 700;
}

.source-file { background: var(--green); }
.source-override { background: var(--blue); }
.source-mixed { background: var(--yellow); }

.cabinet-warning {
  color: #7b2d2a;
  background: var(--red-bg);
  padding: 2px 6px;
  border-radius: 5px;
}

.state-healthy,
.state-ready,
.state-ready_lab,
.status-green { background: var(--green); }

.state-warning,
.state-expiring_soon,
.status-yellow { background: var(--yellow); }

.state-emergency_active,
.state-missing,
.state-invalid,
.status-red { background: var(--red); }

.status-grey,
.state-degraded,
.state-unknown { background: var(--grey); }

.grid {
  display: grid;
  gap: 18px;
  margin-top: 18px;
}

.grid.two {
  grid-template-columns: minmax(0, 1.45fr) minmax(340px, 0.55fr);
}

.panel {
  border: 1px solid var(--line);
  border-radius: 8px;
  overflow: hidden;
}

.panel-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px;
  border-bottom: 1px solid var(--line);
  color: var(--muted);
}

.panel-head-stack {
  flex-direction: column;
}

.panel-title-row,
.toolbar-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.panel-head h2 { color: var(--ink); }

.table-wrap {
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
  min-width: 760px;
}

th,
td {
  padding: 13px 18px;
  border-bottom: 1px solid #e5ece7;
  text-align: left;
  vertical-align: top;
  font-size: 14px;
}

th {
  color: var(--muted);
  font-size: 12px;
  text-transform: uppercase;
}

.sort-button {
  appearance: none;
  border: 0;
  background: transparent;
  color: inherit;
  min-height: auto;
  padding: 0;
  border-radius: 0;
  font-size: inherit;
  font-weight: 700;
  text-transform: uppercase;
}

.sort-button.is-active {
  color: var(--blue);
}

.filter-tabs {
  display: inline-flex;
  border: 1px solid var(--line);
  border-radius: 7px;
  overflow: hidden;
}

.filter-tab {
  border: 0;
  min-height: 30px;
  background: #f2f6f3;
  color: var(--muted);
  border-radius: 0;
  font-size: 12px;
  padding: 0 10px;
}

.filter-tab + .filter-tab {
  border-left: 1px solid var(--line);
}

.filter-tab.is-active {
  background: var(--ink);
  color: #fff;
}

.timeline {
  display: grid;
  gap: 1px;
  background: #e5ece7;
}

.item {
  padding: 14px 18px;
  background: var(--panel);
}

.item strong {
  display: block;
  margin-bottom: 4px;
}

.item span,
.minor,
.muted-text {
  color: var(--muted);
  font-size: 12px;
}

.kv-list {
  display: grid;
  gap: 1px;
  margin: 0;
  background: #e5ece7;
}

.kv-list div {
  display: grid;
  grid-template-columns: 140px minmax(0, 1fr);
  gap: 14px;
  padding: 13px 18px;
  background: var(--panel);
}

.kv-list dt {
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
}

.kv-list dd {
  margin: 0;
  overflow-wrap: anywhere;
  font-size: 14px;
}

.setup-form {
  padding: 18px;
  background: var(--panel);
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.form-grid label {
  display: grid;
  gap: 6px;
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
}

.form-grid input {
  width: 100%;
  min-height: 38px;
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 8px 10px;
  color: var(--ink);
  background: #fbfdfb;
  font: inherit;
  font-size: 14px;
  text-transform: none;
}

.form-grid select {
  width: 100%;
  min-height: 38px;
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 8px 10px;
  color: var(--ink);
  background: #fbfdfb;
  font: inherit;
  font-size: 14px;
  text-transform: none;
}

.form-grid input:focus {
  outline: 2px solid rgba(36, 95, 145, 0.24);
  border-color: var(--blue);
}

.form-grid select:focus {
  outline: 2px solid rgba(36, 95, 145, 0.24);
  border-color: var(--blue);
}

.cert-manager-form {
  display: grid;
  gap: 12px;
}

.cert-form-grid {
  grid-template-columns: minmax(220px, 0.45fr) minmax(260px, 0.55fr);
}

.cert-role-summary {
  padding: 10px 12px;
  border: 1px solid var(--line);
  background: #f8fbf8;
  color: var(--muted);
  font-size: 13px;
}

.cert-manager-details {
  margin-top: 2px;
}

.cert-manager-detail {
  padding: 0 18px 18px;
  overflow-wrap: anywhere;
}

.cert-textarea-label {
  display: grid;
  gap: 6px;
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
}

.cert-textarea-label textarea {
  width: 100%;
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 8px 10px;
  color: var(--ink);
  background: #fbfdfb;
  font-family: "IBM Plex Mono", "Consolas", monospace;
  font-size: 12px;
  line-height: 1.35;
  resize: vertical;
}

.cert-textarea-label textarea:focus {
  outline: 2px solid rgba(36, 95, 145, 0.24);
  border-color: var(--blue);
}

.cert-key-hidden {
  display: none;
}

.cert-actions {
  margin-top: 0;
}

.cert-exports {
  padding: 0 18px 18px;
}

.cert-export-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.cert-export-buttons button {
  min-height: 34px;
}

.token-help {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 12px;
  padding: 10px 12px;
  border: 1px solid var(--line);
  background: #f8fbf8;
  color: var(--muted);
  font-size: 13px;
}

.token-help code {
  color: var(--ink);
  font-weight: 800;
}

.trusted-bypass-hidden {
  display: none !important;
}

.setup-details {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(260px, 0.45fr);
  gap: 1px;
  margin-top: 16px;
  border: 1px solid var(--line);
  background: var(--line);
}

.setup-details > div {
  min-width: 0;
  padding: 13px 14px;
  background: #f8fbf8;
}

.setup-details strong {
  display: block;
  overflow-wrap: anywhere;
  font-size: 14px;
}

.validation-list {
  display: grid;
  gap: 6px;
  margin-top: 12px;
  color: #7b2d2a;
  font-size: 13px;
}

.validation-list:empty {
  display: none;
}

.validation-item {
  border-left: 3px solid var(--red);
  padding: 6px 8px;
  background: var(--red-bg);
}

.setup-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 16px;
}

.secondary-button {
  background: #fff;
  color: var(--ink);
}

button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1px;
  background: #e5ece7;
}

.summary-cell {
  padding: 16px 18px;
  background: var(--panel);
  border-left: 3px solid transparent;
}

.summary-cell strong {
  display: block;
  font-size: 24px;
}

.summary-cell span {
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
}

.summary-cell p {
  margin: 6px 0 0;
  color: var(--muted);
  font-size: 12px;
}

.summary-blocking { border-left-color: var(--red); }
.summary-warning { border-left-color: var(--yellow); }
.summary-lab { border-left-color: var(--blue); }
.summary-healthy { border-left-color: var(--green); }

.cert-item.cert-blocking { border-left: 3px solid var(--red); }
.cert-item.cert-warning { border-left: 3px solid var(--yellow); }
.cert-item.cert-lab { border-left: 3px solid var(--blue); }
.cert-item.cert-healthy { border-left: 3px solid var(--green); }

.cert-impact {
  display: inline-block;
  margin-top: 8px;
  padding: 2px 6px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.02em;
}

.cert-impact-blocking {
  background: var(--red-bg);
  color: #7b2d2a;
}

.cert-impact-warning {
  background: #fbeecd;
  color: #7b5b0e;
}

.cert-impact-lab {
  background: #e8f1fa;
  color: #1d507b;
}

.cert-impact-healthy {
  background: var(--green-bg);
  color: #1e6c47;
}

.first-cabinet-session-panel .kv-list {
  margin-bottom: 0;
}

.first-cabinet-session-blockers-wrap {
  border-top: 1px solid var(--line);
  padding: 12px 16px 16px;
  background: #f8fbf8;
}

.first-cabinet-session-blockers {
  display: grid;
  gap: 6px;
}

.first-cabinet-session-blocker {
  border-left: 3px solid var(--red);
  padding: 7px 9px;
  background: var(--red-bg);
  color: #7b2d2a;
  font-size: 13px;
}

.first-cabinet-session-blockers-empty {
  border-left-color: var(--green);
  background: var(--green-bg);
  color: #1e6c47;
}

.evidence-capture-panel .kv-list {
  margin-bottom: 0;
}

.evidence-notes-label {
  padding: 0 16px 16px;
}

.evidence-actions {
  padding: 0 16px 16px;
}

.session-evidence-history-item {
  display: grid;
  gap: 10px;
}

.session-evidence-history-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.session-evidence-history-actions button {
  min-height: 30px;
  padding: 0 10px;
  font-size: 12px;
}

.session-evidence-selected-detail {
  display: grid;
  gap: 10px;
}

.session-evidence-selected-detail .kv-inline {
  display: grid;
  gap: 4px;
}

.empty {
  padding: 18px;
  color: var(--muted);
  background: var(--panel);
}

@media (max-width: 920px) {
  .topbar {
    align-items: flex-start;
    flex-direction: column;
    padding: 22px;
  }

  .alert-strip {
    padding: 12px 22px;
    flex-direction: column;
    gap: 6px;
  }

  .shell {
    padding: 18px;
  }

  .status-band,
  .grid.two {
    grid-template-columns: 1fr;
  }

  .top-actions {
    width: 100%;
    flex-wrap: wrap;
  }

  .toolbar-row {
    align-items: flex-start;
    flex-direction: column;
  }

  .form-grid,
  .setup-details {
    grid-template-columns: 1fr;
  }
}`

const dashboardJS = `const endpoints = {
  status: "/api/status",
  readyz: "/readyz",
  incidents: "/api/incidents?limit=6",
  egmHistory: "/api/egms/history?limit=8",
  stateHistory: "/api/state-history?limit=8",
  certificates: "/api/certificates",
  sessionEvidence: "/api/session-evidence?limit=8",
  cabinetProfile: "/api/cabinet-profile",
  cabinetPreflight: "/api/cabinet-preflight",
  certificateImport: "/api/certificates/import",
  certificateExport: "/api/certificates/export"
};

const $ = (id) => document.getElementById(id);
const unhealthyStates = new Set(["RED", "GREY"]);
const healthyStates = new Set(["GREEN", "YELLOW"]);

const clientState = {
  lastGoodStatus: null,
  lastGoodAt: 0,
  lastError: "",
  inFlight: false,
  pollIntervalMs: 3000,
  backoffStep: 0,
  timerId: null,
  displaySnapshot: null,
  egmSortKey: "egm_id",
  egmSortDir: "asc",
  egmFilter: "all",
  certSelectedRole: "g2s_ca_cert",
  selectedSessionEvidenceID: 0
};

function currentRuntime() {
  return clientState.displaySnapshot?.status?.runtime || clientState.lastGoodStatus?.status?.runtime || {};
}

function setupActionsRequireToken() {
  return currentRuntime().api_mutation_auth_required === true;
}

function trustedMutationBypassActive() {
  return currentRuntime().trusted_mutation_bypass_active === true;
}

function mutationTokenRequired() {
  return currentRuntime().api_mutation_auth_required === true;
}

function emptySnapshot() {
  return {
    status: null,
    readyz: null,
    incidents: [],
    egmHistory: [],
    stateHistory: [],
    certificates: [],
    sessionEvidence: [],
    cabinetProfile: null,
    cabinetPreflight: null
  };
}

function copySnapshot(snapshot) {
  if (!snapshot) return emptySnapshot();
  return {
    status: snapshot.status,
    readyz: snapshot.readyz ? {
      ok: snapshot.readyz.ok,
      statusCode: snapshot.readyz.statusCode,
      overall: snapshot.readyz.overall,
      issues: Array.isArray(snapshot.readyz.issues) ? snapshot.readyz.issues.slice() : []
    } : null,
    incidents: Array.isArray(snapshot.incidents) ? snapshot.incidents.slice() : [],
    egmHistory: Array.isArray(snapshot.egmHistory) ? snapshot.egmHistory.slice() : [],
    stateHistory: Array.isArray(snapshot.stateHistory) ? snapshot.stateHistory.slice() : [],
    certificates: Array.isArray(snapshot.certificates) ? snapshot.certificates.slice() : [],
    sessionEvidence: Array.isArray(snapshot.sessionEvidence) ? snapshot.sessionEvidence.slice() : [],
    cabinetProfile: snapshot.cabinetProfile || null,
    cabinetPreflight: snapshot.cabinetPreflight || null
  };
}

function stateClass(value, prefix) {
  const normalized = String(value || "unknown").toLowerCase().replace(/[^a-z0-9]+/g, "_");
  return prefix + "-" + normalized;
}

function setStatePill(el, value) {
  el.textContent = value || "-";
  el.className = "state-pill " + stateClass(value, "state");
}

function statusPill(value) {
  return "<span class=\"status-pill " + stateClass(value, "status") + "\">" + (value || "-") + "</span>";
}

function escapeHTML(value) {
  return String(value || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;");
}

function fmtTime(value) {
  if (!value || String(value).startsWith("0001-")) return "-";
  return new Date(value).toLocaleString();
}

function fmtAge(value) {
  if (!value || String(value).startsWith("0001-")) return "never";
  const seconds = Math.max(0, Math.floor((Date.now() - new Date(value).getTime()) / 1000));
  if (seconds < 60) return seconds + "s ago";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return minutes + "m ago";
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return hours + "h " + (minutes % 60) + "m ago";
  const days = Math.floor(hours / 24);
  return days + "d " + (hours % 24) + "h ago";
}

function parseCertState(status) {
  const text = String(status || "UNKNOWN");
  const idx = text.indexOf(":");
  return idx >= 0 ? text.slice(0, idx) : text;
}

function certRequired(runtime, role) {
  const tlsRequired = !!runtime?.tls_required;
  const clientCertRequired = !!runtime?.client_cert_required;
  switch (role) {
    case "web_server_cert":
      return tlsRequired;
    case "g2s_ca_cert":
    case "g2s_client_cert":
      return clientCertRequired;
    default:
      return false;
  }
}

function certSeverity(item, runtime) {
  const state = parseCertState(item.status);
  if (state === "OK" || state === "VALID") return "healthy";
  if (state === "EXPIRING_SOON") return "warning";
  if (state === "NOT_CONFIGURED") return "lab";
  const required = certRequired(runtime, item.role);
  if (state === "MISSING" || state === "INVALID" || state === "UNKNOWN" || state === "EXPIRED" || state === "NOT_YET_VALID") {
    return required ? "blocking" : "lab";
  }
  return required ? "blocking" : "warning";
}

function certExplanation(item, runtime) {
  const state = parseCertState(item.status);
  const required = certRequired(runtime, item.role);
  if (state === "OK" || state === "VALID") return "Loaded and valid for current runtime.";
  if (state === "EXPIRING_SOON") return "Rotation needed soon to avoid readiness degradation.";
  if (state === "NOT_CONFIGURED") return "Expected in lab mode when TLS and mTLS are not required.";
  if (state === "MISSING") return required ? "Blocking: required certificate file is missing." : "Missing, but currently not required.";
  if (state === "INVALID") return required ? "Blocking: certificate failed validation for a required role." : "Invalid, but currently not required.";
  if (state === "EXPIRED") return required ? "Blocking: required certificate is expired." : "Expired, but currently not required.";
  if (state === "NOT_YET_VALID") return required ? "Blocking: required certificate is not yet valid." : "Not yet valid, but currently not required.";
  return required ? "Blocking: certificate state could not be validated." : "Unknown certificate state.";
}

function renderCertificateSummary(summary, certificates, runtime) {
  const counts = { healthy: 0, warning: 0, lab: 0, blocking: 0 };
  const records = Array.isArray(certificates) ? certificates : [];
  if (records.length > 0) {
    records.forEach((item) => {
      const severity = certSeverity(item, runtime || {});
      counts[severity] = (counts[severity] || 0) + 1;
    });
  } else {
    counts.healthy = (summary.OK || 0) + (summary.VALID || 0);
    counts.warning = summary.EXPIRING_SOON || 0;
    counts.lab = summary.NOT_CONFIGURED || 0;
    counts.blocking = (summary.MISSING || 0) + (summary.INVALID || 0) + (summary.UNKNOWN || 0) + (summary.EXPIRED || 0) + (summary.NOT_YET_VALID || 0);
  }
  $("certificate-summary").innerHTML = [
    "<div class=\"summary-cell summary-blocking\"><strong>" + counts.blocking + "</strong><span>Blocking</span><p>Required cert failures in current runtime.</p></div>",
    "<div class=\"summary-cell summary-warning\"><strong>" + counts.warning + "</strong><span>Expiring Soon</span><p>Plan rotation before expiry.</p></div>",
    "<div class=\"summary-cell summary-lab\"><strong>" + counts.lab + "</strong><span>Lab Optional</span><p>Not required in current lab mode.</p></div>",
    "<div class=\"summary-cell summary-healthy\"><strong>" + counts.healthy + "</strong><span>Healthy</span><p>Ready for configured runtime.</p></div>"
  ].join("");
}

function certImpactLabel(severity) {
  if (severity === "blocking") return "Blocking for runtime";
  if (severity === "lab") return "Lab optional";
  if (severity === "warning") return "Needs attention";
  return "Healthy";
}

function summarizeSessionCertCounts(certificates, runtime) {
  const counts = { blocking: 0, labOptional: 0 };
  const records = Array.isArray(certificates) ? certificates : [];
  records.forEach((item) => {
    const severity = certSeverity(item, runtime || {});
    if (severity === "blocking") counts.blocking++;
    if (severity === "lab") counts.labOptional++;
  });
  return counts;
}

function appendUniqueBlocker(blockers, message) {
  if (!message) return;
  if (blockers.indexOf(message) < 0) {
    blockers.push(message);
  }
}

function appendFriendlyPreflightBlockers(preflight, blockers) {
  const checks = Array.isArray(preflight?.checks) ? preflight.checks : [];
  let mappedAny = false;
  checks.forEach((check) => {
    if (!check || check.result !== "FAIL") return;
    switch (check.id) {
      case "cabinet_profile":
        appendUniqueBlocker(blockers, "Cabinet profile is incomplete");
        mappedAny = true;
        break;
      case "certificate_mode_requirements":
        appendUniqueBlocker(blockers, "Required certificate is missing");
        mappedAny = true;
        break;
      case "service_readiness":
        appendUniqueBlocker(blockers, "Readiness is degraded");
        mappedAny = true;
        break;
      case "profile_source":
        appendUniqueBlocker(blockers, "Cabinet profile source should be explicit");
        mappedAny = true;
        break;
      case "certificate_san_wire_identity":
        appendUniqueBlocker(blockers, "Certificate SAN does not match wire identity");
        mappedAny = true;
        break;
      default:
        break;
    }
  });
  if (mappedAny) return;
  const raw = Array.isArray(preflight?.blockers) ? preflight.blockers : [];
  raw.forEach((item) => appendUniqueBlocker(blockers, String(item || "").trim()));
}

function buildFirstCabinetSessionState(snapshot) {
  const status = snapshot?.status || {};
  const runtime = status.runtime || {};
  const readyz = snapshot?.readyz || null;
  const preflight = snapshot?.cabinetPreflight || null;
  const profilePayload = snapshot?.cabinetProfile || null;
  const profile = profilePayload?.effective || status.cabinet_profile || {};
  const profileSource = profilePayload?.profile_source || status.profile_source || "-";
  const certCounts = summarizeSessionCertCounts(snapshot?.certificates, runtime);
  const firstEGMIDs = Array.isArray(profile.first_test_egm_ids) ? profile.first_test_egm_ids : [];
  const trustedBypass = runtime.trusted_mutation_bypass_active === true;
  const authRequired = runtime.api_mutation_auth_required === true;
  const authDisabled = runtime.api_mutation_auth_required === false;
  const authState = trustedBypass ? "TRUSTED_PRIVATE_NETWORK" : (authRequired ? "REQUIRED" : (authDisabled ? "DISABLED" : "UNKNOWN"));
  const readyzState = readyz ? (readyz.overall || (readyz.ok ? "READY" : "DEGRADED")) : "UNAVAILABLE";
  const preflightState = preflight ? (preflight.overall || "UNKNOWN") : "UNAVAILABLE";
  const lastCheckedValue = preflight?.timestamp || (clientState.lastGoodAt ? new Date(clientState.lastGoodAt).toISOString() : "");
  const blockers = [];

  if (!readyz || readyz.ok === false || readyzState === "DEGRADED") {
    appendUniqueBlocker(blockers, "Readiness is degraded");
  }
  if (!preflight) {
    appendUniqueBlocker(blockers, "Preflight API unavailable");
  } else if (preflight.overall !== "PASS") {
    appendFriendlyPreflightBlockers(preflight, blockers);
  }
  if (!profile.wire_host_url || !profile.host_id || firstEGMIDs.length === 0) {
    appendUniqueBlocker(blockers, "Cabinet profile is incomplete");
  }
  if (certCounts.blocking > 0) {
    appendUniqueBlocker(blockers, "Required certificate is missing");
  }
  if (authRequired && !getSetupToken() && !getCertToken()) {
    appendUniqueBlocker(blockers, "API token is required for protected setup actions");
  }

  const readyForSession = blockers.length === 0 && readyzState !== "UNAVAILABLE" && preflightState === "PASS";
  const overallState = readyForSession ? "LAB_READY" : "BLOCKED";
  return {
    overallState: overallState,
    readyForSession: readyForSession,
    message: readyForSession ? "Ready for first cabinet lab session" : "Resolve blockers before first cabinet runbook session.",
    lastCheckedValue: lastCheckedValue,
    readyzState: readyzState,
    preflightState: preflightState,
    profileSource: profileSource || "-",
    profile: profile,
    firstEGMIDs: firstEGMIDs,
    certCounts: certCounts,
    authState: authState,
    blockers: blockers
  };
}

function renderFirstCabinetSession(snapshot) {
  const session = buildFirstCabinetSessionState(snapshot);
  const stateBadge = $("first-cabinet-session-state");
  stateBadge.textContent = session.overallState;
  stateBadge.className = "source-pill " + (session.readyForSession ? "source-file" : "source-mixed");
  $("first-cabinet-session-message").textContent = session.message;

  $("first-cabinet-overall").textContent = session.overallState;
  $("first-cabinet-last-checked").textContent = fmtTime(session.lastCheckedValue);
  $("first-cabinet-readyz").textContent = session.readyzState;
  $("first-cabinet-preflight").textContent = session.preflightState;
  $("first-cabinet-profile-source").textContent = session.profileSource;
  $("first-cabinet-wire-host-url").textContent = session.profile.wire_host_url || "-";
  $("first-cabinet-host-id").textContent = session.profile.host_id || "-";
  $("first-cabinet-egm-ids").textContent = session.firstEGMIDs.length ? session.firstEGMIDs.join(", ") : "-";
  $("first-cabinet-cert-blocking").textContent = String(session.certCounts.blocking);
  $("first-cabinet-cert-lab-optional").textContent = String(session.certCounts.labOptional);
  $("first-cabinet-auth-state").textContent = session.authState;

  const blockerList = $("first-cabinet-session-blockers");
  if (session.blockers.length === 0) {
    blockerList.innerHTML = "<div class=\"first-cabinet-session-blocker first-cabinet-session-blockers-empty\">Ready for first cabinet lab session</div>";
  } else {
    blockerList.innerHTML = session.blockers.map((item) => "<div class=\"first-cabinet-session-blocker\">" + escapeHTML(item) + "</div>").join("");
  }
}

function buildSessionEvidence(snapshot) {
  const session = buildFirstCabinetSessionState(snapshot);
  const status = snapshot?.status || {};
  const runtime = status.runtime || {};
  const readiness = status.readiness || {};
  const profilePayload = snapshot?.cabinetProfile || null;
  const profile = profilePayload?.effective || status.cabinet_profile || {};
  const certificates = Array.isArray(snapshot?.certificates) ? snapshot.certificates : [];
  const incidents = Array.isArray(snapshot?.incidents) ? snapshot.incidents : [];
  const egmHistory = Array.isArray(snapshot?.egmHistory) ? snapshot.egmHistory : [];
  const stateHistory = Array.isArray(snapshot?.stateHistory) ? snapshot.stateHistory : [];
  const notes = $("session-evidence-notes").value.trim();
  return {
    captured_at: new Date().toISOString(),
    session: {
      overall_state: session.overallState,
      ready_for_session: session.readyForSession,
      message: session.message,
      blockers: session.blockers,
      last_checked: session.lastCheckedValue || null,
      readyz_state: session.readyzState,
      preflight_state: session.preflightState,
      certificate_blocking_count: session.certCounts.blocking,
      certificate_lab_optional_count: session.certCounts.labOptional,
      api_auth_state: session.authState
    },
    cabinet_profile: {
      source: session.profileSource,
      profile_differs_from_file: status.profile_differs_from_file === true,
      wire_host_url: profile.wire_host_url || "",
      host_id: profile.host_id || "",
      first_test_egm_ids: session.firstEGMIDs,
      listener_dns_name: profile.listener_dns_name || "",
      listener_ip: profile.listener_ip || ""
    },
    runtime: {
      bind_address: runtime.bind_address || "",
      g2s_host_url: runtime.g2s_host_url || "",
      tls_required: runtime.tls_required === true,
      client_cert_required: runtime.client_cert_required === true,
      trusted_mutation_bypass_active: runtime.trusted_mutation_bypass_active === true
    },
    readiness: {
      overall: readiness.overall || "",
      issues: Array.isArray(readiness.issues) ? readiness.issues : [],
      warnings: Array.isArray(readiness.warnings) ? readiness.warnings : []
    },
    egm_snapshot_count: Array.isArray(status.egms) ? status.egms.length : 0,
    incidents: incidents,
    egm_history: egmHistory,
    state_history: stateHistory,
    certificates: certificates,
    operator_notes: notes
  };
}

function buildSessionEvidenceMarkdown(evidence) {
  const lines = [
    "# Session Evidence Capture",
    "",
    "- Captured at: " + (evidence.captured_at || "-"),
    "- Session state: " + (evidence.session.overall_state || "-"),
    "- Ready for session: " + String(evidence.session.ready_for_session === true),
    "- Readyz state: " + (evidence.session.readyz_state || "-"),
    "- Preflight state: " + (evidence.session.preflight_state || "-"),
    "- API auth state: " + (evidence.session.api_auth_state || "-"),
    "- Cabinet profile source: " + (evidence.cabinet_profile.source || "-"),
    "- Wire host URL: " + (evidence.cabinet_profile.wire_host_url || "-"),
    "- Host ID: " + (evidence.cabinet_profile.host_id || "-"),
    "- First test EGM IDs: " + ((evidence.cabinet_profile.first_test_egm_ids || []).join(", ") || "-"),
    "- Certificate blocking count: " + String(evidence.session.certificate_blocking_count || 0),
    "- Certificate lab optional count: " + String(evidence.session.certificate_lab_optional_count || 0),
    "- EGM snapshot count: " + String(evidence.egm_snapshot_count || 0),
    "- Incident rows captured: " + String((evidence.incidents || []).length),
    "- State history rows captured: " + String((evidence.state_history || []).length),
    ""
  ];
  lines.push("## Blockers", "");
  if (Array.isArray(evidence.session.blockers) && evidence.session.blockers.length) {
    evidence.session.blockers.forEach((item) => lines.push("- " + item));
  } else {
    lines.push("- None");
  }
  lines.push("", "## Readiness Warnings", "");
  if (Array.isArray(evidence.readiness.warnings) && evidence.readiness.warnings.length) {
    evidence.readiness.warnings.forEach((item) => lines.push("- " + item));
  } else {
    lines.push("- None");
  }
  lines.push("", "## Operator Notes", "");
  lines.push(evidence.operator_notes || "None");
  lines.push("", "## JSON Payload", "", "~~~json", JSON.stringify(evidence, null, 2), "~~~");
  return lines.join("\n");
}

function evidenceFilenameBase(evidence) {
  const hostID = String(evidence?.cabinet_profile?.host_id || "cabinet").replace(/[^A-Za-z0-9._-]+/g, "-");
  const stamp = new Date(evidence?.captured_at || Date.now()).toISOString().replace(/[:]/g, "-");
  return hostID + "-session-evidence-" + stamp;
}

function parseSavedSessionEvidencePayload(record) {
  if (!record || !record.payload_json) return null;
  try {
    return JSON.parse(record.payload_json);
  } catch (_) {
    return null;
  }
}

function findSavedSessionEvidenceRecord(id, snapshot) {
  const records = Array.isArray(snapshot?.sessionEvidence) ? snapshot.sessionEvidence : [];
  for (let i = 0; i < records.length; i++) {
    if (String(records[i]?.id) === String(id)) {
      return records[i];
    }
  }
  return null;
}

function selectedSavedSessionEvidence(snapshot) {
  const records = Array.isArray(snapshot?.sessionEvidence) ? snapshot.sessionEvidence : [];
  if (records.length === 0) {
    clientState.selectedSessionEvidenceID = 0;
    return null;
  }
  const selected = findSavedSessionEvidenceRecord(clientState.selectedSessionEvidenceID, snapshot);
  if (selected) {
    return selected;
  }
  clientState.selectedSessionEvidenceID = records[0].id || 0;
  return records[0];
}

function renderSelectedSavedSessionEvidence(record) {
  if (!record) {
    renderItems("session-evidence-selected", [], "Select a saved capture to inspect it here.", () => "");
    return;
  }
  const evidence = parseSavedSessionEvidencePayload(record);
  if (!evidence) {
    renderItems("session-evidence-selected", [record], "", () =>
      "<div class=\"item session-evidence-selected-detail\"><strong>Saved capture #" + escapeHTML(record.id) + "</strong><span>Payload could not be parsed.</span></div>"
    );
    return;
  }
  const blockers = Array.isArray(evidence?.session?.blockers) ? evidence.session.blockers : [];
  const warnings = Array.isArray(evidence?.readiness?.warnings) ? evidence.readiness.warnings : [];
  renderItems("session-evidence-selected", [record], "", () =>
    "<div class=\"item session-evidence-selected-detail\">" +
      "<strong>" + escapeHTML(evidence?.session?.overall_state || "-") + " | " + escapeHTML(evidence?.cabinet_profile?.host_id || "-") + "</strong>" +
      "<span>" + escapeHTML(fmtTime(evidence?.captured_at || record.created_at)) + " | " + escapeHTML(evidence?.cabinet_profile?.wire_host_url || "-") + "</span>" +
      "<div class=\"kv-inline\"><span>Readyz: " + escapeHTML(evidence?.session?.readyz_state || "-") + " | Preflight: " + escapeHTML(evidence?.session?.preflight_state || "-") + "</span></div>" +
      "<div class=\"kv-inline\"><span>Blockers: " + escapeHTML(String(blockers.length)) + " | Warnings: " + escapeHTML(String(warnings.length)) + "</span></div>" +
      "<div class=\"kv-inline\"><span>Notes: " + escapeHTML(record.operator_notes || evidence?.operator_notes || "None") + "</span></div>" +
    "</div>"
  );
}

function viewSavedSessionEvidence(id) {
  clientState.selectedSessionEvidenceID = Number(id) || 0;
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  const record = selectedSavedSessionEvidence(snapshot);
  renderSelectedSavedSessionEvidence(record);
  if (record) {
    $("session-evidence-state").textContent = "saved";
    $("session-evidence-state").className = "source-pill source-file";
    $("session-evidence-message").textContent = "Viewing saved capture #" + record.id + ".";
  }
}

function exportSavedSessionEvidenceJSON(id) {
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  const record = findSavedSessionEvidenceRecord(id, snapshot);
  const evidence = parseSavedSessionEvidencePayload(record);
  if (!record || !evidence) {
    $("session-evidence-state").textContent = "blocked";
    $("session-evidence-state").className = "source-pill source-mixed";
    $("session-evidence-message").textContent = "Saved evidence payload is unavailable.";
    return;
  }
  clientState.selectedSessionEvidenceID = record.id || 0;
  downloadTextMaterial(evidenceFilenameBase(evidence) + ".json", JSON.stringify(evidence, null, 2));
  renderSelectedSavedSessionEvidence(record);
  $("session-evidence-state").textContent = "saved";
  $("session-evidence-state").className = "source-pill source-file";
  $("session-evidence-message").textContent = "Saved JSON evidence downloaded.";
}

function exportSavedSessionEvidenceMarkdown(id) {
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  const record = findSavedSessionEvidenceRecord(id, snapshot);
  const evidence = parseSavedSessionEvidencePayload(record);
  if (!record || !evidence) {
    $("session-evidence-state").textContent = "blocked";
    $("session-evidence-state").className = "source-pill source-mixed";
    $("session-evidence-message").textContent = "Saved evidence payload is unavailable.";
    return;
  }
  clientState.selectedSessionEvidenceID = record.id || 0;
  downloadTextMaterial(evidenceFilenameBase(evidence) + ".md", buildSessionEvidenceMarkdown(evidence));
  renderSelectedSavedSessionEvidence(record);
  $("session-evidence-state").textContent = "saved";
  $("session-evidence-state").className = "source-pill source-file";
  $("session-evidence-message").textContent = "Saved Markdown evidence downloaded.";
}

function exportAllSavedSessionEvidence() {
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  const records = Array.isArray(snapshot?.sessionEvidence) ? snapshot.sessionEvidence : [];
  if (records.length === 0) {
    $("session-evidence-state").textContent = "blocked";
    $("session-evidence-state").className = "source-pill source-mixed";
    $("session-evidence-message").textContent = "No saved evidence captures are available to export.";
    return;
  }
  const payload = records.map((record) => ({
    record: {
      id: record.id,
      created_at: record.created_at,
      overall_state: record.overall_state,
      readyz_state: record.readyz_state,
      preflight_state: record.preflight_state,
      host_id: record.host_id,
      wire_host_url: record.wire_host_url,
      operator_notes: record.operator_notes || ""
    },
    evidence: parseSavedSessionEvidencePayload(record)
  }));
  downloadTextMaterial("saved-session-evidence-history.json", JSON.stringify(payload, null, 2));
  $("session-evidence-state").textContent = "saved";
  $("session-evidence-state").className = "source-pill source-file";
  $("session-evidence-message").textContent = "Saved evidence history downloaded.";
}

async function deleteSavedSessionEvidence(id) {
  const numericID = Number(id) || 0;
  if (!numericID) {
    $("session-evidence-state").textContent = "blocked";
    $("session-evidence-state").className = "source-pill source-mixed";
    $("session-evidence-message").textContent = "Saved evidence record id is invalid.";
    return;
  }
  if (setupActionsRequireToken() && !getSetupToken() && !getCertToken()) {
    $("session-evidence-state").textContent = "blocked";
    $("session-evidence-state").className = "source-pill source-mixed";
    $("session-evidence-message").textContent = "Enter an API token before deleting saved evidence.";
    return;
  }
  try {
    $("session-evidence-state").textContent = "working";
    $("session-evidence-state").className = "source-pill source-override";
    $("session-evidence-message").textContent = "Deleting saved evidence #" + numericID + ".";
    const token = getSetupToken() || getCertToken();
    const headers = {};
    if (token) {
      headers.Authorization = "Bearer " + token;
    }
    const response = await fetch(endpoints.sessionEvidence.replace("?limit=8", "") + "?id=" + encodeURIComponent(String(numericID)), {
      method: "DELETE",
      headers: headers
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      $("session-evidence-state").textContent = "blocked";
      $("session-evidence-state").className = "source-pill source-mixed";
      $("session-evidence-message").textContent = "Delete failed: HTTP " + response.status + (detail ? " " + detail : "");
      return;
    }
    if (String(clientState.selectedSessionEvidenceID) === String(numericID)) {
      clientState.selectedSessionEvidenceID = 0;
    }
    $("session-evidence-state").textContent = "saved";
    $("session-evidence-state").className = "source-pill source-file";
    $("session-evidence-message").textContent = "Saved evidence deleted.";
    schedulePoll(0);
  } catch (err) {
    $("session-evidence-state").textContent = "blocked";
    $("session-evidence-state").className = "source-pill source-mixed";
    $("session-evidence-message").textContent = err && err.message ? err.message : "Delete failed.";
  }
}

function renderSessionEvidence(snapshot) {
  const evidence = buildSessionEvidence(snapshot);
  const selectedRecord = selectedSavedSessionEvidence(snapshot);
  $("session-evidence-overall").textContent = evidence.session.overall_state || "-";
  $("session-evidence-timestamp").textContent = fmtTime(evidence.session.last_checked || evidence.captured_at);
  $("session-evidence-incident-count").textContent = String((evidence.incidents || []).length);
  $("session-evidence-state-count").textContent = String((evidence.state_history || []).length);
  const ready = !!snapshot?.status;
  $("session-evidence-save-button").disabled = !ready || (setupActionsRequireToken() && !getSetupToken() && !getCertToken());
  $("session-evidence-json-button").disabled = !ready;
  $("session-evidence-markdown-button").disabled = !ready;
  $("session-evidence-export-all-button").disabled = !snapshot?.sessionEvidence || snapshot.sessionEvidence.length === 0;
  const badge = $("session-evidence-state");
  badge.textContent = ready ? "ready" : "waiting";
  badge.className = "source-pill " + (ready ? "source-file" : "source-mixed");
  $("session-evidence-message").textContent = ready
    ? "Capture the current cabinet session state as JSON or Markdown."
    : "Waiting for appliance status before capture is available.";
  renderItems("session-evidence-history", snapshot?.sessionEvidence, "No saved captures yet", (item) =>
    "<div class=\"item session-evidence-history-item\"><div><strong>" + escapeHTML(item.overall_state || "-") + "</strong><span>" +
    escapeHTML(item.host_id || "-") + " at " + escapeHTML(fmtTime(item.created_at)) +
    (item.operator_notes ? " - " + escapeHTML(item.operator_notes) : "") + "</span></div>" +
    "<div class=\"session-evidence-history-actions\">" +
      "<button type=\"button\" class=\"secondary-button session-evidence-history-button\" data-evidence-action=\"select\" data-evidence-id=\"" + escapeHTML(item.id) + "\">Select</button>" +
      "<button type=\"button\" class=\"secondary-button session-evidence-history-button\" data-evidence-action=\"delete\" data-evidence-id=\"" + escapeHTML(item.id) + "\">Delete</button>" +
      "<button type=\"button\" class=\"secondary-button session-evidence-history-button\" data-evidence-action=\"json\" data-evidence-id=\"" + escapeHTML(item.id) + "\">Download JSON</button>" +
      "<button type=\"button\" class=\"secondary-button session-evidence-history-button\" data-evidence-action=\"markdown\" data-evidence-id=\"" + escapeHTML(item.id) + "\">Download Markdown</button>" +
    "</div></div>"
  );
  renderSelectedSavedSessionEvidence(selectedRecord);
}

function exportSessionEvidenceJSON() {
  const evidence = buildSessionEvidence(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  downloadTextMaterial(evidenceFilenameBase(evidence) + ".json", JSON.stringify(evidence, null, 2));
  $("session-evidence-state").textContent = "saved";
  $("session-evidence-state").className = "source-pill source-file";
  $("session-evidence-message").textContent = "JSON evidence downloaded.";
}

function exportSessionEvidenceMarkdown() {
  const evidence = buildSessionEvidence(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  downloadTextMaterial(evidenceFilenameBase(evidence) + ".md", buildSessionEvidenceMarkdown(evidence));
  $("session-evidence-state").textContent = "saved";
  $("session-evidence-state").className = "source-pill source-file";
  $("session-evidence-message").textContent = "Markdown evidence downloaded.";
}

async function saveSessionEvidenceToHistory() {
  if (setupActionsRequireToken() && !getSetupToken() && !getCertToken()) {
    $("session-evidence-state").textContent = "blocked";
    $("session-evidence-state").className = "source-pill source-mixed";
    $("session-evidence-message").textContent = "Enter an API token before saving evidence history.";
    return;
  }
  const evidence = buildSessionEvidence(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  try {
    $("session-evidence-state").textContent = "working";
    $("session-evidence-state").className = "source-pill source-override";
    $("session-evidence-message").textContent = "Saving evidence to appliance history.";
    const token = getSetupToken() || getCertToken();
    const headers = { "Content-Type": "application/json" };
    if (token) {
      headers.Authorization = "Bearer " + token;
    }
    const response = await fetch(endpoints.sessionEvidence.replace("?limit=8", ""), {
      method: "POST",
      headers: headers,
      body: JSON.stringify(evidence)
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      $("session-evidence-state").textContent = "blocked";
      $("session-evidence-state").className = "source-pill source-mixed";
      $("session-evidence-message").textContent = "Save failed: HTTP " + response.status + (detail ? " " + detail : "");
      return;
    }
    $("session-evidence-state").textContent = "saved";
    $("session-evidence-state").className = "source-pill source-file";
    $("session-evidence-message").textContent = "Evidence saved to appliance history.";
    schedulePoll(0);
  } catch (err) {
    $("session-evidence-state").textContent = "blocked";
    $("session-evidence-state").className = "source-pill source-mixed";
    $("session-evidence-message").textContent = err && err.message ? err.message : "Save failed.";
  }
}

function renderItems(id, items, emptyText, mapItem) {
  const el = $(id);
  if (!items || items.length === 0) {
    el.innerHTML = "<div class=\"empty\">" + escapeHTML(emptyText) + "</div>";
    return;
  }
  el.innerHTML = items.map(mapItem).join("");
}

function egmSortValue(egm, key) {
  if (key === "status") {
    const weight = { GREEN: 1, YELLOW: 2, GREY: 3, RED: 4 };
    return weight[String(egm.status || "").toUpperCase()] || 99;
  }
  if (key === "last_seen") {
    if (!egm.last_seen || String(egm.last_seen).startsWith("0001-")) return 0;
    return new Date(egm.last_seen).getTime();
  }
  return String(egm.id || "");
}

function compareEGM(a, b) {
  const key = clientState.egmSortKey;
  const dir = clientState.egmSortDir === "asc" ? 1 : -1;
  const av = egmSortValue(a, key);
  const bv = egmSortValue(b, key);
  if (typeof av === "number" && typeof bv === "number") {
    if (av === bv) return String(a.id || "").localeCompare(String(b.id || "")) * dir;
    return (av - bv) * dir;
  }
  return String(av).localeCompare(String(bv)) * dir;
}

function applyEGMFilter(egms) {
  if (clientState.egmFilter === "healthy") {
    return egms.filter((egm) => healthyStates.has(String(egm.status || "").toUpperCase()));
  }
  if (clientState.egmFilter === "unhealthy") {
    return egms.filter((egm) => unhealthyStates.has(String(egm.status || "").toUpperCase()));
  }
  return egms;
}

function updateSortLabels() {
  const headers = document.querySelectorAll(".sort-button");
  headers.forEach((button) => {
    const key = button.dataset.sortKey;
    const active = key === clientState.egmSortKey;
    button.classList.toggle("is-active", active);
    const direction = active ? (clientState.egmSortDir === "asc" ? " (asc)" : " (desc)") : "";
    const base = key === "egm_id" ? "EGM" : key === "last_seen" ? "Last seen" : "Status";
    button.textContent = base + direction;
  });
  $("egm-sort-label").textContent = "Sort: " + (clientState.egmSortKey === "egm_id" ? "EGM ID" : clientState.egmSortKey === "last_seen" ? "Last seen" : "Status") + " " + (clientState.egmSortDir === "asc" ? "(asc)" : "(desc)");
}

function renderEGMTable(status) {
  const all = Array.isArray(status?.egms) ? status.egms.slice() : [];
  const filtered = applyEGMFilter(all);
  const rows = filtered.sort(compareEGM).map((egm) =>
    "<tr>" +
      "<td><strong>" + escapeHTML(egm.id) + "</strong><br><span class=\"minor\">" + escapeHTML((egm.vendor || "") + " " + (egm.cabinet_family || "")).trim() + "</span></td>" +
      "<td>" + statusPill(egm.status) + "</td>" +
      "<td>" + escapeHTML((egm.ip_address || "-") + ":" + (egm.port || "-")) + "</td>" +
      "<td>" + escapeHTML(egm.game_title || "-") + "<br><span class=\"minor\">" + escapeHTML(egm.software_version || "") + "</span></td>" +
      "<td>" + escapeHTML(fmtTime(egm.last_seen)) + "<br><span class=\"minor\">" + escapeHTML(fmtAge(egm.last_seen)) + "</span></td>" +
    "</tr>"
  );
  $("egm-count").textContent = filtered.length + " / " + all.length + " EGMs";
  $("egm-table").innerHTML = rows.length ? rows.join("") : "<tr><td colspan=\"5\">No EGMs match current filter</td></tr>";
  updateSortLabels();
}

function renderCabinetProfile(status) {
  const profile = status?.cabinet_profile || {};
  const source = status?.profile_source || "file";
  const differs = !!status?.profile_differs_from_file;
  const updatedAt = status?.profile_last_updated_at;

  $("cabinet-wire-host-url").textContent = profile.wire_host_url || "-";
  $("cabinet-listener-dns").textContent = profile.listener_dns_name || "-";
  $("cabinet-listener-ip").textContent = profile.listener_ip || "-";
  $("cabinet-required-san-dns").textContent = Array.isArray(profile.required_san_dns) && profile.required_san_dns.length ? profile.required_san_dns.join(", ") : "-";
  $("cabinet-required-san-ips").textContent = Array.isArray(profile.required_san_ips) && profile.required_san_ips.length ? profile.required_san_ips.join(", ") : "-";
  $("cabinet-host-id").textContent = profile.host_id || "-";
  $("cabinet-first-test-egm-ids").textContent = Array.isArray(profile.first_test_egm_ids) && profile.first_test_egm_ids.length ? profile.first_test_egm_ids.join(", ") : "-";
  $("cabinet-profile-updated-at").textContent = updatedAt ? fmtTime(updatedAt) : "file baseline";

  const sourceBadge = $("cabinet-profile-source");
  sourceBadge.textContent = source;
  sourceBadge.className = "source-pill source-" + source;

  const warning = $("cabinet-profile-warning");
  if (source === "override" && differs) {
    warning.innerHTML = "<span class=\"cabinet-warning\">Override differs from file baseline</span>";
  } else if (source === "mixed" && differs) {
    warning.innerHTML = "<span class=\"cabinet-warning\">Mixed profile includes override values</span>";
  } else {
    warning.textContent = "None";
  }
}

function splitList(value) {
  return String(value || "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function joinList(value) {
  return Array.isArray(value) ? value.join(", ") : "";
}

function cabinetProfileFromForm() {
  return {
    wire_host_url: $("setup-wire-host-url").value.trim(),
    listener_dns_name: $("setup-listener-dns").value.trim(),
    listener_ip: $("setup-listener-ip").value.trim(),
    required_san_dns: splitList($("setup-required-san-dns").value),
    required_san_ips: splitList($("setup-required-san-ips").value),
    host_id: $("setup-host-id").value.trim(),
    first_test_egm_ids: splitList($("setup-first-test-egm-ids").value)
  };
}

function fillCabinetSetupForm(profile) {
  $("setup-wire-host-url").value = profile.wire_host_url || "";
  $("setup-listener-dns").value = profile.listener_dns_name || "";
  $("setup-listener-ip").value = profile.listener_ip || "";
  $("setup-required-san-dns").value = joinList(profile.required_san_dns);
  $("setup-required-san-ips").value = joinList(profile.required_san_ips);
  $("setup-host-id").value = profile.host_id || "";
  $("setup-first-test-egm-ids").value = joinList(profile.first_test_egm_ids);
  renderCabinetSetupValidation();
}

function validateCabinetSetupProfile(profile) {
  const problems = [];
  let parsedURL = null;
  if (!profile.wire_host_url) {
    problems.push("Wire Host URL is required.");
  } else {
    try {
      parsedURL = new URL(profile.wire_host_url);
      if (parsedURL.protocol !== "http:" && parsedURL.protocol !== "https:") {
        problems.push("Wire Host URL must use http or https.");
      }
    } catch (_) {
      problems.push("Wire Host URL must be a valid URL.");
    }
  }
  if (!profile.listener_dns_name && !profile.listener_ip) {
    problems.push("Listener DNS or Listener IP is required.");
  }
  if (profile.required_san_dns.length === 0 && profile.required_san_ips.length === 0) {
    problems.push("At least one required SAN DNS or SAN IP value is required.");
  }
  if (!profile.host_id) {
    problems.push("Host ID is required.");
  }
  if (profile.first_test_egm_ids.length === 0) {
    problems.push("At least one first test EGM ID is required.");
  }
  const placeholderText = [
    profile.wire_host_url,
    profile.listener_dns_name,
    profile.listener_ip,
    profile.host_id,
    profile.required_san_dns.join(" "),
    profile.required_san_ips.join(" "),
    profile.first_test_egm_ids.join(" ")
  ].join(" ").toLowerCase();
  if (placeholderText.includes("example") || placeholderText.includes("placeholder")) {
    problems.push("Replace placeholder/example identity values before cabinet use.");
  }
  return { problems, parsedURL };
}

function renderCabinetSetupValidation() {
  const profile = cabinetProfileFromForm();
  const result = validateCabinetSetupProfile(profile);
  const host = result.parsedURL ? result.parsedURL.hostname : "-";
  const tokenPresent = !!getSetupToken();
  const tokenRequired = mutationTokenRequired();
  const sanValues = []
    .concat(profile.required_san_dns.map((item) => "DNS:" + item))
    .concat(profile.required_san_ips.map((item) => "IP:" + item));
  $("setup-san-summary").textContent = "wire host " + host + "; " + (sanValues.length ? sanValues.join(", ") : "no SAN values");
  $("setup-api-token-wrapper").classList.toggle("trusted-bypass-hidden", !tokenRequired);
  $("setup-token-controls").classList.toggle("trusted-bypass-hidden", !tokenRequired);
  $("setup-api-token-label").textContent = tokenRequired ? "API Token Required for Save/Clear" : "API Token Optional for Save/Clear";
  $("setup-token-help-text").textContent = tokenRequired
    ? "Enter the appliance API token to save or clear cabinet setup overrides."
    : "Trusted private network bypass is active for this browser; token is optional for setup actions.";
  $("setup-validation-summary").textContent = result.problems.length ? result.problems.length + " issue(s)" : (tokenPresent || !tokenRequired) ? "Ready to save" : "Token required to save";
  $("setup-validation-list").innerHTML = result.problems.map((item) => "<div class=\"validation-item\">" + escapeHTML(item) + "</div>").join("");
  $("setup-save-button").disabled = result.problems.length > 0 || (tokenRequired && !tokenPresent);
  $("setup-reset-button").disabled = tokenRequired && !tokenPresent;
  $("setup-copy-token-button").disabled = !tokenPresent;
  return result;
}

function getSetupToken() {
  return $("setup-api-token").value.trim();
}

function setupAuthHeaders() {
  const token = getSetupToken();
  const headers = { "Content-Type": "application/json" };
  if (token) {
    headers.Authorization = "Bearer " + token;
  }
  return headers;
}

function setSetupState(level, message) {
  const badge = $("cabinet-setup-state");
  badge.textContent = level;
  badge.className = "source-pill " + (level === "saved" || level === "ready" ? "source-file" : level === "working" ? "source-override" : "source-mixed");
  $("cabinet-setup-message").textContent = message;
}

async function copySetupTokenToClipboard() {
  const token = getSetupToken();
  if (!token) {
    setSetupState("blocked", "Enter an API token before copying.");
    return;
  }
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(token);
    } else {
      const input = $("setup-api-token");
      input.focus();
      input.select();
      document.execCommand("copy");
      input.setSelectionRange(input.value.length, input.value.length);
    }
    setSetupState("ready", "API token copied to clipboard.");
  } catch (err) {
    setSetupState("blocked", err && err.message ? "Copy failed: " + err.message : "Copy failed.");
  }
}

async function reloadCabinetProfileForm() {
  setSetupState("working", "Reloading cabinet profile.");
  const profile = await fetchJSON(endpoints.cabinetProfile);
  clientState.displaySnapshot = clientState.displaySnapshot || emptySnapshot();
  clientState.displaySnapshot.cabinetProfile = profile;
  fillCabinetSetupForm(profile.effective || {});
  setSetupState("ready", "Current values loaded from the appliance.");
  return profile;
}

async function saveCabinetProfileOverride(event) {
  event.preventDefault();
  const validation = renderCabinetSetupValidation();
  if (setupActionsRequireToken() && !getSetupToken()) {
    setSetupState("blocked", "Enter an API token before saving.");
    return;
  }
  if (validation.problems.length) {
    setSetupState("blocked", "Resolve validation issues before saving.");
    return;
  }
  try {
    setSetupState("working", "Saving cabinet profile override.");
    const response = await fetch(endpoints.cabinetProfile, {
      method: "PUT",
      headers: setupAuthHeaders(),
      body: JSON.stringify(cabinetProfileFromForm())
    });
    if (!response.ok) {
      const detail = await response.text();
      setSetupState("blocked", "Save failed: HTTP " + response.status + " " + detail.trim());
      return;
    }
    const profile = await response.json();
    fillCabinetSetupForm(profile.effective || {});
    setSetupState("saved", "Override saved. Refreshing appliance status.");
    schedulePoll(0);
  } catch (err) {
    setSetupState("blocked", err && err.message ? err.message : "Save failed.");
  }
}

async function clearCabinetProfileOverride() {
  if (setupActionsRequireToken() && !getSetupToken()) {
    setSetupState("blocked", "Enter an API token before clearing.");
    return;
  }
  try {
    setSetupState("working", "Clearing cabinet profile override.");
    const response = await fetch(endpoints.cabinetProfile, {
      method: "DELETE",
      headers: setupAuthHeaders()
    });
    if (!response.ok) {
      const detail = await response.text();
      setSetupState("blocked", "Clear failed: HTTP " + response.status + " " + detail.trim());
      return;
    }
    const profile = await response.json();
    fillCabinetSetupForm(profile.effective || {});
    setSetupState("saved", "Override cleared. File defaults are active.");
    schedulePoll(0);
  } catch (err) {
    setSetupState("blocked", err && err.message ? err.message : "Clear failed.");
  }
}

const certificateRoleMeta = {
  g2s_ca_cert: { label: "CA Certificate", requiresKey: false },
  g2s_client_cert: { label: "Client Certificate + Key", requiresKey: true },
  web_server_cert: { label: "Web Server Certificate + Key", requiresKey: true }
};

function selectedCertRole() {
  const value = $("cert-role-select").value || clientState.certSelectedRole || "g2s_ca_cert";
  if (certificateRoleMeta[value]) {
    return value;
  }
  return "g2s_ca_cert";
}

function certRoleDetails(role) {
  return certificateRoleMeta[role] || { label: role, requiresKey: false };
}

function certificateRecordByRole(snapshot, role) {
  const records = Array.isArray(snapshot?.certificates) ? snapshot.certificates : [];
  for (let i = 0; i < records.length; i++) {
    if (records[i] && records[i].role === role) {
      return records[i];
    }
  }
  return null;
}

function certRoleConfigured(record) {
  return !!String(record?.path || "").trim();
}

function privateKeyExportAllowed() {
  return currentRuntime().allow_private_key_export === true;
}

function getCertToken() {
  return $("cert-api-token").value.trim();
}

function certAuthHeaders() {
  const token = getCertToken();
  const headers = { "Content-Type": "application/json" };
  if (token) {
    headers.Authorization = "Bearer " + token;
  }
  return headers;
}

function setCertManagerState(level, message) {
  const badge = $("cert-manager-state");
  badge.textContent = level;
  badge.className = "source-pill " + (level === "ready" || level === "saved" ? "source-file" : level === "working" ? "source-override" : "source-mixed");
  $("cert-manager-message").textContent = message;
  $("cert-manager-detail").textContent = "";
}

function setCertManagerDetail(detail) {
  $("cert-manager-detail").textContent = detail || "";
}

function roleDisplayName(role) {
  return certRoleDetails(role).label;
}

function setCertKeyFieldVisible(visible) {
  $("cert-private-key-wrapper").classList.toggle("cert-key-hidden", !visible);
}

function certificateRoleRulesText(role) {
  const details = certRoleDetails(role);
  if (details.requiresKey) {
    return "Paste a PEM certificate and its matching PEM private key.";
  }
  return "Paste a PEM certificate only. Private key input is not used for this role.";
}

function validateCertificateManagerForm(snapshot) {
  const runtime = snapshot?.status?.runtime || currentRuntime();
  const role = selectedCertRole();
  const details = certRoleDetails(role);
  const record = certificateRecordByRole(snapshot, role);
  const configured = certRoleConfigured(record);
  const certPEM = $("cert-certificate-pem").value.trim();
  const privateKeyPEM = $("cert-private-key-pem").value.trim();
  const tokenPresent = !!getCertToken();
  const tokenRequired = runtime.api_mutation_auth_required === true;
  const exportKeyAllowed = runtime.allow_private_key_export === true;
  const problems = [];

  if (!configured) {
    problems.push(roleDisplayName(role) + " is not configured in appliance runtime.");
  }
  if (!certPEM) {
    problems.push("Certificate PEM is required.");
  }
  if (details.requiresKey && !privateKeyPEM) {
    problems.push("Private key PEM is required for " + roleDisplayName(role) + ".");
  }
  if (!details.requiresKey && privateKeyPEM) {
    problems.push("Private key PEM is not used for " + roleDisplayName(role) + ".");
  }
  if (tokenRequired && !tokenPresent) {
    problems.push("API token is required for certificate import in this browser session.");
  }

  let exportKeyPolicy = "Private key export not applicable for this role.";
  if (details.requiresKey) {
    if (!configured) {
      exportKeyPolicy = "Private key export unavailable until this role is configured.";
    } else if (!exportKeyAllowed) {
      exportKeyPolicy = "Private key export is disabled by appliance policy.";
    } else if (tokenRequired && !tokenPresent) {
      exportKeyPolicy = "Private key export is allowed, but this browser session still needs an API token.";
    } else {
      exportKeyPolicy = "Private key export is allowed for this role.";
    }
  }

  return {
    role: role,
    details: details,
    record: record,
    configured: configured,
    certPEM: certPEM,
    privateKeyPEM: privateKeyPEM,
    tokenRequired: tokenRequired,
    tokenPresent: tokenPresent,
    exportKeyAllowed: exportKeyAllowed,
    exportKeyPolicy: exportKeyPolicy,
    problems: problems
  };
}

function renderCertificateManager(snapshot) {
  const certState = validateCertificateManagerForm(snapshot);
  clientState.certSelectedRole = certState.role;
  const runtime = snapshot?.status?.runtime || {};
  const required = certRequired(runtime, certState.role);
  const status = parseCertState(certState.record?.status || "UNKNOWN");
  const severity = certState.record ? certSeverity(certState.record, runtime) : (required ? "blocking" : "lab");
  const impact = certImpactLabel(severity);
  const support = certState.configured ? "configured at " + certState.record.path : "not configured in current runtime";
  const stateDetail = certState.record ? status + " (" + impact + ")" : "no inventory record available";
  const privateKeyText = certState.details.requiresKey ? "Role supports private key import/export." : "Role is certificate-only.";
  $("cert-role-summary").textContent = roleDisplayName(certState.role) + ": " + support + "; state " + stateDetail + "; " + privateKeyText;
  $("cert-role-rules").textContent = certificateRoleRulesText(certState.role);
  $("cert-role-current-status").textContent = certState.record ? stateDetail : support;
  $("cert-role-export-policy").textContent = certState.exportKeyPolicy;
  $("cert-validation-summary").textContent = certState.problems.length ? certState.problems.length + " issue(s)" : "Ready to import";
  $("cert-validation-list").innerHTML = certState.problems.map((item) => "<div class=\"validation-item\">" + escapeHTML(item) + "</div>").join("");
  $("cert-api-token-wrapper").classList.toggle("trusted-bypass-hidden", !certState.tokenRequired);
  $("cert-token-controls").classList.toggle("trusted-bypass-hidden", !certState.tokenRequired);
  $("cert-api-token-label").textContent = certState.tokenRequired ? "API Token (required for import/export key)" : "API Token (optional on trusted private network)";
  $("cert-token-help-text").textContent = certState.tokenRequired
    ? "Use API token for import and private-key export actions."
    : "Trusted private network bypass is active for this browser; token is optional for import and private-key export.";

  setCertKeyFieldVisible(certState.details.requiresKey);
  $("cert-import-button").disabled = certState.problems.length > 0;
  $("cert-export-cert-button").disabled = !certState.configured;
  $("cert-export-key-button").disabled = !certState.details.requiresKey || !certState.configured || !certState.exportKeyAllowed || (certState.tokenRequired && !certState.tokenPresent);
  $("cert-copy-token-button").disabled = !certState.tokenPresent;
}

async function copyCertTokenToClipboard() {
  const token = getCertToken();
  if (!token) {
    setCertManagerState("blocked", "Enter an API token before copying.");
    return;
  }
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(token);
    } else {
      const input = $("cert-api-token");
      input.focus();
      input.select();
      document.execCommand("copy");
      input.setSelectionRange(input.value.length, input.value.length);
    }
    setCertManagerState("ready", "Certificate manager token copied to clipboard.");
  } catch (err) {
    setCertManagerState("blocked", err && err.message ? "Copy failed: " + err.message : "Copy failed.");
  }
}

function sanitizeHTTPText(raw) {
  return String(raw || "").replace(/\s+/g, " ").trim();
}

function downloadTextMaterial(filename, content) {
  const blob = new Blob([String(content || "")], { type: "text/plain;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1200);
}

async function exportCertificateMaterial(role, includeKey) {
  const details = certRoleDetails(role);
  if (includeKey) {
    if (!details.requiresKey) {
      setCertManagerState("blocked", roleDisplayName(role) + " does not support private key export.");
      return;
    }
    if (!privateKeyExportAllowed()) {
      setCertManagerState("blocked", "Private key export is disabled by appliance policy.");
      return;
    }
    if (setupActionsRequireToken() && !getCertToken()) {
      setCertManagerState("blocked", "Enter an API token before exporting private key material.");
      return;
    }
  }
  try {
    setCertManagerState("working", "Exporting " + roleDisplayName(role) + (includeKey ? " certificate + key." : " certificate."));
    const query = includeKey ? "?role=" + encodeURIComponent(role) + "&include_key=true" : "?role=" + encodeURIComponent(role);
    const response = await fetch(endpoints.certificateExport + query, {
      method: "GET",
      headers: includeKey ? certAuthHeaders() : undefined,
      cache: "no-store"
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setCertManagerState("blocked", "Export failed: HTTP " + response.status + (detail ? " " + detail : ""));
      return;
    }
    const payload = await response.json();
    downloadTextMaterial(role + ".crt.pem", payload.certificate_pem || "");
    if (includeKey && payload.private_key_pem) {
      downloadTextMaterial(role + ".key.pem", payload.private_key_pem);
    }
    setCertManagerState("saved", "Exported " + roleDisplayName(role) + (includeKey ? " certificate + key." : " certificate."));
    setCertManagerDetail("Role: " + payload.role + (includeKey ? " | Included private key: yes" : " | Included private key: no"));
  } catch (err) {
    setCertManagerState("blocked", err && err.message ? err.message : "Export failed.");
  }
}

function certImportPayload() {
  return {
    role: selectedCertRole(),
    certificate_pem: $("cert-certificate-pem").value.trim(),
    private_key_pem: $("cert-private-key-pem").value.trim()
  };
}

async function importCertificateMaterial(event) {
  event.preventDefault();
  const payload = certImportPayload();
  const details = certRoleDetails(payload.role);
  const certState = validateCertificateManagerForm(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  if (certState.problems.length > 0) {
    setCertManagerState("blocked", "Resolve certificate import issues before importing.");
    setCertManagerDetail(certState.problems.join(" "));
    return;
  }
  if (!details.requiresKey) {
    payload.private_key_pem = "";
  }

  try {
    setCertManagerState("working", "Importing " + roleDisplayName(payload.role) + " material.");
    const response = await fetch(endpoints.certificateImport, {
      method: "POST",
      headers: certAuthHeaders(),
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setCertManagerState("blocked", "Import failed: HTTP " + response.status + (detail ? " " + detail : ""));
      return;
    }
    const result = await response.json();
    const certState = parseCertState(result.certificate_status || "UNKNOWN");
    setCertManagerState("saved", "Import complete for " + roleDisplayName(payload.role) + " (" + certState + ").");
    setCertManagerDetail("Subject: " + (result.certificate_subject || "-") + " | Path: " + (result.certificate_path || "-") + (result.private_key_path ? " | Key path: " + result.private_key_path : ""));
    if (!details.requiresKey) {
      $("cert-private-key-pem").value = "";
    }
    schedulePoll(0);
  } catch (err) {
    setCertManagerState("blocked", err && err.message ? err.message : "Import failed.");
  }
}

function clearCertificateManagerForm() {
  $("cert-certificate-pem").value = "";
  $("cert-private-key-pem").value = "";
  setCertManagerState("ready", "Certificate form cleared.");
  setCertManagerDetail("");
  renderCertificateManager(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
}

function cabinetSetupHasFocus() {
  const form = $("cabinet-setup-form");
  return !!(form && form.contains(document.activeElement));
}

function syncCabinetSetupFromSnapshot(snapshot) {
  const profileResponse = snapshot?.cabinetProfile;
  const profile = profileResponse?.effective || snapshot?.status?.cabinet_profile || {};
  if (!profile || cabinetSetupHasFocus()) {
    renderCabinetSetupValidation();
    return;
  }
  fillCabinetSetupForm(profile);
}

function renderStatus(snapshot) {
  const status = snapshot?.status || {};
  const readyz = snapshot?.readyz || {};
  const runtime = status.runtime || {};
  const readiness = status.readiness || {};

  $("controller-id").textContent = status.controller_id || "-";
  setStatePill($("controller-state"), status.state || "UNKNOWN");
  setStatePill($("readiness-state"), readiness.overall || "UNKNOWN");
  setStatePill($("readyz-state"), readyz.overall || "UNKNOWN");
  $("input-mode").textContent = runtime.input_mode || "-";
  $("last-event").textContent = status.last_event || "-";
  $("active-incident").textContent = status.incident ? ("#" + status.incident.id + " " + status.incident.trigger_type) : "None";
  $("uptime").textContent = runtime.uptime_seconds != null ? Math.max(0, runtime.uptime_seconds) + "s uptime" : "-";
  $("bind-address").textContent = runtime.bind_address || "-";
  $("database-path").textContent = runtime.database_path || "-";
  $("g2s-endpoint").textContent = runtime.g2s_host_url ? (runtime.g2s_host_url + " " + (runtime.g2s_endpoint_path || "")) : "-";
  $("tls-mode").textContent = runtime.tls_required ? "TLS required" : "HTTP lab mode";
  $("readyz-http").textContent = readyz.statusCode ? (readyz.statusCode + (readyz.ok ? " OK" : " DEGRADED")) : "-";
  $("readyz-issues").textContent = Array.isArray(readyz.issues) && readyz.issues.length ? readyz.issues.join("; ") : "None";
  $("readiness-warnings").textContent = Array.isArray(readiness.warnings) && readiness.warnings.length ? readiness.warnings.join("; ") : "None";

  renderCertificateSummary(readiness.certificate_summary || {}, snapshot?.certificates, runtime);
  renderCertificateManager(snapshot);
  renderCabinetProfile(status);
  renderFirstCabinetSession(snapshot);
  renderSessionEvidence(snapshot);
  syncCabinetSetupFromSnapshot(snapshot);
  renderEGMTable(status);
  renderItems("incident-list", snapshot?.incidents, "No incidents recorded", (item) =>
    "<div class=\"item\"><strong>#" + escapeHTML(item.id) + " " + escapeHTML(item.trigger_type) + "</strong><span>" + escapeHTML(fmtTime(item.created_at)) + " " + escapeHTML(item.trigger_source || "") + "</span></div>"
  );
  renderItems("egm-history", snapshot?.egmHistory, "No EGM history yet", (item) =>
    "<div class=\"item\"><strong>" + escapeHTML(item.egm_id) + " " + statusPill(item.status) + "</strong><span>" + escapeHTML(item.event_type) + " at " + escapeHTML(fmtTime(item.created_at)) + "</span></div>"
  );
  renderItems("state-history", snapshot?.stateHistory, "No state history yet", (item) =>
    "<div class=\"item\"><strong>" + escapeHTML(item.old_state) + " -> " + escapeHTML(item.new_state) + "</strong><span>" + escapeHTML(item.reason) + " at " + escapeHTML(fmtTime(item.created_at)) + "</span></div>"
  );
  renderItems("certificate-list", snapshot?.certificates, "No certificate inventory yet", (item) => {
    const severity = certSeverity(item, runtime);
    const state = parseCertState(item.status);
    const path = item.path || "not configured";
    const expiry = item.not_after ? (" expires " + fmtTime(item.not_after)) : "";
    const reason = certExplanation(item, runtime);
    const impact = certImpactLabel(severity);
    return "<div class=\"item cert-item cert-" + severity + "\"><strong>" + escapeHTML(item.role) + " " + statusPill(state) + "</strong><span>" + escapeHTML(path + expiry) + "</span><br><span>" + escapeHTML(reason) + "</span><br><span class=\"cert-impact cert-impact-" + severity + "\">" + escapeHTML(impact) + "</span></div>";
  });
}

function setAlert(level, title, detail) {
  const el = $("operator-alert");
  el.className = "alert-strip alert-" + level;
  $("alert-title").textContent = title;
  $("alert-detail").textContent = detail;
}

function renderAlerts(snapshot) {
  const status = snapshot?.status || {};
  const readyz = snapshot?.readyz || {};
  const readiness = status.readiness || {};
  const runtime = status.runtime || {};
  const egms = Array.isArray(status.egms) ? status.egms : [];
  const unhealthy = egms.filter((egm) => unhealthyStates.has(String(egm.status || "").toUpperCase()));
  const blockingCerts = (snapshot?.certificates || []).filter((cert) => certSeverity(cert, runtime) === "blocking");

  const readyzDegraded = readyz.ok === false || readyz.overall === "DEGRADED";
  document.body.classList.toggle("console-degraded", readyzDegraded);

  if (readyzDegraded || readyz.overall === "DEGRADED") {
    const issues = Array.isArray(readyz.issues) && readyz.issues.length ? readyz.issues.join("; ") : "readiness endpoint returned degraded";
    setAlert("critical", "Readiness degraded", issues);
    return;
  }
  if (status.incident) {
    setAlert("warning", "Active incident #" + status.incident.id, status.incident.trigger_type + " from " + (status.incident.trigger_source || "unknown"));
    return;
  }
  if (unhealthy.length > 0) {
    setAlert("warning", "Unhealthy EGMs: " + unhealthy.length, "One or more EGMs are RED or GREY.");
    return;
  }
  if (blockingCerts.length > 0) {
    setAlert("warning", "Blocking certificate issues: " + blockingCerts.length, "Required certificates must be corrected for healthy runtime.");
    return;
  }
  if (Array.isArray(readiness.warnings) && readiness.warnings.length > 0) {
    setAlert("info", "Lab mode warnings", readiness.warnings.join("; "));
    return;
  }
  setAlert("healthy", "Console healthy", "Readiness is stable with no active operator alerts.");
}

function updateStaleBadge() {
  const badge = $("stale-badge");
  if (!clientState.lastGoodAt) {
    badge.className = "stale-badge stale-critical";
    badge.textContent = "No successful snapshot";
    return;
  }
  const ageSeconds = Math.floor((Date.now() - clientState.lastGoodAt) / 1000);
  if (ageSeconds > 30) {
    badge.className = "stale-badge stale-critical";
    badge.textContent = "Stale: " + ageSeconds + "s old";
    return;
  }
  if (ageSeconds > 10) {
    badge.className = "stale-badge stale-warning";
    badge.textContent = "Aging: " + ageSeconds + "s old";
    return;
  }
  badge.className = "stale-badge stale-fresh";
  badge.textContent = "Fresh: " + ageSeconds + "s old";
}

async function fetchJSON(url) {
  const response = await fetch(url, { cache: "no-store" });
  if (!response.ok) {
    throw new Error(url + " -> HTTP " + response.status);
  }
  return response.json();
}

async function fetchReadyz() {
  const response = await fetch(endpoints.readyz, { cache: "no-store" });
  const raw = await response.text();
  let payload = {};
  if (raw) {
    try {
      payload = JSON.parse(raw);
    } catch (_) {
      payload = {};
    }
  }
  return {
    ok: response.ok,
    statusCode: response.status,
    overall: payload.overall || (response.ok ? "READY" : "DEGRADED"),
    issues: Array.isArray(payload.issues) ? payload.issues : []
  };
}

function schedulePoll(delayMs) {
  if (clientState.timerId) {
    clearTimeout(clientState.timerId);
  }
  clientState.timerId = setTimeout(pollOnce, Math.max(0, delayMs));
}

function nextBackoffMs() {
  const steps = [3000, 6000, 10000];
  const delay = steps[Math.min(clientState.backoffStep, steps.length - 1)];
  if (clientState.backoffStep < steps.length - 1) {
    clientState.backoffStep++;
  }
  clientState.pollIntervalMs = delay;
  return delay;
}

function resetBackoff() {
  clientState.backoffStep = 0;
  clientState.pollIntervalMs = 3000;
}

function updateRefreshText(ok) {
  if (ok) {
    $("last-refresh").textContent = "Updated " + new Date(clientState.lastGoodAt).toLocaleTimeString();
    return;
  }
  $("last-refresh").textContent = "Last poll issue: " + (clientState.lastError || "unknown");
}

function showAPIFailureBanner(summary) {
  const banner = $("api-failure-banner");
  banner.classList.remove("api-banner-hidden");
  $("api-failure-detail").textContent = summary || "One or more API requests failed; showing cached data where available.";
}

function hideAPIFailureBanner() {
  $("api-failure-banner").classList.add("api-banner-hidden");
}

async function pollOnce() {
  if (clientState.inFlight) return;
  clientState.inFlight = true;
  $("refresh-button").disabled = true;

  const baseline = copySnapshot(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  const failures = [];
  let statusOK = false;
  let readyzOK = false;

  try {
    const results = await Promise.allSettled([
      fetchJSON(endpoints.status),
      fetchReadyz(),
      fetchJSON(endpoints.incidents),
      fetchJSON(endpoints.egmHistory),
      fetchJSON(endpoints.stateHistory),
      fetchJSON(endpoints.certificates),
      fetchJSON(endpoints.sessionEvidence),
      fetchJSON(endpoints.cabinetProfile),
      fetchJSON(endpoints.cabinetPreflight)
    ]);

    const [statusResult, readyzResult, incidentsResult, egmHistoryResult, stateHistoryResult, certificatesResult, sessionEvidenceResult, cabinetProfileResult, cabinetPreflightResult] = results;
    const snapshot = copySnapshot(baseline);

    if (statusResult.status === "fulfilled") {
      snapshot.status = statusResult.value;
      statusOK = true;
    } else {
      failures.push("status unavailable");
    }

    if (readyzResult.status === "fulfilled") {
      snapshot.readyz = readyzResult.value;
      readyzOK = true;
    } else {
      snapshot.readyz = {
        ok: false,
        statusCode: 0,
        overall: "DEGRADED",
        issues: ["readyz unavailable"]
      };
      failures.push("readyz unavailable");
    }

    if (incidentsResult.status === "fulfilled") snapshot.incidents = incidentsResult.value;
    if (egmHistoryResult.status === "fulfilled") snapshot.egmHistory = egmHistoryResult.value;
    if (stateHistoryResult.status === "fulfilled") snapshot.stateHistory = stateHistoryResult.value;
    if (certificatesResult.status === "fulfilled") snapshot.certificates = certificatesResult.value;
    if (sessionEvidenceResult.status === "fulfilled") snapshot.sessionEvidence = sessionEvidenceResult.value;
    if (cabinetProfileResult.status === "fulfilled") snapshot.cabinetProfile = cabinetProfileResult.value;
    if (cabinetPreflightResult.status === "fulfilled") snapshot.cabinetPreflight = cabinetPreflightResult.value;

    if (incidentsResult.status !== "fulfilled") failures.push("incidents unavailable");
    if (egmHistoryResult.status !== "fulfilled") failures.push("egm history unavailable");
    if (stateHistoryResult.status !== "fulfilled") failures.push("state history unavailable");
    if (certificatesResult.status !== "fulfilled") failures.push("certificates unavailable");
    if (sessionEvidenceResult.status !== "fulfilled") failures.push("session evidence unavailable");
    if (cabinetProfileResult.status !== "fulfilled") failures.push("cabinet profile unavailable");
    if (cabinetPreflightResult.status !== "fulfilled") failures.push("cabinet preflight unavailable");

    clientState.displaySnapshot = snapshot;
    renderStatus(snapshot);
    renderAlerts(snapshot);
    if (failures.length > 0) {
      showAPIFailureBanner(failures.join("; "));
    } else {
      hideAPIFailureBanner();
    }

    const goodPoll = statusOK && readyzOK;
    if (goodPoll) {
      clientState.lastGoodStatus = copySnapshot(snapshot);
      clientState.lastGoodAt = Date.now();
      clientState.lastError = "";
      resetBackoff();
      updateRefreshText(true);
      updateStaleBadge();
      schedulePoll(clientState.pollIntervalMs);
    } else {
      clientState.lastError = failures.join("; ");
      updateRefreshText(false);
      updateStaleBadge();
      schedulePoll(nextBackoffMs());
    }
  } catch (err) {
    clientState.lastError = err && err.message ? err.message : "poll failed";
    showAPIFailureBanner(clientState.lastError);
    if (clientState.lastGoodStatus && !clientState.displaySnapshot) {
      clientState.displaySnapshot = copySnapshot(clientState.lastGoodStatus);
      renderStatus(clientState.displaySnapshot);
      renderAlerts(clientState.displaySnapshot);
    }
    updateRefreshText(false);
    updateStaleBadge();
    schedulePoll(nextBackoffMs());
  } finally {
    clientState.inFlight = false;
    $("refresh-button").disabled = false;
  }
}

function setFilter(filter) {
  clientState.egmFilter = filter;
  document.querySelectorAll(".filter-tab").forEach((tab) => {
    tab.classList.toggle("is-active", tab.dataset.filter === filter);
  });
  if (clientState.displaySnapshot) {
    renderEGMTable(clientState.displaySnapshot.status || {});
    renderAlerts(clientState.displaySnapshot);
  }
}

function setSort(key) {
  if (clientState.egmSortKey === key) {
    clientState.egmSortDir = clientState.egmSortDir === "asc" ? "desc" : "asc";
  } else {
    clientState.egmSortKey = key;
    clientState.egmSortDir = "asc";
  }
  if (clientState.displaySnapshot) {
    renderEGMTable(clientState.displaySnapshot.status || {});
  } else {
    updateSortLabels();
  }
}

function bindControls() {
  $("refresh-button").addEventListener("click", () => {
    schedulePoll(0);
  });

  $("session-evidence-save-button").addEventListener("click", saveSessionEvidenceToHistory);
  $("session-evidence-json-button").addEventListener("click", exportSessionEvidenceJSON);
  $("session-evidence-markdown-button").addEventListener("click", exportSessionEvidenceMarkdown);
  $("session-evidence-export-all-button").addEventListener("click", exportAllSavedSessionEvidence);
  $("session-evidence-history").addEventListener("click", (event) => {
    const button = event.target.closest(".session-evidence-history-button");
    if (!button) return;
    const id = button.getAttribute("data-evidence-id") || "";
    const action = button.getAttribute("data-evidence-action") || "";
    if (action === "select") {
      viewSavedSessionEvidence(id);
      return;
    }
    if (action === "delete") {
      deleteSavedSessionEvidence(id);
      return;
    }
    if (action === "json") {
      exportSavedSessionEvidenceJSON(id);
      return;
    }
    if (action === "markdown") {
      exportSavedSessionEvidenceMarkdown(id);
    }
  });

  $("cert-manager-form").addEventListener("submit", importCertificateMaterial);
  $("cert-role-select").addEventListener("change", () => {
    clientState.certSelectedRole = selectedCertRole();
    renderCertificateManager(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("cert-certificate-pem").addEventListener("input", () => {
    renderCertificateManager(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("cert-private-key-pem").addEventListener("input", () => {
    renderCertificateManager(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("cert-api-token").addEventListener("input", () => {
    const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
    renderCertificateManager(snapshot);
    renderFirstCabinetSession(snapshot);
  });
  $("cert-copy-token-button").addEventListener("click", copyCertTokenToClipboard);
  $("cert-export-cert-button").addEventListener("click", () => {
    exportCertificateMaterial(selectedCertRole(), false);
  });
  $("cert-export-key-button").addEventListener("click", () => {
    exportCertificateMaterial(selectedCertRole(), true);
  });
  $("cert-clear-form-button").addEventListener("click", clearCertificateManagerForm);
  document.querySelectorAll(".cert-export-role-button").forEach((button) => {
    button.addEventListener("click", () => {
      const role = button.getAttribute("data-export-role") || selectedCertRole();
      exportCertificateMaterial(role, false);
    });
  });

  $("cabinet-setup-form").addEventListener("submit", saveCabinetProfileOverride);
  $("setup-reset-button").addEventListener("click", clearCabinetProfileOverride);
  $("setup-copy-token-button").addEventListener("click", copySetupTokenToClipboard);
  $("setup-reload-button").addEventListener("click", () => {
    reloadCabinetProfileForm().catch((err) => {
      setSetupState("blocked", err && err.message ? err.message : "Reload failed.");
    });
  });
  document.querySelectorAll("#cabinet-setup-form input").forEach((input) => {
    input.addEventListener("input", () => {
      renderCabinetSetupValidation();
      renderFirstCabinetSession(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
      renderSessionEvidence(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
    });
  });

  document.querySelectorAll(".filter-tab").forEach((button) => {
    button.addEventListener("click", () => {
      setFilter(button.dataset.filter || "all");
    });
  });

  document.querySelectorAll(".sort-button").forEach((button) => {
    button.addEventListener("click", () => {
      setSort(button.dataset.sortKey || "egm_id");
    });
  });
}

bindControls();
updateSortLabels();
updateStaleBadge();
renderCertificateManager(emptySnapshot());
renderFirstCabinetSession(emptySnapshot());
renderSessionEvidence(emptySnapshot());
schedulePoll(0);
setInterval(updateStaleBadge, 1000);`
