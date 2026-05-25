#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8444}"
STRICT=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --strict)
      STRICT=1
      ;;
    -h|--help)
      echo "Usage: ./packaging/install/verify-pi-field-test.sh [--strict]"
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
  esac
  shift
done

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

echo "runtime build fingerprint"
runtime_code="$(curl -s -o /tmp/g2s-runtime.json -w '%{http_code}' "${BASE_URL}/api/v2/runtime" || true)"
if [[ "${runtime_code}" == "200" ]]; then
  runtime_revision="$(sed -n 's/.*"revision_short"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' /tmp/g2s-runtime.json | head -n 1)"
  runtime_version="$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' /tmp/g2s-runtime.json | head -n 1)"
  runtime_build_time="$(sed -n 's/.*"build_time"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' /tmp/g2s-runtime.json | head -n 1)"
  echo "runtime version=${runtime_version:-unknown} revision=${runtime_revision:-unknown} build_time=${runtime_build_time:-unknown}"
else
  echo "runtime endpoint unavailable: ${runtime_code} /api/v2/runtime"
  status_code="$(curl -s -o /tmp/g2s-status.json -w '%{http_code}' "${BASE_URL}/api/status" || true)"
  if [[ "${status_code}" == "200" ]]; then
    runtime_revision="$(sed -n 's/.*"build_revision_short"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' /tmp/g2s-status.json | head -n 1)"
    runtime_version="$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' /tmp/g2s-status.json | head -n 1)"
    runtime_build_time="$(sed -n 's/.*"build_time"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' /tmp/g2s-status.json | head -n 1)"
    echo "runtime (status endpoint) version=${runtime_version:-unknown} revision=${runtime_revision:-unknown} build_time=${runtime_build_time:-unknown}"
  else
    runtime_revision=""
  fi
fi

repo_head=""
if git rev-parse --short HEAD >/dev/null 2>&1; then
  repo_head="$(git rev-parse --short HEAD 2>/dev/null || true)"
  echo "repo head=${repo_head:-unknown}"
fi
if [[ -n "${runtime_revision}" && -n "${repo_head}" && "${runtime_revision}" != "${repo_head}" ]]; then
  echo "WARNING: RUNNING SERVICE REVISION DIFFERS FROM REPO HEAD"
  if [[ "${STRICT}" -eq 1 ]]; then
    echo "strict mode enabled; failing verification due to revision mismatch" >&2
    exit 1
  fi
fi

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

echo "message delivery check api"
delivery_check_payload='{"egm_id":"EGM-001","template_id":"template-generic-g2s-action","template_action_key":"emergency_broadcast_silence","include_network_check":false,"include_tls_check":false,"timeout_ms":5000}'
delivery_check_code="$(curl -s -o /tmp/g2s-delivery-check.json -w '%{http_code}' -X POST -H 'Content-Type: application/json' --data "${delivery_check_payload}" "${BASE_URL}/api/v2/settings/message-delivery-check" || true)"
echo "${delivery_check_code} /api/v2/settings/message-delivery-check"
if [[ "${delivery_check_code}" == "200" ]]; then
  if grep -q '"overall_status"' /tmp/g2s-delivery-check.json; then
    echo "delivery_check overall_status present"
  else
    echo "delivery_check overall_status missing"
  fi
else
  echo "delivery_check response summary:"
  sed -n '1,5p' /tmp/g2s-delivery-check.json || true
fi

echo "verification complete"
