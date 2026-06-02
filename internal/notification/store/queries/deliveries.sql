-- name: CreateDelivery :one
INSERT INTO notification_deliveries (source, severity, title, body, link, channel, status, topic)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, source, severity, COALESCE(title, '') AS title, body, COALESCE(link, '') AS link, channel, status, topic, provider_message_id, error_message, created_at, sent_at;

-- name: MarkDeliverySent :exec
UPDATE notification_deliveries
SET status = 'sent', provider_message_id = $2, error_message = NULL, sent_at = NOW()
WHERE id = $1;

-- name: MarkDeliveryFailed :exec
UPDATE notification_deliveries
SET status = 'failed', error_message = $2
WHERE id = $1;

-- name: CountDeliveries :one
SELECT COUNT(*)::bigint
FROM notification_deliveries
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('severity')::text IS NULL OR severity = sqlc.narg('severity'))
  AND (sqlc.narg('source')::text IS NULL OR source = sqlc.narg('source'))
  AND (sqlc.narg('topic')::text IS NULL OR topic = sqlc.narg('topic'))
  AND (
    sqlc.narg('keyword')::text IS NULL
    OR title ILIKE sqlc.narg('keyword')
    OR body ILIKE sqlc.narg('keyword')
    OR error_message ILIKE sqlc.narg('keyword')
    OR provider_message_id ILIKE sqlc.narg('keyword')
  );

-- name: ListDeliveries :many
SELECT id, source, severity, COALESCE(title, '') AS title, body, COALESCE(link, '') AS link, channel, status, topic, provider_message_id, error_message, created_at, sent_at
FROM notification_deliveries
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('severity')::text IS NULL OR severity = sqlc.narg('severity'))
  AND (sqlc.narg('source')::text IS NULL OR source = sqlc.narg('source'))
  AND (sqlc.narg('topic')::text IS NULL OR topic = sqlc.narg('topic'))
  AND (
    sqlc.narg('keyword')::text IS NULL
    OR title ILIKE sqlc.narg('keyword')
    OR body ILIKE sqlc.narg('keyword')
    OR error_message ILIKE sqlc.narg('keyword')
    OR provider_message_id ILIKE sqlc.narg('keyword')
  )
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: GetDelivery :one
SELECT id, source, severity, COALESCE(title, '') AS title, body, COALESCE(link, '') AS link, channel, status, topic, provider_message_id, error_message, created_at, sent_at
FROM notification_deliveries
WHERE id = $1;
