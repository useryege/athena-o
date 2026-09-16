package server

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"math/big"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/walletsecret"
	wormtradingapiclient "github.com/useryege/athena/internal/wormtrading/apiclient"
)

const (
	wormEventCatalogPath            = "/api/v1/worm-trading/events/{eventConditionId}"
	wormCombinationCollectionPath   = "/api/v1/worm-trading/combinations"
	wormCombinationResourcePath     = wormCombinationCollectionPath + "/{combinationId}"
	wormCombinationDefaultPageSize  = int32(20)
	wormCombinationMaximumPageSize  = int32(100)
	wormCombinationMaximumBodyBytes = int64(1024 * 1024)
	wormCombinationMaximumNameRunes = 80
	wormCatalogMaximumPriceLength   = 128
)

type wormCombinationInput struct {
	Name             string                         `json:"name"`
	ExpectedRevision int64                          `json:"expectedRevision,omitempty"`
	Items            []wormCombinationItemSelection `json:"items"`
}

type wormCombinationItemSelection struct {
	EventConditionID  string `json:"eventConditionId"`
	MarketConditionID string `json:"marketConditionId"`
	Side              string `json:"side"`
}

type wormEventCatalogResponse struct {
	EventConditionID string                   `json:"eventConditionId"`
	Title            string                   `json:"title"`
	Logo             string                   `json:"logo"`
	Markets          []wormEventCatalogMarket `json:"markets"`
	FetchedAt        int64                    `json:"fetchedAt"`
}

type wormEventCatalogMarket struct {
	MarketConditionID string                    `json:"marketConditionId"`
	Title             string                    `json:"title"`
	Logo              string                    `json:"logo"`
	State             string                    `json:"state"`
	MarginEnabled     bool                      `json:"marginEnabled"`
	Backend           string                    `json:"backend"`
	UnavailableCode   string                    `json:"unavailableCode"`
	Outcomes          []wormEventCatalogOutcome `json:"outcomes"`
}

type wormEventCatalogOutcome struct {
	Side            string `json:"side"`
	Label           string `json:"label"`
	MaxLeverage     string `json:"maxLeverage"`
	LastTradePrice  string `json:"lastTradePrice"`
	Selectable      bool   `json:"selectable"`
	UnavailableCode string `json:"unavailableCode"`
}

type wormCombinationResponse struct {
	ID        string                        `json:"id"`
	Name      string                        `json:"name"`
	Revision  int64                         `json:"revision"`
	Items     []wormCombinationResponseItem `json:"items"`
	CreatedAt int64                         `json:"createdAt"`
	UpdatedAt int64                         `json:"updatedAt"`
}

type wormCombinationResponseItem struct {
	Ordinal           int32  `json:"ordinal"`
	EventConditionID  string `json:"eventConditionId"`
	EventTitle        string `json:"eventTitle"`
	EventLogo         string `json:"eventLogo"`
	MarketConditionID string `json:"marketConditionId"`
	MarketTitle       string `json:"marketTitle"`
	MarketLogo        string `json:"marketLogo"`
	Side              string `json:"side"`
	OutcomeLabel      string `json:"outcomeLabel"`
}

type wormCombinationListResponse struct {
	Items    []wormCombinationResponse `json:"items"`
	Total    int64                     `json:"total"`
	Page     int32                     `json:"page"`
	PageSize int32                     `json:"pageSize"`
}

func registerWormCombinationHandlers(mux *http.ServeMux, server *AthenaServer) {
	if mux == nil || server == nil {
		return
	}
	mux.Handle("GET "+wormEventCatalogPath, traceHTTP(http.HandlerFunc(server.getWormOrderEventCatalog)))
	mux.Handle("GET "+wormCombinationCollectionPath, traceHTTP(http.HandlerFunc(server.listWormCombinations)))
	mux.Handle("POST "+wormCombinationCollectionPath, traceHTTP(http.HandlerFunc(server.createWormCombination)))
	mux.Handle("GET "+wormCombinationResourcePath, traceHTTP(http.HandlerFunc(server.getWormCombination)))
	mux.Handle("PUT "+wormCombinationResourcePath, traceHTTP(http.HandlerFunc(server.updateWormCombination)))
	mux.Handle("DELETE "+wormCombinationResourcePath, traceHTTP(http.HandlerFunc(server.deleteWormCombination)))
}

