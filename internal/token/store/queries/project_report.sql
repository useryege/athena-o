-- name: InsertProjectReportIfNotExists :exec
INSERT INTO project_report (
  project_id
) VALUES (
  @project_id
)
ON CONFLICT (project_id) DO NOTHING;

-- name: ListDueProjectReportEvaluationTasks :many
SELECT
  task.project_id,
  task.revision,
  task.attempts,
  (EXTRACT(EPOCH FROM COALESCE(MAX(collection_task.updated_at), task.updated_at)) * 1000000)::bigint
    AS source_updated_at_unix_micro,
  COALESCE(chain_state.project_id, 0)::bigint AS chain_state_project_id,
  COALESCE(chain_state.chain_state, '{}'::jsonb) AS chain_state
FROM project_report_evaluation_task AS task
LEFT JOIN project_data_collection_task AS collection_task
  ON collection_task.project_id = task.project_id
LEFT JOIN project_chain_state AS chain_state
  ON chain_state.project_id = task.project_id
WHERE task.status = 'pending'
  AND task.attempts < 5
  AND task.next_attempt_at <= now()
GROUP BY
  task.project_id,
  task.revision,
  task.attempts,
  task.updated_at,
  chain_state.project_id,
  chain_state.chain_state,
  task.next_attempt_at,
  task.created_at
ORDER BY task.next_attempt_at ASC, task.created_at ASC, task.project_id ASC
LIMIT sqlc.arg('limit');

-- name: LockPendingProjectReportEvaluationTask :one
SELECT *
FROM project_report_evaluation_task
WHERE project_id = @project_id
  AND revision = @revision
  AND status = 'pending'
FOR UPDATE;

-- name: MarkProjectReportEvaluationTaskSucceeded :exec
UPDATE project_report_evaluation_task
SET status = 'succeeded',
  next_attempt_at = now(),
  last_error = NULL,
  updated_at = now()
WHERE project_id = @project_id
  AND revision = @revision
  AND status = 'pending';

-- name: MarkProjectReportEvaluationTaskFailed :one
UPDATE project_report_evaluation_task
SET attempts = attempts + 1,
  status = CASE
    WHEN attempts + 1 >= 5 THEN 'failed'
    ELSE 'pending'
  END,
  next_attempt_at = CASE
    WHEN attempts + 1 >= 5 THEN now()
    ELSE now() + INTERVAL '1 minute'
  END,
  last_error = @last_error,
  updated_at = now()
WHERE project_id = @project_id
  AND revision = @revision
  AND status = 'pending'
RETURNING *;

-- name: UpdateProjectReportEvaluation :one
UPDATE project_report
SET weth_pair_is_created = sqlc.narg('weth_pair_is_created'),
  weth_pair_is_remove_liquidity = sqlc.narg('weth_pair_is_remove_liquidity'),
  weth_pair_is_mint = sqlc.narg('weth_pair_is_mint'),
  weth_pair_quote_usdt_value_int = sqlc.narg('weth_pair_quote_usdt_value_int'),
  weth_pair_last_swap_timestamp = sqlc.narg('weth_pair_last_swap_timestamp'),
  usdt_pair_is_created = sqlc.narg('usdt_pair_is_created'),
  usdt_pair_is_remove_liquidity = sqlc.narg('usdt_pair_is_remove_liquidity'),
  usdt_pair_is_mint = sqlc.narg('usdt_pair_is_mint'),
  usdt_pair_quote_usdt_value_int = sqlc.narg('usdt_pair_quote_usdt_value_int'),
  usdt_pair_last_swap_timestamp = sqlc.narg('usdt_pair_last_swap_timestamp'),
  source_updated_at = @source_updated_at,
  evaluated_at = @evaluated_at
WHERE project_id = @project_id
RETURNING *;
