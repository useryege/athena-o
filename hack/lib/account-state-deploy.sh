#!/usr/bin/env bash
# Sourced by local/remote deploy entrypoints after defining compose().
# A failed verify can also mean drift or an unavailable DB: up/verify must both
# succeed in an explicitly coordinated maintenance window before any restart.
account_state_prepare() {
  ACCOUNT_STATE_CHANGED=false
  if compose --profile tools run --rm --no-deps athena-account-state-migrate verify; then
    return 0
  fi
  if [[ "${PROD_ACCOUNT_STATE_MAINTENANCE:-false}" != true || "${PROD_ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED:-false}" != true ]]; then
    echo 'Account-state schema is incompatible. Coordinate maintenance and confirm all consumers outside this Compose project are stopped; set PROD_ACCOUNT_STATE_MAINTENANCE=true and PROD_ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED=true.' >&2
    return 1
  fi
  compose stop -t 40 athena-server athena-notification athena-trader-sync
  local running
  running="$(compose ps --status running -q athena-server athena-notification athena-trader-sync)" || return
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
    for module in worm-markets worm-trading wallet sports-live sports-history managed-oo profit-sharing token; do
      compose --profile tools run --rm athena-migrate athena up --module "$module"
    done
  elif [[ "$MIGRATE_MODULE" != account-state ]]; then
    compose --profile tools run --rm athena-migrate athena up --module "$MIGRATE_MODULE"
  fi
}
