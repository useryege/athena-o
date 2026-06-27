-- name: CreateWallet :one
INSERT INTO wallet_private_keys (created_by, chain, type, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, created_by, chain, type, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path, created_at, updated_at;

-- name: CountWallets :one
SELECT COUNT(*)::bigint
FROM wallet_private_keys
WHERE (sqlc.narg('created_by')::text IS NULL OR created_by = sqlc.narg('created_by'))
  AND (sqlc.narg('chain')::text IS NULL OR chain = sqlc.narg('chain'))
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type'))
  AND (
    sqlc.narg('query')::text IS NULL
    OR address ILIKE sqlc.narg('query')
    OR alias ILIKE sqlc.narg('query')
  );

-- name: ListWallets :many
SELECT id, created_by, chain, type, address, alias, source, derivation_path, created_at, updated_at
FROM wallet_private_keys
WHERE (sqlc.narg('created_by')::text IS NULL OR created_by = sqlc.narg('created_by'))
  AND (sqlc.narg('chain')::text IS NULL OR chain = sqlc.narg('chain'))
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type'))
  AND (
    sqlc.narg('query')::text IS NULL
    OR address ILIKE sqlc.narg('query')
    OR alias ILIKE sqlc.narg('query')
  )
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: GetWallet :one
SELECT id, created_by, chain, type, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path, created_at, updated_at
FROM wallet_private_keys
WHERE id = sqlc.arg('id')
  AND (sqlc.narg('created_by')::text IS NULL OR created_by = sqlc.narg('created_by'));

-- name: UpdateWalletAlias :one
UPDATE wallet_private_keys
SET alias = sqlc.arg('alias'), updated_at = NOW()
WHERE id = sqlc.arg('id')
  AND (sqlc.narg('created_by')::text IS NULL OR created_by = sqlc.narg('created_by'))
RETURNING id, created_by, chain, type, address, alias, source, derivation_path, created_at, updated_at;
