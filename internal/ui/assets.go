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
      <button id="refresh-button" type="button">Refresh</button>
    </div>
  </header>

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
        <p class="label">Last event</p>
        <strong id="last-event">-</strong>
      </div>
      <div>
        <p class="label">Active incident</p>
        <strong id="active-incident">None</strong>
      </div>
    </section>

    <section class="grid two">
      <div class="panel wide">
        <div class="panel-head">
          <h2>EGM Roster</h2>
          <span id="egm-count">0 EGMs</span>
        </div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>EGM</th>
                <th>Status</th>
                <th>Address</th>
                <th>Game</th>
                <th>Last seen</th>
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

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 28px 36px 20px;
  border-bottom: 1px solid var(--line);
  background: rgba(244, 247, 242, 0.88);
}

.eyebrow,
.label {
  margin: 0 0 5px;
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0;
  text-transform: uppercase;
}

h1,
h2 {
  margin: 0;
  letter-spacing: 0;
}

h1 { font-size: 30px; }
h2 { font-size: 18px; }

.top-actions {
  display: flex;
  align-items: center;
  gap: 14px;
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

.shell {
  width: min(1440px, 100%);
  margin: 0 auto;
  padding: 24px 36px 40px;
}

.status-band {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  border: 1px solid var(--line);
  background: var(--line);
}

.status-band > div,
.panel {
  background: rgba(255, 255, 255, 0.92);
}

.status-band > div {
  padding: 18px;
  min-width: 0;
}

.status-band strong {
  display: block;
  overflow-wrap: anywhere;
  font-size: 18px;
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

.state-healthy,
.status-green { background: var(--green); }

.state-warning,
.status-yellow { background: var(--yellow); }

.state-emergency_active,
.status-red { background: var(--red); }

.status-grey,
.state-degraded { background: var(--grey); }

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

.item span {
  color: var(--muted);
  font-size: 13px;
}

.empty {
  padding: 18px;
  color: var(--muted);
  background: var(--panel);
}

@media (max-width: 860px) {
  .topbar {
    align-items: flex-start;
    flex-direction: column;
    padding: 22px;
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
    justify-content: space-between;
  }
}`

const dashboardJS = `const endpoints = {
  status: "/api/status",
  incidents: "/api/incidents?limit=6",
  egmHistory: "/api/egms/history?limit=8",
  stateHistory: "/api/state-history?limit=8"
};

const $ = (id) => document.getElementById(id);

async function fetchJSON(url) {
  const response = await fetch(url, { cache: "no-store" });
  if (!response.ok) throw new Error(await response.text());
  return response.json();
}

function fmtTime(value) {
  if (!value || value.startsWith("0001-")) return "-";
  return new Date(value).toLocaleString();
}

function cls(value, prefix) {
  return prefix + "-" + String(value || "").toLowerCase();
}

function setStatePill(el, value) {
  el.textContent = value || "-";
  el.className = "state-pill " + cls(value, "state");
}

function statusPill(value) {
  return "<span class=\"status-pill " + cls(value, "status") + "\">" + (value || "-") + "</span>";
}

function renderStatus(status) {
  $("controller-id").textContent = status.controller_id || "-";
  setStatePill($("controller-state"), status.state);
  $("last-event").textContent = status.last_event || "-";
  $("active-incident").textContent = status.incident ? "#" + status.incident.id + " " + status.incident.trigger_type : "None";
  $("egm-count").textContent = (status.egms?.length || 0) + " EGMs";

  const rows = (status.egms || []).map((egm) =>
    "<tr>" +
      "<td><strong>" + egm.id + "</strong><br><span>" + (egm.vendor || "") + " " + (egm.cabinet_family || "") + "</span></td>" +
      "<td>" + statusPill(egm.status) + "</td>" +
      "<td>" + egm.ip_address + ":" + egm.port + "</td>" +
      "<td>" + (egm.game_title || "-") + "<br><span>" + (egm.software_version || "") + "</span></td>" +
      "<td>" + fmtTime(egm.last_seen) + "</td>" +
    "</tr>"
  );
  $("egm-table").innerHTML = rows.length ? rows.join("") : "<tr><td colspan=\"5\">No EGMs configured</td></tr>";
}

function renderItems(id, items, emptyText, mapItem) {
  const el = $(id);
  if (!items || items.length === 0) {
    el.innerHTML = "<div class=\"empty\">" + emptyText + "</div>";
    return;
  }
  el.innerHTML = items.map(mapItem).join("");
}

async function refresh() {
  const [status, incidents, egmHistory, stateHistory] = await Promise.all([
    fetchJSON(endpoints.status),
    fetchJSON(endpoints.incidents),
    fetchJSON(endpoints.egmHistory),
    fetchJSON(endpoints.stateHistory)
  ]);

  renderStatus(status);
  renderItems("incident-list", incidents, "No incidents recorded", (item) =>
    "<div class=\"item\"><strong>#" + item.id + " " + item.trigger_type + "</strong><span>" + fmtTime(item.created_at) + " " + (item.trigger_source || "") + "</span></div>"
  );
  renderItems("egm-history", egmHistory, "No EGM history yet", (item) =>
    "<div class=\"item\"><strong>" + item.egm_id + " " + statusPill(item.status) + "</strong><span>" + item.event_type + " at " + fmtTime(item.created_at) + "</span></div>"
  );
  renderItems("state-history", stateHistory, "No state history yet", (item) =>
    "<div class=\"item\"><strong>" + item.old_state + " -> " + item.new_state + "</strong><span>" + item.reason + " at " + fmtTime(item.created_at) + "</span></div>"
  );
  $("last-refresh").textContent = "Updated " + new Date().toLocaleTimeString();
}

$("refresh-button").addEventListener("click", () => refresh().catch(console.error));
refresh().catch((err) => {
  $("last-refresh").textContent = err.message;
});
setInterval(() => refresh().catch(console.error), 3000);`
