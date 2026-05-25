package fieldtestui

const fieldTestCSS = `
:root {
  color-scheme: light;
  font-family: "Segoe UI", Tahoma, sans-serif;
}
body {
  margin: 0;
  background: #f6f8fb;
  color: #102032;
}
header {
  background: #0d2747;
  color: #ffffff;
  padding: 14px 20px;
}
header h1 {
  margin: 0 0 10px 0;
  font-size: 20px;
}
nav a {
  color: #d7e5ff;
  margin-right: 14px;
  text-decoration: none;
  font-weight: 600;
}
nav a.active {
  color: #ffffff;
  text-decoration: underline;
}
main {
  padding: 16px 20px 32px 20px;
}
.panel {
  background: #ffffff;
  border: 1px solid #d8e0eb;
  border-radius: 8px;
  margin-bottom: 16px;
  padding: 14px;
}
.panel h2, .panel h3 {
  margin-top: 0;
}
table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 8px;
}
th, td {
  border: 1px solid #d7deea;
  padding: 6px 8px;
  font-size: 13px;
  vertical-align: top;
}
th {
  background: #eef3f9;
  text-align: left;
}
code, pre {
  background: #f2f5f9;
  border: 1px solid #d9e2ef;
  border-radius: 4px;
}
pre {
  padding: 10px;
  white-space: pre-wrap;
  word-break: break-word;
}
label {
  display: inline-block;
  margin: 3px 12px 3px 0;
  font-size: 13px;
}
input[type=text], input[type=number], textarea, select {
  border: 1px solid #b9c8dc;
  border-radius: 4px;
  padding: 5px 6px;
  font-size: 13px;
}
textarea {
  width: 100%;
  min-height: 84px;
  font-family: "Consolas", "Courier New", monospace;
}
button {
  border: 1px solid #2e5f95;
  background: #2e5f95;
  color: #ffffff;
  border-radius: 4px;
  padding: 6px 10px;
  cursor: pointer;
  font-size: 13px;
}
.inline-form {
  margin-top: 8px;
}
.message {
  border-left: 4px solid #2e5f95;
  background: #edf4ff;
  padding: 8px 10px;
  margin-bottom: 12px;
}
.error {
  border-left: 4px solid #8d1e1e;
  background: #fff0f0;
  color: #5f0f0f;
  padding: 8px 10px;
  margin-bottom: 12px;
}
.mono {
  font-family: "Consolas", "Courier New", monospace;
}
.badge {
  display: inline-block;
  border-radius: 999px;
  padding: 2px 8px;
  font-size: 12px;
  border: 1px solid #32526d;
  color: #0f3655;
  background: #e7f2ff;
}
details {
  margin-top: 4px;
}
`
