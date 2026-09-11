package server

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	ts "github.com/useryege/athena/internal/tradersync"
	tsstore "github.com/useryege/athena/internal/tradersync/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestTraderSyncRuntimeComposesExplicitProxyAndBorrowedPool(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://system-proxy.invalid:3128")
	t.Setenv("HTTPS_PROXY", "http://system-proxy.invalid:3128")
	for _, proxy := range []string{"", "http://explicit-proxy.test:10809"} {
		t.Run(proxy, func(t *testing.T) {
			pool := &pgxpool.Pool{}
			s := tsstore.NewSQLStore(pool)
			runtime, e := newTraderSyncRuntime(context.Background(), ts.Config{HTTPURL: "http://127.0.0.1:1", WebSocketURL: "ws://127.0.0.1:1", SiteURL: "https://athena.test", CursorHMACKey: "stable-test-key", ProxyURL: proxy}, pool, s)
			if e != nil {
				t.Fatalf("explicit component construction failed: %v", e)
			}
			defer runtime.Close()
			if runtime.service == nil || runtime.transport == nil {
				t.Fatal("runtime dependencies missing")
			}
			if proxy == "" && runtime.transport.Proxy != nil {
				t.Fatal("direct transport inherited environment proxy")
			}
			if proxy != "" {
				req, _ := http.NewRequest(http.MethodGet, "https://gamma-api.polymarket.com/markets", nil)
				u, e := runtime.transport.Proxy(req)
				if e != nil || u.String() != proxy {
					t.Fatal("explicit proxy missing", u, e)
				}
			}
			// Closing this not-started runtime may close its own clients, never this
			// deliberately uninitialized borrowed pool (which would panic if closed).
			if e = runtime.Close(); e != nil {
				t.Fatal(e)
			}
		})
	}
}

func TestTraderSyncProcessLifecycleOutlivesListenerContext(t *testing.T) {
	root, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{}, 1)
	stopped := make(chan struct{})
	server := &AthenaServer{traderSyncRuntime: &traderSyncRuntime{run: func(ctx context.Context) error { started <- struct{}{}; <-ctx.Done(); close(stopped); return nil }}}
	server.startProcessServices(root)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("process runtime was not started")
	}
	listener, closeListener := context.WithCancel(root)
	closeListener()
	_ = listener
	select {
	case <-stopped:
		t.Fatal("listener cancellation stopped process runtime")
	default:
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("process cancellation did not stop runtime")
	}
	select {
	case <-server.traderSyncDone:
	case <-time.After(time.Second):
		t.Fatal("runtime exit was not joined/published")
	}
}

func TestTraderSyncFatalIsReturnedBeforeListenerRestart(t *testing.T) {
	fatal := errors.New("collector epoch persistence unknown")
	s := &AthenaServer{traderSyncRuntime: &traderSyncRuntime{run: func(context.Context) error { return fatal }}}
	s.startProcessServices(context.Background())
	select {
	case <-s.traderSyncDone:
	case <-time.After(time.Second):
		t.Fatal("fatal did not reach process owner")
	}
	// No listener/service construction is allowed after the owned runtime exits.
	if e := s.Run(context.Background(), nil); !errors.Is(e, fatal) {
		t.Fatal("fatal was swallowed as a listener restart", e)
	}
	s.processWorkers.Wait()
}

func TestTraderSyncInvalidStartupReturnsErrorBeforeResources(t *testing.T) {
	if server, e := NewServer(context.Background(), AthenaServerOpts{}); e == nil || server != nil {
		t.Fatal("missing required configuration accepted", server, e)
	}
}

func TestTraderSyncEarlyFatalClosesListenerAndGatewayConnection(t *testing.T) {
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer listener.Close()
	conn, e := grpc.Dial(listener.Addr().String(), grpc.WithInsecure())
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	fatal := errors.New("background fatal before listener Run")
	done := make(chan struct{})
	close(done)
	s := &AthenaServer{traderSyncDone: done, traderSyncErr: fatal}
	listeners := &Listeners{Main: listener, GatewayConn: conn}
	if e = s.Run(context.Background(), listeners); !errors.Is(e, fatal) {
		t.Fatal(e)
	}
	if conn.GetState() != connectivity.Shutdown || listeners.Main != nil {
		t.Fatal("early fatal leaked listener resources", conn.GetState())
	}
}
func TestListenersCloseJoinsGatewayWhenMainAlreadyClosed(t *testing.T) {
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	conn, e := grpc.Dial(listener.Addr().String(), grpc.WithInsecure())
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	listener.Close()
	l := &Listeners{Main: listener, GatewayConn: conn}
	if e = l.Close(); e != nil {
		t.Fatal("already-closed main hid gateway cleanup", e)
	}
	if conn.GetState() != connectivity.Shutdown {
		t.Fatal("gateway remains open")
	}
	if e = l.Close(); e != nil {
		t.Fatal(e)
	}
}

func TestTraderSyncCompletedRuntimeCannotRestartListenerLoop(t *testing.T) {
	done := make(chan struct{})
	close(done)
	s := &AthenaServer{traderSyncDone: done}
	if e := s.Run(context.Background(), nil); e != nil {
		t.Fatal(e)
	}
	if !s.TerminateRequested() {
		t.Fatal("already-joined runtime allowed CLI listener restart")
	}
}
