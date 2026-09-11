//go:build integration

package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	if err := db.Pool.QueryRow(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest) VALUES('test','info','hello','telegram','pending','test','worker',convert_to(json_build_object('format','html','text','hello','messageThreadId',0)::text,'UTF8'),sha256(convert_to(json_build_object('format','html','text','hello','messageThreadId',0)::text,'UTF8'))) RETURNING id`).Scan(&id); err != nil {
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
		if _, err := db.Pool.Exec(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest) VALUES('test','info','hello','telegram','pending','test','worker',convert_to(json_build_object('format','html','text','hello','messageThreadId',0)::text,'UTF8'),sha256(convert_to(json_build_object('format','html','text','hello','messageThreadId',0)::text,'UTF8')))`); err != nil {
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
		if _, err := db.Pool.Exec(ctx, `INSERT INTO account_notification_deliveries(account_id,idempotency_key,payload_digest,source,severity,body,channel,status,telegram_chat_id,binding_revision,payload,request_digest) VALUES($1,'key',sha256(convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8')),'test','info','body','telegram','pending',$2,1,convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8'),decode(repeat('ab',32),'hex'))`, owner, i); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','mixed',1); INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest) VALUES('test','info','body','telegram','pending','test','mixed',convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8'),sha256(convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8')))`); err != nil {
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
	if err := db.Pool.QueryRow(ctx, `INSERT INTO account_notification_deliveries(account_id,idempotency_key,payload_digest,source,severity,body,channel,status,telegram_chat_id,binding_revision,payload,request_digest) VALUES($1,'key',sha256(convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8')),'test','info','body','telegram','pending',123,1,convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8'),decode(repeat('ab',32),'hex')) RETURNING id`, owner).Scan(&id); err != nil {
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
		_, err := service.sendPermittedNotification(slotCtx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: id}, ChatID: 123}, nil)
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

type observedDispatchSource struct {
	WorkSource
	results chan error
}

func (s *observedDispatchSource) Dispatch(ctx context.Context, c delivery.Candidate, onStarted func(time.Time)) error {
	err := s.WorkSource.Dispatch(ctx, c, onStarted)
	s.results <- err
	return err
}
func TestDispatchRetryAfterInvalidatesHeldValidSlotAndReschedules(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	store := notificationstore.NewSQLStore(db.Pool)
	owner := uuid.NewString()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision) VALUES($1,123,123,'test',1)`, owner); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.Pool.QueryRow(ctx, `INSERT INTO account_notification_deliveries(account_id,idempotency_key,payload_digest,source,severity,body,channel,status,telegram_chat_id,binding_revision,payload,request_digest) VALUES($1,'key',sha256(convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8')),'test','info','body','telegram','pending',123,1,convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8'),decode(repeat('ab',32),'hex')) RETURNING id`, owner).Scan(&id); err != nil {
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
	at := clock.Now()
	budget := NewBudget(20, time.Second, 20, time.Minute)
	probe := &concurrentSendProbe{started: make(chan SendRequest, 1), release: make(chan struct{})}
	close(probe.release)
	service := NewService(store, probe, nil, nil)
	session, err := store.AcquireSender(ctx, service.senderIncarnation)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	service.senderSession = session
	source := &observedDispatchSource{WorkSource: service.workSources()[0], results: make(chan error, 4)}
	finished := make(chan error, 1)
	go func() { finished <- NewDispatcher(clock, []WorkSource{source}, budget, 12).Run(ctx) }()
	defer func() { cancel(); <-finished }()
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
	// Another sender receives 429 while this unpermitted delivery still holds a fresh slot.
	tracker := &retryBudget{store: store, budget: budget, clock: clock, pending: map[uuid.UUID]time.Time{}, wake: make(chan struct{}, 1)}
	tracker.track(uuid.New(), 120*time.Second)
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-source.results:
		if !errors.Is(err, ErrDispatchDeferred) {
			t.Fatalf("fresh slot bypassed later Retry-After: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("waiting source did not defer")
	}
	if !clock.Now().Equal(at) {
		t.Fatal("test must keep original slot unexpired")
	}
	var attempts int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM notification_delivery_attempts`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 0 || len(probe.started) != 0 {
		t.Fatalf("cooldown consumed attempt/HTTP: %d/%d", attempts, len(probe.started))
	}
	clock.advance(120 * time.Second)
	select {
	case req := <-probe.started:
		if req.TelegramChatID != 123 {
			t.Fatal(req)
		}
	case <-ctx.Done():
		t.Fatal("delivery did not reschedule after cooldown")
	}
	select {
	case err = <-source.results:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("rescheduled delivery did not finish")
	}
	var state string
	if err = db.Pool.QueryRow(ctx, `SELECT status,attempts FROM account_notification_deliveries WHERE id=$1`, id).Scan(&state, &attempts); err != nil {
		t.Fatal(err)
	}
	if state != "sent" || attempts != 1 {
		t.Fatalf("requeue consumed attempt before permission: %s/%d", state, attempts)
	}
}

