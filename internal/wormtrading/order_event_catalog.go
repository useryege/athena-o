package wormtrading

import (
	"context"
	"errors"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	orderEventCatalogMarketConcurrency = 8
	orderMarketStateUnknown            = "unknown"
	orderCatalogMaxPriceLength         = 128

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

type orderEventCatalogPrices struct {
	yes string
	no  string
}

type WormCatalogClient interface {
	GetEvent(context.Context, string) (*utilworm.Event, error)
	GetMarket(context.Context, string) (*utilworm.Market, error)
}

type OrderEventCatalogReader interface {
	GetOrderEventCatalog(context.Context, *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error)
}

type orderEventCatalogReader struct {
	client  WormCatalogClient
	budget  time.Duration
	observe func(error)
}

func NewOrderEventCatalogReader(client WormCatalogClient, budget time.Duration, observe func(error)) (OrderEventCatalogReader, error) {
	if client == nil {
		return nil, errors.New("worm catalog client is required")
	}
	if budget <= 0 {
		return nil, errors.New("worm catalog budget must be positive")
	}
	return &orderEventCatalogReader{client: client, budget: budget, observe: observe}, nil
}

func (r *orderEventCatalogReader) GetOrderEventCatalog(ctx context.Context, req *apiclient.GetOrderEventCatalogRequest) (_ *apiclient.GetOrderEventCatalogResponse, resultErr error) {
	startedAt := time.Now()
	eventConditionID := ""
	if req != nil {
		eventConditionID = strings.TrimSpace(req.GetEventConditionId())
	}
	defer func() {
		log.WithFields(log.Fields{
			"duration_ms":        time.Since(startedAt).Milliseconds(),
			"error_code":         status.Code(resultErr).String(),
			"event_condition_id": eventConditionID,
		}).Info("Worm order event catalog read completed")
	}()
	if err := validateOrderConditionID(eventConditionID); err != nil {
		return nil, status.Error(codes.InvalidArgument, "event_condition_id must be a canonical Solana public key")
	}

	catalogCtx, cancel := context.WithTimeout(ctx, r.budget)
	defer cancel()
	if err := catalogCtx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	event, err := r.client.GetEvent(catalogCtx, eventConditionID)
	r.observeProviderResult(err)
	if err != nil {
		return nil, mapOrderCatalogEventError(catalogCtx, eventConditionID, err)
	}
	if err := catalogCtx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
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

	markets, err := r.getOrderEventCatalogMarkets(catalogCtx, eventConditionID, event.Markets, marketConditionIDs)
	if err != nil {
		return nil, err
	}

	return &apiclient.GetOrderEventCatalogResponse{
		Event: &apiclient.OrderEventCatalog{
			EventConditionId: eventConditionID,
			Title:            orderCatalogDisplayTitle(event.Title, eventConditionID),
			Logo:             normalizeOrderCatalogAssetURL(orderCatalogStringValue(event.Logo)),
			Markets:          markets,
		},
		FetchedAt: time.Now().Unix(),
	}, nil
}

func (r *orderEventCatalogReader) getOrderEventCatalogMarkets(
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
				markets[index] = r.getOrderEventCatalogMarket(
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

func (r *orderEventCatalogReader) getOrderEventCatalogMarket(
	ctx context.Context,
	eventConditionID string,
	marketConditionID string,
	summary utilworm.MarketSummary,
) *apiclient.OrderEventCatalogMarket {
	item := orderEventCatalogMarketFromSummary(eventConditionID, marketConditionID, summary)
	market, err := r.client.GetMarket(ctx, marketConditionID)
	r.observeProviderResult(err)
	if err != nil || market == nil {
		applyOrderMarketUnavailable(
			item,
			orderMarketUnavailableDetail,
			summary.Outcomes,
			nil,
			nil,
			nil,
			orderEventCatalogPricesFromLastTrade(summary.LastTradePrice),
		)
		return item
	}
	if market.ConditionID != marketConditionID || validateOrderConditionID(market.ConditionID) != nil {
		applyOrderMarketUnavailable(item, orderMarketUnavailableIDMismatch, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, market.Config, orderEventCatalogPrices{})
		return item
	}
	item.MarketConditionId = marketConditionID
	item.Title = orderCatalogDisplayTitle(market.Title, marketConditionID)
	item.Logo = normalizeOrderCatalogAssetURL(orderCatalogStringValue(market.Logo))
	item.State = orderCatalogMarketState(market.State)
	item.MarginEnabled = market.MarginEnabled
	if market.Config != nil {
		item.Backend = strings.ToLower(strings.TrimSpace(market.Config.Kind))
	}

	if market.Event == nil || market.Event.ConditionID != eventConditionID || validateOrderConditionID(market.Event.ConditionID) != nil {
		applyOrderMarketUnavailable(item, orderMarketUnavailableEventMismatch, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, market.Config, orderEventCatalogPrices{})
		return item
	}
	prices := orderEventCatalogPricesFromLastTrade(market.LastTradePrice)
	item.EventConditionId = eventConditionID
	if item.State != "open" {
		applyOrderMarketUnavailable(item, orderMarketUnavailableNotOpen, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, market.Config, prices)
		return item
	}
	if !market.MarginEnabled {
		applyOrderMarketUnavailable(item, orderMarketUnavailableMarginDisabled, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, market.Config, prices)
		return item
	}
	if market.Config == nil {
		applyOrderMarketUnavailable(item, orderMarketUnavailableConfigMissing, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, nil, prices)
		return item
	}
	if item.Backend != "polymarket" && item.Backend != "hyperliquid" {
		applyOrderMarketUnavailable(item, orderMarketUnavailableBackend, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, market.Config, prices)
		return item
	}

	yesLabel, noLabel, ok := validOrderOutcomeLabels(market.Outcomes)
	if !ok {
		applyOrderMarketUnavailable(item, orderMarketUnavailableOutcomes, market.Outcomes, market.YesOutcomeLabel, market.NoOutcomeLabel, market.Config, prices)
		return item
	}

	yesOutcome := orderEventCatalogOutcome(true, yesLabel, market.Config.MaxLeverageYes, prices.yes)
	noOutcome := orderEventCatalogOutcome(false, noLabel, market.Config.MaxLeverageNo, prices.no)
	item.Outcomes = []*apiclient.OrderEventCatalogOutcome{yesOutcome, noOutcome}
	if !yesOutcome.Selectable && !noOutcome.Selectable {
		item.UnavailableCode = orderMarketUnavailableNoOutcomes
	}
	return item
}

func orderEventCatalogMarketFromSummary(
	eventConditionID string,
	marketConditionID string,
	summary utilworm.MarketSummary,
) *apiclient.OrderEventCatalogMarket {
	return &apiclient.OrderEventCatalogMarket{
		MarketConditionId: marketConditionID,
		EventConditionId:  eventConditionID,
		Title:             orderCatalogDisplayTitle(summary.Title, marketConditionID),
		Logo:              normalizeOrderCatalogAssetURL(orderCatalogStringValue(summary.Logo)),
		State:             orderCatalogMarketState(summary.State),
		MarginEnabled:     summary.MarginEnabled,
		Outcomes: unavailableOrderOutcomes(
			summary.Outcomes,
			nil,
			nil,
			nil,
			orderMarketUnavailableDetail,
			orderEventCatalogPricesFromLastTrade(summary.LastTradePrice),
		),
	}
}

func applyOrderMarketUnavailable(
	item *apiclient.OrderEventCatalogMarket,
	code string,
	outcomes []utilworm.Outcome,
	yesOutcomeLabel *string,
	noOutcomeLabel *string,
	config *utilworm.MarketConfig,
	prices orderEventCatalogPrices,
) {
	if item == nil {
		return
	}
	item.UnavailableCode = code
	item.Outcomes = unavailableOrderOutcomes(outcomes, yesOutcomeLabel, noOutcomeLabel, config, code, prices)
}

func unavailableOrderOutcomes(
	outcomes []utilworm.Outcome,
	yesOutcomeLabel *string,
	noOutcomeLabel *string,
	config *utilworm.MarketConfig,
	code string,
	prices orderEventCatalogPrices,
) []*apiclient.OrderEventCatalogOutcome {
	yesLabel, noLabel := displayOrderOutcomeLabels(outcomes, yesOutcomeLabel, noOutcomeLabel)
	yesMaxLeverage := ""
	noMaxLeverage := ""
	if config != nil {
		yesMaxLeverage = strings.TrimSpace(orderCatalogStringValue(config.MaxLeverageYes))
		noMaxLeverage = strings.TrimSpace(orderCatalogStringValue(config.MaxLeverageNo))
	}
	return []*apiclient.OrderEventCatalogOutcome{
		{
			IsYes:           true,
			Label:           yesLabel,
			MaxLeverage:     yesMaxLeverage,
			UnavailableCode: code,
			LastTradePrice:  prices.yes,
		},
		{
			IsYes:           false,
			Label:           noLabel,
			MaxLeverage:     noMaxLeverage,
			UnavailableCode: code,
			LastTradePrice:  prices.no,
		},
	}
}

func displayOrderOutcomeLabels(outcomes []utilworm.Outcome, yesOutcomeLabel, noOutcomeLabel *string) (string, string) {
	yesLabel := strings.TrimSpace(orderCatalogStringValue(yesOutcomeLabel))
	noLabel := strings.TrimSpace(orderCatalogStringValue(noOutcomeLabel))
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

func orderEventCatalogOutcome(isYes bool, label string, maxLeverage *string, lastTradePrice string) *apiclient.OrderEventCatalogOutcome {
	value := strings.TrimSpace(orderCatalogStringValue(maxLeverage))
	item := &apiclient.OrderEventCatalogOutcome{
		IsYes:          isYes,
		Label:          label,
		MaxLeverage:    value,
		LastTradePrice: lastTradePrice,
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

func orderEventCatalogPricesFromLastTrade(lastTradePrice *string) orderEventCatalogPrices {
	value := strings.TrimSpace(orderCatalogStringValue(lastTradePrice))
	if value == "" || len(value) > orderCatalogMaxPriceLength || !isPlainOrderCatalogDecimal(value) {
		return orderEventCatalogPrices{}
	}
	complement, ok := complementOrderCatalogDecimal(value)
	if !ok {
		return orderEventCatalogPrices{}
	}
	return orderEventCatalogPrices{yes: value, no: complement}
}

func complementOrderCatalogDecimal(value string) (string, bool) {
	parts := strings.SplitN(value, ".", 2)
	whole := parts[0]
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if whole == "" {
		whole = "0"
	}

	unscaled, ok := new(big.Int).SetString(whole+fraction, 10)
	if !ok {
		return "", false
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(len(fraction))), nil)
	if unscaled.Cmp(scale) > 0 {
		return "", false
	}

	complement := new(big.Int).Sub(scale, unscaled)
	if complement.Sign() == 0 {
		return "0", true
	}
	if complement.Cmp(scale) == 0 {
		return "1", true
	}

	digits := complement.String()
	if padding := len(fraction) - len(digits); padding > 0 {
		digits = strings.Repeat("0", padding) + digits
	}
	digits = strings.TrimRight(digits, "0")
	return "0." + digits, true
}

func isPlainOrderCatalogDecimal(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" || (len(parts) == 2 && parts[1] == "") {
		return false
	}
	for _, part := range parts {
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}
	return true
}

func orderCatalogMarketState(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return orderMarketStateUnknown
	}
	return value
}

func (r *orderEventCatalogReader) observeProviderResult(err error) {
	if r.observe != nil {
		r.observe(err)
	}
}

func mapOrderCatalogEventError(ctx context.Context, eventConditionID string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return status.FromContextError(ctxErr).Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return status.FromContextError(err).Err()
	}
	if isWormStatusCode(err, http.StatusNotFound) {
		return status.Errorf(codes.NotFound, "worm event %q was not found", eventConditionID)
	}
	return status.Errorf(codes.Unavailable, "failed to get worm event: %v", err)
}

func normalizeOrderCatalogAssetURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	assetURL, err := url.Parse(value)
	if err != nil || assetURL.IsAbs() {
		return value
	}
	baseURL, err := url.Parse(OfficialWormAPIBaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return value
	}
	return baseURL.ResolveReference(assetURL).String()
}

func orderCatalogStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func orderCatalogDisplayTitle(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func isWormStatusCode(err error, statusCode int) bool {
	var wormErr *utilworm.Error
	return errors.As(err, &wormErr) && wormErr.StatusCode == statusCode
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
