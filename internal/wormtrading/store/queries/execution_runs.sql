-- name: GetExecutionRunSourcePlanForUpdate :one
SELECT plans.*
FROM worm_execution_plans AS plans
JOIN worm_market_combinations AS combinations
  ON combinations.id = plans.combination_id
 AND combinations.owner_account_id = plans.owner_account_id
WHERE plans.id = sqlc.arg(plan_id)::uuid
  AND plans.owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND plans.state = 'READY'
  AND plans.expires_at > sqlc.arg(now)::timestamptz
  AND plans.ready_step_count > 0
  AND plans.combination_revision = sqlc.arg(expected_revision)::bigint
  AND combinations.revision = plans.combination_revision
FOR UPDATE OF plans, combinations;

-- name: ListExecutionRunSourcePlanSteps :many
SELECT *
FROM worm_execution_plan_steps
WHERE plan_id = sqlc.arg(plan_id)::uuid
ORDER BY ordinal;

-- name: GetExecutionRunByCreationKey :one
SELECT *
FROM worm_execution_runs
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND idempotency_key_sha256 = sqlc.arg(idempotency_key_sha256)::bytea;

-- name: CreateExecutionRun :one
INSERT INTO worm_execution_runs (
  id, owner_account_id, plan_id, plan_version, plan_digest_sha256,
  idempotency_key_sha256, request_sha256, combination_id, combination_name,
  combination_revision, state, next_step_ordinal, wallet_count, item_count,
  total_step_count, actionable_step_count, terminal_step_count,
  satisfied_step_count, skipped_step_count, requested_at, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid, sqlc.arg(owner_account_id)::uuid, sqlc.arg(plan_id)::uuid,
  sqlc.arg(plan_version)::bigint, sqlc.arg(plan_digest_sha256)::bytea,
  sqlc.arg(idempotency_key_sha256)::bytea, sqlc.arg(request_sha256)::bytea,
  sqlc.arg(combination_id)::uuid, sqlc.arg(combination_name)::text,
  sqlc.arg(combination_revision)::bigint, 'AWAITING_AUTHORIZATION',
  sqlc.arg(next_step_ordinal)::bigint, sqlc.arg(wallet_count)::bigint,
  sqlc.arg(item_count)::bigint, sqlc.arg(total_step_count)::bigint,
  sqlc.arg(actionable_step_count)::bigint,
  sqlc.arg(satisfied_step_count)::bigint + sqlc.arg(skipped_step_count)::bigint,
  sqlc.arg(satisfied_step_count)::bigint,
  sqlc.arg(skipped_step_count)::bigint, sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz
)
RETURNING *;

-- name: SnapshotExecutionRunWallets :execrows
INSERT INTO worm_execution_run_wallets (
  run_id, ordinal, wallet_id, address, remark, avatar_kind, avatar_preset_id,
  avatar_url, connection_state, connection_warning_code, connected_at,
  credential_version, sol_atomic_amount, sol_amount, sol_decimals,
  sol_observed_slot, sol_availability, sol_error_code, usdc_mint,
  usdc_atomic_amount, usdc_amount, usdc_decimals, usdc_observed_slot,
  usdc_availability, usdc_error_code, usdc_token_account_count, status,
  reason_code
)
SELECT sqlc.arg(run_id)::uuid, ordinal, wallet_id, address, remark, avatar_kind,
       avatar_preset_id, avatar_url, connection_state, connection_warning_code,
       connected_at, credential_version, sol_atomic_amount, sol_amount,
       sol_decimals, sol_observed_slot, sol_availability, sol_error_code,
       usdc_mint, usdc_atomic_amount, usdc_amount, usdc_decimals,
       usdc_observed_slot, usdc_availability, usdc_error_code,
       usdc_token_account_count, status, reason_code
FROM worm_execution_plan_wallets
WHERE plan_id = sqlc.arg(plan_id)::uuid;

-- name: SnapshotExecutionRunItems :execrows
INSERT INTO worm_execution_run_items (
  run_id, ordinal, event_condition_id, event_title, event_logo,
  market_condition_id, market_title, market_logo, is_yes, outcome_label,
  backend, funds, leverage, preview_state, preview_reason_code,
  estimate_average_price, estimate_total_shares, estimate_total_cost,
  estimate_best_ask, estimate_worst_fill_price, estimate_is_fully_filled,
  estimate_fee_amount, estimate_user_funds_needed, estimate_liquidation_price
)
SELECT sqlc.arg(run_id)::uuid, ordinal, event_condition_id, event_title,
       event_logo, market_condition_id, market_title, market_logo, is_yes,
       outcome_label, backend, funds, leverage, state, reason_code,
       estimate_average_price, estimate_total_shares, estimate_total_cost,
       estimate_best_ask, estimate_worst_fill_price, estimate_is_fully_filled,
       estimate_fee_amount, estimate_user_funds_needed,
       estimate_liquidation_price
FROM worm_execution_plan_items
WHERE plan_id = sqlc.arg(plan_id)::uuid;

-- name: SnapshotExecutionRunSteps :execrows
INSERT INTO worm_execution_run_steps (
  run_id, ordinal, plan_step_ordinal, wallet_ordinal, item_ordinal,
  source_disposition, source_reason_code, projected_usdc_before,
  projected_usdc_after, state, reason_code, completed_at, created_at, updated_at
)
SELECT sqlc.arg(run_id)::uuid, ordinal, ordinal, wallet_ordinal, item_ordinal,
       disposition, reason_code, projected_usdc_before, projected_usdc_after,
       CASE
         WHEN disposition = 'READY' THEN 'PENDING'
         WHEN reason_code IN ('ALREADY_HELD', 'REQUEST_IN_FLIGHT') THEN 'SATISFIED'
         ELSE 'SKIPPED'
       END,
       reason_code,
       CASE WHEN disposition = 'SKIPPED' THEN sqlc.arg(now)::timestamptz END,
       sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz
FROM worm_execution_plan_steps
WHERE plan_id = sqlc.arg(plan_id)::uuid;

-- name: CountInvalidExecutionRunWalletSnapshots :one
SELECT COUNT(*)::bigint
FROM worm_execution_run_wallets AS snapshots
LEFT JOIN worm_wallet_connections AS connections
  ON connections.wallet_id = snapshots.wallet_id
 AND connections.address = snapshots.address
LEFT JOIN worm_wallet_credentials AS credentials
  ON credentials.wallet_id = snapshots.wallet_id
 AND credentials.version = snapshots.credential_version
 AND credentials.state = 'ACTIVE'
WHERE snapshots.run_id = sqlc.arg(run_id)::uuid
  AND EXISTS (
    SELECT 1
    FROM worm_execution_run_steps AS steps
    WHERE steps.run_id = snapshots.run_id
      AND steps.wallet_ordinal = snapshots.ordinal
      AND steps.source_disposition = 'READY'
  )
  AND (
    connections.wallet_id IS NULL
    OR connections.state <> 'CONNECTED'
    OR credentials.id IS NULL
  );

