package wormpoly

import (
	"context"
	"sync"

	"github.com/useryege/athena/internal/wormpoly/apiclient"
)

type Service struct {
	apiclient.UnimplementedWormPolyServiceServer
	startStopMu sync.Mutex
	started     bool
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
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
