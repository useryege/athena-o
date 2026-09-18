#!/usr/bin/env bash
set -euo pipefail
caller_trader_sync_image="${TRADER_SYNC_IMAGE:-}"
caller_operation_log_image="${OPERATION_LOG_IMAGE:-}"
caller_worm_trading_image="${WORM_TRADING_IMAGE:-}"
caller_prod_account_state_maintenance="${PROD_ACCOUNT_STATE_MAINTENANCE:-}"
caller_prod_account_state_external_consumers_stopped="${PROD_ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED:-}"
repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo"
env_file="${PROD_ENV_FILE:-.env.prod}"
set -a
# shellcheck disable=SC1090 # Operator-selected local deployment configuration.
source "$env_file"
set +a
if [[ -n "${caller_trader_sync_image}" ]]; then export TRADER_SYNC_IMAGE="${caller_trader_sync_image}"; fi
if [[ -n "${caller_operation_log_image}" ]]; then export OPERATION_LOG_IMAGE="${caller_operation_log_image}"; fi
if [[ -n "${caller_worm_trading_image}" ]]; then export WORM_TRADING_IMAGE="${caller_worm_trading_image}"; fi
if [[ -n "${caller_prod_account_state_maintenance}" ]]; then export PROD_ACCOUNT_STATE_MAINTENANCE="${caller_prod_account_state_maintenance}"; fi
if [[ -n "${caller_prod_account_state_external_consumers_stopped}" ]]; then export PROD_ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED="${caller_prod_account_state_external_consumers_stopped}"; fi
export TRADER_SYNC_IMAGE="${TRADER_SYNC_IMAGE:-athena-trader-sync:local}"
export OPERATION_LOG_IMAGE="${OPERATION_LOG_IMAGE:-athena-operation-log:local}"
export WORM_TRADING_IMAGE="${WORM_TRADING_IMAGE:-athena-worm-trading:local}"
export PROD_IMAGE="${PROD_IMAGE:-athena:local}"
export ATHENA_COMPOSE_ENV_FILE="$env_file"
MIGRATE_MODULE="${PROD_MIGRATE_MODULE:-all}"
compose() {
  docker compose -f "${PROD_COMPOSE_FILE:-docker-compose.prod.yml}" --env-file "$env_file" "$@"
}
# shellcheck source=hack/lib/account-state-deploy.sh
source "$repo/hack/lib/account-state-deploy.sh"
for volume in "${PROD_POSTGRES_VOLUME:-athena-prod-postgres-data}" "${PROD_REDIS_VOLUME:-athena-prod-redis-data}" "${PROD_MINIO_VOLUME:-athena-prod-minio-data}"; do
  docker volume create "$volume" >/dev/null
done
compose up -d postgres
ready=false
for ((attempt=0; attempt<120; attempt++)); do
  # shellcheck disable=SC2016 # Expanded inside the PostgreSQL container.
  if compose exec -T postgres sh -c 'pg_isready --username "$POSTGRES_USER" --dbname "$POSTGRES_DB"' >/dev/null 2>&1; then ready=true; break; fi
  sleep 1
done
if [[ "$ready" != true ]]; then echo 'PostgreSQL did not become ready within 120 seconds' >&2; exit 1; fi
account_state_prepare
other_schema_up
ATHENA_SERVER_DISABLE_AUTH=false compose up -d

account_state_restore
