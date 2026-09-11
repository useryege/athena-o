//go:build integration

package tradersync

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountcredentials"
	ac "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"github.com/useryege/athena/internal/tradersync/store"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func quietObservationWSS(t *testing.T) *httptest.Server {
	t.Helper()
	up := websocket.Upgrader{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, e := up.Upgrade(w, r, nil)
		if e != nil {
			return
		}
		defer c.Close()
		for {
			var req struct {
				ID     uint64 `json:"id"`
				Method string `json:"method"`
			}
			if c.ReadJSON(&req) != nil {
				return
			}
			result := any(true)
			if req.Method == "eth_chainId" {
				result = "0x89"
			}
			if req.Method == "eth_subscribe" {
				result = fmt.Sprintf("sub-%d", req.ID)
			}
			if c.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}) != nil {
				return
			}
		}
	}))
	t.Cleanup(s.Close)
	return s
}
func awaitObservation(t *testing.T, limit time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.NewTimer(limit)
	defer deadline.Stop()
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		if fn() {
			return
		}
		select {
		case <-deadline.C:
			t.Fatal("observation condition did not become true")
		case <-tick.C:
		}
	}
}

func TestObservationCheckpointBudgetRotatesPastBusyOwner(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	accounts := ac.NewSQLStore(db.Pool)
	a, e := accounts.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	b, _, e := accounts.RegisterExternalAccount(ctx, accountcredentials.IdentityProviderGoogle, "checkpoint-owner", "checkpoint@example.test", "checkpoint-owner", accountcredentials.ApplicationRealmMember)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `UPDATE account_module_access SET access_level='read_write' WHERE account_id=$1 AND module='trader_sync'`, b.ID); e != nil {
		t.Fatal(e)
	}
	// These 101 direct relationships are a health-budget fixture, not a subscription
	// quota assertion. Production Collector performs all filtering/baseline/health IO.
	for i := 0; i < 101; i++ {
		owner := b.ID
		if i == 0 {
			owner = a.ID
		}
		wallet := common.BytesToAddress([]byte{byte(i + 1)})
		if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,$2,'enabled','pending_baseline','{"DisplayName":{"Availability":"unavailable","ReasonCode":"fixture_not_queried","Source":"fixture"},"Avatar":{"Availability":"unavailable","ReasonCode":"fixture_not_queried","Source":"fixture"},"ProfileURL":{"Availability":"unavailable","ReasonCode":"fixture_not_queried","Source":"fixture"}}')`, owner, wallet.Bytes()); e != nil {
			t.Fatal(e)
		}
	}
	wss := quietObservationWSS(t)
	node := &collectorNode{}
	collector, e := NewCollector(store.NewSQLStore(db.Pool), node, Config{WebSocketURL: "ws" + strings.TrimPrefix(wss.URL, "http")})
	if e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { done <- collector.Run(ctx) }()
	defer func() {
		cancel()
		select {
		case e := <-done:
			if e != nil {
				t.Error(e)
			}
		case <-time.After(6 * time.Second):
			t.Error("collector did not join")
		}
	}()
	awaitObservation(t, 10*time.Second, func() bool {
		var n int
		if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_monitor_intervals WHERE ended_at IS NULL`).Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n == 101
	})
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_monitor_intervals SET id='00000000-0000-0000-0000-000000000001' WHERE owner_id=$1`, a.ID); e != nil {
		t.Fatal(e)
	}
	gate, e := txgate.AcquireAccountSession(ctx, db.Pool, a.ID)
	if e != nil {
		t.Fatal(e)
	}
	defer gate.Release(context.Background())
	awaitObservation(t, 33*time.Second, func() bool {
		var n int
		if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_monitor_intervals WHERE owner_id=$1 AND last_reliable_at IS NOT NULL`, b.ID).Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n == 100
	})
	var busyCount int
	if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_monitor_intervals WHERE owner_id=$1 AND last_reliable_at IS NOT NULL`, a.ID).Scan(&busyCount); e != nil || busyCount != 0 {
		t.Fatal("busy owner unexpectedly bypassed its gate", busyCount, e)
	}
	if e = gate.Release(ctx); e != nil {
		t.Fatal(e)
	}
	awaitObservation(t, 13*time.Second, func() bool {
		var n int
		if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_monitor_intervals WHERE last_reliable_at IS NOT NULL`).Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n == 101
	})
	if node.finalityCalls.Load() != 0 {
		t.Fatal("health checkpoint created finality work")
	}
}

