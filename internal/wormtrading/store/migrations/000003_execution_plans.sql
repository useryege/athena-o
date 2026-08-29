-- +goose Up

CREATE TABLE worm_execution_plans (
  id UUID PRIMARY KEY,
  owner_account_id UUID NOT NULL,
  combination_id UUID NOT NULL,
  combination_name TEXT NOT NULL CHECK (
    combination_name = btrim(combination_name)
    AND char_length(combination_name) BETWEEN 1 AND 80
  ),
  combination_revision BIGINT NOT NULL CHECK (combination_revision > 0),
  state TEXT NOT NULL CHECK (state IN ('BUILDING', 'READY', 'FAILED')),
  build_stage TEXT NOT NULL DEFAULT '' CHECK (
    build_stage = btrim(build_stage)
    AND char_length(build_stage) <= 100
  ),
  failure_code TEXT NOT NULL DEFAULT '' CHECK (
    failure_code = btrim(failure_code)
    AND char_length(failure_code) <= 100
  ),
  worker_id TEXT NOT NULL DEFAULT '' CHECK (
    worker_id = btrim(worker_id)
    AND char_length(worker_id) <= 200
  ),
  locked_at TIMESTAMPTZ,
  lease_expires_at TIMESTAMPTZ,
  wallet_count BIGINT NOT NULL CHECK (wallet_count > 0),
  item_count BIGINT NOT NULL CHECK (item_count > 0),
  total_step_count BIGINT NOT NULL CHECK (total_step_count > 0),
  completed_step_count BIGINT NOT NULL DEFAULT 0 CHECK (completed_step_count >= 0),
  ready_step_count BIGINT NOT NULL DEFAULT 0 CHECK (ready_step_count >= 0),
  skipped_step_count BIGINT NOT NULL DEFAULT 0 CHECK (skipped_step_count >= 0),
  total_collateral TEXT NOT NULL DEFAULT '0',
  total_opening_fee TEXT NOT NULL DEFAULT '0',
  total_user_funds_needed TEXT NOT NULL DEFAULT '0',
  requested_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,
  retention_until TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CHECK (completed_step_count <= total_step_count),
  CHECK (ready_step_count + skipped_step_count <= completed_step_count),
  CHECK (retention_until >= created_at),
  CHECK (
    (worker_id = '' AND locked_at IS NULL AND lease_expires_at IS NULL)
    OR (worker_id <> '' AND locked_at IS NOT NULL AND lease_expires_at IS NOT NULL)
  ),
  CHECK (state = 'BUILDING' OR worker_id = ''),
  CHECK (
    (state = 'FAILED' AND failure_code <> '')
    OR (state <> 'FAILED' AND failure_code = '')
  ),
  CHECK (
    (state = 'BUILDING' AND completed_at IS NULL AND expires_at IS NULL)
    OR (
      state = 'READY'
      AND completed_at IS NOT NULL
      AND expires_at IS NOT NULL
      AND completed_step_count = total_step_count
      AND ready_step_count + skipped_step_count = total_step_count
    )
    OR (state = 'FAILED' AND completed_at IS NOT NULL)
  )
);

CREATE INDEX worm_execution_plans_owner_requested_idx
  ON worm_execution_plans (owner_account_id, requested_at DESC, id DESC);

CREATE INDEX worm_execution_plans_claim_idx
  ON worm_execution_plans (state, lease_expires_at, requested_at, id)
  WHERE state = 'BUILDING';

CREATE INDEX worm_execution_plans_retention_idx
  ON worm_execution_plans (retention_until, id);

