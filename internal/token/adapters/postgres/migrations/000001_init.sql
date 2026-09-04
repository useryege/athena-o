-- +goose Up

-- Discovery
CREATE TABLE chain (
  id BIGINT,
  name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chain_id_uidx PRIMARY KEY (id),
  CONSTRAINT chain_name_not_empty_check CHECK (btrim(name) <> '')
);

CREATE TABLE chain_processing_checkpoint (
  chain_id BIGINT,
  cursor_block_number BIGINT NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'stopped',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chain_processing_checkpoint_chain_id_uidx PRIMARY KEY (chain_id),
  CONSTRAINT chain_processing_checkpoint_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT chain_processing_checkpoint_cursor_block_number_check CHECK (cursor_block_number >= 0),
  CONSTRAINT chain_processing_checkpoint_status_check CHECK (status IN ('running', 'stopped'))
);

CREATE TABLE chain_block_processing_attempt (
  id BIGSERIAL,
  chain_id BIGINT NOT NULL,
  block_number BIGINT NOT NULL,
  attempt_number INT NOT NULL,
  block_time BIGINT,
  status TEXT NOT NULL DEFAULT 'running',
  terminal_stage TEXT,
  error_message TEXT,
  checkpoint_read_duration_us BIGINT,
  discovery_duration_us BIGINT,
  validation_duration_us BIGINT,
  persistence_duration_us BIGINT,
  total_duration_us BIGINT,
  candidate_count INT,
  validated_count INT,
  rejected_count INT,
  timing_complete BOOLEAN NOT NULL DEFAULT false,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chain_block_processing_attempt_id_uidx PRIMARY KEY (id),
  CONSTRAINT chain_block_processing_attempt_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT chain_block_processing_attempt_position_uidx UNIQUE (chain_id, block_number, attempt_number),
  CONSTRAINT chain_block_processing_attempt_block_number_check CHECK (block_number >= 0),
  CONSTRAINT chain_block_processing_attempt_attempt_number_check CHECK (attempt_number > 0),
  CONSTRAINT chain_block_processing_attempt_block_time_check CHECK (block_time IS NULL OR block_time >= 0),
  CONSTRAINT chain_block_processing_attempt_status_check CHECK (status IN ('running', 'succeeded', 'failed', 'cancelled', 'interrupted')),
  CONSTRAINT chain_block_processing_attempt_terminal_stage_check CHECK (
    terminal_stage IS NULL OR terminal_stage IN ('checkpoint_read', 'candidate_discovery', 'candidate_validation', 'persistence')
  ),
  CONSTRAINT chain_block_processing_attempt_checkpoint_read_duration_check CHECK (checkpoint_read_duration_us IS NULL OR checkpoint_read_duration_us >= 0),
  CONSTRAINT chain_block_processing_attempt_discovery_duration_check CHECK (discovery_duration_us IS NULL OR discovery_duration_us >= 0),
  CONSTRAINT chain_block_processing_attempt_validation_duration_check CHECK (validation_duration_us IS NULL OR validation_duration_us >= 0),
  CONSTRAINT chain_block_processing_attempt_persistence_duration_check CHECK (persistence_duration_us IS NULL OR persistence_duration_us >= 0),
  CONSTRAINT chain_block_processing_attempt_total_duration_check CHECK (total_duration_us IS NULL OR total_duration_us >= 0),
  CONSTRAINT chain_block_processing_attempt_candidate_count_check CHECK (candidate_count IS NULL OR candidate_count >= 0),
  CONSTRAINT chain_block_processing_attempt_validated_count_check CHECK (validated_count IS NULL OR validated_count >= 0),
  CONSTRAINT chain_block_processing_attempt_rejected_count_check CHECK (rejected_count IS NULL OR rejected_count >= 0),
  CONSTRAINT chain_block_processing_attempt_completion_check CHECK (
    (status = 'running' AND completed_at IS NULL AND timing_complete = false)
    OR (status <> 'running' AND completed_at IS NOT NULL)
  ),
  CONSTRAINT chain_block_processing_attempt_complete_timing_check CHECK (
    timing_complete = false
    OR (
      status = 'succeeded'
      AND checkpoint_read_duration_us IS NOT NULL
      AND discovery_duration_us IS NOT NULL
      AND validation_duration_us IS NOT NULL
      AND persistence_duration_us IS NOT NULL
      AND total_duration_us IS NOT NULL
    )
  )
);

