-- +goose Up

CREATE TABLE worm_market_combinations (
  id UUID PRIMARY KEY,
  owner_account_id UUID NOT NULL,
  name TEXT NOT NULL CHECK (
    name = btrim(name)
    AND char_length(name) BETWEEN 1 AND 80
  ),
  name_key TEXT GENERATED ALWAYS AS (lower(name)) STORED,
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT worm_market_combinations_owner_name_key_unique
    UNIQUE (owner_account_id, name_key),
  CHECK (updated_at >= created_at)
);

CREATE INDEX worm_market_combinations_owner_updated_idx
  ON worm_market_combinations (owner_account_id, updated_at DESC, id DESC);

CREATE TABLE worm_market_combination_items (
  combination_id UUID NOT NULL
    REFERENCES worm_market_combinations(id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL CHECK (ordinal > 0),
  event_condition_id TEXT NOT NULL CHECK (
    event_condition_id = btrim(event_condition_id)
    AND char_length(event_condition_id) BETWEEN 32 AND 64
  ),
  event_title TEXT NOT NULL CHECK (
    event_title = btrim(event_title)
    AND char_length(event_title) BETWEEN 1 AND 500
  ),
  event_logo TEXT NOT NULL DEFAULT '' CHECK (
    event_logo = btrim(event_logo)
    AND char_length(event_logo) <= 2048
  ),
  market_condition_id TEXT NOT NULL CHECK (
    market_condition_id = btrim(market_condition_id)
    AND char_length(market_condition_id) BETWEEN 32 AND 64
  ),
  market_title TEXT NOT NULL CHECK (
    market_title = btrim(market_title)
    AND char_length(market_title) BETWEEN 1 AND 500
  ),
  market_logo TEXT NOT NULL DEFAULT '' CHECK (
    market_logo = btrim(market_logo)
    AND char_length(market_logo) <= 2048
  ),
  is_yes BOOLEAN NOT NULL,
  outcome_label TEXT NOT NULL CHECK (
    outcome_label = btrim(outcome_label)
    AND char_length(outcome_label) BETWEEN 1 AND 100
  ),
  PRIMARY KEY (combination_id, ordinal),
  CONSTRAINT worm_market_combination_items_market_unique
    UNIQUE (combination_id, market_condition_id)
);

-- +goose Down

DROP TABLE IF EXISTS worm_market_combination_items;
DROP TABLE IF EXISTS worm_market_combinations;
