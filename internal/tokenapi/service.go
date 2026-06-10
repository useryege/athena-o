package tokenapi

import (
	"sync"

	athenacommon "github.com/useryege/athena/common"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/internal/tokenapi/apiclient"
)

type Service struct {
	apiclient.UnimplementedTokenAPIServiceServer
	startStopMu sync.Mutex
	store       *tokenstore.SQLStore
	nodeWSURLs  map[int64][]string
	useProxy    bool
}

type ServiceOpts struct {
	Store          *tokenstore.SQLStore
	EthNodeWSURLs  []string
	BSCNodeWSURLs  []string
	NodeWSUseProxy bool
}

func NewService(opts ServiceOpts) *Service {
	s := &Service{
		nodeWSURLs: map[int64][]string{
			athenacommon.ChainIDEthereumMainnet: opts.EthNodeWSURLs,
			athenacommon.ChainIDBSCMainnet:      opts.BSCNodeWSURLs,
		},
		useProxy: opts.NodeWSUseProxy,
	}
	if opts.Store != nil {
		s.store = opts.Store
	}
	return s
}

func NewServiceWithStore(stores ...*tokenstore.SQLStore) *Service {
	s := NewService(ServiceOpts{})
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

func (s *Service) nodeConfig(chainID int64) ([]string, bool) {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	urls, ok := s.nodeWSURLs[chainID]
	return urls, ok
}

func (s *Service) nodeWSUseProxy() bool {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	return s.useProxy
}

func (s *Service) Stop() error {
	return nil
}
