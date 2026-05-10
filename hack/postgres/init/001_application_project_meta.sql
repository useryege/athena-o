CREATE TABLE IF NOT EXISTS project (
  id BIGSERIAL PRIMARY KEY,
  project_id UUID NOT NULL,
  block_number BIGINT NOT NULL,
  block_time BIGINT NOT NULL,
  contract BYTEA NOT NULL,
  creator BYTEA NOT NULL,
  tx_hash BYTEA NOT NULL,
  tx_index BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_contract_len CHECK (length(contract) = 20),
  CONSTRAINT project_creator_len CHECK (length(creator) = 20),
  CONSTRAINT project_tx_hash_len CHECK (length(tx_hash) = 32)
);

CREATE UNIQUE INDEX IF NOT EXISTS project_project_id_idx
  ON project (project_id);

CREATE UNIQUE INDEX IF NOT EXISTS project_contract_idx
  ON project (contract);

CREATE UNIQUE INDEX IF NOT EXISTS project_tx_hash_idx
  ON project (tx_hash);

CREATE INDEX IF NOT EXISTS project_block_order_idx
  ON project (block_number, tx_index, id);
