-- name: ActivityDatabaseTime :one
SELECT clock_timestamp()::timestamptz;

-- name: LockActivityTransaction :exec
SELECT pg_advisory_xact_lock(hashtextextended('athena:trade:' || encode(sqlc.arg(transaction_hash)::bytea,'hex'),0));

-- name: GetProjectionSource :one
SELECT * FROM trader_sync_source_records WHERE id=$1 FOR SHARE;

-- name: GetProjectionEligibility :one
SELECT c.*, s.desired_state, s.activation_generation AS current_generation,s.wallet,s.target_display,
 b.state AS baseline_state, i.id AS interval_id,i.effective_at,i.ended_at
FROM trader_sync_source_candidates c
JOIN trader_sync_subscriptions s ON s.owner_id=c.owner_id AND s.id=c.subscription_id
JOIN trader_sync_baseline_attempts b ON b.owner_id=c.owner_id AND b.subscription_id=c.subscription_id AND b.id=c.baseline_attempt_id
LEFT JOIN trader_sync_monitor_intervals i ON i.owner_id=c.owner_id AND i.subscription_id=c.subscription_id AND i.baseline_attempt_id=c.baseline_attempt_id AND i.activation_generation=c.activation_generation
WHERE c.source_record_id=$1 AND c.owner_id=$2 AND c.subscription_id=$3;

-- name: GetProjectedActivity :one
SELECT * FROM trader_sync_activities WHERE owner_id=$1 AND subscription_id=$2 AND source_record_id=$3;

-- name: CountActivityWindow :one
SELECT count(*)::bigint FROM trader_sync_activities WHERE owner_id=$1 AND recorded_at>$2::timestamptz-interval '60 seconds' AND recorded_at<=$2;

-- name: InsertActivity :one
INSERT INTO trader_sync_activities(owner_id,subscription_id,source_record_id,interval_id,activation_generation,trade_json,metadata_key,target_display_snapshot,note_snapshot,notification_mode,notification_reason,settled_at,received_at,recorded_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING *;

-- name: InsertAlertMembership :exec
INSERT INTO trader_sync_alert_memberships(activity_id,owner_id,binding_revision,chat_id,form,state,created_at)VALUES($1,$2,$3,$4,$5,$6,$7);

-- name: FinishProjectionCandidate :exec
UPDATE trader_sync_source_candidates SET disposition=$4,disposition_reason=$5 WHERE source_record_id=$1 AND owner_id=$2 AND subscription_id=$3;

-- name: HasFinalityAnomaly :one
SELECT EXISTS(SELECT 1 FROM trader_sync_finality_anomalies WHERE chain_id=$1 AND transaction_hash=$2)::boolean;

-- name: FindPublishedTransaction :many
SELECT DISTINCT r.block_hash FROM trader_sync_source_records r JOIN trader_sync_activities a ON a.source_record_id=r.id WHERE r.chain_id=$1 AND r.transaction_hash=$2;

-- name: InsertFinalityAnomaly :exec
INSERT INTO trader_sync_finality_anomalies(chain_id,transaction_hash,published_block_hash,conflicting_block_hash,reason)VALUES($1,$2,$3,$4,$5) ON CONFLICT(chain_id,transaction_hash) DO UPDATE SET conflicting_block_hash=COALESCE(trader_sync_finality_anomalies.conflicting_block_hash,EXCLUDED.conflicting_block_hash);

-- name: ProjectionSources :many
SELECT r.* FROM trader_sync_source_records r WHERE
 EXISTS(SELECT 1 FROM trader_sync_source_candidates c WHERE c.source_record_id=r.id AND c.disposition='pending')
 OR (NOT r.metadata_complete AND r.confirmation_state<>'invalid' AND NOT EXISTS(SELECT 1 FROM trader_sync_finality_anomalies f WHERE f.chain_id=r.chain_id AND f.transaction_hash=r.transaction_hash) AND EXISTS(SELECT 1 FROM trader_sync_activities a WHERE a.source_record_id=r.id))
 OR (r.removed AND EXISTS(SELECT 1 FROM trader_sync_activities a WHERE a.source_record_id=r.id) AND NOT EXISTS(SELECT 1 FROM trader_sync_finality_anomalies f WHERE f.chain_id=r.chain_id AND f.transaction_hash=r.transaction_hash))
ORDER BY r.checked_at NULLS FIRST,r.id LIMIT $1;

-- name: ProjectionCandidates :many
SELECT * FROM trader_sync_source_candidates WHERE source_record_id=$1 AND disposition='pending' ORDER BY owner_id,subscription_id;

-- name: SaveProjectionEvidence :exec
UPDATE trader_sync_source_records SET
 confirmation_state=CASE WHEN removed THEN 'invalid' ELSE sqlc.arg(confirmation_state)::text END,
 confirmation_reason=CASE WHEN removed THEN 'removed' ELSE sqlc.arg(confirmation_reason)::text END,
 checked_at=sqlc.arg(checked_at),settled_at=sqlc.narg(settled_at),
 source_version=sqlc.arg(source_version),trade_json=sqlc.narg(trade_json)
WHERE id=sqlc.arg(id);

-- name: SetProjectionMetadataComplete :exec
UPDATE trader_sync_source_records SET metadata_complete=$2 WHERE id=$1;

-- name: ReadOwnerActivitySnapshot :one
SELECT COALESCE(max(id),0)::bigint FROM trader_sync_activities WHERE owner_id=$1;

-- name: ReadOwnerActivities :many
SELECT a.*,m.metadata_json FROM trader_sync_activities a JOIN trader_sync_market_metadata m ON m.cache_key=a.metadata_key WHERE a.owner_id=$1 AND a.id<=$2 ORDER BY a.id DESC LIMIT $3;

-- name: FinishInvalidProjectionCandidates :exec
UPDATE trader_sync_source_candidates SET disposition='ineligible',disposition_reason='invalid_source'
WHERE source_record_id=$1 AND disposition='pending';

-- name: HasPublishedSource :one
SELECT EXISTS(SELECT 1 FROM trader_sync_activities WHERE source_record_id=$1)::boolean;
