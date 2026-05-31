package polymarket

import (
	"context"
	"sync"

	"github.com/useryege/athena/internal/polymarket/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	apiclient.UnimplementedPolymarketServiceServer
	store       *polymarketstore.SQLStore
	startStopMu sync.Mutex
	started     bool
}

func NewService(store *polymarketstore.SQLStore) *Service {
	return &Service{store: store}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}
	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "polymarket store is required")
	}
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	s.started = false
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
