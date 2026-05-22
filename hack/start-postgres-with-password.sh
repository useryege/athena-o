#!/bin/bash

set -euo pipefail

# Default values for environment variables
POSTGRES_PORT="${ATHENA_POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-athena}"
POSTGRES_DB="${POSTGRES_DB:-athena}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"
POSTGRES_IMAGE_TAG="${ATHENA_POSTGRES_IMAGE_TAG:-16}"
ATHENA_LOCAL_DATA_MODE="${ATHENA_LOCAL_DATA_MODE:-ephemeral}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
POSTGRES_DATA_DIR="${ATHENA_POSTGRES_DATA_DIR:-/tmp/athena-local/postgres}"
POSTGRES_INIT_DIR="${ATHENA_POSTGRES_INIT_DIR:-$REPO_ROOT/hack/postgres/init}"

perf_opts=(
  "-c" "fsync=off"
  "-c" "full_page_writes=off"
  "-c" "synchronous_commit=off"
)

run_postgres_init() {
    local database="$1"

    if [ ! -d "$POSTGRES_INIT_DIR" ]; then
        return
    fi

    for init_sql in "$POSTGRES_INIT_DIR"/*.sql; do
        if [ ! -e "$init_sql" ]; then
            continue
        fi
        echo "Running PostgreSQL init script: $init_sql"
        PGPASSWORD="$POSTGRES_PASSWORD" psql -h 127.0.0.1 -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$database" -v ON_ERROR_STOP=1 -f "$init_sql"
    done
}

if [ "${ATHENA_POSTGRES_LOCAL:-false}" = 'true' ]; then
    if ! command -v initdb >/dev/null 2>&1 || ! command -v pg_ctl >/dev/null 2>&1 || ! command -v postgres >/dev/null 2>&1 || ! command -v psql >/dev/null 2>&1; then
      echo "PostgreSQL local tools are not installed. Please install PostgreSQL binaries or set ATHENA_POSTGRES_LOCAL to false."
      exit 1
    fi

    mkdir -p "$POSTGRES_DATA_DIR"

    if [ ! -f "$POSTGRES_DATA_DIR/PG_VERSION" ]; then
        echo "Initializing local PostgreSQL data directory: $POSTGRES_DATA_DIR"
        auth_mode="scram-sha-256"
        pw_file="$(mktemp)"
        if [ -z "$POSTGRES_PASSWORD" ]; then
            auth_mode="trust"
        fi
        printf "%s" "$POSTGRES_PASSWORD" > "$pw_file"
        initdb -D "$POSTGRES_DATA_DIR" --username="$POSTGRES_USER" --pwfile="$pw_file" --auth-local=trust --auth-host="$auth_mode" >/dev/null
        rm -f "$pw_file"
    fi

    if pg_ctl -D "$POSTGRES_DATA_DIR" status >/dev/null 2>&1; then
        echo "A PostgreSQL instance is already running for data dir $POSTGRES_DATA_DIR. Reusing existing cluster."
    else
        echo "Bootstrapping local PostgreSQL cluster."
        pg_ctl -D "$POSTGRES_DATA_DIR" -w start -o "-p $POSTGRES_PORT -c listen_addresses=127.0.0.1 ${perf_opts[*]}" >/dev/null
        if [ "$POSTGRES_DB" != "postgres" ]; then
            if ! PGPASSWORD="$POSTGRES_PASSWORD" psql -h 127.0.0.1 -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='${POSTGRES_DB}'" | grep -q 1; then
                PGPASSWORD="$POSTGRES_PASSWORD" createdb -h 127.0.0.1 -p "$POSTGRES_PORT" -U "$POSTGRES_USER" "$POSTGRES_DB"
            fi
        fi
        run_postgres_init "$POSTGRES_DB"
        pg_ctl -D "$POSTGRES_DATA_DIR" -m fast -w stop >/dev/null
    fi

    echo "Starting local PostgreSQL server on port $POSTGRES_PORT."
    exec postgres -D "$POSTGRES_DATA_DIR" -p "$POSTGRES_PORT" -c listen_addresses=127.0.0.1 "${perf_opts[@]}"
else
    docker_args=(
        "--rm"
        "--name" "athena-postgres"
        "-i"
        "-p" "$POSTGRES_PORT:$POSTGRES_PORT"
        "-e" "POSTGRES_USER=$POSTGRES_USER"
        "-e" "POSTGRES_DB=$POSTGRES_DB"
    )

    if [ "$ATHENA_LOCAL_DATA_MODE" = "persistent" ]; then
        mkdir -p "$POSTGRES_DATA_DIR"
        docker_args+=("-v" "$POSTGRES_DATA_DIR:/var/lib/postgresql/data")
    fi

    if [ -d "$POSTGRES_INIT_DIR" ]; then
        docker_args+=("-v" "$POSTGRES_INIT_DIR:/docker-entrypoint-initdb.d:ro")
    fi

    if [ -z "$POSTGRES_PASSWORD" ]; then
        echo "Starting Docker PostgreSQL container without password (trust auth)."
        docker_args+=("-e" "POSTGRES_HOST_AUTH_METHOD=trust")
        exec docker run "${docker_args[@]}" \
            docker.io/library/postgres:"$POSTGRES_IMAGE_TAG" \
            -p "$POSTGRES_PORT" "${perf_opts[@]}"
    else
        echo "Starting Docker PostgreSQL container with password."
        docker_args+=("-e" "POSTGRES_PASSWORD=$POSTGRES_PASSWORD")
        exec docker run "${docker_args[@]}" \
            docker.io/library/postgres:"$POSTGRES_IMAGE_TAG" \
            -p "$POSTGRES_PORT" "${perf_opts[@]}"
    fi
fi
