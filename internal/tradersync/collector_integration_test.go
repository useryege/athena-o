//go:build integration

package tradersync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/useryege/athena/internal/accountcredentials"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"github.com/useryege/athena/internal/tradersync/store"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type collectorNode struct{ finalityCalls, latestCalls atomic.Int32 }

func (n *collectorNode) FinalizedHeader(context.Context) (*ethtypes.Header, error) {
	n.finalityCalls.Add(1)
	return nil, errors.New("temporary finality outage")
}
func (n *collectorNode) HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error) {
	n.latestCalls.Add(1)
	return &ethtypes.Header{Number: big.NewInt(100), Time: uint64(time.Now().Unix())}, nil
}
func (n *collectorNode) HeaderByHash(context.Context, common.Hash) (*ethtypes.Header, error) {
	return nil, errors.New("unexpected known hash read")
}
func (n *collectorNode) TransactionReceipt(context.Context, common.Hash) (*ethtypes.Receipt, error) {
	return nil, errors.New("must not read receipt before finality")
}
func (n *collectorNode) ChainID(context.Context) (*big.Int, error) { return big.NewInt(137), nil }
func (n *collectorNode) StorageAtHash(context.Context, common.Address, common.Hash, common.Hash) ([]byte, error) {
	return nil, errors.New("unexpected slot read")
}
func (n *collectorNode) CodeAtHash(context.Context, common.Address, common.Hash) ([]byte, error) {
	return nil, errors.New("unexpected code read")
}
func (n *collectorNode) UpgradeLogs(context.Context, common.Hash, common.Address) ([]ethtypes.Log, error) {
	return nil, errors.New("unexpected upgrades read")
}

func TestCollectorCommittedRegistrationPrecedesACKWithoutOwningFinality(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	wallet := common.HexToAddress("0x1111111111111111111111111111111111111111")
	var subscriptions atomic.Int32
	acknowledged := make(chan struct{})
	releaseACK := make(chan struct{})
	defer func() {
		select {
		case <-releaseACK:
		default:
			close(releaseACK)
		}
	}()
	up := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			var request struct {
				ID     uint64            `json:"id"`
				Method string            `json:"method"`
				Params []json.RawMessage `json:"params"`
			}
			if conn.ReadJSON(&request) != nil {
				return
			}
			result := any(true)
			switch request.Method {
			case "eth_chainId":
				result = "0x89"
			case "eth_subscribe":
				n := subscriptions.Add(1)
				result = "filter-" + string(rune('0'+n))
				if n == 1 {
					raw := ethtypes.Log{Address: sourceVersions[CoreExchangeVersion].deployment.address, Topics: []common.Hash{orderFilledEvents[CoreExchangeVersion].ID, {}, common.BytesToHash(wallet.Bytes()), {}}, Data: make([]byte, 224), BlockNumber: 100, BlockHash: common.HexToHash("0xaa"), TxHash: common.HexToHash("0xbb"), Index: 1}
					if conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "method": "eth_subscription", "params": map[string]any{"subscription": result, "result": raw}}) != nil {
						return
					}
					close(acknowledged)
					select {
					case <-releaseACK:
					case <-ctx.Done():
						return
					}
				}
			}
			if conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result}) != nil {
				return
			}
		}
	}))
	defer server.Close()
	node := &collectorNode{}
	collector, err := NewCollector(store.NewSQLStore(db.Pool), node, Config{WebSocketURL: "ws" + strings.TrimPrefix(server.URL, "http")})
	if err != nil {
		t.Fatal(err)
	}
	running := make(chan error, 1)
	go func() { running <- collector.Run(ctx) }()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	var subID string
	if err = tx.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state)VALUES($1,$2,'enabled','pending_baseline')RETURNING id`, owner.ID, wallet.Bytes()).Scan(&subID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('athena:account:' || $1::text,0))`, owner.ID); err != nil {
		t.Fatal(err)
	}
	if err = collector.RegisterTx(ctx, tx, tm.Subscription{ID: subID, OwnerID: owner.ID, Wallet: wallet, Generation: 1, Revision: 1}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(350 * time.Millisecond)
	if subscriptions.Load() != 0 {
		t.Fatal("filter exposed uncommitted registration")
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-acknowledged:
	case e := <-running:
		t.Fatal("collector ended", e)
	case <-time.After(3 * time.Second):
		t.Fatal("no subscription after commit")
	}
	var state string
	var candidates int
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_source_candidates`).Scan(&candidates)
		if candidates == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if candidates != 1 {
		t.Fatal("ACK-before candidate lost", candidates)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT state FROM trader_sync_baseline_attempts WHERE subscription_id=$1 ORDER BY created_at DESC LIMIT 1`, subID).Scan(&state); err != nil || state != "pending" {
		t.Fatal("healthy before all ACK", state, err)
	}
	close(releaseACK)
	deadline = time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		_ = db.Pool.QueryRow(ctx, `SELECT observation_state FROM trader_sync_subscriptions WHERE id=$1`, subID).Scan(&state)
		if state == "healthy" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if state != "healthy" {
		t.Fatal("baseline/finality not processed", state, node.finalityCalls.Load())
	}
	time.Sleep(2100 * time.Millisecond)
	if node.finalityCalls.Load() != 0 {
		t.Fatal("Collector must not own Projector finality scheduling", node.finalityCalls.Load())
	}
	var ended bool
	if err = db.Pool.QueryRow(ctx, `SELECT ended_at IS NOT NULL FROM trader_sync_collector_epochs ORDER BY id DESC LIMIT 1`).Scan(&ended); err != nil || ended {
		t.Fatal("finality failure closed WSS epoch", ended, err)
	}
	cancel()
	select {
	case err = <-running:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("collector did not join")
	}
	if err = db.Pool.QueryRow(context.Background(), `SELECT ended_at IS NOT NULL FROM trader_sync_collector_epochs ORDER BY id DESC LIMIT 1`).Scan(&ended); err != nil || !ended {
		t.Fatal("shutdown did not persist interruption", ended, err)
	}
}

