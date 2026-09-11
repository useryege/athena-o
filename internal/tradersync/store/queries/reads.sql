-- Member queries require the existing account gate and current grant in the caller.
-- name: ReadCommittedOwnerSnapshot :one
SELECT COALESCE(max(id),0)::bigint AS snapshot FROM trader_sync_activities WHERE owner_id=$1;

-- name: HasNewerOwnerActivity :one
SELECT EXISTS(SELECT 1 FROM trader_sync_activities a WHERE a.owner_id=sqlc.arg(owner_id)::uuid
AND (sqlc.narg(subscription_id)::uuid IS NULL OR a.subscription_id=sqlc.narg(subscription_id))
AND (sqlc.narg(batch_id)::bigint IS NULL OR EXISTS(SELECT 1 FROM trader_sync_alert_memberships fm WHERE fm.owner_id=a.owner_id AND fm.activity_id=a.id AND fm.batch_id=sqlc.narg(batch_id)))
AND (sqlc.narg(from_time)::timestamptz IS NULL OR a.settled_at>=sqlc.narg(from_time))
AND (sqlc.narg(to_time)::timestamptz IS NULL OR a.settled_at<sqlc.narg(to_time)) AND a.id>sqlc.arg(snapshot_id)::bigint);

-- name: ReadActivityFacts :many
SELECT a.*,md.metadata_json,r.chain_id,r.exchange_address,r.transaction_hash,r.block_hash,r.block_number,r.log_index,
 fa.reason AS anomaly_reason,fa.detected_at AS anomaly_detected_at,fa.published_block_hash,fa.conflicting_block_hash,
 CASE WHEN d.id IS NULL THEN NULL ELSE jsonb_build_object(
 'ID',d.id,'Status',d.status,'Reason',COALESCE(d.error_message,d.eligibility_revoked_reason,''),
 'AuthorizedAt',latest.authorized_at,'StartedAt',latest.started_at,'ResultAt',latest.result_at,
 'MessageID',COALESCE(latest.message_id,d.provider_message_id),'AttemptCount',COALESCE(latest.attempt_count,0),
 'LatestAttempt',CASE WHEN latest.id IS NULL THEN NULL ELSE jsonb_build_object('Index',latest.attempt_count,
 'AuthorizedAt',latest.authorized_at,'StartedAt',latest.started_at,'ResultAt',latest.result_at,
 'Status',COALESCE(latest.outcome,'sending'),'Reason',COALESCE(latest.outcome_code,'')) END) END::jsonb AS delivery_json,
 CASE WHEN am.form='summary' THEN jsonb_build_object('Phase',CASE WHEN am.state='cancelled' THEN 'cancelled_before_freeze' ELSE am.state END,
 'Reason',am.reason,'BatchID',am.batch_id,'OldestAt',COALESCE(b.oldest_at,a.recorded_at),'FirstStartedAt',b.first_started_at,
 'RelatedPartCounts',(SELECT jsonb_build_object('Total',count(*),'Pending',count(*) FILTER(WHERE counted.status='pending'),'Sending',count(*) FILTER(WHERE counted.status='sending'),'Sent',count(*) FILTER(WHERE counted.status='sent'),'Failed',count(*) FILTER(WHERE counted.status='failed'),'Unknown',count(*) FILTER(WHERE counted.status='unknown'),'Cancelled',count(*) FILTER(WHERE counted.status='cancelled')) FROM (SELECT d.status FROM trader_sync_summary_part_items pi JOIN trader_sync_summary_parts p ON p.id=pi.part_id AND p.owner_id=pi.owner_id JOIN account_notification_deliveries d ON d.id=p.delivery_id AND d.account_id=p.owner_id WHERE pi.owner_id=a.owner_id AND pi.activity_id=a.id) counted),'BatchPartCounts',(SELECT jsonb_build_object('Total',count(*),'Pending',count(*) FILTER(WHERE counted.status='pending'),'Sending',count(*) FILTER(WHERE counted.status='sending'),'Sent',count(*) FILTER(WHERE counted.status='sent'),'Failed',count(*) FILTER(WHERE counted.status='failed'),'Unknown',count(*) FILTER(WHERE counted.status='unknown'),'Cancelled',count(*) FILTER(WHERE counted.status='cancelled')) FROM (SELECT d.status FROM trader_sync_summary_parts p JOIN account_notification_deliveries d ON d.id=p.delivery_id AND d.account_id=p.owner_id WHERE p.owner_id=a.owner_id AND p.batch_id=am.batch_id) counted)) ELSE NULL END::jsonb AS summary_json
