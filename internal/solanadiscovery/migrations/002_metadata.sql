ALTER TABLE solana_discovery.projects
    ADD COLUMN IF NOT EXISTS name text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS symbol text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS metadata_status text NOT NULL DEFAULT 'pending' CHECK (metadata_status IN ('pending','ready','unavailable','error')),
    ADD COLUMN IF NOT EXISTS metadata_source text NOT NULL DEFAULT '' CHECK (metadata_source IN ('','token2022_on_mint','metaplex')),
    ADD COLUMN IF NOT EXISTS metadata_account text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS metadata_observed_slot bigint NOT NULL DEFAULT 0 CHECK (metadata_observed_slot >= 0),
    ADD COLUMN IF NOT EXISTS metadata_updated_at timestamptz,
    ADD COLUMN IF NOT EXISTS metadata_next_at timestamptz DEFAULT now(),
    ADD COLUMN IF NOT EXISTS issuance_source text NOT NULL DEFAULT 'unknown' CHECK (issuance_source IN ('unknown','pump_fun','raydium_launchlab','direct_token')),
    ADD COLUMN IF NOT EXISTS issuance_program text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_status text NOT NULL DEFAULT 'pending' CHECK (source_status IN ('pending','identified','unrecognized','error')),
    ADD COLUMN IF NOT EXISTS source_next_at timestamptz DEFAULT now();

CREATE INDEX IF NOT EXISTS projects_metadata_pending_idx ON solana_discovery.projects (metadata_next_at, discovered_at, mint) WHERE metadata_next_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS projects_source_pending_idx ON solana_discovery.projects (source_next_at, discovered_at, mint) WHERE source_next_at IS NOT NULL;
