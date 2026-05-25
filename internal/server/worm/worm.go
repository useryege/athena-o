package worm

import (
	"context"

	wormapiclient "github.com/useryege/athena/internal/worm/apiclient"
	wormpkg "github.com/useryege/athena/pkg/apiclient/worm"
)

type Server struct {
	wormpkg.UnimplementedWormServiceServer
	wormClientSet wormapiclient.Clientset
}

func NewServer(wormClientSet wormapiclient.Clientset) *Server {
	return &Server{wormClientSet: wormClientSet}
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

func (s *Server) ListWormMarkets(ctx context.Context, req *wormpkg.ListWormMarketsRequest) (*wormpkg.ListWormMarketsResponse, error) {
	closer, client, err := s.wormClientSet.NewWormServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListWormMarkets(ctx, &wormapiclient.ListWormMarketsRequest{
		Limit:  req.GetLimit(),
		Cursor: req.GetCursor(),
	})
	if err != nil {
		return nil, err
	}

	items := make([]*wormpkg.WormMarketItem, 0, len(resp.GetMarkets()))
	for _, market := range resp.GetMarkets() {
		items = append(items, &wormpkg.WormMarketItem{
			ConditionId:      market.GetConditionId(),
			Title:            market.GetTitle(),
			Description:      market.GetDescription(),
			Logo:             market.GetLogo(),
			LastTradePrice:   market.GetLastTradePrice(),
			State:            market.GetState(),
			Category:         market.GetCategory(),
			Created:          market.GetCreated(),
			EventTitle:       market.GetEventTitle(),
			EventConditionId: market.GetEventConditionId(),
			EventLogo:        market.GetEventLogo(),
			MarginEnabled:    market.GetMarginEnabled(),
		})
	}

	return &wormpkg.ListWormMarketsResponse{
		Items:      items,
		NextCursor: resp.GetNextCursor(),
	}, nil
}