func TestDispatchCancellationAfterGuardReleasesTightenWriter(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	store := notificationstore.NewSQLStore(db.Pool)
	owner := uuid.NewString()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision) VALUES($1,123,123,'test',1)`, owner); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.Pool.QueryRow(ctx, `INSERT INTO account_notification_deliveries(account_id,idempotency_key,payload_digest,source,severity,body,channel,status,telegram_chat_id,binding_revision,payload,request_digest) VALUES($1,'key',sha256(convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8')),'test','info','body','telegram','pending',123,1,convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8'),decode(repeat('ab',32),'hex')) RETURNING id`, owner).Scan(&id); err != nil {
		t.Fatal(err)
	}
	// Block the attempt INSERT after the real post-account-gate authorization guard.
	if _, err := db.Pool.Exec(ctx, `CREATE FUNCTION block_test_attempt() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(987654321); RETURN NEW; END $$; CREATE TRIGGER block_test_attempt BEFORE INSERT ON notification_delivery_attempts FOR EACH ROW EXECUTE FUNCTION block_test_attempt()`); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(987654321)`); err != nil {
		t.Fatal(err)
	}
	clock := &manualDispatchClock{now: time.Now()}
	budget := NewBudget(20, time.Second, 20, time.Minute)
	c := delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: id}, OwnerID: owner, ChatID: 123}
	reservation, ok := budget.Reserve(c, clock.Now())
	if !ok {
		t.Fatal("reserve failed")
	}
	defer budget.Release(reservation)
	probe := &concurrentSendProbe{started: make(chan SendRequest, 1), release: make(chan struct{})}
	service := NewService(store, probe, nil, nil)
	dispatchCtx, stop := context.WithCancel(ctx)
	defer stop()
	dispatchCtx = context.WithValue(dispatchCtx, dispatchSlotKey{}, dispatchSlot{clock: clock, until: clock.Now().Add(time.Second), budget: budget, reservation: reservation})
	result := make(chan error, 1)
	go func() {
		_, err := service.sendPermittedNotification(dispatchCtx, c, nil)
		result <- err
	}()
	for {
		var waiting int
		if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event='advisory' AND query LIKE '%INSERT INTO notification_delivery_attempts%'`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			break
		}
		runtime.Gosched()
	}
	tightened := make(chan struct{})
	go func() { budget.Tighten(clock.Now().Add(120 * time.Second)); close(tightened) }()
	waitBudgetWriter(t, budget)
	stop()
	select {
	case err = <-result:
		if err == nil {
			t.Fatal("cancelled attempt insert acquired permit")
		}
	case <-ctx.Done():
		t.Fatal("cancelled authorization did not return")
	}
	select {
	case <-tightened:
	case <-ctx.Done():
		t.Fatal("cancelled authorization retained read lock and blocked Tighten")
	}
	var count int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM notification_delivery_attempts`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 || len(probe.started) != 0 {
		t.Fatal("cancelled post-guard transaction consumed attempt/HTTP")
	}
}

// Pauses the real client after permit commit, before its HTTP transport is entered.
type pausedMessageClient struct {
	utiltelegram.Client
	entered chan struct{}
	resume  chan struct{}
}

