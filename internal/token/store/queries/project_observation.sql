-- name: GetCurrentProjectObservation :one
SELECT
  observation.*,
  current.last_checked_at
FROM project_observation_current AS current
JOIN project_observation AS observation ON observation.id = current.observation_id
WHERE current.project_id = @project_id
  AND current.data_type = @data_type;

-- name: InsertProjectObservation :one
INSERT INTO project_observation (
  project_id,
  data_type,
  content_hash,
  payload,
  block_number,
  observed_at
) VALUES (
  @project_id,
  @data_type,
  @content_hash,
  @payload::jsonb,
  sqlc.narg('block_number'),
  @observed_at
)
RETURNING *;

-- name: UpsertCurrentProjectObservation :one
INSERT INTO project_observation_current (
  project_id,
  data_type,
  observation_id,
  last_checked_at
) VALUES (
  @project_id,
  @data_type,
  @observation_id,
  @last_checked_at
)
ON CONFLICT (project_id, data_type) DO UPDATE
SET observation_id = EXCLUDED.observation_id,
  last_checked_at = EXCLUDED.last_checked_at,
  updated_at = now()
RETURNING *;

-- name: TouchCurrentProjectObservation :execrows
UPDATE project_observation_current
SET last_checked_at = @last_checked_at,
  updated_at = now()
WHERE project_id = @project_id
  AND data_type = @data_type;

-- name: ListCurrentProjectObservations :many
SELECT
  observation.*,
  current.last_checked_at
FROM project_observation_current AS current
JOIN project_observation AS observation ON observation.id = current.observation_id
WHERE current.project_id = @project_id
ORDER BY observation.data_type;

-- name: ListProjectObservations :many
SELECT *
FROM project_observation
WHERE project_id = @project_id
  AND (sqlc.arg('data_type')::text = '' OR data_type = sqlc.arg('data_type')::text)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
