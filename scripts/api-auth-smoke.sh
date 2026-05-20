#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:8444}"
API_TOKEN="${API_TOKEN:-}"
PROFILE_PAYLOAD='{"wire_host_url":"https://localhost:9443/g2s","listener_dns_name":"localhost","listener_ip":"127.0.0.1","required_san_dns":["localhost"],"required_san_ips":["127.0.0.1"],"host_id":"HOST-LOCAL-9443","first_test_egm_ids":["EGM-TLS-01"]}'
CERT_IMPORT_PAYLOAD='{"role":"web_server_cert"}'
STATUS_BODY_FILE="/tmp/api-auth-smoke-status.json"

if [[ "${API_BASE}" == https://* ]]; then
  CURL_TLS=(-k)
else
  CURL_TLS=()
fi

status_code() {
  curl "${CURL_TLS[@]}" -sS -o /tmp/api-auth-smoke-body.txt -w '%{http_code}' "$@" || true
}

is_denied() {
  local code="$1"
  [[ "${code}" == "401" || "${code}" == "403" ]]
}

is_success_delete() {
  local code="$1"
  [[ "${code}" == "200" || "${code}" == "204" ]]
}

EXPECT_FAIL=0

GET_HEALTHZ="$(status_code "${API_BASE}/healthz")"
GET_STATUS="$(curl "${CURL_TLS[@]}" -sS -o "${STATUS_BODY_FILE}" -w '%{http_code}' "${API_BASE}/api/status" || true)"
RUNTIME_AUTH_REQUIRED="unknown"
RUNTIME_TRUSTED_BYPASS="unknown"
if [[ "${GET_STATUS}" == "200" && -s "${STATUS_BODY_FILE}" ]]; then
  RUNTIME_AUTH_REQUIRED="$(sed -n 's/.*"api_mutation_auth_required":[[:space:]]*\(true\|false\).*/\1/p' "${STATUS_BODY_FILE}" | head -n 1)"
  RUNTIME_TRUSTED_BYPASS="$(sed -n 's/.*"trusted_mutation_bypass_active":[[:space:]]*\(true\|false\).*/\1/p' "${STATUS_BODY_FILE}" | head -n 1)"
fi
PUT_PROFILE_NO_TOKEN="$(status_code -X PUT "${API_BASE}/api/cabinet-profile" -H 'Content-Type: application/json' --data "${PROFILE_PAYLOAD}")"
DELETE_PROFILE_NO_TOKEN="$(status_code -X DELETE "${API_BASE}/api/cabinet-profile")"
POST_CERT_IMPORT_NO_TOKEN="$(status_code -X POST "${API_BASE}/api/certificates/import" -H 'Content-Type: application/json' --data "${CERT_IMPORT_PAYLOAD}")"

PUT_PROFILE_WITH_TOKEN="SKIP"
DELETE_PROFILE_WITH_TOKEN="SKIP"
POST_CERT_IMPORT_WITH_TOKEN="SKIP"
if [[ -n "${API_TOKEN}" ]]; then
  AUTH_HEADER="Authorization: Bearer ${API_TOKEN}"
  PUT_PROFILE_WITH_TOKEN="$(status_code -X PUT "${API_BASE}/api/cabinet-profile" -H 'Content-Type: application/json' -H "${AUTH_HEADER}" --data "${PROFILE_PAYLOAD}")"
  DELETE_PROFILE_WITH_TOKEN="$(status_code -X DELETE "${API_BASE}/api/cabinet-profile" -H "${AUTH_HEADER}")"
  POST_CERT_IMPORT_WITH_TOKEN="$(status_code -X POST "${API_BASE}/api/certificates/import" -H 'Content-Type: application/json' -H "${AUTH_HEADER}" --data "${CERT_IMPORT_PAYLOAD}")"
fi

if [[ "${RUNTIME_AUTH_REQUIRED}" == "true" ]]; then
  AUTH_REQUIRED="YES"
elif [[ "${RUNTIME_AUTH_REQUIRED}" == "false" ]]; then
  AUTH_REQUIRED="NO"
else
  AUTH_REQUIRED="UNKNOWN"
fi

if [[ "${GET_HEALTHZ}" != "200" ]]; then
  EXPECT_FAIL=1
  HEALTHZ_DETAIL="FAIL expected 200"
else
  HEALTHZ_DETAIL="PASS"
fi

if [[ "${GET_STATUS}" != "200" ]]; then
  EXPECT_FAIL=1
  STATUS_DETAIL="FAIL expected 200"
else
  STATUS_DETAIL="PASS"
fi

if [[ "${RUNTIME_AUTH_REQUIRED}" == "false" ]]; then
  if [[ "${PUT_PROFILE_NO_TOKEN}" == "200" ]]; then
    PUT_NO_TOKEN_DETAIL="PASS trusted/private bypass"
  else
    EXPECT_FAIL=1
    PUT_NO_TOKEN_DETAIL="FAIL expected 200 with trusted/private bypass"
  fi

  if is_success_delete "${DELETE_PROFILE_NO_TOKEN}"; then
    DELETE_NO_TOKEN_DETAIL="PASS trusted/private bypass"
  else
    EXPECT_FAIL=1
    DELETE_NO_TOKEN_DETAIL="FAIL expected 200/204 with trusted/private bypass"
  fi

  if is_denied "${POST_CERT_IMPORT_NO_TOKEN}"; then
    EXPECT_FAIL=1
    CERT_IMPORT_NO_TOKEN_DETAIL="FAIL expected non-auth result with trusted/private bypass"
  else
    CERT_IMPORT_NO_TOKEN_DETAIL="PASS trusted/private bypass"
  fi
else
  if is_denied "${PUT_PROFILE_NO_TOKEN}"; then
    PUT_NO_TOKEN_DETAIL="PASS auth denied"
  else
    EXPECT_FAIL=1
    PUT_NO_TOKEN_DETAIL="FAIL expected 401/403 auth denied"
  fi

  if is_denied "${DELETE_PROFILE_NO_TOKEN}"; then
    DELETE_NO_TOKEN_DETAIL="PASS auth denied"
  else
    EXPECT_FAIL=1
    DELETE_NO_TOKEN_DETAIL="FAIL expected 401/403 auth denied"
  fi

  if is_denied "${POST_CERT_IMPORT_NO_TOKEN}"; then
    CERT_IMPORT_NO_TOKEN_DETAIL="PASS auth denied"
  else
    EXPECT_FAIL=1
    CERT_IMPORT_NO_TOKEN_DETAIL="FAIL expected 401/403 auth denied"
  fi
fi

if [[ -z "${API_TOKEN}" ]]; then
  PUT_WITH_TOKEN_DETAIL="SKIP token not provided"
  DELETE_WITH_TOKEN_DETAIL="SKIP token not provided"
  CERT_IMPORT_WITH_TOKEN_DETAIL="SKIP token not provided"
else
  if [[ "${PUT_PROFILE_WITH_TOKEN}" == "200" ]]; then
    PUT_WITH_TOKEN_DETAIL="PASS strict success"
  else
    EXPECT_FAIL=1
    PUT_WITH_TOKEN_DETAIL="FAIL expected 200 strict success"
  fi

  if is_success_delete "${DELETE_PROFILE_WITH_TOKEN}"; then
    DELETE_WITH_TOKEN_DETAIL="PASS strict success"
  else
    EXPECT_FAIL=1
    DELETE_WITH_TOKEN_DETAIL="FAIL expected 200/204 strict success"
  fi

  if is_denied "${POST_CERT_IMPORT_WITH_TOKEN}"; then
    EXPECT_FAIL=1
    CERT_IMPORT_WITH_TOKEN_DETAIL="FAIL auth denied with bearer token"
  elif [[ "${POST_CERT_IMPORT_WITH_TOKEN}" == "200" ]]; then
    CERT_IMPORT_WITH_TOKEN_DETAIL="PASS auth passed + payload accepted"
  else
    CERT_IMPORT_WITH_TOKEN_DETAIL="PASS auth passed + payload validation failed"
  fi
fi

echo "GET /healthz (no token) -> ${GET_HEALTHZ} (${HEALTHZ_DETAIL})"
echo "GET /api/status (no token) -> ${GET_STATUS} (${STATUS_DETAIL})"
echo "PUT /api/cabinet-profile (no token) -> ${PUT_PROFILE_NO_TOKEN} (${PUT_NO_TOKEN_DETAIL})"
echo "DELETE /api/cabinet-profile (no token) -> ${DELETE_PROFILE_NO_TOKEN} (${DELETE_NO_TOKEN_DETAIL})"
echo "POST /api/certificates/import (no token) -> ${POST_CERT_IMPORT_NO_TOKEN} (${CERT_IMPORT_NO_TOKEN_DETAIL})"
echo "auth_required_by_runtime -> ${AUTH_REQUIRED}"
echo "trusted_mutation_bypass_active -> ${RUNTIME_TRUSTED_BYPASS}"
echo "PUT /api/cabinet-profile (with token) -> ${PUT_PROFILE_WITH_TOKEN} (${PUT_WITH_TOKEN_DETAIL})"
echo "DELETE /api/cabinet-profile (with token) -> ${DELETE_PROFILE_WITH_TOKEN} (${DELETE_WITH_TOKEN_DETAIL})"
echo "POST /api/certificates/import (with token) -> ${POST_CERT_IMPORT_WITH_TOKEN} (${CERT_IMPORT_WITH_TOKEN_DETAIL})"

if [[ "${EXPECT_FAIL}" -eq 0 ]]; then
  echo "overall -> PASS"
else
  echo "overall -> FAIL"
  exit 1
fi
