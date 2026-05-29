-- name: AddProjectComment :one
INSERT INTO project_comment (
  project_contract,
  username,
  content
) VALUES ($1, $2, $3)
RETURNING id, project_contract, username, content, created_at;

-- name: CountProjectCommentsByContract :one
SELECT COUNT(*)::bigint
FROM project_comment
WHERE project_contract = $1;

-- name: ListProjectCommentsByContract :many
SELECT
  id,
  project_contract,
  username,
  content,
  created_at
FROM project_comment
WHERE project_contract = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;
