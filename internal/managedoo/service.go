package managedoo

import (
	"context"
	"sync"
	"time"

	"github.com/useryege/athena/internal/managedoo/apiclient"
	managedoostore "github.com/useryege/athena/internal/managedoo/store"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultPolygonRPCURL = "https://polygon-rpc.com"

type gammaClient interface {
	ListMarketsKeyset(context.Context, utilpolymarket.ListMarketsKeysetOptions) (*utilpolymarket.MarketKeysetResponse, error)
	GetMarketByID(context.Context, int64, utilpolymarket.GetMarketOptions) (*utilpolymarket.Market, error)
}

type ServiceOption func(*Service)

func WithGammaClient(client gammaClient) ServiceOption {
	return func(service *Service) {
		service.gammaClient = client
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

func WithPolygonRPCURL(rpcURL string) ServiceOption {
	return func(service *Service) {
		if rpcURL != "" {
			service.polygonRPCURL = rpcURL
		}
	}
}

func WithProposedAlertsConfig(config ManagedOOProposedAlertsConfig) ServiceOption {
	return func(service *Service) {
		service.managedOOProposedAlertsConfig = normalizeManagedOOProposedAlertsConfig(config)
	}
}

func WithDisputedAlertsConfig(config ManagedOODisputedAlertsConfig) ServiceOption {
	return func(service *Service) {
		service.managedOODisputedAlertsConfig = normalizeManagedOODisputedAlertsConfig(config)
	}
}

type Service struct {
	apiclient.UnimplementedManagedOOServiceServer

	store                         *managedoostore.SQLStore
	gammaClient                   gammaClient
	notificationClientset         notificationapiclient.Clientset
	notificationInviteCode        string
	polygonRPCURL                 string
	managedOOProposedAlertsConfig ManagedOOProposedAlertsConfig
	managedOODisputedAlertsConfig ManagedOODisputedAlertsConfig
	nowFn                         func() time.Time
	startStopMu                   sync.Mutex
	managedOOPipelineMu           sync.Mutex
	started                       bool
	runCancel                     context.CancelFunc
	runWG                         sync.WaitGroup
}

func NewService(store *managedoostore.SQLStore, opts ...ServiceOption) *Service {
	service := &Service{
		store:                         store,
		polygonRPCURL:                 defaultPolygonRPCURL,
		managedOOProposedAlertsConfig: defaultManagedOOProposedAlertsConfig(),
		managedOODisputedAlertsConfig: defaultManagedOODisputedAlertsConfig(),
		nowFn:                         time.Now,
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
		return status.Error(codes.FailedPrecondition, "managed oo store is required")
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

func (s *Service) GetManagedOOStatus(context.Context, *apiclient.GetManagedOOStatusRequest) (*apiclient.GetManagedOOStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()
	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetManagedOOStatusResponse{Started: started, Status: statusText}, nil
}

func (s *Service) now() time.Time {
	if s.nowFn == nil {
		return time.Now()
	}
	return s.nowFn()
}
