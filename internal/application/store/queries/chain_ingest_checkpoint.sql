-- name: GetChainIngestCheckpoint :one
SELECT c.id AS chain_id,
  c.name AS chain_name,
  c.enabled,
  COALESCE(cp.finalized_block_number, 0)::bigint AS finalized_block_number,
  cp.finalized_block_hash,
  COALESCE(cp.cursor_block_number, 0)::bigint AS cursor_block_number,
  cp.cursor_block_hash,
  COALESCE(cp.status, 'stopped')::text AS status,
  cp.locked_at,
  cp.locked_by,
  COALESCE(cp.updated_at, c.updated_at) AS updated_at
FROM chain c
LEFT JOIN chain_ingest_checkpoint cp ON cp.chain_id = c.id
WHERE c.id = @chain_id;

-- name: UpsertChainIngestCheckpoint :one
INSERT INTO chain_ingest_checkpoint (
  chain_id,
  finalized_block_number,
  finalized_block_hash,
  cursor_block_number,
  cursor_block_hash,
  status
) VALUES (
  @chain_id,
  @finalized_block_number,
  sqlc.narg('finalized_block_hash')::bytea,
  @cursor_block_number,
  sqlc.narg('cursor_block_hash')::bytea,
  @status
)
ON CONFLICT (chain_id) DO UPDATE
SET finalized_block_number = EXCLUDED.finalized_block_number,
  finalized_block_hash = EXCLUDED.finalized_block_hash,
  cursor_block_number = EXCLUDED.cursor_block_number,
  cursor_block_hash = EXCLUDED.cursor_block_hash,
  status = EXCLUDED.status,
  updated_at = now()
RETURNING chain_id,
  finalized_block_number,
  finalized_block_hash,
  cursor_block_number,
  cursor_block_hash,
  status,
  locked_at,
  locked_by,
  updated_at;

-- name: ListChainIngestCheckpoints :many
SELECT c.id AS chain_id,
  c.name AS chain_name,
  c.enabled,
  COALESCE(cp.finalized_block_number, 0)::bigint AS finalized_block_number,
  cp.finalized_block_hash,
  COALESCE(cp.cursor_block_number, 0)::bigint AS cursor_block_number,
  cp.cursor_block_hash,
  COALESCE(cp.status, 'stopped')::text AS status,
  cp.locked_at,
  cp.locked_by,
  COALESCE(cp.updated_at, c.updated_at) AS updated_at
FROM chain c
LEFT JOIN chain_ingest_checkpoint cp ON cp.chain_id = c.id
ORDER BY c.id;
