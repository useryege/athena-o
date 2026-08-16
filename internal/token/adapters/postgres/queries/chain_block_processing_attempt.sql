-- Chain block processing attempt audit and read model.
-- name: LockChainProcessingCheckpoint :one
SELECT cursor_block_number, updated_at
FROM chain_processing_checkpoint
WHERE chain_id = @chain_id
FOR UPDATE;

-- name: ReconcileRunningChainBlockProcessingAttempts :exec
UPDATE chain_block_processing_attempt
SET status = CASE
      WHEN block_number <= @cursor_block_number THEN 'succeeded'
      ELSE 'interrupted'
    END,
  terminal_stage = CASE
      WHEN block_number <= @cursor_block_number THEN 'persistence'
      ELSE terminal_stage
    END,
  error_message = CASE
      WHEN block_number <= @cursor_block_number THEN error_message
      ELSE COALESCE(error_message, 'processor stopped before attempt completion was recorded')
    END,
  timing_complete = false,
  completed_at = CASE
      WHEN block_number <= @cursor_block_number THEN @checkpoint_updated_at
      ELSE now()
    END,
  updated_at = now()
WHERE chain_id = @chain_id
  AND status = 'running';

-- name: CreateChainBlockProcessingAttempt :one
INSERT INTO chain_block_processing_attempt (
  chain_id,
  block_number,
  attempt_number
)
SELECT
  @chain_id,
  @block_number,
  COALESCE(MAX(attempt_number), 0)::int + 1
FROM chain_block_processing_attempt
WHERE chain_id = @chain_id
  AND block_number = @block_number
RETURNING *;

-- name: CompleteChainBlockProcessingAttempt :one
UPDATE chain_block_processing_attempt
SET block_time = sqlc.narg('block_time')::bigint,
  status = @status,
  terminal_stage = NULLIF(@terminal_stage::text, ''),
  error_message = NULLIF(@error_message::text, ''),
  checkpoint_read_duration_us = sqlc.narg('checkpoint_read_duration_us')::bigint,
  discovery_duration_us = sqlc.narg('discovery_duration_us')::bigint,
  validation_duration_us = sqlc.narg('validation_duration_us')::bigint,
  persistence_duration_us = sqlc.narg('persistence_duration_us')::bigint,
  total_duration_us = sqlc.narg('total_duration_us')::bigint,
  candidate_count = sqlc.narg('candidate_count')::int,
  validated_count = sqlc.narg('validated_count')::int,
  rejected_count = sqlc.narg('rejected_count')::int,
  expired_research_state_count = sqlc.narg('expired_research_state_count')::bigint,
  timing_complete = @timing_complete,
  completed_at = now(),
  updated_at = now()
WHERE id = @id
RETURNING *;

-- name: BackfillChainBlockProcessingAttemptBlockTime :exec
UPDATE chain_block_processing_attempt
SET block_time = @block_time,
  updated_at = now()
WHERE chain_id = @chain_id
  AND block_number = @block_number
  AND block_time IS NULL;

-- name: DeleteExpiredChainBlockProcessingAttempts :execrows
DELETE FROM chain_block_processing_attempt
WHERE chain_id = @chain_id
  AND block_time IS NOT NULL
  AND block_time < @cutoff_block_time;

-- name: DeleteExpiredUnknownTimeChainBlockProcessingAttempts :execrows
WITH boundary AS (
  SELECT COALESCE(MAX(expired.block_number), -1)::bigint AS block_number
  FROM chain_block_processing_attempt expired
  WHERE expired.chain_id = @chain_id
    AND expired.block_time IS NOT NULL
    AND expired.block_time < @cutoff_block_time
)
DELETE FROM chain_block_processing_attempt attempt
USING boundary
WHERE attempt.chain_id = @chain_id
  AND attempt.block_time IS NULL
  AND attempt.block_number <= boundary.block_number;

