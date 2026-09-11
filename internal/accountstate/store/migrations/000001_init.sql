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
      'wallet',
      'trader_sync'
    )),
  CHECK (module <> 'trader_sync' OR access_level <> 'read'),
  CONSTRAINT account_module_access_level_check
    CHECK (access_level IN ('none', 'read', 'read_write')),
  CONSTRAINT account_module_access_max_level_check CHECK (
    access_level <> 'read_write'
    OR module IN (
      'sports_history',
      'managed_oo',
      'worm_trading',
      'token',
      'wallet',
      'trader_sync'
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

CREATE TABLE notification_sender_instances (
  incarnation UUID PRIMARY KEY,
  hostname TEXT NOT NULL,
  process_id INTEGER NOT NULL,
  process_identity TEXT NOT NULL,
  registered_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  stopped_at TIMESTAMPTZ,
  stop_confirmation TEXT CHECK (stop_confirmation IN ('graceful', 'operator'))
);

CREATE TABLE notification_delivery_attempts (
  id UUID PRIMARY KEY,
  work_kind TEXT NOT NULL CHECK (work_kind IN ('account', 'system', 'reply')),
  work_id BIGINT NOT NULL CHECK (work_id > 0),
  owner_id UUID,
  sender_incarnation UUID NOT NULL,
  telegram_chat_id BIGINT NOT NULL CHECK (telegram_chat_id <> 0),
  telegram_group BOOLEAN NOT NULL,
  payload_digest BYTEA NOT NULL CHECK (octet_length(payload_digest) = 32),
  authorized_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  started_at TIMESTAMPTZ,
  result_at TIMESTAMPTZ,
  sender_returned_at TIMESTAMPTZ,
  sender_elapsed_ns BIGINT CHECK(sender_elapsed_ns>=0),
  CHECK(sender_elapsed_ns IS NULL OR sender_returned_at IS NOT NULL),
  message_id TEXT,
  outcome TEXT CHECK (outcome IN ('sent', 'retryable', 'failed', 'unknown')),
  outcome_code TEXT,
  retry_after INTERVAL,
  retry_after_released_at TIMESTAMPTZ,
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
    REFERENCES system_notification_topics (telegram_chat, label),
  payload BYTEA NOT NULL CHECK(octet_length(payload)>0),
  payload_digest BYTEA NOT NULL CHECK(octet_length(payload_digest)=32)
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
  CHECK (binding_revision IS NULL OR binding_revision > 0),
  payload BYTEA NOT NULL CHECK(octet_length(payload)>0)
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
    CHECK (channel = 'telegram'),
  payload BYTEA NOT NULL CHECK(octet_length(payload)>0),
  request_digest BYTEA NOT NULL CHECK(octet_length(request_digest)=32),
  activity_id BIGINT
);

CREATE INDEX idx_account_notification_deliveries_account_created
  ON account_notification_deliveries (account_id, created_at DESC);
CREATE INDEX idx_account_notification_deliveries_pending_ready
  ON account_notification_deliveries (status, next_attempt_at, id);

CREATE TABLE trader_sync_targets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  wallet BYTEA NOT NULL UNIQUE CHECK (octet_length(wallet) = 20),
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE TABLE trader_sync_target_confirmations (
  token_digest BYTEA PRIMARY KEY CHECK (octet_length(token_digest) = 32),
  owner_id UUID NOT NULL,
  identity_json JSONB NOT NULL CHECK (jsonb_typeof(identity_json) = 'object'),
  identity_digest BYTEA NOT NULL CHECK (octet_length(identity_digest) = 32),
  card_json JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(card_json) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_request_id TEXT CHECK (consumed_request_id IS NULL OR length(btrim(consumed_request_id)) > 0),
  CHECK (expires_at <= created_at + INTERVAL '5 minutes')
);
CREATE INDEX trader_sync_confirmations_owner_idx ON trader_sync_target_confirmations(owner_id);

CREATE TABLE trader_sync_request_results (
  owner_id UUID NOT NULL,
  operation TEXT NOT NULL CHECK (length(btrim(operation)) > 0),
  request_id TEXT NOT NULL CHECK (length(btrim(request_id)) > 0),
  payload_digest BYTEA NOT NULL CHECK (octet_length(payload_digest) = 32),
  result_json JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (owner_id, operation, request_id)
);

CREATE TABLE trader_sync_target_notes (
 owner_id UUID NOT NULL REFERENCES athena_account(account_id),
 wallet BYTEA NOT NULL CHECK (octet_length(wallet)=20),
 note TEXT NOT NULL CHECK (char_length(note)<=20),
 revision BIGINT NOT NULL CHECK (revision>0),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(owner_id,wallet)
);
CREATE TABLE trader_sync_subscriptions (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id UUID NOT NULL REFERENCES athena_account(account_id),
 wallet BYTEA NOT NULL CHECK (octet_length(wallet)=20),
 desired_state TEXT NOT NULL CHECK (desired_state IN ('enabled','paused','permission_disabled','cancelled')),
 observation_state TEXT NOT NULL CHECK (observation_state IN ('pending_baseline','healthy','interrupted')),
 reason TEXT NOT NULL DEFAULT '',
 revision BIGINT NOT NULL DEFAULT 1 CHECK (revision>0),
 activation_generation BIGINT NOT NULL DEFAULT 1 CHECK (activation_generation>0),
 effective_at TIMESTAMPTZ,
 ended_at TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(owner_id,id)
);
CREATE UNIQUE INDEX trader_sync_one_live_target ON trader_sync_subscriptions(owner_id,wallet) WHERE desired_state<>'cancelled';
CREATE TABLE trader_sync_baseline_attempts (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id UUID NOT NULL,
 subscription_id UUID NOT NULL,
 activation_generation BIGINT NOT NULL CHECK (activation_generation>0),
 collector_epoch BIGINT,
 filter_revision BIGINT,
 expected_revision BIGINT NOT NULL CHECK(expected_revision>0),
 registered_high BIGINT,
 candidate_effective_at TIMESTAMPTZ,
 state TEXT NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','succeeded','failed')),
 effective_at TIMESTAMPTZ,
 ended_at TIMESTAMPTZ,
 reason TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(owner_id,subscription_id) REFERENCES trader_sync_subscriptions(owner_id,id),
 UNIQUE(owner_id,subscription_id,id)
);
-- +goose StatementBegin
CREATE FUNCTION trader_sync_guard_attempt_terminal() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.state <> 'pending' AND NEW IS DISTINCT FROM OLD THEN
  RAISE EXCEPTION 'baseline attempt is terminal';
 END IF;
 RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER trader_sync_attempt_terminal BEFORE UPDATE ON trader_sync_baseline_attempts FOR EACH ROW EXECUTE FUNCTION trader_sync_guard_attempt_terminal();
CREATE TABLE trader_sync_monitor_intervals (
 last_reliable_at TIMESTAMPTZ,
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id UUID NOT NULL,
 subscription_id UUID NOT NULL,
 baseline_attempt_id UUID NOT NULL,
 activation_generation BIGINT NOT NULL CHECK(activation_generation>0),
 collector_epoch BIGINT NOT NULL,
 filter_revision BIGINT NOT NULL,
 expected_revision BIGINT NOT NULL CHECK(expected_revision>0),
 registered_high BIGINT NOT NULL,
 candidate_effective_at TIMESTAMPTZ NOT NULL,
 state TEXT NOT NULL DEFAULT 'open' CHECK(state IN ('open','closed')),
 effective_at TIMESTAMPTZ NOT NULL,
 ended_at TIMESTAMPTZ,
 reason TEXT NOT NULL DEFAULT '',
 FOREIGN KEY(owner_id,subscription_id) REFERENCES trader_sync_subscriptions(owner_id,id),
 FOREIGN KEY(owner_id,subscription_id,baseline_attempt_id) REFERENCES trader_sync_baseline_attempts(owner_id,subscription_id,id)
);

CREATE TABLE trader_sync_market_metadata (
 cache_key TEXT PRIMARY KEY CHECK(length(cache_key)>0),
 metadata_json JSONB NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE trader_sync_combo_leg_index (
 position_id TEXT NOT NULL CHECK(position_id ~ '^(0|[1-9][0-9]{0,77})$'),
 market_id TEXT NOT NULL CHECK(market_id ~ '^[1-9][0-9]*$'),
 condition_id TEXT NOT NULL CHECK(length(condition_id)>0),
 position_ids JSONB NOT NULL CHECK(jsonb_typeof(position_ids)='array'),
 seen_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(position_id,market_id)
);
CREATE TABLE trader_sync_directory_refresh (
 admission_id UUID,
 name TEXT PRIMARY KEY CHECK(name='combo_markets'),
 cursor TEXT NOT NULL DEFAULT '',
 visited_cursors TEXT[] NOT NULL DEFAULT '{}',
 round_started_at TIMESTAMPTZ,
 round_completed_at TIMESTAMPTZ,
 next_page_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

-- Live collector ownership is fenced by a monotonic token, never a time lease.
CREATE TABLE trader_sync_collector_control (
 singleton BOOLEAN PRIMARY KEY DEFAULT true CHECK(singleton),
 fencing_token BIGINT NOT NULL DEFAULT 0 CHECK(fencing_token>=0),
 owner_id UUID,
 active_epoch BIGINT
);
INSERT INTO trader_sync_collector_control(singleton) VALUES(true);
CREATE TABLE trader_sync_collector_epochs (
 id BIGSERIAL PRIMARY KEY,
 fencing_token BIGINT NOT NULL CHECK(fencing_token>0),
 started_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 ended_at TIMESTAMPTZ,
 reason TEXT NOT NULL DEFAULT '',
 filter_revision BIGINT NOT NULL DEFAULT 0 CHECK(filter_revision>=0),
 last_received_at TIMESTAMPTZ,
 last_read_sequence BIGINT NOT NULL DEFAULT 0 CHECK(last_read_sequence>=0)
);
ALTER TABLE trader_sync_collector_control ADD FOREIGN KEY(active_epoch) REFERENCES trader_sync_collector_epochs(id);
CREATE TABLE trader_sync_interruptions (
 id BIGSERIAL PRIMARY KEY,
 collector_epoch BIGINT NOT NULL UNIQUE REFERENCES trader_sync_collector_epochs(id),
 reason TEXT NOT NULL,
 recorded_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 last_received_at TIMESTAMPTZ,
 last_read_sequence BIGINT NOT NULL CHECK(last_read_sequence>=0)
);
ALTER TABLE trader_sync_baseline_attempts ADD COLUMN registered_sequence BIGINT CHECK(registered_sequence>=0);
CREATE UNIQUE INDEX trader_sync_one_pending_baseline ON trader_sync_baseline_attempts(subscription_id) WHERE state='pending';
CREATE UNIQUE INDEX trader_sync_one_interval_per_attempt ON trader_sync_monitor_intervals(baseline_attempt_id);
CREATE TABLE trader_sync_source_records (
 id BIGSERIAL PRIMARY KEY,
 chain_id BIGINT NOT NULL CHECK(chain_id=137),
 exchange_address BYTEA NOT NULL CHECK(octet_length(exchange_address)=20),
 wallet BYTEA NOT NULL CHECK(octet_length(wallet)=20),
 block_hash BYTEA NOT NULL CHECK(octet_length(block_hash)=32),
 transaction_hash BYTEA NOT NULL CHECK(octet_length(transaction_hash)=32),
 log_index BIGINT NOT NULL CHECK(log_index>=0),
 block_number BIGINT NOT NULL CHECK(block_number>=0),
 raw_json JSONB NOT NULL,
 collector_epoch BIGINT NOT NULL REFERENCES trader_sync_collector_epochs(id),
 read_sequence BIGINT NOT NULL CHECK(read_sequence>0),
 received_elapsed_ns BIGINT CHECK(received_elapsed_ns>=0),
 received_at TIMESTAMPTZ NOT NULL,
 removed BOOLEAN NOT NULL,
 confirmation_state TEXT NOT NULL DEFAULT 'unverified' CHECK(confirmation_state IN ('unverified','invalid','confirmed')),
 confirmation_reason TEXT NOT NULL DEFAULT '',
 checked_at TIMESTAMPTZ,
 settled_at TIMESTAMPTZ,
 source_version TEXT NOT NULL DEFAULT '',
 trade_json JSONB,
 UNIQUE(chain_id,exchange_address,block_hash,transaction_hash,log_index)
);
CREATE TABLE trader_sync_source_candidates (
 source_record_id BIGINT NOT NULL REFERENCES trader_sync_source_records(id),
 owner_id UUID NOT NULL,
 subscription_id UUID NOT NULL,
 activation_generation BIGINT NOT NULL CHECK(activation_generation>0),
 baseline_attempt_id UUID NOT NULL,
 received_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY(source_record_id,subscription_id),
 FOREIGN KEY(owner_id,subscription_id,baseline_attempt_id) REFERENCES trader_sync_baseline_attempts(owner_id,subscription_id,id)
);
CREATE INDEX trader_sync_sources_unverified ON trader_sync_source_records(id) WHERE confirmation_state='unverified';


ALTER TABLE trader_sync_subscriptions ADD COLUMN target_display JSONB NOT NULL;
ALTER TABLE trader_sync_source_candidates ADD COLUMN disposition TEXT NOT NULL DEFAULT 'pending' CHECK(disposition IN ('pending','projected','ineligible'));
ALTER TABLE trader_sync_source_candidates ADD COLUMN disposition_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE trader_sync_source_candidates ADD UNIQUE(owner_id,subscription_id,source_record_id);
ALTER TABLE trader_sync_source_records ADD COLUMN metadata_complete BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE trader_sync_monitor_intervals ADD UNIQUE(owner_id,subscription_id,id);
CREATE SEQUENCE trader_sync_activity_id_seq AS bigint INCREMENT BY 1 CACHE 1 NO CYCLE;
CREATE TABLE trader_sync_activities (
 id BIGINT PRIMARY KEY DEFAULT nextval('trader_sync_activity_id_seq'),
 owner_id UUID NOT NULL,
 subscription_id UUID NOT NULL,
 source_record_id BIGINT NOT NULL,
 interval_id UUID NOT NULL,
 activation_generation BIGINT NOT NULL CHECK(activation_generation>0),
 trade_json JSONB NOT NULL,
 metadata_key TEXT NOT NULL REFERENCES trader_sync_market_metadata(cache_key),
 target_display_snapshot JSONB NOT NULL,
 note_snapshot TEXT NOT NULL,
 notification_mode TEXT NOT NULL CHECK(notification_mode IN ('in_app_only','ordinary','summary')),
 notification_reason TEXT NOT NULL DEFAULT '',
 formation_evidence JSONB NOT NULL CHECK(jsonb_typeof(formation_evidence)='object'),
 settled_at TIMESTAMPTZ NOT NULL,
 received_at TIMESTAMPTZ NOT NULL,
 recorded_at TIMESTAMPTZ NOT NULL,
 UNIQUE(subscription_id,source_record_id),
 UNIQUE(owner_id,id),
 FOREIGN KEY(owner_id,subscription_id,source_record_id) REFERENCES trader_sync_source_candidates(owner_id,subscription_id,source_record_id),
 FOREIGN KEY(owner_id,subscription_id,interval_id) REFERENCES trader_sync_monitor_intervals(owner_id,subscription_id,id)
);
ALTER SEQUENCE trader_sync_activity_id_seq OWNED BY trader_sync_activities.id;
CREATE INDEX trader_sync_activity_owner_id ON trader_sync_activities(owner_id,id DESC);
CREATE INDEX trader_sync_activity_owner_time ON trader_sync_activities(owner_id,recorded_at);
-- +goose StatementBegin
CREATE FUNCTION trader_sync_guard_activity_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW IS DISTINCT FROM OLD THEN RAISE EXCEPTION 'activity facts are immutable'; END IF;
 RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER trader_sync_activity_immutable BEFORE UPDATE ON trader_sync_activities FOR EACH ROW EXECUTE FUNCTION trader_sync_guard_activity_immutable();
CREATE TABLE trader_sync_alert_memberships (
 activity_id BIGINT PRIMARY KEY,
 owner_id UUID NOT NULL,
 binding_revision BIGINT NOT NULL CHECK(binding_revision>0),
 chat_id BIGINT NOT NULL CHECK(chat_id>0),
 form TEXT NOT NULL CHECK(form IN ('ordinary','summary')),
 state TEXT NOT NULL CHECK(state IN ('waiting','frozen','cancelled')),
 created_at TIMESTAMPTZ NOT NULL,
 eligibility_revoked_at TIMESTAMPTZ,
 reason TEXT NOT NULL DEFAULT '',
 batch_id BIGINT,
 FOREIGN KEY(owner_id,activity_id) REFERENCES trader_sync_activities(owner_id,id),
 CHECK(form='summary' OR state<>'waiting'),
 CHECK(batch_id IS NULL OR form='summary')
);
CREATE INDEX trader_sync_waiting_memberships ON trader_sync_alert_memberships(owner_id,activity_id) WHERE state='waiting' AND eligibility_revoked_at IS NULL;
ALTER TABLE account_notification_deliveries ADD FOREIGN KEY(account_id,activity_id) REFERENCES trader_sync_activities(owner_id,id);
CREATE UNIQUE INDEX account_notification_activity ON account_notification_deliveries(activity_id) WHERE activity_id IS NOT NULL;
CREATE TABLE trader_sync_finality_anomalies (
 chain_id BIGINT NOT NULL CHECK(chain_id=137),
 transaction_hash BYTEA NOT NULL CHECK(octet_length(transaction_hash)=32),
 published_block_hash BYTEA NOT NULL CHECK(octet_length(published_block_hash)=32),
 conflicting_block_hash BYTEA CHECK(conflicting_block_hash IS NULL OR octet_length(conflicting_block_hash)=32),
 reason TEXT NOT NULL,
 detected_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(chain_id,transaction_hash)
);

CREATE TABLE trader_sync_summary_batches (
 id BIGSERIAL PRIMARY KEY,
 owner_id UUID NOT NULL,
 binding_revision BIGINT NOT NULL CHECK(binding_revision>0),
 chat_id BIGINT NOT NULL CHECK(chat_id>0),
 oldest_at TIMESTAMPTZ NOT NULL,
 frozen_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 sealed BOOLEAN NOT NULL DEFAULT false,
 first_started_at TIMESTAMPTZ,
 recovery_basis_at TIMESTAMPTZ,
 recovery_reason TEXT NOT NULL DEFAULT '',
 budget_wait_started_at TIMESTAMPTZ,
 budget_wait_ended_at TIMESTAMPTZ,
 budget_wait_ms BIGINT NOT NULL DEFAULT 0 CHECK(budget_wait_ms>=0),
 budget_reason TEXT NOT NULL DEFAULT '',
 local_gate_wait_ms BIGINT NOT NULL DEFAULT 0 CHECK(local_gate_wait_ms>=0),
 start_evidence_missing BOOLEAN NOT NULL DEFAULT false,
 UNIQUE(owner_id,id),
 UNIQUE(owner_id,id,binding_revision,chat_id)
);
ALTER TABLE trader_sync_alert_memberships ADD UNIQUE(owner_id,batch_id,activity_id);
ALTER TABLE trader_sync_alert_memberships ADD CONSTRAINT summary_membership_batch_fkey FOREIGN KEY(owner_id,batch_id,binding_revision,chat_id) REFERENCES trader_sync_summary_batches(owner_id,id,binding_revision,chat_id);
ALTER TABLE account_notification_deliveries ADD UNIQUE(account_id,id);
CREATE TABLE trader_sync_summary_parts (
 id BIGSERIAL PRIMARY KEY,
 owner_id UUID NOT NULL,
 batch_id BIGINT NOT NULL,
 part_index INTEGER NOT NULL CHECK(part_index>0),
 total INTEGER NOT NULL CHECK(total>=part_index),
 text TEXT NOT NULL CHECK(length(text)>0),
 payload_digest BYTEA NOT NULL CHECK(octet_length(payload_digest)=32),
 delivery_id BIGINT NOT NULL UNIQUE,
 UNIQUE(batch_id,part_index),
 UNIQUE(owner_id,batch_id,id),
 FOREIGN KEY(owner_id,batch_id) REFERENCES trader_sync_summary_batches(owner_id,id),
 FOREIGN KEY(owner_id,delivery_id) REFERENCES account_notification_deliveries(account_id,id)
);
CREATE TABLE trader_sync_summary_part_items (
 owner_id UUID NOT NULL,
 batch_id BIGINT NOT NULL,
 part_id BIGINT NOT NULL,
 activity_id BIGINT NOT NULL,
 PRIMARY KEY(part_id,activity_id),
 FOREIGN KEY(owner_id,batch_id,part_id) REFERENCES trader_sync_summary_parts(owner_id,batch_id,id),
 FOREIGN KEY(owner_id,batch_id,activity_id) REFERENCES trader_sync_alert_memberships(owner_id,batch_id,activity_id)
);
CREATE TABLE trader_sync_summary_heads (
 id BIGSERIAL PRIMARY KEY,
 owner_id UUID NOT NULL UNIQUE,
 current_batch_id BIGINT,
 current_attempt_id UUID REFERENCES notification_delivery_attempts(id),
 previous_basis_at TIMESTAMPTZ,
 FOREIGN KEY(owner_id,current_batch_id) REFERENCES trader_sync_summary_batches(owner_id,id),
 CHECK(current_attempt_id IS NULL OR current_batch_id IS NOT NULL)
);
-- +goose StatementBegin
CREATE FUNCTION trader_sync_guard_summary_batch() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF (NEW.id,NEW.owner_id,NEW.binding_revision,NEW.chat_id,NEW.oldest_at,NEW.frozen_at) IS DISTINCT FROM
    (OLD.id,OLD.owner_id,OLD.binding_revision,OLD.chat_id,OLD.oldest_at,OLD.frozen_at)
    OR (OLD.sealed AND NOT NEW.sealed)
    OR (OLD.first_started_at IS NOT NULL AND NEW.first_started_at IS DISTINCT FROM OLD.first_started_at)
 THEN RAISE EXCEPTION 'summary batch facts are immutable'; END IF;
 RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER summary_batch_immutable BEFORE UPDATE ON trader_sync_summary_batches FOR EACH ROW EXECUTE FUNCTION trader_sync_guard_summary_batch();
-- +goose StatementBegin
CREATE FUNCTION trader_sync_guard_summary_member() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF (NEW.activity_id,NEW.owner_id,NEW.binding_revision,NEW.chat_id,NEW.form,NEW.created_at) IS DISTINCT FROM
    (OLD.activity_id,OLD.owner_id,OLD.binding_revision,OLD.chat_id,OLD.form,OLD.created_at)
    OR (OLD.batch_id IS NOT NULL AND (NEW.batch_id IS DISTINCT FROM OLD.batch_id OR NEW.state IS DISTINCT FROM OLD.state))
    OR (OLD.eligibility_revoked_at IS NOT NULL AND NEW.eligibility_revoked_at IS DISTINCT FROM OLD.eligibility_revoked_at)
 THEN RAISE EXCEPTION 'summary member identity is immutable'; END IF;
 IF NEW.batch_id IS DISTINCT FROM OLD.batch_id AND EXISTS(SELECT 1 FROM trader_sync_summary_batches WHERE id=NEW.batch_id AND sealed)
 THEN RAISE EXCEPTION 'summary batch is sealed'; END IF;
 RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER summary_member_immutable BEFORE UPDATE ON trader_sync_alert_memberships FOR EACH ROW EXECUTE FUNCTION trader_sync_guard_summary_member();
-- +goose StatementBegin
CREATE FUNCTION trader_sync_guard_summary_content() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP<>'INSERT' THEN RAISE EXCEPTION 'summary content is immutable'; END IF;
 IF EXISTS(SELECT 1 FROM trader_sync_summary_batches WHERE id=NEW.batch_id AND sealed)
 THEN RAISE EXCEPTION 'summary batch is sealed'; END IF;
 RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER summary_part_immutable BEFORE INSERT OR UPDATE OR DELETE ON trader_sync_summary_parts FOR EACH ROW EXECUTE FUNCTION trader_sync_guard_summary_content();
CREATE TRIGGER summary_part_items_immutable BEFORE INSERT OR UPDATE OR DELETE ON trader_sync_summary_part_items FOR EACH ROW EXECUTE FUNCTION trader_sync_guard_summary_content();

-- Every attempt sends the already frozen format/text bytes; status updates cannot rewrite them.
-- +goose StatementBegin
CREATE FUNCTION notification_guard_frozen_payload() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.payload IS DISTINCT FROM OLD.payload OR NEW.payload_digest IS DISTINCT FROM OLD.payload_digest THEN
  RAISE EXCEPTION 'notification payload is immutable';
 END IF;
 RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER system_notification_frozen_payload BEFORE UPDATE ON system_notification_deliveries FOR EACH ROW EXECUTE FUNCTION notification_guard_frozen_payload();
CREATE TRIGGER account_notification_frozen_payload BEFORE UPDATE ON account_notification_deliveries FOR EACH ROW EXECUTE FUNCTION notification_guard_frozen_payload();
CREATE TRIGGER binding_reply_frozen_payload BEFORE UPDATE ON telegram_binding_replies FOR EACH ROW EXECUTE FUNCTION notification_guard_frozen_payload();

-- Safe common observation facts; no note, target display, activity or delivery payload.
-- Ordinary views keep owner/subscription restrictions available to the planner.
CREATE VIEW trader_sync_subscription_interruptions AS
SELECT rel.owner_id,rel.subscription_id,rel.activation_generation,rel.epoch_id,
 rel.id,rel.recorded_at,rel.ended_at,rel.reason,rel.awaiting_cleanup,
 jsonb_build_object('ID',rel.id::text,'Start',NULL,'End',recovered.effective_at,'RecoveredAt',recovered.effective_at,
 'Reason',rel.reason,'Uncertainty','actual_start_unknown;recorded_stop_boundary_available' ||
 CASE WHEN recovered.effective_at<rel.ended_at THEN ';clock_order_uncertain' ELSE '' END,'PossibleMissing',true)::jsonb AS interruption_json
FROM (
 SELECT DISTINCT ba.owner_id,ba.subscription_id,ba.activation_generation,ep.id AS epoch_id,
 ir.id,ir.recorded_at,ep.ended_at,ep.reason,
 (ba.state='pending' OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id AND i.ended_at IS NULL)) AS awaiting_cleanup
 FROM trader_sync_baseline_attempts ba
 JOIN trader_sync_collector_epochs ep ON ep.id=ba.collector_epoch AND ep.ended_at IS NOT NULL
 JOIN trader_sync_interruptions ir ON ir.collector_epoch=ep.id
 WHERE ba.state='pending' OR (ba.state='failed' AND ba.ended_at=ep.ended_at AND ba.reason=ep.reason)
 OR EXISTS(SELECT 1 FROM trader_sync_monitor_intervals i WHERE i.baseline_attempt_id=ba.id
 AND (i.ended_at IS NULL OR (i.ended_at=ep.ended_at AND i.reason=ep.reason)))
) rel
LEFT JOIN LATERAL (
 SELECT i.effective_at FROM trader_sync_monitor_intervals i
 JOIN trader_sync_baseline_attempts ba ON ba.id=i.baseline_attempt_id AND ba.state='succeeded'
 WHERE i.owner_id=rel.owner_id AND i.subscription_id=rel.subscription_id AND i.activation_generation=rel.activation_generation
 AND i.collector_epoch>rel.epoch_id ORDER BY i.collector_epoch,ba.created_at,ba.id LIMIT 1
) recovered ON true;

CREATE VIEW trader_sync_subscription_observations AS
SELECT s.owner_id,s.id AS subscription_id,
 CASE WHEN s.desired_state='enabled' THEN state.observation_state ELSE s.desired_state END::text AS status,
 jsonb_build_object('State',state.observation_state,
 'Reason',CASE WHEN logical.interrupted THEN 'collector_interrupted'
 WHEN s.observation_state='pending_baseline' THEN COALESCE((SELECT ba.reason FROM trader_sync_baseline_attempts ba
 WHERE ba.owner_id=s.owner_id AND ba.subscription_id=s.id AND ba.activation_generation=s.activation_generation AND ba.state='failed'
 ORDER BY ba.created_at DESC,ba.id DESC LIMIT 1),s.reason) ELSE s.reason END,
 'LastReliableAt',(SELECT i.last_reliable_at FROM trader_sync_monitor_intervals i
 WHERE i.owner_id=s.owner_id AND i.subscription_id=s.id AND i.activation_generation=s.activation_generation AND i.last_reliable_at IS NOT NULL
 ORDER BY i.collector_epoch DESC,i.effective_at DESC,i.id DESC LIMIT 1),
 'InterruptionCount',(SELECT count(DISTINCT rel.epoch_id) FROM trader_sync_subscription_interruptions rel WHERE rel.owner_id=s.owner_id AND rel.subscription_id=s.id),
 'LatestInterruption',(SELECT rel.interruption_json FROM trader_sync_subscription_interruptions rel
 WHERE rel.owner_id=s.owner_id AND rel.subscription_id=s.id ORDER BY rel.recorded_at DESC,rel.id DESC LIMIT 1))::jsonb AS observation_json
FROM trader_sync_subscriptions s
CROSS JOIN LATERAL (SELECT s.desired_state='enabled' AND EXISTS(
 SELECT 1 FROM trader_sync_subscription_interruptions rel
 WHERE rel.owner_id=s.owner_id AND rel.subscription_id=s.id AND rel.activation_generation=s.activation_generation AND rel.awaiting_cleanup
) AS interrupted) logical
CROSS JOIN LATERAL (SELECT CASE WHEN logical.interrupted THEN 'interrupted' ELSE s.observation_state END AS observation_state) state;

-- +goose Down
DROP VIEW trader_sync_subscription_observations;
DROP VIEW trader_sync_subscription_interruptions;
DROP TABLE trader_sync_summary_heads;
DROP TABLE trader_sync_summary_part_items;
DROP TABLE trader_sync_summary_parts;
DROP FUNCTION trader_sync_guard_summary_content();
ALTER TABLE trader_sync_alert_memberships DROP CONSTRAINT summary_membership_batch_fkey;
DROP TABLE trader_sync_summary_batches;
DROP FUNCTION trader_sync_guard_summary_batch();
DROP TRIGGER summary_member_immutable ON trader_sync_alert_memberships;
DROP FUNCTION trader_sync_guard_summary_member();
ALTER TABLE account_notification_deliveries DROP CONSTRAINT account_notification_deliveries_account_id_activity_id_fkey;
DROP TABLE trader_sync_finality_anomalies;
DROP TABLE trader_sync_alert_memberships;
DROP TABLE trader_sync_activities;
DROP FUNCTION trader_sync_guard_activity_immutable();
DROP TABLE IF EXISTS trader_sync_source_candidates;
DROP TABLE IF EXISTS trader_sync_source_records;
DROP TABLE IF EXISTS trader_sync_collector_control;
DROP TABLE IF EXISTS trader_sync_interruptions;
DROP TABLE IF EXISTS trader_sync_collector_epochs;
DROP TABLE trader_sync_directory_refresh;
DROP TABLE trader_sync_combo_leg_index;
DROP TABLE trader_sync_market_metadata;
DROP TABLE trader_sync_monitor_intervals;
DROP TABLE trader_sync_baseline_attempts;
DROP FUNCTION trader_sync_guard_attempt_terminal();
DROP TABLE trader_sync_subscriptions;
DROP TABLE trader_sync_target_notes;

DROP TABLE trader_sync_request_results;
DROP TABLE trader_sync_target_confirmations;
DROP TABLE trader_sync_targets;

DROP TABLE account_notification_deliveries;
DROP TABLE telegram_binding_replies;
DROP TABLE telegram_consumed_updates;
DROP TABLE telegram_polling_state;
DROP TABLE telegram_binding_attempts;
DROP TABLE telegram_bindings;
DROP TABLE telegram_binding_versions;
DROP TABLE system_notification_deliveries;
DROP TABLE notification_delivery_attempts;
DROP FUNCTION notification_guard_frozen_payload();
DROP TABLE notification_sender_instances;
DROP TABLE system_notification_topics;

DROP TABLE account_api_key;
DROP TABLE account_preferences;
DROP TABLE account_profile;
DROP TABLE account_module_access;
DROP TABLE account_access;
DROP TABLE athena_account;
DROP FUNCTION athena_is_canonical_solana_public_key(TEXT);
