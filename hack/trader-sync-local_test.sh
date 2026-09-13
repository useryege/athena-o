#!/usr/bin/env bash
set -euo pipefail
repo_dir="$(cd "$(dirname "$0")/.." && pwd)"
probe_dir="$(mktemp -d)"
trap 'rm -rf "$probe_dir"' EXIT
printf '#!/bin/sh\nprintf "default via 192.0.2.1 dev fixture\\n"\n' > "$probe_dir/ip"
printf '#!/bin/sh\nexit 1\n' > "$probe_dir/grep"
chmod +x "$probe_dir/ip" "$probe_dir/grep"
# Expand these variables in the child shell after trader-sync-local.sh sets them.
# shellcheck disable=SC2016
probe='printf "%s|%s" "${ATHENA_TRADER_SYNC_PROXY_URL+x}" "${ATHENA_TRADER_SYNC_PROXY_URL-}"'
check() {
 local expected="$1"; shift
 local actual
 actual="$(env -i PATH="$probe_dir:/usr/bin:/bin" "$@" bash "$repo_dir/hack/trader-sync-local.sh" sh -c "$probe")"
 if [[ "$actual" != "$expected" ]]; then printf 'FAIL expected <%s>, got <%s>\n' "$expected" "$actual"; exit 1; fi
}
check 'x|http://192.0.2.1:10809' WSL_DISTRO_NAME=fixture
check 'x|' WSL_DISTRO_NAME=fixture ATHENA_TRADER_SYNC_PROXY_URL=
check 'x|http://explicit.test:8080' WSL_DISTRO_NAME=fixture ATHENA_TRADER_SYNC_PROXY_URL=http://explicit.test:8080
check '|' HTTP_PROXY=http://global.invalid ATHENA_TOKEN_NODE_WS_PROXY_URL=http://token.invalid
printf 'PASS local post-dotenv proxy: WSL default, explicit empty/value, non-WSL direct\n'
