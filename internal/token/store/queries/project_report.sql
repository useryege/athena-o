-- name: InsertProjectReportIfNotExists :exec
INSERT INTO project_report (
  project_id
) VALUES (
  @project_id
)
ON CONFLICT (project_id) DO NOTHING;

-- name: ListProjectReportsDueForEvaluation :many
WITH task_state AS (
  SELECT
    project_id,
    BOOL_AND(status = 'succeeded') AS is_complete,
    BOOL_OR(data_type = 'chain_state' AND status = 'succeeded') AS chain_state_succeeded,
    (EXTRACT(EPOCH FROM MAX(updated_at)) * 1000000)::bigint AS source_updated_at_unix_micro
  FROM project_data_collection_task
  GROUP BY project_id
  HAVING COUNT(*) = 5
    AND BOOL_AND(status IN ('succeeded', 'failed'))
)
SELECT
  report.project_id,
  task_state.is_complete,
  task_state.chain_state_succeeded,
  task_state.source_updated_at_unix_micro,
  COALESCE(chain_state.project_id, 0)::bigint AS chain_state_project_id,
  COALESCE(chain_state.chain_state, '{}'::jsonb) AS chain_state
FROM project_report AS report
JOIN task_state ON task_state.project_id = report.project_id
LEFT JOIN project_chain_state AS chain_state ON chain_state.project_id = report.project_id
WHERE report.evaluated_at IS NULL
  OR report.source_updated_at IS NULL
  OR task_state.source_updated_at_unix_micro
    > (EXTRACT(EPOCH FROM report.source_updated_at) * 1000000)::bigint
ORDER BY task_state.source_updated_at_unix_micro ASC, report.project_id ASC
LIMIT sqlc.arg('limit');

-- name: UpdateProjectReportEvaluation :one
UPDATE project_report
SET is_complete = @is_complete,
  weth_pair_is_created = sqlc.narg('weth_pair_is_created'),
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
