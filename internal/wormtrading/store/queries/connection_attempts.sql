-- name: CreateConnectionAttempt :one
INSERT INTO worm_wallet_connection_attempts (
  id,
  wallet_id,
  address,
  kind,
  previous_connection_state,
  previous_connected_at,
  nonce,
  challenge_message,
  message_digest,
  state,
  expires_at,
  created_at,
  updated_at
) VALUES (
  sqlc.arg('id')::uuid,
  sqlc.arg('wallet_id'),
  sqlc.arg('address'),
  sqlc.arg('kind'),
  sqlc.arg('previous_connection_state'),
  sqlc.narg('previous_connected_at')::timestamptz,
  sqlc.arg('nonce'),
  sqlc.arg('challenge_message'),
  sqlc.arg('message_digest'),
  'PREPARED',
  sqlc.arg('expires_at'),
  sqlc.arg('now'),
  sqlc.arg('now')
)
RETURNING *;

-- name: GetConnectionAttempt :one
SELECT *
FROM worm_wallet_connection_attempts
WHERE id = sqlc.arg('id')::uuid;

-- name: GetConnectionAttemptForUpdate :one
SELECT *
FROM worm_wallet_connection_attempts
WHERE id = sqlc.arg('id')::uuid
FOR UPDATE;

-- name: GetActiveConnectionAttemptForWallet :one
SELECT *
FROM worm_wallet_connection_attempts
WHERE wallet_id = sqlc.arg('wallet_id')
  AND state IN ('PREPARED', 'COMPLETING')
ORDER BY created_at DESC, id
LIMIT 1
FOR UPDATE;

-- name: ListExpiredConnectionAttemptWalletIDs :many
SELECT wallet_id
FROM worm_wallet_connection_attempts
WHERE state IN ('PREPARED', 'COMPLETING')
  AND expires_at <= sqlc.arg('now')
ORDER BY expires_at, wallet_id, id
LIMIT sqlc.arg('result_limit');

-- name: MarkConnectionAttemptCompleting :one
UPDATE worm_wallet_connection_attempts
SET
  state = 'COMPLETING',
  failure_code = '',
  updated_at = sqlc.arg('now')
WHERE id = sqlc.arg('id')::uuid
  AND state = 'PREPARED'
  AND expires_at > sqlc.arg('now')
RETURNING *;

-- name: MarkConnectionAttemptTerminal :one
UPDATE worm_wallet_connection_attempts
SET
  state = sqlc.arg('state'),
  failure_code = sqlc.arg('failure_code'),
  completed_at = sqlc.arg('now'),
  updated_at = sqlc.arg('now')
WHERE id = sqlc.arg('id')::uuid
  AND state IN ('PREPARED', 'COMPLETING')
RETURNING *;
