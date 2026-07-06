package ethereumapi

import (
	"context"
	"sync"
	"time"

	"github.com/useryege/athena/internal/ethereumapi/apiclient"
	ethereumapistore "github.com/useryege/athena/internal/ethereumapi/store"
	utilethereumapi "github.com/useryege/athena/util/ethereumapi"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	DefaultNormalTransactionCacheTTL       = time.Minute
	DefaultNormalTransactionCacheRetention = 24 * time.Hour
	DefaultEthereumAPIRefreshTimeout       = 45 * time.Second

	normalTransactionCacheCleanupInterval = time.Hour
)

type ServiceOpts struct {
	Store          *ethereumapistore.SQLStore
	EthereumAPI    utilethereumapi.EthereumAPI
	CacheTTL       time.Duration
	CacheRetention time.Duration
	RefreshTimeout time.Duration
}

type Service struct {
	apiclient.UnimplementedEthereumAPIServiceServer
	store          *ethereumapistore.SQLStore
	ethereumAPI    utilethereumapi.EthereumAPI
	cacheTTL       time.Duration
	cacheRetention time.Duration
	refreshTimeout time.Duration
	nowFn          func() time.Time

	startStopMu sync.Mutex
	started     bool
	runCtx      context.Context
	runCancel   context.CancelFunc
	runWG       sync.WaitGroup

	refreshGroup      singleflight.Group
	backgroundMu      sync.Mutex
	backgroundRefresh map[string]struct{}
}

func NewService(opts ServiceOpts) *Service {
	cacheTTL := opts.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = DefaultNormalTransactionCacheTTL
	}
	cacheRetention := opts.CacheRetention
	if cacheRetention <= 0 {
		cacheRetention = DefaultNormalTransactionCacheRetention
	}
	refreshTimeout := opts.RefreshTimeout
	if refreshTimeout <= 0 {
		refreshTimeout = DefaultEthereumAPIRefreshTimeout
	}
	return &Service{
		store:             opts.Store,
		ethereumAPI:       opts.EthereumAPI,
		cacheTTL:          cacheTTL,
		cacheRetention:    cacheRetention,
		refreshTimeout:    refreshTimeout,
		nowFn:             time.Now,
		backgroundRefresh: make(map[string]struct{}),
	}
}

func (s *Service) Start(ctx context.Context) error {
	s.startStopMu.Lock()
	if s.started {
		s.startStopMu.Unlock()
		return nil
	}
	if s.store == nil {
		s.startStopMu.Unlock()
		return status.Error(codes.FailedPrecondition, "ethereumapi store is required")
	}
	if s.ethereumAPI == nil {
		s.startStopMu.Unlock()
		return status.Error(codes.FailedPrecondition, "ethereumapi client is required")
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.started = true
	s.runCtx = runCtx
	s.runCancel = cancel
	s.runWG.Add(1)
	s.startStopMu.Unlock()

	go s.runCacheCleanup(runCtx)
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	if !s.started {
		s.startStopMu.Unlock()
		return nil
	}
	s.started = false
	cancel := s.runCancel
	s.runCtx = nil
	s.runCancel = nil
	s.startStopMu.Unlock()

	if cancel != nil {
		cancel()
	}
	s.runWG.Wait()
	return nil
}

func (s *Service) GetEthereumAPIStatus(context.Context, *apiclient.GetEthereumAPIStatusRequest) (*apiclient.GetEthereumAPIStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetEthereumAPIStatusResponse{
		Started: started,
		Status:  statusText,
	}, nil
}
