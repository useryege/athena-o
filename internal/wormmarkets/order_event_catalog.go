package wormmarkets

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/useryege/athena/internal/wormmarkets/apiclient"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	orderEventCatalogMarketConcurrency = 8
	orderMarketStateUnknown            = "unknown"

	orderMarketUnavailableDetail           = "MARKET_DETAIL_UNAVAILABLE"
	orderMarketUnavailableIDMismatch       = "MARKET_ID_MISMATCH"
	orderMarketUnavailableEventMismatch    = "MARKET_EVENT_MISMATCH"
	orderMarketUnavailableNotOpen          = "MARKET_NOT_OPEN"
	orderMarketUnavailableMarginDisabled   = "MARGIN_DISABLED"
	orderMarketUnavailableConfigMissing    = "CONFIG_MISSING"
	orderMarketUnavailableBackend          = "BACKEND_UNSUPPORTED"
	orderMarketUnavailableOutcomes         = "OUTCOMES_INVALID"
	orderMarketUnavailableNoOutcomes       = "NO_SELECTABLE_OUTCOMES"
	orderOutcomeUnavailableLeverageMissing = "MAX_LEVERAGE_MISSING"
	orderOutcomeUnavailableLeverageInvalid = "MAX_LEVERAGE_INVALID"
	orderOutcomeUnavailableLeverageBelow1  = "MAX_LEVERAGE_BELOW_ONE"
)

// GetOrderEventCatalog returns the provider's current event and all child
// markets as a safe order-combination catalog. It intentionally performs only
// one event read and one detail read per unique child market; no estimate or
// mutation is part of this operation.
func (s *Service) GetOrderEventCatalog(ctx context.Context, req *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
	if s.wormClient == nil {
		return nil, status.Error(codes.FailedPrecondition, "worm API client is required")
	}

	eventConditionID := ""
	if req != nil {
		eventConditionID = strings.TrimSpace(req.GetEventConditionId())
	}
	if err := validateOrderConditionID(eventConditionID); err != nil {
		return nil, status.Error(codes.InvalidArgument, "event_condition_id must be a canonical Solana public key")
	}

	event, err := s.wormClient.GetEvent(ctx, eventConditionID)
	if err != nil {
		if isWormStatusCode(err, http.StatusNotFound) {
			return nil, status.Errorf(codes.NotFound, "worm event %q was not found", eventConditionID)
		}
		return nil, status.Errorf(codes.Unavailable, "failed to get worm event: %v", err)
	}
	if event == nil || event.ConditionID != eventConditionID || validateOrderConditionID(event.ConditionID) != nil {
		return nil, status.Error(codes.Unavailable, "worm event response condition id mismatch")
	}

	marketConditionIDs := make([]string, len(event.Markets))
	seenMarketConditionIDs := make(map[string]struct{}, len(event.Markets))
	for i, market := range event.Markets {
		marketConditionID := strings.TrimSpace(market.ConditionID)
		if market.ConditionID != marketConditionID {
			return nil, status.Errorf(codes.Unavailable, "worm event response contains a non-canonical child market condition id at index %d", i)
		}
		if err := validateOrderConditionID(marketConditionID); err != nil {
			return nil, status.Errorf(codes.Unavailable, "worm event response contains an invalid child market condition id at index %d", i)
		}
		if _, exists := seenMarketConditionIDs[marketConditionID]; exists {
			return nil, status.Errorf(codes.Unavailable, "worm event response contains duplicate child market %q", marketConditionID)
		}
		seenMarketConditionIDs[marketConditionID] = struct{}{}
		marketConditionIDs[i] = marketConditionID
	}

	markets, err := s.getOrderEventCatalogMarkets(ctx, eventConditionID, event.Markets, marketConditionIDs)
	if err != nil {
		return nil, err
	}
	return &apiclient.GetOrderEventCatalogResponse{
		Event: &apiclient.OrderEventCatalog{
			EventConditionId: eventConditionID,
			Title:            orderCatalogDisplayTitle(event.Title, eventConditionID),
			Logo:             s.normalizeAssetURL(stringValue(event.Logo)),
			Markets:          markets,
		},
		FetchedAt: time.Now().Unix(),
	}, nil
}

