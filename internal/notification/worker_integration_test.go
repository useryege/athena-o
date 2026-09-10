//go:build integration

package notification

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/notification/delivery"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/internal/testutil/pgtest"
	utiltelegram "github.com/useryege/athena/util/telegram"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPersistsPermitBeforeSendAndRetriesOnlyResult(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','worker',1)`); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.Pool.QueryRow(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label) VALUES('test','info','hello','telegram','pending','test','worker') RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `ALTER TABLE system_notification_deliveries ADD CONSTRAINT reject_sent CHECK(status <> 'sent')`); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	repairDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		var state string
		var attempts int
		if err := db.Pool.QueryRow(ctx, `SELECT status,attempts FROM system_notification_deliveries WHERE id=$1`, id).Scan(&state, &attempts); err != nil {
			t.Error(err)
		}
		if state != "sending" || attempts != 1 {
			t.Errorf("HTTP without durable permit: %s %d", state, attempts)
		}
		_, _ = io.WriteString(w, `{"ok":true,"result":{"message_id":42}}`)
		go func() {
			defer close(repairDone)
			time.Sleep(200 * time.Millisecond)
			if _, err := db.Pool.Exec(ctx, `ALTER TABLE system_notification_deliveries DROP CONSTRAINT reject_sent`); err != nil {
				t.Error(err)
			}
		}()
	}))
	defer server.Close()
	client, err := utiltelegram.NewClient(utiltelegram.Config{BotToken: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	store := notificationstore.NewSQLStore(db.Pool)
	service := NewService(store, NewTelegramSender(client, map[string]string{"test": "-123"}), nil, nil)
	claimed, err := store.ClaimPendingSystemNotificationDeliveries(ctx, notificationstore.ClaimDeliveriesOptions{Limit: 1, LockedBy: "test"})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim %v %v", claimed, err)
	}
	dispatchTestSystem(t, ctx, service, claimed[0].ID)
	<-repairDone
	item, err := store.GetSystemNotificationDelivery(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != "sent" || item.ProviderMessageID != "42" || calls.Load() != 1 {
		t.Fatalf("result %#v calls %d", item, calls.Load())
	}
	var started bool
	if err := db.Pool.QueryRow(ctx, `SELECT started_at IS NOT NULL FROM notification_delivery_attempts WHERE work_kind='system' AND work_id=$1`, id).Scan(&started); err != nil || !started {
		t.Fatalf("started evidence %v %v", started, err)
	}
}

func TestWorkerDoesNotResendUnknownOrCrashedPermit(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','worker',1)`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := db.Pool.Exec(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label) VALUES('test','info','hello','telegram','pending','test','worker')`); err != nil {
			t.Fatal(err)
		}
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = conn.Close()
	}))
	defer server.Close()
	client, err := utiltelegram.NewClient(utiltelegram.Config{BotToken: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	store := notificationstore.NewSQLStore(db.Pool)
	service := NewService(store, NewTelegramSender(client, map[string]string{"test": "-123"}), nil, nil)
	rows, err := store.ClaimPendingSystemNotificationDeliveries(ctx, notificationstore.ClaimDeliveriesOptions{Limit: 2, LockedBy: "before-crash"})
	if err != nil || len(rows) != 2 {
		t.Fatalf("claims %v %v", rows, err)
	}
	dispatchTestSystem(t, ctx, service, rows[0].ID)
	if _, err = store.Authorize(ctx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "system", ID: rows[1].ID}, ChatID: -123, Group: true}, uuid.New(), nil); err != nil {
		t.Fatal(err)
	}
	// A new store and worker have no memory of the first sender. Even expired claim locks cannot revive these rows.
	if _, err = db.Pool.Exec(ctx, `UPDATE system_notification_deliveries SET locked_at=now()-interval '1 hour'`); err != nil {
		t.Fatal(err)
	}
	restartedStore := notificationstore.NewSQLStore(db.Pool)
	restarted := NewService(restartedStore, NewTelegramSender(client, map[string]string{"test": "-123"}), nil, nil)
	if work, err := restarted.store.DispatchItems(ctx, "system"); err != nil || len(work) != 0 {
		t.Fatalf("terminal or consumed permit was reclaimed: %#v", work)
	}
	unknown, err := store.GetSystemNotificationDelivery(ctx, rows[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	crashed, err := store.GetSystemNotificationDelivery(ctx, rows[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if unknown.Status != "unknown" || crashed.Status != "sending" || crashed.StartedAt != "" || calls.Load() != 1 {
		t.Fatalf("states %s/%s start %q calls %d", unknown.Status, crashed.Status, crashed.StartedAt, calls.Load())
	}
}

func TestWorkerConsumesBindingReplyThroughPermit(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	store := notificationstore.NewSQLStore(db.Pool)
	if err := store.ApplyBotUpdate(ctx, utiltelegram.Update{ID: 42, Message: &utiltelegram.Message{ChatType: "private", UserID: 123, ChatID: 123, Text: "/start invalid"}}); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		var state, kind string
		var attempts int
		if err := db.Pool.QueryRow(ctx, `SELECT r.status,r.attempts,a.work_kind FROM telegram_binding_replies r JOIN notification_delivery_attempts a ON a.id=r.current_attempt_id`).Scan(&state, &attempts, &kind); err != nil {
			t.Error(err)
		}
		if state != "sending" || attempts != 1 || kind != "reply" {
			t.Errorf("reply HTTP without durable permit: %s %d %s", state, attempts, kind)
		}
		_, _ = io.WriteString(w, `{"ok":true,"result":{"message_id":99}}`)
	}))
	defer server.Close()
	client, err := utiltelegram.NewClient(utiltelegram.Config{BotToken: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store, NewTelegramSender(client, nil), nil, nil)
	// The existing dispatch consumer must actually discover reply work.
	claimed, err := service.store.DispatchItems(ctx, "reply")
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 {
		t.Fatalf("worker must claim binding reply, got %d", len(claimed))
	}

	// Release the probe's scheduling lease so the live worker can pick it up.
	if _, err := db.Pool.Exec(ctx, `UPDATE telegram_binding_replies SET locked_at=NULL`); err != nil {
		t.Fatal(err)
	}
	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { defer close(done); service.runWorker(workerCtx) }()
	defer func() { cancel(); <-done }()
	deadline := time.After(3 * time.Second)
	for {
		var state string
		var started bool
		if err := db.Pool.QueryRow(ctx, `SELECT r.status,COALESCE(a.started_at IS NOT NULL,false) FROM telegram_binding_replies r LEFT JOIN notification_delivery_attempts a ON a.id=r.current_attempt_id`).Scan(&state, &started); err != nil {
			t.Fatal(err)
		}
		if state == "sent" {
			if calls.Load() != 1 || !started {
				t.Fatalf("calls/start %d/%v", calls.Load(), started)
			}
			break
		}
		select {
		case <-deadline:
			t.Fatal("worker never sent reply")
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func dispatchTestSystem(t *testing.T, ctx context.Context, s *Service, id int64) {
	t.Helper()
	source := s.workSources()[1]
	items, err := source.Ready(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range items {
		if c.Ref.ID == id {
			if err = source.Dispatch(ctx, c, nil); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatal("system candidate missing")
}

type concurrentSendProbe struct {
	started chan SendRequest
	release chan struct{}
}

func (p *concurrentSendProbe) CreateSystemTopic(context.Context, string, string) (int, error) {
	return 1, nil
}
func (p *concurrentSendProbe) SystemChatID(string) (int64, error) { return -777, nil }
func (p *concurrentSendProbe) Send(ctx context.Context, r SendRequest, started func(time.Time)) delivery.Outcome {
	started(time.Now())
	p.started <- r
	select {
	case <-ctx.Done():
		return delivery.Outcome{Kind: "unknown", Code: "cancelled"}
	case <-p.release:
		return delivery.Outcome{Kind: "sent", MessageID: "55"}
	}
}
func TestDispatcherConsumesAllSourcesWithDurablePhysicalRoutes(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	store := notificationstore.NewSQLStore(db.Pool)
	for i := int64(1); i <= 10; i++ {
		owner := uuid.NewString()
		if _, err := db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision) VALUES($1,$2,$2,'test',1)`, owner, i); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Pool.Exec(ctx, `INSERT INTO account_notification_deliveries(account_id,idempotency_key,payload_digest,source,severity,body,channel,status,telegram_chat_id,binding_revision) VALUES($1,'key',decode(repeat('ab',32),'hex'),'test','info','body','telegram','pending',$2,1)`, owner, i); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','mixed',1); INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label) VALUES('test','info','body','telegram','pending','test','mixed')`); err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyBotUpdate(ctx, utiltelegram.Update{ID: 42, Message: &utiltelegram.Message{ChatType: "private", UserID: 123, ChatID: 123, Text: "/start invalid"}}); err != nil {
		t.Fatal(err)
	}
	probe := &concurrentSendProbe{started: make(chan SendRequest, 12), release: make(chan struct{})}
	service := NewService(store, probe, nil, nil)
	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- service.runWorker(workerCtx) }()
	defer func() { cancel(); <-done }()
	seen := map[int64]bool{}
	for i := 0; i < 12; i++ {
		select {
		case req := <-probe.started:
			seen[req.TelegramChatID] = true
		case <-time.After(3 * time.Second):
			t.Fatal("sources did not run concurrently")
		}
	}
	if len(seen) != 12 || !seen[-777] || !seen[123] {
		t.Fatal("missing physical routes", seen)
	}
	var total, accounts, systems, replies int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE work_kind='account' AND telegram_chat_id BETWEEN 1 AND 10 AND NOT telegram_group),count(*) FILTER(WHERE work_kind='system' AND telegram_chat_id=-777 AND telegram_group),count(*) FILTER(WHERE work_kind='reply' AND telegram_chat_id=123 AND NOT telegram_group) FROM notification_delivery_attempts`).Scan(&total, &accounts, &systems, &replies); err != nil {
		t.Fatal(err)
	}
	if total != 12 || accounts != 10 || systems != 1 || replies != 1 {
		t.Fatalf("permit routes do not match HTTP routes: %d/%d/%d/%d", total, accounts, systems, replies)
	}
}

func TestDispatchExpiredSlotAfterAccountGateLeavesNoAttempt(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	store := notificationstore.NewSQLStore(db.Pool)
	owner := uuid.NewString()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision) VALUES($1,123,123,'test',1)`, owner); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.Pool.QueryRow(ctx, `INSERT INTO account_notification_deliveries(account_id,idempotency_key,payload_digest,source,severity,body,channel,status,telegram_chat_id,binding_revision) VALUES($1,'key',decode(repeat('ab',32),'hex'),'test','info','body','telegram','pending',123,1) RETURNING id`, owner).Scan(&id); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('athena:account:' || $1::text,0))`, owner); err != nil {
		t.Fatal(err)
	}
	clock := &manualDispatchClock{now: time.Now()}
	slotCtx := context.WithValue(ctx, dispatchSlotKey{}, dispatchSlot{clock: clock, until: clock.Now().Add(time.Second)})
	probe := &concurrentSendProbe{started: make(chan SendRequest, 1), release: make(chan struct{})}
	service := NewService(store, probe, nil, nil)
	result := make(chan error, 1)
	go func() {
		_, err := service.sendPermittedNotification(slotCtx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: id}, ChatID: 123}, SendRequest{TelegramChatID: 123, Text: "body"}, nil)
		result <- err
	}()
	for {
		var waiting int
		if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event='advisory'`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			break
		}
		runtime.Gosched()
	}
	clock.advance(2 * time.Second)
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-result; !errors.Is(err, ErrDispatchDeferred) {
		t.Fatalf("expired slot got permit: %v", err)
	}
	var count int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM notification_delivery_attempts`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 || len(probe.started) != 0 {
		t.Fatal("expired slot consumed attempt or HTTP")
	}
}
