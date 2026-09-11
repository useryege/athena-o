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
	"google.golang.org/grpc/health/grpc_health_v1"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRecoverSenderRequiresExplicitStoppedInstance(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	s := notificationstore.NewSQLStore(db.Pool)
	ctx := context.Background()
	id := uuid.New()
	session, err := s.AcquireSender(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AcquireSender(ctx, uuid.New()); err == nil {
		t.Fatal("parallel sender acquired")
	}
	if err = s.RecoverSender(ctx, id); err == nil {
		t.Fatal("live sender recovered")
	}
	session.Close()
	if _, err = s.AcquireSender(ctx, uuid.New()); err == nil {
		t.Fatal("lost lease became stop proof")
	}
	if err = s.ConfirmStoppedSender(ctx, id); err != nil {
		t.Fatal(err)
	}
	next, err := s.AcquireSender(ctx, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	if next.FirstStart() {
		t.Fatal("existing registry incorrectly got first-start exemption")
	}
	if err = next.Finish(ctx); err != nil {
		t.Fatal(err)
	}
}
func TestRecoverSenderUnknownAndHistoricalBudget(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	s := notificationstore.NewSQLStore(db.Pool)
	ctx := context.Background()
	inc := uuid.New()
	session, err := s.AcquireSender(ctx, inc)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','recover',1)`)
	if err != nil {
		t.Fatal(err)
	}
	var id int64
	if err = db.Pool.QueryRow(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest) VALUES('test','info','body','telegram','pending','test','recover',convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8'),sha256(convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8'))) RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	c := delivery.Candidate{Ref: delivery.WorkRef{Kind: "system", ID: id}, ChatID: -123, Group: true}
	p, err := s.Authorize(ctx, c, inc, nil)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Now().Add(-2 * time.Second)
	if err = s.RecordStarted(ctx, p, at); err != nil {
		t.Fatal(err)
	}
	if err = s.RecordOutcome(ctx, p, delivery.Outcome{Kind: "retryable"}, at); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Authorize(ctx, c, inc, nil); err != nil {
		t.Fatal(err)
	}
	session.Close()
	if err = s.ConfirmStoppedSender(ctx, inc); err != nil {
		t.Fatal(err)
	}
	var state string
	var count int
	var missing bool
	if err = db.Pool.QueryRow(ctx, `SELECT d.status,d.attempts,a.started_at IS NULL FROM system_notification_deliveries d JOIN notification_delivery_attempts a ON a.id=d.current_attempt_id WHERE d.id=$1`, id).Scan(&state, &count, &missing); err != nil {
		t.Fatal(err)
	}
	if state != "unknown" || count != 2 || !missing {
		t.Fatalf("recovery changed facts: %s %d %v", state, count, missing)
	}
	b := NewBudget(20, time.Second, 20, time.Minute)
	if err = restoreBudget(ctx, s, b, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := b.Next(c, time.Now()); got.Before(time.Now().Add(59 * time.Second)) {
		t.Fatal("unknown start lost conservative window", got)
	}
	evidence, err := s.BudgetEvidence(ctx, time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence) != 2 {
		t.Fatalf("historical attempt lost: %d", len(evidence))
	}
}

type recoveryProfile struct{}

func (recoveryProfile) SyncProfile(context.Context) (*utiltelegram.BotIdentity, error) {
	return &utiltelegram.BotIdentity{ID: 1, Username: "testbot"}, nil
}
func TestRecoverSenderLossStopsPollerAndSignalsFatal(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	store := notificationstore.NewSQLStore(db.Pool)
	probe := &pollingProbe{requests: make(chan utiltelegram.PollUpdatesRequest, 4), updates: make(chan []utiltelegram.Update)}
	server, err := NewServer(ServerOpts{SiteURL: "https://athena.test", Store: store, Sender: NewTelegramSender(probe, nil), ProfileSyncer: recoveryProfile{}, Poller: NewTelegramPoller(store, probe), InternalAuthToken: strings.Repeat("a", 32)})
	if err != nil {
		t.Fatal(err)
	}
	if err = server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	nextPoll(t, probe)
	if _, err = db.Pool.Exec(context.Background(), `SELECT pg_terminate_backend(pid) FROM pg_locks WHERE locktype='advisory' AND objid=1096042561`); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-server.Errors():
		if err == nil {
			t.Fatal("missing fatal error")
		}
	case <-time.After(4 * time.Second):
		t.Fatal("lost session did not stop service")
	}
	health, err := server.healthService.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil || health.Status != grpc_health_v1.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("health %v %v", health, err)
	}
	_ = server.Stop()
	var stopped bool
	if err = db.Pool.QueryRow(context.Background(), `SELECT stopped_at IS NOT NULL FROM notification_sender_instances`).Scan(&stopped); err != nil {
		t.Fatal(err)
	}
	if stopped {
		t.Fatal("lease loss became graceful stop confirmation")
	}
	if server.service.poller.Status().Active {
		t.Fatal("poller survived lease loss")
	}
}

