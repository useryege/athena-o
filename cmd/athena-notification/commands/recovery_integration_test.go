//go:build integration

package commands

import (
	"context"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/notification/delivery"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"strings"
	"testing"
)

func TestRecoveryCommandConfirmsExactInstanceWithoutTelegramConfiguration(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	var name string
	if err := db.Pool.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil || !strings.HasPrefix(name, "athena_test_") {
		t.Fatalf("unexpected test database %q %v", name, err)
	}
	store := notificationstore.NewSQLStore(db.Pool)
	inc := uuid.New()
	session, err := store.AcquireSender(ctx, inc)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','cli',1)`); err != nil {
		t.Fatal(err)
	}
	payload, err := delivery.EncodePayload(delivery.Payload{Format: "html", Text: "body", MessageThreadID: 1})
	if err != nil {
		t.Fatal(err)
	}
	var id int64
	if err = db.Pool.QueryRow(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest) VALUES('test','info','body','telegram','pending','test','cli',$1,$2) RETURNING id`, payload, delivery.PayloadDigest(payload)).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Authorize(ctx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "system", ID: id}, ChatID: -123, Group: true}, inc, nil); err != nil {
		t.Fatal(err)
	}
	session.Close()
	t.Setenv("ATHENA_SERVER_POSTGRES_DSN", db.DSN)
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN", "")
	t.Setenv("ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN", "")
	command := NewCommand()
	command.SetArgs([]string{"--recover-stopped-sender", inc.String()})
	if err = command.ExecuteContext(ctx); err != nil {
		t.Fatal(err)
	}
	var state, confirmation string
	if err = db.Pool.QueryRow(ctx, `SELECT d.status,i.stop_confirmation FROM system_notification_deliveries d JOIN notification_delivery_attempts a ON a.id=d.current_attempt_id JOIN notification_sender_instances i ON i.incarnation=a.sender_incarnation WHERE d.id=$1`, id).Scan(&state, &confirmation); err != nil {
		t.Fatal(err)
	}
	if state != "unknown" || confirmation != "operator" {
		t.Fatalf("command failed to recover exact stopped instance: %s/%s", state, confirmation)
	}
}
