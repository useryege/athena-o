-- name: EnsureSummaryHeads :exec
INSERT INTO trader_sync_summary_heads(owner_id) SELECT DISTINCT owner_id FROM trader_sync_alert_memberships WHERE form='summary' AND state='waiting' AND eligibility_revoked_at IS NULL ON CONFLICT(owner_id)DO NOTHING;

-- name: EnsureSummaryHead :exec
INSERT INTO trader_sync_summary_heads(owner_id)VALUES($1)ON CONFLICT(owner_id)DO NOTHING;

-- name: LockSummaryHead :one
SELECT * FROM trader_sync_summary_heads WHERE owner_id=$1 FOR UPDATE;

-- name: SetSummaryHeadBatch :execrows
UPDATE trader_sync_summary_heads SET current_batch_id=$2,current_attempt_id=NULL WHERE owner_id=$1 AND current_batch_id IS NULL;

-- name: SetSummaryHeadPermit :execrows
UPDATE trader_sync_summary_heads h SET current_attempt_id=$3 WHERE h.owner_id=$1 AND h.current_batch_id=$2
AND EXISTS(SELECT 1 FROM trader_sync_summary_parts p JOIN notification_delivery_attempts a ON a.work_kind='account' AND a.work_id=p.delivery_id
 WHERE p.owner_id=h.owner_id AND p.batch_id=h.current_batch_id AND a.id=$3 AND a.owner_id=h.owner_id AND a.payload_digest=p.payload_digest);

-- name: ListSummaryHeads :many
SELECT h.*, w.chat_id,w.binding_revision,w.oldest_at,w.delivery_id,w.not_before
FROM trader_sync_summary_heads h
CROSS JOIN LATERAL (
 SELECT m.chat_id,m.binding_revision,min(m.created_at)::timestamptz AS oldest_at,0::bigint AS delivery_id,min(m.created_at)::timestamptz AS not_before
 FROM trader_sync_alert_memberships m JOIN telegram_bindings b ON b.account_id=m.owner_id AND b.revision=m.binding_revision AND b.telegram_chat_id=m.chat_id AND b.telegram_user_id=m.chat_id AND b.status='connected'
 WHERE h.current_batch_id IS NULL AND m.owner_id=h.owner_id AND m.form='summary' AND m.state='waiting' AND m.batch_id IS NULL AND m.eligibility_revoked_at IS NULL
 GROUP BY m.chat_id,m.binding_revision
 UNION ALL
 SELECT b.chat_id,b.binding_revision,b.oldest_at,pending.delivery_id,pending.next_attempt_at
 FROM trader_sync_summary_batches b CROSS JOIN LATERAL (
 SELECT p.delivery_id,d.next_attempt_at FROM trader_sync_summary_parts p JOIN account_notification_deliveries d ON d.id=p.delivery_id
 WHERE p.batch_id=b.id AND NOT EXISTS(SELECT 1 FROM trader_sync_summary_parts busy JOIN account_notification_deliveries bd ON bd.id=busy.delivery_id WHERE busy.batch_id=b.id AND bd.status='sending') AND d.status='pending' AND d.eligibility_revoked_at IS NULL AND d.attempts<5 ORDER BY p.part_index LIMIT 1
 ) pending WHERE b.id=h.current_batch_id
) w
WHERE EXISTS(SELECT 1 FROM account_module_access m JOIN athena_account a USING(account_id) WHERE m.account_id=h.owner_id AND m.module='trader_sync' AND m.access_level='read_write' AND NOT a.administrator)
ORDER BY h.owner_id;

-- name: RecordSummaryStarted :execrows
UPDATE trader_sync_summary_batches SET first_started_at=COALESCE(first_started_at,$2) WHERE id=$1;

-- name: ResolveStartedSummaryHead :execrows
UPDATE trader_sync_summary_heads SET current_batch_id=NULL,current_attempt_id=NULL,previous_basis_at=GREATEST(previous_basis_at,$3)
WHERE owner_id=$1 AND current_batch_id=$2;

-- name: ReadSummaryBatch :one
SELECT * FROM trader_sync_summary_batches WHERE owner_id=$1 AND id=$2;

-- name: ResolveNeverStartedSummaryHeads :exec
UPDATE trader_sync_summary_heads h SET current_batch_id=NULL,current_attempt_id=NULL
WHERE h.owner_id=$1 AND h.current_batch_id IS NOT NULL
AND NOT EXISTS(SELECT 1 FROM trader_sync_summary_parts p JOIN account_notification_deliveries d ON d.id=p.delivery_id WHERE p.batch_id=h.current_batch_id AND d.status IN('pending','sending'))
AND NOT EXISTS(SELECT 1 FROM trader_sync_summary_parts p JOIN notification_delivery_attempts a ON a.work_kind='account' AND a.work_id=p.delivery_id WHERE p.batch_id=h.current_batch_id AND (a.started_at IS NOT NULL OR a.result_at IS NULL OR a.outcome_code IS DISTINCT FROM 'not_started'));

-- name: RecordSummaryBudgetWait :exec
UPDATE trader_sync_summary_batches SET budget_wait_started_at=COALESCE(budget_wait_started_at,sqlc.arg(started_at)::timestamptz),budget_wait_ended_at=sqlc.arg(ended_at),budget_wait_ms=budget_wait_ms+sqlc.arg(wait_ms)::bigint,budget_reason=sqlc.arg(reason)
WHERE id=sqlc.arg(batch_id);

-- name: RecordSummaryGateWait :exec
UPDATE trader_sync_summary_batches SET local_gate_wait_ms=local_gate_wait_ms+$2 WHERE id=$1;

-- name: RecoverableSummaryHeads :many
SELECT h.*,a.sender_incarnation FROM trader_sync_summary_heads h JOIN notification_delivery_attempts a ON a.id=h.current_attempt_id WHERE a.sender_incarnation=$1;

-- name: SummaryBatchStartEvidence :one
SELECT min(a.started_at)::timestamptz FROM trader_sync_summary_parts p JOIN notification_delivery_attempts a ON a.work_kind='account' AND a.work_id=p.delivery_id WHERE p.batch_id=$1;

-- name: RecoverSummaryBatchBasis :exec
UPDATE trader_sync_summary_batches SET recovery_basis_at=$2,recovery_reason=$3,start_evidence_missing=(first_started_at IS NULL) WHERE id=$1;

-- name: SummaryHeadOwners :many
SELECT h.owner_id FROM trader_sync_summary_heads h WHERE h.current_batch_id IS NOT NULL
AND NOT EXISTS(SELECT 1 FROM trader_sync_summary_parts p JOIN account_notification_deliveries d ON d.id=p.delivery_id WHERE p.batch_id=h.current_batch_id AND d.status IN('pending','sending'))
AND NOT EXISTS(SELECT 1 FROM trader_sync_summary_parts p JOIN notification_delivery_attempts a ON a.work_kind='account' AND a.work_id=p.delivery_id WHERE p.batch_id=h.current_batch_id AND (a.started_at IS NOT NULL OR a.result_at IS NULL OR a.outcome_code IS DISTINCT FROM 'not_started'));
