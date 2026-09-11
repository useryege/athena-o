//go:build integration

package txgate_test

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"testing"
	"time"
)

func TestGateTimingSeparatesPoolAndCancelledAdvisoryWait(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	owner := "11111111-1111-4111-8111-111111111111"
	held, e := txgate.AcquireAccountSession(context.Background(), db.Pool, owner)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = held.Release(context.Background()) })
	for _, kind := range []string{"transaction", "session"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
			defer cancel()
			var observations []txgate.Timing
			ctx = txgate.WithTiming(ctx, func(v txgate.Timing) { observations = append(observations, v) })
			if kind == "transaction" {
				e = txgate.WithAccountTx(ctx, db.Pool, owner, func(pgx.Tx) error { t.Error("entered locked account"); return nil })
			} else {
				var gate *txgate.AccountSession
				gate, e = txgate.AcquireAccountSession(ctx, db.Pool, owner)
				if gate != nil {
					_ = gate.Release(context.Background())
					t.Error("entered locked account")
				}
			}
			if e == nil {
				t.Fatal("missing cancellation")
			}
			if len(observations) != 2 {
				t.Fatalf("missing acquisition phase evidence: %+v", observations)
			}
			if observations[0].Phase != "begin" || !observations[0].Succeeded || observations[1].Phase != "advisory" || observations[1].Succeeded || observations[1].Duration <= 0 {
				t.Fatal(observations)
			}
		})
	}
}
