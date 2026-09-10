package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func logFixture() ethtypes.Log {
	return ethtypes.Log{Address: common.HexToAddress("0x123"), Topics: []common.Hash{common.HexToHash("0x456"), {}, common.BytesToHash(common.HexToAddress("0x789").Bytes())}, Data: []byte{}, BlockNumber: 100, BlockHash: common.HexToHash("0xaa"), TxHash: common.HexToHash("0xbb"), Index: 1}
}
func serveSession(t *testing.T, handle func(*websocket.Conn)) string {
	t.Helper()
	up := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		handle(c)
	}))
	t.Cleanup(server.Close)
	return "ws" + strings.TrimPrefix(server.URL, "http")
}
func readRequest(c *websocket.Conn) (map[string]json.RawMessage, error) {
	var r map[string]json.RawMessage
	err := c.ReadJSON(&r)
	return r, err
}
func ack(c *websocket.Conn, r map[string]json.RawMessage, result any) error {
	return c.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": r["id"], "result": result})
}
func chainHandshake(c *websocket.Conn) error {
	r, err := readRequest(c)
	if err != nil {
		return err
	}
	return ack(c, r, "0x89")
}
func TestSessionRegistersObservedLogBeforeFilterACKAndPersistenceRead(t *testing.T) {
	raw := logFixture()
	release := make(chan struct{})
	observed := make(chan struct{})
	url := serveSession(t, func(c *websocket.Conn) {
		if chainHandshake(c) != nil {
			return
		}
		r, err := readRequest(c)
		if err != nil {
			return
		}
		_ = c.WriteJSON(map[string]any{"jsonrpc": "2.0", "method": "eth_subscription", "params": map[string]any{"subscription": "sub1", "result": raw}})
		close(observed)
		<-release
		_ = ack(c, r, "sub1")
		for {
			if _, err := readRequest(c); err != nil {
				return
			}
		}
	})
	s, err := DialSession(context.Background(), url, "")
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	defer s.Close()
	finished := make(chan error, 1)
	go func() {
		id, err := s.Subscribe(context.Background(), ethereum.FilterQuery{Addresses: []common.Address{raw.Address}, Topics: [][]common.Hash{{raw.Topics[0]}, nil, {raw.Topics[2]}}})
		if err == nil && id != "sub1" {
			t.Error(id)
		}
		finished <- err
	}()
	<-observed
	wallet := common.BytesToAddress(raw.Topics[2].Bytes())
	deadline := time.Now().Add(time.Second)
	for s.Snapshot(wallet).High != 100 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	snapshot := s.Snapshot(wallet)
	if snapshot.High != 100 || snapshot.Sequence == 0 {
		close(release)
		t.Fatal("queued decoded log missing from registration snapshot", snapshot)
	}
	select {
	case <-finished:
		close(release)
		t.Fatal("subscribe completed without ACK")
	default:
	}
	close(release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-s.Logs():
		if got.Sequence != snapshot.Sequence || got.ReceivedAt.IsZero() || got.Raw.BlockHash != raw.BlockHash {
			t.Fatal(got)
		}
	case <-time.After(time.Second):
		t.Fatal("log missing")
	}
}

