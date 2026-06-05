-- name: UpsertProjectComponentState :exec
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
  @component,
  @status,
  sqlc.narg('last_attempt_at')::timestamptz,
  sqlc.narg('last_success_at')::timestamptz,
  sqlc.narg('next_run_at')::timestamptz,
  sqlc.narg('last_error')::text
)
ON CONFLICT (project_id, component) DO UPDATE
SET status = EXCLUDED.status,
  last_attempt_at = EXCLUDED.last_attempt_at,
  last_success_at = EXCLUDED.last_success_at,
  next_run_at = EXCLUDED.next_run_at,
  last_error = EXCLUDED.last_error,
  updated_at = now();

-- name: GetProjectComponentState :one
SELECT p.chain_id, p.contract AS project_contract, s.component, s.status, s.last_attempt_at, s.last_success_at, s.next_run_at, s.last_error, s.updated_at
FROM project_component_state s
JOIN project p ON p.id = s.project_id
WHERE p.chain_id = @chain_id
  AND p.contract = @project_contract
  AND s.component = @component;

-- name: MarkProjectComponentSuccessNow :exec
INSERT INTO project_component_state (
  project_id,
  component,
  status,
  last_attempt_at,
  last_success_at
) VALUES (
  (SELECT id FROM project WHERE chain_id = @chain_id AND contract = @project_contract),
  @component,
  'success',
  now(),
  now()
)
ON CONFLICT (project_id, component) DO UPDATE
SET status = EXCLUDED.status,
  last_attempt_at = EXCLUDED.last_attempt_at,
  last_success_at = EXCLUDED.last_success_at,
  last_error = NULL,
  updated_at = now();
