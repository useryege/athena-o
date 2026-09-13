package tradersync

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/accountstate/schema"
	"github.com/useryege/athena/internal/tradersync/rpcconfig"
	"github.com/useryege/athena/internal/tradersync/store"
	pm "github.com/useryege/athena/util/polymarket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const HealthServiceName = "tradersync.internal.v1.TraderSyncService"

type RuntimeDependencies struct {
	// NewRPCServer registers business handlers against the stable Service pointer.
	// Its business interceptor must check ready before invoking any handler; the
	// runtime publishes the fully initialized Service through that atomic gate.
	NewRPCServer func(*Service, func() bool) *grpc.Server
}
type runtimeWorker struct {
	name string
	run  func(context.Context) error
}

// Runtime is the sole resource and session owner. Shutdown never releases its
// session until every worker and accepted RPC has finished using the database.
type Runtime struct {
	cfg          Config
	rpcCfg       rpcconfig.Server
	deps         RuntimeDependencies
	mu           sync.Mutex
	started      bool
	ready        atomic.Bool
	runCtx       context.Context
	cancel       context.CancelFunc
	stopping     chan struct{}
	stopOnce     sync.Once
	runErr       error
	joined       chan struct{}
	workers      sync.WaitGroup
	shutdownOnce sync.Once
	closed       chan struct{}
	closeErr     error
	pool         *pgxpool.Pool
	owner        *store.RuntimeSession
	listener     net.Listener
	server       *grpc.Server
	health       *health.Server
	service      *Service
	client       *ethclient.Client
	transport    *http.Transport
}

func NewRuntime(cfg Config, rpcCfg rpcconfig.Server, deps RuntimeDependencies) (*Runtime, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	p := cfg.Projector
	if p.Interval < 0 || p.MetadataWait < 0 || p.MetadataWait > 2*time.Second || p.MetadataTimeout < 0 || p.MaxInFlightSources < 0 {
		return nil, errors.New("invalid projector resource limits")
	}
	min, max := cfg.ReconnectMin, cfg.ReconnectMax
	if min == 0 {
		min = time.Second
	}
	if max == 0 {
		max = 30 * time.Second
	}
	if min < 0 || max < min {
		return nil, errors.New("invalid collector reconnect bounds")
	}
	if strings.TrimSpace(cfg.AccountStateDSN) == "" {
		return nil, errors.New("ATHENA_ACCOUNT_STATE_POSTGRES_DSN required")
	}
	if _, err := pgxpool.ParseConfig(cfg.AccountStateDSN); err != nil {
		return nil, errors.New("invalid account-state PostgreSQL configuration")
	}
	if cfg.ShutdownTimeout <= 0 {
		return nil, errors.New("Trader Sync shutdown timeout must be positive")
	}
	if err := rpcCfg.Validate(); err != nil {
		return nil, err
	}
	if _, err := rpcconfig.ServerCredentials(rpcCfg); err != nil {
		return nil, err
	}
	if cfg.CursorHMACKey == rpcCfg.Token {
		return nil, errors.New("Trader Sync internal token and cursor key must be independent")
	}
	if deps.NewRPCServer == nil {
		return nil, errors.New("Trader Sync RPC server factory required")
	}
	return &Runtime{cfg: cfg, rpcCfg: rpcCfg, deps: deps, stopping: make(chan struct{}), joined: make(chan struct{}), closed: make(chan struct{})}, nil
}

func (r *Runtime) phase(phase string) {
	fields := log.Fields{"service": "trader-sync", "phase": phase, "instance": "unmanaged", "run": "direct", "runtime_generation": "unknown", "collector_epoch": "unknown"}
	if v := os.Getenv("ATHENA_LOCAL_RUNTIME_INSTANCE"); v != "" {
		fields["instance"] = v
	}
	if v := os.Getenv("ATHENA_LOCAL_RUNTIME_RUN_ID"); v != "" {
		fields["run"] = v
	}
	if r.owner != nil {
		fields["runtime_generation"] = r.owner.RuntimeToken().Generation
	}
	if r.service != nil && r.service.deps.Collector != nil {
		epoch, _, _, available := r.service.deps.Collector.RawSnapshot()
		if available {
			fields["collector_epoch"] = epoch
		}
	}
	log.WithFields(fields).Info("Trader Sync lifecycle")
}