func TestCollectorReconnectCreatesNewAttemptWithoutReassigningOldSource(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	wallet := common.HexToAddress("0x1111111111111111111111111111111111111111")
	var subID string
	if err = db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state)VALUES($1,$2,'enabled','pending_baseline')RETURNING id`, owner.ID, wallet.Bytes()).Scan(&subID); err != nil {
		t.Fatal(err)
	}
	var connections atomic.Int32
	disconnect := make(chan struct{})
	defer func() {
		select {
		case <-disconnect:
		default:
			close(disconnect)
		}
	}()
	up := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		number := connections.Add(1)
		sent := false
		for {
			var request struct {
				ID     uint64 `json:"id"`
				Method string `json:"method"`
			}
			if conn.ReadJSON(&request) != nil {
				return
			}
			result := any(true)
			if request.Method == "eth_chainId" {
				result = "0x89"
			}
			if request.Method == "eth_subscribe" {
				result = fmt.Sprintf("%d/%d", number, request.ID)
				if !sent {
					sent = true
					raw := ethtypes.Log{Address: sourceVersions[CoreExchangeVersion].deployment.address, Topics: []common.Hash{orderFilledEvents[CoreExchangeVersion].ID, {}, common.BytesToHash(wallet.Bytes()), {}}, Data: make([]byte, 224), BlockNumber: 100, BlockHash: common.HexToHash("0xaa"), TxHash: common.HexToHash("0xbb"), Index: 1}
					send := func() bool {
						return conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "method": "eth_subscription", "params": map[string]any{"subscription": result, "result": raw}}) == nil
					}
					if !send() {
						return
					}
					if number == 1 {
						select {
						case <-disconnect:
							return
						case <-ctx.Done():
							return
						}
					}
					raw.Index++
					if !send() {
						return
					}
				}
			}
			if conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result}) != nil {
				return
			}
		}
	}))
	defer server.Close()
	collector, err := NewCollector(store.NewSQLStore(db.Pool), &collectorNode{}, Config{WebSocketURL: "ws" + strings.TrimPrefix(server.URL, "http"), ReconnectMin: 10 * time.Millisecond, ReconnectMax: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	running := make(chan error, 1)
	go func() { running <- collector.Run(ctx) }()
	waitFor := func(label string, predicate func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if predicate() {
				return
			}
			select {
			case e := <-running:
				t.Fatal(label, e)
			default:
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("timed out", label)
	}
	waitFor("first raw committed", func() bool {
		var n int
		_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_source_candidates`).Scan(&n)
		return n == 1
	})
	close(disconnect)
	waitFor("new baseline healthy", func() bool {
		var state string
		_ = db.Pool.QueryRow(ctx, `SELECT observation_state FROM trader_sync_subscriptions WHERE id=$1`, subID).Scan(&state)
		return state == "healthy" && connections.Load() == 2
	})
	var failed, succeeded, candidates, records int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FILTER(WHERE state='failed'),count(*) FILTER(WHERE state='succeeded') FROM trader_sync_baseline_attempts`).Scan(&failed, &succeeded); err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_source_candidates`).Scan(&candidates); err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_source_records`).Scan(&records); err != nil {
		t.Fatal(err)
	}
	if failed != 1 || succeeded != 1 || candidates != 2 || records != 2 {
		t.Fatal("reconnect duplicated/reassigned facts", failed, succeeded, candidates, records)
	}
	var firstState string
	if err = db.Pool.QueryRow(ctx, `SELECT a.state FROM trader_sync_source_candidates c JOIN trader_sync_source_records r ON r.id=c.source_record_id JOIN trader_sync_baseline_attempts a ON a.id=c.baseline_attempt_id WHERE r.log_index=1`).Scan(&firstState); err != nil || firstState != "failed" {
		t.Fatal("failed candidate moved into new attempt", firstState, err)
	}
	cancel()
	select {
	case err = <-running:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("reconnect receiver did not join")
	}
}

func TestCollectorNewFilterFailureStillBaselinesAlreadyCoveredWallet(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	second := uuid.NewString()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO athena_account(account_id,username,identity_provider,identity_subject,verified_email)VALUES($1,'collector-second','google','collector-second','second@example.test')`, second); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO account_access(account_id,login_enabled,api_key_enabled,profit_sharing_enabled,revision)VALUES($1,true,false,false,1)`, second); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO account_module_access(account_id,module,access_level)VALUES($1,'trader_sync','read_write')`, second); err != nil {
		t.Fatal(err)
	}
	insert := func(owner string, wallet common.Address) string {
		t.Helper()
		var id string
		if err := db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state)VALUES($1,$2,'enabled','pending_baseline')RETURNING id`, owner, wallet.Bytes()).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	a := common.HexToAddress("0x1111111111111111111111111111111111111111")
	b := common.HexToAddress("0x2222222222222222222222222222222222222222")
	original := insert(owner.ID, a)
	alsoCovered := insert(owner.ID, common.HexToAddress("0x3333333333333333333333333333333333333333"))
	var rejectNew atomic.Bool
	rejectNew.Store(true)
	var rejected, oldUnsubscribed atomic.Int32
	up := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		subscriptions := 0
		for {
			var req struct {
				ID     uint64            `json:"id"`
				Method string            `json:"method"`
				Params []json.RawMessage `json:"params"`
			}
			if conn.ReadJSON(&req) != nil {
				return
			}
			result := any(true)
			if req.Method == "eth_chainId" {
				result = "0x89"
			}
			if req.Method == "eth_subscribe" {
				subscriptions++
				result = fmt.Sprintf("filter-%d", subscriptions)
				if subscriptions > 2 && subscriptions%2 == 0 && rejectNew.Load() {
					rejected.Add(1)
					if conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32000, "message": "new target temporarily unavailable"}}) != nil {
						return
					}
					continue
				}
			}
			if req.Method == "eth_unsubscribe" {
				var id string
				_ = json.Unmarshal(req.Params[0], &id)
				if id == "filter-1" || id == "filter-2" {
					oldUnsubscribed.Add(1)
				}
			}
			if conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}) != nil {
				return
			}
		}
	}))
	defer server.Close()
	node := &collectorNode{}
	collector, err := NewCollector(store.NewSQLStore(db.Pool), node, Config{WebSocketURL: "ws" + strings.TrimPrefix(server.URL, "http")})
	if err != nil {
		t.Fatal(err)
	}
	running := make(chan error, 1)
	go func() { running <- collector.Run(ctx) }()
	state := func(id string) string {
		var result string
		_ = db.Pool.QueryRow(ctx, `SELECT observation_state FROM trader_sync_subscriptions WHERE id=$1`, id).Scan(&result)
		return result
	}
	waitFor := func(label string, fn func() bool) {
		t.Helper()
		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) {
			if fn() {
				return
			}
			select {
			case err := <-running:
				t.Fatal(label, err)
			default:
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatal(label)
	}
	waitFor("original wallet never healthy", func() bool { return state(original) == "healthy" && state(alsoCovered) == "healthy" })
	if node.latestCalls.Load() != 1 {
		t.Fatal("same committed pending batch repeated latest RPC", node.latestCalls.Load())
	}
	var originalAt time.Time
	if err = db.Pool.QueryRow(ctx, `SELECT effective_at FROM trader_sync_subscriptions WHERE id=$1`, original).Scan(&originalAt); err != nil {
		t.Fatal(err)
	}
	covered := insert(second, a)
	uncovered := insert(second, b)
	waitFor("covered wallet's pending was blocked by unrelated filter failure", func() bool { return rejected.Load() > 0 && state(covered) == "healthy" })
	if state(uncovered) != "pending_baseline" || state(original) != "healthy" || oldUnsubscribed.Load() != 0 {
		t.Fatal("new target failure disturbed old coverage", state(uncovered), state(original), oldUnsubscribed.Load())
	}
	rejectNew.Store(false)
	waitFor("new target did not recover after filter ACK", func() bool { return state(uncovered) == "healthy" })
	var after time.Time
	if err = db.Pool.QueryRow(ctx, `SELECT effective_at FROM trader_sync_subscriptions WHERE id=$1`, original).Scan(&after); err != nil || !after.Equal(originalAt) {
		t.Fatal("old boundary reset", originalAt, after, err)
	}
	cancel()
	select {
	case err = <-running:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("collector did not join")
	}
}

func TestCollectorPersistenceFailureRecordsLastReliableReceiveAndReconnects(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	wallet := common.HexToAddress("0x1111111111111111111111111111111111111111")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state)VALUES($1,$2,'enabled','pending_baseline')`, owner.ID, wallet.Bytes()); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `CREATE FUNCTION reject_second_raw()RETURNS trigger LANGUAGE plpgsql AS $$BEGIN IF NEW.log_index=2 THEN RAISE EXCEPTION 'injected durable receive failure'; END IF;RETURN NEW;END$$;CREATE TRIGGER reject_second_raw BEFORE INSERT ON trader_sync_source_records FOR EACH ROW EXECUTE FUNCTION reject_second_raw()`); err != nil {
		t.Fatal(err)
	}
	var connections atomic.Int32
	up := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		connection := connections.Add(1)
		sent := false
		for {
			var req struct {
				ID     uint64 `json:"id"`
				Method string `json:"method"`
			}
			if conn.ReadJSON(&req) != nil {
				return
			}
			result := any(true)
			if req.Method == "eth_chainId" {
				result = "0x89"
			}
			if req.Method == "eth_subscribe" {
				result = fmt.Sprintf("%d/%d", connection, req.ID)
				if !sent {
					sent = true
					raw := ethtypes.Log{Address: sourceVersions[CoreExchangeVersion].deployment.address, Topics: []common.Hash{orderFilledEvents[CoreExchangeVersion].ID, {}, common.BytesToHash(wallet.Bytes()), {}}, Data: make([]byte, 224), BlockNumber: 100, BlockHash: common.HexToHash("0xaa"), TxHash: common.HexToHash("0xbb"), Index: 1}
					indexes := []uint{3}
					if connection == 1 {
						indexes = []uint{1, 2}
					}
					for _, index := range indexes {
						raw.Index = index
						if conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "method": "eth_subscription", "params": map[string]any{"subscription": result, "result": raw}}) != nil {
							return
						}
					}
				}
			}
			if conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}) != nil {
				return
			}
		}
	}))
	defer server.Close()
	collector, err := NewCollector(store.NewSQLStore(db.Pool), &collectorNode{}, Config{WebSocketURL: "ws" + strings.TrimPrefix(server.URL, "http"), ReconnectMin: 10 * time.Millisecond, ReconnectMax: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	running := make(chan error, 1)
	go func() { running <- collector.Run(ctx) }()
	var count int
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_source_records WHERE log_index=3`).Scan(&count)
		if count == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if count != 1 {
		t.Fatal("receiver did not reconnect after persistence failure")
	}
	var reason string
	var sequence int64
	var receivedAt *time.Time
	if err = db.Pool.QueryRow(ctx, `SELECT reason,last_read_sequence,last_received_at FROM trader_sync_interruptions ORDER BY id LIMIT 1`).Scan(&reason, &sequence, &receivedAt); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reason, "raw persistence failed") || sequence != 1 || receivedAt == nil {
		t.Fatal("interruption lost last reliable receive", reason, sequence, receivedAt)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_source_records WHERE log_index=2`).Scan(&count); err != nil || count != 0 {
		t.Fatal("failed transaction appeared durable", count, err)
	}
	cancel()
	select {
	case err = <-running:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("failed receiver did not join")
	}
}

