#!/usr/bin/env bash
# Full-stack command adapter. Resource ownership lives only in devruntime.
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
case "${1:-start}" in
  start) action=make-run-full-stack ;;
  stop) action=make-stop-full-stack ;;
  reset) action=make-reset-full-stack ;;
  *) printf 'usage: %s {start|stop|reset}\n' "$0" >&2; exit 2 ;;
esac
if (( $# > 1 )); then
  printf 'unexpected positional arguments\n' >&2
  exit 2
fi
cd "$repo_root"
case "${ATHENA_RUN_PROFILE:-}" in
  '') ;;
  solana-discovery|solana-preview)
    if [[ "${1:-start}" == stop ]]; then
      # A legacy profile can only be stopped by the owner that created it.
      exec bash ./hack/solana-local.sh stop "$ATHENA_RUN_PROFILE"
    fi
    printf 'Solana profile startup is retired; use make run-service SERVICE=solana-discovery or make run. Stop existing profiles with their original profile owner.\n' >&2
    exit 2
    ;;
  *) printf 'unknown run profile: %s\n' "$ATHENA_RUN_PROFILE" >&2; exit 2 ;;
esac
exec bash ./hack/run-local-runtime.sh "$action"
