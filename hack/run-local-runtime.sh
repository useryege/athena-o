#!/usr/bin/env bash
# Bootstrap the local launcher, then replace this shell so Make waits for its
# signal relay. This bootstrap compiler does not create instance resources.
set -euo pipefail
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
saved_runner="$(python3 "$script_dir/saved-local-runtime.py" "$@")"
if [[ -n "$saved_runner" ]]; then
  exec "$saved_runner" "$@"
fi
printf 'No saved runner; building local launcher (5 minute budget) ...\n' >&2
launcher_dir="$(mktemp -d "${TMPDIR:-/tmp}/athena-local-launcher.XXXXXXXX")"
compiler_pid=''
interrupt_compile() {
  trap '' INT TERM
  if [[ -n "$compiler_pid" ]]; then
    kill -TERM -- "-$compiler_pid" 2>/dev/null || true
    wait "$compiler_pid" 2>/dev/null || true
  fi
  exit 130
}
trap interrupt_compile INT TERM
setsid timeout --signal=TERM --kill-after=5s 300s go build -o "$launcher_dir/athena-local-runtime" ./cmd/athena-local-runtime &
compiler_pid=$!
wait "$compiler_pid"
compiler_pid=''
trap - INT TERM
exec "$launcher_dir/athena-local-runtime" "$@"
