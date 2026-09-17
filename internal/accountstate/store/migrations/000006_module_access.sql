-- +goose Up
CREATE TABLE athena_module_access_setting (
    module_key TEXT PRIMARY KEY CHECK (module_key IN ('trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm')),
    is_open BOOLEAN NOT NULL DEFAULT FALSE,
    updated_by_account_id UUID REFERENCES athena_account(account_id),
    updated_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE athena_module_access_setting;
