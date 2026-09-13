#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
# shellcheck source=hack/tool-versions.sh
. "$root/hack/tool-versions.sh"
bin_dir=${AI_DEV_TOOLS_BIN_DIR:-$root/dist}
artifact_dir=${AI_DEV_TOOLS_ARTIFACT_DIR:-$root/.tmp/ai-dev-tools}

usage() {
  echo "Usage: $0 {install|check|lint-shell|vuln-check}" >&2
  exit 2
}

require_tool() {
  local binary=$1
  if [[ ! -x "$bin_dir/$binary" ]]; then
    echo "Missing $bin_dir/$binary. Run make install-ai-dev-tools." >&2
    exit 1
  fi
}

new_artifact() {
  mkdir -p "$artifact_dir"
  mktemp "$artifact_dir/$1-$(date -u +%Y%m%dT%H%M%SZ)-XXXXXX.log"
}

check() {
  local failed=0
  if [[ -x "$bin_dir/shellcheck" ]] &&
     "$bin_dir/shellcheck" --version | grep -Fq "version: $shellcheck_version"; then
    echo "ShellCheck $shellcheck_version: ready ($bin_dir/shellcheck)"
  else
    echo "ShellCheck $shellcheck_version: missing or wrong version; run make install-ai-dev-tools" >&2
    failed=1
  fi
  if [[ -x "$bin_dir/grpcurl" ]] &&
     "$bin_dir/grpcurl" -version 2>&1 | grep -Fq "v$grpcurl_version"; then
    echo "grpcurl $grpcurl_version: ready ($bin_dir/grpcurl)"
  else
    echo "grpcurl $grpcurl_version: missing or wrong version; run make install-ai-dev-tools" >&2
    failed=1
  fi
  if [[ -x "$bin_dir/govulncheck" ]] &&
     "$bin_dir/govulncheck" -version 2>&1 | grep -Fq "v$govulncheck_version"; then
    echo "govulncheck $govulncheck_version: ready ($bin_dir/govulncheck)"
  else
    echo "govulncheck $govulncheck_version: missing or wrong version; run make install-ai-dev-tools" >&2
    failed=1
  fi
  if command -v psql >/dev/null 2>&1 && command -v pg_isready >/dev/null 2>&1 &&
     psql --version | grep -Fq "PostgreSQL) $postgresql_client_major." &&
     pg_isready --version | grep -Fq "PostgreSQL) $postgresql_client_major."; then
    echo "PostgreSQL client $postgresql_client_major: ready ($(command -v psql))"
  else
    echo "PostgreSQL client $postgresql_client_major: missing or wrong version; run make install-ai-dev-tools" >&2
    failed=1
  fi
  if command -v node >/dev/null 2>&1 && command -v yarn >/dev/null 2>&1 &&
     node -e 'const version = process.versions.node; if (!/^\d+\.\d+\.\d+$/.test(version)) process.exit(1); const [major, minor, patch] = version.split(".").map(Number); if (major !== 24 || minor < 14 || (minor === 14 && patch < 1)) process.exit(1)' &&
     [[ $(yarn --version) == 1.22.22 ]] &&
     (cd "$root/ui" && AXE_VERSION="$axe_core_playwright_version" node -e 'const p = require("./node_modules/@axe-core/playwright/package.json"); require("@axe-core/playwright"); if (p.version !== process.env.AXE_VERSION) process.exit(1)') 2>/dev/null; then
    echo "@axe-core/playwright $axe_core_playwright_version: ready ($root/ui/node_modules)"
  else
    echo "@axe-core/playwright $axe_core_playwright_version: missing, wrong version, or cannot load with Node >=24.14.1 <25 and Yarn 1.22.22; run make install-ai-dev-tools" >&2
    failed=1
  fi
  return "$failed"
}

lint_shell() {
  require_tool shellcheck
  local log
  log=$(new_artifact shellcheck)
  echo "ShellCheck output: $log"
  cd "$root"
  local -a scripts
  mapfile -d '' -t scripts < <(rg --files -0 -g '*.sh' hack ui/scripts)
  if ((${#scripts[@]} == 0)); then
    echo 'No project shell scripts found under hack/ or ui/scripts/.' >&2
    return 1
  fi
  "$bin_dir/shellcheck" -x -P . "${scripts[@]}" 2>&1 | tee "$log"
}

vuln_check() {
  require_tool govulncheck
  local log
  log=$(new_artifact govulncheck)
  echo "govulncheck output: $log"
  cd "$root"
  GOTOOLCHAIN=local "$bin_dir/govulncheck" ./... 2>&1 | tee "$log"
}

case ${1:-} in
  install) "$root/hack/install.sh" ai-dev-tools ;;
  check) check ;;
  lint-shell) lint_shell ;;
  vuln-check) vuln_check ;;
  *) usage ;;
esac
