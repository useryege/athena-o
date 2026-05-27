\connect solidity

CREATE TABLE IF NOT EXISTS bytecode (
  code_hash BYTEA PRIMARY KEY,
  runtime_bytecode BYTEA NOT NULL,
  source_code TEXT,
  source_code_hash BYTEA,
  source_code_fetched_at TIMESTAMPTZ,
  source_code_origin TEXT,
  source_quality_report TEXT,
  source_quality_report_fetched_at TIMESTAMPTZ,
  source_quality_report_origin TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT bytecode_code_hash_len CHECK (length(code_hash) = 32),
  CONSTRAINT bytecode_runtime_bytecode_not_empty CHECK (length(runtime_bytecode) > 0),
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
