-- +goose Up

CREATE TABLE worm_execution_runs (
  id UUID PRIMARY KEY,
  owner_account_id UUID NOT NULL,
  plan_id UUID NOT NULL
    REFERENCES worm_execution_plans(id) ON DELETE RESTRICT,
  plan_version BIGINT NOT NULL CHECK (plan_version > 0),
  plan_digest_sha256 BYTEA NOT NULL CHECK (octet_length(plan_digest_sha256) = 32),
  idempotency_key_sha256 BYTEA NOT NULL CHECK (octet_length(idempotency_key_sha256) = 32),
  request_sha256 BYTEA NOT NULL CHECK (octet_length(request_sha256) = 32),
  combination_id UUID NOT NULL,
  combination_name TEXT NOT NULL CHECK (
    combination_name = btrim(combination_name)
    AND char_length(combination_name) BETWEEN 1 AND 80
  ),
  combination_revision BIGINT NOT NULL CHECK (combination_revision > 0),
  state TEXT NOT NULL CHECK (
    state IN (
      'AWAITING_AUTHORIZATION',
      'AUTHORIZED',
      'RUNNING',
      'PAUSE_REQUESTED',
      'PAUSED',
      'TERMINATE_REQUESTED',
      'RECONCILIATION_REQUIRED',
      'COMPLETED',
      'TERMINATED',
      'FAILED'
    )
  ),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  current_step_ordinal BIGINT CHECK (current_step_ordinal > 0),
  next_step_ordinal BIGINT CHECK (next_step_ordinal > 0),
  wallet_count BIGINT NOT NULL CHECK (wallet_count > 0),
  item_count BIGINT NOT NULL CHECK (item_count > 0),
  total_step_count BIGINT NOT NULL CHECK (total_step_count > 0),
  actionable_step_count BIGINT NOT NULL CHECK (actionable_step_count > 0),
  terminal_step_count BIGINT NOT NULL DEFAULT 0 CHECK (terminal_step_count >= 0),
  completed_step_count BIGINT NOT NULL DEFAULT 0 CHECK (completed_step_count >= 0),
  satisfied_step_count BIGINT NOT NULL DEFAULT 0 CHECK (satisfied_step_count >= 0),
  skipped_step_count BIGINT NOT NULL DEFAULT 0 CHECK (skipped_step_count >= 0),
  failed_step_count BIGINT NOT NULL DEFAULT 0 CHECK (failed_step_count >= 0),
  not_executed_step_count BIGINT NOT NULL DEFAULT 0 CHECK (not_executed_step_count >= 0),
  pause_code TEXT NOT NULL DEFAULT '' CHECK (
    pause_code = btrim(pause_code) AND char_length(pause_code) <= 100
  ),
  failure_code TEXT NOT NULL DEFAULT '' CHECK (
    failure_code = btrim(failure_code) AND char_length(failure_code) <= 100
  ),
  block_code TEXT NOT NULL DEFAULT '' CHECK (
    block_code = btrim(block_code) AND char_length(block_code) <= 100
  ),
  requested_at TIMESTAMPTZ NOT NULL,
  authorized_at TIMESTAMPTZ,
  started_at TIMESTAMPTZ,
  paused_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT worm_execution_runs_plan_unique UNIQUE (plan_id),
  CONSTRAINT worm_execution_runs_owner_idempotency_unique
    UNIQUE (owner_account_id, idempotency_key_sha256),
  CHECK (actionable_step_count <= total_step_count),
  CHECK (terminal_step_count <= total_step_count),
  CHECK (
    completed_step_count + satisfied_step_count + skipped_step_count
      + failed_step_count + not_executed_step_count = terminal_step_count
  ),
  CHECK (
    (state = 'AWAITING_AUTHORIZATION' AND authorized_at IS NULL AND started_at IS NULL)
    OR (state = 'AUTHORIZED' AND authorized_at IS NOT NULL AND started_at IS NULL)
    OR (
      state IN (
        'RUNNING',
        'PAUSE_REQUESTED',
        'PAUSED',
        'TERMINATE_REQUESTED',
        'RECONCILIATION_REQUIRED',
        'COMPLETED',
        'FAILED'
      )
      AND authorized_at IS NOT NULL
      AND started_at IS NOT NULL
    )
    OR state = 'TERMINATED'
  ),
  CHECK (
    (state IN ('COMPLETED', 'TERMINATED', 'FAILED') AND completed_at IS NOT NULL)
    OR (state NOT IN ('COMPLETED', 'TERMINATED', 'FAILED') AND completed_at IS NULL)
  ),
  CHECK ((state = 'PAUSED' AND paused_at IS NOT NULL) OR state <> 'PAUSED'),
  CHECK ((state = 'FAILED' AND failure_code <> '') OR (state <> 'FAILED' AND failure_code = '')),
  CHECK (
    (state = 'RECONCILIATION_REQUIRED' AND block_code <> '')
    OR (state = 'TERMINATED')
    OR (state NOT IN ('RECONCILIATION_REQUIRED', 'TERMINATED') AND block_code = '')
  )
);

