-- name: LockCollectorControl :one
SELECT * FROM trader_sync_collector_control WHERE singleton FOR UPDATE;
-- name: ClaimCollectorOwnership :one
UPDATE trader_sync_collector_control SET fencing_token=fencing_token+1,owner_id=$1,active_epoch=NULL WHERE singleton AND fencing_token<9223372036854775807 RETURNING *;
-- name: ReleaseCollectorOwnership :execrows
UPDATE trader_sync_collector_control SET owner_id=NULL,active_epoch=NULL WHERE singleton AND fencing_token=$1;
-- name: CreateCollectorEpoch :one
INSERT INTO trader_sync_collector_epochs(fencing_token) VALUES($1) RETURNING *;
-- name: ActivateCollectorEpoch :execrows
UPDATE trader_sync_collector_control SET active_epoch=$2 WHERE singleton AND fencing_token=$1 AND owner_id IS NOT NULL AND active_epoch IS NULL;
-- name: GetCollectorEpoch :one
SELECT * FROM trader_sync_collector_epochs WHERE id=$1;
-- name: EndCollectorEpoch :execrows
UPDATE trader_sync_collector_epochs SET ended_at=clock_timestamp(),reason=$2 WHERE id=$1 AND ended_at IS NULL;
-- name: ClearCollectorEpoch :execrows
UPDATE trader_sync_collector_control SET active_epoch=NULL WHERE singleton AND active_epoch=$1;
-- name: RecordCollectorInterruption :exec
INSERT INTO trader_sync_interruptions(collector_epoch,reason,last_received_at,last_read_sequence)
SELECT e.id,e.reason,e.last_received_at,e.last_read_sequence FROM trader_sync_collector_epochs e WHERE e.id=$1 AND e.ended_at IS NOT NULL ON CONFLICT(collector_epoch) DO NOTHING;
-- name: AckCollectorFilters :execrows
UPDATE trader_sync_collector_epochs SET filter_revision=$2 WHERE id=$1 AND ended_at IS NULL AND filter_revision<$2;
-- name: ObserveCollectorReceipt :execrows
UPDATE trader_sync_collector_epochs SET last_received_at=GREATEST(last_received_at,$2),last_read_sequence=GREATEST(last_read_sequence,$3) WHERE id=$1 AND ended_at IS NULL;
-- name: ListCollectorTargets :many
SELECT DISTINCT wallet FROM trader_sync_subscriptions WHERE desired_state='enabled' ORDER BY wallet;
-- name: ListSubscriptionsNeedingBaseline :many
SELECT s.* FROM trader_sync_subscriptions s WHERE s.desired_state='enabled'
AND NOT EXISTS(SELECT 1 FROM trader_sync_baseline_attempts a WHERE a.subscription_id=s.id AND a.state='pending')
AND NOT EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i JOIN trader_sync_collector_epochs e ON e.id=i.collector_epoch WHERE i.subscription_id=s.id AND i.activation_generation=s.activation_generation AND i.ended_at IS NULL AND e.ended_at IS NULL)
ORDER BY s.owner_id,s.id;
-- name: GetBaselineSubscription :one
SELECT * FROM trader_sync_subscriptions WHERE id=$1;
-- name: CreateBaselineAttempt :one
INSERT INTO trader_sync_baseline_attempts(owner_id,subscription_id,activation_generation,expected_revision,collector_epoch,registered_high,registered_sequence)
VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(subscription_id) WHERE state='pending' DO NOTHING RETURNING *;
-- name: ListPendingBaselines :many
SELECT * FROM trader_sync_baseline_attempts WHERE state='pending' ORDER BY created_at,id;
-- name: GetBaselineAttempt :one
SELECT * FROM trader_sync_baseline_attempts WHERE id=$1;
-- name: BindBaselineEpoch :execrows
UPDATE trader_sync_baseline_attempts SET collector_epoch=$2,registered_high=$3,registered_sequence=$4
WHERE id=$1 AND state='pending' AND collector_epoch IS NULL;
-- name: SaveBaselineBoundary :execrows
UPDATE trader_sync_baseline_attempts SET filter_revision=$2,candidate_effective_at=$3
WHERE id=$1 AND state='pending' AND $3::timestamptz>clock_timestamp();
-- name: SucceedBaselineAttempt :one
UPDATE trader_sync_baseline_attempts SET state='succeeded',effective_at=candidate_effective_at
WHERE id=$1 AND state='pending' AND candidate_effective_at<=clock_timestamp() AND filter_revision IS NOT NULL AND registered_sequence IS NOT NULL RETURNING *;
-- name: InsertBaselineInterval :exec
INSERT INTO trader_sync_monitor_intervals(owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at)
SELECT a.owner_id,a.subscription_id,a.id,a.activation_generation,a.collector_epoch,a.filter_revision,a.expected_revision,a.registered_high,a.candidate_effective_at,a.effective_at FROM trader_sync_baseline_attempts a WHERE a.id=$1 AND a.state='succeeded' ON CONFLICT(baseline_attempt_id) DO NOTHING;
-- name: ProjectBaselineHealthy :execrows
UPDATE trader_sync_subscriptions SET observation_state='healthy',reason='',effective_at=$2,ended_at=NULL,updated_at=clock_timestamp()
WHERE id=$1 AND desired_state='enabled' AND revision=$3 AND activation_generation=$4;
-- name: FailBaselineAttempt :execrows
UPDATE trader_sync_baseline_attempts SET state='failed',ended_at=clock_timestamp(),reason=$2 WHERE id=$1 AND state='pending';
-- name: ListClosedEpochOwners :many
SELECT DISTINCT a.owner_id FROM trader_sync_baseline_attempts a JOIN trader_sync_collector_epochs e ON e.id=a.collector_epoch WHERE e.ended_at IS NOT NULL
AND (a.state='pending' OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=a.id AND i.ended_at IS NULL));
-- name: CloseStoppedEpochIntervals :exec
UPDATE trader_sync_monitor_intervals i SET ended_at=e.ended_at,state='closed',reason=e.reason FROM trader_sync_collector_epochs e
WHERE i.owner_id=$1 AND i.collector_epoch=e.id AND e.ended_at IS NOT NULL AND i.ended_at IS NULL;
-- name: ProjectStoppedEpochSubscriptions :exec
UPDATE trader_sync_subscriptions s SET observation_state='interrupted',reason='collector_interrupted',updated_at=clock_timestamp()
WHERE s.owner_id=$1 AND s.desired_state='enabled' AND EXISTS(SELECT 1 FROM trader_sync_baseline_attempts a JOIN trader_sync_collector_epochs e ON e.id=a.collector_epoch WHERE a.subscription_id=s.id AND a.activation_generation=s.activation_generation AND a.expected_revision=s.revision AND e.ended_at IS NOT NULL AND (a.state='pending' OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=a.id AND i.ended_at IS NULL)));
-- name: FailStoppedEpochBaselines :exec
UPDATE trader_sync_baseline_attempts a SET state='failed',ended_at=e.ended_at,reason=e.reason FROM trader_sync_collector_epochs e
WHERE a.owner_id=$1 AND a.collector_epoch=e.id AND e.ended_at IS NOT NULL AND a.state='pending';
-- name: BaselineDatabaseNow :one
SELECT clock_timestamp()::timestamptz AS now;

