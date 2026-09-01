-- +goose Up

CREATE TABLE worm_position_cash_out_batches (
  id UUID PRIMARY KEY,
  owner_account_id UUID NOT NULL,
  selection_digest_sha256 BYTEA NOT NULL CHECK (
    octet_length(selection_digest_sha256) = 32
  ),
  intent_digest_sha256 BYTEA CHECK (
    intent_digest_sha256 IS NULL OR octet_length(intent_digest_sha256) = 32
  ),
  state TEXT NOT NULL CHECK (
    state IN (
      'BUILDING', 'AWAITING_AUTHORIZATION', 'QUEUED', 'RUNNING',
      'PAUSE_REQUESTED', 'PAUSED', 'TERMINATE_REQUESTED',
      'COMPLETED', 'FAILED', 'RECONCILIATION_REQUIRED',
      'TERMINATED', 'CANCELLED', 'EXPIRED'
    )
  ),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  reason_code TEXT NOT NULL DEFAULT '' CHECK (
    reason_code = btrim(reason_code) AND char_length(reason_code) <= 100
  ),
  build_stage TEXT NOT NULL DEFAULT 'WALLETS' CHECK (
    build_stage = btrim(build_stage) AND char_length(build_stage) <= 100
  ),
  wallet_count BIGINT NOT NULL CHECK (wallet_count BETWEEN 1 AND 100),
  position_count BIGINT NOT NULL DEFAULT 0 CHECK (
    position_count BETWEEN 0 AND 1000
  ),
  completed_count BIGINT NOT NULL DEFAULT 0 CHECK (completed_count >= 0),
  not_executed_count BIGINT NOT NULL DEFAULT 0 CHECK (not_executed_count >= 0),
  next_item_ordinal BIGINT CHECK (next_item_ordinal > 0),
  current_item_ordinal BIGINT CHECK (current_item_ordinal > 0),
  authorization_expires_at TIMESTAMPTZ NOT NULL,
  authorized_at TIMESTAMPTZ,
  execution_started_at TIMESTAMPTZ,
  next_poll_at TIMESTAMPTZ,
  check_requested_at TIMESTAMPTZ,
  claim_id UUID,
  claim_owner TEXT NOT NULL DEFAULT '' CHECK (
    claim_owner = btrim(claim_owner) AND char_length(claim_owner) <= 100
  ),
  claim_expires_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CHECK (authorization_expires_at > created_at),
  CHECK (completed_count + not_executed_count <= position_count),
  CHECK (
    (claim_id IS NULL AND claim_owner = '' AND claim_expires_at IS NULL)
    OR (claim_id IS NOT NULL AND claim_owner <> '' AND claim_expires_at IS NOT NULL)
  ),
  CHECK (
    (state = 'BUILDING' AND intent_digest_sha256 IS NULL AND authorized_at IS NULL)
    OR (state IN ('AWAITING_AUTHORIZATION', 'QUEUED', 'RUNNING',
                  'PAUSE_REQUESTED', 'PAUSED', 'TERMINATE_REQUESTED',
                  'COMPLETED', 'RECONCILIATION_REQUIRED', 'TERMINATED')
      AND intent_digest_sha256 IS NOT NULL)
    OR state IN ('FAILED', 'CANCELLED', 'EXPIRED')
  ),
  CHECK (
    (state IN ('QUEUED', 'RUNNING', 'PAUSE_REQUESTED', 'PAUSED',
               'TERMINATE_REQUESTED', 'COMPLETED',
               'RECONCILIATION_REQUIRED', 'TERMINATED')
      AND authorized_at IS NOT NULL)
    OR (state IN ('BUILDING', 'AWAITING_AUTHORIZATION', 'CANCELLED', 'EXPIRED')
      AND authorized_at IS NULL)
    OR state = 'FAILED'
  ),
  CHECK (
    (state IN ('COMPLETED', 'FAILED', 'TERMINATED', 'CANCELLED', 'EXPIRED')
      AND completed_at IS NOT NULL AND claim_id IS NULL)
    OR (state NOT IN ('COMPLETED', 'FAILED', 'TERMINATED', 'CANCELLED', 'EXPIRED')
      AND completed_at IS NULL)
  ),
  CHECK (
    (state IN ('FAILED', 'RECONCILIATION_REQUIRED', 'EXPIRED') AND reason_code <> '')
    OR state NOT IN ('FAILED', 'RECONCILIATION_REQUIRED', 'EXPIRED')
  )
);

