package worm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	livePositionRequestGateEnv = "ATHENA_WORM_LIVE_POSITION_REQUEST"
	livePositionPrivateKeyEnv  = "ATHENA_WORM_PRIVATE_KEY"
	livePositionBaseURLEnv     = "ATHENA_WORM_API_BASE_URL"

	liveHoldingsGateEnv          = "ATHENA_WORM_LIVE_HOLDINGS"
	liveHoldingsEventIDEnv       = "ATHENA_WORM_HOLDINGS_EVENT_ID"
	liveHoldingsEventConditionID = "FrYrqZ65N7QncdNexPoiGh8AzL2drrmWmdgpVHRtDd1g"
	liveHoldingsLimit            = 100
	liveHoldingsTimeout          = 45 * time.Second

	liveSpainEventConditionID  = "CiPGTY2jcxDBbicstS9bN7xVynVhT1E6SD3YgTfZ3yDN"
	liveSpainMarketConditionID = "6iabtaiF6FrGgX11gGSnTcADTM375z2vxt4eXhZtH2kd"
	liveSpainFunds             = "5"
	liveSpainLeverage          = 2.0
	liveSpainExpectedOutcome   = "belgium"

	livePositionActiveRequestStates       = "created,funding_processing,processing,order_placed,refund_processing"
	livePositionActiveRequestLimit        = 20
	livePositionShareAssetLimit           = 100
	livePositionRequestTimeout            = 3 * time.Minute
	livePositionRequestPollInterval       = 3 * time.Second
	livePositionPostSubmitErrorPolls      = 5
	livePositionPostSubmitErrorPollPeriod = 3 * time.Second
)

func TestLiveListMarketHoldings(t *testing.T) {
	if strings.TrimSpace(os.Getenv(liveHoldingsGateEnv)) != "1" {
		t.Skipf("set %s=1 to run the live Worm holdings test", liveHoldingsGateEnv)
	}

	privateKeyText := strings.TrimSpace(os.Getenv(livePositionPrivateKeyEnv))
	if privateKeyText == "" {
		t.Fatalf("%s is required for the live Worm holdings test", livePositionPrivateKeyEnv)
	}

	baseURL := strings.TrimSpace(os.Getenv(livePositionBaseURLEnv))
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), liveHoldingsTimeout)
	defer cancel()

	bootstrapClient, err := NewClient(Config{BaseURL: baseURL})
	if err != nil {
		t.Fatalf("create bootstrap client: %v", err)
	}

	creds, err := bootstrapClient.CreateAPIKeyFromPrivateKey(ctx, privateKeyText)
	if err != nil {
		t.Fatalf("create Worm API key from private key: %v", err)
	}
	if creds == nil || strings.TrimSpace(creds.APIKey) == "" || strings.TrimSpace(creds.Secret) == "" {
		t.Fatalf("create Worm API key returned incomplete credentials")
	}

	client, err := NewClient(Config{
		BaseURL:   baseURL,
		APIKey:    creds.APIKey,
		APISecret: creds.Secret,
	})
	if err != nil {
		t.Fatalf("create authenticated client: %v", err)
	}
	defer revokeLivePositionRequestAPIKey(t, baseURL, creds)

	summary, err := client.GetAccountSummary(ctx)
	if err != nil {
		t.Fatalf("get Worm account summary for holdings: %v", err)
	}
	if summary == nil {
		t.Fatalf("get Worm account summary for holdings returned nil")
	}
	t.Logf("account summary %s", formatLiveAccountSummary(summary))

	eventID := liveHoldingsEventID()
	event, err := client.GetEvent(ctx, eventID)
	if err != nil {
		t.Fatalf("get holdings event %s: %s", eventID, formatWormError(err))
	}
	if event == nil {
		t.Fatalf("get holdings event %s returned nil", eventID)
	}
	t.Logf("holdings event %s", formatLiveHoldingsEvent(event))
	t.Logf("event markets count=%d markets=%s", len(event.Markets), formatLiveHoldingsEventMarkets(event.Markets))

	eventMarkets := liveEventMarketsByConditionID(event)
	if len(eventMarkets) == 0 {
		t.Fatalf("holdings event %s returned no markets", eventID)
	}

	pnl, err := client.GetAccountPnL(ctx, GetAccountPnLOptions{
		EventConditionID: eventID,
	})
	if err != nil {
		t.Logf("get Worm account PnL for event %s failed: %s", eventID, formatWormError(err))
	} else if pnl == nil {
		t.Logf("get Worm account PnL for event %s returned nil", eventID)
	} else {
		t.Logf("event account pnl %s", formatLiveAccountPnL(pnl))
	}

	assets, err := listLiveHoldingsAssets(ctx, t, client, eventMarkets)
	if err != nil {
		t.Fatalf("list Worm spot share holdings for event %s: %s", eventID, formatWormError(err))
	}
	if len(assets) == 0 {
		t.Logf("event spot share holdings for event %s: none", eventID)
	} else {
		t.Logf("event spot share holdings count=%d holdings=%s", len(assets), formatLiveHoldingsAssetsByMarket(assets, event.Markets))
	}

	positions, err := listLiveOpenMarginPositions(ctx, t, client, eventMarkets)
	if err != nil {
		t.Fatalf("list Worm open margin positions for event %s: %s", eventID, formatWormError(err))
	}
	if len(positions) == 0 {
		t.Logf("event open margin positions for event %s: none", eventID)
		return
	}
	t.Logf("event open margin positions count=%d positions=%s", len(positions), formatLiveMarginPositionsByMarket(positions, event.Markets))
}

