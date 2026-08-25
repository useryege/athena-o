#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

ENV_FILE="${PROD_ENV_FILE:-${REPO_ROOT}/.env}"
COMPOSE_FILE="${PROD_COMPOSE_FILE:-${REPO_ROOT}/docker-compose.prod.yml}"
IMAGE="${PROD_IMAGE:-athena:local}"
MINIO_IMAGE="${MINIO_IMAGE:-athena-minio:9e49d5e7a648}"
MINIO_MC_IMAGE="${MINIO_MC_IMAGE:-athena-minio-mc:7394ce0dd2a8}"
REMOTE_USER="${REMOTE_USER:-root}"
REMOTE_APP_DIR="${REMOTE_APP_DIR:-/root/athena}"
POSTGRES_VOLUME="${PROD_POSTGRES_VOLUME:-athena-prod-postgres-data}"
REDIS_VOLUME="${PROD_REDIS_VOLUME:-athena-prod-redis-data}"
MINIO_VOLUME="${PROD_MINIO_VOLUME:-athena-prod-minio-data}"
MIGRATE_MODULE="${PROD_MIGRATE_MODULE:-all}"
ACTION="${1:-deploy}"
GOOGLE_OIDC_SECRET_ARCHIVE_PATH="secrets/google-oidc-client-secret"
GOOGLE_OIDC_SECRET_SOURCE=""
ATHENA_CONTAINER_UID=999

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
  echo "Removing Athena containers, network, and persistent data volumes from ${REMOTE}..."
  ssh "${REMOTE}" "set -e
docker --version >/dev/null
docker compose version >/dev/null
if [ -f '${REMOTE_APP_DIR}/docker-compose.prod.yml' ]; then
  cd '${REMOTE_APP_DIR}'
  mkdir -p secrets
  if [ ! -f '${GOOGLE_OIDC_SECRET_ARCHIVE_PATH}' ]; then
    install -m 0600 /dev/null '${GOOGLE_OIDC_SECRET_ARCHIVE_PATH}'
  fi
  if [ -f .env ]; then
    compose_env_file=.env
  else
    compose_env_file=/dev/null
  fi
  ATHENA_COMPOSE_ENV_FILE=\"\${compose_env_file}\" ATHENA_TOKEN_ETH_ENABLED=dummy ATHENA_TOKEN_ETH_NODE_WS_URLS=dummy ATHENA_TOKEN_ETH_ATHENA_CONTRACT=dummy ATHENA_TOKEN_ETH_PROCESSOR_INITIAL_LOOKBACK_DURATION=dummy ATHENA_TOKEN_ETH_PROCESSOR_POLL_INTERVAL=dummy ATHENA_TOKEN_ETH_SWAP_POLL_INTERVAL=dummy ATHENA_TOKEN_BSC_ENABLED=dummy ATHENA_TOKEN_BSC_NODE_WS_URLS=dummy ATHENA_TOKEN_BSC_ATHENA_CONTRACT=dummy ATHENA_TOKEN_BSC_PROCESSOR_INITIAL_LOOKBACK_DURATION=dummy ATHENA_TOKEN_BSC_PROCESSOR_POLL_INTERVAL=dummy ATHENA_TOKEN_BSC_SWAP_POLL_INTERVAL=dummy POSTGRES_PASSWORD=dummy REDIS_PASSWORD=dummy ATHENA_JWT_SECRET=dummy-jwt-secret-for-compose-cleanup ATHENA_WALLET_ENCRYPTION_KEY=dummy MINIO_ROOT_PASSWORD=dummy-root-password ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID=dummy-access-key ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY=dummy-secret-key ATHENA_NOTIFICATION_TEST_TELEGRAM_CHAT_ID=dummy ATHENA_NOTIFICATION_PROD_TELEGRAM_CHAT_ID=dummy ATHENA_ETHERSCAN_MANAGER_API_KEYS=dummy ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS=dummy ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN=dummy PROD_IMAGE='${IMAGE}' MINIO_IMAGE='${MINIO_IMAGE}' MINIO_MC_IMAGE='${MINIO_MC_IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' PROD_REDIS_VOLUME='${REDIS_VOLUME}' PROD_MINIO_VOLUME='${MINIO_VOLUME}' docker compose -f docker-compose.prod.yml --env-file \"\${compose_env_file}\" down --remove-orphans
fi
for volume_name in '${POSTGRES_VOLUME}' '${REDIS_VOLUME}' '${MINIO_VOLUME}'; do
  if docker volume inspect \"\${volume_name}\" >/dev/null 2>&1; then
    docker volume rm \"\${volume_name}\" >/dev/null
  fi
done"
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