func (c *pausedMessageClient) SendMessage(ctx context.Context, r utiltelegram.SendMessageRequest) (*utiltelegram.SendMessageResponse, error) {
	close(c.entered)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.resume:
	}
	return c.Client.SendMessage(ctx, r)
}

func TestDispatchCommittedPermitWaitsForLaterRetryAfterAtHTTPEntry(t *testing.T) {
	for _, cancelWait := range []bool{false, true} {
		t.Run(fmt.Sprint("cancel=", cancelWait), func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			store := notificationstore.NewSQLStore(db.Pool)
			owner := uuid.NewString()
			if _, err := db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision) VALUES($1,123,123,'test',1)`, owner); err != nil {
				t.Fatal(err)
			}
			var id int64
			if err := db.Pool.QueryRow(ctx, `INSERT INTO account_notification_deliveries(account_id,idempotency_key,payload_digest,source,severity,body,channel,status,telegram_chat_id,binding_revision,payload,request_digest) VALUES($1,'key',sha256(convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8')),'test','info','body','telegram','pending',123,1,convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8'),decode(repeat('ab',32),'hex')) RETURNING id`, owner).Scan(&id); err != nil {
				t.Fatal(err)
			}
			var calls atomic.Int32
			httpEntered := make(chan struct{})
			httpRelease := make(chan struct{})
			defer func() {
				select {
				case <-httpRelease:
				default:
					close(httpRelease)
				}
			}()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				close(httpEntered)
				select {
				case <-httpRelease:
				case <-r.Context().Done():
				}
				_, _ = io.Copy(io.Discard, r.Body)
				_, _ = io.WriteString(w, `{"ok":true,"result":{"message_id":42}}`)
			}))
			defer server.Close()
			raw, err := utiltelegram.NewClient(utiltelegram.Config{BotToken: "test", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			client := &pausedMessageClient{Client: raw, entered: make(chan struct{}), resume: make(chan struct{})}
			service := NewService(store, NewTelegramSender(client, nil), nil, nil)
			clock := &manualDispatchClock{now: time.Now()}
			budget := NewBudget(20, time.Second, 20, time.Minute)
			c := delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: id}, OwnerID: owner, ChatID: 123}
			reservation, ok := budget.Reserve(c, clock.Now())
			if !ok {
				t.Fatal("reserve failed")
			}
			defer budget.Release(reservation)
			dispatchCtx, stop := context.WithCancel(ctx)
			defer stop()
			dispatchCtx = context.WithValue(dispatchCtx, dispatchSlotKey{}, dispatchSlot{clock: clock, until: clock.Now().Add(time.Second), budget: budget, reservation: reservation})
			done := make(chan delivery.Outcome, 1)
			go func() {
				out, err := service.sendPermittedNotification(dispatchCtx, c, func(time.Time) { budget.Start(c, clock.Now()) })
				if err != nil {
					t.Error(err)
				}
				done <- out
			}()
			select {
			case <-client.entered:
			case <-ctx.Done():
				t.Fatal("permit not committed")
			}
			assertAttempt := func(wantStarted bool) {
				t.Helper()
				var attempts, starts int
				if err := db.Pool.QueryRow(ctx, `SELECT count(*),count(started_at) FROM notification_delivery_attempts`).Scan(&attempts, &starts); err != nil {
					t.Fatal(err)
				}
				if attempts != 1 || (starts > 0) != wantStarted {
					t.Fatalf("attempts=%d starts=%d", attempts, starts)
				}
			}
			assertAttempt(false)
			budget.Tighten(clock.Now().Add(120 * time.Second))
			close(client.resume)
			for {
				clock.mu.Lock()
				waiting := len(clock.waiters) > 0
				clock.mu.Unlock()
				if waiting {
					break
				}
				select {
				case out := <-done:
					t.Fatalf("committed permit bypassed later cooldown: %#v HTTP=%d", out, calls.Load())
				case <-ctx.Done():
					t.Fatal("pre-start gate did not wait")
				default:
					runtime.Gosched()
				}
			}
			if calls.Load() != 0 {
				t.Fatal("HTTP entered during cooldown")
			}
			assertAttempt(false)
			// A fresh account gate can be acquired while this committed attempt waits locally.
			tx, err := db.Pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var locked bool
			if err = tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended('athena:account:' || $1::text,0))`, owner).Scan(&locked); err != nil {
				t.Fatal(err)
			}
			_ = tx.Rollback(ctx)
			if !locked {
				t.Fatal("budget wait retained account gate")
			}
			if cancelWait {
				stop()
			} else {
				clock.advance(120 * time.Second)
				select {
				case <-httpEntered:
				case <-ctx.Done():
					t.Fatal("cooled attempt did not enter HTTP")
				}
				tightened := make(chan struct{})
				go func() { budget.Tighten(clock.Now().Add(120 * time.Second)); close(tightened) }()
				select {
				case <-tightened:
				case <-ctx.Done():
					t.Fatal("Tighten waited for HTTP receipt")
				}
				close(httpRelease)
			}
			select {
			case out := <-done:
				if cancelWait {
					if out.Kind != "failed" || out.Code != "not_started" || calls.Load() != 0 {
						t.Fatalf("cancelled wait: %#v HTTP=%d", out, calls.Load())
					}
				} else if out.Kind != "sent" || calls.Load() != 1 {
					t.Fatalf("resumed same permit: %#v HTTP=%d", out, calls.Load())
				}
			case <-ctx.Done():
				t.Fatal("pre-start wait did not finish")
			}
			assertAttempt(!cancelWait)
			// Neither successful nor cancelled admission may retain a read lock and block shutdown/results.
			tightened := make(chan struct{})
			go func() { budget.Tighten(clock.Now().Add(240 * time.Second)); close(tightened) }()
			select {
			case <-tightened:
			case <-ctx.Done():
				t.Fatal("admission leaked lock")
			}
		})
	}
}

