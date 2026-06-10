#!/bin/bash

set -euo pipefail

# Default values for environment variables
POSTGRES_PORT="${ATHENA_POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-athena}"
POSTGRES_DB="${POSTGRES_DB:-athena}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"
POSTGRES_IMAGE_TAG="${ATHENA_POSTGRES_IMAGE_TAG:-16}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
POSTGRES_INIT_DIR="${ATHENA_POSTGRES_INIT_DIR:-$REPO_ROOT/hack/postgres/init}"

perf_opts=(
  "-c" "fsync=off"
  "-c" "full_page_writes=off"
  "-c" "synchronous_commit=off"
)

docker_args=(
    "--rm"
    "--name" "athena-postgres"
    "-i"
    "-p" "$POSTGRES_PORT:$POSTGRES_PORT"
    "-e" "POSTGRES_USER=$POSTGRES_USER"
    "-e" "POSTGRES_DB=$POSTGRES_DB"
)

if [ -d "$POSTGRES_INIT_DIR" ]; then
    docker_args+=("-v" "$POSTGRES_INIT_DIR:/docker-entrypoint-initdb.d:ro")
fi

if [ -z "$POSTGRES_PASSWORD" ]; then
    echo "Starting ephemeral Docker PostgreSQL container without password (trust auth)."
    docker_args+=("-e" "POSTGRES_HOST_AUTH_METHOD=trust")
    exec docker run "${docker_args[@]}" \
        docker.io/library/postgres:"$POSTGRES_IMAGE_TAG" \
        -p "$POSTGRES_PORT" "${perf_opts[@]}"
else
    echo "Starting ephemeral Docker PostgreSQL container with password."
    docker_args+=("-e" "POSTGRES_PASSWORD=$POSTGRES_PASSWORD")
    exec docker run "${docker_args[@]}" \
        docker.io/library/postgres:"$POSTGRES_IMAGE_TAG" \
        -p "$POSTGRES_PORT" "${perf_opts[@]}"
fi
