\connect application

CREATE TABLE IF NOT EXISTS project (
  id BIGSERIAL PRIMARY KEY,
  block_number BIGINT NOT NULL,
  block_time BIGINT NOT NULL,
  contract BYTEA NOT NULL,
  creator BYTEA NOT NULL,
  weth_pair BYTEA NOT NULL,
  usdt_pair BYTEA NOT NULL,
  fetch_at TIMESTAMPTZ NOT NULL,
  tx_hash BYTEA NOT NULL,
  tx_index BIGINT NOT NULL,
  source_code TEXT,
  source_code_hash BYTEA,
  source_code_fetched_at TIMESTAMPTZ,
  code_bin_hash BYTEA,
  code_bin_hash_fetched_at TIMESTAMPTZ,
  source_quality_report TEXT,
  source_quality_report_fetched_at TIMESTAMPTZ,
  creator_result_can_mint_from_dead_via_transfer_from BOOLEAN NOT NULL DEFAULT false,
  creator_result_can_mint_from_zero_via_transfer_from BOOLEAN NOT NULL DEFAULT false,
  creator_result_can_mint_from_weth_pair_via_transfer_from BOOLEAN NOT NULL DEFAULT false,
  creator_result_can_mint_from_usdt_pair_via_transfer_from BOOLEAN NOT NULL DEFAULT false,
  creator_result_can_mint_via_transfer_to_weth_pair BOOLEAN NOT NULL DEFAULT false,
  creator_result_can_mint_via_transfer_to_usdt_pair BOOLEAN NOT NULL DEFAULT false,
  report_is_policy_evaluated BOOLEAN NOT NULL DEFAULT false,
  report_is_blacklisted_creator_wallet BOOLEAN NOT NULL DEFAULT false,
  report_is_blacklisted_genesis_wallet BOOLEAN NOT NULL DEFAULT false,
  report_is_blacklisted_bytecode BOOLEAN NOT NULL DEFAULT false,
  report_is_blacklisted_source_code BOOLEAN NOT NULL DEFAULT false,
  report_has_mint_risk BOOLEAN NOT NULL DEFAULT false,
  genesis_wallets_fetched_at TIMESTAMPTZ,
  creator_historical_projects_fetched_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_contract_len CHECK (length(contract) = 20),
  CONSTRAINT project_creator_len CHECK (length(creator) = 20),
  CONSTRAINT project_weth_pair_len CHECK (length(weth_pair) = 20),
  CONSTRAINT project_usdt_pair_len CHECK (length(usdt_pair) = 20),
  CONSTRAINT project_tx_hash_len CHECK (length(tx_hash) = 32),
  CONSTRAINT project_source_code_hash_len CHECK (source_code_hash IS NULL OR length(source_code_hash) = 32),
  CONSTRAINT project_code_bin_hash_len CHECK (code_bin_hash IS NULL OR length(code_bin_hash) = 32)
);

CREATE UNIQUE INDEX IF NOT EXISTS project_contract_idx
  ON project (contract);

CREATE UNIQUE INDEX IF NOT EXISTS project_tx_hash_idx
  ON project (tx_hash);

CREATE INDEX IF NOT EXISTS project_weth_pair_idx
  ON project (weth_pair);

CREATE INDEX IF NOT EXISTS project_usdt_pair_idx
  ON project (usdt_pair);

CREATE INDEX IF NOT EXISTS project_block_order_idx
  ON project (block_number, tx_index, id);

CREATE INDEX IF NOT EXISTS project_creator_order_idx
  ON project (creator, block_number, tx_index, id);