func TestWorkerSendsFrozenPlainBytesAndAttemptDigest(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner := uuid.NewString()
	text := "  <鲸>&🙂\nhttps://athena.test/a?x=1&y=2  "
	payload, e := delivery.EncodePayload(delivery.Payload{Format: "plain", Text: text})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, owner); e != nil {
		t.Fatal(e)
	}
	var id int64
	if e = db.Pool.QueryRow(ctx, `INSERT INTO account_notification_deliveries(account_id,idempotency_key,payload_digest,source,severity,body,channel,status,telegram_chat_id,binding_revision,payload,request_digest)VALUES($1,'frozen',sha256($2::bytea),'test','info','changed display body','telegram','pending',123,1,$2,sha256($2::bytea))RETURNING id`, owner, payload).Scan(&id); e != nil {
		t.Fatal(e)
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if e := r.ParseMultipartForm(1 << 20); e != nil {
			t.Error(e)
		}
		if r.Form.Get("text") != text || r.Form.Get("parse_mode") != "" {
			t.Errorf("actual request differs from frozen plain payload: %#v", r.Form)
		}
		var digest []byte
		if e := db.Pool.QueryRow(ctx, `SELECT payload_digest FROM notification_delivery_attempts WHERE work_kind='account' AND work_id=$1`, id).Scan(&digest); e != nil || string(digest) != string(delivery.PayloadDigest(payload)) {
			t.Errorf("attempt digest not actual payload: %x %v", digest, e)
		}
		io.WriteString(w, `{"ok":true,"result":{"message_id":55}}`)
	}))
	defer server.Close()
	client, e := utiltelegram.NewClient(utiltelegram.Config{BotToken: "test", BaseURL: server.URL})
	if e != nil {
		t.Fatal(e)
	}
	s := NewService(notificationstore.NewSQLStore(db.Pool), NewTelegramSender(client, nil), nil, nil)
	source := s.workSources()[0]
	rows, e := source.Ready(ctx, time.Now())
	if e != nil || len(rows) != 1 {
		t.Fatal(rows, e)
	}
	if e = source.Dispatch(ctx, rows[0], nil); e != nil {
		t.Fatal(e)
	}
	if calls.Load() != 1 {
		t.Fatal(calls.Load())
	}
}

