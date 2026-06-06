package token

import (
	"context"
	"sync"

	"github.com/useryege/athena/internal/token/apiclient"
)

type Service struct {
	apiclient.UnimplementedTokenServiceServer
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

func (s *Service) GetTokenStatus(context.Context, *apiclient.GetTokenStatusRequest) (*apiclient.GetTokenStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetTokenStatusResponse{
		Started: started,
		Status:  statusText,
	}, nil
}
