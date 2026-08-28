-- name: CreateWalletConnectionIfMissing :exec
INSERT INTO worm_wallet_connections (
  wallet_id,
  address,
  state,
  created_at,
  updated_at
) VALUES (
  sqlc.arg('wallet_id'),
  sqlc.arg('address'),
  'NOT_CONNECTED',
  sqlc.arg('now'),
  sqlc.arg('now')
)
ON CONFLICT (wallet_id) DO NOTHING;

-- name: GetWalletConnectionForUpdate :one
SELECT *
FROM worm_wallet_connections
WHERE wallet_id = sqlc.arg('wallet_id')
FOR UPDATE;

-- name: UpdateWalletConnectionState :one
UPDATE worm_wallet_connections
SET
  state = sqlc.arg('state'),
  warning_code = sqlc.arg('warning_code'),
  connected_at = sqlc.narg('connected_at')::timestamptz,
  updated_at = sqlc.arg('now')
WHERE wallet_id = sqlc.arg('wallet_id')
  AND address = sqlc.arg('address')
RETURNING *;

-- name: ListWalletConnectionSnapshots :many
WITH input AS (
  SELECT
    sqlc.arg('wallet_ids')::bigint[] AS wallet_ids,
    sqlc.arg('addresses')::text[] AS addresses
), requested AS (
  SELECT
    (input.wallet_ids[ordinality])::bigint AS wallet_id,
    (input.addresses[ordinality])::text AS address,
    ordinality
  FROM input
  CROSS JOIN LATERAL generate_subscripts(input.wallet_ids, 1) AS ordinality
  WHERE cardinality(input.wallet_ids) = cardinality(input.addresses)
)
SELECT
  requested.wallet_id,
  requested.address AS requested_address,
  COALESCE(connection.address, '')::text AS stored_address,
  COALESCE(connection.state, 'NOT_CONNECTED')::text AS connection_state,
  COALESCE(connection.warning_code, '')::text AS warning_code,
  connection.connected_at,
  connection.created_at AS connection_created_at,
  connection.updated_at AS connection_updated_at,
  COALESCE(credential.id, 0)::bigint AS credential_id,
  COALESCE(credential.version, 0)::bigint AS credential_version,
  COALESCE(credential.state, '')::text AS credential_state,
  COALESCE(credential.api_key_ciphertext, ''::bytea)::bytea AS api_key_ciphertext,
  COALESCE(credential.api_secret_ciphertext, ''::bytea)::bytea AS api_secret_ciphertext,
  credential.created_at AS credential_created_at,
  credential.updated_at AS credential_updated_at,
  requested.ordinality::bigint AS input_ordinality
FROM requested
LEFT JOIN worm_wallet_connections AS connection
  ON connection.wallet_id = requested.wallet_id
LEFT JOIN worm_wallet_credentials AS credential
  ON credential.wallet_id = connection.wallet_id
  AND credential.state = 'ACTIVE'
ORDER BY requested.ordinality;

-- name: Ping :one
SELECT 1::int AS ok;
