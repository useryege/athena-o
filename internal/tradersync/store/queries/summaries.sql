-- name: WaitingSummaryActivities :many
SELECT a.*, md.metadata_json FROM trader_sync_alert_memberships m
JOIN trader_sync_activities a ON a.id=m.activity_id AND a.owner_id=m.owner_id
JOIN trader_sync_market_metadata md ON md.cache_key=a.metadata_key
JOIN telegram_bindings b ON b.account_id=m.owner_id AND b.revision=m.binding_revision AND b.telegram_chat_id=m.chat_id AND b.telegram_user_id=m.chat_id AND b.status='connected'
WHERE m.owner_id=$1 AND m.binding_revision=$2 AND m.chat_id=$3 AND m.form='summary' AND m.state='waiting' AND m.batch_id IS NULL AND m.eligibility_revoked_at IS NULL
ORDER BY a.id FOR UPDATE OF m;

-- name: CreateSummaryBatch :one
INSERT INTO trader_sync_summary_batches(owner_id,binding_revision,chat_id,oldest_at)VALUES($1,$2,$3,$4)RETURNING *;

-- name: FreezeSummaryMember :execrows
UPDATE trader_sync_alert_memberships SET state='frozen',batch_id=$3 WHERE owner_id=$1 AND activity_id=$2 AND state='waiting' AND batch_id IS NULL AND eligibility_revoked_at IS NULL;

-- name: CreateSummaryPart :one
INSERT INTO trader_sync_summary_parts(owner_id,batch_id,part_index,total,text,payload_digest,delivery_id)VALUES($1,$2,$3,$4,$5,$6,$7)RETURNING *;

-- name: CreateSummaryPartItem :exec
INSERT INTO trader_sync_summary_part_items(owner_id,batch_id,part_id,activity_id)VALUES($1,$2,$3,$4);

-- name: SealSummaryBatch :exec
UPDATE trader_sync_summary_batches SET sealed=true WHERE id=$1;

-- name: SummaryPartResults :many
SELECT d.status,count(*)::bigint AS count FROM trader_sync_summary_parts p JOIN account_notification_deliveries d ON d.id=p.delivery_id WHERE p.owner_id=$1 AND p.batch_id=$2 GROUP BY d.status;
