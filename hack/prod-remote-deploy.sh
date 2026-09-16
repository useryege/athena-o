#!/usr/bin/env bash
set -euo pipefail
caller_trader_sync_image="${TRADER_SYNC_IMAGE:-}"
caller_prod_account_state_maintenance="${PROD_ACCOUNT_STATE_MAINTENANCE:-}"
caller_prod_account_state_external_consumers_stopped="${PROD_ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED:-}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
# shellcheck source=hack/lib/ssh-command.sh
source "${SCRIPT_DIR}/lib/ssh-command.sh"

ENV_FILE="${PROD_ENV_FILE:-${REPO_ROOT}/.env}"
COMPOSE_FILE="${PROD_COMPOSE_FILE:-${REPO_ROOT}/docker-compose.prod.yml}"
IMAGE="${PROD_IMAGE:-athena:local}"
TRADER_SYNC_IMAGE="${TRADER_SYNC_IMAGE:-athena-trader-sync:local}"
MINIO_IMAGE="${MINIO_IMAGE:-athena-minio:9e49d5e7a648-go1.27.1}"
MINIO_MC_IMAGE="${MINIO_MC_IMAGE:-athena-minio-mc:7394ce0dd2a8-go1.27.1}"
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

if [[ "${ACTION}" != "deploy" && "${ACTION}" != "hot-deploy" && "${ACTION}" != "destroy" && "${ACTION}" != "trader-sync-deploy" ]]; then
  echo "Usage: $0 deploy|hot-deploy|trader-sync-deploy|destroy"
  exit 1
fi

