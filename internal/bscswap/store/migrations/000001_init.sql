-- +goose Up

CREATE TABLE bsc_swap_scan_checkpoint (
  singleton BOOLEAN PRIMARY KEY DEFAULT TRUE,
  start_block_number BIGINT NOT NULL,
  cursor_block_number BIGINT NOT NULL,
  cursor_block_hash BYTEA NOT NULL,
  cursor_block_timestamp BIGINT NOT NULL,
  initialized_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT bsc_swap_scan_checkpoint_singleton_check CHECK (singleton),
  CONSTRAINT bsc_swap_scan_checkpoint_start_block_number_check CHECK (start_block_number > 0),
  CONSTRAINT bsc_swap_scan_checkpoint_cursor_block_number_check CHECK (cursor_block_number >= 0),
  CONSTRAINT bsc_swap_scan_checkpoint_cursor_before_start_check CHECK (cursor_block_number + 1 >= start_block_number),
  CONSTRAINT bsc_swap_scan_checkpoint_cursor_block_hash_length_check CHECK (length(cursor_block_hash) = 32),
  CONSTRAINT bsc_swap_scan_checkpoint_cursor_block_timestamp_check CHECK (cursor_block_timestamp >= 0)
);

CREATE TABLE bsc_swap_transaction (
  transaction_hash BYTEA PRIMARY KEY,
  block_number BIGINT NOT NULL,
  block_hash BYTEA NOT NULL,
  block_timestamp BIGINT NOT NULL,
  transaction_index BIGINT NOT NULL,
  from_address BYTEA NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT bsc_swap_transaction_block_position_uidx UNIQUE (block_number, transaction_index),
  CONSTRAINT bsc_swap_transaction_transaction_hash_length_check CHECK (length(transaction_hash) = 32),
  CONSTRAINT bsc_swap_transaction_block_number_check CHECK (block_number >= 0),
  CONSTRAINT bsc_swap_transaction_block_hash_length_check CHECK (length(block_hash) = 32),
  CONSTRAINT bsc_swap_transaction_block_timestamp_check CHECK (block_timestamp >= 0),
  CONSTRAINT bsc_swap_transaction_transaction_index_check CHECK (transaction_index >= 0),
  CONSTRAINT bsc_swap_transaction_from_address_length_check CHECK (length(from_address) = 20)
);

CREATE INDEX bsc_swap_transaction_from_address_position_idx
  ON bsc_swap_transaction (from_address, block_number DESC, transaction_index DESC);

-- +goose Down

DROP TABLE IF EXISTS bsc_swap_transaction;
DROP TABLE IF EXISTS bsc_swap_scan_checkpoint;
