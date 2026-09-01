-- name: CreateWalletSelection :one
INSERT INTO worm_trading_wallet_selections (
  owner_account_id,
  revision,
  selected_wallet_count,
  created_at,
  updated_at
) VALUES (
  sqlc.arg(owner_account_id)::uuid,
  1,
  sqlc.arg(selected_wallet_count)::integer,
  sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz
)
RETURNING *;

-- name: GetWalletSelection :one
SELECT *
FROM worm_trading_wallet_selections
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid;

-- name: GetWalletSelectionForUpdate :one
SELECT *
FROM worm_trading_wallet_selections
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
FOR UPDATE;

-- name: UpdateWalletSelection :one
UPDATE worm_trading_wallet_selections
SET revision = revision + 1,
    selected_wallet_count = sqlc.arg(selected_wallet_count)::integer,
    updated_at = sqlc.arg(now)::timestamptz
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING *;

-- name: DeleteWalletSelectionItems :exec
DELETE FROM worm_trading_wallet_selection_items
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid;

-- name: CreateWalletSelectionItem :exec
INSERT INTO worm_trading_wallet_selection_items (
  owner_account_id,
  ordinal,
  wallet_id,
  address,
  selected_at
) VALUES (
  sqlc.arg(owner_account_id)::uuid,
  sqlc.arg(ordinal)::integer,
  sqlc.arg(wallet_id)::bigint,
  sqlc.arg(address)::text,
  sqlc.arg(now)::timestamptz
);

-- name: ListWalletSelectionItems :many
SELECT *
FROM worm_trading_wallet_selection_items
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
ORDER BY ordinal;

-- name: GetWalletSelectionItem :one
SELECT *
FROM worm_trading_wallet_selection_items
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND wallet_id = sqlc.arg(wallet_id)::bigint
  AND address = sqlc.arg(address)::text;

-- name: UpsertWalletRetirement :exec
INSERT INTO worm_trading_wallet_retirements (
  owner_account_id,
  wallet_id,
  address,
  prior_ordinal,
  retired_from_revision,
  retired_at,
  updated_at
) VALUES (
  sqlc.arg(owner_account_id)::uuid,
  sqlc.arg(wallet_id)::bigint,
  sqlc.arg(address)::text,
  sqlc.arg(prior_ordinal)::integer,
  sqlc.arg(retired_from_revision)::bigint,
  sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz
)
ON CONFLICT (owner_account_id, wallet_id) DO UPDATE
SET updated_at = EXCLUDED.updated_at;

-- name: DeleteWalletRetirement :execrows
DELETE FROM worm_trading_wallet_retirements
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND wallet_id = sqlc.arg(wallet_id)::bigint
  AND address = sqlc.arg(address)::text
  AND NOT EXISTS (
    SELECT 1
    FROM worm_wallet_connections AS connections
    WHERE connections.wallet_id = sqlc.arg(wallet_id)::bigint
      AND connections.address = sqlc.arg(address)::text
      AND (
        connections.state <> 'NOT_CONNECTED'
        OR connections.warning_code <> ''
      )
  )
  AND NOT EXISTS (
    SELECT 1
    FROM worm_wallet_credentials AS credentials
    WHERE credentials.wallet_id = sqlc.arg(wallet_id)::bigint
  )
  AND NOT EXISTS (
    SELECT 1
    FROM worm_wallet_connection_attempts AS attempts
    WHERE attempts.wallet_id = sqlc.arg(wallet_id)::bigint
      AND attempts.address = sqlc.arg(address)::text
      AND attempts.state IN ('PREPARED', 'COMPLETING', 'OUTCOME_UNKNOWN')
  );

-- name: DeleteWalletRetirementForReselection :exec
DELETE FROM worm_trading_wallet_retirements
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND wallet_id = sqlc.arg(wallet_id)::bigint;

-- name: ListWalletRetirements :many
SELECT *
FROM worm_trading_wallet_retirements
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
ORDER BY retired_at, wallet_id;

-- name: GetWalletRetirement :one
SELECT *
FROM worm_trading_wallet_retirements
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND wallet_id = sqlc.arg(wallet_id)::bigint
  AND address = sqlc.arg(address)::text;

-- name: GetWalletRetirementByWalletID :one
SELECT *
FROM worm_trading_wallet_retirements
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND wallet_id = sqlc.arg(wallet_id)::bigint;

