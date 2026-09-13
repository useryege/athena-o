#!/usr/bin/env bash
set -euo pipefail

source_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
cleanup_test() {
  local status=$? output
  if ((status)); then
    for output in "${tmp}"/*.out "${TEST_LOG:-/nonexistent}"; do
      if [[ -f "$output" ]]; then
        printf '== %s ==\n' "$output" >&2
        cat "$output" >&2
      fi
    done
  fi
  rm -rf "${tmp}"
}
trap cleanup_test EXIT
fixture="${tmp}/repo"
mkdir -p "${fixture}/hack/lib" "${fixture}/hack/postgres/init" "${fixture}/dist" "${fixture}/deploy/bsc-transaction-indexer" "${fixture}/deploy/bsc-swap-indexer" "${tmp}/bin"
cp "${source_root}/hack/prod-remote-deploy.sh" "${source_root}/hack/deploy-etherscan-gateway.sh" \
  "${source_root}/hack/deploy-bsc-transaction-indexer.sh" "${source_root}/hack/deploy-bsc-swap-indexer.sh" "${fixture}/hack/"
cp "${source_root}/hack/lib/ssh-command.sh" "${fixture}/hack/lib/"
sed -i \
  -e "s|/usr/local/bin/athena-etherscan-gateway|${tmp}/gateway-bin|g" \
  -e "s|/etc/athena|${tmp}/gateway-etc|g" \
  -e "s|/etc/systemd/system|${tmp}/gateway-systemd|g" \
  -e "s|/tmp/athena-etherscan-gateway|${tmp}/athena-etherscan-gateway|g" \
  -e "s|/tmp/etherscan-gateway.env|${tmp}/etherscan-gateway.env|g" \
  "${fixture}/hack/deploy-etherscan-gateway.sh"
sed -i -e "s|/tmp/athena-bsc|${tmp}/athena-bsc|g" \
  "${fixture}/hack/deploy-bsc-transaction-indexer.sh" "${fixture}/hack/deploy-bsc-swap-indexer.sh"
printf 'services: {}\n' >"${fixture}/docker-compose.prod.yml"
printf 'services: {}\n' >"${fixture}/deploy/bsc-transaction-indexer/docker-compose.yml"
printf 'services: {}\n' >"${fixture}/deploy/bsc-swap-indexer/docker-compose.yml"
: >"${fixture}/deploy/bsc-transaction-indexer/Dockerfile"
: >"${fixture}/deploy/bsc-swap-indexer/Dockerfile"
printf 'binary\n' >"${fixture}/dist/athena-etherscan-gateway"
printf 'oidc-secret\n' >"${tmp}/oidc"

cat >"${tmp}/bin/ssh" <<'SSH'
#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == -- ]]
shift
host="$1"; shift
printf 'ssh %s\n' "$host" >>"${TEST_LOG}"
TEST_REMOTE_SHELL=1 POSTGRES_USER="${TEST_REMOTE_POSTGRES_USER}" \
  POSTGRES_DB="${TEST_REMOTE_POSTGRES_DB}" REDIS_PASSWORD="${TEST_REMOTE_REDIS_PASSWORD}" \
  /bin/sh -c "$*"
SSH
cat >"${tmp}/bin/scp" <<'SCP'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" == -C ]]; then shift; fi
cp "$1" "${2#*:}"
printf 'scp %s\n' "${2#*:}" >>"${TEST_LOG}"
SCP
cat >"${tmp}/bin/docker" <<'DOCKER'
#!/usr/bin/env bash
set -euo pipefail
printf 'docker %s %s %s %s\n' "${TEST_REMOTE_SHELL:-local}" "${PWD}" "${BSC_INDEXER_IMAGE:-${BSC_SWAP_INDEXER_IMAGE:-${PROD_IMAGE:-}}}" "$*" >>"${TEST_LOG}"
case "${1:-} ${2:-}" in
  'save '*) printf '\000\001\377IMAGE\000';;
  'load '*) cat >>"${TEST_IMAGE_LOG}";;
  'volume inspect') [[ -e "${TEST_VOLUME_DIR}/$3" ]];;
  'volume create') : >"${TEST_VOLUME_DIR}/$3";;
  'volume rm') printf '%s\0' "$3" >>"${TEST_VOLUME_RM_LOG}"; [[ "${TEST_FAIL_VOLUME_RM:-}" != yes ]] || exit 42; rm -f "${TEST_VOLUME_DIR}/$3";;
esac
if [[ "$*" == *'config --services'* ]]; then printf 'athena-a\nathena-server\nathena-migrate\n'; fi
argv=("$@")
for ((index = 0; index < ${#argv[@]}; index++)); do
  if [[ "${argv[index]}" == exec && "${argv[index+1]:-}" == -T && "${argv[index+3]:-}" == sh && "${argv[index+4]:-}" == -c ]]; then
    case "${argv[index+2]:-}" in postgres|redis) ;; *) exit 70;; esac
    ((${#argv[@]} == index + 6)) || exit 71
    env -i PATH="${TEST_CONTAINER_BIN}:/usr/bin:/bin" TEST_CONTAINER_ARGV_DIR="${TEST_CONTAINER_ARGV_DIR}" \
      TEST_FAIL_CREATEDB="${TEST_FAIL_CREATEDB:-}" \
      POSTGRES_USER="${TEST_CONTAINER_POSTGRES_USER}" POSTGRES_DB="${TEST_CONTAINER_POSTGRES_DB}" \
      REDIS_PASSWORD="${TEST_CONTAINER_REDIS_PASSWORD}" \
      /bin/sh -c "${argv[index+5]}"
    exit $?
  fi
done
if [[ "${TEST_FAIL_UP:-}" == yes && "${TEST_REMOTE_SHELL:-}" == 1 && "$*" == *'--wait-timeout 180'* ]]; then exit 39; fi
if [[ "${TEST_FAIL_MIGRATE:-}" == yes && "$*" == *'athena-migrate'* ]]; then exit 41; fi
exit 0
DOCKER
cat >"${tmp}/bin/systemctl" <<'SYSTEMCTL'
#!/usr/bin/env bash
set -euo pipefail
printf 'systemctl %s\n' "$*" >>"${TEST_LOG}"
case "$1" in
  is-active) [[ "${TEST_FAIL_SERVICE:-}" != yes ]] && echo active;;
  is-enabled) echo enabled;;
esac
SYSTEMCTL
cat >"${tmp}/bin/ss" <<'SS'
#!/usr/bin/env bash
printf 'LISTEN 0 128 *:6776\n'
SS
cat >"${tmp}/bin/journalctl" <<'JOURNAL'
#!/usr/bin/env bash
printf 'journalctl %s\n' "$*" >>"${TEST_LOG}"
JOURNAL
cat >"${tmp}/bin/make" <<'MAKE'
#!/usr/bin/env bash
printf 'make %s\n' "$*" >>"${TEST_LOG}"
MAKE
cat >"${tmp}/bin/chown" <<'CHOWN'
#!/usr/bin/env bash
printf 'chown %s\n' "$*" >>"${TEST_LOG}"
CHOWN
cat >"${tmp}/bin/install" <<'INSTALL'
#!/usr/bin/env bash
set -euo pipefail
destination="${@: -1}"
source="${@: -2:1}"
cp "$source" "$destination"
INSTALL
cat >"${tmp}/bin/git" <<'GIT'
#!/usr/bin/env bash
case "$*" in *'rev-parse HEAD'*) echo deadbeef;; esac
GIT
mkdir -p "${tmp}/container-bin"
for container_command in pg_isready createdb psql redis-cli; do
  cat >"${tmp}/container-bin/${container_command}" <<'CONTAINER'
#!/usr/bin/env bash
set -euo pipefail
name="${0##*/}"
printf '%s\0' "$@" >"${TEST_CONTAINER_ARGV_DIR}/${name}"
if [[ "$name" == createdb && "${TEST_FAIL_CREATEDB:-}" == yes ]]; then exit 43; fi
if [[ "$name" == redis-cli ]]; then printf 'PONG\n'; fi
CONTAINER
done
chmod +x "${tmp}/bin/"*
chmod +x "${tmp}/container-bin/"*
export PATH="${tmp}/bin:${PATH}" TEST_LOG="${tmp}/events" TEST_IMAGE_LOG="${tmp}/images" TEST_VOLUME_DIR="${tmp}/volumes"
export TEST_VOLUME_RM_LOG="${tmp}/volume-rm-argv" TEST_CONTAINER_ARGV_DIR="${tmp}/container-argv" TEST_CONTAINER_BIN="${tmp}/container-bin"
mkdir -p "${TEST_VOLUME_DIR}"
mkdir -p "${TEST_CONTAINER_ARGV_DIR}"
mkdir -p "${tmp}/gateway-systemd"