-- name: GetChainBlockProcessingSummary :one
WITH anchor AS (
  SELECT COALESCE(MAX(block_time) FILTER (WHERE status = 'succeeded'), 0)::bigint AS block_time
  FROM chain_block_processing_attempt
  WHERE chain_id = @chain_id
), filtered AS (
  SELECT attempt.*
  FROM chain_block_processing_attempt attempt
  CROSS JOIN anchor
  WHERE attempt.chain_id = @chain_id
    AND (
      (@block_number::bigint > 0 AND attempt.block_number = @block_number)
      OR (
        @block_number::bigint = 0
        AND attempt.block_time IS NOT NULL
        AND attempt.block_time BETWEEN GREATEST(0, anchor.block_time - @window_seconds::bigint) AND anchor.block_time
      )
    )
), aggregate AS (
  SELECT
    COUNT(*)::bigint AS attempt_count,
    COUNT(*) FILTER (WHERE status = 'running')::bigint AS running_count,
    COUNT(*) FILTER (WHERE status = 'succeeded')::bigint AS succeeded_count,
    COUNT(*) FILTER (WHERE status = 'failed')::bigint AS failed_count,
    COUNT(*) FILTER (WHERE status = 'cancelled')::bigint AS cancelled_count,
    COUNT(*) FILTER (WHERE status = 'interrupted')::bigint AS interrupted_count,
    COUNT(*) FILTER (WHERE status = 'succeeded' AND NOT timing_complete)::bigint AS incomplete_succeeded_count,
    COUNT(*) FILTER (WHERE status = 'succeeded' AND timing_complete)::bigint AS measured_succeeded_count,
    COALESCE(ROUND(AVG(total_duration_us) FILTER (WHERE status = 'succeeded' AND timing_complete)), 0)::bigint AS average_duration_us,
    COALESCE(ROUND(AVG(checkpoint_read_duration_us) FILTER (WHERE status = 'succeeded' AND timing_complete)), 0)::bigint AS average_checkpoint_read_duration_us,
    COALESCE(ROUND(AVG(discovery_duration_us) FILTER (WHERE status = 'succeeded' AND timing_complete)), 0)::bigint AS average_discovery_duration_us,
    COALESCE(ROUND(AVG(validation_duration_us) FILTER (WHERE status = 'succeeded' AND timing_complete)), 0)::bigint AS average_validation_duration_us,
    COALESCE(ROUND(AVG(persistence_duration_us) FILTER (WHERE status = 'succeeded' AND timing_complete)), 0)::bigint AS average_persistence_duration_us
  FROM filtered
), fastest AS (
  SELECT block_number, total_duration_us
  FROM filtered
  WHERE status = 'succeeded' AND timing_complete
  ORDER BY total_duration_us, block_number
  LIMIT 1
), slowest AS (
  SELECT block_number, total_duration_us
  FROM filtered
  WHERE status = 'succeeded' AND timing_complete
  ORDER BY total_duration_us DESC, block_number
  LIMIT 1
), exact_range AS (
  SELECT COALESCE(MIN(block_time), 0)::bigint AS start_time,
    COALESCE(MAX(block_time), 0)::bigint AS end_time
  FROM filtered
)
SELECT
  @chain_id::bigint AS chain_id,
  CASE WHEN @block_number::bigint > 0 THEN exact_range.start_time ELSE GREATEST(0, anchor.block_time - @window_seconds::bigint) END::bigint AS range_start_block_time,
  CASE WHEN @block_number::bigint > 0 THEN exact_range.end_time ELSE anchor.block_time END::bigint AS range_end_block_time,
  aggregate.*,
  CASE
    WHEN aggregate.succeeded_count + aggregate.failed_count = 0 THEN 0
    ELSE ROUND(aggregate.failed_count * 10000.0 / (aggregate.succeeded_count + aggregate.failed_count))::bigint
  END AS failure_rate_bps,
  COALESCE(fastest.block_number, 0)::bigint AS fastest_block_number,
  COALESCE(fastest.total_duration_us, 0)::bigint AS fastest_duration_us,
  COALESCE(slowest.block_number, 0)::bigint AS slowest_block_number,
  COALESCE(slowest.total_duration_us, 0)::bigint AS slowest_duration_us
FROM anchor
CROSS JOIN aggregate
CROSS JOIN exact_range
LEFT JOIN fastest ON true
LEFT JOIN slowest ON true;

-- name: ListChainBlockProcessingAttempts :many
WITH anchor AS (
  SELECT COALESCE(MAX(block_time) FILTER (WHERE status = 'succeeded'), 0)::bigint AS block_time
  FROM chain_block_processing_attempt
  WHERE chain_id = @chain_id
), checkpoint AS (
  SELECT cursor_block_number
  FROM chain_processing_checkpoint
  WHERE chain_id = @chain_id
)
SELECT attempt.*
FROM chain_block_processing_attempt attempt
CROSS JOIN anchor
CROSS JOIN checkpoint
WHERE attempt.chain_id = @chain_id
  AND (@status::text = '' OR attempt.status = @status)
  AND (
    (@block_number::bigint > 0 AND attempt.block_number = @block_number)
    OR (
      @block_number::bigint = 0
      AND (
        (attempt.block_time IS NOT NULL AND attempt.block_time BETWEEN GREATEST(0, anchor.block_time - @window_seconds::bigint) AND anchor.block_time)
        OR attempt.block_number > checkpoint.cursor_block_number
      )
    )
  )
ORDER BY attempt.block_number DESC, attempt.attempt_number DESC
LIMIT @page_size
OFFSET @page_offset;

-- name: CountChainBlockProcessingAttempts :one
WITH anchor AS (
  SELECT COALESCE(MAX(block_time) FILTER (WHERE status = 'succeeded'), 0)::bigint AS block_time
  FROM chain_block_processing_attempt
  WHERE chain_id = @chain_id
), checkpoint AS (
  SELECT cursor_block_number
  FROM chain_processing_checkpoint
  WHERE chain_id = @chain_id
)
SELECT COUNT(*)::bigint
FROM chain_block_processing_attempt attempt
CROSS JOIN anchor
CROSS JOIN checkpoint
WHERE attempt.chain_id = @chain_id
  AND (@status::text = '' OR attempt.status = @status)
  AND (
    (@block_number::bigint > 0 AND attempt.block_number = @block_number)
    OR (
      @block_number::bigint = 0
      AND (
        (attempt.block_time IS NOT NULL AND attempt.block_time BETWEEN GREATEST(0, anchor.block_time - @window_seconds::bigint) AND anchor.block_time)
        OR attempt.block_number > checkpoint.cursor_block_number
      )
    )
  );
