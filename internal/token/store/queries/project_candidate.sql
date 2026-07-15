-- name: UpsertProjectCandidate :one
INSERT INTO project_candidate (
  chain_id,
  contract,
  tx_sender,
  tx_hash,
  tx_index,
  block_number,
  block_time,
  status
) VALUES (
  @chain_id,
  @contract,
  @tx_sender,
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
  tx_sender,
  tx_hash,
  tx_index,
  block_number,
  block_time,
  status
)
SELECT
  unnest(sqlc.arg('chain_ids')::bigint[]),
  unnest(sqlc.arg('contracts')::bytea[]),
  unnest(sqlc.arg('tx_senders')::bytea[]),
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

-- name: ClaimProjectCandidateValidations :many
WITH claimable AS (
  SELECT candidate.id
  FROM project_candidate AS candidate
  WHERE candidate.chain_id = sqlc.arg('chain_id')
    AND candidate.status = 'pending'
    AND (
      candidate.validation_lease_expires_at IS NULL
      OR candidate.validation_lease_expires_at <= now()
    )
  ORDER BY candidate.created_at, candidate.id
  LIMIT sqlc.arg('limit_count')
  FOR UPDATE OF candidate SKIP LOCKED
)
UPDATE project_candidate AS candidate
SET validation_lock_token = sqlc.arg('validation_lock_token')::uuid,
  validation_locked_at = now(),
  validation_lease_expires_at = now() + (sqlc.arg('lease_seconds')::bigint * INTERVAL '1 second')
FROM claimable
WHERE candidate.id = claimable.id
RETURNING candidate.*;

-- name: RenewProjectCandidateValidationClaims :execrows
UPDATE project_candidate
SET validation_lease_expires_at = now() + (sqlc.arg('lease_seconds')::bigint * INTERVAL '1 second')
WHERE status = 'pending'
  AND validation_lock_token = sqlc.arg('validation_lock_token')::uuid
  AND validation_lease_expires_at > now();

-- name: ReleaseProjectCandidateValidationClaims :execrows
UPDATE project_candidate
SET validation_lock_token = NULL,
  validation_locked_at = NULL,
  validation_lease_expires_at = NULL
WHERE status = 'pending'
  AND validation_lock_token = sqlc.arg('validation_lock_token')::uuid;

-- name: CompleteProjectCandidateValidation :one
UPDATE project_candidate
SET status = sqlc.arg('status'),
  validation_lock_token = NULL,
  validation_locked_at = NULL,
  validation_lease_expires_at = NULL
WHERE id = sqlc.arg('id')
  AND status = 'pending'
  AND validation_lock_token = sqlc.arg('validation_lock_token')::uuid
  AND validation_lease_expires_at > now()
RETURNING *;

-- name: DeleteProjectCandidate :execrows
DELETE FROM project_candidate
WHERE id = @id;
