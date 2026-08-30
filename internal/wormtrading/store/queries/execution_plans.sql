-- name: CreateExecutionPlan :one
INSERT INTO worm_execution_plans (
  id, owner_account_id, combination_id, combination_name, combination_revision,
  state, build_stage, wallet_count, item_count, total_step_count,
  skip_already_held, skip_in_flight_request, skip_opposite_side_exposure,
  require_full_liquidity,
  requested_at, retention_until, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid, sqlc.arg(owner_account_id)::uuid, sqlc.arg(combination_id)::uuid,
  sqlc.arg(combination_name)::text, sqlc.arg(combination_revision)::bigint,
  'BUILDING', 'QUEUED', sqlc.arg(wallet_count)::bigint, sqlc.arg(item_count)::bigint,
  sqlc.arg(total_step_count)::bigint, sqlc.arg(skip_already_held)::boolean,
  sqlc.arg(skip_in_flight_request)::boolean,
  sqlc.arg(skip_opposite_side_exposure)::boolean,
  sqlc.arg(require_full_liquidity)::boolean, sqlc.arg(now)::timestamptz,
  sqlc.arg(retention_until)::timestamptz, sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz
)
RETURNING *;

-- name: CreateExecutionPlanWallet :exec
INSERT INTO worm_execution_plan_wallets (
  plan_id, ordinal, wallet_id, address, remark, avatar_kind, avatar_preset_id, avatar_url
) VALUES (
  sqlc.arg(plan_id)::uuid, sqlc.arg(ordinal)::integer, sqlc.arg(wallet_id)::bigint,
  sqlc.arg(address)::text, sqlc.arg(remark)::text, sqlc.arg(avatar_kind)::text,
  sqlc.arg(avatar_preset_id)::text, sqlc.arg(avatar_url)::text
);

-- name: CreateExecutionPlanItem :exec
INSERT INTO worm_execution_plan_items (
  plan_id, ordinal, event_condition_id, event_title, event_logo, market_condition_id,
  market_title, market_logo, is_yes, outcome_label
) VALUES (
  sqlc.arg(plan_id)::uuid, sqlc.arg(ordinal)::integer,
  sqlc.arg(event_condition_id)::text, sqlc.arg(event_title)::text, sqlc.arg(event_logo)::text,
  sqlc.arg(market_condition_id)::text, sqlc.arg(market_title)::text,
  sqlc.arg(market_logo)::text, sqlc.arg(is_yes)::boolean, sqlc.arg(outcome_label)::text
);

-- name: GetExecutionPlan :one
SELECT *
FROM worm_execution_plans
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid;

-- name: GetExecutionPlanForUpdate :one
SELECT *
FROM worm_execution_plans
WHERE id = sqlc.arg(id)::uuid
FOR UPDATE;

-- name: GetExecutionPlanForOwnerUpdate :one
SELECT *
FROM worm_execution_plans
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
FOR UPDATE;

-- name: ListExecutionPlanWallets :many
SELECT *
FROM worm_execution_plan_wallets
WHERE plan_id = sqlc.arg(plan_id)::uuid
ORDER BY ordinal;

-- name: ListExecutionPlanItems :many
SELECT *
FROM worm_execution_plan_items
WHERE plan_id = sqlc.arg(plan_id)::uuid
ORDER BY ordinal;

-- name: CountExecutionPlanSteps :one
SELECT COUNT(*)::bigint
FROM worm_execution_plan_steps AS steps
JOIN worm_execution_plans AS plans ON plans.id = steps.plan_id
WHERE steps.plan_id = sqlc.arg(plan_id)::uuid
  AND plans.state = 'READY';

-- name: ListExecutionPlanSteps :many
SELECT steps.*
FROM worm_execution_plan_steps AS steps
JOIN worm_execution_plans AS plans ON plans.id = steps.plan_id
WHERE steps.plan_id = sqlc.arg(plan_id)::uuid
  AND plans.state = 'READY'
  AND steps.ordinal > sqlc.arg(page_offset)::bigint
ORDER BY steps.ordinal
LIMIT sqlc.arg(page_size)::integer;

-- name: ListExecutionPlanReasonCounts :many
SELECT COALESCE(NULLIF(reason_code, ''), disposition)::text AS reason_code,
       COUNT(*)::bigint AS count
FROM worm_execution_plan_steps
WHERE plan_id = sqlc.arg(plan_id)::uuid
GROUP BY COALESCE(NULLIF(reason_code, ''), disposition)
ORDER BY reason_code;

-- name: ListExecutionPlanAdvisoryCounts :many
SELECT advisory_code::text AS reason_code, COUNT(*)::bigint AS count
FROM worm_execution_plan_steps AS steps
CROSS JOIN LATERAL unnest(steps.advisory_codes) AS advisories(advisory_code)
WHERE steps.plan_id = sqlc.arg(plan_id)::uuid
GROUP BY advisory_code
ORDER BY advisory_code;

