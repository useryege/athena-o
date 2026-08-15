package wormmarkets

import (
	"context"

	wormmarketsapiclient "github.com/useryege/athena/internal/wormmarkets/apiclient"
	wormmarketspkg "github.com/useryege/athena/pkg/apiclient/wormmarkets"
)

type Server struct {
	wormmarketspkg.UnimplementedWormMarketsServiceServer
	wormMarketsClientset wormmarketsapiclient.Clientset
}

func NewServer(wormMarketsClientset wormmarketsapiclient.Clientset) *Server {
	return &Server{wormMarketsClientset: wormMarketsClientset}
}

func (s *Server) GetWormMarketsStatus(ctx context.Context, _ *wormmarketspkg.GetWormMarketsStatusRequest) (*wormmarketspkg.GetWormMarketsStatusResponse, error) {
	closer, client, err := s.wormMarketsClientset.NewWormMarketsServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetWormMarketsStatus(ctx, &wormmarketsapiclient.GetWormMarketsStatusRequest{})
	if err != nil {
		return nil, err
	}

	return &wormmarketspkg.GetWormMarketsStatusResponse{
		Started: resp.GetStarted(),
		Status:  resp.GetStatus(),
	}, nil
}

func (s *Server) GetWormEvent(ctx context.Context, req *wormmarketspkg.GetWormEventRequest) (*wormmarketspkg.GetWormEventResponse, error) {
	closer, client, err := s.wormMarketsClientset.NewWormMarketsServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetWormEvent(ctx, &wormmarketsapiclient.GetWormEventRequest{ConditionId: req.GetConditionId()})
	if err != nil {
		return nil, err
	}

	return &wormmarketspkg.GetWormEventResponse{
		Event:     resp.GetEvent(),
		FetchedAt: resp.GetFetchedAt(),
	}, nil
}

func (s *Server) ListWormEvents(ctx context.Context, req *wormmarketspkg.ListWormEventsRequest) (*wormmarketspkg.ListWormEventsResponse, error) {
	closer, client, err := s.wormMarketsClientset.NewWormMarketsServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListWormEvents(ctx, &wormmarketsapiclient.ListWormEventsRequest{
		Limit:        req.GetLimit(),
		Cursor:       req.GetCursor(),
		SortOption:   req.GetSortOption(),
		CategorySlug: req.GetCategorySlug(),
	})
	if err != nil {
		return nil, err
	}

	return &wormmarketspkg.ListWormEventsResponse{
		Items:      resp.GetEvents(),
		NextCursor: resp.GetNextCursor(),
		FetchedAt:  resp.GetFetchedAt(),
		Stale:      resp.GetStale(),
	}, nil
}
