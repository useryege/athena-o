package nodescanner

import (
	"context"
	"fmt"

	"github.com/useryege/athena/internal/nodescanner/apiclient"
)

type Service struct {
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Ping(_ context.Context, req *apiclient.PingRequest) (*apiclient.PingResponse, error) {
	return &apiclient.PingResponse{
		Message: fmt.Sprintf("node-scanner is ready: %s", req.GetClientId()),
	}, nil
}
