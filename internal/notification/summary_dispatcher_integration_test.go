//go:build integration

package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/notification/delivery"
	ts "github.com/useryege/athena/internal/tradersync/store"
	tm "github.com/useryege/athena/internal/tradersync/types"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

// The database and real transport use wall time. Only wakeups are controlled;
// Now keeps the real UTC/monotonic pairing instead of extrapolating a stale UTC
// origin across host clock corrections during this actual 60-second scenario.
type summaryWakeClock struct {
	mu      sync.Mutex
	waiters []chan time.Time
}

func (*summaryWakeClock) Now() time.Time { return time.Now() }
func (c *summaryWakeClock) After(time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan time.Time, 1)
	c.waiters = append(c.waiters, ch)
	return ch
}
func (c *summaryWakeClock) wake() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, ch := range c.waiters {
		ch <- time.Now()
	}
	c.waiters = nil
}

type summaryMixedPause struct {
	batch   int64
	entered time.Time
}

type summaryMixedEvent struct {
	chat int64
	text string
	at   time.Time
}

// This client controls only local preparation before the real HTTP admission.
// All sources, Dispatcher reservations, permits, callbacks and result writes are real.
type summaryMixedClient struct {
	utiltelegram.Client
	mu         sync.Mutex
	late       bool
	firstBatch int64
	delayed    chan summaryMixedPause
	resume     chan struct{}
	once       sync.Once
}

func (c *summaryMixedClient) SendMessage(ctx context.Context, r utiltelegram.SendMessageRequest) (*utiltelegram.SendMessageResponse, error) {
	var batch int64
	var part, total int
	_, _ = fmt.Sscanf(r.Text, "摘要 %d · %d/%d", &batch, &part, &total)
	c.mu.Lock()
	if batch > 0 && c.firstBatch == 0 {
		c.firstBatch = batch
	}
	delay := c.late && batch > 0 && batch != c.firstBatch && part == 1
	c.mu.Unlock()
	if delay {
		c.once.Do(func() { c.delayed <- summaryMixedPause{batch: batch, entered: time.Now()} })
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-c.resume:
		}
	}
	return c.Client.SendMessage(ctx, r)
}

func summaryMixedOrdinary(t *testing.T, pool *pgxpool.Pool, owner string, chat int64, key string) {
	t.Helper()
	payload, e := delivery.EncodePayload(delivery.Payload{Format: "html", Text: key})
	if e != nil {
		t.Fatal(e)
	}
	_, e = pool.Exec(context.Background(), `INSERT INTO account_notification_deliveries(account_id,idempotency_key,payload_digest,source,severity,body,channel,status,telegram_chat_id,binding_revision,payload,request_digest)VALUES($1,$2,$3,'test','info',$2,'telegram','pending',$4,1,$5,$3)`, owner, key, delivery.PayloadDigest(payload), chat, payload)
	if e != nil {
		t.Fatal(e)
	}
}

