-- +goose Up

CREATE TABLE IF NOT EXISTS polymarket_chain_log_cursor (
  sync_name TEXT PRIMARY KEY,
  contract_address TEXT NOT NULL,
  topic TEXT NOT NULL,
  last_block_number BIGINT NOT NULL DEFAULT 0,
  last_polled_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT polymarket_chain_log_cursor_name_not_empty CHECK (btrim(sync_name) <> ''),
  CONSTRAINT polymarket_chain_log_cursor_contract_not_empty CHECK (btrim(contract_address) <> ''),
  CONSTRAINT polymarket_chain_log_cursor_topic_not_empty CHECK (btrim(topic) <> ''),
  CONSTRAINT polymarket_chain_log_cursor_block_nonnegative CHECK (last_block_number >= 0)
);

CREATE TABLE IF NOT EXISTS polymarket_managed_oo_propose_price_log (
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
  proposed_price TEXT NOT NULL,
  expiration_timestamp BIGINT NOT NULL,
  currency TEXT NOT NULL,
  raw_topics JSONB NOT NULL DEFAULT '[]'::jsonb,
  raw_data TEXT NOT NULL DEFAULT '',
  fetched_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tx_hash, log_index),
  CONSTRAINT polymarket_managed_oo_propose_price_log_hash_not_empty CHECK (btrim(tx_hash) <> ''),
  CONSTRAINT polymarket_managed_oo_propose_price_log_index_nonnegative CHECK (log_index >= 0),
  CONSTRAINT polymarket_managed_oo_propose_price_log_block_nonnegative CHECK (block_number >= 0),
  CONSTRAINT polymarket_managed_oo_propose_price_log_tx_index_nonnegative CHECK (tx_index >= 0),
  CONSTRAINT polymarket_managed_oo_propose_price_log_contract_not_empty CHECK (btrim(contract_address) <> ''),
  CONSTRAINT polymarket_managed_oo_propose_price_log_topic_not_empty CHECK (btrim(topic) <> ''),
  CONSTRAINT polymarket_managed_oo_propose_price_log_requester_not_empty CHECK (btrim(requester) <> ''),
  CONSTRAINT polymarket_managed_oo_propose_price_log_proposer_not_empty CHECK (btrim(proposer) <> ''),
  CONSTRAINT polymarket_managed_oo_propose_price_log_identifier_not_empty CHECK (btrim(identifier) <> ''),
  CONSTRAINT polymarket_managed_oo_propose_price_log_request_ts_nonnegative CHECK (request_timestamp >= 0),
  CONSTRAINT polymarket_managed_oo_propose_price_log_expiration_ts_nonnegative CHECK (expiration_timestamp >= 0),
  CONSTRAINT polymarket_managed_oo_propose_price_log_currency_not_empty CHECK (btrim(currency) <> ''),
  CONSTRAINT polymarket_managed_oo_propose_price_log_raw_topics_array CHECK (jsonb_typeof(raw_topics) = 'array')
);

CREATE INDEX IF NOT EXISTS polymarket_chain_log_cursor_updated_idx
  ON polymarket_chain_log_cursor (updated_at DESC);

CREATE INDEX IF NOT EXISTS polymarket_managed_oo_propose_price_log_block_idx
  ON polymarket_managed_oo_propose_price_log (block_number DESC, log_index DESC);

CREATE INDEX IF NOT EXISTS polymarket_managed_oo_propose_price_log_request_idx
  ON polymarket_managed_oo_propose_price_log (requester, identifier, request_timestamp);

-- +goose Down

DROP TABLE IF EXISTS polymarket_managed_oo_propose_price_log;
DROP TABLE IF EXISTS polymarket_chain_log_cursor;
