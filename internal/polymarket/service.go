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
	defaultSportsLiveSnapshotLimit   = 30
	maxSportsLiveSnapshotLimit       = 100
	defaultSportsLiveSyncInterval    = 10 * time.Second
	defaultSportsLiveEventPageLimit  = 500
	defaultSportsLiveInitialSyncWait = 15 * time.Second
	defaultHotMarketListLimit        = 100
	maxHotMarketListLimit            = 500
	defaultHotMarketRefreshInterval  = time.Minute
	defaultHotMarketInitialSyncWait  = 15 * time.Second
	defaultRealtimeListLimit         = 100
	maxRealtimeListLimit             = 500
	defaultRealtimeInitialSyncWait   = 15 * time.Second
	defaultRealtimeSampleInterval    = time.Second
)

type ServiceOption func(*Service)

type sportsLiveGammaClient interface {
	ListEventsKeyset(context.Context, utilpolymarket.ListEventsKeysetOptions) (*utilpolymarket.EventKeysetResponse, error)
	ListMarketsKeyset(context.Context, utilpolymarket.ListMarketsKeysetOptions) (*utilpolymarket.MarketKeysetResponse, error)
}

type sportsLiveWSClient interface {
	Run(context.Context, utilpolymarket.SportsWSHandler) error
}

type clobMarketWSClient interface {
	Run(context.Context, utilpolymarket.CLOBMarketWSSubscription, utilpolymarket.CLOBMarketWSHandler) error
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

func WithCLOBMarketWSClient(client clobMarketWSClient) ServiceOption {
	return func(s *Service) {
		s.clobMarketWSClient = client
	}
}

func WithWSUseProxy(useProxy bool) ServiceOption {
	return func(s *Service) {
		s.wsUseProxy = useProxy
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

func WithHotMarketRefreshInterval(interval time.Duration) ServiceOption {
	return func(s *Service) {
		if interval > 0 {
			s.hotMarketRefreshInterval = interval
		}
	}
}

func WithRealtimeSampleInterval(interval time.Duration) ServiceOption {
	return func(s *Service) {
		if interval > 0 {
			s.realtimeSampleInterval = interval
		}
	}
}

type Service struct {
	apiclient.UnimplementedPolymarketServiceServer
	store                     *polymarketstore.SQLStore
	gammaClient               sportsLiveGammaClient
	sportsWSClient            sportsLiveWSClient
	clobMarketWSClient        clobMarketWSClient
	wsUseProxy                bool
	syncInterval              time.Duration
	eventPageLimit            int
	hotMarketRefreshInterval  time.Duration
	realtimeSampleInterval    time.Duration
	nowFn                     func() time.Time
	startStopMu               sync.Mutex
	started                   bool
	runCancel                 context.CancelFunc
	runWG                     sync.WaitGroup
	cacheMu                   sync.RWMutex
	baseMarkets               []sportsLiveMarket
	snapshotItems             []*v1alpha1.PolymarketSportsLiveMarketItem
	snapshotEvents            []*v1alpha1.PolymarketSportsLiveEventItem
	sportsWSState             map[string]sportsLiveWSState
	hotMarketItems            []*v1alpha1.PolymarketHotMarketItem
	hotMarketMissing          map[string]int
	realtimeStates            map[string]*realtimeTokenState
	realtimeSamples           map[string][]realtimeSample
	snapshotFetched           int64
	snapshotStale             bool
	eventFetched              int64
	eventStale                bool
	hotMarketFetched          int64
	hotMarketStale            bool
	hotMarketCandidateCount   int32
	realtimeFetched           int64
	realtimeStale             bool
	realtimeConnected         bool
	realtimeLastEventAt       int64
	realtimeSubscribedHash    string
	realtimeSubscribedMarkets int32
	realtimeSubscribedTokens  int32
	syncGroup                 singleflight.Group
}

func NewService(store *polymarketstore.SQLStore, opts ...ServiceOption) *Service {
	s := &Service{
		store:                    store,
		wsUseProxy:               true,
		syncInterval:             defaultSportsLiveSyncInterval,
		eventPageLimit:           defaultSportsLiveEventPageLimit,
		hotMarketRefreshInterval: defaultHotMarketRefreshInterval,
		realtimeSampleInterval:   defaultRealtimeSampleInterval,
		nowFn:                    time.Now,
		sportsWSState:            make(map[string]sportsLiveWSState),
		hotMarketMissing:         make(map[string]int),
		realtimeStates:           make(map[string]*realtimeTokenState),
		realtimeSamples:          make(map[string][]realtimeSample),
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
		client, err := utilpolymarket.NewSportsWSClient(utilpolymarket.SportsWSConfig{
			UseProxy: s.wsUseProxy,
		})
		if err != nil {
			s.startStopMu.Unlock()
			return status.Errorf(codes.FailedPrecondition, "failed to create polymarket sports ws client: %v", err)
		}
		s.sportsWSClient = client
	}
	if s.clobMarketWSClient == nil {
		client, err := utilpolymarket.NewCLOBMarketWSClient(utilpolymarket.CLOBMarketWSConfig{
			UseProxy: s.wsUseProxy,
		})
		if err != nil {
			s.startStopMu.Unlock()
			return status.Errorf(codes.FailedPrecondition, "failed to create polymarket clob market ws client: %v", err)
		}
		s.clobMarketWSClient = client
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.runCancel = cancel
	s.started = true
	s.startStopMu.Unlock()

	s.runWG.Add(1)
	go s.runFullSyncLoop(ctx)
	s.runWG.Add(1)
	go s.runHotMarketDiscoveryLoop(ctx)
	s.runWG.Add(1)
	go s.runSportsWSLoop(ctx)
	s.runWG.Add(1)
	go s.runCLOBMarketWSLoop(ctx)
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

func (s *Service) GetPolymarketSportsLiveSnapshot(ctx context.Context, req *apiclient.GetPolymarketSportsLiveSnapshotRequest) (*apiclient.GetPolymarketSportsLiveSnapshotResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()
	if !started {
		return nil, status.Error(codes.FailedPrecondition, "polymarket service is not running")
	}

	limit := defaultSportsLiveSnapshotLimit
	if req != nil && req.GetLimit() > 0 {
		limit = int(req.GetLimit())
	}
	if limit < 1 || limit > maxSportsLiveSnapshotLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between 1 and %d", maxSportsLiveSnapshotLimit)
	}

	if !s.hasEventSnapshot() {
		syncCtx, cancel := context.WithTimeout(ctx, defaultSportsLiveInitialSyncWait)
		err := s.refreshSportsLiveEventSnapshot(syncCtx)
		cancel()
		if err != nil {
			return nil, status.Errorf(codes.Unavailable, "sports live snapshot is unavailable: %v", err)
		}
	}

	resp := s.currentSportsLiveEventResponse(limit)
	if resp == nil {
		return nil, status.Error(codes.Unavailable, "sports live snapshot is unavailable")
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

func errNoSportsLiveEventSnapshot() error {
	return fmt.Errorf("sports live event snapshot not initialized")
}
