package wormpoly

import (
	"context"

	wormpolyapiclient "github.com/useryege/athena/internal/wormpoly/apiclient"
	wormpolypkg "github.com/useryege/athena/pkg/apiclient/wormpoly"
	"github.com/useryege/athena/util/session"
)

type Server struct {
	wormpolypkg.UnimplementedWormPolyServiceServer
	wormPolyClientset wormpolyapiclient.Clientset
}

func NewServer(wormPolyClientset wormpolyapiclient.Clientset) *Server {
	return &Server{wormPolyClientset: wormPolyClientset}
}

func (s *Server) GetWormPolyStatus(ctx context.Context, _ *wormpolypkg.GetWormPolyStatusRequest) (*wormpolypkg.GetWormPolyStatusResponse, error) {
	closer, client, err := s.wormPolyClientset.NewWormPolyServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetWormPolyStatus(ctx, &wormpolyapiclient.GetWormPolyStatusRequest{})
	if err != nil {
		return nil, err
	}

	return &wormpolypkg.GetWormPolyStatusResponse{
		Started: resp.GetStarted(),
		Status:  resp.GetStatus(),
	}, nil
}

func (s *Server) GetWormPolyFIFADashboard(ctx context.Context, _ *wormpolypkg.GetWormPolyFIFADashboardRequest) (*wormpolypkg.GetWormPolyFIFADashboardResponse, error) {
	closer, client, err := s.wormPolyClientset.NewWormPolyServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetWormPolyFIFADashboard(ctx, &wormpolyapiclient.GetWormPolyFIFADashboardRequest{
		Requester: session.GetUserIdentifier(ctx),
	})
	if err != nil {
		return nil, err
	}
	return &wormpolypkg.GetWormPolyFIFADashboardResponse{Dashboard: resp.GetDashboard()}, nil
}

func (s *Server) UpdateWormPolyFIFAEventConfig(ctx context.Context, req *wormpolypkg.UpdateWormPolyFIFAEventConfigRequest) (*wormpolypkg.UpdateWormPolyFIFAEventConfigResponse, error) {
	closer, client, err := s.wormPolyClientset.NewWormPolyServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateWormPolyFIFAEventConfig(ctx, &wormpolyapiclient.UpdateWormPolyFIFAEventConfigRequest{
		Config: req.GetConfig(),
	})
	if err != nil {
		return nil, err
	}
	return &wormpolypkg.UpdateWormPolyFIFAEventConfigResponse{Config: resp.GetConfig()}, nil
}