-- name: CountBlockedExecutionRunStepPairs :one
SELECT COUNT(*)::bigint
FROM worm_execution_run_steps AS steps
JOIN worm_execution_run_wallets AS wallets
  ON wallets.run_id = steps.run_id AND wallets.ordinal = steps.wallet_ordinal
JOIN worm_execution_run_items AS items
  ON items.run_id = steps.run_id AND items.ordinal = steps.item_ordinal
JOIN worm_execution_step_isolations AS isolations
  ON isolations.wallet_id = wallets.wallet_id
 AND isolations.market_condition_id = items.market_condition_id
 AND isolations.resolved_at IS NULL
WHERE steps.run_id = sqlc.arg(run_id)::uuid
  AND steps.source_disposition = 'READY';

-- name: CreateExecutionCombinationLock :exec
INSERT INTO worm_execution_combination_locks (
  combination_id, run_id, combination_revision, acquired_at
) VALUES (
  sqlc.arg(combination_id)::uuid, sqlc.arg(run_id)::uuid,
  sqlc.arg(combination_revision)::bigint, sqlc.arg(now)::timestamptz
);

-- name: CreateExecutionWalletLocks :execrows
INSERT INTO worm_execution_wallet_locks (wallet_id, run_id, address, acquired_at)
SELECT wallet_id, run_id, address, sqlc.arg(now)::timestamptz
FROM worm_execution_run_wallets
WHERE run_id = sqlc.arg(run_id)::uuid;

-- name: GetExecutionRun :one
SELECT *
FROM worm_execution_runs
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid;

-- name: GetExecutionRunForOwnerUpdate :one
SELECT *
FROM worm_execution_runs
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
FOR UPDATE;

-- name: GetExecutionRunByIDForUpdate :one
SELECT *
FROM worm_execution_runs
WHERE id = sqlc.arg(id)::uuid
FOR UPDATE;

-- name: CountExecutionRuns :one
SELECT COUNT(*)::bigint
FROM worm_execution_runs
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid;

-- name: ListExecutionRuns :many
SELECT *
FROM worm_execution_runs
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
ORDER BY requested_at DESC, id DESC
LIMIT sqlc.arg(page_size)::integer
OFFSET sqlc.arg(page_offset)::bigint;

-- name: ListExecutionRunWallets :many
SELECT *
FROM worm_execution_run_wallets
WHERE run_id = sqlc.arg(run_id)::uuid
ORDER BY ordinal;

-- name: ListExecutionRunItems :many
SELECT *
FROM worm_execution_run_items
WHERE run_id = sqlc.arg(run_id)::uuid
ORDER BY ordinal;

-- name: CountExecutionRunSteps :one
SELECT COUNT(*)::bigint
FROM worm_execution_run_steps AS steps
JOIN worm_execution_runs AS runs ON runs.id = steps.run_id
WHERE steps.run_id = sqlc.arg(run_id)::uuid
  AND runs.owner_account_id = sqlc.arg(owner_account_id)::uuid;

-- name: ListExecutionRunSteps :many
SELECT steps.*
FROM worm_execution_run_steps AS steps
JOIN worm_execution_runs AS runs ON runs.id = steps.run_id
WHERE steps.run_id = sqlc.arg(run_id)::uuid
  AND runs.owner_account_id = sqlc.arg(owner_account_id)::uuid
ORDER BY steps.ordinal
LIMIT sqlc.arg(page_size)::integer
OFFSET sqlc.arg(page_offset)::bigint;

-- name: GetExecutionRunStep :one
SELECT *
FROM worm_execution_run_steps
WHERE run_id = sqlc.arg(run_id)::uuid
  AND ordinal = sqlc.arg(ordinal)::bigint;

-- name: GetExecutionRunStepForUpdate :one
SELECT *
FROM worm_execution_run_steps
WHERE run_id = sqlc.arg(run_id)::uuid
  AND ordinal = sqlc.arg(ordinal)::bigint
FOR UPDATE;

-- name: ListExecutionStepMutationAttempts :many
SELECT *
FROM worm_execution_mutation_attempts
WHERE run_id = sqlc.arg(run_id)::uuid
  AND step_ordinal = sqlc.arg(step_ordinal)::bigint
ORDER BY prepared_at, id;

-- name: GetActiveExecutionStepIsolation :one
SELECT isolations.*
FROM worm_execution_step_isolations AS isolations
JOIN worm_execution_run_steps AS steps
  ON steps.run_id = sqlc.arg(run_id)::uuid
 AND steps.wallet_ordinal = sqlc.arg(wallet_ordinal)::integer
 AND steps.item_ordinal = sqlc.arg(item_ordinal)::integer
 AND isolations.run_id = steps.run_id
 AND isolations.step_ordinal = steps.ordinal
JOIN worm_execution_run_wallets AS wallets
  ON wallets.run_id = sqlc.arg(run_id)::uuid
 AND wallets.ordinal = sqlc.arg(wallet_ordinal)::integer
JOIN worm_execution_run_items AS items
  ON items.run_id = sqlc.arg(run_id)::uuid
 AND items.ordinal = sqlc.arg(item_ordinal)::integer
WHERE isolations.wallet_id = wallets.wallet_id
  AND isolations.market_condition_id = items.market_condition_id
  AND isolations.resolved_at IS NULL;

-- name: GetActiveExecutionAuthorization :one
SELECT *
FROM worm_execution_authorizations
WHERE run_id = sqlc.arg(run_id)::uuid
  AND state = 'AUTHORIZED';

-- name: SupersedeExecutionAuthorization :execrows
UPDATE worm_execution_authorizations
SET state = 'SUPERSEDED', ended_at = sqlc.arg(now)::timestamptz,
    end_reason_code = 'REAUTHORIZED', updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND state = 'AUTHORIZED';

-- name: InvalidateExecutionAuthorization :execrows
UPDATE worm_execution_authorizations
SET state = 'SUPERSEDED', ended_at = sqlc.arg(now)::timestamptz,
    end_reason_code = 'SESSION_OR_ACCESS_CHANGED', updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND state = 'AUTHORIZED';

-- name: CreateExecutionAuthorization :one
INSERT INTO worm_execution_authorizations (
  id, run_id, owner_account_id, scope, proof_kind, session_jti_digest,
  access_revision, plan_version, plan_digest_sha256, state, authorized_at,
  created_at, updated_at
) SELECT
  sqlc.arg(id)::uuid, runs.id, runs.owner_account_id, 'WORM_POSITION_EXECUTE',
  sqlc.arg(proof_kind)::text, sqlc.arg(session_jti_digest)::bytea,
  sqlc.arg(access_revision)::bigint, runs.plan_version, runs.plan_digest_sha256,
  'AUTHORIZED', sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz
FROM worm_execution_runs AS runs
WHERE runs.id = sqlc.arg(run_id)::uuid
RETURNING worm_execution_authorizations.*;