-- name: WalletRetirementReselectionBlocked :one
SELECT EXISTS (
  SELECT 1
  FROM worm_trading_wallet_retirements AS retirements
  LEFT JOIN worm_wallet_connections AS connections
    ON connections.wallet_id = retirements.wallet_id
   AND connections.address = retirements.address
  WHERE retirements.owner_account_id = sqlc.arg(owner_account_id)::uuid
    AND retirements.wallet_id = sqlc.arg(wallet_id)::bigint
    AND (
      connections.state IN ('DISCONNECTING', 'REVOCATION_REQUIRED')
      OR EXISTS (
        SELECT 1
        FROM worm_wallet_credentials AS credentials
        WHERE credentials.wallet_id = retirements.wallet_id
          AND credentials.state IN ('REVOKING', 'REVOCATION_REQUIRED')
      )
    )
)::boolean AS blocked;

-- name: CountInvalidSelectedWalletReferencesAtRevision :one
WITH input AS (
  SELECT
    sqlc.arg(wallet_ids)::bigint[] AS wallet_ids,
    sqlc.arg(addresses)::text[] AS addresses
), requested AS (
  SELECT
    input.wallet_ids[ordinality]::bigint AS wallet_id,
    input.addresses[ordinality]::text AS address
  FROM input
  CROSS JOIN LATERAL generate_subscripts(input.wallet_ids, 1) AS ordinality
  WHERE cardinality(input.wallet_ids) = cardinality(input.addresses)
)
SELECT COUNT(*)::bigint
FROM requested
LEFT JOIN worm_trading_wallet_selections AS selections
  ON selections.owner_account_id = sqlc.arg(owner_account_id)::uuid
 AND selections.revision = sqlc.arg(expected_revision)::bigint
LEFT JOIN worm_trading_wallet_selection_items AS items
  ON items.owner_account_id = selections.owner_account_id
 AND items.wallet_id = requested.wallet_id
 AND items.address = requested.address
WHERE selections.owner_account_id IS NULL
   OR items.wallet_id IS NULL;

-- name: ListWalletRetirementBlockers :many
WITH candidates AS (
  SELECT DISTINCT unnest(sqlc.arg(wallet_ids)::bigint[])::bigint AS wallet_id
), blockers AS (
  SELECT candidates.wallet_id, 'EXECUTION_RUN_ACTIVE'::text AS reason_code
  FROM candidates
  JOIN worm_execution_wallet_locks AS locks
    ON locks.wallet_id = candidates.wallet_id
  JOIN worm_execution_runs AS runs
    ON runs.id = locks.run_id
   AND runs.owner_account_id = sqlc.arg(owner_account_id)::uuid
  UNION ALL
  SELECT candidates.wallet_id, 'POSITION_CASH_OUT_ACTIVE'::text
  FROM candidates
  JOIN worm_position_cash_outs AS cash_outs
    ON cash_outs.wallet_id = candidates.wallet_id
   AND cash_outs.owner_account_id = sqlc.arg(owner_account_id)::uuid
   AND cash_outs.state IN (
     'AWAITING_AUTHORIZATION', 'QUEUED', 'PREFLIGHTING', 'CLOSING',
     'AWAITING_COMPLETION', 'RECONCILIATION_REQUIRED'
   )
  UNION ALL
  SELECT candidates.wallet_id, 'POSITION_CASH_OUT_BATCH_ACTIVE'::text
  FROM candidates
  JOIN worm_position_cash_out_batch_wallet_locks AS locks
    ON locks.wallet_id = candidates.wallet_id
  JOIN worm_position_cash_out_batches AS batches
    ON batches.id = locks.batch_id
   AND batches.owner_account_id = sqlc.arg(owner_account_id)::uuid
  UNION ALL
  SELECT candidates.wallet_id, 'EXECUTION_ISOLATION_ACTIVE'::text
  FROM candidates
  JOIN worm_execution_step_isolations AS isolations
    ON isolations.wallet_id = candidates.wallet_id
   AND isolations.owner_account_id = sqlc.arg(owner_account_id)::uuid
   AND isolations.resolved_at IS NULL
  UNION ALL
  SELECT candidates.wallet_id, 'CONNECTION_ATTEMPT_ACTIVE'::text
  FROM candidates
  JOIN worm_wallet_connection_attempts AS attempts
    ON attempts.wallet_id = candidates.wallet_id
   AND attempts.state IN ('PREPARED', 'COMPLETING')
  UNION ALL
  SELECT candidates.wallet_id, 'CONNECT_OUTCOME_UNKNOWN'::text
  FROM candidates
  JOIN worm_wallet_connection_attempts AS attempts
    ON attempts.wallet_id = candidates.wallet_id
   AND attempts.state = 'OUTCOME_UNKNOWN'
  UNION ALL
  SELECT candidates.wallet_id, 'CONNECT_OUTCOME_UNKNOWN'::text
  FROM candidates
  JOIN worm_wallet_connections AS connections
    ON connections.wallet_id = candidates.wallet_id
   AND connections.warning_code = 'CONNECT_OUTCOME_UNKNOWN'
)
SELECT DISTINCT wallet_id, reason_code
FROM blockers
ORDER BY wallet_id, reason_code;

