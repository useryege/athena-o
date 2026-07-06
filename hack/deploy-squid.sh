#!/usr/bin/env bash

set -euo pipefail

SQUID_HOST="${SQUID_HOST:-root@47.245.183.140}"
SQUID_PORT="6776"
SQUID_USER="${SQUID_USER:-athena_probe}"
SQUID_PASSWORD="${SQUID_PASSWORD:-}"
SQUID_PUBLIC_HOST="${SQUID_PUBLIC_HOST:-${SQUID_HOST#*@}}"
SQUID_SSH_ARGS="${SQUID_SSH_ARGS:-}"

if [[ ! "${SQUID_USER}" =~ ^[A-Za-z0-9_.-]+$ ]]; then
	printf 'SQUID_USER may only contain letters, numbers, dots, underscores, and hyphens, got %q\n' "${SQUID_USER}" >&2
	exit 1
fi

if [[ -z "${SQUID_PASSWORD}" ]]; then
	printf 'SQUID_PASSWORD is required. Load .env before running this script.\n' >&2
	exit 1
fi

if [[ ! "${SQUID_PASSWORD}" =~ ^[A-Za-z0-9_.=-]+$ ]]; then
	printf 'SQUID_PASSWORD may only contain letters, numbers, dots, underscores, equals signs, and hyphens.\n' >&2
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

if [[ -n "${SQUID_SSH_ARGS}" ]]; then
	# shellcheck disable=SC2206
	extra_ssh_args=(${SQUID_SSH_ARGS})
	ssh_args+=("${extra_ssh_args[@]}")
fi

{
	printf 'export SQUID_PORT_B64=%q\n' "$(b64 "${SQUID_PORT}")"
	printf 'export SQUID_USER_B64=%q\n' "$(b64 "${SQUID_USER}")"
	printf 'export SQUID_PASSWORD_B64=%q\n' "$(b64 "${SQUID_PASSWORD}")"
	printf 'export SQUID_PUBLIC_HOST_B64=%q\n' "$(b64 "${SQUID_PUBLIC_HOST}")"
	cat <<'REMOTE_SCRIPT'
set -euo pipefail

decode() {
	printf '%s' "$1" | base64 -d
}

SQUID_PORT="$(decode "${SQUID_PORT_B64}")"
SQUID_USER="$(decode "${SQUID_USER_B64}")"
SQUID_PASSWORD="$(decode "${SQUID_PASSWORD_B64}")"
SQUID_PUBLIC_HOST="$(decode "${SQUID_PUBLIC_HOST_B64}")"

SQUID_CONFIG_FILE="/etc/squid/squid.conf"
SQUID_PASSWORD_FILE="/etc/squid/athena-passwd"

if [[ "$(id -u)" != "0" ]]; then
	printf 'This installer must run as root on the target VPS.\n' >&2
	exit 1
fi

if ! command -v apt-get >/dev/null 2>&1; then
	printf 'Only Ubuntu/Debian apt-based systems are supported by this installer.\n' >&2
	exit 1
fi

if [[ ! "${SQUID_PORT}" =~ ^[0-9]+$ ]] || (( SQUID_PORT < 1 || SQUID_PORT > 65535 )); then
	printf 'Invalid SQUID_PORT on remote host: %q\n' "${SQUID_PORT}" >&2
	exit 1
fi

if [[ ! "${SQUID_USER}" =~ ^[A-Za-z0-9_.-]+$ ]]; then
	printf 'Invalid SQUID_USER on remote host: %q\n' "${SQUID_USER}" >&2
	exit 1
fi

if [[ -z "${SQUID_PASSWORD}" ]]; then
	printf 'SQUID_PASSWORD was empty on remote host.\n' >&2
	exit 1
fi

apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y squid apache2-utils

BASIC_AUTH_HELPER=""
for candidate in /usr/lib/squid/basic_ncsa_auth /usr/lib/squid3/basic_ncsa_auth; do
	if [[ -x "${candidate}" ]]; then
		BASIC_AUTH_HELPER="${candidate}"
		break
	fi
done

if [[ -z "${BASIC_AUTH_HELPER}" ]]; then
	printf 'Could not find Squid basic_ncsa_auth helper.\n' >&2
	exit 1
fi

install -d -m 0750 -o proxy -g proxy /etc/squid
printf '%s\n' "${SQUID_PASSWORD}" | htpasswd -B -i -c "${SQUID_PASSWORD_FILE}" "${SQUID_USER}" >/dev/null
chown proxy:proxy "${SQUID_PASSWORD_FILE}"
chmod 0640 "${SQUID_PASSWORD_FILE}"

if [[ -f "${SQUID_CONFIG_FILE}" && ! -f "${SQUID_CONFIG_FILE}.athena.bak" ]]; then
	cp -a "${SQUID_CONFIG_FILE}" "${SQUID_CONFIG_FILE}.athena.bak"
fi

cat >"${SQUID_CONFIG_FILE}" <<EOF
visible_hostname athena-squid-proxy

http_port 0.0.0.0:${SQUID_PORT}

auth_param basic program ${BASIC_AUTH_HELPER} ${SQUID_PASSWORD_FILE}
auth_param basic realm Athena Squid Proxy
auth_param basic credentialsttl 2 hours

acl authenticated proxy_auth REQUIRED
acl SSL_ports port 443
acl CONNECT method CONNECT

http_access deny !authenticated
http_access deny CONNECT !SSL_ports
http_access allow authenticated CONNECT SSL_ports
http_access deny all

access_log /var/log/squid/access.log squid
cache_log /var/log/squid/cache.log
coredump_dir /var/spool/squid
EOF

squid -k parse -f "${SQUID_CONFIG_FILE}" >/dev/null

systemctl enable squid >/dev/null
systemctl restart squid

if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q '^Status: active'; then
	ufw allow "${SQUID_PORT}/tcp" >/dev/null
fi

systemctl is-active --quiet squid
if ! ss -H -ltn "sport = :${SQUID_PORT}" | grep -q .; then
	printf 'Squid is active but port %s is not listening.\n' "${SQUID_PORT}" >&2
	exit 1
fi

public_host="${SQUID_PUBLIC_HOST:-$(hostname -I | awk '{print $1}')}"
printf '\nSquid deployed successfully.\n'
printf 'Service: systemctl status squid\n'
printf 'Listen: 0.0.0.0:%s\n' "${SQUID_PORT}"
printf 'Proxy URL for probes: http://%s:<SQUID_PASSWORD>@%s:%s\n' "${SQUID_USER}" "${public_host}" "${SQUID_PORT}"
printf 'Remember to allow TCP %s from 0.0.0.0/0 in the cloud security group.\n' "${SQUID_PORT}"
REMOTE_SCRIPT
} | ssh "${ssh_args[@]}" "${SQUID_HOST}" 'bash -s'
