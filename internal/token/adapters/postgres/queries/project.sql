-- Catalog persistence and read model.
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

-- name: ListProjects :many
SELECT *
FROM project
WHERE chain_id = @chain_id
ORDER BY block_number, tx_index, id;

-- name: DeleteProject :execrows
DELETE FROM project
WHERE id = @id;
