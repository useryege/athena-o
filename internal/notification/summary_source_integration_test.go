//go:build integration

package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	et "github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountcredentials"
	ac "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	ns "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/internal/testutil/pgtest"
	ts "github.com/useryege/athena/internal/tradersync/store"
	tm "github.com/useryege/athena/internal/tradersync/types"
	utiltelegram "github.com/useryege/athena/util/telegram"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func summaryProjectionFixture(t *testing.T, pool *pgxpool.Pool, owner string, index int) tm.Projection {
	return summaryWalletFixture(t, pool, owner, index, common.HexToAddress("0x1111111111111111111111111111111111111111"))
}
func summaryWalletFixture(t *testing.T, pool *pgxpool.Pool, owner string, index int, wallet common.Address) tm.Projection {
	t.Helper()
	ctx := context.Background()
	sub, attempt, interval := uuid.NewString(), uuid.NewString(), uuid.NewString()
	at := time.Now().UTC().Truncate(time.Second).Add(-time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(id,owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,$2,$3,'enabled','healthy','{"displayName":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"avatar":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"profileURL":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"}}'::jsonb)`, sub, owner, wallet.Bytes()); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(id,owner_id,subscription_id,activation_generation,expected_revision,state,effective_at)VALUES($1,$2,$3,1,1,'succeeded',$4)`, attempt, owner, sub, at.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO trader_sync_monitor_intervals(id,owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at)VALUES($1,$2,$3,$4,1,1,1,1,0,$5,$5)`, interval, owner, sub, attempt, at.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	return summaryProjectionSource(t, pool, owner, sub, attempt, wallet, at, index)
}
func summaryProjectionSource(t *testing.T, pool *pgxpool.Pool, owner, sub, attempt string, wallet common.Address, at time.Time, index int) tm.Projection {
	t.Helper()
	ctx := context.Background()
	var epoch int64
	if err := pool.QueryRow(ctx, `INSERT INTO trader_sync_collector_epochs(fencing_token)VALUES(1) RETURNING id`).Scan(&epoch); err != nil {
		t.Fatal(err)
	}
	raw := et.Log{Topics: []common.Hash{}, Data: []byte{}, Address: common.HexToAddress("0xe111180000d2663c0091e4f400237545b87b996b"), BlockHash: common.HexToHash("0xaa"), TxHash: common.BigToHash(common.Big1), Index: uint(index), BlockNumber: 1}
	data, _ := json.Marshal(raw)
	var id int64
	if err := pool.QueryRow(ctx, `INSERT INTO trader_sync_source_records(chain_id,exchange_address,wallet,block_hash,transaction_hash,log_index,block_number,raw_json,collector_epoch,read_sequence,received_at,removed)VALUES(137,$1,$2,$3,$4,$5,1,$6,$7,1,$8,false) RETURNING id`, raw.Address.Bytes(), wallet.Bytes(), raw.BlockHash.Bytes(), raw.TxHash.Bytes(), index, data, epoch, at).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO trader_sync_source_candidates(source_record_id,owner_id,subscription_id,activation_generation,baseline_attempt_id,received_at)VALUES($1,$2,$3,1,$4,$5)`, id, owner, sub, attempt, at); err != nil {
		t.Fatal(err)
	}
	return tm.Projection{Candidate: tm.Candidate{SourceID: id, OwnerID: owner, SubscriptionID: sub, Generation: 1, AttemptID: attempt, ReceivedAt: at}, Trade: tm.Trade{Wallet: wallet, Exchange: raw.Address, Side: "BUY", PositionID: "123", CollateralRaw: "2500000", SharesRaw: "10000000", FeeRaw: "100", CollateralSymbol: "USDC", CollateralDecimals: 6, SharesDecimals: 6, SourceVersion: "polygon137-core-v2-ccc0596074f4", PriceNumerator: "1", PriceDenominator: "4"}, Confirmation: tm.CanonicalEvidence{Status: "confirmed", BlockHash: raw.BlockHash, SettledAt: at, CheckedAt: at}}
}

