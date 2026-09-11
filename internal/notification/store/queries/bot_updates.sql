-- name: ConsumeTelegramUpdate :one
INSERT INTO telegram_consumed_updates (update_id, consumed_at)
VALUES ($1, clock_timestamp()) ON CONFLICT (update_id) DO NOTHING
RETURNING update_id;

-- name: CreateTelegramBindingReply :exec
INSERT INTO telegram_binding_replies(update_id, account_id, binding_revision, telegram_chat_id, body, payload_digest, payload)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: CancelTelegramBindingReplies :exec
UPDATE telegram_binding_replies
SET status = CASE WHEN status = 'pending' THEN 'cancelled' ELSE status END,
 eligibility_revoked_at = COALESCE(eligibility_revoked_at, clock_timestamp()),
 eligibility_revoked_reason = COALESCE(eligibility_revoked_reason, 'binding_changed'),
 locked_at = NULL, locked_by = NULL
WHERE account_id = $1 AND status IN ('pending', 'sending');

-- name: CancelTelegramBindingRepliesForBinding :exec
UPDATE telegram_binding_replies
SET status = CASE WHEN status = 'pending' THEN 'cancelled' ELSE status END,
 eligibility_revoked_at = COALESCE(eligibility_revoked_at, clock_timestamp()),
 eligibility_revoked_reason = COALESCE(eligibility_revoked_reason, 'binding_changed'),
 locked_at = NULL, locked_by = NULL
WHERE account_id = $1 AND telegram_chat_id = $2 AND binding_revision = $3
 AND status IN ('pending', 'sending');

-- name: ClaimPendingTelegramBindingReplies :many
WITH ready AS (
 SELECT id FROM telegram_binding_replies
 WHERE status = 'pending' AND attempts < 5 AND eligibility_revoked_at IS NULL
 AND next_attempt_at <= NOW()
 AND (locked_at IS NULL OR locked_at < NOW() - sqlc.arg('lock_timeout')::interval)
 ORDER BY created_at, id LIMIT $1 FOR UPDATE SKIP LOCKED
)
UPDATE telegram_binding_replies AS reply
SET locked_at = NOW(), locked_by = $2
FROM ready WHERE reply.id = ready.id RETURNING reply.*;

-- name: GetReplyDeliveryOwner :one
SELECT account_id FROM telegram_binding_replies WHERE id = $1;

-- name: GetReplyDeliveryForPermit :one
SELECT r.*, (r.account_id IS NULL OR EXISTS (
 SELECT 1 FROM telegram_bindings b WHERE b.account_id = r.account_id
 AND b.telegram_chat_id = r.telegram_chat_id AND b.telegram_user_id = b.telegram_chat_id
 AND b.revision = r.binding_revision AND b.status = 'connected'
)) AS binding_eligible
FROM telegram_binding_replies r WHERE r.id = $1 FOR UPDATE OF r;

-- name: AuthorizeReplyDelivery :execrows
UPDATE telegram_binding_replies SET status = 'sending', current_attempt_id = $2,
 attempts = attempts + 1, last_attempt_at = $3
WHERE id = $1 AND status = 'pending' AND eligibility_revoked_at IS NULL AND attempts < 5;

-- name: RecordReplyDeliveryOutcome :execrows
UPDATE telegram_binding_replies SET status = sqlc.arg('status'), provider_message_id = sqlc.arg('provider_message_id'),
 error_message = sqlc.arg('error_message'), sent_at = CASE WHEN sqlc.arg('status')::text = 'sent' THEN sqlc.arg('result_at')::timestamptz ELSE NULL END,
 next_attempt_at = sqlc.arg('next_attempt_at'), locked_at = NULL, locked_by = NULL
WHERE id = sqlc.arg('id') AND current_attempt_id = sqlc.arg('current_attempt_id') AND status = 'sending';

-- name: ListDispatchReplies :many
SELECT * FROM telegram_binding_replies
WHERE status = 'pending' AND attempts < 5 AND eligibility_revoked_at IS NULL
ORDER BY account_id, next_attempt_at, id;
