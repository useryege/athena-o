CREATE SCHEMA IF NOT EXISTS solana_discovery;

CREATE TABLE IF NOT EXISTS solana_discovery.scan_state (
    id smallint PRIMARY KEY CHECK (id = 1),
    start_slot bigint CHECK (start_slot > 0),
    last_processed_slot bigint CHECK (last_processed_slot >= 0),
    latest_finalized_slot bigint NOT NULL DEFAULT 0 CHECK (latest_finalized_slot >= 0),
    last_success_at timestamptz,
    last_error text NOT NULL DEFAULT '',
    CHECK ((start_slot IS NULL) = (last_processed_slot IS NULL))
);

CREATE TABLE IF NOT EXISTS solana_discovery.projects (
    mint text PRIMARY KEY CHECK (length(mint) BETWEEN 32 AND 44),
    token_program text NOT NULL CHECK (token_program IN (
      'TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA',
      'TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb'
    )),
    signature text NOT NULL CHECK (signature <> ''),
    fee_payer text NOT NULL CHECK (fee_payer <> ''),
    mint_authority text NOT NULL CHECK (mint_authority <> ''),
    freeze_authority text NOT NULL DEFAULT '',
    decimals integer NOT NULL CHECK (decimals BETWEEN 0 AND 255),
    slot bigint NOT NULL CHECK (slot >= 0),
    block_time bigint NOT NULL DEFAULT 0,
    discovered_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS projects_recent_idx
    ON solana_discovery.projects (slot DESC, mint ASC);
