-- name: GetPositionCashOutBatchCommand :one
SELECT *
FROM worm_position_cash_out_batch_commands
WHERE id = sqlc.arg(id)::uuid;

-- name: CreatePositionCashOutBatch :one
INSERT INTO worm_position_cash_out_batches (
  id, owner_account_id, selection_digest_sha256, state, wallet_count,
  authorization_expires_at, next_poll_at, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid, sqlc.arg(owner_account_id)::uuid,
  sqlc.arg(selection_digest_sha256)::bytea, 'BUILDING',
  sqlc.arg(wallet_count)::bigint, sqlc.arg(authorization_expires_at)::timestamptz,
  sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz
)
RETURNING *;

-- name: CreatePositionCashOutBatchWallet :one
INSERT INTO worm_position_cash_out_batch_wallets (
  batch_id, ordinal, wallet_id, address, credential_version, remark,
  avatar_kind, avatar_preset_id, avatar_url, created_at, updated_at
) VALUES (
  sqlc.arg(batch_id)::uuid, sqlc.arg(ordinal)::integer,
  sqlc.arg(wallet_id)::bigint, sqlc.arg(address)::text,
  sqlc.arg(credential_version)::bigint, sqlc.arg(remark)::text,
  sqlc.arg(avatar_kind)::text, sqlc.arg(avatar_preset_id)::text,
  sqlc.arg(avatar_url)::text, sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz
)
RETURNING *;

-- name: CreatePositionCashOutBatchWalletLock :exec
INSERT INTO worm_position_cash_out_batch_wallet_locks (
  wallet_id, batch_id, address, acquired_at
) VALUES (
  sqlc.arg(wallet_id)::bigint, sqlc.arg(batch_id)::uuid,
  sqlc.arg(address)::text, sqlc.arg(now)::timestamptz
);

-- name: CreatePositionCashOutBatchCommand :one
INSERT INTO worm_position_cash_out_batch_commands (
  id, batch_id, owner_account_id, kind, state, request_sha256,
  batch_revision_after, result_code, completed_at, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid, sqlc.arg(batch_id)::uuid,
  sqlc.arg(owner_account_id)::uuid, sqlc.arg(kind)::text, 'APPLIED',
  sqlc.arg(request_sha256)::bytea, sqlc.arg(batch_revision_after)::bigint,
  sqlc.arg(result_code)::text, sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz
)
RETURNING *;

-- name: GetPositionCashOutBatch :one
SELECT *
FROM worm_position_cash_out_batches
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid;

-- name: GetPositionCashOutBatchByID :one
SELECT *
FROM worm_position_cash_out_batches
WHERE id = sqlc.arg(id)::uuid;

-- name: GetPositionCashOutBatchForUpdate :one
SELECT *
FROM worm_position_cash_out_batches
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
FOR UPDATE;

-- name: GetPositionCashOutBatchByIDForUpdate :one
SELECT *
FROM worm_position_cash_out_batches
WHERE id = sqlc.arg(id)::uuid
FOR UPDATE;

-- name: GetActivePositionCashOutBatch :one
SELECT *
FROM worm_position_cash_out_batches
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND state IN (
    'BUILDING', 'AWAITING_AUTHORIZATION', 'QUEUED', 'RUNNING',
    'PAUSE_REQUESTED', 'PAUSED', 'TERMINATE_REQUESTED',
    'RECONCILIATION_REQUIRED'
  )
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: ListPositionCashOutBatchWallets :many
SELECT *
FROM worm_position_cash_out_batch_wallets
WHERE batch_id = sqlc.arg(batch_id)::uuid
ORDER BY ordinal;

-- name: ListPositionCashOutBatchWalletIDs :many
SELECT wallet_id
FROM worm_position_cash_out_batch_wallets
WHERE batch_id = sqlc.arg(batch_id)::uuid
ORDER BY wallet_id;

-- name: GetPositionCashOutBatchWallet :one
SELECT *
FROM worm_position_cash_out_batch_wallets
WHERE batch_id = sqlc.arg(batch_id)::uuid
  AND ordinal = sqlc.arg(ordinal)::integer;

