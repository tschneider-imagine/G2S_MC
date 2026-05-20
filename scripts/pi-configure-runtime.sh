#!/usr/bin/env bash
set -euo pipefail

CONFIG_PATH="${CONFIG_PATH:-/etc/g2s-mute/config.json}"
API_BASE="${API_BASE:-http://127.0.0.1:8444}"
API_TOKEN_FILE="${API_TOKEN_FILE:-${HOME}/.g2s_api_token}"
APP_USER="${APP_USER:-g2s-mute}"
APP_GROUP="${APP_GROUP:-g2s-mute}"
RESTART_SERVICE=1
RUN_RELEASE_GATE=1
PRINT_TOKEN=0

usage() {
  cat <<'EOF'
Usage: bash ./scripts/pi-configure-runtime.sh [options]

Safely configures the active Pi runtime config for first-cabinet lab work:
- creates or reuses an API bearer token
- writes non-placeholder cabinet_profile defaults into /etc/g2s-mute/config.json
- restarts g2s-mute.service
- clears stale cabinet profile overrides through the API
- runs scripts/release-gate.sh

Environment overrides:
  CONFIG_PATH              default /etc/g2s-mute/config.json
  API_BASE                 default http://127.0.0.1:8444
  API_TOKEN                explicit token; otherwise token file or random token is used
  API_TOKEN_FILE           default ~/.g2s_api_token
  WIRE_HOST_URL            default https://<hostname>.local:8444/g2s
  LISTENER_DNS_NAME        default <hostname>.local
  LISTENER_IP              default first address from hostname -I
  REQUIRED_SAN_DNS         default LISTENER_DNS_NAME
  REQUIRED_SAN_IPS         default LISTENER_IP
  HOST_ID                  default HOST-<HOSTNAME>-001
  FIRST_TEST_EGM_IDS       default EGM-001

Options:
  --no-restart             update config without restarting service
  --no-release-gate        skip release gate after configuration
  --print-token            print token value at the end
  -h, --help               show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-restart)
      RESTART_SERVICE=0
      ;;
    --no-release-gate)
      RUN_RELEASE_GATE=0
      ;;
    --print-token)
      PRINT_TOKEN=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
PATCHER="${SCRIPT_DIR}/pi-configure-runtime.py"

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "${name} is required" >&2
    exit 1
  fi
}

require_command sudo
require_command python3
require_command openssl
require_command curl

if [[ ! -f "${PATCHER}" ]]; then
  echo "missing helper: ${PATCHER}" >&2
  exit 1
fi

sudo -v

HOST_SHORT="$(hostname -s | tr '[:upper:]' '[:lower:]')"
HOST_UPPER="$(printf '%s' "${HOST_SHORT}" | tr '[:lower:]' '[:upper:]' | tr -cd 'A-Z0-9')"
DEFAULT_DNS="${HOST_SHORT}.local"
DEFAULT_IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
DEFAULT_IP="${DEFAULT_IP:-127.0.0.1}"

LISTENER_DNS_NAME="${LISTENER_DNS_NAME:-${DEFAULT_DNS}}"
LISTENER_IP="${LISTENER_IP:-${DEFAULT_IP}}"
WIRE_HOST_URL="${WIRE_HOST_URL:-https://${LISTENER_DNS_NAME}:8444/g2s}"
REQUIRED_SAN_DNS="${REQUIRED_SAN_DNS:-${LISTENER_DNS_NAME}}"
REQUIRED_SAN_IPS="${REQUIRED_SAN_IPS:-${LISTENER_IP}}"
HOST_ID="${HOST_ID:-HOST-${HOST_UPPER:-PI}-001}"
FIRST_TEST_EGM_IDS="${FIRST_TEST_EGM_IDS:-EGM-001}"

if [[ -n "${API_TOKEN:-}" ]]; then
  TOKEN="${API_TOKEN}"
elif [[ -f "${API_TOKEN_FILE}" ]]; then
  TOKEN="$(tr -d '\r\n' <"${API_TOKEN_FILE}")"
else
  TOKEN="$(openssl rand -hex 32)"
fi

if [[ -z "${TOKEN}" ]]; then
  echo "API token is empty" >&2
  exit 1
fi

install -m 0600 /dev/null "${API_TOKEN_FILE}"
printf '%s\n' "${TOKEN}" >"${API_TOKEN_FILE}"
chmod 600 "${API_TOKEN_FILE}"

TMP_CONFIG="$(mktemp)"
rm -f "${TMP_CONFIG}"
cleanup() {
  if [[ -e "${TMP_CONFIG}" ]]; then
    sudo rm -f "${TMP_CONFIG}" 2>/dev/null || rm -f "${TMP_CONFIG}"
  fi
}
trap cleanup EXIT

sudo env G2S_API_TOKEN="${TOKEN}" python3 "${PATCHER}" \
  --config "${CONFIG_PATH}" \
  --output "${TMP_CONFIG}" \
  --wire-host-url "${WIRE_HOST_URL}" \
  --listener-dns-name "${LISTENER_DNS_NAME}" \
  --listener-ip "${LISTENER_IP}" \
  --required-san-dns "${REQUIRED_SAN_DNS}" \
  --required-san-ips "${REQUIRED_SAN_IPS}" \
  --host-id "${HOST_ID}" \
  --first-test-egm-ids "${FIRST_TEST_EGM_IDS}"

sudo install -m 0640 -o "${APP_USER}" -g "${APP_GROUP}" "${TMP_CONFIG}" "${CONFIG_PATH}"

if [[ "${RESTART_SERVICE}" -eq 1 ]]; then
  sudo systemctl restart g2s-mute.service
  for _ in {1..20}; do
    if curl -fsS "${API_BASE%/}/healthz" >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done
fi

if curl -fsS "${API_BASE%/}/healthz" >/dev/null 2>&1; then
  curl -fsS -X DELETE "${API_BASE%/}/api/cabinet-profile" \
    -H "Authorization: Bearer ${TOKEN}" >/dev/null || true
fi

echo "configured ${CONFIG_PATH}"
echo "token file: ${API_TOKEN_FILE}"
echo "wire_host_url: ${WIRE_HOST_URL}"
echo "listener_dns_name: ${LISTENER_DNS_NAME}"
echo "listener_ip: ${LISTENER_IP}"
echo "host_id: ${HOST_ID}"
echo "first_test_egm_ids: ${FIRST_TEST_EGM_IDS}"

if [[ "${RUN_RELEASE_GATE}" -eq 1 ]]; then
  API_BASE="${API_BASE}" API_TOKEN="${TOKEN}" bash "${REPO_ROOT}/scripts/release-gate.sh"
fi

if [[ "${PRINT_TOKEN}" -eq 1 ]]; then
  echo "api_token: ${TOKEN}"
fi
