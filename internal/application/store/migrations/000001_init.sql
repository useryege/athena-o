
-- +goose Up

CREATE TABLE IF NOT EXISTS chain (
  id BIGINT PRIMARY KEY,
  name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chain_name_not_empty CHECK (btrim(name) <> '')
);

INSERT INTO chain (id, name)
VALUES
  (1, 'Ethereum Mainnet'),
  (56, 'BSC Mainnet')
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
  updated_at = now();

CREATE TABLE IF NOT EXISTS chain_ingest_checkpoint (
  chain_id BIGINT PRIMARY KEY,
  finalized_block_number BIGINT NOT NULL DEFAULT 0,
  finalized_block_hash BYTEA,
  cursor_block_number BIGINT NOT NULL DEFAULT 0,
  cursor_block_hash BYTEA,
  status TEXT NOT NULL DEFAULT 'stopped',
  locked_at TIMESTAMPTZ,
  locked_by TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chain_ingest_checkpoint_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT chain_ingest_checkpoint_finalized_block_nonnegative CHECK (finalized_block_number >= 0),
  CONSTRAINT chain_ingest_checkpoint_cursor_block_nonnegative CHECK (cursor_block_number >= 0),
  CONSTRAINT chain_ingest_checkpoint_finalized_hash_len CHECK (finalized_block_hash IS NULL OR length(finalized_block_hash) = 32),
  CONSTRAINT chain_ingest_checkpoint_cursor_hash_len CHECK (cursor_block_hash IS NULL OR length(cursor_block_hash) = 32),
  CONSTRAINT chain_ingest_checkpoint_status_not_empty CHECK (btrim(status) <> '')
);

INSERT INTO chain_ingest_checkpoint (
  chain_id,
  finalized_block_number,
  cursor_block_number,
  status
)
VALUES
  (56, 101609863, 101609863, 'running')
