//go:build integration

package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/jackc/pgx/v5/pgxpool"
	ts "github.com/useryege/athena/internal/tradersync"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	tsstore "github.com/useryege/athena/internal/tradersync/store"
	tsgrpc "github.com/useryege/athena/internal/tradersync/transport"
	pm "github.com/useryege/athena/util/polymarket"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"
)

// gatewayServiceFixture is test-only domain composition for loopback provider
// fixtures. The production API never constructs this; its facade uses real gRPC.
type gatewayServiceFixture struct {
	service   *ts.Service
	owner     *tsstore.RuntimeSession
	storage   *tsstore.SQLStore
	run       func(context.Context) error
	client    *ethclient.Client
	transport *http.Transport
	once      sync.Once
	closeErr  error
}

func newGatewayServiceFixture(ctx context.Context, cfg ts.Config, pool *pgxpool.Pool, traderStore *tsstore.SQLStore) (result *gatewayServiceFixture, resultErr error) {
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
	owner, e := traderStore.AcquireRuntimeSession(ctx)
	if e != nil {
		return nil, e
	}
	r := &gatewayServiceFixture{transport: transport, owner: owner}
	gate, e := tsstore.NewRuntimeWriteGate(pool, owner.RuntimeToken())
	if e != nil {
		r.Close()
		return nil, e
	}
	traderStore, e = tsstore.NewRuntimeSQLStore(pool, gate)
	if e != nil {
		r.Close()
		return nil, e
	}
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
	resolver, e := ts.NewTargetResolver(traderStore, profiles, traderStore.RequireGrantTx, traderStore.ResolveContextTx)
	if e != nil {
		return nil, e
	}
	collector, e := ts.NewCollector(traderStore, source, cfg)
	if e != nil {
		return nil, e
	}
	subscriptions, e := ts.NewSubscriptionService(traderStore, traderStore, resolver, collector)
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
	r.storage = traderStore
	r.run = func(ctx context.Context) error {
		group, cctx := errgroup.WithContext(ctx)
		for _, worker := range []func(context.Context) error{func(ctx context.Context) error { return collector.Run(ctx, owner) }, projector.Run, directory.Run} {
			group.Go(func() error {
				err := worker(cctx)
				if cctx.Err() != nil && (err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
					return nil
				}
				return err
			})
		}
		return group.Wait()
	}
	return r, nil
}
func (r *gatewayServiceFixture) Close() error {
	if r == nil {
		return nil
	}
	r.once.Do(func() {
		if r.owner != nil {
			r.closeErr = r.owner.CloseAfterWorkers(context.Background())
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

func newGatewayInternalClient(t *testing.T, service *ts.Service) (trpc.TraderSyncServiceClient, *grpc.Server) {
	t.Helper()
	const token = "gateway-internal-token-0123456789"
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(tsgrpc.NewUnaryInterceptor(token, func() bool { return true })))
	trpc.RegisterTraderSyncServiceServer(server, tsgrpc.NewServer(service))
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	conn, err := grpc.DialContext(context.Background(), "passthrough:///internal", grpc.WithInsecure(), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoke(metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token), method, req, reply, cc, opts...)
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return trpc.NewTraderSyncServiceClient(conn), server
}