func TestSummarySourceGateActualStartAndDynamicCooldown(t *testing.T) {
	for _, tc := range []struct{ dynamic, cancel, revoke bool }{{false, false, false}, {true, false, false}, {true, false, true}, {true, true, true}} {
		dynamic := tc.dynamic
		t.Run(fmt.Sprint("dynamic=", dynamic, "/cancel=", tc.cancel, "/revoke=", tc.revoke), func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, owner.ID); e != nil {
				t.Fatal(e)
			}
			trader := ts.NewSQLStore(db.Pool)
			if e = trader.ConfigureActivities("https://athena.test/base"); e != nil {
				t.Fatal(e)
			}
			first := summaryProjectionFixture(t, db.Pool, owner.ID, 1)
			for i := 1; i <= 12; i++ {
				in := first
				if i > 1 {
					in = summaryProjectionSource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, i)
				}
				if _, _, e = trader.Project(ctx, in); e != nil {
					t.Fatal(e)
				}
			}
			var calls atomic.Int32
			httpEntered := make(chan struct{})
			response := make(chan struct{})
			defer func() {
				select {
				case <-response:
				default:
					close(response)
				}
			}()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				_ = r.ParseMultipartForm(1 << 20)
				if r.Form.Get("parse_mode") != "" {
					t.Error("summary was not plain")
				}
				var frozen string
				if e := db.Pool.QueryRow(ctx, `SELECT text FROM trader_sync_summary_parts ORDER BY id LIMIT 1`).Scan(&frozen); e != nil || frozen != r.Form.Get("text") {
					t.Error("request differs from frozen summary", e)
				}
				close(httpEntered)
				select {
				case <-response:
				case <-r.Context().Done():
				}
				io.WriteString(w, `{"ok":true,"result":{"message_id":42}}`)
			}))
			defer func() {
				select {
				case <-response:
				default:
					close(response)
				}
				cancel()
				server.Close()
			}()
			raw, e := utiltelegram.NewClient(utiltelegram.Config{BotToken: "test", BaseURL: server.URL})
			if e != nil {
				t.Fatal(e)
			}
			client := &pausedMessageClient{Client: raw, entered: make(chan struct{}), resume: make(chan struct{})}
			service := NewService(ns.NewSQLStore(db.Pool), NewTelegramSender(client, nil), nil, nil)
			if e = service.ConfigureSummaries(db.Pool, "https://athena.test/base"); e != nil {
				t.Fatal(e)
			}
			source := service.summarySource
			candidates, e := source.Ready(ctx, time.Now())
			if e != nil || len(candidates) != 1 {
				t.Fatal("waiting summary missing from shared WorkSource", candidates, e)
			}
			c := candidates[0]
			if c.Deadline == nil {
				t.Fatal("first head has no deadline")
			}
			clock := &manualDispatchClock{now: time.Now()}
			budget := NewBudget(20, time.Second, 20, time.Minute)
			reservation, ok := budget.Reserve(c, clock.Now())
			if !ok {
				t.Fatal("reserve")
			}
			defer budget.Release(reservation)
			sendCtx, sendCancel := context.WithCancel(ctx)
			defer sendCancel()
			sendCtx = context.WithValue(sendCtx, dispatchSlotKey{}, dispatchSlot{clock: clock, until: clock.Now().Add(time.Second), budget: budget, reservation: reservation})
			done := make(chan error, 1)
			go func() { done <- source.Dispatch(sendCtx, c, func(time.Time) { budget.Start(c, clock.Now()) }) }()
			select {
			case <-client.entered:
			case e := <-done:
				t.Fatal("summary did not enter sender", e)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			var attemptID string
			var batch int64
			if e = db.Pool.QueryRow(ctx, `SELECT current_attempt_id::text,current_batch_id FROM trader_sync_summary_heads WHERE owner_id=$1`, owner.ID).Scan(&attemptID, &batch); e != nil {
				t.Fatal(e)
			}
			next := summaryProjectionSource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, 13)
			projected := make(chan error, 1)
			go func() { _, _, e := trader.Project(ctx, next); projected <- e }()
			select {
			case e := <-projected:
				t.Fatal("activity formed between freeze and ordinary start", e)
			case <-time.After(70 * time.Millisecond):
			}
			if dynamic {
				budget.Tighten(clock.Now().Add(120 * time.Second))
			}
			close(client.resume)
			if dynamic {
				for {
					clock.mu.Lock()
					waiting := len(clock.waiters) > 0
					clock.mu.Unlock()
					if waiting {
						break
					}
					select {
					case e := <-done:
						t.Fatal("unexpected pre-start return", e)
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					default:
						runtime.Gosched()
					}
				}
				if calls.Load() != 0 {
					t.Fatal("HTTP bypassed cooldown")
				}
				select {
				case e := <-projected:
					if e != nil {
						t.Fatal(e)
					}
				case <-ctx.Done():
					t.Fatal("budget wait held account gate")
				}
				var count int
				if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM notification_delivery_attempts WHERE work_kind='account' AND work_id IN(SELECT delivery_id FROM trader_sync_summary_parts)`).Scan(&count); e != nil || count != 1 {
					t.Fatal("cooldown consumed another attempt", count, e)
				}
				pending, e := source.Ready(ctx, clock.Now())
				if e != nil || len(pending) != 0 {
					t.Fatal("later batch overtook durable head", pending, e)
				}
				if tc.revoke {
					if _, e = service.store.DeleteTelegramBinding(ctx, owner.ID); e != nil {
						t.Fatal("revocation blocked during wait", e)
					}
				}
				if tc.cancel {
					sendCancel()
					select {
					case e := <-done:
						if e != nil {
							t.Fatal(e)
						}
					case <-ctx.Done():
						t.Fatal("cancel failed to join sender")
					}
					if calls.Load() != 0 {
						t.Fatal("cancelled prestart issued HTTP")
					}
					if e = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(pgx.Tx) error { return nil }); e != nil {
						t.Fatal("cancel leaked gate", e)
					}
					var state string
					var attempts int
					if e = db.Pool.QueryRow(ctx, `SELECT d.status,d.attempts FROM account_notification_deliveries d JOIN notification_delivery_attempts a ON a.work_id=d.id AND a.work_kind='account' WHERE a.id=$1`, attemptID).Scan(&state, &attempts); e != nil || state != "cancelled" || attempts != 1 {
						t.Fatal(state, attempts, e)
					}
					return
				}
				external, e := txgate.AcquireAccountSession(ctx, db.Pool, owner.ID)
				if e != nil {
					t.Fatal(e)
				}
				defer external.Release(context.Background())
				clock.advance(120 * time.Second)
				for {
					var waiting bool
					if e = db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype='advisory' AND NOT granted)`).Scan(&waiting); e != nil {
						t.Fatal(e)
					}
					if waiting {
						break
					}
					select {
					case <-ctx.Done():
						t.Fatal("sender never reentered gate")
					default:
						runtime.Gosched()
					}
				}
				tightened := make(chan struct{})
				go func() { budget.Tighten(clock.Now().Add(60 * time.Second)); close(tightened) }()
				select {
				case <-tightened:
				case <-ctx.Done():
					t.Fatal("budget read lock was held while waiting account gate")
				}
				if e = external.Release(ctx); e != nil {
					t.Fatal(e)
				}
				for {
					clock.mu.Lock()
					waiting := len(clock.waiters) > 0
					clock.mu.Unlock()
					if waiting {
						break
					}
					select {
					case <-ctx.Done():
						t.Fatal("second tightening was not rechecked")
					default:
						runtime.Gosched()
					}
				}
				if calls.Load() != 0 {
					t.Fatal("reentry bypassed later tightening")
				}
				clock.advance(60 * time.Second)
			}
			select {
			case <-httpEntered:
			case e := <-done:
				t.Fatal("no actual HTTP", e)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if !dynamic {
				select {
				case e := <-projected:
					if e != nil {
						t.Fatal(e)
					}
				case <-ctx.Done():
					t.Fatal("HTTP response held activity gate")
				}
			}
			var hasStart bool
			if e = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error {
				return tx.QueryRow(ctx, `SELECT first_started_at IS NOT NULL FROM trader_sync_summary_batches WHERE id=$1`, batch).Scan(&hasStart)
			}); e != nil || !hasStart {
				t.Fatal("activity passed before start persisted", hasStart, e)
			}
			close(response)
			select {
			case e := <-done:
				if e != nil {
					t.Fatal(e)
				}
			case <-ctx.Done():
				t.Fatal("dispatch did not join", ctx.Err())
			}
			var state string
			var attempts int
			if e = db.Pool.QueryRow(ctx, `SELECT d.status,d.attempts FROM account_notification_deliveries d JOIN notification_delivery_attempts a ON a.work_id=d.id AND a.work_kind='account' WHERE a.id=$1`, attemptID).Scan(&state, &attempts); e != nil || state != "sent" || attempts != 1 || calls.Load() != 1 {
				t.Fatal("same permit was not completed once", state, attempts, calls.Load(), e)
			}
			if dynamic && !tc.revoke {
				var f, recorded, started time.Time
				if e = db.Pool.QueryRow(ctx, `SELECT b.frozen_at,a.recorded_at,b.first_started_at FROM trader_sync_summary_batches b JOIN trader_sync_activities a ON a.owner_id=b.owner_id WHERE b.id=$1 AND a.source_record_id=$2`, batch, next.Candidate.SourceID).Scan(&f, &recorded, &started); e != nil || !f.Before(recorded) || !recorded.Before(started) {
					t.Fatal("f<t<s evidence not retained", f, recorded, started, e)
				}
				future, e := source.Ready(ctx, clock.Now())
				if e != nil || len(future) != 1 || future[0].Deadline == nil || !future[0].Deadline.Equal(recorded.Add(time.Minute)) || !future[0].NotBefore.After(*future[0].Deadline) {
					t.Fatal("impossible dynamic deadline was hidden or rewritten", future, e)
				}
			}
			if dynamic {
				var wait int64
				var reason string
				if e = db.Pool.QueryRow(ctx, `SELECT budget_wait_ms,budget_reason FROM trader_sync_summary_batches WHERE id=$1`, batch).Scan(&wait, &reason); e != nil || wait < 120000 || reason != "telegram_retry_after" {
					t.Fatal("dynamic delay not preserved", wait, e)
				}
			}
		})
	}
}