CREATE TABLE worm_execution_plan_wallets (
  plan_id UUID NOT NULL REFERENCES worm_execution_plans(id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL CHECK (ordinal > 0),
  wallet_id BIGINT NOT NULL CHECK (wallet_id > 0),
  address TEXT NOT NULL CHECK (address = btrim(address) AND char_length(address) BETWEEN 1 AND 64),
  remark TEXT NOT NULL DEFAULT '' CHECK (remark = btrim(remark) AND char_length(remark) <= 50),
  avatar_kind TEXT NOT NULL DEFAULT '' CHECK (avatar_kind = btrim(avatar_kind) AND char_length(avatar_kind) <= 40),
  avatar_preset_id TEXT NOT NULL DEFAULT '' CHECK (avatar_preset_id = btrim(avatar_preset_id) AND char_length(avatar_preset_id) <= 100),
  avatar_url TEXT NOT NULL DEFAULT '' CHECK (avatar_url = btrim(avatar_url) AND char_length(avatar_url) <= 4096),
  connection_state TEXT NOT NULL DEFAULT '' CHECK (connection_state = btrim(connection_state) AND char_length(connection_state) <= 100),
  connection_warning_code TEXT NOT NULL DEFAULT '' CHECK (connection_warning_code = btrim(connection_warning_code) AND char_length(connection_warning_code) <= 100),
  connected_at TIMESTAMPTZ,
  credential_version BIGINT NOT NULL DEFAULT 0 CHECK (credential_version >= 0),
  sol_atomic_amount TEXT NOT NULL DEFAULT '',
  sol_amount TEXT NOT NULL DEFAULT '',
  sol_decimals INTEGER NOT NULL DEFAULT 0 CHECK (sol_decimals >= 0),
  sol_observed_slot BIGINT NOT NULL DEFAULT 0 CHECK (sol_observed_slot >= 0),
  sol_availability TEXT NOT NULL DEFAULT '' CHECK (sol_availability = btrim(sol_availability) AND char_length(sol_availability) <= 100),
  sol_error_code TEXT NOT NULL DEFAULT '' CHECK (sol_error_code = btrim(sol_error_code) AND char_length(sol_error_code) <= 100),
  usdc_mint TEXT NOT NULL DEFAULT '' CHECK (usdc_mint = btrim(usdc_mint) AND char_length(usdc_mint) <= 64),
  usdc_atomic_amount TEXT NOT NULL DEFAULT '',
  usdc_amount TEXT NOT NULL DEFAULT '',
  usdc_decimals INTEGER NOT NULL DEFAULT 0 CHECK (usdc_decimals >= 0),
  usdc_observed_slot BIGINT NOT NULL DEFAULT 0 CHECK (usdc_observed_slot >= 0),
  usdc_availability TEXT NOT NULL DEFAULT '' CHECK (usdc_availability = btrim(usdc_availability) AND char_length(usdc_availability) <= 100),
  usdc_error_code TEXT NOT NULL DEFAULT '' CHECK (usdc_error_code = btrim(usdc_error_code) AND char_length(usdc_error_code) <= 100),
  usdc_token_account_count INTEGER NOT NULL DEFAULT 0 CHECK (usdc_token_account_count >= 0),
  status TEXT NOT NULL DEFAULT '' CHECK (status = btrim(status) AND char_length(status) <= 100),
  reason_code TEXT NOT NULL DEFAULT '' CHECK (reason_code = btrim(reason_code) AND char_length(reason_code) <= 100),
  PRIMARY KEY (plan_id, ordinal),
  CONSTRAINT worm_execution_plan_wallets_wallet_unique UNIQUE (plan_id, wallet_id),
  CONSTRAINT worm_execution_plan_wallets_address_unique UNIQUE (plan_id, address)
);

CREATE TABLE worm_execution_plan_items (
  plan_id UUID NOT NULL REFERENCES worm_execution_plans(id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL CHECK (ordinal > 0),
  event_condition_id TEXT NOT NULL CHECK (event_condition_id = btrim(event_condition_id) AND char_length(event_condition_id) BETWEEN 32 AND 64),
  event_title TEXT NOT NULL CHECK (event_title = btrim(event_title) AND char_length(event_title) BETWEEN 1 AND 500),
  event_logo TEXT NOT NULL DEFAULT '' CHECK (event_logo = btrim(event_logo) AND char_length(event_logo) <= 2048),
  market_condition_id TEXT NOT NULL CHECK (market_condition_id = btrim(market_condition_id) AND char_length(market_condition_id) BETWEEN 32 AND 64),
  market_title TEXT NOT NULL CHECK (market_title = btrim(market_title) AND char_length(market_title) BETWEEN 1 AND 500),
  market_logo TEXT NOT NULL DEFAULT '' CHECK (market_logo = btrim(market_logo) AND char_length(market_logo) <= 2048),
  is_yes BOOLEAN NOT NULL,
  outcome_label TEXT NOT NULL CHECK (outcome_label = btrim(outcome_label) AND char_length(outcome_label) BETWEEN 1 AND 100),
  backend TEXT NOT NULL DEFAULT '' CHECK (backend = btrim(backend) AND char_length(backend) <= 100),
  funds TEXT NOT NULL DEFAULT '',
  leverage TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT '' CHECK (state = btrim(state) AND char_length(state) <= 100),
  reason_code TEXT NOT NULL DEFAULT '' CHECK (reason_code = btrim(reason_code) AND char_length(reason_code) <= 100),
  estimate_average_price TEXT NOT NULL DEFAULT '',
  estimate_total_shares TEXT NOT NULL DEFAULT '',
  estimate_total_cost TEXT NOT NULL DEFAULT '',
  estimate_best_ask TEXT NOT NULL DEFAULT '',
  estimate_worst_fill_price TEXT NOT NULL DEFAULT '',
  estimate_is_fully_filled BOOLEAN NOT NULL DEFAULT FALSE,
  estimate_fee_amount TEXT NOT NULL DEFAULT '',
  estimate_user_funds_needed TEXT NOT NULL DEFAULT '',
  estimate_liquidation_price TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (plan_id, ordinal),
  CONSTRAINT worm_execution_plan_items_market_unique UNIQUE (plan_id, market_condition_id)
);

CREATE TABLE worm_execution_plan_steps (
  plan_id UUID NOT NULL REFERENCES worm_execution_plans(id) ON DELETE CASCADE,
  ordinal BIGINT NOT NULL CHECK (ordinal > 0),
  wallet_ordinal INTEGER NOT NULL CHECK (wallet_ordinal > 0),
  item_ordinal INTEGER NOT NULL CHECK (item_ordinal > 0),
  disposition TEXT NOT NULL CHECK (disposition IN ('READY', 'SKIPPED')),
  reason_code TEXT NOT NULL DEFAULT '' CHECK (reason_code = btrim(reason_code) AND char_length(reason_code) <= 100),
  projected_usdc_before TEXT NOT NULL DEFAULT '',
  projected_usdc_after TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (plan_id, ordinal),
  CONSTRAINT worm_execution_plan_steps_wallet_item_unique UNIQUE (plan_id, wallet_ordinal, item_ordinal),
  CONSTRAINT worm_execution_plan_steps_wallet_fk
    FOREIGN KEY (plan_id, wallet_ordinal) REFERENCES worm_execution_plan_wallets(plan_id, ordinal),
  CONSTRAINT worm_execution_plan_steps_item_fk
    FOREIGN KEY (plan_id, item_ordinal) REFERENCES worm_execution_plan_items(plan_id, ordinal)
);

CREATE INDEX worm_execution_plan_steps_page_idx
  ON worm_execution_plan_steps (plan_id, ordinal);

-- +goose Down

DROP TABLE IF EXISTS worm_execution_plan_steps;
DROP TABLE IF EXISTS worm_execution_plan_items;
DROP TABLE IF EXISTS worm_execution_plan_wallets;
DROP TABLE IF EXISTS worm_execution_plans;
