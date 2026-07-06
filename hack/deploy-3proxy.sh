#!/usr/bin/env bash

set -euo pipefail

THREEPROXY_HOST="${THREEPROXY_HOST:-root@47.245.183.140}"
THREEPROXY_PORT="6776"
THREEPROXY_USER="${THREEPROXY_USER:-athena_probe}"
THREEPROXY_PASSWORD="${THREEPROXY_PASSWORD:-}"
THREEPROXY_ALLOWED_CIDRS="0.0.0.0/0"
THREEPROXY_RELEASE_VERSION="${THREEPROXY_RELEASE_VERSION:-0.9.4}"
THREEPROXY_RELEASE_DEB_URL="${THREEPROXY_RELEASE_DEB_URL:-https://github.com/3proxy/3proxy/releases/download/${THREEPROXY_RELEASE_VERSION}/3proxy-${THREEPROXY_RELEASE_VERSION}.x86_64.deb}"
THREEPROXY_SSH_ARGS="${THREEPROXY_SSH_ARGS:-}"
THREEPROXY_PUBLIC_HOST="${THREEPROXY_PUBLIC_HOST:-${THREEPROXY_HOST#*@}}"

if [[ ! "${THREEPROXY_PORT}" =~ ^[0-9]+$ ]] || (( THREEPROXY_PORT < 1 || THREEPROXY_PORT > 65535 )); then
	printf 'THREEPROXY_PORT must be a TCP port from 1 to 65535, got %q\n' "${THREEPROXY_PORT}" >&2
	exit 1
fi

if [[ ! "${THREEPROXY_USER}" =~ ^[A-Za-z0-9_.-]+$ ]]; then
	printf 'THREEPROXY_USER may only contain letters, numbers, dots, underscores, and hyphens, got %q\n' "${THREEPROXY_USER}" >&2
	exit 1
fi

if [[ -n "${THREEPROXY_PASSWORD}" && ! "${THREEPROXY_PASSWORD}" =~ ^[A-Za-z0-9_.=-]+$ ]]; then
	printf 'THREEPROXY_PASSWORD may only contain letters, numbers, dots, underscores, equals signs, and hyphens.\n' >&2
	exit 1
fi

b64() {
	printf '%s' "$1" | base64 | tr -d '\n'
}

ssh_args=(
	-o BatchMode=yes
	-o ConnectTimeout=10
	-o StrictHostKeyChecking=accept-new
)

if [[ -n "${THREEPROXY_SSH_ARGS}" ]]; then
	# shellcheck disable=SC2206
	extra_ssh_args=(${THREEPROXY_SSH_ARGS})
	ssh_args+=("${extra_ssh_args[@]}")
fi

