-- name: GetAccountDeliveryOwner :one
SELECT account_id FROM account_notification_deliveries WHERE id = $1;

-- name: GetAccountDeliveryForPermit :one
SELECT d.*, EXISTS (
  SELECT 1 FROM telegram_bindings b
  WHERE b.account_id = d.account_id AND b.telegram_chat_id = d.telegram_chat_id
    AND b.telegram_user_id = b.telegram_chat_id AND b.revision = d.binding_revision
    AND b.status = 'connected'
) AS binding_eligible
FROM account_notification_deliveries d WHERE d.id = $1 FOR UPDATE OF d;

-- name: GetSystemDeliveryForPermit :one
SELECT d.*, t.message_thread_id FROM system_notification_deliveries d
JOIN system_notification_topics t ON t.telegram_chat = d.telegram_chat AND t.label = d.topic_label
WHERE d.id = $1 FOR UPDATE OF d;

-- name: CreateDeliveryAttempt :one
INSERT INTO notification_delivery_attempts(id, work_kind, work_id, owner_id, sender_incarnation, payload_digest)
VALUES ($1,$2,$3,$4,$5,$6) RETURNING *;

-- name: GetDeliveryAttempt :one
SELECT * FROM notification_delivery_attempts WHERE id = $1;

-- name: GetDeliveryAttemptForUpdate :one
SELECT * FROM notification_delivery_attempts WHERE id = $1 FOR UPDATE;

-- name: AuthorizeAccountDelivery :execrows
UPDATE account_notification_deliveries SET status = 'sending', current_attempt_id = $2,
  attempts = attempts + 1, last_attempt_at = $3
WHERE id = $1 AND status = 'pending' AND eligibility_revoked_at IS NULL AND attempts < 5;

-- name: AuthorizeSystemDelivery :execrows
UPDATE system_notification_deliveries SET status = 'sending', current_attempt_id = $2,
  attempts = attempts + 1, last_attempt_at = $3
WHERE id = $1 AND status = 'pending' AND attempts < 5;

-- name: RecordDeliveryAttemptStarted :execrows
UPDATE notification_delivery_attempts SET started_at = COALESCE(started_at,$2) WHERE id = $1;

-- name: RecordDeliveryAttemptOutcome :execrows
UPDATE notification_delivery_attempts SET outcome = $2, result_at = $3, message_id = $4,
  outcome_code = $5, retry_after = $6 WHERE id = $1 AND result_at IS NULL;

-- name: RecordAccountDeliveryOutcome :execrows
UPDATE account_notification_deliveries SET status = sqlc.arg('status'), provider_message_id = sqlc.arg('provider_message_id'),
  error_message = sqlc.arg('error_message'), sent_at = CASE WHEN sqlc.arg('status')::text = 'sent' THEN sqlc.arg('result_at')::timestamptz ELSE NULL END,
  next_attempt_at = sqlc.arg('next_attempt_at'), locked_at = NULL, locked_by = NULL
WHERE id = sqlc.arg('id') AND current_attempt_id = sqlc.arg('current_attempt_id') AND status = 'sending';

-- name: RecordSystemDeliveryOutcome :execrows
UPDATE system_notification_deliveries SET status = sqlc.arg('status'), provider_message_id = sqlc.arg('provider_message_id'),
  error_message = sqlc.arg('error_message'), sent_at = CASE WHEN sqlc.arg('status')::text = 'sent' THEN sqlc.arg('result_at')::timestamptz ELSE NULL END,
  next_attempt_at = sqlc.arg('next_attempt_at'), locked_at = NULL, locked_by = NULL
WHERE id = sqlc.arg('id') AND current_attempt_id = sqlc.arg('current_attempt_id') AND status = 'sending';
