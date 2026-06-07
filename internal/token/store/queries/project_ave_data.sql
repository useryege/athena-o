-- name: UpsertProjectAveData :one
INSERT INTO project_ave_data (
  project_id,
  ave_response,
  fetched_at
) VALUES (
  @project_id,
  @ave_response::jsonb,
  @fetched_at
)
ON CONFLICT (project_id) DO UPDATE
SET ave_response = EXCLUDED.ave_response,
  fetched_at = EXCLUDED.fetched_at
RETURNING *;

-- name: GetProjectAveData :one
SELECT *
FROM project_ave_data
WHERE project_id = @project_id;

-- name: CountProjectAveData :one
SELECT COUNT(*)::bigint
FROM project_ave_data
WHERE (sqlc.narg('project_id')::bigint IS NULL OR project_id = sqlc.narg('project_id')::bigint);

-- name: ListProjectAveData :many
SELECT *
FROM project_ave_data
WHERE (sqlc.narg('project_id')::bigint IS NULL OR project_id = sqlc.narg('project_id')::bigint)
ORDER BY created_at DESC, project_id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: DeleteProjectAveData :execrows
DELETE FROM project_ave_data
WHERE project_id = @project_id;