func TestSessionMatchingPongOnlyAndQueueOverflowCloseConnection(t *testing.T) {
	for _, mode := range []string{"quiet healthy", "wrong pong", "silent pong", "queue full"} {
		t.Run(mode, func(t *testing.T) {
			url := serveSession(t, func(c *websocket.Conn) {
				if mode == "wrong pong" {
					c.SetPingHandler(func(string) error {
						return c.WriteControl(websocket.PongMessage, []byte("wrong"), time.Now().Add(time.Second))
					})
				}
				if mode == "silent pong" {
					c.SetPingHandler(func(string) error { return nil })
				}
				if chainHandshake(c) != nil {
					return
				}
				if mode == "queue full" {
					for i := 0; i < 3; i++ {
						_ = c.WriteJSON(map[string]any{"method": "eth_subscription", "params": map[string]any{"subscription": "sub", "result": logFixture()}})
					}
				}
				for {
					if _, err := readRequest(c); err != nil {
						return
					}
				}
			})
			s, err := dialSession(context.Background(), url, "", sessionConfig{1, 15 * time.Millisecond, 20 * time.Millisecond, time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if mode == "quiet healthy" {
				select {
				case <-s.Done():
					t.Fatal("quiet matching-pong connection failed", s.Err())
				case <-time.After(80 * time.Millisecond):
				}
				return
			}
			select {
			case <-s.Done():
				if s.Err() == nil {
					t.Fatal("missing fault")
				}
			case <-time.After(time.Second):
				t.Fatal("unhealthy connection remained open")
			}
		})
	}
}
func TestSessionCancelledSubscribeCleansLateACKWithoutClosingExistingConnection(t *testing.T) {
	seen, cancelled, cleaned := make(chan struct{}), make(chan struct{}), make(chan struct{})
	url := serveSession(t, func(c *websocket.Conn) {
		if chainHandshake(c) != nil {
			return
		}
		r, err := readRequest(c)
		if err != nil {
			return
		}
		close(seen)
		<-cancelled
		_ = ack(c, r, "late-sub")
		r, err = readRequest(c)
		if err != nil {
			return
		}
		var method string
		_ = json.Unmarshal(r["method"], &method)
		if method != "eth_unsubscribe" {
			t.Error(method)
		}
		close(cleaned)
		_ = ack(c, r, true)
		for {
			if _, err = readRequest(c); err != nil {
				return
			}
		}
	})
	s, err := DialSession(context.Background(), url, "")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := s.Subscribe(ctx, ethereum.FilterQuery{}); done <- err }()
	<-seen
	cancel()
	if err = <-done; err == nil {
		t.Fatal("cancelled subscription succeeded")
	}
	close(cancelled)
	select {
	case <-cleaned:
	case <-time.After(time.Second):
		t.Fatal("late subscription leaked")
	}
	select {
	case <-s.Done():
		t.Fatal("new target failure closed old connection", s.Err())
	default:
	}
}

func TestSessionCancellationAfterACKDeliveryStillCleansSubscription(t *testing.T) {
	p := &pendingRequest{method: "eth_subscribe", response: make(chan wireResponse, 1)}
	p.response <- wireResponse{value: json.RawMessage(`"already-acked"`)}
	s := &Session{pending: map[uint64]*pendingRequest{}, writes: make(chan frame, 1), done: make(chan struct{})}
	s.abandon(1, p)
	select {
	case f := <-s.writes:
		if !strings.Contains(string(f.body), "already-acked") {
			t.Fatal(string(f.body))
		}
	default:
		t.Fatal("ACK raced cancellation and leaked subscription")
	}
}

func TestHTTPAdapterIsExplicitDirectReadOnlyEndpoint(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     uint64 `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Method != "eth_chainId" {
			t.Error("unexpected method", request.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": "0x89"})
	}))
	defer server.Close()
	client, err := DialHTTP(context.Background(), server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	chain, err := client.ChainID(context.Background())
	if err != nil || chain.Uint64() != 137 {
		t.Fatal(chain, err)
	}
	if unexpected, err := DialHTTP(context.Background(), "ws://127.0.0.1:1", ""); err == nil {
		unexpected.Close()
		t.Fatal("HTTP adapter accepted reconnecting WSS transport")
	}
}

func TestSessionCancelledBeforeEnqueueDoesNotConsumePendingCapacity(t *testing.T) {
	session := &Session{config: defaultSessionConfig, pending: make(map[uint64]*pendingRequest), writes: make(chan frame), done: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := session.call(ctx, "eth_subscribe", []any{"logs"}); err == nil {
		t.Fatal("cancelled request accepted")
	}
	if len(session.pending) != 0 {
		t.Fatal("never-enqueued request leaked pending ACK slot", len(session.pending))
	}
}

func TestSessionLogLocationRequiresPresenceButAcceptsZero(t *testing.T) {
	for _, field := range []string{"blockNumber", "logIndex", "transactionIndex", "blockNumber/null", "logIndex/null", "transactionIndex/null", "valid-zero"} {
		t.Run(field, func(t *testing.T) {
			raw := logFixture()
			raw.BlockNumber = 0
			raw.Index = 0
			raw.TxIndex = 0
			data, _ := json.Marshal(raw)
			var payload map[string]any
			_ = json.Unmarshal(data, &payload)
			if field != "valid-zero" {
				if strings.HasSuffix(field, "/null") {
					payload[strings.TrimSuffix(field, "/null")] = nil
				} else {
					delete(payload, field)
				}
			}
			endpoint := serveSession(t, func(conn *websocket.Conn) {
				if chainHandshake(conn) != nil {
					return
				}
				_ = conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "method": "eth_subscription", "params": map[string]any{"subscription": "test", "result": payload}})
				for {
					if _, _, err := conn.ReadMessage(); err != nil {
						return
					}
				}
			})
			session, err := DialSession(context.Background(), endpoint, "")
			if err != nil {
				if field == "valid-zero" {
					t.Fatal(err)
				}
				return
			}
			defer session.Close()
			select {
			case received, ok := <-session.Logs():
				if field != "valid-zero" && ok {
					t.Fatal("missing log location became a zero-valued fact", field, received.Raw)
				}
				if field == "valid-zero" && (!ok || received.Raw.BlockNumber != 0 || received.Raw.Index != 0) {
					t.Fatal("valid zero location rejected")
				}
			case <-time.After(time.Second):
				t.Fatal("log admission did not complete")
			}
		})
	}
}

func TestSessionExhaustedSentACKsCloseAndFreshSessionRecovers(t *testing.T) {
	var connections, sent, closed atomic.Int32
	endpoint := serveSession(t, func(conn *websocket.Conn) {
		number := connections.Add(1)
		defer closed.Add(1)
		if chainHandshake(conn) != nil {
			return
		}
		for {
			request, err := readRequest(conn)
			if err != nil {
				return
			}
			if number == 1 {
				if sent.Load() == 0 {
					if ack(conn, request, "old-live-subscription") != nil {
						return
					}
					sent.Add(1)
					continue
				}
				sent.Add(1) // These requests reached the wire, but their ACKs never return.
				continue
			}
			// An old, unrelated request ID cannot satisfy this new Session's call.
			_ = conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": 66, "result": "late-old-ACK"})
			if ack(conn, request, "fresh-subscription") != nil {
				return
			}
		}
	})
	cfg := sessionConfig{queue: 8, ping: 10 * time.Millisecond, pong: 100 * time.Millisecond, request: 10 * time.Millisecond}
	session, err := dialSession(context.Background(), endpoint, "", cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if _, err = session.Subscribe(context.Background(), ethereum.FilterQuery{}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 64; i++ {
		if _, err = session.Subscribe(context.Background(), ethereum.FilterQuery{}); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("expected sent request timeout", i, err)
		}
	}
	if sent.Load() != 65 {
		t.Fatal("test did not actually send all pending requests", sent.Load())
	}
	if _, err = session.Subscribe(context.Background(), ethereum.FilterQuery{}); err == nil {
		t.Fatal("exhausted request admitted")
	}
	select {
	case <-session.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("exhausted ACK capacity stranded a live Session")
	}
	if !strings.Contains(session.Err().Error(), "capacity") {
		t.Fatal(session.Err())
	}
	session.Close()
	if closed.Load() == 0 {
		deadline := time.Now().Add(time.Second)
		for closed.Load() == 0 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
	}
	if closed.Load() == 0 {
		t.Fatal("old connection subscriptions did not close")
	}
	fresh, err := dialSession(context.Background(), endpoint, "", cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Close()
	id, err := fresh.Subscribe(context.Background(), ethereum.FilterQuery{})
	if err != nil || id != "fresh-subscription" {
		t.Fatal("fresh control request did not recover or took stale ACK", id, err)
	}
}
