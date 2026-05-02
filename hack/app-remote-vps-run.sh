#!/usr/bin/env bash
set -euo pipefail

REMOTE_USER="root"
REMOTE_HOST="46.225.214.200"

REMOTE_TMP_PATH="/root/athena.new"
REMOTE_APPLICATION_PATH="/root/athena-application"
REMOTE_LOG="/root/athena.log"
REMOTE_PID_FILE="/root/athena.pid"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/../.env"

if [[ -f "${ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +a
fi

NODE_WS_URL="${ATHENA_APPLICATION_NODE_WS_URL:-}"
if [[ -z "${NODE_WS_URL}" ]]; then
  echo "ATHENA_APPLICATION_NODE_WS_URL is required (set it in .env)"
  exit 1
fi

echo "Starting athena on remote VPS..."
START_RESULT="$(ssh "${REMOTE_USER}@${REMOTE_HOST}" "
  set -e

  mv \"${REMOTE_TMP_PATH}\" \"${REMOTE_APPLICATION_PATH}\" || true
  chmod +x \"${REMOTE_APPLICATION_PATH}\" || true

  export ATHENA_APPLICATION_NODE_WS_URL=\"${NODE_WS_URL}\"

  : > \"${REMOTE_LOG}\"
  nohup \"${REMOTE_APPLICATION_PATH}\" > \"${REMOTE_LOG}\" 2>&1 &
  pid=\$!
  echo \"\$pid\" > \"${REMOTE_PID_FILE}\"

  echo 'athena application started'
  echo 'log: ${REMOTE_LOG}'
  echo \"ATHENA_PID=\$pid\"
")"

echo "${START_RESULT}"

STARTED_PID="$(printf '%s\n' "${START_RESULT}" | awk -F= '/^ATHENA_PID=/{print $2}')"
if [[ -z "${STARTED_PID}" ]]; then
  echo "Failed to parse remote athena pid"
  exit 1
fi

cleanup() {
  echo
  echo "Stopping remote athena process (pid=${STARTED_PID})..."
  ssh "${REMOTE_USER}@${REMOTE_HOST}" "
    if kill -0 \"${STARTED_PID}\" 2>/dev/null; then
      kill \"${STARTED_PID}\"
      wait \"${STARTED_PID}\" 2>/dev/null || true
      echo \"stopped pid ${STARTED_PID}\"
    else
      echo \"pid ${STARTED_PID} not running\"
    fi
    if [ -f \"${REMOTE_PID_FILE}\" ] && [ \"\$(cat \"${REMOTE_PID_FILE}\")\" = \"${STARTED_PID}\" ]; then
      rm -f \"${REMOTE_PID_FILE}\"
    fi
  "
}

trap cleanup INT TERM

echo "Watching realtime logs. Press Ctrl+C to exit log view."
ssh "${REMOTE_USER}@${REMOTE_HOST}" "tail -f \"${REMOTE_LOG}\""