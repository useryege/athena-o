
-- +goose Up

CREATE TABLE IF NOT EXISTS wallet_private_keys (
  id BIGSERIAL PRIMARY KEY,
  chain TEXT NOT NULL CHECK (chain IN ('ETH', 'BSC', 'BASE', 'SOLANA')),
  address TEXT NOT NULL,
  address_key TEXT NOT NULL,
  alias TEXT NOT NULL DEFAULT '',
  private_key_ciphertext BYTEA NOT NULL,
  mnemonic_ciphertext BYTEA,
  source TEXT NOT NULL CHECK (source IN ('created', 'private_key', 'mnemonic')),
  derivation_path TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (chain, address_key)
);

CREATE INDEX IF NOT EXISTS idx_wallet_private_keys_chain ON wallet_private_keys (chain);
CREATE INDEX IF NOT EXISTS idx_wallet_private_keys_created_at ON wallet_private_keys (created_at DESC);

CREATE TABLE IF NOT EXISTS wallet_blacklist (
  wallet BYTEA PRIMARY KEY,
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT wallet_blacklist_wallet_len CHECK (length(wallet) = 20)
);

-- +goose Down

DROP TABLE IF EXISTS wallet_blacklist;
DROP TABLE IF EXISTS wallet_private_keys;
