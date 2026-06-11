package polymarket

import (
	"context"
	"sync"
	"time"

	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/internal/polymarket/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultSportsLiveListLimit      = 200
	maxSportsLiveListLimit          = 1000
	defaultSportsLiveSyncInterval   = 10 * time.Second
	defaultSportsLiveEventPageLimit = 500
	defaultHotMarketListLimit       = 100
	maxHotMarketListLimit           = 500
	defaultHotMarketRefreshInterval = time.Minute
	defaultHotMarketInitialSyncWait = 15 * time.Second
	defaultRealtimeListLimit        = 100
	maxRealtimeListLimit            = 500
	defaultRealtimeInitialSyncWait  = 15 * time.Second
	defaultMoverListLimit           = 100
	maxMoverListLimit               = 500
)

type ServiceOption func(*Service)

type sportsLiveGammaClient interface {
	ListEventsKeyset(context.Context, utilpolymarket.ListEventsKeysetOptions) (*utilpolymarket.EventKeysetResponse, error)
	ListMarketsKeyset(context.Context, utilpolymarket.ListMarketsKeysetOptions) (*utilpolymarket.MarketKeysetResponse, error)
}

func WithGammaClient(client sportsLiveGammaClient) ServiceOption {
	return func(s *Service) {
		s.gammaClient = client
	}
}

func WithSportsLiveSyncInterval(interval time.Duration) ServiceOption {
	return func(s *Service) {
		if interval > 0 {
			s.syncInterval = interval
		}
	}
}

func WithSportsLivePageLimit(limit int) ServiceOption {
	return func(s *Service) {
		if limit > 0 {
			s.sportsLivePageLimit = limit
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

func WithNotificationClientset(clientset notificationapiclient.Clientset) ServiceOption {
	return func(s *Service) {
		s.notificationClientset = clientset
	}
}

func WithMoverAlertsConfig(config MoverAlertsConfig) ServiceOption {
	return func(s *Service) {
		s.moverAlertsConfig = normalizeMoverAlertsConfig(config)
	}
}

type Service struct {
	apiclient.UnimplementedPolymarketServiceServer
	store                     *polymarketstore.SQLStore
	gammaClient               sportsLiveGammaClient
	notificationClientset     notificationapiclient.Clientset
	syncInterval              time.Duration
	sportsLivePageLimit       int
	hotMarketRefreshInterval  time.Duration
	moverAlertsConfig         MoverAlertsConfig
	nowFn                     func() time.Time
	startStopMu               sync.Mutex
	started                   bool
	runCancel                 context.CancelFunc
	runWG                     sync.WaitGroup
	cacheMu                   sync.RWMutex
	hotMarketItems            []*v1alpha1.PolymarketHotMarketItem
	hotMarketMissing          map[string]int
	realtimeStates            map[string]*realtimeTokenState
	realtimeSamples           map[string][]realtimeSample
	hotMarketFetched          int64
	hotMarketStale            bool
	hotMarketCandidateCount   int32
	realtimeFetched           int64
	realtimeStale             bool
	realtimeConnected         bool
	realtimeLastEventAt       int64
	realtimeSubscribedMarkets int32
	realtimeSubscribedTokens  int32
	moverAlertStates          map[string]moverAlertState
	syncGroup                 singleflight.Group
}

func NewService(store *polymarketstore.SQLStore, opts ...ServiceOption) *Service {
	s := &Service{
		store:                    store,
		syncInterval:             defaultSportsLiveSyncInterval,
		sportsLivePageLimit:      defaultSportsLiveEventPageLimit,
		hotMarketRefreshInterval: defaultHotMarketRefreshInterval,
		moverAlertsConfig:        defaultMoverAlertsConfig(),
		nowFn:                    time.Now,
		hotMarketMissing:         make(map[string]int),
		realtimeStates:           make(map[string]*realtimeTokenState),
		realtimeSamples:          make(map[string][]realtimeSample),
		moverAlertStates:         make(map[string]moverAlertState),
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
	ctx, cancel := context.WithCancel(context.Background())
	s.runCancel = cancel
	s.started = true
	s.startStopMu.Unlock()

	s.runWG.Add(1)
	go s.runSportsLiveSyncLoop(ctx)
	s.runWG.Add(1)
	go s.runHotMarketDiscoveryLoop(ctx)
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

func (s *Service) nowUnix() int64 {
	if s.nowFn == nil {
		return time.Now().Unix()
	}
	return s.nowFn().Unix()
}

func ptrBool(value bool) *bool {
	return &value
}

func formatTimeRFC3339(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
