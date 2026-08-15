-- +goose Up

CREATE TABLE IF NOT EXISTS sports_history_sync_state (
  sync_name TEXT PRIMARY KEY,
  last_success_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT sports_history_sync_state_name_not_empty CHECK (btrim(sync_name) <> '')
);

CREATE TABLE IF NOT EXISTS sports_history_event (
  event_key TEXT PRIMARY KEY,
  event_id TEXT NOT NULL DEFAULT '',
  league TEXT NOT NULL,
  slug TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  image TEXT NOT NULL DEFAULT '',
  icon TEXT NOT NULL DEFAULT '',
  score TEXT NOT NULL DEFAULT '',
  period TEXT NOT NULL DEFAULT '',
  elapsed TEXT NOT NULL DEFAULT '',
  game_status TEXT NOT NULL DEFAULT '',
  start_time TIMESTAMPTZ NOT NULL,
  finished_at TIMESTAMPTZ NOT NULL,
  updated_at_gamma TIMESTAMPTZ,
  liquidity DOUBLE PRECISION NOT NULL DEFAULT 0,
  volume DOUBLE PRECISION NOT NULL DEFAULT 0,
  teams JSONB NOT NULL DEFAULT '[]'::jsonb,
  raw JSONB NOT NULL DEFAULT '{}'::jsonb,
  fetched_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT sports_history_event_key_not_empty CHECK (btrim(event_key) <> ''),
  CONSTRAINT sports_history_event_league CHECK (league IN ('ATP', 'WTA')),
  CONSTRAINT sports_history_event_teams_array CHECK (jsonb_typeof(teams) = 'array'),
  CONSTRAINT sports_history_event_raw_object CHECK (jsonb_typeof(raw) = 'object')
);

CREATE TABLE IF NOT EXISTS sports_history_market (
  market_key TEXT PRIMARY KEY,
  event_key TEXT NOT NULL,
  condition_id TEXT NOT NULL DEFAULT '',
  slug TEXT NOT NULL DEFAULT '',
  question TEXT NOT NULL DEFAULT '',
  sports_market_type TEXT NOT NULL DEFAULT '',
  outcomes TEXT NOT NULL DEFAULT '',
  outcome_prices TEXT NOT NULL DEFAULT '',
  clob_token_ids TEXT NOT NULL DEFAULT '',
  best_bid DOUBLE PRECISION NOT NULL DEFAULT 0,
  best_ask DOUBLE PRECISION NOT NULL DEFAULT 0,
  last_trade_price DOUBLE PRECISION NOT NULL DEFAULT 0,
  spread DOUBLE PRECISION NOT NULL DEFAULT 0,
  liquidity_num DOUBLE PRECISION NOT NULL DEFAULT 0,
  volume_num DOUBLE PRECISION NOT NULL DEFAULT 0,
  updated_at_gamma TIMESTAMPTZ,
  raw JSONB NOT NULL DEFAULT '{}'::jsonb,
  fetched_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT sports_history_market_key_not_empty CHECK (btrim(market_key) <> ''),
  CONSTRAINT sports_history_market_raw_object CHECK (jsonb_typeof(raw) = 'object'),
  CONSTRAINT sports_history_market_event_fk
    FOREIGN KEY (event_key) REFERENCES sports_history_event(event_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sports_history_price_point (
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
  CONSTRAINT sports_history_price_point_token_not_empty CHECK (btrim(token_id) <> ''),
  CONSTRAINT sports_history_price_point_market_not_empty CHECK (btrim(market_key) <> ''),
  CONSTRAINT sports_history_price_point_event_not_empty CHECK (btrim(event_key) <> ''),
  CONSTRAINT sports_history_price_point_price CHECK (price >= 0 AND price <= 1),
  CONSTRAINT sports_history_price_point_market_fk
    FOREIGN KEY (market_key) REFERENCES sports_history_market(market_key) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS sports_history_event_sort_idx
  ON sports_history_event (league, start_time DESC, event_key);

CREATE INDEX IF NOT EXISTS sports_history_event_last_seen_idx
  ON sports_history_event (last_seen_at);

CREATE INDEX IF NOT EXISTS sports_history_market_event_idx
  ON sports_history_market (event_key);

CREATE INDEX IF NOT EXISTS sports_history_market_last_seen_idx
  ON sports_history_market (last_seen_at);

CREATE INDEX IF NOT EXISTS sports_history_price_point_market_ts_idx
  ON sports_history_price_point (market_key, price_ts DESC);

-- +goose Down

DROP TABLE IF EXISTS sports_history_price_point;
DROP TABLE IF EXISTS sports_history_market;
DROP TABLE IF EXISTS sports_history_event;
DROP TABLE IF EXISTS sports_history_sync_state;
