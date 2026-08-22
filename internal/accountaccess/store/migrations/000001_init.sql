-- +goose Up

CREATE TABLE account_enabled_override (
  account_name TEXT PRIMARY KEY,
  enabled BOOLEAN NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down

DROP TABLE IF EXISTS account_enabled_override;
