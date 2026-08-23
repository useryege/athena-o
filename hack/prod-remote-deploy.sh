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
MIGRATE_MODULE="${PROD_MIGRATE_MODULE:-all}"
ACTION="${1:-deploy}"

if [[ "${ACTION}" != "deploy" && "${ACTION}" != "hot-deploy" && "${ACTION}" != "destroy" ]]; then
  echo "Usage: $0 deploy|hot-deploy|destroy"
  exit 1
fi

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

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Environment file not found: ${ENV_FILE}"
  exit 1
fi

REMOTE="${REMOTE_USER}@${REMOTE_HOST}"

destroy_remote() {
  echo "Removing Athena containers, network, and PostgreSQL volume from ${REMOTE}..."
  ssh "${REMOTE}" "set -e
docker --version >/dev/null
docker compose version >/dev/null
if [ -f '${REMOTE_APP_DIR}/docker-compose.prod.yml' ]; then
  cd '${REMOTE_APP_DIR}'
  if [ -f .env ]; then
    compose_env_file=.env
  else
    compose_env_file=/dev/null
  fi
  POSTGRES_PASSWORD=dummy REDIS_PASSWORD=dummy ATHENA_WALLET_ENCRYPTION_KEY=dummy PROD_IMAGE='${IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' docker compose -f docker-compose.prod.yml --env-file \"\${compose_env_file}\" down --remove-orphans
fi
docker volume rm '${POSTGRES_VOLUME}' >/dev/null 2>&1 || true"
}

if [[ "${ACTION}" == "destroy" ]]; then
  destroy_remote
  echo "Remote Athena runtime resources removed. Deployment files and image were preserved."
  exit 0
fi

if [[ ! -f "${COMPOSE_FILE}" ]]; then
  echo "Compose file not found: ${COMPOSE_FILE}"
  exit 1
fi

if ! docker image inspect "${IMAGE}" >/dev/null 2>&1; then
  echo "Docker image not found locally: ${IMAGE}"
  exit 1
fi

UPLOAD_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${UPLOAD_DIR}"
}
trap cleanup EXIT

echo "Preparing deployment archive..."
cp "${COMPOSE_FILE}" "${UPLOAD_DIR}/docker-compose.prod.yml"
cp "${ENV_FILE}" "${UPLOAD_DIR}/.env"

if [[ "${ACTION}" == "hot-deploy" ]]; then
  echo "Checking remote host and PostgreSQL volume on ${REMOTE}..."
  ssh "${REMOTE}" "set -e
docker --version >/dev/null
docker compose version >/dev/null
if ! docker volume inspect '${POSTGRES_VOLUME}' >/dev/null 2>&1; then
  echo 'PostgreSQL volume not found: ${POSTGRES_VOLUME}. Run a full deployment first.'
  exit 1
fi
mkdir -p '${REMOTE_APP_DIR}'"

  echo "Uploading compose file and environment file..."
  tar -C "${UPLOAD_DIR}" -cf - docker-compose.prod.yml .env | ssh "${REMOTE}" "set -e
tar -C '${REMOTE_APP_DIR}' -xf -"

  echo "Streaming Docker image ${IMAGE} to ${REMOTE}..."
  docker save "${IMAGE}" | ssh "${REMOTE}" "docker load"

  echo "Ensuring the Profit Sharing database exists on ${REMOTE}..."
  ssh "${REMOTE}" "set -e
cd '${REMOTE_APP_DIR}'
compose() {
  PROD_IMAGE='${IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env \"\$@\"
}
compose up -d postgres
postgres_ready=false
for attempt in \$(seq 1 60); do
  if compose exec -T postgres sh -c 'pg_isready --username \"\$POSTGRES_USER\" --dbname \"\$POSTGRES_DB\"' >/dev/null 2>&1; then
    postgres_ready=true
    break
  fi
  sleep 1
done
if [ \"\${postgres_ready}\" != 'true' ]; then
  echo 'PostgreSQL did not become ready before Profit Sharing database creation.'
  exit 1
fi
if ! compose exec -T postgres sh -c 'createdb --username \"\$POSTGRES_USER\" profit_sharing' >/dev/null 2>&1; then
  compose exec -T postgres sh -c 'psql --username \"\$POSTGRES_USER\" --dbname profit_sharing --command \"SELECT 1\"' >/dev/null
fi"

  echo "Running Athena migrations on ${REMOTE}..."
  ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env --profile tools run --rm athena-migrate athena up --module '${MIGRATE_MODULE}'"

  echo "Recreating Athena backend services on ${REMOTE}..."
  ssh "${REMOTE}" "set -e
cd '${REMOTE_APP_DIR}'
compose() {
  PROD_IMAGE='${IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env \"\$@\"
}
athena_services=\"\$(compose config --services | awk '/^athena-/ && \$0 != \"athena-migrate\" && \$0 != \"athena-server\" { print }')\"
if [ -z \"\${athena_services}\" ]; then
  echo 'No Athena backend services found in docker-compose.prod.yml.'
  exit 1
fi
compose up -d --no-deps --force-recreate \${athena_services}
compose up -d --no-deps --force-recreate athena-server
compose ps"

  echo "Remote hot deployment completed. PostgreSQL and Redis were preserved."
  exit 0
fi

mkdir -p "${UPLOAD_DIR}/hack/postgres"
cp -a "${REPO_ROOT}/hack/postgres/init" "${UPLOAD_DIR}/hack/postgres/init"

echo "Preparing remote host ${REMOTE}..."
destroy_remote
ssh "${REMOTE}" "set -e
docker --version >/dev/null
docker compose version >/dev/null
mkdir -p '${REMOTE_APP_DIR}'
docker volume create '${POSTGRES_VOLUME}' >/dev/null"

echo "Uploading compose file, environment file, and PostgreSQL init scripts..."
tar -C "${UPLOAD_DIR}" -cf - docker-compose.prod.yml .env hack | ssh "${REMOTE}" "set -e
mkdir -p '${REMOTE_APP_DIR}'
tar -C '${REMOTE_APP_DIR}' -xf -"

echo "Streaming Docker image ${IMAGE} to ${REMOTE}..."
docker save "${IMAGE}" | ssh "${REMOTE}" "docker load"

echo "Starting PostgreSQL on ${REMOTE}..."
ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env up -d postgres"

echo "Running Athena migrations on ${REMOTE}..."
ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env --profile tools run --rm athena-migrate athena up --module '${MIGRATE_MODULE}'"

echo "Starting Athena on ${REMOTE}..."
ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env up -d"

echo "Remote deployment status:"
ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env ps"
