package ui

const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <link rel="stylesheet" href="{{.DashboardCSSURL}}">
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
      <div id="api-connection-indicator" class="api-connection-indicator api-connection-disconnected">
        <strong id="api-connection-state">disconnected</strong>
        <span id="api-connection-last-success">last success: never</span>
        <span id="api-connection-poll-attempts">poll attempts: 0</span>
        <span id="api-connection-startup">JS loaded at: - | poll started: no</span>
        <span id="api-connection-last-error">last error: none</span>
      </div>
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

    <section class="grid two operator-toolbar-grid">
      <div class="panel operator-actions-bar">
        <div class="panel-head panel-head-stack">
          <div class="panel-title-row">
            <h2>Operator Actions</h2>
            <span class="muted-text">Quick controls</span>
          </div>
          <span class="muted-text">Use these actions without leaving the current session view.</span>
        </div>
        <div class="setup-actions operator-actions-buttons">
          <button id="operator-actions-refresh-button" type="button">Refresh now</button>
          <button id="operator-actions-capture-evidence-button" type="button" class="secondary-button">Capture evidence</button>
          <button id="operator-actions-export-package-button" type="button" class="secondary-button">Export session package</button>
          <button id="operator-actions-blocker-governance-button" type="button" class="secondary-button">Open Blocker Governance</button>
        </div>
      </div>
      <div class="panel global-view-controls-panel">
        <div class="panel-head panel-head-stack">
          <div class="panel-title-row">
            <h2>Global View Controls</h2>
            <span class="muted-text">Client-side only</span>
          </div>
          <span id="global-view-summary" class="muted-text">Severity: All | Text: none | Compact mode: off</span>
        </div>
        <div class="form-grid global-view-controls">
          <label>Severity Filter
            <select id="global-severity-filter">
              <option value="all">All</option>
              <option value="alerts">Alerts</option>
              <option value="warnings">Warnings</option>
              <option value="healthy">Healthy</option>
            </select>
          </label>
          <label>Text Filter
            <input id="global-text-filter" type="text" placeholder="EGM, endpoint, audit text">
          </label>
          <label class="global-compact-toggle">
            <span>Compact mode</span>
            <input id="global-compact-mode-toggle" type="checkbox">
          </label>
        </div>
      </div>
    </section>

    <section class="grid">
      <div class="panel egm-focus-panel">
        <div class="panel-head panel-head-stack">
          <div class="panel-title-row">
            <h2>EGM Focus</h2>
            <span id="egm-focus-summary" class="muted-text">All EGMs</span>
          </div>
          <span id="egm-focus-description" class="muted-text">Use this to scope EGM-specific views while keeping global context visible.</span>
        </div>
        <div class="focus-controls">
          <label for="egm-focus-select">EGM Focus</label>
          <select id="egm-focus-select">
            <option value="">All EGMs</option>
          </select>
        </div>
        <div class="focus-egm-summary-wrap">
          <p class="label">Multi-EGM Grouped Summary</p>
          <div id="egm-grouped-summary-scope" class="muted-text">Scope: All EGMs</div>
          <div id="egm-grouped-summary" class="timeline egm-grouped-summary"></div>
        </div>
        <div class="focus-selected-egm-wrap">
          <p class="label">Selected EGM Detail</p>
          <div id="selected-egm-detail-scope" class="muted-text">Scope: All EGMs</div>
          <div id="selected-egm-detail" class="timeline selected-egm-detail"></div>
        </div>
      </div>
    </section>

    <section class="grid two">
      <div class="panel wide">
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
          <div><dt>Endpoint Integrity Alerts</dt><dd id="first-cabinet-endpoint-alerts">-</dd></div>
          <div><dt>API Auth State</dt><dd id="first-cabinet-auth-state">-</dd></div>
        </dl>
        <div class="mute-path-status-wrap">
          <p class="label">Mute Path vs Runbook Readiness</p>
          <div id="mute-path-summary-grid" class="mute-path-summary-grid">
            <div id="mute-path-status-card" class="operator-readiness-group group-informational mute-path-status-card">
              <strong id="mute-path-state">-</strong>
              <span id="mute-path-message" class="muted-text">Mute path status will appear with telemetry.</span>
              <span id="mute-path-confidence" class="muted-text">Software signal only.</span>
            </div>
            <div id="runbook-readiness-status-card" class="operator-readiness-group group-informational runbook-readiness-status-card">
              <strong id="runbook-readiness-state">-</strong>
              <span id="runbook-readiness-message" class="muted-text">Runbook readiness status will appear with telemetry.</span>
              <span id="runbook-readiness-next" class="muted-text">Next action will appear here.</span>
            </div>
          </div>
          <div id="mute-path-prep-status" class="muted-text">Cabinet prep status will appear with telemetry.</div>
        </div>
        <div class="first-cabinet-session-blockers-wrap">
          <p class="label">Runbook Blockers and Signals</p>
          <div id="first-cabinet-session-blockers" class="first-cabinet-session-blockers"></div>
        </div>
        <div class="first-cabinet-session-actions-wrap">
          <p class="label">Operator Readiness Model</p>
          <div class="operator-action-summary-grid">
            <div><p class="label">Ready Now</p><strong id="operator-action-ready-count">0</strong></div>
            <div><p class="label">Needs Operator Action</p><strong id="operator-action-needed-count">0</strong></div>
            <div><p class="label">Lab Warning</p><strong id="operator-action-lab-warning-count">0</strong></div>
            <div><p class="label">Informational</p><strong id="operator-action-info-count">0</strong></div>
          </div>
          <div id="operator-readiness-model" class="operator-readiness-model"></div>
        </div>
        <div class="first-cabinet-session-actions-wrap">
          <p class="label">Next Operator Actions</p>
          <div id="next-operator-actions" class="operator-readiness-model"></div>
        </div>
        <div class="first-cabinet-session-workflow-wrap">
          <p class="label">Operator Workflow</p>
          <div id="first-cabinet-session-workflow" class="first-cabinet-session-workflow"></div>
        </div>
        <div class="first-cabinet-session-workflow-progress-wrap">
          <p class="label">Workflow Progress</p>
          <div class="workflow-progress-meta">
            <span id="workflow-progress-last-saved">Last saved: not saved</span>
            <span id="workflow-progress-unsaved" class="workflow-progress-unsaved workflow-progress-unsaved-clean">Saved</span>
          </div>
          <div class="workflow-progress-grid">
            <label>Current Phase
              <select id="workflow-progress-phase">
                <option value="pre_check">Pre-check</option>
                <option value="connect_observe">Connect/Observe</option>
                <option value="run_active">Run Active</option>
                <option value="capture_evidence">Capture Evidence</option>
                <option value="session_complete">Session Complete</option>
              </select>
            </label>
          </div>
          <div id="workflow-progress-steps" class="workflow-progress-steps"></div>
          <label class="cert-textarea-label">Operator Notes
            <textarea id="workflow-progress-notes" rows="4" placeholder="Optional operator notes for workflow continuity."></textarea>
          </label>
          <div class="setup-actions">
            <button id="workflow-progress-save-button" type="button">Save Progress</button>
            <button id="workflow-progress-clear-button" type="button" class="secondary-button">Clear Progress</button>
          </div>
          <div id="workflow-progress-message" class="muted-text">Workflow progress is not saved yet.</div>
        </div>
        <div class="first-cabinet-session-actions-wrap">
          <p class="label">Session Package Export</p>
          <div class="setup-actions">
            <button id="session-package-export-button" type="button" class="secondary-button">Export Session Package</button>
          </div>
          <div id="session-package-export-message" class="muted-text">Download one JSON package with current status, preflight, workflow, heartbeat policy, operator audit, and saved capture metadata.</div>
        </div>
        <div class="first-cabinet-session-actions-wrap">
          <p class="label">Runtime Override Snapshot</p>
          <label class="cert-textarea-label">Restore Snapshot JSON
            <textarea id="runtime-overrides-restore-json" rows="6" placeholder="Paste runtime override snapshot JSON here before restore."></textarea>
          </label>
          <div class="setup-actions">
            <button id="runtime-overrides-save-snapshot-button" type="button" class="secondary-button">Save Runtime Snapshot</button>
            <label class="runtime-overrides-upload-label">Upload Snapshot JSON
              <input id="runtime-overrides-restore-file" type="file" accept=".json,application/json">
            </label>
            <button id="runtime-overrides-restore-button" type="button">Restore Runtime Snapshot</button>
          </div>
          <div id="runtime-overrides-message" class="muted-text">Snapshot includes cabinet profile, heartbeat policy, blocker policy, and EGM registry overrides.</div>
        </div>
        <div class="first-cabinet-session-actions-wrap">
          <p class="label">Runtime Preset Library</p>
          <div class="workflow-progress-grid">
            <label>Preset Name<input id="runtime-preset-name" type="text" maxlength="64" placeholder="lab_baseline"></label>
          </div>
          <label class="cert-textarea-label">Preset Note (optional)
            <textarea id="runtime-preset-note" rows="3" maxlength="2000" placeholder="Optional operator note for this preset."></textarea>
          </label>
          <div class="setup-actions">
            <button id="runtime-preset-save-button" type="button">Save Current as Preset</button>
            <button id="runtime-preset-refresh-button" type="button" class="secondary-button">Refresh Presets</button>
          </div>
          <div id="runtime-preset-message" class="muted-text">Saved presets can be loaded to restore runtime overrides quickly.</div>
          <div id="runtime-presets-list" class="timeline panel-scroll-safe panel-scroll-safe-history"></div>
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
          <div><dt>EGM Focus</dt><dd id="session-evidence-egm-focus">All EGMs</dd></div>
          <div><dt>Incidents in Snapshot (global)</dt><dd id="session-evidence-incident-count">0</dd></div>
          <div><dt>State History Rows (global)</dt><dd id="session-evidence-state-count">0</dd></div>
          <div><dt>Run Markers in Snapshot (global)</dt><dd id="session-evidence-run-marker-count">0</dd></div>
          <div><dt>EGM History Rows (focused)</dt><dd id="session-evidence-egm-count">0</dd></div>
          <div><dt>EGM Groups (focused / all)</dt><dd id="session-evidence-egm-groups">0 / 0</dd></div>
          <div><dt>Heartbeat Events (focused)</dt><dd id="session-evidence-heartbeat-count">0</dd></div>
          <div><dt>Heartbeat Health</dt><dd id="session-evidence-heartbeat-health">-</dd></div>
          <div><dt>Heartbeat Source</dt><dd id="session-evidence-heartbeat-source">-</dd></div>
        </dl>
        <label class="cert-textarea-label evidence-notes-label">Operator Notes
          <textarea id="session-evidence-notes" rows="5" placeholder="Optional test notes, cabinet observations, or follow-up context."></textarea>
        </label>
        <div class="setup-actions evidence-actions">
          <button id="session-evidence-save-button" type="button">Save to Appliance History</button>
          <button id="session-evidence-json-button" type="button">Download JSON Evidence</button>
          <button id="session-evidence-markdown-button" type="button" class="secondary-button">Download Markdown Evidence</button>
          <button id="session-evidence-export-all-button" type="button" class="secondary-button">Export All Captures</button>
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
          <div id="setup-observed-egms-preview" class="muted-text">Observed EGM suggestions will appear here.</div>
          <div class="setup-actions">
            <button id="setup-save-button" type="submit" disabled>Save Override</button>
            <button id="setup-reset-button" type="button" class="secondary-button" disabled>Clear Override</button>
            <button id="setup-use-observed-egms-button" type="button" class="secondary-button">Use Observed EGMs</button>
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
              <button type="button" class="filter-tab" data-filter="endpoint_integrity">Endpoint Alerts</button>
            </div>
            <span id="egm-sort-label" class="muted-text">Sort: EGM ID asc</span>
          </div>
          <div class="setup-actions egm-bulk-actions">
            <button id="egm-select-all-visible-button" type="button" class="secondary-button">Select All Visible</button>
            <button id="egm-clear-selection-button" type="button" class="secondary-button">Clear Selection</button>
            <button id="egm-bulk-promote-button" type="button">Promote Selected</button>
            <button id="egm-bulk-apply-profile-button" type="button" class="secondary-button">Add Selected to First-Test EGM IDs</button>
            <span id="egm-bulk-selection-summary" class="muted-text">0 selected</span>
          </div>
          <span id="egm-bulk-action-message" class="muted-text">Select one or more EGMs to apply bulk actions.</span>
        </div>
        <div class="table-wrap table-wrap-scroll-safe">
          <table>
            <thead>
              <tr>
                <th class="egm-row-select-cell"><input id="egm-select-all-checkbox" type="checkbox" aria-label="Select all visible EGMs"></th>
                <th><button type="button" class="sort-button" data-sort-key="egm_id">EGM</button></th>
                <th>Source</th>
                <th><button type="button" class="sort-button" data-sort-key="status">Status</button></th>
                <th>Configured Address</th>
                <th>Last Endpoint</th>
                <th>Endpoint Drift</th>
                <th>Game</th>
                <th><button type="button" class="sort-button" data-sort-key="last_seen">Last seen</button></th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody id="egm-table">
              <tr><td colspan="10">Loading...</td></tr>
            </tbody>
          </table>
        </div>
        <div id="egm-registry-drawer" class="egm-registry-drawer egm-registry-drawer-hidden">
          <div class="panel-title-row">
            <strong id="egm-registry-drawer-title">EGM Metadata</strong>
            <span id="egm-registry-drawer-state" class="source-pill source-file">ready</span>
          </div>
          <span id="egm-registry-drawer-message" class="muted-text">Select an EGM row to edit registry metadata.</span>
          <form id="egm-registry-form" class="setup-form">
            <div class="form-grid">
              <label>EGM ID<input id="egm-registry-egm-id" type="text" disabled></label>
              <label>Display Name<input id="egm-registry-display-name" type="text" maxlength="120"></label>
              <label>Vendor<input id="egm-registry-vendor" type="text" maxlength="120"></label>
              <label>Cabinet Family<input id="egm-registry-cabinet-family" type="text" maxlength="120"></label>
              <label>Game Title<input id="egm-registry-game-title" type="text" maxlength="160"></label>
              <label>Software Version<input id="egm-registry-software-version" type="text" maxlength="120"></label>
            </div>
            <label class="cert-textarea-label">Notes
              <textarea id="egm-registry-notes" rows="3" maxlength="2000" placeholder="Operator notes for this cabinet"></textarea>
            </label>
            <div class="setup-details">
              <div>
                <p class="label">Override State</p>
                <strong id="egm-registry-override-state">-</strong>
              </div>
              <div>
                <p class="label">Source</p>
                <strong id="egm-registry-source-state">-</strong>
              </div>
            </div>
            <div class="setup-actions">
              <button id="egm-registry-save-button" type="submit">Save Metadata</button>
              <button id="egm-registry-delete-button" type="button" class="secondary-button">Delete Override</button>
              <button id="egm-registry-promote-button" type="button" class="secondary-button">Promote Discovered EGM</button>
              <button id="egm-registry-close-button" type="button" class="secondary-button">Close</button>
            </div>
          </form>
        </div>
      </div>

      <div class="panel endpoint-integrity-panel">
        <div class="panel-head panel-head-stack">
          <div class="panel-title-row">
            <h2>Endpoint Integrity</h2>
            <span id="endpoint-integrity-state" class="source-pill source-file">ready</span>
          </div>
          <span id="endpoint-integrity-summary" class="muted-text">No endpoint collisions detected.</span>
        </div>
        <div class="setup-actions">
          <button id="endpoint-integrity-filter-button" type="button" class="secondary-button">Show Affected EGMs</button>
          <button id="endpoint-integrity-refresh-button" type="button" class="secondary-button">Refresh Alerts</button>
          <label class="endpoint-integrity-custom-snooze-label">Custom Snooze (minutes)<input id="endpoint-integrity-custom-minutes" type="number" min="1" value="30"></label>
        </div>
        <div id="endpoint-integrity-counts" class="muted-text">Active 0 | Suppressed 0</div>
        <div id="endpoint-integrity-message" class="muted-text">Signals only. Endpoint integrity warnings do not block mute path.</div>
        <div class="endpoint-integrity-sections">
          <div class="endpoint-integrity-section">
            <div class="endpoint-integrity-section-head">
              <strong>Active Alerts</strong>
              <span id="endpoint-integrity-active-count" class="source-pill source-mixed">0</span>
            </div>
            <div id="endpoint-integrity-list" class="timeline panel-scroll-safe panel-scroll-safe-integrity"></div>
          </div>
          <div class="endpoint-integrity-section">
            <div class="endpoint-integrity-section-head">
              <strong>Acknowledged</strong>
              <span id="endpoint-integrity-acked-count" class="source-pill source-file">0</span>
            </div>
            <div id="endpoint-integrity-acked-list" class="timeline panel-scroll-safe panel-scroll-safe-integrity"></div>
          </div>
          <div class="endpoint-integrity-section">
            <div class="endpoint-integrity-section-head">
              <strong>Snoozed</strong>
              <span id="endpoint-integrity-snoozed-count" class="source-pill source-override">0</span>
            </div>
            <div id="endpoint-integrity-snoozed-list" class="timeline panel-scroll-safe panel-scroll-safe-integrity"></div>
          </div>
        </div>
      </div>

      <div class="panel cabinet-run-panel">
        <div class="panel-head panel-head-stack">
          <div class="panel-title-row">
            <h2>Cabinet Run Timeline</h2>
            <span id="timeline-count">0 events</span>
          </div>
          <div class="toolbar-row timeline-toolbar">
            <div class="filter-tabs timeline-filter-tabs" role="tablist" aria-label="Cabinet run timeline filter tabs">
              <button type="button" class="timeline-filter-tab is-active" data-timeline-filter="all">All</button>
              <button type="button" class="timeline-filter-tab" data-timeline-filter="incident">Incidents</button>
              <button type="button" class="timeline-filter-tab" data-timeline-filter="egm">EGM</button>
              <button type="button" class="timeline-filter-tab" data-timeline-filter="heartbeat">Heartbeat</button>
              <button type="button" class="timeline-filter-tab" data-timeline-filter="state">State</button>
              <button type="button" class="timeline-filter-tab" data-timeline-filter="marker">Markers</button>
            </div>
            <div class="timeline-rollup-controls">
              <label class="timeline-toggle"><input id="timeline-rollup-heartbeat-toggle" type="checkbox" checked>Roll up heartbeat traffic</label>
              <label class="timeline-toggle"><input id="timeline-show-raw-heartbeat-toggle" type="checkbox">Show raw heartbeat events</label>
            </div>
            <span id="timeline-filter-label" class="muted-text">Showing all timeline events</span>
          </div>
          <span id="timeline-heartbeat-mode-label" class="muted-text">Heartbeat mode: rolled up</span>
          <span id="timeline-grouping-label" class="muted-text">All EGM rows grouped by EGM ID; global rows remain grouped as global.</span>
        </div>
        <form id="run-marker-form" class="setup-form run-marker-form">
          <div class="panel-title-row">
            <strong>Run Markers</strong>
            <span id="run-marker-state" class="source-pill source-file">ready</span>
          </div>
          <span id="run-marker-message" class="muted-text">Mark session start, operator notes, and session end directly into the appliance timeline.</span>
          <div class="form-grid run-marker-grid">
            <label>Marker Title<input id="run-marker-title" name="title" autocomplete="off"></label>
            <label>Operator<input id="run-marker-operator" name="operator" autocomplete="off" placeholder="lab-ui"></label>
          </div>
          <label class="cert-textarea-label run-marker-notes-label">Run Marker Notes
            <textarea id="run-marker-notes" rows="4" placeholder="Optional cabinet notes, attach/detach notes, or operator observations."></textarea>
          </label>
          <div class="setup-actions evidence-actions">
            <button id="run-marker-start-button" type="button">Mark Start</button>
            <button id="run-marker-note-button" type="button" class="secondary-button">Add Note</button>
            <button id="run-marker-end-button" type="button" class="secondary-button">Mark End</button>
          </div>
        </form>
        <form id="run-report-form" class="setup-form run-report-form">
          <div class="panel-title-row">
            <strong>Run Window Report</strong>
            <span id="run-report-state" class="source-pill source-file">ready</span>
          </div>
          <span id="run-report-message" class="muted-text">Pick a start and end marker to export a bounded run report.</span>
          <div class="form-grid run-report-grid">
            <label>Start Marker
              <select id="run-report-start-marker"></select>
            </label>
            <label>End Marker
              <select id="run-report-end-marker"></select>
            </label>
          </div>
          <div class="setup-details run-report-details">
            <div>
              <p class="label">Window</p>
              <strong id="run-report-window-summary">-</strong>
            </div>
            <div>
              <p class="label">Counts</p>
              <strong id="run-report-count-summary">-</strong>
            </div>
          </div>
          <div class="setup-actions evidence-actions">
            <button id="run-report-json-button" type="button">Download Run JSON</button>
            <button id="run-report-markdown-button" type="button" class="secondary-button">Download Run Markdown</button>
          </div>
        </form>
        <form id="heartbeat-policy-form" class="setup-form run-report-form">
          <div class="panel-title-row">
            <strong>Heartbeat Policy</strong>
            <span id="heartbeat-policy-source" class="source-pill source-file">file</span>
          </div>
          <span id="heartbeat-policy-message" class="muted-text">Tune warning and escalation thresholds for heartbeat gap handling.</span>
          <div class="form-grid run-report-grid">
            <label>Interval (ms)<input id="heartbeat-policy-interval" type="number" min="1"></label>
            <label>Updated<input id="heartbeat-policy-updated" type="text" disabled></label>
            <label>Warning After Missed Beats<input id="heartbeat-policy-warning-after-missed" type="number" min="1"></label>
            <label>Escalate Alert After Missed Beats<input id="heartbeat-policy-block-after-missed" type="number" min="1"></label>
          </div>
          <div class="setup-details run-report-details">
            <div>
              <p class="label">Warning Gap</p>
              <strong id="heartbeat-policy-warning-gap">-</strong>
            </div>
            <div>
              <p class="label">Escalation Gap</p>
              <strong id="heartbeat-policy-block-gap">-</strong>
            </div>
          </div>
          <div id="heartbeat-policy-validation-list" class="validation-list"></div>
          <div class="setup-actions evidence-actions">
            <button id="heartbeat-policy-save-button" type="submit">Save Override</button>
            <button id="heartbeat-policy-clear-button" type="button" class="secondary-button">Clear Override</button>
            <button id="heartbeat-policy-reload-button" type="button" class="secondary-button">Reload</button>
          </div>
        </form>
        <div id="blocker-governance-anchor" class="anchor-target" aria-hidden="true"></div>
        <form id="blocker-policy-form" class="setup-form run-report-form blocker-governance-panel">
          <div class="panel-title-row">
            <strong>Blocker Governance</strong>
            <span id="blocker-policy-source" class="source-pill source-file">file</span>
          </div>
          <span id="blocker-policy-message" class="muted-text">Only approved blocker IDs can block runbook readiness.</span>
          <div class="form-grid run-report-grid">
            <label>Updated<input id="blocker-policy-updated" type="text" disabled></label>
            <label>Approved Blocker IDs (comma/newline)<textarea id="blocker-policy-approved-ids" rows="4" placeholder="service_readiness&#10;cabinet_profile"></textarea></label>
          </div>
          <div id="blocker-policy-validation-list" class="validation-list"></div>
          <div class="setup-actions evidence-actions">
            <button id="blocker-policy-save-button" type="submit">Save Override</button>
            <button id="blocker-policy-clear-button" type="button" class="secondary-button">Clear Override</button>
            <button id="blocker-policy-reload-button" type="button" class="secondary-button">Reload</button>
          </div>
          <div id="blocker-policy-summary" class="muted-text blocker-governance-summary">Waiting for blocker governance telemetry.</div>
          <label>Selected Finding ID<input id="blocker-policy-finding-id" type="text" placeholder="cabinet_profile"></label>
          <label>Escalation Rationale<textarea id="blocker-policy-rationale" rows="3" placeholder="Required cabinet deployment reason for blocker escalation approval."></textarea></label>
          <div class="setup-actions evidence-actions">
            <button id="blocker-policy-approve-button" type="button">Approve Selected Finding</button>
            <button id="blocker-policy-revoke-button" type="button" class="secondary-button">Revoke Selected Finding</button>
          </div>
          <div class="first-cabinet-session-blockers-wrap">
            <p class="label">Suggested Escalations</p>
            <div id="blocker-policy-suggestions-list" class="timeline blocker-governance-list"></div>
          </div>
          <div class="first-cabinet-session-blockers-wrap">
            <p class="label">Active Approved Blockers</p>
            <div id="blocker-policy-active-blockers" class="timeline blocker-governance-list"></div>
          </div>
          <div class="first-cabinet-session-blockers-wrap">
            <p class="label">Downgraded to Warning by Policy</p>
            <div id="blocker-policy-downgraded-list" class="timeline blocker-governance-list"></div>
          </div>
          <div class="first-cabinet-session-blockers-wrap">
            <p class="label">Escalation History</p>
            <div id="blocker-policy-history-list" class="timeline blocker-governance-list"></div>
          </div>
        </form>
        <form id="operator-drill-form" class="setup-form operator-drill-form">
          <div class="panel-title-row">
            <strong>Operator Drill</strong>
            <span id="operator-drill-state" class="source-pill source-file">ready</span>
          </div>
          <span id="operator-drill-message" class="muted-text">Drive simulated cabinet session traffic directly from the dashboard.</span>
          <div class="form-grid operator-drill-grid">
            <label>EGM
              <select id="operator-drill-egm-id"></select>
            </label>
            <label>Heartbeat Interval (ms)<input id="operator-drill-interval-ms" type="number" min="250" step="250"></label>
            <label>Burst Count<input id="operator-drill-burst-count" type="number" min="1" max="50" step="1"></label>
            <label>Last Action<input id="operator-drill-last-action" type="text" disabled></label>
          </div>
          <div class="setup-details run-report-details">
            <div>
              <p class="label">Auto Heartbeat</p>
              <strong id="operator-drill-heartbeat-state">-</strong>
            </div>
            <div>
              <p class="label">Last Action At</p>
              <strong id="operator-drill-last-action-at">-</strong>
            </div>
          </div>
          <div class="setup-actions evidence-actions operator-drill-actions">
            <button id="operator-drill-comms-online-button" type="button">Trigger commsOnLine</button>
            <button id="operator-drill-keepalive-button" type="button" class="secondary-button">Send keepAlive</button>
            <button id="operator-drill-burst-button" type="button" class="secondary-button">Send keepAlive Burst</button>
            <button id="operator-drill-resume-button" type="button" class="secondary-button">Resume Auto Heartbeat</button>
            <button id="operator-drill-pause-button" type="button" class="secondary-button">Pause Auto Heartbeat</button>
            <button id="operator-drill-clear-button" type="button" class="secondary-button">Clear Drill State</button>
          </div>
        </form>
        <div class="heartbeat-summary-wrap">
          <p class="label">Heartbeat Summary</p>
          <div class="heartbeat-summary-grid">
            <div><p class="label">Scope</p><strong id="heartbeat-scope">All EGMs</strong></div>
            <div><p class="label">Health</p><strong id="heartbeat-health">-</strong></div>
            <div><p class="label">Observed</p><strong id="heartbeat-observed">-</strong></div>
            <div><p class="label">Last Keepalive</p><strong id="heartbeat-last-keepalive">-</strong></div>
            <div><p class="label">Max Gap</p><strong id="heartbeat-max-gap">-</strong></div>
          </div>
          <div id="heartbeat-summary-message" class="muted-text heartbeat-summary-message">Waiting for heartbeat telemetry.</div>
        </div>
        <div id="cabinet-run-timeline" class="timeline"></div>
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
        <div class="panel-head panel-head-stack">
          <h2>EGM History</h2>
          <span id="egm-history-scope" class="muted-text">Scope: All EGMs</span>
          <span id="egm-history-grouping" class="muted-text">Grouped by EGM ID</span>
        </div>
        <div id="egm-history" class="timeline panel-scroll-safe panel-scroll-safe-history"></div>
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
        <div class="panel-head panel-head-stack">
          <div class="panel-title-row">
            <h2>Operator Audit Timeline</h2>
            <span id="operator-audit-state" class="source-pill source-file">ready</span>
          </div>
          <span id="operator-audit-message" class="muted-text">Sensitive operator actions are recorded here.</span>
        </div>
        <div class="form-grid operator-audit-filters">
          <label>Action
            <select id="operator-audit-action-filter">
              <option value="">All actions</option>
              <option value="cabinet_profile.save">cabinet_profile.save</option>
              <option value="cabinet_profile.clear">cabinet_profile.clear</option>
              <option value="heartbeat_policy.save">heartbeat_policy.save</option>
              <option value="heartbeat_policy.clear">heartbeat_policy.clear</option>
              <option value="blocker_policy.save">blocker_policy.save</option>
              <option value="blocker_policy.clear">blocker_policy.clear</option>
              <option value="certificate.preview">certificate.preview</option>
              <option value="certificate.import">certificate.import</option>
              <option value="certificate.restore">certificate.restore</option>
              <option value="session_workflow.save">session_workflow.save</option>
              <option value="session_workflow.clear">session_workflow.clear</option>
              <option value="session_evidence.delete">session_evidence.delete</option>
              <option value="session_evidence.export_all">session_evidence.export_all</option>
            </select>
          </label>
          <label>Result
            <select id="operator-audit-result-filter">
              <option value="">All results</option>
              <option value="success">success</option>
              <option value="fail">fail</option>
            </select>
          </label>
          <label>Search
            <input id="operator-audit-search-filter" type="text" placeholder="summary or detail">
          </label>
        </div>
        <div id="operator-audit-summary" class="muted-text operator-audit-summary">No operator audit events loaded yet.</div>
        <div id="operator-audit-list" class="timeline operator-audit-list panel-scroll-safe panel-scroll-safe-audit"></div>
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
          <div class="cert-preview-wrap">
            <p class="label">Preview Validation</p>
            <strong id="cert-preview-summary">Run preview before importing certificate material.</strong>
            <div id="cert-preview-detail" class="muted-text cert-preview-detail">Preview is read-only and does not write files.</div>
            <div id="cert-preview-list" class="validation-list cert-preview-list"></div>
          </div>
          <label class="cert-textarea-label">Certificate PEM
            <textarea id="cert-certificate-pem" name="certificate_pem" rows="8" placeholder="-----BEGIN CERTIFICATE-----"></textarea>
          </label>
          <label id="cert-private-key-wrapper" class="cert-textarea-label">Private Key PEM
            <textarea id="cert-private-key-pem" name="private_key_pem" rows="8" placeholder="-----BEGIN PRIVATE KEY-----"></textarea>
          </label>
          <div class="setup-actions cert-actions">
            <button id="cert-preview-button" type="button" class="secondary-button">Preview Certificate</button>
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
        <div class="cert-backup-history">
          <div class="panel-head panel-head-stack">
            <div class="panel-title-row">
              <p class="label">Backup History</p>
              <button id="cert-backup-refresh-button" type="button" class="secondary-button">Refresh Backups</button>
            </div>
            <span id="cert-backup-state" class="muted-text">Loading backup history for selected role.</span>
          </div>
          <div id="cert-backup-list" class="timeline cert-backup-list"></div>
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

  <script src="{{.DashboardJSURL}}"></script>
</body>
</html>`

const dashboardCSS = `:root {
  --ink: #17201b;
  --muted: #65736b;
  --line: #cad7ce;
  --paper: #f4f7f2;
  --panel: #ffffff;
  --panel-pad: 18px;
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

.api-connection-indicator {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 240px;
  padding: 6px 10px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.9);
  font-size: 12px;
}

.api-connection-indicator strong {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.02em;
}

.api-connection-connected {
  border-color: #6fb28a;
  background: #edf9f1;
  color: #1f6d40;
}

.api-connection-attempting {
  border-color: #b7ab75;
  background: #fdf7e5;
  color: #6e5a12;
}

.api-connection-disconnected {
  border-color: #d9aaaa;
  background: #fdf0ef;
  color: #862d29;
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

.operator-toolbar-grid {
  align-items: stretch;
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

.egm-source {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
}

.egm-source-configured {
  background: #d8efe1;
  color: #215a3d;
}

.egm-source-discovered {
  background: #fbe8c4;
  color: #6b4a12;
}

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
  min-width: 0;
}

.panel-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 16px var(--panel-pad);
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

.operator-actions-bar .panel-head,
.global-view-controls-panel .panel-head {
  border-bottom: 0;
}

.operator-actions-buttons,
.global-view-controls {
  padding: 0 var(--panel-pad) 16px;
  margin-top: 0;
}

.global-view-controls {
  grid-template-columns: minmax(180px, 0.28fr) minmax(320px, 0.52fr) minmax(180px, 0.2fr);
}

.global-compact-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-height: 38px;
}

.global-compact-toggle input[type="checkbox"] {
  width: 16px;
  height: 16px;
}

.focus-controls {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px var(--panel-pad) var(--panel-pad);
}

.egm-focus-panel .panel-head {
  border-bottom: 0;
}

.focus-controls label {
  font-size: 12px;
  text-transform: uppercase;
  color: var(--muted);
  min-width: 92px;
}

.focus-controls select {
  min-width: 220px;
  max-width: 100%;
}

.focus-egm-summary-wrap {
  border-top: 1px solid var(--line);
  padding: 12px 16px 16px;
  background: #f8fbf8;
}

.focus-egm-summary-wrap .label {
  margin-bottom: 4px;
}

.egm-grouped-summary .item {
  display: grid;
  gap: 4px;
}

.focus-selected-egm-wrap {
  border-top: 1px solid var(--line);
  padding: 12px 16px 16px;
  background: #f8fbf8;
}

.focus-selected-egm-wrap .label {
  margin-bottom: 4px;
}

.selected-egm-detail .item {
  display: grid;
  gap: 4px;
}

.table-wrap {
  overflow-x: auto;
}

.table-wrap-scroll-safe {
  overflow: auto;
  max-height: 420px;
  border-top: 1px solid var(--line);
}

.table-wrap-scroll-safe thead th {
  position: sticky;
  top: 0;
  z-index: 1;
  background: #f4f8f5;
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

.panel-scroll-safe {
  max-height: 380px;
  overflow-y: auto;
  overflow-x: hidden;
  border-top: 1px solid var(--line);
}

.panel-scroll-safe-integrity {
  max-height: 320px;
}

.panel-scroll-safe-audit {
  max-height: 460px;
}

.panel-scroll-safe-history {
  max-height: 420px;
}

.item {
  padding: 14px var(--panel-pad);
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

.timeline-entry {
  display: grid;
  gap: 6px;
}

.timeline-entry-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.timeline-entry-tags {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.timeline-egm-chip {
  display: inline-flex;
  align-items: center;
  min-height: 20px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  background: #e8f1fa;
  color: #1d507b;
}

.timeline-scope {
  display: inline-flex;
  align-items: center;
  min-height: 20px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
}

.timeline-scope-global {
  background: #e8edf0;
  color: #496171;
}

.timeline-kind {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 72px;
  min-height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
}

.timeline-kind-incident {
  background: var(--red-bg);
  color: #7b2d2a;
}

.timeline-kind-egm {
  background: var(--info-bg);
  color: #245f91;
}

.timeline-kind-heartbeat {
  background: #eef3ff;
  color: #35528f;
}

.timeline-kind-state {
  background: var(--green-bg);
  color: #1e6c47;
}

.timeline-kind-marker {
  background: #f3ebff;
  color: #5b3f91;
}

.timeline-group-heading {
  border-top: 1px solid var(--line);
  padding: 8px 12px;
  background: #f4f8f5;
  color: var(--muted);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.02em;
}

.timeline-group-heading-egm {
  border-left: 3px solid #9dbac2;
}

.timeline-group-heading-global {
  border-left: 3px solid #7f9aaa;
}

.cabinet-run-panel {
  grid-column: 1 / -1;
}

.cabinet-run-panel .panel-title-row,
.cabinet-run-panel .toolbar-row,
.cabinet-run-panel .setup-form,
.cabinet-run-panel .form-grid,
.cabinet-run-panel label,
.cabinet-run-panel textarea,
.cabinet-run-panel input {
  min-width: 0;
}

.timeline-filter-tabs {
  flex-wrap: wrap;
  overflow: visible;
}

.timeline-filter-tabs .timeline-filter-tab {
  flex: 0 0 auto;
}

.timeline-toolbar {
  min-width: 0;
}

.timeline-rollup-controls {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

.timeline-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--muted);
  text-transform: uppercase;
  font-weight: 700;
}

.timeline-toggle input[type="checkbox"] {
  width: 14px;
  height: 14px;
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
  padding: 13px var(--panel-pad);
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
  padding: var(--panel-pad);
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

.cert-preview-wrap {
  display: grid;
  gap: 6px;
  padding: 10px 12px;
  border: 1px solid var(--line);
  background: #f8fbf8;
}

.cert-preview-detail {
  font-size: 13px;
}

.cert-preview-list {
  margin-top: 0;
}

.cert-manager-details {
  margin-top: 2px;
}

.cert-manager-detail {
  padding: 0 18px 18px;
  overflow-wrap: anywhere;
}

.run-marker-form {
  display: grid;
  gap: 12px;
  border-top: 1px solid var(--line);
}

.run-marker-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.run-marker-notes-label {
  padding: 0 0 8px;
}

.run-report-form {
  display: grid;
  gap: 12px;
  border-top: 1px solid var(--line);
}

.blocker-governance-panel {
  border-top: 1px solid var(--line);
}

.blocker-governance-summary {
  padding: 0 2px;
}

.blocker-governance-list {
  padding: 0;
}

.blocker-governance-item {
  display: grid;
  gap: 4px;
}

.blocker-governance-actions {
  margin-top: 2px;
}

.operator-drill-form {
  display: grid;
  gap: 12px;
  border-top: 1px solid var(--line);
}

.operator-drill-grid {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.operator-drill-actions {
  margin-top: 0;
}

.heartbeat-summary-wrap {
  border-top: 1px solid var(--line);
  padding: 12px 16px 16px;
  background: #f8fbf8;
}

.heartbeat-summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  margin-top: 8px;
  background: var(--line);
}

.heartbeat-summary-grid > div {
  min-width: 0;
  padding: 13px 14px;
  background: var(--panel);
}

.heartbeat-summary-grid strong {
  display: block;
  overflow-wrap: anywhere;
  font-size: 14px;
}

.heartbeat-summary-message {
  display: block;
  margin-top: 10px;
}

.run-report-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.run-report-details {
  margin-top: 0;
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

.cert-backup-history {
  padding: 0 18px 12px;
}

.cert-backup-list {
  margin-top: 6px;
}

.cert-backup-item {
  display: grid;
  gap: 6px;
}

.cert-backup-item-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 10px;
}

.cert-backup-meta {
  color: var(--muted);
  font-size: 12px;
  overflow-wrap: anywhere;
}

.cert-backup-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.operator-audit-filters {
  grid-template-columns: minmax(200px, 0.35fr) minmax(140px, 0.2fr) minmax(280px, 0.45fr);
  padding: 14px 18px 10px;
}

.operator-audit-summary {
  padding: 0 var(--panel-pad) 10px;
}

.operator-audit-list {
  padding: 0 var(--panel-pad) var(--panel-pad);
}

.operator-audit-entry details {
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 8px 10px;
  background: #fbfdfb;
}

.operator-audit-entry summary {
  cursor: pointer;
  list-style: none;
}

.operator-audit-entry summary::-webkit-details-marker {
  display: none;
}

.operator-audit-head {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  align-items: baseline;
}

.operator-audit-meta {
  color: var(--muted);
  font-size: 12px;
  margin-top: 6px;
  overflow-wrap: anywhere;
}

.operator-audit-detail {
  margin-top: 8px;
  color: var(--ink);
  font-size: 13px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.operator-audit-pill-success {
  color: var(--green);
  font-weight: 700;
}

.operator-audit-pill-fail {
  color: var(--red);
  font-weight: 700;
}

.endpoint-integrity-panel {
  border-color: rgba(245, 158, 11, 0.35);
}

.egm-registry-drawer {
  border-top: 1px solid var(--line);
  background: #f8fbf8;
  padding: 12px 16px 16px;
}

.egm-registry-drawer-hidden {
  display: none;
}

.egm-row-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.egm-row-actions button {
  min-height: 28px;
  padding: 0 10px;
  font-size: 12px;
}

.egm-bulk-actions {
  align-items: center;
  margin-top: 8px;
}

.egm-row-select-cell {
  width: 38px;
  text-align: center;
}

.egm-row-select-checkbox,
#egm-select-all-checkbox {
  inline-size: 16px;
  block-size: 16px;
}

.endpoint-integrity-sections {
  display: grid;
  gap: 10px;
}

.endpoint-integrity-section {
  border: 1px solid var(--line);
  background: #fff;
}

.endpoint-integrity-section-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--line);
}

.endpoint-integrity-custom-snooze-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--muted);
}

