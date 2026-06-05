
-- +goose Up

CREATE TABLE IF NOT EXISTS wallet_blacklist (
  wallet BYTEA PRIMARY KEY,
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT wallet_blacklist_wallet_len CHECK (length(wallet) = 20)
);

-- +goose Down

DROP TABLE IF EXISTS wallet_blacklist;