CREATE INDEX chain_block_processing_attempt_chain_block_idx
  ON chain_block_processing_attempt (chain_id, block_number DESC, attempt_number DESC);
CREATE INDEX chain_block_processing_attempt_chain_block_time_idx
  ON chain_block_processing_attempt (chain_id, block_time DESC, block_number DESC);
CREATE INDEX chain_block_processing_attempt_chain_status_block_time_idx
  ON chain_block_processing_attempt (chain_id, status, block_time DESC, block_number DESC);

CREATE TABLE project_candidate (
  id BIGSERIAL,
  chain_id BIGINT NOT NULL,
  contract BYTEA NOT NULL,
  tx_sender BYTEA NOT NULL,
  tx_hash BYTEA NOT NULL,
  tx_index BIGINT NOT NULL,
  deployment_nonce BIGINT NOT NULL,
  block_number BIGINT NOT NULL,
  block_time BIGINT NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_candidate_id_uidx PRIMARY KEY (id),
  CONSTRAINT project_candidate_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT project_candidate_contract_length_check CHECK (length(contract) = 20),
  CONSTRAINT project_candidate_tx_sender_length_check CHECK (length(tx_sender) = 20),
  CONSTRAINT project_candidate_tx_hash_length_check CHECK (length(tx_hash) = 32),
  CONSTRAINT project_candidate_tx_index_check CHECK (tx_index >= 0),
  CONSTRAINT project_candidate_deployment_nonce_check CHECK (deployment_nonce >= 0),
  CONSTRAINT project_candidate_block_number_check CHECK (block_number >= 0),
  CONSTRAINT project_candidate_block_time_check CHECK (block_time >= 0),
  CONSTRAINT project_candidate_status_check CHECK (status IN ('validated', 'rejected'))
);

CREATE UNIQUE INDEX project_candidate_chain_id_contract_uidx
  ON project_candidate (chain_id, contract);
CREATE UNIQUE INDEX project_candidate_chain_id_tx_hash_uidx
  ON project_candidate (chain_id, tx_hash);
CREATE INDEX project_candidate_status_created_at_idx
  ON project_candidate (status, created_at, id);

-- Catalog
CREATE TABLE contract_code (
  code_hash BYTEA,
  source_code TEXT,
  source_code_fetched_at TIMESTAMPTZ,
  deployment_count BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT contract_code_code_hash_uidx PRIMARY KEY (code_hash),
  CONSTRAINT contract_code_code_hash_length_check CHECK (length(code_hash) = 32),
  CONSTRAINT contract_code_deployment_count_check CHECK (deployment_count >= 0)
);

CREATE INDEX contract_code_deployment_count_created_at_idx
  ON contract_code (deployment_count DESC, created_at DESC, code_hash);

CREATE TABLE project (
  id BIGSERIAL,
  chain_id BIGINT NOT NULL,
  contract BYTEA NOT NULL,
  tx_sender BYTEA NOT NULL,
  tx_hash BYTEA NOT NULL,
  tx_index BIGINT NOT NULL,
  deployment_nonce BIGINT NOT NULL,
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
  CONSTRAINT project_id_uidx PRIMARY KEY (id),
  CONSTRAINT project_id_chain_id_uidx UNIQUE (id, chain_id),
  CONSTRAINT project_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT project_contract_code_fk FOREIGN KEY (code_hash) REFERENCES contract_code(code_hash),
  CONSTRAINT project_contract_length_check CHECK (length(contract) = 20),
  CONSTRAINT project_tx_sender_length_check CHECK (length(tx_sender) = 20),
  CONSTRAINT project_tx_hash_length_check CHECK (length(tx_hash) = 32),
  CONSTRAINT project_code_hash_length_check CHECK (length(code_hash) = 32),
  CONSTRAINT project_weth_pair_length_check CHECK (weth_pair IS NULL OR length(weth_pair) = 20),
  CONSTRAINT project_usdt_pair_length_check CHECK (usdt_pair IS NULL OR length(usdt_pair) = 20),
  CONSTRAINT project_tx_index_check CHECK (tx_index >= 0),
  CONSTRAINT project_deployment_nonce_check CHECK (deployment_nonce >= 0),
  CONSTRAINT project_block_number_check CHECK (block_number >= 0),
  CONSTRAINT project_block_time_check CHECK (block_time >= 0),
  CONSTRAINT project_decimals_check CHECK (decimals BETWEEN 0 AND 255),
  CONSTRAINT project_total_supply_check CHECK (total_supply >= 0)
);

