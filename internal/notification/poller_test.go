//go:build integration

package notification

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/internal/testutil/pgtest"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

type pollingProbe struct {
	utiltelegram.Client
	requests chan utiltelegram.PollUpdatesRequest
	updates  chan []utiltelegram.Update
	sends    atomic.Int32
}

func (p *pollingProbe) GetWebhookInfo(context.Context) (*utiltelegram.WebhookInfo, error) {
	return &utiltelegram.WebhookInfo{}, nil
}
func (p *pollingProbe) PollUpdates(ctx context.Context, r utiltelegram.PollUpdatesRequest) ([]utiltelegram.Update, error) {
	select {
	case p.requests <- r:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case u := <-p.updates:
		return u, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (p *pollingProbe) SendMessage(context.Context, utiltelegram.SendMessageRequest) (*utiltelegram.SendMessageResponse, error) {
	p.sends.Add(1)
	return &utiltelegram.SendMessageResponse{MessageID: 1}, nil
}
func nextPoll(t *testing.T, p *pollingProbe) utiltelegram.PollUpdatesRequest {
	t.Helper()
	select {
	case r := <-p.requests:
		return r
	case <-time.After(4 * time.Second):
		t.Fatal("poll timeout")
		return utiltelegram.PollUpdatesRequest{}
	}
}
func TestPollerPersistsReplyWithoutSendingAndRestartsFromOffset(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	s := notificationstore.NewSQLStore(db.Pool)
	token := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	digest := sha256.Sum256([]byte(token))
	owner := uuid.NewString()
	if _, err := s.CreateTelegramBindingAttempt(context.Background(), notificationstore.CreateTelegramBindingAttemptRequest{ID: uuid.NewString(), AccountID: owner, TokenDigest: digest[:], ExpiresAt: time.Now().Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	probe := &pollingProbe{requests: make(chan utiltelegram.PollUpdatesRequest, 4), updates: make(chan []utiltelegram.Update, 1)}
	p := NewTelegramPoller(s, probe)
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()
	if r := nextPoll(t, probe); r.Offset != 0 {
		t.Fatal(r)
	}
	probe.updates <- []utiltelegram.Update{{ID: 42, Message: &utiltelegram.Message{ChatType: "private", UserID: 123, ChatID: 123, Text: "/start " + token}}}
	if r := nextPoll(t, probe); r.Offset != 43 {
		t.Fatal(r)
	}
	p.Stop()
	var replies int
	if err := db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM telegram_binding_replies WHERE status='pending'`).Scan(&replies); err != nil {
		t.Fatal(err)
	}
	if replies != 1 || probe.sends.Load() != 0 {
		t.Fatalf("reply count/direct sends %d/%d", replies, probe.sends.Load())
	}
	restarted := NewTelegramPoller(notificationstore.NewSQLStore(db.Pool), probe)
	if err := restarted.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer restarted.Stop()
	if r := nextPoll(t, probe); r.Offset != 43 {
		t.Fatalf("restart lost offset: %+v", r)
	}
}
