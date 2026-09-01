-- name: CreateSystemNotificationDelivery :one
INSERT INTO system_notification_deliveries (source, severity, title, body, link, channel, status, telegram_chat, topic_label)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, source, severity, COALESCE(title, '') AS title, body, COALESCE(link, '') AS link, channel, status, telegram_chat, topic_label, provider_message_id, error_message, created_at, sent_at;

-- name: LockSystemNotificationTopic :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg('telegram_chat') || ':' || sqlc.arg('label'), 0));

-- name: GetSystemNotificationTopic :one
SELECT telegram_chat, label, message_thread_id, created_at
FROM system_notification_topics
WHERE telegram_chat = $1
  AND label = $2;

-- name: CreateSystemNotificationTopic :one
INSERT INTO system_notification_topics (telegram_chat, label, message_thread_id)
VALUES ($1, $2, $3)
RETURNING telegram_chat, label, message_thread_id, created_at;

-- name: ClaimPendingSystemNotificationDeliveries :many
WITH ready AS (
  SELECT id
  FROM system_notification_deliveries
  WHERE status = 'pending'
    AND next_attempt_at <= NOW()
    AND (locked_at IS NULL OR locked_at < NOW() - sqlc.arg('lock_timeout')::interval)
  ORDER BY created_at ASC, id ASC
  LIMIT $1
  FOR UPDATE SKIP LOCKED
)
UPDATE system_notification_deliveries AS delivery
SET locked_at = NOW(),
    locked_by = $2,
    attempts = delivery.attempts + 1,
    last_attempt_at = NOW()
FROM ready
WHERE delivery.id = ready.id
RETURNING delivery.id, delivery.source, delivery.severity, COALESCE(delivery.title, '') AS title,
  delivery.body, COALESCE(delivery.link, '') AS link, delivery.channel, delivery.status,
  delivery.telegram_chat, delivery.topic_label, delivery.attempts,
  (SELECT topic.message_thread_id
   FROM system_notification_topics AS topic
   WHERE topic.telegram_chat = delivery.telegram_chat
     AND topic.label = delivery.topic_label) AS message_thread_id;

-- name: MarkSystemNotificationDeliverySent :exec
UPDATE system_notification_deliveries
SET status = 'sent',
    provider_message_id = $2,
    error_message = NULL,
    sent_at = NOW(),
    locked_at = NULL,
    locked_by = NULL
WHERE id = $1;

-- name: ScheduleSystemNotificationDeliveryRetry :exec
UPDATE system_notification_deliveries
SET error_message = $2,
    next_attempt_at = $3,
    locked_at = NULL,
    locked_by = NULL
WHERE id = $1;

-- name: MarkSystemNotificationDeliveryFailed :exec
UPDATE system_notification_deliveries
SET status = 'failed',
    error_message = $2,
    locked_at = NULL,
    locked_by = NULL
WHERE id = $1;

-- name: CountSystemNotificationDeliveries :one
SELECT COUNT(*)::bigint
FROM system_notification_deliveries
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

-- name: ListSystemNotificationDeliveries :many
SELECT id, source, severity, COALESCE(title, '') AS title, body, COALESCE(link, '') AS link,
  channel, status, telegram_chat, topic_label, provider_message_id, error_message, created_at, sent_at
FROM system_notification_deliveries
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

-- name: GetSystemNotificationDelivery :one
SELECT id, source, severity, COALESCE(title, '') AS title, body, COALESCE(link, '') AS link,
  channel, status, telegram_chat, topic_label, provider_message_id, error_message, created_at, sent_at
FROM system_notification_deliveries
WHERE id = $1;

-- name: GetSystemNotificationDeliveryCounts :one
SELECT
  COUNT(*) FILTER (WHERE status = 'pending')::bigint AS pending_count,
  COUNT(*) FILTER (WHERE status = 'pending' AND attempts > 0)::bigint AS retry_count,
  COUNT(*) FILTER (WHERE status = 'failed')::bigint AS failed_count
FROM system_notification_deliveries;
