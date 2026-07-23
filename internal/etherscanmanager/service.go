package etherscanmanager

import (
	"context"
	"sync"

	"github.com/useryege/athena/internal/etherscanmanager/apiclient"
	"github.com/useryege/athena/util/etherscanapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ServiceOpts struct {
	Manager requestManager
}

type requestManager interface {
	GetSourceCode(ctx context.Context, chainID int64, contractAddress string) (*etherscanapi.SourceCodeResponse, error)
	ListNormalTransactions(ctx context.Context, opts etherscanapi.ListNormalTransactionsOptions) (*etherscanapi.NormalTransactionsResponse, error)
}

type Service struct {
	apiclient.UnimplementedEtherscanManagerServiceServer
	manager requestManager

	startStopMu sync.Mutex
	started     bool
}

func NewService(opts ServiceOpts) *Service {
	return &Service{
		manager: opts.Manager,
	}
}

func (s *Service) Start(context.Context) error {
	s.startStopMu.Lock()
	if s.started {
		s.startStopMu.Unlock()
		return nil
	}
	if s.manager == nil {
		s.startStopMu.Unlock()
		return status.Error(codes.FailedPrecondition, "etherscan manager is required")
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

func (s *Service) GetEtherscanManagerStatus(context.Context, *apiclient.GetEtherscanManagerStatusRequest) (*apiclient.GetEtherscanManagerStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetEtherscanManagerStatusResponse{
		Started: started,
		Status:  statusText,
	}, nil
}
