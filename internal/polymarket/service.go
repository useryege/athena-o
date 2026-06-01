package polymarket

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/useryege/athena/internal/polymarket/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultSportsLiveListLimit       = 200
	maxSportsLiveListLimit           = 1000
	defaultSportsLiveSyncInterval    = 10 * time.Second
	defaultSportsLiveEventPageLimit  = 500
	defaultSportsLiveInitialSyncWait = 15 * time.Second
)

type ServiceOption func(*Service)

type sportsLiveGammaClient interface {
	ListEventsKeyset(context.Context, utilpolymarket.ListEventsKeysetOptions) (*utilpolymarket.EventKeysetResponse, error)
}

type sportsLiveWSClient interface {
	Run(context.Context, utilpolymarket.SportsWSHandler) error
}

func WithGammaClient(client sportsLiveGammaClient) ServiceOption {
	return func(s *Service) {
		s.gammaClient = client
	}
}

func WithSportsWSClient(client sportsLiveWSClient) ServiceOption {
	return func(s *Service) {
		s.sportsWSClient = client
	}
}

func WithSportsLiveSyncInterval(interval time.Duration) ServiceOption {
	return func(s *Service) {
		if interval > 0 {
			s.syncInterval = interval
		}
	}
}

func WithSportsLiveEventPageLimit(limit int) ServiceOption {
	return func(s *Service) {
		if limit > 0 {
			s.eventPageLimit = limit
		}
	}
}

type Service struct {
	apiclient.UnimplementedPolymarketServiceServer
	store           *polymarketstore.SQLStore
	gammaClient     sportsLiveGammaClient
	sportsWSClient  sportsLiveWSClient
	syncInterval    time.Duration
	eventPageLimit  int
	nowFn           func() time.Time
	startStopMu     sync.Mutex
	started         bool
	runCancel       context.CancelFunc
	runWG           sync.WaitGroup
	cacheMu         sync.RWMutex
	baseMarkets     []sportsLiveMarket
	snapshotItems   []*v1alpha1.PolymarketSportsLiveMarketItem
	sportsWSState   map[string]sportsLiveWSState
	snapshotFetched int64
	snapshotStale   bool
	syncGroup       singleflight.Group
}

func NewService(store *polymarketstore.SQLStore, opts ...ServiceOption) *Service {
	s := &Service{
		store:          store,
		syncInterval:   defaultSportsLiveSyncInterval,
		eventPageLimit: defaultSportsLiveEventPageLimit,
		nowFn:          time.Now,
		sportsWSState:  make(map[string]sportsLiveWSState),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	if s.started {
		s.startStopMu.Unlock()
		return nil
	}
	if s.store == nil {
		s.startStopMu.Unlock()
		return status.Error(codes.FailedPrecondition, "polymarket store is required")
	}
	if s.gammaClient == nil {
		client, err := utilpolymarket.NewGammaClient(utilpolymarket.GammaConfig{})
		if err != nil {
			s.startStopMu.Unlock()
			return status.Errorf(codes.FailedPrecondition, "failed to create polymarket gamma client: %v", err)
		}
		s.gammaClient = client
	}
	if s.sportsWSClient == nil {
		client, err := utilpolymarket.NewSportsWSClient(utilpolymarket.SportsWSConfig{})
		if err != nil {
			s.startStopMu.Unlock()
			return status.Errorf(codes.FailedPrecondition, "failed to create polymarket sports ws client: %v", err)
		}
		s.sportsWSClient = client
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.runCancel = cancel
	s.started = true
	s.startStopMu.Unlock()

	s.runWG.Add(1)
	go s.runFullSyncLoop(ctx)
	s.runWG.Add(1)
	go s.runSportsWSLoop(ctx)
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	cancel := s.runCancel
	s.runCancel = nil
	s.started = false
	s.startStopMu.Unlock()
	if cancel != nil {
		cancel()
		s.runWG.Wait()
	}
	return nil
}

func (s *Service) GetPolymarketStatus(context.Context, *apiclient.GetPolymarketStatusRequest) (*v1alpha1.PolymarketStatus, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &v1alpha1.PolymarketStatus{
		Started: started,
		Status:  statusText,
	}, nil
}

func (s *Service) ListPolymarketSportsLiveMarkets(ctx context.Context, req *apiclient.ListPolymarketSportsLiveMarketsRequest) (*apiclient.ListPolymarketSportsLiveMarketsResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()
	if !started {
		return nil, status.Error(codes.FailedPrecondition, "polymarket service is not running")
	}

	limit := defaultSportsLiveListLimit
	if req != nil && req.GetLimit() > 0 {
		limit = int(req.GetLimit())
	}
	if limit < 1 || limit > maxSportsLiveListLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between 1 and %d", maxSportsLiveListLimit)
	}

	if !s.hasSnapshot() {
		syncCtx, cancel := context.WithTimeout(ctx, defaultSportsLiveInitialSyncWait)
		err := s.refreshSportsLiveSnapshot(syncCtx)
		cancel()
		if err != nil {
			return nil, status.Errorf(codes.Unavailable, "sports live markets are unavailable: %v", err)
		}
	}

	resp := s.currentSportsLiveResponse(limit)
	if resp == nil {
		return nil, status.Error(codes.Unavailable, "sports live markets are unavailable")
	}
	return resp, nil
}

func (s *Service) nowUnix() int64 {
	if s.nowFn == nil {
		return time.Now().Unix()
	}
	return s.nowFn().Unix()
}

func ptrBool(value bool) *bool {
	return &value
}

func errNoSportsLiveSnapshot() error {
	return fmt.Errorf("sports live snapshot not initialized")
}
