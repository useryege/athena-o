#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
DEFAULT_RUN_EXCLUDE="application-ingestor-bsc,application-ingestor-eth,application-temporal-eth"
MANAGED_INFRA_SERVICES=(postgres redis temporal)
managed_infra_names=()
managed_infra_pids=()
goreman_pid=""
cleanup_started=false

run_exclude_value() {
	if [[ "${ATHENA_RUN_EXCLUDE+x}" == "x" ]]; then
		printf '%s' "${ATHENA_RUN_EXCLUDE}"
		return
	fi
	printf '%s' "${DEFAULT_RUN_EXCLUDE}"
}

split_excludes() {
	local value="$1"
	local item

	[[ -n "${value}" ]] || return
	while IFS= read -r item; do
		item="$(printf '%s' "${item}" | xargs)"
		[[ -n "${item}" ]] || continue
		printf '%s\n' "${item}"
	done < <(tr ',' '\n' <<<"${value}")
}

is_excluded_service() {
	local name="$1"
	local item

	while IFS= read -r item; do
		[[ "${item}" == "${name}" ]] && return 0
	done < <(split_excludes "$(run_exclude_value)")

	return 1
}

is_managed_infra_service() {
	local name="$1"
	local service

	for service in "${MANAGED_INFRA_SERVICES[@]}"; do
		[[ "${service}" == "${name}" ]] && return 0
	done

	return 1
}

write_filtered_procfile() {
	local source="$1"
	local target="$2"
	local line name

	: >"${target}"
	while IFS= read -r line || [[ -n "${line}" ]]; do
		if [[ "${line}" =~ ^[[:space:]]*([A-Za-z0-9_-]+): ]]; then
			name="${BASH_REMATCH[1]}"
			if is_excluded_service "${name}"; then
				continue
			fi
			if is_managed_infra_service "${name}"; then
				continue
			fi
		fi
		printf '%s\n' "${line}" >>"${target}"
	done <"${source}"
}

procfile_service_command() {
	local source="$1"
	local service="$2"
	local line

	while IFS= read -r line || [[ -n "${line}" ]]; do
		if [[ "${line}" =~ ^[[:space:]]*${service}:[[:space:]]*(.*)$ ]]; then
			printf '%s\n' "${BASH_REMATCH[1]}"
			return 0
		fi
	done <"${source}"

	return 1
}

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
		"polymarket:${ATHENA_POLYMARKET_PORT:-8092}"
		"token:${ATHENA_TOKEN_PORT:-8094}"
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

process_group_alive() {
	local pid="$1"

	kill -0 -- "-${pid}" 2>/dev/null
}

wait_for_process_group_exit() {
	local pid="$1"
	local attempts="$2"
	local attempt

	for ((attempt = 1; attempt <= attempts; attempt++)); do
		if ! process_group_alive "${pid}"; then
			wait "${pid}" 2>/dev/null || true
			return 0
		fi
		sleep 0.1
	done

	return 1
}

stop_process_group() {
	local name="$1"
	local pid="$2"
	local graceful_signal="${3:-TERM}"

	if [[ -z "${pid}" ]]; then
		return
	fi
	if ! process_group_alive "${pid}"; then
		wait "${pid}" 2>/dev/null || true
		return
	fi

	printf 'stopping %s pid=%s signal=%s\n' "${name}" "${pid}" "${graceful_signal}" >&2
	kill "-${graceful_signal}" -- "-${pid}" 2>/dev/null || true
	if wait_for_process_group_exit "${pid}" 200; then
		return
	fi

	printf '%s did not stop after %s, sending KILL pid=%s\n' "${name}" "${graceful_signal}" "${pid}" >&2
	kill -KILL -- "-${pid}" 2>/dev/null || true
	wait_for_process_group_exit "${pid}" 100 || true
}

start_managed_infra_services() {
	local procfile="$1"
	local service command pid

	for service in "${MANAGED_INFRA_SERVICES[@]}"; do
		if is_excluded_service "${service}"; then
			printf 'not starting infrastructure service=%s because it is excluded\n' "${service}" >&2
			continue
		fi

		if ! command="$(procfile_service_command "${procfile}" "${service}")"; then
			printf 'infrastructure service=%s is not present in %s; skipping\n' "${service}" "${procfile}" >&2
			continue
		fi

		printf 'starting infrastructure service=%s command=%s\n' "${service}" "${command}" >&2
		(
			cd "${REPO_ROOT}"
			exec setsid bash -lc "${command}"
		) &
		pid="$!"
		managed_infra_names+=("${service}")
		managed_infra_pids+=("${pid}")
		printf 'started infrastructure service=%s pid=%s\n' "${service}" "${pid}" >&2
	done
}

stop_managed_infra_services() {
	local index service pid

	for ((index = ${#managed_infra_pids[@]} - 1; index >= 0; index--)); do
		service="${managed_infra_names[index]}"
		pid="${managed_infra_pids[index]}"
		stop_process_group "infrastructure service=${service}" "${pid}" TERM
	done
}

start_goreman() {
	local procfile="$1"

	printf 'starting Procfile application services with goreman\n' >&2
	(
		cd "${REPO_ROOT}"
		exec setsid goreman -f "${procfile}" start
	) &
	goreman_pid="$!"
	printf 'started goreman pid=%s\n' "${goreman_pid}" >&2
}

stop_goreman() {
	if [[ -z "${goreman_pid}" ]]; then
		return
	fi
	if ! process_group_alive "${goreman_pid}"; then
		wait "${goreman_pid}" 2>/dev/null || true
		return
	fi

	printf 'stopping Procfile application services before infrastructure pid=%s\n' "${goreman_pid}" >&2
	kill -INT "${goreman_pid}" 2>/dev/null || true
	if wait_for_process_group_exit "${goreman_pid}" 200; then
		return
	fi

	printf 'goreman did not stop after INT, sending TERM to application process group pid=%s\n' "${goreman_pid}" >&2
	kill -TERM -- "-${goreman_pid}" 2>/dev/null || true
	if wait_for_process_group_exit "${goreman_pid}" 100; then
		return
	fi

	printf 'goreman application process group did not stop after TERM, sending KILL pid=%s\n' "${goreman_pid}" >&2
	kill -KILL -- "-${goreman_pid}" 2>/dev/null || true
	wait_for_process_group_exit "${goreman_pid}" 100 || true
}

cleanup() {
	local status="$?"

	if [[ "${cleanup_started}" == "true" ]]; then
		return
	fi
	cleanup_started=true
	trap - EXIT INT TERM

	stop_goreman
	stop_managed_infra_services
	rm -f "${filtered_procfile:-}"

	exit "${status}"
}

if [[ "${ATHENA_RUN_PORT_CLEANUP:-true}" != "false" ]]; then
	cleanup_athena_ports
fi

procfile="${ATHENA_PROCFILE:-Procfile}"
filtered_procfile="$(mktemp -t athena-procfile.XXXXXX)"
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

write_filtered_procfile "${procfile}" "${filtered_procfile}"
if [[ -n "$(run_exclude_value)" ]]; then
	printf 'excluding Procfile services: %s\n' "$(run_exclude_value)" >&2
fi
printf 'managing infrastructure services outside goreman: %s\n' "${MANAGED_INFRA_SERVICES[*]}" >&2

if [[ "${ATHENA_RUN_DRY_RUN:-false}" == "true" ]]; then
	cat "${filtered_procfile}"
	exit 0
fi

start_managed_infra_services "${procfile}"
start_goreman "${filtered_procfile}"

set +e
wait "${goreman_pid}"
goreman_status="$?"
set -e

exit "${goreman_status}"