func (s *Service) getOrderEventCatalogMarkets(
	ctx context.Context,
	eventConditionID string,
	summaries []utilworm.MarketSummary,
	marketConditionIDs []string,
) ([]*apiclient.OrderEventCatalogMarket, error) {
	if len(summaries) == 0 {
		return []*apiclient.OrderEventCatalogMarket{}, nil
	}

	markets := make([]*apiclient.OrderEventCatalogMarket, len(summaries))
	jobs := make(chan int, len(summaries))
	for i := range summaries {
		jobs <- i
	}
	close(jobs)

	workerCount := orderEventCatalogMarketConcurrency
	if workerCount > len(summaries) {
		workerCount = len(summaries)
	}
	var wg sync.WaitGroup
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for index := range jobs {
				if ctx.Err() != nil {
					return
				}
				markets[index] = s.getOrderEventCatalogMarket(
					ctx,
					eventConditionID,
					marketConditionIDs[index],
					summaries[index],
				)
			}
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	return markets, nil
}

func (s *Service) getOrderEventCatalogMarket(
	ctx context.Context,
	eventConditionID string,
	marketConditionID string,
	summary utilworm.MarketSummary,
) *apiclient.OrderEventCatalogMarket {
	item := orderEventCatalogMarketFromSummary(s, eventConditionID, marketConditionID, summary)
	market, err := s.wormClient.GetMarket(ctx, marketConditionID)
	if err != nil || market == nil {
		applyOrderMarketUnavailable(item, orderMarketUnavailableDetail, summary.Outcomes, nil, nil, nil)
		return item
	}

	if market.ConditionID != marketConditionID || validateOrderConditionID(market.ConditionID) != nil {
		applyOrderMarketUnavailable(item, orderMarketUnavailableIDMismatch, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, market.Config)
		return item
	}
	item.MarketConditionId = marketConditionID
	item.Title = orderCatalogDisplayTitle(market.Title, marketConditionID)
	item.Logo = s.normalizeAssetURL(stringValue(market.Logo))
	item.State = orderCatalogMarketState(market.State)
	item.MarginEnabled = market.MarginEnabled
	if market.Config != nil {
		item.Backend = strings.ToLower(strings.TrimSpace(market.Config.Kind))
	}

	if market.Event == nil || market.Event.ConditionID != eventConditionID || validateOrderConditionID(market.Event.ConditionID) != nil {
		applyOrderMarketUnavailable(item, orderMarketUnavailableEventMismatch, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, market.Config)
		return item
	}
	item.EventConditionId = eventConditionID
	if item.State != defaultWormMarketsState {
		applyOrderMarketUnavailable(item, orderMarketUnavailableNotOpen, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, market.Config)
		return item
	}
	if !market.MarginEnabled {
		applyOrderMarketUnavailable(item, orderMarketUnavailableMarginDisabled, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, market.Config)
		return item
	}
	if market.Config == nil {
		applyOrderMarketUnavailable(item, orderMarketUnavailableConfigMissing, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, nil)
		return item
	}
	if item.Backend != "polymarket" && item.Backend != "hyperliquid" {
		applyOrderMarketUnavailable(item, orderMarketUnavailableBackend, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, market.Config)
		return item
	}

	yesLabel, noLabel, ok := validOrderOutcomeLabels(market.Outcomes)
	if !ok {
		applyOrderMarketUnavailable(item, orderMarketUnavailableOutcomes, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, market.Config)
		return item
	}

	yesOutcome := orderEventCatalogOutcome(true, yesLabel, market.Config.MaxLeverageYes)
	noOutcome := orderEventCatalogOutcome(false, noLabel, market.Config.MaxLeverageNo)
	item.Outcomes = []*apiclient.OrderEventCatalogOutcome{yesOutcome, noOutcome}
	if !yesOutcome.Selectable && !noOutcome.Selectable {
		item.UnavailableCode = orderMarketUnavailableNoOutcomes
	}
	return item
}

func orderEventCatalogMarketFromSummary(
	service *Service,
	eventConditionID string,
	marketConditionID string,
	summary utilworm.MarketSummary,
) *apiclient.OrderEventCatalogMarket {
	return &apiclient.OrderEventCatalogMarket{
		MarketConditionId: marketConditionID,
		EventConditionId:  eventConditionID,
		Title:             orderCatalogDisplayTitle(summary.Title, marketConditionID),
		Logo:              service.normalizeAssetURL(stringValue(summary.Logo)),
		State:             orderCatalogMarketState(summary.State),
		MarginEnabled:     summary.MarginEnabled,
		Outcomes:          unavailableOrderOutcomes(summary.Outcomes, nil, nil, nil, orderMarketUnavailableDetail),
	}
}