func TestCollectorOwnershipLossReturnsFatalAndStopsReceiver(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	up := websocket.Upgrader{}
	connected := make(chan struct{})
	closed := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		defer close(closed)
		var req struct {
			ID uint64 `json:"id"`
		}
		if conn.ReadJSON(&req) != nil {
			return
		}
		if conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x89"}) != nil {
			return
		}
		close(connected)
		for {
			if _, _, err = conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()
	collector, err := NewCollector(store.NewSQLStore(db.Pool), &collectorNode{}, Config{WebSocketURL: "ws" + strings.TrimPrefix(server.URL, "http")})
	if err != nil {
		t.Fatal(err)
	}
	running := make(chan error, 1)
	go func() { running <- collector.Run(ctx) }()
	select {
	case <-connected:
	case <-time.After(3 * time.Second):
		t.Fatal("not connected")
	}
	var terminated bool
	if err = db.Pool.QueryRow(ctx, `SELECT pg_terminate_backend(pid) FROM pg_locks WHERE locktype='advisory' AND database=(SELECT oid FROM pg_database WHERE datname=current_database()) AND classid=((hashtextextended('athena:trader-sync:collector',0)>>32)&4294967295)::oid AND objid=(hashtextextended('athena:trader-sync:collector',0)&4294967295)::oid`).Scan(&terminated); err != nil || !terminated {
		t.Fatal("did not terminate dedicated test backend", terminated, err)
	}
	select {
	case err = <-running:
		if err == nil {
			t.Fatal("ownership loss silently reported success")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("lost owner left Run alive")
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("lost owner left WSS alive")
	}
}

func TestCollectorFailedBoundaryCloseReturnsErrorInsteadOfReusingEpoch(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := db.Pool.Exec(ctx, `CREATE FUNCTION reject_epoch_end()RETURNS trigger LANGUAGE plpgsql AS $$BEGIN IF NEW.ended_at IS NOT NULL THEN RAISE EXCEPTION 'injected epoch end failure';END IF;RETURN NEW;END$$;CREATE TRIGGER reject_epoch_end BEFORE UPDATE ON trader_sync_collector_epochs FOR EACH ROW EXECUTE FUNCTION reject_epoch_end()`); err != nil {
		t.Fatal(err)
	}
	up := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		var req struct {
			ID uint64 `json:"id"`
		}
		if conn.ReadJSON(&req) == nil {
			_ = conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x89"})
		}
	}))
	defer server.Close()
	collector, err := NewCollector(store.NewSQLStore(db.Pool), &collectorNode{}, Config{WebSocketURL: "ws" + strings.TrimPrefix(server.URL, "http"), ReconnectMin: 10 * time.Millisecond, ReconnectMax: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	running := make(chan error, 1)
	go func() { running <- collector.Run(ctx) }()
	select {
	case err = <-running:
		if err == nil || !strings.Contains(err.Error(), "epoch end failure") {
			t.Fatal("lost durable boundary error", err)
		}
	case <-time.After(2 * time.Second):
		cancel()
		<-running
		t.Fatal("failed boundary silently trapped collector in reconnect loop")
	}
}

type collectorHealthNode struct {
	collectorNode
	failLatest atomic.Bool
}

func (n *collectorHealthNode) HeaderByNumber(ctx context.Context, number *big.Int) (*ethtypes.Header, error) {
	if n.failLatest.Load() {
		return nil, errors.New("injected latest health failure")
	}
	return n.collectorNode.HeaderByNumber(ctx, number)
}
func TestCollectorLatestHealthContinuesWhileBaselineWaitsForAccount(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	up := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			var req struct {
				ID     uint64 `json:"id"`
				Method string `json:"method"`
			}
			if conn.ReadJSON(&req) != nil {
				return
			}
			result := any(true)
			if req.Method == "eth_chainId" {
				result = "0x89"
			}
			if req.Method == "eth_subscribe" {
				result = fmt.Sprintf("filter-%d", req.ID)
			}
			if conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}) != nil {
				return
			}
		}
	}))
	defer server.Close()
	blocker, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(ctx)
	if _, err = blocker.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('athena:account:' || $1::text,0))`, owner.ID); err != nil {
		t.Fatal(err)
	}
	// Committed work is visible, but its baseline must wait for another account TX.
	if _, err = db.Pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state)VALUES($1,$2,'enabled','pending_baseline')`, owner.ID, common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes()); err != nil {
		t.Fatal(err)
	}
	node := &collectorHealthNode{}
	node.failLatest.Store(true)
	collector, err := NewCollector(store.NewSQLStore(db.Pool), node, Config{WebSocketURL: "ws" + strings.TrimPrefix(server.URL, "http")})
	if err != nil {
		t.Fatal(err)
	}
	running := make(chan error, 1)
	go func() { running <- collector.Run(ctx) }()
	var reason string
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		_ = db.Pool.QueryRow(ctx, `SELECT reason FROM trader_sync_interruptions ORDER BY id LIMIT 1`).Scan(&reason)
		if strings.Contains(reason, "latest health") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = blocker.Rollback(ctx)
	cancel()
	select {
	case <-running:
	case <-time.After(3 * time.Second):
		t.Fatal("blocked account did not cancel/join")
	}
	if !strings.Contains(reason, "latest health") {
		t.Fatal("account wait suppressed 10-second latest health", reason)
	}
}

