-- name: UpsertProjectComponentState :exec
INSERT INTO project_component_state (
  project_contract,
  component,
  status,
  last_attempt_at,
  last_success_at,
  next_run_at,
  last_error
) VALUES (
  $1,
  $2,
  $3,
  sqlc.narg('last_attempt_at')::timestamptz,
  sqlc.narg('last_success_at')::timestamptz,
  sqlc.narg('next_run_at')::timestamptz,
  sqlc.narg('last_error')::text
)
ON CONFLICT (project_contract, component) DO UPDATE
SET status = EXCLUDED.status,
  last_attempt_at = EXCLUDED.last_attempt_at,
  last_success_at = EXCLUDED.last_success_at,
  next_run_at = EXCLUDED.next_run_at,
  last_error = EXCLUDED.last_error,
  updated_at = now();

-- name: GetProjectComponentState :one
SELECT project_contract, component, status, last_attempt_at, last_success_at, next_run_at, last_error, updated_at
FROM project_component_state
WHERE project_contract = $1 AND component = $2;

-- name: MarkProjectComponentSuccessNow :exec
INSERT INTO project_component_state (
  project_contract,
  component,
  status,
  last_attempt_at,
  last_success_at
) VALUES ($1, $2, 'success', now(), now())
ON CONFLICT (project_contract, component) DO UPDATE
SET status = EXCLUDED.status,
  last_attempt_at = EXCLUDED.last_attempt_at,
  last_success_at = EXCLUDED.last_success_at,
  last_error = NULL,
  updated_at = now();
