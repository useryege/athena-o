-- name: CreateWallet :one
INSERT INTO wallet_private_keys (owner_account_id, system_owned, chain, type, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path)
VALUES (sqlc.arg('owner_account_id')::uuid, FALSE, sqlc.arg('chain'), sqlc.arg('type'), sqlc.arg('address'), sqlc.arg('address_key'), sqlc.arg('alias'), sqlc.arg('private_key_ciphertext'), sqlc.narg('mnemonic_ciphertext'), sqlc.arg('source'), sqlc.arg('derivation_path'))
RETURNING id, owner_account_id, system_owned, chain, type, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path, created_at, updated_at;

-- name: CountWallets :one
SELECT COUNT(*)::bigint
FROM wallet_private_keys
WHERE (
    sqlc.arg('requester_administrator')::boolean
    OR (
      NOT system_owned
      AND owner_account_id = sqlc.arg('requester_account_id')::uuid
    )
  )
  AND (sqlc.narg('chain')::text IS NULL OR chain = sqlc.narg('chain'))
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type'))
  AND (
    sqlc.narg('query')::text IS NULL
    OR address ILIKE sqlc.narg('query')
    OR alias ILIKE sqlc.narg('query')
  );

-- name: ListWallets :many
SELECT id, owner_account_id, system_owned, chain, type, address, alias, source, derivation_path, created_at, updated_at
FROM wallet_private_keys
WHERE (
    sqlc.arg('requester_administrator')::boolean
    OR (
      NOT system_owned
      AND owner_account_id = sqlc.arg('requester_account_id')::uuid
    )
  )
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
SELECT id, owner_account_id, system_owned, chain, type, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path, created_at, updated_at
FROM wallet_private_keys
WHERE id = sqlc.arg('id')
  AND (
    sqlc.arg('requester_administrator')::boolean
    OR (
      NOT system_owned
      AND owner_account_id = sqlc.arg('requester_account_id')::uuid
    )
  );

-- name: UpdateWalletAlias :one
UPDATE wallet_private_keys
SET alias = sqlc.arg('alias'), updated_at = NOW()
WHERE id = sqlc.arg('id')
  AND (
    sqlc.arg('requester_administrator')::boolean
    OR (
      NOT system_owned
      AND owner_account_id = sqlc.arg('requester_account_id')::uuid
    )
  )
RETURNING id, owner_account_id, system_owned, chain, type, address, alias, source, derivation_path, created_at, updated_at;