-- name: AuthorizeExecutionRun :one
UPDATE worm_execution_runs
SET state = CASE WHEN state = 'AWAITING_AUTHORIZATION' THEN 'AUTHORIZED' ELSE state END,
    revision = revision + 1,
    authorized_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND revision = sqlc.arg(expected_revision)::bigint
  AND state IN (
    'AWAITING_AUTHORIZATION', 'AUTHORIZED', 'PAUSED', 'RECONCILIATION_REQUIRED'
  )
RETURNING *;

-- name: StartExecutionRun :one
UPDATE worm_execution_runs AS runs
SET state = 'RUNNING', revision = revision + 1,
    started_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now)::timestamptz
WHERE runs.id = sqlc.arg(id)::uuid
  AND runs.owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND runs.revision = sqlc.arg(expected_revision)::bigint
  AND runs.state = 'AUTHORIZED'
  AND EXISTS (
    SELECT 1
    FROM worm_execution_authorizations AS authorizations
    WHERE authorizations.run_id = runs.id
      AND authorizations.state = 'AUTHORIZED'
      AND authorizations.session_jti_digest = sqlc.arg(session_jti_digest)::bytea
      AND authorizations.access_revision = sqlc.arg(access_revision)::bigint
      AND authorizations.plan_version = runs.plan_version
      AND authorizations.plan_digest_sha256 = runs.plan_digest_sha256
  )
RETURNING runs.*;

-- name: RequestExecutionRunPause :one
UPDATE worm_execution_runs
SET state = CASE
      WHEN EXISTS (
        SELECT 1
        FROM worm_execution_run_steps
        WHERE run_id = worm_execution_runs.id
          AND state IN (
            'PREFLIGHTING', 'OPENING', 'OPENED', 'SIGNING', 'FINALIZING',
            'AWAITING_COMPLETION'
          )
      ) THEN 'PAUSE_REQUESTED'
      ELSE 'PAUSED'
    END,
    revision = revision + 1,
    pause_code = sqlc.arg(pause_code)::text,
    paused_at = CASE
      WHEN EXISTS (
        SELECT 1
        FROM worm_execution_run_steps
        WHERE run_id = worm_execution_runs.id
          AND state IN (
            'PREFLIGHTING', 'OPENING', 'OPENED', 'SIGNING', 'FINALIZING',
            'AWAITING_COMPLETION'
          )
      ) THEN NULL
      ELSE sqlc.arg(now)::timestamptz
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND revision = sqlc.arg(expected_revision)::bigint
  AND state = 'RUNNING'
RETURNING *;

-- name: PauseExecutionRunForReauthorization :one
UPDATE worm_execution_runs
SET state = CASE
      WHEN EXISTS (
        SELECT 1
        FROM worm_execution_run_steps
        WHERE run_id = worm_execution_runs.id
          AND state IN (
            'PREFLIGHTING', 'OPENING', 'OPENED', 'SIGNING', 'FINALIZING',
            'AWAITING_COMPLETION'
          )
      ) THEN 'PAUSE_REQUESTED'
      ELSE 'PAUSED'
    END,
    revision = revision + 1,
    pause_code = 'AUTHORIZATION_REQUIRED',
    paused_at = CASE
      WHEN EXISTS (
        SELECT 1
        FROM worm_execution_run_steps
        WHERE run_id = worm_execution_runs.id
          AND state IN (
            'PREFLIGHTING', 'OPENING', 'OPENED', 'SIGNING', 'FINALIZING',
            'AWAITING_COMPLETION'
          )
      ) THEN NULL
      ELSE sqlc.arg(now)::timestamptz
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state IN ('RUNNING', 'PAUSE_REQUESTED')
RETURNING *;

-- name: CheckpointExecutionRunPaused :one
UPDATE worm_execution_runs
SET state = 'PAUSED', revision = revision + 1,
    paused_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'PAUSE_REQUESTED'
RETURNING *;

-- name: ResumeExecutionRun :one
UPDATE worm_execution_runs AS runs
SET state = 'RUNNING', revision = revision + 1, pause_code = '', paused_at = NULL,
    updated_at = sqlc.arg(now)::timestamptz
WHERE runs.id = sqlc.arg(id)::uuid
  AND runs.owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND runs.revision = sqlc.arg(expected_revision)::bigint
  AND runs.state = 'PAUSED'
  AND EXISTS (
    SELECT 1
    FROM worm_execution_authorizations AS authorizations
    WHERE authorizations.run_id = runs.id
      AND authorizations.state = 'AUTHORIZED'
      AND authorizations.session_jti_digest = sqlc.arg(session_jti_digest)::bytea
      AND authorizations.access_revision = sqlc.arg(access_revision)::bigint
      AND authorizations.plan_version = runs.plan_version
      AND authorizations.plan_digest_sha256 = runs.plan_digest_sha256
  )
RETURNING runs.*;

-- name: RequestExecutionRunTermination :one
UPDATE worm_execution_runs
SET state = CASE
      WHEN started_at IS NULL OR NOT EXISTS (
        SELECT 1
        FROM worm_execution_run_steps
        WHERE run_id = worm_execution_runs.id
          AND state IN (
            'PREFLIGHTING', 'OPENING', 'OPENED', 'SIGNING', 'FINALIZING',
            'AWAITING_COMPLETION'
          )
      )
        THEN 'TERMINATED'
      ELSE 'TERMINATE_REQUESTED'
    END,
    revision = revision + 1,
    completed_at = CASE
      WHEN started_at IS NULL OR NOT EXISTS (
        SELECT 1
        FROM worm_execution_run_steps
        WHERE run_id = worm_execution_runs.id
          AND state IN (
            'PREFLIGHTING', 'OPENING', 'OPENED', 'SIGNING', 'FINALIZING',
            'AWAITING_COMPLETION'
          )
      )
        THEN sqlc.arg(now)::timestamptz
      ELSE NULL
    END,
    block_code = CASE
      WHEN state = 'RECONCILIATION_REQUIRED' THEN block_code
      ELSE ''
    END,
    pause_code = '', updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND revision = sqlc.arg(expected_revision)::bigint
  AND state IN (
    'AWAITING_AUTHORIZATION', 'AUTHORIZED', 'RUNNING', 'PAUSE_REQUESTED',
    'PAUSED', 'RECONCILIATION_REQUIRED'
  )
RETURNING *;

-- name: MarkPendingExecutionRunStepsNotExecuted :execrows
UPDATE worm_execution_run_steps
SET state = 'NOT_EXECUTED', reason_code = sqlc.arg(reason_code)::text,
    completed_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND state = 'PENDING';

-- name: CompleteExecutionRunTermination :one
UPDATE worm_execution_runs
SET state = 'TERMINATED', revision = revision + 1,
    completed_at = sqlc.arg(now)::timestamptz, pause_code = '',
    block_code = CASE
      WHEN EXISTS (
        SELECT 1 FROM worm_execution_step_isolations
        WHERE run_id = worm_execution_runs.id AND resolved_at IS NULL
      ) THEN COALESCE(NULLIF(block_code, ''), 'OUTCOME_UNKNOWN')
      ELSE ''
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'TERMINATE_REQUESTED'
RETURNING *;

