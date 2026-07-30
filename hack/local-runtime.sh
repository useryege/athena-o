#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
STATE_DIR="${REPO_ROOT}/.run/athena-local-runtime"
SUPERVISOR_STATE_FILE="${STATE_DIR}/supervisor.state"
FILTERED_PROCFILE="${STATE_DIR}/Procfile"
POSTGRES_CONTAINER="athena-postgres"
REDIS_CONTAINER="athena-redis"
POSTGRES_VOLUME="athena-local-postgres-data"
REDIS_VOLUME="athena-local-redis-data"
RESOURCE_OWNER_LABEL="io.athena.local-runtime"
RESOURCE_OWNER_VALUE="athena"
RESOURCE_COMPONENT_LABEL="io.athena.component"
DEFAULT_RUN_EXCLUDE=""
goreman_pid=""
cleanup_started=false

coverage_dirs=(
	"/tmp/coverage/athena-worm"
	"/tmp/coverage/athena-worm-poly"
	"/tmp/coverage/athena-pred-poly"
	"/tmp/coverage/athena-polymarket"
	"/tmp/coverage/athena-token-chain-processor"
	"/tmp/coverage/athena-token-swap-processor"
	"/tmp/coverage/athena-token-scheduler"
	"/tmp/coverage/athena-token-collector-ave"
	"/tmp/coverage/athena-token-collector-chain-state"
	"/tmp/coverage/athena-token-collector-wallet-asset-state"
	"/tmp/coverage/athena-token-collector-simulation-result"
	"/tmp/coverage/athena-token-collector-contract-code-source"
	"/tmp/coverage/athena-token-collector-wallet-normal-transactions"
	"/tmp/coverage/athena-token-report-builder"
	"/tmp/coverage/athena-token-selector"
	"/tmp/coverage/athena-token-api"
	"/tmp/coverage/athena-etherscan-manager"
	"/tmp/coverage/athena-notification"
	"/tmp/coverage/athena-wallet"
	"/tmp/coverage/api-server"
)

configure_token_node_ws_proxy() {
	unset http_proxy https_proxy all_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY

	if [[ "${ATHENA_TOKEN_NODE_WS_PROXY_URL+x}" == "x" ]]; then
		export ATHENA_TOKEN_NODE_WS_PROXY_URL
		if [[ -n "${ATHENA_TOKEN_NODE_WS_PROXY_URL}" ]]; then
			printf 'using configured Token EVM WebSocket proxy: %s\n' "${ATHENA_TOKEN_NODE_WS_PROXY_URL}" >&2
		else
			printf 'Token EVM WebSocket proxy explicitly disabled\n' >&2
		fi
		return
	fi

	if [[ -z "${WSL_DISTRO_NAME:-}" && -z "${WSL_INTEROP:-}" ]] &&
		! grep -qi microsoft /proc/sys/kernel/osrelease 2>/dev/null; then
		return
	fi
	if ! command -v ip >/dev/null 2>&1; then
		printf 'cannot configure Token EVM WebSocket proxy: ip command is unavailable\n' >&2
		exit 1
	fi

	local wsl_host
	wsl_host="$(ip route show default | awk 'NR == 1 { print $3 }')"
	if [[ -z "${wsl_host}" ]]; then
		printf 'cannot configure Token EVM WebSocket proxy: WSL default gateway was not found\n' >&2
		exit 1
	fi

	ATHENA_TOKEN_NODE_WS_PROXY_URL="http://${wsl_host}:10809"
	export ATHENA_TOKEN_NODE_WS_PROXY_URL
	printf 'using default WSL Token EVM WebSocket proxy: %s\n' "${ATHENA_TOKEN_NODE_WS_PROXY_URL}" >&2
}

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
		fi
		printf '%s\n' "${line}" >>"${target}"
	done <"${source}"
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
		return 1
	}
}

