-- name: UpsertBytecode :exec
INSERT INTO bytecode (code_hash)
VALUES ($1)
ON CONFLICT (code_hash) DO UPDATE
SET updated_at = now();

-- name: UpsertContractBytecodeDeployment :exec
INSERT INTO contract_bytecode_deployment (chain_id, contract, code_hash)
VALUES ($1, $2, $3)
ON CONFLICT (chain_id, contract) DO UPDATE
SET code_hash = EXCLUDED.code_hash,
  updated_at = now();

-- name: GetBytecode :one
SELECT code_hash, source_code, source_code_hash, source_code_fetched_at,
  source_code_origin, created_at, updated_at
FROM bytecode
WHERE code_hash = $1;

-- name: UpdateBytecodeSourceCode :exec
UPDATE bytecode
SET source_code = $2,
  source_code_hash = $3,
  source_code_fetched_at = now(),
  source_code_origin = $4,
  updated_at = now()
WHERE code_hash = $1;

-- name: ListBytecodes :many
WITH deployment_counts AS (
  SELECT code_hash, COUNT(*)::bigint AS deployment_count
  FROM contract_bytecode_deployment
  GROUP BY code_hash
),
filtered AS (
  SELECT b.code_hash,
    COALESCE(dc.deployment_count, 0)::bigint AS deployment_count,
    COALESCE(btrim(b.source_code), '') <> '' AS is_open_source,
    (bl.code_hash IS NOT NULL)::boolean AS is_bytecode_blacklisted,
    b.created_at,
    b.updated_at
  FROM bytecode b
  LEFT JOIN deployment_counts dc ON dc.code_hash = b.code_hash
  LEFT JOIN bytecode_blacklist bl ON bl.code_hash = b.code_hash
  WHERE (sqlc.narg('code_hash')::bytea IS NULL OR b.code_hash = sqlc.narg('code_hash')::bytea)
)
SELECT code_hash, deployment_count, is_open_source,
  is_bytecode_blacklisted, created_at, updated_at, COUNT(*) OVER()::bigint AS total
FROM filtered
ORDER BY updated_at DESC, code_hash
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetBytecodeDetail :one
WITH deployment_counts AS (
  SELECT code_hash, COUNT(*)::bigint AS deployment_count
  FROM contract_bytecode_deployment
  WHERE code_hash = $1
  GROUP BY code_hash
)
SELECT b.code_hash, b.source_code, b.source_code_hash, b.source_code_fetched_at,
  b.source_code_origin, b.created_at, b.updated_at,
  COALESCE(dc.deployment_count, 0)::bigint AS deployment_count,
  (bl.code_hash IS NOT NULL)::boolean AS is_bytecode_blacklisted
FROM bytecode b
LEFT JOIN deployment_counts dc ON dc.code_hash = b.code_hash
LEFT JOIN bytecode_blacklist bl ON bl.code_hash = b.code_hash
WHERE b.code_hash = $1;

-- name: ListBytecodeDeployments :many
SELECT chain_id, contract, code_hash, first_seen_at, updated_at, COUNT(*) OVER()::bigint AS total
FROM contract_bytecode_deployment
WHERE code_hash = $1
  AND (sqlc.narg('chain_id')::bigint IS NULL OR chain_id = sqlc.narg('chain_id')::bigint)
  AND (sqlc.narg('contract')::bytea IS NULL OR contract = sqlc.narg('contract')::bytea)
ORDER BY updated_at DESC, chain_id, contract
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: IsBytecodeBlacklisted :one
SELECT EXISTS(SELECT 1 FROM bytecode_blacklist WHERE code_hash = $1)::boolean;

-- name: ListBytecodeBlacklistEntries :many
SELECT code_hash, note, source_chain_id, source_contract, created_at
FROM bytecode_blacklist
ORDER BY created_at DESC, code_hash;

-- name: AddBytecodeBlacklistEntry :exec
INSERT INTO bytecode_blacklist (code_hash, note, source_chain_id, source_contract)
VALUES ($1, $2, $3, $4);

-- name: UpdateBytecodeBlacklistNote :execrows
UPDATE bytecode_blacklist
SET note = $2
WHERE code_hash = $1;

-- name: DeleteBytecodeBlacklist :execrows
DELETE FROM bytecode_blacklist
WHERE code_hash = $1;

-- name: GetBytecodeBlacklistEntry :one
SELECT code_hash, note, source_chain_id, source_contract, created_at
FROM bytecode_blacklist
WHERE code_hash = $1;