FROM trader_sync_activities a
JOIN trader_sync_source_records r ON r.id=a.source_record_id
JOIN trader_sync_market_metadata md ON md.cache_key=a.metadata_key
LEFT JOIN trader_sync_finality_anomalies fa ON fa.chain_id=r.chain_id AND fa.transaction_hash=r.transaction_hash
LEFT JOIN account_notification_deliveries d ON d.account_id=a.owner_id AND d.activity_id=a.id
LEFT JOIN LATERAL (SELECT t.id,t.authorized_at,t.started_at,t.result_at,t.message_id,t.outcome,t.outcome_code,count(*) OVER() AS attempt_count
 FROM notification_delivery_attempts t WHERE t.work_kind='account' AND t.work_id=d.id AND t.owner_id=d.account_id
 ORDER BY t.authorized_at DESC,t.id DESC LIMIT 1) latest ON true
LEFT JOIN trader_sync_alert_memberships am ON am.owner_id=a.owner_id AND am.activity_id=a.id
LEFT JOIN trader_sync_summary_batches b ON b.owner_id=am.owner_id AND b.id=am.batch_id
WHERE a.owner_id=sqlc.arg(owner_id)::uuid
AND (sqlc.narg(subscription_id)::uuid IS NULL OR a.subscription_id=sqlc.narg(subscription_id))
AND (sqlc.narg(batch_id)::bigint IS NULL OR EXISTS(SELECT 1 FROM trader_sync_alert_memberships fm WHERE fm.owner_id=a.owner_id AND fm.activity_id=a.id AND fm.batch_id=sqlc.narg(batch_id)))
AND (sqlc.narg(from_time)::timestamptz IS NULL OR a.settled_at>=sqlc.narg(from_time))
AND (sqlc.narg(to_time)::timestamptz IS NULL OR a.settled_at<sqlc.narg(to_time))
AND a.id<=sqlc.arg(snapshot_id)::bigint
AND (sqlc.narg(after_id)::bigint IS NULL OR a.id<sqlc.narg(after_id))
AND (sqlc.narg(lower_id)::bigint IS NULL OR a.id>=sqlc.narg(lower_id))
AND (sqlc.narg(upper_id)::bigint IS NULL OR a.id<=sqlc.narg(upper_id))
AND (sqlc.narg(activity_id)::bigint IS NULL OR a.id=sqlc.narg(activity_id))
AND NOT sqlc.arg(empty_page)::boolean
ORDER BY a.id DESC LIMIT sqlc.arg(row_limit)::integer;

-- name: MemberSummaryBatchExists :one
SELECT EXISTS(SELECT 1 FROM trader_sync_summary_batches WHERE owner_id=$1 AND id=$2 AND sealed);

-- name: MemberSubscriptionExists :one
SELECT EXISTS(SELECT 1 FROM trader_sync_subscriptions WHERE owner_id=$1 AND id=$2);

-- name: MemberBatchActivityExists :one
SELECT EXISTS(SELECT 1 FROM trader_sync_alert_memberships WHERE owner_id=$1 AND batch_id=$2 AND activity_id=$3);

-- name: RequireTraderSyncAdministrator :one
SELECT EXISTS(SELECT 1 FROM athena_account WHERE account_id=$1 AND administrator);

