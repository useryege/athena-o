#!/usr/bin/env bash
set -euo pipefail

LOCAL_BIN="dist/athena"
REMOTE_USER="root"
REMOTE_TMP_PATH="/root/athena.new"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/../.env"

if [[ -f "${ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +a
fi

REMOTE_HOST="${REMOTE_HOST:-}"
if [[ -z "${REMOTE_HOST}" ]]; then
  echo "REMOTE_HOST is required (set it in .env)"
  exit 1
fi

if [[ ! -f "${LOCAL_BIN}" ]]; then
  echo "Local binary not found: ${LOCAL_BIN}"
  echo "Build it first, for example: make build"
  exit 1
fi

echo "Uploading ${LOCAL_BIN} to ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_TMP_PATH}"
scp "${LOCAL_BIN}" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_TMP_PATH}"
echo "Upload finished."
