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
	defaultSportsLiveListLimit       = 200
	maxSportsLiveListLimit           = 1000
	defaultSportsLiveSyncInterval    = 10 * time.Second
	defaultSportsLivePriceInterval   = 15 * time.Second
	defaultSportsLiveEventPageLimit  = 500
	defaultHotMarketListLimit        = 100
	maxHotMarketListLimit            = 500
	defaultHotMarketRefreshInterval  = time.Minute
	defaultHotMarketInitialSyncWait  = 15 * time.Second
	defaultRealtimeListLimit         = 100
	maxRealtimeListLimit             = 500
	defaultRealtimeInitialSyncWait   = 15 * time.Second
	defaultMoverListLimit            = 100
	maxMoverListLimit                = 500
	defaultFIFAWalletBalanceCacheTTL = 8 * time.Second
	defaultFIFAPolygonRPCURL         = "https://polygon-rpc.com"
	defaultFIFASolanaRPCURL          = "https://api.mainnet-beta.solana.com"
)

type ServiceOption func(*Service)

type FIFAWalletBalanceConfig struct {
	PolygonRPCURL string
	SolanaRPCURL  string
}

type sportsLiveGammaClient interface {
	ListEventsKeyset(context.Context, utilpolymarket.ListEventsKeysetOptions) (*utilpolymarket.EventKeysetResponse, error)
	ListMarketsKeyset(context.Context, utilpolymarket.ListMarketsKeysetOptions) (*utilpolymarket.MarketKeysetResponse, error)
	GetEventByID(context.Context, int64, utilpolymarket.GetEventOptions) (*utilpolymarket.Event, error)
	GetEventBySlug(context.Context, string, utilpolymarket.GetEventOptions) (*utilpolymarket.Event, error)
}

type sportsLiveCLOBClient interface {
	GetBatchPricesHistory(context.Context, utilpolymarket.CLOBBatchPricesHistoryRequest) (*utilpolymarket.CLOBBatchPricesHistoryResponse, error)
	GetMidpointPrices(context.Context, []string) (map[string]string, error)
}

func WithGammaClient(client sportsLiveGammaClient) ServiceOption {
	return func(s *Service) {
		s.gammaClient = client
	}
}