.endpoint-integrity-custom-snooze-label input {
  width: 90px;
  min-height: 30px;
}

.endpoint-integrity-warning {
  border-left: 3px solid #f59e0b;
  padding-left: 0.5rem;
}

.endpoint-integrity-warning-head {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  align-items: baseline;
}

.endpoint-integrity-warning-meta {
  color: var(--muted);
  font-size: 12px;
}

.endpoint-integrity-warning-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

.endpoint-integrity-warning-actions button {
  min-height: 28px;
  padding: 0 10px;
  font-size: 12px;
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

.runtime-overrides-upload-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--line);
  background: #fff;
  color: var(--muted);
  border-radius: 6px;
  font-size: 13px;
}

.runtime-overrides-upload-label input[type="file"] {
  max-inline-size: 240px;
}

.runtime-preset-item {
  border: 1px solid var(--line);
  background: #fff;
  padding: 10px 12px;
  margin-bottom: 8px;
}

.runtime-preset-item-head {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  align-items: baseline;
}

.runtime-preset-item-meta {
  color: var(--muted);
  font-size: 12px;
  margin-top: 6px;
}

.runtime-preset-item-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
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

.first-cabinet-session-workflow-wrap {
  border-top: 1px solid var(--line);
  padding: 12px 16px 16px;
  background: #f8fbf8;
}

.first-cabinet-session-workflow-progress-wrap {
  border-top: 1px solid var(--line);
  padding: 12px 16px 16px;
  background: #f8fbf8;
  display: grid;
  gap: 10px;
}

.workflow-progress-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 13px;
  color: var(--muted);
}

.workflow-progress-unsaved {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
}

.workflow-progress-unsaved-clean {
  background: var(--green-bg);
  color: #1e6c47;
}

.workflow-progress-unsaved-dirty {
  background: #fbeecd;
  color: #7b5b0e;
}

.workflow-progress-grid {
  display: grid;
  grid-template-columns: repeat(1, minmax(0, 1fr));
  gap: 8px;
}

.workflow-progress-steps {
  display: grid;
  gap: 6px;
}

.workflow-progress-step {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.workflow-progress-step input[type="checkbox"] {
  width: 16px;
  height: 16px;
}

.first-cabinet-session-actions-wrap {
  border-top: 1px solid var(--line);
  padding: 12px 16px 16px;
  background: #f8fbf8;
}

.mute-path-status-wrap {
  border-top: 1px solid var(--line);
  padding: 12px 16px 16px;
  background: #f8fbf8;
}

.mute-path-summary-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 8px;
}

.mute-path-status-card,
.runbook-readiness-status-card {
  min-height: 88px;
}

.mute-path-status-card strong,
.runbook-readiness-status-card strong {
  font-size: 14px;
}

#mute-path-prep-status {
  font-size: 13px;
}

.operator-action-summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 10px;
}

.operator-action-summary-grid > div {
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 8px 10px;
  background: #fff;
}

.operator-readiness-model {
  display: grid;
  gap: 8px;
}

.operator-readiness-group {
  border-left: 3px solid #8fa1aa;
  padding: 8px 10px;
  background: #f3f7f4;
  display: grid;
  gap: 4px;
}

.operator-readiness-group.group-ready_now {
  border-left-color: var(--green);
  background: var(--green-bg);
}

.operator-readiness-group.group-needs_operator_action {
  border-left-color: var(--red);
  background: var(--red-bg);
}

.operator-readiness-group.group-lab_warning {
  border-left-color: var(--yellow);
  background: #fbeecd;
}

.operator-readiness-group.group-informational {
  border-left-color: #7f9aaa;
  background: #eef3f6;
}

.operator-readiness-items {
  margin: 0;
  padding-left: 16px;
  display: grid;
  gap: 4px;
}

.operator-readiness-items li {
  font-size: 13px;
  color: var(--ink);
}

.first-cabinet-session-workflow {
  display: grid;
  gap: 8px;
}

.first-cabinet-session-workflow-step {
  border-left: 3px solid #8fa1aa;
  padding: 8px 10px;
  background: #f3f7f4;
  display: grid;
  gap: 4px;
}

.first-cabinet-session-workflow-step.step-complete {
  border-left-color: var(--green);
  background: var(--green-bg);
}

.first-cabinet-session-workflow-step.step-active {
  border-left-color: #245f91;
  background: var(--info-bg);
}

.first-cabinet-session-workflow-step.step-action_needed {
  border-left-color: var(--yellow);
  background: #fbeecd;
}

.first-cabinet-session-workflow-step .step-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.first-cabinet-session-workflow-step .step-state {
  text-transform: uppercase;
  font-size: 11px;
  font-weight: 700;
  color: var(--muted);
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
  padding: var(--panel-pad);
  color: var(--muted);
  background: var(--panel);
}

.anchor-target {
  display: block;
  position: relative;
  top: -8px;
  visibility: hidden;
}

body.compact-mode .shell {
  padding-top: 18px;
}

body.compact-mode .panel-head {
  padding: 12px 14px;
}

body.compact-mode .setup-form,
body.compact-mode .item,
body.compact-mode .kv-list div,
body.compact-mode th,
body.compact-mode td {
  padding: 10px 12px;
}

body.compact-mode .setup-actions {
  gap: 8px;
}

body.compact-mode .panel-scroll-safe {
  max-height: 320px;
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

  .focus-controls {
    align-items: flex-start;
    flex-direction: column;
  }

  .operator-action-summary-grid {
    grid-template-columns: 1fr 1fr;
  }

  .global-view-controls {
    grid-template-columns: 1fr;
    padding-top: 0;
  }

  .mute-path-summary-grid {
    grid-template-columns: 1fr;
  }

  .form-grid,
  .setup-details {
    grid-template-columns: 1fr;
  }
}`

const dashboardJS = `const endpoints = {
  status: "/api/status",
  readyz: "/readyz",
  incidents: "/api/incidents?limit=20",
  egmHistory: "/api/egms/history",
  egmRegistry: "/api/egm-registry",
  egmRegistryPromote: "/api/egm-registry/promote",
  egmRegistryPromoteBulk: "/api/egm-registry/promote-bulk",
  egmRegistryApplyToCabinetProfile: "/api/egm-registry/apply-to-cabinet-profile",
  stateHistory: "/api/state-history?limit=30",
  runMarkers: "/api/run-markers?limit=30",
  operatorDrill: "/api/operator-drill",
  certificates: "/api/certificates",
  operatorAudit: "/api/operator-audit",
  sessionPackageExport: "/api/session-package/export",
  runtimeOverridesSnapshot: "/api/runtime-overrides/snapshot",
  runtimeOverridesRestore: "/api/runtime-overrides/restore",
  runtimeOverridePresets: "/api/runtime-overrides/presets",
  runtimeOverridePresetSave: "/api/runtime-overrides/presets/save",
  runtimeOverridePresetLoad: "/api/runtime-overrides/presets/load",
  sessionEvidence: "/api/session-evidence?limit=20",
  sessionEvidenceExportAll: "/api/session-evidence/export-all",
  sessionWorkflow: "/api/session-workflow",
  cabinetProfile: "/api/cabinet-profile",
  cabinetProfileSuggestions: "/api/cabinet-profile/suggestions",
  heartbeatPolicy: "/api/heartbeat-policy",
  blockerPolicy: "/api/blocker-policy",
  blockerPolicySuggestions: "/api/blocker-policy/suggestions",
  blockerPolicyApprove: "/api/blocker-policy/approve",
  blockerPolicyRevoke: "/api/blocker-policy/revoke",
  endpointIntegrityAlerts: "/api/endpoint-integrity/alerts",
  cabinetPreflight: "/api/cabinet-preflight",
  certificateBackups: "/api/certificates/backups",
  certificateRestore: "/api/certificates/restore",
  certificatePreview: "/api/certificates/preview",
  certificateImport: "/api/certificates/import",
  certificateExport: "/api/certificates/export"
};

const $ = (id) => document.getElementById(id);
const unhealthyStates = new Set(["RED", "GREY"]);
const healthyStates = new Set(["GREEN", "YELLOW"]);
const heartbeatEventTypes = new Set(["G2S_SESSION_ONLINE", "G2S_KEEPALIVE"]);
const compactModeStorageKey = "g2s.dashboard.compact_mode";

const clientState = {
  lastGoodStatus: null,
  lastGoodAt: 0,
  lastStatusPollAt: 0,
  pollAttempts: 0,
  pollStarted: false,
  jsLoadedAt: new Date().toISOString(),
  lastError: "",
  inFlight: false,
  pollIntervalMs: 3000,
  backoffStep: 0,
  timerId: null,
  displaySnapshot: null,
  egmFocusID: "",
  egmSortKey: "egm_id",
  egmSortDir: "asc",
  egmFilter: "all",
  timelineFilter: "all",
  timelineRollupHeartbeat: true,
  timelineShowRawHeartbeat: false,
  certSelectedRole: "g2s_ca_cert",
  certPreviewFingerprint: "",
  certPreviewResult: null,
  certBackupsByRole: {},
  operatorAuditActionFilter: "",
  operatorAuditResultFilter: "",
  operatorAuditSearchFilter: "",
  globalSeverityFilter: "all",
  globalTextFilter: "",
  compactMode: false,
  selectedSessionEvidenceID: 0,
  selectedRunReportStartID: 0,
  selectedRunReportEndID: 0,
  selectedEGMRegistryID: "",
  egmBulkSelection: {},
  egmVisibleRowIDs: [],
  workflowProgressLoaded: false,
  workflowProgressBaseline: null,
  endpointIntegrityActionInFlight: false
};

function currentRuntime() {
  return clientState.displaySnapshot?.status?.runtime || clientState.lastGoodStatus?.status?.runtime || {};
}

function currentHeartbeatPolicy(snapshot) {
  const source = snapshot?.heartbeatPolicy?.effective || snapshot?.status?.heartbeat_policy || null;
  const runtime = snapshot?.status?.runtime || currentRuntime();
  return {
    interval_ms: Number(source?.interval_ms || runtime.egm_heartbeat_interval_ms || 0),
    warning_after_missed: Number(source?.warning_after_missed || 3),
    block_after_missed: Number(source?.block_after_missed || 6)
  };
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
    runMarkers: [],
    operatorDrill: null,
    egmRegistry: null,
    certificates: [],
    operatorAudit: [],
    sessionEvidence: [],
    sessionWorkflow: null,
    cabinetProfile: null,
    cabinetProfileSuggestions: null,
    heartbeatPolicy: null,
    runtimeOverridePresets: null,
    blockerPolicy: null,
    blockerPolicySuggestions: null,
    cabinetPreflight: null,
    endpointIntegrityAlerts: null
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
    runMarkers: Array.isArray(snapshot.runMarkers) ? snapshot.runMarkers.slice() : [],
    operatorDrill: snapshot.operatorDrill || null,
    egmRegistry: snapshot.egmRegistry || null,
    certificates: Array.isArray(snapshot.certificates) ? snapshot.certificates.slice() : [],
    operatorAudit: Array.isArray(snapshot.operatorAudit) ? snapshot.operatorAudit.slice() : [],
    sessionEvidence: Array.isArray(snapshot.sessionEvidence) ? snapshot.sessionEvidence.slice() : [],
    sessionWorkflow: snapshot.sessionWorkflow || null,
    cabinetProfile: snapshot.cabinetProfile || null,
    cabinetProfileSuggestions: snapshot.cabinetProfileSuggestions || null,
    heartbeatPolicy: snapshot.heartbeatPolicy || null,
    runtimeOverridePresets: snapshot.runtimeOverridePresets || null,
    blockerPolicy: snapshot.blockerPolicy || null,
    blockerPolicySuggestions: snapshot.blockerPolicySuggestions || null,
    cabinetPreflight: snapshot.cabinetPreflight || null,
    endpointIntegrityAlerts: snapshot.endpointIntegrityAlerts || null
  };
}

function currentEGMFocusID() {
  return String(clientState.egmFocusID || "").trim();
}

function currentEGMFocusLabel() {
  const focusID = currentEGMFocusID();
  return focusID || "All EGMs";
}

function effectiveHeartbeatRollupEnabled() {
  return clientState.timelineRollupHeartbeat === true && clientState.timelineShowRawHeartbeat !== true;
}

function egmHistoryEndpointURL() {
  const params = new URLSearchParams();
  params.set("limit", "30");
  if (effectiveHeartbeatRollupEnabled()) {
    params.set("rollup_heartbeat", "true");
  }
  return endpoints.egmHistory + "?" + params.toString();
}

function normalizeHeartbeatRollupCount(row) {
  const count = Number(row?.heartbeat_rollup_count || 0);
  if (row?.heartbeat_rollup === true) {
    return count > 0 ? count : 1;
  }
  return 1;
}

function historyRowFirstSeenAt(row) {
  const first = row?.heartbeat_rollup_first_seen_at;
  return first || row?.created_at || "";
}

function historyRowLastSeenAt(row) {
  const last = row?.heartbeat_rollup_last_seen_at;
  return last || row?.created_at || "";
}

function historyRowDurationMS(row) {
  const first = numericTime(historyRowFirstSeenAt(row));
  const last = numericTime(historyRowLastSeenAt(row));
  if (first <= 0 || last <= 0) return 0;
  return Math.max(0, last - first);
}

function historyRowIsKeepAliveRollup(row) {
  return row?.heartbeat_rollup === true && String(row?.event_type || "").toUpperCase() === "G2S_KEEPALIVE";
}

function normalizeGlobalSeverityFilter(value) {
  const normalized = String(value || "all").trim().toLowerCase();
  if (normalized === "alerts" || normalized === "warnings" || normalized === "healthy") {
    return normalized;
  }
  return "all";
}

function currentGlobalTextFilter() {
  return String(clientState.globalTextFilter || "").trim().toLowerCase();
}

function globalSeverityAllows(bucket) {
  const selected = normalizeGlobalSeverityFilter(clientState.globalSeverityFilter);
  if (selected === "all") return true;
  return String(bucket || "").trim().toLowerCase() === selected;
}

function rowMatchesGlobalText(parts) {
  const query = currentGlobalTextFilter();
  if (!query) return true;
  const joined = (Array.isArray(parts) ? parts : [])
    .map((item) => String(item || "").toLowerCase())
    .join(" ");
  return joined.includes(query);
}

function egmSeverityBucket(egm) {
  const status = String(egm?.status || "").toUpperCase();
  if (status === "RED" || status === "GREY") return "alerts";
  if (status === "GREEN") return "healthy";
  if (status === "YELLOW") return "warnings";
  return "warnings";
}

function endpointIntegritySeverityBucket(row) {
  const collisionType = String(row?.type || row?.collision_type || "").toUpperCase();
  if (collisionType === "SHARED_ENDPOINT") return "alerts";
  if (collisionType === "ID_ENDPOINT_DRIFT") return "warnings";
  return "warnings";
}

function operatorAuditSeverityBucket(row) {
  return String(row?.result || "").toLowerCase() === "success" ? "healthy" : "alerts";
}

function egmMatchesGlobalFilters(egm) {
  const driftIPs = Array.isArray(egm?.endpoint_drift_ips) ? egm.endpoint_drift_ips.join(" ") : "";
  const collisionTypes = Array.isArray(egm?.endpoint_collision_types) ? egm.endpoint_collision_types.join(" ") : "";
  return globalSeverityAllows(egmSeverityBucket(egm)) && rowMatchesGlobalText([
    egm?.id,
    egm?.display_name,
    egm?.source,
    egm?.status,
    egm?.vendor,
    egm?.cabinet_family,
    egm?.ip_address,
    egm?.port,
    egm?.last_endpoint_ip,
    egm?.last_endpoint_port,
    egm?.game_title,
    egm?.software_version,
    egm?.notes,
    driftIPs,
    collisionTypes
  ]);
}

function endpointIntegrityMatchesGlobalFilters(row) {
  const ids = Array.isArray(row?.egm_ids)
    ? row.egm_ids.join(" ")
    : Array.isArray(row?.involved_egm_ids)
      ? row.involved_egm_ids.join(" ")
      : "";
  return globalSeverityAllows(endpointIntegritySeverityBucket(row)) && rowMatchesGlobalText([
    row?.type || row?.collision_type,
    endpointCollisionTypeLabel(row?.type || row?.collision_type),
    row?.severity,
    row?.endpoint,
    ids
  ]);
}

function operatorAuditMatchesGlobalFilters(row) {
  return globalSeverityAllows(operatorAuditSeverityBucket(row)) && rowMatchesGlobalText([
    row?.action,
    row?.result,
    row?.summary,
    row?.detail,
    row?.actor_scope,
    row?.egm_focus
  ]);
}

function compactModeLabel() {
  return clientState.compactMode ? "on" : "off";
}

function applyCompactModeClass() {
  document.body.classList.toggle("compact-mode", clientState.compactMode === true);
}

function loadCompactModePreference() {
  try {
    const value = window.localStorage ? window.localStorage.getItem(compactModeStorageKey) : "";
    clientState.compactMode = value === "1";
  } catch (_) {
    clientState.compactMode = false;
  }
  applyCompactModeClass();
}

function saveCompactModePreference() {
  try {
    if (!window.localStorage) return;
    window.localStorage.setItem(compactModeStorageKey, clientState.compactMode ? "1" : "0");
  } catch (_) {
    // Keep compact mode runtime-only when localStorage is unavailable.
  }
}

function renderGlobalViewControls() {
  const severity = normalizeGlobalSeverityFilter(clientState.globalSeverityFilter);
  const text = String(clientState.globalTextFilter || "").trim();
  $("global-severity-filter").value = severity;
  $("global-text-filter").value = text;
  $("global-compact-mode-toggle").checked = clientState.compactMode === true;
  const severityLabelMap = { all: "All", alerts: "Alerts", warnings: "Warnings", healthy: "Healthy" };
  const textLabel = text ? text : "none";
  $("global-view-summary").textContent = "Severity: " + (severityLabelMap[severity] || "All") +
    " | Text: " + textLabel +
    " | Compact mode: " + compactModeLabel();
}

function refreshGlobalFilteredViews() {
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  renderGlobalViewControls();
  renderEGMTable(snapshot?.status || {});
  renderEndpointIntegrity(snapshot);
  renderOperatorAuditTimeline(snapshot);
}

function setCompactMode(enabled) {
  clientState.compactMode = enabled === true;
  applyCompactModeClass();
  saveCompactModePreference();
  renderGlobalViewControls();
}

function setGlobalSeverityFilter(value) {
  clientState.globalSeverityFilter = normalizeGlobalSeverityFilter(value);
  refreshGlobalFilteredViews();
}

function setGlobalTextFilter(value) {
  clientState.globalTextFilter = String(value || "").trim();
  refreshGlobalFilteredViews();
}

function withEGMFocusHeader(headers) {
  const base = headers && typeof headers === "object" ? Object.assign({}, headers) : {};
  const focusID = currentEGMFocusID();
  if (focusID) {
    base["X-EGM-Focus"] = focusID;
  }
  return base;
}

function uniqueEGMIDsFromSnapshot(snapshot) {
  const ids = new Set();
  const statusEGMs = Array.isArray(snapshot?.status?.egms) ? snapshot.status.egms : [];
  statusEGMs.forEach((egm) => {
    const id = String(egm?.id || "").trim();
    if (id) ids.add(id);
  });
  const history = Array.isArray(snapshot?.egmHistory) ? snapshot.egmHistory : [];
  history.forEach((item) => {
    const id = String(item?.egm_id || "").trim();
    if (id) ids.add(id);
  });
  const drillIDs = Array.isArray(snapshot?.operatorDrill?.available_egm_ids) ? snapshot.operatorDrill.available_egm_ids : [];
  drillIDs.forEach((item) => {
    const id = String(item || "").trim();
    if (id) ids.add(id);
  });
  if (currentEGMFocusID()) {
    ids.add(currentEGMFocusID());
  }
  return Array.from(ids).sort((a, b) => a.localeCompare(b));
}

function filterRecordsByFocusedEGM(records, fieldName) {
  const list = Array.isArray(records) ? records : [];
  const focusID = currentEGMFocusID();
  if (!focusID) {
    return list.slice();
  }
  return list.filter((item) => String(item?.[fieldName] || "").trim() === focusID);
}

function filterStatusEGMsByFocus(egms) {
  return filterRecordsByFocusedEGM(egms, "id");
}

function filterHistoryByFocus(history) {
  return filterRecordsByFocusedEGM(history, "egm_id");
}

function egmFocusScope(snapshot) {
  const focusID = currentEGMFocusID();
  return {
    mode: focusID ? "SINGLE_EGM" : "ALL_EGMS",
    selected_egm_id: focusID,
    label: currentEGMFocusLabel(),
    available_egm_ids: uniqueEGMIDsFromSnapshot(snapshot),
    egm_specific_views_filtered: focusID !== "",
    global_sections_included: true
  };
}

function renderEGMFocusControl(snapshot) {
  const select = $("egm-focus-select");
  const focus = egmFocusScope(snapshot);
  const telemetry = telemetryReadinessForSnapshot(snapshot);
  const options = ["<option value=\"\">All EGMs</option>"]
    .concat(focus.available_egm_ids.map((id) => "<option value=\"" + escapeHTML(id) + "\">" + escapeHTML(id) + "</option>"));
  select.innerHTML = options.join("");
  select.value = focus.selected_egm_id;
  $("egm-focus-summary").textContent = focus.selected_egm_id ? ("Focused: " + focus.selected_egm_id) : "All EGMs";
  if (focus.selected_egm_id && telemetry.selected_not_active_warning) {
    $("egm-focus-description").textContent = "Selected EGM not active; global telemetry is healthy. Choose an active EGM or switch to All EGMs.";
  } else {
    $("egm-focus-description").textContent = focus.selected_egm_id
      ? ("EGM-specific sections are filtered to " + focus.selected_egm_id + "; global sections remain visible.")
      : "Use this to scope EGM-specific views while keeping global context visible.";
  }
  $("egm-history-scope").textContent = "Scope: " + focus.label;
  $("egm-history-grouping").textContent = focus.selected_egm_id
    ? "Single EGM rows with global context retained in timeline"
    : "Grouped by EGM ID";
}

function numericTime(value) {
  if (!value) return 0;
  const ts = new Date(value).getTime();
  return Number.isFinite(ts) ? ts : 0;
}

function buildEGMGroupedSummaryRows(statusEGMs, historyRecords, policy, referenceTime) {
  const rowsByID = {};
  const statusList = Array.isArray(statusEGMs) ? statusEGMs : [];
  const historyList = Array.isArray(historyRecords) ? historyRecords : [];
  const policySnapshot = policy || {};
  const reference = referenceTime || new Date().toISOString();

  const ensure = (egmID) => {
    const id = String(egmID || "").trim();
    if (!id) return null;
    if (!rowsByID[id]) {
      rowsByID[id] = {
        egm_id: id,
        source: "",
        status: "",
        last_seen_at: "",
        last_endpoint_ip: "",
        last_endpoint_port: 0,
        last_endpoint_seen_at: "",
        endpoint_drift_warning: false,
        endpoint_drift_ips: [],
        endpoint_collision_warning: false,
        endpoint_collision_types: [],
        recent_endpoints: [],
        total_events: 0,
        non_heartbeat_events: 0,
        heartbeat_records: [],
        last_history_status: ""
      };
    }
    return rowsByID[id];
  };

  statusList.forEach((egm) => {
    const row = ensure(egm?.id);
    if (!row) return;
    row.source = String(egm?.source || row.source || "CONFIGURED").toUpperCase();
    row.status = String(egm?.status || row.status || "").toUpperCase();
    const seen = String(egm?.last_seen || "").trim();
    if (numericTime(seen) > numericTime(row.last_seen_at)) {
      row.last_seen_at = seen;
    }
    row.last_endpoint_ip = String(egm?.last_endpoint_ip || row.last_endpoint_ip || "").trim();
    row.last_endpoint_port = Number(egm?.last_endpoint_port || row.last_endpoint_port || 0);
    const endpointSeen = String(egm?.last_endpoint_seen_at || "").trim();
    if (numericTime(endpointSeen) > numericTime(row.last_endpoint_seen_at)) {
      row.last_endpoint_seen_at = endpointSeen;
    }
    row.endpoint_drift_warning = egm?.endpoint_drift_warning === true;
    row.endpoint_drift_ips = Array.isArray(egm?.endpoint_drift_ips) ? egm.endpoint_drift_ips.slice() : [];
    row.endpoint_collision_warning = egm?.endpoint_collision_warning === true;
    row.endpoint_collision_types = Array.isArray(egm?.endpoint_collision_types)
      ? egm.endpoint_collision_types.map((item) => String(item || "").toUpperCase()).filter(Boolean)
      : [];
    row.recent_endpoints = Array.isArray(egm?.recent_endpoints) ? egm.recent_endpoints.slice() : [];
  });

  historyList.forEach((record) => {
    const row = ensure(record?.egm_id);
    if (!row) return;
    const eventWeight = isHeartbeatEventType(record?.event_type)
      ? normalizeHeartbeatRollupCount(record)
      : 1;
    row.total_events += eventWeight;
    if (isHeartbeatEventType(record?.event_type)) {
      row.heartbeat_records.push(record);
    } else {
      row.non_heartbeat_events += 1;
      if (record?.status) {
        row.last_history_status = String(record.status).toUpperCase();
      }
    }
    const seen = String(record?.created_at || "").trim();
    if (numericTime(seen) > numericTime(row.last_seen_at)) {
      row.last_seen_at = seen;
    }
    if (!row.source) {
      row.source = "DISCOVERED";
    }
  });

  return Object.keys(rowsByID)
    .sort((a, b) => a.localeCompare(b))
    .map((id) => {
      const row = rowsByID[id];
      const heartbeat = heartbeatSummary(row.heartbeat_records, policySnapshot, reference);
      return {
        egm_id: row.egm_id,
        source: row.source || "DISCOVERED",
        status: row.status || row.last_history_status || "UNKNOWN",
        last_seen_at: row.last_seen_at || "",
        last_endpoint_ip: row.last_endpoint_ip || "",
        last_endpoint_port: row.last_endpoint_port || 0,
        last_endpoint_seen_at: row.last_endpoint_seen_at || "",
        endpoint_drift_warning: row.endpoint_drift_warning === true,
        endpoint_drift_ips: Array.isArray(row.endpoint_drift_ips) ? row.endpoint_drift_ips.slice() : [],
        endpoint_collision_warning: row.endpoint_collision_warning === true,
        endpoint_collision_types: Array.isArray(row.endpoint_collision_types) ? row.endpoint_collision_types.slice() : [],
        recent_endpoints: Array.isArray(row.recent_endpoints) ? row.recent_endpoints.slice() : [],
        total_events: row.total_events,
        non_heartbeat_events: row.non_heartbeat_events,
        heartbeat_events: heartbeat.total,
        keepalive_events: heartbeat.keepalive_count,
        heartbeat_health: heartbeat.health,
        heartbeat_label: heartbeat.label,
        heartbeat_last_keepalive_at: heartbeat.last_keepalive_at || "",
        heartbeat: heartbeat
      };
    });
}

function groupedSummaryRowsForSnapshot(snapshot) {
  const statusEGMs = Array.isArray(snapshot?.status?.egms) ? snapshot.status.egms : [];
  const history = Array.isArray(snapshot?.egmHistory) ? snapshot.egmHistory : [];
  return buildEGMGroupedSummaryRows(statusEGMs, history, currentHeartbeatPolicy(snapshot), new Date().toISOString());
}

function groupedRowsForCurrentFocus(rows) {
  const list = Array.isArray(rows) ? rows : [];
  const focusID = currentEGMFocusID();
  if (!focusID) {
    return list;
  }
  return list.filter((item) => String(item?.egm_id || "").trim() === focusID);
}

function telemetryRecentThresholdMS(snapshot) {
  const policy = currentHeartbeatPolicy(snapshot);
  const intervalMS = Math.max(0, Number(policy?.interval_ms || 0));
  return Math.max(intervalMS > 0 ? intervalMS * 3 : 0, 30000);
}

function rowHasRecentTelemetry(row, thresholdMS, referenceMS) {
  const lastSeenMS = numericTime(row?.last_seen_at);
  if (lastSeenMS <= 0) return false;
  const nowMS = Number.isFinite(referenceMS) ? referenceMS : Date.now();
  return Math.max(0, nowMS - lastSeenMS) <= Math.max(0, Number(thresholdMS || 0));
}

function telemetryReadinessForSnapshot(snapshot) {
  const groupedRows = groupedSummaryRowsForSnapshot(snapshot);
  const focus = egmFocusScope(snapshot);
  const thresholdMS = telemetryRecentThresholdMS(snapshot);
  const referenceMS = Date.now();
  const recentRows = groupedRows.filter((row) => rowHasRecentTelemetry(row, thresholdMS, referenceMS));
  const focusedRow = focus.selected_egm_id
    ? groupedRows.find((row) => String(row?.egm_id || "").trim() === focus.selected_egm_id) || null
    : null;
  const focusedRecent = focusedRow ? rowHasRecentTelemetry(focusedRow, thresholdMS, referenceMS) : false;
  const selectedMissing = !!focus.selected_egm_id && !focusedRow;
  const selectedStale = !!focus.selected_egm_id && !!focusedRow && !focusedRecent;
  const anyRecentGlobal = recentRows.length > 0;
  const selectedNotActiveWarning = !!focus.selected_egm_id && anyRecentGlobal && (selectedMissing || selectedStale);
  return {
    threshold_ms: thresholdMS,
    any_recent_global: anyRecentGlobal,
    recent_global_count: recentRows.length,
    focused_recent: focusedRecent,
    selected_missing: selectedMissing,
    selected_stale: selectedStale,
    selected_not_active_warning: selectedNotActiveWarning
  };
}

function renderEGMGroupedSummary(snapshot) {
  const focusID = currentEGMFocusID();
  const allRows = groupedSummaryRowsForSnapshot(snapshot);
  const rows = groupedRowsForCurrentFocus(allRows);
  $("egm-grouped-summary-scope").textContent = focusID
    ? ("Scope: " + focusID + " (focused EGM)")
    : "Scope: All EGMs grouped by EGM ID";
  renderItems("egm-grouped-summary", rows, "No EGM telemetry observed yet.", (row) =>
    "<div class=\"item timeline-entry\">" +
      "<div class=\"timeline-entry-head\"><strong>" + escapeHTML(row.egm_id) + "</strong><div class=\"timeline-entry-tags\"><span class=\"timeline-egm-chip\">" + escapeHTML(row.egm_id) + "</span>" + egmSourcePill(row.source) + statusPill(row.status) + "</div></div>" +
      "<span>events " + String(row.total_events) + " total | " + String(row.non_heartbeat_events) + " status events | " + String(row.keepalive_events) + " keepAlive | heartbeat " + escapeHTML(row.heartbeat_label) + "</span>" +
      "<span>last seen " + escapeHTML(fmtTime(row.last_seen_at)) + " | last keepAlive " + escapeHTML(fmtTime(row.heartbeat_last_keepalive_at)) + "</span>" +
      "<span>endpoint " + escapeHTML((row.last_endpoint_ip || "-") + ":" + (row.last_endpoint_port || "-")) +
      " | endpoint seen " + escapeHTML(fmtTime(row.last_endpoint_seen_at)) +
      " | drift " + escapeHTML(row.endpoint_drift_warning ? ("warning (" + ((row.endpoint_drift_ips || []).join(", ")) + ")") : "none") +
      " | integrity " + escapeHTML(row.endpoint_collision_warning ? ("warning (" + ((row.endpoint_collision_types || []).join(", ")) + ")") : "none") + "</span>" +
    "</div>"
  );
}

function endpointCollisionTypeLabel(value) {
  const normalized = String(value || "").toUpperCase();
  if (normalized === "SHARED_ENDPOINT") {
    return "Shared Endpoint";
  }
  if (normalized === "ID_ENDPOINT_DRIFT") {
    return "EGM ID Endpoint Drift";
  }
  return normalized || "Unknown";
}

function normalizeEndpointCollisionRows(status) {
  const rows = Array.isArray(status?.endpoint_collisions) ? status.endpoint_collisions.slice() : [];
  return rows
    .map((row) => ({
      collision_type: String(row?.collision_type || "").toUpperCase(),
      involved_egm_ids: Array.isArray(row?.involved_egm_ids) ? row.involved_egm_ids.map((item) => String(item || "").trim()).filter(Boolean) : [],
      endpoint: String(row?.endpoint || "").trim(),
      first_seen_at: String(row?.first_seen_at || "").trim(),
      last_seen_at: String(row?.last_seen_at || "").trim()
    }))
    .sort((a, b) => {
      const timeCompare = numericTime(b?.last_seen_at) - numericTime(a?.last_seen_at);
      if (timeCompare !== 0) return timeCompare;
      if (a.collision_type !== b.collision_type) return a.collision_type.localeCompare(b.collision_type);
      return String(a.endpoint || "").localeCompare(String(b.endpoint || ""));
    });
}

function normalizeEndpointIntegrityAlertsResponse(raw) {
  const payload = raw && typeof raw === "object" ? raw : {};
  const normalizeAlert = (item) => {
    const alert = item && typeof item === "object" ? item : {};
    const ids = Array.isArray(alert.egm_ids)
      ? alert.egm_ids
      : Array.isArray(alert.involved_egm_ids)
        ? alert.involved_egm_ids
        : [];
    return {
      id: String(alert.id || "").trim(),
      type: String(alert.type || alert.collision_type || "").toUpperCase(),
      egm_ids: ids.map((value) => String(value || "").trim()).filter(Boolean),
      endpoint: String(alert.endpoint || "").trim(),
      first_seen_at: String(alert.first_seen_at || "").trim(),
      last_seen_at: String(alert.last_seen_at || "").trim(),
      severity: String(alert.severity || "").toUpperCase(),
      acked_at: String(alert.acked_at || "").trim(),
      acked_by_scope: String(alert.acked_by_scope || "").trim(),
      snoozed_until: String(alert.snoozed_until || "").trim(),
      snooze_reason: String(alert.snooze_reason || "").trim()
    };
  };
  const alerts = Array.isArray(payload.alerts) ? payload.alerts.map(normalizeAlert) : [];
  const activeAlerts = Array.isArray(payload.active_alerts) ? payload.active_alerts.map(normalizeAlert) : [];
  const ackedAlerts = Array.isArray(payload.acked_alerts) ? payload.acked_alerts.map(normalizeAlert) : [];
  const snoozedAlerts = Array.isArray(payload.snoozed_alerts) ? payload.snoozed_alerts.map(normalizeAlert) : [];
  const summary = payload.summary && typeof payload.summary === "object" ? payload.summary : {};
  return {
    generated_at: String(payload.generated_at || "").trim(),
    summary: {
      total: Number(summary.total || alerts.length || 0),
      active_count: Number(summary.active_count || activeAlerts.length || 0),
      acked_count: Number(summary.acked_count || ackedAlerts.length || 0),
      snoozed_count: Number(summary.snoozed_count || snoozedAlerts.length || 0),
      suppressed_count: Number(summary.suppressed_count || (ackedAlerts.length + snoozedAlerts.length))
    },
    alerts: alerts,
    active_alerts: activeAlerts,
    acked_alerts: ackedAlerts,
    snoozed_alerts: snoozedAlerts
  };
}

function fallbackEndpointIntegrityAlertsFromStatus(status) {
  const rows = normalizeEndpointCollisionRows(status).map((row, idx) => ({
    id: "fallback-" + String(idx + 1),
    type: String(row?.collision_type || "").toUpperCase(),
    egm_ids: Array.isArray(row?.involved_egm_ids) ? row.involved_egm_ids.slice() : [],
    endpoint: String(row?.endpoint || "").trim(),
    first_seen_at: String(row?.first_seen_at || "").trim(),
    last_seen_at: String(row?.last_seen_at || "").trim(),
    severity: String(row?.collision_type || "").toUpperCase() === "SHARED_ENDPOINT" ? "ALERT" : "WARNING",
    acked_at: "",
    acked_by_scope: "",
    snoozed_until: "",
    snooze_reason: ""
  }));
  return {
    generated_at: "",
    summary: {
      total: rows.length,
      active_count: rows.length,
      acked_count: 0,
      snoozed_count: 0,
      suppressed_count: 0
    },
    alerts: rows,
    active_alerts: rows,
    acked_alerts: [],
    snoozed_alerts: []
  };
}