func TestCollectorWSSFailureCancelsAccountWaitWithHealthyLatest(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	insert := func(wallet string) string {
		t.Helper()
		var id string
		if err := db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state)VALUES($1,$2,'enabled','pending_baseline')RETURNING id`, owner.ID, common.HexToAddress(wallet).Bytes()).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	original := insert("0x1111111111111111111111111111111111111111")
	connections := make(chan *websocket.Conn, 4)
	up := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, e := up.Upgrade(w, r, nil)
		if e != nil {
			return
		}
		defer conn.Close()
		connections <- conn
		for {
			var req struct {
				ID     uint64 `json:"id"`
				Method string `json:"method"`
			}
			if conn.ReadJSON(&req) != nil {
				return
			}
			result := any(true)
			if req.Method == "eth_chainId" {
				result = "0x89"
			}
			if req.Method == "eth_subscribe" {
				result = fmt.Sprintf("filter-%d", req.ID)
			}
			if conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}) != nil {
				return
			}
		}
	}))
	defer server.Close()
	node := &collectorNode{} // Every latest read succeeds; it cannot cancel the account wait.
	collector, err := NewCollector(store.NewSQLStore(db.Pool), node, Config{WebSocketURL: "ws" + strings.TrimPrefix(server.URL, "http"), ReconnectMin: 10 * time.Millisecond, ReconnectMax: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	running := make(chan error, 1)
	go func() { running <- collector.Run(ctx) }()
	wait := func(label string, fn func() bool) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if fn() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal(label)
	}
	wait("existing A did not become healthy", func() bool {
		var state string
		_ = db.Pool.QueryRow(ctx, `SELECT observation_state FROM trader_sync_subscriptions WHERE id=$1`, original).Scan(&state)
		return state == "healthy"
	})
	var epoch int64
	if err = db.Pool.QueryRow(ctx, `SELECT active_epoch FROM trader_sync_collector_control`).Scan(&epoch); err != nil {
		t.Fatal(err)
	}
	blocker, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(ctx)
	if _, err = blocker.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('athena:account:' || $1::text,0))`, owner.ID); err != nil {
		t.Fatal(err)
	}
	insert("0x2222222222222222222222222222222222222222")
	wait("reconcile did not wait for account gate", func() bool {
		var waiting bool
		_ = db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype='advisory' AND NOT granted AND database=(SELECT oid FROM pg_database WHERE datname=current_database()) AND objid=(hashtextextended('athena:account:' || $1::text,0)&4294967295)::oid)`, owner.ID).Scan(&waiting)
		return waiting
	})
	conn := <-connections
	_ = conn.Close()
	var reason string
	var ended bool
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		_ = db.Pool.QueryRow(ctx, `SELECT ended_at IS NOT NULL,reason FROM trader_sync_collector_epochs WHERE id=$1`, epoch).Scan(&ended, &reason)
		if ended {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ended || !strings.Contains(reason, "WSS session") {
		_ = blocker.Rollback(ctx)
		cancel()
		<-running
		t.Fatal("WSS failure remained hidden behind account gate", ended, reason)
	}
	if node.latestCalls.Load() == 0 {
		t.Fatal("healthy latest fake was never used")
	}
	_ = blocker.Rollback(ctx)
	wait("closed epoch did not recover to a fresh session", func() bool {
		var active int64
		_ = db.Pool.QueryRow(ctx, `SELECT COALESCE(active_epoch,0) FROM trader_sync_collector_control`).Scan(&active)
		return active > epoch
	})
	cancel()
	select {
	case err = <-running:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("WSS watcher did not join")
	}
}

func TestCollectorStartFailureIsFatalAndPreservesCleanupError(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := db.Pool.Exec(ctx, `CREATE FUNCTION reject_epoch_start() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'injected epoch start failure';END$$;
CREATE TRIGGER reject_epoch_start BEFORE INSERT ON trader_sync_collector_epochs FOR EACH ROW EXECUTE FUNCTION reject_epoch_start();
CREATE FUNCTION reject_owner_release() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN IF NEW.owner_id IS NULL THEN RAISE EXCEPTION 'injected owner cleanup failure';END IF;RETURN NEW;END$$;
CREATE TRIGGER reject_owner_release BEFORE UPDATE ON trader_sync_collector_control FOR EACH ROW EXECUTE FUNCTION reject_owner_release()`); err != nil {
		t.Fatal(err)
	}
	var connections atomic.Int32
	up := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		connections.Add(1)
		defer conn.Close()
		for {
			var req struct {
				ID uint64 `json:"id"`
			}
			if conn.ReadJSON(&req) != nil {
				return
			}
			if conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x89"}) != nil {
				return
			}
		}
	}))
	defer server.Close()
	collector, err := NewCollector(store.NewSQLStore(db.Pool), &collectorNode{}, Config{WebSocketURL: "ws" + strings.TrimPrefix(server.URL, "http"), ReconnectMin: 10 * time.Millisecond, ReconnectMax: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	running := make(chan error, 1)
	go func() { running <- collector.Run(ctx) }()
	select {
	case err = <-running:
		if err == nil || !strings.Contains(err.Error(), "injected epoch start failure") || !strings.Contains(err.Error(), "injected owner cleanup failure") {
			t.Fatal("lost start/cleanup cause", err)
		}
		if connections.Load() != 1 {
			t.Fatal("unconfirmed epoch retried same owner", connections.Load())
		}
	case <-time.After(time.Second):
		cancel()
		<-running
		t.Fatal("start failure trapped collector in same-owner reconnect")
	}
	if _, err = db.Pool.Exec(ctx, `DROP TRIGGER reject_epoch_start ON trader_sync_collector_epochs;DROP TRIGGER reject_owner_release ON trader_sync_collector_control`); err != nil {
		t.Fatal(err)
	}
	owner, err := store.NewSQLStore(db.Pool).AcquireCollectorSession(ctx)
	if err != nil {
		t.Fatal("fatal cleanup leaked physical ownership lock", err)
	}
	if err = owner.Close(ctx); err != nil {
		t.Fatal(err)
	}
}