func TestLiveCreateAndSubmitSpainMarginPosition(t *testing.T) {
	if strings.TrimSpace(os.Getenv(livePositionRequestGateEnv)) != "1" {
		t.Skipf("set %s=1 to run the live Worm position request test", livePositionRequestGateEnv)
	}

	privateKeyText := strings.TrimSpace(os.Getenv(livePositionPrivateKeyEnv))
	if privateKeyText == "" {
		t.Fatalf("%s is required for the live Worm position request test", livePositionPrivateKeyEnv)
	}

	baseURL := strings.TrimSpace(os.Getenv(livePositionBaseURLEnv))
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), livePositionRequestTimeout)
	defer cancel()

	bootstrapClient, err := NewClient(Config{BaseURL: baseURL})
	if err != nil {
		t.Fatalf("create bootstrap client: %v", err)
	}

	creds, err := bootstrapClient.CreateAPIKeyFromPrivateKey(ctx, privateKeyText)
	if err != nil {
		t.Fatalf("create Worm API key from private key: %v", err)
	}
	if creds == nil || strings.TrimSpace(creds.APIKey) == "" || strings.TrimSpace(creds.Secret) == "" {
		t.Fatalf("create Worm API key returned incomplete credentials")
	}

	client, err := NewClient(Config{
		BaseURL:     baseURL,
		APIKey:      creds.APIKey,
		APISecret:   creds.Secret,
		RateLimiter: nil,
	})
	if err != nil {
		t.Fatalf("create authenticated client: %v", err)
	}
	defer revokeLivePositionRequestAPIKey(t, baseURL, creds)

	event, err := client.GetEvent(ctx, liveSpainEventConditionID)
	if err != nil {
		t.Fatalf("get event %s: %v", liveSpainEventConditionID, err)
	}
	if event == nil {
		t.Fatalf("get event %s returned nil", liveSpainEventConditionID)
	}
	if event.ConditionID != liveSpainEventConditionID {
		t.Fatalf("event condition mismatch: got %q, want %q", event.ConditionID, liveSpainEventConditionID)
	}
	if !eventContainsMarket(event, liveSpainMarketConditionID) {
		t.Fatalf("event %s does not contain market %s", liveSpainEventConditionID, liveSpainMarketConditionID)
	}

	market, err := client.GetMarket(ctx, liveSpainMarketConditionID)
	if err != nil {
		t.Fatalf("get market %s: %v", liveSpainMarketConditionID, err)
	}
	if market == nil {
		t.Fatalf("get market %s returned nil", liveSpainMarketConditionID)
	}
	if market.ConditionID != liveSpainMarketConditionID {
		t.Fatalf("market condition mismatch: got %q, want %q", market.ConditionID, liveSpainMarketConditionID)
	}
	if !marketYesSideMentions(market, liveSpainExpectedOutcome) {
		t.Fatalf("market %s does not appear to target Spain YES side: title=%q yes_label=%s outcomes=%v", liveSpainMarketConditionID, market.Title, optionalString(market.YesOutcomeLabel), market.Outcomes)
	}
	if !market.MarginEnabled {
		t.Fatalf("market %s is not margin enabled", liveSpainMarketConditionID)
	}
	if market.Config == nil {
		t.Fatalf("market %s has no trading config", liveSpainMarketConditionID)
	}
	if market.Config.MaxLeverageYes == nil {
		t.Fatalf("market %s has no max_leverage_yes", liveSpainMarketConditionID)
	}
	maxLeverageYes, err := strconv.ParseFloat(strings.TrimSpace(*market.Config.MaxLeverageYes), 64)
	if err != nil {
		t.Fatalf("parse market %s max_leverage_yes %q: %v", liveSpainMarketConditionID, *market.Config.MaxLeverageYes, err)
	}
	if maxLeverageYes < liveSpainLeverage {
		t.Fatalf("market %s max_leverage_yes=%s is lower than requested leverage %.2f", liveSpainMarketConditionID, *market.Config.MaxLeverageYes, liveSpainLeverage)
	}

	isYes := true
	leverage := liveSpainLeverage
	estimate, err := client.EstimateMarginPosition(ctx, EstimateMarginPositionOptions{
		MarketConditionID: liveSpainMarketConditionID,
		Funds:             liveSpainFunds,
		IsYes:             &isYes,
		Leverage:          &leverage,
	})
	if err != nil {
		t.Fatalf("estimate Spain YES margin position: %v", err)
	}
	if estimate == nil {
		t.Fatalf("estimate Spain YES margin position returned nil")
	}
	t.Logf(
		"estimate market=%s funds=%s leverage=%.2f average_price=%s total_shares=%s total_cost=%s fee=%s funds_needed=%s liquidation=%s fully_filled=%t",
		liveSpainMarketConditionID,
		liveSpainFunds,
		liveSpainLeverage,
		estimate.AveragePrice,
		estimate.TotalShares,
		estimate.TotalCost,
		estimate.FeeAmount,
		estimate.UserFundsNeeded,
		optionalString(estimate.LiquidationPrice),
		estimate.IsFullyFilled,
	)
	if !estimate.IsFullyFilled {
		t.Fatalf("estimate did not fully fill requested notional funds=%s leverage=%.2f; skip live create to avoid unstable market position request", liveSpainFunds, liveSpainLeverage)
	}

	logLiveAccountDiagnostics(ctx, t, client)
	requireNoLiveActivePositionRequests(ctx, t, client, isYes, leverage)

	draft, err := client.CreatePositionRequest(ctx, CreatePositionRequestRequest{
		Type:              "market",
		MarketConditionID: liveSpainMarketConditionID,
		IsYes:             &isYes,
		Leverage:          &leverage,
		Funds:             liveSpainFunds,
	})
	if err != nil {
		t.Fatalf("create Spain YES position request: %v", err)
	}
	if draft == nil {
		t.Fatalf("create Spain YES position request returned nil")
	}
	if strings.TrimSpace(draft.Pubkey) == "" {
		t.Fatalf("create Spain YES position request returned empty pubkey")
	}
	t.Logf("created position request pubkey=%s state=%s market_title=%q", draft.Pubkey, draft.State, market.Title)
	logLivePositionRequest(t, "created", draft)

	switch livePositionRequestState(draft) {
	case "processing", "order_placed", "completed":
		finalRequest, err := waitForLivePositionRequestCompleted(ctx, client, draft.Pubkey)
		if err != nil {
			t.Fatalf("position request %s advanced before local submit but did not complete: %v", draft.Pubkey, err)
		}
		logLivePositionRequest(t, "completed", finalRequest)
		return
	case "failed", "cancelled", "refund_processing":
		t.Fatalf("position request reached terminal state before submit: %s", formatLivePositionRequest(draft))
	}

	draftMessage := livePositionRequestRawMessage(draft)
	if strings.TrimSpace(draftMessage) == "" {
		cancelLivePositionRequest(t, client, draft)
		t.Fatalf("create Spain YES position request %s returned empty message; request=%s", draft.Pubkey, formatLivePositionRequest(draft))
	}
	signedMessage, err := signPositionRequestMessage(privateKeyText, draftMessage)
	if err != nil {
		cancelLivePositionRequest(t, client, draft)
		t.Fatalf(
			"sign Spain YES position request %s: %v; create_message=%s request=%s",
			draft.Pubkey,
			err,
			formatLiveMessageFingerprint(draftMessage),
			formatLivePositionRequest(draft),
		)
	}
	t.Logf(
		"signed position request transaction version=%s required_signatures=%d signer_index=%d signer=%s create_message=%s",
		signedMessage.transactionVersion,
		signedMessage.requiredSignatures,
		signedMessage.signerIndex,
		signedMessage.signerPublicKey,
		formatLiveMessageFingerprint(draftMessage),
	)

	submitted, err := client.SubmitPositionRequest(ctx, draft.Pubkey, SubmitSignatureRequest{
		Signature: signedMessage.signatureHex,
	})
	if err != nil {
		latest, refreshErr := waitForLivePositionRequestPostSubmitError(ctx, client, draft.Pubkey)
		if refreshErr != nil {
			t.Fatalf(
				"submit Spain YES position request %s: %s; failed to refresh latest request state: %v; create_message=%s",
				draft.Pubkey,
				formatWormError(err),
				refreshErr,
				formatLiveMessageFingerprint(draftMessage),
			)
		}
		logLivePositionRequest(t, "after submit error", latest)
		if shouldContinueAfterSubmitError(latest) {
			finalRequest, waitErr := waitForLivePositionRequestCompleted(ctx, client, latest.Pubkey)
			if waitErr != nil {
				t.Fatalf(
					"submit Spain YES position request %s returned error %s, but request advanced; final wait failed: %v; create_message=%s latest=%s",
					latest.Pubkey,
					formatWormError(err),
					waitErr,
					formatLiveMessageFingerprint(draftMessage),
					formatLivePositionRequest(latest),
				)
			}
			logLivePositionRequest(t, "completed", finalRequest)
			return
		}
		if shouldCancelAfterSubmitError(latest) {
			cancelLivePositionRequest(t, client, latest)
		}
		t.Fatalf(
			"submit Spain YES position request %s: %s; create_message=%s latest=%s",
			draft.Pubkey,
			formatWormError(err),
			formatLiveMessageFingerprint(draftMessage),
			formatLivePositionRequest(latest),
		)
	}
	if submitted == nil {
		t.Fatalf("submit Spain YES position request %s returned nil", draft.Pubkey)
	}
	logLivePositionRequest(t, "submitted", submitted)

	finalRequest, err := waitForLivePositionRequestCompleted(ctx, client, draft.Pubkey)
	if err != nil {
		t.Fatalf("position request %s did not complete: %v", draft.Pubkey, err)
	}
	t.Logf(
		"completed position request pubkey=%s state=%s funds=%s price=%s shares=%s",
		finalRequest.Pubkey,
		finalRequest.State,
		finalRequest.Funds,
		optionalString(finalRequest.Price),
		optionalString(finalRequest.Shares),
	)
}

