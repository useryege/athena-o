-- Research lifecycle persistence.
-- name: CreateProjectResearchState :one
INSERT INTO project_research_state (
  project_id,
  attention_expiry_block_time
) VALUES (
  @project_id,
  @attention_expiry_block_time
)
ON CONFLICT (project_id) DO UPDATE
SET updated_at = project_research_state.updated_at
RETURNING *;

-- name: ExpireProjectResearchStatesForBlock :execrows
UPDATE project_research_state AS research
SET status = 'expired',
  expired_block_number = sqlc.arg('block_number')::bigint,
  expired_block_time = sqlc.arg('block_time')::bigint,
  updated_at = now()
FROM project
WHERE project.id = research.project_id
  AND project.chain_id = @chain_id
  AND research.status = 'researching'
  AND research.attention_expiry_block_time <= sqlc.arg('block_time')::bigint;

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
  project.block_number AS attention_start_block_number,
  project.block_time AS attention_start_block_time,
  COALESCE(selection.outcome, '')::text AS current_selection_outcome
FROM project_research_state AS research
JOIN project ON project.id = research.project_id
LEFT JOIN project_selection AS selection ON selection.id = research.current_selection_id
WHERE (sqlc.arg('chain_id')::bigint = 0 OR project.chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR research.project_id = sqlc.arg('project_id')::bigint)
  AND (sqlc.arg('status')::text = '' OR research.status = sqlc.arg('status')::text)
ORDER BY research.updated_at DESC, research.project_id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