-- name: ClaimNextExecutionPlan :one
WITH candidate AS (
  SELECT id
  FROM worm_execution_plans
  WHERE state = 'BUILDING'
    AND (lease_expires_at IS NULL OR lease_expires_at <= sqlc.arg(now)::timestamptz)
  ORDER BY requested_at, id
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
UPDATE worm_execution_plans AS plans
SET worker_id = sqlc.arg(worker_id)::text,
    locked_at = sqlc.arg(now)::timestamptz,
    lease_expires_at = sqlc.arg(lease_expires_at)::timestamptz,
    build_stage = 'CLAIMED',
    updated_at = sqlc.arg(now)::timestamptz
FROM candidate
WHERE plans.id = candidate.id
RETURNING plans.*;

-- name: UpdateExecutionPlanBuildProgress :one
UPDATE worm_execution_plans
SET build_stage = sqlc.arg(build_stage)::text,
    completed_step_count = sqlc.arg(completed_step_count)::bigint,
    worker_id = sqlc.arg(worker_id)::text,
    lease_expires_at = sqlc.arg(lease_expires_at)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'BUILDING'
  AND worker_id = sqlc.arg(worker_id)::text
  AND lease_expires_at > sqlc.arg(now)::timestamptz
RETURNING *;

-- name: UpdateExecutionPlanWalletObservation :exec
UPDATE worm_execution_plan_wallets
SET connection_state = sqlc.arg(connection_state)::text,
    connection_warning_code = sqlc.arg(connection_warning_code)::text,
    connected_at = sqlc.arg(connected_at)::timestamptz,
    credential_version = sqlc.arg(credential_version)::bigint,
    sol_atomic_amount = sqlc.arg(sol_atomic_amount)::text,
    sol_amount = sqlc.arg(sol_amount)::text,
    sol_decimals = sqlc.arg(sol_decimals)::integer,
    sol_observed_slot = sqlc.arg(sol_observed_slot)::bigint,
    sol_availability = sqlc.arg(sol_availability)::text,
    sol_error_code = sqlc.arg(sol_error_code)::text,
    usdc_mint = sqlc.arg(usdc_mint)::text,
    usdc_atomic_amount = sqlc.arg(usdc_atomic_amount)::text,
    usdc_amount = sqlc.arg(usdc_amount)::text,
    usdc_decimals = sqlc.arg(usdc_decimals)::integer,
    usdc_observed_slot = sqlc.arg(usdc_observed_slot)::bigint,
    usdc_availability = sqlc.arg(usdc_availability)::text,
    usdc_error_code = sqlc.arg(usdc_error_code)::text,
    usdc_token_account_count = sqlc.arg(usdc_token_account_count)::integer,
    status = sqlc.arg(status)::text,
    reason_code = sqlc.arg(reason_code)::text
WHERE plan_id = sqlc.arg(plan_id)::uuid
  AND ordinal = sqlc.arg(ordinal)::integer;

-- name: UpdateExecutionPlanItemObservation :exec
UPDATE worm_execution_plan_items
SET backend = sqlc.arg(backend)::text,
    funds = sqlc.arg(funds)::text,
    leverage = sqlc.arg(leverage)::text,
    state = sqlc.arg(state)::text,
    reason_code = sqlc.arg(reason_code)::text,
    estimate_average_price = sqlc.arg(estimate_average_price)::text,
    estimate_total_shares = sqlc.arg(estimate_total_shares)::text,
    estimate_total_cost = sqlc.arg(estimate_total_cost)::text,
    estimate_best_ask = sqlc.arg(estimate_best_ask)::text,
    estimate_worst_fill_price = sqlc.arg(estimate_worst_fill_price)::text,
    estimate_is_fully_filled = sqlc.arg(estimate_is_fully_filled)::boolean,
    estimate_fee_amount = sqlc.arg(estimate_fee_amount)::text,
    estimate_user_funds_needed = sqlc.arg(estimate_user_funds_needed)::text,
    estimate_liquidation_price = sqlc.arg(estimate_liquidation_price)::text
WHERE plan_id = sqlc.arg(plan_id)::uuid
  AND ordinal = sqlc.arg(ordinal)::integer;

-- name: DeleteExecutionPlanSteps :exec
DELETE FROM worm_execution_plan_steps
WHERE plan_id = sqlc.arg(plan_id)::uuid;

-- name: MarkExecutionPlanReady :one
UPDATE worm_execution_plans
SET state = 'READY',
    build_stage = '',
    failure_code = '',
    worker_id = '',
    locked_at = NULL,
    lease_expires_at = NULL,
    completed_step_count = sqlc.arg(completed_step_count)::bigint,
    ready_step_count = sqlc.arg(ready_step_count)::bigint,
    skipped_step_count = sqlc.arg(skipped_step_count)::bigint,
    total_collateral = sqlc.arg(total_collateral)::text,
    total_opening_fee = sqlc.arg(total_opening_fee)::text,
    total_user_funds_needed = sqlc.arg(total_user_funds_needed)::text,
    completed_at = sqlc.arg(completed_at)::timestamptz,
    expires_at = sqlc.arg(expires_at)::timestamptz,
    retention_until = sqlc.arg(retention_until)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'BUILDING'
  AND worker_id = sqlc.arg(worker_id)::text
  AND lease_expires_at > sqlc.arg(now)::timestamptz
RETURNING *;

-- name: MarkExecutionPlanFailed :one
UPDATE worm_execution_plans
SET state = 'FAILED',
    build_stage = '',
    failure_code = sqlc.arg(failure_code)::text,
    worker_id = '',
    locked_at = NULL,
    lease_expires_at = NULL,
    completed_at = sqlc.arg(completed_at)::timestamptz,
    retention_until = sqlc.arg(retention_until)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND state = 'BUILDING'
  AND worker_id = sqlc.arg(worker_id)::text
  AND lease_expires_at > sqlc.arg(now)::timestamptz
RETURNING *;

-- name: DeleteExpiredExecutionPlans :many
DELETE FROM worm_execution_plans
WHERE id IN (
  SELECT id
  FROM worm_execution_plans
  WHERE state IN ('READY', 'FAILED')
    AND retention_until <= sqlc.arg(now)::timestamptz
    AND NOT EXISTS (
      SELECT 1
      FROM worm_execution_runs
      WHERE worm_execution_runs.plan_id = worm_execution_plans.id
    )
  ORDER BY retention_until, id
  LIMIT sqlc.arg(cleanup_limit)::integer
  FOR UPDATE SKIP LOCKED
)
RETURNING id;
