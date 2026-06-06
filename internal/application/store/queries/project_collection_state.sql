-- name: GetProjectCollectionState :one
SELECT
  p.chain_id,
  p.contract AS project_contract,
  COALESCE(s.status, '')::text AS status,
  s.workflow_id,
  s.last_requested_at,
  s.last_started_at,
  s.last_completed_at,
  s.next_run_at,
  s.last_error,
  s.updated_at
FROM project p
LEFT JOIN project_collection_state s ON s.project_id = p.id
WHERE p.chain_id = @chain_id
  AND p.contract = @project_contract;

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

-- name: MarkProjectCollectionRunning :exec
INSERT INTO project_collection_state (
  project_id,
  status,
  workflow_id,
  last_requested_at,
  last_started_at,
  next_run_at,
  last_error
)
SELECT
  id,
  'running',
  @workflow_id,
  @started_at::timestamptz,
  @started_at::timestamptz,
  NULL,
  NULL
FROM project
WHERE chain_id = @chain_id
  AND contract = @project_contract
ON CONFLICT (project_id) DO UPDATE
SET status = EXCLUDED.status,
  workflow_id = EXCLUDED.workflow_id,
  last_started_at = EXCLUDED.last_started_at,
  next_run_at = NULL,
  last_error = NULL,
  updated_at = now();

-- name: MarkProjectCollectionCompleted :exec
INSERT INTO project_collection_state (
  project_id,
  status,
  last_completed_at,
  next_run_at,
  last_error
)
SELECT
  id,
  'completed',
  @completed_at::timestamptz,
  NULL,
  NULL
FROM project
WHERE chain_id = @chain_id
  AND contract = @project_contract
ON CONFLICT (project_id) DO UPDATE
SET status = EXCLUDED.status,
  last_completed_at = EXCLUDED.last_completed_at,
  next_run_at = NULL,
  last_error = NULL,
  updated_at = now();

-- name: MarkProjectCollectionFailed :exec
INSERT INTO project_collection_state (
  project_id,
  status,
  next_run_at,
  last_error
)
SELECT
  id,
  'failed',
  sqlc.narg('next_run_at')::timestamptz,
  @last_error
FROM project
WHERE chain_id = @chain_id
  AND contract = @project_contract
ON CONFLICT (project_id) DO UPDATE
SET status = EXCLUDED.status,
  next_run_at = EXCLUDED.next_run_at,
  last_error = EXCLUDED.last_error,
  updated_at = now();

-- name: ListProjectComponentStates :many
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
ORDER BY s.component;
