-- +goose Up

CREATE TABLE IF NOT EXISTS chain (
  id BIGINT PRIMARY KEY,
  name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chain_name_not_empty CHECK (btrim(name) <> '')
);

INSERT INTO chain (id, name)
VALUES
  (1, 'Ethereum Mainnet'),
  (56, 'BSC Mainnet')
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS chain_ingest_checkpoint (
  chain_id BIGINT PRIMARY KEY,
  cursor_block_number BIGINT NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'stopped',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chain_ingest_checkpoint_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT chain_ingest_checkpoint_cursor_block_nonnegative CHECK (cursor_block_number >= 0),
  CONSTRAINT chain_ingest_checkpoint_status_allowed CHECK (status IN ('running', 'stopped'))
);

CREATE TABLE IF NOT EXISTS project_candidate (
  id BIGSERIAL PRIMARY KEY,
  chain_id BIGINT NOT NULL,
  contract BYTEA NOT NULL,
  creator BYTEA NOT NULL,
  tx_hash BYTEA NOT NULL,
  tx_index BIGINT NOT NULL,
  block_number BIGINT NOT NULL,
  block_time BIGINT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_candidate_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT project_candidate_contract_len CHECK (length(contract) = 20),
  CONSTRAINT project_candidate_creator_len CHECK (length(creator) = 20),
  CONSTRAINT project_candidate_tx_hash_len CHECK (length(tx_hash) = 32),
  CONSTRAINT project_candidate_tx_index_nonnegative CHECK (tx_index >= 0),
  CONSTRAINT project_candidate_block_number_nonnegative CHECK (block_number >= 0),
  CONSTRAINT project_candidate_block_time_nonnegative CHECK (block_time >= 0),
  CONSTRAINT project_candidate_status_allowed CHECK (status IN ('pending', 'qualified', 'rejected'))
);

CREATE UNIQUE INDEX IF NOT EXISTS project_candidate_chain_contract_uidx
  ON project_candidate (chain_id, contract);

CREATE UNIQUE INDEX IF NOT EXISTS project_candidate_chain_tx_hash_uidx
  ON project_candidate (chain_id, tx_hash);

CREATE INDEX IF NOT EXISTS project_candidate_status_created_idx
  ON project_candidate (status, created_at);

CREATE TABLE IF NOT EXISTS contract_code (
  code_hash BYTEA PRIMARY KEY,
  source_code TEXT,
  source_code_hash BYTEA,
  source_code_fetched_at TIMESTAMPTZ,
  source_code_origin TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT contract_code_code_hash_len CHECK (length(code_hash) = 32),
  CONSTRAINT contract_code_source_code_hash_len CHECK (source_code_hash IS NULL OR length(source_code_hash) = 32)
);

CREATE TABLE IF NOT EXISTS project (
  id BIGSERIAL PRIMARY KEY,
  chain_id BIGINT NOT NULL,
  contract BYTEA NOT NULL,
  creator BYTEA NOT NULL,
  tx_hash BYTEA NOT NULL,
  tx_index BIGINT NOT NULL,
  block_number BIGINT NOT NULL,
  block_time BIGINT NOT NULL,
  code_hash BYTEA NOT NULL,
  weth_pair BYTEA NOT NULL,
  usdt_pair BYTEA NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT project_contract_code_fk FOREIGN KEY (code_hash) REFERENCES contract_code(code_hash),
  CONSTRAINT project_contract_len CHECK (length(contract) = 20),
  CONSTRAINT project_creator_len CHECK (length(creator) = 20),
  CONSTRAINT project_tx_hash_len CHECK (length(tx_hash) = 32),
  CONSTRAINT project_code_hash_len CHECK (length(code_hash) = 32),
  CONSTRAINT project_weth_pair_len CHECK (length(weth_pair) = 20),
  CONSTRAINT project_usdt_pair_len CHECK (length(usdt_pair) = 20),
  CONSTRAINT project_tx_index_nonnegative CHECK (tx_index >= 0),
  CONSTRAINT project_block_number_nonnegative CHECK (block_number >= 0),
  CONSTRAINT project_block_time_nonnegative CHECK (block_time >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS project_chain_contract_uidx
  ON project (chain_id, contract);

CREATE UNIQUE INDEX IF NOT EXISTS project_chain_tx_hash_uidx
  ON project (chain_id, tx_hash);

CREATE INDEX IF NOT EXISTS project_code_hash_idx
  ON project (code_hash);

CREATE INDEX IF NOT EXISTS project_block_order_idx
  ON project (chain_id, block_number, tx_index, id);

-- +goose Down

DROP TABLE IF EXISTS project;
DROP TABLE IF EXISTS contract_code;
DROP TABLE IF EXISTS project_candidate;
DROP TABLE IF EXISTS chain_ingest_checkpoint;
DROP TABLE IF EXISTS chain;
