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
  source_code_fetched_at TIMESTAMPTZ,
  deployment_count BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT contract_code_code_hash_len CHECK (length(code_hash) = 32),
  CONSTRAINT contract_code_deployment_count_nonnegative CHECK (deployment_count >= 0)
);

CREATE INDEX IF NOT EXISTS contract_code_deployment_count_idx
  ON contract_code (deployment_count DESC, created_at DESC, code_hash);

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
  name TEXT NOT NULL DEFAULT '',
  symbol TEXT NOT NULL DEFAULT '',
  decimals SMALLINT NOT NULL DEFAULT 0,
  total_supply NUMERIC(78, 0) NOT NULL DEFAULT 0,
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
  CONSTRAINT project_block_time_nonnegative CHECK (block_time >= 0),
  CONSTRAINT project_decimals_uint8 CHECK (decimals BETWEEN 0 AND 255),
  CONSTRAINT project_total_supply_nonnegative CHECK (total_supply >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS project_chain_contract_uidx
  ON project (chain_id, contract);

CREATE UNIQUE INDEX IF NOT EXISTS project_chain_tx_hash_uidx
  ON project (chain_id, tx_hash);

CREATE INDEX IF NOT EXISTS project_code_hash_idx
  ON project (code_hash);

CREATE INDEX IF NOT EXISTS project_block_order_idx
  ON project (chain_id, block_number, tx_index, id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_contract_code_deployment_count()
RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    UPDATE contract_code
    SET deployment_count = deployment_count + 1
    WHERE code_hash = NEW.code_hash;
    RETURN NEW;
  ELSIF TG_OP = 'DELETE' THEN
    UPDATE contract_code
    SET deployment_count = GREATEST(deployment_count - 1, 0)
    WHERE code_hash = OLD.code_hash;
    RETURN OLD;
  ELSIF TG_OP = 'UPDATE' THEN
    IF OLD.code_hash IS DISTINCT FROM NEW.code_hash THEN
      UPDATE contract_code
      SET deployment_count = GREATEST(deployment_count - 1, 0)
      WHERE code_hash = OLD.code_hash;

      UPDATE contract_code
      SET deployment_count = deployment_count + 1
      WHERE code_hash = NEW.code_hash;
    END IF;
    RETURN NEW;
  END IF;
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER project_contract_code_deployment_count_trigger
AFTER INSERT OR UPDATE OF code_hash OR DELETE ON project
FOR EACH ROW EXECUTE FUNCTION update_contract_code_deployment_count();

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

CREATE TABLE IF NOT EXISTS project_initial_recipient (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT NOT NULL,
  wallet BYTEA NOT NULL,
  ratio_bps BIGINT NOT NULL,
  rank_index INT NOT NULL,
  source_tx_hash BYTEA NOT NULL,
  source_block_number BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_initial_recipient_project_fk
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_initial_recipient_wallet_len CHECK (length(wallet) = 20),
  CONSTRAINT project_initial_recipient_source_tx_hash_len CHECK (length(source_tx_hash) = 32),
  CONSTRAINT project_initial_recipient_ratio_bps_nonnegative CHECK (ratio_bps >= 0),
  CONSTRAINT project_initial_recipient_rank_index_nonnegative CHECK (rank_index >= 0),
  CONSTRAINT project_initial_recipient_source_block_number_nonnegative CHECK (source_block_number >= 0),
  CONSTRAINT project_initial_recipient_project_wallet_uidx UNIQUE (project_id, wallet)
);

CREATE INDEX IF NOT EXISTS project_initial_recipient_project_rank_idx
  ON project_initial_recipient (project_id, rank_index);

CREATE INDEX IF NOT EXISTS project_initial_recipient_wallet_ratio_idx
  ON project_initial_recipient (wallet, ratio_bps DESC, project_id);

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
  revision BIGINT NOT NULL DEFAULT 1,
  attempts INT NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, data_type),
  CONSTRAINT project_data_collection_task_project_fk
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_data_collection_task_data_type_allowed
    CHECK (data_type IN ('ave', 'chain_state', 'wallet_asset_state', 'simulation_result', 'contract_code_source')),
  CONSTRAINT project_data_collection_task_status_allowed
    CHECK (status IN ('pending', 'succeeded', 'failed')),
  CONSTRAINT project_data_collection_task_revision_positive
    CHECK (revision > 0),
  CONSTRAINT project_data_collection_task_attempts_range
    CHECK (attempts >= 0 AND attempts <= 5)
);

