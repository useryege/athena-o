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
	listReq  *wormapiclient.ListWormMarketsRequest
	listResp *wormapiclient.ListWormMarketsResponse
}

func (f *fakeWormServiceClient) GetWormStatus(context.Context, *wormapiclient.GetWormStatusRequest, ...grpc.CallOption) (*wormapiclient.GetWormStatusResponse, error) {
	return &wormapiclient.GetWormStatusResponse{Started: true, Status: "running"}, nil
}

func (f *fakeWormServiceClient) ListWormMarkets(_ context.Context, req *wormapiclient.ListWormMarketsRequest, _ ...grpc.CallOption) (*wormapiclient.ListWormMarketsResponse, error) {
	f.listReq = req
	return f.listResp, nil
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

	resp, err := NewServer(&fakeWormClientset{client: client}).ListWormMarkets(context.Background(), &wormpkg.ListWormMarketsRequest{Limit: 20, Cursor: "cursor-1"})
	if err != nil {
		t.Fatalf("ListWormMarkets: %v", err)
	}
	if client.listReq.GetLimit() != 20 {
		t.Fatalf("limit = %d, want 20", client.listReq.GetLimit())
	}
	if client.listReq.GetCursor() != "cursor-1" {
		t.Fatalf("cursor = %q, want cursor-1", client.listReq.GetCursor())
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
