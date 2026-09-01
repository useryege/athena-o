
-- +goose Up

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
  CONSTRAINT system_notification_deliveries_status_check
    CHECK (status IN ('pending', 'sent', 'failed')),
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
  CONSTRAINT account_notification_deliveries_idempotency_unique
    UNIQUE (account_id, source, idempotency_key),
  CONSTRAINT account_notification_deliveries_payload_digest_check
    CHECK (octet_length(payload_digest) = 32),
  CONSTRAINT account_notification_deliveries_status_check
    CHECK (status IN ('pending', 'sent', 'failed', 'cancelled')),
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
DROP TABLE telegram_polling_state;
DROP TABLE telegram_binding_attempts;
DROP TABLE telegram_bindings;
DROP TABLE telegram_binding_versions;
DROP TABLE system_notification_deliveries;
DROP TABLE system_notification_topics;
