//go:build integration

package tradersync

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/accountstate/schema"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"github.com/useryege/athena/internal/tradersync/store"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func runtimeAddress(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}
func newIntegrationRuntime(t *testing.T, dsn string) (*Runtime, *atomic.Int32) {
	t.Helper()
	cfg := runtimeTestConfig()
	cfg.AccountStateDSN = dsn
	cfg.ShutdownTimeout = 3 * time.Second
	cfg.Projector.Interval = 20 * time.Millisecond
	cfg.ReconnectMin = 20 * time.Millisecond
	cfg.ReconnectMax = 50 * time.Millisecond
	calls := &atomic.Int32{}
	cfg.OnError = func(error) { calls.Add(1) }
	rpcCfg := runtimeTestRPC()
	rpcCfg.ListenAddress = runtimeAddress(t)
	r, err := NewRuntime(cfg, rpcCfg, runtimeTestDeps())
	require.NoError(t, err)
	return r, calls
}
func TestRuntimeStartVerifiesSchemaWithoutDDL(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	r, _ := newIntegrationRuntime(t, db.DSN)
	require.ErrorIs(t, r.Start(context.Background()), schema.ErrVersions)
	var missing bool
	require.NoError(t, db.Pool.QueryRow(context.Background(), "SELECT to_regclass('public.goose_db_version') IS NULL").Scan(&missing))
	require.True(t, missing)
	require.False(t, r.Ready())
}
func TestRuntimeOfflineReadyRejectsSecondBeforeWorkerInitialization(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	r, sourceErrors := newIntegrationRuntime(t, db.DSN)
	require.NoError(t, r.Start(context.Background()))
	require.ErrorContains(t, r.Start(context.Background()), "already started")
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, r.Shutdown(ctx))
	})
	require.Eventually(t, func() bool { return sourceErrors.Load() > 0 }, time.Second, 10*time.Millisecond)
	require.True(t, r.Ready())
	_, _, _, available := r.service.deps.Collector.RawSnapshot()
	require.False(t, available)
	conn, err := grpc.NewClient(r.rpcCfg.ListenAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	check, err := healthpb.NewHealthClient(conn).Check(context.Background(), &healthpb.HealthCheckRequest{Service: HealthServiceName})
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, check.Status)
	require.Same(t, r.service.deps.Collector, r.service.deps.Subscriptions.baseline)
	second, _ := newIntegrationRuntime(t, db.DSN)
	var initialized atomic.Bool
	second.deps.NewRPCServer = func(*Service, func() bool) *grpc.Server { initialized.Store(true); return grpc.NewServer() }
	err = second.Start(context.Background())
	require.ErrorContains(t, err, "another Trader Sync runtime")
	require.True(t, initialized.Load(), "health RPC server is created before ownership")
	require.Nil(t, second.client)
	require.Nil(t, second.service.deps.Collector)
	require.Nil(t, second.service.deps.Projector)
	require.Nil(t, second.service.deps.Directory)
	require.True(t, r.Ready())
}
func TestRuntimeOwnerLossCancelsAllWorkers(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	r, _ := newIntegrationRuntime(t, db.DSN)
	require.NoError(t, r.Start(context.Background()))
	_, err := db.Pool.Exec(context.Background(), `SELECT pg_terminate_backend(pid) FROM pg_locks WHERE locktype='advisory' AND granted AND classid=((hashtextextended('athena:trader-sync:collector',0)>>32)&4294967295)::oid AND objid=(hashtextextended('athena:trader-sync:collector',0)&4294967295)::oid`)
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { done <- r.Wait() }()
	select {
	case err = <-done:
		require.ErrorIs(t, err, store.ErrRuntimeFenced)
	case <-time.After(3 * time.Second):
		t.Fatal("owner loss not observed")
	}
	require.False(t, r.Ready())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = r.Shutdown(ctx)
	select {
	case <-r.closed:
	case <-ctx.Done():
		t.Fatal("owner loss did not join workers")
	}
	require.False(t, r.service.deps.Collector.running.Load())
	require.NoError(t, db.Pool.Ping(context.Background()))
}
func TestRuntimeCannotReleaseOwnerBeforeBlockedWorkerJoins(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	r, _ := newIntegrationRuntime(t, db.DSN)
	owner, err := store.NewSQLStore(db.Pool).AcquireRuntimeSession(context.Background())
	require.NoError(t, err)
	r.owner = owner
	blocked := make(chan struct{})
	r.startWorkers(context.Background(), []runtimeWorker{{"collector", func(context.Context) error { <-blocked; return nil }}})
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, r.Shutdown(ctx), context.DeadlineExceeded)
	_, err = store.NewSQLStore(db.Pool).AcquireRuntimeSession(context.Background())
	require.ErrorContains(t, err, "another Trader Sync runtime")
	require.NoError(t, owner.Check(context.Background()))
	close(blocked)
	// Deadline cleanup may fail, but it is permitted only after the worker joins.
	_ = r.Shutdown(context.Background())
	successor, err := store.NewSQLStore(db.Pool).AcquireRuntimeSession(context.Background())
	require.NoError(t, err)
	require.NoError(t, successor.CloseAfterWorkers(context.Background()))
}
func TestRuntimeFatalWorkerCancelsPeers(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	r, _ := newIntegrationRuntime(t, db.DSN)
	require.NoError(t, r.Start(context.Background()))
	_, err := db.Pool.Exec(context.Background(), `ALTER TABLE trader_sync_source_candidates RENAME TO injected_missing_candidates`)
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { done <- r.Wait() }()
	select {
	case err = <-done:
		require.True(t, strings.Contains(err.Error(), "trader_sync_source_candidates"))
	case <-time.After(3 * time.Second):
		t.Fatal("projector fatal did not end runtime")
	}
	require.False(t, r.Ready())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, r.Shutdown(ctx))
}

