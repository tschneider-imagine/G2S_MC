#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:8444}"
API_TOKEN="${API_TOKEN:-}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
AUTH_SMOKE_SCRIPT="${SCRIPT_DIR}/api-auth-smoke.sh"
REPORT_DIR="${REPO_ROOT}/docs/pi-runs"
REPORT_PATH="${REPORT_DIR}/$(date -u +%F)-release-gate-run.md"

mkdir -p "${REPORT_DIR}"

if [[ "${API_BASE}" == https://* ]]; then
  CURL_TLS=(-k)
else
  CURL_TLS=()
fi

TMP_DIR="$(mktemp -d)"
SUMMARY_FILE="${TMP_DIR}/summary.txt"
AUTH_OUTPUT_FILE="${TMP_DIR}/api-auth-smoke.txt"
PREFLIGHT_BODY_FILE="${TMP_DIR}/preflight.json"
PREFLIGHT_ERROR_FILE="${TMP_DIR}/preflight.err"

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

declare -a ROWS=()
GATE_FAIL=0

record_row() {
  local name="$1"
  local result="$2"
  local detail="$3"
  ROWS+=("${name}|${result}|${detail}")
  if [[ "${result}" == "FAIL" ]]; then
    GATE_FAIL=1
  fi
}

error_text() {
  local file="$1"
  if [[ ! -s "${file}" ]]; then
    return 0
  fi
  tr '\n' ' ' <"${file}" | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//'
}

http_code_for_get() {
  local url="$1"
  local body_file="$2"
  local err_file="$3"
  curl "${CURL_TLS[@]}" -sS -o "${body_file}" -w '%{http_code}' "${url}" 2>"${err_file}" || true
}

check_http_ok() {
  local label="$1"
  local path="$2"
  local body_file="${TMP_DIR}/${label}.body"
  local err_file="${TMP_DIR}/${label}.err"
  local code
  code="$(http_code_for_get "${API_BASE%/}${path}" "${body_file}" "${err_file}")"
  if [[ "${code}" == "200" ]]; then
    record_row "${label}" "PASS" "HTTP 200"
  else
    local err
    err="$(error_text "${err_file}")"
    if [[ -n "${err}" ]]; then
      record_row "${label}" "FAIL" "HTTP ${code}; ${err}"
    else
      record_row "${label}" "FAIL" "HTTP ${code}"
    fi
  fi
}

check_http_ok "healthz" "/healthz"
check_http_ok "readyz" "/readyz"
check_http_ok "status" "/api/status"

preflight_code="$(http_code_for_get "${API_BASE%/}/api/cabinet-preflight" "${PREFLIGHT_BODY_FILE}" "${PREFLIGHT_ERROR_FILE}")"
if [[ "${preflight_code}" != "200" ]]; then
  preflight_err="$(error_text "${PREFLIGHT_ERROR_FILE}")"
  if [[ -n "${preflight_err}" ]]; then
    record_row "cabinet-preflight" "FAIL" "HTTP ${preflight_code}; ${preflight_err}"
  else
    record_row "cabinet-preflight" "FAIL" "HTTP ${preflight_code}"
  fi
else
  preflight_payload="$(tr -d '\n' <"${PREFLIGHT_BODY_FILE}")"
  preflight_overall="$(printf '%s' "${preflight_payload}" | sed -n 's/.*"overall"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  if [[ "${preflight_overall}" == "PASS" ]]; then
    record_row "cabinet-preflight" "PASS" "overall=PASS"
  elif [[ "${preflight_overall}" == "FAIL" ]]; then
    issues_raw="$(printf '%s' "${preflight_payload}" | sed -n 's/.*"issues"[[:space:]]*:[[:space:]]*\[\(.*\)\][[:space:]]*,[[:space:]]*"warnings".*/\1/p')"
    if [[ -n "${issues_raw}" ]]; then
      issues_text="$(printf '%s' "${issues_raw}" | sed 's/","/; /g; s/^"//; s/"$//; s/\\"/"/g')"
      record_row "cabinet-preflight" "FAIL" "overall=FAIL; ${issues_text}"
    else
      record_row "cabinet-preflight" "FAIL" "overall=FAIL"
    fi
  else
    record_row "cabinet-preflight" "FAIL" "unable to parse overall from JSON"
  fi
fi

if [[ ! -x "${AUTH_SMOKE_SCRIPT}" ]]; then
  record_row "api-auth-smoke" "FAIL" "missing executable ${AUTH_SMOKE_SCRIPT}"
  printf 'script not executable: %s\n' "${AUTH_SMOKE_SCRIPT}" >"${AUTH_OUTPUT_FILE}"
else
  if API_BASE="${API_BASE}" API_TOKEN="${API_TOKEN}" "${AUTH_SMOKE_SCRIPT}" >"${AUTH_OUTPUT_FILE}" 2>&1; then
    auth_mode="$(sed -n 's/^auth_required_by_runtime -> //p' "${AUTH_OUTPUT_FILE}" | tail -n 1)"
    if [[ -z "${auth_mode}" ]]; then
      auth_mode="unknown"
    fi
    cert_with_token_line="$(grep -m1 '^POST /api/certificates/import (with token) -> ' "${AUTH_OUTPUT_FILE}" || true)"
    if [[ -n "${cert_with_token_line}" ]]; then
      record_row "api-auth-smoke" "PASS" "auth_required=${auth_mode}; ${cert_with_token_line}"
    else
      record_row "api-auth-smoke" "PASS" "auth_required=${auth_mode}"
    fi
  else
    auth_failure="$(grep -m1 '(FAIL' "${AUTH_OUTPUT_FILE}" || true)"
    if [[ -z "${auth_failure}" ]]; then
      auth_failure="$(tail -n 1 "${AUTH_OUTPUT_FILE}" | tr '\n' ' ' | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//')"
    fi
    if [[ -z "${auth_failure}" ]]; then
      auth_failure="smoke script exited non-zero"
    fi
    record_row "api-auth-smoke" "FAIL" "${auth_failure}"
  fi
fi

{
  printf 'check                 result detail\n'
  for row in "${ROWS[@]}"; do
    IFS='|' read -r name result detail <<<"${row}"
    printf '%-21s %-6s %s\n' "${name}" "${result}" "${detail}"
  done
  if [[ "${GATE_FAIL}" -eq 0 ]]; then
    printf '%-21s %-6s %s\n' "overall" "PASS" "release gate passed"
  else
    printf '%-21s %-6s %s\n' "overall" "FAIL" "release gate failed"
  fi
} | tee "${SUMMARY_FILE}"

{
  printf '# Release Gate Run (%s)\n\n' "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  printf -- '- API_BASE: `%s`\n' "${API_BASE}"
  if [[ -n "${API_TOKEN}" ]]; then
    printf -- '- API_TOKEN: set\n'
  else
    printf -- '- API_TOKEN: not set\n'
  fi
  printf '\n## Summary\n\n'
  printf '```text\n'
  cat "${SUMMARY_FILE}"
  printf '```\n\n'
  printf '## Cabinet Preflight JSON\n\n'
  if [[ -s "${PREFLIGHT_BODY_FILE}" ]]; then
    printf '```json\n'
    cat "${PREFLIGHT_BODY_FILE}"
    printf '\n```\n\n'
  else
    printf '_Unavailable in this run._\n\n'
  fi
  printf '## API Auth Smoke Output\n\n'
  if [[ -s "${AUTH_OUTPUT_FILE}" ]]; then
    printf '```text\n'
    cat "${AUTH_OUTPUT_FILE}"
    printf '\n```\n'
  else
    printf '_Unavailable in this run._\n'
  fi
} >"${REPORT_PATH}"

echo "report: ${REPORT_PATH}"

if [[ "${GATE_FAIL}" -ne 0 ]]; then
  exit 1
fi
