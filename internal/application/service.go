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

func (s *Service) TestChainWatcher(ctx context.Context, req *applicationpkg.TestChainWatcherRequest) (*applicationpkg.TestChainWatcherResponse, error) {
	err := s.chainwatcher.TestChainWatcher(ctx, req.StartBlock, req.EndBlock)
	if err != nil {
		return nil, err
	}
	return &applicationpkg.TestChainWatcherResponse{}, nil
}