func eventContainsMarket(event *Event, conditionID string) bool {
	if event == nil {
		return false
	}
	for _, market := range event.Markets {
		if market.ConditionID == conditionID {
			return true
		}
	}
	return false
}

func marketYesSideMentions(market *Market, expected string) bool {
	if market == nil {
		return false
	}
	expected = strings.ToLower(strings.TrimSpace(expected))
	if expected == "" {
		return false
	}

	candidates := []string{market.Title}
	if market.YesOutcomeLabel != nil {
		candidates = append(candidates, *market.YesOutcomeLabel)
	}
	for _, outcome := range market.Outcomes {
		if outcome.IsYes {
			candidates = append(candidates, outcome.Text)
		}
	}
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate), expected) {
			return true
		}
	}
	return false
}

func logLiveAccountDiagnostics(ctx context.Context, t *testing.T, client Client) {
	t.Helper()

	summary, err := client.GetAccountSummary(ctx)
	if err != nil {
		t.Fatalf("get Worm account summary before live position request: %v", err)
	}
	if summary == nil {
		t.Fatalf("get Worm account summary before live position request returned nil")
	}
	t.Logf("account summary %s", formatLiveAccountSummary(summary))

	assets, err := client.ListAccountAssets(ctx, ListAccountAssetsOptions{
		PageOptions: PageOptions{
			Limit: livePositionShareAssetLimit,
		},
	})
	if err != nil {
		t.Logf("list Worm share assets before live position request failed: %v", err)
		return
	}
	if assets == nil {
		t.Logf("share assets response is nil before live position request")
		return
	}

	t.Logf("share assets count=%d assets=%s", len(assets.Assets), formatLiveAccountAssets(assets.Assets))
}