cleanup_athena_ports() {
	local mode="${1:-start}"
	local ports=(
		"api-server:${ATHENA_SERVER_PORT:-8080}"
		"worm:${ATHENA_WORM_PORT:-8084}"
		"notification:${ATHENA_NOTIFICATION_PORT:-8086}"
		"wallet:${ATHENA_WALLET_PORT:-8088}"
		"polymarket:${ATHENA_POLYMARKET_PORT:-8092}"
		"token-api:${ATHENA_TOKEN_API_PORT:-8096}"
		"etherscan-manager:${ATHENA_ETHERSCAN_MANAGER_PORT:-8100}"
		"token-chain-processor-health:8110"
		"token-swap-processor-health:8111"
		"token-scheduler-health:8112"
		"token-collector-ave-health:8113"
		"token-collector-chain-state-health:8114"
		"token-collector-wallet-asset-state-health:8115"
		"token-collector-simulation-result-health:8116"
		"token-collector-contract-code-source-health:8117"
		"token-report-builder-health:8118"
		"token-selector-health:8119"
		"token-collector-wallet-normal-transactions-health:8120"
	)
	local entry name port pid pids cmd
	local status=0

	for entry in "${ports[@]}"; do
		name="${entry%%:*}"
		port="${entry#*:}"
		pids="$(listening_pids "${port}")"
		[[ -n "${pids}" ]] || continue

		while IFS= read -r pid; do
			[[ -n "${pid}" ]] || continue

			if is_current_repo_athena_process "${pid}"; then
				stop_stale_process "${name}" "${port}" "${pid}" || status=1
				continue
			fi

			cmd="$(process_cmd "${pid}")"
			if [[ "${mode}" == "stop" ]]; then
				printf 'not stopping non-Athena process on port %s for %s: pid=%s cmd=%s\n' "${port}" "${name}" "${pid}" "${cmd}" >&2
				continue
			fi
			printf 'port %s for %s is already in use by a non-Athena process; not killing it.\n' "${port}" "${name}" >&2
			printf 'pid=%s cmd=%s\n' "${pid}" "${cmd}" >&2
			printf 'Stop that process manually, change the corresponding ATHENA_*_PORT, or run ATHENA_RUN_PORT_CLEANUP=false make run to skip this check.\n' >&2
			return 1
		done <<<"${pids}"
	done

	return "${status}"
}

runtime_session_pids() {
	local session_id="$1"

	ps -eo pid=,sid= | awk -v session_id="${session_id}" '$2 == session_id { print $1 }'
}

runtime_session_alive() {
	local session_id="$1"

	[[ -n "$(runtime_session_pids "${session_id}")" ]]
}

signal_runtime_session() {
	local session_id="$1"
	local signal="$2"
	local pgid

	while read -r pgid; do
		[[ -n "${pgid}" ]] || continue
		kill "-${signal}" -- "-${pgid}" 2>/dev/null || true
	done < <(ps -eo pgid=,sid= |
		awk -v session_id="${session_id}" '$2 == session_id { print $1 }' |
		sort -nu)
}

wait_for_runtime_session_exit() {
	local session_id="$1"
	local attempts="$2"
	local attempt

	for ((attempt = 1; attempt <= attempts; attempt++)); do
		if ! runtime_session_alive "${session_id}"; then
			wait "${session_id}" 2>/dev/null || true
			return 0
		fi
		sleep 0.1
	done

	return 1
}

stop_goreman() {
	if [[ -z "${goreman_pid}" ]]; then
		return
	fi
	if ! runtime_session_alive "${goreman_pid}"; then
		wait "${goreman_pid}" 2>/dev/null || true
		return
	fi

	printf 'stopping Procfile services pid=%s signal=INT\n' "${goreman_pid}" >&2
	kill -INT "${goreman_pid}" 2>/dev/null || true
	if wait_for_runtime_session_exit "${goreman_pid}" 200; then
		return
	fi

	printf 'goreman did not stop after INT, sending TERM to runtime session pid=%s\n' "${goreman_pid}" >&2
	signal_runtime_session "${goreman_pid}" "TERM"
	if wait_for_runtime_session_exit "${goreman_pid}" 100; then
		return
	fi

	printf 'goreman runtime session did not stop after TERM, sending KILL pid=%s\n' "${goreman_pid}" >&2
	signal_runtime_session "${goreman_pid}" "KILL"
	wait_for_runtime_session_exit "${goreman_pid}" 100 || true
}

