package solanadiscovery

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("SOLANA_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set SOLANA_TEST_POSTGRES_DSN for PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, dsn)
	require.NoError(t, err)
	name := fmt.Sprintf("solana_discovery_test_%d", time.Now().UnixNano())
	_, err = admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	require.NoError(t, err)
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	parsed.Path = "/" + name
	pool, err := pgxpool.New(ctx, parsed.String())
	require.NoError(t, err)
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
		_ = admin.Close(context.Background())
	})
	store := NewStore(pool)
	require.NoError(t, store.Migrate(ctx))
	return store
}

// A partial write or resetting the start on restart must break this test.
func TestStorePreservesStartAndCommitsCandidatesWithCheckpoint(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	require.NoError(t, store.Initialize(ctx, 100))
	require.NoError(t, store.Initialize(ctx, 200))
	status, err := store.GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), status.StartSlot)
	require.Equal(t, uint64(99), status.LastProcessedSlot)

	project := Project{Mint: usdcMint, TokenProgram: legacyTokenProgram, Signature: "sig-first", FeePayer: wrappedSolMint, MintAuthority: wrappedSolMint, Decimals: 6, Slot: 100, BlockTime: 1720000000}
	require.NoError(t, store.CommitRange(ctx, 99, 100, []Project{project}))
	// A repeated Mint keeps the first successful initialization evidence.
	project.Signature = "sig-later"
	project.Slot = 101
	require.NoError(t, store.CommitRange(ctx, 100, 101, []Project{project}))
	items, total, err := store.ListProjects(ctx, 1, 25, "EPj")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "sig-first", items[0].Signature)
	require.False(t, items[0].DiscoveredAt.IsZero())
	status, err = store.GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(101), status.LastProcessedSlot)
	require.Equal(t, int64(1), status.TotalProjects)
	require.False(t, status.LastSuccessAt.IsZero())
}

// A stale scanner and an invalid row cannot advance the checkpoint.
func TestStoreRejectsStaleOrFailedRangeAtomically(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	require.NoError(t, store.Initialize(ctx, 30))
	require.ErrorIs(t, store.CommitRange(ctx, 28, 30, nil), ErrCheckpointConflict)
	good := Project{Mint: usdcMint, TokenProgram: legacyTokenProgram, Signature: "sig-good", FeePayer: wrappedSolMint, MintAuthority: wrappedSolMint, Slot: 30}
	bad := Project{Mint: wrappedSolMint, TokenProgram: "wrong-token-program", Signature: "sig-bad", FeePayer: wrappedSolMint, MintAuthority: wrappedSolMint, Slot: 30}
	require.Error(t, store.CommitRange(ctx, 29, 30, []Project{good, bad}))
	status, err := store.GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(29), status.LastProcessedSlot)
	require.Equal(t, int64(0), status.TotalProjects)
}

func TestStoreListsNewestFirstWithMintSearchAndPagination(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	require.NoError(t, store.Initialize(ctx, 100))
	projects := []Project{
		{Mint: usdcMint, TokenProgram: legacyTokenProgram, Signature: "sig-100", FeePayer: wrappedSolMint, MintAuthority: wrappedSolMint, Slot: 100},
		{Mint: wrappedSolMint, TokenProgram: token2022Program, Signature: "sig-101", FeePayer: usdcMint, MintAuthority: usdcMint, Slot: 101},
	}
	require.NoError(t, store.CommitRange(ctx, 99, 101, projects))
	first, total, err := store.ListProjects(ctx, 1, 1, "")
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Equal(t, wrappedSolMint, first[0].Mint)
	second, total, err := store.ListProjects(ctx, 2, 1, "")
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Equal(t, usdcMint, second[0].Mint)
	filtered, total, err := store.ListProjects(ctx, 1, 25, "PjFW")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, usdcMint, filtered[0].Mint)
}

func TestStorePersistsNodeErrorBeforeStartWithoutInventingStartSlot(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	require.NoError(t, store.RecordError(ctx, "node unavailable"))
	status, err := NewStore(store.pool).GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, "error", status.Status)
	require.Zero(t, status.StartSlot)
	require.Equal(t, "node unavailable", status.LastError)
	require.NoError(t, store.Initialize(ctx, 100))
	require.NoError(t, store.SetLatestFinalized(ctx, 100))
	require.NoError(t, store.CommitRange(ctx, 99, 100, nil))
	status, err = store.GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), status.StartSlot)
	require.Empty(t, status.LastError)
}

func TestStoreRetainsErrorUntilRangeSucceeds(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	require.NoError(t, store.Initialize(ctx, 10))
	require.NoError(t, store.RecordError(ctx, "block unavailable"))
	require.NoError(t, store.SetLatestFinalized(ctx, 12))
	status, err := store.GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, "error", status.Status)
	require.Equal(t, "block unavailable", status.LastError)
	require.NoError(t, store.CommitRange(ctx, 9, 12, nil))
	status, err = store.GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, "current", status.Status)
	require.Empty(t, status.LastError)
}

func TestStoreConcurrentRangeCommitsHaveSingleWinner(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	require.NoError(t, store.Initialize(ctx, 100))
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, mint := range []string{usdcMint, wrappedSolMint} {
		go func(mint string) {
			<-start
			results <- store.CommitRange(ctx, 99, 100, []Project{{Mint: mint, TokenProgram: TokenProgram, Signature: "sig", FeePayer: usdcMint, MintAuthority: usdcMint, Slot: 100}})
		}(mint)
	}
	close(start)
	first, second := <-results, <-results
	if first == nil {
		require.ErrorIs(t, second, ErrCheckpointConflict)
	} else {
		require.ErrorIs(t, first, ErrCheckpointConflict)
		require.NoError(t, second)
	}
	items, total, err := store.ListProjects(ctx, 1, 25, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	status, err := store.GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), status.LastProcessedSlot)
}

func TestStoreClearsErrorWhenCaughtUpPollRecovers(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	require.NoError(t, store.Initialize(ctx, 10))
	require.NoError(t, store.CommitRange(ctx, 9, 10, nil))
	require.NoError(t, store.RecordError(ctx, "getSlot unavailable"))
	require.NoError(t, store.SetLatestFinalized(ctx, 10))
	status, err := store.GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, "current", status.Status)
	require.Empty(t, status.LastError)
}
