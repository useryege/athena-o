package worm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/worm/apiclient"
	wormstore "github.com/useryege/athena/internal/worm/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultWormMarketsLimit   = 20
	maxWormMarketsLimit       = 100
	wormMarketSyncInterval    = time.Minute
	wormLiveCheckBatchSize    = 20
	wormLiveCheckIdleDelay    = 30 * time.Second
	wormLiveRateLimitBuffer   = 500 * time.Millisecond
	wormLiveRateLimitFallback = 5 * time.Second
	wormLiveCandleWindow      = 30 * time.Minute
	wormLiveCandleInterval    = "5m"
	wormLivePriceThreshold    = 0.05

	defaultWormMarketsCategorySlug = "sports"
	defaultWormMarketsSortOption   = "leverage"
	defaultWormMarketsState        = "open"

	wormMarketLiveStateLive    = "live"
	wormMarketLiveStateNotLive = "not_live"
	wormMarketLiveStateUnknown = "unknown"
)

type wormMarketClient interface {
	ListMarkets(context.Context, utilworm.ListMarketsOptions) (*utilworm.ListMarketsResponse, error)
	GetMarketCandles(context.Context, string, utilworm.GetMarketCandlesOptions) (*utilworm.ListMarketCandlesResponse, error)
}

type Service struct {
	apiclient.UnimplementedWormServiceServer
	store        *wormstore.SQLStore
	wormClient   wormMarketClient
	assetBaseURL string
	syncCancel   context.CancelFunc
	syncWG       sync.WaitGroup
	syncMu       sync.Mutex
	startStopMu  sync.Mutex
	started      bool
}

type ServiceOption func(*Service)

