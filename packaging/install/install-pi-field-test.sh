#!/usr/bin/env bash
set -euo pipefail

APP_USER="${APP_USER:-g2s-mute}"
APP_GROUP="${APP_GROUP:-g2s-mute}"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-/etc/g2s-mute}"
DATA_DIR="${DATA_DIR:-/var/lib/g2s-mute}"
LOG_DIR="${LOG_DIR:-/var/log/g2s-mute}"
SERVICE_PATH="${SERVICE_PATH:-/etc/systemd/system/g2s-mute.service}"

ENABLE_SERVICE=0
START_SERVICE=0
DRY_RUN=0

usage() {
  cat <<'EOF'
Usage: sudo ./packaging/install/install-pi-field-test.sh [--enable] [--start] [--dry-run]

Installs the appliance service runtime layout on a Raspberry Pi:
- /usr/local/bin/g2s-mute
- /etc/g2s-mute/config.json
- /var/lib/g2s-mute/controller.db
- /etc/systemd/system/g2s-mute.service

Options:
  --enable    enable g2s-mute.service at boot
  --start     restart g2s-mute.service after install
  --dry-run   print actions without changing the system
  -h, --help  show this help text
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --enable)
      ENABLE_SERVICE=1
      ;;
    --start)
      START_SERVICE=1
      ;;
    --dry-run)
      DRY_RUN=1
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

run_cmd() {
  if [[ "${DRY_RUN}" -eq 1 ]]; then
    echo "DRY-RUN: $*"
    return 0
  fi
  "$@"
}

if [[ "${DRY_RUN}" -ne 1 && "${EUID}" -ne 0 ]]; then
  echo "please run with sudo (or use --dry-run)" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go is required before install" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
BUILD_DIR="$(mktemp -d)"
trap 'rm -rf "${BUILD_DIR}"' EXIT

cd "${REPO_ROOT}"
go test ./...
go build -trimpath -o "${BUILD_DIR}/g2s-mute" ./cmd/g2s-mute

if ! getent group "${APP_GROUP}" >/dev/null 2>&1; then
  run_cmd groupadd --system "${APP_GROUP}"
fi

if ! id -u "${APP_USER}" >/dev/null 2>&1; then
  run_cmd useradd --system --gid "${APP_GROUP}" --home-dir "${DATA_DIR}" --shell /usr/sbin/nologin "${APP_USER}"
fi

if getent group gpio >/dev/null 2>&1; then
  run_cmd usermod -a -G gpio "${APP_USER}"
fi

run_cmd install -d -m 0755 "${BIN_DIR}"
run_cmd install -d -m 0750 -o "${APP_USER}" -g "${APP_GROUP}" "${CONFIG_DIR}" "${DATA_DIR}" "${LOG_DIR}" "${CONFIG_DIR}/certs"

run_cmd install -m 0755 "${BUILD_DIR}/g2s-mute" "${BIN_DIR}/g2s-mute"

if [[ ! -f "${CONFIG_DIR}/config.json" ]]; then
  run_cmd install -m 0640 -o "${APP_USER}" -g "${APP_GROUP}" "${REPO_ROOT}/configs/config.pi-field-test.example.json" "${CONFIG_DIR}/config.json"
  echo "installed config template at ${CONFIG_DIR}/config.json"
else
  echo "kept existing config at ${CONFIG_DIR}/config.json"
  if ! grep -q '"runtime"' "${CONFIG_DIR}/config.json"; then
    echo "note: existing config does not include a runtime section; runtime behavior may rely on command-line defaults"
  fi
fi

run_cmd install -m 0644 "${REPO_ROOT}/packaging/systemd/g2s-mute.service" "${SERVICE_PATH}"
run_cmd systemctl daemon-reload

if [[ "${ENABLE_SERVICE}" -eq 1 ]]; then
  run_cmd systemctl enable g2s-mute.service
else
  echo "service not enabled (use --enable to enable at boot)"
fi

if [[ "${START_SERVICE}" -eq 1 ]]; then
  run_cmd systemctl restart g2s-mute.service
  run_cmd systemctl --no-pager --full status g2s-mute.service
else
  echo "service not started (use --start to restart now)"
fi

echo "install complete"
