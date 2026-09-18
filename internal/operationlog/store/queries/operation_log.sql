-- name: InsertEvent :execrows
INSERT INTO operation_log.event(event_id,operation_id,phase,producer_id,schema_version,occurred_at,payload,payload_hash)
VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING;
-- name: InsertDelivery :exec
INSERT INTO operation_log.delivery(event_id) VALUES($1);
-- name: FindConflictingEvents :many
SELECT event_id,operation_id,phase,payload_hash FROM operation_log.event WHERE event_id=$1 OR (operation_id=$2 AND phase=$3);
-- name: LockPublication :one
SELECT last_seq FROM operation_log.publication WHERE singleton_id=1 FOR UPDATE SKIP LOCKED;
-- name: PendingBatch :many
SELECT e.* FROM operation_log.event e JOIN operation_log.delivery d USING(event_id)
WHERE d.state='PENDING' AND d.next_attempt_at<=clock_timestamp()
ORDER BY e.received_at,e.ingest_id LIMIT 100 FOR UPDATE OF d SKIP LOCKED;
-- name: ProcessedEvents :many
SELECT e.* FROM operation_log.event e JOIN operation_log.delivery d USING(event_id)
WHERE e.operation_id=$1 AND d.state='PROCESSED' ORDER BY e.received_at,e.ingest_id;
-- name: MarkProcessed :exec
UPDATE operation_log.delivery SET state='PROCESSED',processed_at=clock_timestamp(),reason_code=NULL WHERE event_id=$1;
-- name: Quarantine :exec
UPDATE operation_log.delivery SET state='QUARANTINED',failure_count=failure_count+1,processed_at=clock_timestamp(),reason_code=$2 WHERE event_id=$1;
-- name: CloseVersion :exec
UPDATE operation_log.entry_version SET visible_to_seq=$2 WHERE operation_id=$1 AND visible_to_seq IS NULL AND visible_from_seq<$2;
-- name: InsertVersion :exec
INSERT INTO operation_log.entry_version(operation_id,visible_from_seq,started_at,finished_at,actor_account_id,actor_username,actor_role,realm,credential_kind,module_code,action_code,outcome,observation,target_account_id,primary_resource_type,primary_resource_id,request_id,parent_operation_id,business_request_id,duration_ms,detail,source_event_ids)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
;
-- name: Publish :exec
UPDATE operation_log.publication SET last_seq=$1,last_published_at=clock_timestamp() WHERE singleton_id=1;
-- name: PublishStatus :exec
INSERT INTO operation_log.producer_status(producer_id,started_at,last_seen_at,stopped_at,snapshot_no,attempted_events,confirmed_events,unconfirmed_events,invalid_events,capacity_rejected_events,in_flight_events,last_failure_at,last_failure_code,last_recovered_at,last_confirmed_at,persistence_reachable)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
ON CONFLICT(producer_id) DO UPDATE SET last_seen_at=excluded.last_seen_at,stopped_at=excluded.stopped_at,snapshot_no=excluded.snapshot_no,attempted_events=excluded.attempted_events,confirmed_events=excluded.confirmed_events,unconfirmed_events=excluded.unconfirmed_events,invalid_events=excluded.invalid_events,capacity_rejected_events=excluded.capacity_rejected_events,in_flight_events=excluded.in_flight_events,last_failure_at=excluded.last_failure_at,last_failure_code=excluded.last_failure_code,last_recovered_at=excluded.last_recovered_at,last_confirmed_at=excluded.last_confirmed_at,persistence_reachable=excluded.persistence_reachable
WHERE operation_log.producer_status.snapshot_no<excluded.snapshot_no;