func liveHoldingsEventID() string {
	if value := strings.TrimSpace(os.Getenv(liveHoldingsEventIDEnv)); value != "" {
		return value
	}
	return liveHoldingsEventConditionID
}

func liveEventMarketsByConditionID(event *Event) map[string]MarketSummary {
	markets := make(map[string]MarketSummary)
	if event == nil {
		return markets
	}
	for _, market := range event.Markets {
		conditionID := strings.TrimSpace(market.ConditionID)
		if conditionID != "" {
			markets[conditionID] = market
		}
	}
	return markets
}

func listLiveHoldingsAssets(ctx context.Context, t *testing.T, client Client, eventMarkets map[string]MarketSummary) ([]AccountAsset, error) {
	t.Helper()

	var filtered []AccountAsset
	var cursor string
	for {
		assets, err := client.ListAccountAssets(ctx, ListAccountAssetsOptions{
			PageOptions: PageOptions{
				Limit:  liveHoldingsLimit,
				Cursor: cursor,
			},
		})
		if err != nil {
			return nil, err
		}
		if assets == nil {
			return nil, fmt.Errorf("account assets response is nil")
		}

		for _, asset := range assets.Assets {
			if liveAccountAssetMatchesEvent(asset, eventMarkets) {
				filtered = append(filtered, asset)
			}
		}
		if assets.Meta.NextCursor == nil || strings.TrimSpace(*assets.Meta.NextCursor) == "" {
			break
		}
		cursor = strings.TrimSpace(*assets.Meta.NextCursor)
	}
	return filtered, nil
}

