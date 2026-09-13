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
exec bash ./hack/run-local-runtime.sh "$action"