assert_order() {
  local first="$1" second="$2" first_line second_line
  first_line="$(rg -n -F -m 1 -- "$first" "${TEST_LOG}" | cut -d: -f1)"
  second_line="$(rg -n -F -m 1 -- "$second" "${TEST_LOG}" | cut -d: -f1)"
  [[ -n "$first_line" && -n "$second_line" && "$first_line" -lt "$second_line" ]]
}

special_dir="${tmp}/remote single'quote \$(touch ${tmp}/injected)"
mkdir -p "${special_dir}"
cat >"${tmp}/prod.env" <<EOF
REMOTE_HOST=example
ATHENA_JWT_SECRET=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
ATHENA_WALLET_INTERNAL_AUTH_TOKEN=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN=cccccccccccccccccccccccccccccccc
ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN=dddddddddddddddddddddddddddddddd
ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN=eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee
ATHENA_WORM_TRADING_SOLANA_RPC_URL=https://example.com
ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY=ffffffffffffffffffffffffffffffff
ATHENA_GOOGLE_OIDC_CLIENT_ID=client
ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE=${tmp}/oidc
ATHENA_GOOGLE_OIDC_REDIRECT_URI=https://example.com/auth/google/callback
ATHENA_ADMIN_GOOGLE_EMAIL=admin@example.com
EOF
export PROD_ENV_FILE="${tmp}/prod.env" PROD_COMPOSE_FILE="${fixture}/docker-compose.prod.yml" REMOTE_APP_DIR="${special_dir}"
export PROD_POSTGRES_VOLUME="pg single'quote" PROD_REDIS_VOLUME='redis data' PROD_MINIO_VOLUME='minio data'
export PROD_IMAGE="image'\$(touch ${tmp}/injected)" MINIO_IMAGE=minio-image MINIO_MC_IMAGE=mc-image
export PROD_MIGRATE_MODULE="module'\$(touch ${tmp}/injected)"
unrelated_volume="other volume's data"
: >"${TEST_VOLUME_DIR}/${unrelated_volume}"
: >"${TEST_VOLUME_RM_LOG}"
export POSTGRES_USER="local user \$wrong" POSTGRES_DB='local db *' REDIS_PASSWORD="local redis \`wrong\`"
export TEST_REMOTE_POSTGRES_USER="remote user'wrong" TEST_REMOTE_POSTGRES_DB="remote db \$wrong" TEST_REMOTE_REDIS_PASSWORD='remote redis *'
export TEST_CONTAINER_POSTGRES_USER="container user \$right" TEST_CONTAINER_POSTGRES_DB="container db's right" TEST_CONTAINER_REDIS_PASSWORD="container redis \`right\`"