func listLiveOpenMarginPositions(ctx context.Context, t *testing.T, client Client, eventMarkets map[string]MarketSummary) ([]MarginPosition, error) {
	t.Helper()

	isClosed := false
	var filtered []MarginPosition
	var cursor string
	for {
		positions, err := client.ListMarginPositions(ctx, ListMarginPositionsOptions{
			PageOptions: PageOptions{
				Limit:  liveHoldingsLimit,
				Cursor: cursor,
			},
			IsClosed: &isClosed,
			Sort:     "-created",
		})
		if err != nil {
			return nil, err
		}
		if positions == nil {
			return nil, fmt.Errorf("margin positions response is nil")
		}

		for _, position := range positions.Positions {
			if liveMarginPositionMatchesEvent(position, eventMarkets) {
				filtered = append(filtered, position)
			}
		}
		if positions.Meta.NextCursor == nil || strings.TrimSpace(*positions.Meta.NextCursor) == "" {
			break
		}
		cursor = strings.TrimSpace(*positions.Meta.NextCursor)
	}
	return filtered, nil
}

func liveAccountAssetMatchesEvent(asset AccountAsset, eventMarkets map[string]MarketSummary) bool {
	conditionID := liveAccountAssetMarketConditionID(asset)
	if conditionID == "" {
		return false
	}
	_, ok := eventMarkets[conditionID]
	return ok
}

func liveAccountAssetMarketConditionID(asset AccountAsset) string {
	if conditionID := strings.TrimSpace(asset.Position.Market.ConditionID); conditionID != "" {
		return conditionID
	}
	symbol := strings.TrimSpace(asset.Token.Symbol)
	for _, suffix := range []string{"-LPT", "-SPT"} {
		if strings.HasSuffix(symbol, suffix) {
			return strings.TrimSuffix(symbol, suffix)
		}
	}
	return ""
}

func liveMarginPositionMatchesEvent(position MarginPosition, eventMarkets map[string]MarketSummary) bool {
	_, ok := eventMarkets[strings.TrimSpace(position.Market.ConditionID)]
	return ok
}

func requireNoLiveActivePositionRequests(ctx context.Context, t *testing.T, client Client, isYes bool, leverage float64) {
	t.Helper()

	requests, err := client.ListPositionRequests(ctx, ListPositionRequestsOptions{
		PageOptions: PageOptions{
			Limit: livePositionActiveRequestLimit,
		},
		MarketConditionID: liveSpainMarketConditionID,
		States:            livePositionActiveRequestStates,
		IsYes:             &isYes,
		Leverage:          &leverage,
		Sort:              "-created",
	})
	if err != nil {
		t.Fatalf("list active Spain YES position requests: %v", err)
	}
	if requests == nil || len(requests.Requests) == 0 {
		return
	}

	for _, request := range requests.Requests {
		requestCopy := request
		logLivePositionRequest(t, "active existing", &requestCopy)
	}
	t.Fatalf("found %d active Spain YES position request(s); not creating another live request", len(requests.Requests))
}

