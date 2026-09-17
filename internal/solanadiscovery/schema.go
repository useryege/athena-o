package solanadiscovery

import (
	"context"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/serviceschema"
)

func Schema() serviceschema.Spec {
	return serviceschema.Spec{Module: "solana-discovery", DSN: func() string {
		if dsn := strings.TrimSpace(os.Getenv("ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN")); dsn != "" {
			return dsn
		}
		return strings.TrimSpace(os.Getenv("ATHENA_ACCOUNT_STATE_POSTGRES_DSN"))
	}, CustomUp: func(ctx context.Context, pool *pgxpool.Pool) error { return NewStore(pool).Migrate(ctx) }, Relations: map[string][]string{
		"solana_discovery.scan_state": {"id", "start_slot", "last_processed_slot", "latest_finalized_slot", "last_success_at", "last_error"},
		"solana_discovery.projects":   {"mint", "token_program", "signature", "fee_payer", "mint_authority", "freeze_authority", "decimals", "slot", "block_time", "discovered_at", "name", "symbol", "metadata_status", "metadata_source", "metadata_account", "metadata_observed_slot", "metadata_updated_at", "metadata_next_at", "issuance_source", "issuance_program", "source_status", "source_next_at"},
	}}
}
func (s *Store) Verify(ctx context.Context) error { return Schema().Verify(ctx, s.pool) }
