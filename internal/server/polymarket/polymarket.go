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

func (s *Server) ListPolymarketHotMarkets(ctx context.Context, req *polymarketpkg.ListPolymarketHotMarketsRequest) (*polymarketpkg.ListPolymarketHotMarketsResponse, error) {
	closer, client, err := s.polymarketClientSet.NewPolymarketServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListPolymarketHotMarkets(ctx, &polymarketapiclient.ListPolymarketHotMarketsRequest{
		Limit: req.GetLimit(),
	})
	if err != nil {
		return nil, err
	}

	return &polymarketpkg.ListPolymarketHotMarketsResponse{
		Items:            resp.GetItems(),
		FetchedAt:        resp.GetFetchedAt(),
		Stale:            resp.GetStale(),
		MonitoredMarkets: resp.GetMonitoredMarkets(),
		MonitoredTokens:  resp.GetMonitoredTokens(),
		CandidateCount:   resp.GetCandidateCount(),
	}, nil
}

func (s *Server) ListPolymarketRealtimeMarkets(ctx context.Context, req *polymarketpkg.ListPolymarketRealtimeMarketsRequest) (*polymarketpkg.ListPolymarketRealtimeMarketsResponse, error) {
	closer, client, err := s.polymarketClientSet.NewPolymarketServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListPolymarketRealtimeMarkets(ctx, &polymarketapiclient.ListPolymarketRealtimeMarketsRequest{
		Limit: req.GetLimit(),
	})
	if err != nil {
		return nil, err
	}

	return &polymarketpkg.ListPolymarketRealtimeMarketsResponse{
		Items:             resp.GetItems(),
		FetchedAt:         resp.GetFetchedAt(),
		Stale:             resp.GetStale(),
		SubscribedMarkets: resp.GetSubscribedMarkets(),
		SubscribedTokens:  resp.GetSubscribedTokens(),
		Connected:         resp.GetConnected(),
		LastEventAt:       resp.GetLastEventAt(),
		CandidateCount:    resp.GetCandidateCount(),
	}, nil
}

func (s *Server) ListPolymarketMovers(ctx context.Context, req *polymarketpkg.ListPolymarketMoversRequest) (*polymarketpkg.ListPolymarketMoversResponse, error) {
	closer, client, err := s.polymarketClientSet.NewPolymarketServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListPolymarketMovers(ctx, &polymarketapiclient.ListPolymarketMoversRequest{
		Limit: req.GetLimit(),
	})
	if err != nil {
		return nil, err
	}

	return &polymarketpkg.ListPolymarketMoversResponse{
		Items:            resp.GetItems(),
		FetchedAt:        resp.GetFetchedAt(),
		Stale:            resp.GetStale(),
		Connected:        resp.GetConnected(),
		LastEventAt:      resp.GetLastEventAt(),
		MonitoredMarkets: resp.GetMonitoredMarkets(),
		MonitoredTokens:  resp.GetMonitoredTokens(),
		CandidateCount:   resp.GetCandidateCount(),
	}, nil
}

func (s *Server) ListPolymarketSportsLiveEvents(ctx context.Context, req *polymarketpkg.ListPolymarketSportsLiveEventsRequest) (*polymarketpkg.ListPolymarketSportsLiveEventsResponse, error) {
	closer, client, err := s.polymarketClientSet.NewPolymarketServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListPolymarketSportsLiveEvents(ctx, &polymarketapiclient.ListPolymarketSportsLiveEventsRequest{
		Limit: req.GetLimit(),
	})
	if err != nil {
		return nil, err
	}

	return &polymarketpkg.ListPolymarketSportsLiveEventsResponse{
		Items:     resp.GetItems(),
		FetchedAt: resp.GetFetchedAt(),
		Stale:     resp.GetStale(),
	}, nil
}

func (s *Server) BatchGetPolymarketSportsLivePriceHistory(ctx context.Context, req *polymarketpkg.BatchGetPolymarketSportsLivePriceHistoryRequest) (*polymarketpkg.BatchGetPolymarketSportsLivePriceHistoryResponse, error) {
	closer, client, err := s.polymarketClientSet.NewPolymarketServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.BatchGetPolymarketSportsLivePriceHistory(ctx, &polymarketapiclient.BatchGetPolymarketSportsLivePriceHistoryRequest{
		MarketKeys:    req.GetMarketKeys(),
		LimitPerToken: req.GetLimitPerToken(),
	})
	if err != nil {
		return nil, err
	}

	return &polymarketpkg.BatchGetPolymarketSportsLivePriceHistoryResponse{
		Items: resp.GetItems(),
	}, nil
}

func (s *Server) ListPolymarketSportsHistoryEvents(ctx context.Context, req *polymarketpkg.ListPolymarketSportsHistoryEventsRequest) (*polymarketpkg.ListPolymarketSportsHistoryEventsResponse, error) {
	closer, client, err := s.polymarketClientSet.NewPolymarketServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListPolymarketSportsHistoryEvents(ctx, &polymarketapiclient.ListPolymarketSportsHistoryEventsRequest{
		Limit: req.GetLimit(),
	})
	if err != nil {
		return nil, err
	}
	return &polymarketpkg.ListPolymarketSportsHistoryEventsResponse{
		Items:     resp.GetItems(),
		FetchedAt: resp.GetFetchedAt(),
		Stale:     resp.GetStale(),
	}, nil
}

func (s *Server) BatchGetPolymarketSportsHistoryPriceHistory(ctx context.Context, req *polymarketpkg.BatchGetPolymarketSportsHistoryPriceHistoryRequest) (*polymarketpkg.BatchGetPolymarketSportsHistoryPriceHistoryResponse, error) {
	closer, client, err := s.polymarketClientSet.NewPolymarketServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.BatchGetPolymarketSportsHistoryPriceHistory(ctx, &polymarketapiclient.BatchGetPolymarketSportsHistoryPriceHistoryRequest{
		MarketKeys:    req.GetMarketKeys(),
		LimitPerToken: req.GetLimitPerToken(),
	})
	if err != nil {
		return nil, err
	}
	return &polymarketpkg.BatchGetPolymarketSportsHistoryPriceHistoryResponse{
		Items: resp.GetItems(),
	}, nil
}