JWT_SECRET_VALUE="${ATHENA_JWT_SECRET:-}"
if (( ${#JWT_SECRET_VALUE} < 32 )); then
  echo "ATHENA_JWT_SECRET must contain at least 32 bytes."
  exit 1
fi

if [[ -n "${ATHENA_GOOGLE_OIDC_CLIENT_SECRET:-}" ]]; then
  echo "ATHENA_GOOGLE_OIDC_CLIENT_SECRET must be empty for production deployment; use ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE."
  exit 1
fi

if [[ "${ATHENA_SERVER_DISABLE_AUTH:-false}" == "true" ]]; then
  echo "ATHENA_SERVER_DISABLE_AUTH=true is not supported by production deployment."
  exit 1
fi

required_oidc_variables=(
  ATHENA_GOOGLE_OIDC_CLIENT_ID
  ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE
  ATHENA_GOOGLE_OIDC_REDIRECT_URI
  ATHENA_ACCOUNT_YEGE_GOOGLE_SUB
  ATHENA_ACCOUNT_LINGJIE_GOOGLE_SUB
  ATHENA_ACCOUNT_DONGMEI_GOOGLE_SUB
  ATHENA_ACCOUNT_DINGZHI_GOOGLE_SUB
  ATHENA_ACCOUNT_YUDIAN_GOOGLE_SUB
  ATHENA_ADMIN_GOOGLE_SUB
)
for variable_name in "${required_oidc_variables[@]}"; do
  if [[ -z "${!variable_name:-}" ]]; then
    echo "${variable_name} is required for production deployment."
    exit 1
  fi
done
if [[ ! "${ATHENA_GOOGLE_OIDC_REDIRECT_URI}" =~ ^https://[^/?#]+/auth/google/callback$ ]]; then
  echo "ATHENA_GOOGLE_OIDC_REDIRECT_URI must be an explicit HTTPS URI ending exactly in /auth/google/callback."
  exit 1
fi

if [[ "${ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE}" = /* ]]; then
  GOOGLE_OIDC_SECRET_SOURCE="${ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE}"
else
  GOOGLE_OIDC_SECRET_SOURCE="$(cd "$(dirname "${ENV_FILE}")" && pwd)/${ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE#./}"
fi
if [[ ! -s "${GOOGLE_OIDC_SECRET_SOURCE}" ]]; then
  echo "Google OIDC client secret file is missing or empty: ${GOOGLE_OIDC_SECRET_SOURCE}"
  exit 1
fi

declare -A configured_google_subjects=()
for variable_name in \
  ATHENA_ACCOUNT_YEGE_GOOGLE_SUB \
  ATHENA_ACCOUNT_LINGJIE_GOOGLE_SUB \
  ATHENA_ACCOUNT_DONGMEI_GOOGLE_SUB \
  ATHENA_ACCOUNT_DINGZHI_GOOGLE_SUB \
  ATHENA_ACCOUNT_YUDIAN_GOOGLE_SUB \
  ATHENA_ADMIN_GOOGLE_SUB; do
  google_subject="${!variable_name}"
  if [[ -n "${configured_google_subjects[${google_subject}]:-}" ]]; then
    echo "${variable_name} duplicates the Google sub configured by ${configured_google_subjects[${google_subject}]}."
    exit 1
  fi
  configured_google_subjects["${google_subject}"]="${variable_name}"
done

if ! docker image inspect "${IMAGE}" >/dev/null 2>&1; then
  echo "Docker image not found locally: ${IMAGE}"
  exit 1
fi
if ! docker image inspect "${MINIO_IMAGE}" >/dev/null 2>&1; then
  echo "Docker image not found locally: ${MINIO_IMAGE}"
  exit 1
fi
if ! docker image inspect "${MINIO_MC_IMAGE}" >/dev/null 2>&1; then
  echo "Docker image not found locally: ${MINIO_MC_IMAGE}"
  exit 1
fi

UPLOAD_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${UPLOAD_DIR}"
}
trap cleanup EXIT

echo "Preparing deployment archive..."
cp "${COMPOSE_FILE}" "${UPLOAD_DIR}/docker-compose.prod.yml"
awk -v value="ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE='./${GOOGLE_OIDC_SECRET_ARCHIVE_PATH}'" '
  BEGIN { replaced = 0 }
  /^ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE=/ {
    if (!replaced) {
      print value
      replaced = 1
    }
    next
  }
  { print }
  END {
    if (!replaced) {
      print value
    }
  }
' "${ENV_FILE}" >"${UPLOAD_DIR}/.env"
mkdir -p "${UPLOAD_DIR}/secrets"
if [[ -n "${GOOGLE_OIDC_SECRET_SOURCE}" ]]; then
  install -m 0600 "${GOOGLE_OIDC_SECRET_SOURCE}" "${UPLOAD_DIR}/${GOOGLE_OIDC_SECRET_ARCHIVE_PATH}"
else
  install -m 0600 /dev/null "${UPLOAD_DIR}/${GOOGLE_OIDC_SECRET_ARCHIVE_PATH}"
fi

if [[ "${ACTION}" == "hot-deploy" ]]; then
  echo "Checking remote host and persistent data volumes on ${REMOTE}..."
  ssh "${REMOTE}" "set -e
docker --version >/dev/null
docker compose version >/dev/null
if ! docker volume inspect '${POSTGRES_VOLUME}' >/dev/null 2>&1; then
  echo 'PostgreSQL volume not found: ${POSTGRES_VOLUME}. Run a full deployment first.'
  exit 1
fi
if ! docker volume inspect '${REDIS_VOLUME}' >/dev/null 2>&1; then
  echo 'Redis volume not found: ${REDIS_VOLUME}. Run a full deployment first.'
  exit 1
fi
if ! docker volume inspect '${MINIO_VOLUME}' >/dev/null 2>&1; then
  echo 'MinIO volume not found: ${MINIO_VOLUME}. Run a full deployment first.'
  exit 1
fi
mkdir -p '${REMOTE_APP_DIR}'"

  echo "Uploading compose file, environment file, and Google OIDC client secret..."
  tar -C "${UPLOAD_DIR}" -cf - docker-compose.prod.yml .env secrets | ssh "${REMOTE}" "set -e
tar -C '${REMOTE_APP_DIR}' -xf -
chown \"\$(id -u):\$(id -g)\" '${REMOTE_APP_DIR}/.env'
chmod 0600 '${REMOTE_APP_DIR}/.env'
chown '${ATHENA_CONTAINER_UID}:${ATHENA_CONTAINER_UID}' '${REMOTE_APP_DIR}/${GOOGLE_OIDC_SECRET_ARCHIVE_PATH}'
chmod 0600 '${REMOTE_APP_DIR}/${GOOGLE_OIDC_SECRET_ARCHIVE_PATH}'"

  echo "Streaming Docker image ${IMAGE} to ${REMOTE}..."
  docker save "${IMAGE}" | ssh "${REMOTE}" "docker load"
  echo "Streaming Docker image ${MINIO_IMAGE} to ${REMOTE}..."
  docker save "${MINIO_IMAGE}" | ssh "${REMOTE}" "docker load"
  echo "Streaming Docker image ${MINIO_MC_IMAGE} to ${REMOTE}..."
  docker save "${MINIO_MC_IMAGE}" | ssh "${REMOTE}" "docker load"

  echo "Ensuring the Profit Sharing database exists on ${REMOTE}..."
  ssh "${REMOTE}" "set -e
cd '${REMOTE_APP_DIR}'
compose() {
  PROD_IMAGE='${IMAGE}' MINIO_IMAGE='${MINIO_IMAGE}' MINIO_MC_IMAGE='${MINIO_MC_IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' PROD_REDIS_VOLUME='${REDIS_VOLUME}' PROD_MINIO_VOLUME='${MINIO_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env \"\$@\"
}
compose up -d postgres redis
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
redis_ready=false
for attempt in \$(seq 1 60); do
  if compose exec -T redis sh -c 'redis-cli -a \"\$REDIS_PASSWORD\" ping' 2>/dev/null | grep -qx PONG; then
    redis_ready=true
    break
  fi
  sleep 1
done
if [ \"\${redis_ready}\" != 'true' ]; then
  echo 'Redis did not become ready before Athena service recreation.'
  exit 1
fi
if ! compose exec -T postgres sh -c 'createdb --username \"\$POSTGRES_USER\" profit_sharing' >/dev/null 2>&1; then
  compose exec -T postgres sh -c 'psql --username \"\$POSTGRES_USER\" --dbname profit_sharing --command \"SELECT 1\"' >/dev/null
fi"

  echo "Starting and initializing MinIO on ${REMOTE}..."
  ssh "${REMOTE}" "set -e
cd '${REMOTE_APP_DIR}'
compose() {
  PROD_IMAGE='${IMAGE}' MINIO_IMAGE='${MINIO_IMAGE}' MINIO_MC_IMAGE='${MINIO_MC_IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' PROD_REDIS_VOLUME='${REDIS_VOLUME}' PROD_MINIO_VOLUME='${MINIO_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env \"\$@\"
}
compose up -d minio
minio_ready=false
for attempt in \$(seq 1 60); do
  if compose exec -T minio curl --fail --silent http://127.0.0.1:9000/minio/health/live >/dev/null 2>&1; then
    minio_ready=true
    break
  fi
  sleep 1
done
if [ \"\${minio_ready}\" != 'true' ]; then
  echo 'MinIO did not become ready before bucket initialization.'
  exit 1
fi
compose run --rm --no-deps minio-init"

  echo "Running Athena migrations on ${REMOTE}..."
  ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' MINIO_IMAGE='${MINIO_IMAGE}' MINIO_MC_IMAGE='${MINIO_MC_IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' PROD_REDIS_VOLUME='${REDIS_VOLUME}' PROD_MINIO_VOLUME='${MINIO_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env --profile tools run --rm athena-migrate athena up --module '${MIGRATE_MODULE}'"

  echo "Recreating Athena backend services on ${REMOTE}..."
  ssh "${REMOTE}" "set -e
cd '${REMOTE_APP_DIR}'
compose() {
  PROD_IMAGE='${IMAGE}' MINIO_IMAGE='${MINIO_IMAGE}' MINIO_MC_IMAGE='${MINIO_MC_IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' PROD_REDIS_VOLUME='${REDIS_VOLUME}' PROD_MINIO_VOLUME='${MINIO_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env \"\$@\"
}
athena_services=\"\$(compose config --services | awk '/^athena-/ && \$0 != \"athena-migrate\" && \$0 != \"athena-server\" { print }')\"
if [ -z \"\${athena_services}\" ]; then
  echo 'No Athena backend services found in docker-compose.prod.yml.'
  exit 1
fi
compose up -d --no-deps --force-recreate \${athena_services}
compose up -d --no-deps --force-recreate athena-server
compose ps"

  echo "Remote hot deployment completed. PostgreSQL, Redis, and MinIO data volumes were preserved."
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
docker volume create '${POSTGRES_VOLUME}' >/dev/null
docker volume create '${REDIS_VOLUME}' >/dev/null
docker volume create '${MINIO_VOLUME}' >/dev/null"

echo "Uploading compose file, environment file, Google OIDC client secret, and PostgreSQL init scripts..."
tar -C "${UPLOAD_DIR}" -cf - docker-compose.prod.yml .env secrets hack | ssh "${REMOTE}" "set -e
mkdir -p '${REMOTE_APP_DIR}'
tar -C '${REMOTE_APP_DIR}' -xf -
chown \"\$(id -u):\$(id -g)\" '${REMOTE_APP_DIR}/.env'
chmod 0600 '${REMOTE_APP_DIR}/.env'
chown '${ATHENA_CONTAINER_UID}:${ATHENA_CONTAINER_UID}' '${REMOTE_APP_DIR}/${GOOGLE_OIDC_SECRET_ARCHIVE_PATH}'
chmod 0600 '${REMOTE_APP_DIR}/${GOOGLE_OIDC_SECRET_ARCHIVE_PATH}'"

echo "Streaming Docker image ${IMAGE} to ${REMOTE}..."
docker save "${IMAGE}" | ssh "${REMOTE}" "docker load"
echo "Streaming Docker image ${MINIO_IMAGE} to ${REMOTE}..."
docker save "${MINIO_IMAGE}" | ssh "${REMOTE}" "docker load"
echo "Streaming Docker image ${MINIO_MC_IMAGE} to ${REMOTE}..."
docker save "${MINIO_MC_IMAGE}" | ssh "${REMOTE}" "docker load"

echo "Starting PostgreSQL on ${REMOTE}..."
ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' MINIO_IMAGE='${MINIO_IMAGE}' MINIO_MC_IMAGE='${MINIO_MC_IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' PROD_REDIS_VOLUME='${REDIS_VOLUME}' PROD_MINIO_VOLUME='${MINIO_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env up -d postgres"

echo "Running Athena migrations on ${REMOTE}..."
ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' MINIO_IMAGE='${MINIO_IMAGE}' MINIO_MC_IMAGE='${MINIO_MC_IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' PROD_REDIS_VOLUME='${REDIS_VOLUME}' PROD_MINIO_VOLUME='${MINIO_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env --profile tools run --rm athena-migrate athena up --module '${MIGRATE_MODULE}'"

echo "Starting Athena on ${REMOTE}..."
ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' MINIO_IMAGE='${MINIO_IMAGE}' MINIO_MC_IMAGE='${MINIO_MC_IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' PROD_REDIS_VOLUME='${REDIS_VOLUME}' PROD_MINIO_VOLUME='${MINIO_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env up -d"

echo "Remote deployment status:"
ssh "${REMOTE}" "cd '${REMOTE_APP_DIR}' && PROD_IMAGE='${IMAGE}' MINIO_IMAGE='${MINIO_IMAGE}' MINIO_MC_IMAGE='${MINIO_MC_IMAGE}' PROD_POSTGRES_VOLUME='${POSTGRES_VOLUME}' PROD_REDIS_VOLUME='${REDIS_VOLUME}' PROD_MINIO_VOLUME='${MINIO_VOLUME}' docker compose -f docker-compose.prod.yml --env-file .env ps"
