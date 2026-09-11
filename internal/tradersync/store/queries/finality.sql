-- name: FinalityObservationCutoff :one
SELECT coalesce(max(id),0)::bigint FROM trader_sync_source_records;

-- name: LockFinalityTiming :one
SELECT finality_timing FROM trader_sync_source_records WHERE id=$1 FOR UPDATE;

-- name: SaveFinalityTiming :exec
UPDATE trader_sync_source_records SET finality_timing=$2 WHERE id=$1;

-- name: ReadFinalityRollups :many
-- Only scalar timing evidence enters this administrator query; no raw source,
-- wallet, owner, transaction, payload or private formation JSON is requested.
WITH facts AS (
 SELECT id,coalesce(finality_timing->>'state','') AS state,
 coalesce(finality_timing->>'reason','') AS reason,
 finality_timing->>'clockId' AS clock_id,
 (finality_timing->>'firstStartedNs')::bigint AS first_ns,
 (finality_timing->>'firstRoundNs')::bigint AS round_ns,
 (finality_timing->>'firstConfirmedNs')::bigint AS confirmed_ns
 FROM trader_sync_source_records
), classified AS (
 SELECT first_ns,round_ns,confirmed_ns,
 CASE
 WHEN state='completed' AND first_ns>=0 AND round_ns>=0 AND confirmed_ns>=first_ns+round_ns THEN 'completed'
 WHEN state='unavailable' THEN 'unavailable'
 WHEN state='waiting' AND clock_id=sqlc.arg(clock_id)::text AND sqlc.arg(clock_valid)::boolean AND first_ns>=0 AND round_ns>=0 AND first_ns<=sqlc.arg(elapsed_ns)::bigint THEN 'waiting'
 WHEN state='' AND id>sqlc.arg(cutoff)::bigint AND sqlc.arg(clock_valid)::boolean THEN 'not_started'
 ELSE 'unavailable' END AS state,
 CASE
 WHEN state='unavailable' THEN CASE WHEN reason IN('observation_gap','clock_changed','clock_invalid','first_observation_missing') THEN reason ELSE 'clock_invalid' END
 WHEN state='' AND id<=sqlc.arg(cutoff)::bigint THEN 'first_observation_missing'
 WHEN state='waiting' AND clock_id IS DISTINCT FROM sqlc.arg(clock_id)::text THEN 'clock_changed'
 WHEN state<>'completed' AND NOT sqlc.arg(clock_valid)::boolean THEN 'observation_gap'
 ELSE 'clock_invalid' END AS unavailable_reason
 FROM facts
), durations AS (
 SELECT 'first_attempt_round'::text AS phase,round_ns AS ns FROM classified WHERE state IN('waiting','completed')
 UNION ALL SELECT 'first_attempt_to_first_confirmed',confirmed_ns-first_ns FROM classified WHERE state='completed'
 UNION ALL SELECT 'after_first_attempt_to_first_confirmed',confirmed_ns-first_ns-round_ns FROM classified WHERE state='completed'
), metrics AS (
 SELECT 'finality_sources_total'::text AS name,count(*)::text AS value,'sources'::text AS unit FROM classified
 UNION ALL SELECT 'finality_sources_'||s.state,count(c.state)::text,'sources' FROM (SELECT unnest(ARRAY['not_started','waiting','completed','unavailable'])::text AS state)s LEFT JOIN classified c ON c.state=s.state GROUP BY s.state
 UNION ALL SELECT 'finality_unavailable_'||r.reason,count(c.state)::text,'sources' FROM (SELECT unnest(ARRAY['observation_gap','clock_changed','clock_invalid','first_observation_missing'])::text AS reason)r LEFT JOIN classified c ON c.state='unavailable' AND c.unavailable_reason=r.reason GROUP BY r.reason
 UNION ALL SELECT 'finality_'||p.phase||'_usable',count(d.ns)::text,'sources' FROM (SELECT unnest(ARRAY['first_attempt_round','first_attempt_to_first_confirmed','after_first_attempt_to_first_confirmed'])::text AS phase)p LEFT JOIN durations d ON d.phase=p.phase GROUP BY p.phase
 UNION ALL SELECT 'finality_'||phase||'_p95_monotonic_seconds',(percentile_cont(0.95)WITHIN GROUP(ORDER BY ns::double precision/1000000000))::text,'seconds' FROM durations GROUP BY phase
 UNION ALL SELECT 'finality_'||phase||'_p99_monotonic_seconds',(percentile_cont(0.99)WITHIN GROUP(ORDER BY ns::double precision/1000000000))::text,'seconds' FROM durations GROUP BY phase
 UNION ALL SELECT 'finality_pending_oldest_monotonic_seconds',max((sqlc.arg(elapsed_ns)::bigint-first_ns)::double precision/1000000000)::text,'seconds' FROM classified WHERE state='waiting'
)
SELECT name::text,value::text,unit::text FROM metrics WHERE value IS NOT NULL ORDER BY name;
