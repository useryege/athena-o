package fifamarketdashboard

import (
	"context"

	"github.com/useryege/athena/internal/accountaccess"
	fifamarketdashboardapiclient "github.com/useryege/athena/internal/fifamarketdashboard/apiclient"
	fifamarketdashboardpkg "github.com/useryege/athena/pkg/apiclient/fifamarketdashboard"
	"github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	fifamarketdashboardpkg.UnimplementedFIFAMarketDashboardServiceServer
	fifaMarketDashboardClientset fifamarketdashboardapiclient.Clientset
	accessController             *accountaccess.Controller
}

func NewServer(fifaMarketDashboardClientset fifamarketdashboardapiclient.Clientset, accessController *accountaccess.Controller) *Server {
	return &Server{fifaMarketDashboardClientset: fifaMarketDashboardClientset, accessController: accessController}
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
	accountID, administrator, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.fifaMarketDashboardClientset.FIFAMarketDashboard().GetFIFAMarketDashboard(ctx, &fifamarketdashboardapiclient.GetFIFAMarketDashboardRequest{
		RequesterAccountId:     accountID,
		RequesterAdministrator: administrator,
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

func (s *Server) requester(ctx context.Context) (string, bool, error) {
	accountID := session.AccountID(ctx)
	if accountID == "" {
		return "", false, status.Error(codes.Unauthenticated, "authenticated account ID is missing")
	}
	if s.accessController == nil {
		return "", false, status.Error(codes.Internal, "account access controller is not configured")
	}
	access, err := s.accessController.Get(accountID)
	if err != nil {
		return "", false, err
	}
	return accountID, access.Administrator, nil
}
