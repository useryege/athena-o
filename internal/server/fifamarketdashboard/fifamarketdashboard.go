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
	resp, err := s.fifaMarketDashboardClientset.FIFAMarketDashboard().GetFIFAMarketDashboardStatus(ctx, &fifamarketdashboardapiclient.GetFIFAMarketDashboardStatusRequest{})
	if err != nil {
		return nil, err
	}

	return &fifamarketdashboardpkg.GetFIFAMarketDashboardStatusResponse{
		Started: resp.GetStarted(),
		Status:  resp.GetStatus(),
	}, nil
}

func (s *Server) GetFIFAMarketDashboard(ctx context.Context, _ *fifamarketdashboardpkg.GetFIFAMarketDashboardRequest) (*fifamarketdashboardpkg.GetFIFAMarketDashboardResponse, error) {
	resp, err := s.fifaMarketDashboardClientset.FIFAMarketDashboard().GetFIFAMarketDashboard(ctx, &fifamarketdashboardapiclient.GetFIFAMarketDashboardRequest{
		Requester: session.GetUserIdentifier(ctx),
	})
	if err != nil {
		return nil, err
	}
	return &fifamarketdashboardpkg.GetFIFAMarketDashboardResponse{Dashboard: resp.GetDashboard()}, nil
}

func (s *Server) UpdateFIFAEventConfig(ctx context.Context, req *fifamarketdashboardpkg.UpdateFIFAEventConfigRequest) (*fifamarketdashboardpkg.UpdateFIFAEventConfigResponse, error) {
	resp, err := s.fifaMarketDashboardClientset.FIFAMarketDashboard().UpdateFIFAEventConfig(ctx, &fifamarketdashboardapiclient.UpdateFIFAEventConfigRequest{
		Config: req.GetConfig(),
	})
	if err != nil {
		return nil, err
	}
	return &fifamarketdashboardpkg.UpdateFIFAEventConfigResponse{Config: resp.GetConfig()}, nil
}