process_start_time() {
	local pid="$1"

	awk '{ print $22 }' "/proc/${pid}/stat" 2>/dev/null
}

read_supervisor_state() {
	state_pid=""
	state_start_time=""
	state_controller_pid=""
	state_controller_start_time=""
	state_repo_root=""
	[[ -f "${SUPERVISOR_STATE_FILE}" ]] || return 1

	local key value
	while IFS='=' read -r key value; do
		case "${key}" in
		pid) state_pid="${value}" ;;
		start_time) state_start_time="${value}" ;;
		controller_pid) state_controller_pid="${value}" ;;
		controller_start_time) state_controller_start_time="${value}" ;;
		repo_root) state_repo_root="${value}" ;;
		esac
	done <"${SUPERVISOR_STATE_FILE}"

	[[ "${state_pid}" =~ ^[1-9][0-9]*$ ]] &&
		[[ "${state_start_time}" =~ ^[0-9]+$ ]] &&
		[[ "${state_controller_pid}" =~ ^[1-9][0-9]*$ ]] &&
		[[ "${state_controller_start_time}" =~ ^[0-9]+$ ]] &&
		[[ -n "${state_repo_root}" ]]
}

supervisor_matches_state() {
	read_supervisor_state || return 1
	[[ "${state_repo_root}" == "${REPO_ROOT}" ]] || return 1
	[[ -d "/proc/${state_pid}" ]] || return 1
	[[ "$(process_start_time "${state_pid}")" == "${state_start_time}" ]] || return 1

	local cwd cmd pgid session_id
	cwd="$(readlink -f "/proc/${state_pid}/cwd" 2>/dev/null || true)"
	cmd="$(process_cmd "${state_pid}")"
	pgid="$(ps -o pgid= -p "${state_pid}" 2>/dev/null | xargs || true)"
	session_id="$(ps -o sid= -p "${state_pid}" 2>/dev/null | xargs || true)"
	[[ "${cwd}" == "${REPO_ROOT}" ]] || return 1
	[[ "${cmd}" == *"goreman"* ]] || return 1
	[[ "${pgid}" == "${state_pid}" ]] || return 1
	[[ "${session_id}" == "${state_pid}" ]]
}

recorded_session_is_repo_owned() {
	read_supervisor_state || return 1
	[[ "${state_repo_root}" == "${REPO_ROOT}" ]] || return 1

	local pid cwd process_state
	local found=false
	while read -r pid; do
		[[ -n "${pid}" ]] || continue
		cwd="$(readlink -f "/proc/${pid}/cwd" 2>/dev/null || true)"
		if [[ -z "${cwd}" ]]; then
			process_state="$(ps -o stat= -p "${pid}" 2>/dev/null | xargs || true)"
			[[ "${process_state}" == Z* ]] && continue
			return 1
		fi
		case "${cwd}" in
		"${REPO_ROOT}" | "${REPO_ROOT}/"*) found=true ;;
		*) return 1 ;;
		esac
	done < <(runtime_session_pids "${state_pid}")

	[[ "${found}" == "true" ]]
}

controller_matches_state() {
	read_supervisor_state || return 1
	[[ "${state_repo_root}" == "${REPO_ROOT}" ]] || return 1
	[[ -d "/proc/${state_controller_pid}" ]] || return 1
	[[ "$(process_start_time "${state_controller_pid}")" == "${state_controller_start_time}" ]] || return 1

	local cwd cmd
	cwd="$(readlink -f "/proc/${state_controller_pid}/cwd" 2>/dev/null || true)"
	cmd="$(process_cmd "${state_controller_pid}")"
	[[ "${cwd}" == "${REPO_ROOT}" ]] || return 1
	[[ "${cmd}" == *"local-runtime.sh start"* ]]
}

