#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

ENV_FILE="${PROD_ENV_FILE:-${REPO_ROOT}/.env}"
COMPOSE_FILE="${PROD_COMPOSE_FILE:-${REPO_ROOT}/docker-compose.prod.yml}"
IMAGE="${PROD_IMAGE:-athena:local}"
CLEAR_DATA="${PROD_CLEAR_DATA:-false}"
REMOTE_USER="${REMOTE_USER:-root}"
REMOTE_APP_DIR="${REMOTE_APP_DIR:-/root/athena}"

if [[ -f "${ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +a
fi

REMOTE_HOST="${REMOTE_HOST:-}"
if [[ -z "${REMOTE_HOST}" ]]; then
  echo "REMOTE_HOST is required. Set it in ${ENV_FILE} or export it before running this script."
  exit 1
fi

if [[ ! -f "${COMPOSE_FILE}" ]]; then
  echo "Compose file not found: ${COMPOSE_FILE}"
  exit 1
fi

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Environment file not found: ${ENV_FILE}"
  exit 1
fi

if ! docker image inspect "${IMAGE}" >/dev/null 2>&1; then
  echo "Docker image not found locally: ${IMAGE}"
  echo "Build it first, for example: make prod-build"
  exit 1
fi

REMOTE="${REMOTE_USER}@${REMOTE_HOST}"

echo "Checking Docker on ${REMOTE}..."
ssh "${REMOTE}" "docker --version >/dev/null && docker compose version >/dev/null"

echo "Creating remote deployment directory: ${REMOTE_APP_DIR}"
ssh "${REMOTE}" "mkdir -p '${REMOTE_APP_DIR}/hack/postgres'"

echo "Uploading compose file and environment file..."
scp "${COMPOSE_FILE}" "${REMOTE}:${REMOTE_APP_DIR}/docker-compose.prod.yml"
scp "${ENV_FILE}" "${REMOTE}:${REMOTE_APP_DIR}/.env"

echo "Uploading PostgreSQL init scripts..."
scp -r "${REPO_ROOT}/hack/postgres/init" "${REMOTE}:${REMOTE_APP_DIR}/hack/postgres/"

echo "Streaming Docker image ${IMAGE} to ${REMOTE}..."
docker save "${IMAGE}" | ssh "${REMOTE}" "docker load"

if [[ "${CLEAR_DATA}" == "true" ]]; then
  echo "PROD_CLEAR_DATA=true: stopping remote stack and removing compose volumes..."
  ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && docker compose -f docker-compose.prod.yml --env-file .env down --volumes"
fi

echo "Starting Athena on ${REMOTE}..."
ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && docker compose -f docker-compose.prod.yml --env-file .env up -d"

echo "Remote deployment status:"
ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && docker compose -f docker-compose.prod.yml --env-file .env ps"