func TestRecoverSenderPreservesKnownResultsAcrossAllSources(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	store := notificationstore.NewSQLStore(db.Pool)
	ctx := context.Background()
	inc := uuid.New()
	session, err := store.AcquireSender(ctx, inc)
	if err != nil {
		t.Fatal(err)
	}
	owner := uuid.NewString()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision) VALUES($1,123,123,'test',1);`, owner); err != nil {
		t.Fatal(err)
	}
	var account, system, reply int64
	if err = db.Pool.QueryRow(ctx, `INSERT INTO account_notification_deliveries(account_id,idempotency_key,payload_digest,source,severity,body,channel,status,telegram_chat_id,binding_revision,payload,request_digest) VALUES($1,'recover',sha256(convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8')),'test','info','body','telegram','pending',123,1,convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8'),decode(repeat('ab',32),'hex')) RETURNING id`, owner).Scan(&account); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','recover',1)`); err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest) VALUES('test','info','body','telegram','pending','test','recover',convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8'),sha256(convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8'))) RETURNING id`).Scan(&system); err != nil {
		t.Fatal(err)
	}
	if err = store.ApplyBotUpdate(ctx, utiltelegram.Update{ID: 1, Message: &utiltelegram.Message{ChatType: "private", UserID: 456, ChatID: 456, Text: "/start invalid"}}); err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT id FROM telegram_binding_replies`).Scan(&reply); err != nil {
		t.Fatal(err)
	}
	candidates := []delivery.Candidate{{Ref: delivery.WorkRef{Kind: "account", ID: account}, ChatID: 123}, {Ref: delivery.WorkRef{Kind: "system", ID: system}, ChatID: -123, Group: true}, {Ref: delivery.WorkRef{Kind: "reply", ID: reply}, ChatID: 456}}
	for _, c := range candidates {
		p, err := store.Authorize(ctx, c, inc, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Pool.Exec(ctx, `UPDATE notification_delivery_attempts SET outcome='sent',result_at=clock_timestamp(),message_id='known' WHERE id=$1`, p.AttemptID); err != nil {
			t.Fatal(err)
		}
	}
	session.Close()
	if err = store.ConfirmStoppedSender(ctx, inc); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"account_notification_deliveries", "system_notification_deliveries", "telegram_binding_replies"} {
		var state, message string
		var started bool
		if err = db.Pool.QueryRow(ctx, `SELECT d.status,d.provider_message_id,a.started_at IS NOT NULL FROM `+table+` d JOIN notification_delivery_attempts a ON a.id=d.current_attempt_id`).Scan(&state, &message, &started); err != nil {
			t.Fatal(err)
		}
		if state != "sent" || message != "known" || started {
			t.Fatalf("known receipt rewritten: %s %s %v", state, message, started)
		}
	}
	if err = store.RecoverSender(ctx, inc); err != nil {
		t.Fatal("idempotent recovery", err)
	}
	b := NewBudget(20, time.Second, 20, time.Minute)
	now := time.Now()
	if err = restoreBudget(ctx, store, b, now); err != nil {
		t.Fatal(err)
	}
	if got := b.Next(delivery.Candidate{ChatID: 123}, now); !got.After(now) {
		t.Fatal("missing start ignored known-result upper bound")
	}
}