-- name: RefreshExecutionRunProgress :one
WITH counts AS (
  SELECT
    COUNT(*) FILTER (WHERE state IN ('COMPLETED', 'SATISFIED', 'SKIPPED', 'FAILED', 'NOT_EXECUTED'))::bigint AS terminal_count,
    COUNT(*) FILTER (WHERE state = 'COMPLETED')::bigint AS completed_count,
    COUNT(*) FILTER (WHERE state = 'SATISFIED')::bigint AS satisfied_count,
    COUNT(*) FILTER (WHERE state = 'SKIPPED')::bigint AS skipped_count,
    COUNT(*) FILTER (WHERE state = 'FAILED')::bigint AS failed_count,
    COUNT(*) FILTER (WHERE state = 'NOT_EXECUTED')::bigint AS not_executed_count,
    MIN(ordinal) FILTER (WHERE state = 'PENDING')::bigint AS next_ordinal
  FROM worm_execution_run_steps
  WHERE run_id = sqlc.arg(id)::uuid
)
UPDATE worm_execution_runs AS runs
SET terminal_step_count = counts.terminal_count,
    completed_step_count = counts.completed_count,
    satisfied_step_count = counts.satisfied_count,
    skipped_step_count = counts.skipped_count,
    failed_step_count = counts.failed_count,
    not_executed_step_count = counts.not_executed_count,
    next_step_ordinal = counts.next_ordinal,
    updated_at = sqlc.arg(now)::timestamptz
FROM counts
WHERE runs.id = sqlc.arg(id)::uuid
RETURNING runs.*;

-- name: CompleteExecutionRun :one
UPDATE worm_execution_runs
SET state = 'COMPLETED', revision = revision + 1,
    current_step_ordinal = NULL, next_step_ordinal = NULL,
    completed_at = sqlc.arg(now)::timestamptz, pause_code = '', block_code = '',
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'RUNNING'
  AND terminal_step_count = total_step_count
RETURNING *;

-- name: CompleteReconciledExecutionRun :one
UPDATE worm_execution_runs
SET state = 'COMPLETED', revision = revision + 1,
    current_step_ordinal = NULL, next_step_ordinal = NULL,
    completed_at = sqlc.arg(now)::timestamptz, pause_code = '', block_code = '',
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'RECONCILIATION_REQUIRED'
  AND terminal_step_count = total_step_count
  AND NOT EXISTS (
    SELECT 1
    FROM worm_execution_step_isolations
    WHERE run_id = worm_execution_runs.id
      AND resolved_at IS NULL
  )
RETURNING *;

-- name: MarkExecutionRunReconciliationRequired :one
UPDATE worm_execution_runs
SET state = CASE
      WHEN state = 'TERMINATE_REQUESTED' THEN 'TERMINATED'
      ELSE 'RECONCILIATION_REQUIRED'
    END,
    revision = revision + 1,
    block_code = sqlc.arg(block_code)::text, pause_code = '',
    completed_at = CASE
      WHEN state = 'TERMINATE_REQUESTED' THEN sqlc.arg(now)::timestamptz
      ELSE completed_at
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state IN ('RUNNING', 'PAUSE_REQUESTED', 'TERMINATE_REQUESTED')
RETURNING *;

-- name: ResolveExecutionRunReconciliation :one
UPDATE worm_execution_runs
SET state = 'PAUSED', revision = revision + 1,
    block_code = '', pause_code = 'RECONCILED',
    paused_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'RECONCILIATION_REQUIRED'
  AND NOT EXISTS (
    SELECT 1
    FROM worm_execution_step_isolations
    WHERE run_id = worm_execution_runs.id
      AND resolved_at IS NULL
  )
RETURNING *;

-- name: MarkExecutionRunFailed :one
UPDATE worm_execution_runs
SET state = 'FAILED', revision = revision + 1,
    failure_code = sqlc.arg(failure_code)::text,
    completed_at = sqlc.arg(now)::timestamptz, pause_code = '', block_code = '',
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state NOT IN ('COMPLETED', 'TERMINATED', 'FAILED')
RETURNING *;

-- name: ReleaseExecutionRunLocks :exec
WITH deleted_wallets AS (
  DELETE FROM worm_execution_wallet_locks
  WHERE run_id = sqlc.arg(run_id)::uuid
)
DELETE FROM worm_execution_combination_locks
WHERE run_id = sqlc.arg(run_id)::uuid;

-- name: EndExecutionRunAuthorization :execrows
UPDATE worm_execution_authorizations
SET state = 'CONSUMED', ended_at = sqlc.arg(now)::timestamptz,
    end_reason_code = sqlc.arg(reason_code)::text,
    updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND state = 'AUTHORIZED';

-- name: ExpireExecutionCoordinator :execrows
UPDATE worm_execution_coordinators
SET state = 'EXPIRED', released_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND state = 'ACTIVE'
  AND lease_expires_at <= sqlc.arg(now)::timestamptz;

-- name: ReleaseExecutionCoordinator :execrows
UPDATE worm_execution_coordinators
SET state = 'RELEASED', released_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND state = 'ACTIVE';

-- name: CreateExecutionCoordinator :one
INSERT INTO worm_execution_coordinators (
  id, run_id, generation, token_sha256, session_jti_digest, access_revision,
  state, acquired_at, heartbeat_at, lease_expires_at, created_at, updated_at
)
SELECT sqlc.arg(id)::uuid, sqlc.arg(run_id)::uuid,
       COALESCE(MAX(generation), 0)::bigint + 1,
       sqlc.arg(token_sha256)::bytea, sqlc.arg(session_jti_digest)::bytea,
       sqlc.arg(access_revision)::bigint, 'ACTIVE',
       sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz,
       sqlc.arg(lease_expires_at)::timestamptz, sqlc.arg(now)::timestamptz,
       sqlc.arg(now)::timestamptz
FROM worm_execution_coordinators
WHERE run_id = sqlc.arg(run_id)::uuid
RETURNING *;

-- name: GetActiveExecutionCoordinator :one
SELECT *
FROM worm_execution_coordinators
WHERE run_id = sqlc.arg(run_id)::uuid
  AND state = 'ACTIVE';

-- name: HeartbeatExecutionCoordinator :one
UPDATE worm_execution_coordinators
SET heartbeat_at = sqlc.arg(now)::timestamptz,
    lease_expires_at = sqlc.arg(lease_expires_at)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND state = 'ACTIVE'
  AND token_sha256 = sqlc.arg(token_sha256)::bytea
  AND session_jti_digest = sqlc.arg(session_jti_digest)::bytea
  AND access_revision = sqlc.arg(access_revision)::bigint
  AND lease_expires_at > sqlc.arg(now)::timestamptz
