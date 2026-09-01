-- name: GetPositionCashOutCommand :one
SELECT *
FROM worm_position_cash_out_commands
WHERE id = sqlc.arg(id)::uuid;

-- name: CreatePositionCashOut :one
INSERT INTO worm_position_cash_outs (
  id, owner_account_id, wallet_id, wallet_address, credential_version,
  position_pubkey, market_condition_id, is_yes, position_created_at,
  position_request_pubkey, shares, intent_digest_sha256, state, reason_code,
  provider_state, provider_is_closed, provider_is_liquidated,
  authorization_expires_at, completed_at, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid, sqlc.arg(owner_account_id)::uuid,
  sqlc.arg(wallet_id)::bigint, sqlc.arg(wallet_address)::text,
  sqlc.arg(credential_version)::bigint, sqlc.arg(position_pubkey)::text,
  sqlc.arg(market_condition_id)::text, sqlc.arg(is_yes)::boolean,
  sqlc.arg(position_created_at)::timestamptz,
  sqlc.arg(position_request_pubkey)::text, sqlc.arg(shares)::text,
  sqlc.arg(intent_digest_sha256)::bytea, sqlc.arg(state)::text,
  sqlc.arg(reason_code)::text, sqlc.arg(provider_state)::text,
  sqlc.arg(provider_is_closed)::boolean,
  sqlc.arg(provider_is_liquidated)::boolean,
  sqlc.arg(authorization_expires_at)::timestamptz,
  sqlc.narg(completed_at)::timestamptz,
  sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz
)
RETURNING *;

-- name: CreatePositionCashOutCommand :one
INSERT INTO worm_position_cash_out_commands (
  id, cash_out_id, owner_account_id, kind, state, request_sha256,
  cash_out_revision_after, result_code, completed_at, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid, sqlc.arg(cash_out_id)::uuid,
  sqlc.arg(owner_account_id)::uuid, sqlc.arg(kind)::text, 'APPLIED',
  sqlc.arg(request_sha256)::bytea, sqlc.arg(cash_out_revision_after)::bigint,
  sqlc.arg(result_code)::text, sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz
)
RETURNING *;

-- name: GetPositionCashOut :one
SELECT *
FROM worm_position_cash_outs
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid;

-- name: GetPositionCashOutByID :one
SELECT *
FROM worm_position_cash_outs
WHERE id = sqlc.arg(id)::uuid;

-- name: GetPositionCashOutForUpdate :one
SELECT *
FROM worm_position_cash_outs
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
FOR UPDATE;

-- name: GetPositionCashOutByIDForUpdate :one
SELECT *
FROM worm_position_cash_outs
WHERE id = sqlc.arg(id)::uuid
FOR UPDATE;

-- name: GetPositionCashOutAuthorization :one
SELECT *
FROM worm_position_cash_out_authorizations
WHERE cash_out_id = sqlc.arg(cash_out_id)::uuid;

-- name: GetPositionCashOutAttempt :one
SELECT *
FROM worm_position_cash_out_attempts
WHERE cash_out_id = sqlc.arg(cash_out_id)::uuid;

-- name: CountActiveExecutionWalletLocksForCashOut :one
SELECT COUNT(*)::bigint
FROM worm_execution_wallet_locks
WHERE wallet_id = sqlc.arg(wallet_id)::bigint;

-- name: CountActivePositionCashOutsForWallet :one
SELECT COUNT(*)::bigint
FROM worm_position_cash_outs
WHERE wallet_id = sqlc.arg(wallet_id)::bigint
  AND state IN (
    'AWAITING_AUTHORIZATION', 'QUEUED', 'PREFLIGHTING', 'CLOSING',
    'AWAITING_COMPLETION', 'RECONCILIATION_REQUIRED'
  );

-- name: CountActivePositionCashOutsForWallets :one
SELECT COUNT(*)::bigint
FROM worm_position_cash_outs
WHERE wallet_id = ANY(sqlc.arg(wallet_ids)::bigint[])
  AND state IN (
    'AWAITING_AUTHORIZATION', 'QUEUED', 'PREFLIGHTING', 'CLOSING',
    'AWAITING_COMPLETION', 'RECONCILIATION_REQUIRED'
  );

