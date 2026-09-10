-- name: RequireTraderSyncGrant :one
SELECT EXISTS(SELECT 1 FROM account_module_access m JOIN athena_account a USING(account_id)
WHERE m.account_id=sqlc.arg(owner_id)::uuid AND m.module='trader_sync' AND m.access_level='read_write' AND NOT a.administrator)::boolean;

-- name: ReadSubscriptionRequestResult :one
SELECT payload_digest,result_json FROM trader_sync_request_results WHERE owner_id=$1 AND operation=$2 AND request_id=$3;

-- name: SaveSubscriptionRequestResult :exec
INSERT INTO trader_sync_request_results(owner_id,operation,request_id,payload_digest,result_json) VALUES($1,$2,$3,$4,$5);

-- name: CountLiveSubscriptions :one
SELECT count(*) FROM trader_sync_subscriptions WHERE owner_id=$1 AND desired_state<>'cancelled';

-- name: GetTargetNote :one
SELECT * FROM trader_sync_target_notes WHERE owner_id=$1 AND wallet=$2;

-- name: SaveTargetNote :one
INSERT INTO trader_sync_target_notes(owner_id,wallet,note,revision) VALUES($1,$2,$3,1)
ON CONFLICT(owner_id,wallet) DO UPDATE SET note=EXCLUDED.note,revision=trader_sync_target_notes.revision+1,updated_at=clock_timestamp() RETURNING *;

-- name: GetSubscription :one
SELECT * FROM trader_sync_subscriptions WHERE owner_id=$1 AND id=$2;

-- name: GetLiveSubscription :one
SELECT * FROM trader_sync_subscriptions WHERE owner_id=$1 AND wallet=$2 AND desired_state<>'cancelled';

-- name: CreateSubscription :one
INSERT INTO trader_sync_subscriptions(owner_id,wallet,target_display,desired_state,observation_state) VALUES($1,$2,$3,'enabled','pending_baseline') RETURNING *;

-- name: ChangeSubscription :one
UPDATE trader_sync_subscriptions SET desired_state=sqlc.arg(desired_state),
 observation_state=CASE WHEN sqlc.arg(desired_state)::text='enabled' THEN 'pending_baseline' ELSE observation_state END,
 reason='',revision=revision+1,
 activation_generation=activation_generation+CASE WHEN sqlc.arg(desired_state)::text='enabled' THEN 1 ELSE 0 END,
 effective_at=CASE WHEN sqlc.arg(desired_state)::text='enabled' THEN NULL ELSE effective_at END,
 ended_at=CASE WHEN sqlc.arg(desired_state)::text='enabled' THEN NULL ELSE clock_timestamp() END,updated_at=clock_timestamp()
WHERE owner_id=sqlc.arg(owner_id) AND id=sqlc.arg(id) AND revision=sqlc.arg(expected_revision) RETURNING *;

-- name: CloseSubscriptionIntervals :exec
UPDATE trader_sync_monitor_intervals SET ended_at=clock_timestamp(),state='closed',reason=$3 WHERE owner_id=$1 AND subscription_id=$2 AND ended_at IS NULL;

-- name: FailSubscriptionBaselines :exec
UPDATE trader_sync_baseline_attempts SET state='failed',ended_at=clock_timestamp(),reason=$3 WHERE owner_id=$1 AND subscription_id=$2 AND state='pending';

-- name: RevokeTraderSyncSubscriptions :exec
UPDATE trader_sync_subscriptions SET desired_state='permission_disabled',reason=$2,revision=revision+1,ended_at=clock_timestamp(),updated_at=clock_timestamp() WHERE owner_id=$1 AND desired_state<>'cancelled';

-- name: RevokeTraderSyncIntervals :exec
UPDATE trader_sync_monitor_intervals SET ended_at=clock_timestamp(),state='closed',reason=$2 WHERE owner_id=$1 AND ended_at IS NULL;

-- name: RevokeTraderSyncBaselines :exec
UPDATE trader_sync_baseline_attempts SET state='failed',ended_at=clock_timestamp(),reason=$2 WHERE owner_id=$1 AND state='pending';

-- name: RevokeTraderSyncDeliveries :exec
UPDATE account_notification_deliveries SET eligibility_revoked_at=COALESCE(eligibility_revoked_at,clock_timestamp()),eligibility_revoked_reason=COALESCE(eligibility_revoked_reason,$2),status=CASE WHEN status='pending' THEN 'cancelled' ELSE status END WHERE account_id=$1 AND source='trader_sync';

-- name: RevokeTraderSyncMemberships :exec
UPDATE trader_sync_alert_memberships SET eligibility_revoked_at=COALESCE(eligibility_revoked_at,clock_timestamp()),reason=CASE WHEN eligibility_revoked_at IS NULL THEN $2 ELSE reason END,state=CASE WHEN state='waiting' AND batch_id IS NULL THEN 'cancelled' ELSE state END WHERE owner_id=$1;