-- name: ListPositionCashOutBatchItems :many
SELECT *
FROM worm_position_cash_out_batch_items
WHERE batch_id = sqlc.arg(batch_id)::uuid
ORDER BY ordinal
LIMIT sqlc.arg(page_limit)::integer
OFFSET sqlc.arg(page_offset)::bigint;

-- name: CountPositionCashOutBatchItems :one
SELECT COUNT(*)::bigint
FROM worm_position_cash_out_batch_items
WHERE batch_id = sqlc.arg(batch_id)::uuid;

-- name: GetPositionCashOutBatchItem :one
SELECT *
FROM worm_position_cash_out_batch_items
WHERE id = sqlc.arg(id)::uuid
  AND batch_id = sqlc.arg(batch_id)::uuid;

-- name: GetPositionCashOutBatchItemForUpdate :one
SELECT *
FROM worm_position_cash_out_batch_items
WHERE id = sqlc.arg(id)::uuid
  AND batch_id = sqlc.arg(batch_id)::uuid
FOR UPDATE;

-- name: GetPositionCashOutBatchItemByOrdinalForUpdate :one
SELECT *
FROM worm_position_cash_out_batch_items
WHERE batch_id = sqlc.arg(batch_id)::uuid
  AND ordinal = sqlc.arg(ordinal)::bigint
FOR UPDATE;

-- name: GetPositionCashOutBatchItemByOrdinal :one
SELECT *
FROM worm_position_cash_out_batch_items
WHERE batch_id = sqlc.arg(batch_id)::uuid
  AND ordinal = sqlc.arg(ordinal)::bigint;

-- name: CreatePositionCashOutBatchItem :one
INSERT INTO worm_position_cash_out_batch_items (
  id, batch_id, ordinal, wallet_ordinal, position_ordinal, wallet_id,
  wallet_address, wallet_remark, credential_version, position_pubkey,
  position_request_pubkey, market_condition_id, market_title, is_yes,
  shares, position_created_at, provider_state, state, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid, sqlc.arg(batch_id)::uuid, sqlc.arg(ordinal)::bigint,
  sqlc.arg(wallet_ordinal)::integer, sqlc.arg(position_ordinal)::integer,
  sqlc.arg(wallet_id)::bigint, sqlc.arg(wallet_address)::text,
  sqlc.arg(wallet_remark)::text, sqlc.arg(credential_version)::bigint,
  sqlc.arg(position_pubkey)::text, sqlc.arg(position_request_pubkey)::text,
  sqlc.arg(market_condition_id)::text, sqlc.arg(market_title)::text,
  sqlc.arg(is_yes)::boolean, sqlc.arg(shares)::text,
  sqlc.arg(position_created_at)::timestamptz, sqlc.arg(provider_state)::text,
  'PENDING', sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz
)
RETURNING *;

-- name: SetPositionCashOutBatchWalletCounts :execrows
UPDATE worm_position_cash_out_batch_wallets
SET position_count = sqlc.arg(position_count)::bigint,
    completed_count = 0,
    updated_at = sqlc.arg(now)::timestamptz
WHERE batch_id = sqlc.arg(batch_id)::uuid
  AND ordinal = sqlc.arg(ordinal)::integer;

-- name: CompletePositionCashOutBatchBuild :one
UPDATE worm_position_cash_out_batches
SET state = 'AWAITING_AUTHORIZATION', revision = revision + 1,
    build_stage = 'COMPLETED', intent_digest_sha256 = sqlc.arg(intent_digest_sha256)::bytea,
    position_count = sqlc.arg(position_count)::bigint,
    next_item_ordinal = 1, current_item_ordinal = NULL,
    authorization_expires_at = sqlc.arg(authorization_expires_at)::timestamptz,
    next_poll_at = NULL, claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'BUILDING'
  AND claim_id = sqlc.arg(claim_id)::uuid
RETURNING *;

