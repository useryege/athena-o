-- Research collection persistence.
-- name: CreateProjectDataCollectionTask :one
INSERT INTO project_data_collection_task (
  project_id,
  data_type,
  revision
) VALUES (
  @project_id,
  @data_type,
  @revision
)
ON CONFLICT (project_id, data_type, revision) DO UPDATE
SET updated_at = project_data_collection_task.updated_at
RETURNING *;

-- name: ClaimProjectDataCollectionTasks :many
WITH claimable AS (
  SELECT task.id
  FROM project_data_collection_task AS task
  JOIN project AS project ON project.id = task.project_id
  JOIN project_research_state AS research ON research.project_id = task.project_id
  WHERE task.data_type = @data_type
    AND project.chain_id = ANY(@chain_ids::bigint[])
    AND research.status IN ('researching', 'selected')
    AND (
      (task.status = 'pending' AND task.available_at <= now())
      OR (task.status = 'running' AND task.lease_expires_at <= now())
    )
  ORDER BY task.available_at, task.id
  FOR UPDATE OF task SKIP LOCKED
  LIMIT sqlc.arg('limit')
)
UPDATE project_data_collection_task AS task
SET status = 'running',
  locked_at = now(),
  lease_expires_at = now() + (sqlc.arg('lease_seconds')::bigint * INTERVAL '1 second'),
  updated_at = now()
FROM claimable
WHERE task.id = claimable.id
RETURNING task.*;

-- name: RenewProjectDataCollectionTaskLease :execrows
UPDATE project_data_collection_task
SET lease_expires_at = now() + (sqlc.arg('lease_seconds')::bigint * INTERVAL '1 second'),
  updated_at = now()
WHERE id = @id
  AND status = 'running';

-- name: GetProjectDataCollectionTask :one
SELECT *
FROM project_data_collection_task
WHERE id = @id;

-- name: CountProjectDataCollectionTasks :one
SELECT COUNT(*)::bigint
FROM project_data_collection_task
WHERE (sqlc.arg('project_id')::bigint = 0 OR project_id = sqlc.arg('project_id')::bigint)
  AND (sqlc.arg('data_type')::text = '' OR data_type = sqlc.arg('data_type')::text)
  AND (sqlc.arg('status')::text = '' OR status = sqlc.arg('status')::text);

-- name: ListProjectDataCollectionTasks :many
SELECT *
FROM project_data_collection_task
WHERE (sqlc.arg('project_id')::bigint = 0 OR project_id = sqlc.arg('project_id')::bigint)
  AND (sqlc.arg('data_type')::text = '' OR data_type = sqlc.arg('data_type')::text)
  AND (sqlc.arg('status')::text = '' OR status = sqlc.arg('status')::text)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: MarkProjectDataCollectionTaskSucceeded :execrows
UPDATE project_data_collection_task
SET status = 'succeeded',
  locked_at = NULL,
  lease_expires_at = NULL,
  last_error = NULL,
  updated_at = now()
WHERE id = @id
  AND status = 'running';

-- name: RetryProjectDataCollectionTask :one
UPDATE project_data_collection_task
SET status = 'pending',
  attempts = attempts + 1,
  available_at = @available_at,
  locked_at = NULL,
  lease_expires_at = NULL,
  last_error = @last_error,
  updated_at = now()
WHERE id = @id
  AND status = 'running'
  AND attempts < 4
RETURNING *;

-- name: FailProjectDataCollectionTask :execrows
UPDATE project_data_collection_task
SET status = 'failed',
  attempts = LEAST(attempts + 1, 5),
  locked_at = NULL,
  lease_expires_at = NULL,
  last_error = @last_error,
  updated_at = now()
WHERE id = @id
  AND status = 'running';

-- name: GetProjectForDataCollectionTask :one
SELECT project.*
FROM project_data_collection_task AS task
JOIN project ON project.id = task.project_id
WHERE task.id = @id;
