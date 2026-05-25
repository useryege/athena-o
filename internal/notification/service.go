package notification

import (
	"context"
	"sync"

	"github.com/useryege/athena/internal/notification/apiclient"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	apiclient.UnimplementedNotificationServiceServer
	store       *notificationstore.SQLStore
	startStopMu sync.Mutex
	started     bool
}

func NewService(store *notificationstore.SQLStore) *Service {
	return &Service{store: store}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}
	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "notification store is required")
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

func (s *Service) GetNotificationStatus(context.Context, *apiclient.GetNotificationStatusRequest) (*apiclient.GetNotificationStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetNotificationStatusResponse{
		Started: started,
		Status:  statusText,
	}, nil
}
