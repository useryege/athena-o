package application

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/useryege/athena/internal/application/chainwatcher"

	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
)

type Service struct {
	chainwatcher *chainwatcher.Server
}

func NewService(nodeClient *ethclient.Client) *Service {
	return &Service{
		chainwatcher: chainwatcher.NewServer(nodeClient),
	}
}

func (s *Service) StartChainWatcher(ctx context.Context, req *applicationpkg.StartChainWatcherRequest) (*applicationpkg.StartChainWatcherResponse, error) {
	if err := s.chainwatcher.Start(); err != nil {
		return nil, err
	}
	return &applicationpkg.StartChainWatcherResponse{}, nil
}

func (s *Service) StopChainWatcher(ctx context.Context, req *applicationpkg.StopChainWatcherRequest) (*applicationpkg.StopChainWatcherResponse, error) {
	if err := s.chainwatcher.Stop(); err != nil {
		return nil, err
	}
	return &applicationpkg.StopChainWatcherResponse{}, nil
}
