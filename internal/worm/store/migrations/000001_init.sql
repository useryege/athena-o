
-- +goose Up

CREATE TABLE IF NOT EXISTS worm_market (
  condition_id TEXT PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  logo TEXT NOT NULL DEFAULT '',
  last_trade_price TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT 'open',
  category TEXT NOT NULL DEFAULT 'sports',
  sort_option TEXT NOT NULL DEFAULT 'leverage',
  created BIGINT NOT NULL DEFAULT 0,
  event_title TEXT NOT NULL DEFAULT '',
  event_condition_id TEXT NOT NULL DEFAULT '',
  event_logo TEXT NOT NULL DEFAULT '',
  margin_enabled BOOLEAN NOT NULL DEFAULT false,
  live_state TEXT NOT NULL DEFAULT 'unknown',
  live_checked_at TIMESTAMPTZ,
  live_price_change TEXT NOT NULL DEFAULT '',
  raw JSONB NOT NULL DEFAULT '{}'::jsonb,
  rules JSONB,
  match_start_at TIMESTAMPTZ,
  match_start_analyzed_at TIMESTAMPTZ,
  fetched_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT worm_market_condition_id_not_empty CHECK (btrim(condition_id) <> ''),
  CONSTRAINT worm_market_state_open CHECK (state = 'open'),
  CONSTRAINT worm_market_category_sports CHECK (category = 'sports'),
  CONSTRAINT worm_market_sort_option_leverage CHECK (sort_option = 'leverage'),
  CONSTRAINT worm_market_live_state_valid CHECK (live_state IN ('live', 'not_live', 'unknown')),
  CONSTRAINT worm_market_created_nonnegative CHECK (created >= 0),
  CONSTRAINT worm_market_raw_object CHECK (jsonb_typeof(raw) = 'object'),
  CONSTRAINT worm_market_rules_array CHECK (rules IS NULL OR jsonb_typeof(rules) = 'array')
);

CREATE TABLE IF NOT EXISTS worm_market_price_history (
  condition_id TEXT NOT NULL,
  price NUMERIC NOT NULL,
  sampled_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (condition_id, sampled_at),
  CONSTRAINT worm_market_price_history_market_fk
    FOREIGN KEY (condition_id) REFERENCES worm_market(condition_id) ON DELETE CASCADE,
  CONSTRAINT worm_market_price_history_price_nonnegative CHECK (price >= 0)
);

CREATE INDEX IF NOT EXISTS worm_market_live_sort_idx
  ON worm_market ((live_state = 'live') DESC, created DESC, condition_id);

CREATE INDEX IF NOT EXISTS worm_market_live_check_idx
  ON worm_market (live_state, live_checked_at ASC NULLS FIRST, condition_id);

CREATE INDEX IF NOT EXISTS worm_market_last_seen_idx
  ON worm_market (last_seen_at DESC);

CREATE INDEX IF NOT EXISTS worm_market_created_idx
  ON worm_market (created DESC, condition_id);

CREATE INDEX IF NOT EXISTS worm_market_updated_idx
  ON worm_market (updated_at DESC);

CREATE INDEX IF NOT EXISTS worm_market_event_condition_idx
  ON worm_market (event_condition_id);

CREATE INDEX IF NOT EXISTS worm_market_price_history_market_time_idx
  ON worm_market_price_history (condition_id, sampled_at DESC);

CREATE INDEX IF NOT EXISTS worm_market_price_history_sampled_at_idx
  ON worm_market_price_history (sampled_at);

-- +goose Down

DROP TABLE IF EXISTS worm_market_price_history;
DROP TABLE IF EXISTS worm_market;
