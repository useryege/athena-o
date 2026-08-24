#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd -P)"

MINIO_API_PORT="${ATHENA_MINIO_API_PORT:-9000}"
MINIO_CONSOLE_PORT="${ATHENA_MINIO_CONSOLE_PORT:-9001}"
MINIO_ROOT_USER="${MINIO_ROOT_USER:-athena-local-minio-root}"
MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD:-athena-local-minio-root-password}"
MINIO_REGION="${ATHENA_ACCOUNT_AVATAR_S3_REGION:-us-east-1}"
MINIO_BUCKET="${ATHENA_ACCOUNT_AVATAR_S3_BUCKET:-athena-account-avatars}"
MINIO_APP_ACCESS_KEY="${ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID:-athena-local-avatar}"
MINIO_APP_SECRET_KEY="${ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY:-athena-local-avatar-secret}"
MINIO_IMAGE="${ATHENA_MINIO_IMAGE:-athena-minio:9e49d5e7a648}"
MINIO_MC_IMAGE="${ATHENA_MINIO_MC_IMAGE:-athena-minio-mc:7394ce0dd2a8}"
MINIO_CONTAINER="athena-minio"
MINIO_VOLUME="athena-local-minio-data"
RESOURCE_OWNER_LABEL="io.athena.local-runtime"
RESOURCE_OWNER_VALUE="athena"
RESOURCE_COMPONENT_LABEL="io.athena.component"
container_started=false
logs_pid=""

ensure_image() {
  local image="$1"
  local dockerfile="$2"

  if docker image inspect "${image}" >/dev/null 2>&1; then
    return
  fi
  printf 'building pinned MinIO image %s\n' "${image}" >&2
  DOCKER_BUILDKIT=1 docker build \
    --file "${REPO_ROOT}/deploy/minio/${dockerfile}" \
    --tag "${image}" \
    "${REPO_ROOT}/deploy/minio"
}

ensure_volume() {
  if ! docker volume inspect "${MINIO_VOLUME}" >/dev/null 2>&1; then
    docker volume create \
      --label "${RESOURCE_OWNER_LABEL}=${RESOURCE_OWNER_VALUE}" \
      --label "${RESOURCE_COMPONENT_LABEL}=minio" \
      "${MINIO_VOLUME}" >/dev/null
    printf 'created persistent MinIO volume %s\n' "${MINIO_VOLUME}" >&2
    return
  fi

  local owner component
  owner="$(docker volume inspect --format "{{ with .Labels }}{{ index . \"${RESOURCE_OWNER_LABEL}\" }}{{ end }}" "${MINIO_VOLUME}")"
  component="$(docker volume inspect --format "{{ with .Labels }}{{ index . \"${RESOURCE_COMPONENT_LABEL}\" }}{{ end }}" "${MINIO_VOLUME}")"
  if [[ "${owner}" != "${RESOURCE_OWNER_VALUE}" || "${component}" != "minio" ]]; then
    printf 'MinIO volume %s is not owned by the Athena local runtime; remove or rename it manually.\n' "${MINIO_VOLUME}" >&2
    exit 1
  fi
}

stop_container() {
  trap - INT TERM EXIT
  if [[ -n "${logs_pid}" ]]; then
    kill "${logs_pid}" >/dev/null 2>&1 || true
  fi
  if [[ "${container_started}" == "true" ]] && docker container inspect "${MINIO_CONTAINER}" >/dev/null 2>&1; then
    docker container stop --time 10 "${MINIO_CONTAINER}" >/dev/null 2>&1 || true
  fi
}

trap 'stop_container; exit 130' INT
trap 'stop_container; exit 143' TERM
trap stop_container EXIT

ensure_image "${MINIO_IMAGE}" Dockerfile.server
ensure_image "${MINIO_MC_IMAGE}" Dockerfile.mc
ensure_volume

docker run \
  --detach \
  --rm \
  --name "${MINIO_CONTAINER}" \
  --label "${RESOURCE_OWNER_LABEL}=${RESOURCE_OWNER_VALUE}" \
  --label "${RESOURCE_COMPONENT_LABEL}=minio" \
  --publish "127.0.0.1:${MINIO_API_PORT}:9000" \
  --publish "127.0.0.1:${MINIO_CONSOLE_PORT}:9001" \
  --env "MINIO_ROOT_USER=${MINIO_ROOT_USER}" \
  --env "MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD}" \
  --env "MINIO_REGION_NAME=${MINIO_REGION}" \
  --volume "${MINIO_VOLUME}:/data" \
  "${MINIO_IMAGE}" \
  server /data --address :9000 --console-address :9001 >/dev/null
container_started=true

if ! docker run \
  --rm \
  --network "container:${MINIO_CONTAINER}" \
  --env "MINIO_ENDPOINT=http://127.0.0.1:9000" \
  --env "MINIO_ROOT_USER=${MINIO_ROOT_USER}" \
  --env "MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD}" \
  --env "ATHENA_ACCOUNT_AVATAR_S3_BUCKET=${MINIO_BUCKET}" \
  --env "ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID=${MINIO_APP_ACCESS_KEY}" \
  --env "ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY=${MINIO_APP_SECRET_KEY}" \
  "${MINIO_MC_IMAGE}"; then
  docker logs "${MINIO_CONTAINER}" >&2 || true
  exit 1
fi

docker logs --follow "${MINIO_CONTAINER}" &
logs_pid="$!"
set +e
container_status="$(docker wait "${MINIO_CONTAINER}")"
wait_status="$?"
wait "${logs_pid}" >/dev/null 2>&1 || true
set -e
logs_pid=""
container_started=false
if [[ "${wait_status}" != "0" || ! "${container_status}" =~ ^[0-9]+$ ]]; then
  exit 1
fi
status="${container_status}"
exit "${status}"
