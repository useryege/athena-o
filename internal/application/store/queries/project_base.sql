-- name: InsertProjectBase :exec
INSERT INTO project (
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT DO NOTHING;

-- name: GetMaxProjectBlockNumber :one
SELECT
  COALESCE(MAX(block_number), 0)::bigint AS max_block,
  (COUNT(*)::bigint > 0) AS has_value
FROM project;

-- name: ListProjectBases :many
SELECT block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
ORDER BY block_number, tx_index, id;

-- name: CountProjectBases :one
SELECT COUNT(*)::bigint
FROM project;

-- name: ListProjectBasesPage :many
SELECT block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
ORDER BY block_number, tx_index, id
LIMIT $1 OFFSET $2;

-- name: GetProjectBaseByContract :one
SELECT block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
WHERE contract = $1;

-- name: ListProjectBasesByCreatorBefore :many
SELECT block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
WHERE creator = $1
  AND (block_number < $2 OR (block_number = $2 AND tx_index < $3))
ORDER BY block_number, tx_index, id;