CREATE UNIQUE INDEX worm_position_cash_out_batches_one_active_owner_idx
  ON worm_position_cash_out_batches (owner_account_id)
  WHERE state IN (
    'BUILDING', 'AWAITING_AUTHORIZATION', 'QUEUED', 'RUNNING',
    'PAUSE_REQUESTED', 'PAUSED', 'TERMINATE_REQUESTED',
    'RECONCILIATION_REQUIRED'
  );

CREATE INDEX worm_position_cash_out_batches_owner_created_idx
  ON worm_position_cash_out_batches (owner_account_id, created_at DESC, id DESC);

CREATE INDEX worm_position_cash_out_batches_recovery_idx
  ON worm_position_cash_out_batches (
    COALESCE(next_poll_at, updated_at), claim_expires_at, created_at, id
  )
  WHERE state IN (
    'BUILDING', 'QUEUED', 'RUNNING', 'PAUSE_REQUESTED',
    'TERMINATE_REQUESTED', 'RECONCILIATION_REQUIRED'
  );

CREATE TABLE worm_position_cash_out_batch_wallets (
  batch_id UUID NOT NULL
    REFERENCES worm_position_cash_out_batches(id) ON DELETE RESTRICT,
  ordinal INTEGER NOT NULL CHECK (ordinal > 0),
  wallet_id BIGINT NOT NULL
    REFERENCES worm_wallet_connections(wallet_id) ON DELETE RESTRICT
    CHECK (wallet_id > 0),
  address TEXT NOT NULL CHECK (
    address = btrim(address) AND char_length(address) BETWEEN 32 AND 64
  ),
  credential_version BIGINT NOT NULL CHECK (credential_version > 0),
  remark TEXT NOT NULL DEFAULT '' CHECK (remark = btrim(remark)),
  avatar_kind TEXT NOT NULL DEFAULT '' CHECK (
    avatar_kind = btrim(avatar_kind) AND char_length(avatar_kind) <= 100
  ),
  avatar_preset_id TEXT NOT NULL DEFAULT '' CHECK (
    avatar_preset_id = btrim(avatar_preset_id) AND char_length(avatar_preset_id) <= 100
  ),
  avatar_url TEXT NOT NULL DEFAULT '' CHECK (avatar_url = btrim(avatar_url)),
  position_count BIGINT NOT NULL DEFAULT 0 CHECK (position_count BETWEEN 0 AND 1000),
  completed_count BIGINT NOT NULL DEFAULT 0 CHECK (completed_count >= 0),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (batch_id, ordinal),
  CONSTRAINT worm_position_cash_out_batch_wallets_wallet_unique
    UNIQUE (batch_id, wallet_id),
  CHECK (completed_count <= position_count)
);

CREATE TABLE worm_position_cash_out_batch_wallet_locks (
  wallet_id BIGINT PRIMARY KEY
    REFERENCES worm_wallet_connections(wallet_id) ON DELETE RESTRICT
    CHECK (wallet_id > 0),
  batch_id UUID NOT NULL
    REFERENCES worm_position_cash_out_batches(id) ON DELETE RESTRICT,
  address TEXT NOT NULL CHECK (
    address = btrim(address) AND char_length(address) BETWEEN 32 AND 64
  ),
  acquired_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT worm_position_cash_out_batch_wallet_locks_batch_wallet_fk
    FOREIGN KEY (batch_id, wallet_id)
    REFERENCES worm_position_cash_out_batch_wallets(batch_id, wallet_id)
    ON DELETE RESTRICT
);

