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

CREATE TABLE chain_ingest_checkpoint (
  chain_id BIGINT,
  cursor_block_number BIGINT NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'stopped',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chain_ingest_checkpoint_chain_id_uidx PRIMARY KEY (chain_id),
  CONSTRAINT chain_ingest_checkpoint_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT chain_ingest_checkpoint_cursor_block_number_check CHECK (cursor_block_number >= 0),
  CONSTRAINT chain_ingest_checkpoint_status_check CHECK (status IN ('running', 'stopped'))
);

CREATE TABLE project_candidate (
  id BIGSERIAL,
  chain_id BIGINT NOT NULL,
  contract BYTEA NOT NULL,
  tx_sender BYTEA NOT NULL,
  tx_hash BYTEA NOT NULL,
  tx_index BIGINT NOT NULL,
  block_number BIGINT NOT NULL,
  block_time BIGINT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  validation_lock_token UUID,
  validation_locked_at TIMESTAMPTZ,
  validation_lease_expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_candidate_id_uidx PRIMARY KEY (id),
  CONSTRAINT project_candidate_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT project_candidate_contract_length_check CHECK (length(contract) = 20),
  CONSTRAINT project_candidate_tx_sender_length_check CHECK (length(tx_sender) = 20),
  CONSTRAINT project_candidate_tx_hash_length_check CHECK (length(tx_hash) = 32),
  CONSTRAINT project_candidate_tx_index_check CHECK (tx_index >= 0),
  CONSTRAINT project_candidate_block_number_check CHECK (block_number >= 0),
  CONSTRAINT project_candidate_block_time_check CHECK (block_time >= 0),
  CONSTRAINT project_candidate_status_check CHECK (status IN ('pending', 'validated', 'rejected')),
  CONSTRAINT project_candidate_validation_lease_check CHECK (
    (
      validation_lock_token IS NULL
      AND validation_locked_at IS NULL
      AND validation_lease_expires_at IS NULL
    )
    OR
    (
      status = 'pending'
      AND validation_lock_token IS NOT NULL
      AND validation_locked_at IS NOT NULL
      AND validation_lease_expires_at IS NOT NULL
    )
  )
);

CREATE UNIQUE INDEX project_candidate_chain_id_contract_uidx
  ON project_candidate (chain_id, contract);
CREATE UNIQUE INDEX project_candidate_chain_id_tx_hash_uidx
  ON project_candidate (chain_id, tx_hash);
CREATE INDEX project_candidate_status_created_at_idx
  ON project_candidate (status, created_at, id);
CREATE INDEX project_candidate_pending_validation_lease_idx
  ON project_candidate (validation_lease_expires_at, created_at, id)
  WHERE status = 'pending';

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
  CONSTRAINT project_chain_fk FOREIGN KEY (chain_id) REFERENCES chain(id),
  CONSTRAINT project_contract_code_fk FOREIGN KEY (code_hash) REFERENCES contract_code(code_hash),
  CONSTRAINT project_contract_length_check CHECK (length(contract) = 20),
  CONSTRAINT project_tx_sender_length_check CHECK (length(tx_sender) = 20),
  CONSTRAINT project_tx_hash_length_check CHECK (length(tx_hash) = 32),
  CONSTRAINT project_code_hash_length_check CHECK (length(code_hash) = 32),
  CONSTRAINT project_weth_pair_length_check CHECK (weth_pair IS NULL OR length(weth_pair) = 20),
  CONSTRAINT project_usdt_pair_length_check CHECK (usdt_pair IS NULL OR length(usdt_pair) = 20),
  CONSTRAINT project_tx_index_check CHECK (tx_index >= 0),
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

