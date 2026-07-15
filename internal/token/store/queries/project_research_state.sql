-- name: CreateProjectResearchState :one
INSERT INTO project_research_state (
  project_id
) VALUES (
  @project_id
)
ON CONFLICT (project_id) DO UPDATE
SET updated_at = project_research_state.updated_at
RETURNING *;

-- name: ApplyProjectResearchTTL :execrows
UPDATE project_research_state
SET expires_at = created_at + (sqlc.arg('ttl_seconds')::bigint * INTERVAL '1 second'),
  updated_at = now()
WHERE status = 'researching';

-- name: ExpireProjectResearchStates :execrows
UPDATE project_research_state
SET status = 'expired',
  updated_at = now()
WHERE status = 'researching'
  AND expires_at <= now();

-- name: IncrementProjectEvidenceRevision :one
UPDATE project_research_state
SET evidence_revision = evidence_revision + 1,
  updated_at = now()
WHERE project_id = @project_id
RETURNING *;

-- name: UpdateProjectResearchCurrentReport :execrows
UPDATE project_research_state
SET current_report_revision = @current_report_revision,
  updated_at = now()
WHERE project_id = @project_id;

-- name: UpdateProjectResearchSelection :execrows
UPDATE project_research_state
SET status = CASE
    WHEN @outcome::text = 'selected' AND status = 'researching' THEN 'selected'
    WHEN @outcome::text = 'rejected' AND status IN ('researching', 'selected') THEN 'rejected'
    ELSE status
  END,
  current_selection_id = @selection_id,
  last_evaluated_report_revision = @report_revision,
  last_evaluated_at = @evaluated_at,
  updated_at = now()
WHERE project_id = @project_id;

-- name: UpdateProjectResearchLastEvaluation :execrows
UPDATE project_research_state
SET last_evaluated_report_revision = @report_revision,
  last_evaluated_at = @evaluated_at,
  updated_at = now()
WHERE project_id = @project_id;

-- name: GetProjectResearchState :one
SELECT *
FROM project_research_state
WHERE project_id = @project_id;

-- name: CountProjectResearchStates :one
SELECT COUNT(*)::bigint
FROM project_research_state AS research
JOIN project ON project.id = research.project_id
WHERE (sqlc.arg('chain_id')::bigint = 0 OR project.chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR research.project_id = sqlc.arg('project_id')::bigint)
  AND (sqlc.arg('status')::text = '' OR research.status = sqlc.arg('status')::text);

-- name: ListProjectResearchStates :many
SELECT
  research.*,
  project.chain_id,
  project.contract,
  COALESCE(selection.outcome, '')::text AS current_selection_outcome
FROM project_research_state AS research
JOIN project ON project.id = research.project_id
LEFT JOIN project_selection AS selection ON selection.id = research.current_selection_id
WHERE (sqlc.arg('chain_id')::bigint = 0 OR project.chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR research.project_id = sqlc.arg('project_id')::bigint)
  AND (sqlc.arg('status')::text = '' OR research.status = sqlc.arg('status')::text)
ORDER BY research.updated_at DESC, research.project_id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