-- name: FailPositionCashOutBatchBuild :one
UPDATE worm_position_cash_out_batches
SET state = 'FAILED', revision = revision + 1,
    reason_code = sqlc.arg(reason_code)::text,
    build_stage = sqlc.arg(build_stage)::text,
    next_poll_at = NULL, claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    completed_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'BUILDING'
  AND claim_id = sqlc.arg(claim_id)::uuid
RETURNING *;

-- name: ReleasePositionCashOutBatchWalletLocks :execrows
DELETE FROM worm_position_cash_out_batch_wallet_locks
WHERE batch_id = sqlc.arg(batch_id)::uuid;

-- name: CountActivePositionCashOutBatchWalletLocks :one
SELECT COUNT(*)::bigint
FROM worm_position_cash_out_batch_wallet_locks
WHERE wallet_id = ANY(sqlc.arg(wallet_ids)::bigint[]);

-- name: CountActiveExecutionWalletLocksForCashOutBatch :one
SELECT COUNT(*)::bigint
FROM worm_execution_wallet_locks
WHERE wallet_id = ANY(sqlc.arg(wallet_ids)::bigint[]);

-- name: CountActivePositionCashOutBatchWalletLock :one
SELECT COUNT(*)::bigint
FROM worm_position_cash_out_batch_wallet_locks
WHERE wallet_id = sqlc.arg(wallet_id)::bigint;

-- name: GetPositionCashOutBatchAuthorization :one
SELECT *
FROM worm_position_cash_out_batch_authorizations
WHERE batch_id = sqlc.arg(batch_id)::uuid
ORDER BY authorized_at DESC, id DESC
LIMIT 1;

-- name: SupersedePositionCashOutBatchAuthorization :execrows
UPDATE worm_position_cash_out_batch_authorizations
SET state = 'SUPERSEDED', ended_at = sqlc.arg(now)::timestamptz,
    end_reason_code = 'REAUTHORIZED', updated_at = sqlc.arg(now)::timestamptz
WHERE batch_id = sqlc.arg(batch_id)::uuid
  AND state = 'AUTHORIZED';

-- name: CreatePositionCashOutBatchAuthorization :one
INSERT INTO worm_position_cash_out_batch_authorizations (
  id, batch_id, owner_account_id, scope, proof_kind, session_jti_digest,
  access_revision, intent_digest_sha256, state, authorized_at,
  created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid, sqlc.arg(batch_id)::uuid,
  sqlc.arg(owner_account_id)::uuid, 'WORM_POSITION_CASH_OUT_BATCH',
  sqlc.arg(proof_kind)::text, sqlc.arg(session_jti_digest)::bytea,
  sqlc.arg(access_revision)::bigint, sqlc.arg(intent_digest_sha256)::bytea,
  'AUTHORIZED', sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz
)
RETURNING *;

-- name: QueueAuthorizedPositionCashOutBatch :one
UPDATE worm_position_cash_out_batches
SET state = CASE
      WHEN state = 'AWAITING_AUTHORIZATION' THEN 'QUEUED'
      ELSE state
    END,
    revision = revision + 1, reason_code = '', authorized_at = sqlc.arg(now)::timestamptz,
    next_poll_at = CASE
      WHEN state = 'AWAITING_AUTHORIZATION' THEN sqlc.arg(now)::timestamptz
      ELSE next_poll_at
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND revision = sqlc.arg(expected_revision)::bigint
  AND state IN ('AWAITING_AUTHORIZATION', 'PAUSED')
  AND intent_digest_sha256 IS NOT NULL
  AND (
    state = 'PAUSED'
    OR authorization_expires_at > sqlc.arg(now)::timestamptz
  )
RETURNING *;

-- name: EndPositionCashOutBatchAuthorization :execrows
UPDATE worm_position_cash_out_batch_authorizations
SET state = sqlc.arg(state)::text,
    ended_at = sqlc.arg(now)::timestamptz,
    end_reason_code = sqlc.arg(reason_code)::text,
    updated_at = sqlc.arg(now)::timestamptz
WHERE batch_id = sqlc.arg(batch_id)::uuid
  AND state = 'AUTHORIZED';