-- name: ReadSubscriptionFacts :many
SELECT s.*,obs.observation_json,COALESCE(n.note,'')::text AS note,COALESCE(n.revision,0)::bigint AS note_revision,
 COALESCE(b.status,'unbound')::text AS binding_status,
 (SELECT jsonb_build_object('ID',i.id,'EffectiveAt',i.effective_at,'EndedAt',COALESCE(i.ended_at,(SELECT ep.ended_at FROM trader_sync_collector_epochs ep WHERE ep.id=i.collector_epoch)),'Generation',i.activation_generation,'Epoch',i.collector_epoch)
 FROM trader_sync_monitor_intervals i WHERE i.owner_id=s.owner_id AND i.subscription_id=s.id AND i.activation_generation=s.activation_generation ORDER BY i.collector_epoch DESC,i.effective_at DESC,i.id DESC LIMIT 1)::jsonb AS interval_json,
 (SELECT jsonb_build_object('Total',count(*),'Pending',count(*) FILTER(WHERE counted.status='pending'),'Sending',count(*) FILTER(WHERE counted.status='sending'),'Sent',count(*) FILTER(WHERE counted.status='sent'),'Failed',count(*) FILTER(WHERE counted.status='failed'),'Unknown',count(*) FILTER(WHERE counted.status='unknown'),'Cancelled',count(*) FILTER(WHERE counted.status='cancelled')) FROM (SELECT DISTINCT d.id,d.status FROM account_notification_deliveries d
WHERE d.account_id=s.owner_id AND (
 EXISTS(SELECT 1 FROM trader_sync_activities a WHERE a.owner_id=s.owner_id AND a.subscription_id=s.id AND a.id=d.activity_id)
 OR EXISTS(SELECT 1 FROM trader_sync_summary_parts p JOIN trader_sync_summary_part_items pi ON pi.part_id=p.id AND pi.owner_id=p.owner_id JOIN trader_sync_activities a ON a.owner_id=pi.owner_id AND a.id=pi.activity_id WHERE p.owner_id=s.owner_id AND p.delivery_id=d.id AND a.subscription_id=s.id))) counted)::jsonb AS queue_counts
FROM trader_sync_subscriptions s
LEFT JOIN trader_sync_target_notes n ON n.owner_id=s.owner_id AND n.wallet=s.wallet
LEFT JOIN telegram_bindings b ON b.account_id=s.owner_id
LEFT JOIN LATERAL (SELECT jsonb_build_object('State',CASE WHEN s.desired_state='enabled' AND EXISTS(SELECT 1 FROM trader_sync_baseline_attempts ba JOIN trader_sync_collector_epochs ep ON ep.id=ba.collector_epoch
WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND ba.activation_generation=s.activation_generation AND ep.ended_at IS NOT NULL
AND (ba.state='pending' OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id AND i.ended_at IS NULL))) THEN 'interrupted' ELSE s.observation_state END,'Reason',CASE WHEN s.desired_state='enabled' AND EXISTS(SELECT 1 FROM trader_sync_baseline_attempts ba JOIN trader_sync_collector_epochs ep ON ep.id=ba.collector_epoch
WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND ba.activation_generation=s.activation_generation AND ep.ended_at IS NOT NULL
AND (ba.state='pending' OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id AND i.ended_at IS NULL))) THEN 'collector_interrupted'
 WHEN s.observation_state='pending_baseline' THEN COALESCE((SELECT ba.reason FROM trader_sync_baseline_attempts ba WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND ba.activation_generation=s.activation_generation AND ba.state='failed' ORDER BY ba.created_at DESC,ba.id DESC LIMIT 1),s.reason) ELSE s.reason END,
'LastReliableAt',(SELECT i.last_reliable_at FROM trader_sync_monitor_intervals i WHERE i.owner_id=s.owner_id AND i.subscription_id=s.id AND i.activation_generation=s.activation_generation AND i.last_reliable_at IS NOT NULL ORDER BY i.collector_epoch DESC,i.effective_at DESC,i.id DESC LIMIT 1),
'InterruptionCount',(SELECT count(DISTINCT rr.epoch_id) FROM (SELECT DISTINCT ep.id AS epoch_id,ir.id,ir.recorded_at,ep.ended_at,ep.reason,ba.activation_generation
FROM trader_sync_baseline_attempts ba
JOIN trader_sync_collector_epochs ep ON ep.id=ba.collector_epoch AND ep.ended_at IS NOT NULL
JOIN trader_sync_interruptions ir ON ir.collector_epoch=ep.id
WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND (
 ba.state='pending' OR (ba.state='failed' AND ba.ended_at=ep.ended_at AND ba.reason=ep.reason)
 OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id AND
 (i.ended_at IS NULL OR (i.ended_at=ep.ended_at AND i.reason=ep.reason))))) rr),
