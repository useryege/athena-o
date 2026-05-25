package worm

import (
	"context"
	"testing"

	"github.com/useryege/athena/internal/worm/apiclient"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeWormMarketClient struct {
	options utilworm.ListMarketsOptions
	resp    *utilworm.ListMarketsResponse
	err     error
}

func (f *fakeWormMarketClient) ListMarkets(_ context.Context, options utilworm.ListMarketsOptions) (*utilworm.ListMarketsResponse, error) {
	f.options = options
	return f.resp, f.err
}

func TestListWormMarketsUsesSportsLeverageDefaults(t *testing.T) {
	nextCursor := "next-page"
	description := "market description"
	logo := "https://cdn.worm.wtf/m/1.png"
	price := "0.68"
	created := int64(1714300100)
	eventLogo := "https://cdn.worm.wtf/e/1.png"
	client := &fakeWormMarketClient{
		resp: &utilworm.ListMarketsResponse{
			Markets: []utilworm.MarketSummary{{
				ConditionID:    "market-1",
				Title:          "Will Team A beat Team B?",
				Description:    &description,
				Logo:           &logo,
				LastTradePrice: &price,
				State:          "open",
				Category:       "sports",
				Created:        &created,
				Event: &utilworm.EventMini{
					Title:       "Team A vs Team B",
					ConditionID: "event-1",
					Logo:        &eventLogo,
				},
				MarginEnabled: true,
			}},
			Meta: utilworm.EnvelopeMeta{Limit: 20, NextCursor: &nextCursor},
		},
	}

	resp, err := NewService(nil, client, utilworm.DefaultBaseURL).ListWormMarkets(context.Background(), &apiclient.ListWormMarketsRequest{})
	if err != nil {
		t.Fatalf("ListWormMarkets: %v", err)
	}
	if client.options.Limit != defaultWormMarketsLimit {
		t.Fatalf("limit = %d, want %d", client.options.Limit, defaultWormMarketsLimit)
	}
	if client.options.Cursor != "" {
		t.Fatalf("cursor = %q, want empty", client.options.Cursor)
	}
	if client.options.Category != wormMarketsCategory {
		t.Fatalf("category = %q, want %q", client.options.Category, wormMarketsCategory)
	}
	if client.options.Sort != wormMarketsSort {
		t.Fatalf("sort = %q, want %q", client.options.Sort, wormMarketsSort)
	}
	if resp.GetNextCursor() != nextCursor {
		t.Fatalf("next cursor = %q, want %q", resp.GetNextCursor(), nextCursor)
	}
	if len(resp.GetMarkets()) != 1 {
		t.Fatalf("markets len = %d, want 1", len(resp.GetMarkets()))
	}
	market := resp.GetMarkets()[0]
	if market.ConditionID != "market-1" || market.Title != "Will Team A beat Team B?" {
		t.Fatalf("market = %#v", market)
	}
	if market.Description != description || market.Logo != logo || market.LastTradePrice != price {
		t.Fatalf("market optional fields = %#v", market)
	}
	if market.EventTitle != "Team A vs Team B" || market.EventConditionID != "event-1" || market.EventLogo != eventLogo {
		t.Fatalf("event fields = %#v", market)
	}
	if market.Created != created || !market.MarginEnabled {
		t.Fatalf("created/margin = %#v", market)
	}
}

func TestListWormMarketsUsesRequestedPage(t *testing.T) {
	client := &fakeWormMarketClient{resp: &utilworm.ListMarketsResponse{}}

	_, err := NewService(nil, client, utilworm.DefaultBaseURL).ListWormMarkets(context.Background(), &apiclient.ListWormMarketsRequest{Limit: 50, Cursor: "cursor-1"})
	if err != nil {
		t.Fatalf("ListWormMarkets: %v", err)
	}
	if client.options.Limit != 50 {
		t.Fatalf("limit = %d, want 50", client.options.Limit)
	}
	if client.options.Cursor != "cursor-1" {
		t.Fatalf("cursor = %q, want cursor-1", client.options.Cursor)
	}
}

func TestListWormMarketsRejectsInvalidLimit(t *testing.T) {
	for _, limit := range []int32{-1, 101} {
		_, err := NewService(nil, &fakeWormMarketClient{}, utilworm.DefaultBaseURL).ListWormMarkets(context.Background(), &apiclient.ListWormMarketsRequest{Limit: limit})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("limit %d code = %s, want %s", limit, status.Code(err), codes.InvalidArgument)
		}
	}
}

func TestListWormMarketsNormalizesRelativeAssetURLs(t *testing.T) {
	logo := "/media/events/logos/market.webp"
	eventLogo := "/media/events/logos/event.webp"
	client := &fakeWormMarketClient{
		resp: &utilworm.ListMarketsResponse{
			Markets: []utilworm.MarketSummary{{
				ConditionID: "market-1",
				Title:       "Market",
				Logo:        &logo,
				Event: &utilworm.EventMini{
					Title:       "Event",
					ConditionID: "event-1",
					Logo:        &eventLogo,
				},
			}},
		},
	}

	resp, err := NewService(nil, client, "https://api.worm.wtf").ListWormMarkets(context.Background(), &apiclient.ListWormMarketsRequest{})
	if err != nil {
		t.Fatalf("ListWormMarkets: %v", err)
	}
	market := resp.GetMarkets()[0]
	if market.Logo != "https://api.worm.wtf/media/events/logos/market.webp" {
		t.Fatalf("logo = %q", market.Logo)
	}
	if market.EventLogo != "https://api.worm.wtf/media/events/logos/event.webp" {
		t.Fatalf("event logo = %q", market.EventLogo)
	}
}

func TestListWormMarketsPreservesAbsoluteAndEmptyAssetURLs(t *testing.T) {
	logo := "https://cdn.worm.wtf/m/1.png"
	client := &fakeWormMarketClient{
		resp: &utilworm.ListMarketsResponse{
			Markets: []utilworm.MarketSummary{{
				ConditionID: "market-1",
				Title:       "Market",
				Logo:        &logo,
				Event: &utilworm.EventMini{
					Title:       "Event",
					ConditionID: "event-1",
				},
			}},
		},
	}

	resp, err := NewService(nil, client, "https://api.worm.wtf").ListWormMarkets(context.Background(), &apiclient.ListWormMarketsRequest{})
	if err != nil {
		t.Fatalf("ListWormMarkets: %v", err)
	}
	market := resp.GetMarkets()[0]
	if market.Logo != logo {
		t.Fatalf("logo = %q, want %q", market.Logo, logo)
	}
	if market.EventLogo != "" {
		t.Fatalf("event logo = %q, want empty", market.EventLogo)
	}
}
