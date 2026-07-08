package ethereumapi

import (
	"context"
	"sync"

	"github.com/useryege/athena/internal/ethereumapi/apiclient"
	utilethereumapi "github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ServiceOpts struct {
	EthereumAPI etherscanClient
}

type etherscanClient interface {
	GetSourceCode(ctx context.Context, chainID int64, contractAddress string) (*utilethereumapi.SourceCodeResponse, error)
	ListNormalTransactions(ctx context.Context, opts utilethereumapi.ListNormalTransactionsOptions) (*utilethereumapi.NormalTransactionsResponse, error)
}

type Service struct {
	apiclient.UnimplementedEthereumAPIServiceServer
	ethereumAPI etherscanClient

	startStopMu sync.Mutex
	started     bool
}

func NewService(opts ServiceOpts) *Service {
	return &Service{
		ethereumAPI: opts.EthereumAPI,
	}
}

func (s *Service) Start(context.Context) error {
	s.startStopMu.Lock()
	if s.started {
		s.startStopMu.Unlock()
		return nil
	}
	if s.ethereumAPI == nil {
		s.startStopMu.Unlock()
		return status.Error(codes.FailedPrecondition, "ethereumapi client is required")
	}
	s.started = true
	s.startStopMu.Unlock()
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	if !s.started {
		s.startStopMu.Unlock()
		return nil
	}
	s.started = false
	s.startStopMu.Unlock()
	return nil
}

func (s *Service) GetEthereumAPIStatus(context.Context, *apiclient.GetEthereumAPIStatusRequest) (*apiclient.GetEthereumAPIStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetEthereumAPIStatusResponse{
		Started: started,
		Status:  statusText,
	}, nil
}
