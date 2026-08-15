package sportshistory

import (
	"context"
	"sync"
	"time"

	"github.com/useryege/athena/internal/sportshistory/apiclient"
	sportshistorystore "github.com/useryege/athena/internal/sportshistory/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultSportsHistoryEventPageLimit = 500

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

func WithPageLimit(limit int) ServiceOption {
	return func(service *Service) {
		if limit > 0 {
			service.sportsHistoryPageLimit = limit
		}
	}
}

type Service struct {
	apiclient.UnimplementedSportsHistoryServiceServer

	store                   *sportshistorystore.SQLStore
	gammaClient             gammaClient
	clobClient              clobClient
	sportsHistoryPageLimit  int
	nowFn                   func() time.Time
	startStopMu             sync.Mutex
	started                 bool
	runCancel               context.CancelFunc
	runWG                   sync.WaitGroup
	cacheMu                 sync.RWMutex
	sportsHistoryStale      bool
	sportsHistorySyncStatus *v1alpha1.SportsHistorySyncStatus
	syncGroup               singleflight.Group
}

func NewService(store *sportshistorystore.SQLStore, opts ...ServiceOption) *Service {
	service := &Service{
		store:                   store,
		sportsHistoryPageLimit:  defaultSportsHistoryEventPageLimit,
		nowFn:                   time.Now,
		sportsHistoryStale:      true,
		sportsHistorySyncStatus: &v1alpha1.SportsHistorySyncStatus{State: sportsHistorySyncStateIdle},
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
		return status.Error(codes.FailedPrecondition, "sports history store is required")
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
	go s.runSportsHistorySync(ctx)
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

func (s *Service) GetSportsHistoryStatus(context.Context, *apiclient.GetSportsHistoryStatusRequest) (*apiclient.GetSportsHistoryStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()
	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetSportsHistoryStatusResponse{Started: started, Status: statusText}, nil
}

func (s *Service) now() time.Time {
	if s.nowFn == nil {
		return time.Now()
	}
	return s.nowFn()
}
