#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT
mkdir -p "${tmp}/bin"
cat >"${tmp}/bin/ssh" <<'SSH'
#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == -- ]]
shift
printf '%s' "$1" >"${TEST_HOST_LOG}"
shift
# OpenSSH sends a command string to the remote login shell, which parses it once.
exec /bin/sh -c "$*"
SSH
chmod +x "${tmp}/bin/ssh"
export PATH="${tmp}/bin:${PATH}" TEST_HOST_LOG="${tmp}/host"

# shellcheck source=hack/lib/ssh-command.sh
source "${script_dir}/lib/ssh-command.sh"

set +e
bash -c 'set -u; source "$1"; ssh_exec' _ "${script_dir}/lib/ssh-command.sh" >"${tmp}/invalid.out" 2>&1
missing_all=$?
bash -c 'set -u; source "$1"; ssh_exec test@example' _ "${script_dir}/lib/ssh-command.sh" >"${tmp}/invalid.out" 2>&1
missing_command=$?
set -e
[[ ${missing_all} -eq 2 && ${missing_command} -eq 2 ]]
ssh_exec '-option-like-host' true
[[ "$(cat "${tmp}/host")" == '-option-like-host' ]]

# These are literal hostile arguments, not local expansions.
# shellcheck disable=SC2016
args=('' 'white space' "single'quote" 'dollar$HOME' 'back`tick`' "\$(touch ${tmp}/injected)" '*?[x]' $'first\nsecond')
printf '%s\0' "${args[@]}" >"${tmp}/expected"
ssh_exec 'test@example' bash -c 'printf "%s\0" "$@"' _ "${args[@]}" >"${tmp}/actual"
cmp "${tmp}/expected" "${tmp}/actual"
[[ "$(cat "${tmp}/host")" == 'test@example' ]]
[[ ! -e "${tmp}/injected" ]]

printf '\000\001\377\narchive\000' >"${tmp}/binary"
ssh_exec 'test@example' cat <"${tmp}/binary" >"${tmp}/roundtrip"
cmp "${tmp}/binary" "${tmp}/roundtrip"

set +e
ssh_exec 'test@example' sh -c 'printf remote-error >&2; exit 37' >"${tmp}/stdout" 2>"${tmp}/stderr"
status=$?
set -e
[[ ${status} -eq 37 ]]
[[ ! -s "${tmp}/stdout" ]]
[[ "$(cat "${tmp}/stderr")" == 'remote-error' ]]
echo 'ssh-command tests passed'