-- name: ListRecoverablePositionCashOutBatchKeys :many
SELECT id
FROM worm_position_cash_out_batches
WHERE (
    (
      state IN ('BUILDING', 'QUEUED', 'RUNNING', 'PAUSE_REQUESTED')
      AND (next_poll_at IS NULL OR next_poll_at <= sqlc.arg(now)::timestamptz)
    )
    OR (
      state = 'TERMINATE_REQUESTED'
      AND (
        (next_poll_at IS NOT NULL AND next_poll_at <= sqlc.arg(now)::timestamptz)
        OR check_requested_at IS NOT NULL
      )
    )
    OR (
      state = 'RECONCILIATION_REQUIRED'
      AND check_requested_at IS NOT NULL
    )
    OR (
      state = 'PAUSED'
      AND check_requested_at IS NOT NULL
    )
  )
  AND (claim_id IS NULL OR claim_expires_at <= sqlc.arg(now)::timestamptz)
ORDER BY COALESCE(next_poll_at, updated_at), created_at, id
LIMIT sqlc.arg(recovery_limit)::integer;

-- name: ClaimPositionCashOutBatch :one
UPDATE worm_position_cash_out_batches
SET state = CASE WHEN state = 'QUEUED' THEN 'RUNNING' ELSE state END,
    revision = revision + 1,
    execution_started_at = CASE
      WHEN state = 'QUEUED' THEN COALESCE(execution_started_at, sqlc.arg(now)::timestamptz)
      ELSE execution_started_at
    END,
    claim_id = sqlc.arg(claim_id)::uuid,
    claim_owner = sqlc.arg(claim_owner)::text,
    claim_expires_at = sqlc.arg(claim_expires_at)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND (
    state IN ('BUILDING', 'QUEUED', 'RUNNING', 'PAUSE_REQUESTED')
    OR (
      state = 'TERMINATE_REQUESTED'
      AND (next_poll_at IS NOT NULL OR check_requested_at IS NOT NULL)
    )
    OR (state IN ('PAUSED', 'RECONCILIATION_REQUIRED') AND check_requested_at IS NOT NULL)
  )
  AND (next_poll_at IS NULL OR next_poll_at <= sqlc.arg(now)::timestamptz
       OR check_requested_at IS NOT NULL)
  AND (claim_id IS NULL OR claim_expires_at <= sqlc.arg(now)::timestamptz)
RETURNING *;

-- name: RenewPositionCashOutBatchClaim :one
UPDATE worm_position_cash_out_batches
SET claim_expires_at = sqlc.arg(claim_expires_at)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND claim_id = sqlc.arg(claim_id)::uuid
  AND claim_owner = sqlc.arg(claim_owner)::text
  AND claim_expires_at > sqlc.arg(now)::timestamptz
  AND state IN ('BUILDING', 'RUNNING', 'PAUSE_REQUESTED', 'PAUSED',
                'TERMINATE_REQUESTED', 'RECONCILIATION_REQUIRED')
RETURNING *;

-- name: SetPositionCashOutBatchNextPoll :one
UPDATE worm_position_cash_out_batches
SET next_poll_at = sqlc.narg(next_poll_at)::timestamptz,
    check_requested_at = NULL,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND claim_id = sqlc.arg(claim_id)::uuid
RETURNING *;

-- name: ResumePositionCashOutBatchBalanceAfterReconciliation :one
UPDATE worm_position_cash_out_batches
SET state = CASE
      WHEN state = 'TERMINATE_REQUESTED' THEN 'TERMINATE_REQUESTED'
      ELSE 'PAUSE_REQUESTED'
    END,
    revision = revision + 1, reason_code = '',
    next_poll_at = sqlc.arg(next_poll_at)::timestamptz,
    check_requested_at = NULL,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state IN ('PAUSED', 'RECONCILIATION_REQUIRED', 'TERMINATE_REQUESTED')
  AND current_item_ordinal = sqlc.arg(current_item_ordinal)::bigint
  AND claim_id = sqlc.arg(claim_id)::uuid
RETURNING *;

