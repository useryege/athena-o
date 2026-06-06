-- name: ListProjectAveRefreshCandidates :many
SELECT p.contract
FROM project p
LEFT JOIN project_ave_detail d
  ON d.project_id = p.id
LEFT JOIN project_component_state s
  ON s.project_id = p.id
  AND s.component = 'ave_detail'
WHERE p.chain_id = @chain_id
  AND (d.project_id IS NULL OR d.fetched_at < @stale_before::timestamptz)
  AND (s.next_run_at IS NULL OR s.next_run_at <= @now::timestamptz)
  AND COALESCE(s.status, '') <> 'running'
ORDER BY COALESCE(s.next_run_at, d.fetched_at, p.created_at), p.id
LIMIT @limit_count;

-- name: ScheduleProjectAveRefresh :exec
INSERT INTO project_component_state (
  project_id,
  component,
  status,
  next_run_at
) VALUES (
  (SELECT id FROM project WHERE chain_id = @chain_id AND contract = @project_contract),
  'ave_detail',
  'pending',
  @next_run_at
)
ON CONFLICT (project_id, component) DO UPDATE
SET status = EXCLUDED.status,
  next_run_at = EXCLUDED.next_run_at,
  updated_at = now();

-- name: MarkProjectAveRefreshRunning :exec
INSERT INTO project_component_state (
  project_id,
  component,
  status,
  last_attempt_at,
  next_run_at
) VALUES (
  (SELECT id FROM project WHERE chain_id = @chain_id AND contract = @project_contract),
  'ave_detail',
  'running',
  @last_attempt_at,
  NULL
)
ON CONFLICT (project_id, component) DO UPDATE
SET status = EXCLUDED.status,
  last_attempt_at = EXCLUDED.last_attempt_at,
  next_run_at = NULL,
  updated_at = now();

-- name: MarkProjectAveRefreshSuccess :exec
INSERT INTO project_component_state (
  project_id,
  component,
  status,
  last_attempt_at,
  last_success_at,
  next_run_at,
  last_error
) VALUES (
  (SELECT id FROM project WHERE chain_id = @chain_id AND contract = @project_contract),
  'ave_detail',
  'success',
  @last_attempt_at,
  @last_attempt_at,
  @next_run_at,
  NULL
)
ON CONFLICT (project_id, component) DO UPDATE
SET status = EXCLUDED.status,
  last_attempt_at = EXCLUDED.last_attempt_at,
  last_success_at = EXCLUDED.last_success_at,
  next_run_at = EXCLUDED.next_run_at,
  last_error = NULL,
  updated_at = now();

-- name: MarkProjectAveRefreshFailed :exec
INSERT INTO project_component_state (
  project_id,
  component,
  status,
  last_attempt_at,
  next_run_at,
  last_error
) VALUES (
  (SELECT id FROM project WHERE chain_id = @chain_id AND contract = @project_contract),
  'ave_detail',
  'failed',
  @last_attempt_at,
  @next_run_at,
  @last_error
)
ON CONFLICT (project_id, component) DO UPDATE
SET status = EXCLUDED.status,
  last_attempt_at = EXCLUDED.last_attempt_at,
  next_run_at = EXCLUDED.next_run_at,
  last_error = EXCLUDED.last_error,
  updated_at = now();

-- name: GetProjectAveComponentState :one
SELECT
  p.chain_id,
  p.contract AS project_contract,
  s.component,
  s.status,
  s.last_attempt_at,
  s.last_success_at,
  s.next_run_at,
  s.last_error,
  s.updated_at
FROM project_component_state s
JOIN project p ON p.id = s.project_id
WHERE p.chain_id = @chain_id
  AND p.contract = @project_contract
  AND s.component = 'ave_detail';