-- name: WalletNeedsRetirement :one
SELECT (
  EXISTS (
    SELECT 1
    FROM worm_wallet_connections AS connections
    WHERE connections.wallet_id = sqlc.arg(wallet_id)::bigint
      AND connections.address = sqlc.arg(address)::text
      AND (
        connections.state <> 'NOT_CONNECTED'
        OR connections.warning_code <> ''
      )
  )
  OR EXISTS (
    SELECT 1
    FROM worm_wallet_credentials AS credentials
    WHERE credentials.wallet_id = sqlc.arg(wallet_id)::bigint
  )
  OR EXISTS (
    SELECT 1
    FROM worm_wallet_connection_attempts AS attempts
    WHERE attempts.wallet_id = sqlc.arg(wallet_id)::bigint
      AND attempts.address = sqlc.arg(address)::text
      AND attempts.state IN ('PREPARED', 'COMPLETING', 'OUTCOME_UNKNOWN')
  )
)::boolean AS needs_retirement;

-- name: CountManagedWalletConnections :one
WITH owned_wallets AS (
  SELECT items.wallet_id, items.address, FALSE AS retiring
  FROM worm_trading_wallet_selection_items AS items
  WHERE items.owner_account_id = sqlc.arg(owner_account_id)::uuid
  UNION ALL
  SELECT retirements.wallet_id, retirements.address, TRUE AS retiring
  FROM worm_trading_wallet_retirements AS retirements
  WHERE retirements.owner_account_id = sqlc.arg(owner_account_id)::uuid
)
SELECT COUNT(DISTINCT owned_wallets.wallet_id)::bigint
FROM owned_wallets
LEFT JOIN worm_wallet_connections AS connections
  ON connections.wallet_id = owned_wallets.wallet_id
 AND connections.address = owned_wallets.address
WHERE owned_wallets.retiring
   OR connections.state <> 'NOT_CONNECTED'
   OR connections.warning_code <> ''
   OR EXISTS (
     SELECT 1
     FROM worm_wallet_credentials AS credentials
     WHERE credentials.wallet_id = owned_wallets.wallet_id
   );

-- name: WalletOccupiesManagedConnectionSlot :one
SELECT (
  EXISTS (
    SELECT 1
    FROM worm_trading_wallet_retirements AS retirements
    WHERE retirements.owner_account_id = sqlc.arg(owner_account_id)::uuid
      AND retirements.wallet_id = sqlc.arg(wallet_id)::bigint
      AND retirements.address = sqlc.arg(address)::text
  )
  OR EXISTS (
    SELECT 1
    FROM worm_trading_wallet_selection_items AS items
    JOIN worm_wallet_connections AS connections
      ON connections.wallet_id = items.wallet_id
     AND connections.address = items.address
    WHERE items.owner_account_id = sqlc.arg(owner_account_id)::uuid
      AND items.wallet_id = sqlc.arg(wallet_id)::bigint
      AND items.address = sqlc.arg(address)::text
      AND (
        connections.state <> 'NOT_CONNECTED'
        OR connections.warning_code <> ''
        OR EXISTS (
          SELECT 1
          FROM worm_wallet_credentials AS credentials
          WHERE credentials.wallet_id = items.wallet_id
        )
      )
  )
)::boolean AS occupies_slot;
