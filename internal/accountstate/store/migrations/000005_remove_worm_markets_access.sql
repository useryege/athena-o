-- +goose Up
WITH affected_accounts AS MATERIALIZED (
  SELECT account_id
  FROM account_module_access
  WHERE module = 'worm_markets'
), gated_accounts AS MATERIALIZED (
  SELECT account_id,
         pg_advisory_xact_lock(hashtextextended('athena:account:' || account_id::text, 0)) AS gate
  FROM affected_accounts
  ORDER BY account_id
), locked_accounts AS MATERIALIZED (
  SELECT access.account_id
  FROM account_access AS access
  JOIN gated_accounts USING (account_id)
  ORDER BY access.account_id
  FOR UPDATE OF access
), affected AS (
  DELETE FROM account_module_access AS module_access
  WHERE module_access.module = 'worm_markets'
    AND module_access.account_id IN (SELECT account_id FROM locked_accounts)
  RETURNING module_access.account_id
)
UPDATE account_access AS access
SET revision = access.revision + 1,
    updated_at = NOW()
WHERE access.account_id IN (SELECT account_id FROM affected);

ALTER TABLE account_module_access DROP CONSTRAINT account_module_access_module_check;
ALTER TABLE account_module_access ADD CONSTRAINT account_module_access_module_check
CHECK (module IN ('market_radar', 'managed_oo', 'worm_trading',
                  'token', 'solana', 'wallet', 'trader_sync'));

ALTER TABLE account_module_access DROP CONSTRAINT account_module_access_max_level_check;
ALTER TABLE account_module_access ADD CONSTRAINT account_module_access_max_level_check
CHECK (access_level <> 'read_write' OR module IN
       ('managed_oo', 'worm_trading', 'token', 'wallet', 'trader_sync'));

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
  RAISE EXCEPTION 'Worm Markets removal is irreversible; use the validated current version';
END $$;
-- +goose StatementEnd