func (server *AthenaServer) getWormOrderEventCatalog(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelRead)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	ctx = withWormTradingAccountIdentity(ctx, credential.AccountID)
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	eventConditionID, err := canonicalWormConditionID(request.PathValue("eventConditionId"), "event condition ID")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	catalog, err := server.loadWormOrderEventCatalog(ctx, eventConditionID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, catalog)
}

func (server *AthenaServer) listWormCombinations(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelRead)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	page, pageSize, err := wormCombinationPagination(request)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.ListMarketCombinations(ctx, &wormtradingapiclient.ListMarketCombinationsRequest{
		OwnerAccountId: credential.AccountID,
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormCombinationStoreError(err))
		return
	}
	response, err := projectWormCombinationList(result, page, pageSize, credential.AccountID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) getWormCombination(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelRead)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	combinationID, err := canonicalWormCombinationID(request.PathValue("combinationId"))
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.GetMarketCombination(ctx, &wormtradingapiclient.GetMarketCombinationRequest{
		OwnerAccountId: credential.AccountID,
		Id:             combinationID,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormCombinationStoreError(err))
		return
	}
	response, err := projectWormCombination(result.GetCombination(), credential.AccountID, combinationID, 0)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) createWormCombination(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if !server.validWormConnectionOrigin(request) {
		walletsecret.WriteError(w, status.Error(codes.PermissionDenied, "a same-origin Worm Trading request is required"))
		return
	}
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelReadWrite)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	ctx = withWormTradingAccountIdentity(ctx, credential.AccountID)
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	input, err := decodeWormCombinationInput(w, request, false)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	items, err := wormCombinationSelectionsToProto(input.Items)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.CreateMarketCombination(ctx, &wormtradingapiclient.CreateMarketCombinationRequest{
		OwnerAccountId: credential.AccountID,
		Name:           input.Name,
		Items:          items,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormCombinationStoreError(err))
		return
	}
	response, err := projectWormCombination(result.GetCombination(), credential.AccountID, "", 1)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusCreated, response)
}

