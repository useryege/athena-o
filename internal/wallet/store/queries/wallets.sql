-- name: CreateWallet :one
INSERT INTO wallet_private_keys (chain, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, chain, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path, created_at, updated_at;

-- name: CountWallets :one
SELECT COUNT(*)::bigint
FROM wallet_private_keys
WHERE (sqlc.narg('chain')::text IS NULL OR chain = sqlc.narg('chain'))
  AND (
    sqlc.narg('query')::text IS NULL
    OR address ILIKE sqlc.narg('query')
    OR alias ILIKE sqlc.narg('query')
  );

-- name: ListWallets :many
SELECT id, chain, address, alias, source, derivation_path, created_at, updated_at
FROM wallet_private_keys
WHERE (sqlc.narg('chain')::text IS NULL OR chain = sqlc.narg('chain'))
  AND (
    sqlc.narg('query')::text IS NULL
    OR address ILIKE sqlc.narg('query')
    OR alias ILIKE sqlc.narg('query')
  )
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: GetWallet :one
SELECT id, chain, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path, created_at, updated_at
FROM wallet_private_keys
WHERE id = $1;

-- name: UpdateWalletAlias :one
UPDATE wallet_private_keys
SET alias = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, chain, address, alias, source, derivation_path, created_at, updated_at;
