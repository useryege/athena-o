-- +goose Up

ALTER TABLE account_module_access
  DROP CONSTRAINT account_module_access_module_check;
ALTER TABLE account_module_access
  ADD CONSTRAINT account_module_access_module_check CHECK (module IN (
    'market_radar', 'sports_live', 'sports_history', 'managed_oo',
    'worm_markets', 'worm_trading', 'world_cup_corners', 'token',
    'solana', 'wallet', 'trader_sync'
  ));

INSERT INTO account_module_access (account_id, module, access_level)
SELECT account_id, 'solana', 'none'
FROM account_access
ON CONFLICT (account_id, module) DO NOTHING;

-- +goose Down

DELETE FROM account_module_access WHERE module = 'solana';
ALTER TABLE account_module_access
  DROP CONSTRAINT account_module_access_module_check;
ALTER TABLE account_module_access
  ADD CONSTRAINT account_module_access_module_check CHECK (module IN (
    'market_radar', 'sports_live', 'sports_history', 'managed_oo',
    'worm_markets', 'worm_trading', 'world_cup_corners', 'token',
    'wallet', 'trader_sync'
  ));
