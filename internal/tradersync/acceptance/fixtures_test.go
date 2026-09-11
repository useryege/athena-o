//go:build integration

package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	facade "github.com/useryege/athena/internal/server/tradersync"
	api "github.com/useryege/athena/pkg/apiclient/tradersync"
	gu "github.com/useryege/athena/util/grpc"
	"google.golang.org/grpc"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	et "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	erpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/useryege/athena/internal/accountcredentials"
	ac "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	notification "github.com/useryege/athena/internal/notification"
	ns "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/internal/testutil/pgtest"
	ts "github.com/useryege/athena/internal/tradersync"
	store "github.com/useryege/athena/internal/tradersync/store"
	tm "github.com/useryege/athena/internal/tradersync/types"
	pm "github.com/useryege/athena/util/polymarket"
	telegram "github.com/useryege/athena/util/telegram"
)

type recorded struct {
	Name, Version string
	Log           et.Log
	Expected      struct{ Wallet, Side, PositionID, CollateralRaw, SharesRaw, FeeRaw string }
	Origin        json.RawMessage
}
type filter struct {
	Addresses []common.Address `json:"address"`
	Topics    [][]common.Hash  `json:"topics"`
}
type socket struct {
	conn    *websocket.Conn
	mu      sync.Mutex
	filters map[string]filter
}

func (s *socket) write(value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return s.conn.WriteJSON(value)
}

