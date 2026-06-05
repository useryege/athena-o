-- name: UpsertProjectCandidate :exec
INSERT INTO project_candidate (
  chain_id,
  contract,
  creator,
  tx_hash,
  block_number,
  block_time,
  tx_index,
  weth_pair,
  usdt_pair,
  source,
  status,
  payload
) VALUES (
  @chain_id,
  @contract,
  sqlc.narg('creator')::bytea,
  sqlc.narg('tx_hash')::bytea,
  sqlc.narg('block_number')::bigint,
  sqlc.narg('block_time')::bigint,
  sqlc.narg('tx_index')::bigint,
  sqlc.narg('weth_pair')::bytea,
  sqlc.narg('usdt_pair')::bytea,
  @source,
  'pending',
  @payload::jsonb
)
ON CONFLICT (chain_id, contract) DO UPDATE
SET creator = COALESCE(project_candidate.creator, EXCLUDED.creator),
  tx_hash = COALESCE(project_candidate.tx_hash, EXCLUDED.tx_hash),
  block_number = COALESCE(project_candidate.block_number, EXCLUDED.block_number),
  block_time = COALESCE(project_candidate.block_time, EXCLUDED.block_time),
  tx_index = COALESCE(project_candidate.tx_index, EXCLUDED.tx_index),
  weth_pair = COALESCE(project_candidate.weth_pair, EXCLUDED.weth_pair),
  usdt_pair = COALESCE(project_candidate.usdt_pair, EXCLUDED.usdt_pair),
  source = EXCLUDED.source,
  status = CASE
    WHEN project_candidate.status = 'processed' THEN project_candidate.status
    ELSE 'pending'
  END,
  payload = project_candidate.payload || EXCLUDED.payload,
  updated_at = now();

-- name: UpsertProjectCollectionRequest :exec
INSERT INTO project_collection_state (
  project_id,
  status,
  last_requested_at,
  next_run_at
)
SELECT
  id,
  'requested',
  @requested_at::timestamptz,
  @next_run_at::timestamptz
FROM project
WHERE chain_id = @chain_id
  AND contract = @project_contract
ON CONFLICT (project_id) DO UPDATE
SET status = EXCLUDED.status,
  last_requested_at = EXCLUDED.last_requested_at,
  next_run_at = EXCLUDED.next_run_at,
  updated_at = now();
