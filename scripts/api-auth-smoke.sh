#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-https://127.0.0.1:9443}"
API_TOKEN="${API_TOKEN:-}"
CERT_PATH="${CERT_PATH:-./certs/tls-lab/host.crt}"
KEY_PATH="${KEY_PATH:-./certs/tls-lab/host.key}"

if [[ "${API_BASE}" == https://* ]]; then
  CURL_TLS=(-k)
else
  CURL_TLS=()
fi

PROFILE_PAYLOAD='{"wire_host_url":"https://localhost:9443/g2s","listener_dns_name":"localhost","listener_ip":"127.0.0.1","required_san_dns":["localhost"],"required_san_ips":["127.0.0.1"],"host_id":"HOST-LOCAL-9443","first_test_egm_ids":["EGM-TLS-01"]}'

escape_pem() {
  local path="$1"
  awk '{gsub(/\\/, "\\\\"); gsub(/"/, "\\\""); printf "%s\\n", $0}' "${path}"
}

status_code() {
  curl "${CURL_TLS[@]}" -sS -o /tmp/api-auth-smoke-body.txt -w '%{http_code}' "$@"
}

EXPECT_FAIL=0

GET_HEALTHZ="$(status_code "${API_BASE}/healthz")"
GET_STATUS="$(status_code "${API_BASE}/api/status")"
PUT_PROFILE_NO_TOKEN="$(status_code -X PUT "${API_BASE}/api/cabinet-profile" -H 'Content-Type: application/json' --data "${PROFILE_PAYLOAD}")"
DELETE_PROFILE_NO_TOKEN="$(status_code -X DELETE "${API_BASE}/api/cabinet-profile")"

if [[ ! -f "${CERT_PATH}" || ! -f "${KEY_PATH}" ]]; then
  echo "missing certificate files for import payload: CERT_PATH=${CERT_PATH} KEY_PATH=${KEY_PATH}" >&2
  exit 2
fi

CERT_ESCAPED="$(escape_pem "${CERT_PATH}")"
KEY_ESCAPED="$(escape_pem "${KEY_PATH}")"
IMPORT_PAYLOAD="$(printf '{"role":"web_server_cert","certificate_pem":"%s","private_key_pem":"%s"}' "${CERT_ESCAPED}" "${KEY_ESCAPED}")"
POST_CERT_IMPORT_NO_TOKEN="$(status_code -X POST "${API_BASE}/api/certificates/import" -H 'Content-Type: application/json' --data "${IMPORT_PAYLOAD}")"

auth_required="UNKNOWN"
if [[ "${PUT_PROFILE_NO_TOKEN}" == "401" && "${DELETE_PROFILE_NO_TOKEN}" == "401" && "${POST_CERT_IMPORT_NO_TOKEN}" == "401" ]]; then
  auth_required="YES"
elif [[ "${PUT_PROFILE_NO_TOKEN}" == "200" && ( "${DELETE_PROFILE_NO_TOKEN}" == "200" || "${DELETE_PROFILE_NO_TOKEN}" == "204" ) && "${POST_CERT_IMPORT_NO_TOKEN}" == "200" ]]; then
  auth_required="NO"
else
  auth_required="INCONSISTENT"
  EXPECT_FAIL=1
fi

if [[ -z "${API_TOKEN}" ]]; then
  PUT_PROFILE_WITH_TOKEN="SKIP"
  DELETE_PROFILE_WITH_TOKEN="SKIP"
  POST_CERT_IMPORT_WITH_TOKEN="SKIP"
else
  AUTH_HEADER="Authorization: Bearer ${API_TOKEN}"
  PUT_PROFILE_WITH_TOKEN="$(status_code -X PUT "${API_BASE}/api/cabinet-profile" -H 'Content-Type: application/json' -H "${AUTH_HEADER}" --data "${PROFILE_PAYLOAD}")"
  DELETE_PROFILE_WITH_TOKEN="$(status_code -X DELETE "${API_BASE}/api/cabinet-profile" -H "${AUTH_HEADER}")"
  POST_CERT_IMPORT_WITH_TOKEN="$(status_code -X POST "${API_BASE}/api/certificates/import" -H 'Content-Type: application/json' -H "${AUTH_HEADER}" --data "${IMPORT_PAYLOAD}")"
fi

echo "GET /healthz (no token) -> ${GET_HEALTHZ}"
echo "GET /api/status (no token) -> ${GET_STATUS}"
echo "PUT /api/cabinet-profile (no token) -> ${PUT_PROFILE_NO_TOKEN}"
echo "DELETE /api/cabinet-profile (no token) -> ${DELETE_PROFILE_NO_TOKEN}"
echo "POST /api/certificates/import (no token) -> ${POST_CERT_IMPORT_NO_TOKEN}"
echo "auth_required_by_runtime -> ${auth_required}"
echo "PUT /api/cabinet-profile (with token) -> ${PUT_PROFILE_WITH_TOKEN}"
echo "DELETE /api/cabinet-profile (with token) -> ${DELETE_PROFILE_WITH_TOKEN}"
echo "POST /api/certificates/import (with token) -> ${POST_CERT_IMPORT_WITH_TOKEN}"

if [[ "${GET_HEALTHZ}" != "200" ]]; then
  EXPECT_FAIL=1
fi
if [[ "${GET_STATUS}" != "200" ]]; then
  EXPECT_FAIL=1
fi

if [[ "${auth_required}" == "YES" ]]; then
  if [[ -n "${API_TOKEN}" ]]; then
    if [[ "${PUT_PROFILE_WITH_TOKEN}" != "200" ]]; then
      EXPECT_FAIL=1
    fi
    if [[ "${DELETE_PROFILE_WITH_TOKEN}" != "200" && "${DELETE_PROFILE_WITH_TOKEN}" != "204" ]]; then
      EXPECT_FAIL=1
    fi
    if [[ "${POST_CERT_IMPORT_WITH_TOKEN}" != "200" ]]; then
      EXPECT_FAIL=1
    fi
  fi
elif [[ "${auth_required}" == "NO" ]]; then
  if [[ "${PUT_PROFILE_NO_TOKEN}" != "200" ]]; then
    EXPECT_FAIL=1
  fi
  if [[ "${DELETE_PROFILE_NO_TOKEN}" != "200" && "${DELETE_PROFILE_NO_TOKEN}" != "204" ]]; then
    EXPECT_FAIL=1
  fi
  if [[ "${POST_CERT_IMPORT_NO_TOKEN}" != "200" ]]; then
    EXPECT_FAIL=1
  fi
fi

if [[ "${EXPECT_FAIL}" -ne 0 ]]; then
  exit 1
fi
