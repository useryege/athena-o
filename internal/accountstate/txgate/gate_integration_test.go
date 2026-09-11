//go:build integration

package txgate_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	accountstatemigrations "github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestWithAccountTxSerializesOneAccountAndAllowsDifferentAccounts(t *testing.T) {
	db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	otherPool, err := pgxpool.New(ctx, db.DSN)
	if err != nil {
		t.Fatalf("open second pool: %v", err)
	}
	t.Cleanup(otherPool.Close)

	enteredFirst := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan error, 1)
	var releaseFirstOnce sync.Once
	var transactions sync.WaitGroup
	defer func() {
		releaseFirstOnce.Do(func() { close(releaseFirst) })
		cancel()
		transactions.Wait()
	}()
	transactions.Add(1)
	go func() {
		defer transactions.Done()
		firstDone <- txgate.WithAccountTx(ctx, db.Pool, "5b0ddd23-bfc2-4c86-af84-28a984188f5c", func(pgx.Tx) error {
			close(enteredFirst)
			select {
			case <-releaseFirst:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	waitForSignal(t, ctx, enteredFirst, "first account transaction to enter")

	enteredSameOwner := make(chan struct{})
	sameDone := make(chan error, 1)
	transactions.Add(1)
	go func() {
		defer transactions.Done()
		sameDone <- txgate.WithAccountTx(ctx, otherPool, "5b0ddd23-bfc2-4c86-af84-28a984188f5c", func(pgx.Tx) error {
			close(enteredSameOwner)
			return nil
		})
	}()

	blockedContext, cancelBlocked := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancelBlocked()
	select {
	case <-enteredSameOwner:
		t.Fatal("same account transaction entered before first transaction released its lock")
	case <-blockedContext.Done():
		if blockedContext.Err() != context.DeadlineExceeded {
			t.Fatalf("wait for same account lock: %v", blockedContext.Err())
		}
	}

	differentDone := make(chan error, 1)
	transactions.Add(1)
	go func() {
		defer transactions.Done()
		differentDone <- txgate.WithAccountTx(ctx, otherPool, "47f3cbb5-8388-49c4-9c67-13bd2e36583b", func(pgx.Tx) error {
			return nil
		})
	}()
	select {
	case err := <-differentDone:
		if err != nil {
			t.Fatalf("different account transaction: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("different account transaction was blocked: %v", ctx.Err())
	}

	releaseFirstOnce.Do(func() { close(releaseFirst) })
	for _, result := range []<-chan error{firstDone, sameDone} {
		if err := waitForResult(ctx, result); err != nil {
			t.Fatal(fmt.Errorf("account transaction: %w", err))
		}
	}
}

func waitForSignal(t *testing.T, ctx context.Context, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-ctx.Done():
		t.Fatalf("wait for %s: %v", description, ctx.Err())
	}
}

func waitForResult(ctx context.Context, result <-chan error) error {
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
