
-- +goose Up

CREATE TABLE IF NOT EXISTS notification_topics (
  telegram_chat TEXT NOT NULL,
  label TEXT NOT NULL,
  message_thread_id INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (telegram_chat, label)
);

INSERT INTO notification_topics (telegram_chat, label, message_thread_id)
VALUES ('prod', '[POLY] UMA Disputed', 559)
ON CONFLICT (telegram_chat, label) DO UPDATE
SET message_thread_id = EXCLUDED.message_thread_id;

CREATE TABLE IF NOT EXISTS notification_deliveries (
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
  attempts INT NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_attempt_at TIMESTAMPTZ,
  locked_at TIMESTAMPTZ,
  locked_by TEXT,
  CONSTRAINT notification_deliveries_topic_fk
    FOREIGN KEY (telegram_chat, topic_label)
    REFERENCES notification_topics (telegram_chat, label)
);

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_created_at ON notification_deliveries (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_status ON notification_deliveries (status);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_severity ON notification_deliveries (severity);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_source ON notification_deliveries (source);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_telegram_chat ON notification_deliveries (telegram_chat);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_topic_label ON notification_deliveries (telegram_chat, topic_label);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_pending_ready
  ON notification_deliveries (status, next_attempt_at, id);

-- +goose Down

DROP TABLE IF EXISTS notification_deliveries;
DROP TABLE IF EXISTS notification_topics;