func TestRecoverSenderRestoresLongRetryAfterAndGracefulUnknown(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	s := notificationstore.NewSQLStore(db.Pool)
	ctx := context.Background()
	inc := uuid.New()
	session, err := s.AcquireSender(ctx, inc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO notification_delivery_attempts(id,work_kind,work_id,sender_incarnation,payload_digest,telegram_chat_id,telegram_group,authorized_at,started_at,result_at,outcome,retry_after) VALUES($1,'system',999,$2,decode(repeat('ab',32),'hex'),-123,true,now()-interval '90 seconds',now()-interval '90 seconds',now()-interval '90 seconds','retryable',interval '120 seconds')`, uuid.New(), inc); err != nil {
		t.Fatal(err)
	}
	if err = session.Finish(ctx); err != nil {
		t.Fatal(err)
	}
	session.Close()
	b := NewBudget(20, time.Second, 20, time.Minute)
	now := time.Now()
	if err = restoreBudget(ctx, s, b, now); err != nil {
		t.Fatal(err)
	}
	if got := b.Next(delivery.Candidate{ChatID: 123}, now); got.Before(now.Add(29 * time.Second)) {
		t.Fatal("restart lost long provider retry-after", got)
	}
}

func TestRecoverSenderDoesNotMistakeKnownReceiptForRepairedDelivery(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	s := notificationstore.NewSQLStore(db.Pool)
	ctx := context.Background()
	inc := uuid.New()
	session, err := s.AcquireSender(ctx, inc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','recover',1)`); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err = db.Pool.QueryRow(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest) VALUES('test','info','body','telegram','pending','test','recover',convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8'),sha256(convert_to(json_build_object('format','html','text','body','messageThreadId',0)::text,'UTF8'))) RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	p, err := s.Authorize(ctx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "system", ID: id}, ChatID: -123, Group: true}, inc, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `UPDATE notification_delivery_attempts SET outcome='sent',result_at=clock_timestamp(),message_id='known' WHERE id=$1`, p.AttemptID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `ALTER TABLE system_notification_deliveries ADD CONSTRAINT reject_repair CHECK(status<>'sent')`); err != nil {
		t.Fatal(err)
	}
	session.Close()
	if err = s.ConfirmStoppedSender(ctx, inc); err == nil {
		t.Fatal("reported successful recovery while delivery CAS failed")
	}
	if _, err = db.Pool.Exec(ctx, `ALTER TABLE system_notification_deliveries DROP CONSTRAINT reject_repair`); err != nil {
		t.Fatal(err)
	}
	if err = s.ConfirmStoppedSender(ctx, inc); err != nil {
		t.Fatal(err)
	}
	var state string
	if err = db.Pool.QueryRow(ctx, `SELECT status FROM system_notification_deliveries WHERE id=$1`, id).Scan(&state); err != nil || state != "sent" {
		t.Fatalf("recovery retry %s %v", state, err)
	}
}

