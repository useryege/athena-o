//go:build integration

package store

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/notification/delivery"
	"testing"
	"time"
)

func TestAuthorizeTxSharesFixedSessionCommitAndStartedDoesNotRelockAccount(t *testing.T) {
	s, ref := attemptFixture(t, "account")
	ctx := context.Background()
	candidate := testPermitCandidate(ref)
	owner, e := s.queries.GetAccountDeliveryOwner(ctx, ref.ID)
	if e != nil {
		t.Fatal(e)
	}
	gate, e := txgate.AcquireAccountSession(ctx, s.pool.(*pgxpool.Pool), uuidString(owner))
	if e != nil {
		t.Fatal(e)
	}
	defer gate.Release(ctx)
	tx, e := gate.Conn.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	guardCalled := false
	p, e := s.AuthorizeTx(ctx, tx, candidate, uuid.New(), func() error { guardCalled = true; return nil })
	if e != nil {
		t.Fatal(e)
	}
	if !guardCalled || p.AttemptID == uuid.Nil {
		_ = tx.Rollback(ctx)
		t.Fatal("same-connection authorization did not create guarded permit")
	}
	if e = tx.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	if state := deliveryState(t, s, ref); state != "pending" {
		t.Fatal("AuthorizeTx committed outer transaction", state)
	}
	tx, e = gate.Conn.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	p, e = s.AuthorizeTx(ctx, tx, candidate, uuid.New(), nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	short, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()
	if e = s.RecordStarted(short, p, time.Now()); e != nil {
		t.Fatal("started evidence recursively requested account gate", e)
	}
	if e = gate.Release(ctx); e != nil {
		t.Fatal(e)
	}
	if e = s.RecordOutcome(ctx, p, delivery.Outcome{Kind: "sent", MessageID: "1"}, time.Now()); e != nil {
		t.Fatal(e)
	}
}
