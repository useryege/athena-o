package worm

import (
	"context"
	"testing"

	wormapiclient "github.com/useryege/athena/internal/worm/apiclient"
	wormpkg "github.com/useryege/athena/pkg/apiclient/worm"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilio "github.com/useryege/athena/util/io"
	"google.golang.org/grpc"
)

type fakeWormClientset struct {
	client *fakeWormServiceClient
}

func (f *fakeWormClientset) NewWormServiceClient() (utilio.Closer, wormapiclient.WormServiceClient, error) {
	return utilio.NopCloser, f.client, nil
}

type fakeWormServiceClient struct {
	listReq       *wormapiclient.ListWormMarketsRequest
	listResp      *wormapiclient.ListWormMarketsResponse
	getMarketReq  *wormapiclient.GetWormMarketRequest
	getMarketResp *wormapiclient.GetWormMarketResponse
}

func (f *fakeWormServiceClient) GetWormStatus(context.Context, *wormapiclient.GetWormStatusRequest, ...grpc.CallOption) (*wormapiclient.GetWormStatusResponse, error) {
	return &wormapiclient.GetWormStatusResponse{Started: true, Status: "running"}, nil
}

func (f *fakeWormServiceClient) ListWormMarkets(_ context.Context, req *wormapiclient.ListWormMarketsRequest, _ ...grpc.CallOption) (*wormapiclient.ListWormMarketsResponse, error) {
	f.listReq = req
	return f.listResp, nil
}

func (f *fakeWormServiceClient) GetWormMarket(_ context.Context, req *wormapiclient.GetWormMarketRequest, _ ...grpc.CallOption) (*wormapiclient.GetWormMarketResponse, error) {
	f.getMarketReq = req
	return f.getMarketResp, nil
}

func TestListWormMarketsForwardsRequestAndMapsResponse(t *testing.T) {
	client := &fakeWormServiceClient{
		listResp: &wormapiclient.ListWormMarketsResponse{
			Markets: []*v1alpha1.WormMarketItem{{
				ConditionID:      "market-1",
				Title:            "Will Team A beat Team B?",
				Description:      "description",
				Logo:             "market-logo",
				LastTradePrice:   "0.68",
				State:            "open",
				Category:         "sports",
				Created:          1714300100,
				EventTitle:       "Team A vs Team B",
				EventConditionID: "event-1",
				EventLogo:        "event-logo",
				MarginEnabled:    true,
			}},
			NextCursor: "next-page",
		},
	}

	resp, err := NewServer(&fakeWormClientset{client: client}).ListWormMarkets(context.Background(), &wormpkg.ListWormMarketsRequest{
		Limit:        20,
		Cursor:       "cursor-1",
		SortOption:   "ending_soon",
		CategorySlug: "crypto",
	})
	if err != nil {
		t.Fatalf("ListWormMarkets: %v", err)
	}
	if client.listReq.GetLimit() != 20 {
		t.Fatalf("limit = %d, want 20", client.listReq.GetLimit())
	}
	if client.listReq.GetCursor() != "cursor-1" {
		t.Fatalf("cursor = %q, want cursor-1", client.listReq.GetCursor())
	}
	if client.listReq.GetSortOption() != "ending_soon" {
		t.Fatalf("sort option = %q, want ending_soon", client.listReq.GetSortOption())
	}
	if client.listReq.GetCategorySlug() != "crypto" {
		t.Fatalf("category slug = %q, want crypto", client.listReq.GetCategorySlug())
	}
	if resp.GetNextCursor() != "next-page" {
		t.Fatalf("next cursor = %q, want next-page", resp.GetNextCursor())
	}
	if len(resp.GetItems()) != 1 {
		t.Fatalf("items len = %d, want 1", len(resp.GetItems()))
	}
	item := resp.GetItems()[0]
	if item.ConditionID != "market-1" || item.Title != "Will Team A beat Team B?" {
		t.Fatalf("item = %#v", item)
	}
	if item.EventTitle != "Team A vs Team B" || item.EventConditionID != "event-1" {
		t.Fatalf("event fields = %#v", item)
	}
	if item.LastTradePrice != "0.68" || !item.MarginEnabled {
		t.Fatalf("market fields = %#v", item)
	}
}

func TestGetWormMarketForwardsRequestAndMapsResponse(t *testing.T) {
	client := &fakeWormServiceClient{
		getMarketResp: &wormapiclient.GetWormMarketResponse{
			Market: &v1alpha1.WormMarketDetail{
				Market: v1alpha1.WormMarketItem{
					ConditionID:    "market-1",
					Title:          "Will Team A beat Team B?",
					LastTradePrice: "0.68",
					State:          "open",
					MarginEnabled:  true,
				},
				Stats: v1alpha1.WormMarketStats{
					TotalVolume24H: "250",
					TradeCount:     42,
				},
				Prices: []v1alpha1.WormMarketPrice{{
					ConditionID: "market-1",
					Price:       "0.71",
					IsYes:       true,
				}},
				OrderBooks: []v1alpha1.WormMarketOrderBook{{
					Market: "market-1",
					IsYes:  true,
					Bid:    []v1alpha1.WormOrderBookLevel{{Price: "0.70", TotalAmount: "12"}},
				}},
			},
		},
	}

	resp, err := NewServer(&fakeWormClientset{client: client}).GetWormMarket(context.Background(), &wormpkg.GetWormMarketRequest{ConditionId: "market-1"})
	if err != nil {
		t.Fatalf("GetWormMarket: %v", err)
	}
	if client.getMarketReq.GetConditionId() != "market-1" {
		t.Fatalf("condition id = %q, want market-1", client.getMarketReq.GetConditionId())
	}
	market := resp.GetMarket()
	if market.Market.ConditionID != "market-1" || market.Market.Title != "Will Team A beat Team B?" {
		t.Fatalf("market = %#v", market)
	}
	if market.Stats.TotalVolume24H != "250" || market.Stats.TradeCount != 42 {
		t.Fatalf("stats = %#v", market.Stats)
	}
	if len(market.Prices) != 1 || market.Prices[0].Price != "0.71" || !market.Prices[0].IsYes {
		t.Fatalf("prices = %#v", market.Prices)
	}
	if len(market.OrderBooks) != 1 || market.OrderBooks[0].Bid[0].TotalAmount != "12" {
		t.Fatalf("order books = %#v", market.OrderBooks)
	}
}
