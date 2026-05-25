package worm

import (
	"context"
	"net/url"
	"strings"
	"sync"

	"github.com/useryege/athena/internal/worm/apiclient"
	wormstore "github.com/useryege/athena/internal/worm/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultWormMarketsLimit = 20
	maxWormMarketsLimit     = 100

	wormMarketsCategory = "sports"
	wormMarketsSort     = "leverage"
)

type wormMarketClient interface {
	ListMarkets(context.Context, utilworm.ListMarketsOptions) (*utilworm.ListMarketsResponse, error)
	GetMarket(context.Context, string) (*utilworm.Market, error)
	GetMarketStats(context.Context, string) (*utilworm.MarketStats, error)
	GetMarketPrice(context.Context, string, utilworm.GetMarketPriceOptions) (*utilworm.MarketPrice, error)
	GetMarketOrderBook(context.Context, string, utilworm.GetMarketOrderBookOptions) (*utilworm.MarketOrderBook, error)
}

type Service struct {
	apiclient.UnimplementedWormServiceServer
	store        *wormstore.SQLStore
	wormClient   wormMarketClient
	assetBaseURL string
	startStopMu  sync.Mutex
	started      bool
}

func NewService(store *wormstore.SQLStore, wormClient wormMarketClient, assetBaseURL string) *Service {
	assetBaseURL = strings.TrimSpace(assetBaseURL)
	if assetBaseURL == "" {
		assetBaseURL = utilworm.DefaultBaseURL
	}
	return &Service{store: store, wormClient: wormClient, assetBaseURL: assetBaseURL}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}
	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "worm store is required")
	}
	if s.wormClient == nil {
		return status.Error(codes.FailedPrecondition, "worm API client is required")
	}
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	s.started = false
	return nil
}

func (s *Service) GetWormStatus(context.Context, *apiclient.GetWormStatusRequest) (*apiclient.GetWormStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetWormStatusResponse{
		Started: started,
		Status:  statusText,
	}, nil
}

func (s *Service) ListWormMarkets(ctx context.Context, req *apiclient.ListWormMarketsRequest) (*apiclient.ListWormMarketsResponse, error) {
	if s.wormClient == nil {
		return nil, status.Error(codes.FailedPrecondition, "worm API client is required")
	}

	limit := defaultWormMarketsLimit
	cursor := ""
	if req != nil {
		if req.GetLimit() != 0 {
			limit = int(req.GetLimit())
		}
		cursor = req.GetCursor()
	}
	if limit < 1 || limit > maxWormMarketsLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between 1 and %d", maxWormMarketsLimit)
	}

	markets, err := s.wormClient.ListMarkets(ctx, utilworm.ListMarketsOptions{
		PageOptions: utilworm.PageOptions{
			Limit:  limit,
			Cursor: cursor,
		},
		Category: wormMarketsCategory,
		Sort:     wormMarketsSort,
	})
	if err != nil {
		return nil, err
	}

	resp := &apiclient.ListWormMarketsResponse{}
	if markets.Meta.NextCursor != nil {
		resp.NextCursor = *markets.Meta.NextCursor
	}
	resp.Markets = make([]*v1alpha1.WormMarketItem, 0, len(markets.Markets))
	for i := range markets.Markets {
		resp.Markets = append(resp.Markets, s.toAPIMarketSummary(markets.Markets[i]))
	}
	return resp, nil
}

func (s *Service) GetWormMarket(ctx context.Context, req *apiclient.GetWormMarketRequest) (*apiclient.GetWormMarketResponse, error) {
	if s.wormClient == nil {
		return nil, status.Error(codes.FailedPrecondition, "worm API client is required")
	}

	conditionID := ""
	if req != nil {
		conditionID = strings.TrimSpace(req.GetConditionId())
	}
	if conditionID == "" {
		return nil, status.Error(codes.InvalidArgument, "condition_id is required")
	}

	market, err := s.wormClient.GetMarket(ctx, conditionID)
	if err != nil {
		return nil, err
	}
	if market == nil {
		return nil, status.Error(codes.NotFound, "worm market not found")
	}

	stats, err := s.wormClient.GetMarketStats(ctx, conditionID)
	if err != nil {
		return nil, err
	}

	yes := true
	no := false
	yesPrice, err := s.wormClient.GetMarketPrice(ctx, conditionID, utilworm.GetMarketPriceOptions{IsYes: &yes})
	if err != nil {
		return nil, err
	}
	noPrice, err := s.wormClient.GetMarketPrice(ctx, conditionID, utilworm.GetMarketPriceOptions{IsYes: &no})
	if err != nil {
		return nil, err
	}

	yesBook, err := s.wormClient.GetMarketOrderBook(ctx, conditionID, utilworm.GetMarketOrderBookOptions{Depth: 5, IsYes: &yes})
	if err != nil {
		return nil, err
	}
	noBook, err := s.wormClient.GetMarketOrderBook(ctx, conditionID, utilworm.GetMarketOrderBookOptions{Depth: 5, IsYes: &no})
	if err != nil {
		return nil, err
	}

	return &apiclient.GetWormMarketResponse{
		Market: s.toAPIMarketDetail(market, stats, []*utilworm.MarketPrice{yesPrice, noPrice}, []*utilworm.MarketOrderBook{yesBook, noBook}),
	}, nil
}

func (s *Service) toAPIMarketSummary(market utilworm.MarketSummary) *v1alpha1.WormMarketItem {
	item := &v1alpha1.WormMarketItem{
		ConditionID:    market.ConditionID,
		Title:          market.Title,
		Description:    stringValue(market.Description),
		Logo:           s.normalizeAssetURL(stringValue(market.Logo)),
		LastTradePrice: stringValue(market.LastTradePrice),
		State:          market.State,
		Category:       market.Category,
		Created:        int64Value(market.Created),
		MarginEnabled:  market.MarginEnabled,
	}
	if market.Event != nil {
		item.EventTitle = market.Event.Title
		item.EventConditionID = market.Event.ConditionID
		item.EventLogo = s.normalizeAssetURL(stringValue(market.Event.Logo))
	}
	return item
}