if [[ -f "${ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +a
fi
if [[ -n "${caller_trader_sync_image}" ]]; then export TRADER_SYNC_IMAGE="${caller_trader_sync_image}"; fi
if [[ -n "${caller_prod_account_state_maintenance}" ]]; then export PROD_ACCOUNT_STATE_MAINTENANCE="${caller_prod_account_state_maintenance}"; fi
if [[ -n "${caller_prod_account_state_external_consumers_stopped}" ]]; then export PROD_ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED="${caller_prod_account_state_external_consumers_stopped}"; fi

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

PROD_REMOTE_SETUP="$(cat <<'REMOTE'
set -e
IMAGE="$1"
MINIO_IMAGE="$2"
MINIO_MC_IMAGE="$3"
POSTGRES_VOLUME="$4"
REDIS_VOLUME="$5"
MINIO_VOLUME="$6"
APP_DIR="$7"
MIGRATE_MODULE="$8"
GOOGLE_OIDC_SECRET_ARCHIVE_PATH="$9"
ATHENA_CONTAINER_UID="${10}"
TRADER_SYNC_IMAGE="${11}"
export PROD_ACCOUNT_STATE_MAINTENANCE="${12}" PROD_ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED="${13}"
export PROD_IMAGE="$IMAGE" MINIO_IMAGE MINIO_MC_IMAGE TRADER_SYNC_IMAGE
export PROD_POSTGRES_VOLUME="$POSTGRES_VOLUME" PROD_REDIS_VOLUME="$REDIS_VOLUME" PROD_MINIO_VOLUME="$MINIO_VOLUME"
compose() {
  docker compose -f docker-compose.prod.yml --env-file .env "$@"
}
REMOTE
)"

PROD_REMOTE_SETUP+=$'\n'"$(cat "${SCRIPT_DIR}/lib/account-state-deploy.sh")"

prod_remote_exec() {
  local body="$1"
  ssh_exec "${REMOTE}" bash -c "${PROD_REMOTE_SETUP}"$'\n'"${body}" _ \
    "${IMAGE}" "${MINIO_IMAGE}" "${MINIO_MC_IMAGE}" \
    "${POSTGRES_VOLUME}" "${REDIS_VOLUME}" "${MINIO_VOLUME}" \
    "${REMOTE_APP_DIR}" "${MIGRATE_MODULE}" \
    "${GOOGLE_OIDC_SECRET_ARCHIVE_PATH}" "${ATHENA_CONTAINER_UID}" \
    "${TRADER_SYNC_IMAGE}" "${PROD_ACCOUNT_STATE_MAINTENANCE:-false}" "${PROD_ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED:-false}"
}

destroy_remote() {
  echo "Removing Athena containers, network, and persistent data volumes from ${REMOTE}..."
  prod_remote_exec "$(cat <<'REMOTE'
docker --version >/dev/null
docker compose version >/dev/null
if [ -f "$APP_DIR/docker-compose.prod.yml" ]; then
  cd "$APP_DIR"
  mkdir -p secrets
  if [ ! -f "$GOOGLE_OIDC_SECRET_ARCHIVE_PATH" ]; then
    install -m 0600 /dev/null "$GOOGLE_OIDC_SECRET_ARCHIVE_PATH"
  fi
  if [ -f .env ]; then
    compose_env_file=.env
  else
    compose_env_file=/dev/null
  fi
  export ATHENA_ACCOUNT_STATE_POSTGRES_DSN=postgres://cleanup/unused ATHENA_URL=https://cleanup.invalid ATHENA_TRADER_SYNC_HTTP_URL=https://cleanup.invalid ATHENA_TRADER_SYNC_WSS_URL=wss://cleanup.invalid
  export ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN=dummy-worm-trading-internal-token-32bytes
  export ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN=dummy-notification-internal-token-32bytes
  export ATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN=dummy-telegram-bot-token
  export ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN=dummy-wallet-worm-execution-signer-token-32bytes
  export ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY=dummy-worm-trading-credential-encryption-key-32bytes
  export ATHENA_WORM_TRADING_SOLANA_RPC_URL=https://api.mainnet-beta.solana.com
  ATHENA_COMPOSE_ENV_FILE="$compose_env_file" ATHENA_TOKEN_ETH_ENABLED=dummy ATHENA_TOKEN_ETH_NODE_WS_URLS=dummy ATHENA_TOKEN_ETH_ATHENA_CONTRACT=dummy ATHENA_TOKEN_ETH_PROCESSOR_INITIAL_LOOKBACK_DURATION=dummy ATHENA_TOKEN_ETH_PROCESSOR_POLL_INTERVAL=dummy ATHENA_TOKEN_BSC_ENABLED=dummy ATHENA_TOKEN_BSC_NODE_WS_URLS=dummy ATHENA_TOKEN_BSC_ATHENA_CONTRACT=dummy ATHENA_TOKEN_BSC_PROCESSOR_INITIAL_LOOKBACK_DURATION=dummy ATHENA_TOKEN_BSC_PROCESSOR_POLL_INTERVAL=dummy POSTGRES_PASSWORD=dummy REDIS_PASSWORD=dummy ATHENA_JWT_SECRET=dummy-jwt-secret-for-compose-cleanup ATHENA_ADMIN_GOOGLE_EMAIL=dummy-admin@example.com ATHENA_WALLET_ENCRYPTION_KEY=dummy ATHENA_WALLET_INTERNAL_AUTH_TOKEN=dummy-wallet-internal-auth-token-32bytes MINIO_ROOT_PASSWORD=dummy-root-password ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID=dummy ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY=dummy ATHENA_NOTIFICATION_TEST_TELEGRAM_CHAT_ID=dummy ATHENA_NOTIFICATION_PROD_TELEGRAM_CHAT_ID=dummy ATHENA_ETHERSCAN_MANAGER_API_KEYS=dummy ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS=dummy ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN=dummy docker compose -f docker-compose.prod.yml --env-file "$compose_env_file" down --remove-orphans
fi
for volume_name in "$POSTGRES_VOLUME" "$REDIS_VOLUME" "$MINIO_VOLUME"; do
  if docker volume inspect "$volume_name" >/dev/null 2>&1; then
    docker volume rm "$volume_name" >/dev/null
  fi
done
REMOTE
)"
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

validate_internal_auth_token() {
  local name="$1"
  local value="$2"
  if (( ${#value} < 32 )); then
    echo "${name} must contain at least 32 bytes."
    exit 1
  fi
  if [[ "${value}" =~ [[:space:][:cntrl:]] ]]; then
    echo "${name} must not contain whitespace or control characters."
    exit 1
  fi
}

JWT_SECRET_VALUE="${ATHENA_JWT_SECRET:-}"
if (( ${#JWT_SECRET_VALUE} < 32 )); then
  echo "ATHENA_JWT_SECRET must contain at least 32 bytes."
  exit 1
fi

WALLET_INTERNAL_AUTH_TOKEN_VALUE="${ATHENA_WALLET_INTERNAL_AUTH_TOKEN:-}"
validate_internal_auth_token "ATHENA_WALLET_INTERNAL_AUTH_TOKEN" "${WALLET_INTERNAL_AUTH_TOKEN_VALUE}"

NOTIFICATION_INTERNAL_AUTH_TOKEN_VALUE="${ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN:-}"
validate_internal_auth_token "ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN" "${NOTIFICATION_INTERNAL_AUTH_TOKEN_VALUE}"
if [[ "${NOTIFICATION_INTERNAL_AUTH_TOKEN_VALUE}" == "${WALLET_INTERNAL_AUTH_TOKEN_VALUE}" ]]; then
  echo "ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN must differ from ATHENA_WALLET_INTERNAL_AUTH_TOKEN."
  exit 1
fi

WALLET_WORM_EXECUTION_SIGNER_TOKEN_VALUE="${ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN:-}"
validate_internal_auth_token "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN" "${WALLET_WORM_EXECUTION_SIGNER_TOKEN_VALUE}"
if [[ "${WALLET_WORM_EXECUTION_SIGNER_TOKEN_VALUE}" == "${WALLET_INTERNAL_AUTH_TOKEN_VALUE}" ]]; then
  echo "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN must differ from ATHENA_WALLET_INTERNAL_AUTH_TOKEN."
  exit 1
fi
if [[ "${WALLET_WORM_EXECUTION_SIGNER_TOKEN_VALUE}" == "${NOTIFICATION_INTERNAL_AUTH_TOKEN_VALUE}" ]]; then
  echo "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN must differ from ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN."
  exit 1
fi
if [[ "${WALLET_WORM_EXECUTION_SIGNER_TOKEN_VALUE}" == "${ATHENA_WALLET_ENCRYPTION_KEY:-}" ]]; then
  echo "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN must differ from ATHENA_WALLET_ENCRYPTION_KEY."
  exit 1
fi

WORM_TRADING_INTERNAL_AUTH_TOKEN_VALUE="${ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN:-}"
validate_internal_auth_token "ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN" "${WORM_TRADING_INTERNAL_AUTH_TOKEN_VALUE}"
if [[ "${WORM_TRADING_INTERNAL_AUTH_TOKEN_VALUE}" == "${WALLET_INTERNAL_AUTH_TOKEN_VALUE}" ]]; then
  echo "ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN must differ from ATHENA_WALLET_INTERNAL_AUTH_TOKEN."
  exit 1
fi
if [[ "${WORM_TRADING_INTERNAL_AUTH_TOKEN_VALUE}" == "${NOTIFICATION_INTERNAL_AUTH_TOKEN_VALUE}" ]]; then
  echo "ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN must differ from ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN."
  exit 1
fi
if [[ "${WORM_TRADING_INTERNAL_AUTH_TOKEN_VALUE}" == "${WALLET_WORM_EXECUTION_SIGNER_TOKEN_VALUE}" ]]; then
  echo "ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN must differ from ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN."
  exit 1
fi
WORM_TRADING_SOLANA_RPC_URL_VALUE="${ATHENA_WORM_TRADING_SOLANA_RPC_URL:-}"
if [[ ! "${WORM_TRADING_SOLANA_RPC_URL_VALUE}" =~ ^https?://[^/?#[:space:]]+([/?#][^[:space:]]*)?$ ]]; then
  echo "ATHENA_WORM_TRADING_SOLANA_RPC_URL must be an absolute HTTP or HTTPS URL with a host and no whitespace."
  exit 1
fi
WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY_VALUE="${ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY:-}"
validate_internal_auth_token "ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY" "${WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY_VALUE}"
if [[ "${WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY_VALUE}" == "${WORM_TRADING_INTERNAL_AUTH_TOKEN_VALUE}" || "${WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY_VALUE}" == "${WALLET_INTERNAL_AUTH_TOKEN_VALUE}" || "${WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY_VALUE}" == "${WALLET_WORM_EXECUTION_SIGNER_TOKEN_VALUE}" || "${WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY_VALUE}" == "${ATHENA_WALLET_ENCRYPTION_KEY:-}" ]]; then
  echo "ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY must be independent from Wallet and internal authentication secrets."
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
  ATHENA_ADMIN_GOOGLE_EMAIL
)
for variable_name in "${required_oidc_variables[@]}"; do
  if [[ -z "${!variable_name:-}" ]]; then
    echo "${variable_name} is required for production deployment."
    exit 1
  fi
done
if [[ ! "${ATHENA_GOOGLE_OIDC_REDIRECT_URI}" =~ ^https://[^/?#]+(/[^/?#[:space:]]+)*/auth/google/callback$ ]]; then
  echo "ATHENA_GOOGLE_OIDC_REDIRECT_URI must be an explicit HTTPS URI ending exactly in the deployment path plus /auth/google/callback."
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

trader_secret_names=(INTERNAL_AUTH_TOKEN CURSOR_HMAC_KEY TLS_CERT TLS_KEY TLS_CA)
trader_secret_files=(trader-sync-token trader-sync-cursor-key trader-sync-cert trader-sync-key trader-sync-ca)
trader_secret_sources=()
for suffix in "${trader_secret_names[@]}"; do
  variable="ATHENA_TRADER_SYNC_${suffix}_FILE"
  source_file="${!variable:-}"
  if [[ -z "$source_file" ]]; then echo "$variable is required" >&2; exit 1; fi
  if [[ "$source_file" != /* ]]; then source_file="$(dirname "$ENV_FILE")/$source_file"; fi
  if [[ ! -s "$source_file" ]]; then echo "$variable must reference a nonempty file" >&2; exit 1; fi
  trader_secret_sources+=("$source_file")
done
trader_token="$(cat "${trader_secret_sources[0]}")"
validate_internal_auth_token ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN "$trader_token"
if [[ "$trader_token" == "$(cat "${trader_secret_sources[1]}")" ]]; then echo 'Trader Sync internal token must differ from cursor key' >&2; exit 1; fi
unset trader_token
if [[ -n "${ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN:-}" || -n "${ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY:-}" ]]; then
  echo 'Production Trader Sync secrets require only _FILE values' >&2; exit 1
fi
if [[ -z "${ATHENA_ACCOUNT_STATE_POSTGRES_DSN:-}" ]]; then echo 'ATHENA_ACCOUNT_STATE_POSTGRES_DSN is required' >&2; exit 1; fi
if ! docker image inspect "${TRADER_SYNC_IMAGE}" >/dev/null 2>&1; then
  echo "Docker image not found locally: ${TRADER_SYNC_IMAGE}"; exit 1
fi
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

for index in "${!trader_secret_names[@]}"; do
  variable="ATHENA_TRADER_SYNC_${trader_secret_names[index]}_FILE"
  archive_file="secrets/${trader_secret_files[index]}"
  install -m 0600 "${trader_secret_sources[index]}" "${UPLOAD_DIR}/${archive_file}"
  # Input is a fixed configuration key; values are fixed archive-relative paths.
  sed -i "/^${variable}=/d" "${UPLOAD_DIR}/.env"
  printf "%s='./%s'\n" "$variable" "$archive_file" >>"${UPLOAD_DIR}/.env"
done

if [[ "${ACTION}" == "hot-deploy" || "${ACTION}" == "trader-sync-deploy" ]]; then
  echo "Checking remote host and persistent data volumes on ${REMOTE}..."
  prod_remote_exec "$(cat <<'REMOTE'
docker --version >/dev/null
docker compose version >/dev/null
if ! docker volume inspect "$POSTGRES_VOLUME" >/dev/null 2>&1; then
  echo "PostgreSQL volume not found: $POSTGRES_VOLUME. Run a full deployment first."
  exit 1
fi
if ! docker volume inspect "$REDIS_VOLUME" >/dev/null 2>&1; then
  echo "Redis volume not found: $REDIS_VOLUME. Run a full deployment first."
  exit 1
fi
if ! docker volume inspect "$MINIO_VOLUME" >/dev/null 2>&1; then
  echo "MinIO volume not found: $MINIO_VOLUME. Run a full deployment first."
  exit 1
fi
mkdir -p "$APP_DIR"
REMOTE
)"

  echo "Uploading compose file, environment file, and Google OIDC client secret..."
  tar -C "${UPLOAD_DIR}" -cf - docker-compose.prod.yml .env secrets | prod_remote_exec "$(cat <<'REMOTE'
tar -C "$APP_DIR" -xf -
chown "$(id -u):$(id -g)" "$APP_DIR/.env"
chmod 0600 "$APP_DIR/.env"
chown "$ATHENA_CONTAINER_UID:$ATHENA_CONTAINER_UID" "$APP_DIR/$GOOGLE_OIDC_SECRET_ARCHIVE_PATH"
chmod 0600 "$APP_DIR/$GOOGLE_OIDC_SECRET_ARCHIVE_PATH"
for secret in "$APP_DIR"/secrets/trader-sync-*; do
  chown "$ATHENA_CONTAINER_UID:$ATHENA_CONTAINER_UID" "$secret"
  chmod 0600 "$secret"
done
REMOTE
)"

  for image in "${IMAGE}" "${MINIO_IMAGE}" "${MINIO_MC_IMAGE}" "${TRADER_SYNC_IMAGE}"; do
    echo "Streaming Docker image ${image} to ${REMOTE}..."
    docker save "${image}" | prod_remote_exec 'docker load'
  done

  if [[ "${ACTION}" == "trader-sync-deploy" ]]; then
    prod_remote_exec "$(cat <<'REMOTE'
cd "$APP_DIR"
account_state_prepare
compose stop -t 40 athena-trader-sync
running="$(compose ps --status running -q athena-trader-sync)"
if [ -n "$running" ]; then echo 'Trader Sync has not exited' >&2; exit 1; fi
if [ "$ACCOUNT_STATE_CHANGED" = true ]; then
  if ((${#ACCOUNT_STATE_RESTART_SERVICES[@]})); then
    compose up -d --no-deps --force-recreate "${ACCOUNT_STATE_RESTART_SERVICES[@]}"
  fi
  # The selected service may have been stopped before maintenance.
  if [[ " ${ACCOUNT_STATE_RESTART_SERVICES[*]} " != *" athena-trader-sync "* ]]; then
    compose up -d --no-deps --force-recreate athena-trader-sync
  fi
else
  compose up -d --no-deps --force-recreate athena-trader-sync
fi
compose ps
REMOTE
)"
    echo 'Trader Sync deployment completed.'
    exit 0
  fi

  echo "Ensuring the Profit Sharing database exists on ${REMOTE}..."
  prod_remote_exec "$(cat <<'REMOTE'
cd "$APP_DIR"
compose up -d postgres redis
postgres_ready=false
for attempt in $(seq 1 120); do
  if compose exec -T postgres sh -c 'pg_isready --username "$POSTGRES_USER" --dbname "$POSTGRES_DB"' >/dev/null 2>&1; then
    postgres_ready=true
    break
  fi
  sleep 1
done
if [ "$postgres_ready" != true ]; then
  echo 'PostgreSQL did not become ready before Profit Sharing database creation.'
  exit 1
fi
redis_ready=false
for attempt in $(seq 1 60); do
  if compose exec -T redis sh -c 'redis-cli -a "$REDIS_PASSWORD" ping' 2>/dev/null | grep -qx PONG; then
    redis_ready=true
    break
  fi
  sleep 1
done
if [ "$redis_ready" != true ]; then
  echo 'Redis did not become ready before Athena service recreation.'
  exit 1
fi
if ! compose exec -T postgres sh -c 'createdb --username "$POSTGRES_USER" profit_sharing' >/dev/null 2>&1; then
  compose exec -T postgres sh -c 'psql --username "$POSTGRES_USER" --dbname profit_sharing --command "SELECT 1"' >/dev/null
fi
REMOTE
)"

  echo "Starting and initializing MinIO on ${REMOTE}..."
  prod_remote_exec "$(cat <<'REMOTE'
cd "$APP_DIR"
compose up -d minio
minio_ready=false
for attempt in $(seq 1 60); do
  if compose exec -T minio curl --fail --silent http://127.0.0.1:9000/minio/health/live >/dev/null 2>&1; then
    minio_ready=true
    break
  fi
  sleep 1
done
if [ "$minio_ready" != true ]; then
  echo 'MinIO did not become ready before bucket initialization.'
  exit 1
fi
compose run --rm --no-deps minio-init
REMOTE
)"

  echo "Running Athena migrations on ${REMOTE}..."
  echo "Recreating Athena backend services after verified migrations on ${REMOTE}..."
  prod_remote_exec "$(cat <<'REMOTE'
cd "$APP_DIR"
account_state_prepare
other_schema_up
mapfile -t athena_services < <(compose config --services | awk '/^athena-/ && $0 != "athena-migrate" && $0 != "athena-account-state-migrate" && $0 != "athena-server" { print }')
if ((${#athena_services[@]} == 0)); then
  echo 'No Athena backend services found in docker-compose.prod.yml.'
  exit 1
fi
compose up -d --no-deps --force-recreate "${athena_services[@]}"
compose up -d --no-deps --force-recreate athena-server
account_state_restore
compose ps
REMOTE
)"

  echo "Remote hot deployment completed. PostgreSQL, Redis, and MinIO data volumes were preserved."
  exit 0
fi

mkdir -p "${UPLOAD_DIR}/hack/postgres"
cp -a "${REPO_ROOT}/hack/postgres/init" "${UPLOAD_DIR}/hack/postgres/init"

echo "Preparing remote host ${REMOTE}..."
destroy_remote
prod_remote_exec "$(cat <<'REMOTE'
docker --version >/dev/null
docker compose version >/dev/null
mkdir -p "$APP_DIR"
docker volume create "$POSTGRES_VOLUME" >/dev/null
docker volume create "$REDIS_VOLUME" >/dev/null
docker volume create "$MINIO_VOLUME" >/dev/null
REMOTE
)"

echo "Uploading compose file, environment file, Google OIDC client secret, and PostgreSQL init scripts..."
tar -C "${UPLOAD_DIR}" -cf - docker-compose.prod.yml .env secrets hack | prod_remote_exec "$(cat <<'REMOTE'
mkdir -p "$APP_DIR"
tar -C "$APP_DIR" -xf -
chown "$(id -u):$(id -g)" "$APP_DIR/.env"
chmod 0600 "$APP_DIR/.env"
chown "$ATHENA_CONTAINER_UID:$ATHENA_CONTAINER_UID" "$APP_DIR/$GOOGLE_OIDC_SECRET_ARCHIVE_PATH"
chmod 0600 "$APP_DIR/$GOOGLE_OIDC_SECRET_ARCHIVE_PATH"
for secret in "$APP_DIR"/secrets/trader-sync-*; do
  chown "$ATHENA_CONTAINER_UID:$ATHENA_CONTAINER_UID" "$secret"
  chmod 0600 "$secret"
done
REMOTE
)"

for image in "${IMAGE}" "${MINIO_IMAGE}" "${MINIO_MC_IMAGE}" "${TRADER_SYNC_IMAGE}"; do
  echo "Streaming Docker image ${image} to ${REMOTE}..."
  docker save "${image}" | prod_remote_exec 'docker load'
done

echo "Starting PostgreSQL on ${REMOTE}..."
# Remote variables expand in bash on the target host.
# shellcheck disable=SC2016
prod_remote_exec 'cd "$APP_DIR"; compose up -d --wait --wait-timeout 120 postgres'

echo "Running Athena migrations on ${REMOTE}..."
# Remote variables expand in bash on the target host.
# shellcheck disable=SC2016
prod_remote_exec 'cd "$APP_DIR"; account_state_prepare; other_schema_up; compose up -d; account_state_restore'

echo "Remote deployment status:"
# Remote variables expand in bash on the target host.
# shellcheck disable=SC2016
prod_remote_exec 'cd "$APP_DIR"; compose ps'