func (r *Runtime) Start(ctx context.Context) (err error) {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return errors.New("Trader Sync runtime already started")
	}
	r.started = true
	r.runCtx, r.cancel = context.WithCancel(ctx)
	r.mu.Unlock()
	startCtx, stop := context.WithTimeout(r.runCtx, schema.DefaultTimeout)
	defer stop()
	defer func() {
		if err != nil {
			r.fail(err)
			if cause := r.Wait(); cause != nil && !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
			go func() { r.workers.Wait(); close(r.joined) }()
			cleanup, cancel := context.WithTimeout(context.Background(), r.cfg.ShutdownTimeout)
			defer cancel()
			err = errors.Join(err, r.Shutdown(cleanup))
		}
	}()
	r.phase("verify_schema")
	r.pool, err = schema.ConnectVerified(startCtx, r.cfg.AccountStateDSN)
	if err != nil {
		return err
	}
	r.listener, err = (&net.ListenConfig{}).Listen(startCtx, "tcp", r.rpcCfg.ListenAddress)
	if err != nil {
		return err
	}
	r.health = health.NewServer()
	r.health.SetServingStatus(HealthServiceName, healthpb.HealthCheckResponse_NOT_SERVING)
	r.health.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	// Register once against a stable address. Business methods remain behind
	// ready's atomic publication barrier until the complete Service is installed.
	r.service = &Service{}
	r.server = r.deps.NewRPCServer(r.service, r.Ready)
	if r.server == nil {
		return errors.New("Trader Sync RPC factory returned nil server")
	}
	healthpb.RegisterHealthServer(r.server, r.health)
	r.launchWorker(r.runCtx, runtimeWorker{"grpc", func(context.Context) error { return r.server.Serve(r.listener) }})
	r.phase("not_serving")
	r.phase("acquire_owner")
	r.owner, err = store.NewSQLStore(r.pool).AcquireRuntimeSession(startCtx)
	if err != nil {
		return err
	}
	// Monitor the dedicated owner while recovery is still waiting on accounts.
	r.launchWorker(r.runCtx, runtimeWorker{"owner", r.checkOwner})
	gate, err := store.NewRuntimeWriteGate(r.pool, r.owner.RuntimeToken())
	if err != nil {
		return err
	}
	storage, err := store.NewRuntimeSQLStore(r.pool, gate)
	if err != nil {
		return err
	}
	if err = r.compose(startCtx, storage); err != nil {
		return err
	}
	r.phase("recover_pending")
	if err = r.owner.RecoverPending(startCtx); err != nil {
		return err
	}
	if err = startCtx.Err(); err != nil {
		return err
	}
	r.startWorkers(ctx, []runtimeWorker{
		{"collector", func(ctx context.Context) error { return r.service.deps.Collector.Run(ctx, r.owner) }},
		{"projector", r.service.deps.Projector.Run}, {"directory", r.service.deps.Directory.Run},
	})
	r.phase("serving")
	return nil
}

