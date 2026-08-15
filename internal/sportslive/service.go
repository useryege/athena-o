package sportslive

import (
	"context"
	"sync"
	"time"

	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/internal/sportslive/apiclient"
	sportslivestore "github.com/useryege/athena/internal/sportslive/store"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultSportsLiveListLimit      = 200
	maxSportsLiveListLimit          = 1000
	defaultSportsLiveSyncInterval   = 10 * time.Second
	defaultSportsLivePriceInterval  = 15 * time.Second
	defaultSportsLiveEventPageLimit = 500
)

type gammaClient interface {
	ListEventsKeyset(context.Context, utilpolymarket.ListEventsKeysetOptions) (*utilpolymarket.EventKeysetResponse, error)
}

type clobClient interface {
	GetBatchPricesHistory(context.Context, utilpolymarket.CLOBBatchPricesHistoryRequest) (*utilpolymarket.CLOBBatchPricesHistoryResponse, error)
}

type ServiceOption func(*Service)

func WithGammaClient(client gammaClient) ServiceOption {
	return func(service *Service) {
		service.gammaClient = client
	}
}

func WithCLOBClient(client clobClient) ServiceOption {
	return func(service *Service) {
		service.clobClient = client
	}
}

func WithSyncInterval(interval time.Duration) ServiceOption {
	return func(service *Service) {
		if interval > 0 {
			service.syncInterval = interval
		}
	}
}

func WithPageLimit(limit int) ServiceOption {
	return func(service *Service) {
		if limit > 0 {
			service.sportsLivePageLimit = limit
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

func WithPriceAlertsConfig(config SportsLivePriceAlertsConfig) ServiceOption {
	return func(service *Service) {
		service.sportsLivePriceAlertsConfig = normalizeSportsLivePriceAlertsConfig(config)
	}
}

func WithScoreAlertsConfig(config SportsLiveScoreAlertsConfig) ServiceOption {
	return func(service *Service) {
		service.sportsLiveScoreAlertsConfig = normalizeSportsLiveScoreAlertsConfig(config)
	}
}

type Service struct {
	apiclient.UnimplementedSportsLiveServiceServer

	store                       *sportslivestore.SQLStore
	gammaClient                 gammaClient
	clobClient                  clobClient
	notificationClientset       notificationapiclient.Clientset
	notificationInviteCode      string
	syncInterval                time.Duration
	sportsLivePageLimit         int
	sportsLivePriceAlertsConfig SportsLivePriceAlertsConfig
	sportsLiveScoreAlertsConfig SportsLiveScoreAlertsConfig
	nowFn                       func() time.Time
	startStopMu                 sync.Mutex
	started                     bool
	runCancel                   context.CancelFunc
	runWG                       sync.WaitGroup
}

func NewService(store *sportslivestore.SQLStore, opts ...ServiceOption) *Service {
	service := &Service{
		store:                       store,
		syncInterval:                defaultSportsLiveSyncInterval,
		sportsLivePageLimit:         defaultSportsLiveEventPageLimit,
		sportsLivePriceAlertsConfig: defaultSportsLivePriceAlertsConfig(),
		sportsLiveScoreAlertsConfig: defaultSportsLiveScoreAlertsConfig(),
		nowFn:                       time.Now,
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
	if s.store == nil {
		s.startStopMu.Unlock()
		return status.Error(codes.FailedPrecondition, "sports live store is required")
	}
	if s.gammaClient == nil {
		client, err := utilpolymarket.NewGammaClient(utilpolymarket.GammaConfig{})
		if err != nil {
			s.startStopMu.Unlock()
			return status.Errorf(codes.FailedPrecondition, "create polymarket gamma client: %v", err)
		}
		s.gammaClient = client
	}
	if s.clobClient == nil {
		client, err := utilpolymarket.NewCLOBClient(utilpolymarket.CLOBConfig{})
		if err != nil {
			s.startStopMu.Unlock()
			return status.Errorf(codes.FailedPrecondition, "create polymarket clob client: %v", err)
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

func (s *Service) GetSportsLiveStatus(context.Context, *apiclient.GetSportsLiveStatusRequest) (*apiclient.GetSportsLiveStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()
	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetSportsLiveStatusResponse{Started: started, Status: statusText}, nil
}

func (s *Service) now() time.Time {
	if s.nowFn == nil {
		return time.Now()
	}
	return s.nowFn()
}
