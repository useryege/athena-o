//go:build integration

package notification

import (
	"context"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/notification/delivery"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/internal/testutil/pgtest"
	utiltelegram "github.com/useryege/athena/util/telegram"
	"io"
	"net/http"
	"net/http/httptest"
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
	service.processClaimedSystemNotification(ctx, claimed[0])
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
	service.processClaimedSystemNotification(ctx, rows[0])
	if _, err = store.Authorize(ctx, delivery.WorkRef{Kind: "system", ID: rows[1].ID}, uuid.New()); err != nil {
		t.Fatal(err)
	}
	// A new store and worker have no memory of the first sender. Even expired claim locks cannot revive these rows.
	if _, err = db.Pool.Exec(ctx, `UPDATE system_notification_deliveries SET locked_at=now()-interval '1 hour'`); err != nil {
		t.Fatal(err)
	}
	restartedStore := notificationstore.NewSQLStore(db.Pool)
	restarted := NewService(restartedStore, NewTelegramSender(client, map[string]string{"test": "-123"}), nil, nil)
	if work := restarted.claimFairNotificationBatch(ctx, false); len(work) != 0 {
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
	claimed := service.claimFairNotificationBatch(ctx, false)
	if len(claimed) != 1 {
		t.Fatalf("worker must claim binding reply, got %d", len(claimed))
	}
	service.workerConfig.SendInterval = time.Millisecond
	service.workerConfig.PollInterval = time.Millisecond
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