func TestCheckpointWriteFailureAndCommitConnectionLossPreserveHealthyEpoch(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,$2,'enabled','pending_baseline','{"DisplayName":{"Availability":"unavailable","ReasonCode":"fixture_not_queried","Source":"fixture"},"Avatar":{"Availability":"unavailable","ReasonCode":"fixture_not_queried","Source":"fixture"},"ProfileURL":{"Availability":"unavailable","ReasonCode":"fixture_not_queried","Source":"fixture"}}')`, owner.ID, common.HexToAddress("0x55").Bytes()); e != nil {
		t.Fatal(e)
	}
	wss := quietObservationWSS(t)
	reported := make(chan error, 10)
	collector, e := NewCollector(store.NewSQLStore(db.Pool), &collectorNode{}, Config{WebSocketURL: "ws" + strings.TrimPrefix(wss.URL, "http"), OnError: func(err error) {
		select {
		case reported <- err:
		default:
		}
	}})
	if e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { done <- collector.Run(ctx) }()
	defer func() {
		cancel()
		select {
		case e := <-done:
			if e != nil {
				t.Error(e)
			}
		case <-time.After(6 * time.Second):
			t.Error("collector did not join")
		}
	}()
	var original time.Time
	var epoch int64
	awaitObservation(t, 24*time.Second, func() bool {
		var point *time.Time
		if e = db.Pool.QueryRow(ctx, `SELECT max(last_reliable_at) FROM trader_sync_monitor_intervals`).Scan(&point); e != nil {
			t.Fatal(e)
		}
		if point == nil {
			return false
		}
		original = *point
		return true
	})
	if e = db.Pool.QueryRow(ctx, `SELECT active_epoch FROM trader_sync_collector_control WHERE singleton`).Scan(&epoch); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `CREATE FUNCTION fail_checkpoint_commit() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'checkpoint commit fixture failure'; END$$; CREATE CONSTRAINT TRIGGER checkpoint_commit_fault AFTER UPDATE OF last_reliable_at ON trader_sync_monitor_intervals DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION fail_checkpoint_commit()`); e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"write_failure", "commit_connection_loss"} {
		if mode == "commit_connection_loss" {
			if _, e = db.Pool.Exec(ctx, `CREATE OR REPLACE FUNCTION fail_checkpoint_commit() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN PERFORM pg_terminate_backend(pg_backend_pid()); RETURN NEW; END$$`); e != nil {
				t.Fatal(e)
			}
		}
		select {
		case err := <-reported:
			if !strings.Contains(err.Error(), "checkpoint persistence") {
				t.Fatal("unexpected error category", err)
			}
		case <-time.After(13 * time.Second):
			t.Fatal("checkpoint persistence fault not reported", mode)
		}
		var point time.Time
		var current int64
		if e = db.Pool.QueryRow(ctx, `SELECT max(last_reliable_at) FROM trader_sync_monitor_intervals`).Scan(&point); e != nil || !point.Equal(original) {
			t.Fatal("unconfirmed point became API fact", mode, point, e)
		}
		if e = db.Pool.QueryRow(ctx, `SELECT active_epoch FROM trader_sync_collector_control WHERE singleton`).Scan(&current); e != nil || current != epoch {
			t.Fatal("metadata-only checkpoint fault closed healthy observation", mode, current, e)
		}
		select {
		case err := <-done:
			t.Fatal("checkpoint fault ended collector", mode, err)
		default:
		}
	}
	if _, e = db.Pool.Exec(ctx, `DROP TRIGGER checkpoint_commit_fault ON trader_sync_monitor_intervals`); e != nil {
		t.Fatal(e)
	}
	awaitObservation(t, 13*time.Second, func() bool {
		var point time.Time
		if e = db.Pool.QueryRow(ctx, `SELECT max(last_reliable_at) FROM trader_sync_monitor_intervals`).Scan(&point); e != nil {
			t.Fatal(e)
		}
		return point.After(original)
	})
}

// Drop only PostgreSQL's actual COMMIT CommandComplete, after the backend has
// committed. This differs from terminating a connection inside a deferred trigger,
// which rolls the transaction back. No production fault hook or DB reset is used.
type checkpointCommitEvidence struct {
	Frame string
	At    time.Time
}
type checkpointWireConn struct {
	net.Conn
	pending          []byte
	checkpoint, drop atomic.Bool
	checkpointAt     atomic.Pointer[time.Time]
	observed         chan checkpointCommitEvidence
}

func checkpointTime(at *time.Time) time.Time {
	if at == nil {
		return time.Time{}
	}
	return *at
}

func (c *checkpointWireConn) Read(out []byte) (int, error) {
	if len(c.pending) == 0 {
		header := make([]byte, 5)
		if _, e := io.ReadFull(c.Conn, header); e != nil {
			return 0, e
		}
		size := int(binary.BigEndian.Uint32(header[1:]))
		if size < 4 || size > 32<<20 {
			return 0, fmt.Errorf("invalid PostgreSQL frame length %d", size)
		}
		payload := make([]byte, size-4)
		if _, e := io.ReadFull(c.Conn, payload); e != nil {
			return 0, e
		}
		if header[0] == 'C' && string(payload) == "COMMIT\x00" && c.drop.CompareAndSwap(true, false) {
			select {
			case c.observed <- checkpointCommitEvidence{Frame: string(payload), At: checkpointTime(c.checkpointAt.Load())}:
			default:
			}
			c.Conn.Close()
			return 0, io.EOF
		}
		c.pending = append(header, payload...)
	}
	n := copy(out, c.pending)
	c.pending = c.pending[n:]
	return n, nil
}

type checkpointCommitTracer struct{ armed atomic.Bool }

func (t *checkpointCommitTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	c, ok := conn.PgConn().Conn().(*checkpointWireConn)
	if !ok {
		return ctx
	}
	if strings.Contains(data.SQL, "SET last_reliable_at") {
		c.checkpoint.Store(true)
		if at, ok := data.Args[0].(pgtype.Timestamptz); ok && at.Valid {
			c.checkpointAt.Store(&at.Time)
		}
	}
	if strings.EqualFold(strings.TrimSpace(data.SQL), "commit") && c.checkpoint.Swap(false) && t.armed.CompareAndSwap(true, false) {
		c.drop.Store(true)
	}
	if strings.EqualFold(strings.TrimSpace(data.SQL), "rollback") {
		c.checkpoint.Store(false)
	}
	return ctx
}
func (*checkpointCommitTracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}
func TestCheckpointServerCommittedButAcknowledgementLost(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state,target_display)VALUES($1,$2,'enabled','pending_baseline','{}')`, owner.ID, common.HexToAddress("0x91").Bytes()); e != nil {
		t.Fatal(e)
	}
	tracer := &checkpointCommitTracer{}
	observed := make(chan checkpointCommitEvidence, 1)
	config, e := pgxpool.ParseConfig(db.DSN)
	if e != nil {
		t.Fatal(e)
	}
	config.ConnConfig.Tracer = tracer
	config.ConnConfig.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		c, e := (&net.Dialer{}).DialContext(ctx, network, address)
		if e != nil {
			return nil, e
		}
		return &checkpointWireConn{Conn: c, observed: observed}, nil
	}
	pool, e := pgxpool.NewWithConfig(ctx, config)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(pool.Close)
	wss := quietObservationWSS(t)
	reported := make(chan error, 10)
	collector, e := NewCollector(store.NewSQLStore(pool), &collectorNode{}, Config{WebSocketURL: "ws" + strings.TrimPrefix(wss.URL, "http"), OnError: func(e error) {
		select {
		case reported <- e:
		default:
		}
	}})
	if e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { done <- collector.Run(ctx) }()
	defer func() {
		cancel()
		select {
		case e := <-done:
			if e != nil {
				t.Error(e)
			}
		case <-time.After(6 * time.Second):
			t.Error("collector did not join")
		}
	}()
	var epoch int64
	awaitObservation(t, 5*time.Second, func() bool {
		var healthy int
		e := db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_subscriptions WHERE observation_state='healthy'`).Scan(&healthy)
		return e == nil && healthy == 1
	})
	if e = db.Pool.QueryRow(ctx, `SELECT active_epoch FROM trader_sync_collector_control WHERE singleton`).Scan(&epoch); e != nil {
		t.Fatal(e)
	}
	// The first latest tick can precede the first nonce-matched pong. Wait for
	// one real eligible checkpoint before selecting the following transaction.
	var original time.Time
	awaitObservation(t, 24*time.Second, func() bool {
		var at *time.Time
		e := db.Pool.QueryRow(ctx, `SELECT max(last_reliable_at) FROM trader_sync_monitor_intervals`).Scan(&at)
		if e != nil {
			t.Fatal(e)
		}
		if at == nil {
			return false
		}
		original = *at
		return true
	})
	tracer.armed.Store(true)
	var intercepted checkpointCommitEvidence
	select {
	case intercepted = <-observed:
		t.Logf("suppressed actual PostgreSQL CommandComplete payload=%q observed_at=%s", intercepted.Frame, intercepted.At.Format(time.RFC3339Nano))
	case <-time.After(13 * time.Second):
		t.Fatal("actual checkpoint COMMIT response not intercepted")
	}
	select {
	case e = <-reported:
		if !strings.Contains(e.Error(), "checkpoint persistence") {
			t.Fatal(e)
		}
		t.Logf("actual client commit result: %v", e)
	case <-time.After(3 * time.Second):
		t.Fatal("unknown commit was not reported")
	}
	var point *time.Time
	var current int64
	var intervals int
	if e = db.Pool.QueryRow(ctx, `SELECT max(last_reliable_at),count(*) FROM trader_sync_monitor_intervals`).Scan(&point, &intervals); e != nil || point == nil || !point.After(original) || !point.Equal(intercepted.At.Truncate(time.Microsecond)) || intervals != 1 {
		t.Fatal("server commit not durable", point, intervals, e)
	}
	if e = db.Pool.QueryRow(ctx, `SELECT active_epoch FROM trader_sync_collector_control WHERE singleton`).Scan(&current); e != nil || current != epoch {
		t.Fatal("commit ACK loss closed healthy epoch", current, epoch, e)
	}
	var ended bool
	if e = db.Pool.QueryRow(ctx, `SELECT ended_at IS NOT NULL FROM trader_sync_collector_epochs WHERE id=$1`, epoch).Scan(&ended); e != nil || ended {
		t.Fatal("healthy epoch ended", ended, e)
	}
	t.Logf("independent connection confirms committed checkpoint=%s epoch=%d intervals=%d", point.Format(time.RFC3339Nano), epoch, intervals)
}
