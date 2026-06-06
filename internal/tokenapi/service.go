package tokenapi

import (
	"context"
	"sync"

	"github.com/useryege/athena/internal/tokenapi/apiclient"
)

type Service struct {
	apiclient.UnimplementedTokenAPIServiceServer
	startStopMu sync.Mutex
	started     bool
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	s.started = false
	return nil
}

func (s *Service) GetTokenAPIStatus(context.Context, *apiclient.GetTokenAPIStatusRequest) (*apiclient.GetTokenAPIStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetTokenAPIStatusResponse{
		Started: started,
		Status:  statusText,
	}, nil
}
