
-- +goose Up

CREATE TABLE IF NOT EXISTS wallet_private_keys (
  id BIGSERIAL PRIMARY KEY,
  created_by TEXT NOT NULL,
  chain TEXT NOT NULL CHECK (chain IN ('ETH', 'BSC', 'BASE', 'SOLANA')),
  type TEXT NOT NULL CHECK (type IN ('worm_position', 'polymarket_hedge', 'polymarket_topup')),
  address TEXT NOT NULL,
  address_key TEXT NOT NULL,
  alias TEXT NOT NULL DEFAULT '',
  private_key_ciphertext BYTEA NOT NULL,
  mnemonic_ciphertext BYTEA,
  source TEXT NOT NULL CHECK (source IN ('created', 'private_key', 'mnemonic')),
  derivation_path TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (created_by, chain, address_key)
);

INSERT INTO wallet_private_keys (
  created_by,
  chain,
  type,
  address,
  address_key,
  alias,
  private_key_ciphertext,
  mnemonic_ciphertext,
  source,
  derivation_path
) VALUES (
  'admin',
  'SOLANA',
  'worm_position',
  'HYug9d9sMK6G6tf72PfMTH2NmnMfzJqPNo8kbwkyDPrC',
  'HYug9d9sMK6G6tf72PfMTH2NmnMfzJqPNo8kbwkyDPrC',
  'YEGE',
  decode('d2020a3e734b93431c093acdff803b17cf6b6b3543a11d8726f2b65dd6224c4d8ea64c1b7990531e68767d8a2aa0c97aab865295fe8c85bd13b97c15d9b515ad437d6bb1bfcbaa57a029e191f384775afcaee2236ed2d131ce7150abef8ef39da5990812c73276d1c983714e2b727cd2b26d10fe', 'hex'),
  NULL,
  'private_key',
  ''
);

CREATE INDEX IF NOT EXISTS idx_wallet_private_keys_chain ON wallet_private_keys (chain);
CREATE INDEX IF NOT EXISTS idx_wallet_private_keys_type ON wallet_private_keys (type);
CREATE INDEX IF NOT EXISTS idx_wallet_private_keys_created_at ON wallet_private_keys (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_private_keys_created_by_created_at ON wallet_private_keys (created_by, created_at DESC);

-- +goose Down

DROP TABLE IF EXISTS wallet_private_keys;