func (r *Runtime) compose(ctx context.Context, storage *store.SQLStore) error {
	r.transport = http.DefaultTransport.(*http.Transport).Clone()
	r.transport.Proxy = nil
	if r.cfg.ProxyURL != "" {
		u, err := url.Parse(r.cfg.ProxyURL)
		if err != nil {
			return err
		}
		r.transport.Proxy = http.ProxyURL(u)
	}
	httpClient := &http.Client{Transport: r.transport, Timeout: 5 * time.Second}
	raw, err := rpc.DialOptions(ctx, r.cfg.HTTPURL, rpc.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("create Trader Sync HTTP RPC: %w", err)
	}
	r.client = ethclient.NewClient(raw)
	source := NewSourceRPC(r.client)
	gamma, err := pm.NewGammaClient(pm.GammaConfig{HTTPClient: httpClient, Timeout: 5 * time.Second})
	if err != nil {
		return err
	}
	profiles := pm.NewProfileAdapter(r.transport)
	resolver, err := NewTargetResolver(storage, profiles, storage.RequireGrantTx, storage.ResolveContextTx)
	if err != nil {
		return err
	}
	collector, err := NewCollector(storage, source, r.cfg)
	if err != nil {
		return err
	}
	// The registrar already knows the durable token before RPCs are admitted.
	collector.token = r.owner.CollectorToken()
	subscriptions, err := NewSubscriptionService(storage, storage, resolver, collector)
	if err != nil {
		return err
	}
	if err = storage.ConfigureActivities(r.cfg.SiteURL); err != nil {
		return err
	}
	metadata := NewMetadataResolver(gamma, source, storage)
	projector, err := NewProjector(storage, source, NewVersionVerifier(source), metadata, r.cfg.Projector)
	if err != nil {
		return err
	}
	directory := NewDirectoryRefresher(storage, gamma, r.cfg.OnError)
	service, err := NewService(r.cfg, Dependencies{Pool: r.pool, Resolver: resolver, Subscriptions: subscriptions, Collector: collector, Projector: projector, Directory: directory})
	if err == nil {
		*r.service = *service
	}
	return err
}

func (r *Runtime) startWorkers(ctx context.Context, workers []runtimeWorker) {
	r.mu.Lock()
	r.started = true
	if r.runCtx == nil {
		r.runCtx, r.cancel = context.WithCancel(ctx)
	}
	runCtx := r.runCtx
	r.mu.Unlock()
	for _, worker := range workers {
		r.launchWorker(runCtx, worker)
	}
	r.mu.Lock()
	if runCtx.Err() == nil {
		r.ready.Store(true)
		if r.health != nil {
			r.health.SetServingStatus(HealthServiceName, healthpb.HealthCheckResponse_SERVING)
			r.health.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
		}
	}
	r.mu.Unlock()
	go func() { <-runCtx.Done(); r.fail(nil) }()
	go func() { r.workers.Wait(); close(r.joined) }()
}
func (r *Runtime) launchWorker(ctx context.Context, worker runtimeWorker) {
	r.workers.Add(1)
	go func() {
		defer r.workers.Done()
		err := worker.run(ctx)
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			err = errors.New("stopped unexpectedly")
		}
		r.fail(fmt.Errorf("Trader Sync %s: %w", worker.name, err))
	}()
}
func (r *Runtime) checkOwner(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			check, stop := context.WithTimeout(ctx, 5*time.Second)
			err := r.owner.Check(check)
			stop()
			if err != nil {
				return err
			}
		}
	}
}
func (r *Runtime) fail(err error) {
	r.stopOnce.Do(func() {
		r.mu.Lock()
		r.runErr = err
		r.ready.Store(false)
		if r.health != nil {
			r.health.Shutdown()
		}
		cancel := r.cancel
		r.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		close(r.stopping)
	})
}
func (r *Runtime) Ready() bool { return r.ready.Load() }
func (r *Runtime) Wait() error { <-r.stopping; r.mu.Lock(); defer r.mu.Unlock(); return r.runErr }

func (r *Runtime) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	if !r.started {
		r.started = true
		close(r.joined)
	}
	r.mu.Unlock()
	r.fail(nil)
	r.shutdownOnce.Do(func() {
		go func() {
			// The caller's budget controls waiting, never resource release. If a borrower
			// ignores cancellation this goroutine remains blocked until process exit.
			r.phase("drain")
			drained := make(chan struct{})
			go func() {
				if r.server != nil {
					r.server.GracefulStop()
				}
				close(drained)
			}()
			<-drained
			<-r.joined
			r.phase("workers_joined")
			if r.owner != nil {
				r.closeErr = r.owner.CloseAfterWorkers(ctx)
			}
			if r.client != nil {
				r.client.Close()
			}
			if r.transport != nil {
				r.transport.CloseIdleConnections()
			}
			if r.listener != nil {
				_ = r.listener.Close()
			}
			if r.pool != nil {
				r.pool.Close()
			}
			r.phase("closed")
			close(r.closed)
		}()
	})
	select {
	case <-r.closed:
		return r.closeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}