func (server *AthenaServer) updateWormCombination(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if !server.validWormConnectionOrigin(request) {
		walletsecret.WriteError(w, status.Error(codes.PermissionDenied, "a same-origin Worm Trading request is required"))
		return
	}
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelReadWrite)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	ctx = withWormTradingAccountIdentity(ctx, credential.AccountID)
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	combinationID, err := canonicalWormCombinationID(request.PathValue("combinationId"))
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	input, err := decodeWormCombinationInput(w, request, true)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	items, err := wormCombinationSelectionsToProto(input.Items)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.UpdateMarketCombination(ctx, &wormtradingapiclient.UpdateMarketCombinationRequest{
		OwnerAccountId:   credential.AccountID,
		Id:               combinationID,
		Name:             input.Name,
		ExpectedRevision: input.ExpectedRevision,
		Items:            items,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormCombinationStoreError(err))
		return
	}
	response, err := projectWormCombination(result.GetCombination(), credential.AccountID, combinationID, input.ExpectedRevision+1)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) deleteWormCombination(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if !server.validWormConnectionOrigin(request) {
		walletsecret.WriteError(w, status.Error(codes.PermissionDenied, "a same-origin Worm Trading request is required"))
		return
	}
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelReadWrite)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	combinationID, err := canonicalWormCombinationID(request.PathValue("combinationId"))
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	expectedRevision, err := wormCombinationDeleteRevision(request)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	_, err = client.DeleteMarketCombination(ctx, &wormtradingapiclient.DeleteMarketCombinationRequest{
		OwnerAccountId:   credential.AccountID,
		Id:               combinationID,
		ExpectedRevision: expectedRevision,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormCombinationStoreError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (server *AthenaServer) loadWormOrderEventCatalog(ctx context.Context, eventConditionID string) (wormEventCatalogResponse, error) {
	if server.WormTradingClientset == nil || server.WormTradingClientset.WormTrading() == nil {
		return wormEventCatalogResponse{}, status.Error(codes.Unavailable, "Worm Trading is unavailable")
	}
	result, err := server.WormTradingClientset.WormTrading().GetOrderEventCatalog(ctx, &wormtradingapiclient.GetOrderEventCatalogRequest{
		EventConditionId: eventConditionID,
	})
	if err != nil {
		return wormEventCatalogResponse{}, sanitizeWormCatalogDependencyError(err)
	}
	return projectWormOrderEventCatalog(result, eventConditionID)
}

func wormCombinationSelectionsToProto(selections []wormCombinationItemSelection) ([]*wormtradingapiclient.MarketCombinationItemInput, error) {
	if len(selections) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one market is required")
	}
	seenMarkets := make(map[string]struct{}, len(selections))
	converted := make([]*wormtradingapiclient.MarketCombinationItemInput, 0, len(selections))
	for index, selection := range selections {
		eventConditionID, err := canonicalWormConditionID(selection.EventConditionID, "item event condition ID")
		if err != nil {
			return nil, err
		}
		marketConditionID, err := canonicalWormConditionID(selection.MarketConditionID, "item market condition ID")
		if err != nil {
			return nil, err
		}
		if _, exists := seenMarkets[marketConditionID]; exists {
			return nil, status.Errorf(codes.InvalidArgument, "market condition ID at item %d is duplicated", index+1)
		}
		seenMarkets[marketConditionID] = struct{}{}
		isYes := false
		switch selection.Side {
		case "YES":
			isYes = true
		case "NO":
		default:
			return nil, status.Errorf(codes.InvalidArgument, "item %d side must be YES or NO", index+1)
		}
		converted = append(converted, &wormtradingapiclient.MarketCombinationItemInput{
			EventConditionId: eventConditionID, MarketConditionId: marketConditionID, IsYes: isYes,
		})
	}
	return converted, nil
}

func projectWormOrderEventCatalog(
	response *wormtradingapiclient.GetOrderEventCatalogResponse,
	expectedEventConditionID string,
) (wormEventCatalogResponse, error) {
	if response == nil || response.GetEvent() == nil || response.GetFetchedAt() <= 0 {
		return wormEventCatalogResponse{}, status.Error(codes.Internal, "Worm Trading returned an incomplete event catalog")
	}
	event := response.GetEvent()
	if event.GetEventConditionId() != expectedEventConditionID {
		return wormEventCatalogResponse{}, status.Error(codes.Internal, "Worm Trading returned a mismatched event catalog")
	}
	if _, err := canonicalWormConditionID(event.GetEventConditionId(), "event condition ID"); err != nil {
		return wormEventCatalogResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid event catalog ID")
	}
	title := strings.TrimSpace(event.GetTitle())
	if title == "" {
		return wormEventCatalogResponse{}, status.Error(codes.Internal, "Worm Trading returned an event without a title")
	}
	result := wormEventCatalogResponse{
		EventConditionID: event.GetEventConditionId(),
		Title:            title,
		Logo:             strings.TrimSpace(event.GetLogo()),
		Markets:          make([]wormEventCatalogMarket, 0, len(event.GetMarkets())),
		FetchedAt:        response.GetFetchedAt(),
	}
	seenMarkets := make(map[string]struct{}, len(event.GetMarkets()))
	for _, market := range event.GetMarkets() {
		projected, err := projectWormOrderEventMarket(market, expectedEventConditionID)
		if err != nil {
			return wormEventCatalogResponse{}, err
		}
		if _, exists := seenMarkets[projected.MarketConditionID]; exists {
			return wormEventCatalogResponse{}, status.Error(codes.Internal, "Worm Trading returned duplicate market catalog IDs")
		}
		seenMarkets[projected.MarketConditionID] = struct{}{}
		result.Markets = append(result.Markets, projected)
	}
	return result, nil
}

func projectWormOrderEventMarket(
	market *wormtradingapiclient.OrderEventCatalogMarket,
	expectedEventConditionID string,
) (wormEventCatalogMarket, error) {
	if market == nil || market.GetEventConditionId() != expectedEventConditionID {
		return wormEventCatalogMarket{}, status.Error(codes.Internal, "Worm Trading returned a mismatched child market")
	}
	marketConditionID, err := canonicalWormConditionID(market.GetMarketConditionId(), "market condition ID")
	if err != nil || marketConditionID != market.GetMarketConditionId() {
		return wormEventCatalogMarket{}, status.Error(codes.Internal, "Worm Trading returned an invalid child market ID")
	}
	title := strings.TrimSpace(market.GetTitle())
	state := strings.ToLower(strings.TrimSpace(market.GetState()))
	backend := strings.ToLower(strings.TrimSpace(market.GetBackend()))
	unavailableCode := strings.TrimSpace(market.GetUnavailableCode())
	if title == "" || state == "" || unavailableCode != market.GetUnavailableCode() {
		return wormEventCatalogMarket{}, status.Error(codes.Internal, "Worm Trading returned an invalid child market projection")
	}
	projected := wormEventCatalogMarket{
		MarketConditionID: marketConditionID,
		Title:             title,
		Logo:              strings.TrimSpace(market.GetLogo()),
		State:             state,
		MarginEnabled:     market.GetMarginEnabled(),
		Backend:           backend,
		UnavailableCode:   unavailableCode,
		Outcomes:          make([]wormEventCatalogOutcome, 0, len(market.GetOutcomes())),
	}
	seenSides := make(map[string]struct{}, len(market.GetOutcomes()))
	for _, outcome := range market.GetOutcomes() {
		projectedOutcome, err := projectWormOrderEventOutcome(outcome)
		if err != nil {
			return wormEventCatalogMarket{}, err
		}
		if _, exists := seenSides[projectedOutcome.Side]; exists {
			return wormEventCatalogMarket{}, status.Error(codes.Internal, "Worm Trading returned duplicate market outcome sides")
		}
		seenSides[projectedOutcome.Side] = struct{}{}
		projected.Outcomes = append(projected.Outcomes, projectedOutcome)
	}
	if len(projected.Outcomes) != 2 {
		return wormEventCatalogMarket{}, status.Error(codes.Internal, "Worm Trading returned an incomplete market outcome catalog")
	}
	if _, ok := seenSides["YES"]; !ok {
		return wormEventCatalogMarket{}, status.Error(codes.Internal, "Worm Trading omitted the YES market outcome")
	}
	if _, ok := seenSides["NO"]; !ok {
		return wormEventCatalogMarket{}, status.Error(codes.Internal, "Worm Trading omitted the NO market outcome")
	}
	if err := validateWormOrderEventPrices(projected.Outcomes); err != nil {
		return wormEventCatalogMarket{}, err
	}
	for _, outcome := range projected.Outcomes {
		if outcome.Selectable && (projected.UnavailableCode != "" || projected.State != "open" || !projected.MarginEnabled || (projected.Backend != "polymarket" && projected.Backend != "hyperliquid")) {
			return wormEventCatalogMarket{}, status.Error(codes.Internal, "Worm Trading returned an unsafe selectable market outcome")
		}
	}
	return projected, nil
}

func projectWormOrderEventOutcome(outcome *wormtradingapiclient.OrderEventCatalogOutcome) (wormEventCatalogOutcome, error) {
	if outcome == nil {
		return wormEventCatalogOutcome{}, status.Error(codes.Internal, "Worm Trading returned an empty market outcome")
	}
	side := "NO"
	if outcome.GetIsYes() {
		side = "YES"
	}
	label := strings.TrimSpace(outcome.GetLabel())
	maxLeverage := strings.TrimSpace(outcome.GetMaxLeverage())
	lastTradePrice := strings.TrimSpace(outcome.GetLastTradePrice())
	unavailableCode := strings.TrimSpace(outcome.GetUnavailableCode())
	if label == "" || lastTradePrice != outcome.GetLastTradePrice() || unavailableCode != outcome.GetUnavailableCode() {
		return wormEventCatalogOutcome{}, status.Error(codes.Internal, "Worm Trading returned an invalid market outcome")
	}
	if lastTradePrice != "" {
		if _, ok := parseWormOrderEventPrice(lastTradePrice); !ok {
			return wormEventCatalogOutcome{}, status.Error(codes.Internal, "Worm Trading returned an invalid market last-trade price")
		}
	}
	if outcome.GetSelectable() {
		leverage, err := strconv.ParseFloat(maxLeverage, 64)
		if unavailableCode != "" || err != nil || math.IsNaN(leverage) || math.IsInf(leverage, 0) || leverage < 1 {
			return wormEventCatalogOutcome{}, status.Error(codes.Internal, "Worm Trading returned an invalid selectable market outcome")
		}
	} else if unavailableCode == "" {
		return wormEventCatalogOutcome{}, status.Error(codes.Internal, "Worm Trading returned an unavailable outcome without a reason")
	}
	return wormEventCatalogOutcome{
		Side:            side,
		Label:           label,
		MaxLeverage:     maxLeverage,
		LastTradePrice:  lastTradePrice,
		Selectable:      outcome.GetSelectable(),
		UnavailableCode: unavailableCode,
	}, nil
}

func validateWormOrderEventPrices(outcomes []wormEventCatalogOutcome) error {
	prices := make(map[string]string, len(outcomes))
	for _, outcome := range outcomes {
		prices[outcome.Side] = outcome.LastTradePrice
	}
	yesPrice := prices["YES"]
	noPrice := prices["NO"]
	if (yesPrice == "") != (noPrice == "") {
		return status.Error(codes.Internal, "Worm Trading returned an incomplete market last-trade price pair")
	}
	if yesPrice == "" {
		return nil
	}
	yes, yesOK := parseWormOrderEventPrice(yesPrice)
	no, noOK := parseWormOrderEventPrice(noPrice)
	if !yesOK || !noOK || new(big.Rat).Add(yes, no).Cmp(big.NewRat(1, 1)) != 0 {
		return status.Error(codes.Internal, "Worm Trading returned a non-complementary market last-trade price pair")
	}
	return nil
}

func parseWormOrderEventPrice(value string) (*big.Rat, bool) {
	if value == "" || len(value) > wormCatalogMaximumPriceLength || value != strings.TrimSpace(value) {
		return nil, false
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" || (len(parts) == 2 && parts[1] == "") {
		return nil, false
	}
	for _, part := range parts {
		for _, character := range part {
			if character < '0' || character > '9' {
				return nil, false
			}
		}
	}
	price, ok := new(big.Rat).SetString(value)
	if !ok || price.Sign() < 0 || price.Cmp(big.NewRat(1, 1)) > 0 {
		return nil, false
	}
	return price, true
}

func projectWormCombination(
	combination *wormtradingapiclient.MarketCombination,
	expectedOwnerAccountID string,
	expectedCombinationID string,
	expectedRevision int64,
) (wormCombinationResponse, error) {
	if combination == nil {
		return wormCombinationResponse{}, status.Error(codes.Internal, "Worm Trading returned an empty market combination")
	}
	ownerAccountID, err := accountcredentials.CanonicalAccountID(combination.GetOwnerAccountId())
	if err != nil || ownerAccountID != combination.GetOwnerAccountId() || ownerAccountID != expectedOwnerAccountID {
		return wormCombinationResponse{}, status.Error(codes.Internal, "Worm Trading returned a mismatched market combination owner")
	}
	id, err := canonicalWormCombinationID(combination.GetId())
	if err != nil || id != combination.GetId() {
		return wormCombinationResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid market combination ID")
	}
	if expectedCombinationID != "" && id != expectedCombinationID {
		return wormCombinationResponse{}, status.Error(codes.Internal, "Worm Trading returned a mismatched market combination ID")
	}
	name := strings.TrimSpace(combination.GetName())
	if err := validateWormCombinationName(name); err != nil || name != combination.GetName() || combination.GetRevision() <= 0 || combination.GetCreatedAt() <= 0 || combination.GetUpdatedAt() < combination.GetCreatedAt() {
		return wormCombinationResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid market combination")
	}
	if expectedRevision > 0 && combination.GetRevision() != expectedRevision {
		return wormCombinationResponse{}, status.Error(codes.Internal, "Worm Trading returned a mismatched market combination revision")
	}
	items := combination.GetItems()
	if len(items) == 0 {
		return wormCombinationResponse{}, status.Error(codes.Internal, "Worm Trading returned a market combination without items")
	}
	response := wormCombinationResponse{
		ID:        id,
		Name:      name,
		Revision:  combination.GetRevision(),
		Items:     make([]wormCombinationResponseItem, 0, len(items)),
		CreatedAt: combination.GetCreatedAt(),
		UpdatedAt: combination.GetUpdatedAt(),
	}
	seenMarkets := make(map[string]struct{}, len(items))
	for index, item := range items {
		if item == nil || item.GetOrdinal() != int32(index+1) {
			return wormCombinationResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid market combination order")
		}
		eventConditionID, eventErr := canonicalWormConditionID(item.GetEventConditionId(), "event condition ID")
		marketConditionID, marketErr := canonicalWormConditionID(item.GetMarketConditionId(), "market condition ID")
		if eventErr != nil || marketErr != nil || eventConditionID != item.GetEventConditionId() || marketConditionID != item.GetMarketConditionId() {
			return wormCombinationResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid market combination item ID")
		}
		if _, exists := seenMarkets[marketConditionID]; exists {
			return wormCombinationResponse{}, status.Error(codes.Internal, "Worm Trading returned a duplicate market combination item")
		}
		seenMarkets[marketConditionID] = struct{}{}
		eventTitle := strings.TrimSpace(item.GetEventTitle())
		marketTitle := strings.TrimSpace(item.GetMarketTitle())
		outcomeLabel := strings.TrimSpace(item.GetOutcomeLabel())
		if eventTitle == "" || marketTitle == "" || outcomeLabel == "" {
			return wormCombinationResponse{}, status.Error(codes.Internal, "Worm Trading returned an incomplete market combination snapshot")
		}
		side := "NO"
		if item.GetIsYes() {
			side = "YES"
		}
		response.Items = append(response.Items, wormCombinationResponseItem{
			Ordinal:           item.GetOrdinal(),
			EventConditionID:  eventConditionID,
			EventTitle:        eventTitle,
			EventLogo:         strings.TrimSpace(item.GetEventLogo()),
			MarketConditionID: marketConditionID,
			MarketTitle:       marketTitle,
			MarketLogo:        strings.TrimSpace(item.GetMarketLogo()),
			Side:              side,
			OutcomeLabel:      outcomeLabel,
		})
	}
	return response, nil
}

func projectWormCombinationList(
	result *wormtradingapiclient.ListMarketCombinationsResponse,
	expectedPage int32,
	expectedPageSize int32,
	expectedOwnerAccountID string,
) (wormCombinationListResponse, error) {
	if result == nil || result.GetPage() != expectedPage || result.GetPageSize() != expectedPageSize || result.GetTotal() < 0 || len(result.GetItems()) > int(expectedPageSize) {
		return wormCombinationListResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid market combination page")
	}
	offset := int64(expectedPage-1) * int64(expectedPageSize)
	remaining := result.GetTotal() - offset
	if remaining < 0 {
		remaining = 0
	}
	expectedCount := int64(expectedPageSize)
	if remaining < expectedCount {
		expectedCount = remaining
	}
	if int64(len(result.GetItems())) != expectedCount {
		return wormCombinationListResponse{}, status.Error(codes.Internal, "Worm Trading returned an inconsistent market combination page")
	}
	response := wormCombinationListResponse{
		Items:    make([]wormCombinationResponse, 0, len(result.GetItems())),
		Total:    result.GetTotal(),
		Page:     expectedPage,
		PageSize: expectedPageSize,
	}
	seenIDs := make(map[string]struct{}, len(result.GetItems()))
	for _, combination := range result.GetItems() {
		projected, err := projectWormCombination(combination, expectedOwnerAccountID, "", 0)
		if err != nil {
			return wormCombinationListResponse{}, err
		}
		if _, exists := seenIDs[projected.ID]; exists {
			return wormCombinationListResponse{}, status.Error(codes.Internal, "Worm Trading returned duplicate market combination IDs")
		}
		seenIDs[projected.ID] = struct{}{}
		response.Items = append(response.Items, projected)
	}
	return response, nil
}

func decodeWormCombinationInput(w http.ResponseWriter, request *http.Request, requireRevision bool) (wormCombinationInput, error) {
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return wormCombinationInput{}, status.Error(codes.InvalidArgument, "Content-Type must be application/json")
	}
	request.Body = http.MaxBytesReader(w, request.Body, wormCombinationMaximumBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var input wormCombinationInput
	if err := decoder.Decode(&input); err != nil {
		return wormCombinationInput{}, status.Error(codes.InvalidArgument, "request body must be one valid JSON object")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return wormCombinationInput{}, status.Error(codes.InvalidArgument, "request body must contain exactly one JSON object")
	}
	input.Name = strings.TrimSpace(input.Name)
	if err := validateWormCombinationName(input.Name); err != nil {
		return wormCombinationInput{}, err
	}
	if requireRevision {
		if input.ExpectedRevision <= 0 || input.ExpectedRevision == math.MaxInt64 {
			return wormCombinationInput{}, status.Error(codes.InvalidArgument, "expectedRevision must be a positive incrementable integer")
		}
	} else if input.ExpectedRevision != 0 {
		return wormCombinationInput{}, status.Error(codes.InvalidArgument, "expectedRevision is not allowed when creating a combination")
	}
	return input, nil
}

func validateWormCombinationName(name string) error {
	if name == "" || utf8.RuneCountInString(name) > wormCombinationMaximumNameRunes || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return status.Errorf(codes.InvalidArgument, "name must contain between 1 and %d visible characters", wormCombinationMaximumNameRunes)
	}
	return nil
}

