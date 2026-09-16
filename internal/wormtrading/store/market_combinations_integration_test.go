//go:build integration

package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestMarketCombinationCreateUpdateAndOrderedItems(t *testing.T) {
	db := pgtest.New(t, Migrations(), "migrations")
	store := NewSQLStore(db.Pool)
	owner := uuid.NewString()
	first, second := marketCombinationTestItem(1), marketCombinationTestItem(2)

	created, err := store.CreateMarketCombination(context.Background(), owner, "Combination", []MarketCombinationItemInput{second, first})
	require.NoError(t, err)
	require.Equal(t, int64(1), created.Revision)
	require.Equal(t, []string{second.MarketConditionID, first.MarketConditionID}, []string{created.Items[0].MarketConditionID, created.Items[1].MarketConditionID})
	require.Equal(t, []int32{1, 2}, []int32{created.Items[0].Ordinal, created.Items[1].Ordinal})

	updated, err := store.UpdateMarketCombination(context.Background(), owner, created.ID, "Updated", 1, []MarketCombinationItemInput{first, second})
	require.NoError(t, err)
	require.Equal(t, int64(2), updated.Revision)
	require.Equal(t, []string{first.MarketConditionID, second.MarketConditionID}, []string{updated.Items[0].MarketConditionID, updated.Items[1].MarketConditionID})

	conflict, err := store.UpdateMarketCombination(context.Background(), owner, created.ID, "Stale", 1, []MarketCombinationItemInput{first})
	require.Nil(t, conflict)
	require.ErrorIs(t, err, ErrMarketCombinationRevision)
}

func TestMarketCombinationActiveRunLockBlocksUpdateAndDelete(t *testing.T) {
	db := pgtest.New(t, Migrations(), "migrations")
	store := NewSQLStore(db.Pool)
	owner := uuid.NewString()
	created, err := store.CreateMarketCombination(context.Background(), owner, "Locked", []MarketCombinationItemInput{marketCombinationTestItem(3)})
	require.NoError(t, err)
	insertMarketCombinationRunLock(t, db, owner, created)

	updated, err := store.UpdateMarketCombination(context.Background(), owner, created.ID, "Blocked", created.Revision, []MarketCombinationItemInput{marketCombinationTestItem(4)})
	require.Nil(t, updated)
	require.ErrorIs(t, err, ErrMarketCombinationRevision)
	require.ErrorIs(t, store.DeleteMarketCombination(context.Background(), owner, created.ID, created.Revision), ErrMarketCombinationRevision)

	unchanged, err := store.GetMarketCombination(context.Background(), owner, created.ID)
	require.NoError(t, err)
	require.Equal(t, "Locked", unchanged.Name)
	require.Equal(t, int64(1), unchanged.Revision)
}

func TestMarketCombinationUpdateRechecksRevisionAfterLockWait(t *testing.T) {
	db := pgtest.New(t, Migrations(), "migrations")
	store := NewSQLStore(db.Pool)
	owner := uuid.NewString()
	created, err := store.CreateMarketCombination(context.Background(), owner, "Original", []MarketCombinationItemInput{marketCombinationTestItem(5)})
	require.NoError(t, err)

	blocker, err := db.Pool.Begin(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = blocker.Rollback(context.Background()) })
	var revision int64
	require.NoError(t, blocker.QueryRow(context.Background(), `
		SELECT revision FROM worm_market_combinations WHERE id = $1 FOR UPDATE`, created.ID).Scan(&revision))
	require.Equal(t, int64(1), revision)
	_, err = blocker.Exec(context.Background(), `
		UPDATE worm_market_combinations SET revision = revision + 1 WHERE id = $1`, created.ID)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		_, updateErr := store.UpdateMarketCombination(context.Background(), owner, created.ID, "Resolved snapshot", 1, []MarketCombinationItemInput{marketCombinationTestItem(6)})
		done <- updateErr
	}()
	waitForMarketCombinationLockWait(t, db)
	require.NoError(t, blocker.Commit(context.Background()))

	select {
	case err := <-done:
		require.ErrorIs(t, err, ErrMarketCombinationRevision)
	case <-time.After(5 * time.Second):
		t.Fatal("market combination update remained blocked after barrier release")
	}
	current, err := store.GetMarketCombination(context.Background(), owner, created.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), current.Revision)
	require.Equal(t, "Original", current.Name)
}

func marketCombinationTestItem(seed byte) MarketCombinationItemInput {
	var eventKey, marketKey solana.PublicKey
	eventKey[len(eventKey)-1] = seed
	marketKey[len(marketKey)-2] = seed
	marketKey[len(marketKey)-1] = seed + 1
	return MarketCombinationItemInput{
		EventConditionID: eventKey.String(), EventTitle: "Event", EventLogo: "https://worm.example/event.png",
		MarketConditionID: marketKey.String(), MarketTitle: "Market", MarketLogo: "https://worm.example/market.png",
		IsYes: seed%2 == 0, OutcomeLabel: "Outcome",
	}
}

func insertMarketCombinationRunLock(t *testing.T, db *pgtest.DB, owner string, combination *MarketCombination) {
	t.Helper()
	planID, runID := uuid.NewString(), uuid.NewString()
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO worm_execution_plans (
			id, owner_account_id, combination_id, combination_name, combination_revision,
			state, wallet_selection_revision, wallet_count, item_count, total_step_count, retention_until,
			requested_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, 'BUILDING', 1, 1, 1, 1, now() + interval '1 hour', now(), now(), now())`,
		planID, owner, combination.ID, combination.Name, combination.Revision)
	require.NoError(t, err)
	_, err = db.Pool.Exec(context.Background(), `
		INSERT INTO worm_execution_runs (
			id, owner_account_id, plan_id, plan_version, plan_digest_sha256,
			idempotency_key_sha256, request_sha256, combination_id, combination_name,
			combination_revision, state, wallet_count, item_count, total_step_count,
			actionable_step_count, requested_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, 1, decode(repeat('01', 32), 'hex'), decode(repeat('02', 32), 'hex'),
			decode(repeat('03', 32), 'hex'), $4, $5, $6, 'AWAITING_AUTHORIZATION',
			1, 1, 1, 1, now(), now(), now()
		)`, runID, owner, planID, combination.ID, combination.Name, combination.Revision)
	require.NoError(t, err)
	_, err = db.Pool.Exec(context.Background(), `
		INSERT INTO worm_execution_combination_locks (combination_id, run_id, combination_revision, acquired_at)
		VALUES ($1, $2, $3, now())`, combination.ID, runID, combination.Revision)
	require.NoError(t, err)
}

func waitForMarketCombinationLockWait(t *testing.T, db *pgtest.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		var waiting bool
		err := db.Pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE datname = current_database()
				  AND pid <> pg_backend_pid()
				  AND wait_event_type = 'Lock'
				  AND query ILIKE '%worm_market_combinations%FOR UPDATE%'
			)`).Scan(&waiting)
		if err == nil && waiting {
			return
		}
		if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) && !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("query blocked market combination update: %v", err)
		}
		if ctx.Err() != nil {
			t.Fatal("market combination update did not reach the database lock barrier")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