func TestSummaryMissingStartKeepsSentAndDurableHead(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, owner.ID); e != nil {
		t.Fatal(e)
	}
	trader := ts.NewSQLStore(db.Pool)
	if e = trader.ConfigureActivities("https://athena.test/base"); e != nil {
		t.Fatal(e)
	}
	first := summaryProjectionFixture(t, db.Pool, owner.ID, 1)
	for i := 1; i <= 12; i++ {
		in := first
		if i > 1 {
			in = summaryProjectionSource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, i)
		}
		if _, _, e = trader.Project(ctx, in); e != nil {
			t.Fatal(e)
		}
	}

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		io.Copy(io.Discard, r.Body)
		io.WriteString(w, `{"ok":true,"result":{"message_id":42}}`)
	}))
	defer server.Close()
	raw, e := utiltelegram.NewClient(utiltelegram.Config{BotToken: "test", BaseURL: server.URL})
	if e != nil {
		t.Fatal(e)
	}
	service := NewService(ns.NewSQLStore(db.Pool), NewTelegramSender(raw, nil), nil, nil)
	if e = service.ConfigureSummaries(db.Pool, "https://athena.test/base"); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `CREATE FUNCTION reject_summary_start()RETURNS trigger LANGUAGE plpgsql AS $$BEGIN PERFORM pg_sleep(2);RAISE EXCEPTION 'start write failed';END$$;CREATE TRIGGER reject_summary_start BEFORE UPDATE ON notification_delivery_attempts FOR EACH ROW WHEN(NEW.started_at IS DISTINCT FROM OLD.started_at)EXECUTE FUNCTION reject_summary_start()`); e != nil {
		t.Fatal(e)
	}
	source := service.summarySource
	ready, e := source.Ready(ctx, time.Now())
	if e != nil || len(ready) != 1 {
		t.Fatal(ready, e)
	}
	began := time.Now()
	if e = source.Dispatch(ctx, ready[0], nil); e != nil {
		t.Fatal(e)
	}
	if time.Since(began) > 2500*time.Millisecond {
		t.Fatal("start persistence held gate beyond bounded deadline")
	}
	var state string
	var starts int
	var head bool
	if e = db.Pool.QueryRow(ctx, `SELECT d.status,CASE WHEN b.first_started_at IS NULL THEN 0 ELSE 1 END,h.current_batch_id IS NOT NULL FROM trader_sync_summary_heads h JOIN trader_sync_summary_batches b ON b.owner_id=h.owner_id JOIN trader_sync_summary_parts p ON p.batch_id=b.id JOIN account_notification_deliveries d ON d.id=p.delivery_id WHERE h.owner_id=$1`, owner.ID).Scan(&state, &starts, &head); e != nil || state != "sent" || starts != 0 || !head || calls.Load() != 1 {
		t.Fatal("missing start corrupted known ACK", state, starts, head, calls.Load(), e)
	}
	// A receipt without an actual persisted start cannot be mistaken for proof
	// of never having entered HTTP, even when retrying that write still fails.
	if _, e = source.Ready(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	if e = db.Pool.QueryRow(ctx, `SELECT current_batch_id IS NOT NULL FROM trader_sync_summary_heads WHERE owner_id=$1`, owner.ID).Scan(&head); e != nil || !head {
		t.Fatal("sent-without-start head was prematurely cleared", head, e)
	}
	if _, e = db.Pool.Exec(ctx, `DROP TRIGGER reject_summary_start ON notification_delivery_attempts;DROP FUNCTION reject_summary_start()`); e != nil {
		t.Fatal(e)
	}
	if _, e = source.Ready(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	if e = db.Pool.QueryRow(ctx, `SELECT current_batch_id IS NOT NULL FROM trader_sync_summary_heads WHERE owner_id=$1`, owner.ID).Scan(&head); e != nil || head {
		t.Fatal("live observed start was not repaired", head, e)
	}
}

// This fixture constructs actual activities through Task10's Project transaction.
func summaryServiceFixture(t *testing.T, client utiltelegram.Client) (*Service, *pgxpool.Pool, string, tm.Projection) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, owner.ID); e != nil {
		t.Fatal(e)
	}
	trader := ts.NewSQLStore(db.Pool)
	if e = trader.ConfigureActivities("https://athena.test/base"); e != nil {
		t.Fatal(e)
	}
	first := summaryProjectionFixture(t, db.Pool, owner.ID, 1)
	for i := 1; i <= 12; i++ {
		in := first
		if i > 1 {
			in = summaryProjectionSource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, i)
		}
		if _, _, e = trader.Project(ctx, in); e != nil {
			t.Fatal(e)
		}
	}

	service := NewService(ns.NewSQLStore(db.Pool), NewTelegramSender(client, nil), nil, nil)
	if e = service.ConfigureSummaries(db.Pool, "https://athena.test/base"); e != nil {
		t.Fatal(e)
	}
	return service, db.Pool, owner.ID, first
}

