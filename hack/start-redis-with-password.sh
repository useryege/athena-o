#!/usr/bin/env bash

set -euo pipefail

REDIS_PORT="${ATHENA_REDIS_PORT:-6379}"
REDIS_IMAGE_TAG="${ATHENA_REDIS_IMAGE_TAG:-8.2.3}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"
REDIS_CONTAINER="athena-redis"
REDIS_VOLUME="athena-local-redis-data"
RESOURCE_OWNER_LABEL="io.athena.local-runtime"
RESOURCE_OWNER_VALUE="athena"
RESOURCE_COMPONENT_LABEL="io.athena.component"
REDIS_IMAGE="docker.io/library/redis:${REDIS_IMAGE_TAG}"

ensure_redis_volume() {
	if ! docker volume inspect "${REDIS_VOLUME}" >/dev/null 2>&1; then
		docker volume create \
			--label "${RESOURCE_OWNER_LABEL}=${RESOURCE_OWNER_VALUE}" \
			--label "${RESOURCE_COMPONENT_LABEL}=redis" \
			"${REDIS_VOLUME}" >/dev/null
		printf 'created persistent Redis volume %s\n' "${REDIS_VOLUME}" >&2
		return
	fi

	local owner component
	owner="$(docker volume inspect --format "{{ with .Labels }}{{ index . \"${RESOURCE_OWNER_LABEL}\" }}{{ end }}" "${REDIS_VOLUME}")"
	component="$(docker volume inspect --format "{{ with .Labels }}{{ index . \"${RESOURCE_COMPONENT_LABEL}\" }}{{ end }}" "${REDIS_VOLUME}")"
	if [[ "${owner}" != "${RESOURCE_OWNER_VALUE}" || "${component}" != "redis" ]]; then
		printf 'Redis volume %s is not owned by the Athena local runtime; remove or rename it manually.\n' "${REDIS_VOLUME}" >&2
		exit 1
	fi
}

ensure_redis_volume

docker_args=(
	"--rm"
	"--name" "${REDIS_CONTAINER}"
	"--label" "${RESOURCE_OWNER_LABEL}=${RESOURCE_OWNER_VALUE}"
	"--label" "${RESOURCE_COMPONENT_LABEL}=redis"
	"-i"
	"-p" "${REDIS_PORT}:${REDIS_PORT}"
	"-v" "${REDIS_VOLUME}:/data"
)
redis_args=(
	"--port" "${REDIS_PORT}"
	"--dir" "/data"
	"--save" ""
	"--appendonly" "yes"
)

if [[ -z "${REDIS_PASSWORD}" ]]; then
	printf 'Starting Docker Redis with persistent AOF data and no password.\n'
	exec docker run "${docker_args[@]}" "${REDIS_IMAGE}" redis-server "${redis_args[@]}"
fi

printf 'Starting Docker Redis with persistent AOF data and password auth.\n'
exec docker run "${docker_args[@]}" \
	"-e" "REDIS_PASSWORD=${REDIS_PASSWORD}" \
	"${REDIS_IMAGE}" \
	redis-server --requirepass "${REDIS_PASSWORD}" "${redis_args[@]}"