type request struct {
	ID     json.RawMessage   `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}
type sendEvent struct {
	Chat int64
	Text string
	At   time.Time
}
type ownerResult struct{ Activities, Sent, Pending, Failed, Unknown, Cancelled int }
type snapshot struct {
	Relationships, UniqueTargets, Activities, HTTPCalls, HistoricalRangeCalls, SideEffects int
	Sources, Confirmed, Unverified                                                         int
	Owners                                                                                 map[string]ownerResult
	Costs                                                                                  map[string]int
	Cohorts                                                                                map[string]int
}
type harness struct {
	t                                            *testing.T
	db                                           *pgtest.DB
	ctx                                          context.Context
	cancel                                       context.CancelFunc
	once                                         sync.Once
	mu                                           sync.Mutex
	headers                                      map[common.Hash]*et.Header
	numbers                                      map[uint64]*et.Header
	receipts                                     map[common.Hash]*et.Receipt
	tip                                          *et.Header
	code                                         map[common.Address]string
	sockets                                      []*socket
	costs                                        map[string]int
	events                                       []sendEvent
	sideEffects, rangeCalls                      int
	nextFilter                                   int
	httpFault                                    string
	codeFault                                    bool
	sendFault                                    string
	sendHold                                     <-chan struct{}
	chainHTTP, wsHTTP, profileHTTP, telegramHTTP *httptest.Server
	eth                                          *ethclient.Client
	transport                                    *http.Transport
	service                                      *ts.Service
	notifier                                     *notification.Service
	runDone                                      chan error
	ownerIDs                                     []string
	wallets                                      []common.Address
	subs                                         map[string]tm.SubscriptionDetails
	fixtures                                     []recorded
}

// Now always remains the real process monotonic clock. Advance waits real time;
// it does not fabricate advancement of PostgreSQL or SourceRPC's private cache.
func (h *harness) Advance(d time.Duration) {
	h.t.Helper()
	select {
	case <-time.After(d):
	case <-h.ctx.Done():
		h.t.Fatal(h.ctx.Err())
	}
}
func (h *harness) Close() {
	h.once.Do(func() {
		h.cancel()
		if h.notifier != nil {
			if e := h.notifier.Stop(); e != nil {
				h.t.Errorf("notification stop: %v", e)
			}
		}
		if h.service != nil {
			if e := h.service.Close(); e != nil {
				h.t.Errorf("trader close: %v", e)
			}
		}
		if h.runDone != nil {
			select {
			case e := <-h.runDone:
				if e != nil {
					h.t.Errorf("trader run: %v", e)
				}
			case <-time.After(8 * time.Second):
				h.t.Error("trader service did not join")
			}
		}
		if h.eth != nil {
			h.eth.Close()
		}
		if h.transport != nil {
			h.transport.CloseIdleConnections()
		}
		for _, server := range []*httptest.Server{h.chainHTTP, h.wsHTTP, h.profileHTTP, h.telegramHTTP} {
			if server != nil {
				server.Close()
			}
		}
	})
}
func newHarness(t *testing.T, owners, targets int) *harness {
	t.Helper()
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithCancel(context.Background())
	h := &harness{t: t, db: db, ctx: ctx, cancel: cancel, headers: map[common.Hash]*et.Header{}, numbers: map[uint64]*et.Header{}, receipts: map[common.Hash]*et.Receipt{}, code: map[common.Address]string{}, costs: map[string]int{}, subs: map[string]tm.SubscriptionDetails{}}
	t.Cleanup(h.Close)
	raw, e := os.ReadFile("../testdata/source_records.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(raw, &h.fixtures); e != nil {
		t.Fatal(e)
	}
	for address, file := range map[string]string{"0xe111180000d2663c0091e4f400237545b87b996b": "core-exchange.hex", "0xe2222d279d744050d28e00520010520000310f59": "neg-risk-exchange.hex", "0xe3333700ca9d93003f00f0f71f8515005f6c00aa": "combo-proxy.hex", "0x641b40ec414a076b9e79e703fc7bf4ebec248bb7": "combo-implementation.hex"} {
		b, e := os.ReadFile(filepath.Join("../testdata/runtime", file))
		if e != nil {
			t.Fatal(e)
		}
		h.code[common.HexToAddress(address)] = strings.TrimSpace(string(b))
	}
	h.tip = &et.Header{Number: big.NewInt(10000), Difficulty: big.NewInt(0), Time: uint64(time.Now().Unix()), Extra: []byte("synthetic acceptance parent")}
	h.headers[h.tip.Hash()] = h.tip
	h.numbers[10000] = h.tip
	h.chainHTTP = httptest.NewServer(http.HandlerFunc(h.serveRPC))
	h.wsHTTP = httptest.NewServer(http.HandlerFunc(h.serveWSS))
	h.profileHTTP = httptest.NewServer(http.HandlerFunc(h.serveProfile))
	h.telegramHTTP = httptest.NewServer(http.HandlerFunc(h.serveTelegram))
	rpc, e := erpc.DialOptions(ctx, h.chainHTTP.URL, erpc.WithHTTPClient(&http.Client{Timeout: 5 * time.Second}))
	if e != nil {
		t.Fatal(e)
	}
	h.eth = ethclient.NewClient(rpc)
	node := ts.NewSourceRPC(h.eth)
	trader := store.NewSQLStore(db.Pool)
	if e = trader.ConfigureActivities("https://athena.test"); e != nil {
		t.Fatal(e)
	}
	h.transport = http.DefaultTransport.(*http.Transport).Clone()
	h.transport.Proxy = nil
	rewrite := loopbackTransport{transport: h.transport, target: h.profileHTTP.URL}
	resolver, e := ts.NewTargetResolver(db.Pool, pm.NewProfileAdapter(rewrite), trader.RequireGrantTx, trader.ResolveContextTx)
	if e != nil {
		t.Fatal(e)
	}
	config := ts.Config{HTTPURL: h.chainHTTP.URL, WebSocketURL: "ws" + strings.TrimPrefix(h.wsHTTP.URL, "http"), SiteURL: "https://athena.test", CursorHMACKey: "acceptance-stable-fixture-key", Projector: ts.ProjectorConfig{Interval: 2 * time.Second, MetadataWait: 2 * time.Second, MetadataTimeout: 30 * time.Second, MaxInFlightSources: 100}, ReconnectMin: 20 * time.Millisecond, ReconnectMax: 100 * time.Millisecond, OnError: func(e error) { t.Logf("collector/directory observed: %v", e) }}
	collector, e := ts.NewCollector(trader, node, config)
	if e != nil {
		t.Fatal(e)
	}
	subscriptions, e := ts.NewSubscriptionService(db.Pool, trader, resolver, collector)
	if e != nil {
		t.Fatal(e)
	}
	gamma, e := pm.NewGammaClient(pm.GammaConfig{GammaBaseURL: h.profileHTTP.URL, ComboBaseURL: h.profileHTTP.URL, HTTPClient: &http.Client{Timeout: 5 * time.Second}, Timeout: 5 * time.Second})
	if e != nil {
		t.Fatal(e)
	}
	projector, e := ts.NewProjector(trader, node, ts.NewVersionVerifier(node), ts.NewMetadataResolver(gamma, node, trader), config.Projector)
	if e != nil {
		t.Fatal(e)
	}
	h.service, e = ts.NewService(config, ts.Dependencies{Pool: db.Pool, Resolver: resolver, Subscriptions: subscriptions, Collector: collector, Projector: projector, Directory: ts.NewDirectoryRefresher(trader, gamma, config.OnError)})
	if e != nil {
		t.Fatal(e)
	}
	h.runDone = make(chan error, 1)
	go func() { h.runDone <- h.service.Run(ctx) }()
	client, e := telegram.NewClient(telegram.Config{BotToken: "acceptance", BaseURL: h.telegramHTTP.URL})
	if e != nil {
		t.Fatal(e)
	}
	notifications := ns.NewSQLStore(db.Pool)
	h.notifier = notification.NewService(notifications, notification.NewTelegramSender(client, map[string]string{"test": "-123"}), notification.NewTelegramProfileSyncer(client, telegram.BotProfileConfig{Name: "Acceptance"}), notification.NewTelegramPoller(notifications, client))
	if e = h.notifier.ConfigureSummaries(db.Pool, "https://athena.test"); e != nil {
		t.Fatal(e)
	}
	if e = h.notifier.Start(ctx); e != nil {
		t.Fatal(e)
	}
	base, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < owners; i++ {
		owner := base.ID
		if i > 0 {
			owner = uuid.NewString()
			for _, sql := range []string{`INSERT INTO athena_account(account_id,username,identity_provider,identity_subject,verified_email)VALUES($1::text::uuid,$1::text,'google',$1::text,$1::text||'@test.invalid')`, `INSERT INTO account_access(account_id,login_enabled,api_key_enabled,profit_sharing_enabled,revision)VALUES($1,true,false,false,1)`} {
				if _, e = db.Pool.Exec(ctx, sql, owner); e != nil {
					t.Fatal(e)
				}
			}
			if _, e = db.Pool.Exec(ctx, `INSERT INTO account_module_access(account_id,module,access_level)SELECT $1,module,access_level FROM account_module_access WHERE account_id=$2`, owner, base.ID); e != nil {
				t.Fatal(e)
			}
		}
		h.ownerIDs = append(h.ownerIDs, owner)
		chat := int64(1000 + i)
		if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,$2,$2,'acceptance',1)`, owner, chat); e != nil {
			t.Fatal(e)
		}
	}
	for j := 0; j < targets; j++ {
		h.wallets = append(h.wallets, common.BigToAddress(big.NewInt(int64(j+1))))
	}
	perOwner := targets
	if owners > 1 && targets == owners*10 {
		perOwner = 10
	}
	for i, owner := range h.ownerIDs {
		for j := 0; j < perOwner; j++ {
			index := j
			if targets == owners*10 && owners > 1 {
				index = i*10 + j
			}
			wallet := h.wallets[index]
			resolved, e := h.service.ResolveTarget(ctx, owner, wallet.Hex())
			if e != nil {
				t.Fatal("resolve", e)
			}
			if !strings.Contains(resolved.Card.UsageNotice, "low-frequency") {
				t.Fatal("missing low-frequency notice")
			}
			note := fmt.Sprintf("owner-%d-target-%d", i, j)
			sub, e := h.service.CreateSubscription(ctx, owner, tm.CreateInput{Token: resolved.Token, RequestID: uuid.NewString(), Note: &note})
			if e != nil {
				t.Fatal("create", e)
			}
			h.subs[owner+wallet.Hex()] = sub
		}
	}
	h.wait("all real baselines healthy", 15*time.Second, func() bool {
		var count int
		e := db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_subscriptions WHERE observation_state='healthy'`).Scan(&count)
		return e == nil && count == owners*perOwner
	})
	return h
}

type loopbackTransport struct {
	transport http.RoundTripper
	target    string
}

func (r loopbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := req.Clone(req.Context())
	u, e := url.Parse(r.target)
	if e != nil {
		return nil, e
	}
	copy.URL.Scheme = u.Scheme
	copy.URL.Host = u.Host
	copy.Host = u.Host
	return r.transport.RoundTrip(copy)
}
func (h *harness) wait(what string, d time.Duration, ready func() bool) {
	h.t.Helper()
	timer := time.NewTimer(d)
	defer timer.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		if ready() {
			return
		}
		select {
		case <-h.ctx.Done():
			h.t.Fatal(what, h.ctx.Err())
		case <-timer.C:
			h.t.Fatalf("%s timed out; stats=%+v", what, h.Stats())
		case <-tick.C:
		}
	}
}
func (h *harness) newHeaderLocked() *et.Header {
	n := new(big.Int).Add(h.tip.Number, big.NewInt(1))
	v := &et.Header{Number: n, Difficulty: big.NewInt(0), Time: uint64(time.Now().Unix()), ParentHash: h.tip.Hash(), Extra: []byte("synthetic replay chain location")}
	h.headers[v.Hash()] = v
	h.numbers[n.Uint64()] = v
	h.tip = v
	return v
}
func (h *harness) Replay(wallet common.Address, index int) et.Log {
	h.t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	v := h.fixtures[index].Log
	v.Topics = append([]common.Hash(nil), v.Topics...)
	v.Data = append([]byte(nil), v.Data...)
	v.Topics[2] = common.BytesToHash(wallet.Bytes())
	head := h.newHeaderLocked()
	v.BlockHash = head.Hash()
	v.BlockNumber = head.Number.Uint64()
	v.TxHash = common.BigToHash(head.Number)
	v.TxIndex = 0
	v.Index = 0
	v.Removed = false
	copy := v
	h.receipts[v.TxHash] = &et.Receipt{Status: 1, BlockHash: v.BlockHash, BlockNumber: new(big.Int).Set(head.Number), TxHash: v.TxHash, TransactionIndex: 0, Logs: []*et.Log{&copy}}
	return v
}
func (h *harness) Push(raw et.Log) {
	h.t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	matches := 0
	for _, s := range h.sockets {
		for id, q := range s.filters {
			address := false
			for _, a := range q.Addresses {
				address = address || a == raw.Address
			}
			if !address {
				continue
			}
			matched := true
			for i, choices := range q.Topics {
				if len(choices) == 0 {
					continue
				}
				hit := false
				for _, topic := range choices {
					hit = hit || (i < len(raw.Topics) && raw.Topics[i] == topic)
				}
				matched = matched && hit
			}
			if matched {
				if e := s.write(map[string]any{"jsonrpc": "2.0", "method": "eth_subscription", "params": map[string]any{"subscription": id, "result": raw}}); e == nil {
					matches++
					h.costs["wss_notification"]++
				}
			}
		}
	}
	if matches == 0 {
		h.t.Fatal("replay did not match an acknowledged real filter")
	}
}
func (h *harness) serveWSS(w http.ResponseWriter, r *http.Request) {
	conn, e := (&websocket.Upgrader{}).Upgrade(w, r, nil)
	if e != nil {
		return
	}
	defer conn.Close()
	s := &socket{conn: conn, filters: map[string]filter{}}
	h.mu.Lock()
	h.sockets = append(h.sockets, s)
	h.costs["wss_connection"]++
	h.mu.Unlock()
	defer func() { h.mu.Lock(); s.filters = map[string]filter{}; h.mu.Unlock() }()
	for {
		var q request
		if conn.ReadJSON(&q) != nil {
			return
		}
		h.mu.Lock()
		h.costs["wss_"+q.Method]++
		var result any = true
		switch q.Method {
		case "eth_chainId":
			result = "0x89"
		case "eth_subscribe":
			var f filter
			if len(q.Params) != 2 || json.Unmarshal(q.Params[1], &f) != nil {
				h.t.Error("invalid subscribe shape")
			}
			h.nextFilter++
			id := fmt.Sprintf("sub-%d", h.nextFilter)
			s.filters[id] = f
			result = id
		case "eth_unsubscribe":
			var id string
			json.Unmarshal(q.Params[0], &id)
			delete(s.filters, id)
		default:
			h.sideEffects++
			h.t.Errorf("unexpected WSS method: %s", q.Method)
		}
		h.mu.Unlock()
		if s.write(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result}) != nil {
			return
		}
	}
}
func (h *harness) serveRPC(w http.ResponseWriter, r *http.Request) {
	var q request
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		w.WriteHeader(400)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.costs["rpc_"+q.Method]++
	var result any
	var rpcErr any
	selector := func(i int) string {
		var s string
		if i < len(q.Params) {
			json.Unmarshal(q.Params[i], &s)
		}
		return s
	}
	switch q.Method {
	case "eth_chainId":
		result = "0x89"
	case "eth_getBlockByNumber":
		key := selector(0)
		h.costs["block_"+key]++
		if key == "latest" {
			result = h.newHeaderLocked()
		} else if key == "finalized" {
			if h.httpFault == "403" {
				w.WriteHeader(403)
				io.WriteString(w, "fixture HTTP forbidden")
				return
			}
			if h.httpFault != "null" {
				result = h.tip
			}
		} else {
			number, e := strconv.ParseUint(strings.TrimPrefix(key, "0x"), 16, 64)
			if e == nil {
				result = h.numbers[number]
			}
		}
	case "eth_getBlockByHash":
		result = h.headers[common.HexToHash(selector(0))]
	case "eth_getTransactionReceipt":
		result = h.receipts[common.HexToHash(selector(0))]
	case "eth_getCode", "eth_getStorageAt":
		var location struct {
			BlockHash common.Hash `json:"blockHash"`
		}
		slot := 1
		if q.Method == "eth_getStorageAt" {
			slot = 2
		}
		if slot >= len(q.Params) || json.Unmarshal(q.Params[slot], &location) != nil || h.headers[location.BlockHash] == nil {
			rpcErr = map[string]any{"code": -32000, "message": "exact known hash required"}
			break
		}
		address := common.HexToAddress(selector(0))
		if q.Method == "eth_getCode" {
			result = h.code[address]
			if h.codeFault {
				result = "0x00"
			}
		} else {
			result = "0x000000000000000000000000641b40ec414a076b9e79e703fc7bf4ebec248bb7"
		}
	case "eth_getLogs":
		var query map[string]json.RawMessage
		json.Unmarshal(q.Params[0], &query)
		if query["fromBlock"] != nil || query["toBlock"] != nil || query["blockHash"] == nil {
			h.rangeCalls++
			h.t.Error("forbidden range log request")
		}
		result = []et.Log{}
	default:
		h.sideEffects++
		h.t.Errorf("unexpected or side-effect RPC: %s", q.Method)
		rpcErr = map[string]any{"code": -32601, "message": "unsupported fixture call"}
	}
	w.Header().Set("Content-Type", "application/json")
	body := map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result}
	if rpcErr != nil {
		delete(body, "result")
		body["error"] = rpcErr
	}
	json.NewEncoder(w).Encode(body)
}
func (h *harness) serveProfile(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.costs["profile_"+r.URL.Path]++
	h.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/public-profile":
		json.NewEncoder(w).Encode(map[string]any{"proxyWallet": r.URL.Query().Get("address"), "name": "Recorded payload replay target", "verifiedBadge": false})
	case "/v1/user-stats":
		io.WriteString(w, `{"joinDate":"2024-06-19T21:35:45Z","largestWin":0}`)
	case "/traded":
		fmt.Fprintf(w, `{"user":%q,"traded":0}`, r.URL.Query().Get("user"))
	case "/value":
		fmt.Fprintf(w, `[{"user":%q,"value":0}]`, r.URL.Query().Get("user"))
	case "/user-pnl":
		io.WriteString(w, `[{"t":1788000000,"p":0},{"t":1789000000,"p":0}]`)
	case "/v1/rfq/combo-markets":
		io.WriteString(w, `{"markets":[],"next_cursor":""}`)
	case "/markets":
		token := r.URL.Query().Get("clob_token_ids")
		ids, _ := json.Marshal([]string{token})
		json.NewEncoder(w).Encode([]any{map[string]any{"id": "1", "conditionId": "0xabc", "question": "Synthetic replay market", "slug": "synthetic-replay", "clobTokenIds": string(ids), "outcomes": "[\"Yes\"]"}})
	default:
		h.t.Errorf("unexpected metadata/profile path %s", r.URL)
		w.WriteHeader(404)
	}
}
func (h *harness) serveTelegram(w http.ResponseWriter, r *http.Request) {
	method := filepath.Base(r.URL.Path)
	h.mu.Lock()
	h.costs["telegram_"+method]++
	h.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	var result any = true
	switch method {
	case "getMe":
		result = map[string]any{"id": 1, "is_bot": true, "username": "acceptance_bot", "first_name": "Acceptance"}
	case "getMyName":
		result = map[string]any{"name": "Acceptance"}
	case "getMyDescription":
		result = map[string]any{"description": ""}
	case "getMyShortDescription":
		result = map[string]any{"short_description": ""}
	case "getWebhookInfo":
		result = map[string]any{"url": ""}
	case "getUpdates":
		select {
		case <-h.ctx.Done():
			return
		case <-r.Context().Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
		result = []any{}
	case "sendMessage":
		r.ParseMultipartForm(2 << 20)
		chat, _ := strconv.ParseInt(r.FormValue("chat_id"), 10, 64)
		if r.FormValue("parse_mode") != "" {
			h.t.Error("ordinary/summary request was not explicit plain")
		}
		h.mu.Lock()
		h.events = append(h.events, sendEvent{Chat: chat, Text: r.FormValue("text"), At: time.Now()})
		id := len(h.events)
		fault := h.sendFault
		hold := h.sendHold
		h.mu.Unlock()
		if hold != nil {
			select {
			case <-hold:
			case <-r.Context().Done():
				return
			case <-h.ctx.Done():
				return
			}
		}
		if fault == "429" {
			w.WriteHeader(429)
			io.WriteString(w, `{"ok":false,"error_code":429,"description":"fixture retry","parameters":{"retry_after":1}}`)
			return
		}
		if fault == "timeout" {
			select {
			case <-r.Context().Done():
			case <-h.ctx.Done():
			}
			return
		}
		result = map[string]any{"message_id": id}
	default:
		h.mu.Lock()
		h.sideEffects++
		h.mu.Unlock()
		h.t.Errorf("unexpected Telegram mutation: %s", method)
	}
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": result})
}
func (h *harness) Stats() snapshot {
	h.t.Helper()
	s := snapshot{Owners: map[string]ownerResult{}, Costs: map[string]int{}, Cohorts: map[string]int{}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for _, item := range []struct {
		sql string
		out *int
	}{{`SELECT count(*) FROM trader_sync_subscriptions`, &s.Relationships}, {`SELECT count(DISTINCT wallet) FROM trader_sync_subscriptions`, &s.UniqueTargets}, {`SELECT count(*) FROM trader_sync_activities`, &s.Activities}, {`SELECT count(*) FROM trader_sync_source_records`, &s.Sources}, {`SELECT count(*) FROM trader_sync_source_records WHERE confirmation_state='confirmed'`, &s.Confirmed}, {`SELECT count(*) FROM trader_sync_source_records WHERE confirmation_state='unverified'`, &s.Unverified}} {
		if e := h.db.Pool.QueryRow(ctx, item.sql).Scan(item.out); e != nil {
			h.t.Fatal(e)
		}
	}
	for _, owner := range h.ownerIDs {
		var v ownerResult
		if e := h.db.Pool.QueryRow(ctx, `SELECT count(*),count(*)FILTER(WHERE d.status='sent'),count(*)FILTER(WHERE d.status='pending'),count(*)FILTER(WHERE d.status='failed'),count(*)FILTER(WHERE d.status='unknown'),count(*)FILTER(WHERE d.status='cancelled') FROM trader_sync_activities a LEFT JOIN account_notification_deliveries d ON d.activity_id=a.id WHERE a.owner_id=$1`, owner).Scan(&v.Activities, &v.Sent, &v.Pending, &v.Failed, &v.Unknown, &v.Cancelled); e != nil {
			h.t.Fatal(e)
		}
		s.Owners[owner] = v
	}
	rows, e := h.db.Pool.Query(ctx, `SELECT formation_evidence->>'cohort',count(*) FROM trader_sync_activities GROUP BY 1`)
	if e != nil {
		h.t.Fatal(e)
	}
	for rows.Next() {
		var key string
		var n int
		if e = rows.Scan(&key, &n); e != nil {
			h.t.Fatal(e)
		}
		s.Cohorts[key] = n
	}
	rows.Close()
	h.mu.Lock()
	s.HTTPCalls = len(h.events)
	s.HistoricalRangeCalls = h.rangeCalls
	s.SideEffects = h.sideEffects
	for k, v := range h.costs {
		s.Costs[k] = v
	}
	h.mu.Unlock()
	return s
}

func (h *harness) captureRuntime(name string) map[string]string {
	h.t.Helper()
	admin, e := ac.NewSQLStore(h.db.Pool).EnsureDevelopmentAccount(h.ctx, accountcredentials.DevelopmentRoleAdministrator)
	if e != nil {
		h.t.Fatal(e)
	}
	lis, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		h.t.Fatal(e)
	}
	// A fixed authenticated-principal context is the test boundary. Database admin
	// permission and the actual runtime/facade/JSON path remain real; JWT policy has
	// its existing independent server tests and is not claimed by this fixture.
	g := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, r any, i *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		return next(context.WithValue(ctx, "claims", jwt.MapClaims{"sub": admin.ID}), r)
	}))
	api.RegisterTraderSyncServiceServer(g, facade.New(h.service))
	go g.Serve(lis)
	defer g.Stop()
	conn, e := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if e != nil {
		h.t.Fatal(e)
	}
	defer conn.Close()
	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, new(gu.JSONMarshaler)))
	if e = api.RegisterTraderSyncServiceHandler(h.ctx, mux, conn); e != nil {
		h.t.Fatal(e)
	}
	server := httptest.NewServer(mux)
	defer server.Close()
	response, e := server.Client().Get(server.URL + "/api/v1/admin/trader-sync/status")
	if e != nil {
		h.t.Fatal(e)
	}
	raw, e := io.ReadAll(response.Body)
	response.Body.Close()
	if e != nil || response.StatusCode != http.StatusOK {
		h.t.Fatal(response.Status, string(raw), e)
	}
	var body struct {
		Status struct {
			Metrics []struct{ Name, Value string }
		}
	}
	if e = json.Unmarshal(raw, &body); e != nil {
		h.t.Fatal(e)
	}
	metrics := map[string]string{}
	for _, m := range body.Status.Metrics {
		metrics[m.Name] = m.Value
	}
	if metrics["timing_activity_all_total"] != strconv.Itoa(h.Stats().Activities) || metrics["projector_confirmation_round_count"] == "" {
		h.t.Fatalf("actual runtime omitted complete totals/rounds: %s", raw)
	}
	if dir := os.Getenv("ATHENA_TASK13_GATEWAY_FIXTURE_DIR"); dir != "" {
		if e = os.MkdirAll(dir, 0755); e != nil {
			h.t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dir, name+".json"), raw, 0644); e != nil {
			h.t.Fatal(e)
		}
	}
	h.t.Logf("actual runtime metrics %s=%s", name, string(raw))
	return metrics
}
func (h *harness) captureTiming(name string) {
	h.t.Helper()
	rows, e := h.db.Pool.Query(h.ctx, `SELECT a.id,a.received_at,a.recorded_at,a.formation_evidence,d.status,t.authorized,t.started,t.ack FROM trader_sync_activities a LEFT JOIN account_notification_deliveries d ON d.activity_id=a.id LEFT JOIN LATERAL(SELECT min(authorized_at)AS authorized,min(started_at)AS started,max(sender_returned_at)FILTER(WHERE outcome='sent')AS ack FROM notification_delivery_attempts WHERE work_kind='account' AND work_id=d.id)t ON true ORDER BY a.id`)
	if e != nil {
		h.t.Fatal(e)
	}
	defer rows.Close()
	type measured struct {
		ID        int64
		Sample    ts.TimingSample
		Result    ts.TimingResult
		Formation tm.FormationEvidence
	}
	var samples []measured
	for rows.Next() {
		var v measured
		var evidence []byte
		var outcome *string
		if e = rows.Scan(&v.ID, &v.Sample.ReceivedAt, &v.Sample.RecordedAt, &evidence, &outcome, &v.Sample.AuthorizedAt, &v.Sample.StartedAt, &v.Sample.AckAt); e != nil {
			h.t.Fatal(e)
		}
		if e = json.Unmarshal(evidence, &v.Formation); e != nil {
			h.t.Fatal(e)
		}
		if v.Formation.Current.ElapsedNS == nil || v.Formation.Current.Sequence == 0 {
			h.t.Fatal("real original receipt monotonic evidence missing")
		}
		v.Sample.Cohort = v.Formation.Cohort
		if outcome != nil {
			v.Sample.Outcome = *outcome
		}
		v.Result = ts.EvaluateTiming(v.Sample)
		if v.Result.PublicAssessable {
			h.t.Fatal("replay invented independent public timestamp")
		}
		samples = append(samples, v)
	}
	if e = rows.Err(); e != nil {
		h.t.Fatal(e)
	}
	raw, e := json.MarshalIndent(samples, "", "  ")
	if e != nil {
		h.t.Fatal(e)
	}
	if dir := os.Getenv("ATHENA_TASK13_GATEWAY_FIXTURE_DIR"); dir != "" {
		if e = os.MkdirAll(dir, 0755); e != nil {
			h.t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dir, name+"-samples.json"), raw, 0644); e != nil {
			h.t.Fatal(e)
		}
	}
	h.t.Logf("%s original frozen timing samples=%d; public interval missing, unassessable", name, len(samples))
}
