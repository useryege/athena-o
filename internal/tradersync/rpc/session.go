// Package rpc owns a single physical live Polygon WebSocket, never a reconnecting client.
package rpc

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"github.com/useryege/athena/util/ethws"
)

type sessionConfig struct {
	queue               int
	ping, pong, request time.Duration
}

var defaultSessionConfig = sessionConfig{1024, 15 * time.Second, 5 * time.Second, 5 * time.Second}

type wireRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      uint64 `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}
type wireResponse struct {
	value json.RawMessage
	err   error
}
type pendingRequest struct {
	response  chan wireResponse
	method    string
	abandoned bool
}
type frame struct {
	kind int
	body []byte
}
type Session struct {
	conn             *websocket.Conn
	config           sessionConfig
	done             chan struct{}
	logs             chan tm.ReceivedLog
	writes           chan frame
	pong             chan string
	wg               sync.WaitGroup
	once             sync.Once
	mu               sync.Mutex
	err              error
	nextID, sequence uint64
	pending          map[uint64]*pendingRequest
	high             map[common.Address]uint64
}

func DialSession(ctx context.Context, endpoint, proxyURL string) (*Session, error) {
	return dialSession(ctx, endpoint, proxyURL, defaultSessionConfig)
}
func dialSession(ctx context.Context, endpoint, proxyURL string, cfg sessionConfig) (*Session, error) {
	if err := ethws.ValidateProxyURL(proxyURL); err != nil {
		return nil, err
	}
	dialer := websocket.Dialer{HandshakeTimeout: cfg.request}
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		dialer.Proxy = http.ProxyURL(u)
	}
	conn, resp, err := dialer.DialContext(ctx, endpoint, nil)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, err
	}
	s := &Session{conn: conn, config: cfg, done: make(chan struct{}), logs: make(chan tm.ReceivedLog, cfg.queue), writes: make(chan frame, 64), pong: make(chan string, 1), pending: make(map[uint64]*pendingRequest), high: make(map[common.Address]uint64)}
	conn.SetReadLimit(2 << 20)
	conn.SetPingHandler(func(data string) error {
		select {
		case s.writes <- frame{websocket.PongMessage, []byte(data)}:
			return nil
		default:
			return errors.New("WSS control queue full")
		}
	})
	conn.SetPongHandler(func(data string) error {
		select {
		case s.pong <- data:
		default:
		}
		return nil
	})
	s.wg.Add(2)
	go s.readLoop()
	go s.writeLoop()
	var chain string
	result, err := s.call(ctx, "eth_chainId", []any{})
	if err == nil {
		err = json.Unmarshal(result, &chain)
		if err == nil && chain != "0x89" {
			err = errors.New("WSS endpoint is not Polygon 137")
		}
	}
	if err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}
func (s *Session) fail(err error) {
	s.once.Do(func() { s.mu.Lock(); s.err = err; s.mu.Unlock(); close(s.done); _ = s.conn.Close() })
}
func (s *Session) Close() error                { s.fail(context.Canceled); s.wg.Wait(); return nil }
func (s *Session) Done() <-chan struct{}       { return s.done }
func (s *Session) Err() error                  { s.mu.Lock(); defer s.mu.Unlock(); return s.err }
func (s *Session) Logs() <-chan tm.ReceivedLog { return s.logs }

// Snapshot linearizes registration against all logs already decoded by this read
// loop, including messages still waiting in the persistence queue. No SQL here.
func (s *Session) Snapshot(wallet common.Address) tm.WalletObservation {
	s.mu.Lock()
	defer s.mu.Unlock()
	return tm.WalletObservation{High: s.high[wallet], Sequence: s.sequence}
}
func (s *Session) Subscribe(ctx context.Context, q ethereum.FilterQuery) (string, error) {
	if q.FromBlock != nil || q.ToBlock != nil || q.BlockHash != nil {
		return "", errors.New("live subscription cannot specify historical blocks")
	}
	result, err := s.call(ctx, "eth_subscribe", []any{"logs", map[string]any{"address": q.Addresses, "topics": q.Topics}})
	var id string
	if err == nil {
		err = json.Unmarshal(result, &id)
		if err == nil && id == "" {
			err = errors.New("missing subscription ACK")
		}
	}
	return id, err
}
func (s *Session) Unsubscribe(ctx context.Context, id string) error {
	result, err := s.call(ctx, "eth_unsubscribe", []any{id})
	var ok bool
	if err == nil {
		err = json.Unmarshal(result, &ok)
		if err == nil && !ok {
			err = errors.New("subscription removal not acknowledged")
		}
	}
	return err
}
func (s *Session) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.request)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if len(s.pending) >= 64 {
		s.mu.Unlock()
		return nil, errors.New("WSS pending request capacity exhausted")
	}
	s.nextID++
	id := s.nextID
	p := &pendingRequest{response: make(chan wireResponse, 1), method: method}
	s.pending[id] = p
	s.mu.Unlock()
	forgetUnsent := func() { s.mu.Lock(); delete(s.pending, id); s.mu.Unlock() }
	body, err := json.Marshal(wireRequest{"2.0", id, method, params})
	if err != nil {
		forgetUnsent()
		return nil, err
	}
	select {
	case s.writes <- frame{websocket.TextMessage, body}:
	case <-ctx.Done():
		forgetUnsent()
		return nil, ctx.Err()
	case <-s.done:
		forgetUnsent()
		return nil, s.Err()
	}
	select {
	case r := <-p.response:
		return r.value, r.err
	case <-ctx.Done():
		s.abandon(id, p)
		return nil, ctx.Err()
	case <-s.done:
		s.abandon(id, p)
		return nil, s.Err()
	}
}
func (s *Session) abandon(id uint64, delivered *pendingRequest) {
	s.mu.Lock()
	if p := s.pending[id]; p != nil {
		p.abandoned = true
		s.mu.Unlock()
		return
	}
	// The reader publishes its buffered response before deleting the map entry.
	var result wireResponse
	select {
	case result = <-delivered.response:
	default:
	}
	s.mu.Unlock()
	if delivered.method == "eth_subscribe" && result.err == nil {
		var sub string
		if json.Unmarshal(result.value, &sub) == nil && sub != "" {
			s.removeLateSubscription(sub)
		}
	}
}
func (s *Session) readLoop() {
	defer s.wg.Done()
	defer close(s.logs)
	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			s.fail(err)
			return
		}
		var msg struct {
			ID     uint64          `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			Params struct {
				Subscription string          `json:"subscription"`
				Result       json.RawMessage `json:"result"`
			} `json:"params"`
		}
		if err = json.Unmarshal(data, &msg); err != nil {
			s.fail(err)
			return
		}
		if msg.ID != 0 {
			var responseErr error
			if msg.Error != nil {
				responseErr = fmt.Errorf("JSON-RPC %d: %s", msg.Error.Code, msg.Error.Message)
			}
			s.mu.Lock()
			p := s.pending[msg.ID]
			abandoned := p != nil && p.abandoned
			if p != nil && !abandoned {
				p.response <- wireResponse{msg.Result, responseErr}
			}
			delete(s.pending, msg.ID)
			s.mu.Unlock()
			if abandoned && p.method == "eth_subscribe" && responseErr == nil {
				var sub string
				if json.Unmarshal(msg.Result, &sub) == nil && sub != "" {
					s.removeLateSubscription(sub)
				}
			}
			continue
		}

		if msg.Method != "eth_subscription" {
			continue
		}
		var received tm.ReceivedLog
		if msg.Params.Subscription == "" {
			s.fail(errors.New("log missing subscription identifier"))
			return
		}
		// Log's generated decoder defaults absent positions to zero. Preserve the
		// distinction: incomplete wire facts cannot become durable zero locations.
		var fields map[string]json.RawMessage
		if err = json.Unmarshal(msg.Params.Result, &fields); err != nil {
			s.fail(err)
			return
		}
		for _, key := range []string{"address", "topics", "data", "blockNumber", "blockHash", "transactionHash", "transactionIndex", "logIndex", "removed"} {
			value := bytes.TrimSpace(fields[key])
			if len(value) == 0 || bytes.Equal(value, []byte("null")) {
				s.fail(fmt.Errorf("log missing required field %s", key))
				return
			}
		}
		if err = json.Unmarshal(msg.Params.Result, &received.Raw); err != nil {
			s.fail(err)
			return
		}
		received.ReceivedAt = time.Now().UTC()
		if received.Raw.BlockNumber > math.MaxInt64 {
			s.fail(errors.New("observed block exceeds persistent bigint"))
			return
		}
		s.mu.Lock()
		if s.sequence == math.MaxInt64 {
			s.mu.Unlock()
			s.fail(errors.New("read sequence exhausted"))
			return
		}
		s.sequence++
		received.Sequence = s.sequence
		if len(received.Raw.Topics) >= 3 {
			wallet := common.BytesToAddress(received.Raw.Topics[2].Bytes())
			if received.Raw.BlockNumber > s.high[wallet] {
				s.high[wallet] = received.Raw.BlockNumber
			}
		}
		s.mu.Unlock()
		select {
		case s.logs <- received:
		case <-s.done:
			return
		default:
			s.fail(errors.New("WSS persistence queue full"))
			return
		}
	}
}
func (s *Session) removeLateSubscription(id string) {
	s.mu.Lock()
	s.nextID++
	requestID := s.nextID
	s.mu.Unlock()
	body, _ := json.Marshal(wireRequest{"2.0", requestID, "eth_unsubscribe", []any{id}})
	select {
	case s.writes <- frame{websocket.TextMessage, body}:
	case <-s.done:
	default:
		s.fail(errors.New("late ACK cleanup queue full"))
	}
}
func (s *Session) writeLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.ping)
	defer ticker.Stop()
	var nonce string
	var deadline <-chan time.Time
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-s.done:
			return
		case f := <-s.writes:
			_ = s.conn.SetWriteDeadline(time.Now().Add(s.config.request))
			if err := s.conn.WriteMessage(f.kind, f.body); err != nil {
				s.fail(err)
				return
			}
		case <-ticker.C:
			if nonce != "" {
				continue
			}
			var b [16]byte
			if _, err := rand.Read(b[:]); err != nil {
				s.fail(err)
				return
			}
			nonce = hex.EncodeToString(b[:])
			if err := s.conn.WriteControl(websocket.PingMessage, []byte(nonce), time.Now().Add(s.config.request)); err != nil {
				s.fail(err)
				return
			}
			timer = time.NewTimer(s.config.pong)
			deadline = timer.C
		case value := <-s.pong:
			if nonce != "" && value == nonce {
				nonce = ""
				if timer != nil {
					timer.Stop()
				}
				deadline = nil
			}
		case <-deadline:
			s.fail(errors.New("WSS matching pong timeout"))
			return
		}
	}
}
