package tokenapi

import (
	"sync"

	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/internal/tokenapi/apiclient"
)

type Service struct {
	apiclient.UnimplementedTokenAPIServiceServer
	startStopMu sync.Mutex
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
	return nil
}

func (s *Service) tokenStore() *tokenstore.SQLStore {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	return s.store
}

func (s *Service) Stop() error {
	return nil
}
