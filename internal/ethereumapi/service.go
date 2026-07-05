package ethereumapi

import (
	"context"
	"sync"

	"github.com/useryege/athena/internal/ethereumapi/apiclient"
	ethereumapistore "github.com/useryege/athena/internal/ethereumapi/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	apiclient.UnimplementedEthereumAPIServiceServer
	store       *ethereumapistore.SQLStore
	startStopMu sync.Mutex
	started     bool
}

func NewService(store *ethereumapistore.SQLStore) *Service {
	return &Service{store: store}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}
	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "ethereumapi store is required")
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

func (s *Service) GetEthereumAPIStatus(context.Context, *apiclient.GetEthereumAPIStatusRequest) (*apiclient.GetEthereumAPIStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetEthereumAPIStatusResponse{
		Started: started,
		Status:  statusText,
	}, nil
}