func WithCLOBClient(client sportsLiveCLOBClient) ServiceOption {
	return func(s *Service) {
		s.clobClient = client
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

func WithSportsLivePriceAlertsConfig(config SportsLivePriceAlertsConfig) ServiceOption {
	return func(s *Service) {
		s.sportsLivePriceAlertsConfig = normalizeSportsLivePriceAlertsConfig(config)
	}
}

func WithManagedOOProposedAlertsConfig(config ManagedOOProposedAlertsConfig) ServiceOption {
	return func(s *Service) {
		s.managedOOProposedAlertsConfig = normalizeManagedOOProposedAlertsConfig(config)
	}
}

func WithManagedOODisputedAlertsConfig(config ManagedOODisputedAlertsConfig) ServiceOption {
	return func(s *Service) {
		s.managedOODisputedAlertsConfig = normalizeManagedOODisputedAlertsConfig(config)
	}
}

func WithFIFAWalletBalanceConfig(config FIFAWalletBalanceConfig) ServiceOption {
	return func(s *Service) {
		if config.PolygonRPCURL != "" {
			s.fifaPolygonRPCURL = config.PolygonRPCURL
		}
		if config.SolanaRPCURL != "" {
			s.fifaSolanaRPCURL = config.SolanaRPCURL
		}
	}
}

type Service struct {
	apiclient.UnimplementedPolymarketServiceServer
	store                         *polymarketstore.SQLStore
	gammaClient                   sportsLiveGammaClient
	clobClient                    sportsLiveCLOBClient
	notificationClientset         notificationapiclient.Clientset
	syncInterval                  time.Duration
	sportsLivePageLimit           int
	hotMarketRefreshInterval      time.Duration
	fifaPolygonRPCURL             string
	fifaSolanaRPCURL              string
	moverAlertsConfig             MoverAlertsConfig
	sportsLivePriceAlertsConfig   SportsLivePriceAlertsConfig
	managedOOProposedAlertsConfig ManagedOOProposedAlertsConfig
	managedOODisputedAlertsConfig ManagedOODisputedAlertsConfig
	nowFn                         func() time.Time
	startStopMu                   sync.Mutex
	managedOOPipelineMu           sync.Mutex
	started                       bool
	runCancel                     context.CancelFunc
	runWG                         sync.WaitGroup
	cacheMu                       sync.RWMutex
	hotMarketItems                []*v1alpha1.PolymarketHotMarketItem
	hotMarketMissing              map[string]int
	realtimeStates                map[string]*realtimeTokenState
	realtimeSamples               map[string][]realtimeSample
	hotMarketFetched              int64
	hotMarketStale                bool
	hotMarketCandidateCount       int32
	realtimeFetched               int64
	realtimeStale                 bool
	realtimeConnected             bool
	realtimeLastEventAt           int64
	realtimeSubscribedMarkets     int32
	realtimeSubscribedTokens      int32
	sportsHistoryStale            bool
	sportsHistorySyncStatus       *v1alpha1.PolymarketSportsHistorySyncStatus
	moverAlertStates              map[string]moverAlertState
	fifaWalletBalances            []*v1alpha1.PolymarketFIFAWalletBalanceItem
	fifaWalletBalancesFetched     int64
	fifaWalletBalancesCachedAt    time.Time
	syncGroup                     singleflight.Group
}

func NewService(store *polymarketstore.SQLStore, opts ...ServiceOption) *Service {
	s := &Service{
		store:                         store,
		syncInterval:                  defaultSportsLiveSyncInterval,
		sportsLivePageLimit:           defaultSportsLiveEventPageLimit,
		hotMarketRefreshInterval:      defaultHotMarketRefreshInterval,
		fifaPolygonRPCURL:             defaultFIFAPolygonRPCURL,
		fifaSolanaRPCURL:              defaultFIFASolanaRPCURL,
		moverAlertsConfig:             defaultMoverAlertsConfig(),
		sportsLivePriceAlertsConfig:   defaultSportsLivePriceAlertsConfig(),
		managedOOProposedAlertsConfig: defaultManagedOOProposedAlertsConfig(),
		managedOODisputedAlertsConfig: defaultManagedOODisputedAlertsConfig(),
		nowFn:                         time.Now,
		hotMarketMissing:              make(map[string]int),
		realtimeStates:                make(map[string]*realtimeTokenState),
		realtimeSamples:               make(map[string][]realtimeSample),
		moverAlertStates:              make(map[string]moverAlertState),
		sportsHistoryStale:            true,
		sportsHistorySyncStatus:       &v1alpha1.PolymarketSportsHistorySyncStatus{State: sportsHistorySyncStateIdle},
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
	if s.clobClient == nil {
		client, err := utilpolymarket.NewCLOBClient(utilpolymarket.CLOBConfig{})
		if err != nil {
			s.startStopMu.Unlock()
			return status.Errorf(codes.FailedPrecondition, "failed to create polymarket clob client: %v", err)
		}
		s.clobClient = client
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.runCancel = cancel
	s.started = true
	s.startStopMu.Unlock()

	s.runWG.Add(1)
	go s.runSportsLiveSyncLoop(ctx)
	s.runWG.Add(1)
	go s.runSportsLivePriceHistorySyncLoop(ctx)
	s.runWG.Add(1)
	go s.runSportsHistorySync(ctx)
	s.runWG.Add(1)
	go s.runHotMarketDiscoveryLoop(ctx)
	s.runWG.Add(1)
	go s.runManagedOOProposePriceLogSyncLoop(ctx)
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
