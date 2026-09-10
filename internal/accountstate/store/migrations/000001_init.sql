-- +goose Up

-- +goose StatementBegin
CREATE FUNCTION athena_is_canonical_solana_public_key(value TEXT)
RETURNS BOOLEAN
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
AS $$
DECLARE
  alphabet CONSTANT TEXT := '123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz';
  encoded_length INTEGER := char_length(value);
  leading_zero_bytes INTEGER := 0;
  accumulator NUMERIC := 0;
  remaining NUMERIC;
  decoded_nonzero_bytes INTEGER := 0;
  position_index INTEGER;
  digit INTEGER;
BEGIN
  IF encoded_length < 32 OR encoded_length > 44 THEN
    RETURN FALSE;
  END IF;

  FOR position_index IN 1..encoded_length LOOP
    digit := strpos(alphabet, substr(value, position_index, 1)) - 1;
    IF digit < 0 THEN
      RETURN FALSE;
    END IF;
    IF position_index = leading_zero_bytes + 1 AND digit = 0 THEN
      leading_zero_bytes := leading_zero_bytes + 1;
    END IF;
    accumulator := accumulator * 58 + digit;
  END LOOP;

  remaining := accumulator;
  WHILE remaining > 0 LOOP
    decoded_nonzero_bytes := decoded_nonzero_bytes + 1;
    remaining := trunc(remaining / 256);
  END LOOP;

  RETURN leading_zero_bytes + decoded_nonzero_bytes = 32;
END;
$$;
-- +goose StatementEnd

CREATE TABLE athena_account (
  account_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username TEXT NOT NULL,
  identity_provider TEXT NOT NULL,
  identity_subject TEXT,
  verified_email TEXT NOT NULL DEFAULT '',
  administrator BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_login_at TIMESTAMPTZ,
  CONSTRAINT athena_account_username_check CHECK (
    char_length(username) BETWEEN 3 AND 42
    AND username ~ '^[A-Za-z0-9.-]+$'
    AND username ~ '[A-Za-z0-9]'
    AND username !~* '^0x[0-9a-f]{40}$'
  ),
  CONSTRAINT athena_account_admin_username_check CHECK (
    lower(username) <> 'admin'
    OR (administrator AND username = 'admin')
  ),
  CONSTRAINT athena_account_identity_provider_check CHECK (
    identity_provider IN ('google', 'solana_wallet', 'development')
  ),
  CONSTRAINT athena_account_identity_subject_check CHECK (
    identity_subject IS NULL
    OR (
      identity_subject = btrim(identity_subject)
      AND identity_subject <> ''
      AND identity_subject !~ '[[:cntrl:]]'
    )
  ),
  CONSTRAINT athena_account_verified_email_check CHECK (
    verified_email = ''
    OR (
      verified_email = btrim(verified_email)
      AND verified_email !~ '[[:cntrl:]]'
    )
  ),
  CONSTRAINT athena_account_identity_binding_check CHECK (
    (
      identity_provider = 'google'
      AND identity_subject IS NOT NULL
      AND verified_email <> ''
    )
    OR (
      identity_provider = 'solana_wallet'
      AND identity_subject IS NOT NULL
      AND athena_is_canonical_solana_public_key(identity_subject)
      AND verified_email = ''
      AND NOT administrator
    )
    OR (
      identity_provider = 'development'
      AND identity_subject IS NULL
      AND verified_email = ''
      AND (
        (administrator AND username = 'local-admin')
        OR (NOT administrator AND username = 'local-user')
      )
    )
  ),
  CONSTRAINT athena_account_administrator_provider_check CHECK (
    NOT administrator OR identity_provider IN ('google', 'development')
  ),
  CONSTRAINT athena_account_last_login_check CHECK (
    last_login_at IS NULL OR last_login_at >= created_at
  )
);

CREATE UNIQUE INDEX athena_account_username_lower_uidx
  ON athena_account (lower(username));

CREATE UNIQUE INDEX athena_account_identity_uidx
  ON athena_account (identity_provider, identity_subject, administrator)
  WHERE identity_subject IS NOT NULL;

CREATE UNIQUE INDEX athena_account_single_administrator_uidx
  ON athena_account (administrator)
  WHERE administrator;

CREATE TABLE account_access (
  account_id UUID PRIMARY KEY,
  login_enabled BOOLEAN NOT NULL,
  api_key_enabled BOOLEAN NOT NULL,
  profit_sharing_enabled BOOLEAN NOT NULL,
  revision BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT account_access_account_fk
    FOREIGN KEY (account_id)
    REFERENCES athena_account (account_id)
    ON DELETE CASCADE,
  CONSTRAINT account_access_revision_check CHECK (revision > 0)
);

