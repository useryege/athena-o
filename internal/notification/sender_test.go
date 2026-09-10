package notification

import (
	"context"
	utiltelegram "github.com/useryege/athena/util/telegram"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestSenderOutcomes(t *testing.T) {
	for _, tt := range []struct {
		name, body, text, kind, code, id string
		lost                             bool
		wait                             time.Duration
		calls                            int32
	}{
		{name: "response lost", text: "hello", kind: "unknown", code: "response_lost", lost: true, calls: 1},
		{name: "rate limited", text: "hello", body: `{"ok":false,"error_code":429,"parameters":{"retry_after":7}}`, kind: "retryable", code: "provider_429", wait: 7 * time.Second, calls: 1},
		{name: "empty rejected locally", text: " ", kind: "failed", code: "not_started"},
		{name: "receipt", text: "hello", body: `{"ok":true,"result":{"message_id":91}}`, kind: "sent", id: "91", calls: 1},
		{name: "unreachable", text: "hello", body: `{"ok":false,"error_code":403}`, kind: "failed", code: "recipient_unreachable", calls: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var calls, started atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				_, _ = io.Copy(io.Discard, r.Body)
				if tt.lost {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					_ = conn.Close()
					return
				}
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()
			client, err := utiltelegram.NewClient(utiltelegram.Config{BotToken: "test", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			sender := NewTelegramSender(client, nil)
			o := sender.Send(context.Background(), SendRequest{TelegramChatID: 123, Text: tt.text}, func(time.Time) { started.Add(1) })
			if o.Kind != tt.kind || o.Code != tt.code || o.MessageID != tt.id || o.RetryAfter != tt.wait || calls.Load() != tt.calls || started.Load() != tt.calls {
				t.Fatalf("outcome %#v calls %d started %d", o, calls.Load(), started.Load())
			}
		})
	}
}
