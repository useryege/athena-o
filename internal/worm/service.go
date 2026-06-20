package worm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/internal/worm/apiclient"
	wormstore "github.com/useryege/athena/internal/worm/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultWormEventsLimit    = 20
	maxWormEventsLimit        = 100
	maxWormMarketsSyncLimit   = 100
	wormMarketSyncInterval    = time.Minute
	wormMarketRulesInterval   = time.Minute
	wormLiveStateLoopInterval = time.Minute
	wormLivePriceWindow       = 30 * time.Minute

	defaultWormMarketsCategorySlug = "sports"
	defaultWormMarketsSortOption   = "leverage"
	defaultWormMarketsState        = "open"

	wormMarketLiveStateLive    = "live"
	wormMarketLiveStateNotLive = "not_live"
	wormMarketLiveStateUnknown = "unknown"
)

type wormMarketClient interface {
	ListMarkets(context.Context, utilworm.ListMarketsOptions) (*utilworm.ListMarketsResponse, error)
	GetMarket(context.Context, string) (*utilworm.Market, error)
	GetEvent(context.Context, string) (*utilworm.Event, error)
}

type Service struct {
	apiclient.UnimplementedWormServiceServer
	store                 *wormstore.SQLStore
	wormClient            wormMarketClient
	notificationClientset notificationapiclient.Clientset
	assetBaseURL          string
	syncCancel            context.CancelFunc
	syncWG                sync.WaitGroup
	syncMu                sync.Mutex
	startStopMu           sync.Mutex
	started               bool
}

type ServiceOption func(*Service)

func WithNotificationClientset(clientset notificationapiclient.Clientset) ServiceOption {
	return func(s *Service) {
		s.notificationClientset = clientset
	}
}