type summaryLostCommit struct {
	pgx.Tx
	committed bool
}

func (tx summaryLostCommit) Commit(ctx context.Context) error {
	if tx.committed {
		if e := tx.Tx.Commit(ctx); e != nil {
			return e
		}
	} else {
		if e := tx.Tx.Rollback(ctx); e != nil {
			return e
		}
	}
	return fmt.Errorf("injected lost COMMIT acknowledgement")
}
func TestSummaryPermitUnknownCommitUsesExactReadback(t *testing.T) {
	for _, committed := range []bool{true, false} {
		t.Run(fmt.Sprint(committed), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				io.Copy(io.Discard, r.Body)
				io.WriteString(w, `{"ok":true,"result":{"message_id":42}}`)
			}))
			defer server.Close()
			raw, e := utiltelegram.NewClient(utiltelegram.Config{BotToken: "test", BaseURL: server.URL})
			if e != nil {
				t.Fatal(e)
			}
			service, pool, owner, _ := summaryServiceFixture(t, raw)
			source := service.summarySource
			source.beginTx = func(ctx context.Context, c *pgxpool.Conn) (pgx.Tx, error) {
				tx, e := c.Begin(ctx)
				if e != nil {
					return nil, e
				}
				return summaryLostCommit{Tx: tx, committed: committed}, nil
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			ready, e := source.Ready(ctx, time.Now())
			if e != nil || len(ready) != 1 {
				t.Fatal(ready, e)
			}
			e = source.Dispatch(ctx, ready[0], nil)
			if (e == nil) != committed {
				t.Fatal("commit evidence classification", committed, e)
			}
			var attempts int
			if e = pool.QueryRow(ctx, `SELECT count(*) FROM notification_delivery_attempts`).Scan(&attempts); e != nil {
				t.Fatal(e)
			}
			want := 0
			if committed {
				want = 1
			}
			if attempts != want || int(calls.Load()) != want {
				t.Fatal("unknown commit repeated or assumed permit", attempts, calls.Load())
			}
			if e = txgate.WithAccountTx(ctx, pool, owner, func(pgx.Tx) error { return nil }); e != nil {
				t.Fatal("commit branch leaked account gate", e)
			}
		})
	}
}

