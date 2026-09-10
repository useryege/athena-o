-- name: ConfirmationExpiry :one
SELECT (clock_timestamp() + INTERVAL '5 minutes')::timestamptz AS expires_at;

-- name: SaveConfirmation :exec
INSERT INTO trader_sync_target_confirmations(owner_id,identity_json,identity_digest,token_digest,expires_at)
VALUES($1,$2,$3,$4,$5);

-- name: SaveConfirmationCard :execrows
UPDATE trader_sync_target_confirmations SET card_json=$3
WHERE owner_id=$1 AND token_digest=$2 AND consumed_request_id IS NULL AND expires_at>clock_timestamp();

-- name: ReadConfirmation :one
SELECT identity_json FROM trader_sync_target_confirmations
WHERE owner_id=$1 AND token_digest=$2 AND consumed_request_id IS NULL AND expires_at>clock_timestamp();

-- name: ConsumeConfirmation :one
UPDATE trader_sync_target_confirmations SET consumed_request_id=$3
WHERE owner_id=$1 AND token_digest=$2 AND expires_at>clock_timestamp()
AND consumed_request_id IS NULL AND identity_digest=$4
RETURNING identity_json;