func TestRecoveryBarrierPreservesUnreleasedRetryAfterAcrossUTCJump(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	s := notificationstore.NewSQLStore(db.Pool)
	ctx := context.Background()
	inc := uuid.New()
	session, err := s.AcquireSender(ctx, inc)
	if err != nil {
		t.Fatal(err)
	}
	if !session.FirstStart() {
		t.Fatal("empty registry and attempts must prove first start")
	}
	if err = session.Finish(ctx); err != nil {
		t.Fatal(err)
	}
	session.Close()
	attempt := uuid.New()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO notification_delivery_attempts(id,work_kind,work_id,sender_incarnation,payload_digest,telegram_chat_id,telegram_group,authorized_at,started_at,result_at,outcome,retry_after) VALUES($1,'system',999,$2,decode(repeat('ab',32),'hex'),-123,true,now()-interval '1 day',now()-interval '1 day',now()-interval '1 day','retryable',interval '120 seconds')`, attempt, inc); err != nil {
		t.Fatal(err)
	}
	clock := &manualDispatchClock{now: time.Now().Add(24 * time.Hour)}
	b := NewBudget(20, time.Second, 20, time.Minute)
	done := make(chan error, 1)
	barrierCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { _, err := recoverStartupBudget(barrierCtx, s, b, clock, false); done <- err }()
	waitClockTimer(t, clock)
	clock.advance(60 * time.Second)
	waitClockTimer(t, clock)
	select {
	case err = <-done:
		t.Fatal("UTC jump erased 120-second retry", err)
	default:
	}
	clock.advance(60 * time.Second)
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("barrier did not finish")
	}
	var released bool
	if err = db.Pool.QueryRow(ctx, `SELECT retry_after_released_at IS NOT NULL FROM notification_delivery_attempts WHERE id=$1`, attempt).Scan(&released); err != nil || !released {
		t.Fatalf("monotonic completion not recorded: %v %v", released, err)
	}
	// Released historical retry evidence no longer causes another 120-second cooldown.
	go func() { _, err := recoverStartupBudget(barrierCtx, s, b, clock, false); done <- err }()
	waitClockTimer(t, clock)
	clock.advance(60 * time.Second)
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("released history repeated 120-second wait")
	}
}
func TestRecoveryBarrierFirstStartAndCancellation(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	s := notificationstore.NewSQLStore(db.Pool)
	clock := &manualDispatchClock{now: time.Now()}
	b := NewBudget(20, time.Second, 20, time.Minute)
	if _, err := recoverStartupBudget(context.Background(), s, b, clock, true); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := recoverStartupBudget(ctx, s, b, clock, false); done <- err }()
	waitClockTimer(t, clock)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancel leaked startup barrier")
	}
}
func waitClockTimer(t *testing.T, c *manualDispatchClock) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		c.mu.Lock()
		waiting := len(c.waiters) > 0
		c.mu.Unlock()
		if waiting {
			return
		}
		select {
		case <-deadline:
			t.Fatal("clock timer not registered")
		default:
			runtime.Gosched()
		}
	}
}

func TestRetryBudgetReleasesOnlyAfterMonotonicWait(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	s := notificationstore.NewSQLStore(db.Pool)
	ctx := context.Background()
	id := uuid.New()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO notification_delivery_attempts(id,work_kind,work_id,sender_incarnation,payload_digest,telegram_chat_id,telegram_group,result_at,outcome,retry_after) VALUES($1,'system',999,$2,decode(repeat('ab',32),'hex'),-123,true,now()-interval '1 day','retryable',interval '1 second')`, id, uuid.New()); err != nil {
		t.Fatal(err)
	}
	clock := &manualDispatchClock{now: time.Now()}
	r := &retryBudget{store: s, budget: NewBudget(20, time.Second, 20, time.Minute), clock: clock, pending: map[uuid.UUID]time.Time{}, wake: make(chan struct{}, 1)}
	r.track(id, time.Second)
	clock.advance(500 * time.Millisecond)
	if err := r.releaseElapsed(ctx); err != nil {
		t.Fatal(err)
	}
	var released bool
	if err := db.Pool.QueryRow(ctx, `SELECT retry_after_released_at IS NOT NULL FROM notification_delivery_attempts WHERE id=$1`, id).Scan(&released); err != nil || released {
		t.Fatalf("UTC age prematurely released monotonic budget: %v/%v", released, err)
	}
	clock.advance(500 * time.Millisecond)
	if err := r.releaseElapsed(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.Pool.QueryRow(ctx, `SELECT retry_after_released_at IS NOT NULL FROM notification_delivery_attempts WHERE id=$1`, id).Scan(&released); err != nil || !released {
		t.Fatalf("elapsed budget not released: %v/%v", released, err)
	}
	session, err := s.AcquireSender(ctx, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if session.FirstStart() {
		t.Fatal("attempt history incorrectly got first-start exemption")
	}
}