ON CONFLICT (chain_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS project (
  id BIGSERIAL PRIMARY KEY,
  chain_id BIGINT NOT NULL,
  block_number BIGINT NOT NULL,
  block_time BIGINT NOT NULL,
  contract BYTEA NOT NULL,
  creator BYTEA NOT NULL,
  tx_hash BYTEA NOT NULL,
  tx_index BIGINT NOT NULL,
  weth_pair BYTEA,
  usdt_pair BYTEA,
  source TEXT NOT NULL DEFAULT 'catch_up',
  status TEXT NOT NULL DEFAULT 'processed',
  reason TEXT,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  code_hash BYTEA,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT project_contract_len CHECK (length(contract) = 20),
  CONSTRAINT project_creator_len CHECK (length(creator) = 20),
  CONSTRAINT project_tx_hash_len CHECK (length(tx_hash) = 32),
  CONSTRAINT project_weth_pair_len CHECK (weth_pair IS NULL OR length(weth_pair) = 20),
  CONSTRAINT project_usdt_pair_len CHECK (usdt_pair IS NULL OR length(usdt_pair) = 20),
  CONSTRAINT project_source_not_empty CHECK (btrim(source) <> ''),
  CONSTRAINT project_status_not_empty CHECK (btrim(status) <> ''),
  CONSTRAINT project_code_hash_len CHECK (code_hash IS NULL OR length(code_hash) = 32),
  CONSTRAINT project_block_number_nonnegative CHECK (block_number >= 0),
  CONSTRAINT project_block_time_nonnegative CHECK (block_time >= 0),
  CONSTRAINT project_tx_index_nonnegative CHECK (tx_index >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS project_chain_contract_idx
  ON project (chain_id, contract);

CREATE UNIQUE INDEX IF NOT EXISTS project_chain_tx_hash_idx
  ON project (chain_id, tx_hash);

CREATE INDEX IF NOT EXISTS project_block_order_idx
  ON project (chain_id, block_number, tx_index, id);

CREATE INDEX IF NOT EXISTS project_creator_order_idx
  ON project (chain_id, creator, block_number, tx_index, id);

CREATE INDEX IF NOT EXISTS project_status_discovered_idx
  ON project (status, discovered_at);

CREATE INDEX IF NOT EXISTS project_weth_pair_idx
  ON project (weth_pair);

CREATE INDEX IF NOT EXISTS project_usdt_pair_idx
  ON project (usdt_pair);

CREATE INDEX IF NOT EXISTS project_code_hash_idx
  ON project (code_hash);

CREATE TABLE IF NOT EXISTS project_collection_state (
  project_id BIGINT PRIMARY KEY,
  status TEXT NOT NULL,
  workflow_id TEXT,
  last_requested_at TIMESTAMPTZ,
  last_started_at TIMESTAMPTZ,
  last_completed_at TIMESTAMPTZ,
  next_run_at TIMESTAMPTZ,
  last_error TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_collection_state_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_collection_state_status_not_empty CHECK (btrim(status) <> '')
);

CREATE INDEX IF NOT EXISTS project_collection_state_next_run_idx
  ON project_collection_state (status, next_run_at);

CREATE TABLE IF NOT EXISTS outbox_event (
  id BIGSERIAL PRIMARY KEY,
  type TEXT NOT NULL,
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  chain_id BIGINT NOT NULL,
  dedup_key TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  status TEXT NOT NULL DEFAULT 'pending',
  attempts INT NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  locked_at TIMESTAMPTZ,
  locked_by TEXT,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT outbox_event_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT outbox_event_type_not_empty CHECK (btrim(type) <> ''),
  CONSTRAINT outbox_event_aggregate_type_not_empty CHECK (btrim(aggregate_type) <> ''),
  CONSTRAINT outbox_event_aggregate_id_not_empty CHECK (btrim(aggregate_id) <> ''),
  CONSTRAINT outbox_event_dedup_key_not_empty CHECK (btrim(dedup_key) <> ''),
  CONSTRAINT outbox_event_status_not_empty CHECK (btrim(status) <> ''),
  CONSTRAINT outbox_event_status_allowed CHECK (status IN ('pending', 'processing', 'processed', 'discarded')),
  CONSTRAINT outbox_event_attempts_nonnegative CHECK (attempts >= 0)
);

CREATE INDEX IF NOT EXISTS outbox_event_pending_idx
  ON outbox_event (status, next_attempt_at, id);

CREATE UNIQUE INDEX IF NOT EXISTS outbox_event_pending_dedup_uidx
  ON outbox_event (type, chain_id, dedup_key)
  WHERE status IN ('pending', 'processing');

CREATE TABLE IF NOT EXISTS project_chain_state (
  project_id BIGINT PRIMARY KEY,
  chain_state JSONB NOT NULL DEFAULT '{}'::jsonb,
  weth_pair BYTEA NOT NULL,
  usdt_pair BYTEA NOT NULL,
  token_name TEXT,
  token_symbol TEXT,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_chain_state_weth_pair_len CHECK (length(weth_pair) = 20),
  CONSTRAINT project_chain_state_usdt_pair_len CHECK (length(usdt_pair) = 20),
  CONSTRAINT project_chain_state_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS project_chain_state_weth_pair_idx
  ON project_chain_state (weth_pair);

CREATE INDEX IF NOT EXISTS project_chain_state_usdt_pair_idx
  ON project_chain_state (usdt_pair);

CREATE TABLE IF NOT EXISTS project_simulation_result (
  project_id BIGINT PRIMARY KEY,
  can_mint_from_dead_via_transfer_from BOOLEAN NOT NULL DEFAULT false,
  can_mint_from_zero_via_transfer_from BOOLEAN NOT NULL DEFAULT false,
  can_mint_from_weth_pair_via_transfer_from BOOLEAN NOT NULL DEFAULT false,
  can_mint_from_usdt_pair_via_transfer_from BOOLEAN NOT NULL DEFAULT false,
  can_mint_via_transfer_to_weth_pair BOOLEAN NOT NULL DEFAULT false,
  can_mint_via_transfer_to_usdt_pair BOOLEAN NOT NULL DEFAULT false,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_simulation_result_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS project_component_state (
  project_id BIGINT NOT NULL,
  component TEXT NOT NULL,
  status TEXT NOT NULL,
  last_attempt_at TIMESTAMPTZ,
  last_success_at TIMESTAMPTZ,
  next_run_at TIMESTAMPTZ,
  last_error TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_component_state_component_not_empty CHECK (length(btrim(component)) > 0),
  CONSTRAINT project_component_state_status_not_empty CHECK (length(btrim(status)) > 0),
  CONSTRAINT project_component_state_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_component_state_uidx UNIQUE (project_id, component)
);

CREATE INDEX IF NOT EXISTS project_component_state_next_run_idx
  ON project_component_state (component, next_run_at);

CREATE TABLE IF NOT EXISTS project_ave_detail (
  project_id BIGINT PRIMARY KEY,
  ave_response JSONB NOT NULL DEFAULT '{}'::jsonb,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_ave_detail_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS project_creator_historical_project (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT NOT NULL,
  historical_project_contract BYTEA NOT NULL,
  rank_index INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_creator_historical_project_historical_project_contract_len CHECK (length(historical_project_contract) = 20),
  CONSTRAINT project_creator_historical_project_rank_index_nonnegative CHECK (rank_index >= 0),
  CONSTRAINT project_creator_historical_project_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_creator_historical_project_uidx UNIQUE (project_id, historical_project_contract)
);

CREATE INDEX IF NOT EXISTS project_creator_historical_project_rank_idx
  ON project_creator_historical_project (project_id, rank_index);

CREATE TABLE IF NOT EXISTS project_genesis_wallet (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT NOT NULL,
  wallet BYTEA NOT NULL,
  net_amount NUMERIC(78,0) NOT NULL,
  ratio_bps BIGINT NOT NULL,
  rank_index INT NOT NULL,
  total_supply NUMERIC(78,0) NOT NULL,
  source_tx_hash BYTEA NOT NULL,
  source_block_number BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_genesis_wallet_wallet_len CHECK (length(wallet) = 20),
  CONSTRAINT project_genesis_wallet_source_tx_hash_len CHECK (length(source_tx_hash) = 32),
  CONSTRAINT project_genesis_wallet_net_amount_positive CHECK (net_amount > 0),
  CONSTRAINT project_genesis_wallet_ratio_bps_nonnegative CHECK (ratio_bps >= 0),
  CONSTRAINT project_genesis_wallet_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_genesis_wallet_project_wallet_uidx UNIQUE (project_id, wallet)
);

CREATE INDEX IF NOT EXISTS project_genesis_wallet_project_rank_idx
  ON project_genesis_wallet (project_id, rank_index);

CREATE INDEX IF NOT EXISTS project_genesis_wallet_wallet_ratio_idx
  ON project_genesis_wallet (wallet, ratio_bps DESC, project_id);

CREATE TABLE IF NOT EXISTS bytecode (
  code_hash BYTEA PRIMARY KEY,
  source_code TEXT,
  source_code_hash BYTEA,
  source_code_fetched_at TIMESTAMPTZ,
  source_code_origin TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT bytecode_code_hash_len CHECK (length(code_hash) = 32),
  CONSTRAINT bytecode_source_code_hash_len CHECK (source_code_hash IS NULL OR length(source_code_hash) = 32)
);

CREATE TABLE IF NOT EXISTS contract_bytecode_deployment (
  chain_id BIGINT NOT NULL,
  contract BYTEA NOT NULL,
  code_hash BYTEA NOT NULL,
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (chain_id, contract),
  CONSTRAINT contract_bytecode_deployment_contract_len CHECK (length(contract) = 20),
  CONSTRAINT contract_bytecode_deployment_code_hash_len CHECK (length(code_hash) = 32),
  CONSTRAINT contract_bytecode_deployment_bytecode_fk FOREIGN KEY (code_hash) REFERENCES bytecode(code_hash)
);

CREATE INDEX IF NOT EXISTS contract_bytecode_deployment_code_hash_idx
  ON contract_bytecode_deployment (code_hash);

CREATE TABLE IF NOT EXISTS bytecode_blacklist (
  code_hash BYTEA PRIMARY KEY,
  note TEXT,
  source_chain_id BIGINT,
  source_contract BYTEA,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT bytecode_blacklist_code_hash_len CHECK (length(code_hash) = 32),
  CONSTRAINT bytecode_blacklist_source_contract_len CHECK (source_contract IS NULL OR length(source_contract) = 20)
);

CREATE TABLE IF NOT EXISTS wallet_blacklist (
  wallet BYTEA PRIMARY KEY,
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT wallet_blacklist_wallet_len CHECK (length(wallet) = 20)
);

-- +goose Down

DROP TABLE IF EXISTS wallet_blacklist;
DROP TABLE IF EXISTS bytecode_blacklist;
DROP TABLE IF EXISTS contract_bytecode_deployment;
DROP TABLE IF EXISTS bytecode;
DROP TABLE IF EXISTS project_genesis_wallet;
DROP TABLE IF EXISTS project_creator_historical_project;
DROP TABLE IF EXISTS project_ave_detail;
DROP TABLE IF EXISTS project_component_state;
DROP TABLE IF EXISTS project_simulation_result;
DROP TABLE IF EXISTS project_chain_state;
DROP TABLE IF EXISTS outbox_event;
DROP TABLE IF EXISTS project_collection_state;
DROP TABLE IF EXISTS project;
DROP TABLE IF EXISTS chain_ingest_checkpoint;
DROP TABLE IF EXISTS chain;