func NewService(store *wormstore.SQLStore, wormClient wormMarketClient, assetBaseURL string, opts ...ServiceOption) *Service {
	assetBaseURL = strings.TrimSpace(assetBaseURL)
	if assetBaseURL == "" {
		assetBaseURL = utilworm.DefaultBaseURL
	}
	service := &Service{
		store:        store,
		wormClient:   wormClient,
		assetBaseURL: assetBaseURL,
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
	ctx, cancel := context.WithCancel(context.Background())
	s.syncCancel = cancel
	s.syncWG.Add(2)
	go s.syncLoop(ctx)
	go s.liveStateLoop(ctx)
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	cancel := s.syncCancel
	s.syncCancel = nil
	s.started = false
	s.startStopMu.Unlock()
	if cancel != nil {
		cancel()
		s.syncWG.Wait()
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
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "worm store is required")
	}

	params := wormMarketsListParams{
		Limit:        defaultWormMarketsLimit,
		SortOption:   defaultWormMarketsSortOption,
		CategorySlug: defaultWormMarketsCategorySlug,
		State:        defaultWormMarketsState,
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
	if err := validateFixedWormMarketParams(params); err != nil {
		return nil, err
	}
	offset, err := parseWormMarketCursor(params.Cursor)
	if err != nil {
		return nil, err
	}
	page := int32(offset/params.Limit) + 1
	result, err := s.store.ListWormMarketsPage(ctx, "", "", page, int32(params.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to list worm markets: %v", err)
	}
	resp := &apiclient.ListWormMarketsResponse{
		Markets: make([]*v1alpha1.WormMarketItem, 0, len(result.Items)),
	}
	var fetchedAt time.Time
	for _, item := range result.Items {
		if item.FetchedAt.After(fetchedAt) {
			fetchedAt = item.FetchedAt
		}
		resp.Markets = append(resp.Markets, s.wormMarketToAPIItem(item))
	}
	if !fetchedAt.IsZero() {
		resp.FetchedAt = fetchedAt.Unix()
	}
	nextOffset := offset + len(result.Items)
	if int64(nextOffset) < result.Total {
		resp.NextCursor = strconv.Itoa(nextOffset)
	}
	return resp, nil
}

type wormMarketsListParams struct {
	Limit        int
	Cursor       string
	SortOption   string
	CategorySlug string
	State        string
}

func validateFixedWormMarketParams(params wormMarketsListParams) error {
	sortOption := strings.ToLower(strings.TrimSpace(params.SortOption))
	if sortOption == "" {
		sortOption = defaultWormMarketsSortOption
	}
	categorySlug := strings.ToLower(strings.TrimSpace(params.CategorySlug))
	if categorySlug == "" {
		categorySlug = defaultWormMarketsCategorySlug
	}
	if sortOption != defaultWormMarketsSortOption {
		return status.Errorf(codes.InvalidArgument, "sort_option must be %s", defaultWormMarketsSortOption)
	}
	if categorySlug != defaultWormMarketsCategorySlug {
		return status.Errorf(codes.InvalidArgument, "category_slug must be %s", defaultWormMarketsCategorySlug)
	}
	return nil
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
		LiveState:      wormMarketLiveStateUnknown,
	}
	if market.Event != nil {
		item.EventTitle = market.Event.Title
		item.EventConditionID = market.Event.ConditionID
		item.EventLogo = s.normalizeAssetURL(stringValue(market.Event.Logo))
	}
	return item
}

func (s *Service) wormMarketToAPIItem(market wormstore.WormMarket) *v1alpha1.WormMarketItem {
	return &v1alpha1.WormMarketItem{
		ConditionID:      market.ConditionID,
		Title:            market.Title,
		Description:      market.Description,
		Logo:             market.Logo,
		LastTradePrice:   market.LastTradePrice,
		State:            market.State,
		Category:         market.Category,
		Created:          market.Created,
		EventTitle:       market.EventTitle,
		EventConditionID: market.EventConditionID,
		EventLogo:        market.EventLogo,
		MarginEnabled:    market.MarginEnabled,
		LiveState:        market.LiveState,
		LiveCheckedAt:    unixTime(market.LiveCheckedAt),
		LivePriceChange:  market.LivePriceChange,
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

func (s *Service) syncLoop(ctx context.Context) {
	defer s.syncWG.Done()
	s.syncWormMarkets(ctx)
	ticker := time.NewTicker(wormMarketSyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncWormMarkets(ctx)
		}
	}
}

func (s *Service) syncWormMarkets(ctx context.Context) {
	if err := s.syncWormMarketsOnce(ctx); err != nil {
		log.Warnf("failed to sync worm markets: %v", err)
	}
}

func (s *Service) syncWormMarketsOnce(ctx context.Context) error {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "worm store is required")
	}
	if s.wormClient == nil {
		return status.Error(codes.FailedPrecondition, "worm API client is required")
	}

	syncStartedAt := time.Now().UTC()
	cursor := ""
	seen := map[string]bool{}
	for {
		markets, err := s.wormClient.ListMarkets(ctx, utilworm.ListMarketsOptions{
			PageOptions: utilworm.PageOptions{
				Limit:  maxWormMarketsLimit,
				Cursor: cursor,
			},
			State:    defaultWormMarketsState,
			Category: defaultWormMarketsCategorySlug,
			Sort:     defaultWormMarketsSortOption,
		})
		if err != nil {
			return fmt.Errorf("list upstream worm markets: %w", err)
		}
		items := make([]wormstore.WormMarket, 0, len(markets.Markets))
		fetchedAt := time.Now().UTC()
		for i := range markets.Markets {
			market := markets.Markets[i]
			if !isSyncedWormMarket(market) || seen[market.ConditionID] {
				continue
			}
			seen[market.ConditionID] = true
			raw, err := json.Marshal(market)
			if err != nil {
				return fmt.Errorf("marshal worm market %s: %w", market.ConditionID, err)
			}
			item := s.toAPIMarketSummary(market)
			items = append(items, wormstore.WormMarket{
				ConditionID:      item.ConditionID,
				Title:            item.Title,
				Description:      item.Description,
				Logo:             item.Logo,
				LastTradePrice:   item.LastTradePrice,
				State:            defaultWormMarketsState,
				Category:         defaultWormMarketsCategorySlug,
				SortOption:       defaultWormMarketsSortOption,
				Created:          item.Created,
				EventTitle:       item.EventTitle,
				EventConditionID: item.EventConditionID,
				EventLogo:        item.EventLogo,
				MarginEnabled:    item.MarginEnabled,
				LiveState:        wormMarketLiveStateUnknown,
				Raw:              raw,
				FetchedAt:        fetchedAt,
				LastSeenAt:       syncStartedAt,
			})
		}
		if err := s.store.BatchUpsertWormMarkets(ctx, items); err != nil {
			return err
		}
		nextCursor := ""
		if markets.Meta.NextCursor != nil {
			nextCursor = strings.TrimSpace(*markets.Meta.NextCursor)
		}
		if nextCursor == "" || nextCursor == cursor {
			break
		}
		cursor = nextCursor
	}
	if _, err := s.store.DeleteWormMarketsNotSeenSince(ctx, syncStartedAt); err != nil {
		return err
	}
	log.Debugf("synced %d worm markets", len(seen))
	return nil
}

