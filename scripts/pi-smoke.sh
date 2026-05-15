#!/usr/bin/env bash
set -euo pipefail

CONFIG_PATH="${CONFIG_PATH:-/etc/g2s-mute/config.json}"
HOST_URL="${HOST_URL:-http://127.0.0.1:8444/g2s}"
EGM_ID="${EGM_ID:-EGM-01}"

if ! command -v g2s-mute >/dev/null 2>&1; then
  echo "g2s-mute binary is not installed" >&2
  exit 1
fi

if ! command -v g2s-fake-egm >/dev/null 2>&1; then
  echo "g2s-fake-egm binary is not installed" >&2
  exit 1
fi

if [[ ! -f "${CONFIG_PATH}" ]]; then
  echo "config not found: ${CONFIG_PATH}" >&2
  exit 1
fi

cleanup() {
  if [[ -n "${SERVER_PID:-}" ]] && kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

g2s-mute -config "${CONFIG_PATH}" -simulate-trigger &
SERVER_PID="$!"

for _ in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:8444/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done

curl -fsS "http://127.0.0.1:8444/healthz" >/dev/null
g2s-fake-egm -host-url "${HOST_URL}" -egm-id "${EGM_ID}" -keepalive-count 1 -keepalive-interval 0s

echo "Pi smoke test passed"
