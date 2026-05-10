#!/usr/bin/env bash

set -euo pipefail

POSTGRES_PORT="${ATHENA_POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-athena}"
POSTGRES_DB="${POSTGRES_DB:-athena}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"
POSTGRES_DATA_DIR="${ATHENA_POSTGRES_DATA_DIR:-/tmp/athena-local/postgres}"

echo "Starting postgres on port ${POSTGRES_PORT}"
mkdir -p "${POSTGRES_DATA_DIR}"

if [ ! -f "${POSTGRES_DATA_DIR}/PG_VERSION" ]; then
  auth_host="scram-sha-256"
  if [ -z "${POSTGRES_PASSWORD}" ]; then
    auth_host="trust"
  fi

  if [ -z "${POSTGRES_PASSWORD}" ]; then
    initdb -D "${POSTGRES_DATA_DIR}" \
      --username="${POSTGRES_USER}" \
      --auth-local=trust \
      --auth-host="${auth_host}"
  else
    pw_file="$(mktemp)"
    printf "%s" "${POSTGRES_PASSWORD}" > "${pw_file}"
    initdb -D "${POSTGRES_DATA_DIR}" \
      --username="${POSTGRES_USER}" \
      --pwfile="${pw_file}" \
      --auth-local=trust \
      --auth-host="${auth_host}"
    rm -f "${pw_file}"
  fi
fi

if ! pg_ctl -D "${POSTGRES_DATA_DIR}" status >/dev/null 2>&1; then
  pg_ctl -D "${POSTGRES_DATA_DIR}" -w start -o "-p ${POSTGRES_PORT} -c listen_addresses=127.0.0.1 -c fsync=off -c full_page_writes=off -c synchronous_commit=off"
  if [ "${POSTGRES_DB}" != "postgres" ]; then
    db_exists="$(PGPASSWORD="${POSTGRES_PASSWORD}" psql -h 127.0.0.1 -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='${POSTGRES_DB}'")"
    if [ "${db_exists}" != "1" ]; then
      PGPASSWORD="${POSTGRES_PASSWORD}" createdb -h 127.0.0.1 -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" "${POSTGRES_DB}"
    fi
  fi
  pg_ctl -D "${POSTGRES_DATA_DIR}" -m fast -w stop
fi

exec postgres -D "${POSTGRES_DATA_DIR}" -p "${POSTGRES_PORT}" -c listen_addresses=127.0.0.1 -c fsync=off -c full_page_writes=off -c synchronous_commit=off
