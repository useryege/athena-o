package worm

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/useryege/athena/internal/worm/apiclient"
	wormstore "github.com/useryege/athena/internal/worm/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilworm "github.com/useryege/athena/util/worm"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultWormMarketsLimit = 20
	maxWormMarketsLimit     = 100

	defaultWormMarketsCategorySlug = "sports"
	defaultWormMarketsSortOption   = "leverage"
)

type wormMarketClient interface {
	ListMarkets(context.Context, utilworm.ListMarketsOptions) (*utilworm.ListMarketsResponse, error)
	GetMarket(context.Context, string) (*utilworm.Market, error)
}

type Service struct {
	apiclient.UnimplementedWormServiceServer
	store         *wormstore.SQLStore
	wormClient    wormMarketClient
	assetBaseURL  string
	redisClient   *redis.Client
	cacheConfig   CacheConfig
	refreshCh     chan refreshRequest
	refreshGroup  singleflight.Group
	refreshCancel context.CancelFunc
	refreshWG     sync.WaitGroup
	startStopMu   sync.Mutex
	started       bool
}

type ServiceOption func(*Service)

func WithRedisClient(client *redis.Client) ServiceOption {
	return func(s *Service) {
		s.redisClient = client
	}
}

func WithCacheConfig(config CacheConfig) ServiceOption {
	return func(s *Service) {
		s.cacheConfig = config.withDefaults()
	}
}

func NewService(store *wormstore.SQLStore, wormClient wormMarketClient, assetBaseURL string, opts ...ServiceOption) *Service {
	assetBaseURL = strings.TrimSpace(assetBaseURL)
	if assetBaseURL == "" {
		assetBaseURL = utilworm.DefaultBaseURL
	}
	service := &Service{
		store:        store,
		wormClient:   wormClient,
		assetBaseURL: assetBaseURL,
		cacheConfig:  DefaultCacheConfig(),
		refreshCh:    make(chan refreshRequest, 256),
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
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
	if s.redisClient != nil && s.cacheConfig.RefreshWorkers > 0 {
		ctx, cancel := context.WithCancel(context.Background())
		s.refreshCancel = cancel
		for i := 0; i < s.cacheConfig.RefreshWorkers; i++ {
			s.refreshWG.Add(1)
			go s.refreshWorker(ctx)
		}
		s.refreshWG.Add(1)
		go s.warmupLoop(ctx)
	}
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	cancel := s.refreshCancel
	s.refreshCancel = nil
	s.started = false
	s.startStopMu.Unlock()
	if cancel != nil {
		cancel()
		s.refreshWG.Wait()
	}
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

	params := listWormMarketsParams{
		Limit:        defaultWormMarketsLimit,
		SortOption:   defaultWormMarketsSortOption,
		CategorySlug: defaultWormMarketsCategorySlug,
	}
	if req != nil {
		if req.GetLimit() != 0 {
			params.Limit = int(req.GetLimit())
		}
		params.Cursor = req.GetCursor()
		params.SortOption = strings.TrimSpace(req.GetSortOption())
		params.CategorySlug = strings.TrimSpace(req.GetCategorySlug())
	}
	if params.Limit < 1 || params.Limit > maxWormMarketsLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between 1 and %d", maxWormMarketsLimit)
	}
	sortOption, err := normalizeWormMarketSortOption(params.SortOption)
	if err != nil {
		return nil, err
	}
	categorySlug, err := normalizeWormMarketCategorySlug(params.CategorySlug)
	if err != nil {
		return nil, err
	}
	params.SortOption = sortOption
	params.CategorySlug = categorySlug

	if s.redisClient != nil {
		return s.listWormMarketsCached(ctx, params)
	}
	return s.fetchWormMarkets(ctx, params)
}

func (s *Service) fetchWormMarkets(ctx context.Context, params listWormMarketsParams) (*apiclient.ListWormMarketsResponse, error) {
	markets, err := s.wormClient.ListMarkets(ctx, utilworm.ListMarketsOptions{
		PageOptions: utilworm.PageOptions{
			Limit:  params.Limit,
			Cursor: params.Cursor,
		},
		Category: upstreamWormMarketCategory(params.CategorySlug),
		Sort:     upstreamWormMarketSort(params.SortOption),
	})
	if err != nil {
		return nil, err
	}

	resp := &apiclient.ListWormMarketsResponse{FetchedAt: time.Now().Unix()}
	if markets.Meta.NextCursor != nil {
		resp.NextCursor = *markets.Meta.NextCursor
	}
	resp.Markets = make([]*v1alpha1.WormMarketItem, 0, len(markets.Markets))
	for i := range markets.Markets {
		resp.Markets = append(resp.Markets, s.toAPIMarketSummary(markets.Markets[i]))
	}
	return resp, nil
}

func normalizeWormMarketSortOption(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return defaultWormMarketsSortOption, nil
	}
	switch value {
	case "new", "trending", "ending_soon", "leverage":
		return value, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "sort_option must be one of new, trending, ending_soon, leverage")
	}
}

func normalizeWormMarketCategorySlug(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return defaultWormMarketsCategorySlug, nil
	}
	switch value {
	case "all", "politics", "sports", "crypto", "tech", "finance", "wtf":
		return value, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "category_slug must be one of all, politics, sports, crypto, tech, finance, wtf")
	}
}

func upstreamWormMarketSort(value string) string {
	if value == "new" {
		return ""
	}
	return value
}

func upstreamWormMarketCategory(value string) string {
	if value == "all" {
		return ""
	}
	return value
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
	if s.redisClient != nil {
		return s.getWormMarketCached(ctx, conditionID)
	}
	return s.fetchWormMarket(ctx, conditionID)
}

func (s *Service) fetchWormMarket(ctx context.Context, conditionID string) (*apiclient.GetWormMarketResponse, error) {
	market, err := s.wormClient.GetMarket(ctx, conditionID)
	if err != nil {
		return nil, err
	}
	if market == nil {
		return nil, status.Error(codes.NotFound, "worm market not found")
	}

	return &apiclient.GetWormMarketResponse{
		Market:    s.toAPIMarketDetail(market),
		FetchedAt: time.Now().Unix(),
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

func (s *Service) toAPIMarketDetail(market *utilworm.Market) *v1alpha1.WormMarketDetail {
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
		Outcomes:        make([]v1alpha1.WormMarketOutcome, 0, len(market.Outcomes)),
	}
	for i := range market.Outcomes {
		detail.Outcomes = append(detail.Outcomes, v1alpha1.WormMarketOutcome{
			IsYes: market.Outcomes[i].IsYes,
			Text:  market.Outcomes[i].Text,
		})
	}
	return detail
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
