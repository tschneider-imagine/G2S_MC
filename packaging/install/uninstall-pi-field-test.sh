#!/usr/bin/env bash
set -euo pipefail

BIN_PATH="${BIN_PATH:-/usr/local/bin/g2s-mute}"
SERVICE_PATH="${SERVICE_PATH:-/etc/systemd/system/g2s-mute.service}"
CONFIG_DIR="${CONFIG_DIR:-/etc/g2s-mute}"
DATA_DIR="${DATA_DIR:-/var/lib/g2s-mute}"
LOG_DIR="${LOG_DIR:-/var/log/g2s-mute}"

REMOVE_CONFIG=0
PURGE_DATA=0
DRY_RUN=0

usage() {
  cat <<'EOF'
Usage: sudo ./packaging/install/uninstall-pi-field-test.sh [--remove-config] [--purge-data] [--dry-run]

Removes the appliance service install layout.

Options:
  --remove-config  remove /etc/g2s-mute
  --purge-data     remove /var/lib/g2s-mute and /var/log/g2s-mute
  --dry-run        print actions without changing the system
  -h, --help       show this help text
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --remove-config)
      REMOVE_CONFIG=1
      ;;
    --purge-data)
      PURGE_DATA=1
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

run_cmd systemctl disable --now g2s-mute.service || true

if [[ -f "${SERVICE_PATH}" ]]; then
  run_cmd rm -f "${SERVICE_PATH}"
fi
run_cmd systemctl daemon-reload

if [[ -f "${BIN_PATH}" ]]; then
  run_cmd rm -f "${BIN_PATH}"
fi

if [[ "${REMOVE_CONFIG}" -eq 1 && -d "${CONFIG_DIR}" ]]; then
  run_cmd rm -rf "${CONFIG_DIR}"
fi

if [[ "${PURGE_DATA}" -eq 1 ]]; then
  if [[ -d "${DATA_DIR}" ]]; then
    run_cmd rm -rf "${DATA_DIR}"
  fi
  if [[ -d "${LOG_DIR}" ]]; then
    run_cmd rm -rf "${LOG_DIR}"
  fi
fi

echo "uninstall complete"
