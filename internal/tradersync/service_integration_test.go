//go:build integration

package tradersync

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"github.com/useryege/athena/internal/tradersync/store"
	pm "github.com/useryege/athena/util/polymarket"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type unavailableDirectory struct{}

func (unavailableDirectory) ListComboMarkets(context.Context, string, int) (pm.ComboMarketPage, error) {
	return pm.ComboMarketPage{}, errors.New("fixture directory provider unavailable")
}

func TestRuntimeOwnsSessionJoinsAndSeparatesDirectoryErrors(t *testing.T) {
	for _, mode := range []string{"cancel", "projector_fatal"} {
		t.Run(mode, func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			up := websocket.Upgrader{}
			wss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, e := up.Upgrade(w, r, nil)
				if e != nil {
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
						result = fmt.Sprint(req.ID)
					}
					if conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}) != nil {
						return
					}
				}
			}))
			defer wss.Close()
			cfg := Config{HTTPURL: "http://127.0.0.1:1", WebSocketURL: "ws" + strings.TrimPrefix(wss.URL, "http"), SiteURL: "https://athena.test", CursorHMACKey: "runtime-test-key"}
			storage := store.NewSQLStore(db.Pool)
			runtimeOwner, ownerErr := storage.AcquireRuntimeSession(ctx)
			if ownerErr != nil {
				t.Fatal(ownerErr)
			}
			storage, ownerErr = storage.WithRuntime(runtimeOwner.RuntimeToken())
			if ownerErr != nil {
				t.Fatal(ownerErr)
			}
			if ownerErr = runtimeOwner.RecoverPending(ctx); ownerErr != nil {
				t.Fatal(ownerErr)
			}
			if e := storage.ConfigureActivities(cfg.SiteURL); e != nil {
				t.Fatal(e)
			}
			node := &collectorNode{}
			collector, e := NewCollector(storage, node, cfg)
			if e != nil {
				t.Fatal(e)
			}
			projector, e := NewProjector(storage, node, projectorVersionFake{}, &projectorMetadataFake{}, ProjectorConfig{Interval: 20 * time.Millisecond})
			if e != nil {
				t.Fatal(e)
			}
			directoryErrors := make(chan error, 10)
			directory := NewDirectoryRefresher(storage, unavailableDirectory{}, func(e error) {
				select {
				case directoryErrors <- e:
				default:
				}
			})
			subscriptions, e := NewSubscriptionService(db.Pool, storage, &TargetResolver{}, collector)
			if e != nil {
				t.Fatal(e)
			}
			service, e := NewService(cfg, Dependencies{Pool: db.Pool, Resolver: &TargetResolver{}, Subscriptions: subscriptions, Collector: collector, Projector: projector, Directory: directory})
			if e != nil {
				t.Fatal(e)
			}
			done := make(chan error, 1)
			runtimeCfg := cfg
			runtimeCfg.AccountStateDSN = db.DSN
			runtimeCfg.ShutdownTimeout = 5 * time.Second
			runtime, e := NewRuntime(runtimeCfg, runtimeTestRPC(), runtimeTestDeps())
			if e != nil {
				t.Fatal(e)
			}
			runtime.owner = runtimeOwner
			runtime.service = service
			runtime.startWorkers(ctx, []runtimeWorker{{"collector", func(ctx context.Context) error { return collector.Run(ctx, runtimeOwner) }}, {"projector", projector.Run}, {"directory", directory.Run}, {"owner", runtime.checkOwner}})
			go func() { done <- runtime.Wait() }()
			select {
			case <-directoryErrors:
			case <-time.After(2 * time.Second):
				t.Fatal("directory failure was hidden")
			}
			select {
			case e := <-done:
				t.Fatal("directory failure stopped healthy runtime", e)
			default:
			}
			if mode == "projector_fatal" {
				if _, e = db.Pool.Exec(ctx, `ALTER TABLE trader_sync_source_candidates RENAME TO injected_missing_candidates`); e != nil {
					t.Fatal(e)
				}
			} else {
				cancel()
			}
			select {
			case e = <-done:
				if mode == "projector_fatal" && (e == nil || !strings.Contains(e.Error(), "trader_sync_source_candidates")) {
					t.Fatal("fatal not propagated", e)
				}
				if mode == "cancel" && e != nil {
					t.Fatal(e)
				}
			case <-time.After(6 * time.Second):
				t.Fatal("runtime failed to join all workers")
			}
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if e = runtime.Shutdown(shutdownCtx); e != nil {
				t.Fatal(e)
			}
			if e = db.Pool.Ping(context.Background()); e != nil {
				t.Fatal("service closed borrowed pool", e)
			}
			var active bool
			if e = db.Pool.QueryRow(context.Background(), `SELECT active_epoch IS NOT NULL FROM trader_sync_collector_control WHERE singleton`).Scan(&active); e != nil || active {
				t.Fatal("joined runtime left active epoch", active, e)
			}
		})
	}
}