CREATE UNIQUE INDEX worm_execution_runs_one_nonterminal_per_owner_idx
  ON worm_execution_runs (owner_account_id)
  WHERE state IN (
    'AWAITING_AUTHORIZATION',
    'AUTHORIZED',
    'RUNNING',
    'PAUSE_REQUESTED',
    'PAUSED',
    'TERMINATE_REQUESTED',
    'RECONCILIATION_REQUIRED'
  );

CREATE INDEX worm_execution_runs_owner_requested_idx
  ON worm_execution_runs (owner_account_id, requested_at DESC, id DESC);

CREATE INDEX worm_execution_runs_state_updated_idx
  ON worm_execution_runs (state, updated_at, id);

CREATE TABLE worm_execution_run_wallets (
  run_id UUID NOT NULL REFERENCES worm_execution_runs(id) ON DELETE RESTRICT,
  ordinal INTEGER NOT NULL CHECK (ordinal > 0),
  wallet_id BIGINT NOT NULL CHECK (wallet_id > 0),
  address TEXT NOT NULL CHECK (
    address = btrim(address) AND char_length(address) BETWEEN 32 AND 64
  ),
  remark TEXT NOT NULL DEFAULT '' CHECK (
    remark = btrim(remark) AND char_length(remark) <= 50
  ),
  avatar_kind TEXT NOT NULL DEFAULT '' CHECK (
    avatar_kind = btrim(avatar_kind) AND char_length(avatar_kind) <= 40
  ),
  avatar_preset_id TEXT NOT NULL DEFAULT '' CHECK (
    avatar_preset_id = btrim(avatar_preset_id) AND char_length(avatar_preset_id) <= 100
  ),
  avatar_url TEXT NOT NULL DEFAULT '' CHECK (
    avatar_url = btrim(avatar_url) AND char_length(avatar_url) <= 4096
  ),
  connection_state TEXT NOT NULL CHECK (
    connection_state = btrim(connection_state) AND char_length(connection_state) <= 100
  ),
  connection_warning_code TEXT NOT NULL DEFAULT '' CHECK (
    connection_warning_code = btrim(connection_warning_code)
    AND char_length(connection_warning_code) <= 100
  ),
  connected_at TIMESTAMPTZ,
  credential_version BIGINT NOT NULL CHECK (credential_version >= 0),
  sol_atomic_amount TEXT NOT NULL DEFAULT '',
  sol_amount TEXT NOT NULL DEFAULT '',
  sol_decimals INTEGER NOT NULL CHECK (sol_decimals >= 0),
  sol_observed_slot BIGINT NOT NULL CHECK (sol_observed_slot >= 0),
  sol_availability TEXT NOT NULL CHECK (
    sol_availability = btrim(sol_availability) AND char_length(sol_availability) <= 100
  ),
  sol_error_code TEXT NOT NULL DEFAULT '' CHECK (
    sol_error_code = btrim(sol_error_code) AND char_length(sol_error_code) <= 100
  ),
  usdc_mint TEXT NOT NULL CHECK (
    usdc_mint = btrim(usdc_mint) AND char_length(usdc_mint) <= 64
  ),
  usdc_atomic_amount TEXT NOT NULL,
  usdc_amount TEXT NOT NULL,
  usdc_decimals INTEGER NOT NULL CHECK (usdc_decimals >= 0),
  usdc_observed_slot BIGINT NOT NULL CHECK (usdc_observed_slot >= 0),
  usdc_availability TEXT NOT NULL CHECK (
    usdc_availability = btrim(usdc_availability) AND char_length(usdc_availability) <= 100
  ),
  usdc_error_code TEXT NOT NULL DEFAULT '' CHECK (
    usdc_error_code = btrim(usdc_error_code) AND char_length(usdc_error_code) <= 100
  ),
  usdc_token_account_count INTEGER NOT NULL CHECK (usdc_token_account_count >= 0),
  status TEXT NOT NULL CHECK (
    status = btrim(status) AND char_length(status) <= 100
  ),
  reason_code TEXT NOT NULL DEFAULT '' CHECK (
    reason_code = btrim(reason_code) AND char_length(reason_code) <= 100
  ),
  PRIMARY KEY (run_id, ordinal),
  CONSTRAINT worm_execution_run_wallets_wallet_unique UNIQUE (run_id, wallet_id),
  CONSTRAINT worm_execution_run_wallets_address_unique UNIQUE (run_id, address)
);