-- Research
CREATE TABLE project_research_state (
  project_id BIGINT,
  status TEXT NOT NULL DEFAULT 'researching',
  evidence_revision BIGINT NOT NULL DEFAULT 0,
  current_report_revision BIGINT,
  current_selection_id BIGINT,
  last_evaluated_report_revision BIGINT,
  last_evaluated_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + INTERVAL '7 days'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_research_state_project_id_uidx PRIMARY KEY (project_id),
  CONSTRAINT project_research_state_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_research_state_status_check CHECK (status IN ('researching', 'selected', 'rejected', 'expired')),
  CONSTRAINT project_research_state_evidence_revision_check CHECK (evidence_revision >= 0),
  CONSTRAINT project_research_state_current_report_revision_check CHECK (current_report_revision IS NULL OR current_report_revision > 0),
  CONSTRAINT project_research_state_last_evaluated_report_revision_check CHECK (last_evaluated_report_revision IS NULL OR last_evaluated_report_revision > 0)
);

CREATE INDEX project_research_state_status_expires_at_idx
  ON project_research_state (status, expires_at, project_id);

CREATE TABLE project_data_collection_schedule (
  project_id BIGINT NOT NULL,
  data_type TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  refresh_interval_seconds BIGINT NOT NULL,
  next_run_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  latest_task_revision BIGINT NOT NULL DEFAULT 0,
  consecutive_failures INT NOT NULL DEFAULT 0,
  last_error TEXT,
  last_checked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_data_collection_schedule_project_id_data_type_uidx PRIMARY KEY (project_id, data_type),
  CONSTRAINT project_data_collection_schedule_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_data_collection_schedule_data_type_check CHECK (data_type IN ('ave', 'chain_state', 'wallet_asset_state', 'simulation_result', 'contract_code_source')),
  CONSTRAINT project_data_collection_schedule_status_check CHECK (status IN ('active', 'completed', 'paused')),
  CONSTRAINT project_data_collection_schedule_refresh_interval_seconds_check CHECK (refresh_interval_seconds > 0),
  CONSTRAINT project_data_collection_schedule_latest_task_revision_check CHECK (latest_task_revision >= 0),
  CONSTRAINT project_data_collection_schedule_consecutive_failures_check CHECK (consecutive_failures >= 0)
);

CREATE INDEX project_data_collection_schedule_status_next_run_at_idx
  ON project_data_collection_schedule (status, next_run_at, data_type, project_id);

CREATE TABLE project_data_collection_task (
  id BIGSERIAL,
  project_id BIGINT NOT NULL,
  data_type TEXT NOT NULL,
  revision BIGINT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  attempts INT NOT NULL DEFAULT 0,
  available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  locked_at TIMESTAMPTZ,
  lease_expires_at TIMESTAMPTZ,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_data_collection_task_id_uidx PRIMARY KEY (id),
  CONSTRAINT project_data_collection_task_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_data_collection_task_project_id_data_type_revision_uidx UNIQUE (project_id, data_type, revision),
  CONSTRAINT project_data_collection_task_data_type_check CHECK (data_type IN ('ave', 'chain_state', 'wallet_asset_state', 'simulation_result', 'contract_code_source')),
  CONSTRAINT project_data_collection_task_status_check CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
  CONSTRAINT project_data_collection_task_revision_check CHECK (revision > 0),
  CONSTRAINT project_data_collection_task_attempts_check CHECK (attempts BETWEEN 0 AND 5)
);

CREATE INDEX project_data_collection_task_status_available_at_idx
  ON project_data_collection_task (status, available_at, data_type, project_id);
CREATE INDEX project_data_collection_task_lease_expires_at_idx
  ON project_data_collection_task (lease_expires_at) WHERE status = 'running';

CREATE TABLE project_observation (
  id BIGSERIAL,
  project_id BIGINT NOT NULL,
  data_type TEXT NOT NULL,
  schema_version INT NOT NULL,
  content_hash BYTEA NOT NULL,
  payload JSONB NOT NULL,
  block_number BIGINT,
  observed_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_observation_id_uidx PRIMARY KEY (id),
  CONSTRAINT project_observation_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_observation_data_type_check CHECK (data_type IN ('ave', 'chain_state', 'wallet_asset_state', 'simulation_result', 'contract_code_source')),
  CONSTRAINT project_observation_schema_version_check CHECK (schema_version > 0),
  CONSTRAINT project_observation_content_hash_length_check CHECK (length(content_hash) = 32),
  CONSTRAINT project_observation_block_number_check CHECK (block_number IS NULL OR block_number >= 0)
);