function endpointIntegrityAlertsState(snapshot) {
  const status = snapshot?.status || {};
  const payload = snapshot?.endpointIntegrityAlerts;
  if (payload && typeof payload === "object" && Array.isArray(payload.alerts)) {
    return payload;
  }
  return fallbackEndpointIntegrityAlertsFromStatus(status);
}

function endpointIntegrityAlertSections(alertPayload, referenceTime) {
  const payload = alertPayload && typeof alertPayload === "object" ? alertPayload : {};
  const alerts = Array.isArray(payload.alerts) ? payload.alerts.slice() : [];
  const hasStructuredSections =
    Array.isArray(payload.active_alerts) ||
    Array.isArray(payload.acked_alerts) ||
    Array.isArray(payload.snoozed_alerts);
  if (hasStructuredSections) {
    return {
      active: Array.isArray(payload.active_alerts) ? payload.active_alerts.slice() : [],
      acked: Array.isArray(payload.acked_alerts) ? payload.acked_alerts.slice() : [],
      snoozed: Array.isArray(payload.snoozed_alerts) ? payload.snoozed_alerts.slice() : []
    };
  }
  const nowMS = numericTime(referenceTime || new Date().toISOString());
  const active = [];
  const acked = [];
  const snoozed = [];
  alerts.forEach((alert) => {
    const snoozedUntilMS = numericTime(alert?.snoozed_until);
    const ackedAtMS = numericTime(alert?.acked_at);
    if (snoozedUntilMS > nowMS) {
      snoozed.push(alert);
      return;
    }
    if (ackedAtMS > 0) {
      acked.push(alert);
      return;
    }
    active.push(alert);
  });
  return { active, acked, snoozed };
}

function selectedEndpointIntegrityCustomSnoozeMinutes() {
  const input = $("endpoint-integrity-custom-minutes");
  const value = Number((input && input.value) || 0);
  if (!Number.isFinite(value) || value <= 0) {
    return 30;
  }
  return Math.floor(value);
}

function endpointIntegrityAlertActionButtons(alert, state) {
  const id = escapeHTML(alert.id);
  const disabledAttr = clientState.endpointIntegrityActionInFlight ? " disabled" : "";
  if (state === "snoozed") {
    return "<div class=\"endpoint-integrity-warning-actions\">" +
      "<button type=\"button\" class=\"secondary-button endpoint-integrity-action-button\" data-endpoint-alert-action=\"unsnooze\" data-endpoint-alert-id=\"" + id + "\"" + disabledAttr + ">Unsnooze</button>" +
      "</div>";
  }
  return "<div class=\"endpoint-integrity-warning-actions\">" +
    (state === "active"
      ? "<button type=\"button\" class=\"secondary-button endpoint-integrity-action-button\" data-endpoint-alert-action=\"ack\" data-endpoint-alert-id=\"" + id + "\"" + disabledAttr + ">Ack</button>"
      : "") +
    "<button type=\"button\" class=\"secondary-button endpoint-integrity-action-button\" data-endpoint-alert-action=\"snooze\" data-endpoint-alert-minutes=\"15\" data-endpoint-alert-id=\"" + id + "\"" + disabledAttr + ">Snooze 15m</button>" +
    "<button type=\"button\" class=\"secondary-button endpoint-integrity-action-button\" data-endpoint-alert-action=\"snooze\" data-endpoint-alert-minutes=\"60\" data-endpoint-alert-id=\"" + id + "\"" + disabledAttr + ">Snooze 1h</button>" +
    "<button type=\"button\" class=\"secondary-button endpoint-integrity-action-button\" data-endpoint-alert-action=\"snooze_custom\" data-endpoint-alert-id=\"" + id + "\"" + disabledAttr + ">Snooze Custom</button>" +
    "</div>";
}

function endpointIntegrityAlertRowHTML(alert, state) {
  const typeLabel = endpointCollisionTypeLabel(alert.type);
  const ids = Array.isArray(alert.egm_ids) && alert.egm_ids.length ? alert.egm_ids.join(", ") : "-";
  const severity = String(alert.severity || "").trim() || (endpointIntegritySeverityBucket(alert) === "alerts" ? "ALERT" : "WARNING");
  const ackMeta = alert.acked_at ? (" | acked " + fmtTime(alert.acked_at) + (alert.acked_by_scope ? (" by " + alert.acked_by_scope) : "")) : "";
  const snoozeMeta = alert.snoozed_until ? (" | snoozed until " + fmtTime(alert.snoozed_until) + " (" + fmtAge(alert.snoozed_until) + ")" + (alert.snooze_reason ? " | reason " + alert.snooze_reason : "")) : "";
  return "<div class=\"item endpoint-integrity-warning\">" +
    "<div class=\"endpoint-integrity-warning-head\"><strong>" + escapeHTML(typeLabel) + "</strong><span class=\"timeline-egm-chip\">" + escapeHTML(alert.endpoint || "-") + "</span></div>" +
    "<div class=\"endpoint-integrity-warning-meta\">Severity: " + escapeHTML(severity) + " | EGMs: " + escapeHTML(ids) + " | first seen " + escapeHTML(fmtTime(alert.first_seen_at)) + " | last seen " + escapeHTML(fmtTime(alert.last_seen_at)) + " (" + escapeHTML(fmtAge(alert.last_seen_at)) + ")" + escapeHTML(ackMeta + snoozeMeta) + "</div>" +
    endpointIntegrityAlertActionButtons(alert, state) +
    "</div>";
}

function renderEndpointIntegrity(snapshot) {
  const status = snapshot?.status || {};
  const collisionSummary = status?.endpoint_collision_summary || {};
  const alertPayload = endpointIntegrityAlertsState(snapshot);
  const sections = endpointIntegrityAlertSections(alertPayload, new Date().toISOString());
  const allAlerts = Array.isArray(alertPayload?.alerts) ? alertPayload.alerts.slice() : [];
  const activeRows = sections.active.filter((row) => endpointIntegrityMatchesGlobalFilters(row));
  const ackedRows = sections.acked.filter((row) => endpointIntegrityMatchesGlobalFilters(row));
  const snoozedRows = sections.snoozed.filter((row) => endpointIntegrityMatchesGlobalFilters(row));
  const filteredTotal = activeRows.length + ackedRows.length + snoozedRows.length;
  const total = Number(alertPayload?.summary?.total || allAlerts.length || 0);
  const activeCount = Number(alertPayload?.summary?.active_count || sections.active.length || 0);
  const ackedCount = Number(alertPayload?.summary?.acked_count || sections.acked.length || 0);
  const snoozedCount = Number(alertPayload?.summary?.snoozed_count || sections.snoozed.length || 0);
  const suppressedCount = Number(alertPayload?.summary?.suppressed_count || (ackedCount + snoozedCount));
  const sharedCount = Number(collisionSummary?.shared_endpoint_count || 0);
  const driftCount = Number(collisionSummary?.id_endpoint_drift_count || 0);
  const affectedIDs = Array.isArray(collisionSummary?.affected_egm_ids) ? collisionSummary.affected_egm_ids.map((item) => String(item || "").trim()).filter(Boolean) : [];

  $("endpoint-integrity-state").textContent = activeCount > 0 ? "warning" : "ready";
  $("endpoint-integrity-state").className = "source-pill " + (activeCount > 0 ? "source-mixed" : "source-file");
  $("endpoint-integrity-summary").textContent = total > 0
    ? ("Detected: " + String(total) + " | active " + String(activeCount) + " | suppressed " + String(suppressedCount) +
      " | shared endpoint " + String(sharedCount) + " | ID endpoint drift " + String(driftCount) +
      " | showing " + String(filteredTotal) + " row(s)" +
      (affectedIDs.length ? (" | affected EGMs " + affectedIDs.join(", ")) : ""))
    : "No endpoint collisions detected.";
  $("endpoint-integrity-message").textContent = total > 0
    ? "Signals only. Acknowledge or snooze to manage operator noise; endpoint integrity warnings do not block mute path."
    : "Signals only. Endpoint integrity warnings do not block mute path.";
  $("endpoint-integrity-counts").textContent = "Active " + String(activeCount) + " | Acknowledged " + String(ackedCount) + " | Snoozed " + String(snoozedCount) + " | Suppressed " + String(suppressedCount);
  $("endpoint-integrity-active-count").textContent = String(activeCount);
  $("endpoint-integrity-acked-count").textContent = String(ackedCount);
  $("endpoint-integrity-snoozed-count").textContent = String(snoozedCount);
  $("endpoint-integrity-refresh-button").disabled = clientState.inFlight || clientState.endpointIntegrityActionInFlight;

  const filterButton = $("endpoint-integrity-filter-button");
  const integrityFilterActive = clientState.egmFilter === "endpoint_integrity";
  filterButton.textContent = integrityFilterActive ? "Show All EGMs" : "Show Affected EGMs";
  filterButton.disabled = total === 0 && !integrityFilterActive;

  renderItems("endpoint-integrity-list", activeRows, "No active endpoint integrity rows match current global filters.", (item) => endpointIntegrityAlertRowHTML(item, "active"));
  renderItems("endpoint-integrity-acked-list", ackedRows, "No acknowledged endpoint integrity rows match current global filters.", (item) => endpointIntegrityAlertRowHTML(item, "acked"));
  renderItems("endpoint-integrity-snoozed-list", snoozedRows, "No snoozed endpoint integrity rows match current global filters.", (item) => endpointIntegrityAlertRowHTML(item, "snoozed"));
}

async function reloadEndpointIntegrityAlerts() {
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  try {
    $("endpoint-integrity-message").textContent = "Refreshing endpoint integrity alerts.";
    const response = await fetchJSON(endpoints.endpointIntegrityAlerts);
    snapshot.endpointIntegrityAlerts = normalizeEndpointIntegrityAlertsResponse(response);
    clientState.displaySnapshot = snapshot;
    renderEndpointIntegrity(snapshot);
    $("endpoint-integrity-message").textContent = "Endpoint integrity alerts refreshed.";
  } catch (err) {
    $("endpoint-integrity-message").textContent = err && err.message ? err.message : "Endpoint integrity alert refresh failed.";
  }
}

