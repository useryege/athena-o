#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
# shellcheck source=hack/lib/ssh-command.sh
source "${SCRIPT_DIR}/lib/ssh-command.sh"

provided_remote_host="${REMOTE_HOST:-}"
provided_remote_user="${REMOTE_USER:-}"
provided_target_arch="${TARGET_ARCH:-}"
provided_image="${BSC_INDEXER_IMAGE:-}"
provided_dockerfile="${BSC_INDEXER_DOCKERFILE:-}"
provided_compose_file="${BSC_INDEXER_COMPOSE_FILE:-}"
provided_remote_app_dir="${BSC_INDEXER_REMOTE_APP_DIR:-}"

ENV_FILE="${BSC_INDEXER_ENV_FILE:-${REPO_ROOT}/.env.bsc-transaction-indexer}"
if [[ "${ENV_FILE}" != /* ]]; then
  ENV_FILE="${REPO_ROOT}/${ENV_FILE}"
fi
if [[ ! -f "${ENV_FILE}" ]]; then
  echo "BSC indexer environment file not found: ${ENV_FILE}" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "${ENV_FILE}"
set +a

if [[ -n "${provided_remote_host}" ]]; then REMOTE_HOST="${provided_remote_host}"; fi
if [[ -n "${provided_remote_user}" ]]; then REMOTE_USER="${provided_remote_user}"; fi
if [[ -n "${provided_target_arch}" ]]; then TARGET_ARCH="${provided_target_arch}"; fi
if [[ -n "${provided_image}" ]]; then BSC_INDEXER_IMAGE="${provided_image}"; fi
if [[ -n "${provided_dockerfile}" ]]; then BSC_INDEXER_DOCKERFILE="${provided_dockerfile}"; fi
if [[ -n "${provided_compose_file}" ]]; then BSC_INDEXER_COMPOSE_FILE="${provided_compose_file}"; fi
if [[ -n "${provided_remote_app_dir}" ]]; then BSC_INDEXER_REMOTE_APP_DIR="${provided_remote_app_dir}"; fi

REMOTE_HOST="${REMOTE_HOST:-}"
REMOTE_USER="${REMOTE_USER:-root}"
TARGET_ARCH="${TARGET_ARCH:-linux/amd64}"
BSC_INDEXER_IMAGE="${BSC_INDEXER_IMAGE:-athena-bsc-transaction-indexer:local}"
BSC_INDEXER_DOCKERFILE="${BSC_INDEXER_DOCKERFILE:-deploy/bsc-transaction-indexer/Dockerfile}"
BSC_INDEXER_COMPOSE_FILE="${BSC_INDEXER_COMPOSE_FILE:-deploy/bsc-transaction-indexer/docker-compose.yml}"
BSC_INDEXER_REMOTE_APP_DIR="${BSC_INDEXER_REMOTE_APP_DIR:-/opt/athena-bsc-transaction-indexer}"
BSC_INDEXER_GRPC_PORT="${BSC_INDEXER_GRPC_PORT:-8130}"
BSC_INDEXER_TELEMETRY_PORT="${BSC_INDEXER_TELEMETRY_PORT:-8131}"

if [[ "${BSC_INDEXER_DOCKERFILE}" != /* ]]; then
  BSC_INDEXER_DOCKERFILE="${REPO_ROOT}/${BSC_INDEXER_DOCKERFILE}"
fi
if [[ "${BSC_INDEXER_COMPOSE_FILE}" != /* ]]; then
  BSC_INDEXER_COMPOSE_FILE="${REPO_ROOT}/${BSC_INDEXER_COMPOSE_FILE}"
fi

required_value() {
  local name="$1"
  local value="$2"
  if [[ -z "${value}" ]]; then
    echo "${name} is required. Set it in ${ENV_FILE} or pass it to make." >&2
    exit 1
  fi
}

valid_port() {
  local name="$1"
  local value="$2"
  if [[ ! "${value}" =~ ^[0-9]+$ ]]; then
    echo "${name} must be an integer from 1 through 65535." >&2
    exit 1
  fi
  local numeric_value=$((10#${value}))
  if ((numeric_value < 1 || numeric_value > 65535)); then
    echo "${name} must be an integer from 1 through 65535." >&2
    exit 1
  fi
}

required_value REMOTE_HOST "${REMOTE_HOST}"
required_value POSTGRES_PASSWORD "${POSTGRES_PASSWORD:-}"
required_value ATHENA_BSC_INBOUND_NODE_RPC_URL "${ATHENA_BSC_INBOUND_NODE_RPC_URL:-}"
valid_port BSC_INDEXER_GRPC_PORT "${BSC_INDEXER_GRPC_PORT}"
valid_port BSC_INDEXER_TELEMETRY_PORT "${BSC_INDEXER_TELEMETRY_PORT}"

if [[ ! -f "${BSC_INDEXER_DOCKERFILE}" ]]; then
  echo "BSC indexer Dockerfile not found: ${BSC_INDEXER_DOCKERFILE}" >&2
  exit 1
fi
if [[ ! -f "${BSC_INDEXER_COMPOSE_FILE}" ]]; then
  echo "BSC indexer Compose file not found: ${BSC_INDEXER_COMPOSE_FILE}" >&2
  exit 1
fi

command -v docker >/dev/null
command -v scp >/dev/null
command -v ssh >/dev/null
docker compose version >/dev/null

echo "Validating the standalone BSC indexer Compose configuration..."
BSC_INDEXER_IMAGE="${BSC_INDEXER_IMAGE}" docker compose \
  -f "${BSC_INDEXER_COMPOSE_FILE}" \
  --env-file "${ENV_FILE}" \
  config --quiet

echo "Building ${BSC_INDEXER_IMAGE} for ${TARGET_ARCH}..."
git_commit="$(git -C "${REPO_ROOT}" rev-parse HEAD)"
git_tree_state="dirty"
if [[ -z "$(git -C "${REPO_ROOT}" status --porcelain)" ]]; then
  git_tree_state="clean"
fi
git_tag="$(git -C "${REPO_ROOT}" describe --exact-match --tags HEAD 2>/dev/null || true)"
build_date="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
DOCKER_BUILDKIT=1 docker build \
  --platform="${TARGET_ARCH}" \
  --file="${BSC_INDEXER_DOCKERFILE}" \
  --tag="${BSC_INDEXER_IMAGE}" \
  --build-arg="GIT_COMMIT=${git_commit}" \
  --build-arg="GIT_TREE_STATE=${git_tree_state}" \
  --build-arg="GIT_TAG=${git_tag}" \
  --build-arg="BUILD_DATE=${build_date}" \
  "${REPO_ROOT}"

remote="${REMOTE_USER}@${REMOTE_HOST}"
remote_tmp_compose="/tmp/athena-bsc-transaction-indexer-compose.$$"
remote_tmp_env="/tmp/athena-bsc-transaction-indexer-env.$$"

# Remote variables expand in bash on the target host.
# shellcheck disable=SC2016
remote_setup='set -euo pipefail
APP_DIR="$1"
IMAGE="$2"
export BSC_INDEXER_IMAGE="$IMAGE"'
remote_exec() {
  local body="$1"
  ssh_exec "${remote}" bash -c "${remote_setup}"$'\n'"${body}" _ "${BSC_INDEXER_REMOTE_APP_DIR}" "${BSC_INDEXER_IMAGE}" "${remote_tmp_compose}" "${remote_tmp_env}"
}

echo "Checking Docker on ${remote}..."
# Remote variables expand in bash on the target host.
# shellcheck disable=SC2016
remote_exec 'docker --version >/dev/null
docker compose version >/dev/null
mkdir -p "$APP_DIR"'

echo "Uploading the standalone Compose and environment files..."
scp "${BSC_INDEXER_COMPOSE_FILE}" "${remote}:${remote_tmp_compose}"
scp "${ENV_FILE}" "${remote}:${remote_tmp_env}"
remote_exec "$(cat <<'REMOTE'
install -m 0644 "$3" "$APP_DIR/docker-compose.yml"
install -m 0600 "$4" "$APP_DIR/.env"
rm -f "$3" "$4"
REMOTE
)"

echo "Streaming ${BSC_INDEXER_IMAGE} to ${remote}..."
docker save "${BSC_INDEXER_IMAGE}" | ssh_exec "${remote}" docker load

echo "Starting PostgreSQL 18 and the BSC indexer..."
# Remote variables expand in bash on the target host.
# shellcheck disable=SC2016
if ! remote_exec 'cd "$APP_DIR"
docker compose -f docker-compose.yml --env-file .env up -d --force-recreate --remove-orphans --wait --wait-timeout 180'; then
  echo "Remote deployment failed. Recent service logs follow." >&2
  # Remote variables expand in bash on the target host.
  # shellcheck disable=SC2016
  remote_exec 'cd "$APP_DIR"; docker compose -f docker-compose.yml --env-file .env ps' >&2 || true
  # Remote variables expand in bash on the target host.
  # shellcheck disable=SC2016
  remote_exec 'cd "$APP_DIR"; docker compose -f docker-compose.yml --env-file .env logs --tail=120' >&2 || true
  exit 1
fi

echo "Remote deployment status:"
# Remote variables expand in bash on the target host.
# shellcheck disable=SC2016
remote_exec 'cd "$APP_DIR"; docker compose -f docker-compose.yml --env-file .env ps'

echo
echo "BSC transaction indexer deployed successfully."
echo "gRPC endpoint: ${REMOTE_HOST}:${BSC_INDEXER_GRPC_PORT}"
echo "Readiness: ssh ${remote} \"cd ${BSC_INDEXER_REMOTE_APP_DIR} && docker compose exec -T indexer curl -i http://127.0.0.1:8131/readyz\""
echo "Logs: ssh ${remote} \"cd ${BSC_INDEXER_REMOTE_APP_DIR} && docker compose logs -f indexer\""