func (s *Service) liveStateLoop(ctx context.Context) {
	defer s.syncWG.Done()
	for {
		processed, err := s.updateWormMarketLiveStates(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if backoff, ok := wormLiveRateLimitBackoff(err); ok {
				log.Warnf("worm live state worker rate limited; pausing for %s: %v", backoff, err)
				if !waitWormLiveWorker(ctx, backoff) {
					return
				}
			} else {
				log.Warnf("failed to update worm market live states: %v", err)
			}
		}
		if processed == 0 {
			if !waitWormLiveWorker(ctx, wormLiveCheckIdleDelay) {
				return
			}
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (s *Service) updateWormMarketLiveStates(ctx context.Context) (int, error) {
	if s.store == nil {
		return 0, status.Error(codes.FailedPrecondition, "worm store is required")
	}
	if s.wormClient == nil {
		return 0, status.Error(codes.FailedPrecondition, "worm API client is required")
	}
	markets, err := s.store.ListWormMarketsPendingLiveCheck(ctx, wormLiveCheckBatchSize)
	if err != nil {
		return 0, err
	}
	for i := range markets {
		if ctx.Err() != nil {
			return i, ctx.Err()
		}
		market := markets[i]
		liveState, liveCheckedAt, livePriceChange, err := s.detectWormMarketLiveState(ctx, market)
		market.LiveState = liveState
		market.LiveCheckedAt = liveCheckedAt
		market.LivePriceChange = livePriceChange
		if _, err := s.store.UpdateWormMarketLiveState(ctx, market); err != nil {
			return i, err
		}
		if err != nil {
			return i + 1, err
		}
	}
	return len(markets), nil
}

func (s *Service) detectWormMarketLiveState(ctx context.Context, market wormstore.WormMarket) (string, time.Time, string, error) {
	if market.LiveState == wormMarketLiveStateLive {
		return market.LiveState, market.LiveCheckedAt, market.LivePriceChange, nil
	}

	checkedAt := time.Now().UTC()
	isYes := true
	candles, err := s.wormClient.GetMarketCandles(ctx, market.ConditionID, utilworm.GetMarketCandlesOptions{
		StartTime: checkedAt.Add(-wormLiveCandleWindow).Unix(),
		EndTime:   checkedAt.Unix(),
		Interval:  wormLiveCandleInterval,
		IsYes:     &isYes,
	})
	if err != nil {
		log.Warnf("failed to get worm market candles for %s: %v", market.ConditionID, err)
		return wormMarketLiveStateUnknown, checkedAt, "", err
	}

	priceChange, ok := wormMarketCandlePriceChange(candles.Candles)
	if !ok {
		return wormMarketLiveStateNotLive, checkedAt, "", nil
	}
	priceChangeText := strconv.FormatFloat(priceChange, 'f', 6, 64)
	if priceChange >= wormLivePriceThreshold {
		return wormMarketLiveStateLive, checkedAt, priceChangeText, nil
	}
	return wormMarketLiveStateNotLive, checkedAt, priceChangeText, nil
}

func waitWormLiveWorker(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func wormLiveRateLimitBackoff(err error) (time.Duration, bool) {
	var wormErr *utilworm.Error
	if !errors.As(err, &wormErr) || wormErr == nil {
		return 0, false
	}
	if wormErr.StatusCode != http.StatusTooManyRequests && !strings.EqualFold(wormErr.Slug, "throttled") {
		return 0, false
	}
	backoff := wormErr.RetryAfter
	if backoff <= 0 && wormErr.RateLimitReset > 0 {
		backoff = time.Until(time.Unix(wormErr.RateLimitReset, 0))
	}
	if backoff <= 0 {
		backoff = wormLiveRateLimitFallback
	}
	return backoff + wormLiveRateLimitBuffer, true
}

func wormMarketCandlePriceChange(candles []utilworm.MarketCandle) (float64, bool) {
	var maxHigh float64
	var minLow float64
	found := false
	for _, candle := range candles {
		high, err := strconv.ParseFloat(strings.TrimSpace(candle.High), 64)
		if err != nil {
			continue
		}
		low, err := strconv.ParseFloat(strings.TrimSpace(candle.Low), 64)
		if err != nil {
			continue
		}
		if !found || high > maxHigh {
			maxHigh = high
		}
		if !found || low < minLow {
			minLow = low
		}
		found = true
	}
	if !found || maxHigh < minLow {
		return 0, false
	}
	return maxHigh - minLow, true
}

func isSyncedWormMarket(market utilworm.MarketSummary) bool {
	return strings.TrimSpace(market.ConditionID) != "" &&
		strings.EqualFold(strings.TrimSpace(market.State), defaultWormMarketsState) &&
		strings.EqualFold(strings.TrimSpace(market.Category), defaultWormMarketsCategorySlug)
}

func parseWormMarketCursor(cursor string) (int, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(cursor)
	if err != nil || offset < 0 {
		return 0, status.Error(codes.InvalidArgument, "cursor must be a nonnegative offset")
	}
	return offset, nil
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

func unixTime(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.Unix()
}
