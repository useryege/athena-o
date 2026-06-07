-- name: UpsertContractCode :exec
INSERT INTO contract_code (code_hash)
VALUES (@code_hash)
ON CONFLICT (code_hash) DO NOTHING;

-- name: GetContractCode :one
SELECT *
FROM contract_code
WHERE code_hash = @code_hash;

-- name: CountContractCodes :one
SELECT COUNT(*)::bigint
FROM contract_code
WHERE (sqlc.narg('code_hash')::bytea IS NULL OR code_hash = sqlc.narg('code_hash')::bytea);

-- name: ListContractCodes :many
SELECT *
FROM contract_code
WHERE (sqlc.narg('code_hash')::bytea IS NULL OR code_hash = sqlc.narg('code_hash')::bytea)
ORDER BY created_at DESC, code_hash
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: UpdateContractCodeSource :one
UPDATE contract_code
SET source_code = sqlc.narg('source_code')::text,
  source_code_hash = sqlc.narg('source_code_hash')::bytea,
  source_code_fetched_at = sqlc.narg('source_code_fetched_at')::timestamptz
WHERE code_hash = @code_hash
RETURNING *;

-- name: DeleteContractCode :execrows
DELETE FROM contract_code
WHERE code_hash = @code_hash;