-- name: CreateBatchChildPositionCashOut :one
INSERT INTO worm_position_cash_outs (
  id, owner_account_id, wallet_id, wallet_address, credential_version,
  position_pubkey, market_condition_id, is_yes, position_created_at,
  position_request_pubkey, shares, intent_digest_sha256, state, reason_code,
  provider_state, provider_is_closed, provider_is_liquidated,
  authorization_expires_at, execution_expires_at, authorized_at,
  next_poll_at, created_at, updated_at, batch_id, batch_item_id
)
SELECT
  sqlc.arg(cash_out_id)::uuid, batches.owner_account_id, items.wallet_id,
  items.wallet_address, items.credential_version, items.position_pubkey,
  items.market_condition_id, items.is_yes, items.position_created_at,
  items.position_request_pubkey, items.shares,
  sqlc.arg(intent_digest_sha256)::bytea, 'QUEUED', '',
  sqlc.arg(provider_state)::text, FALSE, FALSE,
  TIMESTAMPTZ '9999-12-31 23:59:59+00',
  TIMESTAMPTZ '9999-12-31 23:59:59+00', sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz, batches.id, items.id
FROM worm_position_cash_out_batches AS batches
JOIN worm_position_cash_out_batch_items AS items
  ON items.batch_id = batches.id
WHERE batches.id = sqlc.arg(batch_id)::uuid
  AND batches.state = 'RUNNING'
  AND batches.claim_id = sqlc.arg(claim_id)::uuid
  AND batches.current_item_ordinal IS NULL
  AND items.id = sqlc.arg(item_id)::uuid
  AND items.ordinal = batches.next_item_ordinal
  AND items.state = 'PENDING'
RETURNING worm_position_cash_outs.*;

-- name: DeferPositionCashOutBatchPreflight :one
UPDATE worm_position_cash_out_batches AS batches
SET next_poll_at = sqlc.arg(next_poll_at)::timestamptz,
    check_requested_at = NULL,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    updated_at = sqlc.arg(now)::timestamptz
FROM worm_position_cash_out_batch_items AS items
WHERE batches.id = sqlc.arg(batch_id)::uuid
  AND batches.state = 'RUNNING'
  AND batches.claim_id = sqlc.arg(claim_id)::uuid
  AND batches.current_item_ordinal IS NULL
  AND items.id = sqlc.arg(item_id)::uuid
  AND items.batch_id = batches.id
  AND items.ordinal = batches.next_item_ordinal
  AND items.state = 'PENDING'
  AND items.child_cash_out_id IS NULL
RETURNING batches.*;

-- name: DeferPositionCashOutBatchCheck :one
UPDATE worm_position_cash_out_batches AS batches
SET next_poll_at = sqlc.arg(next_poll_at)::timestamptz,
    check_requested_at = sqlc.arg(now)::timestamptz,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    updated_at = sqlc.arg(now)::timestamptz
FROM worm_position_cash_out_batch_items AS items
WHERE batches.id = sqlc.arg(batch_id)::uuid
  AND batches.state IN ('PAUSED', 'RECONCILIATION_REQUIRED', 'TERMINATE_REQUESTED')
  AND batches.claim_id = sqlc.arg(claim_id)::uuid
  AND batches.current_item_ordinal = items.ordinal
  AND items.batch_id = batches.id
  AND items.state = 'RECONCILIATION_REQUIRED'
  AND items.child_cash_out_id IS NOT NULL
RETURNING batches.*;

-- name: ActivatePositionCashOutBatchItem :one
UPDATE worm_position_cash_out_batch_items
SET state = 'PREFLIGHTING', reason_code = '',
    child_cash_out_id = sqlc.arg(cash_out_id)::uuid,
    provider_state = sqlc.arg(provider_state)::text,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(item_id)::uuid
  AND batch_id = sqlc.arg(batch_id)::uuid
  AND state = 'PENDING'
  AND child_cash_out_id IS NULL
RETURNING *;

