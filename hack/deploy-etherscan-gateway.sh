#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

ETHERSCAN_GATEWAY_ENV_FILE="${ETHERSCAN_GATEWAY_ENV_FILE:-${REPO_ROOT}/.env}"
provided_ips="${ETHERSCAN_GATEWAY_IPS:-}"
provided_auth_token="${ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN:-}"

if [[ -f "${ETHERSCAN_GATEWAY_ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${ETHERSCAN_GATEWAY_ENV_FILE}"
  set +a
fi

if [[ -n "${provided_ips}" ]]; then
  ETHERSCAN_GATEWAY_IPS="${provided_ips}"
fi
if [[ -n "${provided_auth_token}" ]]; then
  ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN="${provided_auth_token}"
fi

ETHERSCAN_GATEWAY_LISTEN_ADDRESS="0.0.0.0:6776"
ETHERSCAN_GATEWAY_TIMEOUT="30s"
REMOTE_BINARY="/usr/local/bin/athena-etherscan-gateway"
REMOTE_ENV_FILE="/etc/athena/etherscan-gateway.env"
REMOTE_SERVICE_FILE="/etc/systemd/system/athena-etherscan-gateway.service"
SERVICE_NAME="athena-etherscan-gateway"

command -v openssl >/dev/null
command -v scp >/dev/null
command -v ssh >/dev/null

if [[ -z "${ETHERSCAN_GATEWAY_IPS:-}" ]]; then
  echo "ETHERSCAN_GATEWAY_IPS is required. Set it in ${ETHERSCAN_GATEWAY_ENV_FILE} or export it before running this script." >&2
  exit 1
fi

read -r -a gateway_ips <<<"${ETHERSCAN_GATEWAY_IPS}"

if ((${#gateway_ips[@]} == 0)); then
  echo "ETHERSCAN_GATEWAY_IPS must contain at least one IP address." >&2
  exit 1
fi

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
GOOS=linux GOARCH=amd64 STATIC_BUILD=true make athena-etherscan-gateway

if [[ ! -f "${REPO_ROOT}/dist/athena-etherscan-gateway" ]]; then
  echo "Build output not found: ${REPO_ROOT}/dist/athena-etherscan-gateway" >&2
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

deployed_ips=()

for gateway_ip in "${gateway_ips[@]}"; do
  gateway_host="root@${gateway_ip}"
  remote_tmp_binary="/tmp/athena-etherscan-gateway.$$"
  remote_tmp_env="/tmp/etherscan-gateway.env.$$"

  echo
  echo "Deploying Etherscan Gateway to ${gateway_host}..."

  echo "Uploading gateway binary to ${gateway_host}..."
  scp -C "${REPO_ROOT}/dist/athena-etherscan-gateway" "${gateway_host}:${remote_tmp_binary}"

  echo "Uploading gateway environment file to ${gateway_host}..."
  scp "${env_tmp}" "${gateway_host}:${remote_tmp_env}"

  echo "Installing systemd service on ${gateway_host}..."
  ssh "${gateway_host}" "set -euo pipefail
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

  echo "Checking remote service status on ${gateway_host}..."
  active_status="$(ssh "${gateway_host}" "systemctl is-active '${SERVICE_NAME}'" || true)"
  enabled_status="$(ssh "${gateway_host}" "systemctl is-enabled '${SERVICE_NAME}'" || true)"
  listen_status="$(ssh "${gateway_host}" "ss -H -ltnp 'sport = :6776' || true" || true)"

  if [[ "${active_status}" != "active" ]]; then
    echo "Service is not active on ${gateway_host}: ${active_status:-unknown}" >&2
    ssh "${gateway_host}" "journalctl -u '${SERVICE_NAME}' -n 80 --no-pager" >&2 || true
    exit 1
  fi

  if [[ "${enabled_status}" != "enabled" ]]; then
    echo "Service is not enabled on ${gateway_host}: ${enabled_status:-unknown}" >&2
    ssh "${gateway_host}" "journalctl -u '${SERVICE_NAME}' -n 80 --no-pager" >&2 || true
    exit 1
  fi

  if [[ -z "${listen_status}" ]]; then
    echo "Service is active on ${gateway_host}, but port 6776 is not listening." >&2
    ssh "${gateway_host}" "journalctl -u '${SERVICE_NAME}' -n 80 --no-pager" >&2 || true
    exit 1
  fi

  echo "Etherscan Gateway deployed successfully on ${gateway_host}."
  echo "Active: ${active_status}"
  echo "Enabled: ${enabled_status}"
  deployed_ips+=("${gateway_ip}")
done

echo
echo "Etherscan Gateway deployed successfully to all hosts."
echo "IPs: ${deployed_ips[*]}"
echo "Service: ${SERVICE_NAME}"
echo "Listen address: ${ETHERSCAN_GATEWAY_LISTEN_ADDRESS}"
echo "Auth token: ${token_source} ($(token_fingerprint "${auth_token}"))"
echo "Remember to allow inbound TCP 6776/6776 in each Alibaba Cloud security group."
