#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

ETHERSCAN_GATEWAY_ENV_FILE="${ETHERSCAN_GATEWAY_ENV_FILE:-${REPO_ROOT}/.env}"
provided_host="${ETHERSCAN_GATEWAY_HOST:-}"
provided_auth_token="${ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN:-}"

if [[ -f "${ETHERSCAN_GATEWAY_ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${ETHERSCAN_GATEWAY_ENV_FILE}"
  set +a
fi

if [[ -n "${provided_host}" ]]; then
  ETHERSCAN_GATEWAY_HOST="${provided_host}"
fi
if [[ -n "${provided_auth_token}" ]]; then
  ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN="${provided_auth_token}"
fi

ETHERSCAN_GATEWAY_HOST="${ETHERSCAN_GATEWAY_HOST:-root@47.245.183.140}"
ETHERSCAN_GATEWAY_LISTEN_ADDRESS="0.0.0.0:6776"
ETHERSCAN_GATEWAY_TIMEOUT="30s"
REMOTE_BINARY="/usr/local/bin/athena-etherscan-gateway"
REMOTE_ENV_FILE="/etc/athena/etherscan-gateway.env"
REMOTE_SERVICE_FILE="/etc/systemd/system/athena-etherscan-gateway.service"
SERVICE_NAME="athena-etherscan-gateway"

command -v openssl >/dev/null
command -v scp >/dev/null
command -v ssh >/dev/null

if [[ -n "${ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN:-}" ]]; then
  auth_token="${ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN}"
  token_source="provided"
else
  auth_token="$(openssl rand -hex 32)"
  token_source="generated"
fi

token_fingerprint() {
  local token="$1"
  if ((${#token} <= 8)); then
    printf '<redacted>'
    return
  fi
  printf '%s...%s' "${token:0:4}" "${token: -4}"
}

cleanup_files=()
cleanup() {
  local file
  for file in "${cleanup_files[@]}"; do
    rm -f "${file}"
  done
}
trap cleanup EXIT

if [[ -z "${auth_token}" ]]; then
  echo "ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN must not be empty." >&2
  exit 1
fi

cd "${REPO_ROOT}"

echo "Building Athena for linux/amd64..."
GOOS=linux GOARCH=amd64 STATIC_BUILD=true make athena-all

if [[ ! -f "${REPO_ROOT}/dist/athena" ]]; then
  echo "Build output not found: ${REPO_ROOT}/dist/athena" >&2
  exit 1
fi

env_tmp="$(mktemp)"
cleanup_files+=("${env_tmp}")
chmod 600 "${env_tmp}"
cat >"${env_tmp}" <<EOF
ATHENA_ETHERSCAN_GATEWAY_LISTEN_ADDRESS=${ETHERSCAN_GATEWAY_LISTEN_ADDRESS}
ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN=${auth_token}
ATHENA_ETHERSCAN_GATEWAY_TIMEOUT=${ETHERSCAN_GATEWAY_TIMEOUT}
EOF

remote_tmp_binary="/tmp/athena-etherscan-gateway.$$"
remote_tmp_env="/tmp/etherscan-gateway.env.$$"

echo "Uploading gateway binary to ${ETHERSCAN_GATEWAY_HOST}..."
scp "${REPO_ROOT}/dist/athena" "${ETHERSCAN_GATEWAY_HOST}:${remote_tmp_binary}"

echo "Uploading gateway environment file to ${ETHERSCAN_GATEWAY_HOST}..."
scp "${env_tmp}" "${ETHERSCAN_GATEWAY_HOST}:${remote_tmp_env}"

echo "Installing systemd service on ${ETHERSCAN_GATEWAY_HOST}..."
ssh "${ETHERSCAN_GATEWAY_HOST}" "set -euo pipefail
systemctl stop '${SERVICE_NAME}' >/dev/null 2>&1 || true
install -m 0755 '${remote_tmp_binary}' '${REMOTE_BINARY}'
rm -f '${remote_tmp_binary}'
mkdir -p /etc/athena
install -m 0600 -o root -g root '${remote_tmp_env}' '${REMOTE_ENV_FILE}'
rm -f '${remote_tmp_env}'
cat >'${REMOTE_SERVICE_FILE}' <<'UNIT'
[Unit]
Description=Athena Etherscan Gateway
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/athena/etherscan-gateway.env
ExecStart=/usr/local/bin/athena-etherscan-gateway
Restart=always
RestartSec=5
KillSignal=SIGTERM

[Install]
WantedBy=multi-user.target
UNIT
systemctl daemon-reload
systemctl enable --now '${SERVICE_NAME}'
"

echo "Checking remote service status..."
active_status="$(ssh "${ETHERSCAN_GATEWAY_HOST}" "systemctl is-active '${SERVICE_NAME}'")"
enabled_status="$(ssh "${ETHERSCAN_GATEWAY_HOST}" "systemctl is-enabled '${SERVICE_NAME}'")"
listen_status="$(ssh "${ETHERSCAN_GATEWAY_HOST}" "ss -H -ltnp 'sport = :6776' || true")"

if [[ "${active_status}" != "active" ]]; then
  echo "Service is not active: ${active_status}" >&2
  ssh "${ETHERSCAN_GATEWAY_HOST}" "journalctl -u '${SERVICE_NAME}' -n 80 --no-pager" >&2 || true
  exit 1
fi

if [[ -z "${listen_status}" ]]; then
  echo "Service is active, but port 6776 is not listening." >&2
  ssh "${ETHERSCAN_GATEWAY_HOST}" "journalctl -u '${SERVICE_NAME}' -n 80 --no-pager" >&2 || true
  exit 1
fi

echo
echo "Etherscan Gateway deployed successfully."
echo "Host: ${ETHERSCAN_GATEWAY_HOST}"
echo "Service: ${SERVICE_NAME}"
echo "Active: ${active_status}"
echo "Enabled: ${enabled_status}"
echo "Listen address: ${ETHERSCAN_GATEWAY_LISTEN_ADDRESS}"
echo "Auth token: ${token_source} ($(token_fingerprint "${auth_token}"))"
echo "Remember to allow inbound TCP 6776/6776 in the Alibaba Cloud security group."