CREATE INDEX project_observation_project_id_data_type_created_at_idx
  ON project_observation (project_id, data_type, created_at DESC, id DESC);

CREATE TABLE project_observation_current (
  project_id BIGINT NOT NULL,
  data_type TEXT NOT NULL,
  observation_id BIGINT NOT NULL,
  last_checked_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_observation_current_project_id_data_type_uidx PRIMARY KEY (project_id, data_type),
  CONSTRAINT project_observation_current_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_observation_current_observation_fk FOREIGN KEY (observation_id) REFERENCES project_observation(id) ON DELETE CASCADE,
  CONSTRAINT project_observation_current_data_type_check CHECK (data_type IN ('ave', 'chain_state', 'wallet_asset_state', 'simulation_result', 'contract_code_source'))
);

CREATE INDEX project_observation_current_observation_id_idx
  ON project_observation_current (observation_id);

-- Reporting
CREATE TABLE project_report_revision (
  id BIGSERIAL,
  project_id BIGINT NOT NULL,
  revision BIGINT NOT NULL,
  schema_version INT NOT NULL,
  content_hash BYTEA NOT NULL,
  completeness_status TEXT NOT NULL,
  evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
  report JSONB NOT NULL DEFAULT '{}'::jsonb,
  observed_block_number BIGINT,
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
  built_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_report_revision_id_uidx PRIMARY KEY (id),
  CONSTRAINT project_report_revision_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_report_revision_project_id_revision_uidx UNIQUE (project_id, revision),
  CONSTRAINT project_report_revision_revision_check CHECK (revision > 0),
  CONSTRAINT project_report_revision_schema_version_check CHECK (schema_version > 0),
  CONSTRAINT project_report_revision_content_hash_length_check CHECK (length(content_hash) = 32),
  CONSTRAINT project_report_revision_completeness_status_check CHECK (completeness_status IN ('incomplete', 'complete')),
  CONSTRAINT project_report_revision_observed_block_number_check CHECK (observed_block_number IS NULL OR observed_block_number >= 0),
  CONSTRAINT project_report_revision_weth_pair_quote_usdt_value_int_check CHECK (weth_pair_quote_usdt_value_int IS NULL OR weth_pair_quote_usdt_value_int >= 0),
  CONSTRAINT project_report_revision_weth_pair_last_swap_timestamp_check CHECK (weth_pair_last_swap_timestamp IS NULL OR weth_pair_last_swap_timestamp >= 0),
  CONSTRAINT project_report_revision_usdt_pair_quote_usdt_value_int_check CHECK (usdt_pair_quote_usdt_value_int IS NULL OR usdt_pair_quote_usdt_value_int >= 0),
  CONSTRAINT project_report_revision_usdt_pair_last_swap_timestamp_check CHECK (usdt_pair_last_swap_timestamp IS NULL OR usdt_pair_last_swap_timestamp >= 0)
);

CREATE INDEX project_report_revision_project_id_revision_idx
  ON project_report_revision (project_id, revision DESC);

CREATE TABLE project_report_build_task (
  id BIGSERIAL,
  project_id BIGINT NOT NULL,
  evidence_revision BIGINT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  attempts INT NOT NULL DEFAULT 0,
  available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  locked_at TIMESTAMPTZ,
  lease_expires_at TIMESTAMPTZ,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_report_build_task_id_uidx PRIMARY KEY (id),
  CONSTRAINT project_report_build_task_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_report_build_task_project_id_evidence_revision_uidx UNIQUE (project_id, evidence_revision),
  CONSTRAINT project_report_build_task_evidence_revision_check CHECK (evidence_revision > 0),
  CONSTRAINT project_report_build_task_status_check CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
  CONSTRAINT project_report_build_task_attempts_check CHECK (attempts BETWEEN 0 AND 5)
);

CREATE INDEX project_report_build_task_status_available_at_idx
  ON project_report_build_task (status, available_at, project_id);

