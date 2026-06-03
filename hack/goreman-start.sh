#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

listening_pids() {
	local port="$1"
	local output pids

	if command -v ss >/dev/null 2>&1; then
		output="$(ss -H -ltnp "sport = :${port}" 2>/dev/null || true)"
		if [[ -n "${output}" ]]; then
			pids="$(printf '%s\n' "${output}" | grep -oE 'pid=[0-9]+' | cut -d= -f2 | sort -u || true)"
			if [[ -n "${pids}" ]]; then
				printf '%s\n' "${pids}"
				return
			fi
		fi
	fi

	if command -v lsof >/dev/null 2>&1; then
		lsof -nP -t -iTCP:"${port}" -sTCP:LISTEN 2>/dev/null | sort -u || true
	fi
}

process_cmd() {
	local pid="$1"

	if [[ -r "/proc/${pid}/cmdline" ]]; then
		tr '\0' ' ' <"/proc/${pid}/cmdline" | sed 's/[[:space:]]*$//'
		return
	fi

	ps -p "${pid}" -o command= 2>/dev/null || true
}

is_current_repo_athena_process() {
	local pid="$1"
	local cwd env

	[[ -r "/proc/${pid}/environ" ]] || return 1
	env="$(tr '\0' '\n' <"/proc/${pid}/environ")"
	grep -qE '^ATHENA_BINARY_NAME=athena-.+' <<<"${env}" || return 1

	cwd="$(readlink -f "/proc/${pid}/cwd" 2>/dev/null || true)"
	[[ "${cwd}" == "${REPO_ROOT}" ]]
}

pid_list_contains() {
	local pid="$1"
	local pids="$2"
	local item

	while IFS= read -r item; do
		[[ "${item}" == "${pid}" ]] && return 0
	done <<<"${pids}"

	return 1
}

wait_for_pid_port_release() {
	local port="$1"
	local pid="$2"
	local attempt

	for attempt in {1..20}; do
		if ! pid_list_contains "${pid}" "$(listening_pids "${port}")"; then
			return 0
		fi
		sleep 0.1
	done

	return 1
}

stop_stale_process() {
	local name="$1"
	local port="$2"
	local pid="$3"
	local cmd

	cmd="$(process_cmd "${pid}")"
	printf 'stopping stale Athena process service=%s port=%s pid=%s cmd=%s\n' "${name}" "${port}" "${pid}" "${cmd}"

	kill -TERM "${pid}" 2>/dev/null || true
	if wait_for_pid_port_release "${port}" "${pid}"; then
		return
	fi

	printf 'Athena process did not stop after TERM, sending KILL service=%s port=%s pid=%s\n' "${name}" "${port}" "${pid}"
	kill -KILL "${pid}" 2>/dev/null || true
	wait_for_pid_port_release "${port}" "${pid}" || {
		printf 'failed to release port %s after killing pid %s\n' "${port}" "${pid}" >&2
		exit 1
	}
}

cleanup_athena_ports() {
	local ports=(
		"api-server:${ATHENA_SERVER_PORT:-8080}"
		"application:${ATHENA_APPLICATION_PORT:-8082}"
		"worm:${ATHENA_WORM_PORT:-8084}"
		"notification:${ATHENA_NOTIFICATION_PORT:-8086}"
		"wallet:${ATHENA_WALLET_PORT:-8088}"
		"solidity:${ATHENA_SOLIDITY_PORT:-8090}"
		"polymarket:${ATHENA_POLYMARKET_PORT:-8092}"
	)
	local entry name port pid pids cmd

	for entry in "${ports[@]}"; do
		name="${entry%%:*}"
		port="${entry#*:}"
		pids="$(listening_pids "${port}")"
		[[ -n "${pids}" ]] || continue

		while IFS= read -r pid; do
			[[ -n "${pid}" ]] || continue

			if is_current_repo_athena_process "${pid}"; then
				stop_stale_process "${name}" "${port}" "${pid}"
				continue
			fi

			cmd="$(process_cmd "${pid}")"
			printf 'port %s for %s is already in use by a non-Athena process; not killing it.\n' "${port}" "${name}" >&2
			printf 'pid=%s cmd=%s\n' "${pid}" "${cmd}" >&2
			printf 'Stop that process manually, change the corresponding ATHENA_*_PORT, or run ATHENA_RUN_PORT_CLEANUP=false make run to skip this check.\n' >&2
			exit 1
		done <<<"${pids}"
	done
}

if [[ "${ATHENA_RUN_PORT_CLEANUP:-true}" != "false" ]]; then
	cleanup_athena_ports
fi

exec goreman start
