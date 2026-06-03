package polymarket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewCLOBMarketWSClientDefaults(t *testing.T) {
	raw, err := NewCLOBMarketWSClient(CLOBMarketWSConfig{})
	if err != nil {
		t.Fatalf("NewCLOBMarketWSClient: %v", err)
	}
	impl, ok := raw.(*clobMarketWSClientImpl)
	if !ok {
		t.Fatalf("client type = %T, want *clobMarketWSClientImpl", raw)
	}
	if impl.config.WSURL != DefaultCLOBMarketWSURL {
		t.Fatalf("WSURL = %q, want %q", impl.config.WSURL, DefaultCLOBMarketWSURL)
	}
	if impl.config.ReconnectInitial <= 0 || impl.config.ReconnectMax <= 0 {
		t.Fatalf("invalid reconnect defaults: %+v", impl.config)
	}
}

func TestNewCLOBMarketWSClientInvalidURL(t *testing.T) {
	_, err := NewCLOBMarketWSClient(CLOBMarketWSConfig{WSURL: "://bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCLOBMarketWSSubscribeHeartbeatAndDispatch(t *testing.T) {
	upgrader := websocket.Upgrader{}

	bookSeen := make(chan struct{}, 1)
	unknownSeen := make(chan struct{}, 1)
	subSeen := make(chan CLOBMarketWSSubscription, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()

		var sub CLOBMarketWSSubscription
		if err := conn.ReadJSON(&sub); err != nil {
			t.Errorf("read sub: %v", err)
			return
		}
		subSeen <- sub

		_ = conn.WriteJSON(CLOBMarketBookEvent{
			EventType: "book",
			AssetID:   "a1",
			Market:    "m1",
			Bids:      []CLOBOrderSummary{{Price: "0.5", Size: "1"}},
			Asks:      []CLOBOrderSummary{{Price: "0.6", Size: "1"}},
			Timestamp: "1",
			Hash:      "h",
		})
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"event_type":"not_supported","x":1}`))
		<-time.After(200 * time.Millisecond)
	}))
	defer ts.Close()

	client, err := NewCLOBMarketWSClient(CLOBMarketWSConfig{
		WSURL:            toWSURL(ts.URL),
		ReconnectInitial: 10 * time.Millisecond,
		ReconnectMax:     20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewCLOBMarketWSClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Run(ctx, CLOBMarketWSSubscription{
			AssetIDs: []string{"a1"},
		}, CLOBMarketWSHandler{
			OnBook: func(event CLOBMarketBookEvent) {
				if event.EventType == "book" {
					select {
					case bookSeen <- struct{}{}:
					default:
					}
				}
			},
			OnUnknown: func(_ json.RawMessage) {
				select {
				case unknownSeen <- struct{}{}:
				default:
				}
			},
		})
	}()

	select {
	case sub := <-subSeen:
		if sub.Type != "market" || len(sub.AssetIDs) != 1 || sub.AssetIDs[0] != "a1" {
			t.Fatalf("unexpected sub: %#v", sub)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for subscription")
	}

	wait := func(ch <-chan struct{}, label string) {
		select {
		case <-ch:
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s", label)
		}
	}
	wait(bookSeen, "book")
	wait(unknownSeen, "unknown")

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

func TestCLOBMarketWSHeartbeatReply(t *testing.T) {
	upgrader := websocket.Upgrader{}
	heartbeatSeen := make(chan struct{}, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var sub CLOBMarketWSSubscription
		if err := conn.ReadJSON(&sub); err != nil {
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, []byte("PING")); err != nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if strings.TrimSpace(string(msg)) == "PONG" {
			select {
			case heartbeatSeen <- struct{}{}:
			default:
			}
		}
	}))
	defer ts.Close()

	client, err := NewCLOBMarketWSClient(CLOBMarketWSConfig{
		WSURL:            toWSURL(ts.URL),
		ReconnectInitial: 10 * time.Millisecond,
		ReconnectMax:     20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewCLOBMarketWSClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Run(ctx, CLOBMarketWSSubscription{
			AssetIDs: []string{"a1"},
		}, CLOBMarketWSHandler{
			OnHeartbeat: func(_ string) {},
		})
	}()

	select {
	case <-heartbeatSeen:
	case <-ctx.Done():
		t.Fatal("timed out waiting for heartbeat reply")
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

func TestCLOBMarketWSReconnectAndResubscribe(t *testing.T) {
	upgrader := websocket.Upgrader{}
	var mu sync.Mutex
	connections := 0
	subs := 0

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		mu.Lock()
		connections++
		connNum := connections
		mu.Unlock()

		var sub CLOBMarketWSSubscription
		if err := conn.ReadJSON(&sub); err == nil {
			mu.Lock()
			subs++
			mu.Unlock()
		}

		if connNum == 1 {
			_ = conn.Close()
			return
		}
		_ = conn.WriteJSON(CLOBMarketBookEvent{
			EventType: "book",
			AssetID:   "a1",
			Market:    "m1",
			Timestamp: "1",
			Hash:      "h",
		})
		<-time.After(100 * time.Millisecond)
	}))
	defer ts.Close()

	client, err := NewCLOBMarketWSClient(CLOBMarketWSConfig{
		WSURL:            toWSURL(ts.URL),
		ReconnectInitial: 10 * time.Millisecond,
		ReconnectMax:     20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewCLOBMarketWSClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Run(ctx, CLOBMarketWSSubscription{
			AssetIDs: []string{"a1"},
		}, CLOBMarketWSHandler{
			OnBook: func(_ CLOBMarketBookEvent) {
				cancel()
			},
		})
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit")
	}

	mu.Lock()
	defer mu.Unlock()
	if connections < 2 {
		t.Fatalf("connections = %d, want >= 2", connections)
	}
	if subs < 2 {
		t.Fatalf("subs = %d, want >= 2", subs)
	}
}

func TestCLOBMarketWSSubscriptionValidation(t *testing.T) {
	client, err := NewCLOBMarketWSClient(CLOBMarketWSConfig{WSURL: "ws://example.com/ws"})
	if err != nil {
		t.Fatalf("NewCLOBMarketWSClient: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err = client.Run(ctx, CLOBMarketWSSubscription{}, CLOBMarketWSHandler{})
	if err == nil || !strings.Contains(err.Error(), "requires at least one asset id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatchMarketEventObjectPayload(t *testing.T) {
	bookCount := 0
	err := dispatchMarketEvent([]byte(`{"event_type":"book","asset_id":"a1","market":"m1","timestamp":"1","hash":"h"}`), CLOBMarketWSHandler{
		OnBook: func(event CLOBMarketBookEvent) {
			bookCount++
			if event.AssetID != "a1" || event.Market != "m1" {
				t.Fatalf("book event = %#v, want asset a1 market m1", event)
			}
		},
	})
	if err != nil {
		t.Fatalf("dispatchMarketEvent: %v", err)
	}
	if bookCount != 1 {
		t.Fatalf("book count = %d, want 1", bookCount)
	}
}

func TestDispatchMarketEventArrayPayload(t *testing.T) {
	bookCount := 0
	priceChangeCount := 0
	unknownCount := 0
	err := dispatchMarketEvent([]byte(`[
		{"event_type":"book","asset_id":"a1","market":"m1","timestamp":"1","hash":"h"},
		{"event_type":"book","asset_id":"a2","market":"m2","timestamp":"2","hash":"h"},
		{"event_type":"price_change","market":"m1","timestamp":"3","price_changes":[{"asset_id":"a1","price":"0.5","size":"1","side":"BUY","hash":"h"}]},
		{"event_type":"not_supported","x":1}
	]`), CLOBMarketWSHandler{
		OnBook: func(CLOBMarketBookEvent) {
			bookCount++
		},
		OnPriceChange: func(CLOBMarketPriceChangeEvent) {
			priceChangeCount++
		},
		OnUnknown: func(json.RawMessage) {
			unknownCount++
		},
	})
	if err != nil {
		t.Fatalf("dispatchMarketEvent: %v", err)
	}
	if bookCount != 2 || priceChangeCount != 1 || unknownCount != 1 {
		t.Fatalf("counts book=%d price=%d unknown=%d, want 2/1/1", bookCount, priceChangeCount, unknownCount)
	}
}

func TestDispatchMarketEventArrayContinuesAfterBadElement(t *testing.T) {
	bookCount := 0
	unknownCount := 0
	err := dispatchMarketEvent([]byte(`[
		1,
		{"event_type":"book","asset_id":"a1","market":"m1","timestamp":"1","hash":"h"}
	]`), CLOBMarketWSHandler{
		OnBook: func(CLOBMarketBookEvent) {
			bookCount++
		},
		OnUnknown: func(json.RawMessage) {
			unknownCount++
		},
	})
	if err == nil {
		t.Fatal("expected decode error")
	}
	var decodeErr *CLOBMarketWSDecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("error = %T %v, want CLOBMarketWSDecodeError", err, err)
	}
	if bookCount != 1 || unknownCount != 1 {
		t.Fatalf("counts book=%d unknown=%d, want 1/1", bookCount, unknownCount)
	}
}

func TestDispatchMarketEventInvalidPayloadUnknownAndDecodeError(t *testing.T) {
	unknownCount := 0
	err := dispatchMarketEvent([]byte(`42`), CLOBMarketWSHandler{
		OnUnknown: func(json.RawMessage) {
			unknownCount++
		},
	})
	if err == nil {
		t.Fatal("expected decode error")
	}
	var decodeErr *CLOBMarketWSDecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("error = %T %v, want CLOBMarketWSDecodeError", err, err)
	}
	if unknownCount != 1 {
		t.Fatalf("unknown count = %d, want 1", unknownCount)
	}
}

func toWSURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}
