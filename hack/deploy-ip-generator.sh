#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

ENV_FILE="${IP_GENERATOR_DEPLOY_ENV_FILE:-${REPO_ROOT}/.env}"
provided_remote_host="${REMOTE_HOST:-}"
provided_remote_user="${REMOTE_USER:-}"
provided_remote_path="${IP_GENERATOR_REMOTE_PATH:-}"

if [[ -f "${ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +a
fi

if [[ -n "${provided_remote_host}" ]]; then
  REMOTE_HOST="${provided_remote_host}"
fi
if [[ -n "${provided_remote_user}" ]]; then
  REMOTE_USER="${provided_remote_user}"
fi
if [[ -n "${provided_remote_path}" ]]; then
  IP_GENERATOR_REMOTE_PATH="${provided_remote_path}"
fi

REMOTE_HOST="${REMOTE_HOST:-}"
REMOTE_USER="${REMOTE_USER:-root}"
IP_GENERATOR_REMOTE_PATH="${IP_GENERATOR_REMOTE_PATH:-/root/ip-generator}"
LOCAL_BINARY="${REPO_ROOT}/dist/ip-generator"

if [[ -z "${REMOTE_HOST}" ]]; then
  echo "REMOTE_HOST is required. Set it in ${ENV_FILE} or export it before running this script." >&2
  exit 1
fi

for required_command in go scp; do
  if ! command -v "${required_command}" >/dev/null 2>&1; then
    echo "Required command not found: ${required_command}" >&2
    exit 1
  fi
done

mkdir -p "$(dirname "${LOCAL_BINARY}")"
cd "${REPO_ROOT}"

echo "Building IP Generator for linux/amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -trimpath \
  -ldflags="-s -w" \
  -o "${LOCAL_BINARY}" \
  ./tools/ip-generator

if [[ ! -x "${LOCAL_BINARY}" ]]; then
  echo "Build output not found or not executable: ${LOCAL_BINARY}" >&2
  exit 1
fi

remote="${REMOTE_USER}@${REMOTE_HOST}"
echo "Uploading ${LOCAL_BINARY} to ${remote}:${IP_GENERATOR_REMOTE_PATH}..."
scp -C "${LOCAL_BINARY}" "${remote}:${IP_GENERATOR_REMOTE_PATH}"

echo "IP Generator uploaded successfully."
echo "Local: ${LOCAL_BINARY}"
echo "Remote: ${remote}:${IP_GENERATOR_REMOTE_PATH}"