RETURNING *;

-- name: GetExecutionCommand :one
SELECT *
FROM worm_execution_commands
WHERE id = sqlc.arg(id)::uuid;

-- name: CreateExecutionCommand :one
INSERT INTO worm_execution_commands (
  id, run_id, coordinator_id, coordinator_generation, kind, state,
  request_sha256, run_revision_before, step_ordinal, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid, sqlc.arg(run_id)::uuid, sqlc.narg(coordinator_id)::uuid,
  sqlc.narg(coordinator_generation)::bigint, sqlc.arg(kind)::text, 'ACCEPTED',
  sqlc.arg(request_sha256)::bytea, sqlc.arg(run_revision_before)::bigint,
  sqlc.narg(step_ordinal)::bigint, sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz
)
RETURNING *;

-- name: BeginExecutionCommand :one
UPDATE worm_execution_commands
SET state = 'IN_PROGRESS', updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'ACCEPTED'
RETURNING *;

-- name: BindExecutionCommandCoordinator :one
UPDATE worm_execution_commands
SET coordinator_id = sqlc.arg(coordinator_id)::uuid,
    coordinator_generation = sqlc.arg(coordinator_generation)::bigint,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND run_id = sqlc.arg(run_id)::uuid
  AND state = 'IN_PROGRESS'
  AND coordinator_id IS NULL
  AND EXISTS (
    SELECT 1
    FROM worm_execution_coordinators AS coordinators
    WHERE coordinators.id = sqlc.arg(coordinator_id)::uuid
      AND coordinators.run_id = worm_execution_commands.run_id
      AND coordinators.generation = sqlc.arg(coordinator_generation)::bigint
  )
RETURNING *;

-- name: CompleteExecutionCommand :one
UPDATE worm_execution_commands
SET state = sqlc.arg(state)::text,
    run_revision_after = sqlc.arg(run_revision_after)::bigint,
    result_code = sqlc.arg(result_code)::text,
    completed_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state IN ('ACCEPTED', 'IN_PROGRESS')
RETURNING *;

-- name: ClaimFreshExecutionRunStep :one
UPDATE worm_execution_run_steps AS steps
SET state = 'PREFLIGHTING', active_command_id = sqlc.arg(command_id)::uuid,
    claim_id = sqlc.arg(command_id)::uuid, claim_owner = 'COORDINATOR',
    claim_expires_at = sqlc.arg(claim_expires_at)::timestamptz,
    started_at = COALESCE(started_at, sqlc.arg(now)::timestamptz),
    updated_at = sqlc.arg(now)::timestamptz
FROM worm_execution_runs AS runs,
     worm_execution_coordinators AS coordinators,
     worm_execution_authorizations AS authorizations,
     worm_execution_commands AS commands
WHERE steps.run_id = sqlc.arg(run_id)::uuid
  AND steps.ordinal = sqlc.arg(step_ordinal)::bigint
  AND steps.state = 'PENDING'
  AND runs.id = steps.run_id
  AND runs.state = 'RUNNING'
  AND runs.revision = sqlc.arg(expected_run_revision)::bigint
  AND runs.next_step_ordinal = steps.ordinal
  AND coordinators.run_id = runs.id
  AND coordinators.state = 'ACTIVE'
  AND coordinators.token_sha256 = sqlc.arg(coordinator_token_sha256)::bytea
  AND coordinators.session_jti_digest = sqlc.arg(session_jti_digest)::bytea
  AND coordinators.access_revision = sqlc.arg(access_revision)::bigint
  AND coordinators.lease_expires_at > sqlc.arg(now)::timestamptz
  AND authorizations.run_id = runs.id
  AND authorizations.state = 'AUTHORIZED'
  AND authorizations.session_jti_digest = sqlc.arg(session_jti_digest)::bytea
  AND authorizations.access_revision = sqlc.arg(access_revision)::bigint
  AND authorizations.plan_version = runs.plan_version
  AND authorizations.plan_digest_sha256 = runs.plan_digest_sha256
  AND commands.id = sqlc.arg(command_id)::uuid
  AND commands.run_id = runs.id
  AND commands.kind = 'EXECUTE_NEXT'
  AND commands.state = 'IN_PROGRESS'
  AND commands.step_ordinal = steps.ordinal
  AND NOT EXISTS (
    SELECT 1
    FROM worm_execution_run_wallets AS wallets
    JOIN worm_execution_run_items AS items
      ON items.run_id = wallets.run_id
    JOIN worm_execution_step_isolations AS isolations
      ON isolations.wallet_id = wallets.wallet_id
     AND isolations.market_condition_id = items.market_condition_id
     AND isolations.resolved_at IS NULL
    WHERE wallets.run_id = steps.run_id
      AND wallets.ordinal = steps.wallet_ordinal
      AND items.ordinal = steps.item_ordinal
  )
RETURNING steps.*;

-- name: ClaimExecutionRunStepForRecovery :one
UPDATE worm_execution_run_steps AS steps
SET claim_id = sqlc.arg(claim_id)::uuid,
    claim_owner = sqlc.arg(claim_owner)::text,
    claim_expires_at = sqlc.arg(claim_expires_at)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
FROM worm_execution_runs AS runs
WHERE steps.run_id = sqlc.arg(run_id)::uuid
  AND steps.ordinal = sqlc.arg(step_ordinal)::bigint
  AND steps.state IN (
    'PREFLIGHTING', 'OPENING', 'OPENED', 'SIGNING', 'FINALIZING',
    'AWAITING_COMPLETION', 'OUTCOME_UNKNOWN'
  )
  AND (
    steps.claim_owner = 'COORDINATOR'
    OR steps.claim_id IS NULL
    OR steps.claim_expires_at <= sqlc.arg(now)::timestamptz
    OR (
      steps.claim_id = sqlc.arg(claim_id)::uuid
      AND steps.claim_owner = sqlc.arg(claim_owner)::text
    )
  )
  AND runs.id = steps.run_id
  AND runs.state IN (
    'RUNNING', 'PAUSE_REQUESTED', 'TERMINATE_REQUESTED',
    'RECONCILIATION_REQUIRED', 'TERMINATED'
  )
  AND runs.current_step_ordinal = steps.ordinal
  AND (steps.state <> 'OUTCOME_UNKNOWN' OR steps.reconcile_requested_at IS NOT NULL)
  AND (steps.state <> 'AWAITING_COMPLETION' OR steps.next_poll_at <= sqlc.arg(now)::timestamptz)
RETURNING steps.*;

-- name: ListRecoverableExecutionRunStepKeys :many
SELECT steps.run_id, steps.ordinal AS step_ordinal,
       runs.owner_account_id, runs.state AS run_state, runs.revision AS run_revision,
       steps.active_command_id AS command_id,
       commands.coordinator_id, commands.coordinator_generation
