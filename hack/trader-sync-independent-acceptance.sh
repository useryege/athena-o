#!/usr/bin/env bash
# Exercise a new dedicated managed instance through the public Make commands.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
command -v go >/dev/null
command -v docker >/dev/null
command -v make >/dev/null
docker info >/dev/null
acceptance_root="$(pwd -P)/.tmp/trader-sync-independent-acceptance"
mkdir -p "$acceptance_root"
acceptance_dir="$(mktemp -d "$acceptance_root/$(date -u +%Y%m%dT%H%M%SZ)-XXXXXXXX")"
export ATHENA_INDEPENDENT_ACCEPTANCE=1
export ATHENA_INDEPENDENT_INSTANCE="${INSTANCE:-ts-controlled-acceptance}"
export ATHENA_INDEPENDENT_ARTIFACTS="$acceptance_dir"
printf 'Independent runtime acceptance evidence: %s\n' "$acceptance_dir"
set +e
go test -tags=integration ./internal/tradersync/acceptance -run '^TestIndependentRuntimeLifecycle$' -count=1 -v -timeout=10m 2>&1 | tee "$acceptance_dir/test.log"
acceptance_exit=${PIPESTATUS[0]}
set -e
printf '%s\n' "$acceptance_exit" > "$acceptance_dir/exit-code"
exit "$acceptance_exit"