-- name: SetPositionCashOutBatchCurrentItem :one
UPDATE worm_position_cash_out_batches
SET current_item_ordinal = sqlc.arg(item_ordinal)::bigint,
    next_poll_at = sqlc.arg(next_poll_at)::timestamptz,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND claim_id = sqlc.arg(claim_id)::uuid
  AND current_item_ordinal IS NULL
  AND next_item_ordinal = sqlc.arg(item_ordinal)::bigint
RETURNING *;

-- name: SetPositionCashOutBatchItemBalanceBaseline :one
UPDATE worm_position_cash_out_batch_items
SET state = 'CLOSING',
    baseline_usdc_mint = sqlc.arg(mint)::text,
    baseline_usdc_decimals = sqlc.arg(decimals)::integer,
    baseline_usdc_atomic_amount = sqlc.arg(atomic_amount)::text,
    baseline_usdc_observed_slot = sqlc.arg(observed_slot)::bigint,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(item_id)::uuid
  AND batch_id = sqlc.arg(batch_id)::uuid
  AND child_cash_out_id = sqlc.arg(cash_out_id)::uuid
  AND state IN ('PREFLIGHTING', 'CLOSING')
  AND baseline_usdc_mint = ''
RETURNING *;

-- name: TouchPositionCashOutBatchAfterDispatch :execrows
UPDATE worm_position_cash_out_batches
SET revision = revision + 1, next_poll_at = sqlc.arg(next_poll_at)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND current_item_ordinal = sqlc.arg(item_ordinal)::bigint;

-- name: RecordPositionCashOutBatchItemState :one
UPDATE worm_position_cash_out_batch_items
SET state = sqlc.arg(next_state)::text,
    reason_code = sqlc.arg(reason_code)::text,
    provider_state = sqlc.arg(provider_state)::text,
    balance_started_at = CASE
      WHEN sqlc.arg(next_state)::text = 'AWAITING_BALANCE'
        THEN COALESCE(balance_started_at, sqlc.arg(now)::timestamptz)
      ELSE balance_started_at
    END,
    balance_deadline_at = CASE
      WHEN sqlc.arg(next_state)::text = 'AWAITING_BALANCE'
        THEN COALESCE(balance_deadline_at, sqlc.arg(balance_deadline_at)::timestamptz)
      ELSE balance_deadline_at
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(item_id)::uuid
  AND batch_id = sqlc.arg(batch_id)::uuid
  AND state = sqlc.arg(expected_state)::text
RETURNING *;

-- name: RecordPositionCashOutBatchObservedBalance :one
UPDATE worm_position_cash_out_batch_items
SET observed_usdc_mint = sqlc.arg(mint)::text,
    observed_usdc_decimals = sqlc.arg(decimals)::integer,
    observed_usdc_atomic_amount = sqlc.arg(atomic_amount)::text,
    observed_usdc_slot = sqlc.arg(observed_slot)::bigint,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(item_id)::uuid
  AND batch_id = sqlc.arg(batch_id)::uuid
  AND state = 'AWAITING_BALANCE'
RETURNING *;

-- name: CompletePositionCashOutBatchItemBalance :one
UPDATE worm_position_cash_out_batch_items
SET state = 'COMPLETED', reason_code = '',
    observed_usdc_mint = sqlc.arg(mint)::text,
    observed_usdc_decimals = sqlc.arg(decimals)::integer,
    observed_usdc_atomic_amount = sqlc.arg(atomic_amount)::text,
    observed_usdc_slot = sqlc.arg(observed_slot)::bigint,
    delta_usdc_atomic_amount = sqlc.arg(delta_atomic_amount)::text,
    balance_confirmed_at = sqlc.arg(now)::timestamptz,
    completed_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(item_id)::uuid
  AND batch_id = sqlc.arg(batch_id)::uuid
  AND state = 'AWAITING_BALANCE'
RETURNING *;

-- name: IncrementPositionCashOutBatchWalletCompleted :execrows
UPDATE worm_position_cash_out_batch_wallets
SET completed_count = completed_count + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE batch_id = sqlc.arg(batch_id)::uuid
  AND ordinal = sqlc.arg(wallet_ordinal)::integer
  AND completed_count < position_count;