type summaryPrestartFailure struct{ utiltelegram.Client }

func (c summaryPrestartFailure) SendMessage(context.Context, utiltelegram.SendMessageRequest) (*utiltelegram.SendMessageResponse, error) {
	return nil, &utiltelegram.SendError{Kind: "failed", Code: "not_started", Err: fmt.Errorf("local validation rejected before HTTP")}
}
func TestSummaryFirstNotStartedAndActual429KeepDifferentBoundaries(t *testing.T) {
	for _, kind := range []string{"not_started", "429"} {
		t.Run(kind, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				io.Copy(io.Discard, r.Body)
				w.WriteHeader(429)
				io.WriteString(w, `{"ok":false,"error_code":429,"description":"slow down","parameters":{"retry_after":1}}`)
			}))
			defer server.Close()
			raw, e := utiltelegram.NewClient(utiltelegram.Config{BotToken: "test", BaseURL: server.URL})
			if e != nil {
				t.Fatal(e)
			}
			client := raw
			if kind == "not_started" {
				client = summaryPrestartFailure{raw}
			}
			service, pool, owner, first := summaryServiceFixture(t, client)
			source := service.summarySource
			ctx := context.Background()
			ready, e := source.Ready(ctx, time.Now())
			if e != nil || len(ready) != 1 {
				t.Fatal(ready, e)
			}
			e = source.Dispatch(ctx, ready[0], nil)
			if kind == "429" {
				var rate *RateLimitError
				if !errors.As(e, &rate) {
					t.Fatal("lost rate limit", e)
				}
			} else if e != nil {
				t.Fatal(e)
			}
			if _, e = source.Ready(ctx, time.Now()); e != nil {
				t.Fatal(e)
			}
			var started *time.Time
			var head bool
			var batch int64
			if e = pool.QueryRow(ctx, `SELECT b.id,b.first_started_at,h.current_batch_id IS NOT NULL FROM trader_sync_summary_batches b JOIN trader_sync_summary_heads h USING(owner_id) WHERE b.owner_id=$1`, owner).Scan(&batch, &started, &head); e != nil || head || (started != nil) != (kind == "429") {
				t.Fatal("false first start boundary", started, head, e)
			}
			if kind == "not_started" {
				if calls.Load() != 0 {
					t.Fatal("local failure entered HTTP")
				}
				return
			}
			next := summaryProjectionSource(t, pool, owner, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, 13)
			if _, _, e = source.trader.Project(ctx, next); e != nil {
				t.Fatal(e)
			}
			future, e := source.Ready(ctx, time.Now())
			if e != nil || len(future) != 1 || future[0].NotBefore.Before(started.Add(time.Minute)) || future[0].Deadline == nil {
				t.Fatal("next batch lost 60-second reservation", future, e)
			}
			old, e := service.store.DispatchItems(ctx, "account")
			if e != nil {
				t.Fatal(e)
			}
			found := false
			for _, item := range old {
				if item.Source == "trader_sync" {
					var exists bool
					if e = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM trader_sync_summary_parts WHERE delivery_id=$1 AND batch_id=$2)`, item.Candidate.Ref.ID, batch).Scan(&exists); e != nil {
						t.Fatal(e)
					}
					found = found || exists
				}
			}
			if !found {
				t.Fatal("old batch retry vanished while next first head has future deadline")
			}
		})
	}
}
