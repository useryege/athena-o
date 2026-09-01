-- +goose Up

CREATE TABLE worm_trading_wallet_selections (
  owner_account_id UUID PRIMARY KEY,
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  selected_wallet_count INTEGER NOT NULL CHECK (
    selected_wallet_count BETWEEN 0 AND 20
  ),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CHECK (updated_at >= created_at)
);

CREATE TABLE worm_trading_wallet_selection_items (
  owner_account_id UUID NOT NULL
    REFERENCES worm_trading_wallet_selections(owner_account_id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL CHECK (ordinal BETWEEN 1 AND 20),
  wallet_id BIGINT NOT NULL CHECK (wallet_id > 0),
  address TEXT NOT NULL CHECK (
    address = btrim(address)
    AND char_length(address) BETWEEN 32 AND 64
  ),
  selected_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (owner_account_id, ordinal),
  CONSTRAINT worm_trading_wallet_selection_items_owner_wallet_unique
    UNIQUE (owner_account_id, wallet_id),
  CONSTRAINT worm_trading_wallet_selection_items_owner_address_unique
    UNIQUE (owner_account_id, address)
);

CREATE TABLE worm_trading_wallet_retirements (
  owner_account_id UUID NOT NULL
    REFERENCES worm_trading_wallet_selections(owner_account_id) ON DELETE CASCADE,
  wallet_id BIGINT NOT NULL CHECK (wallet_id > 0),
  address TEXT NOT NULL CHECK (
    address = btrim(address)
    AND char_length(address) BETWEEN 32 AND 64
  ),
  prior_ordinal INTEGER NOT NULL CHECK (prior_ordinal > 0),
  retired_from_revision BIGINT NOT NULL CHECK (retired_from_revision > 0),
  retired_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (owner_account_id, wallet_id),
  CONSTRAINT worm_trading_wallet_retirements_owner_address_unique
    UNIQUE (owner_account_id, address),
  CHECK (updated_at >= retired_at)
);

CREATE INDEX worm_trading_wallet_retirements_owner_retired_idx
  ON worm_trading_wallet_retirements (owner_account_id, retired_at, wallet_id);

TRUNCATE TABLE worm_execution_plans CASCADE;

ALTER TABLE worm_execution_plans
  ADD COLUMN wallet_selection_revision BIGINT NOT NULL CHECK (
    wallet_selection_revision > 0
  );

-- +goose Down

ALTER TABLE worm_execution_plans
  DROP COLUMN IF EXISTS wallet_selection_revision;

DROP TABLE IF EXISTS worm_trading_wallet_retirements;
DROP TABLE IF EXISTS worm_trading_wallet_selection_items;
DROP TABLE IF EXISTS worm_trading_wallet_selections;
