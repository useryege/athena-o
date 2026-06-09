
-- +goose Up

CREATE TABLE IF NOT EXISTS notification_topics (
  label TEXT PRIMARY KEY,
  message_thread_id INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

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
  topic_label TEXT NOT NULL REFERENCES notification_topics (label),
  attempts INT NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_attempt_at TIMESTAMPTZ,
  locked_at TIMESTAMPTZ,
  locked_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_created_at ON notification_deliveries (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_status ON notification_deliveries (status);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_severity ON notification_deliveries (severity);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_source ON notification_deliveries (source);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_topic_label ON notification_deliveries (topic_label);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_pending_ready
  ON notification_deliveries (status, next_attempt_at, id);

-- +goose Down

DROP TABLE IF EXISTS notification_deliveries;
DROP TABLE IF EXISTS notification_topics;
