#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

ENV_FILE="${PROD_ENV_FILE:-${REPO_ROOT}/.env}"
COMPOSE_FILE="${PROD_COMPOSE_FILE:-${REPO_ROOT}/docker-compose.prod.yml}"
IMAGE="${PROD_IMAGE:-athena:local}"
REMOTE_USER="${REMOTE_USER:-root}"
REMOTE_APP_DIR="${REMOTE_APP_DIR:-/root/athena}"
POSTGRES_VOLUME="${PROD_POSTGRES_VOLUME:-athena-prod-postgres-data}"
RESET_REMOTE_DATA="${PROD_RESET_REMOTE_DATA:-}"
RUN_REMOTE_MIGRATIONS="${PROD_RUN_REMOTE_MIGRATIONS:-}"
MIGRATE_MODULE="${PROD_MIGRATE_MODULE:-all}"

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
  echo "Build it first, for example: make prod-build-local"
  exit 1
fi

REMOTE="${REMOTE_USER}@${REMOTE_HOST}"
UPLOAD_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${UPLOAD_DIR}"
}
trap cleanup EXIT

echo "Preparing remote host ${REMOTE}..."
if [[ "${RESET_REMOTE_DATA}" == "yes" ]]; then
  echo "Resetting Athena containers and PostgreSQL volume on ${REMOTE}..."
  ssh "${REMOTE}" "set -e
docker --version >/dev/null
docker compose version >/dev/null
mkdir -p '${REMOTE_APP_DIR}'
if [ -f '${REMOTE_APP_DIR}/docker-compose.prod.yml' ]; then
  cd '${REMOTE_APP_DIR}'
  if [ -f .env ]; then
    compose_env_file=.env
  else
    compose_env_file=/dev/null
  fi
  POSTGRES_PASSWORD=dummy REDIS_PASSWORD=dummy ATHENA_WALLET_ENCRYPTION_KEY=dummy PROD_IMAGE='${IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' docker compose -f docker-compose.prod.yml --env-file \"\${compose_env_file}\" down --remove-orphans
fi
docker volume rm '${POSTGRES_VOLUME}' >/dev/null 2>&1 || true
docker volume create '${POSTGRES_VOLUME}' >/dev/null"
else
  ssh "${REMOTE}" "set -e
docker --version >/dev/null
docker compose version >/dev/null
mkdir -p '${REMOTE_APP_DIR}'
docker volume create '${POSTGRES_VOLUME}' >/dev/null"
fi

echo "Preparing deployment archive..."
cp "${COMPOSE_FILE}" "${UPLOAD_DIR}/docker-compose.prod.yml"
cp "${ENV_FILE}" "${UPLOAD_DIR}/.env"
mkdir -p "${UPLOAD_DIR}/hack/postgres"
cp -a "${REPO_ROOT}/hack/postgres/init" "${UPLOAD_DIR}/hack/postgres/init"

echo "Uploading compose file, environment file, and PostgreSQL init scripts..."
tar -C "${UPLOAD_DIR}" -cf - docker-compose.prod.yml .env hack | ssh "${REMOTE}" "set -e
mkdir -p '${REMOTE_APP_DIR}'
tar -C '${REMOTE_APP_DIR}' -xf -"

echo "Streaming Docker image ${IMAGE} to ${REMOTE}..."
docker save "${IMAGE}" | ssh "${REMOTE}" "docker load"

if [[ "${RUN_REMOTE_MIGRATIONS}" == "yes" ]]; then
  echo "Starting PostgreSQL on ${REMOTE}..."
  ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env up -d postgres"

  echo "Running Athena migrations on ${REMOTE}..."
  ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env --profile tools run --rm athena-migrate athena up --module '${MIGRATE_MODULE}'"
fi

echo "Starting Athena on ${REMOTE}..."
ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env up -d"

echo "Remote deployment status:"
ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env ps"
