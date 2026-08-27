-- name: CreateWallet :one
INSERT INTO wallets (
  owner_account_id,
  wallet_type,
  address,
  address_key,
  remark,
  source,
  private_key_ciphertext,
  avatar_preset_id
) VALUES (
  sqlc.arg('owner_account_id')::uuid,
  sqlc.arg('wallet_type'),
  sqlc.arg('address'),
  sqlc.arg('address_key'),
  sqlc.arg('remark'),
  sqlc.arg('source'),
  sqlc.arg('private_key_ciphertext'),
  sqlc.arg('avatar_preset_id')
)
RETURNING *;

-- name: CountWallets :one
SELECT COUNT(*)::bigint
FROM wallets
WHERE owner_account_id = sqlc.arg('owner_account_id')::uuid
  AND (sqlc.narg('wallet_type')::text IS NULL OR wallet_type = sqlc.narg('wallet_type'))
  AND (
    sqlc.narg('query')::text IS NULL
    OR address ILIKE sqlc.narg('query')
    OR remark ILIKE sqlc.narg('query')
  );

-- name: ListWallets :many
SELECT
  id,
  wallet_type,
  address,
  remark,
  source,
  avatar_preset_id,
  avatar_object_key,
  revision,
  created_at,
  updated_at
FROM wallets
WHERE owner_account_id = sqlc.arg('owner_account_id')::uuid
  AND (sqlc.narg('wallet_type')::text IS NULL OR wallet_type = sqlc.narg('wallet_type'))
  AND (
    sqlc.narg('query')::text IS NULL
    OR address ILIKE sqlc.narg('query')
    OR remark ILIKE sqlc.narg('query')
  )
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: GetWallet :one
SELECT *
FROM wallets
WHERE id = sqlc.arg('id')
  AND owner_account_id = sqlc.arg('owner_account_id')::uuid;

-- name: UpdateWalletRemark :one
UPDATE wallets
SET
  remark = sqlc.arg('remark'),
  revision = revision + 1,
  updated_at = NOW()
WHERE id = sqlc.arg('id')
  AND owner_account_id = sqlc.arg('owner_account_id')::uuid
  AND revision = sqlc.arg('expected_revision')
RETURNING
  id,
  wallet_type,
  address,
  remark,
  source,
  avatar_preset_id,
  avatar_object_key,
  revision,
  created_at,
  updated_at;

-- name: UpdateWalletAvatarPreset :one
WITH current AS (
  SELECT id, avatar_object_key
  FROM wallets
  WHERE wallets.id = sqlc.arg('id')
    AND wallets.owner_account_id = sqlc.arg('owner_account_id')::uuid
    AND wallets.revision = sqlc.arg('expected_revision')
), updated AS (
  UPDATE wallets
  SET
    avatar_preset_id = sqlc.arg('avatar_preset_id'),
    avatar_object_key = '',
    avatar_content_type = '',
    avatar_etag = '',
    avatar_size_bytes = 0,
    revision = revision + 1,
    updated_at = NOW()
  FROM current
  WHERE wallets.id = current.id
  RETURNING
    wallets.id,
    wallets.wallet_type,
    wallets.address,
    wallets.remark,
    wallets.source,
    wallets.avatar_preset_id,
    wallets.avatar_object_key,
    wallets.revision,
    wallets.created_at,
    wallets.updated_at
)
SELECT updated.*, current.avatar_object_key AS previous_avatar_object_key
FROM updated
JOIN current ON current.id = updated.id;

-- name: ListWalletAvatarObjectKeys :many
SELECT avatar_object_key
FROM wallets
WHERE avatar_object_key <> ''
ORDER BY avatar_object_key;

-- name: ReplaceWalletAvatarMetadata :one
WITH current AS (
  SELECT id, avatar_object_key
  FROM wallets
  WHERE wallets.id = sqlc.arg('id')
    AND wallets.owner_account_id = sqlc.arg('owner_account_id')::uuid
    AND wallets.revision = sqlc.arg('expected_revision')
), updated AS (
  UPDATE wallets
  SET
    avatar_preset_id = '',
    avatar_object_key = sqlc.arg('avatar_object_key'),
    avatar_content_type = sqlc.arg('avatar_content_type'),
    avatar_etag = sqlc.arg('avatar_etag'),
    avatar_size_bytes = sqlc.arg('avatar_size_bytes'),
    revision = revision + 1,
    updated_at = NOW()
  FROM current
  WHERE wallets.id = current.id
  RETURNING
    wallets.id,
    wallets.wallet_type,
    wallets.address,
    wallets.remark,
    wallets.source,
    wallets.avatar_preset_id,
    wallets.avatar_object_key,
    wallets.avatar_content_type,
    wallets.avatar_etag,
    wallets.avatar_size_bytes,
    wallets.revision,
    wallets.created_at,
    wallets.updated_at
)
SELECT updated.*, current.avatar_object_key AS previous_avatar_object_key
FROM updated
JOIN current ON current.id = updated.id;

-- name: ResetWalletAvatarMetadata :one
WITH current AS (
  SELECT id, avatar_object_key
  FROM wallets
  WHERE wallets.id = sqlc.arg('id')
    AND wallets.owner_account_id = sqlc.arg('owner_account_id')::uuid
    AND wallets.revision = sqlc.arg('expected_revision')
), updated AS (
  UPDATE wallets
  SET
    avatar_preset_id = '',
    avatar_object_key = '',
    avatar_content_type = '',
    avatar_etag = '',
    avatar_size_bytes = 0,
    revision = revision + 1,
    updated_at = NOW()
  FROM current
  WHERE wallets.id = current.id
  RETURNING
    wallets.id,
    wallets.wallet_type,
    wallets.address,
    wallets.remark,
    wallets.source,
    wallets.avatar_preset_id,
    wallets.avatar_object_key,
    wallets.revision,
    wallets.created_at,
    wallets.updated_at
)
SELECT updated.*, current.avatar_object_key AS previous_avatar_object_key
FROM updated
JOIN current ON current.id = updated.id;