: >"${TEST_LOG}"
bash "${fixture}/hack/prod-remote-deploy.sh" deploy >"${tmp}/prod-deploy.out" 2>&1
[[ ! -e "${tmp}/injected" ]]
rg -q -F "docker 1 ${special_dir} ${PROD_IMAGE} compose -f docker-compose.prod.yml --env-file .env up -d postgres" "${TEST_LOG}"
assert_order 'compose -f docker-compose.prod.yml --env-file .env up -d postgres' '--profile tools run --rm athena-migrate'
assert_order '--profile tools run --rm athena-migrate' 'compose -f docker-compose.prod.yml --env-file .env ps'
for volume in "${PROD_POSTGRES_VOLUME}" "${PROD_REDIS_VOLUME}" "${PROD_MINIO_VOLUME}"; do [[ -e "${TEST_VOLUME_DIR}/${volume}" ]]; done
[[ -e "${TEST_VOLUME_DIR}/${unrelated_volume}" ]]
[[ "$(find "${TEST_VOLUME_DIR}" -type f | wc -l)" -eq 4 ]]
[[ ! -s "${TEST_VOLUME_RM_LOG}" ]]
cmp <(printf '\000\001\377IMAGE\000\000\001\377IMAGE\000\000\001\377IMAGE\000') "${TEST_IMAGE_LOG}"

