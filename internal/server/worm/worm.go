package worm

import (
	"context"

	serverfifa "github.com/useryege/athena/internal/server/fifa"
	wormapiclient "github.com/useryege/athena/internal/worm/apiclient"
	wormpkg "github.com/useryege/athena/pkg/apiclient/worm"
)

type Server struct {
	wormpkg.UnimplementedWormServiceServer
	wormClientSet wormapiclient.Clientset
	fifaCache     *serverfifa.Cache
}

func NewServer(wormClientSet wormapiclient.Clientset, fifaCache *serverfifa.Cache) *Server {
	return &Server{wormClientSet: wormClientSet, fifaCache: fifaCache}
}

func (s *Server) GetWormStatus(ctx context.Context, _ *wormpkg.GetWormStatusRequest) (*wormpkg.GetWormStatusResponse, error) {
	closer, client, err := s.wormClientSet.NewWormServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetWormStatus(ctx, &wormapiclient.GetWormStatusRequest{})
	if err != nil {
		return nil, err
	}

	return &wormpkg.GetWormStatusResponse{
		Started: resp.GetStarted(),
		Status:  resp.GetStatus(),
	}, nil
}

func (s *Server) GetWormEvent(_ context.Context, req *wormpkg.GetWormEventRequest) (*wormpkg.GetWormEventResponse, error) {
	item, fetchedAt, err := s.fifaCache.GetWormEvent(req.GetConditionId())
	if err != nil {
		return nil, err
	}

	return &wormpkg.GetWormEventResponse{
		Item:      item,
		FetchedAt: fetchedAt,
	}, nil
}

func (s *Server) ListWormEvents(ctx context.Context, req *wormpkg.ListWormEventsRequest) (*wormpkg.ListWormEventsResponse, error) {
	closer, client, err := s.wormClientSet.NewWormServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListWormEvents(ctx, &wormapiclient.ListWormEventsRequest{
		Limit:        req.GetLimit(),
		Cursor:       req.GetCursor(),
		SortOption:   req.GetSortOption(),
		CategorySlug: req.GetCategorySlug(),
	})
	if err != nil {
		return nil, err
	}

	return &wormpkg.ListWormEventsResponse{
		Items:      resp.GetEvents(),
		NextCursor: resp.GetNextCursor(),
		FetchedAt:  resp.GetFetchedAt(),
		Stale:      resp.GetStale(),
	}, nil
}