'LatestInterruption',(SELECT jsonb_build_object('ID',rel.id::text,'Start',NULL,'End',recovered.effective_at,'RecoveredAt',recovered.effective_at,
'Reason',rel.reason,'Uncertainty','actual_start_unknown;recorded_stop_boundary_available' ||
 CASE WHEN recovered.effective_at<rel.ended_at THEN ';clock_order_uncertain' ELSE '' END,'PossibleMissing',true) FROM (SELECT DISTINCT ep.id AS epoch_id,ir.id,ir.recorded_at,ep.ended_at,ep.reason,ba.activation_generation
FROM trader_sync_baseline_attempts ba
JOIN trader_sync_collector_epochs ep ON ep.id=ba.collector_epoch AND ep.ended_at IS NOT NULL
JOIN trader_sync_interruptions ir ON ir.collector_epoch=ep.id
WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND (
 ba.state='pending' OR (ba.state='failed' AND ba.ended_at=ep.ended_at AND ba.reason=ep.reason)
 OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id AND
 (i.ended_at IS NULL OR (i.ended_at=ep.ended_at AND i.reason=ep.reason))))) rel LEFT JOIN LATERAL(SELECT i.effective_at FROM trader_sync_monitor_intervals i
JOIN trader_sync_baseline_attempts ba ON ba.id=i.baseline_attempt_id AND ba.state='succeeded'
WHERE i.owner_id=s.owner_id AND i.subscription_id=s.id AND i.activation_generation=rel.activation_generation
AND i.collector_epoch>rel.epoch_id ORDER BY i.collector_epoch,ba.created_at,ba.id LIMIT 1) recovered ON true ORDER BY rel.recorded_at DESC,rel.id DESC LIMIT 1))::jsonb AS observation_json) obs ON true
WHERE s.owner_id=sqlc.arg(owner_id)::uuid
AND (sqlc.narg(subscription_id)::uuid IS NOT NULL OR (s.desired_state='cancelled')=sqlc.arg(cancelled_view)::boolean)
AND (sqlc.narg(after_time)::timestamptz IS NULL OR (s.created_at,s.id)<(sqlc.narg(after_time),sqlc.narg(after_uuid)::uuid))
AND (sqlc.narg(subscription_id)::uuid IS NULL OR s.id=sqlc.narg(subscription_id))
AND (sqlc.arg(state_filter)::text='' OR CASE WHEN s.desired_state='enabled' THEN (CASE WHEN s.desired_state='enabled' AND EXISTS(SELECT 1 FROM trader_sync_baseline_attempts ba JOIN trader_sync_collector_epochs ep ON ep.id=ba.collector_epoch
WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND ba.activation_generation=s.activation_generation AND ep.ended_at IS NOT NULL
AND (ba.state='pending' OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id AND i.ended_at IS NULL))) THEN 'interrupted' ELSE s.observation_state END) ELSE s.desired_state END=sqlc.arg(state_filter))
ORDER BY s.created_at DESC,s.id DESC LIMIT sqlc.arg(row_limit)::integer;

