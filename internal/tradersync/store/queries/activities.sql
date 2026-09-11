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
INSERT INTO trader_sync_activities(owner_id,subscription_id,source_record_id,interval_id,activation_generation,trade_json,metadata_key,target_display_snapshot,note_snapshot,notification_mode,notification_reason,settled_at,received_at,recorded_at,formation_evidence)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING *;

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


-- name: ReadFormationSnapshot :one
-- One statement snapshot, before current activity insertion. Original arrival
-- predecessor selection deliberately precedes activity/route eligibility joins.
WITH current_arrival AS (
 SELECT r.*,c.owner_id,c.subscription_id,c.baseline_attempt_id,c.activation_generation
 FROM trader_sync_source_records r JOIN trader_sync_source_candidates c ON c.source_record_id=r.id
 WHERE r.id=sqlc.arg(source_id) AND c.owner_id=sqlc.arg(owner_id) AND c.subscription_id=sqlc.arg(subscription_id)
), observed AS MATERIALIZED (SELECT clock_timestamp() AS at), previous_arrival AS (
 SELECT r.*,c.subscription_id,c.baseline_attempt_id,c.activation_generation
 FROM current_arrival cur JOIN trader_sync_source_candidates c ON c.owner_id=cur.owner_id
 JOIN trader_sync_source_records r ON r.id=c.source_record_id
 WHERE r.collector_epoch=cur.collector_epoch AND r.read_sequence<cur.read_sequence
 ORDER BY r.read_sequence DESC,r.id DESC,c.subscription_id LIMIT 1
)
SELECT o.at::timestamptz AS observed_at,
 (SELECT count(*) FROM trader_sync_activities a WHERE a.owner_id=cur.owner_id AND a.recorded_at>o.at-interval '60 seconds' AND a.recorded_at<=o.at)::bigint AS window_count,
 jsonb_build_object('sourceId',cur.id,'subscriptionId',cur.subscription_id,'attemptId',cur.baseline_attempt_id,'generation',cur.activation_generation,
  'epoch',cur.collector_epoch,'sequence',cur.read_sequence,'receivedAt',cur.received_at,'elapsedNs',cur.received_elapsed_ns,
  'confirmation',cur.confirmation_state,'removed',cur.removed,'chatId',b.telegram_chat_id,'bindingRevision',b.revision) AS current_evidence,
 (CASE WHEN prev.id IS NULL THEN 'null'::jsonb ELSE jsonb_build_object(
  'sourceId',prev.id,'subscriptionId',prev.subscription_id,'attemptId',prev.baseline_attempt_id,'generation',prev.activation_generation,
  'epoch',prev.collector_epoch,'sequence',prev.read_sequence,'receivedAt',prev.received_at,'elapsedNs',prev.received_elapsed_ns,
  'confirmation',prev.confirmation_state,'removed',prev.removed,'formed',pa.id IS NOT NULL,'mode',COALESCE(pa.notification_mode,''),
  'chatId',pd.telegram_chat_id,'bindingRevision',pd.binding_revision,'deliveryStatus',COALESCE(pd.status,''),'deliveryAttempts',COALESCE(pd.attempts,0),
  'deliveryEligible',COALESCE(pd.eligibility_revoked_at IS NULL AND b.status='connected' AND b.telegram_user_id=b.telegram_chat_id AND pd.telegram_chat_id=b.telegram_chat_id AND pd.binding_revision=b.revision,false),
  'everStarted',EXISTS(SELECT 1 FROM notification_delivery_attempts a WHERE a.work_kind='account' AND a.work_id=pd.id AND a.started_at IS NOT NULL)
 ) END)::jsonb AS previous_evidence,
 jsonb_build_object(
  'ordinaryPending',(SELECT count(*) FROM account_notification_deliveries d WHERE d.account_id=cur.owner_id AND d.activity_id IS NOT NULL AND d.status='pending'),
  'ordinarySending',(SELECT count(*) FROM account_notification_deliveries d WHERE d.account_id=cur.owner_id AND d.activity_id IS NOT NULL AND d.status='sending'),
  'summaryWaiting',(SELECT count(*) FROM trader_sync_alert_memberships m WHERE m.owner_id=cur.owner_id AND m.form='summary' AND m.state='waiting' AND m.batch_id IS NULL),
  'partPending',(SELECT count(*) FROM trader_sync_summary_parts p JOIN account_notification_deliveries d ON d.id=p.delivery_id WHERE p.owner_id=cur.owner_id AND d.status='pending'),
  'partSending',(SELECT count(*) FROM trader_sync_summary_parts p JOIN account_notification_deliveries d ON d.id=p.delivery_id WHERE p.owner_id=cur.owner_id AND d.status='sending'),
  'replyPending',(SELECT count(*) FROM telegram_binding_replies r WHERE r.account_id=cur.owner_id AND r.status='pending'),
  'replySending',(SELECT count(*) FROM telegram_binding_replies r WHERE r.account_id=cur.owner_id AND r.status='sending'),
  'policy','committed_owner_statuses_all_routes_including_revoked; predecessor_eligibility_and_route_separate'
 ) AS queue_evidence
FROM current_arrival cur CROSS JOIN observed o
LEFT JOIN telegram_bindings b ON b.account_id=cur.owner_id
LEFT JOIN previous_arrival prev ON true
LEFT JOIN trader_sync_activities pa ON pa.owner_id=cur.owner_id AND pa.subscription_id=prev.subscription_id AND pa.source_record_id=prev.id
LEFT JOIN account_notification_deliveries pd ON pd.account_id=cur.owner_id AND pd.activity_id=pa.id;
