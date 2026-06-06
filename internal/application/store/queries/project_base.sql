-- name: InsertProjectBase :exec
INSERT INTO project (
  chain_id,
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index
) VALUES (@chain_id, @block_number, @block_time, @contract, @creator, @tx_hash, @tx_index)
ON CONFLICT DO NOTHING;

-- name: UpsertProjectFromDiscovery :exec
INSERT INTO project (
  chain_id,
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index,
  weth_pair,
  usdt_pair,
  source,
  status,
  reason,
  payload,
  discovered_at
) VALUES (
  @chain_id,
  @block_number,
  @block_time,
  @contract,
  @creator,
  @tx_hash,
  @tx_index,
  sqlc.narg('weth_pair')::bytea,
  sqlc.narg('usdt_pair')::bytea,
  @source,
  'processed',
  sqlc.narg('reason')::text,
  @payload::jsonb,
  now()
)
ON CONFLICT (chain_id, contract) DO UPDATE
SET weth_pair = COALESCE(project.weth_pair, EXCLUDED.weth_pair),
  usdt_pair = COALESCE(project.usdt_pair, EXCLUDED.usdt_pair),
  source = EXCLUDED.source,
  status = 'processed',
  reason = COALESCE(EXCLUDED.reason, project.reason),
  payload = project.payload || EXCLUDED.payload,
  updated_at = now();

-- name: GetMaxProjectBlockNumber :one
SELECT
  COALESCE(MAX(block_number), 0)::bigint AS max_block,
  (COUNT(*)::bigint > 0) AS has_value
FROM project
WHERE chain_id = @chain_id;

-- name: ListProjectBases :many
SELECT chain_id, block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
WHERE chain_id = @chain_id
ORDER BY block_number, tx_index, id;

-- name: CountProjectBases :one
SELECT COUNT(*)::bigint
FROM project
WHERE chain_id = @chain_id;

-- name: ListProjectBasesPage :many
SELECT chain_id, block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
WHERE chain_id = @chain_id
ORDER BY block_number, tx_index, id
LIMIT @limit_count OFFSET @offset_count;

-- name: GetProjectBaseByContract :one
SELECT chain_id, block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
WHERE chain_id = @chain_id
  AND contract = @contract;

-- name: ListProjectBasesByCreatorBefore :many
SELECT chain_id, block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
WHERE chain_id = @chain_id
  AND creator = @creator
  AND (block_number < @block_number OR (block_number = @block_number AND tx_index < @tx_index))
ORDER BY block_number, tx_index, id;
