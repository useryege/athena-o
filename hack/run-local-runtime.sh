#!/usr/bin/env bash
# Bootstrap the local launcher, then replace this shell so Make waits for its
# signal relay. This bootstrap compiler does not create instance resources.
set -euo pipefail
launcher_dir="$(mktemp -d "${TMPDIR:-/tmp}/athena-local-launcher.XXXXXXXX")"
compiler_pid=''
interrupt_compile() {
  trap '' INT TERM
  if [[ -n "$compiler_pid" ]]; then
    kill -TERM "$compiler_pid" 2>/dev/null || true
    wait "$compiler_pid" 2>/dev/null || true
  fi
  exit 130
}
trap interrupt_compile INT TERM
go build -o "$launcher_dir/athena-local-runtime" ./cmd/athena-local-runtime &
compiler_pid=$!
wait "$compiler_pid"
compiler_pid=''
trap - INT TERM
exec "$launcher_dir/athena-local-runtime" "$@"
