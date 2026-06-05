-- name: InsertOutboxEvent :one
WITH inserted AS (
  INSERT INTO outbox_event (
    type,
    aggregate_type,
    aggregate_id,
    chain_id,
    dedup_key,
    payload,
    status,
    next_attempt_at
  ) VALUES (
    @type,
    @aggregate_type,
    @aggregate_id,
    @chain_id,
    @dedup_key,
    @payload::jsonb,
    'pending',
    COALESCE(sqlc.narg('next_attempt_at')::timestamptz, now())
  )
  ON CONFLICT (type, chain_id, dedup_key)
    WHERE status IN ('pending', 'processing')
  DO UPDATE SET
    aggregate_type = EXCLUDED.aggregate_type,
    aggregate_id = EXCLUDED.aggregate_id,
    payload = EXCLUDED.payload,
    next_attempt_at = LEAST(outbox_event.next_attempt_at, EXCLUDED.next_attempt_at),
    updated_at = now()
  RETURNING id, type, aggregate_type, aggregate_id, chain_id, dedup_key, payload, status, attempts, next_attempt_at, locked_at, locked_by, last_error, created_at, updated_at
)
SELECT id, type, aggregate_type, aggregate_id, chain_id, dedup_key, payload, status, attempts, next_attempt_at, locked_at, locked_by, last_error, created_at, updated_at
FROM inserted;

-- name: ClaimOutboxEvents :many
WITH picked AS (
  SELECT id
  FROM outbox_event
  WHERE status = 'pending'
    AND next_attempt_at <= @now::timestamptz
  ORDER BY id
  LIMIT @limit_count
  FOR UPDATE SKIP LOCKED
)
UPDATE outbox_event e
SET status = 'processing',
  locked_at = @now::timestamptz,
  locked_by = @locked_by,
  attempts = attempts + 1,
  updated_at = now()
FROM picked
WHERE e.id = picked.id
RETURNING e.id, e.type, e.aggregate_type, e.aggregate_id, e.chain_id, e.dedup_key, e.payload, e.status, e.attempts, e.next_attempt_at, e.locked_at, e.locked_by, e.last_error, e.created_at, e.updated_at;

-- name: ClaimOutboxEventsByTypes :many
WITH picked AS (
  SELECT id
  FROM outbox_event
  WHERE status = 'pending'
    AND type = ANY(@types::text[])
    AND next_attempt_at <= @now::timestamptz
  ORDER BY id
  LIMIT @limit_count
  FOR UPDATE SKIP LOCKED
)
UPDATE outbox_event e
SET status = 'processing',
  locked_at = @now::timestamptz,
  locked_by = @locked_by,
  attempts = attempts + 1,
  updated_at = now()
FROM picked
WHERE e.id = picked.id
RETURNING e.id, e.type, e.aggregate_type, e.aggregate_id, e.chain_id, e.dedup_key, e.payload, e.status, e.attempts, e.next_attempt_at, e.locked_at, e.locked_by, e.last_error, e.created_at, e.updated_at;

-- name: MarkOutboxEventProcessed :exec
UPDATE outbox_event
SET status = 'processed',
  locked_at = NULL,
  locked_by = NULL,
  last_error = NULL,
  updated_at = now()
WHERE id = @id;

-- name: MarkOutboxEventDiscarded :exec
UPDATE outbox_event
SET status = 'discarded',
  locked_at = NULL,
  locked_by = NULL,
  last_error = @last_error,
  updated_at = now()
WHERE id = @id;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_event
SET status = 'pending',
  locked_at = NULL,
  locked_by = NULL,
  last_error = @last_error,
  next_attempt_at = @next_attempt_at,
  updated_at = now()
WHERE id = @id;