CREATE INDEX worm_position_cash_out_batch_wallet_locks_batch_idx
  ON worm_position_cash_out_batch_wallet_locks (batch_id, wallet_id);

CREATE TABLE worm_position_cash_out_batch_items (
  id UUID PRIMARY KEY,
  batch_id UUID NOT NULL
    REFERENCES worm_position_cash_out_batches(id) ON DELETE RESTRICT,
  ordinal BIGINT NOT NULL CHECK (ordinal > 0),
  wallet_ordinal INTEGER NOT NULL CHECK (wallet_ordinal > 0),
  position_ordinal INTEGER NOT NULL CHECK (position_ordinal > 0),
  wallet_id BIGINT NOT NULL CHECK (wallet_id > 0),
  wallet_address TEXT NOT NULL CHECK (
    wallet_address = btrim(wallet_address)
    AND char_length(wallet_address) BETWEEN 32 AND 64
  ),
  wallet_remark TEXT NOT NULL DEFAULT '' CHECK (wallet_remark = btrim(wallet_remark)),
  credential_version BIGINT NOT NULL CHECK (credential_version > 0),
  position_pubkey TEXT NOT NULL CHECK (
    position_pubkey = btrim(position_pubkey)
    AND char_length(position_pubkey) BETWEEN 32 AND 64
  ),
  position_request_pubkey TEXT NOT NULL DEFAULT '' CHECK (
    position_request_pubkey = ''
    OR (
      position_request_pubkey = btrim(position_request_pubkey)
      AND char_length(position_request_pubkey) BETWEEN 32 AND 64
    )
  ),
  market_condition_id TEXT NOT NULL CHECK (
    market_condition_id = btrim(market_condition_id)
    AND char_length(market_condition_id) BETWEEN 32 AND 64
  ),
  market_title TEXT NOT NULL CHECK (
    market_title = btrim(market_title) AND market_title <> ''
  ),
  is_yes BOOLEAN NOT NULL,
  shares TEXT NOT NULL CHECK (
    shares = btrim(shares) AND char_length(shares) BETWEEN 1 AND 128
  ),
  position_created_at TIMESTAMPTZ NOT NULL CHECK (
    position_created_at > TIMESTAMPTZ '1970-01-01 00:00:00+00'
  ),
  provider_state TEXT NOT NULL DEFAULT '' CHECK (
    provider_state = btrim(provider_state) AND char_length(provider_state) <= 100
  ),
  state TEXT NOT NULL CHECK (
    state IN (
      'PENDING', 'PREFLIGHTING', 'CLOSING', 'AWAITING_POSITION',
      'AWAITING_BALANCE', 'COMPLETED', 'FAILED',
      'RECONCILIATION_REQUIRED', 'NOT_EXECUTED'
    )
  ),
  reason_code TEXT NOT NULL DEFAULT '' CHECK (
    reason_code = btrim(reason_code) AND char_length(reason_code) <= 100
  ),
  child_cash_out_id UUID UNIQUE
    REFERENCES worm_position_cash_outs(id) ON DELETE RESTRICT,
  baseline_usdc_mint TEXT NOT NULL DEFAULT '' CHECK (
    baseline_usdc_mint = btrim(baseline_usdc_mint)
    AND char_length(baseline_usdc_mint) <= 64
  ),
  baseline_usdc_decimals INTEGER CHECK (baseline_usdc_decimals >= 0),
  baseline_usdc_atomic_amount TEXT NOT NULL DEFAULT '' CHECK (
    baseline_usdc_atomic_amount = btrim(baseline_usdc_atomic_amount)
    AND char_length(baseline_usdc_atomic_amount) <= 128
  ),
  baseline_usdc_observed_slot BIGINT CHECK (baseline_usdc_observed_slot > 0),
  observed_usdc_mint TEXT NOT NULL DEFAULT '' CHECK (
    observed_usdc_mint = btrim(observed_usdc_mint)
    AND char_length(observed_usdc_mint) <= 64
  ),
  observed_usdc_decimals INTEGER CHECK (observed_usdc_decimals >= 0),
  observed_usdc_atomic_amount TEXT NOT NULL DEFAULT '' CHECK (
    observed_usdc_atomic_amount = btrim(observed_usdc_atomic_amount)
    AND char_length(observed_usdc_atomic_amount) <= 128
  ),
  observed_usdc_slot BIGINT CHECK (observed_usdc_slot > 0),
  delta_usdc_atomic_amount TEXT NOT NULL DEFAULT '' CHECK (
    delta_usdc_atomic_amount = btrim(delta_usdc_atomic_amount)
    AND char_length(delta_usdc_atomic_amount) <= 128
  ),
  balance_started_at TIMESTAMPTZ,
  balance_deadline_at TIMESTAMPTZ,
  balance_confirmed_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT worm_position_cash_out_batch_items_ordinal_unique
    UNIQUE (batch_id, ordinal),
  CONSTRAINT worm_position_cash_out_batch_items_wallet_position_unique
    UNIQUE (batch_id, wallet_ordinal, position_ordinal),
  CONSTRAINT worm_position_cash_out_batch_items_position_unique
    UNIQUE (batch_id, position_pubkey),
  CONSTRAINT worm_position_cash_out_batch_items_wallet_fk
    FOREIGN KEY (batch_id, wallet_ordinal)
    REFERENCES worm_position_cash_out_batch_wallets(batch_id, ordinal)
    ON DELETE RESTRICT,
  CHECK (
    (baseline_usdc_mint = '' AND baseline_usdc_decimals IS NULL
      AND baseline_usdc_atomic_amount = '' AND baseline_usdc_observed_slot IS NULL)
    OR (baseline_usdc_mint <> '' AND baseline_usdc_decimals IS NOT NULL
      AND baseline_usdc_atomic_amount <> '' AND baseline_usdc_observed_slot IS NOT NULL)
  ),
  CHECK (
    (observed_usdc_mint = '' AND observed_usdc_decimals IS NULL
      AND observed_usdc_atomic_amount = '' AND observed_usdc_slot IS NULL)
    OR (observed_usdc_mint <> '' AND observed_usdc_decimals IS NOT NULL
      AND observed_usdc_atomic_amount <> '' AND observed_usdc_slot IS NOT NULL)
  ),
  CHECK (
    (state = 'COMPLETED' AND completed_at IS NOT NULL
      AND balance_confirmed_at IS NOT NULL AND delta_usdc_atomic_amount <> '')
    OR (state IN ('FAILED', 'NOT_EXECUTED') AND completed_at IS NOT NULL)
    OR (state NOT IN ('COMPLETED', 'FAILED', 'NOT_EXECUTED') AND completed_at IS NULL)
  ),
  CHECK (
    (state IN ('FAILED', 'RECONCILIATION_REQUIRED', 'NOT_EXECUTED') AND reason_code <> '')
    OR state NOT IN ('FAILED', 'RECONCILIATION_REQUIRED', 'NOT_EXECUTED')
  )
);