CREATE TABLE worm_execution_run_items (
  run_id UUID NOT NULL REFERENCES worm_execution_runs(id) ON DELETE RESTRICT,
  ordinal INTEGER NOT NULL CHECK (ordinal > 0),
  event_condition_id TEXT NOT NULL CHECK (
    event_condition_id = btrim(event_condition_id)
    AND char_length(event_condition_id) BETWEEN 32 AND 64
  ),
  event_title TEXT NOT NULL CHECK (
    event_title = btrim(event_title) AND char_length(event_title) BETWEEN 1 AND 500
  ),
  event_logo TEXT NOT NULL DEFAULT '' CHECK (
    event_logo = btrim(event_logo) AND char_length(event_logo) <= 2048
  ),
  market_condition_id TEXT NOT NULL CHECK (
    market_condition_id = btrim(market_condition_id)
    AND char_length(market_condition_id) BETWEEN 32 AND 64
  ),
  market_title TEXT NOT NULL CHECK (
    market_title = btrim(market_title) AND char_length(market_title) BETWEEN 1 AND 500
  ),
  market_logo TEXT NOT NULL DEFAULT '' CHECK (
    market_logo = btrim(market_logo) AND char_length(market_logo) <= 2048
  ),
  is_yes BOOLEAN NOT NULL,
  outcome_label TEXT NOT NULL CHECK (
    outcome_label = btrim(outcome_label) AND char_length(outcome_label) BETWEEN 1 AND 100
  ),
  backend TEXT NOT NULL CHECK (
    backend = btrim(backend) AND char_length(backend) BETWEEN 1 AND 100
  ),
  funds TEXT NOT NULL,
  leverage TEXT NOT NULL,
  preview_state TEXT NOT NULL CHECK (
    preview_state = btrim(preview_state) AND char_length(preview_state) BETWEEN 1 AND 100
  ),
  preview_reason_code TEXT NOT NULL DEFAULT '' CHECK (
    preview_reason_code = btrim(preview_reason_code)
    AND char_length(preview_reason_code) <= 100
  ),
  estimate_average_price TEXT NOT NULL DEFAULT '',
  estimate_total_shares TEXT NOT NULL DEFAULT '',
  estimate_total_cost TEXT NOT NULL DEFAULT '',
  estimate_best_ask TEXT NOT NULL DEFAULT '',
  estimate_worst_fill_price TEXT NOT NULL DEFAULT '',
  estimate_is_fully_filled BOOLEAN NOT NULL,
  estimate_fee_amount TEXT NOT NULL DEFAULT '',
  estimate_user_funds_needed TEXT NOT NULL DEFAULT '',
  estimate_liquidation_price TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (run_id, ordinal),
  CONSTRAINT worm_execution_run_items_market_unique UNIQUE (run_id, market_condition_id)
);