-- name: ReadAdminSubscriptionFacts :many
SELECT s.id,s.owner_id,ac.username,COALESCE(ac.verified_email,'')::text AS email,s.wallet,
 CASE WHEN s.desired_state='enabled' THEN (CASE WHEN s.desired_state='enabled' AND EXISTS(SELECT 1 FROM trader_sync_baseline_attempts ba JOIN trader_sync_collector_epochs ep ON ep.id=ba.collector_epoch
WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND ba.activation_generation=s.activation_generation AND ep.ended_at IS NOT NULL
AND (ba.state='pending' OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id AND i.ended_at IS NULL))) THEN 'interrupted' ELSE s.observation_state END) ELSE s.desired_state END::text AS status,s.desired_state,s.observation_state,s.reason,s.created_at,s.updated_at,s.ended_at,obs.observation_json,
 (SELECT count(*) FROM trader_sync_activities a WHERE a.owner_id=s.owner_id AND a.subscription_id=s.id)::bigint AS activity_count,
 (SELECT jsonb_build_object('Total',count(*),'Pending',count(*) FILTER(WHERE counted.status='pending'),'Sending',count(*) FILTER(WHERE counted.status='sending'),'Sent',count(*) FILTER(WHERE counted.status='sent'),'Failed',count(*) FILTER(WHERE counted.status='failed'),'Unknown',count(*) FILTER(WHERE counted.status='unknown'),'Cancelled',count(*) FILTER(WHERE counted.status='cancelled')) FROM (SELECT DISTINCT d.id,d.status FROM account_notification_deliveries d
WHERE d.account_id=s.owner_id AND (
 EXISTS(SELECT 1 FROM trader_sync_activities a WHERE a.owner_id=s.owner_id AND a.subscription_id=s.id AND a.id=d.activity_id)
 OR EXISTS(SELECT 1 FROM trader_sync_summary_parts p JOIN trader_sync_summary_part_items pi ON pi.part_id=p.id AND pi.owner_id=p.owner_id JOIN trader_sync_activities a ON a.owner_id=pi.owner_id AND a.id=pi.activity_id WHERE p.owner_id=s.owner_id AND p.delivery_id=d.id AND a.subscription_id=s.id))) counted)::jsonb AS delivery_counts
FROM trader_sync_subscriptions s JOIN athena_account ac ON ac.account_id=s.owner_id
LEFT JOIN LATERAL (SELECT jsonb_build_object('State',CASE WHEN s.desired_state='enabled' AND EXISTS(SELECT 1 FROM trader_sync_baseline_attempts ba JOIN trader_sync_collector_epochs ep ON ep.id=ba.collector_epoch
WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND ba.activation_generation=s.activation_generation AND ep.ended_at IS NOT NULL
AND (ba.state='pending' OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id AND i.ended_at IS NULL))) THEN 'interrupted' ELSE s.observation_state END,'Reason',CASE WHEN s.desired_state='enabled' AND EXISTS(SELECT 1 FROM trader_sync_baseline_attempts ba JOIN trader_sync_collector_epochs ep ON ep.id=ba.collector_epoch
WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND ba.activation_generation=s.activation_generation AND ep.ended_at IS NOT NULL
AND (ba.state='pending' OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id AND i.ended_at IS NULL))) THEN 'collector_interrupted'
 WHEN s.observation_state='pending_baseline' THEN COALESCE((SELECT ba.reason FROM trader_sync_baseline_attempts ba WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND ba.activation_generation=s.activation_generation AND ba.state='failed' ORDER BY ba.created_at DESC,ba.id DESC LIMIT 1),s.reason) ELSE s.reason END,
'LastReliableAt',(SELECT i.last_reliable_at FROM trader_sync_monitor_intervals i WHERE i.owner_id=s.owner_id AND i.subscription_id=s.id AND i.activation_generation=s.activation_generation AND i.last_reliable_at IS NOT NULL ORDER BY i.collector_epoch DESC,i.effective_at DESC,i.id DESC LIMIT 1),
'InterruptionCount',(SELECT count(DISTINCT rr.epoch_id) FROM (SELECT DISTINCT ep.id AS epoch_id,ir.id,ir.recorded_at,ep.ended_at,ep.reason,ba.activation_generation
FROM trader_sync_baseline_attempts ba
JOIN trader_sync_collector_epochs ep ON ep.id=ba.collector_epoch AND ep.ended_at IS NOT NULL
JOIN trader_sync_interruptions ir ON ir.collector_epoch=ep.id
WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND (
 ba.state='pending' OR (ba.state='failed' AND ba.ended_at=ep.ended_at AND ba.reason=ep.reason)
 OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id AND
 (i.ended_at IS NULL OR (i.ended_at=ep.ended_at AND i.reason=ep.reason))))) rr),
