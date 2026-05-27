package solidity

import (
	"context"

	solidityapiclient "github.com/useryege/athena/internal/solidity/apiclient"
	soliditypkg "github.com/useryege/athena/pkg/apiclient/solidity"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type Server struct {
	soliditypkg.UnimplementedSolidityServiceServer
	solidityClientSet solidityapiclient.Clientset
}

func NewServer(solidityClientSet solidityapiclient.Clientset) *Server {
	return &Server{solidityClientSet: solidityClientSet}
}

func (s *Server) GetSolidityStatus(ctx context.Context, _ *soliditypkg.GetSolidityStatusRequest) (*v1alpha1.SolidityStatus, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetSolidityStatus(ctx, &solidityapiclient.GetSolidityStatusRequest{})
}
