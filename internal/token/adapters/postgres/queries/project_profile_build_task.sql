-- Unique project-profile build task persistence.

-- name: EnqueueProjectProfileBuildTask :exec
INSERT INTO project_profile_build_task (project_id)
SELECT task.project_id
FROM project_data_collection_task AS task
WHERE task.project_id = @project_id
GROUP BY task.project_id
HAVING COUNT(*) = 6
  AND COUNT(*) FILTER (WHERE task.status IN ('succeeded', 'failed')) = 6
ON CONFLICT (project_id) DO NOTHING;

-- name: ClaimProjectProfileBuildTask :one
WITH claimable AS (
  SELECT project_id
  FROM project_profile_build_task
  WHERE (status = 'pending' AND available_at <= clock_timestamp())
    OR (status = 'running' AND lease_expires_at <= clock_timestamp())
  ORDER BY
    CASE WHEN status = 'running' THEN 0 ELSE 1 END,
    COALESCE(lease_expires_at, available_at),
    project_id
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE project_profile_build_task AS task
SET status = 'running',
  claim_generation = task.claim_generation + 1,
  locked_at = clock_timestamp(),
  lease_expires_at = clock_timestamp() + (sqlc.arg('lease_seconds')::bigint * INTERVAL '1 second'),
  updated_at = now()
FROM claimable
WHERE task.project_id = claimable.project_id
RETURNING task.*;

-- name: RenewProjectProfileBuildTaskLease :execrows
UPDATE project_profile_build_task
SET lease_expires_at = clock_timestamp() + (sqlc.arg('lease_seconds')::bigint * INTERVAL '1 second'),
  updated_at = now()
WHERE project_id = @project_id
  AND status = 'running'
  AND claim_generation = @claim_generation
  AND lease_expires_at > clock_timestamp();

-- name: LockProjectProfileBuildTaskForCompletion :one
SELECT *
FROM project_profile_build_task
WHERE project_id = @project_id
  AND claim_generation = @claim_generation
  AND (
    (status = 'running' AND lease_expires_at > clock_timestamp())
    OR status = 'succeeded'
  )
FOR UPDATE;

-- name: MarkProjectProfileBuildTaskSucceeded :one
UPDATE project_profile_build_task
SET status = 'succeeded',
  lease_expires_at = NULL,
  last_error = NULL,
  finished_at = @finished_at,
  updated_at = now()
WHERE project_id = @project_id
  AND status = 'running'
  AND claim_generation = @claim_generation
  AND lease_expires_at > clock_timestamp()
RETURNING *;

-- name: RetryProjectProfileBuildTask :one
UPDATE project_profile_build_task
SET status = 'pending',
  failure_count = failure_count + 1,
  available_at = @available_at,
  locked_at = NULL,
  lease_expires_at = NULL,
  last_error = @last_error,
  updated_at = now()
WHERE project_id = @project_id
  AND status = 'running'
  AND claim_generation = @claim_generation
  AND lease_expires_at > clock_timestamp()
  AND failure_count < 2
RETURNING *;

-- name: FailProjectProfileBuildTask :one
UPDATE project_profile_build_task
SET status = 'failed',
  failure_count = failure_count + 1,
  lease_expires_at = NULL,
  last_error = @last_error,
  finished_at = @finished_at,
  updated_at = now()
WHERE project_id = @project_id
  AND status = 'running'
  AND claim_generation = @claim_generation
  AND lease_expires_at > clock_timestamp()
  AND failure_count = 2
RETURNING *;

-- name: GetProjectProfileBuildTask :one
SELECT *
FROM project_profile_build_task
WHERE project_id = @project_id;