wait_for_controller_exit() {
	local pid="$1"
	local start_time="$2"
	local attempt

	for attempt in {1..200}; do
		if [[ ! -d "/proc/${pid}" ]] || [[ "$(process_start_time "${pid}")" != "${start_time}" ]]; then
			return 0
		fi
		sleep 0.1
	done

	return 1
}

write_supervisor_state() {
	local pid="$1"
	local start_time controller_start_time tmp

	start_time="$(process_start_time "${pid}")"
	[[ "${start_time}" =~ ^[0-9]+$ ]] || {
		printf 'failed to read goreman process start time for pid %s\n' "${pid}" >&2
		exit 1
	}
	controller_start_time="$(process_start_time "$$")"
	[[ "${controller_start_time}" =~ ^[0-9]+$ ]] || {
		printf 'failed to read local runtime controller start time for pid %s\n' "$$" >&2
		exit 1
	}
	tmp="${SUPERVISOR_STATE_FILE}.$$"
	{
		printf 'pid=%s\n' "${pid}"
		printf 'start_time=%s\n' "${start_time}"
		printf 'controller_pid=%s\n' "$$"
		printf 'controller_start_time=%s\n' "${controller_start_time}"
		printf 'repo_root=%s\n' "${REPO_ROOT}"
	} >"${tmp}"
	mv "${tmp}" "${SUPERVISOR_STATE_FILE}"
}

remove_runtime_state() {
	rm -f -- "${SUPERVISOR_STATE_FILE}" "${FILTERED_PROCFILE}"
	rmdir "${STATE_DIR}" 2>/dev/null || true
	rmdir "${REPO_ROOT}/.run" 2>/dev/null || true
}

ensure_no_running_supervisor() {
	if [[ ! -f "${SUPERVISOR_STATE_FILE}" ]]; then
		if [[ -d "${STATE_DIR}" ]]; then
			printf 'Athena local runtime startup or stale control state exists; use make stop first.\n' >&2
			exit 1
		fi
		return
	fi

	if supervisor_matches_state && runtime_session_alive "${state_pid}"; then
		printf 'Athena local runtime is already running with goreman pid %s; use make stop first.\n' "${state_pid}" >&2
		exit 1
	fi
	if recorded_session_is_repo_owned && runtime_session_alive "${state_pid}"; then
		printf 'Athena local runtime has an orphaned process session %s; use make stop first.\n' "${state_pid}" >&2
		exit 1
	fi
	if controller_matches_state; then
		printf 'Athena local runtime controller pid %s is still shutting down; use make stop first.\n' "${state_controller_pid}" >&2
		exit 1
	fi

	if read_supervisor_state &&
		[[ "${state_repo_root}" == "${REPO_ROOT}" ]] &&
		[[ -d "/proc/${state_pid}" ]] &&
		[[ "$(process_start_time "${state_pid}")" == "${state_start_time}" ]]; then
		printf 'refusing to replace supervisor state for live unverified process pid %s\n' "${state_pid}" >&2
		exit 1
	fi

	printf 'removing stale Athena local runtime state\n' >&2
	remove_runtime_state
}

resource_label() {
	local kind="$1"
	local name="$2"
	local label="$3"

	docker "${kind}" inspect --format "{{ with .Config.Labels }}{{ index . \"${label}\" }}{{ end }}" "${name}" 2>/dev/null
}

