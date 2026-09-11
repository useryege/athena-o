-- name: ReadTimingRollups :many
-- Only finite aggregate names/values leave this query. Private formation evidence
-- and owner identities never become runtime labels or response objects.
WITH links AS MATERIALIZED (
 SELECT activity_id, id AS delivery_id FROM account_notification_deliveries WHERE activity_id IS NOT NULL
 UNION
 SELECT i.activity_id,p.delivery_id FROM trader_sync_summary_part_items i JOIN trader_sync_summary_parts p ON p.id=i.part_id
), deliveries AS MATERIALIZED (
 SELECT d.id,d.status,d.created_at FROM account_notification_deliveries d WHERE EXISTS(SELECT 1 FROM links l WHERE l.delivery_id=d.id)
), attempts AS MATERIALIZED (
 SELECT a.id,a.work_id,a.authorized_at,a.started_at,a.result_at,a.sender_returned_at,a.sender_elapsed_ns,a.outcome FROM notification_delivery_attempts a JOIN deliveries d ON a.work_kind='account' AND a.work_id=d.id
), delivery_times AS (
 SELECT work_id,min(authorized_at) AS authorized_at,min(started_at) AS started_at,
 max(sender_returned_at) FILTER(WHERE outcome='sent') AS ack_at FROM attempts GROUP BY work_id
), activity_times AS MATERIALIZED (
 SELECT a.id,a.notification_mode,a.recorded_at,a.received_at,
 a.formation_evidence->'gate' AS gate_phases,
 a.formation_evidence->'processing'->>'confirmationRoundNs' AS confirmation_round_ns,
 a.formation_evidence->'processing'->>'versionRoundNs' AS version_round_ns,
 a.formation_evidence->'processing'->>'metadataExtraWaitNs' AS metadata_extra_wait_ns,
 CASE WHEN a.formation_evidence->>'cohort' IN('ordinary_default','ordinary_unclassified','ordinary_burst','summary','in_app_only')
 THEN a.formation_evidence->>'cohort' ELSE 'ordinary_unclassified' END AS cohort,
 d.authorized_at,d.started_at,d.ack_at
 FROM trader_sync_activities a LEFT JOIN LATERAL (
 SELECT min(t.authorized_at) AS authorized_at,min(t.started_at) AS started_at,
 CASE WHEN count(*)=count(t.ack_at) AND bool_and(delivery.status='sent') THEN max(t.ack_at) END AS ack_at
 FROM links l JOIN deliveries delivery ON delivery.id=l.delivery_id LEFT JOIN delivery_times t ON t.work_id=l.delivery_id WHERE l.activity_id=a.id
 ) d ON true
), expanded AS (
 SELECT id,cohort AS scope,recorded_at,received_at,authorized_at,started_at,ack_at FROM activity_times
 UNION ALL SELECT id,'all',recorded_at,received_at,authorized_at,started_at,ack_at FROM activity_times
 UNION ALL SELECT id,'ordinary_all',recorded_at,received_at,authorized_at,started_at,ack_at FROM activity_times WHERE notification_mode='ordinary'
 UNION ALL SELECT id,'ordinary_nonburst',recorded_at,received_at,authorized_at,started_at,ack_at FROM activity_times WHERE notification_mode='ordinary' AND cohort<>'ordinary_burst'
), scopes AS (SELECT unnest(ARRAY['all','ordinary_all','ordinary_nonburst','ordinary_default','ordinary_unclassified','ordinary_burst','summary','in_app_only'])::text AS scope),
 stats AS (
 SELECT s.scope,count(a.id) AS total,count(a.id) FILTER(WHERE a.ack_at IS NULL) AS no_ack,
 count(a.id) FILTER(WHERE a.recorded_at<a.received_at OR a.recorded_at>clock_timestamp() OR a.ack_at<a.recorded_at OR a.started_at<a.recorded_at OR a.authorized_at<a.recorded_at) AS clock_anomalies,
 max(extract(epoch FROM clock_timestamp()-a.recorded_at)) FILTER(WHERE a.ack_at IS NULL AND a.recorded_at<=clock_timestamp()) AS no_ack_oldest_age,
 percentile_cont(0.95) WITHIN GROUP(ORDER BY extract(epoch FROM a.recorded_at-a.received_at)) FILTER(WHERE a.recorded_at>=a.received_at) AS received_recorded_p95,
 percentile_cont(0.99) WITHIN GROUP(ORDER BY extract(epoch FROM a.recorded_at-a.received_at)) FILTER(WHERE a.recorded_at>=a.received_at) AS received_recorded_p99,
 percentile_cont(0.95) WITHIN GROUP(ORDER BY extract(epoch FROM a.authorized_at-a.recorded_at)) FILTER(WHERE a.authorized_at>=a.recorded_at) AS authorized_p95,
 percentile_cont(0.99) WITHIN GROUP(ORDER BY extract(epoch FROM a.authorized_at-a.recorded_at)) FILTER(WHERE a.authorized_at>=a.recorded_at) AS authorized_p99,
 percentile_cont(0.95) WITHIN GROUP(ORDER BY extract(epoch FROM a.started_at-a.recorded_at)) FILTER(WHERE a.started_at>=a.recorded_at) AS started_p95,
 percentile_cont(0.99) WITHIN GROUP(ORDER BY extract(epoch FROM a.started_at-a.recorded_at)) FILTER(WHERE a.started_at>=a.recorded_at) AS started_p99,
 percentile_cont(0.95) WITHIN GROUP(ORDER BY extract(epoch FROM a.ack_at-a.recorded_at)) FILTER(WHERE a.ack_at>=a.recorded_at) AS ack_p95,
 percentile_cont(0.99) WITHIN GROUP(ORDER BY extract(epoch FROM a.ack_at-a.recorded_at)) FILTER(WHERE a.ack_at>=a.recorded_at) AS ack_p99
 FROM scopes s LEFT JOIN expanded a ON a.scope=s.scope GROUP BY s.scope
), summary_times AS (
 SELECT b.oldest_at,b.first_started_at,lag(first_started_at)OVER(PARTITION BY owner_id ORDER BY id) AS previous_started_at,row_number()OVER(PARTITION BY owner_id ORDER BY id) AS batch_ordinal FROM trader_sync_summary_batches b
), phases AS (
 SELECT 'gate_'||(p->>'phase') AS phase,(p->>'elapsedNs')::double precision/1000000000 AS seconds
 FROM activity_times a CROSS JOIN LATERAL jsonb_array_elements(a.gate_phases) p WHERE p->>'phase' IN('begin','advisory')
 UNION ALL SELECT 'confirmation_round',confirmation_round_ns::double precision/1000000000 FROM activity_times
 UNION ALL SELECT 'version_round',version_round_ns::double precision/1000000000 FROM activity_times
 UNION ALL SELECT 'metadata_extra_wait',metadata_extra_wait_ns::double precision/1000000000 FROM activity_times
), metrics AS (
 SELECT 'timing_activity_'||scope||'_total' AS name,total::text AS value,'activities' AS unit FROM stats
 UNION ALL SELECT 'timing_activity_'||scope||'_no_ack',no_ack::text,'activities' FROM stats
 UNION ALL SELECT 'timing_activity_'||scope||'_clock_anomalies',clock_anomalies::text,'activities' FROM stats
 UNION ALL SELECT 'timing_activity_'||scope||'_no_ack_oldest_utc_seconds',no_ack_oldest_age::text,'seconds' FROM stats
 UNION ALL SELECT 'timing_activity_'||scope||'_received_recorded_p95_utc_seconds',received_recorded_p95::text,'seconds' FROM stats
 UNION ALL SELECT 'timing_activity_'||scope||'_received_recorded_p99_utc_seconds',received_recorded_p99::text,'seconds' FROM stats
 UNION ALL SELECT 'timing_activity_'||scope||'_authorized_p95_utc_seconds',authorized_p95::text,'seconds' FROM stats
 UNION ALL SELECT 'timing_activity_'||scope||'_authorized_p99_utc_seconds',authorized_p99::text,'seconds' FROM stats
 UNION ALL SELECT 'timing_activity_'||scope||'_started_p95_utc_seconds',started_p95::text,'seconds' FROM stats
 UNION ALL SELECT 'timing_activity_'||scope||'_started_p99_utc_seconds',started_p99::text,'seconds' FROM stats
 UNION ALL SELECT 'timing_activity_'||scope||'_ack_p95_utc_seconds',ack_p95::text,'seconds' FROM stats
 UNION ALL SELECT 'timing_activity_'||scope||'_ack_p99_utc_seconds',ack_p99::text,'seconds' FROM stats
 UNION ALL SELECT 'timing_public_time_unassessable',count(*)::text,'activities' FROM activity_times
 UNION ALL SELECT 'timing_logical_'||s.state,count(d.id)::text,'deliveries' FROM (SELECT unnest(ARRAY['pending','sending','sent','failed','unknown','cancelled'])::text AS state)s LEFT JOIN deliveries d ON d.status=s.state GROUP BY s.state

 UNION ALL SELECT 'timing_logical_'||status||'_oldest_age_utc_seconds',max(extract(epoch FROM clock_timestamp()-created_at))::text,'seconds' FROM deliveries WHERE status IN('pending','sending','failed','unknown','cancelled') AND created_at<=clock_timestamp() GROUP BY status
 UNION ALL SELECT 'timing_'||phase||'_p95_monotonic_seconds',(percentile_cont(0.95)WITHIN GROUP(ORDER BY seconds))::text,'seconds' FROM phases WHERE seconds>=0 GROUP BY phase
 UNION ALL SELECT 'timing_'||phase||'_p99_monotonic_seconds',(percentile_cont(0.99)WITHIN GROUP(ORDER BY seconds))::text,'seconds' FROM phases WHERE seconds>=0 GROUP BY phase
 UNION ALL SELECT 'timing_summary_waiting_members',count(*)::text,'activities' FROM trader_sync_alert_memberships WHERE form='summary' AND state='waiting' AND batch_id IS NULL
 UNION ALL SELECT 'timing_attempts_total',count(*)::text,'attempts' FROM attempts
 UNION ALL SELECT 'timing_http_started_total',count(*)::text,'attempts' FROM attempts WHERE started_at IS NOT NULL
 UNION ALL SELECT 'timing_attempts_start_missing',count(*)::text,'attempts' FROM attempts WHERE started_at IS NULL
 UNION ALL SELECT 'timing_attempts_result_unconfirmed',count(*)::text,'attempts' FROM attempts WHERE result_at IS NULL
 UNION ALL SELECT 'timing_attempts_'||s.state,count(a.id)::text,'attempts' FROM (SELECT unnest(ARRAY['sent','failed','unknown','retryable'])::text AS state)s LEFT JOIN attempts a ON a.outcome=s.state GROUP BY s.state
 UNION ALL SELECT 'timing_sender_p95_monotonic_seconds',(percentile_cont(0.95)WITHIN GROUP(ORDER BY sender_elapsed_ns::double precision/1000000000))::text,'seconds' FROM attempts WHERE sender_elapsed_ns IS NOT NULL
 UNION ALL SELECT 'timing_sender_p99_monotonic_seconds',(percentile_cont(0.99)WITHIN GROUP(ORDER BY sender_elapsed_ns::double precision/1000000000))::text,'seconds' FROM attempts WHERE sender_elapsed_ns IS NOT NULL
 UNION ALL SELECT 'timing_worker_result_wait_usable',count(*)::text,'attempts' FROM attempts WHERE result_at>=sender_returned_at
 UNION ALL SELECT 'timing_worker_result_wait_clock_anomalies',count(*)::text,'attempts' FROM attempts WHERE result_at<sender_returned_at
 UNION ALL SELECT 'timing_worker_result_wait_missing',count(*)::text,'attempts' FROM attempts WHERE result_at IS NULL OR sender_returned_at IS NULL
 UNION ALL SELECT 'timing_summary_oldest_start_usable',count(*)::text,'summary_batches' FROM summary_times WHERE first_started_at>=oldest_at
 UNION ALL SELECT 'timing_summary_oldest_start_clock_anomalies',count(*)::text,'summary_batches' FROM summary_times WHERE first_started_at<oldest_at
 UNION ALL SELECT 'timing_summary_oldest_start_missing',count(*)::text,'summary_batches' FROM summary_times WHERE first_started_at IS NULL
 UNION ALL SELECT 'timing_summary_adjacent_start_usable',count(*)::text,'summary_batches' FROM summary_times WHERE batch_ordinal>1 AND first_started_at>=previous_started_at
 UNION ALL SELECT 'timing_summary_adjacent_start_clock_anomalies',count(*)::text,'summary_batches' FROM summary_times WHERE batch_ordinal>1 AND first_started_at<previous_started_at
 UNION ALL SELECT 'timing_summary_adjacent_start_missing',count(*)::text,'summary_batches' FROM summary_times WHERE batch_ordinal>1 AND (first_started_at IS NULL OR previous_started_at IS NULL)
 UNION ALL SELECT 'timing_summary_adjacent_start_no_predecessor',count(*)::text,'summary_batches' FROM summary_times WHERE batch_ordinal=1
 UNION ALL SELECT 'timing_summary_batches_total',count(*)::text,'summary_batches' FROM summary_times
 UNION ALL SELECT 'timing_worker_result_wait_p95_utc_seconds',(percentile_cont(0.95)WITHIN GROUP(ORDER BY extract(epoch FROM result_at-sender_returned_at)))::text,'seconds' FROM attempts WHERE result_at>=sender_returned_at
 UNION ALL SELECT 'timing_summary_oldest_start_p95_utc_seconds',(percentile_cont(0.95)WITHIN GROUP(ORDER BY extract(epoch FROM first_started_at-oldest_at)))::text,'seconds' FROM summary_times WHERE first_started_at>=oldest_at
 UNION ALL SELECT 'timing_summary_adjacent_start_min_utc_seconds',min(extract(epoch FROM first_started_at-previous_started_at))::text,'seconds' FROM summary_times WHERE batch_ordinal>1 AND first_started_at>=previous_started_at
 UNION ALL SELECT 'timing_summary_deadline_misses',count(*)::text,'summary_batches' FROM summary_times WHERE first_started_at>oldest_at+interval '60 seconds'
 UNION ALL SELECT 'timing_summary_start_missing',count(*)::text,'summary_batches' FROM summary_times WHERE first_started_at IS NULL
)
SELECT name::text,value::text,unit::text FROM metrics WHERE value IS NOT NULL ORDER BY name;
