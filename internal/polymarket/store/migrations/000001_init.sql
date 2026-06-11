
-- +goose Up

CREATE TABLE IF NOT EXISTS polymarket_sports_live_market (
  condition_id TEXT PRIMARY KEY,
  market_slug TEXT NOT NULL DEFAULT '',
  event_slug TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  image TEXT NOT NULL DEFAULT '',
  score TEXT NOT NULL DEFAULT '',
  period TEXT NOT NULL DEFAULT '',
  elapsed TEXT NOT NULL DEFAULT '',
  gamma_updated_at TIMESTAMPTZ,
  liquidity_num DOUBLE PRECISION NOT NULL DEFAULT 0,
  volume_num DOUBLE PRECISION NOT NULL DEFAULT 0,
  fetched_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT polymarket_sports_live_market_condition_id_not_empty CHECK (btrim(condition_id) <> ''),
  CONSTRAINT polymarket_sports_live_market_liquidity_nonnegative CHECK (liquidity_num >= 0),
  CONSTRAINT polymarket_sports_live_market_volume_nonnegative CHECK (volume_num >= 0)
);

CREATE TABLE IF NOT EXISTS polymarket_sync_state (
  sync_name TEXT PRIMARY KEY,
  last_success_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT polymarket_sync_state_name_not_empty CHECK (btrim(sync_name) <> '')
);

CREATE INDEX IF NOT EXISTS polymarket_sports_live_market_sort_idx
  ON polymarket_sports_live_market (gamma_updated_at DESC NULLS LAST, liquidity_num DESC, condition_id);

CREATE INDEX IF NOT EXISTS polymarket_sports_live_market_last_seen_idx
  ON polymarket_sports_live_market (last_seen_at);

-- +goose Down

DROP TABLE IF EXISTS polymarket_sync_state;
DROP TABLE IF EXISTS polymarket_sports_live_market;
