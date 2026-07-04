-- name: UpsertProject :one
INSERT INTO project (
  chain_id,
  contract,
  tx_sender,
  tx_hash,
  tx_index,
  block_number,
  block_time,
  code_hash,
  name,
  symbol,
  decimals,
  total_supply,
  weth_pair,
  usdt_pair
) VALUES (
  @chain_id,
  @contract,
  @tx_sender,
  @tx_hash,
  @tx_index,
  @block_number,
  @block_time,
  @code_hash,
  @name,
  @symbol,
  @decimals,
  @total_supply,
  @weth_pair,
  @usdt_pair
)
ON CONFLICT (chain_id, contract) DO UPDATE
SET code_hash = EXCLUDED.code_hash,
  name = EXCLUDED.name,
  symbol = EXCLUDED.symbol,
  decimals = EXCLUDED.decimals,
  total_supply = EXCLUDED.total_supply,
  weth_pair = EXCLUDED.weth_pair,
  usdt_pair = EXCLUDED.usdt_pair
RETURNING *;

-- name: GetProject :one
SELECT *
FROM project
WHERE id = @id;

-- name: GetProjectByContract :one
SELECT *
FROM project
WHERE chain_id = @chain_id
  AND contract = @contract;

-- name: CountProjects :one
SELECT COUNT(*)::bigint
FROM project
WHERE (sqlc.arg('chain_id')::bigint = 0 OR chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.narg('code_hash')::bytea IS NULL OR code_hash = sqlc.narg('code_hash')::bytea)
  AND (sqlc.narg('contract')::bytea IS NULL OR contract = sqlc.narg('contract')::bytea);

-- name: ListProjects :many
SELECT *
FROM project
WHERE chain_id = @chain_id
ORDER BY block_number, tx_index, id;

-- name: ListProjectsPage :many
SELECT *
FROM project
WHERE (sqlc.arg('chain_id')::bigint = 0 OR chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.narg('code_hash')::bytea IS NULL OR code_hash = sqlc.narg('code_hash')::bytea)
  AND (sqlc.narg('contract')::bytea IS NULL OR contract = sqlc.narg('contract')::bytea)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: DeleteProject :execrows
DELETE FROM project
WHERE id = @id;