-- name: HasBaselineInterval :one
SELECT EXISTS(SELECT 1 FROM trader_sync_monitor_intervals WHERE baseline_attempt_id=$1);

-- name: ListCheckpointIntervals :many
SELECT i.*,s.wallet FROM trader_sync_monitor_intervals i
JOIN trader_sync_subscriptions s ON s.owner_id=i.owner_id AND s.id=i.subscription_id
JOIN trader_sync_collector_epochs ep ON ep.id=i.collector_epoch
WHERE i.collector_epoch=sqlc.arg(epoch)::bigint AND i.ended_at IS NULL AND ep.ended_at IS NULL
AND s.desired_state='enabled' AND s.activation_generation=i.activation_generation
AND (sqlc.narg(after_id)::uuid IS NULL OR i.id>sqlc.narg(after_id))
ORDER BY i.id LIMIT sqlc.arg(row_limit)::integer;

-- name: GetCheckpointInterval :one
SELECT i.*,s.wallet,s.desired_state,s.activation_generation AS current_generation,ep.ended_at AS epoch_ended_at
FROM trader_sync_monitor_intervals i
JOIN trader_sync_subscriptions s ON s.owner_id=i.owner_id AND s.id=i.subscription_id
JOIN trader_sync_collector_epochs ep ON ep.id=i.collector_epoch
WHERE i.owner_id=$1 AND i.id=$2 FOR UPDATE OF i;

-- name: SaveIntervalCheckpoint :execrows
UPDATE trader_sync_monitor_intervals SET last_reliable_at=sqlc.arg(observed_at)::timestamptz
WHERE owner_id=sqlc.arg(owner_id)::uuid AND id=sqlc.arg(id)::uuid
AND collector_epoch=sqlc.arg(epoch)::bigint AND activation_generation=sqlc.arg(generation)::bigint
AND ended_at IS NULL;
