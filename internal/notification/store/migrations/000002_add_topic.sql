
-- +goose Up

ALTER TABLE notification_deliveries
  ADD COLUMN IF NOT EXISTS topic TEXT NOT NULL DEFAULT 'token';

ALTER TABLE notification_deliveries
  ALTER COLUMN topic DROP DEFAULT;

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_topic ON notification_deliveries (topic);

-- +goose Down

DROP INDEX IF EXISTS idx_notification_deliveries_topic;

ALTER TABLE notification_deliveries
  DROP COLUMN IF EXISTS topic;