async function postEndpointIntegrityAlertAction(alertID, action, payload) {
  const tokenRequired = mutationTokenRequired();
  const token = getSetupToken() || getCertToken();
  if (tokenRequired && !token) {
    $("endpoint-integrity-message").textContent = "Enter an API token before endpoint integrity alert actions.";
    return;
  }
  const normalizedID = String(alertID || "").trim();
  if (!normalizedID) {
    $("endpoint-integrity-message").textContent = "Alert id is required.";
    return;
  }
  const normalizedAction = String(action || "").trim();
  if (!normalizedAction) {
    $("endpoint-integrity-message").textContent = "Alert action is required.";
    return;
  }

  const headers = {};
  if (token) {
    headers.Authorization = "Bearer " + token;
  }
  if (payload != null) {
    headers["Content-Type"] = "application/json";
  }

  let finalMessage = "";
  try {
    clientState.endpointIntegrityActionInFlight = true;
    renderEndpointIntegrity(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
    $("endpoint-integrity-message").textContent = "Applying endpoint integrity alert action.";
    const response = await fetch(endpoints.endpointIntegrityAlerts + "/" + encodeURIComponent(normalizedID) + "/" + normalizedAction, {
      method: "POST",
      headers: withEGMFocusHeader(headers),
      body: payload != null ? JSON.stringify(payload) : undefined
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      finalMessage = "Alert action failed: HTTP " + response.status + (detail ? " " + detail : "");
      return;
    }
    const result = normalizeEndpointIntegrityAlertsResponse(await response.json());
    const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
    snapshot.endpointIntegrityAlerts = result;
    clientState.displaySnapshot = snapshot;
    renderEndpointIntegrity(snapshot);
    finalMessage = normalizedAction === "ack"
      ? "Endpoint integrity alert acknowledged."
      : normalizedAction === "snooze"
        ? "Endpoint integrity alert snoozed."
        : "Endpoint integrity alert unsnoozed.";
  } catch (err) {
    finalMessage = err && err.message ? err.message : "Endpoint integrity alert action failed.";
  } finally {
    clientState.endpointIntegrityActionInFlight = false;
    renderEndpointIntegrity(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
    if (finalMessage) {
      $("endpoint-integrity-message").textContent = finalMessage;
    }
  }
}

async function handleEndpointIntegrityAlertActionFromUI(button) {
  if (!button) return;
  const alertID = button.getAttribute("data-endpoint-alert-id") || "";
  const action = button.getAttribute("data-endpoint-alert-action") || "";
  if (!alertID || !action) return;
  if (action === "ack" || action === "unsnooze") {
    await postEndpointIntegrityAlertAction(alertID, action, null);
    return;
  }
  if (action === "snooze") {
    const minutes = Number(button.getAttribute("data-endpoint-alert-minutes") || 0);
    await postEndpointIntegrityAlertAction(alertID, "snooze", {
      minutes: Number.isFinite(minutes) && minutes > 0 ? Math.floor(minutes) : 15,
      snooze_reason: "operator_snooze"
    });
    return;
  }
  if (action === "snooze_custom") {
    const minutes = selectedEndpointIntegrityCustomSnoozeMinutes();
    await postEndpointIntegrityAlertAction(alertID, "snooze", {
      minutes: minutes,
      snooze_reason: "operator_custom_" + String(minutes) + "m"
    });
  }
}

function firstTestEGMIDSet(snapshot) {
  const profile = snapshot?.cabinetProfile?.effective || snapshot?.status?.cabinet_profile || {};
  const ids = Array.isArray(profile?.first_test_egm_ids) ? profile.first_test_egm_ids : [];
  return new Set(ids.map((item) => String(item || "").trim()).filter(Boolean));
}

function selectedEGMDetailForSnapshot(snapshot) {
  const focus = egmFocusScope(snapshot);
  const telemetry = telemetryReadinessForSnapshot(snapshot);
  const rows = groupedSummaryRowsForSnapshot(snapshot);
  const firstTestIDs = firstTestEGMIDSet(snapshot);
  const focusedID = focus.selected_egm_id || "";
  let selected = null;
  if (focusedID) {
    selected = rows.find((row) => String(row?.egm_id || "").trim() === focusedID) || null;
  } else if (rows.length === 1) {
    selected = rows[0];
  }
  if (!selected) {
    return {
      scope_label: focus.label,
      egm_id: "",
      source: "",
      status: "",
      last_seen_at: "",
      heartbeat_label: "",
      heartbeat_last_keepalive_at: "",
      last_endpoint_ip: "",
      last_endpoint_port: 0,
      last_endpoint_seen_at: "",
      endpoint_drift_warning: false,
      endpoint_drift_ips: [],
      endpoint_collision_warning: false,
      endpoint_collision_types: [],
      recent_endpoints: [],
      endpoint_warning_text: "",
      live_signal: "UNKNOWN",
      live_signal_detail: "",
      in_first_test_set: false,
      focus_mode: focus.mode,
      message: focusedID
        ? (telemetry.selected_not_active_warning
          ? ("Selected EGM not active: " + focusedID + ". Global telemetry is healthy; choose an active EGM or All EGMs.")
          : ("No telemetry has been observed yet for " + focusedID + "."))
        : "Select one EGM focus to view cabinet-level detail."
    };
  }
  const liveObserved = selected.total_events > 0 || numericTime(selected.last_seen_at) > 0;
  const recentEndpoints = Array.isArray(selected.recent_endpoints)
    ? selected.recent_endpoints.slice().sort((a, b) => numericTime(b?.last_seen_at) - numericTime(a?.last_seen_at))
    : [];
  const endpointWarningText = recentEndpoints.length > 1
    ? "Operator signal: multiple endpoints observed recently for this EGM ID."
    : "";
  const liveSignalDetail = selected.heartbeat_events > 0
    ? "commsOnLine/keepAlive traffic observed."
    : (liveObserved ? "EGM telemetry observed." : "No EGM telemetry observed yet.");
  return {
    scope_label: focus.label,
    egm_id: selected.egm_id,
    source: selected.source,
    status: selected.status,
    last_seen_at: selected.last_seen_at,
    heartbeat_label: selected.heartbeat_label,
    heartbeat_last_keepalive_at: selected.heartbeat_last_keepalive_at,
    last_endpoint_ip: selected.last_endpoint_ip || "",
    last_endpoint_port: Number(selected.last_endpoint_port || 0),
    last_endpoint_seen_at: selected.last_endpoint_seen_at || "",
    endpoint_drift_warning: selected.endpoint_drift_warning === true,
    endpoint_drift_ips: Array.isArray(selected.endpoint_drift_ips) ? selected.endpoint_drift_ips.slice() : [],
    endpoint_collision_warning: selected.endpoint_collision_warning === true,
    endpoint_collision_types: Array.isArray(selected.endpoint_collision_types) ? selected.endpoint_collision_types.slice() : [],
    recent_endpoints: recentEndpoints,
    endpoint_warning_text: endpointWarningText,
    live_signal: liveObserved ? "OBSERVED" : "NOT_OBSERVED",
    live_signal_detail: liveSignalDetail,
    in_first_test_set: firstTestIDs.has(selected.egm_id),
    focus_mode: focus.mode,
    message: liveObserved
      ? "Live telemetry is present for this EGM."
      : "No EGM history rows yet; waiting for session traffic."
  };
}

function renderSelectedEGMDetail(snapshot) {
  const detail = selectedEGMDetailForSnapshot(snapshot);
  $("selected-egm-detail-scope").textContent = "Scope: " + (detail.scope_label || "All EGMs");
  if (!detail.egm_id) {
    renderItems("selected-egm-detail", [], detail.message || "Select one EGM to inspect cabinet detail.", () => "");
    return;
  }
  renderItems("selected-egm-detail", [detail], "", (item) =>
    (() => {
      const recentRows = (Array.isArray(item.recent_endpoints) ? item.recent_endpoints : [])
        .map((entry) => (entry?.ip || "-") + ":" + (entry?.port || "-") + " | seen " + String(entry?.seen_count || 0) +
          " | first " + fmtTime(entry?.first_seen_at || "") +
          " | last " + fmtTime(entry?.last_seen_at || "") +
          " (" + fmtAge(entry?.last_seen_at || "") + ")");
      const recentHTML = recentRows.length
        ? recentRows.map((line) => "<li>" + escapeHTML(line) + "</li>").join("")
        : "<li>No endpoint observations yet.</li>";
      return "" +
    "<div class=\"item timeline-entry\">" +
      "<div class=\"timeline-entry-head\"><strong>" + escapeHTML(item.egm_id) + "</strong><div class=\"timeline-entry-tags\"><span class=\"timeline-egm-chip\">" + escapeHTML(item.egm_id) + "</span>" + egmSourcePill(item.source) + statusPill(item.status) + "</div></div>" +
      "<span>live signal " + escapeHTML(item.live_signal || "-") + " | last seen " + escapeHTML(fmtTime(item.last_seen_at)) + " (" + escapeHTML(fmtAge(item.last_seen_at)) + ") | heartbeat " + escapeHTML(item.heartbeat_label || "-") + " | last keepAlive " + escapeHTML(fmtTime(item.heartbeat_last_keepalive_at)) + "</span>" +
      "<span>endpoint " + escapeHTML((item.last_endpoint_ip || "-") + ":" + (item.last_endpoint_port || "-")) + " | endpoint seen " + escapeHTML(fmtTime(item.last_endpoint_seen_at)) + " | endpoint drift " + escapeHTML(item.endpoint_drift_warning ? "warning" : "none") + (item.endpoint_drift_warning && item.endpoint_drift_ips?.length ? (" (" + escapeHTML(item.endpoint_drift_ips.join(", ")) + ")") : "") + "</span>" +
      "<span>endpoint integrity " + escapeHTML(item.endpoint_collision_warning ? "warning" : "none") + (item.endpoint_collision_warning && item.endpoint_collision_types?.length ? (" (" + escapeHTML(item.endpoint_collision_types.join(", ")) + ")") : "") + "</span>" +
      "<span>Recent Endpoints (newest first)</span>" +
      "<ul class=\"operator-readiness-items\">" + recentHTML + "</ul>" +
      (item.endpoint_warning_text ? ("<span>" + escapeHTML(item.endpoint_warning_text) + "</span>") : "") +
      "<span>first-test set: " + escapeHTML(item.in_first_test_set ? "yes" : "no") + " | " + escapeHTML(item.message || "") + "</span>" +
    "</div>";
    })()
  );
}

function isHeartbeatEventType(eventType) {
  return heartbeatEventTypes.has(String(eventType || "").toUpperCase());
}

function fmtDurationMs(ms) {
  if (!ms || ms <= 0) return "0s";
  const seconds = Math.round(ms / 1000);
  if (seconds < 60) return seconds + "s";
  const minutes = Math.floor(seconds / 60);
  const remSeconds = seconds % 60;
  if (minutes < 60) return remSeconds ? (minutes + "m " + remSeconds + "s") : (minutes + "m");
  const hours = Math.floor(minutes / 60);
  const remMinutes = minutes % 60;
  return remMinutes ? (hours + "h " + remMinutes + "m") : (hours + "h");
}

function heartbeatEventsFromHistory(records) {
  return (Array.isArray(records) ? records : []).filter((item) => isHeartbeatEventType(item?.event_type));
}

function heartbeatSummary(records, policy, referenceTime) {
  const intervalMs = Number(policy?.interval_ms || 0);
  const warningAfterMissed = Math.max(1, Number(policy?.warning_after_missed || 3));
  const blockAfterMissed = Math.max(warningAfterMissed, Number(policy?.block_after_missed || 6));
  const heartbeats = heartbeatEventsFromHistory(records)
    .slice()
    .sort((a, b) => new Date(a.created_at || 0).getTime() - new Date(b.created_at || 0).getTime());
  const commsOnline = heartbeats.filter((item) => String(item.event_type || "").toUpperCase() === "G2S_SESSION_ONLINE");
  const keepAlives = heartbeats.filter((item) => String(item.event_type || "").toUpperCase() === "G2S_KEEPALIVE");
  const keepaliveCount = keepAlives.reduce((total, item) => total + normalizeHeartbeatRollupCount(item), 0);
  const lastKeepAliveAt = keepAlives.length ? historyRowLastSeenAt(keepAlives[keepAlives.length - 1]) : "";
  const lastObserved = heartbeats.length ? heartbeats[heartbeats.length - 1] : null;
  let maxGapMs = 0;
  for (let i = 1; i < keepAlives.length; i++) {
    const prev = numericTime(historyRowLastSeenAt(keepAlives[i - 1]));
    const next = numericTime(historyRowFirstSeenAt(keepAlives[i]));
    if (Number.isFinite(prev) && Number.isFinite(next) && next >= prev) {
      maxGapMs = Math.max(maxGapMs, next - prev);
    }
  }
  const refTimeMs = referenceTime ? new Date(referenceTime).getTime() : Date.now();
  const lastKeepAliveMs = lastKeepAliveAt ? new Date(lastKeepAliveAt).getTime() : 0;
  const lastObservedMs = lastObserved ? new Date(lastObserved.created_at || 0).getTime() : 0;
  const sinceLastKeepAliveMs = lastKeepAliveMs && Number.isFinite(refTimeMs) ? Math.max(0, refTimeMs - lastKeepAliveMs) : 0;
  const sinceLastObservedMs = lastObservedMs && Number.isFinite(refTimeMs) ? Math.max(0, refTimeMs - lastObservedMs) : 0;
  const warningThresholdMs = intervalMs > 0 ? Math.max(intervalMs * warningAfterMissed, intervalMs + 1000) : 0;
  const blockThresholdMs = intervalMs > 0 ? Math.max(intervalMs * blockAfterMissed, warningThresholdMs) : 0;
  let health = "NO_TRAFFIC";
  let label = "No heartbeat observed";
  let message = "No commsOnLine or keepAlive traffic is present in the current window.";
  let severity = "idle";
  if (heartbeats.length > 0) {
    if (keepaliveCount === 0 && intervalMs > 0 && sinceLastObservedMs > blockThresholdMs) {
      health = "ESCALATED";
      label = "Keepalive missing";
      message = "commsOnLine was observed, but keepAlive traffic did not follow before the escalation threshold.";
      severity = "critical";
    } else if (keepaliveCount === 0 && intervalMs > 0 && sinceLastObservedMs > warningThresholdMs) {
      health = "WARNING";
      label = "Keepalive delayed";
      message = "commsOnLine was observed, but keepAlive traffic has not started within the warning threshold.";
      severity = "warning";
    } else if (keepaliveCount === 0) {
      health = "ONLINE_ONLY";
      label = "Online only";
      message = "commsOnLine was observed, but keepAlive traffic has not started in this window.";
      severity = "info";
    } else if (intervalMs <= 0) {
      health = "OBSERVED";
      label = "Observed";
      message = "Heartbeat traffic is present; configured interval is unavailable so gap checks are disabled.";
      severity = "info";
    } else if (maxGapMs > blockThresholdMs || sinceLastKeepAliveMs > blockThresholdMs) {
      health = "ESCALATED";
      label = "Gap detected";
      message = "Heartbeat traffic is present, but a keepAlive gap exceeded the escalation threshold.";
      severity = "critical";
    } else if (maxGapMs > warningThresholdMs || sinceLastKeepAliveMs > warningThresholdMs) {
      health = "WARNING";
      label = "Gap warning";
      message = "Heartbeat traffic is present, but a keepAlive gap exceeded the warning threshold.";
      severity = "warning";
    } else {
      health = "HEALTHY";
      label = "Healthy";
      message = "Heartbeat traffic matches the configured cadence for this run window.";
      severity = "healthy";
    }
  }
  return {
    health: health,
    severity: severity,
    label: label,
    message: message,
    total: commsOnline.length + keepaliveCount,
    comms_online_count: commsOnline.length,
    keepalive_count: keepaliveCount,
    first_comms_online_at: commsOnline.length ? commsOnline[0].created_at : "",
    last_observed_at: lastObserved ? lastObserved.created_at : "",
    last_keepalive_at: lastKeepAliveAt,
    interval_ms: intervalMs || 0,
    warning_after_missed: warningAfterMissed,
    block_after_missed: blockAfterMissed,
    warning_threshold_ms: warningThresholdMs,
    block_threshold_ms: blockThresholdMs,
    max_gap_ms: maxGapMs,
    since_last_observed_ms: sinceLastObservedMs,
    since_last_keepalive_ms: sinceLastKeepAliveMs
  };
}

function isOperatorDrillEvent(record) {
  const detail = String(record?.detail || "").toLowerCase();
  return detail.indexOf("operator drill") >= 0;
}

function operatorDrillEvidence(records, drillState) {
  const heartbeatRecords = heartbeatEventsFromHistory(records);
  const drillRecords = heartbeatRecords.filter((item) => isOperatorDrillEvent(item));
  const liveRecords = heartbeatRecords.filter((item) => !isOperatorDrillEvent(item));
  const weightedCount = (rows) => rows.reduce((total, row) => total + normalizeHeartbeatRollupCount(row), 0);
  let source = "NONE";
  if (drillRecords.length > 0 && liveRecords.length > 0) {
    source = "MIXED";
  } else if (drillRecords.length > 0) {
    source = "DRILL_ONLY";
  } else if (liveRecords.length > 0) {
    source = "LIVE_ONLY";
  }
  return {
    source: source,
    total_events: weightedCount(heartbeatRecords),
    drill_events: weightedCount(drillRecords),
    live_events: weightedCount(liveRecords),
    egm_ids: Array.from(new Set(drillRecords.map((item) => String(item?.egm_id || "").trim()).filter(Boolean))),
    last_drill_event_at: drillRecords.length ? drillRecords[drillRecords.length - 1].created_at : "",
    state: drillState || null
  };
}

function runWindowIsActive(snapshot) {
  const markers = Array.isArray(snapshot?.runMarkers) ? snapshot.runMarkers.slice() : [];
  markers.sort((a, b) => new Date(a.created_at || 0).getTime() - new Date(b.created_at || 0).getTime());
  let active = false;
  markers.forEach((marker) => {
    const type = String(marker?.marker_type || "").toLowerCase();
    if (type === "start") active = true;
    if (type === "end") active = false;
  });
  return active;
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

function egmSourcePill(source) {
  const normalized = String(source || "").toUpperCase();
  if (normalized === "DISCOVERED") {
    return "<span class=\"egm-source egm-source-discovered\" data-egm-source=\"discovered\">discovered</span>";
  }
  return "<span class=\"egm-source egm-source-configured\" data-egm-source=\"configured\">configured</span>";
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

function approvedBlockerIDSetFromPreflight(preflight) {
  const approved = new Set();
  const ids = Array.isArray(preflight?.blocker_policy?.effective?.approved_blocker_ids)
    ? preflight.blocker_policy.effective.approved_blocker_ids
    : [];
  ids.forEach((id) => {
    const value = String(id || "").trim();
    if (!value) return;
    approved.add(value);
  });
  return approved;
}

function appendFriendlyPreflightBlockers(preflight, blockers) {
  const checks = Array.isArray(preflight?.checks) ? preflight.checks : [];
  const approvedIDs = approvedBlockerIDSetFromPreflight(preflight);
  if (approvedIDs.size === 0) {
    const rawOnly = Array.isArray(preflight?.blockers) ? preflight.blockers : [];
    rawOnly.forEach((item) => appendUniqueBlocker(blockers, String(item || "").trim()));
    return;
  }
  let mappedAny = false;
  checks.forEach((check) => {
    if (!check || check.result !== "FAIL") return;
    const checkID = String(check.id || "").trim();
    if (checkID && !approvedIDs.has(checkID)) {
      return;
    }
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
  const heartbeat = heartbeatSummary(snapshot?.egmHistory || [], currentHeartbeatPolicy(snapshot), new Date().toISOString());

  if (preflight && preflight.overall !== "PASS") {
    appendFriendlyPreflightBlockers(preflight, blockers);
  }
  const readyForSession = blockers.length === 0;
  const overallState = readyForSession ? "LAB_READY" : "BLOCKED";
  return {
    overallState: overallState,
    readyForSession: readyForSession,
    message: readyForSession ? "Ready for first cabinet lab session" : "Resolve readiness issues before first cabinet runbook session.",
    lastCheckedValue: lastCheckedValue,
    readyzState: readyzState,
    preflightState: preflightState,
    profileSource: profileSource || "-",
    profile: profile,
    firstEGMIDs: firstEGMIDs,
    certCounts: certCounts,
    authState: authState,
    blockers: blockers,
    heartbeat: heartbeat
  };
}

function cabinetSessionStepStateClass(value) {
  return String(value || "action_needed").toLowerCase().replace(/[^a-z0-9]+/g, "_");
}

function buildCabinetSessionWorkflow(snapshot, session) {
  const focus = egmFocusScope(snapshot);
  const telemetry = telemetryReadinessForSnapshot(snapshot);
  const readyzOK = snapshot?.readyz?.ok === true || String(session?.readyzState || "").toUpperCase() === "READY";
  const preflightPASS = String(snapshot?.cabinetPreflight?.overall || session?.preflightState || "").toUpperCase() === "PASS";
  const runActive = runWindowIsActive(snapshot);
  const markers = Array.isArray(snapshot?.runMarkers) ? snapshot.runMarkers : [];
  const startCount = markers.filter((item) => String(item?.marker_type || "").toLowerCase() === "start").length;
  const endCount = markers.filter((item) => String(item?.marker_type || "").toLowerCase() === "end").length;
  const evidenceCount = Array.isArray(snapshot?.sessionEvidence) ? snapshot.sessionEvidence.length : 0;
  const focusTarget = focus.selected_egm_id || "all active EGMs";
  const readinessSignalReady = preflightPASS || readyzOK;

  const precheckComplete = session.readyForSession === true && readinessSignalReady;
  const connectComplete = telemetry.any_recent_global;
  const runComplete = !runActive && startCount > 0 && endCount > 0;
  const captureComplete = evidenceCount > 0;
  const sessionComplete = runComplete && captureComplete;

  const steps = [
    {
      id: "pre_check",
      title: "Pre-check",
      state: precheckComplete ? "COMPLETE" : "ACTION_NEEDED",
      detail: precheckComplete
        ? "Preflight or readyz is healthy; profile/certificate/auth checks are ready for runbook prep."
        : "Resolve readiness blockers before starting cabinet session traffic."
    },
    {
      id: "connect_observe",
      title: "Connect/Observe",
      state: connectComplete ? "COMPLETE" : "ACTION_NEEDED",
      detail: connectComplete
        ? (telemetry.selected_not_active_warning
          ? ("Selected EGM not active: " + focusTarget + ". Global telemetry is healthy; choose an active EGM or All EGMs.")
          : ("Recent commsOnLine/keepAlive observed for " + focusTarget + "."))
        : ("No recent commsOnLine/keepAlive observed (threshold " + fmtDurationMs(telemetry.threshold_ms) + ").")
    },
    {
      id: "run_active",
      title: "Run Active",
      state: runActive ? "ACTIVE" : (runComplete ? "COMPLETE" : "ACTION_NEEDED"),
      detail: runActive
        ? "Run window is active; continue operator actions and timeline notes."
        : (runComplete ? "Run window markers show start/end captured." : "Mark run start and end to bound the cabinet session window.")
    },
    {
      id: "capture_evidence",
      title: "Capture Evidence",
      state: captureComplete ? "COMPLETE" : "ACTION_NEEDED",
      detail: captureComplete
        ? ("Saved captures available: " + String(evidenceCount) + ".")
        : "Capture JSON or Markdown evidence before ending the session."
    },
    {
      id: "session_complete",
      title: "Session Complete",
      state: sessionComplete ? "COMPLETE" : "ACTION_NEEDED",
      detail: sessionComplete
        ? "Run markers and saved evidence are complete for this cabinet session."
        : "Complete run markers and evidence capture to close the cabinet session."
    }
  ];

  let currentStep = "Session Complete";
  for (let i = 0; i < steps.length; i++) {
    if (steps[i].state === "ACTION_NEEDED" || steps[i].state === "ACTIVE") {
      currentStep = steps[i].title;
      break;
    }
  }

  return {
    current_step: currentStep,
    focus_label: focus.label,
    focus_mode: focus.mode,
    status_state: snapshot?.status?.state || "",
    steps: steps
  };
}

const workflowProgressSteps = [
  { id: "pre_check", label: "Pre-check complete" },
  { id: "connect_observe", label: "Connect/Observe complete" },
  { id: "run_active", label: "Run Active complete" },
  { id: "capture_evidence", label: "Capture Evidence complete" },
  { id: "session_complete", label: "Session Complete" }
];

function defaultSessionWorkflowProgress() {
  return {
    current_phase: "pre_check",
    completed_steps: [],
    operator_notes: "",
    last_updated_at: "",
    persisted: false
  };
}

function normalizeSessionWorkflowProgress(payload) {
  const fallback = defaultSessionWorkflowProgress();
  const phase = String(payload?.current_phase || fallback.current_phase).trim();
  const completed = Array.isArray(payload?.completed_steps)
    ? payload.completed_steps.map((item) => String(item || "").trim()).filter(Boolean)
    : [];
  const notes = String(payload?.operator_notes || "").trim();
  const validStepIDs = new Set(workflowProgressSteps.map((step) => step.id));
  const uniqueCompleted = [];
  completed.forEach((stepID) => {
    if (!validStepIDs.has(stepID)) return;
    if (uniqueCompleted.indexOf(stepID) >= 0) return;
    uniqueCompleted.push(stepID);
  });
  const validPhase = validStepIDs.has(phase) ? phase : fallback.current_phase;
  return {
    current_phase: validPhase,
    completed_steps: uniqueCompleted,
    operator_notes: notes,
    last_updated_at: String(payload?.last_updated_at || "").trim(),
    persisted: payload?.persisted === true
  };
}

function workflowProgressHasFocus() {
  const steps = $("workflow-progress-steps");
  const notes = $("workflow-progress-notes");
  const phase = $("workflow-progress-phase");
  const active = document.activeElement;
  if (!active) return false;
  if (notes === active || phase === active) return true;
  return !!(steps && steps.contains(active));
}

function sessionWorkflowProgressEquivalent(a, b) {
  const left = normalizeSessionWorkflowProgress(a || {});
  const right = normalizeSessionWorkflowProgress(b || {});
  if (left.current_phase !== right.current_phase) return false;
  if (left.operator_notes !== right.operator_notes) return false;
  if (left.completed_steps.length !== right.completed_steps.length) return false;
  for (let i = 0; i < left.completed_steps.length; i++) {
    if (left.completed_steps[i] !== right.completed_steps[i]) return false;
  }
  return true;
}

function workflowProgressFromForm() {
  const completed = [];
  workflowProgressSteps.forEach((step) => {
    const checkbox = $("workflow-progress-step-" + step.id);
    if (checkbox && checkbox.checked) {
      completed.push(step.id);
    }
  });
  return {
    current_phase: String($("workflow-progress-phase").value || "pre_check").trim(),
    completed_steps: completed,
    operator_notes: $("workflow-progress-notes").value.trim()
  };
}

function workflowProgressAuthHeaders() {
  const headers = { "Content-Type": "application/json" };
  const token = getSetupToken() || getCertToken();
  if (token) {
    headers.Authorization = "Bearer " + token;
  }
  return withEGMFocusHeader(headers);
}

function renderWorkflowProgressStepCheckboxes(progress) {
  const completed = new Set(Array.isArray(progress?.completed_steps) ? progress.completed_steps : []);
  $("workflow-progress-steps").innerHTML = workflowProgressSteps.map((step) =>
    "<label class=\"workflow-progress-step\">" +
      "<input id=\"workflow-progress-step-" + escapeHTML(step.id) + "\" type=\"checkbox\" " + (completed.has(step.id) ? "checked" : "") + ">" +
      "<span>" + escapeHTML(step.label) + "</span>" +
    "</label>"
  ).join("");
}

function setWorkflowProgressMessage(text) {
  $("workflow-progress-message").textContent = text;
}

function setWorkflowProgressUnsavedState(dirty) {
  const badge = $("workflow-progress-unsaved");
  badge.textContent = dirty ? "Unsaved changes" : "Saved";
  badge.className = "workflow-progress-unsaved " + (dirty ? "workflow-progress-unsaved-dirty" : "workflow-progress-unsaved-clean");
}

function updateWorkflowProgressDirtyState() {
  const baseline = normalizeSessionWorkflowProgress(clientState.workflowProgressBaseline || defaultSessionWorkflowProgress());
  const current = normalizeSessionWorkflowProgress(workflowProgressFromForm());
  const dirty = !sessionWorkflowProgressEquivalent(baseline, current);
  setWorkflowProgressUnsavedState(dirty);
  const tokenRequired = setupActionsRequireToken();
  const tokenPresent = !!getSetupToken() || !!getCertToken();
  $("workflow-progress-save-button").disabled = !dirty || (tokenRequired && !tokenPresent);
}

function fillWorkflowProgressForm(progress) {
  const normalized = normalizeSessionWorkflowProgress(progress);
  $("workflow-progress-phase").value = normalized.current_phase;
  renderWorkflowProgressStepCheckboxes(normalized);
  $("workflow-progress-notes").value = normalized.operator_notes || "";
}

function renderSessionWorkflowProgress(snapshot, workflow) {
  const progress = normalizeSessionWorkflowProgress(snapshot?.sessionWorkflow || defaultSessionWorkflowProgress());
  const shouldFill = !clientState.workflowProgressLoaded || !workflowProgressHasFocus();
  if (shouldFill) {
    fillWorkflowProgressForm(progress);
    clientState.workflowProgressBaseline = normalizeSessionWorkflowProgress(progress);
    clientState.workflowProgressLoaded = true;
  }
  $("workflow-progress-last-saved").textContent = progress.persisted && progress.last_updated_at
    ? ("Last saved: " + fmtTime(progress.last_updated_at))
    : "Last saved: not saved";

  if (progress.persisted) {
    setWorkflowProgressMessage("Workflow progress saved. Current phase: " + String(progress.current_phase || "pre_check").replace(/_/g, " ") + ".");
  } else {
    setWorkflowProgressMessage("Workflow progress is not saved yet. Current workflow step: " + (workflow?.current_step || "Pre-check") + ".");
  }

  const tokenRequired = setupActionsRequireToken();
  const tokenPresent = !!getSetupToken() || !!getCertToken();
  $("workflow-progress-clear-button").disabled = !progress.persisted || (tokenRequired && !tokenPresent);
  updateWorkflowProgressDirtyState();
}

async function saveSessionWorkflowProgress() {
  if (setupActionsRequireToken() && !getSetupToken() && !getCertToken()) {
    setWorkflowProgressMessage("Enter an API token before saving workflow progress.");
    return;
  }
  const payload = normalizeSessionWorkflowProgress(workflowProgressFromForm());
  try {
    setWorkflowProgressMessage("Saving workflow progress.");
    const response = await fetch(endpoints.sessionWorkflow, {
      method: "PUT",
      headers: workflowProgressAuthHeaders(),
      body: JSON.stringify({
        current_phase: payload.current_phase,
        completed_steps: payload.completed_steps,
        operator_notes: payload.operator_notes
      })
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setWorkflowProgressMessage("Save failed: HTTP " + response.status + (detail ? " " + detail : ""));
      return;
    }
    const saved = normalizeSessionWorkflowProgress(await response.json());
    const snapshot = copySnapshot(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
    snapshot.sessionWorkflow = saved;
    clientState.displaySnapshot = snapshot;
    clientState.workflowProgressBaseline = saved;
    fillWorkflowProgressForm(saved);
    setWorkflowProgressUnsavedState(false);
    setWorkflowProgressMessage("Workflow progress saved.");
    schedulePoll(0);
  } catch (err) {
    setWorkflowProgressMessage(err && err.message ? err.message : "Workflow progress save failed.");
  }
}

async function clearSessionWorkflowProgress() {
  if (setupActionsRequireToken() && !getSetupToken() && !getCertToken()) {
    setWorkflowProgressMessage("Enter an API token before clearing workflow progress.");
    return;
  }
  try {
    setWorkflowProgressMessage("Clearing workflow progress.");
    const response = await fetch(endpoints.sessionWorkflow, {
      method: "DELETE",
      headers: workflowProgressAuthHeaders()
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setWorkflowProgressMessage("Clear failed: HTTP " + response.status + (detail ? " " + detail : ""));
      return;
    }
    const cleared = normalizeSessionWorkflowProgress(await response.json());
    const snapshot = copySnapshot(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
    snapshot.sessionWorkflow = cleared;
    clientState.displaySnapshot = snapshot;
    clientState.workflowProgressBaseline = cleared;
    fillWorkflowProgressForm(cleared);
    setWorkflowProgressUnsavedState(false);
    setWorkflowProgressMessage("Workflow progress cleared.");
    schedulePoll(0);
  } catch (err) {
    setWorkflowProgressMessage(err && err.message ? err.message : "Workflow progress clear failed.");
  }
}

function pushUniqueString(list, text) {
  const value = String(text || "").trim();
  if (!value) return;
  if (list.indexOf(value) >= 0) return;
  list.push(value);
}

function preflightCheckByID(preflight, id) {
  const checks = Array.isArray(preflight?.checks) ? preflight.checks : [];
  for (let i = 0; i < checks.length; i++) {
    if (String(checks[i]?.id || "") === id) {
      return checks[i];
    }
  }
  return null;
}

function groupedReadinessClass(key) {
  return String(key || "informational").toLowerCase().replace(/[^a-z0-9]+/g, "_");
}

function blockingCertificateRoles(certificates, runtime) {
  const labels = [];
  (Array.isArray(certificates) ? certificates : []).forEach((item) => {
    if (certSeverity(item, runtime) !== "blocking") return;
    pushUniqueString(labels, roleDisplayName(item?.role || ""));
  });
  return labels;
}

function buildOperatorReadinessModel(snapshot, session, workflow) {
  const status = snapshot?.status || {};
  const runtime = status.runtime || {};
  const readyz = snapshot?.readyz || null;
  const preflight = snapshot?.cabinetPreflight || null;
  const certificates = Array.isArray(snapshot?.certificates) ? snapshot.certificates : [];
  const telemetry = telemetryReadinessForSnapshot(snapshot);
  const focus = egmFocusScope(snapshot);
  const selectedEGM = selectedEGMDetailForSnapshot(snapshot);
  const markers = Array.isArray(snapshot?.runMarkers) ? snapshot.runMarkers : [];
  const startCount = markers.filter((item) => String(item?.marker_type || "").toLowerCase() === "start").length;
  const endCount = markers.filter((item) => String(item?.marker_type || "").toLowerCase() === "end").length;
  const evidenceCount = Array.isArray(snapshot?.sessionEvidence) ? snapshot.sessionEvidence.length : 0;
  const heartbeat = session?.heartbeat || heartbeatSummary(snapshot?.egmHistory || [], currentHeartbeatPolicy(snapshot), new Date().toISOString());

  const readyNow = [];
  const needsAction = [];
  const labWarning = [];
  const informational = [];
  const nextActions = [];

  if (readyz?.ok === true || session?.readyzState === "READY") {
    pushUniqueString(readyNow, "Readiness endpoint is healthy.");
  } else {
    pushUniqueString(needsAction, "Restore appliance readiness to READY.");
    pushUniqueString(nextActions, "Check /readyz and resolve degraded readiness issues.");
  }

  if (!preflight) {
    pushUniqueString(needsAction, "Cabinet preflight API is unavailable.");
    pushUniqueString(nextActions, "Restore /api/cabinet-preflight and rerun pre-check.");
  } else if (String(preflight?.overall || "").toUpperCase() === "PASS") {
    pushUniqueString(readyNow, "Cabinet preflight checks are passing.");
  } else {
    const checks = Array.isArray(preflight?.checks) ? preflight.checks : [];
    checks.filter((item) => item?.result === "FAIL").forEach((check) => {
      switch (String(check?.id || "")) {
        case "cabinet_profile":
          pushUniqueString(needsAction, "Cabinet profile values are incomplete.");
          pushUniqueString(nextActions, "Complete cabinet profile values (wire_host_url, host_id, first_test_egm_ids).");
          break;
        case "certificate_mode_requirements":
          pushUniqueString(needsAction, "Required certificate material is missing or invalid.");
          pushUniqueString(nextActions, "Import required web/server or client certificate material.");
          break;
        case "certificate_san_wire_identity":
          pushUniqueString(needsAction, "Certificate SAN does not match cabinet wire identity.");
          pushUniqueString(nextActions, "Regenerate/import certificate matching wire_host_url and required SAN fields.");
          break;
        default:
          pushUniqueString(needsAction, "Preflight check failed: " + (check?.id || "unknown") + ".");
          break;
      }
    });
    if (needsAction.length === 0) {
      pushUniqueString(needsAction, "Cabinet preflight has unresolved failures.");
      pushUniqueString(nextActions, "Review failed preflight checks and resolve blocking items.");
    }
  }
  const downgradedFindings = Array.isArray(preflight?.downgraded_findings) ? preflight.downgraded_findings : [];
  downgradedFindings.forEach((finding) => {
    const id = String(finding?.id || "").trim();
    if (!id) return;
    pushUniqueString(labWarning, "Preflight finding downgraded by blocker policy: " + id + ".");
    pushUniqueString(nextActions, "Review downgraded preflight finding " + id + " before production deployment.");
  });
  const preflightProfileCheck = preflightCheckByID(preflight, "cabinet_profile");
  const placeholderFirstTestWarning = String(preflightProfileCheck?.detail || "").indexOf("lab_warning_code=FIRST_TEST_EGM_IDS_PLACEHOLDER") >= 0;
  if (placeholderFirstTestWarning) {
    pushUniqueString(labWarning, "Replace placeholder first-test EGM IDs before real cabinet deployment.");
    pushUniqueString(nextActions, "Replace placeholder first-test EGM IDs before real cabinet deployment.");
  }

  if (session?.profile?.wire_host_url && session?.profile?.host_id && Array.isArray(session?.firstEGMIDs) && session.firstEGMIDs.length > 0) {
    pushUniqueString(readyNow, "Cabinet profile core identity values are present.");
  } else {
    pushUniqueString(needsAction, "Cabinet profile values are incomplete.");
    pushUniqueString(nextActions, "Complete cabinet profile values (wire_host_url, host_id, first_test_egm_ids).");
  }

  const blockingRoles = blockingCertificateRoles(certificates, runtime);
  if (blockingRoles.length > 0) {
    pushUniqueString(needsAction, "Required certificate roles failing: " + blockingRoles.join(", ") + ".");
  } else {
    pushUniqueString(readyNow, "Required certificate roles are valid for current runtime.");
  }
  if (session?.certCounts?.labOptional > 0) {
    pushUniqueString(labWarning, "Lab-optional certificate roles not configured: " + String(session.certCounts.labOptional) + ".");
  }
  const expiringSoon = certificates.filter((item) => certSeverity(item, runtime) === "warning").length;
  if (expiringSoon > 0) {
    pushUniqueString(labWarning, "Certificate rotation warning: " + String(expiringSoon) + " role(s) expiring soon.");
  }

  if (runtime.api_mutation_auth_required === true) {
    if (getSetupToken() || getCertToken()) {
      pushUniqueString(readyNow, "API token is present for protected setup actions.");
    } else {
      pushUniqueString(needsAction, "Protected setup actions require an API token.");
      pushUniqueString(nextActions, "Enter API token to enable setup, run-marker, and evidence-save actions.");
    }
  } else if (runtime.trusted_mutation_bypass_active === true) {
    pushUniqueString(labWarning, "Trusted mutation bypass is active; verify lab network boundaries.");
  } else {
    pushUniqueString(informational, "Mutation auth is disabled for current runtime.");
  }

  if (telemetry.any_recent_global) {
    pushUniqueString(readyNow, "Recent EGM telemetry observed across " + String(telemetry.recent_global_count) + " EGM(s) within " + fmtDurationMs(telemetry.threshold_ms) + ".");
  } else {
    pushUniqueString(informational, "No recent EGM telemetry observed yet.");
    pushUniqueString(nextActions, "Start cabinet session and confirm commsOnLine/keepAlive traffic.");
  }
  const endpointIntegrity = status?.endpoint_collision_summary || {};
  const endpointAlerts = Number(endpointIntegrity?.total || 0);
  if (endpointAlerts > 0) {
    const sharedCount = Number(endpointIntegrity?.shared_endpoint_count || 0);
    const driftCount = Number(endpointIntegrity?.id_endpoint_drift_count || 0);
    pushUniqueString(labWarning, "Endpoint integrity warnings: " + String(endpointAlerts) + " (shared endpoint " + String(sharedCount) + ", ID endpoint drift " + String(driftCount) + ").");
    pushUniqueString(nextActions, "Review Endpoint Integrity panel and inspect affected EGMs.");
  } else {
    pushUniqueString(informational, "Endpoint integrity: no active endpoint collisions detected.");
  }

  if (!focus.selected_egm_id) {
    pushUniqueString(informational, "All-EGMs focus is active for session-wide monitoring.");
    pushUniqueString(nextActions, "Select one EGM focus for cabinet-level validation detail.");
  } else if (telemetry.selected_not_active_warning) {
    pushUniqueString(labWarning, "Selected EGM not active; global telemetry is healthy.");
    pushUniqueString(nextActions, "Choose an active EGM focus or switch to All EGMs.");
  } else if (!selectedEGM.egm_id || selectedEGM.message.indexOf("No telemetry") >= 0) {
    pushUniqueString(informational, "Focused EGM has no telemetry yet.");
    pushUniqueString(nextActions, "Trigger commsOnLine for " + focus.selected_egm_id + " and confirm heartbeat.");
  } else {
    pushUniqueString(readyNow, "Focused EGM detail is available for " + focus.selected_egm_id + ".");
  }

  if (workflow?.steps?.find((item) => item.id === "run_active")?.state === "ACTION_NEEDED" && startCount === 0) {
    pushUniqueString(nextActions, "Mark run start before executing cabinet validation steps.");
  }
  if (workflow?.steps?.find((item) => item.id === "capture_evidence")?.state === "ACTION_NEEDED") {
    pushUniqueString(nextActions, "Capture session evidence before ending run.");
  }
  if (startCount > 0 && endCount === 0) {
    pushUniqueString(nextActions, "Mark run end to close the session window.");
  }
  if (evidenceCount > 0) {
    pushUniqueString(readyNow, "Saved session evidence is available for follow-up.");
  }

  if (heartbeat?.severity === "critical" || heartbeat?.severity === "warning") {
    pushUniqueString(informational, "Heartbeat signal: " + (heartbeat.label || "warning") + " (" + (heartbeat.message || "check cadence") + ").");
  } else if (heartbeat?.total > 0) {
    pushUniqueString(informational, "Heartbeat signal: " + (heartbeat.label || "observed") + ".");
  } else {
    pushUniqueString(informational, "Heartbeat signal: no traffic yet.");
  }

  return {
    ready_now: readyNow,
    needs_operator_action: needsAction,
    lab_warning: labWarning,
    informational: informational,
    next_actions: nextActions,
    groups: [
      { key: "ready_now", label: "Ready Now", items: readyNow },
      { key: "needs_operator_action", label: "Needs Operator Action", items: needsAction },
      { key: "lab_warning", label: "Lab Warning", items: labWarning },
      { key: "informational", label: "Informational", items: informational }
    ],
    counts: {
      ready_now: readyNow.length,
      needs_operator_action: needsAction.length,
      lab_warning: labWarning.length,
      informational: informational.length
    }
  };
}

function mutePathClassForState(state) {
  if (state === "NOT_GATED_BY_RUNBOOK") return "group-ready_now";
  if (state === "UNKNOWN") return "group-lab_warning";
  return "group-informational";
}

function runbookReadinessClassForState(state) {
  if (state === "LAB_READY") return "group-ready_now";
  if (state === "BLOCKED") return "group-needs_operator_action";
  return "group-informational";
}

function buildMutePathStatus(snapshot, session, readinessModel, workflow) {
  const runbookState = session?.overallState || "UNKNOWN";
  const runbookBlocked = runbookState === "BLOCKED";
  const runbookBlockers = Array.isArray(session?.blockers) ? session.blockers : [];
  const runbookWarnings = Array.isArray(readinessModel?.lab_warning) ? readinessModel.lab_warning : [];
  const nextActions = Array.isArray(readinessModel?.next_actions) ? readinessModel.next_actions : [];
  const groupedRows = groupedSummaryRowsForSnapshot(snapshot);
  const observedEGMs = groupedRows.filter((row) => row.total_events > 0 || numericTime(row.last_seen_at) > 0).length;

  const mutePathState = "NOT_GATED_BY_RUNBOOK";
  const mutePathNote = runbookBlocked
    ? "Mute path is not gated by runbook readiness; this BLOCKED state applies to runbook prep only."
    : "Mute path is not gated by runbook readiness.";
  const confidenceNote = "Software signal only; physical mute-actuator verification is outside this dashboard.";
  const runbookMessage = runbookBlocked
    ? "Runbook readiness is blocked for first cabinet session."
    : "Runbook readiness is clear for first cabinet session.";
  const prepStatus = runbookBlocked
    ? "Cabinet prep can continue by resolving runbook actions."
    : "Cabinet prep can continue and first cabinet session can start.";
  const nextAction = nextActions[0] || (runbookBlocked
    ? "Resolve listed runbook actions before first cabinet session."
    : "Start cabinet session and capture evidence.");

  return {
    mute_path_state: mutePathState,
    mute_path_note: mutePathNote,
    confidence_note: confidenceNote,
    runbook_readiness_state: runbookState,
    runbook_message: runbookMessage,
    runbook_blocker_count: runbookBlockers.length,
    runbook_warning_count: runbookWarnings.length,
    next_action: nextAction,
    can_continue_cabinet_prep: true,
    cabinet_prep_status: prepStatus,
    workflow_step: workflow?.current_step || "",
    observed_egm_count: observedEGMs
  };
}

function renderMutePathStatus(model) {
  const summary = model || {};
  $("mute-path-state").textContent = summary.mute_path_state || "UNKNOWN";
  $("mute-path-message").textContent = summary.mute_path_note || "Mute path status unavailable.";
  $("mute-path-confidence").textContent = summary.confidence_note || "Software signal only.";

  const muteCard = $("mute-path-status-card");
  muteCard.className = "operator-readiness-group mute-path-status-card " + mutePathClassForState(summary.mute_path_state || "UNKNOWN");

  $("runbook-readiness-state").textContent = summary.runbook_readiness_state || "UNKNOWN";
  $("runbook-readiness-message").textContent = (summary.runbook_message || "Runbook readiness status unavailable.") +
    " Blockers: " + String(summary.runbook_blocker_count || 0) +
    " | Lab warnings: " + String(summary.runbook_warning_count || 0);
  $("runbook-readiness-next").textContent = "Next action: " + (summary.next_action || "-");

  const runbookCard = $("runbook-readiness-status-card");
  runbookCard.className = "operator-readiness-group runbook-readiness-status-card " + runbookReadinessClassForState(summary.runbook_readiness_state || "UNKNOWN");

  const prepStatus = summary.can_continue_cabinet_prep === true ? "YES" : "NO";
  $("mute-path-prep-status").textContent = "Cabinet prep can continue: " + prepStatus +
    ". " + (summary.cabinet_prep_status || "-") +
    " Current workflow step: " + (summary.workflow_step || "-") +
    ". Observed EGMs: " + String(summary.observed_egm_count || 0) + ".";
}

function renderOperatorReadinessModel(model) {
  const readiness = model || { groups: [], counts: {}, next_actions: [] };
  $("operator-action-ready-count").textContent = String(readiness?.counts?.ready_now || 0);
  $("operator-action-needed-count").textContent = String(readiness?.counts?.needs_operator_action || 0);
  $("operator-action-lab-warning-count").textContent = String(readiness?.counts?.lab_warning || 0);
  $("operator-action-info-count").textContent = String(readiness?.counts?.informational || 0);
  renderItems("operator-readiness-model", readiness.groups, "No readiness classification available yet.", (group) => {
    const items = Array.isArray(group?.items) ? group.items : [];
    const listHTML = items.length > 0
      ? ("<ul class=\"operator-readiness-items\">" + items.map((item) => "<li>" + escapeHTML(item) + "</li>").join("") + "</ul>")
      : "<span class=\"muted-text\">No items.</span>";
    return "<div class=\"operator-readiness-group group-" + groupedReadinessClass(group?.key) + "\"><strong>" + escapeHTML(group?.label || "-") + "</strong>" + listHTML + "</div>";
  });
  renderItems("next-operator-actions", readiness?.next_actions, "No immediate operator action required.", (item) =>
    "<div class=\"operator-readiness-group group-needs_operator_action\"><strong>Action</strong><ul class=\"operator-readiness-items\"><li>" + escapeHTML(item) + "</li></ul></div>"
  );
}

function renderFirstCabinetSession(snapshot) {
  const session = buildFirstCabinetSessionState(snapshot);
  const workflow = buildCabinetSessionWorkflow(snapshot, session);
  const readinessModel = buildOperatorReadinessModel(snapshot, session, workflow);
  const mutePathStatus = buildMutePathStatus(snapshot, session, readinessModel, workflow);
  const stateBadge = $("first-cabinet-session-state");
  stateBadge.textContent = session.overallState;
  stateBadge.className = "source-pill " + (session.readyForSession ? "source-file" : "source-mixed");
  $("first-cabinet-session-message").textContent = session.message + " Current workflow step: " + workflow.current_step + ". Runbook readiness is separate from mute-path status.";
  $("session-package-export-button").disabled = !snapshot?.status;
  $("runtime-overrides-save-snapshot-button").disabled = !snapshot?.status;
  $("runtime-overrides-restore-button").disabled = mutationTokenRequired() && !getSetupToken() && !getCertToken();
  $("runtime-preset-save-button").disabled = !snapshot?.status || (mutationTokenRequired() && !getSetupToken() && !getCertToken());
  $("runtime-preset-refresh-button").disabled = !snapshot?.status;
  if (!snapshot?.status) {
    $("session-package-export-message").textContent = "Session package export waits for a status snapshot.";
    $("runtime-overrides-message").textContent = "Runtime snapshot/restore waits for a status snapshot.";
    $("runtime-preset-message").textContent = "Runtime preset library waits for a status snapshot.";
  } else if (mutationTokenRequired() && !getSetupToken() && !getCertToken()) {
    $("runtime-overrides-message").textContent = "Enter a setup or certificate API token before restoring runtime overrides.";
    $("runtime-preset-message").textContent = "Enter a setup or certificate API token before saving/loading/deleting presets.";
  }

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
  const endpointSummary = snapshot?.status?.endpoint_collision_summary || {};
  const endpointAlerts = Number(endpointSummary?.total || 0);
  $("first-cabinet-endpoint-alerts").textContent = endpointAlerts > 0
    ? (String(endpointAlerts) + " warning(s)")
    : "None";
  $("first-cabinet-auth-state").textContent = session.authState;

  const blockerList = $("first-cabinet-session-blockers");
  if (session.blockers.length === 0) {
    blockerList.innerHTML = "<div class=\"first-cabinet-session-blocker first-cabinet-session-blockers-empty\">Ready for first cabinet lab session</div>";
  } else {
    blockerList.innerHTML = session.blockers.map((item) => "<div class=\"first-cabinet-session-blocker\">" + escapeHTML(item) + "</div>").join("");
  }
  renderMutePathStatus(mutePathStatus);
  renderOperatorReadinessModel(readinessModel);

  renderItems("first-cabinet-session-workflow", workflow.steps, "No operator workflow data yet.", (step) =>
    "<div class=\"first-cabinet-session-workflow-step step-" + cabinetSessionStepStateClass(step.state) + "\">" +
      "<div class=\"step-head\"><strong>" + escapeHTML(step.title) + "</strong><span class=\"step-state\">" + escapeHTML(step.state) + "</span></div>" +
      "<span>" + escapeHTML(step.detail) + "</span>" +
    "</div>"
  );
  renderSessionWorkflowProgress(snapshot, workflow);
}

function buildSessionEvidence(snapshot) {
  const focus = egmFocusScope(snapshot);
  const session = buildFirstCabinetSessionState(snapshot);
  const workflow = buildCabinetSessionWorkflow(snapshot, session);
  const status = snapshot?.status || {};
  const runtime = status.runtime || {};
  const readiness = status.readiness || {};
  const profilePayload = snapshot?.cabinetProfile || null;
  const profile = profilePayload?.effective || status.cabinet_profile || {};
  const certificates = Array.isArray(snapshot?.certificates) ? snapshot.certificates : [];
  const incidents = Array.isArray(snapshot?.incidents) ? snapshot.incidents : [];
  const egmHistoryAll = Array.isArray(snapshot?.egmHistory) ? snapshot.egmHistory : [];
  const egmHistoryFocused = filterHistoryByFocus(egmHistoryAll);
  const stateHistory = Array.isArray(snapshot?.stateHistory) ? snapshot.stateHistory : [];
  const runMarkers = Array.isArray(snapshot?.runMarkers) ? snapshot.runMarkers : [];
  const drillState = currentOperatorDrill(snapshot);
  const heartbeat = heartbeatSummary(egmHistoryFocused, currentHeartbeatPolicy(snapshot), new Date().toISOString());
  const drillEvidence = operatorDrillEvidence(egmHistoryFocused, drillState);
  const groupedSummaryAll = buildEGMGroupedSummaryRows(status?.egms, egmHistoryAll, currentHeartbeatPolicy(snapshot), new Date().toISOString());
  const groupedSummaryFocused = groupedRowsForCurrentFocus(groupedSummaryAll);
  const selectedEGMDetail = selectedEGMDetailForSnapshot(snapshot);
  const readinessModel = buildOperatorReadinessModel(snapshot, session, workflow);
  const mutePathStatus = buildMutePathStatus(snapshot, session, readinessModel, workflow);
  const notes = $("session-evidence-notes").value.trim();
  return {
    captured_at: new Date().toISOString(),
    egm_focus: focus,
    scope: {
      egm_history_scope: focus.egm_specific_views_filtered ? "FILTERED_TO_EGM" : "FULL_SESSION",
      selected_egm_id: focus.selected_egm_id || "",
      grouped_summary_scope: focus.egm_specific_views_filtered ? "FILTERED_TO_EGM" : "FULL_SESSION",
      global_sections_scope: "FULL_SESSION_GLOBAL_INCLUDED"
    },
    workflow: workflow,
    selected_egm_detail: selectedEGMDetail,
    action_model: readinessModel,
    mute_path: mutePathStatus,
    runbook_readiness: {
      state: mutePathStatus.runbook_readiness_state || "UNKNOWN",
      blocker_count: mutePathStatus.runbook_blocker_count || 0,
      warning_count: mutePathStatus.runbook_warning_count || 0,
      can_continue_cabinet_prep: mutePathStatus.can_continue_cabinet_prep === true,
      next_action: mutePathStatus.next_action || ""
    },
    next_operator_actions: readinessModel.next_actions,
    lab_warnings: readinessModel.lab_warning,
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
    egm_history_total_count: egmHistoryAll.length,
    egm_history_focused_count: egmHistoryFocused.length,
    egm_grouped_summary_count: groupedSummaryFocused.length,
    egm_grouped_summary_count_all: groupedSummaryAll.length,
    heartbeat_summary: heartbeat,
    operator_drill: drillEvidence,
    incidents: incidents,
    egm_history: egmHistoryFocused,
    egm_history_all: egmHistoryAll,
    egm_grouped_summary: groupedSummaryFocused,
    egm_grouped_summary_all: groupedSummaryAll,
    state_history: stateHistory,
    run_markers: runMarkers,
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
    "- EGM focus: " + (evidence?.egm_focus?.label || "All EGMs"),
    "- EGM focus mode: " + (evidence?.egm_focus?.mode || "ALL_EGMS"),
    "- EGM history scope: " + (evidence?.scope?.egm_history_scope || "FULL_SESSION"),
    "- EGM-specific views filtered: " + String(evidence?.egm_focus?.egm_specific_views_filtered === true),
    "- Ready for session: " + String(evidence.session.ready_for_session === true),
    "- Readyz state: " + (evidence.session.readyz_state || "-"),
    "- Preflight state: " + (evidence.session.preflight_state || "-"),
    "- Runbook readiness state: " + (evidence?.runbook_readiness?.state || "UNKNOWN"),
    "- Runbook prep can continue: " + String(evidence?.runbook_readiness?.can_continue_cabinet_prep === true),
    "- Mute path state: " + (evidence?.mute_path?.mute_path_state || "UNKNOWN"),
    "- Mute path note: " + (evidence?.mute_path?.mute_path_note || "-"),
    "- API auth state: " + (evidence.session.api_auth_state || "-"),
    "- Cabinet profile source: " + (evidence.cabinet_profile.source || "-"),
    "- Wire host URL: " + (evidence.cabinet_profile.wire_host_url || "-"),
    "- Host ID: " + (evidence.cabinet_profile.host_id || "-"),
    "- First test EGM IDs: " + ((evidence.cabinet_profile.first_test_egm_ids || []).join(", ") || "-"),
    "- Certificate blocking count: " + String(evidence.session.certificate_blocking_count || 0),
    "- Certificate lab optional count: " + String(evidence.session.certificate_lab_optional_count || 0),
    "- EGM snapshot count: " + String(evidence.egm_snapshot_count || 0),
    "- EGM history rows (focused): " + String(evidence.egm_history_focused_count || 0),
    "- EGM history rows (all): " + String(evidence.egm_history_total_count || 0),
    "- EGM grouped rows (focused): " + String(evidence.egm_grouped_summary_count || 0),
    "- EGM grouped rows (all): " + String(evidence.egm_grouped_summary_count_all || 0),
    "- Incident rows captured: " + String((evidence.incidents || []).length),
    "- State history rows captured: " + String((evidence.state_history || []).length),
    "- Run markers captured: " + String((evidence.run_markers || []).length),
    "- Heartbeat events captured: " + String(evidence?.heartbeat_summary?.total || 0),
    "- Heartbeat health: " + (evidence?.heartbeat_summary?.label || "-"),
    "- Heartbeat source: " + (evidence?.operator_drill?.source || "-"),
    "- Drill heartbeat events: " + String(evidence?.operator_drill?.drill_events || 0),
    ""
  ];
  lines.push("## Mute Path Note", "");
  lines.push("- " + (evidence?.mute_path?.mute_path_note || "Mute path status unavailable."));
  lines.push("- " + (evidence?.mute_path?.confidence_note || "Software signal only."));
  lines.push("", "## Runbook Readiness", "");
  lines.push("- State: " + (evidence?.runbook_readiness?.state || "UNKNOWN"));
  lines.push("- Blocker count: " + String(evidence?.runbook_readiness?.blocker_count || 0));
  lines.push("- Lab warning count: " + String(evidence?.runbook_readiness?.warning_count || 0));
  lines.push("- Next action: " + (evidence?.runbook_readiness?.next_action || "-"));
  lines.push("", "## Runbook Blockers", "");
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
  lines.push("", "## Action Items", "");
  const actionItems = Array.isArray(evidence?.action_model?.needs_operator_action) ? evidence.action_model.needs_operator_action : [];
  if (actionItems.length > 0) {
    actionItems.forEach((item) => lines.push("- " + item));
  } else {
    lines.push("- None");
  }
  lines.push("", "## Lab Warnings", "");
  const labWarnings = Array.isArray(evidence?.action_model?.lab_warning) ? evidence.action_model.lab_warning : [];
  if (labWarnings.length > 0) {
    labWarnings.forEach((item) => lines.push("- " + item));
  } else {
    lines.push("- None");
  }
  lines.push("", "## Next Operator Actions", "");
  const nextActions = Array.isArray(evidence?.next_operator_actions) ? evidence.next_operator_actions : [];
  if (nextActions.length > 0) {
    nextActions.forEach((item) => lines.push("- " + item));
  } else {
    lines.push("- None");
  }
  lines.push("", "## Selected EGM Detail", "");
  const selectedEGMDetail = evidence?.selected_egm_detail || {};
  if (selectedEGMDetail?.egm_id) {
    lines.push("- EGM ID: " + selectedEGMDetail.egm_id);
    lines.push("- Source: " + (selectedEGMDetail.source || "-"));
    lines.push("- Status: " + (selectedEGMDetail.status || "-"));
    lines.push("- Live signal: " + (selectedEGMDetail.live_signal || "-"));
    lines.push("- Live signal detail: " + (selectedEGMDetail.live_signal_detail || "-"));
    lines.push("- Last endpoint: " + ((selectedEGMDetail.last_endpoint_ip || "-") + ":" + (selectedEGMDetail.last_endpoint_port || "-")));
    lines.push("- Endpoint seen: " + (selectedEGMDetail.last_endpoint_seen_at || "-"));
    lines.push("- Endpoint drift warning: " + String(selectedEGMDetail.endpoint_drift_warning === true));
    lines.push("- Endpoint drift IPs: " + ((selectedEGMDetail.endpoint_drift_ips || []).join(", ") || "-"));
    lines.push("- Last seen: " + (selectedEGMDetail.last_seen_at || "-"));
    lines.push("- Heartbeat: " + (selectedEGMDetail.heartbeat_label || "-"));
    lines.push("- Last keepAlive: " + (selectedEGMDetail.heartbeat_last_keepalive_at || "-"));
    lines.push("- First-test set: " + String(selectedEGMDetail.in_first_test_set === true));
  } else {
    lines.push("- " + (selectedEGMDetail?.message || "No selected EGM detail available."));
  }
  lines.push("", "## Operator Notes", "");
  lines.push(evidence.operator_notes || "None");
  lines.push("", "## Workflow", "");
  const workflowSteps = Array.isArray(evidence?.workflow?.steps) ? evidence.workflow.steps : [];
  lines.push("- Current step: " + (evidence?.workflow?.current_step || "-"));
  if (workflowSteps.length > 0) {
    workflowSteps.forEach((step) => lines.push("- " + [step.title || "-", step.state || "-", step.detail || ""].filter(Boolean).join(" | ")));
  } else {
    lines.push("- None");
  }
  lines.push("", "## Grouped EGM Summary", "");
  const groupedRows = Array.isArray(evidence?.egm_grouped_summary) ? evidence.egm_grouped_summary : [];
  const groupedRowsAll = Array.isArray(evidence?.egm_grouped_summary_all) ? evidence.egm_grouped_summary_all : [];
  lines.push("- Scope rows: " + String(groupedRows.length));
  lines.push("- All-session rows: " + String(groupedRowsAll.length));
  if (groupedRows.length > 0) {
    groupedRows.forEach((row) => lines.push("- " + [
      row.egm_id || "-",
      row.source || "-",
      row.status || "-",
      "events=" + String(row.total_events || 0),
      "heartbeat=" + String(row.heartbeat_label || "-"),
      "last_seen=" + String(row.last_seen_at || "-")
    ].join(" | ")));
  } else {
    lines.push("- None");
  }
  lines.push("", "## Run Markers", "");
  if (Array.isArray(evidence.run_markers) && evidence.run_markers.length) {
    evidence.run_markers.forEach((item) => lines.push("- " + [item.marker_type || "marker", item.title || "-", item.created_at || "-", item.notes || ""].filter(Boolean).join(" | ")));
  } else {
    lines.push("- None");
  }
  lines.push("", "## Heartbeat Summary", "");
  lines.push("- Health: " + (evidence?.heartbeat_summary?.label || "-"));
  lines.push("- Total events: " + String(evidence?.heartbeat_summary?.total || 0));
  lines.push("- keepAlive events: " + String(evidence?.heartbeat_summary?.keepalive_count || 0));
  lines.push("- Last keepAlive: " + (evidence?.heartbeat_summary?.last_keepalive_at || "-"));
  lines.push("- Max gap: " + fmtDurationMs(evidence?.heartbeat_summary?.max_gap_ms || 0));
  lines.push("- Notes: " + (evidence?.heartbeat_summary?.message || "-"));
  lines.push("", "## Operator Drill", "");
  lines.push("- Source: " + (evidence?.operator_drill?.source || "-"));
  lines.push("- Drill events: " + String(evidence?.operator_drill?.drill_events || 0));
  lines.push("- Live events: " + String(evidence?.operator_drill?.live_events || 0));
  lines.push("- Drill EGM IDs: " + (((evidence?.operator_drill?.egm_ids || []).join(", ")) || "-"));
  lines.push("- Auto heartbeat running: " + String(evidence?.operator_drill?.state?.auto_heartbeat_running === true));
  lines.push("- Auto heartbeat paused: " + String(evidence?.operator_drill?.state?.auto_heartbeat_paused === true));
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
  const runMarkers = Array.isArray(evidence?.run_markers) ? evidence.run_markers : [];
  const heartbeat = evidence?.heartbeat_summary || {};
  const drill = evidence?.operator_drill || {};
  const focusLabel = evidence?.egm_focus?.label || "All EGMs";
  const workflowStep = evidence?.workflow?.current_step || "-";
  const groupedFocusedCount = Number(evidence?.egm_grouped_summary_count || (Array.isArray(evidence?.egm_grouped_summary) ? evidence.egm_grouped_summary.length : 0));
  const groupedAllCount = Number(evidence?.egm_grouped_summary_count_all || (Array.isArray(evidence?.egm_grouped_summary_all) ? evidence.egm_grouped_summary_all.length : groupedFocusedCount));
  renderItems("session-evidence-selected", [record], "", () =>
    "<div class=\"item session-evidence-selected-detail\">" +
      "<strong>" + escapeHTML(evidence?.session?.overall_state || "-") + " | " + escapeHTML(evidence?.cabinet_profile?.host_id || "-") + "</strong>" +
      "<span>" + escapeHTML(fmtTime(evidence?.captured_at || record.created_at)) + " | " + escapeHTML(evidence?.cabinet_profile?.wire_host_url || "-") + "</span>" +
      "<div class=\"kv-inline\"><span>Readyz: " + escapeHTML(evidence?.session?.readyz_state || "-") + " | Preflight: " + escapeHTML(evidence?.session?.preflight_state || "-") + " | Focus: " + escapeHTML(focusLabel) + " | Workflow: " + escapeHTML(workflowStep) + "</span></div>" +
      "<div class=\"kv-inline\"><span>Blockers: " + escapeHTML(String(blockers.length)) + " | Warnings: " + escapeHTML(String(warnings.length)) + " | Run markers: " + escapeHTML(String(runMarkers.length)) + " | Heartbeat: " + escapeHTML(String(heartbeat.total || 0)) + " (" + String(heartbeat.label || "-") + ") | Source: " + escapeHTML(String(drill.source || "-")) + "</span></div>" +
      "<div class=\"kv-inline\"><span>Grouped EGMs (focused/all): " + escapeHTML(String(groupedFocusedCount)) + " / " + escapeHTML(String(groupedAllCount)) + "</span></div>" +
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
    setAlert("warning", "Export All failed", "No saved captures are available for export.");
    return;
  }
  const exportHeaders = {};
  const exportToken = getSetupToken() || getCertToken();
  if (exportToken) {
    exportHeaders.Authorization = "Bearer " + exportToken;
  }
  fetch(endpoints.sessionEvidenceExportAll, { cache: "no-store", headers: withEGMFocusHeader(exportHeaders) })
    .then(async (response) => {
      if (!response.ok) {
        const detail = sanitizeHTTPText(await response.text());
        throw new Error("Export failed: HTTP " + response.status + (detail ? " " + detail : ""));
      }
      const text = await response.text();
      const disposition = String(response.headers.get("Content-Disposition") || "");
      const match = disposition.match(/filename=\"([^\"]+)\"/i);
      const filename = match && match[1] ? match[1] : "saved-session-evidence-archive.json";
      downloadTextMaterial(filename, text);
      $("session-evidence-state").textContent = "saved";
      $("session-evidence-state").className = "source-pill source-file";
      $("session-evidence-message").textContent = "All saved evidence captures exported.";
      setAlert("info", "Export All complete", "Saved session evidence archive downloaded.");
    })
    .catch((err) => {
      $("session-evidence-state").textContent = "blocked";
      $("session-evidence-state").className = "source-pill source-mixed";
      $("session-evidence-message").textContent = err && err.message ? err.message : "Export failed.";
      setAlert("warning", "Export All failed", "Unable to export all saved evidence captures.");
    });
}

function exportSessionPackage() {
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  if (!snapshot?.status) {
    $("session-package-export-message").textContent = "Session package export waits for a status snapshot.";
    setAlert("warning", "Session package export unavailable", "Wait for status polling to complete before exporting.");
    return;
  }
  const headers = {};
  const token = getSetupToken() || getCertToken();
  if (token) {
    headers.Authorization = "Bearer " + token;
  }
  $("session-package-export-message").textContent = "Exporting session package.";
  fetch(endpoints.sessionPackageExport, { cache: "no-store", headers: withEGMFocusHeader(headers) })
    .then(async (response) => {
      if (!response.ok) {
        const detail = sanitizeHTTPText(await response.text());
        throw new Error("Session package export failed: HTTP " + response.status + (detail ? " " + detail : ""));
      }
      const text = await response.text();
      const disposition = String(response.headers.get("Content-Disposition") || "");
      const match = disposition.match(/filename=\"([^\"]+)\"/i);
      const filename = match && match[1] ? match[1] : "session-package-export.json";
      downloadTextMaterial(filename, text);
      $("session-package-export-message").textContent = "Session package exported.";
      setAlert("info", "Session package exported", "Full session package archive downloaded.");
    })
    .catch((err) => {
      $("session-package-export-message").textContent = err && err.message ? err.message : "Session package export failed.";
      setAlert("warning", "Session package export failed", "Unable to export session package.");
    });
}

function runtimeOverridesRestoreHeaders() {
  const headers = { "Content-Type": "application/json" };
  const token = getSetupToken() || getCertToken();
  if (token) {
    headers.Authorization = "Bearer " + token;
  }
  return withEGMFocusHeader(headers);
}

function runtimeOverridesSnapshotFilename(payload) {
  const generatedAt = payload && payload.generated_at ? payload.generated_at : new Date().toISOString();
  const compact = String(generatedAt).replace(/[:]/g, "").replace(/[.]/g, "-");
  return "runtime-overrides-snapshot-" + compact + ".json";
}

function setRuntimeOverridesMessage(text) {
  $("runtime-overrides-message").textContent = text;
}

async function saveRuntimeOverridesSnapshot() {
  setRuntimeOverridesMessage("Generating runtime override snapshot.");
  try {
    const response = await fetch(endpoints.runtimeOverridesSnapshot, { cache: "no-store", headers: withEGMFocusHeader({}) });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setRuntimeOverridesMessage("Snapshot failed: HTTP " + response.status + (detail ? " " + detail : ""));
      setAlert("warning", "Runtime snapshot failed", "Unable to generate runtime override snapshot.");
      return;
    }
    const payload = await response.json();
    const serialized = JSON.stringify(payload, null, 2);
    $("runtime-overrides-restore-json").value = serialized;
    downloadTextMaterial(runtimeOverridesSnapshotFilename(payload), serialized);
    setRuntimeOverridesMessage("Runtime override snapshot downloaded and loaded into restore editor.");
    setAlert("info", "Runtime snapshot saved", "Runtime override snapshot downloaded.");
  } catch (err) {
    setRuntimeOverridesMessage(err && err.message ? err.message : "Snapshot failed.");
    setAlert("warning", "Runtime snapshot failed", "Unable to generate runtime override snapshot.");
  }
}

async function loadRuntimeOverridesRestoreFile(event) {
  const input = event?.target;
  const file = input && input.files && input.files[0] ? input.files[0] : null;
  if (!file) {
    return;
  }
  try {
    const text = await file.text();
    $("runtime-overrides-restore-json").value = text;
    setRuntimeOverridesMessage("Loaded snapshot JSON from file. Review and restore when ready.");
  } catch (err) {
    setRuntimeOverridesMessage(err && err.message ? err.message : "Unable to read snapshot file.");
  }
}

async function restoreRuntimeOverridesSnapshot() {
  if (setupActionsRequireToken() && !getSetupToken() && !getCertToken()) {
    setRuntimeOverridesMessage("Enter a setup or certificate API token before restoring runtime overrides.");
    return;
  }
  const raw = String($("runtime-overrides-restore-json").value || "").trim();
  if (!raw) {
    setRuntimeOverridesMessage("Paste or upload a runtime override snapshot JSON payload before restore.");
    return;
  }

  let payload = null;
  try {
    payload = JSON.parse(raw);
  } catch (_) {
    setRuntimeOverridesMessage("Restore payload JSON is invalid.");
    return;
  }

  setRuntimeOverridesMessage("Restoring runtime overrides from snapshot payload.");
  try {
    const response = await fetch(endpoints.runtimeOverridesRestore, {
      method: "POST",
      headers: runtimeOverridesRestoreHeaders(),
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setRuntimeOverridesMessage("Restore failed: HTTP " + response.status + (detail ? " " + detail : ""));
      setAlert("warning", "Runtime restore failed", "Unable to restore runtime override snapshot.");
      return;
    }
    setRuntimeOverridesMessage("Runtime overrides restored. Refreshing operator panels.");
    setAlert("info", "Runtime overrides restored", "Runtime override snapshot applied.");
    schedulePoll(0);
  } catch (err) {
    setRuntimeOverridesMessage(err && err.message ? err.message : "Restore failed.");
    setAlert("warning", "Runtime restore failed", "Unable to restore runtime override snapshot.");
  }
}

function normalizeRuntimeOverridePresetListResponse(payload) {
  const data = payload && typeof payload === "object" ? payload : {};
  const presets = Array.isArray(data.presets) ? data.presets.map((item) => {
    const row = item && typeof item === "object" ? item : {};
    return {
      name: String(row.name || "").trim(),
      note: String(row.note || "").trim(),
      created_at: String(row.created_at || "").trim(),
      updated_at: String(row.updated_at || "").trim()
    };
  }).filter((row) => row.name) : [];
  return {
    generated_at: String(data.generated_at || "").trim(),
    presets: presets
  };
}

function runtimeOverridePresetsFromSnapshot(snapshot) {
  return normalizeRuntimeOverridePresetListResponse(snapshot?.runtimeOverridePresets);
}

function setRuntimePresetMessage(text) {
  $("runtime-preset-message").textContent = text;
}

function renderRuntimePresetLibrary(snapshot) {
  const payload = runtimeOverridePresetsFromSnapshot(snapshot);
  const rows = Array.isArray(payload.presets) ? payload.presets : [];
  if (rows.length === 0) {
    $("runtime-presets-list").innerHTML = "<div class=\"item\">No runtime presets saved yet.</div>";
    return;
  }
  $("runtime-presets-list").innerHTML = rows.map((row) => {
    const note = row.note ? ("<div class=\"runtime-preset-item-meta\">" + escapeHTML(row.note) + "</div>") : "";
    return "" +
      "<div class=\"runtime-preset-item\">" +
        "<div class=\"runtime-preset-item-head\"><strong>" + escapeHTML(row.name) + "</strong><span class=\"muted-text\">Updated " + escapeHTML(fmtTime(row.updated_at)) + "</span></div>" +
        note +
        "<div class=\"runtime-preset-item-meta\">Created " + escapeHTML(fmtTime(row.created_at)) + "</div>" +
        "<div class=\"runtime-preset-item-actions\">" +
          "<button type=\"button\" class=\"secondary-button runtime-preset-action-button\" data-runtime-preset-action=\"load\" data-runtime-preset-name=\"" + escapeHTML(row.name) + "\">Load</button>" +
          "<button type=\"button\" class=\"secondary-button runtime-preset-action-button\" data-runtime-preset-action=\"delete\" data-runtime-preset-name=\"" + escapeHTML(row.name) + "\">Delete</button>" +
        "</div>" +
      "</div>";
  }).join("");
}

async function reloadRuntimePresetList() {
  try {
    const response = await fetch(endpoints.runtimeOverridePresets, { cache: "no-store", headers: withEGMFocusHeader({}) });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setRuntimePresetMessage("Runtime preset refresh failed: HTTP " + response.status + (detail ? " " + detail : ""));
      return;
    }
    const payload = await response.json();
    const snapshot = copySnapshot(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
    snapshot.runtimeOverridePresets = normalizeRuntimeOverridePresetListResponse(payload);
    clientState.displaySnapshot = snapshot;
    renderRuntimePresetLibrary(snapshot);
    setRuntimePresetMessage("Runtime preset list refreshed.");
  } catch (err) {
    setRuntimePresetMessage(err && err.message ? err.message : "Runtime preset refresh failed.");
  }
}

async function saveCurrentRuntimePreset() {
  if (mutationTokenRequired() && !getSetupToken() && !getCertToken()) {
    setRuntimePresetMessage("Enter a setup or certificate API token before saving presets.");
    return;
  }
  const name = String($("runtime-preset-name").value || "").trim();
  const note = String($("runtime-preset-note").value || "").trim();
  if (!name) {
    setRuntimePresetMessage("Preset name is required.");
    return;
  }
  setRuntimePresetMessage("Saving runtime preset " + name + ".");
  try {
    const response = await fetch(endpoints.runtimeOverridePresetSave, {
      method: "POST",
      headers: runtimeOverridesRestoreHeaders(),
      body: JSON.stringify({ name: name, note: note })
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setRuntimePresetMessage("Preset save failed: HTTP " + response.status + (detail ? " " + detail : ""));
      setAlert("warning", "Preset save failed", "Unable to save runtime preset.");
      return;
    }
    const payload = await response.json();
    const snapshot = copySnapshot(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
    snapshot.runtimeOverridePresets = normalizeRuntimeOverridePresetListResponse(payload);
    clientState.displaySnapshot = snapshot;
    renderRuntimePresetLibrary(snapshot);
    setRuntimePresetMessage("Preset saved: " + name + ".");
    setAlert("info", "Runtime preset saved", "Saved runtime preset " + name + ".");
  } catch (err) {
    setRuntimePresetMessage(err && err.message ? err.message : "Preset save failed.");
    setAlert("warning", "Preset save failed", "Unable to save runtime preset.");
  }
}

async function loadRuntimePresetByName(name) {
  const presetName = String(name || "").trim();
  if (!presetName) return;
  if (mutationTokenRequired() && !getSetupToken() && !getCertToken()) {
    setRuntimePresetMessage("Enter a setup or certificate API token before loading presets.");
    return;
  }
  setRuntimePresetMessage("Loading preset " + presetName + ".");
  try {
    const response = await fetch(endpoints.runtimeOverridePresetLoad, {
      method: "POST",
      headers: runtimeOverridesRestoreHeaders(),
      body: JSON.stringify({ name: presetName })
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setRuntimePresetMessage("Preset load failed: HTTP " + response.status + (detail ? " " + detail : ""));
      setAlert("warning", "Preset load failed", "Unable to load runtime preset.");
      return;
    }
    setRuntimePresetMessage("Preset loaded: " + presetName + ". Refreshing console state.");
    setAlert("info", "Runtime preset loaded", "Loaded runtime preset " + presetName + ".");
    schedulePoll(0);
  } catch (err) {
    setRuntimePresetMessage(err && err.message ? err.message : "Preset load failed.");
    setAlert("warning", "Preset load failed", "Unable to load runtime preset.");
  }
}

async function deleteRuntimePresetByName(name) {
  const presetName = String(name || "").trim();
  if (!presetName) return;
  if (mutationTokenRequired() && !getSetupToken() && !getCertToken()) {
    setRuntimePresetMessage("Enter a setup or certificate API token before deleting presets.");
    return;
  }
  if (!window.confirm("Delete runtime preset " + presetName + "?")) {
    return;
  }
  setRuntimePresetMessage("Deleting preset " + presetName + ".");
  try {
    const response = await fetch(endpoints.runtimeOverridePresets + "/" + encodeURIComponent(presetName), {
      method: "DELETE",
      headers: runtimeOverridesRestoreHeaders()
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setRuntimePresetMessage("Preset delete failed: HTTP " + response.status + (detail ? " " + detail : ""));
      setAlert("warning", "Preset delete failed", "Unable to delete runtime preset.");
      return;
    }
    const payload = await response.json();
    const snapshot = copySnapshot(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
    snapshot.runtimeOverridePresets = normalizeRuntimeOverridePresetListResponse(payload);
    clientState.displaySnapshot = snapshot;
    renderRuntimePresetLibrary(snapshot);
    setRuntimePresetMessage("Preset deleted: " + presetName + ".");
    setAlert("info", "Runtime preset deleted", "Deleted runtime preset " + presetName + ".");
  } catch (err) {
    setRuntimePresetMessage(err && err.message ? err.message : "Preset delete failed.");
    setAlert("warning", "Preset delete failed", "Unable to delete runtime preset.");
  }
}

async function deleteSavedSessionEvidence(id) {
  const numericID = Number(id) || 0;
  if (!numericID) {
    $("session-evidence-state").textContent = "blocked";
    $("session-evidence-state").className = "source-pill source-mixed";
    $("session-evidence-message").textContent = "Saved evidence record id is invalid.";
    return;
  }
  if (!window.confirm("Delete saved capture #" + numericID + "? This cannot be undone.")) {
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
    const response = await fetch("/api/session-evidence/" + encodeURIComponent(String(numericID)), {
      method: "DELETE",
      headers: withEGMFocusHeader(headers)
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
    setAlert("info", "Saved capture deleted", "Session evidence capture #" + numericID + " was removed.");
    schedulePoll(0);
  } catch (err) {
    $("session-evidence-state").textContent = "blocked";
    $("session-evidence-state").className = "source-pill source-mixed";
    $("session-evidence-message").textContent = err && err.message ? err.message : "Delete failed.";
    setAlert("warning", "Delete failed", "Unable to delete saved evidence capture.");
  }
}

function renderSessionEvidence(snapshot) {
  const evidence = buildSessionEvidence(snapshot);
  const selectedRecord = selectedSavedSessionEvidence(snapshot);
  $("session-evidence-overall").textContent = evidence.session.overall_state || "-";
  $("session-evidence-timestamp").textContent = fmtTime(evidence.session.last_checked || evidence.captured_at);
  $("session-evidence-egm-focus").textContent = evidence?.egm_focus?.label || "All EGMs";
  $("session-evidence-incident-count").textContent = String((evidence.incidents || []).length);
  $("session-evidence-state-count").textContent = String((evidence.state_history || []).length);
  $("session-evidence-run-marker-count").textContent = String((evidence.run_markers || []).length);
  $("session-evidence-egm-count").textContent = String(evidence.egm_history_focused_count || 0);
  $("session-evidence-egm-groups").textContent = String(evidence.egm_grouped_summary_count || 0) + " / " + String(evidence.egm_grouped_summary_count_all || 0);
  $("session-evidence-heartbeat-count").textContent = String(evidence?.heartbeat_summary?.total || 0);
  $("session-evidence-heartbeat-health").textContent = evidence?.heartbeat_summary?.label || "-";
  $("session-evidence-heartbeat-source").textContent = (evidence?.operator_drill?.source || "-") + " | " + (evidence?.egm_focus?.label || "All EGMs");
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

function currentCabinetProfileSnapshot() {
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  return snapshot?.cabinetProfile?.effective || snapshot?.status?.cabinet_profile || {};
}

function defaultRunMarkerTitle(markerType) {
  const profile = currentCabinetProfileSnapshot();
  const hostID = profile.host_id || "cabinet";
  if (markerType === "start") return "Session start - " + hostID;
  if (markerType === "end") return "Session end - " + hostID;
  return "Operator note - " + hostID;
}

function operatorDrillHasFocus() {
  const form = $("operator-drill-form");
  return !!(form && form.contains(document.activeElement));
}

function currentOperatorDrill(snapshot) {
  const state = snapshot?.operatorDrill || {};
  const intervalMS = Number(state.interval_ms || currentRuntime().egm_heartbeat_interval_ms || 5000);
  const burstCount = Number(state.burst_count || 5);
  return {
    selected_egm_id: state.selected_egm_id || "",
    available_egm_ids: Array.isArray(state.available_egm_ids) ? state.available_egm_ids.slice() : [],
    auto_heartbeat_running: state.auto_heartbeat_running === true,
    auto_heartbeat_paused: state.auto_heartbeat_paused === true,
    interval_ms: intervalMS > 0 ? intervalMS : 5000,
    burst_count: burstCount > 0 ? burstCount : 5,
    last_action: state.last_action || "idle",
    last_action_at: state.last_action_at || ""
  };
}

function setOperatorDrillState(level, message) {
  const badge = $("operator-drill-state");
  badge.textContent = level;
  badge.className = "source-pill " + (level === "ready" || level === "saved" ? "source-file" : level === "working" ? "source-override" : "source-mixed");
  $("operator-drill-message").textContent = message;
}

function operatorDrillPayload(action) {
  return {
    action: action,
    egm_id: $("operator-drill-egm-id").value || "",
    interval_ms: Number($("operator-drill-interval-ms").value || 0),
    burst_count: Number($("operator-drill-burst-count").value || 0)
  };
}

function renderOperatorDrill(snapshot) {
  const drill = currentOperatorDrill(snapshot);
  const select = $("operator-drill-egm-id");
  const available = drill.available_egm_ids;
  if (available.length === 0) {
    select.innerHTML = "<option value=\"\">No configured EGM</option>";
  } else {
    select.innerHTML = available.map((id) => "<option value=\"" + escapeHTML(id) + "\">" + escapeHTML(id) + "</option>").join("");
  }
  if (!operatorDrillHasFocus()) {
    select.value = drill.selected_egm_id || (available[0] || "");
    $("operator-drill-interval-ms").value = String(drill.interval_ms || 5000);
    $("operator-drill-burst-count").value = String(drill.burst_count || 5);
  } else if (!select.value && available.length > 0) {
    select.value = available[0];
  }
  $("operator-drill-last-action").value = String(drill.last_action || "idle").replace(/_/g, " ");
  $("operator-drill-last-action-at").textContent = drill.last_action_at ? fmtTime(drill.last_action_at) : "-";

  let heartbeatState = "idle";
  if (drill.auto_heartbeat_running) {
    heartbeatState = "running every " + fmtDurationMs(drill.interval_ms);
  } else if (drill.auto_heartbeat_paused) {
    heartbeatState = "paused";
  }
  $("operator-drill-heartbeat-state").textContent = heartbeatState;

  const tokenRequired = setupActionsRequireToken();
  const tokenPresent = !!getSetupToken() || !!getCertToken();
  const blockedForToken = tokenRequired && !tokenPresent;
  const unavailable = available.length === 0;
  [
    "operator-drill-comms-online-button",
    "operator-drill-keepalive-button",
    "operator-drill-burst-button",
    "operator-drill-resume-button",
    "operator-drill-pause-button",
    "operator-drill-clear-button"
  ].forEach((id) => {
    $(id).disabled = unavailable || blockedForToken;
  });

  if (!snapshot?.status && unavailable) {
    setOperatorDrillState("waiting", "Waiting for appliance status before operator drill controls are available.");
    return;
  }
  if (unavailable) {
    setOperatorDrillState("blocked", "No configured EGM is available for operator drill.");
    return;
  }
  if (blockedForToken) {
    setOperatorDrillState("blocked", "Enter a setup or certificate API token before using operator drill actions.");
    return;
  }
  if (drill.auto_heartbeat_running) {
    setOperatorDrillState("ready", "Auto heartbeat drill is running at " + fmtDurationMs(drill.interval_ms) + ".");
    return;
  }
  if (drill.auto_heartbeat_paused) {
    setOperatorDrillState("ready", "Auto heartbeat drill is paused.");
    return;
  }
  setOperatorDrillState("ready", "Operator drill is ready for simulated cabinet traffic.");
}

async function submitOperatorDrillAction(action) {
  if (setupActionsRequireToken() && !getSetupToken() && !getCertToken()) {
    setOperatorDrillState("blocked", "Enter a setup or certificate API token before using operator drill actions.");
    return;
  }
  const payload = operatorDrillPayload(action);
  if (!payload.egm_id && action !== "pause" && action !== "clear") {
    setOperatorDrillState("blocked", "Choose an EGM before sending operator drill traffic.");
    return;
  }
  try {
    setOperatorDrillState("working", "Submitting operator drill action.");
    const headers = { "Content-Type": "application/json" };
    const token = getSetupToken() || getCertToken();
    if (token) {
      headers.Authorization = "Bearer " + token;
    }
    const response = await fetch(endpoints.operatorDrill, {
      method: "POST",
      headers: headers,
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setOperatorDrillState("blocked", "Operator drill failed: HTTP " + response.status + (detail ? " " + detail : ""));
      return;
    }
    const state = await response.json();
    const snapshot = copySnapshot(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
    snapshot.operatorDrill = state;
    clientState.displaySnapshot = snapshot;
    renderOperatorDrill(snapshot);
    setOperatorDrillState("saved", "Operator drill action sent: " + String(action).replace(/_/g, " ") + ".");
    schedulePoll(0);
  } catch (err) {
    setOperatorDrillState("blocked", err && err.message ? err.message : "Operator drill failed.");
  }
}

function setRunMarkerState(level, message) {
  const badge = $("run-marker-state");
  badge.textContent = level;
  badge.className = "source-pill " + (level === "ready" || level === "saved" ? "source-file" : level === "working" ? "source-override" : "source-mixed");
  $("run-marker-message").textContent = message;
}

function runMarkerPayload(markerType) {
  const profile = currentCabinetProfileSnapshot();
  const title = $("run-marker-title").value.trim() || defaultRunMarkerTitle(markerType);
  const operator = $("run-marker-operator").value.trim() || "lab-ui";
  return {
    created_at: new Date().toISOString(),
    marker_type: markerType,
    title: title,
    notes: $("run-marker-notes").value.trim(),
    host_id: profile.host_id || "",
    wire_host_url: profile.wire_host_url || "",
    operator: operator
  };
}

function renderRunMarkerControls(snapshot) {
  const runtime = snapshot?.status?.runtime || currentRuntime();
  const tokenRequired = runtime.api_mutation_auth_required === true;
  const tokenPresent = !!getSetupToken() || !!getCertToken();
  $("run-marker-start-button").disabled = tokenRequired && !tokenPresent;
  $("run-marker-note-button").disabled = tokenRequired && !tokenPresent;
  $("run-marker-end-button").disabled = tokenRequired && !tokenPresent;
  if (tokenRequired) {
    setRunMarkerState("ready", "Run markers require an API token in this browser session.");
  } else {
    setRunMarkerState("ready", "Run markers are ready for trusted lab mode.");
  }
}

async function submitRunMarker(markerType) {
  if (setupActionsRequireToken() && !getSetupToken() && !getCertToken()) {
    setRunMarkerState("blocked", "Enter an API token before writing run markers.");
    return;
  }
  const payload = runMarkerPayload(markerType);
  try {
    setRunMarkerState("working", "Saving run marker.");
    const token = getSetupToken() || getCertToken();
    const headers = { "Content-Type": "application/json" };
    if (token) {
      headers.Authorization = "Bearer " + token;
    }
    const response = await fetch(endpoints.runMarkers.replace("?limit=12", ""), {
      method: "POST",
      headers: headers,
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setRunMarkerState("blocked", "Run marker failed: HTTP " + response.status + (detail ? " " + detail : ""));
      return;
    }
    $("run-marker-title").value = "";
    $("run-marker-notes").value = "";
    setRunMarkerState("saved", "Run marker saved to cabinet timeline.");
    schedulePoll(0);
  } catch (err) {
    setRunMarkerState("blocked", err && err.message ? err.message : "Run marker failed.");
  }
}

function buildCabinetRunTimeline(snapshot) {
  const incidents = Array.isArray(snapshot?.incidents) ? snapshot.incidents : [];
  const egmHistory = Array.isArray(snapshot?.egmHistory) ? snapshot.egmHistory : [];
  const stateHistory = Array.isArray(snapshot?.stateHistory) ? snapshot.stateHistory : [];
  const runMarkers = Array.isArray(snapshot?.runMarkers) ? snapshot.runMarkers : [];
  const timeline = [];

  incidents.forEach((item) => {
    timeline.push({
      kind: "incident",
      egm_id: "",
      global: true,
      createdAt: item.created_at || "",
      sortTime: item.created_at ? new Date(item.created_at).getTime() : 0,
      title: "#" + String(item.id || "-") + " " + String(item.trigger_type || "incident"),
      detail: (item.trigger_source || "unknown source") + " | final state " + String(item.final_state || "-"),
      meta: item.resolved_at ? ("resolved " + fmtTime(item.resolved_at)) : "active or unresolved"
    });
  });

  egmHistory.forEach((item) => {
    const heartbeat = isHeartbeatEventType(item.event_type);
    const keepAliveRollup = historyRowIsKeepAliveRollup(item);
    const keepAliveCount = normalizeHeartbeatRollupCount(item);
    const firstSeen = historyRowFirstSeenAt(item);
    const lastSeen = historyRowLastSeenAt(item);
    const rollupDuration = historyRowDurationMS(item);
    const rollupLabel = keepAliveRollup
      ? ("keepAlive x" + String(keepAliveCount) + " over " + fmtDurationMs(rollupDuration))
      : "";
    timeline.push({
      kind: heartbeat ? "heartbeat" : "egm",
      egm_id: String(item.egm_id || "").trim(),
      global: false,
      createdAt: item.created_at || "",
      sortTime: item.created_at ? new Date(item.created_at).getTime() : 0,
      title: keepAliveRollup
        ? (String(item.egm_id || "-") + " " + rollupLabel)
        : (String(item.egm_id || "-") + " " + (heartbeat ? String(item.event_type || "heartbeat") : String(item.status || "-"))),
      detail: keepAliveRollup
        ? ("rolled-up heartbeat bucket | first " + fmtTime(firstSeen) + " | last " + fmtTime(lastSeen))
        : (String(item.event_type || "-") + (item.detail ? " | " + String(item.detail) : "")),
      meta: heartbeat
        ? (keepAliveRollup ? "heartbeat rollup traffic" : "heartbeat traffic")
        : (item.last_error ? ("last error: " + String(item.last_error)) : "egm history"),
      rolled_up: keepAliveRollup
    });
  });

  stateHistory.forEach((item) => {
    timeline.push({
      kind: "state",
      egm_id: "",
      global: true,
      createdAt: item.created_at || "",
      sortTime: item.created_at ? new Date(item.created_at).getTime() : 0,
      title: String(item.old_state || "-") + " -> " + String(item.new_state || "-"),
      detail: String(item.reason || "state transition"),
      meta: "controller state history"
    });
  });

  runMarkers.forEach((item) => {
    timeline.push({
      kind: "marker",
      egm_id: "",
      global: true,
      createdAt: item.created_at || "",
      sortTime: item.created_at ? new Date(item.created_at).getTime() : 0,
      title: String(item.title || item.marker_type || "marker"),
      detail: (item.notes ? String(item.notes) + " | " : "") + (item.host_id || "-") + " | " + (item.wire_host_url || "-"),
      meta: (item.operator || "operator") + " | " + String(item.marker_type || "marker")
    });
  });

  timeline.sort((a, b) => b.sortTime - a.sortTime);
  return timeline;
}

function applyTimelineFocus(items) {
  const focusID = currentEGMFocusID();
  if (!focusID) {
    return items.slice();
  }
  return items.filter((item) => item.global === true || String(item.egm_id || "").trim() === focusID);
}

function applyTimelineFilter(items) {
  if (clientState.timelineFilter === "all") {
    return items.filter((item) =>
      item.kind !== "heartbeat" ||
      item.rolled_up === true ||
      clientState.timelineShowRawHeartbeat === true
    );
  }
  return items.filter((item) => item.kind === clientState.timelineFilter);
}

function updateTimelineFilterLabels(total, filtered, heartbeatCount) {
  const focusID = currentEGMFocusID();
  const map = {
    all: heartbeatCount > 0 ? ("Showing all timeline events with " + heartbeatCount + " heartbeat row(s) filtered by timeline mode") : "Showing all timeline events",
    incident: "Showing incident timeline events",
    egm: "Showing EGM timeline events",
    heartbeat: effectiveHeartbeatRollupEnabled()
      ? "Showing rolled-up heartbeat timeline events"
      : "Showing raw heartbeat timeline events",
    state: "Showing controller state timeline events",
    marker: "Showing operator run markers"
  };
  $("timeline-count").textContent = filtered + " / " + total + " events";
  $("timeline-filter-label").textContent = (focusID ? ("Focus " + focusID + " | ") : "") + (map[clientState.timelineFilter] || map.all);
  $("timeline-heartbeat-mode-label").textContent = "Heartbeat mode: " +
    (effectiveHeartbeatRollupEnabled() ? "rolled up" : "raw events") +
    (clientState.timelineShowRawHeartbeat ? " (show raw enabled)" : "");
  document.querySelectorAll(".timeline-filter-tab").forEach((button) => {
    button.classList.toggle("is-active", (button.dataset.timelineFilter || "all") === clientState.timelineFilter);
  });
}

function timelineEntryHTML(item) {
  const egmID = String(item?.egm_id || "").trim();
  const globalChip = item?.global ? "<span class=\"timeline-scope timeline-scope-global\">global</span>" : "";
  const egmChip = egmID ? "<span class=\"timeline-egm-chip\">" + escapeHTML(egmID) + "</span>" : "";
  return "<div class=\"item timeline-entry\">" +
    "<div class=\"timeline-entry-head\"><strong>" + escapeHTML(item?.title || "-") + "</strong><div class=\"timeline-entry-tags\">" + globalChip + egmChip + "<span class=\"timeline-kind timeline-kind-" + escapeHTML(item?.kind || "egm") + "\">" + escapeHTML(item?.kind || "egm") + "</span></div></div>" +
    "<span>" + escapeHTML(item?.detail || "-") + "</span>" +
    "<span>" + escapeHTML(fmtTime(item?.createdAt || "")) + " | " + escapeHTML(item?.meta || "-") + "</span>" +
  "</div>";
}

function renderGroupedTimelineAll(items) {
  const list = Array.isArray(items) ? items : [];
  if (list.length === 0) {
    $("cabinet-run-timeline").innerHTML = "<div class=\"empty\">No cabinet run events captured yet</div>";
    return;
  }
  const globalItems = [];
  const byEGM = {};
  list.forEach((item) => {
    const egmID = String(item?.egm_id || "").trim();
    if (item?.global || !egmID) {
      globalItems.push(item);
      return;
    }
    if (!byEGM[egmID]) byEGM[egmID] = [];
    byEGM[egmID].push(item);
  });
  let html = "";
  if (globalItems.length > 0) {
    html += "<div class=\"timeline-group-heading timeline-group-heading-global\">Global Session Events (" + String(globalItems.length) + ")</div>";
    html += globalItems.map((item) => timelineEntryHTML(item)).join("");
  }
  Object.keys(byEGM).sort((a, b) => a.localeCompare(b)).forEach((egmID) => {
    const records = byEGM[egmID];
    html += "<div class=\"timeline-group-heading timeline-group-heading-egm\">EGM " + escapeHTML(egmID) + " (" + String(records.length) + ")</div>";
    html += records.map((item) => timelineEntryHTML(item)).join("");
  });
  $("cabinet-run-timeline").innerHTML = html || "<div class=\"empty\">No cabinet run events captured yet</div>";
}

function markerLabel(marker) {
  if (!marker) return "-";
  return (marker.marker_type || "marker").toUpperCase() + " | " + fmtTime(marker.created_at) + " | " + (marker.title || "-");
}

function findRunMarkerByID(id, snapshot) {
  const records = Array.isArray(snapshot?.runMarkers) ? snapshot.runMarkers : [];
  for (let i = 0; i < records.length; i++) {
    if (String(records[i]?.id) === String(id)) {
      return records[i];
    }
  }
  return null;
}

function normalizeRunReportSelections(snapshot) {
  const markers = Array.isArray(snapshot?.runMarkers) ? snapshot.runMarkers : [];
  if (markers.length === 0) {
    clientState.selectedRunReportStartID = 0;
    clientState.selectedRunReportEndID = 0;
    return { start: null, end: null };
  }
  if (!findRunMarkerByID(clientState.selectedRunReportStartID, snapshot)) {
    clientState.selectedRunReportStartID = markers[markers.length - 1].id || markers[0].id || 0;
  }
  if (!findRunMarkerByID(clientState.selectedRunReportEndID, snapshot)) {
    clientState.selectedRunReportEndID = markers[0].id || 0;
  }
  return {
    start: findRunMarkerByID(clientState.selectedRunReportStartID, snapshot),
    end: findRunMarkerByID(clientState.selectedRunReportEndID, snapshot)
  };
}

function recordInRange(createdAt, startTime, endTime) {
  if (!createdAt) return false;
  const ts = new Date(createdAt).getTime();
  return ts >= startTime && ts <= endTime;
}

function boundedRunReport(snapshot) {
  const focus = egmFocusScope(snapshot);
  const session = buildFirstCabinetSessionState(snapshot);
  const workflow = buildCabinetSessionWorkflow(snapshot, session);
  const actionModel = buildOperatorReadinessModel(snapshot, session, workflow);
  const mutePathStatus = buildMutePathStatus(snapshot, session, actionModel, workflow);
  const selectedEGMDetail = selectedEGMDetailForSnapshot(snapshot);
  const selection = normalizeRunReportSelections(snapshot);
  if (!selection.start || !selection.end) {
    return null;
  }
  const firstTime = new Date(selection.start.created_at).getTime();
  const secondTime = new Date(selection.end.created_at).getTime();
  const startMarker = firstTime <= secondTime ? selection.start : selection.end;
  const endMarker = firstTime <= secondTime ? selection.end : selection.start;
  const startTime = new Date(startMarker.created_at).getTime();
  const endTime = new Date(endMarker.created_at).getTime();
  const profile = currentCabinetProfileSnapshot();
  const incidents = (snapshot?.incidents || []).filter((item) => recordInRange(item.created_at, startTime, endTime));
  const egmHistoryAll = (snapshot?.egmHistory || []).filter((item) => recordInRange(item.created_at, startTime, endTime));
  const egmHistory = filterHistoryByFocus(egmHistoryAll);
  const stateHistory = (snapshot?.stateHistory || []).filter((item) => recordInRange(item.created_at, startTime, endTime));
  const runMarkers = (snapshot?.runMarkers || []).filter((item) => recordInRange(item.created_at, startTime, endTime));
  const sessionEvidence = (snapshot?.sessionEvidence || []).filter((item) => recordInRange(item.created_at, startTime, endTime));
  const drillState = currentOperatorDrill(snapshot);
  const heartbeat = heartbeatSummary(egmHistory, currentHeartbeatPolicy(snapshot), endMarker.created_at || "");
  const drillEvidence = operatorDrillEvidence(egmHistory, drillState);
  const groupedSummaryAll = buildEGMGroupedSummaryRows(snapshot?.status?.egms, egmHistoryAll, currentHeartbeatPolicy(snapshot), endMarker.created_at || "");
  const groupedSummaryFocused = groupedRowsForCurrentFocus(groupedSummaryAll);
  return {
    generated_at: new Date().toISOString(),
    egm_focus: focus,
    scope: {
      egm_history_scope: focus.egm_specific_views_filtered ? "FILTERED_TO_EGM" : "FULL_SESSION",
      selected_egm_id: focus.selected_egm_id || "",
      grouped_summary_scope: focus.egm_specific_views_filtered ? "FILTERED_TO_EGM" : "FULL_SESSION",
      global_sections_scope: "FULL_SESSION_GLOBAL_INCLUDED"
    },
    workflow: workflow,
    selected_egm_detail: selectedEGMDetail,
    action_model: actionModel,
    mute_path: mutePathStatus,
    runbook_readiness: {
      state: mutePathStatus.runbook_readiness_state || "UNKNOWN",
      blocker_count: mutePathStatus.runbook_blocker_count || 0,
      warning_count: mutePathStatus.runbook_warning_count || 0,
      can_continue_cabinet_prep: mutePathStatus.can_continue_cabinet_prep === true,
      next_action: mutePathStatus.next_action || ""
    },
    next_operator_actions: actionModel.next_actions,
    lab_warnings: actionModel.lab_warning,
    window: {
      start_marker: startMarker,
      end_marker: endMarker,
      started_at: startMarker.created_at,
      ended_at: endMarker.created_at,
      duration_seconds: Math.max(0, Math.floor((endTime - startTime) / 1000))
    },
    cabinet_profile: {
      host_id: profile.host_id || "",
      wire_host_url: profile.wire_host_url || ""
    },
    summary: {
      incidents: incidents.length,
      egm_events: egmHistory.length,
      egm_events_total: egmHistoryAll.length,
      egm_groups: groupedSummaryFocused.length,
      egm_groups_total: groupedSummaryAll.length,
      heartbeat_events: heartbeat.total,
      state_changes: stateHistory.length,
      run_markers: runMarkers.length,
      saved_evidence: sessionEvidence.length
    },
    heartbeat_summary: heartbeat,
    operator_drill: drillEvidence,
    incidents: incidents,
    egm_history: egmHistory,
    egm_history_all: egmHistoryAll,
    egm_grouped_summary: groupedSummaryFocused,
    egm_grouped_summary_all: groupedSummaryAll,
    state_history: stateHistory,
    run_markers: runMarkers,
    saved_evidence: sessionEvidence
  };
}

function buildRunReportMarkdown(report) {
  const workflowSteps = Array.isArray(report?.workflow?.steps) ? report.workflow.steps : [];
  const groupedRows = Array.isArray(report?.egm_grouped_summary) ? report.egm_grouped_summary : [];
  const groupedRowsAll = Array.isArray(report?.egm_grouped_summary_all) ? report.egm_grouped_summary_all : [];
  const actionItems = Array.isArray(report?.action_model?.needs_operator_action) ? report.action_model.needs_operator_action : [];
  const labWarnings = Array.isArray(report?.action_model?.lab_warning) ? report.action_model.lab_warning : [];
  const nextActions = Array.isArray(report?.next_operator_actions) ? report.next_operator_actions : [];
  const selectedEGMDetail = report?.selected_egm_detail || {};
  const lines = [
    "# Cabinet Run Report",
    "",
    "- Generated at: " + (report.generated_at || "-"),
    "- EGM focus: " + (report?.egm_focus?.label || "All EGMs"),
    "- EGM history scope: " + (report?.scope?.egm_history_scope || "FULL_SESSION"),
    "- Grouped summary scope: " + (report?.scope?.grouped_summary_scope || "FULL_SESSION"),
    "- Runbook readiness state: " + (report?.runbook_readiness?.state || "UNKNOWN"),
    "- Runbook prep can continue: " + String(report?.runbook_readiness?.can_continue_cabinet_prep === true),
    "- Mute path state: " + (report?.mute_path?.mute_path_state || "UNKNOWN"),
    "- Mute path note: " + (report?.mute_path?.mute_path_note || "-"),
    "- Host ID: " + (report.cabinet_profile.host_id || "-"),
    "- Wire host URL: " + (report.cabinet_profile.wire_host_url || "-"),
    "- Start marker: " + (report.window.start_marker.title || "-"),
    "- Start time: " + (report.window.started_at || "-"),
    "- End marker: " + (report.window.end_marker.title || "-"),
    "- End time: " + (report.window.ended_at || "-"),
    "- Duration (s): " + String(report.window.duration_seconds || 0),
    "- Incidents: " + String(report.summary.incidents || 0),
    "- EGM events (focused scope): " + String(report.summary.egm_events || 0),
    "- EGM events (all EGMs in window): " + String(report.summary.egm_events_total || 0),
    "- EGM groups (focused scope): " + String(report.summary.egm_groups || 0),
    "- EGM groups (all EGMs in window): " + String(report.summary.egm_groups_total || 0),
    "- Heartbeat events: " + String(report.summary.heartbeat_events || 0),
    "- State changes: " + String(report.summary.state_changes || 0),
    "- Run markers: " + String(report.summary.run_markers || 0),
    "- Saved evidence captures: " + String(report.summary.saved_evidence || 0),
    "- Heartbeat health: " + (report?.heartbeat_summary?.label || "-"),
    "- Heartbeat source: " + (report?.operator_drill?.source || "-"),
    "",
    "## Heartbeat Summary",
    "",
    "- Total events: " + String(report?.heartbeat_summary?.total || 0),
    "- keepAlive events: " + String(report?.heartbeat_summary?.keepalive_count || 0),
    "- Last keepAlive: " + (report?.heartbeat_summary?.last_keepalive_at || "-"),
    "- Max gap: " + fmtDurationMs(report?.heartbeat_summary?.max_gap_ms || 0),
    "- Notes: " + (report?.heartbeat_summary?.message || "-"),
    "",
    "## Operator Drill",
    "",
    "- Source: " + (report?.operator_drill?.source || "-"),
    "- Drill events: " + String(report?.operator_drill?.drill_events || 0),
    "- Live events: " + String(report?.operator_drill?.live_events || 0),
    "- Drill EGM IDs: " + (((report?.operator_drill?.egm_ids || []).join(", ")) || "-"),
    "- Auto heartbeat running: " + String(report?.operator_drill?.state?.auto_heartbeat_running === true),
    "- Auto heartbeat paused: " + String(report?.operator_drill?.state?.auto_heartbeat_paused === true),
    "",
    "## Mute Path Note",
    "",
    "- " + (report?.mute_path?.mute_path_note || "Mute path status unavailable."),
    "- " + (report?.mute_path?.confidence_note || "Software signal only."),
    "",
    "## Runbook Readiness",
    "",
    "- State: " + (report?.runbook_readiness?.state || "UNKNOWN"),
    "- Blocker count: " + String(report?.runbook_readiness?.blocker_count || 0),
    "- Lab warning count: " + String(report?.runbook_readiness?.warning_count || 0),
    "- Next action: " + (report?.runbook_readiness?.next_action || "-"),
    "",
    "## Workflow",
    "",
    "- Current step: " + (report?.workflow?.current_step || "-"),
    "- Focus mode: " + (report?.workflow?.focus_mode || "-")
  ];
  if (workflowSteps.length > 0) {
    workflowSteps.forEach((step) => lines.push("- " + [step.title || "-", step.state || "-", step.detail || ""].filter(Boolean).join(" | ")));
  } else {
    lines.push("- No workflow steps available");
  }
  lines.push("", "## Action Items", "");
  if (actionItems.length > 0) {
    actionItems.forEach((item) => lines.push("- " + item));
  } else {
    lines.push("- None");
  }
  lines.push("", "## Lab Warnings", "");
  if (labWarnings.length > 0) {
    labWarnings.forEach((item) => lines.push("- " + item));
  } else {
    lines.push("- None");
  }
  lines.push("", "## Next Operator Actions", "");
  if (nextActions.length > 0) {
    nextActions.forEach((item) => lines.push("- " + item));
  } else {
    lines.push("- None");
  }
  lines.push("", "## Selected EGM Detail", "");
  if (selectedEGMDetail?.egm_id) {
    lines.push("- EGM ID: " + selectedEGMDetail.egm_id);
    lines.push("- Source: " + (selectedEGMDetail.source || "-"));
    lines.push("- Status: " + (selectedEGMDetail.status || "-"));
    lines.push("- Live signal: " + (selectedEGMDetail.live_signal || "-"));
    lines.push("- Live signal detail: " + (selectedEGMDetail.live_signal_detail || "-"));
    lines.push("- Last endpoint: " + ((selectedEGMDetail.last_endpoint_ip || "-") + ":" + (selectedEGMDetail.last_endpoint_port || "-")));
    lines.push("- Endpoint seen: " + (selectedEGMDetail.last_endpoint_seen_at || "-"));
    lines.push("- Endpoint drift warning: " + String(selectedEGMDetail.endpoint_drift_warning === true));
    lines.push("- Endpoint drift IPs: " + ((selectedEGMDetail.endpoint_drift_ips || []).join(", ") || "-"));
    lines.push("- Last seen: " + (selectedEGMDetail.last_seen_at || "-"));
    lines.push("- Heartbeat: " + (selectedEGMDetail.heartbeat_label || "-"));
    lines.push("- Last keepAlive: " + (selectedEGMDetail.heartbeat_last_keepalive_at || "-"));
    lines.push("- First-test set: " + String(selectedEGMDetail.in_first_test_set === true));
  } else {
    lines.push("- " + (selectedEGMDetail?.message || "No selected EGM detail available."));
  }
  lines.push("", "## Grouped EGM Summary", "");
  lines.push("- Scope rows: " + String(groupedRows.length));
  lines.push("- All-window rows: " + String(groupedRowsAll.length));
  if (groupedRows.length > 0) {
    groupedRows.forEach((row) => lines.push("- " + [
      row.egm_id || "-",
      row.source || "-",
      row.status || "-",
      "events=" + String(row.total_events || 0),
      "heartbeat=" + String(row.heartbeat_label || "-"),
      "last_seen=" + String(row.last_seen_at || "-")
    ].join(" | ")));
  } else {
    lines.push("- None");
  }
  lines.push("", "## JSON Payload", "", "~~~json", JSON.stringify(report, null, 2), "~~~");
  return lines.join("\n");
}

function runReportFilenameBase(report) {
  const hostID = String(report?.cabinet_profile?.host_id || "cabinet-run").replace(/[^A-Za-z0-9._-]+/g, "-");
  const startID = report?.window?.start_marker?.id || "start";
  const endID = report?.window?.end_marker?.id || "end";
  return hostID + "-run-report-" + startID + "-to-" + endID;
}

function renderRunReportControls(snapshot) {
  const markers = Array.isArray(snapshot?.runMarkers) ? snapshot.runMarkers : [];
  const startSelect = $("run-report-start-marker");
  const endSelect = $("run-report-end-marker");
  const selection = normalizeRunReportSelections(snapshot);
  startSelect.innerHTML = markers.map((marker) => "<option value=\"" + escapeHTML(marker.id) + "\">" + escapeHTML(markerLabel(marker)) + "</option>").join("");
  endSelect.innerHTML = markers.map((marker) => "<option value=\"" + escapeHTML(marker.id) + "\">" + escapeHTML(markerLabel(marker)) + "</option>").join("");
  if (selection.start) startSelect.value = String(selection.start.id);
  if (selection.end) endSelect.value = String(selection.end.id);
  const report = boundedRunReport(snapshot);
  const enabled = !!report;
  $("run-report-json-button").disabled = !enabled;
  $("run-report-markdown-button").disabled = !enabled;
  if (!report) {
    $("run-report-window-summary").textContent = "Need at least one saved run marker.";
    $("run-report-count-summary").textContent = "-";
    $("run-report-state").textContent = "waiting";
    $("run-report-state").className = "source-pill source-mixed";
    $("run-report-message").textContent = "Mark a run start/end before exporting a bounded report.";
    return;
  }
  $("run-report-window-summary").textContent = fmtTime(report.window.started_at) + " -> " + fmtTime(report.window.ended_at) + " (" + report.window.duration_seconds + "s)";
  $("run-report-count-summary").textContent = "focus " + (report?.egm_focus?.label || "All EGMs") + " [" + (report?.scope?.egm_history_scope || "FULL_SESSION") + "], incidents " + report.summary.incidents + ", egm events " + report.summary.egm_events + "/" + report.summary.egm_events_total + ", egm groups " + report.summary.egm_groups + "/" + report.summary.egm_groups_total + ", heartbeat " + report.summary.heartbeat_events + " (" + (report?.operator_drill?.source || "-") + "), state " + report.summary.state_changes + ", markers " + report.summary.run_markers + ", evidence " + report.summary.saved_evidence;
  $("run-report-state").textContent = "ready";
  $("run-report-state").className = "source-pill source-file";
  $("run-report-message").textContent = "Run report window is ready to export. Heartbeat: " + (report?.heartbeat_summary?.label || "-") + ". Workflow step: " + (report?.workflow?.current_step || "-") + ".";
}

function renderHeartbeatSummary(snapshot) {
  const focusLabel = currentEGMFocusLabel();
  const focusedHistory = filterHistoryByFocus(snapshot?.egmHistory || []);
  const summary = heartbeatSummary(focusedHistory, currentHeartbeatPolicy(snapshot), new Date().toISOString());
  $("heartbeat-scope").textContent = focusLabel;
  $("heartbeat-health").textContent = summary.label;
  $("heartbeat-observed").textContent = summary.total + " total / " + summary.keepalive_count + " keepAlive";
  $("heartbeat-last-keepalive").textContent = summary.last_keepalive_at ? fmtTime(summary.last_keepalive_at) : "-";
  $("heartbeat-max-gap").textContent = fmtDurationMs(summary.max_gap_ms || 0);
  const intervalText = summary.interval_ms > 0 ? ("configured " + fmtDurationMs(summary.interval_ms)) : "configured interval unavailable";
  $("heartbeat-summary-message").textContent = (currentEGMFocusID() ? ("Focused on " + focusLabel + ". ") : "") + summary.message + " (" + intervalText + ")";
}

function renderEGMHistory(snapshot) {
  const focusID = currentEGMFocusID();
  const scopedHistory = filterHistoryByFocus(snapshot?.egmHistory || []);
  const grouped = {};
  scopedHistory.forEach((item) => {
    const egmID = String(item?.egm_id || "").trim();
    if (!egmID) return;
    if (!grouped[egmID]) {
      grouped[egmID] = { egm_id: egmID, records: [], non_heartbeat: [] };
    }
    grouped[egmID].records.push(item);
    if (!isHeartbeatEventType(item?.event_type)) {
      grouped[egmID].non_heartbeat.push(item);
    }
  });

  const heartbeatRowsForDisplay = (rows) => {
    const heartbeatRows = (Array.isArray(rows) ? rows : []).filter((item) => isHeartbeatEventType(item?.event_type));
    if (clientState.timelineShowRawHeartbeat === true) {
      return heartbeatRows.slice();
    }
    return heartbeatRows.filter((item) =>
      historyRowIsKeepAliveRollup(item) ||
      String(item?.event_type || "").toUpperCase() === "G2S_SESSION_ONLINE"
    );
  };

  const historyRowHTML = (item) => {
    const rolled = historyRowIsKeepAliveRollup(item);
    const eventType = String(item?.event_type || "-");
    const summaryText = rolled
      ? ("keepAlive x" + String(normalizeHeartbeatRollupCount(item)) + " over " + fmtDurationMs(historyRowDurationMS(item)) +
        " | first " + fmtTime(historyRowFirstSeenAt(item)) +
        " | last " + fmtTime(historyRowLastSeenAt(item)))
      : eventType + (item?.detail ? " | " + String(item.detail) : "");
    return "<div class=\"item timeline-entry\">" +
      "<div class=\"timeline-entry-head\"><strong>" + escapeHTML(item.egm_id || "-") + (item.status ? (" " + statusPill(item.status)) : "") + "</strong><div class=\"timeline-entry-tags\"><span class=\"timeline-egm-chip\">" + escapeHTML(item.egm_id || "-") + "</span></div></div>" +
      "<span>" + escapeHTML(summaryText) + " at " + escapeHTML(fmtTime(item.created_at)) + "</span>" +
    "</div>";
  };

  if (focusID) {
    const bucket = grouped[focusID] || { egm_id: focusID, records: [], non_heartbeat: [] };
    const heartbeat = heartbeatSummary(bucket.records, currentHeartbeatPolicy(snapshot), new Date().toISOString());
    const rows = [];
    if (heartbeat.total > 0) {
      rows.push({
        egm_id: focusID,
        status: "",
        event_type: heartbeat.label,
        created_at: heartbeat.last_keepalive_at || heartbeat.first_comms_online_at || "",
        detail: heartbeat.message,
        heartbeat_summary: true
      });
    }
    const heartbeatRows = heartbeatRowsForDisplay(bucket.records)
      .slice()
      .sort((a, b) => numericTime(b?.created_at) - numericTime(a?.created_at));
    heartbeatRows.forEach((record) => rows.push(record));
    bucket.non_heartbeat
      .slice()
      .sort((a, b) => numericTime(b?.created_at) - numericTime(a?.created_at))
      .forEach((record) => rows.push(record));
    renderItems("egm-history", rows, "No EGM history yet", (item) =>
      item?.heartbeat_summary === true
        ? ("<div class=\"item timeline-entry\">" +
          "<div class=\"timeline-entry-head\"><strong>" + escapeHTML(item.egm_id || "-") + "</strong><div class=\"timeline-entry-tags\"><span class=\"timeline-egm-chip\">" + escapeHTML(item.egm_id || "-") + "</span><span class=\"timeline-kind timeline-kind-heartbeat\">heartbeat</span></div></div>" +
          "<span>" + escapeHTML(item.event_type || "-") + " at " + escapeHTML(fmtTime(item.created_at)) + (item.detail ? " | " + escapeHTML(item.detail) : "") + "</span>" +
          "</div>")
        : historyRowHTML(item)
    );
    return;
  }

  const egmIDs = Object.keys(grouped).sort((a, b) => a.localeCompare(b));
  if (egmIDs.length === 0) {
    $("egm-history").innerHTML = "<div class=\"empty\">No EGM history yet</div>";
    return;
  }
  let html = "";
  egmIDs.forEach((egmID) => {
    const bucket = grouped[egmID];
    const heartbeat = heartbeatSummary(bucket.records, currentHeartbeatPolicy(snapshot), new Date().toISOString());
    const heartbeatRows = heartbeatRowsForDisplay(bucket.records)
      .slice()
      .sort((a, b) => numericTime(b?.created_at) - numericTime(a?.created_at));
    const nonHeartbeatRows = bucket.non_heartbeat
      .slice()
      .sort((a, b) => numericTime(b?.created_at) - numericTime(a?.created_at));
    html += "<div class=\"timeline-group-heading timeline-group-heading-egm\">EGM " + escapeHTML(egmID) + " (" + String(bucket.records.length) + ")</div>";
    if (heartbeat.total > 0) {
      html += "<div class=\"item timeline-entry\"><div class=\"timeline-entry-head\"><strong>" + escapeHTML(egmID) + " heartbeat</strong><div class=\"timeline-entry-tags\"><span class=\"timeline-egm-chip\">" + escapeHTML(egmID) + "</span><span class=\"timeline-kind timeline-kind-heartbeat\">heartbeat</span></div></div><span>" + escapeHTML(heartbeat.label) + " at " + escapeHTML(fmtTime(heartbeat.last_keepalive_at || heartbeat.first_comms_online_at || "")) + "</span><span>" + escapeHTML(heartbeat.message) + "</span></div>";
    }
    if (heartbeatRows.length > 0) {
      html += heartbeatRows.map((item) => historyRowHTML(item)).join("");
    }
    if (nonHeartbeatRows.length === 0 && heartbeatRows.length === 0) {
      html += "<div class=\"item\"><span>No non-heartbeat rows yet for this EGM.</span></div>";
      return;
    }
    html += nonHeartbeatRows.map((item) => historyRowHTML(item)).join("");
  });
  $("egm-history").innerHTML = html;
}

function operatorAuditEndpointURL() {
  const params = new URLSearchParams();
  params.set("limit", "200");
  const action = String(clientState.operatorAuditActionFilter || "").trim();
  const result = String(clientState.operatorAuditResultFilter || "").trim();
  const search = String(clientState.operatorAuditSearchFilter || "").trim();
  if (action) params.set("action", action);
  if (result) params.set("result", result);
  if (search) params.set("q", search);
  return endpoints.operatorAudit + "?" + params.toString();
}

function normalizeOperatorAuditEvents(rows) {
  return (Array.isArray(rows) ? rows : []).map((item) => ({
    id: Number(item?.id || 0),
    timestamp: item?.timestamp || "",
    action: String(item?.action || "").trim(),
    result: String(item?.result || "").toLowerCase(),
    actor_scope: String(item?.actor_scope || "").trim(),
    egm_focus: String(item?.egm_focus || "").trim(),
    summary: String(item?.summary || "").trim(),
    detail: String(item?.detail || "").trim()
  })).filter((item) => item.id > 0 && item.action);
}

function operatorAuditResultClass(result) {
  return String(result || "").toLowerCase() === "success" ? "operator-audit-pill-success" : "operator-audit-pill-fail";
}

function syncOperatorAuditFilterControls() {
  $("operator-audit-action-filter").value = clientState.operatorAuditActionFilter || "";
  $("operator-audit-result-filter").value = clientState.operatorAuditResultFilter || "";
  $("operator-audit-search-filter").value = clientState.operatorAuditSearchFilter || "";
}

function renderOperatorAuditTimeline(snapshot) {
  syncOperatorAuditFilterControls();
  const sourceRows = normalizeOperatorAuditEvents(snapshot?.operatorAudit || []);
  const rows = sourceRows.filter((item) => operatorAuditMatchesGlobalFilters(item));
  const parts = [];
  if (clientState.operatorAuditActionFilter) parts.push("action=" + clientState.operatorAuditActionFilter);
  if (clientState.operatorAuditResultFilter) parts.push("result=" + clientState.operatorAuditResultFilter);
  if (clientState.operatorAuditSearchFilter) parts.push("q=" + clientState.operatorAuditSearchFilter);
  const severityFilter = normalizeGlobalSeverityFilter(clientState.globalSeverityFilter);
  const globalText = String(clientState.globalTextFilter || "").trim();
  if (severityFilter !== "all") parts.push("global_severity=" + severityFilter);
  if (globalText) parts.push("global_q=" + globalText);
  $("operator-audit-summary").textContent = rows.length
    ? ("Showing " + String(rows.length) + " / " + String(sourceRows.length) + " audit event(s)" + (parts.length ? " (" + parts.join(", ") + ")" : "") + ".")
    : ("No audit events found" + (parts.length ? " for current filters." : "."));
  $("operator-audit-state").textContent = rows.length > 0 ? "ready" : "idle";
  $("operator-audit-state").className = "source-pill " + (rows.length > 0 ? "source-file" : "source-mixed");
  $("operator-audit-message").textContent = "Sensitive operator actions are recorded here.";
  renderItems("operator-audit-list", rows, "No operator audit events yet.", (item) => {
    const resultText = item.result || "fail";
    const meta = [
      "actor_scope=" + (item.actor_scope || "-"),
      "egm_focus=" + (item.egm_focus || "all"),
      "id=" + String(item.id)
    ];
    return "<div class=\"item operator-audit-entry\">" +
      "<details>" +
      "<summary>" +
      "<div class=\"operator-audit-head\"><strong>" + escapeHTML(item.action) + "</strong><span class=\"" + operatorAuditResultClass(resultText) + "\">" + escapeHTML(resultText) + "</span></div>" +
      "<div class=\"operator-audit-meta\">" + escapeHTML(fmtTime(item.timestamp) + " | " + meta.join(" | ")) + "</div>" +
      "<div class=\"operator-audit-meta\">" + escapeHTML(item.summary || "-") + "</div>" +
      "</summary>" +
      "<div class=\"operator-audit-detail\">" + escapeHTML(item.detail || "No additional detail.") + "</div>" +
      "</details>" +
      "</div>";
  });
}

function heartbeatPolicyHasFocus() {
  const form = $("heartbeat-policy-form");
  return !!(form && form.contains(document.activeElement));
}

function fillHeartbeatPolicyForm(policyResponse) {
  const effective = policyResponse?.effective || currentHeartbeatPolicy({ heartbeatPolicy: policyResponse });
  $("heartbeat-policy-interval").value = String(effective.interval_ms || 0);
  $("heartbeat-policy-warning-after-missed").value = String(effective.warning_after_missed || 3);
  $("heartbeat-policy-block-after-missed").value = String(effective.block_after_missed || 6);
  $("heartbeat-policy-updated").value = policyResponse?.policy_last_updated_at ? fmtTime(policyResponse.policy_last_updated_at) : "file baseline";
  renderHeartbeatPolicy(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
}

function heartbeatPolicyPayloadFromForm() {
  return {
    interval_ms: Number($("heartbeat-policy-interval").value || 0),
    warning_after_missed: Number($("heartbeat-policy-warning-after-missed").value || 0),
    block_after_missed: Number($("heartbeat-policy-block-after-missed").value || 0)
  };
}

function validateHeartbeatPolicyForm(snapshot) {
  const payload = heartbeatPolicyPayloadFromForm();
  const problems = [];
  if (!Number.isInteger(payload.interval_ms) || payload.interval_ms <= 0) {
    problems.push("Interval (ms) must be a whole number greater than zero.");
  }
  if (!Number.isInteger(payload.warning_after_missed) || payload.warning_after_missed <= 0) {
    problems.push("Warning After Missed Beats must be a whole number greater than zero.");
  }
  if (!Number.isInteger(payload.block_after_missed) || payload.block_after_missed <= 0) {
    problems.push("Escalate Alert After Missed Beats must be a whole number greater than zero.");
  }
  if (payload.block_after_missed > 0 && payload.warning_after_missed > 0 && payload.block_after_missed < payload.warning_after_missed) {
    problems.push("Escalate Alert After Missed Beats must be greater than or equal to Warning After Missed Beats.");
  }
  const intervalMs = Number(payload.interval_ms || 0);
  $("heartbeat-policy-warning-gap").textContent = intervalMs > 0 && payload.warning_after_missed > 0 ? fmtDurationMs(intervalMs * payload.warning_after_missed) : "-";
  $("heartbeat-policy-block-gap").textContent = intervalMs > 0 && payload.block_after_missed > 0 ? fmtDurationMs(intervalMs * payload.block_after_missed) : "-";
  $("heartbeat-policy-validation-list").innerHTML = problems.map((item) => "<div class=\"validation-item\">" + escapeHTML(item) + "</div>").join("");
  return { payload, problems };
}

function renderHeartbeatPolicy(snapshot) {
  const policyResponse = snapshot?.heartbeatPolicy;
  const effective = currentHeartbeatPolicy(snapshot);
  const source = policyResponse?.policy_source || snapshot?.status?.heartbeat_policy_source || "file";
  const sourceBadge = $("heartbeat-policy-source");
  sourceBadge.textContent = source;
  sourceBadge.className = "source-pill source-" + source;
  $("heartbeat-policy-interval").value = String(effective.interval_ms || 0);
  if (!heartbeatPolicyHasFocus()) {
    $("heartbeat-policy-interval").value = String(effective.interval_ms || 0);
    $("heartbeat-policy-warning-after-missed").value = String(effective.warning_after_missed || 3);
    $("heartbeat-policy-block-after-missed").value = String(effective.block_after_missed || 6);
  }
  $("heartbeat-policy-updated").value = policyResponse?.policy_last_updated_at ? fmtTime(policyResponse.policy_last_updated_at) : "file baseline";
  const tokenRequired = setupActionsRequireToken();
  const tokenPresent = !!getSetupToken() || !!getCertToken();
  const validation = validateHeartbeatPolicyForm(snapshot);
  $("heartbeat-policy-save-button").disabled = validation.problems.length > 0 || (tokenRequired && !tokenPresent);
  $("heartbeat-policy-clear-button").disabled = !policyResponse?.override_present || (tokenRequired && !tokenPresent);
  $("heartbeat-policy-message").textContent = tokenRequired && !tokenPresent
    ? "Enter a setup or certificate API token before changing heartbeat policy."
    : "Tune warning and escalation thresholds for heartbeat gap handling.";
}

function syncHeartbeatPolicyFromSnapshot(snapshot) {
  renderHeartbeatPolicy(snapshot);
}

function blockerPolicyHasFocus() {
  const form = $("blocker-policy-form");
  return !!(form && form.contains(document.activeElement));
}

function normalizeBlockerPolicyResponse(payload) {
  const effectiveIDs = Array.isArray(payload?.effective?.approved_blocker_ids)
    ? payload.effective.approved_blocker_ids
    : [];
  const normalizedEffective = [];
  effectiveIDs.forEach((item) => {
    const id = String(item || "").trim();
    if (!id) return;
    if (normalizedEffective.indexOf(id) >= 0) return;
    normalizedEffective.push(id);
  });
  normalizedEffective.sort((a, b) => a.localeCompare(b));

  const overrideIDs = Array.isArray(payload?.override?.approved_blocker_ids)
    ? payload.override.approved_blocker_ids
    : [];
  const normalizedOverride = [];
  overrideIDs.forEach((item) => {
    const id = String(item || "").trim();
    if (!id) return;
    if (normalizedOverride.indexOf(id) >= 0) return;
    normalizedOverride.push(id);
  });
  normalizedOverride.sort((a, b) => a.localeCompare(b));

  return {
    effective: { approved_blocker_ids: normalizedEffective },
    policy_source: String(payload?.policy_source || "file").trim() || "file",
    policy_last_updated_at: String(payload?.policy_last_updated_at || "").trim(),
    override_present: payload?.override_present === true,
    override: payload?.override
      ? {
          approved_blocker_ids: normalizedOverride,
          updated_at: String(payload.override.updated_at || "").trim(),
          updated_by: String(payload.override.updated_by || "").trim(),
          last_change_action: String(payload.override.last_change_action || "").trim(),
          last_change_rationale: String(payload.override.last_change_rationale || "").trim(),
          last_change_actor_scope: String(payload.override.last_change_actor_scope || "").trim()
        }
      : null,
    escalation_history: normalizeBlockerPolicyEscalationHistory(payload?.escalation_history)
  };
}

function normalizeBlockerPolicyEscalationHistory(rawRows) {
  const rows = Array.isArray(rawRows) ? rawRows : [];
  return rows.map((item) => ({
    id: Number(item?.id || 0),
    created_at: String(item?.created_at || "").trim(),
    action: String(item?.action || "").trim(),
    finding_id: String(item?.finding_id || "").trim(),
    rationale: String(item?.rationale || "").trim(),
    actor_scope: String(item?.actor_scope || "").trim(),
    egm_focus: String(item?.egm_focus || "").trim(),
    updated_by: String(item?.updated_by || "").trim()
  }));
}

function normalizeBlockerPolicySuggestions(payload) {
  const suggestions = Array.isArray(payload?.suggestions) ? payload.suggestions : [];
  return {
    generated_at: String(payload?.generated_at || "").trim(),
    policy: normalizeBlockerPolicyResponse(payload?.policy || {}),
    suggestions: suggestions.map((item) => ({
      finding_id: String(item?.finding_id || "").trim(),
      message: String(item?.message || "").trim(),
      downgraded_by_policy: item?.downgraded_by_policy === true
    })).filter((item) => item.finding_id !== "")
  };
}

function blockerPolicyIDsFromForm() {
  const raw = String($("blocker-policy-approved-ids").value || "");
  const split = raw.split(/[\s,]+/);
  const values = [];
  split.forEach((item) => {
    const id = String(item || "").trim();
    if (!id) return;
    if (values.indexOf(id) >= 0) return;
    values.push(id);
  });
  values.sort((a, b) => a.localeCompare(b));
  return values;
}

function validateBlockerPolicyForm() {
  const ids = blockerPolicyIDsFromForm();
  const problems = [];
  const pattern = /^[a-z0-9_]+$/;
  ids.forEach((id) => {
    if (!pattern.test(id)) {
      problems.push("Approved blocker IDs must match ^[a-z0-9_]+$: " + id);
    }
  });
  $("blocker-policy-validation-list").innerHTML = problems.map((item) => "<div class=\"validation-item\">" + escapeHTML(item) + "</div>").join("");
  return { approved_blocker_ids: ids, problems: problems };
}

function renderBlockerGovernance(snapshot) {
  const preflight = snapshot?.cabinetPreflight || null;
  const fallbackPolicy = preflight?.blocker_policy || null;
  const policy = normalizeBlockerPolicyResponse(snapshot?.blockerPolicy || fallbackPolicy || {});
  const suggestionsPayload = normalizeBlockerPolicySuggestions(snapshot?.blockerPolicySuggestions || {});
  const suggestions = Array.isArray(suggestionsPayload.suggestions) ? suggestionsPayload.suggestions : [];
  const tokenRequired = setupActionsRequireToken();
  const tokenPresent = !!getSetupToken() || !!getCertToken();

  const sourceBadge = $("blocker-policy-source");
  sourceBadge.textContent = policy.policy_source || "file";
  sourceBadge.className = "source-pill source-" + (policy.policy_source || "file");

  $("blocker-policy-updated").value = policy.policy_last_updated_at ? fmtTime(policy.policy_last_updated_at) : "file baseline";

  if (!blockerPolicyHasFocus()) {
    $("blocker-policy-approved-ids").value = (policy.effective.approved_blocker_ids || []).join("\n");
  }

  const validation = validateBlockerPolicyForm();
  $("blocker-policy-save-button").disabled = validation.problems.length > 0 || (tokenRequired && !tokenPresent);
  $("blocker-policy-clear-button").disabled = !policy.override_present || (tokenRequired && !tokenPresent);

  const selectedFindingID = String($("blocker-policy-finding-id").value || "").trim();
  const selectedApproved = (policy.effective.approved_blocker_ids || []).indexOf(selectedFindingID) >= 0;
  const selectedSuggested = suggestions.some((item) => item.finding_id === selectedFindingID);
  $("blocker-policy-approve-button").disabled = !selectedSuggested || (tokenRequired && !tokenPresent);
  $("blocker-policy-revoke-button").disabled = !selectedApproved || (tokenRequired && !tokenPresent);

  renderItems("blocker-policy-suggestions-list", suggestions, "No suggested escalations.", (item) =>
    "<div class=\"item blocker-governance-item\">" +
      "<strong>" + escapeHTML(item.finding_id) + "</strong>" +
      "<span>" + escapeHTML(item.message || \"-\") + "</span>" +
      "<span class=\"muted-text\">" + escapeHTML(item.downgraded_by_policy ? \"downgraded_by_policy=true\" : \"downgraded_by_policy=false\") + "</span>" +
      "<div class=\"setup-actions evidence-actions blocker-governance-actions\">" +
        "<button type=\"button\" class=\"secondary-button blocker-policy-select-suggestion\" data-finding-id=\"" + escapeHTML(item.finding_id) + "\">Use</button>" +
        "<button type=\"button\" class=\"blocker-policy-approve-suggestion\" data-finding-id=\"" + escapeHTML(item.finding_id) + "\">Approve</button>" +
      "</div>" +
    "</div>"
  );

  const activeBlockers = Array.isArray(preflight?.blockers) ? preflight.blockers.map((item) => String(item || "").trim()).filter(Boolean) : [];
  renderItems("blocker-policy-active-blockers", activeBlockers, "No active approved blockers.", (item) =>
    "<div class=\"item blocker-governance-item\"><strong>BLOCKER</strong><span>" + escapeHTML(item) + "</span></div>"
  );

  const downgraded = Array.isArray(preflight?.downgraded_findings) ? preflight.downgraded_findings : [];
  const downgradedRows = downgraded.map((item) => ({
    id: String(item?.id || "").trim(),
    marker: String(item?.marker || "").trim(),
    message: String(item?.message || "").trim(),
    detail: String(item?.detail || "").trim()
  }));
  renderItems("blocker-policy-downgraded-list", downgradedRows, "No downgraded findings.", (item) =>
    "<div class=\"item blocker-governance-item\">" +
      "<strong>" + escapeHTML(item.id || "-") + "</strong>" +
      "<span>" + escapeHTML(item.message || "-") + "</span>" +
      "<span class=\"muted-text\">" + escapeHTML((item.marker || "DOWNGRADED_BY_BLOCKER_POLICY") + (item.detail ? (" | " + item.detail) : "")) + "</span>" +
    "</div>"
  );

  const history = Array.isArray(policy?.escalation_history) ? policy.escalation_history : [];
  renderItems("blocker-policy-history-list", history, "No escalation history yet.", (item) =>
    "<div class=\"item blocker-governance-item\">" +
      "<strong>" + escapeHTML((item.action || \"-\") + \" \" + (item.finding_id || \"\")) + "</strong>" +
      "<span>" + escapeHTML(\"at \" + fmtTime(item.created_at) + \" | actor_scope=\" + (item.actor_scope || \"-\") + \" | by=\" + (item.updated_by || \"lab-api\")) + "</span>" +
      (item.rationale ? ("<span>" + escapeHTML(\"rationale: \" + item.rationale) + "</span>") : \"\") +
      (item.egm_focus ? ("<span class=\\\"muted-text\\\">" + escapeHTML(\"egm_focus=\" + item.egm_focus) + \"</span>\") : \"\") +
    "</div>"
  );

  const summaryParts = [
    "approved IDs " + String((policy.effective.approved_blocker_ids || []).length),
    "suggested escalations " + String(suggestions.length),
    "active blockers " + String(activeBlockers.length),
    "downgraded findings " + String(downgradedRows.length),
    "history rows " + String(history.length)
  ];
  $("blocker-policy-summary").textContent = "Governance summary: " + summaryParts.join(" | ");
  $("blocker-policy-message").textContent = tokenRequired && !tokenPresent
    ? "Enter a setup or certificate API token before changing blocker governance."
    : "Only approved blocker IDs can block runbook readiness.";
}

function syncBlockerPolicyFromSnapshot(snapshot) {
  renderBlockerGovernance(snapshot);
}

function blockerPolicyMutationHeaders() {
  const headers = { "Content-Type": "application/json" };
  const token = getSetupToken() || getCertToken();
  if (token) {
    headers.Authorization = "Bearer " + token;
  }
  return withEGMFocusHeader(headers);
}

async function sendBlockerPolicyEscalationAction(endpoint, findingID, rationale, successMessage, failMessage) {
  if (setupActionsRequireToken() && !getSetupToken() && !getCertToken()) {
    $("blocker-policy-message").textContent = "Enter a setup or certificate API token before changing blocker governance.";
    return;
  }
  const trimmedFindingID = String(findingID || "").trim();
  if (!trimmedFindingID) {
    $("blocker-policy-message").textContent = "finding_id is required.";
    return;
  }
  $("blocker-policy-message").textContent = "Applying blocker governance change.";
  const response = await fetch(endpoint, {
    method: "POST",
    headers: blockerPolicyMutationHeaders(),
    body: JSON.stringify({
      finding_id: trimmedFindingID,
      rationale: String(rationale || "").trim()
    })
  });
  if (!response.ok) {
    const detail = sanitizeHTTPText(await response.text());
    $("blocker-policy-message").textContent = failMessage + ": HTTP " + response.status + (detail ? " " + detail : "");
    setAlert("warning", "Blocker governance update failed", failMessage + ".");
    return;
  }
  $("blocker-policy-message").textContent = successMessage;
  setAlert("info", "Blocker governance updated", successMessage);
  schedulePoll(0);
}

async function approveSelectedBlockerFinding() {
  const findingID = String($("blocker-policy-finding-id").value || "").trim();
  const rationale = String($("blocker-policy-rationale").value || "").trim();
  if (!rationale) {
    $("blocker-policy-message").textContent = "Rationale is required for blocker escalation approval.";
    return;
  }
  await sendBlockerPolicyEscalationAction(
    endpoints.blockerPolicyApprove,
    findingID,
    rationale,
    "Blocker escalation approved.",
    "Approve failed",
  );
}

async function revokeSelectedBlockerFinding() {
  const findingID = String($("blocker-policy-finding-id").value || "").trim();
  const rationale = String($("blocker-policy-rationale").value || "").trim();
  await sendBlockerPolicyEscalationAction(
    endpoints.blockerPolicyRevoke,
    findingID,
    rationale,
    "Blocker escalation revoked.",
    "Revoke failed",
  );
}

async function reloadBlockerPolicyForm() {
  const [policyResponse, suggestionsResponse] = await Promise.all([
    fetch(endpoints.blockerPolicy, { cache: "no-store" }),
    fetch(endpoints.blockerPolicySuggestions, { cache: "no-store" })
  ]);
  if (!policyResponse.ok) {
    throw new Error("Blocker policy reload failed: HTTP " + policyResponse.status);
  }
  if (!suggestionsResponse.ok) {
    throw new Error("Blocker policy suggestions reload failed: HTTP " + suggestionsResponse.status);
  }
  const payload = normalizeBlockerPolicyResponse(await policyResponse.json());
  const suggestionsPayload = normalizeBlockerPolicySuggestions(await suggestionsResponse.json());
  const snapshot = copySnapshot(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  snapshot.blockerPolicy = payload;
  snapshot.blockerPolicySuggestions = suggestionsPayload;
  clientState.displaySnapshot = snapshot;
  syncBlockerPolicyFromSnapshot(snapshot);
}

async function saveBlockerPolicyOverride(event) {
  event.preventDefault();
  if (setupActionsRequireToken() && !getSetupToken() && !getCertToken()) {
    $("blocker-policy-message").textContent = "Enter a setup or certificate API token before saving blocker governance.";
    return;
  }
  const validation = validateBlockerPolicyForm();
  if (validation.problems.length > 0) {
    $("blocker-policy-message").textContent = "Resolve blocker governance validation issues before saving.";
    return;
  }
  const headers = { "Content-Type": "application/json" };
  const token = getSetupToken() || getCertToken();
  if (token) {
    headers.Authorization = "Bearer " + token;
  }
  const response = await fetch(endpoints.blockerPolicy, {
    method: "PUT",
    headers: withEGMFocusHeader(headers),
    body: JSON.stringify({ approved_blocker_ids: validation.approved_blocker_ids })
  });
  if (!response.ok) {
    const detail = sanitizeHTTPText(await response.text());
    $("blocker-policy-message").textContent = "Save failed: HTTP " + response.status + (detail ? " " + detail : "");
    setAlert("warning", "Blocker governance save failed", "Unable to save approved blocker IDs.");
    return;
  }
  $("blocker-policy-message").textContent = "Blocker governance saved.";
  setAlert("info", "Blocker governance override saved", "Runbook blocking is now restricted to approved blocker IDs.");
  schedulePoll(0);
}

async function clearBlockerPolicyOverride() {
  if (setupActionsRequireToken() && !getSetupToken() && !getCertToken()) {
    $("blocker-policy-message").textContent = "Enter a setup or certificate API token before clearing blocker governance.";
    return;
  }
  const headers = {};
  const token = getSetupToken() || getCertToken();
  if (token) {
    headers.Authorization = "Bearer " + token;
  }
  const response = await fetch(endpoints.blockerPolicy, {
    method: "DELETE",
    headers: withEGMFocusHeader(headers)
  });
  if (!response.ok) {
    const detail = sanitizeHTTPText(await response.text());
    $("blocker-policy-message").textContent = "Clear failed: HTTP " + response.status + (detail ? " " + detail : "");
    setAlert("warning", "Blocker governance clear failed", "Unable to clear blocker governance override.");
    return;
  }
  $("blocker-policy-message").textContent = "Blocker governance override cleared.";
  setAlert("info", "Blocker governance override cleared", "Blocker governance reverted to file policy.");
  schedulePoll(0);
}

async function reloadHeartbeatPolicyForm() {
  const response = await fetch(endpoints.heartbeatPolicy, { cache: "no-store" });
  if (!response.ok) {
    throw new Error("Heartbeat policy reload failed: HTTP " + response.status);
  }
  const payload = await response.json();
  const snapshot = copySnapshot(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  snapshot.heartbeatPolicy = payload;
  clientState.displaySnapshot = snapshot;
  syncHeartbeatPolicyFromSnapshot(snapshot);
}

async function saveHeartbeatPolicyOverride(event) {
  event.preventDefault();
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  const validation = validateHeartbeatPolicyForm(snapshot);
  if (validation.problems.length > 0) {
    $("heartbeat-policy-message").textContent = "Resolve heartbeat policy issues before saving.";
    return;
  }
  if (setupActionsRequireToken() && !getSetupToken() && !getCertToken()) {
    $("heartbeat-policy-message").textContent = "Enter a setup or certificate API token before saving heartbeat policy.";
    return;
  }
  const headers = { "Content-Type": "application/json" };
  const token = getSetupToken() || getCertToken();
  if (token) {
    headers.Authorization = "Bearer " + token;
  }
  const response = await fetch(endpoints.heartbeatPolicy, {
    method: "PUT",
    headers: withEGMFocusHeader(headers),
    body: JSON.stringify(validation.payload)
  });
  if (!response.ok) {
    const detail = sanitizeHTTPText(await response.text());
    $("heartbeat-policy-message").textContent = "Save failed: HTTP " + response.status + (detail ? " " + detail : "");
    setAlert("warning", "Heartbeat policy save failed", "Resolve heartbeat policy validation and retry.");
    return;
  }
  $("heartbeat-policy-message").textContent = "Heartbeat policy saved.";
  setAlert("info", "Heartbeat policy override saved", "Heartbeat warning behavior updated in this appliance session.");
  schedulePoll(0);
}

async function clearHeartbeatPolicyOverride() {
  if (setupActionsRequireToken() && !getSetupToken() && !getCertToken()) {
    $("heartbeat-policy-message").textContent = "Enter a setup or certificate API token before clearing heartbeat policy.";
    return;
  }
  const headers = {};
  const token = getSetupToken() || getCertToken();
  if (token) {
    headers.Authorization = "Bearer " + token;
  }
  const response = await fetch(endpoints.heartbeatPolicy, {
    method: "DELETE",
    headers: withEGMFocusHeader(headers)
  });
  if (!response.ok) {
    const detail = sanitizeHTTPText(await response.text());
    $("heartbeat-policy-message").textContent = "Clear failed: HTTP " + response.status + (detail ? " " + detail : "");
    setAlert("warning", "Heartbeat policy clear failed", "Unable to clear heartbeat policy override.");
    return;
  }
  $("heartbeat-policy-message").textContent = "Heartbeat policy override cleared.";
  setAlert("info", "Heartbeat policy override cleared", "Heartbeat behavior reverted to file policy.");
  schedulePoll(0);
}

function exportRunReportJSON() {
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  const report = boundedRunReport(snapshot);
  if (!report) {
    return;
  }
  downloadTextMaterial(runReportFilenameBase(report) + ".json", JSON.stringify(report, null, 2));
  $("run-report-state").textContent = "saved";
  $("run-report-state").className = "source-pill source-file";
  $("run-report-message").textContent = "Run JSON report downloaded.";
}

function exportRunReportMarkdown() {
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  const report = boundedRunReport(snapshot);
  if (!report) {
    return;
  }
  downloadTextMaterial(runReportFilenameBase(report) + ".md", buildRunReportMarkdown(report));
  $("run-report-state").textContent = "saved";
  $("run-report-state").className = "source-pill source-file";
  $("run-report-message").textContent = "Run Markdown report downloaded.";
}

function renderCabinetRunTimeline(snapshot) {
  const items = buildCabinetRunTimeline(snapshot);
  const focusScoped = applyTimelineFocus(items);
  const filtered = applyTimelineFilter(focusScoped);
  const heartbeatCount = focusScoped.filter((item) => item.kind === "heartbeat").length;
  const focusID = currentEGMFocusID();
  $("timeline-rollup-heartbeat-toggle").checked = clientState.timelineRollupHeartbeat === true;
  $("timeline-show-raw-heartbeat-toggle").checked = clientState.timelineShowRawHeartbeat === true;
  $("timeline-rollup-heartbeat-toggle").disabled = clientState.timelineShowRawHeartbeat === true;
  updateTimelineFilterLabels(focusScoped.length, filtered.length, heartbeatCount);
  $("timeline-grouping-label").textContent = focusID
    ? ("Focused on " + focusID + "; EGM rows are filtered and global rows remain visible.")
    : "All EGM rows grouped by EGM ID; global rows remain grouped as global.";
  if (!focusID) {
    renderGroupedTimelineAll(filtered);
    return;
  }
  renderItems("cabinet-run-timeline", filtered, "No cabinet run events captured yet", (item) => timelineEntryHTML(item));
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
  if (clientState.egmFilter === "endpoint_integrity") {
    return egms.filter((egm) =>
      egm?.endpoint_collision_warning === true ||
      egm?.endpoint_drift_warning === true
    );
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

function normalizeEGMRegistryResponse(payload) {
  const data = payload && typeof payload === "object" ? payload : {};
  const normalize = (item) => {
    const row = item && typeof item === "object" ? item : {};
    return {
      egm_id: String(row.egm_id || "").trim(),
      display_name: String(row.display_name || "").trim(),
      vendor: String(row.vendor || "").trim(),
      cabinet_family: String(row.cabinet_family || "").trim(),
      game_title: String(row.game_title || "").trim(),
      software_version: String(row.software_version || "").trim(),
      notes: String(row.notes || "").trim(),
      updated_at: String(row.updated_at || "").trim(),
      updated_by: String(row.updated_by || "").trim()
    };
  };
  const overrides = Array.isArray(data.overrides) ? data.overrides.map(normalize).filter((row) => row.egm_id) : [];
  return {
    generated_at: String(data.generated_at || "").trim(),
    overrides: overrides
  };
}

function egmRegistryOverridesFromSnapshot(snapshot) {
  const payload = snapshot?.egmRegistry;
  const normalized = normalizeEGMRegistryResponse(payload);
  const map = {};
  (normalized.overrides || []).forEach((item) => {
    map[item.egm_id] = item;
  });
  return map;
}

function egmRegistryOverrideForID(snapshot, egmID) {
  const id = String(egmID || "").trim();
  if (!id) return null;
  const map = egmRegistryOverridesFromSnapshot(snapshot);
  return map[id] || null;
}

function setEGMRegistryDrawerState(level, message) {
  const badge = $("egm-registry-drawer-state");
  badge.textContent = level;
  badge.className = "source-pill " + (level === "ready" || level === "saved" ? "source-file" : level === "working" ? "source-override" : "source-mixed");
  $("egm-registry-drawer-message").textContent = message;
}

function openEGMRegistryDrawerFor(egmID) {
  clientState.selectedEGMRegistryID = String(egmID || "").trim();
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  renderEGMRegistryDrawer(snapshot);
}

function closeEGMRegistryDrawer() {
  clientState.selectedEGMRegistryID = "";
  const drawer = $("egm-registry-drawer");
  drawer.classList.add("egm-registry-drawer-hidden");
}

function selectedEGMFromSnapshot(snapshot) {
  const selectedID = String(clientState.selectedEGMRegistryID || "").trim();
  if (!selectedID) return null;
  const rows = Array.isArray(snapshot?.status?.egms) ? snapshot.status.egms : [];
  return rows.find((row) => String(row?.id || "").trim() === selectedID) || null;
}

function currentEGMRegistryFormValue() {
  return {
    display_name: $("egm-registry-display-name").value.trim(),
    vendor: $("egm-registry-vendor").value.trim(),
    cabinet_family: $("egm-registry-cabinet-family").value.trim(),
    game_title: $("egm-registry-game-title").value.trim(),
    software_version: $("egm-registry-software-version").value.trim(),
    notes: $("egm-registry-notes").value.trim()
  };
}

function egmRegistryMutationHeaders() {
  const headers = { "Content-Type": "application/json" };
  const token = getSetupToken() || getCertToken();
  if (token) {
    headers.Authorization = "Bearer " + token;
  }
  return withEGMFocusHeader(headers);
}

function renderEGMRegistryDrawer(snapshot) {
  const drawer = $("egm-registry-drawer");
  const selected = selectedEGMFromSnapshot(snapshot);
  if (!selected) {
    drawer.classList.add("egm-registry-drawer-hidden");
    return;
  }
  drawer.classList.remove("egm-registry-drawer-hidden");
  const selectedID = String(selected.id || "").trim();
  const override = egmRegistryOverrideForID(snapshot, selectedID);
  const tokenRequired = mutationTokenRequired();
  const tokenPresent = !!(getSetupToken() || getCertToken());
  $("egm-registry-drawer-title").textContent = "EGM Metadata: " + selectedID;
  $("egm-registry-egm-id").value = selectedID;
  $("egm-registry-display-name").value = override?.display_name || selected.display_name || selectedID;
  $("egm-registry-vendor").value = override?.vendor || selected.vendor || "";
  $("egm-registry-cabinet-family").value = override?.cabinet_family || selected.cabinet_family || "";
  $("egm-registry-game-title").value = override?.game_title || selected.game_title || "";
  $("egm-registry-software-version").value = override?.software_version || selected.software_version || "";
  $("egm-registry-notes").value = override?.notes || selected.notes || "";
  $("egm-registry-override-state").textContent = override
    ? ("override active; updated " + fmtTime(override.updated_at) + (override.updated_by ? (" by " + override.updated_by) : ""))
    : "no registry override; live metadata only";
  $("egm-registry-source-state").textContent = String(selected.source || "UNKNOWN") + (selected.registry_override ? " (registry override merged)" : "");
  $("egm-registry-delete-button").disabled = !override || (tokenRequired && !tokenPresent);
  $("egm-registry-save-button").disabled = tokenRequired && !tokenPresent;
  const promoteEnabled = String(selected.source || "").toUpperCase() === "DISCOVERED" && selected.registry_override !== true;
  $("egm-registry-promote-button").disabled = !promoteEnabled || (tokenRequired && !tokenPresent);
  setEGMRegistryDrawerState("ready", tokenRequired && !tokenPresent
    ? "Enter a setup or certificate API token before saving, promoting, or deleting registry metadata."
    : "Edit metadata and save. Promote converts discovered EGM metadata into configured override.");
}

async function promoteSelectedEGMRegistry(event) {
  if (event) event.preventDefault();
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  const selected = selectedEGMFromSnapshot(snapshot);
  if (!selected) return;
  if (mutationTokenRequired() && !getSetupToken() && !getCertToken()) {
    setEGMRegistryDrawerState("blocked", "Enter a setup or certificate API token before promoting.");
    return;
  }
  try {
    setEGMRegistryDrawerState("working", "Promoting discovered EGM metadata.");
    const formValue = currentEGMRegistryFormValue();
    const response = await fetch(endpoints.egmRegistryPromote, {
      method: "POST",
      headers: egmRegistryMutationHeaders(),
      body: JSON.stringify(Object.assign({ egm_id: selected.id }, formValue))
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setEGMRegistryDrawerState("blocked", "Promote failed: HTTP " + response.status + (detail ? " " + detail : ""));
      return;
    }
    setEGMRegistryDrawerState("saved", "Discovered EGM promoted into configured registry metadata.");
    schedulePoll(0);
  } catch (err) {
    setEGMRegistryDrawerState("blocked", err && err.message ? err.message : "Promote failed.");
  }
}

async function saveSelectedEGMRegistry(event) {
  if (event) event.preventDefault();
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  const selected = selectedEGMFromSnapshot(snapshot);
  if (!selected) return;
  if (mutationTokenRequired() && !getSetupToken() && !getCertToken()) {
    setEGMRegistryDrawerState("blocked", "Enter a setup or certificate API token before saving.");
    return;
  }
  try {
    setEGMRegistryDrawerState("working", "Saving EGM registry metadata.");
    const response = await fetch(endpoints.egmRegistry + "/" + encodeURIComponent(String(selected.id || "").trim()), {
      method: "PUT",
      headers: egmRegistryMutationHeaders(),
      body: JSON.stringify(currentEGMRegistryFormValue())
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setEGMRegistryDrawerState("blocked", "Save failed: HTTP " + response.status + (detail ? " " + detail : ""));
      return;
    }
    setEGMRegistryDrawerState("saved", "EGM registry metadata saved.");
    schedulePoll(0);
  } catch (err) {
    setEGMRegistryDrawerState("blocked", err && err.message ? err.message : "Save failed.");
  }
}

async function deleteSelectedEGMRegistryOverride() {
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  const selected = selectedEGMFromSnapshot(snapshot);
  if (!selected) return;
  if (mutationTokenRequired() && !getSetupToken() && !getCertToken()) {
    setEGMRegistryDrawerState("blocked", "Enter a setup or certificate API token before deleting override.");
    return;
  }
  const egmID = String(selected.id || "").trim();
  if (!egmID) return;
  if (!window.confirm("Delete EGM registry override for " + egmID + "?")) {
    return;
  }
  try {
    setEGMRegistryDrawerState("working", "Deleting EGM registry override.");
    const response = await fetch(endpoints.egmRegistry + "/" + encodeURIComponent(egmID), {
      method: "DELETE",
      headers: egmRegistryMutationHeaders()
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setEGMRegistryDrawerState("blocked", "Delete failed: HTTP " + response.status + (detail ? " " + detail : ""));
      return;
    }
    setEGMRegistryDrawerState("saved", "EGM registry override deleted.");
    schedulePoll(0);
  } catch (err) {
    setEGMRegistryDrawerState("blocked", err && err.message ? err.message : "Delete failed.");
  }
}

function selectedEGMBulkIDSet() {
  const selection = clientState.egmBulkSelection && typeof clientState.egmBulkSelection === "object"
    ? clientState.egmBulkSelection
    : {};
  const ids = Object.keys(selection).filter((id) => selection[id] === true && String(id || "").trim() !== "");
  return new Set(ids);
}

function selectedEGMBulkIDs() {
  return Array.from(selectedEGMBulkIDSet()).sort();
}

function setEGMBulkSelectionForID(egmID, selected) {
  const id = String(egmID || "").trim();
  if (!id) return;
  const next = Object.assign({}, clientState.egmBulkSelection || {});
  if (selected) {
    next[id] = true;
  } else {
    delete next[id];
  }
  clientState.egmBulkSelection = next;
}

function clearEGMBulkSelection() {
  clientState.egmBulkSelection = {};
}

function reconcileEGMBulkSelection(allEGMs) {
  const valid = new Set((Array.isArray(allEGMs) ? allEGMs : [])
    .map((egm) => String(egm?.id || "").trim())
    .filter(Boolean));
  const next = {};
  const current = clientState.egmBulkSelection && typeof clientState.egmBulkSelection === "object"
    ? clientState.egmBulkSelection
    : {};
  Object.keys(current).forEach((id) => {
    if (current[id] === true && valid.has(id)) {
      next[id] = true;
    }
  });
  clientState.egmBulkSelection = next;
}

function setAllVisibleEGMBulkSelection(selected) {
  const visibleIDs = Array.isArray(clientState.egmVisibleRowIDs) ? clientState.egmVisibleRowIDs : [];
  if (visibleIDs.length === 0) {
    return;
  }
  visibleIDs.forEach((id) => {
    setEGMBulkSelectionForID(id, selected);
  });
}

function renderEGMBulkSelectionSummary(visibleCount) {
  const selected = selectedEGMBulkIDs();
  const selectedCount = selected.length;
  const focusID = currentEGMFocusID();
  const focusLabel = focusID ? ("focus " + focusID) : "all EGMs";
  $("egm-bulk-selection-summary").textContent = selectedCount + " selected (" + focusLabel + ", " + visibleCount + " visible)";
  $("egm-bulk-promote-button").disabled = selectedCount === 0;
  $("egm-bulk-apply-profile-button").disabled = selectedCount === 0;
  $("egm-clear-selection-button").disabled = selectedCount === 0;
}

function setEGMBulkActionMessage(message) {
  $("egm-bulk-action-message").textContent = message;
}

async function promoteSelectedEGMBulk() {
  const ids = selectedEGMBulkIDs();
  if (ids.length === 0) {
    setEGMBulkActionMessage("Select one or more EGMs before bulk promote.");
    return;
  }
  if (mutationTokenRequired() && !getSetupToken() && !getCertToken()) {
    setEGMBulkActionMessage("Enter a setup or certificate API token before bulk promote.");
    return;
  }
  setEGMBulkActionMessage("Promoting selected EGMs...");
  try {
    const response = await fetch(endpoints.egmRegistryPromoteBulk, {
      method: "POST",
      headers: egmRegistryMutationHeaders(),
      body: JSON.stringify({ egm_ids: ids })
    });
    const rawBody = await response.text();
    let payload = {};
    try {
      payload = rawBody ? JSON.parse(rawBody) : {};
    } catch (_) {
      payload = {};
    }
    if (!response.ok) {
      const detail = sanitizeHTTPText(rawBody);
      setEGMBulkActionMessage("Bulk promote failed: HTTP " + response.status + (detail ? " " + detail : ""));
      setAlert("warning", "Bulk promote failed", "Unable to promote selected EGMs.");
      return;
    }
    const promotedCount = Number(payload?.promoted_count || 0);
    const skippedIDs = Array.isArray(payload?.skipped_ids) ? payload.skipped_ids : [];
    const errors = Array.isArray(payload?.errors) ? payload.errors : [];
    setEGMBulkActionMessage("Bulk promote complete: promoted " + promotedCount + ", skipped " + skippedIDs.length + ", errors " + errors.length + ".");
    if (errors.length > 0) {
      setAlert("warning", "Bulk promote completed with skips", "Promoted " + promotedCount + ", skipped " + skippedIDs.length + ".");
    } else {
      setAlert("info", "Bulk promote complete", "Promoted " + promotedCount + " selected EGMs.");
    }
    schedulePoll(0);
  } catch (err) {
    setEGMBulkActionMessage(err && err.message ? err.message : "Bulk promote failed.");
    setAlert("warning", "Bulk promote failed", "Unable to promote selected EGMs.");
  }
}

async function applySelectedToCabinetProfileFirstTest() {
  const ids = selectedEGMBulkIDs();
  if (ids.length === 0) {
    setEGMBulkActionMessage("Select one or more EGMs before applying to cabinet profile.");
    return;
  }
  if (mutationTokenRequired() && !getSetupToken() && !getCertToken()) {
    setEGMBulkActionMessage("Enter a setup or certificate API token before updating first-test EGM IDs.");
    return;
  }
  setEGMBulkActionMessage("Applying selected EGMs to first-test EGM IDs...");
  try {
    const response = await fetch(endpoints.egmRegistryApplyToCabinetProfile, {
      method: "POST",
      headers: egmRegistryMutationHeaders(),
      body: JSON.stringify({ egm_ids: ids })
    });
    const rawBody = await response.text();
    let payload = {};
    try {
      payload = rawBody ? JSON.parse(rawBody) : {};
    } catch (_) {
      payload = {};
    }
    if (!response.ok) {
      const detail = sanitizeHTTPText(rawBody);
      setEGMBulkActionMessage("Apply to first-test failed: HTTP " + response.status + (detail ? " " + detail : ""));
      setAlert("warning", "Apply to first-test failed", "Unable to update cabinet profile first-test EGM IDs.");
      return;
    }
    const appliedIDs = Array.isArray(payload?.applied_first_test_egm_ids) ? payload.applied_first_test_egm_ids : ids;
    const profile = payload?.cabinet_profile?.effective;
    if (profile && typeof profile === "object") {
      fillCabinetSetupForm(profile);
      renderCabinetProfileSuggestions(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
    } else {
      $("setup-first-test-egm-ids").value = appliedIDs.join(", ");
      renderCabinetSetupValidation();
    }
    setEGMBulkActionMessage("Applied " + appliedIDs.length + " EGM IDs to first-test profile values. Save Override to persist if needed.");
    setAlert("info", "First-test EGM IDs updated", "Applied selected EGM IDs into cabinet profile override.");
    schedulePoll(0);
  } catch (err) {
    setEGMBulkActionMessage(err && err.message ? err.message : "Apply to first-test failed.");
    setAlert("warning", "Apply to first-test failed", "Unable to update cabinet profile first-test EGM IDs.");
  }
}

function renderEGMTable(status) {
  const all = Array.isArray(status?.egms) ? status.egms.slice() : [];
  reconcileEGMBulkSelection(all);
  const focusScoped = filterStatusEGMsByFocus(all);
  const sectionFiltered = applyEGMFilter(focusScoped);
  const filtered = sectionFiltered.filter((egm) => egmMatchesGlobalFilters(egm));
  const sorted = filtered.sort(compareEGM);
  const selectedSet = selectedEGMBulkIDSet();
  const visibleEGMIDs = sorted
    .map((egm) => String(egm?.id || "").trim())
    .filter(Boolean);
  clientState.egmVisibleRowIDs = visibleEGMIDs;
  const selectAllChecked = visibleEGMIDs.length > 0 && visibleEGMIDs.every((id) => selectedSet.has(id));
  $("egm-select-all-checkbox").checked = selectAllChecked;
  renderEGMBulkSelectionSummary(visibleEGMIDs.length);
  const rows = sorted.map((egm) =>
    (() => {
      const egmID = String(egm.id || "").trim();
      const checked = selectedSet.has(egmID) ? " checked" : "";
      const configuredAddress = (egm.ip_address || "-") + ":" + (egm.port || "-");
      const endpointAddress = (egm.last_endpoint_ip || "-") + ":" + (egm.last_endpoint_port || "-");
      const driftLabel = egm.endpoint_drift_warning === true
        ? ("warning" + (Array.isArray(egm.endpoint_drift_ips) && egm.endpoint_drift_ips.length ? (" (" + egm.endpoint_drift_ips.join(", ") + ")") : ""))
        : "none";
      const collisionTypes = Array.isArray(egm.endpoint_collision_types)
        ? egm.endpoint_collision_types.map((item) => String(item || "").toUpperCase()).filter(Boolean)
        : [];
      const integrityLabel = egm.endpoint_collision_warning === true
        ? ("warning" + (collisionTypes.length ? (" (" + collisionTypes.join(", ") + ")") : ""))
        : "none";
      const promoteVisible = String(egm.source || "").toUpperCase() === "DISCOVERED" && egm.registry_override !== true;
      const actionButtons = "<div class=\"egm-row-actions\">" +
        "<button type=\"button\" class=\"secondary-button egm-row-action-button\" data-egm-action=\"edit\" data-egm-id=\"" + escapeHTML(String(egm.id || "")) + "\">Edit</button>" +
        (promoteVisible
          ? "<button type=\"button\" class=\"secondary-button egm-row-action-button\" data-egm-action=\"promote\" data-egm-id=\"" + escapeHTML(String(egm.id || "")) + "\">Promote</button>"
          : "") +
        "</div>";
      return "" +
    "<tr>" +
      "<td class=\"egm-row-select-cell\"><input type=\"checkbox\" class=\"egm-row-select-checkbox\" data-egm-id=\"" + escapeHTML(egmID) + "\"" + checked + "></td>" +
      "<td><strong>" + escapeHTML(egm.display_name || egm.id) + "</strong><br><span class=\"minor\">" + escapeHTML((egm.vendor || "") + " " + (egm.cabinet_family || "")).trim() + "</span><br><span class=\"minor\">" + escapeHTML("id " + (egm.id || "-")) + (egm.registry_override === true ? " | override" : "") + "</span></td>" +
      "<td>" + egmSourcePill(egm.source) + "</td>" +
      "<td>" + statusPill(egm.status) + "</td>" +
      "<td>" + escapeHTML(configuredAddress) + "</td>" +
      "<td>" + escapeHTML(endpointAddress) + "<br><span class=\"minor\">seen " + escapeHTML(fmtTime(egm.last_endpoint_seen_at)) + "</span></td>" +
      "<td>" + escapeHTML(driftLabel) + "<br><span class=\"minor\">integrity " + escapeHTML(integrityLabel) + "</span></td>" +
      "<td>" + escapeHTML(egm.game_title || "-") + "<br><span class=\"minor\">" + escapeHTML(egm.software_version || "") + "</span></td>" +
      "<td>" + escapeHTML(fmtTime(egm.last_seen)) + "<br><span class=\"minor\">" + escapeHTML(fmtAge(egm.last_seen)) + "</span></td>" +
      "<td>" + actionButtons + "</td>" +
    "</tr>";
    })()
  );
  const focusID = currentEGMFocusID();
  const severityFilter = normalizeGlobalSeverityFilter(clientState.globalSeverityFilter);
  const globalText = String(clientState.globalTextFilter || "").trim();
  const globalSuffix = (severityFilter !== "all" || globalText)
    ? (" | global " + (severityFilter === "all" ? "All" : severityFilter) + (globalText ? (' | q="' + globalText + '"') : ""))
    : "";
  $("egm-count").textContent = focusID
    ? ("Focus " + focusID + " | " + filtered.length + " / " + all.length + " EGMs" + globalSuffix)
    : (filtered.length + " / " + all.length + " EGMs" + globalSuffix);
  $("egm-table").innerHTML = rows.length ? rows.join("") : "<tr><td colspan=\"10\">No EGMs match current filter</td></tr>";
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

function normalizeCabinetProfileSuggestions(payload) {
  const observed = Array.isArray(payload?.observed_egm_ids)
    ? payload.observed_egm_ids.map((item) => String(item || "").trim()).filter(Boolean)
    : [];
  const recommended = Array.isArray(payload?.recommended_first_test_egm_ids)
    ? payload.recommended_first_test_egm_ids.map((item) => String(item || "").trim()).filter(Boolean)
    : [];
  const messages = Array.isArray(payload?.messages)
    ? payload.messages.map((item) => String(item || "").trim()).filter(Boolean)
    : [];
  return {
    observed_egm_ids: observed,
    recommended_first_test_egm_ids: recommended,
    placeholder_detected: payload?.placeholder_detected === true,
    reason: String(payload?.reason || "").trim(),
    messages: messages
  };
}

function cabinetProfileSuggestionsFromSnapshot(snapshot) {
  return normalizeCabinetProfileSuggestions(snapshot?.cabinetProfileSuggestions);
}

function renderCabinetProfileSuggestions(snapshot) {
  const suggestions = cabinetProfileSuggestionsFromSnapshot(snapshot);
  let preview = "";
  if (suggestions.recommended_first_test_egm_ids.length > 0) {
    preview = "Preview apply: " + suggestions.recommended_first_test_egm_ids.join(", ") + " (form only; Save Override persists).";
  } else if (suggestions.observed_egm_ids.length > 0) {
    preview = "Observed EGMs: " + suggestions.observed_egm_ids.join(", ") + ".";
  } else if (suggestions.reason) {
    preview = suggestions.reason;
  } else {
    preview = "No observed EGMs yet; start cabinet traffic to generate suggestions.";
  }
  if (suggestions.placeholder_detected) {
    preview += " Placeholder IDs detected in current first-test values.";
  }
  $("setup-observed-egms-preview").textContent = preview;
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
  return withEGMFocusHeader(headers);
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
  const [profile, suggestions] = await Promise.all([
    fetchJSON(endpoints.cabinetProfile),
    fetchJSON(endpoints.cabinetProfileSuggestions)
  ]);
  clientState.displaySnapshot = clientState.displaySnapshot || emptySnapshot();
  clientState.displaySnapshot.cabinetProfile = profile;
  clientState.displaySnapshot.cabinetProfileSuggestions = normalizeCabinetProfileSuggestions(suggestions);
  fillCabinetSetupForm(profile.effective || {});
  renderCabinetProfileSuggestions(clientState.displaySnapshot);
  setSetupState("ready", "Current values loaded from the appliance.");
  return profile;
}

async function useObservedEGMSuggestions() {
  try {
    setSetupState("working", "Loading observed EGM suggestions.");
    const suggestionsRaw = await fetchJSON(endpoints.cabinetProfileSuggestions);
    const suggestions = normalizeCabinetProfileSuggestions(suggestionsRaw);
    clientState.displaySnapshot = clientState.displaySnapshot || emptySnapshot();
    clientState.displaySnapshot.cabinetProfileSuggestions = suggestions;
    renderCabinetProfileSuggestions(clientState.displaySnapshot);
    if (suggestions.recommended_first_test_egm_ids.length === 0) {
      setSetupState("ready", suggestions.reason || "No observed EGM recommendation is available yet.");
      return;
    }
    $("setup-first-test-egm-ids").value = joinList(suggestions.recommended_first_test_egm_ids);
    renderCabinetSetupValidation();
    renderFirstCabinetSession(clientState.displaySnapshot);
    renderSessionEvidence(clientState.displaySnapshot);
    setSetupState("ready", "Applied observed EGM recommendation to the form. Save Override to persist.");
  } catch (err) {
    setSetupState("blocked", err && err.message ? err.message : "Unable to load observed EGM suggestions.");
  }
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
  return withEGMFocusHeader(headers);
}

function certificatePreviewFingerprint(role, certificatePEM, privateKeyPEM, requiresKey) {
  return [
    String(role || "").trim(),
    String(certificatePEM || "").trim(),
    requiresKey ? String(privateKeyPEM || "").trim() : ""
  ].join("\n---\n");
}

function clearCertificatePreviewState() {
  clientState.certPreviewFingerprint = "";
  clientState.certPreviewResult = null;
}

function certBackupCache(role) {
  const key = String(role || "").trim();
  if (!key) {
    return { loaded: false, loading: false, error: "", items: [] };
  }
  if (!clientState.certBackupsByRole[key]) {
    clientState.certBackupsByRole[key] = { loaded: false, loading: false, error: "", items: [] };
  }
  return clientState.certBackupsByRole[key];
}

function formatByteCount(value) {
  const size = Number(value || 0);
  if (!Number.isFinite(size) || size <= 0) {
    return "0 B";
  }
  if (size < 1024) return size + " B";
  const kib = size / 1024;
  if (kib < 1024) return kib.toFixed(1) + " KiB";
  const mib = kib / 1024;
  if (mib < 1024) return mib.toFixed(1) + " MiB";
  return (mib / 1024).toFixed(1) + " GiB";
}

function normalizeCertificateBackups(raw, role) {
  const payload = raw && typeof raw === "object" ? raw : {};
  const records = Array.isArray(payload.backups) ? payload.backups : [];
  return {
    role: String(payload.role || role || "").trim(),
    backups: records.map((item) => ({
      id: String(item?.id || "").trim(),
      created_at: item?.created_at || "",
      certificate_size_bytes: Number(item?.certificate?.size_bytes || 0),
      certificate_sha256: String(item?.certificate?.sha256 || "").trim(),
      private_key_size_bytes: Number(item?.private_key?.size_bytes || 0),
      private_key_sha256: String(item?.private_key?.sha256 || "").trim(),
      total_size_bytes: Number(item?.total_size_bytes || 0),
      restorable: item?.restorable === true
    })).filter((item) => item.id)
  };
}

function renderCertificateBackupHistory(certState) {
  const role = certState.role;
  const cache = certBackupCache(role);
  const backups = Array.isArray(cache.items) ? cache.items : [];
  let message = "Backup history unavailable.";
  if (cache.loading) {
    message = "Loading backup history for selected role.";
  } else if (cache.error) {
    message = "Backup history load failed: " + cache.error;
  } else if (!backups.length) {
    message = "No backups found for " + roleDisplayName(role) + ".";
  } else if (certState.tokenRequired && !certState.tokenPresent) {
    message = "Restore actions require API token in this session.";
  } else {
    message = "Showing " + String(backups.length) + " backup record(s) for " + roleDisplayName(role) + ".";
  }
  $("cert-backup-state").textContent = message;
  if (!backups.length) {
    $("cert-backup-list").innerHTML = cache.loading ? "" : "<div class=\"item\">No backups recorded yet.</div>";
    return;
  }

  $("cert-backup-list").innerHTML = backups.map((item) => {
    const createdAt = item.created_at ? fmtTime(item.created_at) : item.id;
    const certMeta = "Cert " + formatByteCount(item.certificate_size_bytes) + (item.certificate_sha256 ? " sha256=" + item.certificate_sha256.slice(0, 12) + "…" : "");
    const keyMeta = item.private_key_size_bytes > 0 || item.private_key_sha256
      ? "Key " + formatByteCount(item.private_key_size_bytes) + (item.private_key_sha256 ? " sha256=" + item.private_key_sha256.slice(0, 12) + "…" : "")
      : "";
    const restoreBlocked = !item.restorable || (certState.tokenRequired && !certState.tokenPresent);
    const restoreLabel = item.restorable ? "Restore" : "Incomplete Backup";
    const metaParts = [certMeta];
    if (keyMeta) metaParts.push(keyMeta);
    metaParts.push("Total " + formatByteCount(item.total_size_bytes));
    return "<div class=\"item cert-backup-item\">" +
      "<div class=\"cert-backup-item-head\"><strong>" + escapeHTML(item.id) + "</strong><span>" + escapeHTML(createdAt) + "</span></div>" +
      "<div class=\"cert-backup-meta\">" + escapeHTML(metaParts.join(" | ")) + "</div>" +
      "<div class=\"cert-backup-actions\"><button type=\"button\" class=\"secondary-button cert-restore-backup-button\" data-backup-id=\"" + escapeHTML(item.id) + "\"" + (restoreBlocked ? " disabled" : "") + ">" + escapeHTML(restoreLabel) + "</button></div>" +
      "</div>";
  }).join("");
}

async function loadCertificateBackups(role, forceReload) {
  const normalizedRole = String(role || "").trim();
  if (!normalizedRole) return;
  const cache = certBackupCache(normalizedRole);
  if (cache.loading) return;
  if (cache.loaded && !forceReload) return;
  cache.loading = true;
  cache.error = "";
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  renderCertificateManager(snapshot);
  try {
    const response = await fetch(endpoints.certificateBackups + "?role=" + encodeURIComponent(normalizedRole), {
      method: "GET",
      cache: "no-store"
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      throw new Error("HTTP " + response.status + (detail ? " " + detail : ""));
    }
    const payload = await response.json();
    const normalized = normalizeCertificateBackups(payload, normalizedRole);
    cache.items = normalized.backups;
    cache.loaded = true;
    cache.error = "";
  } catch (err) {
    cache.items = [];
    cache.error = err && err.message ? err.message : "backup history request failed";
  } finally {
    cache.loading = false;
    renderCertificateManager(snapshot);
  }
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
  const baseProblems = [];

  if (!configured) {
    baseProblems.push(roleDisplayName(role) + " is not configured in appliance runtime.");
  }
  if (!certPEM) {
    baseProblems.push("Certificate PEM is required.");
  }
  if (details.requiresKey && !privateKeyPEM) {
    baseProblems.push("Private key PEM is required for " + roleDisplayName(role) + ".");
  }
  if (!details.requiresKey && privateKeyPEM) {
    baseProblems.push("Private key PEM is not used for " + roleDisplayName(role) + ".");
  }

  const previewFingerprint = certificatePreviewFingerprint(role, certPEM, privateKeyPEM, details.requiresKey);
  const preview = clientState.certPreviewResult && clientState.certPreviewFingerprint === previewFingerprint
    ? clientState.certPreviewResult
    : null;
  const importProblems = baseProblems.slice();
  if (tokenRequired && !tokenPresent) {
    importProblems.push("API token is required for certificate import in this browser session.");
  }
  if (!preview) {
    importProblems.push("Run Preview before importing certificate material.");
  } else if (!preview.parse_ok) {
    importProblems.push("Resolve preview validation errors before importing.");
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
    baseProblems: baseProblems,
    importProblems: importProblems,
    preview: preview,
    previewFingerprint: previewFingerprint
  };
}

function normalizeCertificatePreview(raw, certState) {
  const payload = raw && typeof raw === "object" ? raw : {};
  const errors = Array.isArray(payload.errors) ? payload.errors.map((item) => String(item || "").trim()).filter(Boolean) : [];
  const parseOK = payload.parse_ok === true && errors.length === 0;
  return {
    role: String(payload.role || certState.role || "").trim(),
    parse_ok: parseOK,
    cert_subject: String(payload.cert_subject || "").trim(),
    cert_issuer: String(payload.cert_issuer || "").trim(),
    not_before: payload.not_before || "",
    not_after: payload.not_after || "",
    san_dns: Array.isArray(payload.san_dns) ? payload.san_dns.map((item) => String(item || "").trim()).filter(Boolean) : [],
    san_ips: Array.isArray(payload.san_ips) ? payload.san_ips.map((item) => String(item || "").trim()).filter(Boolean) : [],
    key_required: payload.key_required === true,
    key_present: payload.key_present === true,
    key_matches_cert: payload.key_matches_cert === true,
    errors: errors
  };
}

function renderCertificatePreview(certState) {
  const preview = certState.preview;
  if (!preview) {
    $("cert-preview-summary").textContent = "Run Preview before importing certificate material.";
    $("cert-preview-detail").textContent = "Preview is read-only and does not write files.";
    $("cert-preview-list").innerHTML = "";
    return;
  }
  const summary = preview.parse_ok ? "Preview passed." : "Preview found " + String(preview.errors.length) + " issue(s).";
  const keyLine = preview.key_required
    ? (preview.key_present ? (preview.key_matches_cert ? "Private key present and matches certificate." : "Private key present but does not match certificate.") : "Private key is required and not present.")
    : (preview.key_present ? "Private key provided for certificate-only role (not allowed)." : "Private key not required for this role.");
  const dnsLine = preview.san_dns.length ? preview.san_dns.join(", ") : "-";
  const ipsLine = preview.san_ips.length ? preview.san_ips.join(", ") : "-";
  $("cert-preview-summary").textContent = summary;
  $("cert-preview-detail").textContent = [
    "Subject: " + (preview.cert_subject || "-"),
    "Issuer: " + (preview.cert_issuer || "-"),
    "Not Before: " + fmtTime(preview.not_before),
    "Not After: " + fmtTime(preview.not_after),
    "SAN DNS: " + dnsLine,
    "SAN IPs: " + ipsLine,
    keyLine
  ].join(" | ");
  $("cert-preview-list").innerHTML = preview.errors.map((item) => "<div class=\"validation-item\">" + escapeHTML(item) + "</div>").join("");
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
  $("cert-validation-summary").textContent = certState.importProblems.length ? certState.importProblems.length + " issue(s)" : "Ready to import";
  $("cert-validation-list").innerHTML = certState.importProblems.map((item) => "<div class=\"validation-item\">" + escapeHTML(item) + "</div>").join("");
  $("cert-api-token-wrapper").classList.toggle("trusted-bypass-hidden", !certState.tokenRequired);
  $("cert-token-controls").classList.toggle("trusted-bypass-hidden", !certState.tokenRequired);
  $("cert-api-token-label").textContent = certState.tokenRequired ? "API Token (required for import/export key)" : "API Token (optional on trusted private network)";
  $("cert-token-help-text").textContent = certState.tokenRequired
    ? "Use API token for import and private-key export actions."
    : "Trusted private network bypass is active for this browser; token is optional for import and private-key export.";

  renderCertificatePreview(certState);
  renderCertificateBackupHistory(certState);
  loadCertificateBackups(certState.role, false).catch((err) => {
    const cache = certBackupCache(certState.role);
    cache.loading = false;
    cache.error = err && err.message ? err.message : "backup history request failed";
    renderCertificateBackupHistory(certState);
  });
  setCertKeyFieldVisible(certState.details.requiresKey);
  $("cert-preview-button").disabled = certState.baseProblems.length > 0;
  $("cert-import-button").disabled = certState.importProblems.length > 0;
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

async function previewCertificateMaterial() {
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  const certState = validateCertificateManagerForm(snapshot);
  if (certState.baseProblems.length > 0) {
    setCertManagerState("blocked", "Resolve certificate form issues before preview.");
    setCertManagerDetail(certState.baseProblems.join(" "));
    return;
  }

  const payload = certImportPayload();
  if (!certState.details.requiresKey) {
    payload.private_key_pem = "";
  }

  try {
    setCertManagerState("working", "Previewing " + roleDisplayName(payload.role) + " material.");
    const response = await fetch(endpoints.certificatePreview, {
      method: "POST",
      headers: certAuthHeaders(),
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      clearCertificatePreviewState();
      setCertManagerState("blocked", "Preview failed: HTTP " + response.status + (detail ? " " + detail : ""));
      renderCertificateManager(snapshot);
      return;
    }
    const result = await response.json();
    clientState.certPreviewFingerprint = certState.previewFingerprint;
    clientState.certPreviewResult = normalizeCertificatePreview(result, certState);
    if (clientState.certPreviewResult.parse_ok) {
      setCertManagerState("ready", "Preview valid. Import is now enabled for this payload.");
    } else {
      setCertManagerState("blocked", "Preview found validation issues.");
    }
    renderCertificateManager(snapshot);
  } catch (err) {
    clearCertificatePreviewState();
    setCertManagerState("blocked", err && err.message ? err.message : "Preview failed.");
    renderCertificateManager(snapshot);
  }
}

async function restoreCertificateBackup(backupID) {
  const role = selectedCertRole();
  const normalizedBackupID = String(backupID || "").trim();
  if (!normalizedBackupID) {
    setCertManagerState("blocked", "Backup ID is required for restore.");
    return;
  }
  if (setupActionsRequireToken() && !getCertToken()) {
    setCertManagerState("blocked", "Enter an API token before restoring backup material.");
    return;
  }
  if (!window.confirm("Restore backup " + normalizedBackupID + " for " + roleDisplayName(role) + "?")) {
    return;
  }
  try {
    setCertManagerState("working", "Restoring backup " + normalizedBackupID + " for " + roleDisplayName(role) + ".");
    const response = await fetch(endpoints.certificateRestore, {
      method: "POST",
      headers: certAuthHeaders(),
      body: JSON.stringify({
        role: role,
        backup_id: normalizedBackupID
      })
    });
    if (!response.ok) {
      const detail = sanitizeHTTPText(await response.text());
      setCertManagerState("blocked", "Restore failed: HTTP " + response.status + (detail ? " " + detail : ""));
      return;
    }
    const result = await response.json();
    clearCertificatePreviewState();
    await loadCertificateBackups(role, true);
    setCertManagerState("saved", "Restore complete for " + roleDisplayName(role) + ".");
    setCertManagerDetail("Backup: " + normalizedBackupID + " | Path: " + (result.certificate_path || "-") + (result.private_key_path ? " | Key path: " + result.private_key_path : "") + " | Status: " + (result.certificate_status || "-"));
    schedulePoll(0);
  } catch (err) {
    setCertManagerState("blocked", err && err.message ? err.message : "Restore failed.");
  }
}

async function importCertificateMaterial(event) {
  event.preventDefault();
  const payload = certImportPayload();
  const details = certRoleDetails(payload.role);
  const certState = validateCertificateManagerForm(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  if (certState.importProblems.length > 0) {
    setCertManagerState("blocked", "Resolve certificate import issues before importing.");
    setCertManagerDetail(certState.importProblems.join(" "));
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
    clearCertificatePreviewState();
    loadCertificateBackups(payload.role, true).catch(() => {});
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
  clearCertificatePreviewState();
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
    renderCabinetProfileSuggestions(snapshot);
    return;
  }
  fillCabinetSetupForm(profile);
  renderCabinetProfileSuggestions(snapshot);
}

function syncOperatorActionsFromCurrentState() {
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot();
  const hasStatus = !!snapshot?.status;
  $("operator-actions-refresh-button").disabled = clientState.inFlight === true;
  $("operator-actions-capture-evidence-button").disabled = !hasStatus || $("session-evidence-save-button").disabled;
  $("operator-actions-export-package-button").disabled = !hasStatus || $("session-package-export-button").disabled;
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
  renderGlobalViewControls();

  renderCertificateSummary(readiness.certificate_summary || {}, snapshot?.certificates, runtime);
  renderCertificateManager(snapshot);
  renderCabinetProfile(status);
  renderFirstCabinetSession(snapshot);
  renderRuntimePresetLibrary(snapshot);
  renderSessionEvidence(snapshot);
  syncCabinetSetupFromSnapshot(snapshot);
  syncHeartbeatPolicyFromSnapshot(snapshot);
  syncBlockerPolicyFromSnapshot(snapshot);
  renderOperatorDrill(snapshot);
  renderEGMFocusControl(snapshot);
  renderEGMGroupedSummary(snapshot);
  renderSelectedEGMDetail(snapshot);
  renderEGMTable(status);
  renderEGMRegistryDrawer(snapshot);
  renderEndpointIntegrity(snapshot);
  renderRunMarkerControls(snapshot);
  renderRunReportControls(snapshot);
  renderHeartbeatSummary(snapshot);
  renderCabinetRunTimeline(snapshot);
  renderItems("incident-list", snapshot?.incidents, "No incidents recorded", (item) =>
    "<div class=\"item\"><strong>#" + escapeHTML(item.id) + " " + escapeHTML(item.trigger_type) + "</strong><span>" + escapeHTML(fmtTime(item.created_at)) + " " + escapeHTML(item.trigger_source || "") + "</span></div>"
  );
  renderEGMHistory(snapshot);
  renderOperatorAuditTimeline(snapshot);
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
  syncOperatorActionsFromCurrentState();
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
  const heartbeat = heartbeatSummary(snapshot?.egmHistory || [], currentHeartbeatPolicy(snapshot), new Date().toISOString());

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
  if (heartbeat.severity === "critical" || heartbeat.severity === "warning") {
    setAlert(heartbeat.severity === "critical" ? "critical" : "warning", "Heartbeat anomaly detected", heartbeat.message);
    return;
  }
  if (heartbeat.health === "ONLINE_ONLY") {
    setAlert("info", "Heartbeat session online", heartbeat.message);
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
  if (!clientState.pollStarted) {
    clientState.pollStarted = true;
    updateAPIConnectionIndicator("disconnected");
  }
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

function fmtClockTime(value) {
  const ts = numericTime(value);
  if (!ts) return "-";
  return new Date(ts).toLocaleTimeString();
}

function updateAPIConnectionIndicator(state) {
  const indicator = $("api-connection-indicator");
  const stateValue = String(state || "disconnected").trim().toLowerCase();
  const stateEl = $("api-connection-state");
  const lastSuccess = $("api-connection-last-success");
  const attempts = $("api-connection-poll-attempts");
  const startup = $("api-connection-startup");
  const lastError = $("api-connection-last-error");
  let className = "api-connection-disconnected";
  if (stateValue === "connected") {
    className = "api-connection-connected";
  } else if (stateValue === "attempting") {
    className = "api-connection-attempting";
  }
  indicator.className = "api-connection-indicator " + className;
  stateEl.textContent = stateValue;
  lastSuccess.textContent = clientState.lastStatusPollAt
    ? ("last success: " + new Date(clientState.lastStatusPollAt).toLocaleTimeString())
    : "last success: never";
  attempts.textContent = "poll attempts: " + String(clientState.pollAttempts || 0);
  startup.textContent = "JS loaded at: " + fmtClockTime(clientState.jsLoadedAt) +
    " | poll started: " + (clientState.pollStarted ? "yes" : "no");
  lastError.textContent = clientState.lastError
    ? ("last error: " + clientState.lastError)
    : "last error: none";
}

function settledFailureSummary(label, result) {
  if (!result || result.status !== "rejected") return "";
  const reason = result.reason;
  const detail = reason && reason.message
    ? String(reason.message).trim()
    : String(reason || "").trim();
  if (!detail) {
    return label + " unavailable";
  }
  return label + " unavailable: " + detail;
}

function showAPIFailureBanner(summary) {
  const banner = $("api-failure-banner");
  banner.classList.remove("api-banner-hidden");
  $("api-failure-detail").textContent = summary || "One or more API requests failed; showing cached data where available.";
}

function hideAPIFailureBanner() {
  $("api-failure-banner").classList.add("api-banner-hidden");
}

function safePollRender(label, renderFn) {
  try {
    renderFn();
    return true;
  } catch (err) {
    const detail = err && err.message ? err.message : String(err || "unknown render error");
    clientState.lastError = label + ": " + detail;
    showAPIFailureBanner(clientState.lastError);
    return false;
  }
}

async function pollOnce() {
  if (clientState.inFlight) return;
  clientState.pollAttempts = Number(clientState.pollAttempts || 0) + 1;
  clientState.inFlight = true;
  updateAPIConnectionIndicator("attempting");
  const refreshButton = $("refresh-button");
  if (refreshButton) {
    refreshButton.disabled = true;
  }
  safePollRender("syncOperatorActionsFromCurrentState failed", () => syncOperatorActionsFromCurrentState());

  const baseline = copySnapshot(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  const failures = [];
  let statusOK = false;
  let readyzOK = false;

  try {
    const results = await Promise.allSettled([
      fetchJSON(endpoints.status),
      fetchReadyz(),
      fetchJSON(endpoints.incidents),
      fetchJSON(egmHistoryEndpointURL()),
      fetchJSON(endpoints.egmRegistry),
      fetchJSON(endpoints.stateHistory),
      fetchJSON(endpoints.runMarkers),
      fetchJSON(endpoints.operatorDrill),
      fetchJSON(endpoints.certificates),
      fetchJSON(operatorAuditEndpointURL()),
      fetchJSON(endpoints.sessionEvidence),
      fetchJSON(endpoints.sessionWorkflow),
      fetchJSON(endpoints.cabinetProfile),
      fetchJSON(endpoints.cabinetProfileSuggestions),
      fetchJSON(endpoints.heartbeatPolicy),
      fetchJSON(endpoints.blockerPolicy),
      fetchJSON(endpoints.blockerPolicySuggestions),
      fetchJSON(endpoints.cabinetPreflight),
      fetchJSON(endpoints.endpointIntegrityAlerts),
      fetchJSON(endpoints.runtimeOverridePresets)
    ]);

    const [statusResult, readyzResult, incidentsResult, egmHistoryResult, egmRegistryResult, stateHistoryResult, runMarkersResult, operatorDrillResult, certificatesResult, operatorAuditResult, sessionEvidenceResult, sessionWorkflowResult, cabinetProfileResult, cabinetProfileSuggestionsResult, heartbeatPolicyResult, blockerPolicyResult, blockerPolicySuggestionsResult, cabinetPreflightResult, endpointIntegrityAlertsResult, runtimeOverridePresetsResult] = results;
    const snapshot = copySnapshot(baseline);

    if (statusResult.status === "fulfilled") {
      snapshot.status = statusResult.value;
      statusOK = true;
      clientState.lastStatusPollAt = Date.now();
    } else {
      failures.push(settledFailureSummary("status", statusResult));
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
      failures.push(settledFailureSummary("readyz", readyzResult));
    }

    if (incidentsResult.status === "fulfilled") snapshot.incidents = incidentsResult.value;
    if (egmHistoryResult.status === "fulfilled") snapshot.egmHistory = egmHistoryResult.value;
    if (egmRegistryResult.status === "fulfilled") snapshot.egmRegistry = normalizeEGMRegistryResponse(egmRegistryResult.value);
    if (stateHistoryResult.status === "fulfilled") snapshot.stateHistory = stateHistoryResult.value;
    if (runMarkersResult.status === "fulfilled") snapshot.runMarkers = runMarkersResult.value;
    if (operatorDrillResult.status === "fulfilled") snapshot.operatorDrill = operatorDrillResult.value;
    if (certificatesResult.status === "fulfilled") snapshot.certificates = certificatesResult.value;
    if (operatorAuditResult.status === "fulfilled") snapshot.operatorAudit = operatorAuditResult.value;
    if (sessionEvidenceResult.status === "fulfilled") snapshot.sessionEvidence = sessionEvidenceResult.value;
    if (sessionWorkflowResult.status === "fulfilled") snapshot.sessionWorkflow = normalizeSessionWorkflowProgress(sessionWorkflowResult.value);
    if (cabinetProfileResult.status === "fulfilled") snapshot.cabinetProfile = cabinetProfileResult.value;
    if (cabinetProfileSuggestionsResult.status === "fulfilled") snapshot.cabinetProfileSuggestions = normalizeCabinetProfileSuggestions(cabinetProfileSuggestionsResult.value);
    if (heartbeatPolicyResult.status === "fulfilled") snapshot.heartbeatPolicy = heartbeatPolicyResult.value;
    if (blockerPolicyResult.status === "fulfilled") snapshot.blockerPolicy = normalizeBlockerPolicyResponse(blockerPolicyResult.value);
    if (blockerPolicySuggestionsResult.status === "fulfilled") snapshot.blockerPolicySuggestions = normalizeBlockerPolicySuggestions(blockerPolicySuggestionsResult.value);
    if (cabinetPreflightResult.status === "fulfilled") snapshot.cabinetPreflight = cabinetPreflightResult.value;
    if (endpointIntegrityAlertsResult.status === "fulfilled") snapshot.endpointIntegrityAlerts = normalizeEndpointIntegrityAlertsResponse(endpointIntegrityAlertsResult.value);
    if (runtimeOverridePresetsResult.status === "fulfilled") snapshot.runtimeOverridePresets = normalizeRuntimeOverridePresetListResponse(runtimeOverridePresetsResult.value);

    const settledFailures = [
      settledFailureSummary("incidents", incidentsResult),
      settledFailureSummary("egm history", egmHistoryResult),
      settledFailureSummary("egm registry", egmRegistryResult),
      settledFailureSummary("state history", stateHistoryResult),
      settledFailureSummary("run markers", runMarkersResult),
      settledFailureSummary("operator drill", operatorDrillResult),
      settledFailureSummary("certificates", certificatesResult),
      settledFailureSummary("operator audit", operatorAuditResult),
      settledFailureSummary("session evidence", sessionEvidenceResult),
      settledFailureSummary("session workflow", sessionWorkflowResult),
      settledFailureSummary("cabinet profile", cabinetProfileResult),
      settledFailureSummary("cabinet profile suggestions", cabinetProfileSuggestionsResult),
      settledFailureSummary("heartbeat policy", heartbeatPolicyResult),
      settledFailureSummary("blocker policy", blockerPolicyResult),
      settledFailureSummary("blocker policy suggestions", blockerPolicySuggestionsResult),
      settledFailureSummary("cabinet preflight", cabinetPreflightResult),
      settledFailureSummary("endpoint integrity alerts", endpointIntegrityAlertsResult),
      settledFailureSummary("runtime override presets", runtimeOverridePresetsResult)
    ].filter(Boolean);
    failures.push(...settledFailures);

    clientState.displaySnapshot = snapshot;
    const renderStatusOK = safePollRender("renderStatus failed", () => renderStatus(snapshot));
    const renderAlertsOK = safePollRender("renderAlerts failed", () => renderAlerts(snapshot));
    const renderOK = renderStatusOK && renderAlertsOK;
    if (failures.length > 0) {
      showAPIFailureBanner(failures.join("; "));
    } else {
      hideAPIFailureBanner();
    }

    const goodPoll = statusOK && readyzOK && renderOK;
    if (goodPoll) {
      clientState.lastGoodStatus = copySnapshot(snapshot);
      clientState.lastGoodAt = Date.now();
      clientState.lastError = "";
      resetBackoff();
      updateRefreshText(true);
      updateAPIConnectionIndicator("connected");
      updateStaleBadge();
      schedulePoll(clientState.pollIntervalMs);
    } else {
      if (!clientState.lastError) {
        clientState.lastError = failures.filter(Boolean).join("; ") || "poll incomplete";
      }
      updateRefreshText(false);
      updateAPIConnectionIndicator("disconnected");
      updateStaleBadge();
      schedulePoll(nextBackoffMs());
    }
  } catch (err) {
    clientState.lastError = err && err.message ? err.message : "poll failed";
    showAPIFailureBanner(clientState.lastError);
    if (clientState.lastGoodStatus && !clientState.displaySnapshot) {
      clientState.displaySnapshot = copySnapshot(clientState.lastGoodStatus);
      safePollRender("renderStatus failed", () => renderStatus(clientState.displaySnapshot));
      safePollRender("renderAlerts failed", () => renderAlerts(clientState.displaySnapshot));
    }
    updateRefreshText(false);
    updateAPIConnectionIndicator("disconnected");
    updateStaleBadge();
    schedulePoll(nextBackoffMs());
  } finally {
    clientState.inFlight = false;
    if (refreshButton) {
      refreshButton.disabled = false;
    }
    safePollRender("syncOperatorActionsFromCurrentState failed", () => syncOperatorActionsFromCurrentState());
    updateAPIConnectionIndicator(clientState.lastError ? "disconnected" : "connected");
  }
}

function setFilter(filter) {
  clientState.egmFilter = filter;
  document.querySelectorAll(".filter-tab").forEach((tab) => {
    tab.classList.toggle("is-active", tab.dataset.filter === filter);
  });
  if (clientState.displaySnapshot) {
    renderEGMTable(clientState.displaySnapshot.status || {});
    renderEndpointIntegrity(clientState.displaySnapshot);
    renderAlerts(clientState.displaySnapshot);
  }
}

function setEGMFocus(value) {
  clientState.egmFocusID = String(value || "").trim();
  const snapshot = clientState.displaySnapshot || clientState.lastGoodStatus || null;
  if (!snapshot) {
    const empty = emptySnapshot();
    renderEGMFocusControl(empty);
    renderEGMGroupedSummary(empty);
    renderSelectedEGMDetail(empty);
    renderHeartbeatSummary(empty);
    renderCabinetRunTimeline(empty);
    renderEGMHistory(empty);
    return;
  }
  renderStatus(snapshot);
  renderAlerts(snapshot);
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
  $("operator-actions-refresh-button").addEventListener("click", () => {
    schedulePoll(0);
  });
  $("operator-actions-capture-evidence-button").addEventListener("click", () => {
    saveSessionEvidenceToHistory();
  });
  $("operator-actions-export-package-button").addEventListener("click", () => {
    exportSessionPackage();
  });
  $("operator-actions-blocker-governance-button").addEventListener("click", () => {
    const anchor = $("blocker-governance-anchor") || $("blocker-policy-form");
    if (!anchor) return;
    anchor.scrollIntoView({ behavior: "smooth", block: "start" });
  });

  $("global-severity-filter").addEventListener("change", (event) => {
    setGlobalSeverityFilter(event.target.value || "all");
  });
  $("global-text-filter").addEventListener("input", (event) => {
    setGlobalTextFilter(event.target.value || "");
  });
  $("global-compact-mode-toggle").addEventListener("change", (event) => {
    setCompactMode(event.target.checked === true);
  });

  $("egm-focus-select").addEventListener("change", (event) => {
    setEGMFocus(event.target.value || "");
  });

  $("session-evidence-save-button").addEventListener("click", saveSessionEvidenceToHistory);
  $("session-evidence-json-button").addEventListener("click", exportSessionEvidenceJSON);
  $("session-evidence-markdown-button").addEventListener("click", exportSessionEvidenceMarkdown);
  $("session-evidence-export-all-button").addEventListener("click", exportAllSavedSessionEvidence);
  $("endpoint-integrity-filter-button").addEventListener("click", () => {
    if (clientState.egmFilter === "endpoint_integrity") {
      setFilter("all");
      return;
    }
    setFilter("endpoint_integrity");
  });
  $("endpoint-integrity-refresh-button").addEventListener("click", () => {
    reloadEndpointIntegrityAlerts();
  });
  $("endpoint-integrity-list").addEventListener("click", (event) => {
    const button = event.target.closest(".endpoint-integrity-action-button");
    if (!button) return;
    handleEndpointIntegrityAlertActionFromUI(button);
  });
  $("endpoint-integrity-acked-list").addEventListener("click", (event) => {
    const button = event.target.closest(".endpoint-integrity-action-button");
    if (!button) return;
    handleEndpointIntegrityAlertActionFromUI(button);
  });
  $("endpoint-integrity-snoozed-list").addEventListener("click", (event) => {
    const button = event.target.closest(".endpoint-integrity-action-button");
    if (!button) return;
    handleEndpointIntegrityAlertActionFromUI(button);
  });
  $("egm-select-all-visible-button").addEventListener("click", () => {
    setAllVisibleEGMBulkSelection(true);
    renderEGMTable((clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot()).status || {});
  });
  $("egm-clear-selection-button").addEventListener("click", () => {
    clearEGMBulkSelection();
    renderEGMTable((clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot()).status || {});
    setEGMBulkActionMessage("Selection cleared.");
  });
  $("egm-bulk-promote-button").addEventListener("click", promoteSelectedEGMBulk);
  $("egm-bulk-apply-profile-button").addEventListener("click", applySelectedToCabinetProfileFirstTest);
  $("egm-select-all-checkbox").addEventListener("change", (event) => {
    setAllVisibleEGMBulkSelection(event.target.checked === true);
    renderEGMTable((clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot()).status || {});
  });
  $("egm-table").addEventListener("change", (event) => {
    const checkbox = event.target.closest(".egm-row-select-checkbox");
    if (!checkbox) return;
    const egmID = checkbox.getAttribute("data-egm-id") || "";
    setEGMBulkSelectionForID(egmID, checkbox.checked === true);
    renderEGMTable((clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot()).status || {});
  });
  $("egm-table").addEventListener("click", async (event) => {
    const button = event.target.closest(".egm-row-action-button");
    if (!button) return;
    const egmID = button.getAttribute("data-egm-id") || "";
    const action = button.getAttribute("data-egm-action") || "";
    if (!egmID || !action) return;
    if (action === "edit") {
      openEGMRegistryDrawerFor(egmID);
      return;
    }
    if (action === "promote") {
      openEGMRegistryDrawerFor(egmID);
      await promoteSelectedEGMRegistry();
    }
  });
  $("egm-registry-form").addEventListener("submit", saveSelectedEGMRegistry);
  $("egm-registry-delete-button").addEventListener("click", deleteSelectedEGMRegistryOverride);
  $("egm-registry-promote-button").addEventListener("click", promoteSelectedEGMRegistry);
  $("egm-registry-close-button").addEventListener("click", closeEGMRegistryDrawer);
  $("session-package-export-button").addEventListener("click", exportSessionPackage);
  $("runtime-overrides-save-snapshot-button").addEventListener("click", saveRuntimeOverridesSnapshot);
  $("runtime-overrides-restore-button").addEventListener("click", restoreRuntimeOverridesSnapshot);
  $("runtime-overrides-restore-file").addEventListener("change", loadRuntimeOverridesRestoreFile);
  $("runtime-preset-save-button").addEventListener("click", saveCurrentRuntimePreset);
  $("runtime-preset-refresh-button").addEventListener("click", reloadRuntimePresetList);
  $("runtime-presets-list").addEventListener("click", (event) => {
    const button = event.target.closest(".runtime-preset-action-button");
    if (!button) return;
    const action = String(button.getAttribute("data-runtime-preset-action") || "").trim();
    const presetName = String(button.getAttribute("data-runtime-preset-name") || "").trim();
    if (!action || !presetName) return;
    if (action === "load") {
      loadRuntimePresetByName(presetName);
      return;
    }
    if (action === "delete") {
      deleteRuntimePresetByName(presetName);
    }
  });
  $("workflow-progress-save-button").addEventListener("click", saveSessionWorkflowProgress);
  $("workflow-progress-clear-button").addEventListener("click", clearSessionWorkflowProgress);
  $("workflow-progress-phase").addEventListener("change", updateWorkflowProgressDirtyState);
  $("workflow-progress-notes").addEventListener("input", updateWorkflowProgressDirtyState);
  $("workflow-progress-steps").addEventListener("change", updateWorkflowProgressDirtyState);
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
  $("run-marker-start-button").addEventListener("click", () => {
    submitRunMarker("start");
  });
  $("run-marker-note-button").addEventListener("click", () => {
    submitRunMarker("note");
  });
  $("run-marker-end-button").addEventListener("click", () => {
    submitRunMarker("end");
  });
  $("operator-drill-comms-online-button").addEventListener("click", () => {
    submitOperatorDrillAction("comms_online");
  });
  $("operator-drill-keepalive-button").addEventListener("click", () => {
    submitOperatorDrillAction("keepalive");
  });
  $("operator-drill-burst-button").addEventListener("click", () => {
    submitOperatorDrillAction("keepalive_burst");
  });
  $("operator-drill-resume-button").addEventListener("click", () => {
    submitOperatorDrillAction("resume");
  });
  $("operator-drill-pause-button").addEventListener("click", () => {
    submitOperatorDrillAction("pause");
  });
  $("operator-drill-clear-button").addEventListener("click", () => {
    submitOperatorDrillAction("clear");
  });
  $("operator-drill-egm-id").addEventListener("change", () => {
    renderOperatorDrill(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("operator-drill-interval-ms").addEventListener("input", () => {
    renderOperatorDrill(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("operator-drill-burst-count").addEventListener("input", () => {
    renderOperatorDrill(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("run-report-start-marker").addEventListener("change", (event) => {
    clientState.selectedRunReportStartID = Number(event.target.value) || 0;
    renderRunReportControls(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("run-report-end-marker").addEventListener("change", (event) => {
    clientState.selectedRunReportEndID = Number(event.target.value) || 0;
    renderRunReportControls(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("run-report-json-button").addEventListener("click", exportRunReportJSON);
  $("run-report-markdown-button").addEventListener("click", exportRunReportMarkdown);
  $("heartbeat-policy-form").addEventListener("submit", saveHeartbeatPolicyOverride);
  $("heartbeat-policy-clear-button").addEventListener("click", clearHeartbeatPolicyOverride);
  $("heartbeat-policy-reload-button").addEventListener("click", () => {
    reloadHeartbeatPolicyForm().catch((err) => {
      $("heartbeat-policy-message").textContent = err && err.message ? err.message : "Heartbeat policy reload failed.";
    });
  });
  $("heartbeat-policy-warning-after-missed").addEventListener("input", () => {
    renderHeartbeatPolicy(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("heartbeat-policy-interval").addEventListener("input", () => {
    renderHeartbeatPolicy(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("heartbeat-policy-block-after-missed").addEventListener("input", () => {
    renderHeartbeatPolicy(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("blocker-policy-form").addEventListener("submit", saveBlockerPolicyOverride);
  $("blocker-policy-clear-button").addEventListener("click", clearBlockerPolicyOverride);
  $("blocker-policy-reload-button").addEventListener("click", () => {
    reloadBlockerPolicyForm().catch((err) => {
      $("blocker-policy-message").textContent = err && err.message ? err.message : "Blocker policy reload failed.";
    });
  });
  $("blocker-policy-approved-ids").addEventListener("input", () => {
    renderBlockerGovernance(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("blocker-policy-finding-id").addEventListener("input", () => {
    renderBlockerGovernance(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("blocker-policy-rationale").addEventListener("input", () => {
    renderBlockerGovernance(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("blocker-policy-approve-button").addEventListener("click", approveSelectedBlockerFinding);
  $("blocker-policy-revoke-button").addEventListener("click", revokeSelectedBlockerFinding);
  $("blocker-policy-suggestions-list").addEventListener("click", (event) => {
    const selectButton = event.target.closest(".blocker-policy-select-suggestion");
    if (selectButton) {
      $("blocker-policy-finding-id").value = String(selectButton.getAttribute("data-finding-id") || "").trim();
      renderBlockerGovernance(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
      return;
    }
    const approveButton = event.target.closest(".blocker-policy-approve-suggestion");
    if (approveButton) {
      $("blocker-policy-finding-id").value = String(approveButton.getAttribute("data-finding-id") || "").trim();
      approveSelectedBlockerFinding();
    }
  });

  $("cert-manager-form").addEventListener("submit", importCertificateMaterial);
  $("cert-preview-button").addEventListener("click", previewCertificateMaterial);
  $("cert-backup-refresh-button").addEventListener("click", () => {
    loadCertificateBackups(selectedCertRole(), true).catch((err) => {
      setCertManagerState("blocked", err && err.message ? err.message : "Backup history refresh failed.");
    });
  });
  $("cert-backup-list").addEventListener("click", (event) => {
    const button = event.target.closest(".cert-restore-backup-button");
    if (!button || button.disabled) {
      return;
    }
    const backupID = button.getAttribute("data-backup-id") || "";
    restoreCertificateBackup(backupID);
  });
  $("cert-role-select").addEventListener("change", () => {
    clientState.certSelectedRole = selectedCertRole();
    clearCertificatePreviewState();
    renderCertificateManager(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("cert-certificate-pem").addEventListener("input", () => {
    clearCertificatePreviewState();
    renderCertificateManager(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
  });
  $("cert-private-key-pem").addEventListener("input", () => {
    clearCertificatePreviewState();
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

  $("operator-audit-action-filter").addEventListener("change", () => {
    clientState.operatorAuditActionFilter = String($("operator-audit-action-filter").value || "").trim();
    schedulePoll(0);
  });
  $("operator-audit-result-filter").addEventListener("change", () => {
    clientState.operatorAuditResultFilter = String($("operator-audit-result-filter").value || "").trim();
    schedulePoll(0);
  });
  $("operator-audit-search-filter").addEventListener("input", () => {
    clientState.operatorAuditSearchFilter = String($("operator-audit-search-filter").value || "").trim();
    schedulePoll(0);
  });

  $("cabinet-setup-form").addEventListener("submit", saveCabinetProfileOverride);
  $("setup-reset-button").addEventListener("click", clearCabinetProfileOverride);
  $("setup-use-observed-egms-button").addEventListener("click", useObservedEGMSuggestions);
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

  document.querySelectorAll(".timeline-filter-tab").forEach((button) => {
    button.addEventListener("click", () => {
      clientState.timelineFilter = button.dataset.timelineFilter || "all";
      renderCabinetRunTimeline(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
    });
  });
  $("timeline-rollup-heartbeat-toggle").addEventListener("change", (event) => {
    clientState.timelineRollupHeartbeat = event.target.checked === true;
    renderCabinetRunTimeline(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
    schedulePoll(0);
  });
  $("timeline-show-raw-heartbeat-toggle").addEventListener("change", (event) => {
    clientState.timelineShowRawHeartbeat = event.target.checked === true;
    renderCabinetRunTimeline(clientState.displaySnapshot || clientState.lastGoodStatus || emptySnapshot());
    schedulePoll(0);
  });

  document.querySelectorAll(".sort-button").forEach((button) => {
    button.addEventListener("click", () => {
      setSort(button.dataset.sortKey || "egm_id");
    });
  });
}

function runStartupStep(label, fn) {
  try {
    fn();
    return true;
  } catch (err) {
    const detail = err && err.message ? err.message : String(err || "startup error");
    clientState.lastError = label + ": " + detail;
    showAPIFailureBanner(clientState.lastError);
    updateRefreshText(false);
    updateAPIConnectionIndicator("disconnected");
    return false;
  }
}

function bootstrapDashboard() {
  const empty = emptySnapshot();
  runStartupStep("loadCompactModePreference failed", loadCompactModePreference);
  runStartupStep("bindControls failed", bindControls);
  runStartupStep("renderGlobalViewControls failed", renderGlobalViewControls);
  runStartupStep("updateSortLabels failed", updateSortLabels);
  runStartupStep("updateStaleBadge failed", updateStaleBadge);
  runStartupStep("updateAPIConnectionIndicator failed", () => updateAPIConnectionIndicator("disconnected"));
  runStartupStep("renderEGMFocusControl failed", () => renderEGMFocusControl(empty));
  runStartupStep("renderEGMGroupedSummary failed", () => renderEGMGroupedSummary(empty));
  runStartupStep("renderSelectedEGMDetail failed", () => renderSelectedEGMDetail(empty));
  runStartupStep("renderOperatorAuditTimeline failed", () => renderOperatorAuditTimeline(empty));
  runStartupStep("renderCertificateManager failed", () => renderCertificateManager(empty));
  runStartupStep("renderFirstCabinetSession failed", () => renderFirstCabinetSession(empty));
  runStartupStep("renderRuntimePresetLibrary failed", () => renderRuntimePresetLibrary(empty));
  runStartupStep("renderSessionEvidence failed", () => renderSessionEvidence(empty));
  runStartupStep("renderRunMarkerControls failed", () => renderRunMarkerControls(empty));
  runStartupStep("renderRunReportControls failed", () => renderRunReportControls(empty));
  runStartupStep("renderHeartbeatPolicy failed", () => renderHeartbeatPolicy(empty));
  runStartupStep("renderBlockerGovernance failed", () => renderBlockerGovernance(empty));
  runStartupStep("renderOperatorDrill failed", () => renderOperatorDrill(empty));
  runStartupStep("renderHeartbeatSummary failed", () => renderHeartbeatSummary(empty));
  runStartupStep("renderCabinetRunTimeline failed", () => renderCabinetRunTimeline(empty));
  runStartupStep("renderEGMHistory failed", () => renderEGMHistory(empty));
  runStartupStep("syncOperatorActionsFromCurrentState failed", syncOperatorActionsFromCurrentState);
  runStartupStep("poll startup failed", () => schedulePoll(0));
  runStartupStep("stale badge timer failed", () => setInterval(updateStaleBadge, 1000));
}

bootstrapDashboard();`