-- name: ListActivePositionCashOuts :many
SELECT *
FROM worm_position_cash_outs
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND state IN (
    'AWAITING_AUTHORIZATION', 'QUEUED', 'PREFLIGHTING', 'CLOSING',
    'AWAITING_COMPLETION', 'RECONCILIATION_REQUIRED'
  )
ORDER BY created_at, id;

-- name: CreatePositionCashOutAuthorization :one
INSERT INTO worm_position_cash_out_authorizations (
  id, cash_out_id, owner_account_id, scope, proof_kind,
  session_jti_digest, access_revision, intent_digest_sha256, state,
  authorized_at, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid, sqlc.arg(cash_out_id)::uuid,
  sqlc.arg(owner_account_id)::uuid, 'WORM_POSITION_CASH_OUT',
  sqlc.arg(proof_kind)::text, sqlc.arg(session_jti_digest)::bytea,
  sqlc.arg(access_revision)::bigint, sqlc.arg(intent_digest_sha256)::bytea,
  'AUTHORIZED', sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz
)
RETURNING *;

-- name: QueueAuthorizedPositionCashOut :one
UPDATE worm_position_cash_outs
SET state = 'QUEUED', revision = revision + 1,
    authorized_at = sqlc.arg(now)::timestamptz,
    execution_expires_at = sqlc.arg(execution_expires_at)::timestamptz,
    next_poll_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND revision = sqlc.arg(expected_revision)::bigint
  AND state = 'AWAITING_AUTHORIZATION'
  AND authorization_expires_at > sqlc.arg(now)::timestamptz
RETURNING *;

-- name: EndPositionCashOutAuthorization :execrows
UPDATE worm_position_cash_out_authorizations
SET state = sqlc.arg(state)::text,
    ended_at = sqlc.arg(now)::timestamptz,
    end_reason_code = sqlc.arg(reason_code)::text,
    updated_at = sqlc.arg(now)::timestamptz
WHERE cash_out_id = sqlc.arg(cash_out_id)::uuid
  AND state = 'AUTHORIZED';

-- name: ListRecoverablePositionCashOutKeys :many
SELECT id
FROM worm_position_cash_outs
WHERE (
    state = 'QUEUED'
    OR (
      state IN ('PREFLIGHTING', 'CLOSING')
      AND (claim_id IS NULL OR claim_expires_at <= sqlc.arg(now)::timestamptz)
    )
    OR (
      state = 'AWAITING_COMPLETION'
      AND next_poll_at <= sqlc.arg(now)::timestamptz
      AND (claim_id IS NULL OR claim_expires_at <= sqlc.arg(now)::timestamptz)
    )
    OR (
      state = 'RECONCILIATION_REQUIRED'
      AND next_poll_at <= sqlc.arg(now)::timestamptz
      AND (claim_id IS NULL OR claim_expires_at <= sqlc.arg(now)::timestamptz)
    )
  )
ORDER BY COALESCE(next_poll_at, updated_at), created_at, id
LIMIT sqlc.arg(recovery_limit)::integer;

-- name: ClaimPositionCashOut :one
UPDATE worm_position_cash_outs
SET state = CASE WHEN state = 'QUEUED' THEN 'PREFLIGHTING' ELSE state END,
    revision = revision + 1,
    claim_id = sqlc.arg(claim_id)::uuid,
    claim_owner = sqlc.arg(claim_owner)::text,
    claim_expires_at = sqlc.arg(claim_expires_at)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND authorized_at IS NOT NULL
  AND execution_expires_at IS NOT NULL
  AND (
    state = 'QUEUED'
    OR (
      state IN ('PREFLIGHTING', 'CLOSING')
      AND (claim_id IS NULL OR claim_expires_at <= sqlc.arg(now)::timestamptz)
    )
    OR (
      state = 'AWAITING_COMPLETION'
      AND next_poll_at <= sqlc.arg(now)::timestamptz
      AND (claim_id IS NULL OR claim_expires_at <= sqlc.arg(now)::timestamptz)
    )
    OR (
      state = 'RECONCILIATION_REQUIRED'
      AND next_poll_at <= sqlc.arg(now)::timestamptz
      AND (claim_id IS NULL OR claim_expires_at <= sqlc.arg(now)::timestamptz)
    )
  )
