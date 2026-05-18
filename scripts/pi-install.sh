#!/usr/bin/env bash
set -euo pipefail

APP_USER="${APP_USER:-g2s-mute}"
APP_GROUP="${APP_GROUP:-g2s-mute}"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-/etc/g2s-mute}"
DATA_DIR="${DATA_DIR:-/var/lib/g2s-mute}"
LOG_DIR="${LOG_DIR:-/var/log/g2s-mute}"
SERVICE_PATH="${SERVICE_PATH:-/etc/systemd/system/g2s-mute.service}"
START_SERVICE=0

usage() {
  cat <<'EOF'
Usage: sudo bash ./scripts/pi-install.sh [--start]

Builds and installs the G2S Muting Controller on a Raspberry Pi:
- /usr/local/bin/g2s-mute
- /usr/local/bin/g2s-fake-egm
- /usr/local/bin/g2s-dev-certs
- /etc/g2s-mute/config.json
- /etc/systemd/system/g2s-mute.service
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --start)
      START_SERVICE=1
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

if [[ "${EUID}" -ne 0 ]]; then
  echo "please run with sudo" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go is required on the Pi before install" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"
BUILD_DIR="$(mktemp -d)"
trap 'rm -rf "${BUILD_DIR}"' EXIT

if ! getent group "${APP_GROUP}" >/dev/null 2>&1; then
  groupadd --system "${APP_GROUP}"
fi

if ! id -u "${APP_USER}" >/dev/null 2>&1; then
  useradd --system --gid "${APP_GROUP}" --home-dir "${DATA_DIR}" --shell /usr/sbin/nologin "${APP_USER}"
fi

install -d -m 0755 "${BIN_DIR}"
install -d -m 0750 -o "${APP_USER}" -g "${APP_GROUP}" "${CONFIG_DIR}" "${CONFIG_DIR}/certs" "${DATA_DIR}" "${LOG_DIR}"

go test ./...
go build -buildvcs=false -trimpath -o "${BUILD_DIR}/g2s-mute" ./cmd/g2s-mute
go build -buildvcs=false -trimpath -o "${BUILD_DIR}/g2s-fake-egm" ./cmd/g2s-fake-egm
go build -buildvcs=false -trimpath -o "${BUILD_DIR}/g2s-dev-certs" ./cmd/g2s-dev-certs

install -m 0755 "${BUILD_DIR}/g2s-mute" "${BIN_DIR}/g2s-mute"
install -m 0755 "${BUILD_DIR}/g2s-fake-egm" "${BIN_DIR}/g2s-fake-egm"
install -m 0755 "${BUILD_DIR}/g2s-dev-certs" "${BIN_DIR}/g2s-dev-certs"

if [[ ! -f "${CONFIG_DIR}/config.json" ]]; then
  install -m 0640 -o "${APP_USER}" -g "${APP_GROUP}" ./configs/config.pi.example.json "${CONFIG_DIR}/config.json"
  echo "installed starter config at ${CONFIG_DIR}/config.json"
else
  echo "kept existing config at ${CONFIG_DIR}/config.json"
fi

install -m 0644 ./packaging/systemd/g2s-mute.service "${SERVICE_PATH}"
systemctl daemon-reload
systemctl enable g2s-mute.service

if [[ "${START_SERVICE}" -eq 1 ]]; then
  systemctl restart g2s-mute.service
  systemctl --no-pager --full status g2s-mute.service
else
  echo "install complete; start with: sudo systemctl restart g2s-mute.service"
fi
