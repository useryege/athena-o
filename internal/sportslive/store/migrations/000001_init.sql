-- +goose Up

CREATE TABLE IF NOT EXISTS sports_live_event (
  event_key TEXT PRIMARY KEY,
  event_id TEXT NOT NULL DEFAULT '',
  ticker TEXT NOT NULL DEFAULT '',
  slug TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  resolution_source TEXT NOT NULL DEFAULT '',
  start_date TIMESTAMPTZ,
  creation_date TIMESTAMPTZ,
  end_date TIMESTAMPTZ,
  start_time TIMESTAMPTZ,
  created_at_gamma TIMESTAMPTZ,
  updated_at_gamma TIMESTAMPTZ,
  image TEXT NOT NULL DEFAULT '',
  icon TEXT NOT NULL DEFAULT '',
  active BOOLEAN NOT NULL DEFAULT false,
  closed BOOLEAN NOT NULL DEFAULT false,
  archived BOOLEAN NOT NULL DEFAULT false,
  featured BOOLEAN NOT NULL DEFAULT false,
  restricted BOOLEAN NOT NULL DEFAULT false,
  live BOOLEAN NOT NULL DEFAULT false,
  ended BOOLEAN NOT NULL DEFAULT false,
  liquidity DOUBLE PRECISION NOT NULL DEFAULT 0,
  volume DOUBLE PRECISION NOT NULL DEFAULT 0,
  open_interest DOUBLE PRECISION NOT NULL DEFAULT 0,
  category TEXT NOT NULL DEFAULT '',
  score TEXT NOT NULL DEFAULT '',
  period TEXT NOT NULL DEFAULT '',
  elapsed TEXT NOT NULL DEFAULT '',
  finished_timestamp TEXT NOT NULL DEFAULT '',
  game_id BIGINT,
  event_date TEXT NOT NULL DEFAULT '',
  game_status TEXT NOT NULL DEFAULT '',
  comment_count BIGINT NOT NULL DEFAULT 0,
  sport JSONB NOT NULL DEFAULT '{}'::jsonb,
  teams JSONB NOT NULL DEFAULT '[]'::jsonb,
  tags JSONB NOT NULL DEFAULT '[]'::jsonb,
  raw JSONB NOT NULL DEFAULT '{}'::jsonb,
  fetched_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT sports_live_event_key_not_empty CHECK (btrim(event_key) <> ''),
  CONSTRAINT sports_live_event_raw_object CHECK (jsonb_typeof(raw) = 'object'),
  CONSTRAINT sports_live_event_sport_object CHECK (jsonb_typeof(sport) = 'object'),
  CONSTRAINT sports_live_event_teams_array CHECK (jsonb_typeof(teams) = 'array'),
  CONSTRAINT sports_live_event_tags_array CHECK (jsonb_typeof(tags) = 'array')
);

