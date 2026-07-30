-- Discovery processing persistence.
-- name: GetChainProcessingCheckpoint :one
SELECT
  c.id AS chain_id,
  c.name AS chain_name,
  c.enabled,
  COALESCE(cp.cursor_block_number, 0)::bigint AS cursor_block_number,
  COALESCE(cp.status, 'stopped')::text AS status,
  COALESCE(cp.created_at, c.created_at) AS created_at,
  COALESCE(cp.updated_at, c.created_at) AS updated_at
FROM chain c
LEFT JOIN chain_processing_checkpoint cp ON cp.chain_id = c.id
WHERE c.id = @chain_id;

-- name: UpsertChainProcessingCheckpoint :one
INSERT INTO chain_processing_checkpoint (
  chain_id,
  cursor_block_number,
  status
) VALUES (
  @chain_id,
  @cursor_block_number,
  @status
)
ON CONFLICT (chain_id) DO UPDATE
SET cursor_block_number = EXCLUDED.cursor_block_number,
  status = EXCLUDED.status,
  updated_at = now()
RETURNING *;

-- name: UpsertChainProcessingCheckpointCursor :one
INSERT INTO chain_processing_checkpoint (
  chain_id,
  cursor_block_number,
  status
) VALUES (
  @chain_id,
  @cursor_block_number,
  COALESCE(NULLIF(sqlc.arg('status')::text, ''), 'running')
)
ON CONFLICT (chain_id) DO UPDATE
SET cursor_block_number = EXCLUDED.cursor_block_number,
  updated_at = now()
RETURNING *;

-- name: UpdateChainProcessingCheckpointStatus :one
UPDATE chain_processing_checkpoint
SET status = @status,
  updated_at = now()
WHERE chain_id = @chain_id
RETURNING *;

-- name: ListChainProcessingCheckpoints :many
SELECT
  c.id AS chain_id,
  c.name AS chain_name,
  c.enabled,
  COALESCE(cp.cursor_block_number, 0)::bigint AS cursor_block_number,
  COALESCE(cp.status, 'stopped')::text AS status,
  COALESCE(cp.created_at, c.created_at) AS created_at,
  COALESCE(cp.updated_at, c.created_at) AS updated_at
FROM chain c
LEFT JOIN chain_processing_checkpoint cp ON cp.chain_id = c.id
ORDER BY c.id;
