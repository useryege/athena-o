-- name: InsertSourceRecord :one
INSERT INTO trader_sync_source_records(chain_id,exchange_address,wallet,block_hash,transaction_hash,log_index,block_number,raw_json,collector_epoch,read_sequence,received_at,removed)
VALUES(137,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT(chain_id,exchange_address,block_hash,transaction_hash,log_index) DO NOTHING RETURNING *;
-- name: UpdateSourceRemoved :exec
UPDATE trader_sync_source_records SET removed=removed OR $6,confirmation_state=CASE WHEN $6 THEN 'invalid' ELSE confirmation_state END,confirmation_reason=CASE WHEN $6 THEN 'removed' ELSE confirmation_reason END
WHERE chain_id=137 AND exchange_address=$1 AND block_hash=$2 AND transaction_hash=$3 AND log_index=$4 AND wallet=$5;
-- name: InsertSourceCandidates :exec
INSERT INTO trader_sync_source_candidates(source_record_id,owner_id,subscription_id,activation_generation,baseline_attempt_id,received_at)
SELECT r.id,a.owner_id,a.subscription_id,a.activation_generation,a.id,r.received_at
FROM trader_sync_source_records r JOIN trader_sync_subscriptions s ON s.wallet=r.wallet
JOIN trader_sync_baseline_attempts a ON a.subscription_id=s.id AND a.activation_generation=s.activation_generation
WHERE r.id=$1 AND s.desired_state='enabled' AND a.state IN ('pending','succeeded') AND a.collector_epoch=r.collector_epoch AND a.registered_sequence<r.read_sequence;
