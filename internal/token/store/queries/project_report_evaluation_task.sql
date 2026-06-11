-- name: EnqueueProjectReportEvaluationTask :one
INSERT INTO project_report_evaluation_task (
  project_id
) VALUES (
  @project_id
)
ON CONFLICT (project_id) DO UPDATE
SET status = 'pending',
  revision = project_report_evaluation_task.revision + 1,
  attempts = 0,
  next_attempt_at = now(),
  last_error = NULL,
  updated_at = now()
RETURNING *;
