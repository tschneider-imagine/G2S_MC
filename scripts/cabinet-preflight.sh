#!/usr/bin/env bash
set -euo pipefail

PREFLIGHT_URL="${PREFLIGHT_URL:-http://127.0.0.1:8444/api/cabinet-preflight}"

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required for cabinet preflight" >&2
  exit 2
fi

payload="$(curl -fsS "${PREFLIGHT_URL}")"
overall="$(printf '%s' "${payload}" | sed -n 's/.*"overall":"\([^"]*\)".*/\1/p')"

if [[ -z "${overall}" ]]; then
  echo "Cabinet preflight: FAIL"
  echo " - unable to parse preflight response"
  exit 2
fi

echo "Cabinet preflight: ${overall}"

if [[ "${overall}" == "PASS" ]]; then
  echo " - blockers: none"
  exit 0
fi

blockers_blob="$(printf '%s' "${payload}" | sed -n 's/.*"blockers":\[\(.*\)\],"timestamp".*/\1/p')"
if [[ -n "${blockers_blob}" ]]; then
  printf '%s\n' "${blockers_blob}" \
    | sed 's/","/\n/g' \
    | sed 's/^"//; s/"$//; s/\\"/"/g' \
    | while IFS= read -r blocker; do
        if [[ -n "${blocker}" ]]; then
          echo " - ${blocker}"
        fi
      done
else
  echo " - blockers: none reported"
fi

exit 1