CREATE INDEX IF NOT EXISTS project_data_collection_task_due_idx
  ON project_data_collection_task (status, next_attempt_at, data_type, project_id);

CREATE TABLE IF NOT EXISTS project_report (
  project_id BIGINT PRIMARY KEY,
  weth_pair_is_created BOOLEAN,
  weth_pair_is_remove_liquidity BOOLEAN,
  weth_pair_is_mint BOOLEAN,
  weth_pair_quote_usdt_value_int NUMERIC(78, 0),
  weth_pair_last_swap_timestamp BIGINT,
  usdt_pair_is_created BOOLEAN,
  usdt_pair_is_remove_liquidity BOOLEAN,
  usdt_pair_is_mint BOOLEAN,
  usdt_pair_quote_usdt_value_int NUMERIC(78, 0),
  usdt_pair_last_swap_timestamp BIGINT,
  source_updated_at TIMESTAMPTZ,
  evaluated_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_report_project_fk
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_report_weth_pair_quote_usdt_value_nonnegative
    CHECK (weth_pair_quote_usdt_value_int IS NULL OR weth_pair_quote_usdt_value_int >= 0),
  CONSTRAINT project_report_weth_pair_last_swap_timestamp_nonnegative
    CHECK (weth_pair_last_swap_timestamp IS NULL OR weth_pair_last_swap_timestamp >= 0),
  CONSTRAINT project_report_usdt_pair_quote_usdt_value_nonnegative
    CHECK (usdt_pair_quote_usdt_value_int IS NULL OR usdt_pair_quote_usdt_value_int >= 0),
  CONSTRAINT project_report_usdt_pair_last_swap_timestamp_nonnegative
    CHECK (usdt_pair_last_swap_timestamp IS NULL OR usdt_pair_last_swap_timestamp >= 0)
);

CREATE TABLE IF NOT EXISTS project_report_evaluation_task (
  project_id BIGINT PRIMARY KEY,
  status TEXT NOT NULL DEFAULT 'pending',
  revision BIGINT NOT NULL DEFAULT 1,
  attempts INT NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_report_evaluation_task_report_fk
    FOREIGN KEY (project_id) REFERENCES project_report(project_id) ON DELETE CASCADE,
  CONSTRAINT project_report_evaluation_task_status_allowed
    CHECK (status IN ('pending', 'succeeded', 'failed')),
  CONSTRAINT project_report_evaluation_task_revision_positive
    CHECK (revision > 0),
  CONSTRAINT project_report_evaluation_task_attempts_range
    CHECK (attempts >= 0 AND attempts <= 5)
);

CREATE INDEX IF NOT EXISTS project_report_evaluation_task_due_idx
  ON project_report_evaluation_task (status, next_attempt_at, project_id);

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
DROP TABLE IF EXISTS project_report_evaluation_task;
DROP TABLE IF EXISTS project_report;
DROP TABLE IF EXISTS project_data_collection_task;
DROP TABLE IF EXISTS project_simulation_result;
DROP TABLE IF EXISTS project_chain_state;
DROP TABLE IF EXISTS project_ave_data;
DROP TABLE IF EXISTS wallet_asset_state;
DROP TABLE IF EXISTS project_initial_recipient;
DROP TABLE IF EXISTS project_related_wallet;
DROP TABLE IF EXISTS project;
DROP TABLE IF EXISTS contract_code;
DROP TABLE IF EXISTS project_candidate;
DROP TABLE IF EXISTS chain_ingest_checkpoint;
DROP TABLE IF EXISTS chain;
DROP FUNCTION IF EXISTS update_contract_code_deployment_count();
