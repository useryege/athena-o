-- name: RegisterSender :exec
INSERT INTO notification_sender_instances(incarnation,hostname,process_id,process_identity) VALUES ($1,$2,$3,$4);

-- name: ListUnstoppedSenders :many
SELECT * FROM notification_sender_instances WHERE stopped_at IS NULL ORDER BY registered_at;

-- name: GetSenderInstance :one
SELECT * FROM notification_sender_instances WHERE incarnation=$1;

-- name: StopSenderInstance :execrows
UPDATE notification_sender_instances SET stopped_at=clock_timestamp(),stop_confirmation=$2
WHERE incarnation=$1 AND stopped_at IS NULL;

-- name: ListSenderRecoveryAttempts :many
SELECT a.* FROM notification_delivery_attempts a WHERE sender_incarnation=$1 AND (
 EXISTS (SELECT 1 FROM account_notification_deliveries d WHERE a.work_kind='account' AND d.id=a.work_id AND d.current_attempt_id=a.id AND d.status='sending') OR
 EXISTS (SELECT 1 FROM system_notification_deliveries d WHERE a.work_kind='system' AND d.id=a.work_id AND d.current_attempt_id=a.id AND d.status='sending') OR
 EXISTS (SELECT 1 FROM telegram_binding_replies d WHERE a.work_kind='reply' AND d.id=a.work_id AND d.current_attempt_id=a.id AND d.status='sending'))
ORDER BY a.authorized_at;

-- name: ListBudgetEvidence :many
SELECT a.* FROM notification_delivery_attempts a
WHERE started_at >= $1 OR result_at >= $1 OR result_at + retry_after >= $1 OR result_at IS NULL
ORDER BY COALESCE(started_at,result_at,authorized_at);

-- name: HasSenderHistory :one
SELECT EXISTS(SELECT 1 FROM notification_sender_instances) OR EXISTS(SELECT 1 FROM notification_delivery_attempts) AS has_history;

-- name: ListUnreleasedRetryAfters :many
SELECT id, retry_after FROM notification_delivery_attempts
WHERE retry_after > interval '0 seconds' AND retry_after_released_at IS NULL;

-- name: ReleaseRetryAfter :execrows
UPDATE notification_delivery_attempts SET retry_after_released_at=clock_timestamp()
WHERE id=$1 AND retry_after > interval '0 seconds' AND retry_after_released_at IS NULL;