-- name: AdvancePositionCashOutBatchAfterBalance :one
UPDATE worm_position_cash_out_batches
SET revision = revision + 1,
    completed_count = completed_count + 1,
    current_item_ordinal = NULL,
    next_item_ordinal = CASE
      WHEN completed_count + 1 < position_count THEN sqlc.arg(next_item_ordinal)::bigint
      ELSE NULL
    END,
    state = CASE
      WHEN state IN ('PAUSED', 'RECONCILIATION_REQUIRED') THEN 'PAUSED'
      WHEN state = 'PAUSE_REQUESTED' THEN 'PAUSED'
      WHEN state = 'TERMINATE_REQUESTED' THEN 'TERMINATE_REQUESTED'
      ELSE 'RUNNING'
    END,
    reason_code = CASE
      WHEN state IN ('PAUSED', 'RECONCILIATION_REQUIRED') THEN 'BALANCE_CONFIRMED'
      WHEN state = 'PAUSE_REQUESTED' THEN ''
      ELSE reason_code
    END,
    next_poll_at = CASE
      WHEN state IN ('PAUSED', 'RECONCILIATION_REQUIRED', 'PAUSE_REQUESTED') THEN NULL
      ELSE sqlc.arg(now)::timestamptz
    END,
    check_requested_at = NULL,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND current_item_ordinal = sqlc.arg(current_item_ordinal)::bigint
  AND claim_id = sqlc.arg(claim_id)::uuid
RETURNING *;

-- name: PausePositionCashOutBatchForBalance :one
UPDATE worm_position_cash_out_batches
SET state = CASE
      WHEN state = 'TERMINATE_REQUESTED' THEN 'TERMINATE_REQUESTED'
      ELSE 'PAUSED'
    END,
    revision = revision + 1, reason_code = sqlc.arg(reason_code)::text,
    next_poll_at = NULL, check_requested_at = NULL,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND current_item_ordinal = sqlc.arg(current_item_ordinal)::bigint
  AND claim_id = sqlc.arg(claim_id)::uuid
RETURNING *;

-- name: MarkPositionCashOutBatchItemBlocking :one
UPDATE worm_position_cash_out_batch_items
SET state = sqlc.arg(next_state)::text,
    reason_code = sqlc.arg(reason_code)::text,
    completed_at = CASE
      WHEN sqlc.arg(next_state)::text = 'FAILED' THEN sqlc.arg(now)::timestamptz
      ELSE NULL
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(item_id)::uuid
  AND batch_id = sqlc.arg(batch_id)::uuid
  AND state NOT IN ('COMPLETED', 'FAILED', 'NOT_EXECUTED')
RETURNING *;

-- name: MarkPositionCashOutBatchBlocking :one
UPDATE worm_position_cash_out_batches
SET state = sqlc.arg(next_state)::text,
    revision = revision + 1, reason_code = sqlc.arg(reason_code)::text,
    not_executed_count = CASE
      WHEN sqlc.arg(next_state)::text = 'FAILED'
        THEN sqlc.arg(not_executed_count)::bigint
      ELSE not_executed_count
    END,
    next_poll_at = NULL, check_requested_at = NULL,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    current_item_ordinal = CASE
      WHEN sqlc.arg(next_state)::text = 'FAILED' THEN NULL
      ELSE COALESCE(current_item_ordinal, sqlc.arg(item_ordinal)::bigint)
    END,
    next_item_ordinal = CASE
      WHEN sqlc.arg(next_state)::text = 'FAILED' THEN NULL
      ELSE next_item_ordinal
    END,
    completed_at = CASE
      WHEN sqlc.arg(next_state)::text = 'FAILED' THEN sqlc.arg(now)::timestamptz
      ELSE NULL
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND claim_id = sqlc.arg(claim_id)::uuid
  AND sqlc.arg(next_state)::text IN (
    'PAUSED', 'FAILED', 'RECONCILIATION_REQUIRED', 'TERMINATE_REQUESTED'
  )
RETURNING *;