CREATE TABLE worm_execution_run_steps (
  id UUID NOT NULL DEFAULT gen_random_uuid(),
  run_id UUID NOT NULL REFERENCES worm_execution_runs(id) ON DELETE RESTRICT,
  ordinal BIGINT NOT NULL CHECK (ordinal > 0),
  plan_step_ordinal BIGINT NOT NULL CHECK (plan_step_ordinal > 0),
  wallet_ordinal INTEGER NOT NULL CHECK (wallet_ordinal > 0),
  item_ordinal INTEGER NOT NULL CHECK (item_ordinal > 0),
  source_disposition TEXT NOT NULL CHECK (source_disposition IN ('READY', 'SKIPPED')),
  source_reason_code TEXT NOT NULL DEFAULT '' CHECK (
    source_reason_code = btrim(source_reason_code) AND char_length(source_reason_code) <= 100
  ),
  projected_usdc_before TEXT NOT NULL,
  projected_usdc_after TEXT NOT NULL,
  state TEXT NOT NULL CHECK (
    state IN (
      'PENDING',
      'PREFLIGHTING',
      'OPENING',
      'OPENED',
      'SIGNING',
      'FINALIZING',
      'AWAITING_COMPLETION',
      'COMPLETED',
      'SATISFIED',
      'SKIPPED',
      'FAILED',
      'NOT_EXECUTED',
      'OUTCOME_UNKNOWN'
    )
  ),
  reason_code TEXT NOT NULL DEFAULT '' CHECK (
    reason_code = btrim(reason_code) AND char_length(reason_code) <= 100
  ),
  position_request_id BIGINT CHECK (position_request_id > 0),
  finalize_mode TEXT NOT NULL DEFAULT '' CHECK (
    finalize_mode IN ('', 'signature', 'signed_transaction')
  ),
  transaction_message_sha256 BYTEA CHECK (
    transaction_message_sha256 IS NULL OR octet_length(transaction_message_sha256) = 32
  ),
  transaction_version TEXT NOT NULL DEFAULT '' CHECK (
    transaction_version = btrim(transaction_version)
    AND char_length(transaction_version) <= 20
  ),
  required_signature_count INTEGER NOT NULL DEFAULT 0 CHECK (required_signature_count >= 0),
  wallet_signer_index INTEGER NOT NULL DEFAULT -1 CHECK (wallet_signer_index >= -1),
  provider_state TEXT NOT NULL DEFAULT '' CHECK (
    provider_state = btrim(provider_state) AND char_length(provider_state) <= 100
  ),
  provider_order_state TEXT NOT NULL DEFAULT '' CHECK (
    provider_order_state = btrim(provider_order_state)
    AND char_length(provider_order_state) <= 100
  ),
  funding_txid TEXT NOT NULL DEFAULT '' CHECK (
    funding_txid = btrim(funding_txid) AND char_length(funding_txid) <= 200
  ),
  refund_txid TEXT NOT NULL DEFAULT '' CHECK (
    refund_txid = btrim(refund_txid) AND char_length(refund_txid) <= 200
  ),
  active_command_id UUID,
  claim_id UUID,
  claim_owner TEXT NOT NULL DEFAULT '' CHECK (
    claim_owner = btrim(claim_owner) AND char_length(claim_owner) <= 200
  ),
  claim_expires_at TIMESTAMPTZ,
  next_poll_at TIMESTAMPTZ,
  reconcile_requested_at TIMESTAMPTZ,
  poll_count INTEGER NOT NULL DEFAULT 0 CHECK (poll_count >= 0),
  started_at TIMESTAMPTZ,
  opened_at TIMESTAMPTZ,
  finalized_at TIMESTAMPTZ,
  last_observed_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (run_id, ordinal),
  CONSTRAINT worm_execution_run_steps_id_unique UNIQUE (id),
  CONSTRAINT worm_execution_run_steps_plan_step_unique UNIQUE (run_id, plan_step_ordinal),
  CONSTRAINT worm_execution_run_steps_wallet_item_unique UNIQUE (run_id, wallet_ordinal, item_ordinal),
  CONSTRAINT worm_execution_run_steps_wallet_fk
    FOREIGN KEY (run_id, wallet_ordinal) REFERENCES worm_execution_run_wallets(run_id, ordinal),
  CONSTRAINT worm_execution_run_steps_item_fk
    FOREIGN KEY (run_id, item_ordinal) REFERENCES worm_execution_run_items(run_id, ordinal),
  CHECK (
    (source_disposition = 'READY' AND source_reason_code = '')
    OR (source_disposition = 'SKIPPED' AND source_reason_code <> '')
  ),
  CHECK (
    (state IN ('COMPLETED', 'SATISFIED', 'SKIPPED', 'FAILED', 'NOT_EXECUTED') AND completed_at IS NOT NULL)
    OR (state NOT IN ('COMPLETED', 'SATISFIED', 'SKIPPED', 'FAILED', 'NOT_EXECUTED') AND completed_at IS NULL)
  ),
  CHECK (
    (state IN ('OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION', 'COMPLETED') AND position_request_id IS NOT NULL)
    OR state NOT IN ('OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION', 'COMPLETED')
  ),
  CHECK (
    (state = 'COMPLETED' AND lower(provider_state) = 'completed')
    OR state <> 'COMPLETED'
  ),
  CHECK (
    (claim_id IS NULL AND claim_owner = '' AND claim_expires_at IS NULL)
    OR (claim_id IS NOT NULL AND claim_owner <> '' AND claim_expires_at IS NOT NULL)
  ),
  CHECK (
    (state IN ('OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION', 'COMPLETED')
      AND transaction_message_sha256 IS NOT NULL)
    OR state NOT IN ('OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION', 'COMPLETED')
  ),
  CHECK (
    (state IN ('FINALIZING', 'AWAITING_COMPLETION')
      AND finalize_mode <> ''
      AND transaction_version <> ''
      AND required_signature_count > 0
      AND wallet_signer_index >= 0
      AND wallet_signer_index < required_signature_count)
    OR (
      state = 'COMPLETED'
      AND (
        (
          finalize_mode <> ''
          AND transaction_version <> ''
          AND required_signature_count > 0
          AND wallet_signer_index >= 0
          AND wallet_signer_index < required_signature_count
        )
        OR (
          finalize_mode = ''
          AND transaction_version = ''
          AND required_signature_count = 0
          AND wallet_signer_index = -1
        )
      )
    )
    OR state NOT IN ('FINALIZING', 'AWAITING_COMPLETION', 'COMPLETED')
  )
);