func NewService(
	store *wormstore.SQLStore,
	wormClient wormMarketClient,
	assetBaseURL string,
	opts ...ServiceOption,
) *Service {
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
	s.syncWG.Add(3)
	go s.syncLoop(ctx)
	go s.rulesLoop(ctx)
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

func (s *Service) GetWormEvent(ctx context.Context, req *apiclient.GetWormEventRequest) (*apiclient.GetWormEventResponse, error) {
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
	event, err := s.wormClient.GetEvent(ctx, conditionID)
	if err != nil {
		if isWormStatusCode(err, http.StatusNotFound) {
			return nil, status.Errorf(codes.NotFound, "worm event %q was not found", conditionID)
		}
		return nil, status.Errorf(codes.Unavailable, "failed to get worm event: %v", err)
	}
	if event == nil || strings.TrimSpace(event.ConditionID) == "" {
		return nil, status.Errorf(codes.NotFound, "worm event %q was not found", conditionID)
	}
	return &apiclient.GetWormEventResponse{
		Event:     s.toAPIEvent(event),
		FetchedAt: time.Now().Unix(),
	}, nil
}

func (s *Service) ListWormEvents(ctx context.Context, req *apiclient.ListWormEventsRequest) (*apiclient.ListWormEventsResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "worm store is required")
	}

	params := wormEventsListParams{
		Limit:        defaultWormEventsLimit,
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
	if params.Limit < 1 || params.Limit > maxWormEventsLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between 1 and %d", maxWormEventsLimit)
	}
	if err := validateFixedWormEventParams(params); err != nil {
		return nil, err
	}
	offset, err := parseWormMarketCursor(params.Cursor)
	if err != nil {
		return nil, err
	}
	page := int32(offset/params.Limit) + 1
	result, err := s.store.ListWormEventsPage(ctx, page, int32(params.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to list worm events: %v", err)
	}
	resp := &apiclient.ListWormEventsResponse{
		Events: make([]*v1alpha1.WormEventItem, 0, len(result.Items)),
	}
	var fetchedAt time.Time
	for _, item := range result.Items {
		if item.FetchedAt.After(fetchedAt) {
			fetchedAt = item.FetchedAt
		}
		resp.Events = append(resp.Events, s.wormEventToAPIItem(item))
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

func (s *Service) wormEventToAPIItem(event wormstore.WormEvent) *v1alpha1.WormEventItem {
	item := &v1alpha1.WormEventItem{
		ConditionID: event.ConditionID,
		Title:       event.Title,
		Logo:        event.Logo,
		Live:        event.Live,
		MarketCount: event.MarketCount,
		Markets:     make([]*v1alpha1.WormMarketItem, 0, len(event.Markets)),
	}
	for _, market := range event.Markets {
		item.Markets = append(item.Markets, s.wormMarketToAPIItem(market))
	}
	return item
}

type wormEventsListParams struct {
	Limit        int
	Cursor       string
	SortOption   string
	CategorySlug string
	State        string
}

func validateFixedWormEventParams(params wormEventsListParams) error {
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

func (s *Service) toAPIEvent(event *utilworm.Event) *v1alpha1.WormEventItem {
	if event == nil {
		return nil
	}
	item := &v1alpha1.WormEventItem{
		ConditionID: event.ConditionID,
		Title:       event.Title,
		Description: stringValue(event.Description),
		Logo:        s.normalizeAssetURL(stringValue(event.Logo)),
		Category:    event.Category,
		Created:     int64Value(event.Created),
		MarketCount: int64(len(event.Markets)),
		Markets:     make([]*v1alpha1.WormMarketItem, 0, len(event.Markets)),
	}
	for _, market := range event.Markets {
		item.Markets = append(item.Markets, s.toAPIMarketSummary(market))
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

	marketCount, err := s.store.CountWormMarkets(ctx)
	if err != nil {
		return err
	}
	existingEventIDs, err := s.store.ListWormEventConditionIDs(ctx)
	if err != nil {
		return err
	}
	existingEvents := make(map[string]struct{}, len(existingEventIDs))
	for _, eventConditionID := range existingEventIDs {
		existingEvents[eventConditionID] = struct{}{}
	}
	suppressNewEventNotifications := marketCount == 0
	newEvents := make(map[string]wormstore.WormMarket)

	syncStartedAt := time.Now().UTC()
	cursor := ""
	seen := map[string]bool{}
	for {
		markets, err := s.wormClient.ListMarkets(ctx, utilworm.ListMarketsOptions{
			PageOptions: utilworm.PageOptions{
				Limit:  maxWormMarketsSyncLimit,
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
		priceSamples := make([]wormstore.WormMarketPriceSample, 0, len(markets.Markets))
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
			if price, ok := parseWormMarketPrice(item.LastTradePrice); ok {
				priceSamples = append(priceSamples, wormstore.WormMarketPriceSample{
					ConditionID: item.ConditionID,
					Price:       price,
					SampledAt:   fetchedAt,
				})
			}
		}
		insertedConditionIDs, err := s.store.BatchUpsertWormMarkets(ctx, items)
		if err != nil {
			return err
		}
		inserted := make(map[string]struct{}, len(insertedConditionIDs))
		for _, conditionID := range insertedConditionIDs {
			inserted[conditionID] = struct{}{}
		}
		for _, item := range items {
			if _, ok := inserted[item.ConditionID]; !ok {
				continue
			}
			if _, existed := existingEvents[item.EventConditionID]; existed {
				continue
			}
			if _, collected := newEvents[item.EventConditionID]; !collected {
				newEvents[item.EventConditionID] = item
			}
		}
		if err := s.store.BatchInsertWormMarketPriceHistory(ctx, priceSamples); err != nil {
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
	s.updateWormMarketPriceAlerts(ctx)
	if !suppressNewEventNotifications {
		s.sendWormNotifications(ctx, newWormEventNotifications(newEvents))
	}
	log.Debugf("synced %d worm markets", len(seen))
	return nil
}

func (s *Service) rulesLoop(ctx context.Context) {
	defer s.syncWG.Done()
	s.updateWormMarketRules(ctx)
	ticker := time.NewTicker(wormMarketRulesInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.updateWormMarketRules(ctx)
		}
	}
}

func (s *Service) updateWormMarketRules(ctx context.Context) {
	if s.store == nil || s.wormClient == nil {
		return
	}
	conditionIDs, err := s.store.ListWormMarketConditionIDsMissingRules(ctx)
	if err != nil {
		if ctx.Err() == nil {
			log.Warnf("failed to list worm markets missing rules: %v", err)
		}
		return
	}
	updated := 0
	for _, conditionID := range conditionIDs {
		if ctx.Err() != nil {
			return
		}
		market, err := s.wormClient.GetMarket(ctx, conditionID)
		if err != nil {
			if ctx.Err() == nil {
				log.Warnf("failed to get worm market %s rules: %v", conditionID, err)
			}
			continue
		}
		if market == nil || strings.TrimSpace(market.ConditionID) != conditionID {
			log.Warnf("worm market rules response condition ID mismatch for %s", conditionID)
			continue
		}
		if market.Rules == nil {
			log.Warnf("worm market %s returned missing rules", conditionID)
			continue
		}
		rules, err := json.Marshal(market.Rules)
		if err != nil {
			log.Warnf("failed to marshal worm market %s rules: %v", conditionID, err)
			continue
		}
		rowsAffected, err := s.store.SetWormMarketRulesIfMissing(ctx, conditionID, rules)
		if err != nil {
			if ctx.Err() == nil {
				log.Warnf("failed to store worm market %s rules: %v", conditionID, err)
			}
			continue
		}
		if rowsAffected > 0 {
			updated++
		}
	}
	log.Debugf("updated rules for %d worm markets", updated)
}

func (s *Service) liveStateLoop(ctx context.Context) {
	defer s.syncWG.Done()
	s.updateWormMarketLiveStates(ctx)
	ticker := time.NewTicker(wormLiveStateLoopInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.updateWormMarketLiveStates(ctx)
		}
	}
}

func (s *Service) updateWormMarketLiveStates(ctx context.Context) {
	if s.store == nil {
		return
	}
	checkedAt := time.Now().UTC()
	windowStart := checkedAt.Add(-wormLivePriceWindow)
	if _, err := s.store.DeleteWormMarketPriceHistoryBefore(ctx, windowStart); err != nil {
		if ctx.Err() == nil {
			log.Warnf("failed to delete expired worm market price history: %v", err)
		}
		return
	}
	priceChanges, err := s.store.ListWormMarketLivePriceChanges(ctx, windowStart)
	if err != nil {
		if ctx.Err() == nil {
			log.Warnf("failed to list worm market live price changes: %v", err)
		}
		return
	}
	liveNotifications := make([]wormNotification, 0)
	notifiedEvents := make(map[string]struct{})
	for _, priceChange := range priceChanges {
		if ctx.Err() != nil {
			return
		}
		liveState := wormMarketLiveStateUnknown
		livePriceChange := ""
		if priceChange.SampleCount >= 2 {
			liveState = wormMarketLiveStateNotLive
			livePriceChange = priceChange.PriceChange
			if priceChange.IsLive {
				liveState = wormMarketLiveStateLive
			}
		}
		if _, err := s.store.UpdateWormMarketLiveState(ctx, wormstore.WormMarket{
			ConditionID:     priceChange.ConditionID,
			LiveState:       liveState,
			LiveCheckedAt:   checkedAt,
			LivePriceChange: livePriceChange,
		}); err != nil {
			if ctx.Err() == nil {
				log.Warnf("failed to update worm market live state for %s: %v", priceChange.ConditionID, err)
			}
			s.sendWormNotifications(ctx, liveNotifications)
			return
		}
		if liveState != wormMarketLiveStateLive || priceChange.EventHasLive {
			continue
		}
		if _, exists := notifiedEvents[priceChange.EventConditionID]; exists {
			continue
		}
		notifiedEvents[priceChange.EventConditionID] = struct{}{}
		liveNotifications = append(liveNotifications, newWormLiveNotification(priceChange))
	}
	s.sendWormNotifications(ctx, liveNotifications)
	log.Debugf("updated live states for %d worm markets", len(priceChanges))
}

func (s *Service) updateWormMarketPriceAlerts(ctx context.Context) {
	if s.store == nil {
		return
	}
	markets, err := s.store.ListWormLiveMarketsForPriceAlerts(ctx)
	if err != nil {
		if ctx.Err() == nil {
			log.Warnf("failed to list live worm markets for price alerts: %v", err)
		}
		return
	}
	for _, market := range markets {
		if ctx.Err() != nil {
			return
		}
		currentBand := strings.TrimSpace(market.PriceAlertBand)
		if currentBand == "" {
			currentBand = wormPriceAlertBandNone
		}
		nextBand := classifyWormPriceAlertBand(market.LastTradePrice)
		if nextBand == currentBand {
			continue
		}
		if nextBand != wormPriceAlertBandNone {
			results := s.sendWormNotifications(ctx, []wormNotification{
				newWormPriceAlertNotification(market, nextBand),
			})
			if len(results) != 1 || results[0].err != nil {
				continue
			}
		}
		updated, err := s.store.UpdateWormMarketPriceAlertBand(ctx, market.ConditionID, currentBand, nextBand)
		if err != nil {
			if ctx.Err() == nil {
				log.Warnf("failed to update worm market price alert band for %s: %v", market.ConditionID, err)
			}
			continue
		}
		if !updated && ctx.Err() == nil {
			log.Warnf("worm market price alert band changed concurrently for %s", market.ConditionID)
		}
	}
}

func classifyWormPriceAlertBand(value string) string {
	price, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(price) || math.IsInf(price, 0) || price < 0 || price > 1 {
		return wormPriceAlertBandNone
	}
	switch {
	case price <= 0.05 || price >= 0.95:
		return wormPriceAlertBandC
	case price < 0.1 || price >= 0.9:
		return wormPriceAlertBandB
	case price <= 0.2 || price >= 0.8:
		return wormPriceAlertBandA
	default:
		return wormPriceAlertBandNone
	}
}

func parseWormMarketPrice(value string) (string, bool) {
	value = strings.TrimSpace(value)
	price, err := strconv.ParseFloat(value, 64)
	if err != nil || price < 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return "", false
	}
	return value, true
}

func isSyncedWormMarket(market utilworm.MarketSummary) bool {
	return strings.TrimSpace(market.ConditionID) != "" &&
		market.Event != nil &&
		strings.TrimSpace(market.Event.ConditionID) != "" &&
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

func isWormStatusCode(err error, statusCode int) bool {
	var wormErr *utilworm.Error
	return errors.As(err, &wormErr) && wormErr.StatusCode == statusCode
}
