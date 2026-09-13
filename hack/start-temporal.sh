#!/bin/bash

set -euo pipefail

POSTGRES_PORT="${ATHENA_POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-athena}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"
POSTGRES_CONTAINER_NAME="${ATHENA_POSTGRES_CONTAINER_NAME:-athena-postgres}"
TEMPORAL_PORT="${ATHENA_TEMPORAL_PORT:-7233}"
TEMPORAL_IMAGE_TAG="${ATHENA_TEMPORAL_IMAGE_TAG:-1.24}"
TEMPORAL_CONTAINER_NAME="${ATHENA_TEMPORAL_CONTAINER_NAME:-athena-temporal}"
TEMPORAL_DB="${ATHENA_TEMPORAL_DB:-temporal}"
TEMPORAL_VISIBILITY_DB="${ATHENA_TEMPORAL_VISIBILITY_DB:-temporal_visibility}"

run_psql() {
    if command -v psql >/dev/null 2>&1; then
        PGPASSWORD="$POSTGRES_PASSWORD" psql -h 127.0.0.1 -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d postgres "$@"
        return
    fi

    if docker ps --format '{{.Names}}' | grep -Fxq "$POSTGRES_CONTAINER_NAME"; then
        docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$POSTGRES_CONTAINER_NAME" psql -U "$POSTGRES_USER" -d postgres "$@"
        return
    fi

    return 127
}

wait_for_postgres() {
    local attempt

    for ((attempt = 0; attempt < 120; attempt++)); do
        if run_psql -tAc "SELECT 1" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done

    echo "Timed out waiting for PostgreSQL on 127.0.0.1:$POSTGRES_PORT." >&2
    return 1
}

ensure_database() {
    local database="$1"

    if ! run_psql -tAc "SELECT 1 FROM pg_database WHERE datname='${database}'" | grep -q 1; then
        run_psql -v ON_ERROR_STOP=1 -c "CREATE DATABASE ${database}"
    fi
}

wait_for_postgres
ensure_database "$TEMPORAL_DB"
ensure_database "$TEMPORAL_VISIBILITY_DB"

if docker ps -a --format '{{.Names}}' | grep -Fxq "$TEMPORAL_CONTAINER_NAME"; then
    echo "Removing stale Temporal container $TEMPORAL_CONTAINER_NAME."
    docker rm -f "$TEMPORAL_CONTAINER_NAME" >/dev/null
fi

docker_args=(
    "--rm"
    "--name" "$TEMPORAL_CONTAINER_NAME"
    "-i"
    "-p" "$TEMPORAL_PORT:7233"
    "--add-host" "host.docker.internal:host-gateway"
    "-e" "DB=postgres12"
    "-e" "DB_PORT=$POSTGRES_PORT"
    "-e" "POSTGRES_SEEDS=host.docker.internal"
    "-e" "POSTGRES_USER=$POSTGRES_USER"
    "-e" "POSTGRES_PWD=$POSTGRES_PASSWORD"
    "-e" "DBNAME=$TEMPORAL_DB"
    "-e" "VISIBILITY_DBNAME=$TEMPORAL_VISIBILITY_DB"
)

echo "Starting Docker Temporal container on port $TEMPORAL_PORT."
exec docker run "${docker_args[@]}" docker.io/temporalio/auto-setup:"$TEMPORAL_IMAGE_TAG"