: >"${TEST_LOG}"
TEST_FAIL_CREATEDB=yes bash "${fixture}/hack/prod-remote-deploy.sh" hot-deploy >"${tmp}/prod-hot.out" 2>&1
rg -q -F 'up -d --no-deps --force-recreate athena-a' "${TEST_LOG}"
assert_order 'up -d postgres redis' '--profile tools run --rm athena-migrate'
cmp <(printf '%s\0' '--username' "${TEST_CONTAINER_POSTGRES_USER}" '--dbname' "${TEST_CONTAINER_POSTGRES_DB}") "${TEST_CONTAINER_ARGV_DIR}/pg_isready"
cmp <(printf '%s\0' '--username' "${TEST_CONTAINER_POSTGRES_USER}" profit_sharing) "${TEST_CONTAINER_ARGV_DIR}/createdb"
cmp <(printf '%s\0' '--username' "${TEST_CONTAINER_POSTGRES_USER}" '--dbname' profit_sharing '--command' 'SELECT 1') "${TEST_CONTAINER_ARGV_DIR}/psql"
cmp <(printf '%s\0' -a "${TEST_CONTAINER_REDIS_PASSWORD}" ping) "${TEST_CONTAINER_ARGV_DIR}/redis-cli"
for volume in "${PROD_POSTGRES_VOLUME}" "${PROD_REDIS_VOLUME}" "${PROD_MINIO_VOLUME}"; do [[ -e "${TEST_VOLUME_DIR}/${volume}" ]]; done
[[ -e "${TEST_VOLUME_DIR}/${unrelated_volume}" ]]
[[ "$(find "${TEST_VOLUME_DIR}" -type f | wc -l)" -eq 4 ]]
[[ ! -s "${TEST_VOLUME_RM_LOG}" ]]

: >"${TEST_LOG}"
set +e
TEST_FAIL_MIGRATE=yes bash "${fixture}/hack/prod-remote-deploy.sh" hot-deploy >"${tmp}/prod-hot-fail.out" 2>&1
status=$?
set -e
[[ ${status} -eq 41 ]]
rg -q -F -- '--profile tools run --rm athena-migrate' "${TEST_LOG}"
if rg -q -F -- '--force-recreate athena-a' "${TEST_LOG}"; then exit 1; fi
for volume in "${PROD_POSTGRES_VOLUME}" "${PROD_REDIS_VOLUME}" "${PROD_MINIO_VOLUME}"; do [[ -e "${TEST_VOLUME_DIR}/${volume}" ]]; done
[[ -e "${TEST_VOLUME_DIR}/${unrelated_volume}" ]]
[[ ! -s "${TEST_VOLUME_RM_LOG}" ]]

: >"${TEST_LOG}"
set +e
TEST_FAIL_VOLUME_RM=yes bash "${fixture}/hack/prod-remote-deploy.sh" destroy >"${tmp}/prod-destroy-fail.out" 2>&1
status=$?
set -e
[[ ${status} -eq 42 ]]
[[ -e "${TEST_VOLUME_DIR}/${PROD_POSTGRES_VOLUME}" ]]
[[ -e "${TEST_VOLUME_DIR}/${unrelated_volume}" ]]
cmp <(printf '%s\0' "${PROD_POSTGRES_VOLUME}") "${TEST_VOLUME_RM_LOG}"

