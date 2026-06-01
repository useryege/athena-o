package polymarket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewSportsWSClientDefaults(t *testing.T) {
	raw, err := NewSportsWSClient(SportsWSConfig{})
	if err != nil {
		t.Fatalf("NewSportsWSClient: %v", err)
	}
	impl, ok := raw.(*sportsWSClientImpl)
	if !ok {
		t.Fatalf("client type = %T, want *sportsWSClientImpl", raw)
	}
	if impl.config.WSURL != DefaultSportsWSURL {
		t.Fatalf("WSURL = %q, want %q", impl.config.WSURL, DefaultSportsWSURL)
	}
}

func TestNewSportsWSClientInvalidURL(t *testing.T) {
	_, err := NewSportsWSClient(SportsWSConfig{WSURL: "://bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSportsWSPingPongAndUpdate(t *testing.T) {
	upgrader := websocket.Upgrader{}
	updateSeen := make(chan SportsWSUpdate, 1)
	pongSeen := make(chan struct{}, 1)
	unknownSeen := make(chan struct{}, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()

		if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
			t.Errorf("write ping: %v", err)
			return
		}
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("read pong: %v", err)
			return
		}
		if strings.TrimSpace(string(msg)) == "pong" {
			pongSeen <- struct{}{}
		}

		_ = conn.WriteJSON(map[string]any{
			"slug":        "abc-def",
			"live":        true,
			"score":       "1-0",
			"last_update": "2026-01-01T00:00:00Z",
		})
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"foo":"bar"}`))
		<-time.After(200 * time.Millisecond)
	}))
	defer ts.Close()

	client, err := NewSportsWSClient(SportsWSConfig{
		WSURL:            toWSURL(ts.URL),
		ReconnectInitial: 10 * time.Millisecond,
		ReconnectMax:     20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewSportsWSClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Run(ctx, SportsWSHandler{
			OnUpdate: func(update SportsWSUpdate) {
				select {
				case updateSeen <- update:
				default:
				}
			},
			OnUnknown: func(_ json.RawMessage) {
				select {
				case unknownSeen <- struct{}{}:
				default:
				}
			},
			OnHeartbeat: func(msg string) {
				if msg == "ping" {
					select {
					case pongSeen <- struct{}{}:
					default:
					}
				}
			},
		})
	}()

	select {
	case <-pongSeen:
	case <-ctx.Done():
		t.Fatal("timed out waiting for pong")
	}

	select {
	case update := <-updateSeen:
		if update.Slug != "abc-def" {
			t.Fatalf("slug = %q", update.Slug)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for update")
	}

	select {
	case <-unknownSeen:
	case <-ctx.Done():
		t.Fatal("timed out waiting for unknown")
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit")
	}
}