RETURNING *;

-- name: RenewPositionCashOutClaim :one
UPDATE worm_position_cash_outs
SET claim_expires_at = sqlc.arg(claim_expires_at)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND claim_id = sqlc.arg(claim_id)::uuid
  AND claim_owner = sqlc.arg(claim_owner)::text
  AND claim_expires_at > sqlc.arg(now)::timestamptz
  AND state IN (
    'PREFLIGHTING', 'CLOSING', 'AWAITING_COMPLETION',
    'RECONCILIATION_REQUIRED'
  )
RETURNING *;

-- name: BeginPositionCashOutClosing :one
UPDATE worm_position_cash_outs
SET state = 'CLOSING', revision = revision + 1,
    reason_code = '', provider_state = sqlc.arg(provider_state)::text,
    provider_is_closed = FALSE, provider_is_liquidated = FALSE,
    poll_count = poll_count + 1, next_poll_at = NULL,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'PREFLIGHTING'
  AND claim_id = sqlc.arg(claim_id)::uuid
  AND claim_expires_at > sqlc.arg(now)::timestamptz
  AND execution_expires_at > sqlc.arg(now)::timestamptz
RETURNING *;

-- name: CreatePositionCashOutAttempt :one
INSERT INTO worm_position_cash_out_attempts (
  id, cash_out_id, state, request_sha256, prepared_at, created_at, updated_at
)
SELECT sqlc.arg(id)::uuid, cash_outs.id, 'PREPARED',
       sqlc.arg(request_sha256)::bytea, sqlc.arg(now)::timestamptz,
       sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz
FROM worm_position_cash_outs AS cash_outs
WHERE cash_outs.id = sqlc.arg(cash_out_id)::uuid
  AND cash_outs.state = 'CLOSING'
  AND cash_outs.claim_id = sqlc.arg(claim_id)::uuid
  AND cash_outs.claim_expires_at > sqlc.arg(now)::timestamptz
  AND cash_outs.execution_expires_at > sqlc.arg(now)::timestamptz
RETURNING worm_position_cash_out_attempts.*;

-- name: DispatchPositionCashOutAttempt :one
UPDATE worm_position_cash_out_attempts AS attempts
SET state = 'DISPATCHED', dispatched_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
FROM worm_position_cash_outs AS cash_outs
WHERE attempts.id = sqlc.arg(id)::uuid
  AND attempts.cash_out_id = sqlc.arg(cash_out_id)::uuid
  AND attempts.state = 'PREPARED'
  AND cash_outs.id = attempts.cash_out_id
  AND cash_outs.state = 'CLOSING'
  AND cash_outs.claim_id = sqlc.arg(claim_id)::uuid
  AND cash_outs.claim_expires_at > sqlc.arg(now)::timestamptz
  AND cash_outs.execution_expires_at > sqlc.arg(now)::timestamptz
RETURNING attempts.*;

-- name: TouchPositionCashOutAfterDispatch :execrows
UPDATE worm_position_cash_outs
SET revision = revision + 1, updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'CLOSING'
  AND claim_id = sqlc.arg(claim_id)::uuid;

-- name: ResolvePositionCashOutAttempt :one
UPDATE worm_position_cash_out_attempts
SET state = sqlc.arg(state)::text,
    http_status = sqlc.narg(http_status)::integer,
    provider_code = sqlc.narg(provider_code)::integer,
    provider_slug = sqlc.arg(provider_slug)::text,
    provider_state = sqlc.arg(provider_state)::text,
    error_code = sqlc.arg(error_code)::text,
    completed_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND cash_out_id = sqlc.arg(cash_out_id)::uuid
  AND state = 'DISPATCHED'
  AND sqlc.arg(state)::text IN ('ACKNOWLEDGED', 'REJECTED', 'OUTCOME_UNKNOWN')
