-- +goose Up

CREATE TABLE IF NOT EXISTS normal_transaction (
  chain_id BIGINT NOT NULL,
  tx_hash BYTEA NOT NULL,
  block_number BIGINT NOT NULL,
  block_hash BYTEA NOT NULL,
  block_timestamp BIGINT NOT NULL,
  nonce NUMERIC(78, 0) NOT NULL,
  transaction_index BIGINT NOT NULL,
  from_address BYTEA NOT NULL,
  to_address BYTEA,
  value NUMERIC(78, 0) NOT NULL,
  gas NUMERIC(78, 0) NOT NULL,
  gas_price NUMERIC(78, 0) NOT NULL,
  input TEXT NOT NULL,
  method_id BYTEA,
  function_name TEXT NOT NULL,
  contract_address BYTEA,
  cumulative_gas_used NUMERIC(78, 0) NOT NULL,
  tx_receipt_status SMALLINT NOT NULL,
  gas_used NUMERIC(78, 0) NOT NULL,
  confirmations NUMERIC(78, 0) NOT NULL,
  is_error BOOLEAN NOT NULL,
  fetched_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (chain_id, tx_hash),
  CONSTRAINT normal_transaction_chain_positive CHECK (chain_id > 0),
  CONSTRAINT normal_transaction_tx_hash_len CHECK (length(tx_hash) = 32),
  CONSTRAINT normal_transaction_block_nonnegative CHECK (block_number >= 0),
  CONSTRAINT normal_transaction_block_hash_len CHECK (length(block_hash) = 32),
  CONSTRAINT normal_transaction_timestamp_nonnegative CHECK (block_timestamp >= 0),
  CONSTRAINT normal_transaction_nonce_nonnegative CHECK (nonce >= 0),
  CONSTRAINT normal_transaction_index_nonnegative CHECK (transaction_index >= 0),
  CONSTRAINT normal_transaction_from_address_len CHECK (length(from_address) = 20),
  CONSTRAINT normal_transaction_to_address_len CHECK (to_address IS NULL OR length(to_address) = 20),
  CONSTRAINT normal_transaction_value_nonnegative CHECK (value >= 0),
  CONSTRAINT normal_transaction_gas_nonnegative CHECK (gas >= 0),
  CONSTRAINT normal_transaction_gas_price_nonnegative CHECK (gas_price >= 0),
  CONSTRAINT normal_transaction_method_id_len CHECK (method_id IS NULL OR length(method_id) = 4),
  CONSTRAINT normal_transaction_contract_address_len CHECK (contract_address IS NULL OR length(contract_address) = 20),
  CONSTRAINT normal_transaction_cumulative_gas_nonnegative CHECK (cumulative_gas_used >= 0),
  CONSTRAINT normal_transaction_receipt_status_allowed CHECK (tx_receipt_status IN (-1, 0, 1)),
  CONSTRAINT normal_transaction_gas_used_nonnegative CHECK (gas_used >= 0),
  CONSTRAINT normal_transaction_confirmations_nonnegative CHECK (confirmations >= 0)
);

CREATE INDEX IF NOT EXISTS normal_transaction_block_order_idx
  ON normal_transaction (chain_id, block_number DESC, transaction_index DESC);

CREATE TABLE IF NOT EXISTS normal_transaction_query_cache (
  id BIGSERIAL PRIMARY KEY,
  chain_id BIGINT NOT NULL,
  address BYTEA NOT NULL,
  start_block BIGINT NOT NULL,
  end_block BIGINT NOT NULL,
  page INT NOT NULL,
  page_size INT NOT NULL,
  sort_order TEXT NOT NULL,
  fetched_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT normal_transaction_query_chain_positive CHECK (chain_id > 0),
  CONSTRAINT normal_transaction_query_address_len CHECK (length(address) = 20),
  CONSTRAINT normal_transaction_query_start_block_nonnegative CHECK (start_block >= 0),
  CONSTRAINT normal_transaction_query_end_block_nonnegative CHECK (end_block >= start_block),
  CONSTRAINT normal_transaction_query_page_positive CHECK (page > 0),
  CONSTRAINT normal_transaction_query_page_size_valid CHECK (page_size BETWEEN 1 AND 1000),
  CONSTRAINT normal_transaction_query_sort_allowed CHECK (sort_order IN ('asc', 'desc')),
  UNIQUE (chain_id, address, start_block, end_block, page, page_size, sort_order)
);

CREATE INDEX IF NOT EXISTS normal_transaction_query_cache_expiry_idx
  ON normal_transaction_query_cache (expires_at);

CREATE TABLE IF NOT EXISTS normal_transaction_query_item (
  query_id BIGINT NOT NULL,
  position INT NOT NULL,
  chain_id BIGINT NOT NULL,
  tx_hash BYTEA NOT NULL,
  PRIMARY KEY (query_id, position),
  UNIQUE (query_id, chain_id, tx_hash),
  CONSTRAINT normal_transaction_query_item_query_fk
    FOREIGN KEY (query_id)
    REFERENCES normal_transaction_query_cache(id)
    ON DELETE CASCADE,
  CONSTRAINT normal_transaction_query_item_transaction_fk
    FOREIGN KEY (chain_id, tx_hash)
    REFERENCES normal_transaction(chain_id, tx_hash),
  CONSTRAINT normal_transaction_query_item_position_nonnegative CHECK (position >= 0),
  CONSTRAINT normal_transaction_query_item_tx_hash_len CHECK (length(tx_hash) = 32)
);

-- +goose Down

DROP TABLE IF EXISTS normal_transaction_query_item;
DROP TABLE IF EXISTS normal_transaction_query_cache;
DROP TABLE IF EXISTS normal_transaction;
