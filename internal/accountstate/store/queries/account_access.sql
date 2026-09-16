-- name: ListAccountAccessHeads :many
SELECT access.account_id,
       account.administrator,
       access.login_enabled,
       access.api_key_enabled,
       access.profit_sharing_enabled,
       access.revision
FROM account_access AS access
JOIN athena_account AS account USING (account_id)
ORDER BY access.account_id;

-- name: GetAccountAccessHead :one
SELECT access.account_id,
       account.administrator,
       access.login_enabled,
       access.api_key_enabled,
       access.profit_sharing_enabled,
       access.revision
FROM account_access AS access
JOIN athena_account AS account USING (account_id)
WHERE access.account_id = sqlc.arg(account_id)::uuid;

-- name: ListAccountModuleAccess :many
SELECT account_id, module, access_level
FROM account_module_access
ORDER BY account_id, module;

-- name: ListAccountModuleAccessByAccount :many
SELECT account_id, module, access_level
FROM account_module_access
WHERE account_id = sqlc.arg(account_id)::uuid
ORDER BY module;

-- name: UpdateAccountAccessHead :one
UPDATE account_access AS access
SET login_enabled = sqlc.arg(login_enabled)::boolean,
    api_key_enabled = sqlc.arg(api_key_enabled)::boolean,
    profit_sharing_enabled = sqlc.arg(profit_sharing_enabled)::boolean,
    revision = access.revision + 1,
    updated_at = NOW()
WHERE access.account_id = sqlc.arg(account_id)::uuid
  AND EXISTS (
    SELECT 1
    FROM athena_account AS account
    WHERE account.account_id = access.account_id
      AND NOT account.administrator
  )
  AND sqlc.arg(expected_revision)::bigint > 0
  AND access.revision = sqlc.arg(expected_revision)::bigint
RETURNING access.account_id,
          access.login_enabled,
          access.api_key_enabled,
          access.profit_sharing_enabled,
          access.revision;

-- name: ReplaceAccountModuleAccess :execrows
UPDATE account_module_access AS module_access
SET access_level = CASE module_access.module
  WHEN 'market_radar' THEN sqlc.arg(market_radar_access_level)::text
  WHEN 'managed_oo' THEN sqlc.arg(managed_oo_access_level)::text
  WHEN 'worm_markets' THEN sqlc.arg(worm_markets_access_level)::text
  WHEN 'worm_trading' THEN sqlc.arg(worm_trading_access_level)::text
  WHEN 'token' THEN sqlc.arg(token_access_level)::text
  WHEN 'solana' THEN sqlc.arg(solana_access_level)::text
  WHEN 'wallet' THEN sqlc.arg(wallet_access_level)::text
  WHEN 'trader_sync' THEN sqlc.arg(trader_sync_access_level)::text
  ELSE module_access.access_level
END
WHERE module_access.account_id = sqlc.arg(account_id)::uuid
  AND EXISTS (
    SELECT 1
    FROM athena_account AS account
    WHERE account.account_id = module_access.account_id
      AND NOT account.administrator
  );
