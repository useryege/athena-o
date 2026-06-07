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

INSERT INTO chain_ingest_checkpoint (chain_id, cursor_block_number, status)
VALUES
  (1, 25211026, 'stopped'),
  (56, 101719440, 'stopped')
ON CONFLICT (chain_id) DO NOTHING;

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
  weth_pair BYTEA,
  usdt_pair BYTEA,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT project_contract_code_fk FOREIGN KEY (code_hash) REFERENCES contract_code(code_hash),
  CONSTRAINT project_contract_len CHECK (length(contract) = 20),
  CONSTRAINT project_creator_len CHECK (length(creator) = 20),
  CONSTRAINT project_tx_hash_len CHECK (length(tx_hash) = 32),
  CONSTRAINT project_code_hash_len CHECK (length(code_hash) = 32),
  CONSTRAINT project_weth_pair_len CHECK (weth_pair IS NULL OR length(weth_pair) = 20),
  CONSTRAINT project_usdt_pair_len CHECK (usdt_pair IS NULL OR length(usdt_pair) = 20),
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

CREATE TABLE IF NOT EXISTS project_related_wallet (
  project_id BIGINT NOT NULL,
  wallet BYTEA NOT NULL,
  role TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, wallet, role),
  CONSTRAINT project_related_wallet_project_fk
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_related_wallet_wallet_len CHECK (length(wallet) = 20),
  CONSTRAINT project_related_wallet_role_not_empty CHECK (btrim(role) <> '')
);

CREATE INDEX IF NOT EXISTS project_related_wallet_project_role_idx
  ON project_related_wallet (project_id, role);

CREATE INDEX IF NOT EXISTS project_related_wallet_wallet_idx
  ON project_related_wallet (wallet);

CREATE TABLE IF NOT EXISTS wallet_asset_state (
  chain_id BIGINT NOT NULL,
  wallet BYTEA NOT NULL,
  weth_balance NUMERIC(78, 0) NOT NULL DEFAULT 0,
  usdt_balance NUMERIC(78, 0) NOT NULL DEFAULT 0,
  native_balance NUMERIC(78, 0) NOT NULL DEFAULT 0,
  usdt_value NUMERIC(78, 0) NOT NULL DEFAULT 0,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (chain_id, wallet),
  CONSTRAINT wallet_asset_state_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT wallet_asset_state_wallet_len CHECK (length(wallet) = 20),
  CONSTRAINT wallet_asset_state_weth_balance_nonnegative CHECK (weth_balance >= 0),
  CONSTRAINT wallet_asset_state_usdt_balance_nonnegative CHECK (usdt_balance >= 0),
  CONSTRAINT wallet_asset_state_native_balance_nonnegative CHECK (native_balance >= 0),
  CONSTRAINT wallet_asset_state_usdt_value_nonnegative CHECK (usdt_value >= 0)
);

CREATE INDEX IF NOT EXISTS wallet_asset_state_usdt_value_idx
  ON wallet_asset_state (chain_id, usdt_value DESC);

CREATE TABLE IF NOT EXISTS project_data_collection_task (
  project_id BIGINT NOT NULL,
  data_type TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  attempts INT NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, data_type),
  CONSTRAINT project_data_collection_task_project_fk
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_data_collection_task_data_type_allowed
    CHECK (data_type IN ('ave', 'chain_state', 'wallet_asset_state', 'simulation_result')),
  CONSTRAINT project_data_collection_task_status_allowed
    CHECK (status IN ('pending', 'succeeded', 'failed')),
  CONSTRAINT project_data_collection_task_attempts_range
    CHECK (attempts >= 0 AND attempts <= 5)
);

CREATE INDEX IF NOT EXISTS project_data_collection_task_due_idx
  ON project_data_collection_task (status, next_attempt_at, data_type, project_id);

CREATE TABLE IF NOT EXISTS project_ave_data (
  project_id BIGINT PRIMARY KEY,
  ave_response JSONB NOT NULL DEFAULT '{}'::jsonb,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_ave_data_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS project_chain_state (
  project_id BIGINT PRIMARY KEY,
  chain_state JSONB NOT NULL DEFAULT '{}'::jsonb,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_chain_state_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS project_simulation_result (
  project_id BIGINT NOT NULL,
  wallet BYTEA NOT NULL,
  can_mint_from_dead_via_transfer_from BOOLEAN NOT NULL DEFAULT false,
  can_mint_from_zero_via_transfer_from BOOLEAN NOT NULL DEFAULT false,
  can_mint_from_weth_pair_via_transfer_from BOOLEAN NOT NULL DEFAULT false,
  can_mint_from_usdt_pair_via_transfer_from BOOLEAN NOT NULL DEFAULT false,
  can_mint_via_transfer_to_weth_pair BOOLEAN NOT NULL DEFAULT false,
  can_mint_via_transfer_to_usdt_pair BOOLEAN NOT NULL DEFAULT false,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, wallet),
  CONSTRAINT project_simulation_result_project_fk
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_simulation_result_wallet_len CHECK (length(wallet) = 20)
);

CREATE INDEX IF NOT EXISTS project_simulation_result_wallet_idx
  ON project_simulation_result (wallet);

CREATE INDEX IF NOT EXISTS project_simulation_result_risk_idx
  ON project_simulation_result (project_id)
  WHERE can_mint_from_dead_via_transfer_from
    OR can_mint_from_zero_via_transfer_from
    OR can_mint_from_weth_pair_via_transfer_from
    OR can_mint_from_usdt_pair_via_transfer_from
    OR can_mint_via_transfer_to_weth_pair
    OR can_mint_via_transfer_to_usdt_pair;

-- +goose Down

DROP TABLE IF EXISTS project_data_collection_task;
DROP TABLE IF EXISTS project_simulation_result;
DROP TABLE IF EXISTS project_chain_state;
DROP TABLE IF EXISTS project_ave_data;
DROP TABLE IF EXISTS wallet_asset_state;
DROP TABLE IF EXISTS project_related_wallet;
DROP TABLE IF EXISTS project;
DROP TABLE IF EXISTS contract_code;
DROP TABLE IF EXISTS project_candidate;
DROP TABLE IF EXISTS chain_ingest_checkpoint;
DROP TABLE IF EXISTS chain;
