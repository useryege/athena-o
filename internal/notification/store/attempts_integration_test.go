//go:build integration

package store

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/notification/delivery"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"testing"
	"time"
)

func attemptFixture(t *testing.T, kind string) (*SQLStore, delivery.WorkRef) {
	t.Helper()
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	var id int64
	if kind == "system" {
		if _, err := db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','default',1)`); err != nil {
			t.Fatal(err)
		}
		err := db.Pool.QueryRow(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label) VALUES('test','info','hello','telegram','pending','test','default') RETURNING id`).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		owner := uuid.NewString()
		if _, err := db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision) VALUES($1,123,123,'test',1)`, owner); err != nil {
			t.Fatal(err)
		}
		err := db.Pool.QueryRow(ctx, `INSERT INTO account_notification_deliveries(account_id,idempotency_key,payload_digest,source,severity,body,channel,status,telegram_chat_id,binding_revision) VALUES($1,'key',decode(repeat('ab',32),'hex'),'test','info','hello','telegram','pending',123,1) RETURNING id`, owner).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
	}
	return NewSQLStore(db.Pool), delivery.WorkRef{Kind: kind, ID: id}
}
func deliveryState(t *testing.T, s *SQLStore, ref delivery.WorkRef) string {
	t.Helper()
	var state string
	if err := s.pool.QueryRow(context.Background(), `SELECT status FROM `+ref.Kind+`_notification_deliveries WHERE id=$1`, ref.ID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestAttemptPermitIsSingleUseAndOutcomeCAS(t *testing.T) {
	for _, kind := range []string{"account", "system"} {
		t.Run(kind, func(t *testing.T) {
			s, ref := attemptFixture(t, kind)
			ctx := context.Background()
			p, err := s.Authorize(ctx, ref, uuid.New())
			if err != nil {
				t.Fatal(err)
			}
			if p.AttemptID == uuid.Nil || len(p.PayloadDigest) != 32 || p.AuthorizedAt.IsZero() {
				t.Fatalf("invalid permit %#v", p)
			}
			if state := deliveryState(t, s, ref); state != "sending" {
				t.Fatalf("permit not persisted: %s", state)
			}
			// A crash immediately after authorization can never put this work back in the claim queue.
			if _, err = s.Authorize(ctx, ref, uuid.New()); !errors.Is(err, ErrDeliveryNotEligible) {
				t.Fatalf("second permit: %v", err)
			}
			stale := p
			stale.AttemptID = uuid.New()
			if err = s.RecordOutcome(ctx, stale, delivery.Outcome{Kind: "sent", MessageID: "stale"}, time.Now()); !errors.Is(err, ErrStalePermit) {
				t.Fatalf("stale outcome %v", err)
			}
			o := delivery.Outcome{Kind: "sent", MessageID: "77"}
			// The receipt is authoritative even when the asynchronous started evidence was not recorded.
			if err = s.RecordOutcome(ctx, p, o, time.Now()); err != nil {
				t.Fatal(err)
			}
			if err = s.RecordOutcome(ctx, p, o, time.Now()); err != nil {
				t.Fatalf("idempotent result retry %v", err)
			}
			if state := deliveryState(t, s, ref); state != "sent" {
				t.Fatal(state)
			}
			if err = s.RecordOutcome(ctx, p, delivery.Outcome{Kind: "retryable"}, time.Now()); !errors.Is(err, ErrStalePermit) {
				t.Fatalf("conflicting late outcome %v", err)
			}
			if err = s.RecordStarted(ctx, p, p.AuthorizedAt.Add(time.Millisecond)); err != nil {
				t.Fatalf("late start evidence: %v", err)
			}
		})
	}
}

func TestOutcomeFailedWriteRetriesOnlyResult(t *testing.T) {
	s, ref := attemptFixture(t, "system")
	ctx := context.Background()
	p, err := s.Authorize(ctx, ref, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	// Fail the real result UPDATE once, as a database outage would. The committed permit remains consumed.
	_, err = s.pool.Exec(ctx, `ALTER TABLE system_notification_deliveries ADD CONSTRAINT reject_sent CHECK(status <> 'sent')`)
	if err != nil {
		t.Fatal(err)
	}
	outcome := delivery.Outcome{Kind: "sent", MessageID: "88"}
	if err = s.RecordOutcome(ctx, p, outcome, time.Now()); err == nil {
		t.Fatal("expected result transaction failure")
	}
	if _, err = s.Authorize(ctx, ref, uuid.New()); !errors.Is(err, ErrDeliveryNotEligible) {
		t.Fatalf("failed write allowed resend: %v", err)
	}
	_, err = s.pool.Exec(ctx, `ALTER TABLE system_notification_deliveries DROP CONSTRAINT reject_sent`)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordOutcome(ctx, p, outcome, time.Now()); err != nil {
		t.Fatal(err)
	}
	if state := deliveryState(t, s, ref); state != "sent" {
		t.Fatal(state)
	}
}

func TestOutcomeRevocationCannotBeUndoneByRegrant(t *testing.T) {
	s, ref := attemptFixture(t, "account")
	ctx := context.Background()
	p, err := s.Authorize(ctx, ref, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DeleteTelegramBinding(ctx, p.OwnerID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision) VALUES($1,123,123,'test',2)`, p.OwnerID); err != nil {
		t.Fatal(err)
	}
	if state := deliveryState(t, s, ref); state != "sending" {
		t.Fatalf("permitted send prematurely cancelled: %s", state)
	}
	if err = s.RecordOutcome(ctx, p, delivery.Outcome{Kind: "retryable", RetryAfter: 7 * time.Second}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if state := deliveryState(t, s, ref); state != "cancelled" {
		t.Fatal(state)
	}
	if _, err = s.Authorize(ctx, ref, uuid.New()); !errors.Is(err, ErrDeliveryNotEligible) {
		t.Fatalf("revived: %v", err)
	}
}

func TestOutcomeUnknownRemainsTerminal(t *testing.T) {
	s, ref := attemptFixture(t, "account")
	ctx := context.Background()
	p, err := s.Authorize(ctx, ref, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordOutcome(ctx, p, delivery.Outcome{Kind: "unknown", Code: "response_lost"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Authorize(ctx, ref, uuid.New()); !errors.Is(err, ErrDeliveryNotEligible) {
		t.Fatalf("unknown revived: %v", err)
	}
	counts, err := s.GetAccountNotificationRuntimeCounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Unknown != 1 || counts.Pending != 0 || counts.Failed != 0 || counts.Sending != 0 {
		t.Fatalf("unknown miscounted: %#v", counts)
	}
}

func TestAttemptBindingGateSerializesDeleteAndAuthorize(t *testing.T) {
	s, ref := attemptFixture(t, "account")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	owner, err := s.queries.GetAccountDeliveryOwner(ctx, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('athena:account:' || $1::text,0))`, uuidString(owner)); err != nil {
		t.Fatal(err)
	}
	deleted := make(chan error, 1)
	go func() { _, err := s.DeleteTelegramBinding(ctx, uuidString(owner)); deleted <- err }()
	waitForAccountGateWaiters(t, ctx, s, 1)
	authorized := make(chan error, 1)
	go func() { _, err := s.Authorize(ctx, ref, uuid.New()); authorized <- err }()
	waitForAccountGateWaiters(t, ctx, s, 2)
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-deleted; err != nil {
		t.Fatal(err)
	}
	if err = <-authorized; !errors.Is(err, ErrDeliveryNotEligible) {
		t.Fatalf("authorization overtook binding deletion: %v", err)
	}
	if state := deliveryState(t, s, ref); state != "cancelled" {
		t.Fatal(state)
	}
}

func waitForAccountGateWaiters(t *testing.T, ctx context.Context, s *SQLStore, want int) {
	t.Helper()
	for {
		var count int
		err := s.pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND wait_event='advisory'`).Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count >= want {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("expected %d blocked account transactions", want)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func TestOutcomeRetryDelaysAndFifthAttemptCeiling(t *testing.T) {
	s, ref := attemptFixture(t, "system")
	ctx := context.Background()
	for attempt := 1; attempt <= 5; attempt++ {
		p, err := s.Authorize(ctx, ref, uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		at := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
		o := delivery.Outcome{Kind: "retryable", RetryAfter: 7 * time.Second, Code: "provider_429"}
		if err = s.RecordOutcome(ctx, p, o, at); err != nil {
			t.Fatal(err)
		}
		var state string
		var attempts int
		var next time.Time
		if err = s.pool.QueryRow(ctx, `SELECT status,attempts,next_attempt_at FROM system_notification_deliveries WHERE id=$1`, ref.ID).Scan(&state, &attempts, &next); err != nil {
			t.Fatal(err)
		}
		if attempts != attempt {
			t.Fatalf("attempts %d want %d", attempts, attempt)
		}
		if attempt < 5 {
			wait := 7 * time.Second
			if attempt == 4 {
				wait = 8 * time.Second
			}
			if state != "pending" || !next.Equal(at.Add(wait)) {
				t.Fatalf("retry %d: %s next %v", attempt, state, next)
			}
		} else if state != "failed" {
			t.Fatalf("fifth result: %s", state)
		}
	}
	if _, err := s.Authorize(ctx, ref, uuid.New()); !errors.Is(err, ErrDeliveryNotEligible) {
		t.Fatalf("sixth send permitted: %v", err)
	}
}

func TestAttemptPermitFieldsCannotBeReused(t *testing.T) {
	s, ref := attemptFixture(t, "account")
	ctx := context.Background()
	p, err := s.Authorize(ctx, ref, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	mutations := []func(*delivery.Permit){func(p *delivery.Permit) { p.OwnerID = uuid.NewString() }, func(p *delivery.Permit) { p.SenderIncarnation = uuid.New() }, func(p *delivery.Permit) { p.PayloadDigest = make([]byte, 32) }, func(p *delivery.Permit) { p.Work.Kind = "system" }, func(p *delivery.Permit) { p.AuthorizedAt = p.AuthorizedAt.Add(time.Second) }}
	for _, mutate := range mutations {
		bad := p
		mutate(&bad)
		if err = s.RecordStarted(ctx, bad, time.Now()); !errors.Is(err, ErrStalePermit) {
			t.Fatalf("bad permit accepted: %#v %v", bad, err)
		}
	}
	if state := deliveryState(t, s, ref); state != "sending" {
		t.Fatal(state)
	}
}

func TestAttemptBindingGateSerializesRebindAndAuthorize(t *testing.T) {
	s, ref := attemptFixture(t, "account")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	owner, err := s.queries.GetAccountDeliveryOwner(ctx, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	digest := make([]byte, 32)
	digest[0] = 1
	if _, err = s.CreateTelegramBindingAttempt(ctx, CreateTelegramBindingAttemptRequest{ID: uuid.NewString(), AccountID: uuidString(owner), TokenDigest: digest, ExpiresAt: time.Now().Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('athena:account:' || $1::text,0))`, uuidString(owner)); err != nil {
		t.Fatal(err)
	}
	rebound := make(chan error, 1)
	go func() {
		_, err := s.CompleteTelegramBindingAttempt(ctx, CompleteTelegramBindingAttemptRequest{TokenDigest: digest, TelegramUserID: 456, TelegramChatID: 456, TelegramDisplayName: "rebound"})
		rebound <- err
	}()
	waitForAccountGateWaiters(t, ctx, s, 1)
	authorized := make(chan error, 1)
	go func() { _, err := s.Authorize(ctx, ref, uuid.New()); authorized <- err }()
	waitForAccountGateWaiters(t, ctx, s, 2)
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-rebound; err != nil {
		t.Fatal(err)
	}
	if err = <-authorized; !errors.Is(err, ErrDeliveryNotEligible) {
		t.Fatalf("authorization overtook rebind: %v", err)
	}
	binding, err := s.GetTelegramBinding(ctx, uuidString(owner))
	if err != nil || binding.TelegramChatID != 456 {
		t.Fatalf("binding %#v %v", binding, err)
	}
	if state := deliveryState(t, s, ref); state != "cancelled" {
		t.Fatal(state)
	}
}

func TestOutcomeRevokedPermanentFailureCancelsWorkPreservesAttempt(t *testing.T) {
	s, ref := attemptFixture(t, "account")
	ctx := context.Background()
	p, err := s.Authorize(ctx, ref, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DeleteTelegramBinding(ctx, p.OwnerID); err != nil {
		t.Fatal(err)
	}
	outcome := delivery.Outcome{Kind: "failed", Code: "provider_400"}
	if err = s.RecordOutcome(ctx, p, outcome, time.Now()); err != nil {
		t.Fatal(err)
	}
	if state := deliveryState(t, s, ref); state != "cancelled" {
		t.Fatalf("revoked failed work: %s", state)
	}
	var kind, code string
	if err = s.pool.QueryRow(ctx, `SELECT outcome,outcome_code FROM notification_delivery_attempts WHERE id=$1`, p.AttemptID).Scan(&kind, &code); err != nil {
		t.Fatal(err)
	}
	if kind != "failed" || code != "provider_400" {
		t.Fatalf("attempt lost provider evidence: %s %s", kind, code)
	}
}
