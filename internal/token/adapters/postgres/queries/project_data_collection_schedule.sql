-- Research scheduling persistence.
-- name: UpsertProjectDataCollectionSchedule :one
INSERT INTO project_data_collection_schedule (
  project_id,
  data_type,
  retry_interval_seconds,
  next_run_at
) VALUES (
  @project_id,
  @data_type,
  @retry_interval_seconds,
  @next_run_at
)
ON CONFLICT (project_id, data_type) DO UPDATE
SET retry_interval_seconds = EXCLUDED.retry_interval_seconds,
  updated_at = now()
RETURNING *;

-- name: ApplyProjectDataCollectionSchedulePolicy :execrows
UPDATE project_data_collection_schedule
SET retry_interval_seconds = @retry_interval_seconds,
  updated_at = now()
WHERE data_type = @data_type
  AND status = 'active';

-- name: ListDueProjectDataCollectionSchedules :many
SELECT schedule.*
FROM project_data_collection_schedule AS schedule
JOIN project_research_state AS research ON research.project_id = schedule.project_id
WHERE schedule.status = 'active'
  AND schedule.next_run_at <= now()
  AND research.status IN ('researching', 'selected')
  AND NOT EXISTS (
    SELECT 1
    FROM project_data_collection_task AS task
    WHERE task.project_id = schedule.project_id
      AND task.data_type = schedule.data_type
      AND task.revision = schedule.latest_task_revision
      AND task.status IN ('pending', 'running')
  )
ORDER BY schedule.next_run_at, schedule.project_id, schedule.data_type
LIMIT sqlc.arg('limit');

-- name: LockProjectDataCollectionSchedule :one
SELECT *
FROM project_data_collection_schedule
WHERE project_id = @project_id
  AND data_type = @data_type
FOR UPDATE;

-- name: AdvanceProjectDataCollectionSchedule :one
UPDATE project_data_collection_schedule
SET latest_task_revision = @latest_task_revision,
  next_run_at = @next_run_at,
  updated_at = now()
WHERE project_id = @project_id
  AND data_type = @data_type
RETURNING *;

-- name: CompleteProjectDataCollectionSchedule :execrows
UPDATE project_data_collection_schedule
SET status = 'completed',
  consecutive_failures = 0,
  last_error = NULL,
  last_checked_at = @last_checked_at,
  next_run_at = NULL,
  updated_at = now()
WHERE project_id = @project_id
  AND data_type = @data_type;

-- name: MarkProjectDataCollectionScheduleRetrying :execrows
UPDATE project_data_collection_schedule
SET consecutive_failures = @consecutive_failures,
  last_error = @last_error,
  next_run_at = @next_run_at,
  updated_at = now()
WHERE project_id = @project_id
  AND data_type = @data_type
  AND status = 'active';

-- name: FailProjectDataCollectionSchedule :execrows
UPDATE project_data_collection_schedule
SET status = 'failed',
  consecutive_failures = @consecutive_failures,
  last_error = @last_error,
  next_run_at = NULL,
  updated_at = now()
WHERE project_id = @project_id
  AND data_type = @data_type
  AND status = 'active';

-- name: PauseTerminalProjectDataCollectionSchedules :execrows
UPDATE project_data_collection_schedule AS schedule
SET status = 'paused',
  next_run_at = NULL,
  updated_at = now()
FROM project_research_state AS research
WHERE research.project_id = schedule.project_id
  AND research.status IN ('rejected', 'expired')
  AND schedule.status = 'active';

-- name: GetProjectDataCollectionSchedule :one
SELECT *
FROM project_data_collection_schedule
WHERE project_id = @project_id
  AND data_type = @data_type;

-- name: ListProjectDataCollectionSchedulesByProject :many
SELECT *
FROM project_data_collection_schedule
WHERE project_id = @project_id
ORDER BY data_type;