FROM worm_execution_run_steps AS steps
JOIN worm_execution_runs AS runs ON runs.id = steps.run_id
JOIN worm_execution_commands AS commands
  ON commands.id = steps.active_command_id
 AND commands.run_id = steps.run_id
 AND commands.step_ordinal = steps.ordinal
 AND commands.state = 'IN_PROGRESS'
WHERE steps.state IN (
    'PREFLIGHTING', 'OPENING', 'OPENED', 'SIGNING', 'FINALIZING',
    'AWAITING_COMPLETION', 'OUTCOME_UNKNOWN'
  )
  AND runs.state IN (
    'RUNNING', 'PAUSE_REQUESTED', 'TERMINATE_REQUESTED',
    'RECONCILIATION_REQUIRED', 'TERMINATED'
  )
  AND runs.current_step_ordinal = steps.ordinal
  AND (steps.state <> 'AWAITING_COMPLETION' OR steps.next_poll_at <= sqlc.arg(now)::timestamptz)
  AND (steps.state <> 'OUTCOME_UNKNOWN' OR steps.reconcile_requested_at IS NOT NULL)
  AND (
    steps.claim_owner = 'COORDINATOR'
    OR steps.claim_id IS NULL
    OR steps.claim_expires_at <= sqlc.arg(now)::timestamptz
  )
ORDER BY COALESCE(steps.next_poll_at, steps.claim_expires_at, steps.updated_at),
         steps.run_id, steps.ordinal
LIMIT sqlc.arg(recovery_limit)::integer;

-- name: RenewExecutionRunStepClaim :one
UPDATE worm_execution_run_steps
SET claim_expires_at = sqlc.arg(claim_expires_at)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND ordinal = sqlc.arg(step_ordinal)::bigint
  AND claim_id = sqlc.arg(claim_id)::uuid
  AND claim_owner = sqlc.arg(claim_owner)::text
  AND claim_expires_at > sqlc.arg(now)::timestamptz
  AND state IN (
    'PREFLIGHTING', 'OPENING', 'OPENED', 'SIGNING', 'FINALIZING',
    'AWAITING_COMPLETION', 'OUTCOME_UNKNOWN'
  )
RETURNING *;

-- name: CompleteExecutionStepPreflight :one
UPDATE worm_execution_run_steps
SET state = sqlc.arg(next_state)::text,
    reason_code = sqlc.arg(reason_code)::text,
    active_command_id = CASE
      WHEN sqlc.arg(next_state)::text = 'OPENING' THEN active_command_id
      ELSE NULL
    END,
    claim_id = CASE
      WHEN sqlc.arg(next_state)::text = 'OPENING' THEN claim_id
      ELSE NULL
    END,
    claim_owner = CASE
      WHEN sqlc.arg(next_state)::text = 'OPENING' THEN claim_owner
      ELSE ''
    END,
    claim_expires_at = CASE
      WHEN sqlc.arg(next_state)::text = 'OPENING' THEN claim_expires_at
      ELSE NULL
    END,
    completed_at = CASE
      WHEN sqlc.arg(next_state)::text IN ('SATISFIED', 'SKIPPED', 'FAILED')
        THEN sqlc.arg(now)::timestamptz
      ELSE NULL
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND ordinal = sqlc.arg(step_ordinal)::bigint
  AND state = 'PREFLIGHTING'
  AND claim_id = sqlc.arg(claim_id)::uuid
RETURNING *;

-- name: SkipScopedPendingExecutionSteps :execrows
UPDATE worm_execution_run_steps AS candidate
SET state = 'SKIPPED', reason_code = sqlc.arg(reason_code)::text,
    completed_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now)::timestamptz
FROM worm_execution_run_steps AS source,
     worm_execution_run_items AS source_item,
     worm_execution_run_items AS candidate_item
WHERE source.run_id = sqlc.arg(run_id)::uuid
  AND source.ordinal = sqlc.arg(source_step_ordinal)::bigint
  AND source_item.run_id = source.run_id
  AND source_item.ordinal = source.item_ordinal
  AND candidate.run_id = source.run_id
  AND candidate.ordinal > source.ordinal
  AND candidate.state = 'PENDING'
  AND candidate_item.run_id = candidate.run_id
  AND candidate_item.ordinal = candidate.item_ordinal
  AND (
    (sqlc.arg(scope)::text = 'REMAINING_WALLET'
      AND candidate.wallet_ordinal = source.wallet_ordinal)
    OR (sqlc.arg(scope)::text = 'REMAINING_MARKET'
      AND candidate_item.market_condition_id = source_item.market_condition_id)
  );

-- name: RecordExecutionRunStepOpened :one
UPDATE worm_execution_run_steps
SET state = 'OPENED', position_request_id = sqlc.arg(position_request_id)::bigint,
    transaction_message_sha256 = sqlc.arg(transaction_message_sha256)::bytea,
    provider_state = sqlc.arg(provider_state)::text,
    provider_order_state = sqlc.arg(provider_order_state)::text,
    opened_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND ordinal = sqlc.arg(step_ordinal)::bigint
  AND state = 'OPENING'
  AND claim_id = sqlc.arg(claim_id)::uuid
  AND EXISTS (
    SELECT 1
    FROM worm_execution_mutation_attempts
    WHERE run_id = worm_execution_run_steps.run_id
      AND step_ordinal = worm_execution_run_steps.ordinal
      AND kind = 'OPEN'
      AND state = 'SUCCEEDED'
      AND position_request_id = sqlc.arg(position_request_id)::bigint
  )
RETURNING *;

-- name: MarkExecutionRunStepSigning :one
UPDATE worm_execution_run_steps
SET state = 'SIGNING', updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND ordinal = sqlc.arg(step_ordinal)::bigint
  AND state = 'OPENED'
  AND claim_id = sqlc.arg(claim_id)::uuid
RETURNING *;

-- name: RecordExecutionRunStepSigned :one
UPDATE worm_execution_run_steps
SET state = 'FINALIZING', finalize_mode = sqlc.arg(finalize_mode)::text,
    transaction_version = sqlc.arg(transaction_version)::text,
    required_signature_count = sqlc.arg(required_signature_count)::integer,
    wallet_signer_index = sqlc.arg(wallet_signer_index)::integer,
    updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND ordinal = sqlc.arg(step_ordinal)::bigint
  AND state = 'SIGNING'
  AND claim_id = sqlc.arg(claim_id)::uuid
RETURNING *;

