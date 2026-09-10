package telegram

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestSendTransportOutcomes(t *testing.T) {
	for _, tt := range []struct {
		name, body, text, kind string
		lost                   bool
		wait                   time.Duration
		calls                  int32
	}{
		{name: "accepted response lost", text: "hello", kind: "unknown", lost: true, calls: 1},
		{name: "rate limited", body: `{"ok":false,"error_code":429,"parameters":{"retry_after":7}}`, text: "hello", kind: "retryable", wait: 7 * time.Second, calls: 1},
		{name: "local validation", text: " ", kind: "failed", calls: 0},
		{name: "malformed response", body: `{"ok":`, text: "hello", kind: "unknown", calls: 1},
		{name: "provider unavailable", body: `{"ok":false,"error_code":503}`, text: "hello", kind: "retryable", calls: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var calls, starts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				_, _ = io.Copy(io.Discard, r.Body)
				if starts.Load() != 1 {
					t.Error("HTTP handler reached before started callback")
				}
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
			client, err := NewClient(Config{BotToken: "test", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.SendMessage(WithSendStarted(context.Background(), func(at time.Time) {
				if at.IsZero() {
					t.Error("zero start")
				}
				starts.Add(1)
			}), SendMessageRequest{ChatID: "123", Text: tt.text})
			var sendErr *SendError
			if !errors.As(err, &sendErr) || sendErr.Kind != tt.kind || sendErr.RetryAfter != tt.wait {
				t.Fatalf("outcome: %#v / %v", sendErr, err)
			}
			if calls.Load() != tt.calls || starts.Load() != tt.calls {
				t.Fatalf("calls %d starts %d want %d", calls.Load(), starts.Load(), tt.calls)
			}
		})
	}
}

func TestSendTransportDoesNotFollowRedirect(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Location", "/replay")
		w.WriteHeader(302)
	}))
	defer server.Close()
	client, err := NewClient(Config{BotToken: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SendMessage(context.Background(), SendMessageRequest{ChatID: "123", Text: "hello"})
	var sendErr *SendError
	if !errors.As(err, &sendErr) || sendErr.Kind != "unknown" || calls.Load() != 1 {
		t.Fatalf("calls %d error %v", calls.Load(), err)
	}
}
