-- name: ListAccountAccessOverrideHeads :many
SELECT account_name, login_enabled, revision
FROM account_access_override
ORDER BY account_name;

-- name: ListAccountModuleAccessOverrides :many
SELECT account_name, module, access_level
FROM account_module_access_override
ORDER BY account_name, module;

-- name: CreateAccountAccessOverrideHead :one
INSERT INTO account_access_override (
  account_name,
  login_enabled,
  revision
)
VALUES (
  sqlc.arg(account_name)::text,
  sqlc.arg(login_enabled)::boolean,
  1
)
ON CONFLICT (account_name) DO NOTHING
RETURNING account_name, login_enabled, revision;

-- name: UpdateAccountAccessOverrideHead :one
UPDATE account_access_override
SET login_enabled = sqlc.arg(login_enabled)::boolean,
    revision = account_access_override.revision + 1,
    updated_at = NOW()
WHERE account_name = sqlc.arg(account_name)::text
  AND sqlc.arg(expected_revision)::bigint > 0
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING account_name, login_enabled, revision;

-- name: UpsertAccountModuleAccessOverrides :exec
INSERT INTO account_module_access_override (
  account_name,
  module,
  access_level
)
VALUES
  (sqlc.arg(account_name)::text, 'market_radar', sqlc.arg(market_radar_access_level)::text),
  (sqlc.arg(account_name)::text, 'sports_live', sqlc.arg(sports_live_access_level)::text),
  (sqlc.arg(account_name)::text, 'sports_history', sqlc.arg(sports_history_access_level)::text),
  (sqlc.arg(account_name)::text, 'managed_oo', sqlc.arg(managed_oo_access_level)::text),
  (sqlc.arg(account_name)::text, 'worm_markets', sqlc.arg(worm_markets_access_level)::text),
  (sqlc.arg(account_name)::text, 'fifa_market_dashboard', sqlc.arg(fifa_market_dashboard_access_level)::text),
  (sqlc.arg(account_name)::text, 'world_cup_corners', sqlc.arg(world_cup_corners_access_level)::text),
  (sqlc.arg(account_name)::text, 'token', sqlc.arg(token_access_level)::text),
  (sqlc.arg(account_name)::text, 'wallet', sqlc.arg(wallet_access_level)::text),
  (sqlc.arg(account_name)::text, 'notifications', sqlc.arg(notifications_access_level)::text)
ON CONFLICT (account_name, module) DO UPDATE
SET access_level = EXCLUDED.access_level;
