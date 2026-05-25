#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8444}"

check_path() {
  local path="$1"
  local expected="$2"
  local code
  code="$(curl -s -o /dev/null -w '%{http_code}' "${BASE_URL}${path}" || true)"
  echo "${code} ${path}"
  if [[ "${code}" != "${expected}" ]]; then
    echo "expected ${expected} for ${path}, got ${code}" >&2
    return 1
  fi
}

check_path_any() {
  local path="$1"
  shift
  local code
  code="$(curl -s -o /dev/null -w '%{http_code}' "${BASE_URL}${path}" || true)"
  echo "${code} ${path}"
  local expected
  for expected in "$@"; do
    if [[ "${code}" == "${expected}" ]]; then
      return 0
    fi
  done
  echo "expected one of [$*] for ${path}, got ${code}" >&2
  return 1
}

echo "service status"
systemctl status g2s-mute --no-pager || true

check_path "/healthz" "200"
check_path_any "/readyz" "200" "503" "404"
check_path "/operator" "200"
check_path "/operator/inputs" "200"
check_path "/operator/actions" "200"
check_path "/operator/comms" "200"
check_path "/operator/egms" "200"
check_path "/operator/templates" "200"
check_path "/operator/audit" "200"
check_path "/operator/settings" "200"
check_path "/field-test" "404"
check_path "/dashboard" "404"
check_path "/static/dashboard.js" "404"
check_path "/static/dashboard.css" "404"
check_path "/operator/readiness" "404"
check_path "/operator/readiness.json" "404"
check_path "/operator/settings/system-check" "404"
check_path "/operator/settings/system-check.json" "404"

echo "verification complete"