func applyOrderMarketUnavailable(
	item *apiclient.OrderEventCatalogMarket,
	code string,
	outcomes []utilworm.Outcome,
	yesOutcomeLabel *string,
	noOutcomeLabel *string,
	config *utilworm.MarketConfig,
) {
	if item == nil {
		return
	}
	item.UnavailableCode = code
	item.Outcomes = unavailableOrderOutcomes(outcomes, yesOutcomeLabel, noOutcomeLabel, config, code)
}

func unavailableOrderOutcomes(
	outcomes []utilworm.Outcome,
	yesOutcomeLabel *string,
	noOutcomeLabel *string,
	config *utilworm.MarketConfig,
	code string,
) []*apiclient.OrderEventCatalogOutcome {
	yesLabel, noLabel := displayOrderOutcomeLabels(outcomes, yesOutcomeLabel, noOutcomeLabel)
	yesMaxLeverage := ""
	noMaxLeverage := ""
	if config != nil {
		yesMaxLeverage = strings.TrimSpace(stringValue(config.MaxLeverageYes))
		noMaxLeverage = strings.TrimSpace(stringValue(config.MaxLeverageNo))
	}
	return []*apiclient.OrderEventCatalogOutcome{
		{
			IsYes:           true,
			Label:           yesLabel,
			MaxLeverage:     yesMaxLeverage,
			UnavailableCode: code,
		},
		{
			IsYes:           false,
			Label:           noLabel,
			MaxLeverage:     noMaxLeverage,
			UnavailableCode: code,
		},
	}
}

func displayOrderOutcomeLabels(outcomes []utilworm.Outcome, yesOutcomeLabel, noOutcomeLabel *string) (string, string) {
	yesLabel := strings.TrimSpace(stringValue(yesOutcomeLabel))
	noLabel := strings.TrimSpace(stringValue(noOutcomeLabel))
	for _, outcome := range outcomes {
		label := strings.TrimSpace(outcome.Text)
		if label == "" {
			continue
		}
		if outcome.IsYes && yesLabel == "" {
			yesLabel = label
		}
		if !outcome.IsYes && noLabel == "" {
			noLabel = label
		}
	}
	if yesLabel == "" {
		yesLabel = "YES"
	}
	if noLabel == "" {
		noLabel = "NO"
	}
	return yesLabel, noLabel
}

func validOrderOutcomeLabels(outcomes []utilworm.Outcome) (string, string, bool) {
	if len(outcomes) != 2 {
		return "", "", false
	}
	var yesLabel, noLabel string
	var yesCount, noCount int
	for _, outcome := range outcomes {
		label := strings.TrimSpace(outcome.Text)
		if label == "" {
			return "", "", false
		}
		if outcome.IsYes {
			yesCount++
			yesLabel = label
		} else {
			noCount++
			noLabel = label
		}
	}
	return yesLabel, noLabel, yesCount == 1 && noCount == 1
}

func orderEventCatalogOutcome(isYes bool, label string, maxLeverage *string) *apiclient.OrderEventCatalogOutcome {
	value := strings.TrimSpace(stringValue(maxLeverage))
	item := &apiclient.OrderEventCatalogOutcome{
		IsYes:       isYes,
		Label:       label,
		MaxLeverage: value,
	}
	if value == "" {
		item.UnavailableCode = orderOutcomeUnavailableLeverageMissing
		return item
	}
	leverage, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(leverage) || math.IsInf(leverage, 0) {
		item.UnavailableCode = orderOutcomeUnavailableLeverageInvalid
		return item
	}
	if leverage < 1 {
		item.UnavailableCode = orderOutcomeUnavailableLeverageBelow1
		return item
	}
	item.Selectable = true
	return item
}

func validateOrderConditionID(value string) error {
	if value == "" {
		return status.Error(codes.InvalidArgument, "condition id is required")
	}
	publicKey, err := solana.PublicKeyFromBase58(value)
	if err != nil || publicKey.String() != value {
		return status.Error(codes.InvalidArgument, "condition id is invalid")
	}
	return nil
}

func orderCatalogDisplayTitle(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func orderCatalogMarketState(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return orderMarketStateUnknown
	}
	return value
}