-- name: RecordExecutionRunProviderObservation :one
UPDATE worm_execution_run_steps
SET state = sqlc.arg(next_state)::text,
    reason_code = sqlc.arg(reason_code)::text,
    position_request_id = CASE
      WHEN sqlc.arg(position_request_id)::bigint > 0
        THEN sqlc.arg(position_request_id)::bigint
      ELSE position_request_id
    END,
    provider_state = sqlc.arg(provider_state)::text,
    provider_order_state = sqlc.arg(provider_order_state)::text,
    funding_txid = sqlc.arg(funding_txid)::text,
    refund_txid = sqlc.arg(refund_txid)::text,
    finalized_at = CASE
      WHEN state = 'FINALIZING' THEN COALESCE(finalized_at, sqlc.arg(now)::timestamptz)
      ELSE finalized_at
    END,
    last_observed_at = sqlc.arg(now)::timestamptz,
    next_poll_at = CASE
      WHEN sqlc.arg(next_state)::text = 'AWAITING_COMPLETION'
        THEN sqlc.arg(next_poll_at)::timestamptz
      ELSE NULL
    END,
    poll_count = poll_count + 1,
    active_command_id = CASE
      WHEN sqlc.arg(next_state)::text = 'AWAITING_COMPLETION' THEN active_command_id
      ELSE NULL
    END,
    claim_id = NULL,
    claim_owner = '',
    claim_expires_at = NULL,
    reconcile_requested_at = NULL,
    completed_at = CASE
      WHEN sqlc.arg(next_state)::text IN ('COMPLETED', 'SATISFIED', 'FAILED')
        THEN sqlc.arg(now)::timestamptz
      ELSE NULL
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND ordinal = sqlc.arg(step_ordinal)::bigint
  AND state = sqlc.arg(expected_state)::text
  AND claim_id = sqlc.arg(claim_id)::uuid
  AND (
    sqlc.arg(position_request_id)::bigint = 0
    OR (
      sqlc.arg(expected_state)::text = 'OPENING'
      AND (
        position_request_id IS NULL
        OR position_request_id = sqlc.arg(position_request_id)::bigint
      )
      AND EXISTS (
        SELECT 1
        FROM worm_execution_mutation_attempts AS request_attempts
        WHERE request_attempts.run_id = worm_execution_run_steps.run_id
          AND request_attempts.step_ordinal = worm_execution_run_steps.ordinal
          AND request_attempts.kind = 'OPEN'
          AND request_attempts.position_request_id = sqlc.arg(position_request_id)::bigint
          AND (
            (
              sqlc.arg(next_state)::text = 'OUTCOME_UNKNOWN'
              AND request_attempts.state = 'OUTCOME_UNKNOWN'
            )
            OR (
              sqlc.arg(next_state)::text = 'COMPLETED'
              AND request_attempts.state IN ('SUCCEEDED', 'DEFINITE_FAILURE')
            )
            OR (
              sqlc.arg(next_state)::text = 'FAILED'
              AND request_attempts.state IN ('SUCCEEDED', 'DEFINITE_FAILURE')
              AND lower(sqlc.arg(provider_state)::text) IN ('failed', 'cancelled', 'canceled')
            )
          )
      )
    )
  )
  AND (
    sqlc.arg(next_state)::text <> 'COMPLETED'
    OR lower(sqlc.arg(provider_state)::text) = 'completed'
  )
  AND (
    sqlc.arg(expected_state)::text IN (
      'OPENED', 'SIGNING', 'AWAITING_COMPLETION', 'OUTCOME_UNKNOWN'
    )
    OR (
      sqlc.arg(expected_state)::text = 'FINALIZING'
      AND sqlc.arg(next_state)::text IN ('COMPLETED', 'FAILED')
      AND (
        sqlc.arg(next_state)::text = 'COMPLETED'
        OR lower(sqlc.arg(provider_state)::text) IN ('failed', 'cancelled', 'canceled')
      )
      AND NOT EXISTS (
        SELECT 1
        FROM worm_execution_mutation_attempts AS undispatched_finalize
        WHERE undispatched_finalize.run_id = worm_execution_run_steps.run_id
          AND undispatched_finalize.step_ordinal = worm_execution_run_steps.ordinal
          AND undispatched_finalize.kind = 'FINALIZE'
          AND undispatched_finalize.state <> 'PREPARED'
      )
    )
    OR EXISTS (
      SELECT 1
      FROM worm_execution_mutation_attempts AS attempts
      WHERE attempts.run_id = worm_execution_run_steps.run_id
        AND attempts.step_ordinal = worm_execution_run_steps.ordinal
        AND attempts.kind = CASE
          WHEN sqlc.arg(expected_state)::text = 'OPENING' THEN 'OPEN'
          ELSE 'FINALIZE'
        END
        AND (
          (sqlc.arg(next_state)::text = 'FAILED' AND attempts.state = 'DEFINITE_FAILURE')
          OR (
            sqlc.arg(expected_state)::text = 'OPENING'
            AND sqlc.arg(next_state)::text = 'FAILED'
            AND attempts.state = 'SUCCEEDED'
            AND lower(sqlc.arg(provider_state)::text) IN ('failed', 'cancelled', 'canceled')
          )
          OR (sqlc.arg(next_state)::text = 'OUTCOME_UNKNOWN' AND attempts.state = 'OUTCOME_UNKNOWN')
          OR (
            sqlc.arg(next_state)::text IN ('AWAITING_COMPLETION', 'COMPLETED')
            AND attempts.state IN ('SUCCEEDED', 'OUTCOME_UNKNOWN')
          )
          OR (
            sqlc.arg(expected_state)::text = 'OPENING'
            AND sqlc.arg(next_state)::text = 'COMPLETED'
            AND attempts.state = 'DEFINITE_FAILURE'
          )
        )
    )
  )
RETURNING *;

-- name: RequestExecutionRunStepReconciliation :one
UPDATE worm_execution_run_steps
SET active_command_id = sqlc.arg(command_id)::uuid,
    reconcile_requested_at = sqlc.arg(now)::timestamptz,
    next_poll_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE run_id = sqlc.arg(run_id)::uuid
  AND id = sqlc.arg(step_id)::uuid
  AND ordinal = sqlc.arg(step_ordinal)::bigint
  AND state = 'OUTCOME_UNKNOWN'
  AND reconcile_requested_at IS NULL
  AND claim_id IS NULL
  AND EXISTS (
    SELECT 1
    FROM worm_execution_step_isolations
    WHERE run_id = worm_execution_run_steps.run_id
      AND step_ordinal = worm_execution_run_steps.ordinal
      AND resolved_at IS NULL
  )
RETURNING *;

-- name: PauseExecutionRunForFailure :one
UPDATE worm_execution_runs
SET state = CASE
      WHEN EXISTS (
        SELECT 1
        FROM worm_execution_run_steps
        WHERE run_id = worm_execution_runs.id
          AND state IN ('OPENING', 'OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION')
      ) THEN 'PAUSE_REQUESTED'
      ELSE 'PAUSED'
    END,
    revision = revision + 1,
    pause_code = sqlc.arg(pause_code)::text,
    paused_at = CASE
      WHEN EXISTS (
        SELECT 1
        FROM worm_execution_run_steps
        WHERE run_id = worm_execution_runs.id
          AND state IN ('OPENING', 'OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION')
      ) THEN NULL
      ELSE sqlc.arg(now)::timestamptz
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(run_id)::uuid
  AND state = 'RUNNING'
RETURNING *;