func waitForLivePositionRequestCompleted(ctx context.Context, client Client, pubkey string) (*PositionRequest, error) {
	ticker := time.NewTicker(livePositionRequestPollInterval)
	defer ticker.Stop()

	var last *PositionRequest
	for {
		request, err := refreshLivePositionRequest(ctx, client, pubkey)
		if err != nil {
			return nil, err
		}
		if request != nil {
			last = request
			switch livePositionRequestState(request) {
			case "completed":
				return request, nil
			case "failed", "cancelled", "refund_processing":
				return nil, &livePositionRequestTerminalStateError{Request: request}
			}
		}

		select {
		case <-ctx.Done():
			if last != nil {
				return nil, &livePositionRequestTimeoutError{Pubkey: pubkey, State: last.State, Cause: ctx.Err()}
			}
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func waitForLivePositionRequestPostSubmitError(ctx context.Context, client Client, pubkey string) (*PositionRequest, error) {
	ticker := time.NewTicker(livePositionPostSubmitErrorPollPeriod)
	defer ticker.Stop()

	var last *PositionRequest
	for attempt := 0; attempt < livePositionPostSubmitErrorPolls; attempt++ {
		request, err := refreshLivePositionRequest(ctx, client, pubkey)
		if err != nil {
			return last, err
		}
		last = request
		if shouldContinueAfterSubmitError(request) || !shouldCancelAfterSubmitError(request) {
			return request, nil
		}

		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-ticker.C:
		}
	}

	return last, nil
}

func refreshLivePositionRequest(ctx context.Context, client Client, pubkey string) (*PositionRequest, error) {
	if strings.TrimSpace(pubkey) == "" {
		return nil, fmt.Errorf("position request pubkey is required")
	}
	request, err := client.GetPositionRequest(ctx, pubkey)
	if err != nil {
		return nil, err
	}
	if request == nil {
		return nil, fmt.Errorf("position request %s returned nil", pubkey)
	}
	return request, nil
}

func cancelLivePositionRequest(t *testing.T, client Client, request *PositionRequest) {
	t.Helper()
	if request == nil || strings.TrimSpace(request.Pubkey) == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cancelled, err := client.CancelPositionRequest(ctx, request.Pubkey)
	if err != nil {
		t.Logf("failed to cancel position request after %s: %v", formatLivePositionRequest(request), err)
		return
	}
	logLivePositionRequest(t, "cancelled", cancelled)
}

func logLivePositionRequest(t *testing.T, label string, request *PositionRequest) {
	t.Helper()
	t.Logf("%s position request %s", label, formatLivePositionRequest(request))
}

func revokeLivePositionRequestAPIKey(t *testing.T, baseURL string, creds *APIKeySecret) {
	t.Helper()
	if creds == nil || strings.TrimSpace(creds.APIKey) == "" || strings.TrimSpace(creds.Secret) == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := NewClient(Config{
		BaseURL:   baseURL,
		APIKey:    creds.APIKey,
		APISecret: creds.Secret,
	})
	if err != nil {
		t.Logf("failed to create revoke client for temporary Worm API key: %v", err)
		return
	}
	if _, err := client.RevokeAPIKey(ctx, creds.APIKey); err != nil {
		t.Logf("failed to revoke temporary Worm API key: %v", err)
	}
}

func optionalString(value *string) string {
	if value == nil {
		return "-"
	}
	return *value
}

func optionalInt64(value *int64) string {
	if value == nil {
		return "-"
	}
	return strconv.FormatInt(*value, 10)
}

func livePositionRequestState(request *PositionRequest) string {
	if request == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(request.State))
}

func livePositionRequestRawMessage(request *PositionRequest) string {
	if request == nil || request.Message == nil {
		return ""
	}
	return *request.Message
}

func shouldContinueAfterSubmitError(request *PositionRequest) bool {
	switch livePositionRequestState(request) {
	case "processing", "order_placed", "completed":
		return true
	default:
		return hasLivePositionRequestTx(request) || hasLivePositionRequestOrderProgress(request)
	}
}

func shouldCancelAfterSubmitError(request *PositionRequest) bool {
	switch livePositionRequestState(request) {
	case "created", "funding_processing":
		return !hasLivePositionRequestTx(request)
	default:
		return false
	}
}

func hasLivePositionRequestOrderProgress(request *PositionRequest) bool {
	if request == nil || request.OrderState == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(*request.OrderState)) {
	case "created", "open", "opened", "partially_filled", "filled":
		return true
	default:
		return false
	}
}

func hasLivePositionRequestTx(request *PositionRequest) bool {
	if request == nil {
		return false
	}
	return hasOptionalString(request.FundingTxID) || hasOptionalString(request.RefundTxID)
}

func hasOptionalString(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}

func positionRequestMarketConditionID(request *PositionRequest) string {
	if request == nil || request.Market == nil || strings.TrimSpace(request.Market.ConditionID) == "" {
		return "-"
	}
	return request.Market.ConditionID
}

func formatLivePositionRequest(request *PositionRequest) string {
	if request == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"pubkey=%s type=%s state=%s order_state=%s market_condition_id=%s is_yes=%t leverage=%s funding_txid=%s refund_txid=%s message_present=%t funds=%s price=%s shares=%s created=%s",
		request.Pubkey,
		request.Type,
		request.State,
		optionalString(request.OrderState),
		positionRequestMarketConditionID(request),
		request.IsYes,
		request.Leverage,
		optionalString(request.FundingTxID),
		optionalString(request.RefundTxID),
		request.Message != nil && strings.TrimSpace(*request.Message) != "",
		request.Funds,
		optionalString(request.Price),
		optionalString(request.Shares),
		optionalInt64(request.Created),
	)
}

func formatLiveAccountSummary(summary *AccountSummary) string {
	if summary == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"username=%s twitter=%s joined_at=%s",
		summary.Username,
		optionalString(summary.TwitterUsername),
		optionalInt64(summary.JoinedAt),
	)
}

func formatLiveAccountAssets(assets []AccountAsset) string {
	if len(assets) == 0 {
		return "[]"
	}

	const limit = 12
	formatted := make([]string, 0, min(len(assets), limit))
	for i, asset := range assets {
		if i >= limit {
			formatted = append(formatted, fmt.Sprintf("...(+%d more)", len(assets)-limit))
			break
		}
		formatted = append(formatted, formatLiveAccountAsset(asset))
	}
	return "[" + strings.Join(formatted, " ") + "]"
}

func formatLiveAccountAsset(asset AccountAsset) string {
	return fmt.Sprintf(
		"{symbol=%s kind=%s available=%s locked=%s total=%s value_usdt=%s value_basis=%s}",
		asset.Token.Symbol,
		asset.AssetKind,
		asset.Amounts.Available,
		asset.Amounts.Locked,
		asset.Amounts.Total,
		asset.Value.USDT,
		asset.Value.Basis,
	)
}

