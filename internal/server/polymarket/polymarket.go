package polymarket

import (
	"context"

	polymarketapiclient "github.com/useryege/athena/internal/polymarket/apiclient"
	polymarketpkg "github.com/useryege/athena/pkg/apiclient/polymarket"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type Server struct {
	polymarketpkg.UnimplementedPolymarketServiceServer
	polymarketClientSet polymarketapiclient.Clientset
}

func NewServer(polymarketClientSet polymarketapiclient.Clientset) *Server {
	return &Server{polymarketClientSet: polymarketClientSet}
}

func (s *Server) GetPolymarketStatus(ctx context.Context, _ *polymarketpkg.GetPolymarketStatusRequest) (*v1alpha1.PolymarketStatus, error) {
	closer, client, err := s.polymarketClientSet.NewPolymarketServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetPolymarketStatus(ctx, &polymarketapiclient.GetPolymarketStatusRequest{})
}
