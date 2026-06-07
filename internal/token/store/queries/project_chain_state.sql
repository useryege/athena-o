-- name: UpsertProjectChainState :one
INSERT INTO project_chain_state (
  project_id,
  chain_state,
  fetched_at
) VALUES (
  @project_id,
  @chain_state::jsonb,
  @fetched_at
)
ON CONFLICT (project_id) DO UPDATE
SET chain_state = EXCLUDED.chain_state,
  fetched_at = EXCLUDED.fetched_at
RETURNING *;

-- name: GetProjectChainState :one
SELECT *
FROM project_chain_state
WHERE project_id = @project_id;

-- name: CountProjectChainStates :one
SELECT COUNT(*)::bigint
FROM project_chain_state
WHERE (sqlc.narg('project_id')::bigint IS NULL OR project_id = sqlc.narg('project_id')::bigint);

-- name: ListProjectChainStates :many
SELECT *
FROM project_chain_state
WHERE (sqlc.narg('project_id')::bigint IS NULL OR project_id = sqlc.narg('project_id')::bigint)
ORDER BY created_at DESC, project_id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: DeleteProjectChainState :execrows
DELETE FROM project_chain_state
WHERE project_id = @project_id;
