-- name: MarkActiveCredentialPendingRevocation :execrows
UPDATE worm_wallet_credentials
SET
  state = 'PENDING_REVOCATION',
  updated_at = sqlc.arg('now')
WHERE wallet_id = sqlc.arg('wallet_id')
  AND state = 'ACTIVE';

-- name: CreateActiveCredential :one
INSERT INTO worm_wallet_credentials (
  wallet_id,
  version,
  state,
  api_key_ciphertext,
  api_secret_ciphertext,
  created_at,
  updated_at
)
SELECT
  sqlc.arg('wallet_id'),
  COALESCE(MAX(version), 0) + 1,
  'ACTIVE',
  sqlc.arg('api_key_ciphertext'),
  sqlc.arg('api_secret_ciphertext'),
  sqlc.arg('now'),
  sqlc.arg('now')
FROM worm_wallet_credentials
WHERE wallet_id = sqlc.arg('wallet_id')
RETURNING *;

-- name: GetActiveCredentialForUpdate :one
SELECT *
FROM worm_wallet_credentials
WHERE wallet_id = sqlc.arg('wallet_id')
  AND state = 'ACTIVE'
FOR UPDATE;

-- name: GetCredentialForUpdate :one
SELECT *
FROM worm_wallet_credentials
WHERE id = sqlc.arg('credential_id')
  AND wallet_id = sqlc.arg('wallet_id')
FOR UPDATE;

-- name: GetNextCredentialForRevocation :one
SELECT *
FROM worm_wallet_credentials
WHERE wallet_id = sqlc.arg('wallet_id')
  AND state IN ('ACTIVE', 'PENDING_REVOCATION', 'REVOKING', 'REVOCATION_REQUIRED')
ORDER BY
  CASE state
    WHEN 'ACTIVE' THEN 0
    WHEN 'REVOKING' THEN 1
    WHEN 'PENDING_REVOCATION' THEN 2
    ELSE 3
  END,
  version DESC,
  id DESC
LIMIT 1
FOR UPDATE;

-- name: ListCredentialsNeedingRevocation :many
SELECT credential.*
FROM worm_wallet_credentials AS credential
JOIN worm_wallet_connections AS connection
  ON connection.wallet_id = credential.wallet_id
WHERE credential.state = 'PENDING_REVOCATION'
   OR (
     credential.state = 'REVOKING'
     AND connection.state = 'CONNECTED'
     AND credential.updated_at <= sqlc.arg('retry_before')
   )
ORDER BY credential.updated_at, credential.wallet_id, credential.version
LIMIT sqlc.arg('limit');

-- name: UpdateCredentialState :one
UPDATE worm_wallet_credentials
SET
  state = sqlc.arg('state'),
  updated_at = sqlc.arg('now')
WHERE id = sqlc.arg('credential_id')
  AND wallet_id = sqlc.arg('wallet_id')
RETURNING *;

-- name: DeleteCredential :execrows
DELETE FROM worm_wallet_credentials
WHERE id = sqlc.arg('credential_id')
  AND wallet_id = sqlc.arg('wallet_id');

-- name: CountCredentialsForWallet :one
SELECT COUNT(*)::bigint
FROM worm_wallet_credentials
WHERE wallet_id = sqlc.arg('wallet_id');

-- name: CountActiveCredentialsForWallet :one
SELECT COUNT(*)::bigint
FROM worm_wallet_credentials
WHERE wallet_id = sqlc.arg('wallet_id')
  AND state = 'ACTIVE';

-- name: GetCredentialCleanupCounts :one
SELECT
  COUNT(*) FILTER (WHERE state <> 'ACTIVE')::bigint AS cleanup_count,
  COUNT(*) FILTER (WHERE state = 'REVOCATION_REQUIRED')::bigint AS revocation_required_count
FROM worm_wallet_credentials
WHERE wallet_id = sqlc.arg('wallet_id');
