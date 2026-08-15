package sportslive

import (
	"context"

	sportsliveapiclient "github.com/useryege/athena/internal/sportslive/apiclient"
	sportslivepkg "github.com/useryege/athena/pkg/apiclient/sportslive"
)

type Server struct {
	sportslivepkg.UnimplementedSportsLiveServiceServer
	sportsLiveClientset sportsliveapiclient.Clientset
}

func NewServer(sportsLiveClientset sportsliveapiclient.Clientset) *Server {
	return &Server{sportsLiveClientset: sportsLiveClientset}
}

func (s *Server) GetSportsLiveStatus(ctx context.Context, _ *sportslivepkg.GetSportsLiveStatusRequest) (*sportslivepkg.GetSportsLiveStatusResponse, error) {
	closer, client, err := s.sportsLiveClientset.NewSportsLiveServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetSportsLiveStatus(ctx, &sportsliveapiclient.GetSportsLiveStatusRequest{})
	if err != nil {
		return nil, err
	}

	return &sportslivepkg.GetSportsLiveStatusResponse{
		Started: resp.GetStarted(),
		Status:  resp.GetStatus(),
	}, nil
}

func (s *Server) ListSportsLiveEvents(ctx context.Context, req *sportslivepkg.ListSportsLiveEventsRequest) (*sportslivepkg.ListSportsLiveEventsResponse, error) {
	closer, client, err := s.sportsLiveClientset.NewSportsLiveServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListSportsLiveEvents(ctx, &sportsliveapiclient.ListSportsLiveEventsRequest{Limit: req.GetLimit()})
	if err != nil {
		return nil, err
	}

	return &sportslivepkg.ListSportsLiveEventsResponse{
		Items:     resp.GetItems(),
		FetchedAt: resp.GetFetchedAt(),
		Stale:     resp.GetStale(),
	}, nil
}

func (s *Server) BatchGetSportsLivePriceHistories(ctx context.Context, req *sportslivepkg.BatchGetSportsLivePriceHistoriesRequest) (*sportslivepkg.BatchGetSportsLivePriceHistoriesResponse, error) {
	closer, client, err := s.sportsLiveClientset.NewSportsLiveServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.BatchGetSportsLivePriceHistories(ctx, &sportsliveapiclient.BatchGetSportsLivePriceHistoriesRequest{
		MarketKeys:    req.GetMarketKeys(),
		LimitPerToken: req.GetLimitPerToken(),
	})
	if err != nil {
		return nil, err
	}

	return &sportslivepkg.BatchGetSportsLivePriceHistoriesResponse{Items: resp.GetItems()}, nil
}