{
	printf 'export THREEPROXY_PORT_B64=%q\n' "$(b64 "${THREEPROXY_PORT}")"
	printf 'export THREEPROXY_USER_B64=%q\n' "$(b64 "${THREEPROXY_USER}")"
	printf 'export THREEPROXY_PASSWORD_B64=%q\n' "$(b64 "${THREEPROXY_PASSWORD}")"
	printf 'export THREEPROXY_ALLOWED_CIDRS_B64=%q\n' "$(b64 "${THREEPROXY_ALLOWED_CIDRS}")"
	printf 'export THREEPROXY_RELEASE_DEB_URL_B64=%q\n' "$(b64 "${THREEPROXY_RELEASE_DEB_URL}")"
	printf 'export THREEPROXY_PUBLIC_HOST_B64=%q\n' "$(b64 "${THREEPROXY_PUBLIC_HOST}")"
	cat <<'REMOTE_SCRIPT'
set -euo pipefail

decode() {
	printf '%s' "$1" | base64 -d
}

THREEPROXY_PORT="$(decode "${THREEPROXY_PORT_B64}")"
THREEPROXY_USER="$(decode "${THREEPROXY_USER_B64}")"
THREEPROXY_PASSWORD="$(decode "${THREEPROXY_PASSWORD_B64}")"
THREEPROXY_ALLOWED_CIDRS="$(decode "${THREEPROXY_ALLOWED_CIDRS_B64}")"
THREEPROXY_RELEASE_DEB_URL="$(decode "${THREEPROXY_RELEASE_DEB_URL_B64}")"
THREEPROXY_PUBLIC_HOST="$(decode "${THREEPROXY_PUBLIC_HOST_B64}")"
FIXED_THREEPROXY_PORT="${THREEPROXY_PORT}"
FIXED_THREEPROXY_ALLOWED_CIDRS="${THREEPROXY_ALLOWED_CIDRS}"

STATE_FILE="/root/.athena-3proxy.env"
CONFIG_DIR="/etc/3proxy"
CONFIG_FILE="${CONFIG_DIR}/3proxy.cfg"
SERVICE_FILE="/etc/systemd/system/3proxy.service"
LOG_DIR="/var/log/3proxy"

if [[ "$(id -u)" != "0" ]]; then
	printf 'This installer must run as root on the target VPS.\n' >&2
	exit 1
fi

if ! command -v apt-get >/dev/null 2>&1; then
	printf 'Only Ubuntu/Debian apt-based systems are supported by this installer.\n' >&2
	exit 1
fi

if [[ ! "${THREEPROXY_PORT}" =~ ^[0-9]+$ ]] || (( THREEPROXY_PORT < 1 || THREEPROXY_PORT > 65535 )); then
	printf 'Invalid THREEPROXY_PORT on remote host: %q\n' "${THREEPROXY_PORT}" >&2
	exit 1
fi

if [[ ! "${THREEPROXY_USER}" =~ ^[A-Za-z0-9_.-]+$ ]]; then
	printf 'Invalid THREEPROXY_USER on remote host: %q\n' "${THREEPROXY_USER}" >&2
	exit 1
fi

if [[ -f "${STATE_FILE}" ]]; then
	# shellcheck disable=SC1090
	source "${STATE_FILE}"
fi

THREEPROXY_PORT="${FIXED_THREEPROXY_PORT}"
THREEPROXY_ALLOWED_CIDRS="${FIXED_THREEPROXY_ALLOWED_CIDRS}"

if [[ -z "${THREEPROXY_PASSWORD}" ]]; then
	if [[ -n "${ATHENA_3PROXY_PASSWORD:-}" ]]; then
		THREEPROXY_PASSWORD="${ATHENA_3PROXY_PASSWORD}"
	elif command -v openssl >/dev/null 2>&1; then
		THREEPROXY_PASSWORD="$(openssl rand -hex 18)"
	else
		THREEPROXY_PASSWORD="$(od -An -N18 -tx1 /dev/urandom | tr -d ' \n')"
	fi
fi

if [[ -z "${THREEPROXY_PASSWORD}" ]]; then
	printf '3proxy password generation failed.\n' >&2
	exit 1
fi

if [[ ! "${THREEPROXY_PASSWORD}" =~ ^[A-Za-z0-9_.=-]+$ ]]; then
	printf '3proxy password may only contain letters, numbers, dots, underscores, equals signs, and hyphens.\n' >&2
	exit 1
fi

umask 077
{
	printf 'ATHENA_3PROXY_USER=%q\n' "${THREEPROXY_USER}"
	printf 'ATHENA_3PROXY_PASSWORD=%q\n' "${THREEPROXY_PASSWORD}"
	printf 'ATHENA_3PROXY_PORT=%q\n' "${THREEPROXY_PORT}"
	printf 'ATHENA_3PROXY_ALLOWED_CIDRS=%q\n' "${THREEPROXY_ALLOWED_CIDRS}"
} >"${STATE_FILE}"
chmod 600 "${STATE_FILE}"

install_3proxy_from_apt() {
	apt-get update
	DEBIAN_FRONTEND=noninteractive apt-get install -y 3proxy
}

install_3proxy_from_release_deb() {
	tmp_deb="$(mktemp /tmp/3proxy.XXXXXX.deb)"
	trap 'rm -f "${tmp_deb}"' RETURN
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "${THREEPROXY_RELEASE_DEB_URL}" -o "${tmp_deb}"
	elif command -v wget >/dev/null 2>&1; then
		wget -q "${THREEPROXY_RELEASE_DEB_URL}" -O "${tmp_deb}"
	else
		DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl
		curl -fsSL "${THREEPROXY_RELEASE_DEB_URL}" -o "${tmp_deb}"
	fi
	DEBIAN_FRONTEND=noninteractive apt-get install -y "${tmp_deb}"
}

if ! command -v 3proxy >/dev/null 2>&1; then
	if ! install_3proxy_from_apt; then
		apt-get update
		DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl
		install_3proxy_from_release_deb
	fi
fi

if ! command -v 3proxy >/dev/null 2>&1; then
	printf '3proxy binary was not installed successfully.\n' >&2
	exit 1
fi

THREEPROXY_BIN="$(command -v 3proxy)"

mkdir -p "${CONFIG_DIR}" "${LOG_DIR}"
chmod 700 "${CONFIG_DIR}"
chmod 750 "${LOG_DIR}"

allow_lines=""
while IFS= read -r cidr; do
	cidr="$(printf '%s' "${cidr}" | xargs)"
	[[ -n "${cidr}" ]] || continue
	allow_lines="${allow_lines}allow ${THREEPROXY_USER} ${cidr} * 443 HTTPS"$'\n'
done < <(tr ',' '\n' <<<"${THREEPROXY_ALLOWED_CIDRS}")

if [[ -z "${allow_lines}" ]]; then
	printf 'No allowed CIDRs were configured.\n' >&2
	exit 1
fi

cat >"${CONFIG_FILE}" <<EOF
nscache 65536
timeouts 1 5 30 60 180 1800 15 60
log ${LOG_DIR}/3proxy.log D
rotate 30
auth strong
users ${THREEPROXY_USER}:CL:${THREEPROXY_PASSWORD}
${allow_lines}deny *
proxy -n -a -p${THREEPROXY_PORT} -i0.0.0.0
flush
EOF
chmod 600 "${CONFIG_FILE}"

cat >"${SERVICE_FILE}" <<EOF
[Unit]
Description=3proxy HTTP proxy for Athena Etherscan probes
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${THREEPROXY_BIN} ${CONFIG_FILE}
ExecReload=/bin/kill -HUP \$MAINPID
Restart=on-failure
RestartSec=3s
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable 3proxy
systemctl restart 3proxy

if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q '^Status: active'; then
	while IFS= read -r cidr; do
		cidr="$(printf '%s' "${cidr}" | xargs)"
		[[ -n "${cidr}" ]] || continue
		ufw allow from "${cidr}" to any port "${THREEPROXY_PORT}" proto tcp >/dev/null
	done < <(tr ',' '\n' <<<"${THREEPROXY_ALLOWED_CIDRS}")
fi

systemctl is-active --quiet 3proxy
if ! ss -H -ltn "sport = :${THREEPROXY_PORT}" | grep -q .; then
	printf '3proxy is active but port %s is not listening.\n' "${THREEPROXY_PORT}" >&2
	exit 1
fi

public_host="${THREEPROXY_PUBLIC_HOST:-$(hostname -I | awk '{print $1}')}"
printf '\n3proxy deployed successfully.\n'
printf 'Service: systemctl status 3proxy\n'
printf 'Allowed CIDRs: %s\n' "${THREEPROXY_ALLOWED_CIDRS}"
printf 'Proxy URL for probes: http://%s:%s@%s:%s\n' "${THREEPROXY_USER}" "${THREEPROXY_PASSWORD}" "${public_host}" "${THREEPROXY_PORT}"
printf 'Remember to allow TCP %s from %s in the cloud security group.\n' "${THREEPROXY_PORT}" "${THREEPROXY_ALLOWED_CIDRS}"
REMOTE_SCRIPT
} | ssh "${ssh_args[@]}" "${THREEPROXY_HOST}" 'bash -s'