: >"${TEST_LOG}"
: >"${TEST_VOLUME_RM_LOG}"
bash "${fixture}/hack/prod-remote-deploy.sh" destroy >"${tmp}/prod-destroy.out" 2>&1
for volume in "${PROD_POSTGRES_VOLUME}" "${PROD_REDIS_VOLUME}" "${PROD_MINIO_VOLUME}"; do [[ ! -e "${TEST_VOLUME_DIR}/${volume}" ]]; done
[[ -e "${TEST_VOLUME_DIR}/${unrelated_volume}" ]]
[[ "$(find "${TEST_VOLUME_DIR}" -type f | wc -l)" -eq 1 ]]
cmp <(printf '%s\0' "${PROD_POSTGRES_VOLUME}" "${PROD_REDIS_VOLUME}" "${PROD_MINIO_VOLUME}") "${TEST_VOLUME_RM_LOG}"
rg -q -F 'down --remove-orphans' "${TEST_LOG}"

: >"${TEST_LOG}"
set +e
TEST_FAIL_MIGRATE=yes bash "${fixture}/hack/prod-remote-deploy.sh" deploy >"${tmp}/prod-deploy-fail.out" 2>&1
status=$?
set -e
[[ ${status} -eq 41 ]]
rg -q -F -- '--profile tools run --rm athena-migrate' "${TEST_LOG}"
[[ "$(tail -n 1 "${TEST_LOG}")" == *'athena-migrate'* ]]
[[ -e "${TEST_VOLUME_DIR}/${unrelated_volume}" ]]
cmp <(printf '%s\0' "${PROD_POSTGRES_VOLUME}" "${PROD_REDIS_VOLUME}" "${PROD_MINIO_VOLUME}") "${TEST_VOLUME_RM_LOG}"

for kind in transaction swap; do
  if [[ "$kind" == transaction ]]; then prefix=BSC_INDEXER; script=deploy-bsc-transaction-indexer.sh; rpc=ATHENA_BSC_INBOUND_NODE_RPC_URL; else prefix=BSC_SWAP_INDEXER; script=deploy-bsc-swap-indexer.sh; rpc=ATHENA_BSC_SWAP_NODE_RPC_URL; fi
  envfile="${tmp}/${kind}.env"
  printf 'REMOTE_HOST=example\nPOSTGRES_PASSWORD=password\n%s=https://example.com\n' "$rpc" >"${envfile}"
  app_dir="${tmp}/${kind} single'quote \$(touch ${tmp}/injected)"
  image="${kind}'\$(touch ${tmp}/injected)"
  mkdir -p "${app_dir}"
  : >"${TEST_LOG}"
  env "${prefix}_ENV_FILE=${envfile}" "${prefix}_REMOTE_APP_DIR=${app_dir}" "${prefix}_IMAGE=${image}" \
    bash "${fixture}/hack/${script}" >"${tmp}/${kind}.out" 2>&1
  rg -q -F "docker 1 ${app_dir} ${image} compose -f docker-compose.yml --env-file .env up -d --force-recreate" "${TEST_LOG}"
  [[ ! -e "${tmp}/injected" ]]
  : >"${TEST_LOG}"
  if env TEST_FAIL_UP=yes "${prefix}_ENV_FILE=${envfile}" "${prefix}_REMOTE_APP_DIR=${app_dir}" "${prefix}_IMAGE=${image}" \
    bash "${fixture}/hack/${script}" >"${tmp}/${kind}-fail.out" 2>&1; then exit 1; fi
  rg -q -F 'compose -f docker-compose.yml --env-file .env ps' "${TEST_LOG}"
  rg -q -F 'compose -f docker-compose.yml --env-file .env logs --tail=120' "${TEST_LOG}"
done

: >"${TEST_LOG}"
ETHERSCAN_GATEWAY_IPS=example ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN=token \
  bash "${fixture}/hack/deploy-etherscan-gateway.sh" >"${tmp}/gateway.out" 2>&1
rg -q -F 'systemctl enable --now athena-etherscan-gateway' "${TEST_LOG}"
if TEST_FAIL_SERVICE=yes ETHERSCAN_GATEWAY_IPS=example ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN=token \
  bash "${fixture}/hack/deploy-etherscan-gateway.sh" >"${tmp}/gateway-fail.out" 2>&1; then exit 1; fi
rg -q -F 'journalctl -u athena-etherscan-gateway' "${TEST_LOG}"
echo 'deploy-scripts tests passed'
