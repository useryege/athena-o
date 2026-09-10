package telegram

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

type admissionRoundTripper func(*http.Request) (*http.Response, error)

func (f admissionRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSendAdmissionExcludesLocalWaitFromHTTPTimeoutAndReleasesBeforeIO(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	entered, resume := make(chan struct{}), make(chan struct{})
	var released, started atomic.Bool
	ctx = WithSendTimeout(ctx, 40*time.Millisecond)
	ctx = WithSendAdmission(ctx, func(ctx context.Context) (func(), error) {
		close(entered)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-resume:
		}
		return func() { released.Store(true) }, nil
	})
	observation := &sendObservation{callback: func(time.Time) {
		if released.Load() {
			t.Error("admission released before started event")
		}
		started.Store(true)
	}}
	ctx = context.WithValue(ctx, sendObservationKey{}, observation)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost/sendMessage", strings.NewReader("body"))
	if err != nil {
		t.Fatal(err)
	}
	client := &sendHTTPClient{client: &http.Client{Timeout: 40 * time.Millisecond, Transport: &sendTransport{base: admissionRoundTripper(func(req *http.Request) (*http.Response, error) {
		if !started.Load() || !released.Load() {
			t.Error("HTTP I/O entered before started/release")
		}
		deadline, ok := req.Context().Deadline()
		if !ok || time.Until(deadline) <= 0 {
			t.Error("HTTP timeout spent during admission")
		}
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}}}
	done := make(chan error, 1)
	go func() { _, err := client.Do(req); done <- err }()
	<-entered
	select {
	case err := <-done:
		t.Fatalf("local wait consumed HTTP timeout: %v", err)
	case <-time.After(80 * time.Millisecond):
	}
	close(resume)
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("HTTP timeout not enforced: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("HTTP timeout did not run after admission")
	}
}

func TestSendAdmissionCancellationClosesBodyAndPreTransportFailureReleases(t *testing.T) {
	for _, cancelAdmission := range []bool{true, false} {
		ctx, cancel := context.WithCancel(context.Background())
		var releases atomic.Int32
		ctx = WithSendAdmission(ctx, func(context.Context) (func(), error) {
			if cancelAdmission {
				return nil, context.Canceled
			}
			return func() { releases.Add(1) }, nil
		})
		ctx = context.WithValue(ctx, sendObservationKey{}, &sendObservation{})
		reader, writer := io.Pipe()
		writerDone := make(chan error, 1)
		go func() { _, err := writer.Write([]byte("body")); _ = writer.Close(); writerDone <- err }()
		// Empty URL fails in http.Client before RoundTrip; both error paths close the SDK body.
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "", reader)
		if err != nil {
			t.Fatal(err)
		}
		client := &sendHTTPClient{client: &http.Client{}}
		_, err = client.Do(req)
		cancel()
		if err == nil {
			t.Fatal("expected pre-transport failure")
		}
		want := int32(1)
		if cancelAdmission {
			want = 0
		}
		if releases.Load() != want {
			t.Fatalf("release count %d", releases.Load())
		}
		select {
		case <-writerDone:
		case <-time.After(time.Second):
			t.Fatal("SDK request writer leaked")
		}
	}
}
