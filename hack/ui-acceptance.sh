#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
if [[ -n "${ATHENA_UI_ACCEPTANCE_NODE:-}" ]]; then
  selected_node="$ATHENA_UI_ACCEPTANCE_NODE"
elif command -v node >/dev/null 2>&1; then
  selected_node="$(command -v node)"
else
  nvm_dir="${NVM_DIR:-$HOME/.nvm}"
  if [[ ! -r "$nvm_dir/nvm.sh" ]]; then
    printf '%s\n' 'Node unavailable: set ATHENA_UI_ACCEPTANCE_NODE to a WSL Node executable.' >&2
    exit 1
  fi
  # Resolve NVM aliases (including lts/*) in a subshell; do not change the caller PATH.
  # NVM is installed outside this repository at the user's configured location.
  # shellcheck source=/dev/null
  selected_node="$(set +u; . "$nvm_dir/nvm.sh" --no-use; nvm which default)" || {
    printf '%s\n' 'Node unavailable: NVM default does not resolve to an installed version.' >&2
    exit 1
  }
fi
if [[ ! -f "$selected_node" || ! -x "$selected_node" ]]; then
  printf '%s\n' "Node unavailable: ATHENA_UI_ACCEPTANCE_NODE/selected Node is not executable: $selected_node" >&2
  exit 1
fi
exec "$selected_node" "$repo_root/ui/scripts/acceptance-runner.mjs" "$@"