'LatestInterruption',(SELECT jsonb_build_object('ID',rel.id::text,'Start',NULL,'End',recovered.effective_at,'RecoveredAt',recovered.effective_at,
'Reason',rel.reason,'Uncertainty','actual_start_unknown;recorded_stop_boundary_available' ||
 CASE WHEN recovered.effective_at<rel.ended_at THEN ';clock_order_uncertain' ELSE '' END,'PossibleMissing',true) FROM (SELECT DISTINCT ep.id AS epoch_id,ir.id,ir.recorded_at,ep.ended_at,ep.reason,ba.activation_generation
FROM trader_sync_baseline_attempts ba
JOIN trader_sync_collector_epochs ep ON ep.id=ba.collector_epoch AND ep.ended_at IS NOT NULL
JOIN trader_sync_interruptions ir ON ir.collector_epoch=ep.id
WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND (
 ba.state='pending' OR (ba.state='failed' AND ba.ended_at=ep.ended_at AND ba.reason=ep.reason)
 OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id AND
 (i.ended_at IS NULL OR (i.ended_at=ep.ended_at AND i.reason=ep.reason))))) rel LEFT JOIN LATERAL(SELECT i.effective_at FROM trader_sync_monitor_intervals i
JOIN trader_sync_baseline_attempts ba ON ba.id=i.baseline_attempt_id AND ba.state='succeeded'
WHERE i.owner_id=s.owner_id AND i.subscription_id=s.id AND i.activation_generation=rel.activation_generation
AND i.collector_epoch>rel.epoch_id ORDER BY i.collector_epoch,ba.created_at,ba.id LIMIT 1) recovered ON true ORDER BY rel.recorded_at DESC,rel.id DESC LIMIT 1))::jsonb AS observation_json) obs ON true
WHERE (sqlc.narg(owner_id)::uuid IS NULL OR s.owner_id=sqlc.narg(owner_id))
AND (sqlc.narg(wallet)::bytea IS NULL OR s.wallet=sqlc.narg(wallet))
AND (sqlc.arg(include_cancelled)::boolean OR s.desired_state<>'cancelled' OR sqlc.narg(subscription_id)::uuid IS NOT NULL)
AND (sqlc.narg(after_time)::timestamptz IS NULL OR (s.created_at,s.id)<(sqlc.narg(after_time),sqlc.narg(after_uuid)::uuid))
AND (sqlc.narg(subscription_id)::uuid IS NULL OR s.id=sqlc.narg(subscription_id))
AND (sqlc.arg(state_filter)::text='' OR CASE WHEN s.desired_state='enabled' THEN (CASE WHEN s.desired_state='enabled' AND EXISTS(SELECT 1 FROM trader_sync_baseline_attempts ba JOIN trader_sync_collector_epochs ep ON ep.id=ba.collector_epoch
WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND ba.activation_generation=s.activation_generation AND ep.ended_at IS NOT NULL
AND (ba.state='pending' OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id AND i.ended_at IS NULL))) THEN 'interrupted' ELSE s.observation_state END) ELSE s.desired_state END=sqlc.arg(state_filter))
ORDER BY s.created_at DESC,s.id DESC LIMIT sqlc.arg(row_limit)::integer;

-- name: ReadSummaryBatchFacts :one
SELECT b.id,b.oldest_at,b.first_started_at,
 min(a.settled_at)::timestamptz AS settled_from,max(a.settled_at)::timestamptz AS settled_to,
 min(a.recorded_at)::timestamptz AS recorded_from,max(a.recorded_at)::timestamptz AS recorded_to,count(*)::bigint AS activity_count
FROM trader_sync_summary_batches b
JOIN trader_sync_alert_memberships m ON m.owner_id=b.owner_id AND m.batch_id=b.id
JOIN trader_sync_activities a ON a.owner_id=m.owner_id AND a.id=m.activity_id
WHERE b.owner_id=$1 AND b.id=$2 AND b.sealed GROUP BY b.id;

