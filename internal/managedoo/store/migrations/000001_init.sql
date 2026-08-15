-- +goose Up

CREATE TABLE IF NOT EXISTS managed_oo_chain_log_cursor (
  sync_name TEXT PRIMARY KEY,
  contract_address TEXT NOT NULL,
  topic TEXT NOT NULL,
  last_block_number BIGINT NOT NULL DEFAULT 0,
  last_polled_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT managed_oo_chain_log_cursor_name_not_empty CHECK (btrim(sync_name) <> ''),
  CONSTRAINT managed_oo_chain_log_cursor_contract_not_empty CHECK (btrim(contract_address) <> ''),
  CONSTRAINT managed_oo_chain_log_cursor_topic_not_empty CHECK (btrim(topic) <> ''),
  CONSTRAINT managed_oo_chain_log_cursor_block_nonnegative CHECK (last_block_number >= 0)
);

CREATE TABLE IF NOT EXISTS managed_oo_propose_price_log (
  tx_hash TEXT NOT NULL,
  log_index BIGINT NOT NULL,
  block_number BIGINT NOT NULL,
  block_hash TEXT NOT NULL,
  tx_index BIGINT NOT NULL,
  contract_address TEXT NOT NULL,
  topic TEXT NOT NULL,
  requester TEXT NOT NULL,
  proposer TEXT NOT NULL,
  identifier TEXT NOT NULL,
  request_timestamp BIGINT NOT NULL,
  ancillary_data_hex TEXT NOT NULL,
  ancillary_data_text TEXT NOT NULL DEFAULT '',
  market_id TEXT NOT NULL DEFAULT '',
  proposed_price TEXT NOT NULL,
  expiration_timestamp BIGINT NOT NULL,
  currency TEXT NOT NULL,
  raw_topics JSONB NOT NULL DEFAULT '[]'::jsonb,
  raw_data TEXT NOT NULL DEFAULT '',
  fetched_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tx_hash, log_index),
  CONSTRAINT managed_oo_propose_price_log_hash_not_empty CHECK (btrim(tx_hash) <> ''),
  CONSTRAINT managed_oo_propose_price_log_index_nonnegative CHECK (log_index >= 0),
  CONSTRAINT managed_oo_propose_price_log_block_nonnegative CHECK (block_number >= 0),
  CONSTRAINT managed_oo_propose_price_log_tx_index_nonnegative CHECK (tx_index >= 0),
  CONSTRAINT managed_oo_propose_price_log_contract_not_empty CHECK (btrim(contract_address) <> ''),
  CONSTRAINT managed_oo_propose_price_log_topic_not_empty CHECK (btrim(topic) <> ''),
  CONSTRAINT managed_oo_propose_price_log_requester_not_empty CHECK (btrim(requester) <> ''),
  CONSTRAINT managed_oo_propose_price_log_proposer_not_empty CHECK (btrim(proposer) <> ''),
  CONSTRAINT managed_oo_propose_price_log_identifier_not_empty CHECK (btrim(identifier) <> ''),
  CONSTRAINT managed_oo_propose_price_log_request_ts_nonnegative CHECK (request_timestamp >= 0),
  CONSTRAINT managed_oo_propose_price_log_expiration_ts_nonnegative CHECK (expiration_timestamp >= 0),
  CONSTRAINT managed_oo_propose_price_log_currency_not_empty CHECK (btrim(currency) <> ''),
  CONSTRAINT managed_oo_propose_price_log_raw_topics_array CHECK (jsonb_typeof(raw_topics) = 'array')
);

