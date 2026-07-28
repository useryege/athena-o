#!/usr/bin/env bash

set -euo pipefail

POSTGRES_PORT="${ATHENA_POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-athena}"
POSTGRES_DB="${POSTGRES_DB:-athena}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"
POSTGRES_IMAGE_TAG="${ATHENA_POSTGRES_IMAGE_TAG:-16}"
POSTGRES_CONTAINER="athena-postgres"
POSTGRES_VOLUME="athena-local-postgres-data"
RESOURCE_OWNER_LABEL="io.athena.local-runtime"
RESOURCE_OWNER_VALUE="athena"
RESOURCE_COMPONENT_LABEL="io.athena.component"
RESOURCE_CONFIG_LABEL="io.athena.postgres-config-sha256"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd -P)"
POSTGRES_INIT_DIR="${ATHENA_POSTGRES_INIT_DIR:-${REPO_ROOT}/hack/postgres/init}"
POSTGRES_IMAGE="docker.io/library/postgres:${POSTGRES_IMAGE_TAG}"

postgres_config_fingerprint() {
	{
		printf 'format=1\0'
		printf 'image=%s\0' "${POSTGRES_IMAGE}"
		printf 'user=%s\0' "${POSTGRES_USER}"
		printf 'database=%s\0' "${POSTGRES_DB}"
		printf 'password=%s\0' "${POSTGRES_PASSWORD}"
		if [[ -d "${POSTGRES_INIT_DIR}" ]]; then
			while IFS= read -r -d '' file; do
				printf 'init-file=%s\0' "${file#"${POSTGRES_INIT_DIR}"/}"
				sha256sum "${file}" | awk '{ print $1 }'
			done < <(find "${POSTGRES_INIT_DIR}" -type f -print0 | sort -z)
		else
			printf 'init-directory=missing\0'
		fi
	} | sha256sum | awk '{ print $1 }'
}

ensure_postgres_volume() {
	local desired_fingerprint="$1"

	if ! docker volume inspect "${POSTGRES_VOLUME}" >/dev/null 2>&1; then
		docker volume create \
			--label "${RESOURCE_OWNER_LABEL}=${RESOURCE_OWNER_VALUE}" \
			--label "${RESOURCE_COMPONENT_LABEL}=postgres" \
			--label "${RESOURCE_CONFIG_LABEL}=${desired_fingerprint}" \
			"${POSTGRES_VOLUME}" >/dev/null
		printf 'created persistent PostgreSQL volume %s\n' "${POSTGRES_VOLUME}" >&2
		return
	fi

	local owner component actual_fingerprint
	owner="$(docker volume inspect --format "{{ with .Labels }}{{ index . \"${RESOURCE_OWNER_LABEL}\" }}{{ end }}" "${POSTGRES_VOLUME}")"
	component="$(docker volume inspect --format "{{ with .Labels }}{{ index . \"${RESOURCE_COMPONENT_LABEL}\" }}{{ end }}" "${POSTGRES_VOLUME}")"
	actual_fingerprint="$(docker volume inspect --format "{{ with .Labels }}{{ index . \"${RESOURCE_CONFIG_LABEL}\" }}{{ end }}" "${POSTGRES_VOLUME}")"
	if [[ "${owner}" != "${RESOURCE_OWNER_VALUE}" || "${component}" != "postgres" ]]; then
		printf 'PostgreSQL volume %s is not owned by the Athena local runtime; remove or rename it manually.\n' "${POSTGRES_VOLUME}" >&2
		exit 1
	fi
	if [[ "${actual_fingerprint}" != "${desired_fingerprint}" ]]; then
		printf 'PostgreSQL local initialization configuration changed; run make run-reset before make run.\n' >&2
		exit 1
	fi
}

perf_opts=(
	"-c" "fsync=off"
	"-c" "full_page_writes=off"
	"-c" "synchronous_commit=off"
)

config_fingerprint="$(postgres_config_fingerprint)"
ensure_postgres_volume "${config_fingerprint}"

docker_args=(
	"--rm"
	"--name" "${POSTGRES_CONTAINER}"
	"--label" "${RESOURCE_OWNER_LABEL}=${RESOURCE_OWNER_VALUE}"
	"--label" "${RESOURCE_COMPONENT_LABEL}=postgres"
	"-i"
	"-p" "${POSTGRES_PORT}:${POSTGRES_PORT}"
	"-e" "POSTGRES_USER=${POSTGRES_USER}"
	"-e" "POSTGRES_DB=${POSTGRES_DB}"
	"-v" "${POSTGRES_VOLUME}:/var/lib/postgresql/data"
)

if [[ -d "${POSTGRES_INIT_DIR}" ]]; then
	docker_args+=("-v" "${POSTGRES_INIT_DIR}:/docker-entrypoint-initdb.d:ro")
fi

if [[ -z "${POSTGRES_PASSWORD}" ]]; then
	printf 'Starting Docker PostgreSQL with persistent data and trust auth.\n'
	docker_args+=("-e" "POSTGRES_HOST_AUTH_METHOD=trust")
else
	printf 'Starting Docker PostgreSQL with persistent data and password auth.\n'
	docker_args+=("-e" "POSTGRES_PASSWORD=${POSTGRES_PASSWORD}")
fi

exec docker run "${docker_args[@]}" \
	"${POSTGRES_IMAGE}" \
	-p "${POSTGRES_PORT}" "${perf_opts[@]}"
