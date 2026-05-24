package worm

import (
	"context"
	"sync"

	"github.com/useryege/athena/internal/worm/apiclient"
	wormstore "github.com/useryege/athena/internal/worm/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	apiclient.UnimplementedWormServiceServer
	store       *wormstore.SQLStore
	startStopMu sync.Mutex
	started     bool
}

func NewService(store *wormstore.SQLStore) *Service {
	return &Service{store: store}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}
	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "worm store is required")
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

func (s *Service) GetWormStatus(context.Context, *apiclient.GetWormStatusRequest) (*apiclient.GetWormStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetWormStatusResponse{
		Started: started,
		Status:  statusText,
	}, nil
}
