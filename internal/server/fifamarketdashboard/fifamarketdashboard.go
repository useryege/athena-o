package fifamarketdashboard

import (
	"context"

	fifamarketdashboardapiclient "github.com/useryege/athena/internal/fifamarketdashboard/apiclient"
	fifamarketdashboardpkg "github.com/useryege/athena/pkg/apiclient/fifamarketdashboard"
	"github.com/useryege/athena/util/session"
)

type Server struct {
	fifamarketdashboardpkg.UnimplementedFIFAMarketDashboardServiceServer
	fifaMarketDashboardClientset fifamarketdashboardapiclient.Clientset
}

func NewServer(fifaMarketDashboardClientset fifamarketdashboardapiclient.Clientset) *Server {
	return &Server{fifaMarketDashboardClientset: fifaMarketDashboardClientset}
}

func (s *Server) GetFIFAMarketDashboardStatus(ctx context.Context, _ *fifamarketdashboardpkg.GetFIFAMarketDashboardStatusRequest) (*fifamarketdashboardpkg.GetFIFAMarketDashboardStatusResponse, error) {
	closer, client, err := s.fifaMarketDashboardClientset.NewFIFAMarketDashboardServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetFIFAMarketDashboardStatus(ctx, &fifamarketdashboardapiclient.GetFIFAMarketDashboardStatusRequest{})
	if err != nil {
		return nil, err
	}

	return &fifamarketdashboardpkg.GetFIFAMarketDashboardStatusResponse{
		Started: resp.GetStarted(),
		Status:  resp.GetStatus(),
	}, nil
}

func (s *Server) GetFIFAMarketDashboard(ctx context.Context, _ *fifamarketdashboardpkg.GetFIFAMarketDashboardRequest) (*fifamarketdashboardpkg.GetFIFAMarketDashboardResponse, error) {
	closer, client, err := s.fifaMarketDashboardClientset.NewFIFAMarketDashboardServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetFIFAMarketDashboard(ctx, &fifamarketdashboardapiclient.GetFIFAMarketDashboardRequest{
		Requester: session.GetUserIdentifier(ctx),
	})
	if err != nil {
		return nil, err
	}
	return &fifamarketdashboardpkg.GetFIFAMarketDashboardResponse{Dashboard: resp.GetDashboard()}, nil
}

func (s *Server) UpdateFIFAEventConfig(ctx context.Context, req *fifamarketdashboardpkg.UpdateFIFAEventConfigRequest) (*fifamarketdashboardpkg.UpdateFIFAEventConfigResponse, error) {
	closer, client, err := s.fifaMarketDashboardClientset.NewFIFAMarketDashboardServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateFIFAEventConfig(ctx, &fifamarketdashboardapiclient.UpdateFIFAEventConfigRequest{
		Config: req.GetConfig(),
	})
	if err != nil {
		return nil, err
	}
	return &fifamarketdashboardpkg.UpdateFIFAEventConfigResponse{Config: resp.GetConfig()}, nil
}
