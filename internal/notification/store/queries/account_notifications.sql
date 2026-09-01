-- name: CreateAccountNotificationDelivery :one
INSERT INTO account_notification_deliveries (
  account_id, idempotency_key, payload_digest, source, severity, title, body,
  link, channel, status, telegram_chat_id, binding_revision
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (account_id, source, idempotency_key) DO NOTHING
RETURNING id, account_id, idempotency_key, payload_digest, source, severity,
  COALESCE(title, '') AS title, body, COALESCE(link, '') AS link, channel, status,
  telegram_chat_id, binding_revision, provider_message_id, error_message,
  created_at, sent_at;

-- name: GetAccountNotificationDeliveryByIdempotency :one
SELECT id, account_id, idempotency_key, payload_digest, source, severity,
  COALESCE(title, '') AS title, body, COALESCE(link, '') AS link, channel, status,
  telegram_chat_id, binding_revision, provider_message_id, error_message,
  created_at, sent_at
FROM account_notification_deliveries
WHERE account_id = $1
  AND source = $2
  AND idempotency_key = $3;

-- name: ClaimPendingAccountNotificationDeliveries :many
WITH ready AS (
  SELECT id
  FROM account_notification_deliveries
  WHERE status = 'pending'
    AND next_attempt_at <= NOW()
    AND (locked_at IS NULL OR locked_at < NOW() - sqlc.arg('lock_timeout')::interval)
    AND EXISTS (
      SELECT 1
      FROM telegram_bindings AS binding
      WHERE binding.account_id = account_notification_deliveries.account_id
        AND binding.telegram_chat_id = account_notification_deliveries.telegram_chat_id
        AND binding.revision = account_notification_deliveries.binding_revision
        AND binding.status = 'connected'
    )
  ORDER BY created_at ASC, id ASC
  LIMIT $1
  FOR UPDATE SKIP LOCKED
)
UPDATE account_notification_deliveries AS delivery
SET locked_at = NOW(),
    locked_by = $2,
    attempts = delivery.attempts + 1,
    last_attempt_at = NOW()
FROM ready
WHERE delivery.id = ready.id
RETURNING delivery.id, delivery.account_id, delivery.source, delivery.severity,
  COALESCE(delivery.title, '') AS title, delivery.body,
  COALESCE(delivery.link, '') AS link, delivery.channel, delivery.status,
  delivery.telegram_chat_id, delivery.binding_revision, delivery.attempts;

-- name: GetPendingAccountNotificationDeliveryForUpdate :one
SELECT id
FROM account_notification_deliveries
WHERE id = $1
  AND account_id = $2
  AND telegram_chat_id = $3
  AND binding_revision = $4
  AND status = 'pending'
FOR UPDATE;

-- name: MarkAccountNotificationDeliverySent :exec
UPDATE account_notification_deliveries
SET status = 'sent',
    provider_message_id = $2,
    error_message = NULL,
    sent_at = NOW(),
    locked_at = NULL,
    locked_by = NULL
WHERE id = $1;

-- name: ScheduleAccountNotificationDeliveryRetry :execrows
UPDATE account_notification_deliveries
SET error_message = sqlc.arg('error_message'),
    next_attempt_at = sqlc.arg('next_attempt_at'),
    locked_at = NULL,
    locked_by = NULL
WHERE id = sqlc.arg('id')
  AND account_id = sqlc.arg('account_id')
  AND telegram_chat_id = sqlc.arg('telegram_chat_id')
  AND binding_revision = sqlc.arg('binding_revision')
  AND status = 'pending';

-- name: MarkAccountNotificationDeliveryFailed :execrows
UPDATE account_notification_deliveries
SET status = 'failed',
    error_message = sqlc.arg('error_message'),
    locked_at = NULL,
    locked_by = NULL
WHERE id = sqlc.arg('id')
  AND account_id = sqlc.arg('account_id')
  AND telegram_chat_id = sqlc.arg('telegram_chat_id')
  AND binding_revision = sqlc.arg('binding_revision')
  AND status = 'pending';

-- name: CancelPendingAccountNotificationDeliveries :execrows
UPDATE account_notification_deliveries
SET status = 'cancelled',
    error_message = 'telegram binding disconnected',
    locked_at = NULL,
    locked_by = NULL
WHERE account_id = $1
  AND status = 'pending';

-- name: CancelPendingAccountNotificationDeliveriesForBinding :execrows
UPDATE account_notification_deliveries
SET status = 'cancelled',
    error_message = 'telegram recipient is unreachable',
    locked_at = NULL,
    locked_by = NULL
WHERE account_id = $1
  AND telegram_chat_id = $2
  AND binding_revision = $3
  AND status = 'pending';

-- name: GetAccountNotificationRuntimeCounts :one
SELECT
  (SELECT COUNT(*)::bigint FROM account_notification_deliveries WHERE status = 'pending') AS pending_count,
  (SELECT COUNT(*)::bigint FROM account_notification_deliveries WHERE status = 'pending' AND attempts > 0) AS retry_count,
  (SELECT COUNT(*)::bigint FROM account_notification_deliveries WHERE status = 'failed') AS failed_count,
  (SELECT COUNT(*)::bigint FROM telegram_bindings WHERE status = 'unreachable') AS unreachable_binding_count;