// Real Sender/transport plus a database start-write barrier demonstrates that
// local evidence bookkeeping must not be attributed to Telegram ACK latency.
func TestWorkerPersistsSenderReturnBeforeLocalStartBookkeeping(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, e := db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id)VALUES('test','timing',1)`); e != nil {
		t.Fatal(e)
	}
	var id int64
	if e := db.Pool.QueryRow(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest)VALUES('test','info','timing','telegram','pending','test','timing',convert_to('{"format":"html","text":"timing","messageThreadId":0}','UTF8'),sha256(convert_to('{"format":"html","text":"timing","messageThreadId":0}','UTF8'))) RETURNING id`).Scan(&id); e != nil {
		t.Fatal(e)
	}
	held, e := db.Pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer held.Rollback(context.Background())
	if _, e = held.Exec(ctx, `SELECT pg_advisory_xact_lock(135713)`); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `CREATE FUNCTION delay_start_evidence() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(135713); RETURN NEW; END $$; CREATE TRIGGER delay_start BEFORE UPDATE OF started_at ON notification_delivery_attempts FOR EACH ROW EXECUTE FUNCTION delay_start_evidence()`); e != nil {
		t.Fatal(e)
	}
	var calls atomic.Int32
	sendResponse := make(chan struct{})
	responseCtx, stopResponse := context.WithCancel(ctx)
	defer stopResponse()
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-sendResponse:
		case <-responseCtx.Done():
			return
		}
		_, _ = io.WriteString(w, `{"ok":true,"result":{"message_id":17}}`)
	}))
	t.Cleanup(httpServer.Close)
	client, e := utiltelegram.NewClient(utiltelegram.Config{BotToken: "test", BaseURL: httpServer.URL})
	if e != nil {
		t.Fatal(e)
	}
	returned := make(chan time.Time, 1)
	sender := &returnObservedSender{Sender: NewTelegramSender(client, map[string]string{"test": "-123"}), returned: returned}
	st := notificationstore.NewSQLStore(db.Pool)
	svc := NewService(st, sender, nil, nil)
	done := make(chan error, 1)
	go func() {
		_, e := svc.sendPermittedNotification(ctx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "system", ID: id}, ChatID: -123, Group: true}, nil)
		done <- e
	}()
	// A blocked lock request proves the worker selected start before HTTP completed.
	for {
		var blocked bool
		if e = db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype='advisory' AND database=(SELECT oid FROM pg_database WHERE datname=current_database()) AND objid=135713 AND NOT granted)`).Scan(&blocked); e != nil {
			t.Fatal(e)
		}
		if blocked {
			break
		}
		runtime.Gosched()
	}
	close(sendResponse)
	var actualReturn time.Time
	select {
	case actualReturn = <-returned:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	released := time.Now()
	if e = held.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	select {
	case e = <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var raw []byte
	if e = db.Pool.QueryRow(ctx, `SELECT to_jsonb(a) FROM notification_delivery_attempts a WHERE work_kind='system' AND work_id=$1`, id).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	var got struct {
		Returned *time.Time `json:"sender_returned_at"`
		Elapsed  *int64     `json:"sender_elapsed_ns"`
		Result   time.Time  `json:"result_at"`
		Outcome  string     `json:"outcome"`
	}
	if e = json.Unmarshal(raw, &got); e != nil {
		t.Fatal(e)
	}
	if got.Returned == nil || got.Elapsed == nil {
		t.Fatalf("missing actual sender timing: %s", raw)
	}
	if got.Returned.After(released) || got.Returned.Before(actualReturn.UTC().Add(-time.Millisecond)) || got.Result.Before(released) || *got.Elapsed < 0 || got.Outcome != "sent" || calls.Load() != 1 {
		t.Fatalf("wrong ACK/local distinction: %+v actual=%v released=%v calls=%d", got, actualReturn, released, calls.Load())
	}
}

type returnObservedSender struct {
	Sender
	returned chan time.Time
}

func (s *returnObservedSender) Send(ctx context.Context, r SendRequest, start func(time.Time)) delivery.Outcome {
	o := s.Sender.Send(ctx, r, start)
	s.returned <- time.Now()
	return o
}
