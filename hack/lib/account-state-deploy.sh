#!/usr/bin/env bash
# Sourced by local/remote deploy entrypoints after defining compose().
# A failed verify can also mean drift or an unavailable DB: up/verify must both
# succeed in an explicitly coordinated maintenance window before any restart.
account_state_prepare() {
  ACCOUNT_STATE_CHANGED=false
  ACCOUNT_STATE_RESTART_SERVICES=()
  if compose --profile tools run --rm --no-deps athena-account-state-migrate verify; then
    return 0
  fi
  if [[ "${PROD_ACCOUNT_STATE_MAINTENANCE:-false}" != true || "${PROD_ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED:-false}" != true ]]; then
    echo 'Account-state schema is incompatible. Coordinate maintenance and confirm all consumers outside this Compose project are stopped; set PROD_ACCOUNT_STATE_MAINTENANCE=true and PROD_ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED=true.' >&2
    return 1
  fi
  # Resolve the actual configured consumers, including profile-specific Solana.
  # Differently spelled DSNs require an operator identity check, not a guess.
  local config consumers running restart
  config="$(compose --profile '*' config --format json)" || return
  consumers="$(printf '%s' "$config" | python3 -c '
import json, sys
services = json.load(sys.stdin)["services"]
key = "ATHENA_ACCOUNT_STATE_POSTGRES_DSN"
authority = services["athena-account-state-migrate"]["environment"][key]
if not authority: raise SystemExit("Missing account-state schema authority DSN")
names = []
for name, service in services.items():
    if "tools" in service.get("profiles", []): continue
    if service.get("labels", {}).get("io.athena.account-state.consumer") != "true": continue
    dsn = service.get("environment", {}).get(key)
    if not dsn: raise SystemExit("Missing account-state DSN for " + name)
    if dsn != authority: raise SystemExit("Reconcile account-state database identity for " + name)
    names.append(name)
if not names: raise SystemExit("No managed account-state consumers identified")
print("\n".join(names))
')" || return
  local -a account_state_consumers
  mapfile -t account_state_consumers <<<"$consumers"
  restart="$(compose ps --status running --services "${account_state_consumers[@]}")" || return
  # shellcheck disable=SC2034 # Used by the sourcing TS-only deployment path.
  ACCOUNT_STATE_RESTART_SERVICES=()
  if [[ -n "$restart" ]]; then
    # shellcheck disable=SC2034 # Used by the sourcing TS-only deployment path.
    mapfile -t ACCOUNT_STATE_RESTART_SERVICES <<<"$restart"
  fi
  compose stop -t 30 "${account_state_consumers[@]}"
  running="$(compose ps --status running -q "${account_state_consumers[@]}")" || return
  if [[ -n "$running" ]]; then
    echo 'Account-state consumers have not exited; refusing migration.' >&2
    return 1
  fi
  compose --profile tools run --rm --no-deps athena-account-state-migrate up
  compose --profile tools run --rm --no-deps athena-account-state-migrate verify
  # shellcheck disable=SC2034 # Read by the sourcing deployment entrypoint.
  ACCOUNT_STATE_CHANGED=true
}

# The aggregate image never migrates account-state; its schema authority is the
# dedicated tool in TRADER_SYNC_IMAGE, including when selecting all modules.
other_schema_up() {
  local module
  if [[ "${MIGRATE_MODULE:-all}" == all ]]; then
    compose --profile tools run --rm athena-worm-trading-migrate up
    compose --profile tools run --rm athena-worm-trading-migrate verify
    for module in wallet managed-oo profit-sharing token; do
      compose --profile tools run --rm athena-migrate athena up --module "$module"
    done
  elif [[ "$MIGRATE_MODULE" == worm-trading ]]; then
    compose --profile tools run --rm athena-worm-trading-migrate up
    compose --profile tools run --rm athena-worm-trading-migrate verify
  elif [[ "$MIGRATE_MODULE" != account-state ]]; then
    compose --profile tools run --rm athena-migrate athena up --module "$MIGRATE_MODULE"
  fi
}

# Restore explicitly named services even when they use an opt-in profile.
account_state_restore() {
  if [[ "$ACCOUNT_STATE_CHANGED" == true ]] && ((${#ACCOUNT_STATE_RESTART_SERVICES[@]})); then
    compose up -d --no-deps --force-recreate "${ACCOUNT_STATE_RESTART_SERVICES[@]}"
  fi
}
