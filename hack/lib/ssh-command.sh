#!/usr/bin/env bash

# OpenSSH passes the command to the remote login shell as one string. Quote each
# argument for that single POSIX shell parse, including empty and multiline data.
ssh_exec() {
  if (($# < 2)); then
    printf 'ssh_exec requires host and command\n' >&2
    return 2
  fi
  local host="$1"
  shift
  local remote_command='' argument
  for argument in "$@"; do
    remote_command+=" '${argument//\'/\'\\\'\'}'"
  done
  # The client-side expansion is the POSIX-quoted remote command by design.
  # shellcheck disable=SC2029
  ssh -- "${host}" "${remote_command# }"
}
