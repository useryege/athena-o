
-- +goose Up

CREATE TABLE IF NOT EXISTS wallet_private_keys (
  id BIGSERIAL PRIMARY KEY,
  owner_account_id UUID,
  system_owned BOOLEAN NOT NULL DEFAULT FALSE,
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
  CHECK (
    (system_owned AND owner_account_id IS NULL)
    OR (NOT system_owned AND owner_account_id IS NOT NULL)
  )
);

INSERT INTO wallet_private_keys (
  owner_account_id,
  system_owned,
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
  NULL,
  TRUE,
  'SOLANA',
  'worm_position',
  'HYug9d9sMK6G6tf72PfMTH2NmnMfzJqPNo8kbwkyDPrC',
  'HYug9d9sMK6G6tf72PfMTH2NmnMfzJqPNo8kbwkyDPrC',
  'YEGE_01',
  decode('d2020a3e734b93431c093acdff803b17cf6b6b3543a11d8726f2b65dd6224c4d8ea64c1b7990531e68767d8a2aa0c97aab865295fe8c85bd13b97c15d9b515ad437d6bb1bfcbaa57a029e191f384775afcaee2236ed2d131ce7150abef8ef39da5990812c73276d1c983714e2b727cd2b26d10fe', 'hex'),
  NULL,
  'private_key',
  ''
), (
  NULL,
  TRUE,
  'SOLANA',
  'worm_position',
  'DbYhbC6FdyNaMvy5aBPo2ntsCR6WwFqcfuatQqZNHVoT',
  'DbYhbC6FdyNaMvy5aBPo2ntsCR6WwFqcfuatQqZNHVoT',
  'YEGE_02',
  decode('755bd3f2d1c6b2e9cc252675ed0f14b0bcc514082450c61ba7fe7da5af078af7c7f0d1d6093f584a5dec7eb59a86488b106b456791dbd4df2f2780847a5f7adb376bde3d696a5462dfce1d67c53d3528d0d5247ce09b2dd7bd4ccab262944c5e39da9e2afcf4dc327b0a4346a7ac3fee6d83175c', 'hex'),
  NULL,
  'private_key',
  ''
);


CREATE INDEX IF NOT EXISTS idx_wallet_private_keys_chain ON wallet_private_keys (chain);
CREATE INDEX IF NOT EXISTS idx_wallet_private_keys_type ON wallet_private_keys (type);
CREATE INDEX IF NOT EXISTS idx_wallet_private_keys_created_at ON wallet_private_keys (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_private_keys_owner_created_at ON wallet_private_keys (owner_account_id, created_at DESC)
  WHERE NOT system_owned;
CREATE UNIQUE INDEX IF NOT EXISTS uq_wallet_private_keys_owner_chain_address
  ON wallet_private_keys (owner_account_id, chain, address_key)
  WHERE NOT system_owned;
CREATE UNIQUE INDEX IF NOT EXISTS uq_wallet_private_keys_system_chain_address
  ON wallet_private_keys (chain, address_key)
  WHERE system_owned;

-- +goose Down

DROP TABLE IF EXISTS wallet_private_keys;
