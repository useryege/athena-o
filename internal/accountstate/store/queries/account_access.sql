-- name: ListAccountAccessHeads :many
SELECT account_name,
       login_enabled,
       api_key_enabled,
       profit_sharing_enabled,
       revision
FROM account_access
ORDER BY account_name;

-- name: GetAccountAccessHead :one
SELECT account_name,
       login_enabled,
       api_key_enabled,
       profit_sharing_enabled,
       revision
FROM account_access
WHERE account_name = sqlc.arg(account_name)::text;

-- name: ListAccountModuleAccess :many
SELECT account_name, module, access_level
FROM account_module_access
ORDER BY account_name, module;

-- name: ListAccountModuleAccessByAccount :many
SELECT account_name, module, access_level
FROM account_module_access
WHERE account_name = sqlc.arg(account_name)::text
ORDER BY module;

-- name: UpdateAccountAccessHead :one
UPDATE account_access
SET login_enabled = sqlc.arg(login_enabled)::boolean,
    api_key_enabled = sqlc.arg(api_key_enabled)::boolean,
    profit_sharing_enabled = sqlc.arg(profit_sharing_enabled)::boolean,
    revision = account_access.revision + 1,
    updated_at = NOW()
WHERE account_name = sqlc.arg(account_name)::text
  AND account_name <> 'admin'
  AND sqlc.arg(expected_revision)::bigint > 0
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING account_name,
          login_enabled,
          api_key_enabled,
          profit_sharing_enabled,
          revision;

-- name: ReplaceAccountModuleAccess :execrows
UPDATE account_module_access
SET access_level = CASE module
  WHEN 'market_radar' THEN sqlc.arg(market_radar_access_level)::text
  WHEN 'sports_live' THEN sqlc.arg(sports_live_access_level)::text
  WHEN 'sports_history' THEN sqlc.arg(sports_history_access_level)::text
  WHEN 'managed_oo' THEN sqlc.arg(managed_oo_access_level)::text
  WHEN 'worm_markets' THEN sqlc.arg(worm_markets_access_level)::text
  WHEN 'fifa_market_dashboard' THEN sqlc.arg(fifa_market_dashboard_access_level)::text
  WHEN 'world_cup_corners' THEN sqlc.arg(world_cup_corners_access_level)::text
  WHEN 'token' THEN sqlc.arg(token_access_level)::text
  WHEN 'wallet' THEN sqlc.arg(wallet_access_level)::text
  WHEN 'notifications' THEN sqlc.arg(notifications_access_level)::text
  ELSE access_level
END
WHERE account_name = sqlc.arg(account_name)::text;
