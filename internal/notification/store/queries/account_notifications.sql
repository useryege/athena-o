-- name: CreateAccountNotificationDelivery :one
INSERT INTO account_notification_deliveries (
  account_id, idempotency_key, payload_digest, source, severity, title, body,
  link, channel, status, telegram_chat_id, binding_revision, payload, request_digest, activity_id, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
ON CONFLICT (account_id, source, idempotency_key) DO NOTHING
RETURNING id, account_id, idempotency_key, payload_digest, request_digest, source, severity,
  COALESCE(title, '') AS title, body, COALESCE(link, '') AS link, channel, status,
  telegram_chat_id, binding_revision, provider_message_id, error_message,
  created_at, sent_at;

-- name: GetAccountNotificationDeliveryByIdempotency :one
SELECT id, account_id, idempotency_key, payload_digest, request_digest, source, severity,
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
    AND eligibility_revoked_at IS NULL
    AND attempts < 5
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
    locked_by = $2
FROM ready
WHERE delivery.id = ready.id
RETURNING delivery.id, delivery.account_id, delivery.source, delivery.severity,
  COALESCE(delivery.title, '') AS title, delivery.body,
  COALESCE(delivery.link, '') AS link, delivery.channel, delivery.status,
  delivery.telegram_chat_id, delivery.binding_revision, delivery.attempts;

-- name: CancelPendingAccountNotificationDeliveries :execrows
WITH revoked_memberships AS (UPDATE trader_sync_alert_memberships AS m SET eligibility_revoked_at=COALESCE(eligibility_revoked_at,clock_timestamp()),reason=CASE WHEN eligibility_revoked_at IS NULL THEN 'binding_changed' ELSE reason END,state=CASE WHEN state='waiting' AND batch_id IS NULL THEN 'cancelled' ELSE state END WHERE m.owner_id=$1 RETURNING m.activity_id)
UPDATE account_notification_deliveries AS d
SET status = CASE WHEN status = 'pending' THEN 'cancelled' ELSE status END,
    eligibility_revoked_at = COALESCE(eligibility_revoked_at, clock_timestamp()),
    eligibility_revoked_reason = COALESCE(eligibility_revoked_reason, 'binding_changed'),
    error_message = 'telegram binding disconnected',
    locked_at = NULL,
    locked_by = NULL
WHERE account_id = $1
  AND status IN ('pending', 'sending');

-- name: CancelPendingAccountNotificationDeliveriesForBinding :execrows
WITH revoked_memberships AS (UPDATE trader_sync_alert_memberships AS m SET eligibility_revoked_at=COALESCE(eligibility_revoked_at,clock_timestamp()),reason=CASE WHEN eligibility_revoked_at IS NULL THEN 'binding_changed' ELSE reason END,state=CASE WHEN state='waiting' AND batch_id IS NULL THEN 'cancelled' ELSE state END WHERE m.owner_id=$1 AND m.chat_id=$2 AND m.binding_revision=$3 RETURNING m.activity_id)
UPDATE account_notification_deliveries AS d
SET status = CASE WHEN status = 'pending' THEN 'cancelled' ELSE status END,
    eligibility_revoked_at = COALESCE(eligibility_revoked_at, clock_timestamp()),
    eligibility_revoked_reason = COALESCE(eligibility_revoked_reason, 'binding_changed'),
    error_message = 'telegram recipient is unreachable',
    locked_at = NULL,
    locked_by = NULL
WHERE d.account_id = $1
  AND d.telegram_chat_id = $2
  AND d.binding_revision = $3
  AND status IN ('pending', 'sending');

-- name: GetAccountNotificationRuntimeCounts :one
SELECT
  (SELECT COUNT(*)::bigint FROM account_notification_deliveries WHERE status = 'pending') AS pending_count,
  (SELECT COUNT(*)::bigint FROM account_notification_deliveries WHERE status = 'pending' AND attempts > 0) AS retry_count,
  (SELECT COUNT(*)::bigint FROM account_notification_deliveries WHERE status = 'failed') AS failed_count,
  (SELECT COUNT(*)::bigint FROM account_notification_deliveries WHERE status = 'sending') AS sending_count,
  (SELECT COUNT(*)::bigint FROM account_notification_deliveries WHERE status = 'unknown') AS unknown_count,
  (SELECT COUNT(*)::bigint FROM telegram_bindings WHERE status = 'unreachable') AS unreachable_binding_count;

-- name: ListDispatchAccounts :many
SELECT * FROM account_notification_deliveries
WHERE status = 'pending' AND attempts < 5 AND eligibility_revoked_at IS NULL
ORDER BY account_id, next_attempt_at, id;