CREATE UNIQUE INDEX worm_execution_run_steps_position_request_idx
  ON worm_execution_run_steps (run_id, wallet_ordinal, position_request_id)
  WHERE position_request_id IS NOT NULL;

CREATE INDEX worm_execution_run_steps_page_idx
  ON worm_execution_run_steps (run_id, ordinal);

CREATE INDEX worm_execution_run_steps_state_idx
  ON worm_execution_run_steps (run_id, state, ordinal);

CREATE INDEX worm_execution_run_steps_recovery_idx
  ON worm_execution_run_steps (next_poll_at, claim_expires_at, run_id, ordinal)
  WHERE state IN (
    'PREFLIGHTING', 'OPENING', 'OPENED', 'SIGNING', 'FINALIZING',
    'AWAITING_COMPLETION', 'OUTCOME_UNKNOWN'
  );

CREATE TABLE worm_execution_authorizations (
  id UUID PRIMARY KEY,
  run_id UUID NOT NULL REFERENCES worm_execution_runs(id) ON DELETE RESTRICT,
  owner_account_id UUID NOT NULL,
  scope TEXT NOT NULL CHECK (scope = 'WORM_POSITION_EXECUTE'),
  proof_kind TEXT NOT NULL CHECK (
    proof_kind = btrim(proof_kind) AND char_length(proof_kind) BETWEEN 1 AND 100
  ),
  session_jti_digest BYTEA NOT NULL CHECK (octet_length(session_jti_digest) = 32),
  access_revision BIGINT NOT NULL CHECK (access_revision > 0),
  plan_version BIGINT NOT NULL CHECK (plan_version > 0),
  plan_digest_sha256 BYTEA NOT NULL CHECK (octet_length(plan_digest_sha256) = 32),
  state TEXT NOT NULL CHECK (
    state IN ('AUTHORIZED', 'CONSUMED', 'REVOKED', 'SUPERSEDED')
  ),
  authorized_at TIMESTAMPTZ NOT NULL,
  ended_at TIMESTAMPTZ,
  end_reason_code TEXT NOT NULL DEFAULT '' CHECK (
    end_reason_code = btrim(end_reason_code) AND char_length(end_reason_code) <= 100
  ),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CHECK (
    (state = 'AUTHORIZED' AND ended_at IS NULL AND end_reason_code = '')
    OR (state <> 'AUTHORIZED' AND ended_at IS NOT NULL AND end_reason_code <> '')
  )
);