RETURNING *;

-- name: RecordPositionCashOutObservation :one
UPDATE worm_position_cash_outs
SET state = sqlc.arg(next_state)::text,
    revision = revision + 1,
    reason_code = sqlc.arg(reason_code)::text,
    provider_state = sqlc.arg(provider_state)::text,
    provider_is_closed = sqlc.arg(provider_is_closed)::boolean,
    provider_is_liquidated = sqlc.arg(provider_is_liquidated)::boolean,
    next_poll_at = sqlc.narg(next_poll_at)::timestamptz,
    poll_count = poll_count + 1,
    reconcile_requested_at = NULL,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    completed_at = CASE
      WHEN sqlc.arg(next_state)::text IN ('COMPLETED', 'FAILED')
        THEN sqlc.arg(now)::timestamptz
      ELSE NULL
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = sqlc.arg(expected_state)::text
  AND claim_id = sqlc.arg(claim_id)::uuid
  AND sqlc.arg(next_state)::text IN (
    'AWAITING_COMPLETION', 'COMPLETED', 'FAILED', 'RECONCILIATION_REQUIRED'
  )
  AND (
    sqlc.arg(next_state)::text <> 'COMPLETED'
    OR sqlc.arg(provider_is_closed)::boolean
  )
  AND (
    sqlc.arg(next_state)::text NOT IN ('FAILED', 'RECONCILIATION_REQUIRED')
    OR sqlc.arg(reason_code)::text <> ''
  )
RETURNING *;

-- name: RequestPositionCashOutReconciliation :one
UPDATE worm_position_cash_outs
SET revision = revision + 1,
    reconcile_requested_at = sqlc.arg(now)::timestamptz,
    next_poll_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND revision = sqlc.arg(expected_revision)::bigint
  AND state = 'RECONCILIATION_REQUIRED'
  AND claim_id IS NULL
  AND reconcile_requested_at IS NULL
RETURNING *;

-- name: ExpireUnauthorizedPositionCashOuts :many
UPDATE worm_position_cash_outs
SET state = 'EXPIRED', revision = revision + 1,
    reason_code = 'AUTHORIZATION_EXPIRED',
    completed_at = sqlc.arg(now)::timestamptz,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id IN (
  SELECT id
  FROM worm_position_cash_outs
  WHERE state = 'AWAITING_AUTHORIZATION'
    AND authorization_expires_at <= sqlc.arg(now)::timestamptz
  ORDER BY authorization_expires_at, id
  LIMIT sqlc.arg(expire_limit)::integer
  FOR UPDATE SKIP LOCKED
)
RETURNING *;

-- name: ExpireUndispatchedPositionCashOuts :many
UPDATE worm_position_cash_outs AS cash_outs
SET state = 'FAILED', revision = revision + 1,
    reason_code = 'AUTHORIZATION_EXPIRED',
    completed_at = sqlc.arg(now)::timestamptz,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    updated_at = sqlc.arg(now)::timestamptz
WHERE cash_outs.id IN (
  SELECT candidates.id
  FROM worm_position_cash_outs AS candidates
  LEFT JOIN worm_position_cash_out_attempts AS attempts
    ON attempts.cash_out_id = candidates.id
  WHERE candidates.state IN ('QUEUED', 'PREFLIGHTING', 'CLOSING')
    AND candidates.execution_expires_at <= sqlc.arg(now)::timestamptz
    AND (attempts.id IS NULL OR attempts.state = 'PREPARED')
  ORDER BY candidates.execution_expires_at, candidates.id
  LIMIT sqlc.arg(expire_limit)::integer
  FOR UPDATE OF candidates SKIP LOCKED
)
RETURNING cash_outs.*;
