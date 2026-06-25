-- name: CreateDelivery :one
INSERT INTO notification_deliveries (source, severity, title, body, link, channel, status, telegram_chat, topic_label)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, source, severity, COALESCE(title, '') AS title, body, COALESCE(link, '') AS link, channel, status, telegram_chat, topic_label, provider_message_id, error_message, created_at, sent_at;

-- name: LockTopic :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg('telegram_chat') || ':' || sqlc.arg('label'), 0));

-- name: GetTopic :one
SELECT telegram_chat, label, message_thread_id, created_at
FROM notification_topics
WHERE telegram_chat = $1
  AND label = $2;

-- name: CreateTopic :one
INSERT INTO notification_topics (telegram_chat, label, message_thread_id)
VALUES ($1, $2, $3)
RETURNING telegram_chat, label, message_thread_id, created_at;

-- name: ClaimPendingDeliveries :many
WITH ready AS (
  SELECT id
  FROM notification_deliveries
  WHERE status = 'pending'
    AND next_attempt_at <= NOW()
    AND (locked_at IS NULL OR locked_at < NOW() - sqlc.arg('lock_timeout')::interval)
  ORDER BY created_at ASC, id ASC
  LIMIT $1
  FOR UPDATE SKIP LOCKED
)
UPDATE notification_deliveries AS d
SET locked_at = NOW(),
    locked_by = $2,
    attempts = d.attempts + 1,
    last_attempt_at = NOW()
FROM ready
WHERE d.id = ready.id
RETURNING d.id, d.source, d.severity, COALESCE(d.title, '') AS title, d.body, COALESCE(d.link, '') AS link, d.channel, d.status, d.telegram_chat, d.topic_label, d.provider_message_id, d.error_message, d.created_at, d.sent_at, d.attempts,
  (SELECT t.message_thread_id FROM notification_topics AS t WHERE t.telegram_chat = d.telegram_chat AND t.label = d.topic_label) AS message_thread_id;

-- name: MarkDeliverySent :exec
UPDATE notification_deliveries
SET status = 'sent',
    provider_message_id = $2,
    error_message = NULL,
    sent_at = NOW(),
    locked_at = NULL,
    locked_by = NULL
WHERE id = $1;

-- name: ScheduleDeliveryRetry :exec
UPDATE notification_deliveries
SET error_message = $2,
    next_attempt_at = $3,
    locked_at = NULL,
    locked_by = NULL
WHERE id = $1;

-- name: MarkDeliveryFailed :exec
UPDATE notification_deliveries
SET status = 'failed',
    error_message = $2,
    locked_at = NULL,
    locked_by = NULL
WHERE id = $1;

-- name: CountDeliveries :one
SELECT COUNT(*)::bigint
FROM notification_deliveries
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('severity')::text IS NULL OR severity = sqlc.narg('severity'))
  AND (sqlc.narg('source')::text IS NULL OR source = sqlc.narg('source'))
  AND (sqlc.narg('telegram_chat')::text IS NULL OR telegram_chat = sqlc.narg('telegram_chat'))
  AND (sqlc.narg('topic_label')::text IS NULL OR topic_label = sqlc.narg('topic_label'))
  AND (
    sqlc.narg('keyword')::text IS NULL
    OR title ILIKE sqlc.narg('keyword')
    OR body ILIKE sqlc.narg('keyword')
    OR error_message ILIKE sqlc.narg('keyword')
    OR provider_message_id ILIKE sqlc.narg('keyword')
  );

-- name: ListDeliveries :many
SELECT id, source, severity, COALESCE(title, '') AS title, body, COALESCE(link, '') AS link, channel, status, telegram_chat, topic_label, provider_message_id, error_message, created_at, sent_at
FROM notification_deliveries
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('severity')::text IS NULL OR severity = sqlc.narg('severity'))
  AND (sqlc.narg('source')::text IS NULL OR source = sqlc.narg('source'))
  AND (sqlc.narg('telegram_chat')::text IS NULL OR telegram_chat = sqlc.narg('telegram_chat'))
  AND (sqlc.narg('topic_label')::text IS NULL OR topic_label = sqlc.narg('topic_label'))
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
SELECT id, source, severity, COALESCE(title, '') AS title, body, COALESCE(link, '') AS link, channel, status, telegram_chat, topic_label, provider_message_id, error_message, created_at, sent_at
FROM notification_deliveries
WHERE id = $1;
