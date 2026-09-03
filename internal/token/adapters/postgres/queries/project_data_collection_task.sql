-- One-time project data-collection task persistence.

-- name: CreateProjectDataCollectionTasks :many
WITH required_data_types(data_type, ordinal) AS (
  SELECT data_type, ordinal
  FROM unnest(ARRAY[
    'chain_state',
    'wallet_asset_state',
    'simulation_result',
    'ave',
    'contract_code_source',
    'wallet_normal_transactions'
  ]::text[]) WITH ORDINALITY AS value(data_type, ordinal)
)
INSERT INTO project_data_collection_task (
  project_id,
  data_type
)
SELECT
  @project_id,
  required_data_types.data_type
FROM required_data_types
ORDER BY required_data_types.ordinal
ON CONFLICT (project_id, data_type) DO NOTHING
RETURNING *;

-- name: CountProjectDataCollectionTaskTypes :one
SELECT COUNT(DISTINCT data_type)::bigint
FROM project_data_collection_task
WHERE project_id = @project_id;

-- name: ClaimProjectDataCollectionTask :one
WITH claimable AS (
  SELECT task.id
  FROM project_data_collection_task AS task
  JOIN project ON project.id = task.project_id
  WHERE task.data_type = @data_type
    AND project.chain_id = ANY(@chain_ids::bigint[])
    AND (
      (task.status = 'pending' AND task.available_at <= clock_timestamp())
      OR (task.status = 'running' AND task.lease_expires_at <= clock_timestamp())
    )
  ORDER BY
    CASE WHEN task.status = 'running' THEN 0 ELSE 1 END,
    COALESCE(task.lease_expires_at, task.available_at),
    task.id
  FOR UPDATE OF task SKIP LOCKED
  LIMIT 1
)
UPDATE project_data_collection_task AS task
SET status = 'running',
  claim_generation = task.claim_generation + 1,
  locked_at = clock_timestamp(),
  lease_expires_at = clock_timestamp() + (sqlc.arg('lease_seconds')::bigint * INTERVAL '1 second'),
  updated_at = now()
FROM claimable
WHERE task.id = claimable.id
RETURNING task.*;

-- name: RenewProjectDataCollectionTaskLease :execrows
UPDATE project_data_collection_task
SET lease_expires_at = clock_timestamp() + (sqlc.arg('lease_seconds')::bigint * INTERVAL '1 second'),
  updated_at = now()
WHERE id = @id
  AND status = 'running'
  AND claim_generation = @claim_generation
  AND lease_expires_at > clock_timestamp();

-- name: LockProjectForCollectionCompletion :one
SELECT id
FROM project
WHERE id = @project_id
FOR UPDATE;

-- name: LockProjectDataCollectionTaskForCompletion :one
SELECT *
FROM project_data_collection_task
WHERE id = @id
  AND project_id = @project_id
  AND claim_generation = @claim_generation
  AND (
    (status = 'running' AND lease_expires_at > clock_timestamp())
    OR status = 'succeeded'
  )
FOR UPDATE;

-- name: MarkProjectDataCollectionTaskSucceeded :one
UPDATE project_data_collection_task
SET status = 'succeeded',
  lease_expires_at = NULL,
  last_error = NULL,
  finished_at = @finished_at,
  updated_at = now()
WHERE id = @id
  AND status = 'running'
  AND claim_generation = @claim_generation
  AND lease_expires_at > clock_timestamp()
RETURNING *;

-- name: RetryProjectDataCollectionTask :one
UPDATE project_data_collection_task
SET status = 'pending',
  failure_count = failure_count + 1,
  available_at = @available_at,
  locked_at = NULL,
  lease_expires_at = NULL,
  last_error = @last_error,
  updated_at = now()
WHERE id = @id
  AND status = 'running'
  AND claim_generation = @claim_generation
  AND lease_expires_at > clock_timestamp()
  AND failure_count < 2
RETURNING *;

-- name: FailProjectDataCollectionTask :one
UPDATE project_data_collection_task
SET status = 'failed',
  failure_count = failure_count + 1,
  lease_expires_at = NULL,
  last_error = @last_error,
  finished_at = @finished_at,
  updated_at = now()
WHERE id = @id
  AND status = 'running'
  AND claim_generation = @claim_generation
  AND lease_expires_at > clock_timestamp()
  AND failure_count = 2
RETURNING *;

-- name: GetProjectCollectionBarrierState :one
SELECT
  COUNT(*)::bigint AS task_count,
  COUNT(*) FILTER (WHERE status IN ('succeeded', 'failed'))::bigint AS terminal_count,
  COUNT(*) FILTER (WHERE status = 'succeeded')::bigint AS succeeded_count,
  COUNT(*) FILTER (WHERE status = 'failed')::bigint AS failed_count
FROM project_data_collection_task
WHERE project_id = @project_id;

-- name: GetProjectDataCollectionTask :one
SELECT *
FROM project_data_collection_task
WHERE id = @id;

-- name: CountProjectDataCollectionTasks :one
SELECT COUNT(*)::bigint
FROM project_data_collection_task AS task
JOIN project ON project.id = task.project_id
WHERE (sqlc.arg('chain_id')::bigint = 0 OR project.chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR task.project_id = sqlc.arg('project_id')::bigint)
  AND (sqlc.arg('data_type')::text = '' OR task.data_type = sqlc.arg('data_type')::text)
  AND (sqlc.arg('status')::text = '' OR task.status = sqlc.arg('status')::text);

-- name: ListProjectDataCollectionTasks :many
SELECT task.*
FROM project_data_collection_task AS task
JOIN project ON project.id = task.project_id
WHERE (sqlc.arg('chain_id')::bigint = 0 OR project.chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR task.project_id = sqlc.arg('project_id')::bigint)
  AND (sqlc.arg('data_type')::text = '' OR task.data_type = sqlc.arg('data_type')::text)
  AND (sqlc.arg('status')::text = '' OR task.status = sqlc.arg('status')::text)
ORDER BY task.created_at DESC, task.id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListProjectDataCollectionTasksByProject :many
SELECT *
FROM project_data_collection_task
WHERE project_id = @project_id
ORDER BY CASE data_type
  WHEN 'chain_state' THEN 1
  WHEN 'wallet_asset_state' THEN 2
  WHEN 'simulation_result' THEN 3
  WHEN 'ave' THEN 4
  WHEN 'contract_code_source' THEN 5
  WHEN 'wallet_normal_transactions' THEN 6
  ELSE 7
END;

-- name: GetProjectForDataCollectionTask :one
SELECT project.*
FROM project_data_collection_task AS task
JOIN project ON project.id = task.project_id
WHERE task.id = @id;
