package predpoly

import (
	"context"
	"sync"

	"github.com/useryege/athena/internal/predpoly/apiclient"
)

type Service struct {
	apiclient.UnimplementedPredPolyServiceServer
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

func (s *Service) GetPredPolyStatus(context.Context, *apiclient.GetPredPolyStatusRequest) (*apiclient.GetPredPolyStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetPredPolyStatusResponse{
		Started: started,
		Status:  statusText,
	}, nil
}