CREATE TABLE IF NOT EXISTS sports_live_market (
  market_key TEXT PRIMARY KEY,
  event_key TEXT NOT NULL,
  event_id TEXT NOT NULL DEFAULT '',
  event_slug TEXT NOT NULL DEFAULT '',
  market_id TEXT NOT NULL DEFAULT '',
  condition_id TEXT NOT NULL DEFAULT '',
  slug TEXT NOT NULL DEFAULT '',
  question TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  resolution_source TEXT NOT NULL DEFAULT '',
  sports_market_type TEXT NOT NULL DEFAULT '',
  group_item_title TEXT NOT NULL DEFAULT '',
  image TEXT NOT NULL DEFAULT '',
  icon TEXT NOT NULL DEFAULT '',
  outcomes TEXT NOT NULL DEFAULT '',
  outcome_prices TEXT NOT NULL DEFAULT '',
  clob_token_ids TEXT NOT NULL DEFAULT '',
  active BOOLEAN NOT NULL DEFAULT false,
  closed BOOLEAN NOT NULL DEFAULT false,
  archived BOOLEAN NOT NULL DEFAULT false,
  restricted BOOLEAN NOT NULL DEFAULT false,
  enable_order_book BOOLEAN NOT NULL DEFAULT false,
  volume TEXT NOT NULL DEFAULT '',
  volume_num DOUBLE PRECISION NOT NULL DEFAULT 0,
  liquidity_num DOUBLE PRECISION NOT NULL DEFAULT 0,
  volume_24hr DOUBLE PRECISION NOT NULL DEFAULT 0,
  volume_1wk DOUBLE PRECISION NOT NULL DEFAULT 0,
  volume_1mo DOUBLE PRECISION NOT NULL DEFAULT 0,
  volume_1yr DOUBLE PRECISION NOT NULL DEFAULT 0,
  spread DOUBLE PRECISION NOT NULL DEFAULT 0,
  best_bid DOUBLE PRECISION NOT NULL DEFAULT 0,
  best_ask DOUBLE PRECISION NOT NULL DEFAULT 0,
  last_trade_price DOUBLE PRECISION NOT NULL DEFAULT 0,
  start_date TIMESTAMPTZ,
  end_date TIMESTAMPTZ,
  created_at_gamma TIMESTAMPTZ,
  updated_at_gamma TIMESTAMPTZ,
  tags JSONB NOT NULL DEFAULT '[]'::jsonb,
  raw JSONB NOT NULL DEFAULT '{}'::jsonb,
  fetched_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT sports_live_market_key_not_empty CHECK (btrim(market_key) <> ''),
  CONSTRAINT sports_live_market_raw_object CHECK (jsonb_typeof(raw) = 'object'),
  CONSTRAINT sports_live_market_tags_array CHECK (jsonb_typeof(tags) = 'array'),
  CONSTRAINT sports_live_market_event_fk
    FOREIGN KEY (event_key) REFERENCES sports_live_event(event_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sports_live_sync_state (
  sync_name TEXT PRIMARY KEY,
  last_success_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT sports_live_sync_state_name_not_empty CHECK (btrim(sync_name) <> '')
);

CREATE TABLE IF NOT EXISTS sports_live_price_point (
  token_id TEXT NOT NULL,
  market_key TEXT NOT NULL,
  event_key TEXT NOT NULL,
  condition_id TEXT NOT NULL DEFAULT '',
  outcome TEXT NOT NULL DEFAULT '',
  price_ts TIMESTAMPTZ NOT NULL,
  price DOUBLE PRECISION NOT NULL,
  fetched_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (token_id, price_ts),
  CONSTRAINT sports_live_price_point_token_not_empty CHECK (btrim(token_id) <> ''),
  CONSTRAINT sports_live_price_point_market_not_empty CHECK (btrim(market_key) <> ''),
  CONSTRAINT sports_live_price_point_event_not_empty CHECK (btrim(event_key) <> ''),
  CONSTRAINT sports_live_price_point_market_fk
    FOREIGN KEY (market_key) REFERENCES sports_live_market(market_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sports_live_price_alert_state (
  token_id TEXT PRIMARY KEY,
  market_key TEXT NOT NULL,
  event_key TEXT NOT NULL,
  condition_id TEXT NOT NULL DEFAULT '',
  outcome TEXT NOT NULL DEFAULT '',
  alert_band TEXT NOT NULL,
  last_alerted_at TIMESTAMPTZ NOT NULL,
  last_price_ts TIMESTAMPTZ NOT NULL,
  last_price DOUBLE PRECISION NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT sports_live_price_alert_state_token_not_empty CHECK (btrim(token_id) <> ''),
  CONSTRAINT sports_live_price_alert_state_market_not_empty CHECK (btrim(market_key) <> ''),
  CONSTRAINT sports_live_price_alert_state_event_not_empty CHECK (btrim(event_key) <> ''),
  CONSTRAINT sports_live_price_alert_state_band CHECK (alert_band IN ('a', 'b', 'c', 'd', 'e')),
  CONSTRAINT sports_live_price_alert_state_price CHECK (last_price >= 0 AND last_price <= 1),
  CONSTRAINT sports_live_price_alert_state_market_fk
    FOREIGN KEY (market_key) REFERENCES sports_live_market(market_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sports_live_score_alert_state (
  event_key TEXT PRIMARY KEY,
  last_score TEXT NOT NULL,
  notification_id BIGINT NOT NULL DEFAULT 0,
  last_notified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT sports_live_score_alert_state_event_not_empty CHECK (btrim(event_key) <> ''),
  CONSTRAINT sports_live_score_alert_state_score_not_empty CHECK (btrim(last_score) <> ''),
  CONSTRAINT sports_live_score_alert_state_event_fk
    FOREIGN KEY (event_key) REFERENCES sports_live_event(event_key) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS sports_live_event_slug_idx
  ON sports_live_event (slug)
  WHERE slug <> '';

CREATE INDEX IF NOT EXISTS sports_live_event_live_sort_idx
  ON sports_live_event (live DESC, updated_at_gamma DESC NULLS LAST, event_key);

CREATE INDEX IF NOT EXISTS sports_live_event_last_seen_idx
  ON sports_live_event (last_seen_at);

CREATE INDEX IF NOT EXISTS sports_live_market_event_idx
  ON sports_live_market (event_key);

CREATE INDEX IF NOT EXISTS sports_live_market_condition_idx
  ON sports_live_market (condition_id)
  WHERE condition_id <> '';

CREATE INDEX IF NOT EXISTS sports_live_market_price_sort_idx
  ON sports_live_market (updated_at_gamma DESC NULLS LAST, liquidity_num DESC, market_key);

CREATE INDEX IF NOT EXISTS sports_live_market_last_seen_idx
  ON sports_live_market (last_seen_at);

CREATE INDEX IF NOT EXISTS sports_live_price_point_market_ts_idx
  ON sports_live_price_point (market_key, price_ts DESC);

CREATE INDEX IF NOT EXISTS sports_live_price_point_event_ts_idx
  ON sports_live_price_point (event_key, price_ts DESC);

CREATE INDEX IF NOT EXISTS sports_live_price_point_fetched_idx
  ON sports_live_price_point (fetched_at);

CREATE INDEX IF NOT EXISTS sports_live_price_alert_state_alerted_idx
  ON sports_live_price_alert_state (last_alerted_at);

-- +goose Down

DROP TABLE IF EXISTS sports_live_score_alert_state;
DROP TABLE IF EXISTS sports_live_price_alert_state;
DROP TABLE IF EXISTS sports_live_price_point;
DROP TABLE IF EXISTS sports_live_sync_state;
DROP TABLE IF EXISTS sports_live_market;
DROP TABLE IF EXISTS sports_live_event;
