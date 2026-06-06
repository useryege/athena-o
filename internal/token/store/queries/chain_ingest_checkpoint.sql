-- name: GetChainIngestCheckpoint :one
SELECT
  c.id AS chain_id,
  c.name AS chain_name,
  c.enabled,
  COALESCE(cp.cursor_block_number, 0)::bigint AS cursor_block_number,
  COALESCE(cp.status, 'stopped')::text AS status,
  COALESCE(cp.created_at, c.created_at) AS created_at
FROM chain c
LEFT JOIN chain_ingest_checkpoint cp ON cp.chain_id = c.id
WHERE c.id = @chain_id;

-- name: UpsertChainIngestCheckpoint :one
INSERT INTO chain_ingest_checkpoint (
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
  status = EXCLUDED.status
RETURNING *;

-- name: UpdateChainIngestCheckpointStatus :one
UPDATE chain_ingest_checkpoint
SET status = @status
WHERE chain_id = @chain_id
RETURNING *;

-- name: ListChainIngestCheckpoints :many
SELECT
  c.id AS chain_id,
  c.name AS chain_name,
  c.enabled,
  COALESCE(cp.cursor_block_number, 0)::bigint AS cursor_block_number,
  COALESCE(cp.status, 'stopped')::text AS status,
  COALESCE(cp.created_at, c.created_at) AS created_at
FROM chain c
LEFT JOIN chain_ingest_checkpoint cp ON cp.chain_id = c.id
ORDER BY c.id;
