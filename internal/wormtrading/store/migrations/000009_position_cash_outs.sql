-- +goose Up

CREATE TABLE worm_position_cash_outs (
  id UUID PRIMARY KEY,
  owner_account_id UUID NOT NULL,
  wallet_id BIGINT NOT NULL
    REFERENCES worm_wallet_connections(wallet_id) ON DELETE RESTRICT
    CHECK (wallet_id > 0),
  wallet_address TEXT NOT NULL CHECK (
    wallet_address = btrim(wallet_address)
    AND char_length(wallet_address) BETWEEN 32 AND 64
  ),
  credential_version BIGINT NOT NULL CHECK (credential_version > 0),
  position_pubkey TEXT NOT NULL CHECK (
    position_pubkey = btrim(position_pubkey)
    AND char_length(position_pubkey) BETWEEN 32 AND 64
  ),
  market_condition_id TEXT NOT NULL CHECK (
    market_condition_id = btrim(market_condition_id)
    AND char_length(market_condition_id) BETWEEN 32 AND 64
  ),
  is_yes BOOLEAN NOT NULL,
  position_created_at TIMESTAMPTZ NOT NULL CHECK (
    position_created_at > TIMESTAMPTZ '1970-01-01 00:00:00+00'
  ),
  position_request_pubkey TEXT NOT NULL DEFAULT '' CHECK (
    position_request_pubkey = ''
    OR (
      position_request_pubkey = btrim(position_request_pubkey)
      AND char_length(position_request_pubkey) BETWEEN 32 AND 64
    )
  ),
  shares TEXT NOT NULL CHECK (
    shares = btrim(shares) AND char_length(shares) BETWEEN 1 AND 128
  ),
  intent_digest_sha256 BYTEA NOT NULL CHECK (
    octet_length(intent_digest_sha256) = 32
  ),
  state TEXT NOT NULL CHECK (
    state IN (
      'AWAITING_AUTHORIZATION',
      'QUEUED',
      'PREFLIGHTING',
      'CLOSING',
      'AWAITING_COMPLETION',
      'COMPLETED',
      'FAILED',
      'RECONCILIATION_REQUIRED',
      'EXPIRED'
    )
  ),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  reason_code TEXT NOT NULL DEFAULT '' CHECK (
    reason_code = btrim(reason_code) AND char_length(reason_code) <= 100
  ),
  provider_state TEXT NOT NULL DEFAULT '' CHECK (
    provider_state = btrim(provider_state) AND char_length(provider_state) <= 100
  ),
  provider_is_closed BOOLEAN NOT NULL DEFAULT FALSE,
  provider_is_liquidated BOOLEAN NOT NULL DEFAULT FALSE,
  authorization_expires_at TIMESTAMPTZ NOT NULL,
  execution_expires_at TIMESTAMPTZ,
  authorized_at TIMESTAMPTZ,
  next_poll_at TIMESTAMPTZ,
  poll_count INTEGER NOT NULL DEFAULT 0 CHECK (poll_count >= 0),
  reconcile_requested_at TIMESTAMPTZ,
  claim_id UUID,
  claim_owner TEXT NOT NULL DEFAULT '' CHECK (
    claim_owner = btrim(claim_owner) AND char_length(claim_owner) <= 100
  ),
  claim_expires_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CHECK (authorization_expires_at > created_at),
  CHECK (
    execution_expires_at IS NULL
    OR (authorized_at IS NOT NULL AND execution_expires_at > authorized_at)
  ),
  CHECK (
    (claim_id IS NULL AND claim_owner = '' AND claim_expires_at IS NULL)
    OR (claim_id IS NOT NULL AND claim_owner <> '' AND claim_expires_at IS NOT NULL)
  ),
  CHECK (
    (
      state = 'AWAITING_AUTHORIZATION'
      AND authorized_at IS NULL
      AND execution_expires_at IS NULL
    )
    OR (
      state IN (
        'QUEUED', 'PREFLIGHTING', 'CLOSING', 'AWAITING_COMPLETION',
        'RECONCILIATION_REQUIRED'
      )
      AND authorized_at IS NOT NULL
      AND execution_expires_at IS NOT NULL
    )
    OR (
      state IN ('COMPLETED', 'FAILED')
      AND (
        (authorized_at IS NULL AND execution_expires_at IS NULL)
        OR (authorized_at IS NOT NULL AND execution_expires_at IS NOT NULL)
      )
    )
    OR state = 'EXPIRED'
  ),
  CHECK (
    (
      state IN ('COMPLETED', 'FAILED', 'EXPIRED')
      AND completed_at IS NOT NULL
      AND claim_id IS NULL
    )
    OR (
      state NOT IN ('COMPLETED', 'FAILED', 'EXPIRED')
      AND completed_at IS NULL
    )
  ),
  CHECK (
    (state = 'COMPLETED' AND provider_is_closed AND NOT provider_is_liquidated)
    OR state <> 'COMPLETED'
  ),
  CHECK (
    (state IN ('AWAITING_COMPLETION', 'RECONCILIATION_REQUIRED') AND next_poll_at IS NOT NULL)
    OR state NOT IN ('AWAITING_COMPLETION', 'RECONCILIATION_REQUIRED')
  ),
  CHECK (
    (state IN ('FAILED', 'RECONCILIATION_REQUIRED', 'EXPIRED') AND reason_code <> '')
    OR (state NOT IN ('FAILED', 'RECONCILIATION_REQUIRED', 'EXPIRED'))
  )
);

