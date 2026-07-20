-- +goose Up

CREATE TABLE bsc_inbound_scan_checkpoint (
  singleton BOOLEAN PRIMARY KEY DEFAULT TRUE,
  start_block_number BIGINT NOT NULL,
  cursor_block_number BIGINT NOT NULL,
  cursor_block_hash BYTEA NOT NULL,
  cursor_block_timestamp BIGINT NOT NULL,
  initialized_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT bsc_inbound_scan_checkpoint_singleton_check CHECK (singleton),
  CONSTRAINT bsc_inbound_scan_checkpoint_start_block_number_check CHECK (start_block_number > 0),
  CONSTRAINT bsc_inbound_scan_checkpoint_cursor_block_number_check CHECK (cursor_block_number >= 0),
  CONSTRAINT bsc_inbound_scan_checkpoint_cursor_before_start_check CHECK (cursor_block_number + 1 >= start_block_number),
  CONSTRAINT bsc_inbound_scan_checkpoint_cursor_block_hash_length_check CHECK (length(cursor_block_hash) = 32),
  CONSTRAINT bsc_inbound_scan_checkpoint_cursor_block_timestamp_check CHECK (cursor_block_timestamp >= 0)
);

CREATE TABLE inbound_normal_transaction (
  transaction_hash BYTEA PRIMARY KEY,
  block_number BIGINT NOT NULL,
  block_hash BYTEA NOT NULL,
  block_timestamp BIGINT NOT NULL,
  transaction_index BIGINT NOT NULL,
  from_address BYTEA NOT NULL,
  to_address BYTEA NOT NULL,
  value_wei NUMERIC(78, 0) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT inbound_normal_transaction_block_position_uidx UNIQUE (block_number, transaction_index),
  CONSTRAINT inbound_normal_transaction_transaction_hash_length_check CHECK (length(transaction_hash) = 32),
  CONSTRAINT inbound_normal_transaction_block_number_check CHECK (block_number >= 0),
  CONSTRAINT inbound_normal_transaction_block_hash_length_check CHECK (length(block_hash) = 32),
  CONSTRAINT inbound_normal_transaction_block_timestamp_check CHECK (block_timestamp >= 0),
  CONSTRAINT inbound_normal_transaction_transaction_index_check CHECK (transaction_index >= 0),
  CONSTRAINT inbound_normal_transaction_from_address_length_check CHECK (length(from_address) = 20),
  CONSTRAINT inbound_normal_transaction_to_address_length_check CHECK (length(to_address) = 20),
  CONSTRAINT inbound_normal_transaction_value_wei_check CHECK (value_wei > 10000000000000000)
);

CREATE INDEX inbound_normal_transaction_to_address_position_idx
  ON inbound_normal_transaction (to_address, block_number DESC, transaction_index DESC);

-- +goose Down

DROP TABLE IF EXISTS inbound_normal_transaction;
DROP TABLE IF EXISTS bsc_inbound_scan_checkpoint;
