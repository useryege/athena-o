package worm

import (
	"context"
	"errors"
	"testing"

	"github.com/useryege/athena/internal/worm/apiclient"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeWormMarketClient struct {
	options          utilworm.ListMarketsOptions
	resp             *utilworm.ListMarketsResponse
	err              error
	conditionID      string
	marketResp       *utilworm.Market
	marketStatsResp  *utilworm.MarketStats
	priceOptions     []utilworm.GetMarketPriceOptions
	priceResp        []*utilworm.MarketPrice
	orderBookOptions []utilworm.GetMarketOrderBookOptions
	orderBookResp    []*utilworm.MarketOrderBook
	detailErr        error
}

func (f *fakeWormMarketClient) ListMarkets(_ context.Context, options utilworm.ListMarketsOptions) (*utilworm.ListMarketsResponse, error) {
	f.options = options
	return f.resp, f.err
}

func (f *fakeWormMarketClient) GetMarket(_ context.Context, conditionID string) (*utilworm.Market, error) {
	f.conditionID = conditionID
	if f.detailErr != nil {
		return nil, f.detailErr
	}
	return f.marketResp, nil
}

func (f *fakeWormMarketClient) GetMarketStats(_ context.Context, conditionID string) (*utilworm.MarketStats, error) {
	f.conditionID = conditionID
	if f.detailErr != nil {
		return nil, f.detailErr
	}
	return f.marketStatsResp, nil
}

func (f *fakeWormMarketClient) GetMarketPrice(_ context.Context, conditionID string, options utilworm.GetMarketPriceOptions) (*utilworm.MarketPrice, error) {
	f.conditionID = conditionID
	f.priceOptions = append(f.priceOptions, options)
	if f.detailErr != nil {
		return nil, f.detailErr
	}
	price := f.priceResp[0]
	f.priceResp = f.priceResp[1:]
	return price, nil
}

func (f *fakeWormMarketClient) GetMarketOrderBook(_ context.Context, conditionID string, options utilworm.GetMarketOrderBookOptions) (*utilworm.MarketOrderBook, error) {
	f.conditionID = conditionID
	f.orderBookOptions = append(f.orderBookOptions, options)
	if f.detailErr != nil {
		return nil, f.detailErr
	}
	book := f.orderBookResp[0]
	f.orderBookResp = f.orderBookResp[1:]
	return book, nil
}

func TestListWormMarketsUsesNewAllDefaults(t *testing.T) {
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
	if client.options.Category != "" {
		t.Fatalf("category = %q, want empty", client.options.Category)
	}
	if client.options.Sort != "" {
		t.Fatalf("sort = %q, want empty", client.options.Sort)
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

func TestListWormMarketsUsesRequestedBrowseFilters(t *testing.T) {
	client := &fakeWormMarketClient{resp: &utilworm.ListMarketsResponse{}}

	_, err := NewService(nil, client, utilworm.DefaultBaseURL).ListWormMarkets(context.Background(), &apiclient.ListWormMarketsRequest{
		SortOption:   "leverage",
		CategorySlug: "sports",
	})
	if err != nil {
		t.Fatalf("ListWormMarkets: %v", err)
	}
	if client.options.Sort != "leverage" {
		t.Fatalf("sort = %q, want leverage", client.options.Sort)
	}
	if client.options.Category != "sports" {
		t.Fatalf("category = %q, want sports", client.options.Category)
	}
}

func TestListWormMarketsMapsEndingSoonCryptoFilters(t *testing.T) {
	client := &fakeWormMarketClient{resp: &utilworm.ListMarketsResponse{}}

	_, err := NewService(nil, client, utilworm.DefaultBaseURL).ListWormMarkets(context.Background(), &apiclient.ListWormMarketsRequest{
		SortOption:   "ending_soon",
		CategorySlug: "crypto",
	})
	if err != nil {
		t.Fatalf("ListWormMarkets: %v", err)
	}
	if client.options.Sort != "ending_soon" {
		t.Fatalf("sort = %q, want ending_soon", client.options.Sort)
	}
	if client.options.Category != "crypto" {
		t.Fatalf("category = %q, want crypto", client.options.Category)
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

func TestListWormMarketsRejectsInvalidBrowseFilters(t *testing.T) {
	tests := []apiclient.ListWormMarketsRequest{
		{SortOption: "oldest"},
		{CategorySlug: "gaming"},
	}
	for i := range tests {
		_, err := NewService(nil, &fakeWormMarketClient{}, utilworm.DefaultBaseURL).ListWormMarkets(context.Background(), &tests[i])
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("case %d code = %s, want %s", i, status.Code(err), codes.InvalidArgument)
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

func TestGetWormMarketMapsDetail(t *testing.T) {
	description := "market description"
	logo := "/media/events/logos/market.webp"
	price := "0.68"
	created := int64(1714300100)
	eventLogo := "/media/events/logos/event.webp"
	yesLabel := "Team A"
	noLabel := "Team B"
	resolutionDate := int64(1714386500)
	makerFee := "0.01"
	takerFee := "0.02"
	maxLeverage := "5"
	orderMinSize := "10"
	priceDecimals := 3
	yesPrice := "0.71"
	noPrice := "0.29"
	client := &fakeWormMarketClient{
		marketResp: &utilworm.Market{
			MarketSummary: utilworm.MarketSummary{
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
			},
			YesOutcomeLabel: &yesLabel,
			NoOutcomeLabel:  &noLabel,
			Outcomes: []utilworm.Outcome{
				{IsYes: true, Text: "Yes"},
				{IsYes: false, Text: "No"},
			},
			Rules:          []string{"Rule 1", "Rule 2"},
			ResolutionDate: &resolutionDate,
			MakerFee:       &makerFee,
			TakerFee:       &takerFee,
			Config: &utilworm.MarketConfig{
				Kind:          "binary",
				MaxLeverage:   &maxLeverage,
				OrderMinSize:  &orderMinSize,
				PriceDecimals: &priceDecimals,
			},
		},
		marketStatsResp: &utilworm.MarketStats{
			TotalVolume:    "1000",
			TotalVolume24H: "250",
			MarketCap:      "5000",
			TradeCount:     42,
		},
		priceResp: []*utilworm.MarketPrice{
			{ConditionID: "market-1", Price: &yesPrice, PriceKind: "last", IsYes: true},
			{ConditionID: "market-1", Price: &noPrice, PriceKind: "last", IsYes: false},
		},
		orderBookResp: []*utilworm.MarketOrderBook{
			{
				Market: "market-1",
				IsYes:  true,
				Bid:    []utilworm.OrderBookLevel{{Price: "0.70", TotalAmount: "12"}},
				Ask:    []utilworm.OrderBookLevel{{Price: "0.72", TotalAmount: "8"}},
			},
			{
				Market: "market-1",
				IsYes:  false,
				Bid:    []utilworm.OrderBookLevel{{Price: "0.28", TotalAmount: "9"}},
				Ask:    []utilworm.OrderBookLevel{{Price: "0.30", TotalAmount: "11"}},
			},
		},
	}

	resp, err := NewService(nil, client, "https://api.worm.wtf").GetWormMarket(context.Background(), &apiclient.GetWormMarketRequest{ConditionId: " market-1 "})
	if err != nil {
		t.Fatalf("GetWormMarket: %v", err)
	}
	if client.conditionID != "market-1" {
		t.Fatalf("condition id = %q, want market-1", client.conditionID)
	}
	if len(client.priceOptions) != 2 || client.priceOptions[0].IsYes == nil || !*client.priceOptions[0].IsYes || client.priceOptions[1].IsYes == nil || *client.priceOptions[1].IsYes {
		t.Fatalf("price options = %#v", client.priceOptions)
	}
	if len(client.orderBookOptions) != 2 || client.orderBookOptions[0].Depth != 5 || client.orderBookOptions[1].Depth != 5 {
		t.Fatalf("order book options = %#v", client.orderBookOptions)
	}

	market := resp.GetMarket()
	if market.Market.Logo != "https://api.worm.wtf/media/events/logos/market.webp" || market.Market.EventLogo != "https://api.worm.wtf/media/events/logos/event.webp" {
		t.Fatalf("logos = %#v", market.Market)
	}
	if market.YesOutcomeLabel != yesLabel || market.NoOutcomeLabel != noLabel || market.ResolutionDate != resolutionDate {
		t.Fatalf("detail labels/date = %#v", market)
	}
	if len(market.Outcomes) != 2 || !market.Outcomes[0].IsYes || market.Outcomes[1].Text != "No" {
		t.Fatalf("outcomes = %#v", market.Outcomes)
	}
	if len(market.Rules) != 2 || market.Rules[1] != "Rule 2" {
		t.Fatalf("rules = %#v", market.Rules)
	}
	if market.MakerFee != makerFee || market.TakerFee != takerFee {
		t.Fatalf("fees = %#v", market)
	}
	if market.Config.Kind != "binary" || market.Config.MaxLeverage != maxLeverage || market.Config.PriceDecimals != int32(priceDecimals) {
		t.Fatalf("config = %#v", market.Config)
	}
	if market.Stats.TotalVolume24H != "250" || market.Stats.TradeCount != 42 {
		t.Fatalf("stats = %#v", market.Stats)
	}
	if len(market.Prices) != 2 || market.Prices[0].Price != yesPrice || market.Prices[1].Price != noPrice {
		t.Fatalf("prices = %#v", market.Prices)
	}
	if len(market.OrderBooks) != 2 || market.OrderBooks[0].Bid[0].Price != "0.70" || market.OrderBooks[1].Ask[0].TotalAmount != "11" {
		t.Fatalf("order books = %#v", market.OrderBooks)
	}
}

func TestGetWormMarketRejectsEmptyConditionID(t *testing.T) {
	_, err := NewService(nil, &fakeWormMarketClient{}, utilworm.DefaultBaseURL).GetWormMarket(context.Background(), &apiclient.GetWormMarketRequest{ConditionId: "  "})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}
}

func TestGetWormMarketPropagatesClientError(t *testing.T) {
	wantErr := errors.New("worm detail failed")
	_, err := NewService(nil, &fakeWormMarketClient{detailErr: wantErr}, utilworm.DefaultBaseURL).GetWormMarket(context.Background(), &apiclient.GetWormMarketRequest{ConditionId: "market-1"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}
