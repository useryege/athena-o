package server

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/jackc/pgx/v5/pgxpool"
	ts "github.com/useryege/athena/internal/tradersync"
	tsstore "github.com/useryege/athena/internal/tradersync/store"
	pm "github.com/useryege/athena/util/polymarket"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// traderSyncRuntime owns clients and transport, but borrows the account pool and
// the store whose grant hook was already installed by NewServer.
type traderSyncRuntime struct {
	service   *ts.Service
	run       func(context.Context) error
	client    *ethclient.Client
	transport *http.Transport
	once      sync.Once
	closeErr  error
}

func newTraderSyncRuntime(ctx context.Context, cfg ts.Config, pool *pgxpool.Pool, traderStore *tsstore.SQLStore) (result *traderSyncRuntime, resultErr error) {
	if e := cfg.Validate(); e != nil {
		return nil, e
	}
	if pool == nil || traderStore == nil {
		return nil, fmt.Errorf("Trader Sync account pool and existing hooked store required")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	if cfg.ProxyURL != "" {
		u, e := url.Parse(cfg.ProxyURL)
		if e != nil {
			return nil, e
		}
		transport.Proxy = http.ProxyURL(u)
	}
	r := &traderSyncRuntime{transport: transport}
	defer func() {
		if resultErr != nil {
			_ = r.Close()
		}
	}()
	httpClient := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	raw, e := rpc.DialOptions(ctx, cfg.HTTPURL, rpc.WithHTTPClient(httpClient))
	if e != nil {
		return nil, fmt.Errorf("create Trader Sync HTTP RPC: %w", e)
	}
	r.client = ethclient.NewClient(raw)
	source := ts.NewSourceRPC(r.client)
	gamma, e := pm.NewGammaClient(pm.GammaConfig{HTTPClient: httpClient, Timeout: 5 * time.Second})
	if e != nil {
		return nil, e
	}
	profiles := pm.NewProfileAdapter(transport)
	resolver, e := ts.NewTargetResolver(pool, profiles, traderStore.RequireGrantTx, traderStore.ResolveContextTx)
	if e != nil {
		return nil, e
	}
	collector, e := ts.NewCollector(traderStore, source, cfg)
	if e != nil {
		return nil, e
	}
	subscriptions, e := ts.NewSubscriptionService(pool, traderStore, resolver, collector)
	if e != nil {
		return nil, e
	}
	if e = traderStore.ConfigureActivities(cfg.SiteURL); e != nil {
		return nil, e
	}
	// Initial resolution and late enrichment share this exact four-slot resolver.
	metadata := ts.NewMetadataResolver(gamma, source, traderStore)
	projector, e := ts.NewProjector(traderStore, source, ts.NewVersionVerifier(source), metadata, cfg.Projector)
	if e != nil {
		return nil, e
	}
	directory := ts.NewDirectoryRefresher(traderStore, gamma, cfg.OnError)
	r.service, e = ts.NewService(cfg, ts.Dependencies{Pool: pool, Resolver: resolver, Subscriptions: subscriptions, Collector: collector, Projector: projector, Directory: directory})
	if e != nil {
		return nil, e
	}
	r.run = r.service.Run
	return r, nil
}
func (r *traderSyncRuntime) Close() error {
	if r == nil {
		return nil
	}
	r.once.Do(func() {
		if r.service != nil {
			r.closeErr = r.service.Close()
		}
		if r.client != nil {
			r.client.Close()
		}
		if r.transport != nil {
			r.transport.CloseIdleConnections()
		}
	})
	return r.closeErr
}

func (s *AthenaServer) startProcessServices(ctx context.Context) {
	s.traderSyncDone = make(chan struct{})
	s.processWorkers.Add(1)
	go func() {
		defer s.processWorkers.Done()
		s.traderSyncErr = s.traderSyncRuntime.run(ctx)
		close(s.traderSyncDone)
	}()
}