func TestRuntimeWSSDisconnectDegradesCollectorWithoutLosingReady(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	up := websocket.Upgrader{}
	connected := make(chan *websocket.Conn, 1)
	var disconnected atomic.Bool
	wss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if disconnected.Load() {
			http.Error(w, "fixture offline", 503)
			return
		}
		conn, err := up.Upgrade(w, req, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		connected <- conn
		for {
			var request struct {
				ID     uint64 `json:"id"`
				Method string `json:"method"`
			}
			if conn.ReadJSON(&request) != nil {
				return
			}
			var result any = true
			if request.Method == "eth_chainId" {
				result = "0x89"
			}
			if request.Method == "eth_subscribe" {
				result = fmt.Sprint(request.ID)
			}
			if conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result}) != nil {
				return
			}
		}
	}))
	defer wss.Close()
	r, _ := newIntegrationRuntime(t, db.DSN)
	r.cfg.WebSocketURL = "ws" + strings.TrimPrefix(wss.URL, "http")
	r.cfg.ProxyURL = ""
	// Only the source transport is substituted; the real WSS and runtime/SQL
	// lifecycle run normally. Deny HTTP so this test cannot call public providers.
	previous := http.DefaultTransport
	denied := previous.(*http.Transport).Clone()
	denied.DialContext = func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("fixture HTTP source offline")
	}
	http.DefaultTransport = denied
	err := r.Start(context.Background())
	http.DefaultTransport = previous
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, r.Shutdown(ctx))
	})
	var conn *websocket.Conn
	select {
	case conn = <-connected:
	case <-time.After(time.Second):
		t.Fatal("collector never connected")
	}
	require.Eventually(t, func() bool { _, _, _, available := r.service.deps.Collector.RawSnapshot(); return available }, time.Second, 10*time.Millisecond)
	disconnected.Store(true)
	require.NoError(t, conn.Close())
	require.Eventually(t, func() bool { _, _, _, available := r.service.deps.Collector.RawSnapshot(); return !available }, time.Second, 10*time.Millisecond)
	require.True(t, r.Ready())
	client, err := grpc.NewClient(r.rpcCfg.ListenAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer client.Close()
	response, err := healthpb.NewHealthClient(client).Check(context.Background(), &healthpb.HealthCheckRequest{Service: HealthServiceName})
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, response.Status)
}

func TestRuntimeOfflineListPauseResumeCancelKeepsPendingBaseline(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	member, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	require.NoError(t, err)
	var subID string
	require.NoError(t, db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,decode(repeat('44',20),'hex'),'enabled','pending_baseline','{"displayName":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"avatar":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"profileURL":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"}}'::jsonb) RETURNING id`, member.ID).Scan(&subID))
	r, _ := newIntegrationRuntime(t, db.DSN)
	require.NoError(t, r.Start(ctx))
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, r.Shutdown(cleanup))
	})
	page, err := r.service.ListSubscriptions(ctx, member.ID, 10, "", tm.SubscriptionFilter{})
	require.NoError(t, err)
	require.Len(t, page.Subscriptions, 1)
	sub := page.Subscriptions[0]
	for _, action := range []string{"pause", "resume", "cancel"} {
		sub, err = r.service.ChangeSubscription(ctx, member.ID, action, tm.ChangeInput{SubscriptionID: subID, RequestID: "offline-" + action, ExpectedRevision: sub.Revision})
		require.NoError(t, err)
		if action == "resume" {
			require.Equal(t, "pending_baseline", sub.ObservationState)
			require.Nil(t, sub.EffectiveAt)
			var pending int
			require.NoError(t, db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_baseline_attempts WHERE subscription_id=$1 AND state='pending' AND collector_epoch IS NULL`, subID).Scan(&pending))
			require.Equal(t, 1, pending)
		}
	}
	require.Equal(t, "cancelled", sub.DesiredState)
	require.True(t, r.Ready())
}