CREATE UNIQUE INDEX worm_position_cash_outs_one_active_per_wallet_idx
  ON worm_position_cash_outs (wallet_id)
  WHERE state IN (
    'AWAITING_AUTHORIZATION', 'QUEUED', 'PREFLIGHTING', 'CLOSING',
    'AWAITING_COMPLETION', 'RECONCILIATION_REQUIRED'
  );

CREATE UNIQUE INDEX worm_position_cash_outs_one_active_per_position_idx
  ON worm_position_cash_outs (position_pubkey)
  WHERE state IN (
    'AWAITING_AUTHORIZATION', 'QUEUED', 'PREFLIGHTING', 'CLOSING',
    'AWAITING_COMPLETION', 'RECONCILIATION_REQUIRED'
  );

CREATE INDEX worm_position_cash_outs_owner_created_idx
  ON worm_position_cash_outs (owner_account_id, created_at DESC, id DESC);

CREATE INDEX worm_position_cash_outs_recovery_idx
  ON worm_position_cash_outs (state, next_poll_at, claim_expires_at, updated_at, id)
  WHERE state IN (
    'QUEUED', 'PREFLIGHTING', 'CLOSING', 'AWAITING_COMPLETION',
    'RECONCILIATION_REQUIRED'
  );

CREATE TABLE worm_position_cash_out_authorizations (
  id UUID PRIMARY KEY,
  cash_out_id UUID NOT NULL UNIQUE
    REFERENCES worm_position_cash_outs(id) ON DELETE RESTRICT,
  owner_account_id UUID NOT NULL,
  scope TEXT NOT NULL CHECK (scope = 'WORM_POSITION_CASH_OUT'),
  proof_kind TEXT NOT NULL CHECK (
    proof_kind = btrim(proof_kind) AND char_length(proof_kind) BETWEEN 1 AND 100
  ),
  session_jti_digest BYTEA NOT NULL CHECK (octet_length(session_jti_digest) = 32),
  access_revision BIGINT NOT NULL CHECK (access_revision > 0),
  intent_digest_sha256 BYTEA NOT NULL CHECK (octet_length(intent_digest_sha256) = 32),
  state TEXT NOT NULL CHECK (state IN ('AUTHORIZED', 'CONSUMED', 'REVOKED')),
  authorized_at TIMESTAMPTZ NOT NULL,
  ended_at TIMESTAMPTZ,
  end_reason_code TEXT NOT NULL DEFAULT '' CHECK (
    end_reason_code = btrim(end_reason_code)
    AND char_length(end_reason_code) <= 100
  ),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CHECK (
    (state = 'AUTHORIZED' AND ended_at IS NULL AND end_reason_code = '')
    OR (state <> 'AUTHORIZED' AND ended_at IS NOT NULL AND end_reason_code <> '')
  )
);

CREATE INDEX worm_position_cash_out_authorizations_owner_idx
  ON worm_position_cash_out_authorizations (owner_account_id, authorized_at DESC);

CREATE TABLE worm_position_cash_out_commands (
  id UUID PRIMARY KEY,
  cash_out_id UUID NOT NULL
    REFERENCES worm_position_cash_outs(id) ON DELETE RESTRICT,
  owner_account_id UUID NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('CREATE', 'AUTHORIZE', 'RECONCILE')),
  state TEXT NOT NULL CHECK (state IN ('APPLIED', 'FAILED')),
  request_sha256 BYTEA NOT NULL CHECK (octet_length(request_sha256) = 32),
  cash_out_revision_after BIGINT NOT NULL CHECK (cash_out_revision_after > 0),
  result_code TEXT NOT NULL DEFAULT '' CHECK (
    result_code = btrim(result_code) AND char_length(result_code) <= 100
  ),
  completed_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX worm_position_cash_out_commands_operation_idx
  ON worm_position_cash_out_commands (cash_out_id, created_at, id);

CREATE TABLE worm_position_cash_out_attempts (
  id UUID PRIMARY KEY,
  cash_out_id UUID NOT NULL UNIQUE
    REFERENCES worm_position_cash_outs(id) ON DELETE RESTRICT,
  state TEXT NOT NULL CHECK (
    state IN ('PREPARED', 'DISPATCHED', 'ACKNOWLEDGED', 'REJECTED', 'OUTCOME_UNKNOWN')
  ),
  request_sha256 BYTEA NOT NULL CHECK (octet_length(request_sha256) = 32),
  http_status INTEGER CHECK (http_status BETWEEN 100 AND 599),
  provider_code INTEGER,
  provider_slug TEXT NOT NULL DEFAULT '' CHECK (
    provider_slug = btrim(provider_slug) AND char_length(provider_slug) <= 100
  ),
  provider_state TEXT NOT NULL DEFAULT '' CHECK (
    provider_state = btrim(provider_state) AND char_length(provider_state) <= 100
  ),
  error_code TEXT NOT NULL DEFAULT '' CHECK (
    error_code = btrim(error_code) AND char_length(error_code) <= 100
  ),
  prepared_at TIMESTAMPTZ NOT NULL,
  dispatched_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CHECK (
    (state = 'PREPARED' AND dispatched_at IS NULL AND completed_at IS NULL)
    OR (state = 'DISPATCHED' AND dispatched_at IS NOT NULL AND completed_at IS NULL)
    OR (
      state IN ('ACKNOWLEDGED', 'REJECTED', 'OUTCOME_UNKNOWN')
      AND dispatched_at IS NOT NULL
      AND completed_at IS NOT NULL
    )
  )
);

-- +goose Down

DROP TABLE IF EXISTS worm_position_cash_out_attempts;
DROP TABLE IF EXISTS worm_position_cash_out_commands;
DROP TABLE IF EXISTS worm_position_cash_out_authorizations;
DROP TABLE IF EXISTS worm_position_cash_outs;
