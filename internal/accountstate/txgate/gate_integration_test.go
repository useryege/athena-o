//go:build integration

package txgate_test

import (
	"context"
	"fmt"
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
	go func() {
		firstDone <- txgate.WithAccountTx(ctx, db.Pool, "5b0ddd23-bfc2-4c86-af84-28a984188f5c", func(pgx.Tx) error {
			close(enteredFirst)
			<-releaseFirst
			return nil
		})
	}()
	<-enteredFirst

	enteredSameOwner := make(chan struct{})
	sameDone := make(chan error, 1)
	go func() {
		sameDone <- txgate.WithAccountTx(ctx, otherPool, "5b0ddd23-bfc2-4c86-af84-28a984188f5c", func(pgx.Tx) error {
			close(enteredSameOwner)
			return nil
		})
	}()

	select {
	case <-enteredSameOwner:
		t.Fatal("same account transaction entered before first transaction released its lock")
	case <-time.After(200 * time.Millisecond):
	}

	differentDone := make(chan error, 1)
	go func() {
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

	close(releaseFirst)
	for _, result := range []<-chan error{firstDone, sameDone} {
		if err := <-result; err != nil {
			t.Fatal(fmt.Errorf("account transaction: %w", err))
		}
	}
}