func wormCombinationPagination(request *http.Request) (int32, int32, error) {
	query := request.URL.Query()
	for key := range query {
		if key != "page" && key != "pageSize" {
			return 0, 0, status.Errorf(codes.InvalidArgument, "unsupported query parameter %q", key)
		}
	}
	page, err := positiveWormConnectionQueryValue(query["page"], "page", 1, 0)
	if err != nil {
		return 0, 0, err
	}
	pageSize, err := positiveWormConnectionQueryValue(query["pageSize"], "pageSize", wormCombinationDefaultPageSize, wormCombinationMaximumPageSize)
	if err != nil {
		return 0, 0, err
	}
	return page, pageSize, nil
}

func wormCombinationDeleteRevision(request *http.Request) (int64, error) {
	query := request.URL.Query()
	for key := range query {
		if key != "expectedRevision" {
			return 0, status.Errorf(codes.InvalidArgument, "unsupported query parameter %q", key)
		}
	}
	values := query["expectedRevision"]
	if len(values) != 1 || values[0] == "" || values[0] != strings.TrimSpace(values[0]) {
		return 0, status.Error(codes.InvalidArgument, "expectedRevision is required")
	}
	revision, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil || revision <= 0 {
		return 0, status.Error(codes.InvalidArgument, "expectedRevision must be a positive integer")
	}
	return revision, nil
}

