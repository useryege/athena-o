-- +goose Up
DELETE FROM account_module_access
WHERE module IN ('sports_live', 'sports_history', 'world_cup_corners');

ALTER TABLE account_module_access DROP CONSTRAINT account_module_access_module_check;
ALTER TABLE account_module_access ADD CONSTRAINT account_module_access_module_check
CHECK (module IN ('market_radar', 'managed_oo', 'worm_markets', 'worm_trading',
                  'token', 'solana', 'wallet', 'trader_sync'));

ALTER TABLE account_module_access DROP CONSTRAINT account_module_access_max_level_check;
ALTER TABLE account_module_access ADD CONSTRAINT account_module_access_max_level_check
CHECK (access_level <> 'read_write' OR module IN
       ('managed_oo', 'worm_trading', 'token', 'wallet', 'trader_sync'));

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
  RAISE EXCEPTION 'sports removal is irreversible; use the validated current version';
END $$;
-- +goose StatementEnd
