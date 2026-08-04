-- Selection persistence and read model.
-- name: GetProjectSelectionByID :one
SELECT *
FROM project_selection
WHERE id = @id
  AND project_id = @project_id;

-- name: GetLatestProjectSelection :one
SELECT *
FROM project_selection
WHERE project_id = @project_id
ORDER BY decided_at DESC, id DESC
LIMIT 1;

-- name: InsertProjectSelection :one
INSERT INTO project_selection (
  project_id,
  outcome,
  strategy_key,
  strategy_version,
  report_revision,
  reason_codes,
  reason_detail,
  decided_at
) VALUES (
  @project_id,
  @outcome,
  @strategy_key,
  @strategy_version,
  @report_revision,
  @reason_codes,
  @reason_detail,
  @decided_at
)
RETURNING *;

-- name: CountProjectSelections :one
SELECT COUNT(*)::bigint
FROM project_selection AS selection
JOIN project ON project.id = selection.project_id
WHERE (sqlc.arg('chain_id')::bigint = 0 OR project.chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR selection.project_id = sqlc.arg('project_id')::bigint)
  AND (sqlc.arg('outcome')::text = '' OR selection.outcome = sqlc.arg('outcome')::text);

-- name: ListProjectSelections :many
SELECT
  selection.*,
  project.chain_id,
  project.contract
FROM project_selection AS selection
JOIN project ON project.id = selection.project_id
WHERE (sqlc.arg('chain_id')::bigint = 0 OR project.chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR selection.project_id = sqlc.arg('project_id')::bigint)
  AND (sqlc.arg('outcome')::text = '' OR selection.outcome = sqlc.arg('outcome')::text)
ORDER BY selection.decided_at DESC, selection.id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