func rejectWormCombinationQuery(request *http.Request) error {
	for key := range request.URL.Query() {
		return status.Errorf(codes.InvalidArgument, "unsupported query parameter %q", key)
	}
	return nil
}

func canonicalWormConditionID(value, label string) (string, error) {
	value = strings.TrimSpace(value)
	publicKey, err := solana.PublicKeyFromBase58(value)
	if err != nil || publicKey.String() != value {
		return "", status.Errorf(codes.InvalidArgument, "%s must be a canonical Solana public key", label)
	}
	return value, nil
}

func canonicalWormCombinationID(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil || parsed.String() != value {
		return "", status.Error(codes.InvalidArgument, "market combination ID must be a canonical non-zero UUID")
	}
	return value, nil
}

func (server *AthenaServer) wormCombinationClient() (wormtradingapiclient.WormTradingServiceClient, error) {
	if server.WormTradingClientset == nil || server.WormTradingClientset.WormTrading() == nil {
		return nil, status.Error(codes.Unavailable, "Worm Trading is unavailable")
	}
	return server.WormTradingClientset.WormTrading(), nil
}

func sanitizeWormCatalogDependencyError(err error) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.InvalidArgument, codes.NotFound, codes.Canceled, codes.DeadlineExceeded, codes.Unauthenticated, codes.PermissionDenied:
		return err
	case codes.Unavailable, codes.FailedPrecondition:
		return status.Error(codes.Unavailable, "Worm Trading is unavailable")
	default:
		return status.Error(codes.Internal, "Worm Trading request failed")
	}
}

func sanitizeWormCombinationStoreError(err error) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.Aborted:
		return err
	case codes.Canceled, codes.DeadlineExceeded, codes.Unavailable, codes.FailedPrecondition, codes.Unauthenticated, codes.PermissionDenied:
		return status.Error(codes.Unavailable, "Worm Trading is unavailable")
	default:
		return status.Error(codes.Internal, "Worm Trading request failed")
	}
}

func writeWormCombinationJSON(w http.ResponseWriter, statusCode int, value any) {
	walletsecret.SetSecretResponseHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}

// withWormTradingAccountIdentity replaces all untrusted identity values while
// retaining internal tracing and authentication metadata.
func withWormTradingAccountIdentity(ctx context.Context, accountID string) context.Context {
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Set("x-athena-account-id", accountID)
	return metadata.NewOutgoingContext(ctx, md)
}
