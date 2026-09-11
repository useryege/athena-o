-- name: GetAccountDeliveryOwner :one
SELECT account_id FROM account_notification_deliveries WHERE id = $1;

-- name: GetAccountDeliveryForPermit :one
SELECT d.*, EXISTS (
  SELECT 1 FROM telegram_bindings b
  WHERE b.account_id = d.account_id AND b.telegram_chat_id = d.telegram_chat_id
    AND b.telegram_user_id = b.telegram_chat_id AND b.revision = d.binding_revision
    AND b.status = 'connected'
) AND (d.source <> 'trader_sync' OR (
 EXISTS (SELECT 1 FROM account_module_access m JOIN athena_account a USING(account_id) WHERE m.account_id=d.account_id AND m.module='trader_sync' AND m.access_level='read_write' AND NOT a.administrator)
 AND (
 EXISTS (SELECT 1 FROM trader_sync_alert_memberships m WHERE m.activity_id=d.activity_id AND m.owner_id=d.account_id AND m.binding_revision=d.binding_revision AND m.chat_id=d.telegram_chat_id AND m.form='ordinary' AND m.eligibility_revoked_at IS NULL)
 OR (d.activity_id IS NULL AND EXISTS (
 SELECT 1 FROM trader_sync_summary_parts p JOIN trader_sync_summary_batches b ON b.id=p.batch_id AND b.owner_id=p.owner_id
 WHERE p.delivery_id=d.id AND p.owner_id=d.account_id AND b.binding_revision=d.binding_revision AND b.chat_id=d.telegram_chat_id AND b.sealed
 AND EXISTS(SELECT 1 FROM trader_sync_summary_part_items i WHERE i.part_id=p.id)
 AND NOT EXISTS(SELECT 1 FROM trader_sync_summary_part_items i JOIN trader_sync_alert_memberships m ON m.activity_id=i.activity_id
 WHERE i.part_id=p.id AND (m.owner_id<>d.account_id OR m.batch_id<>p.batch_id OR m.binding_revision<>d.binding_revision OR m.chat_id<>d.telegram_chat_id OR m.state<>'frozen' OR m.eligibility_revoked_at IS NOT NULL))
 ))
 )
)) AS binding_eligible
FROM account_notification_deliveries d WHERE d.id = $1 FOR UPDATE OF d;

-- name: GetSystemDeliveryForPermit :one
SELECT d.*, t.message_thread_id FROM system_notification_deliveries d
JOIN system_notification_topics t ON t.telegram_chat = d.telegram_chat AND t.label = d.topic_label
WHERE d.id = $1 FOR UPDATE OF d;

-- name: CreateDeliveryAttempt :one
INSERT INTO notification_delivery_attempts(id, work_kind, work_id, owner_id, sender_incarnation, payload_digest, telegram_chat_id, telegram_group)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING *;

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
  outcome_code = $5, retry_after = $6, sender_returned_at = $7, sender_elapsed_ns = $8 WHERE id = $1 AND result_at IS NULL;

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
