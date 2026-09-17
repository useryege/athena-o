#!/usr/bin/env bash
# Legacy Solana profiles are stop-only. New runs use devruntime ownership.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
action="${1:-start}"
profile="${2:-solana-discovery}"
case "$profile" in solana-discovery|solana-preview) ;; *) echo 'Unknown Solana profile' >&2;exit 2;; esac
state="$root/.run/$profile"
procfile="$root/Procfile.$profile"

# /proc comm may contain spaces; fields below begin after its closing parenthesis.
process_info() {
  local stat
  { IFS= read -r stat < "/proc/$1/stat"; } 2>/dev/null || return 1
  read -r -a fields <<< "${stat##*) }"
  process_state="${fields[0]}"
  process_sid="${fields[3]}"
  process_ticks="${fields[19]}"
}

# PID is also the SID created by setsid; start time prevents accepting a reused PID.
# Goreman puts every service in a different PGID, so the SID is the cleanup boundary.
session_snapshot() {
  local member cwd command
  members=()
  if process_info "$pid"; then
    [[ "$process_ticks" == "$ticks" && "$process_sid" == "$pid" ]] || return 1
    if [[ "$process_state" != Z ]]; then
      command="$( { tr '\0' ' ' < "/proc/$pid/cmdline"; } 2>/dev/null)" || command=""
      if [[ -n "$command" ]]; then
        [[ "$command" == *goreman* && "$command" == *"$procfile"* ]] || return 1
      fi
    fi
  fi
  while read -r member; do
    process_info "$member" || continue
    [[ "$process_state" != Z ]] || continue
    [[ "$process_sid" == "$pid" && "$process_ticks" -ge "$ticks" ]] || return 1
    cwd="$(readlink "/proc/$member/cwd" 2>/dev/null)" || continue
    case "$cwd" in "$root"|"$root/"*) ;; *) return 1;; esac
    members+=("$member:$process_ticks")
  done < <(ps -eo pid=,sid= | awk -v sid="$pid" '$2 == sid { print $1 }')
}

signal_members() {
  local entry member expected signal="$1"
  for entry in "${members[@]}"; do
    member="${entry%%:*}"; expected="${entry#*:}"
    process_info "$member" || continue
    [[ "$process_sid" == "$pid" && "$process_ticks" == "$expected" ]] || continue
    kill "-$signal" "$member" 2>/dev/null || true
  done
}

stop() (
  # Serialize controller EXIT and external stop, retaining this lock across runs.
  [[ -d "$root/.run" ]] || return 0
  exec 9> "$root/.run/$profile.lock"
  flock -x 9
  [[ -d "$state" ]] || return 0
  [[ -f "$state/process" ]] || { echo "Missing process identity: $state" >&2; return 1; }
  read -r pid ticks < "$state/process"
  # A late controller cleanup must not stop a subsequent run of this profile.
  if [[ -n "${1:-}" && ( "$pid" != "$1" || "$ticks" != "$2" ) ]]; then return 0; fi
  if [[ ! "$pid" =~ ^[0-9]+$ || ! "$ticks" =~ ^[0-9]+$ ]] || ! session_snapshot; then
    echo "Refusing to stop unverified process; inspect $state/process" >&2
    return 1
  fi
  signal_members TERM
  for ((i=0;i<100;i++)); do
    session_snapshot || return 1
    if ((${#members[@]} == 0)); then printf 'stopped\n' > "$state/status"; return 0; fi
    sleep 0.1
  done
  # Revalidate each remaining member before forcing shutdown; never signal a global port.
  signal_members KILL
  for ((i=0;i<20;i++)); do
    session_snapshot || return 1
    if ((${#members[@]} == 0)); then printf 'stopped\n' > "$state/status"; return 0; fi
    sleep 0.1
  done
  echo "Solana session did not stop; retaining $state/process" >&2
  return 1
)

case "$action" in
start)
  echo 'Legacy Solana profile startup is retired; use make run-service SERVICE=solana-discovery.' >&2
  exit 2
  ;;
stop) stop "" "" ;;
*) echo 'usage: solana-local.sh {start|stop} {solana-discovery|solana-preview}' >&2;exit 2;;
esac
