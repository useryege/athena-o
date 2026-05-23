CREATE TABLE IF NOT EXISTS project (
  id BIGSERIAL PRIMARY KEY,
  block_number BIGINT NOT NULL,
  block_time BIGINT NOT NULL,
  contract BYTEA NOT NULL,
  creator BYTEA NOT NULL,
  tx_hash BYTEA NOT NULL,
  tx_index BIGINT NOT NULL,
  source_code TEXT,
  source_code_hash BYTEA,
  code_bin_hash BYTEA,
  source_quality_report TEXT,
  source_quality_reported_at TIMESTAMPTZ,
  is_archived BOOLEAN NOT NULL DEFAULT FALSE,
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_contract_len CHECK (length(contract) = 20),
  CONSTRAINT project_creator_len CHECK (length(creator) = 20),
  CONSTRAINT project_tx_hash_len CHECK (length(tx_hash) = 32),
  CONSTRAINT project_source_code_hash_len CHECK (source_code_hash IS NULL OR length(source_code_hash) = 32),
  CONSTRAINT project_code_bin_hash_len CHECK (code_bin_hash IS NULL OR length(code_bin_hash) = 32)
);

CREATE UNIQUE INDEX IF NOT EXISTS project_contract_idx
  ON project (contract);

CREATE UNIQUE INDEX IF NOT EXISTS project_tx_hash_idx
  ON project (tx_hash);

CREATE INDEX IF NOT EXISTS project_block_order_idx
  ON project (block_number, tx_index, id);

CREATE INDEX IF NOT EXISTS project_archived_time_idx
  ON project (is_archived, archived_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS project_creator_order_idx
  ON project (creator, block_number, tx_index, id);

CREATE TABLE IF NOT EXISTS bytecode_blacklist_contract (
  contract BYTEA PRIMARY KEY,
  code_hash BYTEA NOT NULL,
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT bytecode_blacklist_contract_len CHECK (length(contract) = 20),
  CONSTRAINT bytecode_blacklist_contract_code_hash_len CHECK (length(code_hash) = 32)
);

CREATE INDEX IF NOT EXISTS bytecode_blacklist_contract_code_hash_idx
  ON bytecode_blacklist_contract (code_hash);

CREATE TABLE IF NOT EXISTS sourcecode_blacklist_contract (
  contract BYTEA PRIMARY KEY,
  source_hash BYTEA NOT NULL,
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT sourcecode_blacklist_contract_contract_len CHECK (length(contract) = 20),
  CONSTRAINT sourcecode_blacklist_contract_source_hash_len CHECK (length(source_hash) = 32),
  CONSTRAINT sourcecode_blacklist_contract_source_hash_unique UNIQUE (source_hash)
);

CREATE INDEX IF NOT EXISTS sourcecode_blacklist_contract_source_hash_idx
  ON sourcecode_blacklist_contract (source_hash);

CREATE TABLE IF NOT EXISTS wallet_blacklist (
  wallet BYTEA PRIMARY KEY,
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT wallet_blacklist_wallet_len CHECK (length(wallet) = 20)
);

CREATE TABLE IF NOT EXISTS project_event_log (
  id BIGSERIAL PRIMARY KEY,
  contract BYTEA NOT NULL,
  event_type SMALLINT NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  message TEXT,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  idempotency_key TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_event_log_contract_len CHECK (length(contract) = 20),
  CONSTRAINT project_event_log_event_type_positive CHECK (event_type > 0),
  CONSTRAINT project_event_log_idempotency_key_not_empty CHECK (length(btrim(idempotency_key)) > 0),
  CONSTRAINT project_event_log_project_fk FOREIGN KEY (contract) REFERENCES project(contract)
);

CREATE UNIQUE INDEX IF NOT EXISTS project_event_log_contract_idempotency_key_uidx
  ON project_event_log (contract, idempotency_key);

CREATE INDEX IF NOT EXISTS project_event_log_contract_timeline_idx
  ON project_event_log (contract, occurred_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS project_event_log_event_type_time_idx
  ON project_event_log (event_type, occurred_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS project_genesis_wallet (
  id BIGSERIAL PRIMARY KEY,
  project_contract BYTEA NOT NULL,
  wallet BYTEA NOT NULL,
  net_amount NUMERIC(78,0) NOT NULL,
  ratio_bps BIGINT NOT NULL,
  rank_index INT NOT NULL,
  total_supply NUMERIC(78,0) NOT NULL,
  source_tx_hash BYTEA NOT NULL,
  source_block_number BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_genesis_wallet_project_contract_len CHECK (length(project_contract) = 20),
  CONSTRAINT project_genesis_wallet_wallet_len CHECK (length(wallet) = 20),
  CONSTRAINT project_genesis_wallet_source_tx_hash_len CHECK (length(source_tx_hash) = 32),
  CONSTRAINT project_genesis_wallet_net_amount_positive CHECK (net_amount > 0),
  CONSTRAINT project_genesis_wallet_ratio_bps_nonnegative CHECK (ratio_bps >= 0),
  CONSTRAINT project_genesis_wallet_project_fk FOREIGN KEY (project_contract) REFERENCES project(contract),
  CONSTRAINT project_genesis_wallet_project_wallet_uidx UNIQUE (project_contract, wallet)
);

CREATE INDEX IF NOT EXISTS project_genesis_wallet_project_rank_idx
  ON project_genesis_wallet (project_contract, rank_index);

CREATE INDEX IF NOT EXISTS project_genesis_wallet_wallet_ratio_idx
  ON project_genesis_wallet (wallet, ratio_bps DESC, project_contract);

CREATE TABLE IF NOT EXISTS project_comment (
  id BIGSERIAL PRIMARY KEY,
  project_contract BYTEA NOT NULL,
  username TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_comment_project_contract_len CHECK (length(project_contract) = 20),
  CONSTRAINT project_comment_username_not_empty CHECK (length(btrim(username)) > 0),
  CONSTRAINT project_comment_content_not_empty CHECK (length(btrim(content)) > 0),
  CONSTRAINT project_comment_content_max_len CHECK (char_length(content) <= 1000),
  CONSTRAINT project_comment_project_fk FOREIGN KEY (project_contract) REFERENCES project(contract)
);

CREATE INDEX IF NOT EXISTS project_comment_project_timeline_idx
  ON project_comment (project_contract, created_at DESC, id DESC);
