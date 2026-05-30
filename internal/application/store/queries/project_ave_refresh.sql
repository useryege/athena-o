-- name: ListProjectAveRefreshCandidates :many
SELECT p.contract
FROM project p
LEFT JOIN project_ave_token_detail d
  ON d.project_contract = p.contract
LEFT JOIN project_component_state s
  ON s.project_contract = p.contract
  AND s.component = 'ave_detail'
WHERE (d.project_contract IS NULL OR d.fetched_at < @stale_before::timestamptz)
  AND (s.next_run_at IS NULL OR s.next_run_at <= @now::timestamptz)
  AND COALESCE(s.status, '') <> 'running'
ORDER BY COALESCE(s.next_run_at, d.fetched_at, p.created_at), p.id
LIMIT @limit_count;

-- name: ScheduleProjectAveRefresh :exec
INSERT INTO project_component_state (
  project_contract,
  component,
  status,
  next_run_at
) VALUES ($1, 'ave_detail', 'pending', $2)
ON CONFLICT (project_contract, component) DO UPDATE
SET status = EXCLUDED.status,
  next_run_at = EXCLUDED.next_run_at,
  updated_at = now();

-- name: MarkProjectAveRefreshRunning :exec
INSERT INTO project_component_state (
  project_contract,
  component,
  status,
  last_attempt_at,
  next_run_at
) VALUES ($1, 'ave_detail', 'running', $2, NULL)
ON CONFLICT (project_contract, component) DO UPDATE
SET status = EXCLUDED.status,
  last_attempt_at = EXCLUDED.last_attempt_at,
  next_run_at = NULL,
  updated_at = now();

-- name: MarkProjectAveRefreshSuccess :exec
INSERT INTO project_component_state (
  project_contract,
  component,
  status,
  last_attempt_at,
  last_success_at,
  next_run_at,
  last_error
) VALUES ($1, 'ave_detail', 'success', $2, $2, $3, NULL)
ON CONFLICT (project_contract, component) DO UPDATE
SET status = EXCLUDED.status,
  last_attempt_at = EXCLUDED.last_attempt_at,
  last_success_at = EXCLUDED.last_success_at,
  next_run_at = EXCLUDED.next_run_at,
  last_error = NULL,
  updated_at = now();

-- name: MarkProjectAveRefreshFailed :exec
INSERT INTO project_component_state (
  project_contract,
  component,
  status,
  last_attempt_at,
  next_run_at,
  last_error
) VALUES ($1, 'ave_detail', 'failed', $2, $3, $4)
ON CONFLICT (project_contract, component) DO UPDATE
SET status = EXCLUDED.status,
  last_attempt_at = EXCLUDED.last_attempt_at,
  next_run_at = EXCLUDED.next_run_at,
  last_error = EXCLUDED.last_error,
  updated_at = now();

-- name: GetProjectAveComponentState :one
SELECT
  project_contract,
  component,
  status,
  last_attempt_at,
  last_success_at,
  next_run_at,
  last_error,
  updated_at
FROM project_component_state
WHERE project_contract = $1
  AND component = 'ave_detail';