remove_owned_container() {
	local name="$1"
	local component="$2"

	docker container inspect "${name}" >/dev/null 2>&1 || return 0

	local owner actual_component
	owner="$(resource_label container "${name}" "${RESOURCE_OWNER_LABEL}")"
	actual_component="$(resource_label container "${name}" "${RESOURCE_COMPONENT_LABEL}")"
	if [[ "${owner}" != "${RESOURCE_OWNER_VALUE}" || "${actual_component}" != "${component}" ]]; then
		printf 'refusing to remove unowned container %s; remove or rename it manually\n' "${name}" >&2
		return 1
	fi

	printf 'removing local %s container %s\n' "${component}" "${name}" >&2
	docker container rm -f "${name}" >/dev/null
}

cleanup_local_containers() {
	local status=0

	if ! docker info >/dev/null 2>&1; then
		printf 'cannot inspect Athena local containers because the Docker daemon is unavailable\n' >&2
		return 1
	fi
	remove_owned_container "${POSTGRES_CONTAINER}" "postgres" || status=1
	remove_owned_container "${REDIS_CONTAINER}" "redis" || status=1
	return "${status}"
}

remove_owned_volume() {
	local name="$1"
	local component="$2"

	docker volume inspect "${name}" >/dev/null 2>&1 || return 0

	local owner actual_component
	owner="$(docker volume inspect --format "{{ with .Labels }}{{ index . \"${RESOURCE_OWNER_LABEL}\" }}{{ end }}" "${name}")"
	actual_component="$(docker volume inspect --format "{{ with .Labels }}{{ index . \"${RESOURCE_COMPONENT_LABEL}\" }}{{ end }}" "${name}")"
	if [[ "${owner}" != "${RESOURCE_OWNER_VALUE}" || "${actual_component}" != "${component}" ]]; then
		printf 'refusing to remove unowned volume %s; remove or rename it manually\n' "${name}" >&2
		return 1
	fi

	printf 'removing local %s data volume %s\n' "${component}" "${name}" >&2
	docker volume rm "${name}" >/dev/null
}

cleanup_default_runtime_data() {
	local path

	if [[ -e "/tmp/athena-local" || -L "/tmp/athena-local" ]]; then
		printf 'removing default Athena runtime data /tmp/athena-local\n' >&2
		rm -rf -- "/tmp/athena-local"
	fi

	for path in "${coverage_dirs[@]}"; do
		if [[ -e "${path}" || -L "${path}" ]]; then
			printf 'removing default Athena coverage data %s\n' "${path}" >&2
			rm -rf -- "${path}"
		fi
	done
}

start_goreman() {
	local procfile="$1"

	printf 'starting Procfile services with goreman\n' >&2
	(
		cd "${REPO_ROOT}"
		exec setsid goreman -f "${procfile}" start
	) &
	goreman_pid="$!"
	write_supervisor_state "${goreman_pid}"
	printf 'started goreman pid=%s\n' "${goreman_pid}" >&2
}

cleanup_foreground_run() {
	local status="$?"
	local cleanup_status=0

	if [[ "${cleanup_started}" == "true" ]]; then
		return
	fi
	cleanup_started=true
	trap - EXIT INT TERM

	set +e
	stop_goreman
	cleanup_status="$?"
	if [[ "${ATHENA_RUN_PORT_CLEANUP:-true}" != "false" ]]; then
		cleanup_athena_ports stop
		[[ "$?" == "0" ]] || cleanup_status=1
	fi
	cleanup_local_containers
	[[ "$?" == "0" ]] || cleanup_status=1
	remove_runtime_state
	set -e

	if [[ "${status}" == "0" && "${cleanup_status}" != "0" ]]; then
		status="${cleanup_status}"
	fi

	exit "${status}"
}

start_runtime() {
	local procfile="${ATHENA_PROCFILE:-Procfile}"

	if [[ "${ATHENA_RUN_DRY_RUN:-false}" == "true" ]]; then
		local dry_run_procfile
		dry_run_procfile="$(mktemp -t athena-procfile.XXXXXX)"
		trap 'rm -f -- "${dry_run_procfile}"' RETURN
		write_filtered_procfile "${procfile}" "${dry_run_procfile}"
		cat "${dry_run_procfile}"
		rm -f -- "${dry_run_procfile}"
		trap - RETURN
		return
	fi

	ensure_no_running_supervisor
	mkdir -p "${REPO_ROOT}/.run"
	if ! mkdir "${STATE_DIR}"; then
		printf 'Athena local runtime state was created concurrently; use make stop before retrying.\n' >&2
		exit 1
	fi
	trap cleanup_foreground_run EXIT
	trap 'exit 130' INT
	trap 'exit 143' TERM

	write_filtered_procfile "${procfile}" "${FILTERED_PROCFILE}"
	if [[ -n "$(run_exclude_value)" ]]; then
		printf 'excluding Procfile services: %s\n' "$(run_exclude_value)" >&2
	fi

	cleanup_local_containers
	if [[ "${ATHENA_RUN_PORT_CLEANUP:-true}" != "false" ]]; then
		cleanup_athena_ports start
	fi
	configure_token_node_ws_proxy

	start_goreman "${FILTERED_PROCFILE}"

	set +e
	wait "${goreman_pid}"
	local goreman_status="$?"
	set -e

	return "${goreman_status}"
}

stop_runtime() {
	local controller_pid_to_wait=""
	local controller_start_time_to_wait=""
	local cleanup_status=0
	if controller_matches_state; then
		controller_pid_to_wait="${state_controller_pid}"
		controller_start_time_to_wait="${state_controller_start_time}"
	fi

	if [[ -f "${SUPERVISOR_STATE_FILE}" ]]; then
		if supervisor_matches_state; then
			goreman_pid="${state_pid}"
			stop_goreman
		elif recorded_session_is_repo_owned; then
			goreman_pid="${state_pid}"
			printf 'stopping orphaned Athena local process session pid=%s signal=TERM\n' "${goreman_pid}" >&2
			signal_runtime_session "${goreman_pid}" "TERM"
			if ! wait_for_runtime_session_exit "${goreman_pid}" 100; then
				printf 'orphaned Athena local process session did not stop after TERM, sending KILL pid=%s\n' "${goreman_pid}" >&2
				signal_runtime_session "${goreman_pid}" "KILL"
				wait_for_runtime_session_exit "${goreman_pid}" 100 || true
			fi
		elif read_supervisor_state &&
			[[ "${state_repo_root}" == "${REPO_ROOT}" ]] &&
			[[ -d "/proc/${state_pid}" ]] &&
			[[ "$(process_start_time "${state_pid}")" == "${state_start_time}" ]]; then
			printf 'refusing to stop live unverified process pid %s from supervisor state\n' "${state_pid}" >&2
			exit 1
		else
			printf 'discarding stale Athena local runtime state\n' >&2
		fi
	fi

	if [[ -n "${controller_pid_to_wait}" ]] &&
		! wait_for_controller_exit "${controller_pid_to_wait}" "${controller_start_time_to_wait}"; then
		printf 'Athena local runtime controller pid %s did not exit after its process session stopped\n' "${controller_pid_to_wait}" >&2
		exit 1
	fi

	if [[ "${ATHENA_RUN_PORT_CLEANUP:-true}" != "false" ]]; then
		cleanup_athena_ports stop || cleanup_status=1
	fi
	cleanup_local_containers || cleanup_status=1
	remove_runtime_state
	return "${cleanup_status}"
}

reset_runtime() {
	stop_runtime
	remove_owned_volume "${POSTGRES_VOLUME}" "postgres"
	remove_owned_volume "${REDIS_VOLUME}" "redis"
	cleanup_default_runtime_data
	rm -rf -- "${STATE_DIR}"
	rmdir "${REPO_ROOT}/.run" 2>/dev/null || true
}

usage() {
	printf 'usage: %s {start|stop|reset}\n' "${BASH_SOURCE[0]}" >&2
}

action="${1:-start}"
case "${action}" in
start)
	start_runtime
	;;
stop)
	stop_runtime
	;;
reset)
	reset_runtime
	;;
*)
	usage
	exit 2
	;;
esac
