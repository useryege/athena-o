#!/usr/bin/env bash
set -euo pipefail

LOCAL_BIN="dist/athena"
REMOTE_USER="root"
REMOTE_HOST="46.225.214.200"
REMOTE_TMP_PATH="/root/athena.new"

if [[ ! -f "${LOCAL_BIN}" ]]; then
  echo "Local binary not found: ${LOCAL_BIN}"
  echo "Build it first, for example: make build"
  exit 1
fi

echo "Uploading ${LOCAL_BIN} to ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_TMP_PATH}"
scp "${LOCAL_BIN}" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_TMP_PATH}"
echo "Upload finished."
