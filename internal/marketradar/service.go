package marketradar

import (
	"context"
	"sync"
	"time"

	"github.com/useryege/athena/internal/marketradar/apiclient"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
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

type gammaClient interface {
	ListMarketsKeyset(context.Context, utilpolymarket.ListMarketsKeysetOptions) (*utilpolymarket.MarketKeysetResponse, error)
}

type ServiceOption func(*Service)

func WithGammaClient(client gammaClient) ServiceOption {
	return func(service *Service) {
		service.gammaClient = client
	}
}

func WithHotMarketRefreshInterval(interval time.Duration) ServiceOption {
	return func(service *Service) {
		if interval > 0 {
			service.hotMarketRefreshInterval = interval
		}
	}
}

func WithNotificationClientset(clientset notificationapiclient.Clientset) ServiceOption {
	return func(service *Service) {
		service.notificationClientset = clientset
	}
}

func WithNotificationInviteCode(inviteCode string) ServiceOption {
	return func(service *Service) {
		service.notificationInviteCode = inviteCode
	}
}

func WithMoverAlertsConfig(config MoverAlertsConfig) ServiceOption {
	return func(service *Service) {
		service.moverAlertsConfig = normalizeMoverAlertsConfig(config)
	}
}

type Service struct {
	apiclient.UnimplementedMarketRadarServiceServer

	gammaClient               gammaClient
	notificationClientset     notificationapiclient.Clientset
	notificationInviteCode    string
	hotMarketRefreshInterval  time.Duration
	moverAlertsConfig         MoverAlertsConfig
	nowFn                     func() time.Time
	startStopMu               sync.Mutex
	started                   bool
	runCancel                 context.CancelFunc
	runWG                     sync.WaitGroup
	cacheMu                   sync.RWMutex
	hotMarketItems            []*v1alpha1.MarketRadarHotMarketItem
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

func NewService(opts ...ServiceOption) *Service {
	service := &Service{
		hotMarketRefreshInterval: defaultHotMarketRefreshInterval,
		moverAlertsConfig:        defaultMoverAlertsConfig(),
		nowFn:                    time.Now,
		hotMarketMissing:         make(map[string]int),
		realtimeStates:           make(map[string]*realtimeTokenState),
		realtimeSamples:          make(map[string][]realtimeSample),
		moverAlertStates:         make(map[string]moverAlertState),
	}
	for _, option := range opts {
		option(service)
	}
	return service
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	if s.started {
		s.startStopMu.Unlock()
		return nil
	}
	if s.gammaClient == nil {
		client, err := utilpolymarket.NewGammaClient(utilpolymarket.GammaConfig{})
		if err != nil {
			s.startStopMu.Unlock()
			return status.Errorf(codes.FailedPrecondition, "create polymarket gamma client: %v", err)
		}
		s.gammaClient = client
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.runCancel = cancel
	s.started = true
	s.startStopMu.Unlock()

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

func (s *Service) GetMarketRadarStatus(context.Context, *apiclient.GetMarketRadarStatusRequest) (*apiclient.GetMarketRadarStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()
	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetMarketRadarStatusResponse{Started: started, Status: statusText}, nil
}

func (s *Service) nowUnix() int64 {
	return s.now().Unix()
}