CREATE UNIQUE INDEX project_chain_id_contract_uidx ON project (chain_id, contract);
CREATE UNIQUE INDEX project_chain_id_tx_hash_uidx ON project (chain_id, tx_hash);
CREATE INDEX project_code_hash_idx ON project (code_hash);
CREATE INDEX project_chain_id_block_number_tx_index_idx
  ON project (chain_id, block_number, tx_index, id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_contract_code_deployment_count()
RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    UPDATE contract_code SET deployment_count = deployment_count + 1 WHERE code_hash = NEW.code_hash;
    RETURN NEW;
  ELSIF TG_OP = 'DELETE' THEN
    UPDATE contract_code SET deployment_count = GREATEST(deployment_count - 1, 0) WHERE code_hash = OLD.code_hash;
    RETURN OLD;
  ELSIF OLD.code_hash IS DISTINCT FROM NEW.code_hash THEN
    UPDATE contract_code SET deployment_count = GREATEST(deployment_count - 1, 0) WHERE code_hash = OLD.code_hash;
    UPDATE contract_code SET deployment_count = deployment_count + 1 WHERE code_hash = NEW.code_hash;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER project_contract_code_deployment_count_trigger
AFTER INSERT OR UPDATE OF code_hash OR DELETE ON project
FOR EACH ROW EXECUTE FUNCTION update_contract_code_deployment_count();

CREATE TABLE project_related_wallet (
  project_id BIGINT NOT NULL,
  wallet BYTEA NOT NULL,
  role TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_related_wallet_project_id_wallet_role_uidx PRIMARY KEY (project_id, wallet, role),
  CONSTRAINT project_related_wallet_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_related_wallet_wallet_length_check CHECK (length(wallet) = 20),
  CONSTRAINT project_related_wallet_role_check CHECK (btrim(role) <> '')
);

CREATE INDEX project_related_wallet_project_id_role_idx ON project_related_wallet (project_id, role);
CREATE INDEX project_related_wallet_wallet_idx ON project_related_wallet (wallet);

CREATE TABLE project_wallet_normal_transaction (
  project_id BIGINT NOT NULL,
  wallet BYTEA NOT NULL,
  transaction_hash BYTEA NOT NULL,
  block_number BIGINT NOT NULL,
  block_timestamp BIGINT NOT NULL,
  transaction_index BIGINT NOT NULL,
  nonce BIGINT NOT NULL,
  from_address BYTEA NOT NULL,
  to_address BYTEA,
  value NUMERIC(78, 0) NOT NULL,
  gas BIGINT NOT NULL,
  gas_price NUMERIC(78, 0) NOT NULL,
  gas_used BIGINT NOT NULL,
  input TEXT NOT NULL,
  method_id TEXT NOT NULL,
  function_name TEXT NOT NULL,
  receipt_status TEXT NOT NULL,
  is_error BOOLEAN NOT NULL,
  collected_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT project_wallet_normal_transaction_project_id_wallet_transaction_hash_uidx
    PRIMARY KEY (project_id, wallet, transaction_hash),
  CONSTRAINT project_wallet_normal_transaction_project_fk
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_wallet_normal_transaction_wallet_length_check CHECK (length(wallet) = 20),
  CONSTRAINT project_wallet_normal_transaction_transaction_hash_length_check CHECK (length(transaction_hash) = 32),
  CONSTRAINT project_wallet_normal_transaction_block_number_check CHECK (block_number >= 0),
  CONSTRAINT project_wallet_normal_transaction_block_timestamp_check CHECK (block_timestamp >= 0),
  CONSTRAINT project_wallet_normal_transaction_transaction_index_check CHECK (transaction_index >= 0),
  CONSTRAINT project_wallet_normal_transaction_nonce_check CHECK (nonce >= 0),
  CONSTRAINT project_wallet_normal_transaction_from_address_length_check CHECK (length(from_address) = 20),
  CONSTRAINT project_wallet_normal_transaction_to_address_length_check CHECK (to_address IS NULL OR length(to_address) = 20),
  CONSTRAINT project_wallet_normal_transaction_value_check CHECK (value >= 0),
  CONSTRAINT project_wallet_normal_transaction_gas_check CHECK (gas >= 0),
  CONSTRAINT project_wallet_normal_transaction_gas_price_check CHECK (gas_price >= 0),
  CONSTRAINT project_wallet_normal_transaction_gas_used_check CHECK (gas_used >= 0),
  CONSTRAINT project_wallet_normal_transaction_receipt_status_check
    CHECK (receipt_status IN ('unspecified', 'failed', 'success'))
);

CREATE INDEX project_wallet_normal_transaction_project_wallet_position_idx
  ON project_wallet_normal_transaction (project_id, wallet, block_number DESC, transaction_index DESC);

CREATE TABLE project_initial_recipient (
  id BIGSERIAL,
  project_id BIGINT NOT NULL,
  wallet BYTEA NOT NULL,
  ratio_bps BIGINT NOT NULL,
  rank_index INT NOT NULL,
  source_tx_hash BYTEA NOT NULL,
  source_block_number BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_initial_recipient_id_uidx PRIMARY KEY (id),
  CONSTRAINT project_initial_recipient_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_initial_recipient_wallet_length_check CHECK (length(wallet) = 20),
  CONSTRAINT project_initial_recipient_source_tx_hash_length_check CHECK (length(source_tx_hash) = 32),
  CONSTRAINT project_initial_recipient_ratio_bps_check CHECK (ratio_bps >= 0),
  CONSTRAINT project_initial_recipient_rank_index_check CHECK (rank_index >= 0),
  CONSTRAINT project_initial_recipient_source_block_number_check CHECK (source_block_number >= 0),
  CONSTRAINT project_initial_recipient_project_id_wallet_uidx UNIQUE (project_id, wallet)
);

CREATE INDEX project_initial_recipient_project_id_rank_index_idx
  ON project_initial_recipient (project_id, rank_index);
CREATE INDEX project_initial_recipient_wallet_ratio_bps_idx
  ON project_initial_recipient (wallet, ratio_bps DESC, project_id);

-- One-time project data collection
CREATE TABLE project_data_collection_task (
  id BIGSERIAL,
  project_id BIGINT NOT NULL,
  data_type TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  failure_count INT NOT NULL DEFAULT 0,
  available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  claim_generation BIGINT NOT NULL DEFAULT 0,
  locked_at TIMESTAMPTZ,
  lease_expires_at TIMESTAMPTZ,
  last_error TEXT,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_data_collection_task_id_uidx PRIMARY KEY (id),
  CONSTRAINT project_data_collection_task_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_data_collection_task_project_id_data_type_uidx UNIQUE (project_id, data_type),
  CONSTRAINT project_data_collection_task_identity_uidx UNIQUE (id, project_id, data_type),
  CONSTRAINT project_data_collection_task_data_type_check CHECK (data_type IN ('ave', 'chain_state', 'wallet_asset_state', 'simulation_result', 'contract_code_source', 'wallet_normal_transactions')),
  CONSTRAINT project_data_collection_task_status_check CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
  CONSTRAINT project_data_collection_task_failure_count_check CHECK (failure_count BETWEEN 0 AND 3),
  CONSTRAINT project_data_collection_task_claim_generation_check CHECK (claim_generation >= 0),
  CONSTRAINT project_data_collection_task_state_check CHECK (
    (status = 'pending' AND failure_count < 3 AND locked_at IS NULL AND lease_expires_at IS NULL AND finished_at IS NULL)
    OR (status = 'running' AND failure_count < 3 AND locked_at IS NOT NULL AND lease_expires_at IS NOT NULL AND finished_at IS NULL)
    OR (status = 'succeeded' AND failure_count < 3 AND locked_at IS NOT NULL AND lease_expires_at IS NULL AND finished_at IS NOT NULL)
    OR (status = 'failed' AND failure_count = 3 AND locked_at IS NOT NULL AND lease_expires_at IS NULL AND finished_at IS NOT NULL)
  )
);

CREATE INDEX project_data_collection_task_pending_claim_idx
  ON project_data_collection_task (data_type, available_at, id)
  WHERE status = 'pending';
CREATE INDEX project_data_collection_task_running_lease_idx
  ON project_data_collection_task (data_type, lease_expires_at, id)
  WHERE status = 'running';

CREATE TABLE project_data_collection_result (
  task_id BIGINT,
  project_id BIGINT NOT NULL,
  data_type TEXT NOT NULL,
  schema_version INT NOT NULL,
  payload JSONB NOT NULL,
  content_hash BYTEA NOT NULL,
  block_number BIGINT,
  collected_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_data_collection_result_task_id_uidx PRIMARY KEY (task_id),
  CONSTRAINT project_data_collection_result_project_id_data_type_uidx UNIQUE (project_id, data_type),
  CONSTRAINT project_data_collection_result_task_fk
    FOREIGN KEY (task_id, project_id, data_type)
    REFERENCES project_data_collection_task(id, project_id, data_type) ON DELETE CASCADE,
  CONSTRAINT project_data_collection_result_data_type_check CHECK (data_type IN ('ave', 'chain_state', 'wallet_asset_state', 'simulation_result', 'contract_code_source', 'wallet_normal_transactions')),
  CONSTRAINT project_data_collection_result_schema_version_check CHECK (schema_version > 0),
  CONSTRAINT project_data_collection_result_content_hash_length_check CHECK (length(content_hash) = 32),
  CONSTRAINT project_data_collection_result_block_number_check CHECK (block_number IS NULL OR block_number >= 0)
);

-- Unique project profile construction
CREATE TABLE project_profile_build_task (
  project_id BIGINT,
  status TEXT NOT NULL DEFAULT 'pending',
  failure_count INT NOT NULL DEFAULT 0,
  available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  claim_generation BIGINT NOT NULL DEFAULT 0,
  locked_at TIMESTAMPTZ,
  lease_expires_at TIMESTAMPTZ,
  last_error TEXT,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_profile_build_task_project_id_uidx PRIMARY KEY (project_id),
  CONSTRAINT project_profile_build_task_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_profile_build_task_status_check CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
  CONSTRAINT project_profile_build_task_failure_count_check CHECK (failure_count BETWEEN 0 AND 3),
  CONSTRAINT project_profile_build_task_claim_generation_check CHECK (claim_generation >= 0),
  CONSTRAINT project_profile_build_task_state_check CHECK (
    (status = 'pending' AND failure_count < 3 AND locked_at IS NULL AND lease_expires_at IS NULL AND finished_at IS NULL)
    OR (status = 'running' AND failure_count < 3 AND locked_at IS NOT NULL AND lease_expires_at IS NOT NULL AND finished_at IS NULL)
    OR (status = 'succeeded' AND failure_count < 3 AND locked_at IS NOT NULL AND lease_expires_at IS NULL AND finished_at IS NOT NULL)
    OR (status = 'failed' AND failure_count = 3 AND locked_at IS NOT NULL AND lease_expires_at IS NULL AND finished_at IS NOT NULL)
  )
);

CREATE INDEX project_profile_build_task_pending_claim_idx
  ON project_profile_build_task (available_at, project_id)
  WHERE status = 'pending';
CREATE INDEX project_profile_build_task_running_lease_idx
  ON project_profile_build_task (lease_expires_at, project_id)
  WHERE status = 'running';

CREATE TABLE project_profile (
  project_id BIGINT,
  schema_version INT NOT NULL,
  completeness_status TEXT NOT NULL,
  failed_data_types TEXT[] NOT NULL DEFAULT '{}',
  profile JSONB NOT NULL,
  content_hash BYTEA NOT NULL,
  logo_url TEXT NOT NULL DEFAULT '',
  current_price_usd NUMERIC,
  market_cap_usd NUMERIC,
  fdv_usd NUMERIC,
  tvl_usd NUMERIC,
  holders BIGINT,
  contract_source_status TEXT,
  weth_pair_is_created BOOLEAN,
  weth_pair_token_balance_exceeds_total_supply BOOLEAN,
  weth_pair_lp_minimum_supply_only BOOLEAN,
  weth_pair_fixed_fee_address_lp_share_gte_90_percent BOOLEAN,
  weth_pair_quote_usdt_value_int NUMERIC(78, 0),
  weth_pair_reserve_updated_at BIGINT,
  usdt_pair_is_created BOOLEAN,
  usdt_pair_token_balance_exceeds_total_supply BOOLEAN,
  usdt_pair_lp_minimum_supply_only BOOLEAN,
  usdt_pair_fixed_fee_address_lp_share_gte_90_percent BOOLEAN,
  usdt_pair_quote_usdt_value_int NUMERIC(78, 0),
  usdt_pair_reserve_updated_at BIGINT,
  built_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_profile_project_id_uidx PRIMARY KEY (project_id),
  CONSTRAINT project_profile_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_profile_schema_version_check CHECK (schema_version > 0),
  CONSTRAINT project_profile_completeness_status_check CHECK (completeness_status IN ('complete', 'incomplete')),
  CONSTRAINT project_profile_failed_data_types_check CHECK (
    failed_data_types <@ ARRAY['ave', 'chain_state', 'wallet_asset_state', 'simulation_result', 'contract_code_source', 'wallet_normal_transactions']::text[]
    AND cardinality(failed_data_types) <= 6
    AND (
      (completeness_status = 'complete' AND cardinality(failed_data_types) = 0)
      OR (completeness_status = 'incomplete' AND cardinality(failed_data_types) > 0)
    )
  ),
  CONSTRAINT project_profile_content_hash_length_check CHECK (length(content_hash) = 32),
  CONSTRAINT project_profile_market_values_check CHECK (
    (current_price_usd IS NULL OR current_price_usd >= 0)
    AND (market_cap_usd IS NULL OR market_cap_usd >= 0)
    AND (fdv_usd IS NULL OR fdv_usd >= 0)
    AND (tvl_usd IS NULL OR tvl_usd >= 0)
    AND (holders IS NULL OR holders >= 0)
  ),
  CONSTRAINT project_profile_contract_source_status_check CHECK (contract_source_status IS NULL OR contract_source_status IN ('verified', 'unverified')),
  CONSTRAINT project_profile_weth_pair_values_check CHECK (
    (weth_pair_quote_usdt_value_int IS NULL OR weth_pair_quote_usdt_value_int >= 0)
    AND (weth_pair_reserve_updated_at IS NULL OR weth_pair_reserve_updated_at >= 0)
  ),
  CONSTRAINT project_profile_usdt_pair_values_check CHECK (
    (usdt_pair_quote_usdt_value_int IS NULL OR usdt_pair_quote_usdt_value_int >= 0)
    AND (usdt_pair_reserve_updated_at IS NULL OR usdt_pair_reserve_updated_at >= 0)
  )
);

CREATE INDEX project_profile_completeness_built_at_idx
  ON project_profile (completeness_status, built_at DESC, project_id);

-- Policy
CREATE TABLE contract_code_blocklist (
  code_hash BYTEA,
  note TEXT,
  source_chain_id BIGINT,
  source_contract BYTEA,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT contract_code_blocklist_code_hash_uidx PRIMARY KEY (code_hash),
  CONSTRAINT contract_code_blocklist_source_chain_fk FOREIGN KEY (source_chain_id) REFERENCES chain(id),
  CONSTRAINT contract_code_blocklist_code_hash_length_check CHECK (length(code_hash) = 32),
  CONSTRAINT contract_code_blocklist_source_contract_length_check CHECK (source_contract IS NULL OR length(source_contract) = 20)
);

CREATE TABLE wallet_blocklist (
  wallet BYTEA,
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT wallet_blocklist_wallet_uidx PRIMARY KEY (wallet),
  CONSTRAINT wallet_blocklist_wallet_length_check CHECK (length(wallet) = 20)
);

-- +goose Down

DROP TABLE IF EXISTS wallet_blocklist;
DROP TABLE IF EXISTS contract_code_blocklist;
DROP TABLE IF EXISTS project_profile;
DROP TABLE IF EXISTS project_profile_build_task;
DROP TABLE IF EXISTS project_data_collection_result;
DROP TABLE IF EXISTS project_data_collection_task;
DROP TABLE IF EXISTS project_initial_recipient;
DROP TABLE IF EXISTS project_wallet_normal_transaction;
DROP TABLE IF EXISTS project_related_wallet;
DROP TRIGGER IF EXISTS project_contract_code_deployment_count_trigger ON project;
DROP TABLE IF EXISTS project;
DROP FUNCTION IF EXISTS update_contract_code_deployment_count();
DROP TABLE IF EXISTS contract_code;
DROP TABLE IF EXISTS project_candidate;
DROP TABLE IF EXISTS chain_block_processing_attempt;
DROP TABLE IF EXISTS chain_processing_checkpoint;
DROP TABLE IF EXISTS chain;
