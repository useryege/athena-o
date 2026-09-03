-- Per-project Swap collection persistence.

-- name: GetChainProcessingCursorForSwap :one
SELECT cursor_block_number
FROM chain_processing_checkpoint
WHERE chain_id = @chain_id;

-- name: GetChainSwapProcessingCheckpoint :one
SELECT *
FROM chain_swap_processing_checkpoint
WHERE chain_id = @chain_id;

-- name: SetChainSwapProcessingCheckpointStatus :one
INSERT INTO chain_swap_processing_checkpoint (
  chain_id,
  status
) VALUES (
  @chain_id,
  @status
)
ON CONFLICT (chain_id) DO UPDATE
SET status = EXCLUDED.status,
  updated_at = now()
RETURNING *;

-- name: InitializeChainSwapProcessingCheckpoint :one
INSERT INTO chain_swap_processing_checkpoint (
  chain_id,
  cursor_block_number,
  initialized
) VALUES (
  @chain_id,
  @cursor_block_number,
  true
)
ON CONFLICT (chain_id) DO UPDATE
SET cursor_block_number = EXCLUDED.cursor_block_number,
  initialized = true,
  updated_at = now()
WHERE NOT chain_swap_processing_checkpoint.initialized
RETURNING *;

-- name: AdvanceChainSwapProcessingCheckpoint :one
UPDATE chain_swap_processing_checkpoint
SET cursor_block_number = @cursor_block_number,
  updated_at = now()
WHERE chain_id = @chain_id
  AND cursor_block_number = @expected_cursor_block_number
  AND initialized
  AND status = 'running'
RETURNING *;

-- name: FindNextCollectingProjectSwapPairStartBlock :one
SELECT COALESCE(MIN(start_block_number), 0)::bigint AS start_block_number
FROM project_swap_pair
WHERE chain_id = @chain_id
  AND status = 'collecting'
  AND start_block_number <= @source_cursor_block_number;

-- name: ListMatchingCollectingProjectSwapPairs :many
SELECT *
FROM project_swap_pair
WHERE chain_id = @chain_id
  AND status = 'collecting'
  AND start_block_number <= @block_number
  AND pair_address = ANY(sqlc.arg('pair_addresses')::bytea[])
ORDER BY pair_address, id;

-- name: GetProjectSwapPairForUpdate :one
SELECT *
FROM project_swap_pair
WHERE id = @project_swap_pair_id
FOR UPDATE;

-- name: ListDueCollectingProjectSwapPairsForUpdate :many
SELECT *
FROM project_swap_pair
WHERE chain_id = @chain_id
  AND status = 'collecting'
  AND start_block_number <= @block_number
  AND next_expiry_block_time <= sqlc.arg('block_time')::bigint
ORDER BY next_expiry_block_time, id
FOR UPDATE;

-- name: CreateProjectSwapPair :one
INSERT INTO project_swap_pair (
  project_id,
  chain_id,
  pair_kind,
  pair_address,
  start_block_number,
  start_block_time,
  absolute_expiry_block_time,
  next_expiry_block_time
) VALUES (
  @project_id,
  @chain_id,
  @pair_kind,
  @pair_address,
  @start_block_number,
  @start_block_time,
  @absolute_expiry_block_time,
  sqlc.arg('next_expiry_block_time')::bigint
)
ON CONFLICT (project_id, pair_kind) DO UPDATE
SET project_id = project_swap_pair.project_id
WHERE project_swap_pair.chain_id = EXCLUDED.chain_id
  AND project_swap_pair.pair_address = EXCLUDED.pair_address
  AND project_swap_pair.start_block_number = EXCLUDED.start_block_number
  AND project_swap_pair.start_block_time = EXCLUDED.start_block_time
RETURNING *;

-- name: CreateProjectSwapBlock :one
INSERT INTO project_swap_block (
  project_swap_pair_id,
  block_number,
  block_time,
  sample_index
)
SELECT
  project_swap_pair.id,
  @block_number,
  @block_time,
  project_swap_pair.swap_block_count + 1
FROM project_swap_pair
WHERE project_swap_pair.id = @project_swap_pair_id
  AND project_swap_pair.status = 'collecting'
  AND project_swap_pair.swap_block_count < 100
RETURNING *;

-- name: CreateProjectSwapEvent :one
INSERT INTO project_swap_event (
  project_swap_pair_id,
  project_swap_block_id,
  transaction_hash,
  transaction_index,
  log_index,
  tx_from,
  sender,
  to_address,
  amount0_in,
  amount1_in,
  amount0_out,
  amount1_out
) VALUES (
  @project_swap_pair_id,
  @project_swap_block_id,
  @transaction_hash,
  @transaction_index,
  @log_index,
  @tx_from,
  @sender,
  @to_address,
  @amount0_in,
  @amount1_in,
  @amount0_out,
  @amount1_out
)
RETURNING *;

-- name: ObserveProjectSwapPairBlock :one
UPDATE project_swap_pair
SET swap_block_count = swap_block_count + 1,
  first_swap_block_number = COALESCE(first_swap_block_number, sqlc.arg('block_number')::bigint),
  first_swap_block_time = COALESCE(first_swap_block_time, sqlc.arg('block_time')::bigint),
  last_swap_block_number = sqlc.arg('block_number')::bigint,
  last_swap_block_time = sqlc.arg('block_time')::bigint,
  next_expiry_block_time = sqlc.arg('next_expiry_block_time')::bigint,
  updated_at = now()
WHERE id = @project_swap_pair_id
  AND status = 'collecting'
  AND swap_block_count < 99
RETURNING *;

-- name: CompleteProjectSwapPair :one
UPDATE project_swap_pair
SET swap_block_count = 100,
  first_swap_block_number = COALESCE(first_swap_block_number, sqlc.arg('block_number')::bigint),
  first_swap_block_time = COALESCE(first_swap_block_time, sqlc.arg('block_time')::bigint),
  last_swap_block_number = sqlc.arg('block_number')::bigint,
  last_swap_block_time = sqlc.arg('block_time')::bigint,
  status = 'completed',
  next_expiry_block_time = NULL,
  completed_block_number = sqlc.arg('block_number')::bigint,
  completed_block_time = sqlc.arg('block_time')::bigint,
  updated_at = now()
WHERE id = @project_swap_pair_id
  AND status = 'collecting'
  AND swap_block_count = 99
RETURNING *;

-- name: ExpireProjectSwapPair :execrows
UPDATE project_swap_pair
SET status = 'expired',
  next_expiry_block_time = NULL,
  expired_block_number = sqlc.arg('expired_block_number')::bigint,
  expired_block_time = sqlc.arg('expired_block_time')::bigint,
  expired_reason = sqlc.arg('expired_reason')::text,
  updated_at = now()
WHERE id = @project_swap_pair_id
  AND status = 'collecting'
  AND swap_block_count < 100
  AND (
    (
      sqlc.arg('expired_reason')::text = 'max_duration'
      AND absolute_expiry_block_time <= sqlc.arg('expired_block_time')::bigint
    )
    OR
    (
      sqlc.arg('expired_reason')::text = 'no_swap'
      AND swap_block_count = 0
      AND absolute_expiry_block_time > sqlc.arg('expired_block_time')::bigint
      AND next_expiry_block_time <= sqlc.arg('expired_block_time')::bigint
    )
    OR
    (
      sqlc.arg('expired_reason')::text = 'inactive'
      AND swap_block_count > 0
      AND absolute_expiry_block_time > sqlc.arg('expired_block_time')::bigint
      AND next_expiry_block_time <= sqlc.arg('expired_block_time')::bigint
    )
  );