CREATE UNIQUE INDEX worm_execution_authorizations_one_active_idx
  ON worm_execution_authorizations (run_id)
  WHERE state = 'AUTHORIZED';

CREATE TABLE worm_execution_coordinators (
  id UUID PRIMARY KEY,
  run_id UUID NOT NULL REFERENCES worm_execution_runs(id) ON DELETE RESTRICT,
  generation BIGINT NOT NULL CHECK (generation > 0),
  token_sha256 BYTEA NOT NULL CHECK (octet_length(token_sha256) = 32),
  session_jti_digest BYTEA NOT NULL CHECK (octet_length(session_jti_digest) = 32),
  access_revision BIGINT NOT NULL CHECK (access_revision > 0),
  state TEXT NOT NULL CHECK (state IN ('ACTIVE', 'RELEASED', 'EXPIRED')),
  acquired_at TIMESTAMPTZ NOT NULL,
  heartbeat_at TIMESTAMPTZ NOT NULL,
  lease_expires_at TIMESTAMPTZ NOT NULL,
  released_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT worm_execution_coordinators_generation_unique UNIQUE (run_id, generation),
  CONSTRAINT worm_execution_coordinators_binding_unique UNIQUE (run_id, id, generation),
  CHECK (lease_expires_at > acquired_at),
  CHECK (
    (state = 'ACTIVE' AND released_at IS NULL)
    OR (state <> 'ACTIVE' AND released_at IS NOT NULL)
  )
);

CREATE UNIQUE INDEX worm_execution_coordinators_one_active_idx
  ON worm_execution_coordinators (run_id)
  WHERE state = 'ACTIVE';

CREATE INDEX worm_execution_coordinators_expiry_idx
  ON worm_execution_coordinators (lease_expires_at, run_id)
  WHERE state = 'ACTIVE';