CREATE TABLE IF NOT EXISTS project_ave_token_detail (
  project_contract BYTEA PRIMARY KEY,
  status INT NOT NULL,
  msg TEXT,
  data_type INT NOT NULL,
  is_audited BOOLEAN NOT NULL DEFAULT false,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  total TEXT,
  launch_price TEXT,
  current_price_eth TEXT,
  current_price_usd TEXT,
  price_change_1d TEXT,
  price_change_24h TEXT,
  price_change_1h TEXT,
  lock_amount TEXT,
  burn_amount TEXT,
  other_amount TEXT,
  tx_amount_24h TEXT,
  tx_volume_u_24h TEXT,
  locked_percent TEXT,
  market_cap TEXT,
  fdv TEXT,
  tvl TEXT,
  main_pair_tvl TEXT,
  token_price_change_5m TEXT,
  token_price_change_1h TEXT,
  token_price_change_4h TEXT,
  token_price_change_24h TEXT,
  token_tx_volume_usd_5m TEXT,
  token_tx_volume_usd_1h TEXT,
  token_tx_volume_usd_4h TEXT,
  token_tx_volume_usd_24h TEXT,
  token_buy_volume_u_5m TEXT,
  token_sell_volume_u_5m TEXT,
  token TEXT,
  chain TEXT,
  decimal INT NOT NULL DEFAULT 0,
  name TEXT,
  symbol TEXT,
  holders INT NOT NULL DEFAULT 0,
  appendix TEXT,
  risk_level INT NOT NULL DEFAULT 0,
  logo_url TEXT,
  risk_info TEXT,
  risk_score TEXT,
  launch_at BIGINT NOT NULL DEFAULT 0,
  created_at BIGINT NOT NULL DEFAULT 0,
  tx_count_24h INT NOT NULL DEFAULT 0,
  lock_platform TEXT,
  is_mintable TEXT,
  updated_at BIGINT NOT NULL DEFAULT 0,
  main_pair TEXT,
  has_mint_method BOOLEAN NOT NULL DEFAULT false,
  is_lp_not_locked BOOLEAN NOT NULL DEFAULT false,
  has_not_renounced BOOLEAN NOT NULL DEFAULT false,
  has_not_audited BOOLEAN NOT NULL DEFAULT false,
  has_not_open_source BOOLEAN NOT NULL DEFAULT false,
  is_in_blacklist BOOLEAN NOT NULL DEFAULT false,
  is_honeypot BOOLEAN NOT NULL DEFAULT false,
  ave_risk_level INT NOT NULL DEFAULT 0,
  CONSTRAINT project_ave_token_detail_project_contract_len CHECK (length(project_contract) = 20),
  CONSTRAINT project_ave_token_detail_project_fk FOREIGN KEY (project_contract) REFERENCES project(contract)
);

CREATE TABLE IF NOT EXISTS project_ave_pair (
  id BIGSERIAL PRIMARY KEY,
  project_contract BYTEA NOT NULL,
  rank_index INT NOT NULL,
  reserve0 TEXT,
  reserve1 TEXT,
  token0_price_eth TEXT,
  token0_price_usd TEXT,
  token1_price_eth TEXT,
  token1_price_usd TEXT,
  price_change TEXT,
  price_change_24h TEXT,
  price_change_1h TEXT,
  volume_u TEXT,
  low_u TEXT,
  high_u TEXT,
  fee TEXT,
  total_supply TEXT,
  tx_amount TEXT,
  pair TEXT,
  chain TEXT,
  amm TEXT,
  token0_address TEXT,
  token0_symbol TEXT,
  token0_decimal INT NOT NULL DEFAULT 0,
  token1_address TEXT,
  token1_symbol TEXT,
  token1_decimal INT NOT NULL DEFAULT 0,
  target_token TEXT,
  price_change_1d TEXT,
  created_at BIGINT NOT NULL DEFAULT 0,
  tx_count INT NOT NULL DEFAULT 0,
  updated_at BIGINT NOT NULL DEFAULT 0,
  market_cap TEXT,
  fdv TEXT,
  is_fake BOOLEAN NOT NULL DEFAULT false,
  persisted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_ave_pair_project_contract_len CHECK (length(project_contract) = 20),
  CONSTRAINT project_ave_pair_rank_index_nonnegative CHECK (rank_index >= 0),
  CONSTRAINT project_ave_pair_project_fk FOREIGN KEY (project_contract) REFERENCES project(contract),
  CONSTRAINT project_ave_pair_project_rank_uidx UNIQUE (project_contract, rank_index)
);

CREATE INDEX IF NOT EXISTS project_ave_pair_project_rank_idx
  ON project_ave_pair (project_contract, rank_index);

CREATE TABLE IF NOT EXISTS project_creator_historical_project (
  id BIGSERIAL PRIMARY KEY,
  project_contract BYTEA NOT NULL,
  historical_project_contract BYTEA NOT NULL,
  rank_index INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_creator_historical_project_project_contract_len CHECK (length(project_contract) = 20),
  CONSTRAINT project_creator_historical_project_historical_project_contract_len CHECK (length(historical_project_contract) = 20),
  CONSTRAINT project_creator_historical_project_rank_index_nonnegative CHECK (rank_index >= 0),
  CONSTRAINT project_creator_historical_project_project_fk FOREIGN KEY (project_contract) REFERENCES project(contract),
  CONSTRAINT project_creator_historical_project_uidx UNIQUE (project_contract, historical_project_contract)
);

CREATE INDEX IF NOT EXISTS project_creator_historical_project_rank_idx
  ON project_creator_historical_project (project_contract, rank_index);

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
