
-- +goose Up

CREATE TABLE IF NOT EXISTS wallets (
  id BIGSERIAL PRIMARY KEY,
  owner_account_id UUID NOT NULL,
  wallet_type TEXT NOT NULL CHECK (wallet_type IN ('EVM', 'SOLANA')),
  address TEXT NOT NULL,
  address_key TEXT NOT NULL,
  remark TEXT NOT NULL CHECK (char_length(remark) BETWEEN 1 AND 50 AND remark = btrim(remark)),
  source TEXT NOT NULL CHECK (source IN ('created', 'imported')),
  private_key_ciphertext BYTEA NOT NULL,
  avatar_preset_id TEXT NOT NULL DEFAULT '' CHECK (
    avatar_preset_id IN (
      '',
      'star-violet',
      'bolt-blue',
      'gem-cyan',
      'leaf-green',
      'sun-amber',
      'flame-orange',
      'heart-rose',
      'moon-indigo'
    )
  ),
  avatar_object_key TEXT NOT NULL DEFAULT '',
  avatar_content_type TEXT NOT NULL DEFAULT '',
  avatar_etag TEXT NOT NULL DEFAULT '',
  avatar_size_bytes BIGINT NOT NULL DEFAULT 0,
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (avatar_preset_id = '' OR avatar_object_key = ''),
  CHECK (
    (
      avatar_object_key = ''
      AND avatar_content_type = ''
      AND avatar_etag = ''
      AND avatar_size_bytes = 0
    )
    OR (
      avatar_object_key <> ''
      AND avatar_content_type IN ('image/jpeg', 'image/png', 'image/webp')
      AND avatar_etag <> ''
      AND avatar_size_bytes > 0
      AND avatar_size_bytes <= 2097152
    )
  ),
  UNIQUE (owner_account_id, wallet_type, address_key)
);

CREATE INDEX IF NOT EXISTS idx_wallets_owner_created_at
  ON wallets (owner_account_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_wallets_owner_type_created_at
  ON wallets (owner_account_id, wallet_type, created_at DESC, id DESC);

-- +goose Down

DROP TABLE IF EXISTS wallets;