-- Selection
CREATE TABLE project_selection (
  id BIGSERIAL,
  project_id BIGINT NOT NULL,
  outcome TEXT NOT NULL,
  strategy_key TEXT NOT NULL,
  strategy_version TEXT NOT NULL,
  report_revision BIGINT NOT NULL,
  reason_codes TEXT[] NOT NULL DEFAULT '{}',
  reason_detail TEXT NOT NULL DEFAULT '',
  decided_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_selection_id_uidx PRIMARY KEY (id),
  CONSTRAINT project_selection_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_selection_project_report_fk FOREIGN KEY (project_id, report_revision) REFERENCES project_report_revision(project_id, revision) ON DELETE CASCADE,
  CONSTRAINT project_selection_outcome_check CHECK (outcome IN ('selected', 'rejected', 'deferred')),
  CONSTRAINT project_selection_strategy_key_check CHECK (btrim(strategy_key) <> ''),
  CONSTRAINT project_selection_strategy_version_check CHECK (btrim(strategy_version) <> ''),
  CONSTRAINT project_selection_report_revision_check CHECK (report_revision > 0)
);

CREATE INDEX project_selection_project_id_decided_at_idx
  ON project_selection (project_id, decided_at DESC, id DESC);
CREATE INDEX project_selection_outcome_decided_at_idx
  ON project_selection (outcome, decided_at DESC, id DESC);

CREATE TABLE project_selection_evaluation_task (
  id BIGSERIAL,
  project_id BIGINT NOT NULL,
  report_revision BIGINT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  attempts INT NOT NULL DEFAULT 0,
  available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  locked_at TIMESTAMPTZ,
  lease_expires_at TIMESTAMPTZ,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_selection_evaluation_task_id_uidx PRIMARY KEY (id),
  CONSTRAINT project_selection_evaluation_task_project_fk FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  CONSTRAINT project_selection_evaluation_task_project_report_fk FOREIGN KEY (project_id, report_revision) REFERENCES project_report_revision(project_id, revision) ON DELETE CASCADE,
  CONSTRAINT project_selection_evaluation_task_project_report_uidx UNIQUE (project_id, report_revision),
  CONSTRAINT project_selection_evaluation_task_report_revision_check CHECK (report_revision > 0),
  CONSTRAINT project_selection_evaluation_task_status_check CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
  CONSTRAINT project_selection_evaluation_task_attempts_check CHECK (attempts BETWEEN 0 AND 5)
);

CREATE INDEX project_selection_evaluation_task_status_available_at_idx
  ON project_selection_evaluation_task (status, available_at, project_id);

ALTER TABLE project_research_state
  ADD CONSTRAINT project_research_state_current_report_fk
  FOREIGN KEY (project_id, current_report_revision)
  REFERENCES project_report_revision(project_id, revision);

ALTER TABLE project_research_state
  ADD CONSTRAINT project_research_state_current_selection_fk
  FOREIGN KEY (current_selection_id)
  REFERENCES project_selection(id);

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
ALTER TABLE IF EXISTS project_research_state DROP CONSTRAINT IF EXISTS project_research_state_current_selection_fk;
ALTER TABLE IF EXISTS project_research_state DROP CONSTRAINT IF EXISTS project_research_state_current_report_fk;
DROP TABLE IF EXISTS project_selection_evaluation_task;
DROP TABLE IF EXISTS project_selection;
DROP TABLE IF EXISTS project_report_build_task;
DROP TABLE IF EXISTS project_report_revision;
DROP TABLE IF EXISTS project_observation_current;
DROP TABLE IF EXISTS project_observation;
DROP TABLE IF EXISTS project_data_collection_task;
DROP TABLE IF EXISTS project_data_collection_schedule;
DROP TABLE IF EXISTS project_research_state;
DROP TABLE IF EXISTS project_initial_recipient;
DROP TABLE IF EXISTS project_related_wallet;
DROP TRIGGER IF EXISTS project_contract_code_deployment_count_trigger ON project;
DROP TABLE IF EXISTS project;
DROP FUNCTION IF EXISTS update_contract_code_deployment_count();
DROP TABLE IF EXISTS contract_code;
DROP TABLE IF EXISTS project_candidate;
DROP TABLE IF EXISTS chain_ingest_checkpoint;
DROP TABLE IF EXISTS chain;