CREATE INDEX worm_position_cash_out_batch_items_page_idx
  ON worm_position_cash_out_batch_items (batch_id, ordinal);

CREATE INDEX worm_position_cash_out_batch_items_wallet_idx
  ON worm_position_cash_out_batch_items (batch_id, wallet_ordinal, position_ordinal);

CREATE TABLE worm_position_cash_out_batch_authorizations (
  id UUID PRIMARY KEY,
  batch_id UUID NOT NULL
    REFERENCES worm_position_cash_out_batches(id) ON DELETE RESTRICT,
  owner_account_id UUID NOT NULL,
  scope TEXT NOT NULL CHECK (scope = 'WORM_POSITION_CASH_OUT_BATCH'),
  proof_kind TEXT NOT NULL CHECK (
    proof_kind = btrim(proof_kind) AND char_length(proof_kind) BETWEEN 1 AND 100
  ),
  session_jti_digest BYTEA NOT NULL CHECK (octet_length(session_jti_digest) = 32),
  access_revision BIGINT NOT NULL CHECK (access_revision > 0),
  intent_digest_sha256 BYTEA NOT NULL CHECK (octet_length(intent_digest_sha256) = 32),
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

CREATE UNIQUE INDEX worm_position_cash_out_batch_authorizations_one_active_idx
  ON worm_position_cash_out_batch_authorizations (batch_id)
  WHERE state = 'AUTHORIZED';

CREATE INDEX worm_position_cash_out_batch_authorizations_owner_idx
  ON worm_position_cash_out_batch_authorizations (owner_account_id, authorized_at DESC);

CREATE TABLE worm_position_cash_out_batch_commands (
  id UUID PRIMARY KEY,
  batch_id UUID NOT NULL
    REFERENCES worm_position_cash_out_batches(id) ON DELETE RESTRICT,
  owner_account_id UUID NOT NULL,
  kind TEXT NOT NULL CHECK (
    kind IN ('CREATE', 'AUTHORIZE', 'CANCEL', 'PAUSE', 'CONTINUE',
             'TERMINATE', 'CHECK_STATUS')
  ),
  state TEXT NOT NULL CHECK (state IN ('APPLIED', 'FAILED')),
  request_sha256 BYTEA NOT NULL CHECK (octet_length(request_sha256) = 32),
  batch_revision_after BIGINT NOT NULL CHECK (batch_revision_after > 0),
  result_code TEXT NOT NULL DEFAULT '' CHECK (
    result_code = btrim(result_code) AND char_length(result_code) <= 100
  ),
  completed_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX worm_position_cash_out_batch_commands_batch_idx
  ON worm_position_cash_out_batch_commands (batch_id, created_at, id);

ALTER TABLE worm_position_cash_outs
  ADD COLUMN batch_id UUID
    REFERENCES worm_position_cash_out_batches(id) ON DELETE RESTRICT,
  ADD COLUMN batch_item_id UUID
    REFERENCES worm_position_cash_out_batch_items(id) ON DELETE RESTRICT,
  ADD CONSTRAINT worm_position_cash_outs_batch_source_pair CHECK (
    (batch_id IS NULL AND batch_item_id IS NULL)
    OR (batch_id IS NOT NULL AND batch_item_id IS NOT NULL)
  ),
  ADD CONSTRAINT worm_position_cash_outs_batch_item_unique UNIQUE (batch_item_id),
  ADD CONSTRAINT worm_position_cash_outs_batch_id_id_unique UNIQUE (batch_id, id);

ALTER TABLE worm_position_cash_out_batch_items
  ADD CONSTRAINT worm_position_cash_out_batch_items_child_source_fk
  FOREIGN KEY (batch_id, child_cash_out_id)
  REFERENCES worm_position_cash_outs(batch_id, id)
  DEFERRABLE INITIALLY DEFERRED;

-- +goose Down

ALTER TABLE worm_position_cash_out_batch_items
  DROP CONSTRAINT IF EXISTS worm_position_cash_out_batch_items_child_source_fk;

ALTER TABLE worm_position_cash_outs
  DROP CONSTRAINT IF EXISTS worm_position_cash_outs_batch_id_id_unique,
  DROP CONSTRAINT IF EXISTS worm_position_cash_outs_batch_item_unique,
  DROP CONSTRAINT IF EXISTS worm_position_cash_outs_batch_source_pair,
  DROP COLUMN IF EXISTS batch_item_id,
  DROP COLUMN IF EXISTS batch_id;

DROP TABLE IF EXISTS worm_position_cash_out_batch_commands;
DROP TABLE IF EXISTS worm_position_cash_out_batch_authorizations;
DROP TABLE IF EXISTS worm_position_cash_out_batch_items;
DROP TABLE IF EXISTS worm_position_cash_out_batch_wallet_locks;
DROP TABLE IF EXISTS worm_position_cash_out_batch_wallets;
DROP TABLE IF EXISTS worm_position_cash_out_batches;
