#!/usr/bin/env bash
# Goreman has already merged inherited env and .env before invoking this helper.
# Only the local API entry uses the WSL default; deployment config stays explicit.
set -euo pipefail
if [[ "${ATHENA_TRADER_SYNC_PROXY_URL+x}" != x ]]; then
 if [[ -n "${WSL_DISTRO_NAME:-}" || -n "${WSL_INTEROP:-}" ]] || grep -qi microsoft /proc/sys/kernel/osrelease 2>/dev/null; then
  if ! command -v ip >/dev/null 2>&1; then
   printf 'cannot configure Trader Sync proxy: ip command is unavailable\n' >&2
   exit 1
  fi
  trader_sync_gateway="$(ip route show default | awk 'NR == 1 { print $3 }')"
  if [[ -z "$trader_sync_gateway" ]]; then
   printf 'cannot configure Trader Sync proxy: WSL default gateway was not found\n' >&2
   exit 1
  fi
  export ATHENA_TRADER_SYNC_PROXY_URL="http://${trader_sync_gateway}:10809"
 fi
fi
exec "$@"
