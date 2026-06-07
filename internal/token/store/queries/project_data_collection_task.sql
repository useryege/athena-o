-- name: UpsertProjectDataCollectionTask :one
INSERT INTO project_data_collection_task (
  project_id,
  data_type,
  status,
  attempts,
  next_attempt_at,
  last_error
) VALUES (
  @project_id,
  @data_type,
  @status,
  @attempts,
  @next_attempt_at,
  sqlc.narg('last_error')
)
ON CONFLICT (project_id, data_type) DO UPDATE
SET status = EXCLUDED.status,
  attempts = EXCLUDED.attempts,
  next_attempt_at = EXCLUDED.next_attempt_at,
  last_error = EXCLUDED.last_error
RETURNING *;

-- name: GetProjectDataCollectionTask :one
SELECT *
FROM project_data_collection_task
WHERE project_id = @project_id
  AND data_type = @data_type;

-- name: CountProjectDataCollectionTasks :one
SELECT COUNT(*)::bigint
FROM project_data_collection_task
WHERE (sqlc.narg('project_id')::bigint IS NULL OR project_id = sqlc.narg('project_id')::bigint)
  AND (sqlc.narg('data_type')::text IS NULL OR data_type = sqlc.narg('data_type')::text)
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text);

-- name: ListProjectDataCollectionTasks :many
SELECT *
FROM project_data_collection_task
WHERE (sqlc.narg('project_id')::bigint IS NULL OR project_id = sqlc.narg('project_id')::bigint)
  AND (sqlc.narg('data_type')::text IS NULL OR data_type = sqlc.narg('data_type')::text)
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY created_at DESC, project_id DESC, data_type DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListDueProjectDataCollectionTasks :many
SELECT
  t.project_id,
  t.data_type,
  t.status,
  t.attempts,
  t.next_attempt_at,
  t.last_error,
  t.created_at,
  p.chain_id,
  p.contract,
  p.creator,
  p.tx_hash,
  p.tx_index,
  p.block_number,
  p.block_time,
  p.code_hash,
  p.weth_pair,
  p.usdt_pair,
  p.created_at AS project_created_at
FROM project_data_collection_task AS t
JOIN project AS p ON p.id = t.project_id
WHERE t.data_type = @data_type
  AND t.status = 'pending'
  AND t.attempts < 5
  AND t.next_attempt_at <= now()
ORDER BY t.next_attempt_at ASC, t.created_at ASC, t.project_id ASC
LIMIT sqlc.arg('limit');

-- name: DeleteProjectDataCollectionTask :execrows
DELETE FROM project_data_collection_task
WHERE project_id = @project_id
  AND data_type = @data_type;
