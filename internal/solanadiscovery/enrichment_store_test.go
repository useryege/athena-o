package solanadiscovery

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func seedEnrichment(t *testing.T, store *Store) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, store.Initialize(ctx, 100))
	require.NoError(t, store.CommitRange(ctx, 99, 100, []Project{{Mint: usdcMint, TokenProgram: TokenProgram, Signature: "same-signature", FeePayer: wrappedSolMint, MintAuthority: wrappedSolMint, Slot: 100}, {Mint: wrappedSolMint, TokenProgram: Token2022Program, Signature: "same-signature", FeePayer: usdcMint, MintAuthority: usdcMint, Slot: 100}}))
}

// Resetting migrations/queue state or storing failure blanks must break this test.
func TestMetadataStoreRestartPartialPreservationAndQuery(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	seedEnrichment(t, store)
	now := time.Now().UTC().Add(time.Minute)
	require.NoError(t, store.Migrate(ctx))
	restarted := NewStore(store.pool)
	pending, err := restarted.PendingEnrichments(ctx, now, 10)
	require.NoError(t, err)
	require.Len(t, pending, 2)
	require.True(t, pending[0].MetadataDue)
	require.True(t, pending[0].SourceDue)
	observed := now.Add(-time.Second)
	require.NoError(t, store.WriteMetadata(ctx, usdcMint, MetadataResult{Name: "Dollar_%", Symbol: "USD", Status: "ready", Source: "metaplex", Account: "5x38Kp4hvdomTCnCrAny4UtMUt5rQBdB6px2K1Ui45Wq", ObservedSlot: 120}, observed, time.Time{}))
	require.NoError(t, store.WriteSource(ctx, usdcMint, SourceResult{Source: "pump_fun", Program: "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P", Status: "identified"}, time.Time{}))
	require.NoError(t, store.WriteMetadata(ctx, usdcMint, MetadataResult{Status: "error"}, now, now.Add(5*time.Minute)))
	require.NoError(t, store.WriteSource(ctx, usdcMint, SourceResult{Status: "error"}, now.Add(5*time.Minute)))
	items, total, err := store.ListProjects(ctx, 1, 20, "dollar_%")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "Dollar_%", items[0].Name)
	require.Equal(t, "USD", items[0].Symbol)
	require.Equal(t, "metaplex", items[0].MetadataSource)
	require.Equal(t, uint64(120), items[0].MetadataObservedSlot)
	require.WithinDuration(t, observed, items[0].MetadataUpdatedAt, time.Microsecond)
	require.Equal(t, "pump_fun", items[0].IssuanceSource)
	_, total, err = store.ListProjects(ctx, 1, 20, "%")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	_, total, err = store.ListProjects(ctx, 1, 20, "epjfw")
	require.NoError(t, err)
	require.Zero(t, total)
	pending, err = restarted.PendingEnrichments(ctx, now, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, wrappedSolMint, pending[0].Project.Mint)
	pending, err = restarted.PendingEnrichments(ctx, now.Add(6*time.Minute), 10)
	require.NoError(t, err)
	require.Len(t, pending, 2)
	// A later partial successful read must not clear an earlier verified field.
	require.NoError(t, store.WriteMetadata(ctx, usdcMint, MetadataResult{Symbol: "NEW", Status: "ready", Source: "metaplex", ObservedSlot: 121}, now, time.Time{}))
	items, _, err = store.ListProjects(ctx, 1, 20, "Dollar_%")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Dollar_%", items[0].Name)
	require.Equal(t, "NEW", items[0].Symbol)
	status, err := store.GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), status.LastProcessedSlot)
	require.Empty(t, status.LastError)
}
func TestSourceStoreCompletedSideNotQueuedAgain(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	seedEnrichment(t, store)
	now := time.Now().Add(time.Minute)
	require.NoError(t, store.WriteSource(ctx, usdcMint, SourceResult{Source: "unknown", Status: "unrecognized"}, time.Time{}))
	pending, err := store.PendingEnrichments(ctx, now, 10)
	require.NoError(t, err)
	for _, w := range pending {
		if w.Project.Mint == usdcMint {
			require.True(t, w.MetadataDue)
			require.False(t, w.SourceDue)
		}
	}
}

func TestMetadataMigrationPreservesLegacyRowsAndCursor(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	_, err := store.pool.Exec(ctx, "DROP SCHEMA solana_discovery CASCADE")
	require.NoError(t, err)
	old, err := migrationFiles.ReadFile("migrations/001_init.sql")
	require.NoError(t, err)
	_, err = store.pool.Exec(ctx, string(old))
	require.NoError(t, err)
	require.NoError(t, store.Initialize(ctx, 100))
	_, err = store.pool.Exec(ctx, `INSERT INTO solana_discovery.projects(mint,token_program,signature,fee_payer,mint_authority,decimals,slot) VALUES($1,$2,'legacy',$1,$1,6,100)`, usdcMint, TokenProgram)
	require.NoError(t, err)
	require.NoError(t, store.Migrate(ctx))
	require.NoError(t, store.Migrate(ctx))
	items, total, err := store.ListProjects(ctx, 1, 10, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "pending", items[0].MetadataStatus)
	require.Equal(t, "pending", items[0].SourceStatus)
	status, err := store.GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(99), status.LastProcessedSlot)
}
