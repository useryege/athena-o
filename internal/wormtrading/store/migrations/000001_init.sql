-- +goose Up

CREATE TABLE IF NOT EXISTS worm_wallet_connections (
  wallet_id BIGINT PRIMARY KEY CHECK (wallet_id > 0),
  address TEXT NOT NULL CHECK (
    address = btrim(address)
    AND char_length(address) BETWEEN 32 AND 64
  ),
  state TEXT NOT NULL CHECK (
    state IN (
      'NOT_CONNECTED',
      'CONNECTING',
      'CONNECTED',
      'RECONNECT_REQUIRED',
      'DISCONNECTING',
      'REVOCATION_REQUIRED'
    )
  ),
  warning_code TEXT NOT NULL DEFAULT '' CHECK (
    warning_code = btrim(warning_code)
    AND char_length(warning_code) <= 100
  ),
  connected_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (wallet_id, address),
  CHECK (
    state IN ('CONNECTED', 'RECONNECT_REQUIRED', 'DISCONNECTING', 'REVOCATION_REQUIRED')
    OR connected_at IS NULL
  )
);

CREATE TABLE IF NOT EXISTS worm_wallet_credentials (
  id BIGSERIAL PRIMARY KEY,
  wallet_id BIGINT NOT NULL REFERENCES worm_wallet_connections(wallet_id) ON DELETE CASCADE,
  version BIGINT NOT NULL CHECK (version > 0),
  state TEXT NOT NULL CHECK (
    state IN ('ACTIVE', 'PENDING_REVOCATION', 'REVOKING', 'REVOCATION_REQUIRED')
  ),
  api_key_ciphertext BYTEA NOT NULL CHECK (octet_length(api_key_ciphertext) > 0),
  api_secret_ciphertext BYTEA NOT NULL CHECK (octet_length(api_secret_ciphertext) > 0),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (wallet_id, version)
);

CREATE UNIQUE INDEX IF NOT EXISTS worm_wallet_credentials_one_active_idx
  ON worm_wallet_credentials (wallet_id)
  WHERE state = 'ACTIVE';

CREATE INDEX IF NOT EXISTS worm_wallet_credentials_revocation_idx
  ON worm_wallet_credentials (state, updated_at, wallet_id, version);

CREATE TABLE IF NOT EXISTS worm_wallet_connection_attempts (
  id UUID PRIMARY KEY,
  wallet_id BIGINT NOT NULL,
  address TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('CONNECT', 'RECONNECT')),
  previous_connection_state TEXT NOT NULL CHECK (
    previous_connection_state IN (
      'NOT_CONNECTED',
      'CONNECTED',
      'RECONNECT_REQUIRED',
      'REVOCATION_REQUIRED'
    )
  ),
  previous_connected_at TIMESTAMPTZ,
  nonce TEXT NOT NULL CHECK (
    nonce = btrim(nonce)
    AND char_length(nonce) BETWEEN 1 AND 512
  ),
  challenge_message TEXT NOT NULL CHECK (
    char_length(challenge_message) BETWEEN 1 AND 1024
  ),
  message_digest BYTEA NOT NULL CHECK (octet_length(message_digest) = 32),
  state TEXT NOT NULL CHECK (
    state IN (
      'PREPARED',
      'COMPLETING',
      'COMPLETED',
      'FAILED',
      'CANCELLED',
      'OUTCOME_UNKNOWN'
    )
  ),
  failure_code TEXT NOT NULL DEFAULT '' CHECK (
    failure_code = btrim(failure_code)
    AND char_length(failure_code) <= 100
  ),
  expires_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT worm_wallet_connection_attempts_connection_fk
    FOREIGN KEY (wallet_id, address)
    REFERENCES worm_wallet_connections(wallet_id, address)
    ON DELETE RESTRICT,
  CHECK (expires_at > created_at),
  CHECK (
    (state IN ('PREPARED', 'COMPLETING') AND completed_at IS NULL)
    OR (state NOT IN ('PREPARED', 'COMPLETING') AND completed_at IS NOT NULL)
  )
);

CREATE UNIQUE INDEX IF NOT EXISTS worm_wallet_connection_attempts_one_active_idx
  ON worm_wallet_connection_attempts (wallet_id)
  WHERE state IN ('PREPARED', 'COMPLETING');

CREATE INDEX IF NOT EXISTS worm_wallet_connection_attempts_expiry_idx
  ON worm_wallet_connection_attempts (expires_at)
  WHERE state IN ('PREPARED', 'COMPLETING');

CREATE INDEX IF NOT EXISTS worm_wallet_connection_attempts_wallet_created_idx
  ON worm_wallet_connection_attempts (wallet_id, created_at DESC, id);

-- +goose Down

DROP TABLE IF EXISTS worm_wallet_connection_attempts;
DROP TABLE IF EXISTS worm_wallet_credentials;
DROP TABLE IF EXISTS worm_wallet_connections;