CREATE TABLE account_module_access (
  account_id UUID NOT NULL,
  module TEXT NOT NULL,
  access_level TEXT NOT NULL,
  CONSTRAINT account_module_access_pk
    PRIMARY KEY (account_id, module),
  CONSTRAINT account_module_access_account_fk
    FOREIGN KEY (account_id)
    REFERENCES account_access (account_id)
    ON DELETE CASCADE,
  CONSTRAINT account_module_access_module_check
    CHECK (module IN (
      'market_radar',
      'sports_live',
      'sports_history',
      'managed_oo',
      'worm_markets',
      'worm_trading',
      'world_cup_corners',
      'token',
      'wallet'
    )),
  CONSTRAINT account_module_access_level_check
    CHECK (access_level IN ('none', 'read', 'read_write')),
  CONSTRAINT account_module_access_max_level_check CHECK (
    access_level <> 'read_write'
    OR module IN (
      'sports_history',
      'managed_oo',
      'worm_trading',
      'token',
      'wallet'
    )
  )
);

CREATE TABLE account_profile (
  account_id UUID PRIMARY KEY,
  display_name TEXT NOT NULL,
  account_tier TEXT NOT NULL DEFAULT 'standard',
  avatar_object_key TEXT NOT NULL DEFAULT '',
  avatar_content_type TEXT NOT NULL DEFAULT '',
  avatar_etag TEXT NOT NULL DEFAULT '',
  avatar_size_bytes BIGINT NOT NULL DEFAULT 0,
  revision BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT account_profile_account_fk
    FOREIGN KEY (account_id)
    REFERENCES athena_account (account_id)
    ON DELETE CASCADE,
  CONSTRAINT account_profile_display_name_check CHECK (
    char_length(display_name) BETWEEN 1 AND 80
    AND display_name !~ '[[:cntrl:]]'
  ),
  CONSTRAINT account_profile_tier_check CHECK (account_tier IN ('standard', 'pro')),
  CONSTRAINT account_profile_avatar_check CHECK (
    (
      avatar_object_key = ''
      AND avatar_content_type = ''
      AND avatar_etag = ''
      AND avatar_size_bytes = 0
    )
    OR (
      avatar_object_key <> ''
      AND avatar_content_type IN ('image/jpeg', 'image/png', 'image/webp')
      AND avatar_etag <> ''
      AND avatar_size_bytes > 0
    )
  ),
  CONSTRAINT account_profile_revision_check CHECK (revision > 0)
);

CREATE TABLE account_preferences (
  account_id UUID PRIMARY KEY,
  theme TEXT NOT NULL,
  revision BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT account_preferences_account_fk
    FOREIGN KEY (account_id)
    REFERENCES athena_account (account_id)
    ON DELETE CASCADE,
  CONSTRAINT account_preferences_theme_check CHECK (theme IN ('system', 'light', 'dark')),
  CONSTRAINT account_preferences_revision_check CHECK (revision > 0)
);

CREATE TABLE account_api_key (
  account_id UUID NOT NULL,
  display_id TEXT NOT NULL,
  jti TEXT NOT NULL UNIQUE,
  issued_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ,
  CONSTRAINT account_api_key_pk PRIMARY KEY (account_id, display_id),
  CONSTRAINT account_api_key_account_fk
    FOREIGN KEY (account_id)
    REFERENCES athena_account (account_id)
    ON DELETE CASCADE,
  CONSTRAINT account_api_key_display_id_check CHECK (
    display_id ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$'
  ),
  CONSTRAINT account_api_key_jti_check CHECK (
    jti <> '' AND jti !~ '[[:cntrl:]]'
  ),
  CONSTRAINT account_api_key_expiry_check CHECK (
    expires_at IS NULL OR expires_at > issued_at
  )
);

CREATE INDEX account_api_key_account_issued_idx
  ON account_api_key (account_id, issued_at DESC, display_id);

CREATE INDEX athena_account_recent_login_idx
  ON athena_account (last_login_at DESC NULLS LAST, account_id);

-- Normal authentication starts with no account rows. External registration creates
-- a complete account aggregate, while disabled-auth development explicitly
-- creates one selected isolated local development identity through the
-- account-state store. The member and administrator identities may coexist.