-- name: SetExecutionRunCurrentStep :one
UPDATE worm_execution_runs
SET current_step_ordinal = sqlc.narg(current_step_ordinal)::bigint,
    next_step_ordinal = NULL,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
RETURNING *;

-- name: TouchExecutionRunRevision :one
UPDATE worm_execution_runs
SET revision = revision + 1, updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
RETURNING *;

-- name: GetExecutionRunStepByID :one
SELECT *
FROM worm_execution_run_steps
WHERE run_id = sqlc.arg(run_id)::uuid
  AND id = sqlc.arg(id)::uuid;

-- name: GetExecutionMutationAttempt :one
SELECT *
FROM worm_execution_mutation_attempts
WHERE run_id = sqlc.arg(run_id)::uuid
  AND step_ordinal = sqlc.arg(step_ordinal)::bigint
  AND kind = sqlc.arg(kind)::text;

-- name: GetExecutionMutationAttemptByID :one
SELECT *
FROM worm_execution_mutation_attempts
WHERE id = sqlc.arg(id)::uuid;

-- name: CreateExecutionMutationAttempt :one
INSERT INTO worm_execution_mutation_attempts (
  id, run_id, step_ordinal, coordinator_id, coordinator_generation, command_id,
  kind, state, request_sha256, position_request_id, prepared_at, created_at,
  updated_at
) SELECT
  sqlc.arg(id)::uuid, steps.run_id, steps.ordinal, coordinators.id,
  coordinators.generation, commands.id, sqlc.arg(kind)::text, 'PREPARED',
  sqlc.arg(request_sha256)::bytea, sqlc.narg(position_request_id)::bigint,
  sqlc.arg(now)::timestamptz, sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz
FROM worm_execution_run_steps AS steps
JOIN worm_execution_commands AS commands
  ON commands.id = sqlc.arg(command_id)::uuid
 AND commands.run_id = steps.run_id
 AND commands.kind = 'EXECUTE_NEXT'
 AND commands.state = 'IN_PROGRESS'
 AND commands.id = steps.active_command_id
 AND commands.step_ordinal = steps.ordinal
JOIN worm_execution_coordinators AS coordinators
  ON coordinators.id = commands.coordinator_id
 AND coordinators.run_id = steps.run_id
 AND coordinators.generation = commands.coordinator_generation
WHERE steps.run_id = sqlc.arg(run_id)::uuid
  AND steps.ordinal = sqlc.arg(step_ordinal)::bigint
  AND steps.claim_id = sqlc.arg(claim_id)::uuid
  AND steps.claim_expires_at > sqlc.arg(now)::timestamptz
  AND (
    sqlc.arg(kind)::text <> 'FINALIZE'
    OR steps.position_request_id = sqlc.narg(position_request_id)::bigint
  )
  AND (
    (sqlc.arg(kind)::text = 'OPEN' AND steps.state = 'OPENING')
    OR (sqlc.arg(kind)::text = 'FINALIZE' AND steps.state = 'FINALIZING')
  )
RETURNING *;

-- name: DispatchExecutionMutationAttempt :one
UPDATE worm_execution_mutation_attempts AS attempts
SET state = 'DISPATCHED', dispatched_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
FROM worm_execution_run_steps AS steps
WHERE attempts.id = sqlc.arg(id)::uuid
  AND attempts.state = 'PREPARED'
  AND attempts.run_id = sqlc.arg(run_id)::uuid
  AND attempts.step_ordinal = sqlc.arg(step_ordinal)::bigint
  AND steps.run_id = attempts.run_id
  AND steps.ordinal = attempts.step_ordinal
  AND steps.active_command_id = attempts.command_id
  AND steps.claim_id = sqlc.arg(claim_id)::uuid
  AND steps.claim_expires_at > sqlc.arg(now)::timestamptz
  AND (
    (attempts.kind = 'OPEN' AND steps.state = 'OPENING')
    OR (attempts.kind = 'FINALIZE' AND steps.state = 'FINALIZING')
  )
RETURNING attempts.*;

-- name: CompleteExecutionMutationAttempt :one
UPDATE worm_execution_mutation_attempts
SET state = sqlc.arg(state)::text,
    position_request_id = COALESCE(position_request_id, sqlc.narg(position_request_id)::bigint),
    http_status = sqlc.narg(http_status)::integer,
    provider_code = sqlc.narg(provider_code)::integer,
    provider_slug = sqlc.arg(provider_slug)::text,
    error_code = sqlc.arg(error_code)::text,
    completed_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'DISPATCHED'
  AND sqlc.arg(state)::text IN ('SUCCEEDED', 'DEFINITE_FAILURE', 'OUTCOME_UNKNOWN')
  AND (
    position_request_id IS NULL
    OR sqlc.narg(position_request_id)::bigint IS NULL
    OR position_request_id = sqlc.narg(position_request_id)::bigint
  )
  AND (
    kind <> 'OPEN'
    OR sqlc.arg(state)::text <> 'SUCCEEDED'
    OR sqlc.narg(position_request_id)::bigint IS NOT NULL
  )
RETURNING *;

-- name: CreateExecutionStepIsolation :one
INSERT INTO worm_execution_step_isolations (
  id, owner_account_id, wallet_id, market_condition_id, run_id, step_ordinal,
  attempt_id, reason_code, created_at
)
SELECT sqlc.arg(id)::uuid, runs.owner_account_id, wallets.wallet_id,
       items.market_condition_id, steps.run_id, steps.ordinal,
       sqlc.arg(attempt_id)::uuid, sqlc.arg(reason_code)::text,
       sqlc.arg(now)::timestamptz
FROM worm_execution_run_steps AS steps
JOIN worm_execution_runs AS runs ON runs.id = steps.run_id
JOIN worm_execution_mutation_attempts AS attempts
  ON attempts.id = sqlc.arg(attempt_id)::uuid
 AND attempts.run_id = steps.run_id
 AND attempts.step_ordinal = steps.ordinal
 AND attempts.state IN ('SUCCEEDED', 'OUTCOME_UNKNOWN')
JOIN worm_execution_run_wallets AS wallets
  ON wallets.run_id = steps.run_id AND wallets.ordinal = steps.wallet_ordinal
JOIN worm_execution_run_items AS items
  ON items.run_id = steps.run_id AND items.ordinal = steps.item_ordinal
WHERE steps.run_id = sqlc.arg(run_id)::uuid
  AND steps.ordinal = sqlc.arg(step_ordinal)::bigint
  AND steps.state = 'OUTCOME_UNKNOWN'
RETURNING worm_execution_step_isolations.*;

-- name: ResolveExecutionStepIsolation :one
UPDATE worm_execution_step_isolations
SET resolved_at = sqlc.arg(now)::timestamptz,
    resolution_code = sqlc.arg(resolution_code)::text
WHERE id = sqlc.arg(id)::uuid
  AND run_id = sqlc.arg(run_id)::uuid
  AND step_ordinal = sqlc.arg(step_ordinal)::bigint
  AND resolved_at IS NULL
RETURNING *;