func TestSummaryDispatcherMixedSourcesAndNextHead(t *testing.T) {
	for _, miss := range []bool{false, true} {
		t.Run(fmt.Sprint("local_miss=", miss), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 95*time.Second)
			defer cancel()
			events := make(chan summaryMixedEvent, 512)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = r.ParseMultipartForm(1 << 20)
				chat, _ := strconv.ParseInt(r.Form.Get("chat_id"), 10, 64)
				events <- summaryMixedEvent{chat: chat, text: r.Form.Get("text"), at: time.Now()}
				io.WriteString(w, `{"ok":true,"result":{"message_id":42}}`)
			}))
			defer server.Close()
			raw, e := utiltelegram.NewClient(utiltelegram.Config{BotToken: "test", BaseURL: server.URL})
			if e != nil {
				t.Fatal(e)
			}
			client := &summaryMixedClient{Client: raw, late: miss, delayed: make(chan summaryMixedPause, 1), resume: make(chan struct{})}
			defer func() {
				select {
				case <-client.resume:
				default:
					close(client.resume)
				}
			}()
			service, pool, owner, first := summaryServiceFixture(t, client)
			session, e := service.store.AcquireSender(ctx, service.senderIncarnation)
			if e != nil {
				t.Fatal(e)
			}
			defer session.Close()
			service.senderSession = session
			service.sender = NewTelegramSender(client, map[string]string{"test": "-777"})
			trader := ts.NewSQLStore(pool)
			if e = trader.ConfigureActivities("https://athena.test/base"); e != nil {
				t.Fatal(e)
			}
			// Real Project produced ten ordinary deliveries and summary members 11 and 12.
			var ordinary, summary int
			if e = pool.QueryRow(ctx, `SELECT count(*) FILTER(WHERE notification_mode='ordinary'),count(*) FILTER(WHERE notification_mode='summary') FROM trader_sync_activities WHERE owner_id=$1`, owner).Scan(&ordinary, &summary); e != nil || ordinary != 10 || summary != 2 {
				t.Fatal("10/11 classification", ordinary, summary, e)
			}
			metadata := tm.TradeMetadata{Relationship: "AND", LegsEvidence: tm.Evidence{Availability: "available"}}
			// A bounded large batch has more than 60 complete parts. A private chat cannot
			// drain it before the next head; no result/next_attempt_at/cohort is forged.
			for i := 0; i < 450; i++ {
				metadata.Legs = append(metadata.Legs, tm.ComboLeg{PositionID: fmt.Sprint(i + 1), Market: tm.MarketRef{Evidence: tm.Evidence{Availability: "available"}, Title: strings.Repeat("🙂", 90), ConditionID: fmt.Sprint(i + 2), Outcome: "YES", URL: fmt.Sprintf("https://market.test/leg/%d", i)}})
			}
			data, _ := json.Marshal(metadata)
			if _, e = pool.Exec(ctx, `UPDATE trader_sync_market_metadata SET metadata_json=$1`, data); e != nil {
				t.Fatal(e)
			}
			other := uuid.NewString()
			if _, e = pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,456,456,'other',1)`, other); e != nil {
				t.Fatal(e)
			}
			if _, e = pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id)VALUES('test','mixed-summary',1)`); e != nil {
				t.Fatal(e)
			}
			enqueueCompetition := func(suffix string) {
				summaryMixedOrdinary(t, pool, owner, 123, "same-owner-ordinary-"+suffix)
				summaryMixedOrdinary(t, pool, other, 456, "other-"+suffix)
				payload, err := delivery.EncodePayload(delivery.Payload{Format: "html", Text: "system-" + suffix, MessageThreadID: 1})
				if err != nil {
					t.Fatal(err)
				}
				if _, err = pool.Exec(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest)VALUES('test','info',$1,'telegram','pending','test','mixed-summary',$2,$3)`, "system-"+suffix, payload, delivery.PayloadDigest(payload)); err != nil {
					t.Fatal(err)
				}
				if err = service.store.ApplyBotUpdate(ctx, utiltelegram.Update{ID: time.Now().UnixNano(), Message: &utiltelegram.Message{ChatType: "private", UserID: 789, ChatID: 789, Text: "/start invalid"}}); err != nil {
					t.Fatal(err)
				}
			}
			enqueueCompetition("initial")
			clock := &summaryWakeClock{}
			budget := NewBudget(20, time.Second, 20, time.Minute)
			done := make(chan error, 1)
			go func() { done <- NewDispatcher(clock, service.workSources(), budget, 12).Run(ctx) }()
			defer func() {
				cancel()
				select {
				case e := <-done:
					if e != nil && !errors.Is(e, context.Canceled) {
						t.Error(e)
					}
				case <-time.After(6 * time.Second):
					t.Error("dispatcher failed to join")
				}
			}()
			tick := clock.wake
			waitUntil := func(test func() bool, limit time.Duration) {
				end := time.Now().Add(limit)
				for !test() {
					select {
					case e := <-done:
						done <- e
						t.Fatal("dispatcher stopped", e)
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					default:
					}
					if time.Now().After(end) {
						t.Fatal("mixed dispatcher condition timed out")
					}
					tick()
					time.Sleep(10 * time.Millisecond)
				}
			}
			var firstStart, oldest time.Time
			waitUntil(func() bool {
				return pool.QueryRow(ctx, `SELECT first_started_at,oldest_at FROM trader_sync_summary_batches WHERE id=1`).Scan(&firstStart, &oldest) == nil
			}, 18*time.Second)
			if firstStart.After(oldest.Add(time.Minute)) {
				t.Fatal("initial head missed normal deadline", firstStart, oldest)
			}
			var parts int
			if e = pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_summary_parts WHERE batch_id=1`).Scan(&parts); e != nil || parts <= 60 {
				t.Fatal("old batch cannot remain pending at next window", parts, e)
			}
			memberOffset := 5 * time.Second
			if miss {
				memberOffset = time.Second
			}
			waitUntil(func() bool { return time.Now().After(firstStart.Add(memberOffset)) }, 6*time.Second)
			in := summaryProjectionSource(t, pool, owner, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, 13)
			in.Trade.PositionID = "456" // A new market key; do not inherit the old 450-leg metadata.
			if _, _, e = trader.Project(ctx, in); e != nil {
				t.Fatal(e)
			}
			// A member formed just inside the original 60s burst still belongs to summary.
			var origin time.Time
			if e = pool.QueryRow(ctx, `SELECT min(recorded_at) FROM trader_sync_activities WHERE owner_id=$1`, owner).Scan(&origin); e != nil {
				t.Fatal(e)
			}
			waitUntil(func() bool { return time.Now().After(origin.Add(59 * time.Second)) }, 60*time.Second)
			in = summaryProjectionSource(t, pool, owner, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, 14)
			in.Trade.PositionID = "456"
			id, _, e := trader.Project(ctx, in)
			if e != nil {
				t.Fatal(e)
			}
			var mode string
			if e = pool.QueryRow(ctx, `SELECT notification_mode FROM trader_sync_activities WHERE id=$1`, id).Scan(&mode); e != nil || mode != "summary" {
				t.Fatal("member just inside 60s window changed cohort", mode, e)
			}
			enqueueCompetition("near-head")
			if miss {
				var pause summaryMixedPause
				waitUntil(func() bool {
					select {
					case pause = <-client.delayed:
						return true
					default:
						return false
					}
				}, 18*time.Second)
				// Deliberate local preparation stall, not a provider RetryAfter. Do not alter
				// recorded_at/oldest/frozen_at; the persisted actual start must retain the miss.
				var observedBatch int64
				if e = pool.QueryRow(ctx, `SELECT p.batch_id FROM trader_sync_summary_parts p JOIN account_notification_deliveries d ON d.id=p.delivery_id WHERE p.batch_id=$1 AND p.part_index=1 AND d.status='sending'`, pause.batch).Scan(&observedBatch); e != nil {
					t.Fatal("delay was not attached to the committed first permit", e)
				}
				releaseAt := firstStart.Add(63 * time.Second)
				if pause.entered.Add(2 * time.Second).After(releaseAt) {
					releaseAt = pause.entered.Add(2 * time.Second)
				}
				waitUntil(func() bool { return time.Now().After(releaseAt) }, 5*time.Second)
				close(client.resume)
			}
			var nextStart, nextOldest, frozen time.Time
			waitUntil(func() bool {
				return pool.QueryRow(ctx, `SELECT first_started_at,oldest_at,frozen_at FROM trader_sync_summary_batches WHERE id=2`).Scan(&nextStart, &nextOldest, &frozen) == nil
			}, 18*time.Second)
			if nextStart.Before(firstStart.Add(time.Minute)) {
				t.Fatal("head violated actual 60s start interval", firstStart, nextStart)
			}
			t.Logf("timing first=%s oldest2=%s frozen2=%s started2=%s local=%s", firstStart, nextOldest, frozen, nextStart, nextStart.Sub(frozen))
			actualMiss := nextStart.After(nextOldest.Add(time.Minute))
			if actualMiss != miss {
				var info string
				_ = pool.QueryRow(ctx, `SELECT row_to_json(b)::text FROM trader_sync_summary_batches b WHERE id=2`).Scan(&info)
				t.Log(info)
				t.Fatal("normal/miss sample changed", miss, firstStart, nextStart, nextOldest)
			}
			var misses, waitingOld, missingLinks, totalMembers int
			if e = pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_summary_batches WHERE first_started_at>oldest_at+interval '60 seconds'`).Scan(&misses); e != nil || misses != map[bool]int{false: 0, true: 1}[miss] {
				t.Fatal("persisted miss count", misses, e)
			}
			if e = pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_summary_parts p JOIN account_notification_deliveries d ON d.id=p.delivery_id WHERE p.batch_id=1 AND d.status='pending'`).Scan(&waitingOld); e != nil || waitingOld == 0 {
				t.Fatal("old parts drained before next head", waitingOld, e)
			}
			if e = pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_alert_memberships m WHERE m.owner_id=$1 AND m.form='summary' AND NOT EXISTS(SELECT 1 FROM trader_sync_summary_part_items i WHERE i.batch_id=m.batch_id AND i.activity_id=m.activity_id)`, owner).Scan(&missingLinks); e != nil || missingLinks != 0 {
				t.Fatal("summary member lost", missingLinks, e)
			}
			if e = pool.QueryRow(ctx, `SELECT count(DISTINCT activity_id) FROM trader_sync_summary_part_items`).Scan(&totalMembers); e != nil || totalMembers != 4 {
				t.Fatal("incomplete frozen members", totalMembers, e)
			}
			var retryWait int64
			if e = pool.QueryRow(ctx, `SELECT budget_wait_ms FROM trader_sync_summary_batches WHERE id=2`).Scan(&retryWait); e != nil || retryWait != 0 {
				t.Fatal("local delay mislabeled external budget", retryWait, e)
			}
			if miss && nextStart.Sub(frozen) < 2*time.Second {
				t.Fatal("local miss timing was not preserved")
			}
			// Every actual source has entered the real transport. Verify persisted attempts
			// and routes, then inspect the actual shared-budget events for private/Bot limits.
			var kinds int
			waitUntil(func() bool {
				return pool.QueryRow(ctx, `SELECT count(DISTINCT work_kind) FROM notification_delivery_attempts WHERE started_at IS NOT NULL`).Scan(&kinds) == nil && kinds == 3
			}, 2*time.Second)
			seen := map[int64]bool{}
			lateSeen := map[string]bool{}
			var firstActual, nextActual time.Time
			waitUntil(func() bool {
				for len(events) > 0 {
					ev := <-events
					seen[ev.chat] = true
					if ev.text == "system-near-head" || ev.text == "other-near-head" {
						lateSeen[ev.text] = true
					}
					var batch int64
					var part, total int
					_, _ = fmt.Sscanf(ev.text, "摘要 %d · %d/%d", &batch, &part, &total)
					if part == 1 {
						if batch == 1 {
							firstActual = ev.at
						}
						if batch == 2 {
							nextActual = ev.at
						}
					}
				}
				return !firstActual.IsZero() && !nextActual.IsZero() && len(lateSeen) == 2
			}, 2*time.Second)
			if nextActual.Sub(firstActual) < time.Minute {
				t.Fatal("actual monotonic first-start spacing", nextActual.Sub(firstActual))
			}
			var ordinaryWaiting int
			if e = pool.QueryRow(ctx, `SELECT count(*) FROM account_notification_deliveries WHERE account_id=$1 AND idempotency_key='same-owner-ordinary-near-head' AND status='pending'`, owner).Scan(&ordinaryWaiting); e != nil || ordinaryWaiting != 1 {
				t.Fatal("same-owner ordinary was not queued behind reserved head", ordinaryWaiting, e)
			}
			cancel()
			select {
			case e := <-done:
				done <- e
				if e != nil && !errors.Is(e, context.Canceled) {
					t.Fatal(e)
				}
			case <-time.After(6 * time.Second):
				t.Fatal("dispatcher cancellation did not join")
			}
			t.Logf("actual start interval monotonic=%s UTC=%s", nextActual.Sub(firstActual), time.Duration(nextActual.UnixNano()-firstActual.UnixNano()))
			for _, chat := range []int64{123, 456, 789, -777} {
				if !seen[chat] {
					t.Fatal("actual source route never started", chat)
				}
			}
			budget.mu.Lock()
			history := append([]budgetEvent(nil), budget.events...)
			reservations := len(budget.reservations)
			budget.mu.Unlock()
			for i, a := range history {
				bot := 0
				for j, b := range history {
					if b.at.After(a.at.Add(-time.Second)) && !b.at.After(a.at) {
						bot++
					}
					if j < i && b.chat == a.chat && a.chat > 0 && a.at.Sub(b.at) < time.Second {
						t.Fatal("private budget spacing", a, b)
					}
				}
				if bot > 20 {
					t.Fatal("Bot rate exceeded", bot)
				}
			}
			if reservations != 0 {
				t.Fatal("reservation leak", reservations)
			}
			t.Logf("parts=%d old_pending=%d members=%d interval=%s misses=%d local_frozen_to_start=%s routes=%v", parts, waitingOld, totalMembers, nextStart.Sub(firstStart), misses, nextStart.Sub(frozen), seen)
		})
	}
}