CREATE TABLE system_notification_topics (
  telegram_chat TEXT NOT NULL,
  label TEXT NOT NULL,
  message_thread_id INT NOT NULL CHECK (message_thread_id > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (telegram_chat, label),
  CONSTRAINT system_notification_topics_chat_check
    CHECK (telegram_chat IN ('test', 'prod'))
);

INSERT INTO system_notification_topics (telegram_chat, label, message_thread_id)
VALUES ('prod', '[POLY] UMA Disputed', 559)
ON CONFLICT (telegram_chat, label) DO UPDATE
SET message_thread_id = EXCLUDED.message_thread_id;

CREATE TABLE notification_delivery_attempts (
  id UUID PRIMARY KEY,
  work_kind TEXT NOT NULL CHECK (work_kind IN ('account', 'system', 'reply')),
  work_id BIGINT NOT NULL CHECK (work_id > 0),
  owner_id UUID,
  sender_incarnation UUID NOT NULL,
  payload_digest BYTEA NOT NULL CHECK (octet_length(payload_digest) = 32),
  authorized_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  started_at TIMESTAMPTZ,
  result_at TIMESTAMPTZ,
  message_id TEXT,
  outcome TEXT CHECK (outcome IN ('sent', 'retryable', 'failed', 'unknown')),
  outcome_code TEXT,
  retry_after INTERVAL,
  CHECK ((work_kind <> 'account' OR owner_id IS NOT NULL) AND (work_kind <> 'system' OR owner_id IS NULL)),
  CHECK ((result_at IS NULL) = (outcome IS NULL))
);
CREATE INDEX idx_notification_delivery_attempts_work
  ON notification_delivery_attempts (work_kind, work_id, authorized_at);

CREATE TABLE system_notification_deliveries (
  id BIGSERIAL PRIMARY KEY,
  source TEXT NOT NULL,
  severity TEXT NOT NULL,
  title TEXT,
  body TEXT NOT NULL,
  link TEXT,
  channel TEXT NOT NULL,
  status TEXT NOT NULL,
  provider_message_id TEXT,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  sent_at TIMESTAMPTZ,
  telegram_chat TEXT NOT NULL,
  topic_label TEXT NOT NULL,
  attempts INT NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_attempt_at TIMESTAMPTZ,
  locked_at TIMESTAMPTZ,
  locked_by TEXT,
  current_attempt_id UUID REFERENCES notification_delivery_attempts(id),
  CONSTRAINT system_notification_deliveries_status_check
    CHECK (status IN ('pending', 'sending', 'sent', 'failed', 'unknown', 'cancelled')),
  CONSTRAINT system_notification_deliveries_severity_check
    CHECK (severity IN ('info', 'warning', 'error', 'critical')),
  CONSTRAINT system_notification_deliveries_channel_check
    CHECK (channel = 'telegram'),
  CONSTRAINT system_notification_deliveries_topic_fk
    FOREIGN KEY (telegram_chat, topic_label)
    REFERENCES system_notification_topics (telegram_chat, label)
);

CREATE INDEX idx_system_notification_deliveries_created_at
  ON system_notification_deliveries (created_at DESC);
CREATE INDEX idx_system_notification_deliveries_status
  ON system_notification_deliveries (status);
CREATE INDEX idx_system_notification_deliveries_severity
  ON system_notification_deliveries (severity);
CREATE INDEX idx_system_notification_deliveries_source
  ON system_notification_deliveries (source);
CREATE INDEX idx_system_notification_deliveries_chat_topic
  ON system_notification_deliveries (telegram_chat, topic_label);
CREATE INDEX idx_system_notification_deliveries_pending_ready
  ON system_notification_deliveries (status, next_attempt_at, id);

CREATE TABLE telegram_binding_versions (
  account_id UUID PRIMARY KEY,
  revision BIGINT NOT NULL CHECK (revision > 0)
);

CREATE TABLE telegram_bindings (
  account_id UUID PRIMARY KEY,
  telegram_user_id BIGINT NOT NULL UNIQUE CHECK (telegram_user_id > 0),
  telegram_chat_id BIGINT NOT NULL UNIQUE CHECK (telegram_chat_id > 0),
  telegram_username TEXT,
  telegram_display_name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'connected',
  revision BIGINT NOT NULL CHECK (revision > 0),
  bound_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_error TEXT,
  CONSTRAINT telegram_bindings_status_check
    CHECK (status IN ('connected', 'unreachable'))
);

CREATE TABLE telegram_binding_attempts (
  id UUID PRIMARY KEY,
  account_id UUID NOT NULL UNIQUE,
  token_digest BYTEA NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'pending',
  failure_reason TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT telegram_binding_attempts_token_digest_check
    CHECK (octet_length(token_digest) = 32),
  CONSTRAINT telegram_binding_attempts_status_check
    CHECK (status IN ('pending', 'failed')),
  CONSTRAINT telegram_binding_attempts_failure_check
    CHECK (
      (status = 'pending' AND failure_reason IS NULL)
      OR (status = 'failed' AND failure_reason IS NOT NULL)
    )
);

CREATE INDEX idx_telegram_binding_attempts_expiry
  ON telegram_binding_attempts (status, expires_at);

CREATE TABLE telegram_polling_state (
  singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
  next_update_id BIGINT NOT NULL DEFAULT 0 CHECK (next_update_id >= 0),
  last_poll_at TIMESTAMPTZ,
  last_update_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO telegram_polling_state (singleton)
VALUES (TRUE);

CREATE TABLE telegram_consumed_updates (
  update_id BIGINT PRIMARY KEY CHECK (update_id >= 0),
  consumed_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE TABLE telegram_binding_replies (
  id BIGSERIAL PRIMARY KEY,
  update_id BIGINT NOT NULL UNIQUE REFERENCES telegram_consumed_updates(update_id),
  account_id UUID,
  binding_revision BIGINT,
  telegram_chat_id BIGINT NOT NULL CHECK (telegram_chat_id > 0),
  body TEXT NOT NULL,
  payload_digest BYTEA NOT NULL CHECK (octet_length(payload_digest) = 32),
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sending', 'sent', 'failed', 'unknown', 'cancelled')),
  attempts INT NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 5),
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  last_attempt_at TIMESTAMPTZ,
  locked_at TIMESTAMPTZ,
  locked_by TEXT,
  current_attempt_id UUID REFERENCES notification_delivery_attempts(id),
  provider_message_id TEXT,
  error_message TEXT,
  sent_at TIMESTAMPTZ,
  eligibility_revoked_at TIMESTAMPTZ,
  eligibility_revoked_reason TEXT,
  CHECK ((account_id IS NULL) = (binding_revision IS NULL)),
  CHECK (binding_revision IS NULL OR binding_revision > 0)
);
CREATE INDEX idx_telegram_binding_replies_ready ON telegram_binding_replies(status, next_attempt_at, id);

CREATE TABLE account_notification_deliveries (
  id BIGSERIAL PRIMARY KEY,
  account_id UUID NOT NULL,
  idempotency_key TEXT NOT NULL,
  payload_digest BYTEA NOT NULL,
  source TEXT NOT NULL,
  severity TEXT NOT NULL,
  title TEXT,
  body TEXT NOT NULL,
  link TEXT,
  channel TEXT NOT NULL,
  status TEXT NOT NULL,
  telegram_chat_id BIGINT NOT NULL CHECK (telegram_chat_id > 0),
  binding_revision BIGINT NOT NULL CHECK (binding_revision > 0),
  provider_message_id TEXT,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  sent_at TIMESTAMPTZ,
  attempts INT NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_attempt_at TIMESTAMPTZ,
  locked_at TIMESTAMPTZ,
  locked_by TEXT,
  current_attempt_id UUID REFERENCES notification_delivery_attempts(id),
  eligibility_revoked_at TIMESTAMPTZ,
  eligibility_revoked_reason TEXT,
  CONSTRAINT account_notification_deliveries_idempotency_unique
    UNIQUE (account_id, source, idempotency_key),
  CONSTRAINT account_notification_deliveries_payload_digest_check
    CHECK (octet_length(payload_digest) = 32),
  CONSTRAINT account_notification_deliveries_status_check
    CHECK (status IN ('pending', 'sending', 'sent', 'failed', 'unknown', 'cancelled')),
  CONSTRAINT account_notification_deliveries_severity_check
    CHECK (severity IN ('info', 'warning', 'error', 'critical')),
  CONSTRAINT account_notification_deliveries_channel_check
    CHECK (channel = 'telegram')
);

CREATE INDEX idx_account_notification_deliveries_account_created
  ON account_notification_deliveries (account_id, created_at DESC);
CREATE INDEX idx_account_notification_deliveries_pending_ready
  ON account_notification_deliveries (status, next_attempt_at, id);

-- +goose Down

DROP TABLE account_notification_deliveries;
DROP TABLE telegram_binding_replies;
DROP TABLE telegram_consumed_updates;
DROP TABLE telegram_polling_state;
DROP TABLE telegram_binding_attempts;
DROP TABLE telegram_bindings;
DROP TABLE telegram_binding_versions;
DROP TABLE system_notification_deliveries;
DROP TABLE notification_delivery_attempts;
DROP TABLE system_notification_topics;

DROP TABLE account_api_key;
DROP TABLE account_preferences;
DROP TABLE account_profile;
DROP TABLE account_module_access;
DROP TABLE account_access;
DROP TABLE athena_account;
DROP FUNCTION athena_is_canonical_solana_public_key(TEXT);