CREATE TABLE worm_execution_commands (
  id UUID PRIMARY KEY,
  run_id UUID NOT NULL REFERENCES worm_execution_runs(id) ON DELETE RESTRICT,
  coordinator_id UUID,
  coordinator_generation BIGINT CHECK (coordinator_generation > 0),
  kind TEXT NOT NULL CHECK (
    kind IN (
      'AUTHORIZE',
      'START',
      'PAUSE',
      'RESUME',
      'TERMINATE',
      'HEARTBEAT',
      'EXECUTE_NEXT',
      'RECONCILE'
    )
  ),
  state TEXT NOT NULL CHECK (
    state IN ('ACCEPTED', 'IN_PROGRESS', 'APPLIED', 'FAILED', 'ABANDONED')
  ),
  request_sha256 BYTEA NOT NULL CHECK (octet_length(request_sha256) = 32),
  run_revision_before BIGINT NOT NULL CHECK (run_revision_before > 0),
  run_revision_after BIGINT CHECK (run_revision_after > 0),
  step_ordinal BIGINT CHECK (step_ordinal > 0),
  result_code TEXT NOT NULL DEFAULT '' CHECK (
    result_code = btrim(result_code) AND char_length(result_code) <= 100
  ),
  created_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT worm_execution_commands_run_id_unique UNIQUE (run_id, id),
  CONSTRAINT worm_execution_commands_coordinator_fk
    FOREIGN KEY (run_id, coordinator_id, coordinator_generation)
    REFERENCES worm_execution_coordinators(run_id, id, generation) ON DELETE RESTRICT,
  CHECK (
    (coordinator_id IS NULL AND coordinator_generation IS NULL)
    OR (coordinator_id IS NOT NULL AND coordinator_generation IS NOT NULL)
  ),
  CHECK (
    (state IN ('APPLIED', 'FAILED', 'ABANDONED') AND completed_at IS NOT NULL)
    OR (state IN ('ACCEPTED', 'IN_PROGRESS') AND completed_at IS NULL)
  )
);

CREATE INDEX worm_execution_commands_run_created_idx
  ON worm_execution_commands (run_id, created_at, id);

ALTER TABLE worm_execution_run_steps
  ADD CONSTRAINT worm_execution_run_steps_active_command_fk
  FOREIGN KEY (run_id, active_command_id)
  REFERENCES worm_execution_commands(run_id, id) ON DELETE RESTRICT;

CREATE TABLE worm_execution_mutation_attempts (
  id UUID PRIMARY KEY,
  run_id UUID NOT NULL,
  step_ordinal BIGINT NOT NULL,
  coordinator_id UUID NOT NULL,
  coordinator_generation BIGINT NOT NULL CHECK (coordinator_generation > 0),
  command_id UUID NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('OPEN', 'FINALIZE')),
  state TEXT NOT NULL CHECK (
    state IN ('PREPARED', 'DISPATCHED', 'SUCCEEDED', 'DEFINITE_FAILURE', 'OUTCOME_UNKNOWN')
  ),
  request_sha256 BYTEA NOT NULL CHECK (octet_length(request_sha256) = 32),
  position_request_id BIGINT CHECK (position_request_id > 0),
  http_status INTEGER CHECK (http_status BETWEEN 100 AND 599),
  provider_code INTEGER,
  provider_slug TEXT NOT NULL DEFAULT '' CHECK (
    provider_slug = btrim(provider_slug) AND char_length(provider_slug) <= 100
  ),
  error_code TEXT NOT NULL DEFAULT '' CHECK (
    error_code = btrim(error_code) AND char_length(error_code) <= 100
  ),
  prepared_at TIMESTAMPTZ NOT NULL,
  dispatched_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT worm_execution_mutation_attempts_step_fk
    FOREIGN KEY (run_id, step_ordinal) REFERENCES worm_execution_run_steps(run_id, ordinal),
  CONSTRAINT worm_execution_mutation_attempts_coordinator_fk
    FOREIGN KEY (run_id, coordinator_id, coordinator_generation)
    REFERENCES worm_execution_coordinators(run_id, id, generation) ON DELETE RESTRICT,
  CONSTRAINT worm_execution_mutation_attempts_command_fk
    FOREIGN KEY (run_id, command_id)
    REFERENCES worm_execution_commands(run_id, id) ON DELETE RESTRICT,
  CONSTRAINT worm_execution_mutation_attempts_one_kind_per_step
    UNIQUE (run_id, step_ordinal, kind),
  CHECK (
    (state = 'PREPARED' AND dispatched_at IS NULL AND completed_at IS NULL)
    OR (state = 'DISPATCHED' AND dispatched_at IS NOT NULL AND completed_at IS NULL)
    OR (state IN ('SUCCEEDED', 'DEFINITE_FAILURE', 'OUTCOME_UNKNOWN')
      AND dispatched_at IS NOT NULL AND completed_at IS NOT NULL)
  ),
  CHECK ((kind = 'FINALIZE' AND position_request_id IS NOT NULL) OR kind = 'OPEN'),
  CHECK (kind <> 'OPEN' OR state <> 'SUCCEEDED' OR position_request_id IS NOT NULL)
);