CREATE TABLE IF NOT EXISTS managed_oo_dispute_price_log (
  tx_hash TEXT NOT NULL,
  log_index BIGINT NOT NULL,
  block_number BIGINT NOT NULL,
  block_hash TEXT NOT NULL,
  tx_index BIGINT NOT NULL,
  contract_address TEXT NOT NULL,
  topic TEXT NOT NULL,
  requester TEXT NOT NULL,
  proposer TEXT NOT NULL,
  disputer TEXT NOT NULL,
  identifier TEXT NOT NULL,
  request_timestamp BIGINT NOT NULL,
  ancillary_data_hex TEXT NOT NULL,
  ancillary_data_text TEXT NOT NULL DEFAULT '',
  market_id TEXT NOT NULL DEFAULT '',
  proposed_price TEXT NOT NULL,
  raw_topics JSONB NOT NULL DEFAULT '[]'::jsonb,
  raw_data TEXT NOT NULL DEFAULT '',
  fetched_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tx_hash, log_index),
  CONSTRAINT managed_oo_dispute_price_log_hash_not_empty CHECK (btrim(tx_hash) <> ''),
  CONSTRAINT managed_oo_dispute_price_log_index_nonnegative CHECK (log_index >= 0),
  CONSTRAINT managed_oo_dispute_price_log_block_nonnegative CHECK (block_number >= 0),
  CONSTRAINT managed_oo_dispute_price_log_tx_index_nonnegative CHECK (tx_index >= 0),
  CONSTRAINT managed_oo_dispute_price_log_contract_not_empty CHECK (btrim(contract_address) <> ''),
  CONSTRAINT managed_oo_dispute_price_log_topic_not_empty CHECK (btrim(topic) <> ''),
  CONSTRAINT managed_oo_dispute_price_log_requester_not_empty CHECK (btrim(requester) <> ''),
  CONSTRAINT managed_oo_dispute_price_log_proposer_not_empty CHECK (btrim(proposer) <> ''),
  CONSTRAINT managed_oo_dispute_price_log_disputer_not_empty CHECK (btrim(disputer) <> ''),
  CONSTRAINT managed_oo_dispute_price_log_identifier_not_empty CHECK (btrim(identifier) <> ''),
  CONSTRAINT managed_oo_dispute_price_log_request_ts_nonnegative CHECK (request_timestamp >= 0),
  CONSTRAINT managed_oo_dispute_price_log_raw_topics_array CHECK (jsonb_typeof(raw_topics) = 'array')
);

CREATE TABLE IF NOT EXISTS managed_oo_market (
  market_id TEXT PRIMARY KEY,
  condition_id TEXT NOT NULL DEFAULT '',
  slug TEXT NOT NULL DEFAULT '',
  event_slug TEXT NOT NULL DEFAULT '',
  question TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  resolution_source TEXT NOT NULL DEFAULT '',
  question_id TEXT NOT NULL DEFAULT '',
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
  accepting_orders BOOLEAN NOT NULL DEFAULT false,
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
  fetch_status TEXT NOT NULL DEFAULT 'ok',
  last_error TEXT NOT NULL DEFAULT '',
  last_error_at TIMESTAMPTZ,
  fetched_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT managed_oo_market_id_not_empty CHECK (btrim(market_id) <> ''),
  CONSTRAINT managed_oo_market_tags_array CHECK (jsonb_typeof(tags) = 'array'),
  CONSTRAINT managed_oo_market_raw_object CHECK (jsonb_typeof(raw) = 'object'),
  CONSTRAINT managed_oo_market_fetch_status CHECK (fetch_status IN ('ok', 'not_found'))
);