-- name: ReadSummaryTargetCounts :many
SELECT s.wallet,count(*)::bigint AS count FROM trader_sync_alert_memberships m
JOIN trader_sync_activities a ON a.owner_id=m.owner_id AND a.id=m.activity_id
JOIN trader_sync_subscriptions s ON s.owner_id=a.owner_id AND s.id=a.subscription_id
WHERE m.owner_id=$1 AND m.batch_id=$2 GROUP BY s.wallet ORDER BY s.wallet;

-- name: ReadSummaryPartCounts :one
SELECT (SELECT jsonb_build_object('Total',count(*),'Pending',count(*) FILTER(WHERE counted.status='pending'),'Sending',count(*) FILTER(WHERE counted.status='sending'),'Sent',count(*) FILTER(WHERE counted.status='sent'),'Failed',count(*) FILTER(WHERE counted.status='failed'),'Unknown',count(*) FILTER(WHERE counted.status='unknown'),'Cancelled',count(*) FILTER(WHERE counted.status='cancelled')) FROM (SELECT d.status FROM trader_sync_summary_parts p JOIN account_notification_deliveries d ON d.account_id=p.owner_id AND d.id=p.delivery_id WHERE p.owner_id=sqlc.arg(owner_id)::uuid AND p.batch_id=sqlc.arg(batch_id)::bigint) counted)::jsonb AS counts;

-- name: ReadSummaryPartFacts :many
SELECT p.id,p.part_index,p.total,
 CASE WHEN d.id IS NULL THEN NULL ELSE jsonb_build_object(
 'ID',d.id,'Status',d.status,'Reason',COALESCE(d.error_message,d.eligibility_revoked_reason,''),
 'AuthorizedAt',latest.authorized_at,'StartedAt',latest.started_at,'ResultAt',latest.result_at,
 'MessageID',COALESCE(latest.message_id,d.provider_message_id),'AttemptCount',COALESCE(latest.attempt_count,0),
 'LatestAttempt',CASE WHEN latest.id IS NULL THEN NULL ELSE jsonb_build_object('Index',latest.attempt_count,
 'AuthorizedAt',latest.authorized_at,'StartedAt',latest.started_at,'ResultAt',latest.result_at,
 'Status',COALESCE(latest.outcome,'sending'),'Reason',COALESCE(latest.outcome_code,'')) END) END::jsonb AS delivery_json,
 (SELECT count(*) FROM trader_sync_summary_part_items pi WHERE pi.owner_id=p.owner_id AND pi.part_id=p.id)::bigint AS activity_count
FROM trader_sync_summary_parts p JOIN account_notification_deliveries d ON d.account_id=p.owner_id AND d.id=p.delivery_id
LEFT JOIN LATERAL (SELECT t.id,t.authorized_at,t.started_at,t.result_at,t.message_id,t.outcome,t.outcome_code,count(*) OVER() AS attempt_count
 FROM notification_delivery_attempts t WHERE t.work_kind='account' AND t.work_id=d.id AND t.owner_id=d.account_id
 ORDER BY t.authorized_at DESC,t.id DESC LIMIT 1) latest ON true
WHERE p.owner_id=sqlc.arg(owner_id)::uuid AND p.batch_id=sqlc.arg(batch_id)::bigint
AND (sqlc.narg(activity_id)::bigint IS NULL OR EXISTS(SELECT 1 FROM trader_sync_summary_part_items pi WHERE pi.owner_id=p.owner_id AND pi.part_id=p.id AND pi.activity_id=sqlc.narg(activity_id)))
AND p.part_index>sqlc.arg(after_index)::integer ORDER BY p.part_index LIMIT sqlc.arg(row_limit)::integer;