CREATE INDEX worm_execution_mutation_attempts_run_step_idx
  ON worm_execution_mutation_attempts (run_id, step_ordinal, kind);

CREATE TABLE worm_execution_combination_locks (
  combination_id UUID PRIMARY KEY
    REFERENCES worm_market_combinations(id) ON DELETE RESTRICT,
  run_id UUID NOT NULL UNIQUE REFERENCES worm_execution_runs(id) ON DELETE RESTRICT,
  combination_revision BIGINT NOT NULL CHECK (combination_revision > 0),
  acquired_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE worm_execution_wallet_locks (
  wallet_id BIGINT PRIMARY KEY
    REFERENCES worm_wallet_connections(wallet_id) ON DELETE RESTRICT,
  run_id UUID NOT NULL REFERENCES worm_execution_runs(id) ON DELETE RESTRICT,
  address TEXT NOT NULL CHECK (
    address = btrim(address) AND char_length(address) BETWEEN 32 AND 64
  ),
  acquired_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT worm_execution_wallet_locks_run_wallet_unique UNIQUE (run_id, wallet_id)
);

CREATE TABLE worm_execution_step_isolations (
  id UUID PRIMARY KEY,
  owner_account_id UUID NOT NULL,
  wallet_id BIGINT NOT NULL CHECK (wallet_id > 0),
  market_condition_id TEXT NOT NULL CHECK (
    market_condition_id = btrim(market_condition_id)
    AND char_length(market_condition_id) BETWEEN 32 AND 64
  ),
  run_id UUID NOT NULL REFERENCES worm_execution_runs(id) ON DELETE RESTRICT,
  step_ordinal BIGINT NOT NULL,
  attempt_id UUID NOT NULL REFERENCES worm_execution_mutation_attempts(id) ON DELETE RESTRICT,
  reason_code TEXT NOT NULL CHECK (
    reason_code = btrim(reason_code) AND char_length(reason_code) BETWEEN 1 AND 100
  ),
  created_at TIMESTAMPTZ NOT NULL,
  resolved_at TIMESTAMPTZ,
  resolution_code TEXT NOT NULL DEFAULT '' CHECK (
    resolution_code = btrim(resolution_code) AND char_length(resolution_code) <= 100
  ),
  CONSTRAINT worm_execution_step_isolations_step_fk
    FOREIGN KEY (run_id, step_ordinal) REFERENCES worm_execution_run_steps(run_id, ordinal),
  CHECK (
    (resolved_at IS NULL AND resolution_code = '')
    OR (resolved_at IS NOT NULL AND resolution_code <> '')
  )
);

CREATE UNIQUE INDEX worm_execution_step_isolations_active_idx
  ON worm_execution_step_isolations (wallet_id, market_condition_id)
  WHERE resolved_at IS NULL;

-- +goose Down

DROP TABLE IF EXISTS worm_execution_step_isolations;
DROP TABLE IF EXISTS worm_execution_wallet_locks;
DROP TABLE IF EXISTS worm_execution_combination_locks;
DROP TABLE IF EXISTS worm_execution_mutation_attempts;
DROP TABLE IF EXISTS worm_execution_run_steps;
DROP TABLE IF EXISTS worm_execution_commands;
DROP TABLE IF EXISTS worm_execution_coordinators;
DROP TABLE IF EXISTS worm_execution_authorizations;
DROP TABLE IF EXISTS worm_execution_run_items;
DROP TABLE IF EXISTS worm_execution_run_wallets;
DROP TABLE IF EXISTS worm_execution_runs;
