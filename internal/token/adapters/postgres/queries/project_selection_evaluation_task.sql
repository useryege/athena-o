-- Selection task persistence.
-- name: EnqueueProjectSelectionEvaluationTask :one
INSERT INTO project_selection_evaluation_task (
  project_id,
  report_revision
) VALUES (
  @project_id,
  @report_revision
)
ON CONFLICT (project_id, report_revision) DO UPDATE
SET updated_at = project_selection_evaluation_task.updated_at
RETURNING *;

-- name: GetProjectSelectionEvaluationTask :one
SELECT *
FROM project_selection_evaluation_task
WHERE project_id = @project_id
  AND report_revision = @report_revision;

-- name: ClaimProjectSelectionEvaluationTasks :many
WITH claimable AS (
  SELECT task.id
  FROM project_selection_evaluation_task AS task
  JOIN project_research_state AS research ON research.project_id = task.project_id
  WHERE research.status IN ('researching', 'selected')
    AND (
      (task.status = 'pending' AND task.available_at <= now())
      OR (task.status = 'running' AND task.lease_expires_at <= now())
    )
  ORDER BY task.available_at, task.id
  FOR UPDATE OF task SKIP LOCKED
  LIMIT sqlc.arg('limit')
)
UPDATE project_selection_evaluation_task AS task
SET status = 'running',
  locked_at = now(),
  lease_expires_at = now() + (sqlc.arg('lease_seconds')::bigint * INTERVAL '1 second'),
  updated_at = now()
FROM claimable
WHERE task.id = claimable.id
RETURNING task.*;

-- name: MarkProjectSelectionEvaluationTaskSucceeded :execrows
UPDATE project_selection_evaluation_task
SET status = 'succeeded',
  locked_at = NULL,
  lease_expires_at = NULL,
  last_error = NULL,
  updated_at = now()
WHERE id = @id
  AND status = 'running';

-- name: RetryProjectSelectionEvaluationTask :one
UPDATE project_selection_evaluation_task
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

-- name: FailProjectSelectionEvaluationTask :execrows
UPDATE project_selection_evaluation_task
SET status = 'failed',
  attempts = LEAST(attempts + 1, 5),
  locked_at = NULL,
  lease_expires_at = NULL,
  last_error = @last_error,
  updated_at = now()
WHERE id = @id
  AND status = 'running';