-- name: ApplyPositionCashOutBatchControl :one
UPDATE worm_position_cash_out_batches
SET state = sqlc.arg(next_state)::text,
    revision = revision + 1,
    reason_code = sqlc.arg(reason_code)::text,
    next_poll_at = sqlc.narg(next_poll_at)::timestamptz,
    check_requested_at = sqlc.narg(check_requested_at)::timestamptz,
    completed_at = CASE
      WHEN sqlc.arg(next_state)::text IN ('TERMINATED', 'CANCELLED')
        THEN sqlc.arg(now)::timestamptz
      ELSE completed_at
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND revision = sqlc.arg(expected_revision)::bigint
  AND claim_id IS NULL
RETURNING *;

-- name: MarkRemainingPositionCashOutBatchItemsNotExecuted :many
UPDATE worm_position_cash_out_batch_items
SET state = 'NOT_EXECUTED', reason_code = sqlc.arg(reason_code)::text,
    completed_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE batch_id = sqlc.arg(batch_id)::uuid
  AND state = 'PENDING'
RETURNING *;

-- name: MarkCurrentPositionCashOutBatchItemNotExecuted :one
UPDATE worm_position_cash_out_batch_items
SET state = 'NOT_EXECUTED', reason_code = sqlc.arg(reason_code)::text,
    completed_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(item_id)::uuid
  AND batch_id = sqlc.arg(batch_id)::uuid
  AND state IN ('FAILED', 'RECONCILIATION_REQUIRED')
RETURNING *;

-- name: FailUndispatchedPositionCashOutBatchChild :one
UPDATE worm_position_cash_outs
SET state = 'FAILED', revision = revision + 1,
    reason_code = 'BATCH_TERMINATED', next_poll_at = NULL,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    completed_at = COALESCE(completed_at, sqlc.arg(now)::timestamptz),
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(cash_out_id)::uuid
  AND batch_id = sqlc.arg(batch_id)::uuid
  AND batch_item_id = sqlc.arg(item_id)::uuid
  AND state IN ('FAILED', 'RECONCILIATION_REQUIRED')
  AND (claim_id IS NULL OR claim_expires_at <= sqlc.arg(now)::timestamptz)
RETURNING *;

-- name: CompletePositionCashOutBatch :one
UPDATE worm_position_cash_out_batches
SET state = sqlc.arg(terminal_state)::text,
    revision = revision + 1, reason_code = '',
    not_executed_count = sqlc.arg(not_executed_count)::bigint,
    next_item_ordinal = NULL, current_item_ordinal = NULL,
    next_poll_at = NULL, check_requested_at = NULL,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    completed_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = sqlc.arg(expected_state)::text
  AND sqlc.arg(terminal_state)::text IN ('COMPLETED', 'TERMINATED')
RETURNING *;

-- name: PausePositionCashOutBatchAtBoundary :one
UPDATE worm_position_cash_out_batches
SET state = 'PAUSED', revision = revision + 1,
    reason_code = '', next_poll_at = NULL, check_requested_at = NULL,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'PAUSE_REQUESTED'
  AND current_item_ordinal IS NULL
  AND claim_id = sqlc.arg(claim_id)::uuid
RETURNING *;

-- name: ExpirePositionCashOutBatches :many
UPDATE worm_position_cash_out_batches
SET state = 'EXPIRED', revision = revision + 1,
    reason_code = 'AUTHORIZATION_EXPIRED',
    next_poll_at = NULL, check_requested_at = NULL,
    claim_id = NULL, claim_owner = '', claim_expires_at = NULL,
    completed_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id IN (
  SELECT id
  FROM worm_position_cash_out_batches
  WHERE state IN ('BUILDING', 'AWAITING_AUTHORIZATION')
    AND authorization_expires_at <= sqlc.arg(now)::timestamptz
    AND (claim_id IS NULL OR claim_expires_at <= sqlc.arg(now)::timestamptz)
  ORDER BY authorization_expires_at, id
  LIMIT sqlc.arg(expire_limit)::integer
  FOR UPDATE SKIP LOCKED
)
RETURNING *;
