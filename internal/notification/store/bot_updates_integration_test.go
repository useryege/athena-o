//go:build integration

package store

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/notification/delivery"
	"github.com/useryege/athena/internal/testutil/pgtest"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

type botUpdateApplier interface {
	ApplyBotUpdate(context.Context, utiltelegram.Update) error
}

func applyTestUpdate(t *testing.T, s *SQLStore, u utiltelegram.Update) error {
	t.Helper()
	a, ok := any(s).(botUpdateApplier)
	if !ok {
		t.Fatal("SQLStore must atomically apply Bot update, binding, reply and offset")
	}
	return a.ApplyBotUpdate(context.Background(), u)
}
func botUpdateFixture(t *testing.T) (*SQLStore, string, utiltelegram.Update) {
	t.Helper()
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	s := NewSQLStore(db.Pool)
	owner := uuid.NewString()
	token := base64.RawURLEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	digest := sha256.Sum256([]byte(token))
	_, err := s.CreateTelegramBindingAttempt(context.Background(), CreateTelegramBindingAttemptRequest{ID: uuid.NewString(), AccountID: owner, TokenDigest: digest[:], ExpiresAt: time.Now().Add(5 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	return s, owner, utiltelegram.Update{ID: 42, Message: &utiltelegram.Message{ChatType: "private", UserID: 123, ChatID: 123, Text: "/start " + token, FirstName: "Member"}}
}
func TestApplyBotUpdateDuplicate(t *testing.T) {
	s, owner, u := botUpdateFixture(t)
	for range 2 {
		if err := applyTestUpdate(t, s, u); err != nil {
			t.Fatal(err)
		}
	}
	b, err := s.GetTelegramBinding(context.Background(), owner)
	if err != nil || b.Revision != 1 {
		t.Fatalf("binding=%+v error=%v", b, err)
	}
	assertBotUpdateCounts(t, s, 1, 1, 43)
}
func assertBotUpdateCounts(t *testing.T, s *SQLStore, consumed, replies, next int) {
	t.Helper()
	var c, r, n int
	err := s.pool.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM telegram_consumed_updates),(SELECT count(*) FROM telegram_binding_replies),(SELECT next_update_id FROM telegram_polling_state)`).Scan(&c, &r, &n)
	if err != nil {
		t.Fatal(err)
	}
	if c != consumed || r != replies || n != next {
		t.Fatalf("consumed/replies/offset=%d/%d/%d want %d/%d/%d", c, r, n, consumed, replies, next)
	}
}
func TestApplyBotUpdateRejectedMessages(t *testing.T) {
	for _, branch := range []string{"invalid", "unknown", "group", "bot", "identity_used", "expired"} {
		t.Run(branch, func(t *testing.T) {
			s, owner, u := botUpdateFixture(t)
			replies := 1
			switch branch {
			case "invalid":
				u.Message.Text = "/start invalid"
			case "unknown":
				u.Message.Text = "/start " + base64.RawURLEncoding.EncodeToString(make([]byte, 32))
			case "group":
				u.Message.ChatType = "group"
				replies = 0
			case "bot":
				u.Message.IsBot = true
				replies = 0
			case "expired":
				if _, err := s.pool.Exec(context.Background(), `UPDATE telegram_binding_attempts SET expires_at=now()-interval '1 second'`); err != nil {
					t.Fatal(err)
				}
			case "identity_used":
				if _, err := s.pool.Exec(context.Background(), `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision) VALUES($1,123,123,'other',1)`, uuid.NewString()); err != nil {
					t.Fatal(err)
				}
			}
			for range 2 {
				if err := applyTestUpdate(t, s, u); err != nil {
					t.Fatal(err)
				}
			}
			if b, err := s.GetTelegramBinding(context.Background(), owner); err == nil {
				t.Fatalf("unexpected binding %+v", b)
			}
			assertBotUpdateCounts(t, s, 1, replies, 43)
		})
	}
}

// Wrap the real database transaction: no business hooks or synthetic database state.
type commitFaultPool struct {
	sqlPool
	after bool
}

func (p commitFaultPool) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := p.sqlPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return commitFaultTx{Tx: tx, after: p.after}, nil
}

type commitFaultTx struct {
	pgx.Tx
	after bool
}

var errCommitDisconnected = errors.New("test: commit connection lost")

func (tx commitFaultTx) Commit(ctx context.Context) error {
	if tx.after {
		if err := tx.Tx.Commit(ctx); err != nil {
			return err
		}
	}
	return errCommitDisconnected
}
func TestApplyBotUpdateCommitFaults(t *testing.T) {
	for _, after := range []bool{false, true} {
		t.Run(fmt.Sprintf("after_commit_%v", after), func(t *testing.T) {
			s, owner, u := botUpdateFixture(t)
			real := s.pool
			s.pool = commitFaultPool{sqlPool: real, after: after}
			if err := applyTestUpdate(t, s, u); !errors.Is(err, errCommitDisconnected) {
				t.Fatalf("expected injected commit error, got %v", err)
			}
			if !after {
				assertBotUpdateCounts(t, s, 0, 0, 0)
				var b, v, a int
				if err := s.pool.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM telegram_bindings),(SELECT count(*) FROM telegram_binding_versions),(SELECT count(*) FROM telegram_binding_attempts WHERE status='pending')`).Scan(&b, &v, &a); err != nil {
					t.Fatal(err)
				}
				if b != 0 || v != 0 || a != 1 {
					t.Fatalf("transaction leaked binding/version/attempt=%d/%d/%d", b, v, a)
				}
			} else {
				assertBotUpdateCounts(t, s, 1, 1, 43)
			}
			s.pool = real
			if err := applyTestUpdate(t, s, u); err != nil {
				t.Fatal(err)
			}
			assertBotUpdateCounts(t, s, 1, 1, 43)
			b, err := s.GetTelegramBinding(context.Background(), owner)
			if err != nil || b.Revision != 1 {
				t.Fatalf("replayed binding=%+v %v", b, err)
			}
		})
	}
}
func TestApplyBotUpdateChatMemberRevokesOldWork(t *testing.T) {
	s, owner, u := botUpdateFixture(t)
	if err := applyTestUpdate(t, s, u); err != nil {
		t.Fatal(err)
	}
	u = utiltelegram.Update{ID: 43, MyChatMember: &utiltelegram.MyChatMemberUpdate{ChatType: "private", UserID: 123, ChatID: 123, NewStatus: "kicked"}}
	if err := applyTestUpdate(t, s, u); err != nil {
		t.Fatal(err)
	}
	b, err := s.GetTelegramBinding(context.Background(), owner)
	if err != nil || b.Status != "unreachable" {
		t.Fatalf("binding=%+v %v", b, err)
	}
	var state string
	if err := s.pool.QueryRow(context.Background(), `SELECT status FROM telegram_binding_replies`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "cancelled" {
		t.Fatal(state)
	}
	u.ID = 44
	u.MyChatMember.NewStatus = "member"
	if err := applyTestUpdate(t, s, u); err != nil {
		t.Fatal(err)
	}
	b, err = s.GetTelegramBinding(context.Background(), owner)
	if err != nil || b.Status != "connected" || b.Revision != 1 {
		t.Fatalf("binding=%+v %v", b, err)
	}
	if err := s.pool.QueryRow(context.Background(), `SELECT status FROM telegram_binding_replies`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "cancelled" {
		t.Fatal("old reply revived", state)
	}
	assertBotUpdateCounts(t, s, 3, 1, 45)
}

func TestApplyBotUpdateRebindRevokesReplyAttempts(t *testing.T) {
	for _, sending := range []bool{false, true} {
		t.Run(fmt.Sprintf("sending_%v", sending), func(t *testing.T) {
			s, owner, u := botUpdateFixture(t)
			ctx := context.Background()
			if err := applyTestUpdate(t, s, u); err != nil {
				t.Fatal(err)
			}
			var id int64
			if err := s.pool.QueryRow(ctx, `SELECT id FROM telegram_binding_replies`).Scan(&id); err != nil {
				t.Fatal(err)
			}
			ref := delivery.WorkRef{Kind: "reply", ID: id}
			var permit delivery.Permit
			if sending {
				var err error
				permit, err = s.Authorize(ctx, ref, uuid.New())
				if err != nil {
					t.Fatal(err)
				}
			}
			token := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
			digest := sha256.Sum256([]byte(token))
			if _, err := s.CreateTelegramBindingAttempt(ctx, CreateTelegramBindingAttemptRequest{ID: uuid.NewString(), AccountID: owner, TokenDigest: digest[:], ExpiresAt: time.Now().Add(time.Minute)}); err != nil {
				t.Fatal(err)
			}
			u.ID++
			u.Message.Text = "/start " + token
			u.Message.UserID = 456
			u.Message.ChatID = 456
			if err := applyTestUpdate(t, s, u); err != nil {
				t.Fatal(err)
			}
			var state string
			var revoked bool
			if err := s.pool.QueryRow(ctx, `SELECT status,eligibility_revoked_at IS NOT NULL FROM telegram_binding_replies WHERE id=$1`, id).Scan(&state, &revoked); err != nil {
				t.Fatal(err)
			}
			if !revoked || (!sending && state != "cancelled") || (sending && state != "sending") {
				t.Fatalf("state/revoked %s/%v", state, revoked)
			}
			if sending {
				if err := s.RecordOutcome(ctx, permit, delivery.Outcome{Kind: "retryable", Code: "provider_429", RetryAfter: 7 * time.Second}, time.Now()); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.Authorize(ctx, ref, uuid.New()); !errors.Is(err, ErrDeliveryNotEligible) {
				t.Fatalf("revived old reply %v", err)
			}
			b, err := s.GetTelegramBinding(ctx, owner)
			if err != nil || b.Revision != 2 || b.TelegramChatID != 456 {
				t.Fatalf("binding %+v %v", b, err)
			}
			assertBotUpdateCounts(t, s, 2, 2, 44)
		})
	}
}
func TestApplyBotUpdateConcurrentDuplicate(t *testing.T) {
	s, _, u := botUpdateFixture(t)
	results := make(chan error, 2)
	for range 2 {
		go func() { results <- s.ApplyBotUpdate(context.Background(), u) }()
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	assertBotUpdateCounts(t, s, 1, 1, 43)
}
func TestReplyRetryCeilingAndUnknown(t *testing.T) {
	for _, kind := range []string{"retryable", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			s, ref := attemptFixture(t, "reply")
			ctx := context.Background()
			n := 5
			if kind == "unknown" {
				n = 1
			}
			for i := 1; i <= n; i++ {
				p, err := s.Authorize(ctx, ref, uuid.New())
				if err != nil {
					t.Fatal(err)
				}
				at := time.Now().Add(-time.Minute).Truncate(time.Microsecond)
				if err := s.RecordOutcome(ctx, p, delivery.Outcome{Kind: kind, RetryAfter: 7 * time.Second, Code: "test"}, at); err != nil {
					t.Fatal(err)
				}
				var state string
				var next time.Time
				var attempts int
				if err := s.pool.QueryRow(ctx, `SELECT status,attempts,next_attempt_at FROM telegram_binding_replies WHERE id=$1`, ref.ID).Scan(&state, &attempts, &next); err != nil {
					t.Fatal(err)
				}
				if attempts != i {
					t.Fatal(attempts)
				}
				if kind == "unknown" {
					if state != "unknown" {
						t.Fatal(state)
					}
				} else if i == 5 {
					if state != "failed" {
						t.Fatal(state)
					}
				} else {
					wait := 7 * time.Second
					if i == 4 {
						wait = 8 * time.Second
					}
					if state != "pending" || !next.Equal(at.Add(wait)) {
						t.Fatalf("retry state/time %s %v", state, next)
					}
				}
			}
			if _, err := s.Authorize(ctx, ref, uuid.New()); !errors.Is(err, ErrDeliveryNotEligible) {
				t.Fatalf("terminal reply revived %v", err)
			}
		})
	}
}