-- name: ReadSubscriptionHistoryFacts :many
WITH scope AS(SELECT * FROM trader_sync_subscriptions WHERE owner_id=sqlc.arg(owner_id)::uuid AND id=sqlc.arg(subscription_id)::uuid),
 entries AS(
 SELECT 'interval/'||i.id::text AS id,'interval'::text AS kind,i.effective_at AS sort_at,
 jsonb_build_object('ID',i.id,'EffectiveAt',i.effective_at,'EndedAt',COALESCE(i.ended_at,ep.ended_at),'Generation',i.activation_generation,'Epoch',i.collector_epoch)::jsonb AS interval_json,NULL::jsonb AS interruption_json
 FROM scope s JOIN trader_sync_monitor_intervals i ON i.owner_id=s.owner_id AND i.subscription_id=s.id JOIN trader_sync_collector_epochs ep ON ep.id=i.collector_epoch
 UNION ALL
 SELECT DISTINCT 'interruption/'||rel.id::text,'interruption'::text,rel.recorded_at,NULL::jsonb,jsonb_build_object('ID',rel.id::text,'Start',NULL,'End',recovered.effective_at,'RecoveredAt',recovered.effective_at,
'Reason',rel.reason,'Uncertainty','actual_start_unknown;recorded_stop_boundary_available' ||
 CASE WHEN recovered.effective_at<rel.ended_at THEN ';clock_order_uncertain' ELSE '' END,'PossibleMissing',true)::jsonb
 FROM scope s JOIN LATERAL (SELECT DISTINCT ep.id AS epoch_id,ir.id,ir.recorded_at,ep.ended_at,ep.reason,ba.activation_generation
FROM trader_sync_baseline_attempts ba
JOIN trader_sync_collector_epochs ep ON ep.id=ba.collector_epoch AND ep.ended_at IS NOT NULL
JOIN trader_sync_interruptions ir ON ir.collector_epoch=ep.id
WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND (
 ba.state='pending' OR (ba.state='failed' AND ba.ended_at=ep.ended_at AND ba.reason=ep.reason)
 OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id AND
 (i.ended_at IS NULL OR (i.ended_at=ep.ended_at AND i.reason=ep.reason))))) rel ON true LEFT JOIN LATERAL(SELECT i.effective_at FROM trader_sync_monitor_intervals i
JOIN trader_sync_baseline_attempts ba ON ba.id=i.baseline_attempt_id AND ba.state='succeeded'
WHERE i.owner_id=s.owner_id AND i.subscription_id=s.id AND i.activation_generation=rel.activation_generation
AND i.collector_epoch>rel.epoch_id ORDER BY i.collector_epoch,ba.created_at,ba.id LIMIT 1) recovered ON true
 ) SELECT * FROM entries
 WHERE (sqlc.narg(after_time)::timestamptz IS NULL OR (sort_at,id)<(sqlc.narg(after_time),sqlc.arg(after_id)::text))
 ORDER BY sort_at DESC,id DESC LIMIT sqlc.arg(row_limit)::integer;

-- name: ReadTraderSyncRuntime :one
SELECT c.active_epoch,c.fencing_token,COALESCE(e.filter_revision,0)::bigint AS filter_revision,e.started_at,
 (SELECT count(*) FROM trader_sync_source_records WHERE confirmation_state='unverified')::bigint AS confirmation_backlog,
 (SELECT count(*) FROM trader_sync_source_candidates WHERE disposition='pending')::bigint AS projection_backlog,
 (SELECT count(*) FROM trader_sync_source_records WHERE NOT metadata_complete)::bigint AS metadata_missing,
 (SELECT count(*) FROM trader_sync_finality_anomalies)::bigint AS finality_anomalies,
 (SELECT count(*) FROM trader_sync_summary_batches WHERE start_evidence_missing)::bigint AS clock_or_start_evidence_missing,
 (SELECT jsonb_build_object('Total',count(*),'Pending',count(*) FILTER(WHERE counted.status='pending'),'Sending',count(*) FILTER(WHERE counted.status='sending'),'Sent',count(*) FILTER(WHERE counted.status='sent'),'Failed',count(*) FILTER(WHERE counted.status='failed'),'Unknown',count(*) FILTER(WHERE counted.status='unknown'),'Cancelled',count(*) FILTER(WHERE counted.status='cancelled')) FROM (SELECT status FROM account_notification_deliveries WHERE source='trader_sync') counted)::jsonb AS deliveries
FROM trader_sync_collector_control c LEFT JOIN trader_sync_collector_epochs e ON e.id=c.active_epoch AND e.ended_at IS NULL WHERE c.singleton;
