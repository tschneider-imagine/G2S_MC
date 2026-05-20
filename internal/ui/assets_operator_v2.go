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
            <label>API Token<input id="setup-api-token" name="api_token" type="password" autocomplete="off"></label>
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
            <button id="setup-save-button" type="submit">Save Override</button>
            <button id="setup-reset-button" type="button" class="secondary-button">Clear Override</button>
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

    <section class="grid">
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

.form-grid input:focus {
  outline: 2px solid rgba(36, 95, 145, 0.24);
  border-color: var(--blue);
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
  cabinetProfile: "/api/cabinet-profile"
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
  egmFilter: "all"
};

function emptySnapshot() {
  return {
    status: null,
    readyz: null,
    incidents: [],
    egmHistory: [],
    stateHistory: [],
    certificates: [],
    cabinetProfile: null
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
    cabinetProfile: snapshot.cabinetProfile || null
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
  if (state === "OK") return "healthy";
  if (state === "EXPIRING_SOON") return "warning";
  if (state === "NOT_CONFIGURED") return "lab";
  return certRequired(runtime, item.role) ? "blocking" : "warning";
}

function certExplanation(item, runtime) {
  const state = parseCertState(item.status);
  const required = certRequired(runtime, item.role);
  if (state === "OK") return "Loaded and valid for current runtime.";
  if (state === "EXPIRING_SOON") return "Rotation needed soon to avoid readiness degradation.";
  if (state === "NOT_CONFIGURED") return "Expected in lab mode when TLS and mTLS are not required.";
  if (state === "MISSING") return required ? "Blocking: required certificate file is missing." : "Missing, but currently not required.";
  if (state === "INVALID") return required ? "Blocking: certificate failed validation for a required role." : "Invalid, but currently not required.";
  return required ? "Blocking: certificate state could not be validated." : "Unknown certificate state.";
}

function renderCertificateSummary(summary) {
  const counts = {
    healthy: summary.OK || 0,
    warning: summary.EXPIRING_SOON || 0,
    lab: summary.NOT_CONFIGURED || 0,
    blocking: (summary.MISSING || 0) + (summary.INVALID || 0) + (summary.UNKNOWN || 0)
  };
  $("certificate-summary").innerHTML = [
    "<div class=\"summary-cell summary-blocking\"><strong>" + counts.blocking + "</strong><span>Blocking</span><p>MISSING / INVALID / UNKNOWN</p></div>",
    "<div class=\"summary-cell summary-warning\"><strong>" + counts.warning + "</strong><span>Expiring Soon</span><p>Plan rotation before expiry.</p></div>",
    "<div class=\"summary-cell summary-lab\"><strong>" + counts.lab + "</strong><span>Lab Expected</span><p>NOT_CONFIGURED in local lab mode.</p></div>",
    "<div class=\"summary-cell summary-healthy\"><strong>" + counts.healthy + "</strong><span>Healthy</span><p>Ready for configured runtime.</p></div>"
  ].join("");
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
  const sanValues = []
    .concat(profile.required_san_dns.map((item) => "DNS:" + item))
    .concat(profile.required_san_ips.map((item) => "IP:" + item));
  $("setup-san-summary").textContent = "wire host " + host + "; " + (sanValues.length ? sanValues.join(", ") : "no SAN values");
  $("setup-validation-summary").textContent = result.problems.length ? result.problems.length + " issue(s)" : "Ready to save";
  $("setup-validation-list").innerHTML = result.problems.map((item) => "<div class=\"validation-item\">" + escapeHTML(item) + "</div>").join("");
  return result;
}

function setupAuthHeaders() {
  const token = $("setup-api-token").value.trim();
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

  renderCertificateSummary(readiness.certificate_summary || {});
  renderCabinetProfile(status);
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
    return "<div class=\"item cert-item cert-" + severity + "\"><strong>" + escapeHTML(item.role) + " " + statusPill(state) + "</strong><span>" + escapeHTML(path + expiry) + "</span><br><span>" + escapeHTML(reason) + "</span></div>";
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
      fetchJSON(endpoints.cabinetProfile)
    ]);

    const [statusResult, readyzResult, incidentsResult, egmHistoryResult, stateHistoryResult, certificatesResult, cabinetProfileResult] = results;
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
    if (cabinetProfileResult.status === "fulfilled") snapshot.cabinetProfile = cabinetProfileResult.value;

    if (incidentsResult.status !== "fulfilled") failures.push("incidents unavailable");
    if (egmHistoryResult.status !== "fulfilled") failures.push("egm history unavailable");
    if (stateHistoryResult.status !== "fulfilled") failures.push("state history unavailable");
    if (certificatesResult.status !== "fulfilled") failures.push("certificates unavailable");
    if (cabinetProfileResult.status !== "fulfilled") failures.push("cabinet profile unavailable");

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

  $("cabinet-setup-form").addEventListener("submit", saveCabinetProfileOverride);
  $("setup-reset-button").addEventListener("click", clearCabinetProfileOverride);
  $("setup-reload-button").addEventListener("click", () => {
    reloadCabinetProfileForm().catch((err) => {
      setSetupState("blocked", err && err.message ? err.message : "Reload failed.");
    });
  });
  document.querySelectorAll("#cabinet-setup-form input").forEach((input) => {
    input.addEventListener("input", renderCabinetSetupValidation);
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
schedulePoll(0);
setInterval(updateStaleBadge, 1000);`
