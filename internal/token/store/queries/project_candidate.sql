-- name: UpsertProjectCandidate :one
INSERT INTO project_candidate (
  chain_id,
  contract,
  creator,
  tx_hash,
  tx_index,
  block_number,
  block_time,
  status
) VALUES (
  @chain_id,
  @contract,
  @creator,
  @tx_hash,
  @tx_index,
  @block_number,
  @block_time,
  @status
)
ON CONFLICT (chain_id, contract) DO UPDATE
SET status = CASE
  WHEN project_candidate.status = 'pending' THEN EXCLUDED.status
  ELSE project_candidate.status
END
RETURNING *;

-- name: BatchUpsertProjectCandidates :exec
INSERT INTO project_candidate (
  chain_id,
  contract,
  creator,
  tx_hash,
  tx_index,
  block_number,
  block_time,
  status
)
SELECT
  unnest(sqlc.arg('chain_ids')::bigint[]),
  unnest(sqlc.arg('contracts')::bytea[]),
  unnest(sqlc.arg('creators')::bytea[]),
  unnest(sqlc.arg('tx_hashes')::bytea[]),
  unnest(sqlc.arg('tx_indexes')::bigint[]),
  unnest(sqlc.arg('block_numbers')::bigint[]),
  unnest(sqlc.arg('block_times')::bigint[]),
  unnest(sqlc.arg('statuses')::text[])
ON CONFLICT (chain_id, contract) DO UPDATE
SET status = CASE
  WHEN project_candidate.status = 'pending' THEN EXCLUDED.status
  ELSE project_candidate.status
END;

-- name: GetProjectCandidate :one
SELECT *
FROM project_candidate
WHERE id = @id;

-- name: GetProjectCandidateByContract :one
SELECT *
FROM project_candidate
WHERE chain_id = @chain_id
  AND contract = @contract;

-- name: CountProjectCandidates :one
SELECT COUNT(*)::bigint
FROM project_candidate
WHERE chain_id = @chain_id
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text);

-- name: ListProjectCandidates :many
SELECT *
FROM project_candidate
WHERE chain_id = @chain_id
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListProjectCandidatesByStatus :many
SELECT *
FROM project_candidate
WHERE status = @status
ORDER BY created_at, id
LIMIT @limit_count;

-- name: MarkProjectCandidateStatus :one
UPDATE project_candidate
SET status = @status
WHERE id = @id
RETURNING *;

-- name: DeleteProjectCandidate :execrows
DELETE FROM project_candidate
WHERE id = @id;
