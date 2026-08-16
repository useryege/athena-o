package marketradar

import (
	"context"

	marketradarapiclient "github.com/useryege/athena/internal/marketradar/apiclient"
	marketradarpkg "github.com/useryege/athena/pkg/apiclient/marketradar"
)

type Server struct {
	marketradarpkg.UnimplementedMarketRadarServiceServer
	marketRadarClientset marketradarapiclient.Clientset
}

func NewServer(marketRadarClientset marketradarapiclient.Clientset) *Server {
	return &Server{marketRadarClientset: marketRadarClientset}
}

func (s *Server) GetMarketRadarStatus(ctx context.Context, _ *marketradarpkg.GetMarketRadarStatusRequest) (*marketradarpkg.GetMarketRadarStatusResponse, error) {
	resp, err := s.marketRadarClientset.MarketRadar().GetMarketRadarStatus(ctx, &marketradarapiclient.GetMarketRadarStatusRequest{})
	if err != nil {
		return nil, err
	}

	return &marketradarpkg.GetMarketRadarStatusResponse{
		Started: resp.GetStarted(),
		Status:  resp.GetStatus(),
	}, nil
}

func (s *Server) ListHotMarkets(ctx context.Context, req *marketradarpkg.ListHotMarketsRequest) (*marketradarpkg.ListHotMarketsResponse, error) {
	resp, err := s.marketRadarClientset.MarketRadar().ListHotMarkets(ctx, &marketradarapiclient.ListHotMarketsRequest{Limit: req.GetLimit()})
	if err != nil {
		return nil, err
	}

	return &marketradarpkg.ListHotMarketsResponse{
		Items:            resp.GetItems(),
		FetchedAt:        resp.GetFetchedAt(),
		Stale:            resp.GetStale(),
		MonitoredMarkets: resp.GetMonitoredMarkets(),
		MonitoredTokens:  resp.GetMonitoredTokens(),
		CandidateCount:   resp.GetCandidateCount(),
	}, nil
}

func (s *Server) ListRealtimeMarkets(ctx context.Context, req *marketradarpkg.ListRealtimeMarketsRequest) (*marketradarpkg.ListRealtimeMarketsResponse, error) {
	resp, err := s.marketRadarClientset.MarketRadar().ListRealtimeMarkets(ctx, &marketradarapiclient.ListRealtimeMarketsRequest{Limit: req.GetLimit()})
	if err != nil {
		return nil, err
	}

	return &marketradarpkg.ListRealtimeMarketsResponse{
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

func (s *Server) ListMarketMovers(ctx context.Context, req *marketradarpkg.ListMarketMoversRequest) (*marketradarpkg.ListMarketMoversResponse, error) {
	resp, err := s.marketRadarClientset.MarketRadar().ListMarketMovers(ctx, &marketradarapiclient.ListMarketMoversRequest{Limit: req.GetLimit()})
	if err != nil {
		return nil, err
	}

	return &marketradarpkg.ListMarketMoversResponse{
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
