package tokenapi

import (
	"context"
	"sync"

	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/internal/tokenapi/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type Service struct {
	apiclient.UnimplementedTokenAPIServiceServer
	startStopMu sync.Mutex
	started     bool
	store       *tokenstore.SQLStore
}

func NewService(stores ...*tokenstore.SQLStore) *Service {
	s := &Service{}
	if len(stores) > 0 {
		s.store = stores[0]
	}
	return s
}

func (s *Service) SetStore(store *tokenstore.SQLStore) {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	s.store = store
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	s.started = true
	return nil
}

func (s *Service) tokenStore() *tokenstore.SQLStore {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	return s.store
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	s.started = false
	return nil
}

func (s *Service) GetTokenAPIStatus(context.Context, *apiclient.GetTokenAPIStatusRequest) (*v1alpha1.TokenAPIStatus, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &v1alpha1.TokenAPIStatus{
		Started: started,
		Status:  statusText,
	}, nil
}
