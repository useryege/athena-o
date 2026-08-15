package sportshistory

import (
	"context"

	sportshistoryapiclient "github.com/useryege/athena/internal/sportshistory/apiclient"
	sportshistorypkg "github.com/useryege/athena/pkg/apiclient/sportshistory"
)

type Server struct {
	sportshistorypkg.UnimplementedSportsHistoryServiceServer
	sportsHistoryClientset sportshistoryapiclient.Clientset
}

func NewServer(sportsHistoryClientset sportshistoryapiclient.Clientset) *Server {
	return &Server{sportsHistoryClientset: sportsHistoryClientset}
}

func (s *Server) GetSportsHistoryStatus(ctx context.Context, _ *sportshistorypkg.GetSportsHistoryStatusRequest) (*sportshistorypkg.GetSportsHistoryStatusResponse, error) {
	closer, client, err := s.sportsHistoryClientset.NewSportsHistoryServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetSportsHistoryStatus(ctx, &sportshistoryapiclient.GetSportsHistoryStatusRequest{})
	if err != nil {
		return nil, err
	}

	return &sportshistorypkg.GetSportsHistoryStatusResponse{
		Started: resp.GetStarted(),
		Status:  resp.GetStatus(),
	}, nil
}

func (s *Server) ListSportsHistoryEvents(ctx context.Context, req *sportshistorypkg.ListSportsHistoryEventsRequest) (*sportshistorypkg.ListSportsHistoryEventsResponse, error) {
	closer, client, err := s.sportsHistoryClientset.NewSportsHistoryServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListSportsHistoryEvents(ctx, &sportshistoryapiclient.ListSportsHistoryEventsRequest{Limit: req.GetLimit()})
	if err != nil {
		return nil, err
	}

	return &sportshistorypkg.ListSportsHistoryEventsResponse{
		Items:     resp.GetItems(),
		FetchedAt: resp.GetFetchedAt(),
		Stale:     resp.GetStale(),
	}, nil
}

func (s *Server) BatchGetSportsHistoryPriceHistories(ctx context.Context, req *sportshistorypkg.BatchGetSportsHistoryPriceHistoriesRequest) (*sportshistorypkg.BatchGetSportsHistoryPriceHistoriesResponse, error) {
	closer, client, err := s.sportsHistoryClientset.NewSportsHistoryServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.BatchGetSportsHistoryPriceHistories(ctx, &sportshistoryapiclient.BatchGetSportsHistoryPriceHistoriesRequest{
		MarketKeys:    req.GetMarketKeys(),
		LimitPerToken: req.GetLimitPerToken(),
	})
	if err != nil {
		return nil, err
	}

	return &sportshistorypkg.BatchGetSportsHistoryPriceHistoriesResponse{Items: resp.GetItems()}, nil
}

func (s *Server) GetSportsHistorySyncStatus(ctx context.Context, _ *sportshistorypkg.GetSportsHistorySyncStatusRequest) (*sportshistorypkg.GetSportsHistorySyncStatusResponse, error) {
	closer, client, err := s.sportsHistoryClientset.NewSportsHistoryServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetSportsHistorySyncStatus(ctx, &sportshistoryapiclient.GetSportsHistorySyncStatusRequest{})
	if err != nil {
		return nil, err
	}

	return &sportshistorypkg.GetSportsHistorySyncStatusResponse{Status: resp.GetStatus()}, nil
}

func (s *Server) RefreshSportsHistory(ctx context.Context, _ *sportshistorypkg.RefreshSportsHistoryRequest) (*sportshistorypkg.RefreshSportsHistoryResponse, error) {
	closer, client, err := s.sportsHistoryClientset.NewSportsHistoryServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.RefreshSportsHistory(ctx, &sportshistoryapiclient.RefreshSportsHistoryRequest{})
	if err != nil {
		return nil, err
	}

	return &sportshistorypkg.RefreshSportsHistoryResponse{Status: resp.GetStatus()}, nil
}