CREATE TABLE IF NOT EXISTS managed_oo_market_label (
  market_id TEXT NOT NULL,
  label TEXT NOT NULL,
  tag_id TEXT NOT NULL DEFAULT '',
  slug TEXT NOT NULL DEFAULT '',
  position BIGINT NOT NULL DEFAULT 0,
  fetched_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (market_id, label),
  CONSTRAINT managed_oo_market_label_market_not_empty CHECK (btrim(market_id) <> ''),
  CONSTRAINT managed_oo_market_label_label_not_empty CHECK (btrim(label) <> ''),
  CONSTRAINT managed_oo_market_label_position_nonnegative CHECK (position >= 0),
  CONSTRAINT managed_oo_market_label_market_fk
    FOREIGN KEY (market_id) REFERENCES managed_oo_market(market_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS managed_oo_propose_price_alert_state (
  tx_hash TEXT NOT NULL,
  log_index BIGINT NOT NULL,
  notification_id BIGINT NOT NULL DEFAULT 0,
  notified_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tx_hash, log_index),
  CONSTRAINT managed_oo_propose_price_alert_state_hash_not_empty CHECK (btrim(tx_hash) <> ''),
  CONSTRAINT managed_oo_propose_price_alert_state_index_nonnegative CHECK (log_index >= 0),
  CONSTRAINT managed_oo_propose_price_alert_state_notification_nonnegative CHECK (notification_id >= 0),
  CONSTRAINT managed_oo_propose_price_alert_state_log_fk
    FOREIGN KEY (tx_hash, log_index) REFERENCES managed_oo_propose_price_log(tx_hash, log_index) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS managed_oo_dispute_price_alert_state (
  tx_hash TEXT NOT NULL,
  log_index BIGINT NOT NULL,
  notification_id BIGINT NOT NULL DEFAULT 0,
  notified_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tx_hash, log_index),
  CONSTRAINT managed_oo_dispute_price_alert_state_hash_not_empty CHECK (btrim(tx_hash) <> ''),
  CONSTRAINT managed_oo_dispute_price_alert_state_index_nonnegative CHECK (log_index >= 0),
  CONSTRAINT managed_oo_dispute_price_alert_state_notification_nonnegative CHECK (notification_id >= 0),
  CONSTRAINT managed_oo_dispute_price_alert_state_log_fk
    FOREIGN KEY (tx_hash, log_index) REFERENCES managed_oo_dispute_price_log(tx_hash, log_index) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS managed_oo_chain_log_cursor_updated_idx
  ON managed_oo_chain_log_cursor (updated_at DESC);

CREATE INDEX IF NOT EXISTS managed_oo_propose_price_log_block_idx
  ON managed_oo_propose_price_log (block_number DESC, log_index DESC);

CREATE INDEX IF NOT EXISTS managed_oo_propose_price_log_request_idx
  ON managed_oo_propose_price_log (requester, identifier, request_timestamp);

CREATE INDEX IF NOT EXISTS managed_oo_propose_price_log_market_idx
  ON managed_oo_propose_price_log (market_id)
  WHERE market_id <> '';

CREATE INDEX IF NOT EXISTS managed_oo_dispute_price_log_block_idx
  ON managed_oo_dispute_price_log (block_number DESC, log_index DESC);

CREATE INDEX IF NOT EXISTS managed_oo_dispute_price_log_request_idx
  ON managed_oo_dispute_price_log (requester, identifier, request_timestamp);

CREATE INDEX IF NOT EXISTS managed_oo_dispute_price_log_market_idx
  ON managed_oo_dispute_price_log (market_id)
  WHERE market_id <> '';

CREATE INDEX IF NOT EXISTS managed_oo_market_condition_idx
  ON managed_oo_market (condition_id)
  WHERE condition_id <> '';

CREATE INDEX IF NOT EXISTS managed_oo_market_slug_idx
  ON managed_oo_market (slug)
  WHERE slug <> '';

CREATE INDEX IF NOT EXISTS managed_oo_market_label_label_idx
  ON managed_oo_market_label (label)
  WHERE label <> '';

CREATE INDEX IF NOT EXISTS managed_oo_market_label_slug_idx
  ON managed_oo_market_label (slug)
  WHERE slug <> '';

CREATE INDEX IF NOT EXISTS managed_oo_market_label_tag_idx
  ON managed_oo_market_label (tag_id)
  WHERE tag_id <> '';

CREATE INDEX IF NOT EXISTS managed_oo_propose_price_alert_state_notified_idx
  ON managed_oo_propose_price_alert_state (notified_at DESC);

CREATE INDEX IF NOT EXISTS managed_oo_dispute_price_alert_state_notified_idx
  ON managed_oo_dispute_price_alert_state (notified_at DESC);

-- +goose Down

DROP TABLE IF EXISTS managed_oo_dispute_price_alert_state;
DROP TABLE IF EXISTS managed_oo_propose_price_alert_state;
DROP TABLE IF EXISTS managed_oo_market_label;
DROP TABLE IF EXISTS managed_oo_market;
DROP TABLE IF EXISTS managed_oo_dispute_price_log;
DROP TABLE IF EXISTS managed_oo_propose_price_log;
DROP TABLE IF EXISTS managed_oo_chain_log_cursor;