func formatLiveHoldingsEvent(event *Event) string {
	if event == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"condition_id=%s title=%q category=%s created=%s markets=%d",
		event.ConditionID,
		event.Title,
		event.Category,
		optionalInt64(event.Created),
		len(event.Markets),
	)
}

func formatLiveHoldingsEventMarkets(markets []MarketSummary) string {
	if len(markets) == 0 {
		return "[]"
	}

	formatted := make([]string, 0, len(markets))
	for _, market := range markets {
		formatted = append(formatted, formatLiveHoldingsMarketSummary(market))
	}
	return "[" + strings.Join(formatted, " ") + "]"
}

func formatLiveHoldingsMarketSummary(market MarketSummary) string {
	return fmt.Sprintf(
		"{condition_id=%s title=%q state=%s margin_enabled=%t last_trade_price=%s}",
		market.ConditionID,
		market.Title,
		market.State,
		market.MarginEnabled,
		optionalString(market.LastTradePrice),
	)
}

func formatLiveAccountPnL(pnl *AccountPnL) string {
	if pnl == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"margin_settlements=%s margin_unrealized=%s redeems_funds=%s active_assets_value=%s creator_fees=%s total=%s",
		pnl.MarginPositionSettlementsPnL,
		pnl.MarginPositionsUnrealizedPnL,
		pnl.RedeemsFunds,
		pnl.ActiveAssetsValue,
		pnl.CreatorFees,
		pnl.TotalPnL,
	)
}

func formatLiveHoldingsAssets(assets []AccountAsset) string {
	if len(assets) == 0 {
		return "[]"
	}

	formatted := make([]string, 0, len(assets))
	for _, asset := range assets {
		formatted = append(formatted, formatLiveHoldingsAsset(asset))
	}
	return "[" + strings.Join(formatted, " ") + "]"
}

func formatLiveHoldingsAssetsByMarket(assets []AccountAsset, markets []MarketSummary) string {
	if len(assets) == 0 {
		return "[]"
	}

	groups := make(map[string][]AccountAsset)
	for _, asset := range assets {
		conditionID := liveAccountAssetMarketConditionID(asset)
		groups[conditionID] = append(groups[conditionID], asset)
	}

	marketByID := liveMarketSummariesByConditionID(markets)
	formatted := make([]string, 0, len(groups))
	for _, conditionID := range orderedLiveHoldingMarketIDs(groups, markets) {
		market := marketByID[conditionID]
		formatted = append(formatted, fmt.Sprintf(
			"{market_condition_id=%s market_title=%q assets=%s}",
			conditionID,
			market.Title,
			formatLiveHoldingsAssets(groups[conditionID]),
		))
	}
	return "[" + strings.Join(formatted, " ") + "]"
}

func formatLiveHoldingsAsset(asset AccountAsset) string {
	return fmt.Sprintf(
		"{side=%s outcome=%q final=%t available=%s locked=%s total=%s value_usdt=%s value_basis=%s avg_trade_price=%s token=%s token_address=%s created=%s}",
		formatLiveSide(asset.Position.IsYes),
		asset.Position.OutcomeText,
		asset.Position.IsFinal,
		asset.Amounts.Available,
		asset.Amounts.Locked,
		asset.Amounts.Total,
		asset.Value.USDT,
		asset.Value.Basis,
		asset.Position.AvgTradePrice,
		asset.Token.Symbol,
		optionalString(asset.Token.Address),
		optionalInt64(asset.Created),
	)
}

func formatLiveMarginPositions(positions []MarginPosition) string {
	if len(positions) == 0 {
		return "[]"
	}

	formatted := make([]string, 0, len(positions))
	for _, position := range positions {
		formatted = append(formatted, formatLiveMarginPosition(position))
	}
	return "[" + strings.Join(formatted, " ") + "]"
}

func formatLiveMarginPositionsByMarket(positions []MarginPosition, markets []MarketSummary) string {
	if len(positions) == 0 {
		return "[]"
	}

	groups := make(map[string][]MarginPosition)
	for _, position := range positions {
		conditionID := strings.TrimSpace(position.Market.ConditionID)
		groups[conditionID] = append(groups[conditionID], position)
	}

	marketByID := liveMarketSummariesByConditionID(markets)
	formatted := make([]string, 0, len(groups))
	for _, conditionID := range orderedLivePositionMarketIDs(groups, markets) {
		market := marketByID[conditionID]
		formatted = append(formatted, fmt.Sprintf(
			"{market_condition_id=%s market_title=%q positions=%s}",
			conditionID,
			market.Title,
			formatLiveMarginPositions(groups[conditionID]),
		))
	}
	return "[" + strings.Join(formatted, " ") + "]"
}