func (s *Service) toAPIMarketDetail(
	market *utilworm.Market,
	stats *utilworm.MarketStats,
	prices []*utilworm.MarketPrice,
	orderBooks []*utilworm.MarketOrderBook,
) *v1alpha1.WormMarketDetail {
	item := s.toAPIMarketSummary(market.MarketSummary)
	detail := &v1alpha1.WormMarketDetail{
		Market:          *item,
		YesOutcomeLabel: stringValue(market.YesOutcomeLabel),
		NoOutcomeLabel:  stringValue(market.NoOutcomeLabel),
		Rules:           append([]string(nil), market.Rules...),
		ResolutionDate:  int64Value(market.ResolutionDate),
		MakerFee:        stringValue(market.MakerFee),
		TakerFee:        stringValue(market.TakerFee),
		Config:          toAPIMarketConfig(market.Config),
		Stats:           toAPIMarketStats(stats),
		Outcomes:        make([]v1alpha1.WormMarketOutcome, 0, len(market.Outcomes)),
		Prices:          make([]v1alpha1.WormMarketPrice, 0, len(prices)),
		OrderBooks:      make([]v1alpha1.WormMarketOrderBook, 0, len(orderBooks)),
	}
	for i := range market.Outcomes {
		detail.Outcomes = append(detail.Outcomes, v1alpha1.WormMarketOutcome{
			IsYes: market.Outcomes[i].IsYes,
			Text:  market.Outcomes[i].Text,
		})
	}
	for _, price := range prices {
		if price == nil {
			continue
		}
		detail.Prices = append(detail.Prices, v1alpha1.WormMarketPrice{
			ConditionID: price.ConditionID,
			Price:       stringValue(price.Price),
			PriceKind:   price.PriceKind,
			IsYes:       price.IsYes,
		})
	}
	for _, book := range orderBooks {
		if book == nil {
			continue
		}
		detail.OrderBooks = append(detail.OrderBooks, toAPIOrderBook(book))
	}
	return detail
}

func toAPIMarketStats(stats *utilworm.MarketStats) v1alpha1.WormMarketStats {
	if stats == nil {
		return v1alpha1.WormMarketStats{}
	}
	return v1alpha1.WormMarketStats{
		TotalVolume:    stats.TotalVolume,
		TotalVolume24H: stats.TotalVolume24H,
		MarketCap:      stats.MarketCap,
		TradeCount:     stats.TradeCount,
	}
}

func toAPIMarketConfig(config *utilworm.MarketConfig) v1alpha1.WormMarketConfig {
	if config == nil {
		return v1alpha1.WormMarketConfig{}
	}
	return v1alpha1.WormMarketConfig{
		Kind:                config.Kind,
		MaxLeverage:         stringValue(config.MaxLeverage),
		OpeningFee:          stringValue(config.OpeningFee),
		ClosingFee:          stringValue(config.ClosingFee),
		AnnualFeeRate:       stringValue(config.AnnualFeeRate),
		OrderMinSize:        stringValue(config.OrderMinSize),
		PriceDecimals:       int32Value(config.PriceDecimals),
		SharesDecimals:      int32Value(config.SharesDecimals),
		MinPrice:            stringValue(config.MinPrice),
		MaxPrice:            stringValue(config.MaxPrice),
		MinAmount:           stringValue(config.MinAmount),
		MaxAmount:           stringValue(config.MaxAmount),
		MinFunds:            stringValue(config.MinFunds),
		MaxFunds:            stringValue(config.MaxFunds),
		PricePrecision:      int32Value(config.PricePrecision),
		AmountPrecision:     int32Value(config.AmountPrecision),
		FundsPrecision:      int32Value(config.FundsPrecision),
		MakerFeeRate:        stringValue(config.MakerFeeRate),
		TakerFeeRate:        stringValue(config.TakerFeeRate),
		DefaultSlippageRate: stringValue(config.DefaultSlippageRate),
	}
}

func toAPIOrderBook(book *utilworm.MarketOrderBook) v1alpha1.WormMarketOrderBook {
	apiBook := v1alpha1.WormMarketOrderBook{
		Market: book.Market,
		IsYes:  book.IsYes,
		Bid:    make([]v1alpha1.WormOrderBookLevel, 0, len(book.Bid)),
		Ask:    make([]v1alpha1.WormOrderBookLevel, 0, len(book.Ask)),
	}
	for i := range book.Bid {
		apiBook.Bid = append(apiBook.Bid, toAPIOrderBookLevel(book.Bid[i]))
	}
	for i := range book.Ask {
		apiBook.Ask = append(apiBook.Ask, toAPIOrderBookLevel(book.Ask[i]))
	}
	return apiBook
}

func toAPIOrderBookLevel(level utilworm.OrderBookLevel) v1alpha1.WormOrderBookLevel {
	return v1alpha1.WormOrderBookLevel{
		Price:       level.Price,
		TotalAmount: level.TotalAmount,
	}
}

func (s *Service) normalizeAssetURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	assetURL, err := url.Parse(value)
	if err != nil || assetURL.IsAbs() {
		return value
	}
	baseURL, err := url.Parse(s.assetBaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return value
	}
	return baseURL.ResolveReference(assetURL).String()
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func int32Value(value *int) int32 {
	if value == nil {
		return 0
	}
	return int32(*value)
}
