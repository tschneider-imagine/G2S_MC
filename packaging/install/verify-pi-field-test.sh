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

echo "service status"
systemctl status g2s-mute --no-pager || true

check_path "/healthz" "200"
check_path "/operator" "200"
check_path "/operator/inputs" "200"
check_path "/operator/actions" "200"
check_path "/operator/comms" "200"
check_path "/operator/audit" "200"
check_path "/operator/settings" "200"
check_path "/field-test" "404"
check_path "/dashboard" "404"

echo "verification complete"