func formatLiveMarginPosition(position MarginPosition) string {
	return fmt.Sprintf(
		"{pubkey=%s request=%s side=%s leverage=%s shares=%s avg_entry=%s liquidation=%s unrealized_pnl=%s realized_pnl=%s user_liquidity=%s total_liquidity=%s liquidated=%t claimed=%t tp_sl=%s created=%s market_title=%q}",
		position.Pubkey,
		optionalString(position.PositionRequestPubkey),
		formatLiveSide(position.IsYes),
		position.Leverage,
		position.TotalShares,
		position.AvgEntryPrice,
		position.LiquidationPrice,
		optionalString(position.UnrealizedPnL),
		position.RealizedPnL,
		position.UserLiquidity,
		position.TotalLiquidity,
		position.IsLiquidated,
		position.IsClaimed,
		formatLiveTPSL(position.TPSL),
		optionalInt64(position.Created),
		position.Market.Title,
	)
}

func formatLiveTPSL(tpsl *TPSL) string {
	if tpsl == nil {
		return "-"
	}
	return fmt.Sprintf(
		"{state=%s take_profit=%s stop_loss=%s trigger_type=%s triggered_price=%s triggered_at=%s}",
		tpsl.State,
		optionalString(tpsl.TakeProfitPrice),
		optionalString(tpsl.StopLossPrice),
		optionalString(tpsl.TriggerType),
		optionalString(tpsl.TriggeredPrice),
		optionalInt64(tpsl.TriggeredAt),
	)
}

func formatLiveSide(isYes bool) string {
	if isYes {
		return "YES"
	}
	return "NO"
}

func liveMarketSummariesByConditionID(markets []MarketSummary) map[string]MarketSummary {
	byID := make(map[string]MarketSummary)
	for _, market := range markets {
		if conditionID := strings.TrimSpace(market.ConditionID); conditionID != "" {
			byID[conditionID] = market
		}
	}
	return byID
}

func orderedLiveHoldingMarketIDs(groups map[string][]AccountAsset, markets []MarketSummary) []string {
	seen := make(map[string]bool)
	ordered := make([]string, 0, len(groups))
	for _, market := range markets {
		conditionID := strings.TrimSpace(market.ConditionID)
		if len(groups[conditionID]) > 0 {
			ordered = append(ordered, conditionID)
			seen[conditionID] = true
		}
	}
	return append(ordered, sortedLiveRemainingMarketIDs(groups, seen)...)
}

func orderedLivePositionMarketIDs(groups map[string][]MarginPosition, markets []MarketSummary) []string {
	seen := make(map[string]bool)
	ordered := make([]string, 0, len(groups))
	for _, market := range markets {
		conditionID := strings.TrimSpace(market.ConditionID)
		if len(groups[conditionID]) > 0 {
			ordered = append(ordered, conditionID)
			seen[conditionID] = true
		}
	}
	return append(ordered, sortedLiveRemainingMarketIDs(groups, seen)...)
}

func sortedLiveRemainingMarketIDs[T any](groups map[string][]T, seen map[string]bool) []string {
	remaining := make([]string, 0, len(groups))
	for conditionID := range groups {
		if !seen[conditionID] {
			remaining = append(remaining, conditionID)
		}
	}
	sort.Strings(remaining)
	return remaining
}

func formatLiveMessageFingerprint(message string) string {
	sum := sha256.Sum256([]byte(message))
	digest := hex.EncodeToString(sum[:])
	if len(digest) > 16 {
		digest = digest[:16]
	}
	return fmt.Sprintf("len=%d sha256=%s", len(message), digest)
}

func formatWormError(err error) string {
	if err == nil {
		return "<nil>"
	}

	var wormErr *Error
	if !errors.As(err, &wormErr) {
		return err.Error()
	}

	details := "-"
	if len(wormErr.Details) > 0 {
		rawDetails, marshalErr := json.Marshal(wormErr.Details)
		if marshalErr != nil {
			details = fmt.Sprintf("marshal_error=%v", marshalErr)
		} else {
			details = string(rawDetails)
		}
	}

	return fmt.Sprintf(
		"status=%s status_code=%d code=%d slug=%s message=%q details=%s",
		wormErr.Status,
		wormErr.StatusCode,
		wormErr.Code,
		wormErr.Slug,
		wormErr.Message,
		details,
	)
}

type livePositionRequestTerminalStateError struct {
	Request *PositionRequest
}

func (e *livePositionRequestTerminalStateError) Error() string {
	if e == nil || e.Request == nil {
		return "position request entered terminal state"
	}
	return "position request entered terminal state: " + formatLivePositionRequest(e.Request)
}

type livePositionRequestTimeoutError struct {
	Pubkey string
	State  string
	Cause  error
}

func (e *livePositionRequestTimeoutError) Error() string {
	if e == nil {
		return "timed out waiting for position request"
	}
	message := "timed out waiting for position request " + e.Pubkey
	if strings.TrimSpace(e.State) != "" {
		message += " in state " + e.State
	}
	if e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	return message
}
