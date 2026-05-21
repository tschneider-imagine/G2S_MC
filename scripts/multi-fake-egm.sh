#!/usr/bin/env bash
set -euo pipefail

CONFIG_PATH="${CONFIG_PATH:-configs/config.pi.example.json}"
HOST_URL="${HOST_URL:-http://127.0.0.1:8444/g2s}"
KEEPALIVE_COUNT="${KEEPALIVE_COUNT:--1}"
KEEPALIVE_INTERVAL="${KEEPALIVE_INTERVAL:-5s}"
STARTUP_DELAY="${STARTUP_DELAY:-0s}"
LAUNCH_SPACING_SECONDS="${LAUNCH_SPACING_SECONDS:-0.25}"
TIMEOUT="${TIMEOUT:-5s}"
CA_PATH="${CA_PATH:-}"
CERT_PATH="${CERT_PATH:-}"
KEY_PATH="${KEY_PATH:-}"
EGM_IDS="${EGM_IDS:-}"

if [[ ! -f "${CONFIG_PATH}" ]]; then
  echo "config not found: ${CONFIG_PATH}" >&2
  exit 1
fi

if command -v g2s-fake-egm >/dev/null 2>&1; then
  RUNNER=(g2s-fake-egm)
else
  RUNNER=(go run ./cmd/g2s-fake-egm)
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required to parse the EGM roster from ${CONFIG_PATH}" >&2
  exit 1
fi

mapfile -t ROSTER_EGM_IDS < <(
  python3 - "$CONFIG_PATH" "$EGM_IDS" <<'PY'
import json
import sys

config_path = sys.argv[1]
requested = [value.strip() for value in sys.argv[2].split(",") if value.strip()]

with open(config_path, "r", encoding="utf-8") as handle:
    payload = json.load(handle)

roster = []
for entry in payload.get("egm_roster", []):
    egm_id = str(entry.get("egm_id", "")).strip()
    if egm_id:
        roster.append(egm_id)

if requested:
    requested_set = set(requested)
    roster = [egm_id for egm_id in roster if egm_id in requested_set]

for egm_id in roster:
    print(egm_id)
PY
)

if [[ "${#ROSTER_EGM_IDS[@]}" -eq 0 ]]; then
  echo "no EGM IDs were resolved from ${CONFIG_PATH}" >&2
  exit 1
fi

cleanup() {
  for pid in "${PIDS[@]:-}"; do
    if [[ -n "${pid}" ]] && kill -0 "${pid}" >/dev/null 2>&1; then
      kill "${pid}" >/dev/null 2>&1 || true
    fi
  done
  wait >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

PIDS=()
echo "Launching ${#ROSTER_EGM_IDS[@]} fake EGM client(s) against ${HOST_URL}"
printf 'EGMs: %s\n' "${ROSTER_EGM_IDS[*]}"
printf 'keepalive_count=%s keepalive_interval=%s startup_delay=%s launch_spacing_seconds=%s\n' "${KEEPALIVE_COUNT}" "${KEEPALIVE_INTERVAL}" "${STARTUP_DELAY}" "${LAUNCH_SPACING_SECONDS}"

for index in "${!ROSTER_EGM_IDS[@]}"; do
  egm_id="${ROSTER_EGM_IDS[$index]}"
  args=(
    -host-url "${HOST_URL}"
    -egm-id "${egm_id}"
    -keepalive-count "${KEEPALIVE_COUNT}"
    -keepalive-interval "${KEEPALIVE_INTERVAL}"
    -startup-delay "${STARTUP_DELAY}"
    -timeout "${TIMEOUT}"
  )
  if [[ -n "${CA_PATH}" ]]; then
    args+=(-ca "${CA_PATH}")
  fi
  if [[ -n "${CERT_PATH}" || -n "${KEY_PATH}" ]]; then
    args+=(-cert "${CERT_PATH}" -key "${KEY_PATH}")
  fi
  echo "starting fake EGM ${egm_id}"
  "${RUNNER[@]}" "${args[@]}" &
  PIDS+=("$!")
  if [[ "$index" -lt $((${#ROSTER_EGM_IDS[@]} - 1)) ]]; then
    sleep "${LAUNCH_SPACING_SECONDS}"
  fi
done

wait
