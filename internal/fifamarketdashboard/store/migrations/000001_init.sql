-- +goose Up

CREATE TABLE IF NOT EXISTS fifa_market_dashboard_event_config (
  singleton BOOLEAN PRIMARY KEY DEFAULT true,
  worm_event_id TEXT NOT NULL,
  event_ref TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT fifa_market_dashboard_event_config_singleton CHECK (singleton),
  CONSTRAINT fifa_market_dashboard_event_config_worm_event_id_not_empty CHECK (btrim(worm_event_id) <> ''),
  CONSTRAINT fifa_market_dashboard_event_config_event_ref_not_empty CHECK (btrim(event_ref) <> '')
);

-- +goose Down

DROP TABLE IF EXISTS fifa_market_dashboard_event_config;
