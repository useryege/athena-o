package wormpoly

import (
	"context"
	"sync"
	"time"

	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	wormapiclient "github.com/useryege/athena/internal/worm/apiclient"
	"github.com/useryege/athena/internal/wormpoly/apiclient"
	wormpolystore "github.com/useryege/athena/internal/wormpoly/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultFIFADashboardRefreshInterval     = time.Second
	defaultFIFAWalletBalanceRefreshInterval = 3 * time.Second
	defaultFIFAPolygonRPCURL                = "https://polygon-rpc.com"
	defaultFIFASolanaRPCURL                 = "https://api.mainnet-beta.solana.com"
)

type ServiceOption func(*Service)

type FIFAWalletBalanceConfig struct {
	PolygonRPCURL   string
	SolanaRPCURL    string
	RefreshInterval time.Duration
}

type polymarketGammaClient interface {
	GetEventByID(context.Context, int64, utilpolymarket.GetEventOptions) (*utilpolymarket.Event, error)
	GetEventBySlug(context.Context, string, utilpolymarket.GetEventOptions) (*utilpolymarket.Event, error)
}

type polymarketCLOBClient interface {
	GetMidpointPrices(context.Context, []string) (map[string]string, error)
	GetMarketPrices(context.Context, []string, []string) (map[string]map[string]string, error)
	GetSpreads(context.Context, []utilpolymarket.CLOBBookRequest) (map[string]string, error)
}

func WithGammaClient(client polymarketGammaClient) ServiceOption {
	return func(s *Service) {
		s.gammaClient = client
	}
}

func WithCLOBClient(client polymarketCLOBClient) ServiceOption {
	return func(s *Service) {
		s.clobClient = client
	}
}

func WithWormClientset(clientset wormapiclient.Clientset) ServiceOption {
	return func(s *Service) {
		s.wormClientset = clientset
	}
}

func WithWalletClientset(clientset walletapiclient.Clientset) ServiceOption {
	return func(s *Service) {
		s.walletClientset = clientset
	}
}

func WithFIFADashboardRefreshInterval(interval time.Duration) ServiceOption {
	return func(s *Service) {
		if interval > 0 {
			s.fifaDashboardRefreshInterval = interval
		}
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
		if config.RefreshInterval > 0 {
			s.fifaWalletBalanceRefreshInterval = config.RefreshInterval
		}
	}
}

type Service struct {
	apiclient.UnimplementedWormPolyServiceServer
	store                            *wormpolystore.SQLStore
	wormClientset                    wormapiclient.Clientset
	walletClientset                  walletapiclient.Clientset
	gammaClient                      polymarketGammaClient
	clobClient                       polymarketCLOBClient
	fifaPolygonRPCURL                string
	fifaSolanaRPCURL                 string
	fifaDashboardRefreshInterval     time.Duration
	fifaWalletBalanceRefreshInterval time.Duration
	nowFn                            func() time.Time
	startStopMu                      sync.Mutex
	started                          bool
	runCancel                        context.CancelFunc
	runWG                            sync.WaitGroup
	cacheMu                          sync.RWMutex
	refreshCh                        chan struct{}
	revision                         uint64
	config                           *v1alpha1.WormPolyFIFAEventConfig
	configErr                        error
	wormItem                         *v1alpha1.WormEventItem
	wormFetchedAt                    int64
	wormErr                          error
	polymarketItem                   *v1alpha1.PolymarketFIFAMoneylineEventItem
	polymarketFetchedAt              int64
	polymarketErr                    error
	fifaWalletBalances               []*v1alpha1.PolymarketFIFAWalletBalanceItem
	fifaWalletBalancesFetched        int64
	fifaWalletBalancesCachedAt       time.Time
	fifaWalletHoldings               map[string]*fifaWalletHoldingsCacheEntry
	syncGroup                        singleflight.Group
}

func NewService(store *wormpolystore.SQLStore, opts ...ServiceOption) *Service {
	s := &Service{
		store:                            store,
		fifaPolygonRPCURL:                defaultFIFAPolygonRPCURL,
		fifaSolanaRPCURL:                 defaultFIFASolanaRPCURL,
		fifaDashboardRefreshInterval:     defaultFIFADashboardRefreshInterval,
		fifaWalletBalanceRefreshInterval: defaultFIFAWalletBalanceRefreshInterval,
		nowFn:                            time.Now,
		refreshCh:                        make(chan struct{}, 1),
		configErr:                        errCacheNotReady,
		wormErr:                          errCacheNotReady,
		polymarketErr:                    errCacheNotReady,
		fifaWalletHoldings:               make(map[string]*fifaWalletHoldingsCacheEntry),
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
		return status.Error(codes.FailedPrecondition, "worm-poly store is required")
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
	go s.runFIFADashboardRefreshLoop(ctx)
	s.runWG.Add(1)
	go s.runFIFAWalletBalanceRefreshLoop(ctx)
	s.runWG.Add(1)
	go s.runFIFAWalletHoldingRefreshLoop(ctx)
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

func (s *Service) GetWormPolyStatus(context.Context, *apiclient.GetWormPolyStatusRequest) (*apiclient.GetWormPolyStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetWormPolyStatusResponse{
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

func (s *Service) nowTime() time.Time {
	if s.nowFn == nil {
		return time.Now()
	}
	return s.nowFn()
}
