package worm

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// 59mJJLBC22xe2Bg9mTozn47fYdwfeDkwswE9t8RFnmrNmn3Lr6bf3Abo8ua4GUpFdaEnikfLhrhAfkykWWwyoejN
const (
	livePositionRequestGateEnv = "ATHENA_WORM_LIVE_POSITION_REQUEST"
	livePositionPrivateKeyEnv  = "ATHENA_WORM_PRIVATE_KEY"
	livePositionBaseURLEnv     = "ATHENA_WORM_API_BASE_URL"

	liveSpainEventConditionID  = "DhuB8Qdh7GTCU5cwLAG5LL8jJistoPmrePatRL7wxqSR"
	liveSpainMarketConditionID = "4zzkSGY7btrwXxS1yesufd3znkpsyRJmV4TPXCVxxGhA"
	liveSpainFunds             = "5"
	liveSpainLeverage          = 2.0
	liveSpainExpectedOutcome   = "spain"

	livePositionActiveRequestStates       = "created,funding_processing,processing,order_placed,refund_processing"
	livePositionActiveRequestLimit        = 20
	livePositionRequestTimeout            = 3 * time.Minute
	livePositionRequestPollInterval       = 3 * time.Second
	livePositionPostSubmitErrorPolls      = 5
	livePositionPostSubmitErrorPollPeriod = 3 * time.Second
)

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

	privateKey, err := parseSolanaPrivateKey(privateKeyText)
	if err != nil {
		t.Fatalf("parse %s: %v", livePositionPrivateKeyEnv, err)
	}

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
	if draft.Message == nil || strings.TrimSpace(*draft.Message) == "" {
		t.Fatalf("create Spain YES position request %s returned empty message", draft.Pubkey)
	}
	t.Logf("created position request pubkey=%s state=%s market_title=%q", draft.Pubkey, draft.State, market.Title)

	current, err := refreshLivePositionRequest(ctx, client, draft.Pubkey)
	if err != nil {
		t.Fatalf("refresh created Spain YES position request %s: %v", draft.Pubkey, err)
	}
	logLivePositionRequest(t, "created", current)

	message := livePositionRequestMessage(current)
	if message == "" {
		t.Fatalf("created Spain YES position request %s returned empty refreshed message; request=%s", draft.Pubkey, formatLivePositionRequest(current))
	}

	switch livePositionRequestState(current) {
	case "created", "funding_processing":
	case "processing", "order_placed", "completed":
		finalRequest, err := waitForLivePositionRequestCompleted(ctx, client, current.Pubkey)
		if err != nil {
			t.Fatalf("position request %s advanced before submit but did not complete: %v", current.Pubkey, err)
		}
		logLivePositionRequest(t, "completed", finalRequest)
		return
	case "failed", "cancelled", "refund_processing":
		t.Fatalf("position request reached terminal state before submit: %s", formatLivePositionRequest(current))
	default:
		t.Fatalf("position request reached unexpected state before submit: %s", formatLivePositionRequest(current))
	}

	signature := ed25519.Sign(privateKey, []byte(message))
	submitted, err := client.SubmitPositionRequest(ctx, current.Pubkey, SubmitSignatureRequest{
		Signature: hex.EncodeToString(signature),
	})
	if err != nil {
		latest, refreshErr := waitForLivePositionRequestPostSubmitError(ctx, client, current.Pubkey)
		if refreshErr != nil {
			t.Fatalf("submit Spain YES position request %s: %v; failed to refresh latest request state: %v", current.Pubkey, err, refreshErr)
		}
		logLivePositionRequest(t, "after submit error", latest)
		if shouldContinueAfterSubmitError(latest) {
			finalRequest, waitErr := waitForLivePositionRequestCompleted(ctx, client, latest.Pubkey)
			if waitErr != nil {
				t.Fatalf("submit Spain YES position request %s returned error %v, but request advanced; final wait failed: %v", latest.Pubkey, err, waitErr)
			}
			logLivePositionRequest(t, "completed", finalRequest)
			return
		}
		if shouldCancelAfterSubmitError(latest) {
			cancelLivePositionRequest(t, client, latest)
		}
		t.Fatalf("submit Spain YES position request %s: %v; latest=%s", current.Pubkey, err, formatLivePositionRequest(latest))
	}
	if submitted == nil {
		t.Fatalf("submit Spain YES position request %s returned nil", current.Pubkey)
	}
	logLivePositionRequest(t, "submitted", submitted)

	finalRequest, err := waitForLivePositionRequestCompleted(ctx, client, current.Pubkey)
	if err != nil {
		t.Fatalf("position request %s did not complete: %v", current.Pubkey, err)
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

func livePositionRequestState(request *PositionRequest) string {
	if request == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(request.State))
}

func livePositionRequestMessage(request *PositionRequest) string {
	if request == nil || request.Message == nil {
		return ""
	}
	return strings.TrimSpace(*request.Message)
}

func shouldContinueAfterSubmitError(request *PositionRequest) bool {
	switch livePositionRequestState(request) {
	case "processing", "order_placed", "completed":
		return true
	default:
		return hasLivePositionRequestTx(request)
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

func hasLivePositionRequestTx(request *PositionRequest) bool {
	if request == nil {
		return false
	}
	return hasOptionalString(request.FundingTxID) || hasOptionalString(request.RefundTxID)
}

func hasOptionalString(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}

func formatLivePositionRequest(request *PositionRequest) string {
	if request == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"pubkey=%s type=%s state=%s funding_txid=%s refund_txid=%s message_present=%t funds=%s price=%s shares=%s",
		request.Pubkey,
		request.Type,
		request.State,
		optionalString(request.FundingTxID),
		optionalString(request.RefundTxID),
		request.Message != nil && strings.TrimSpace(*request.Message) != "",
		request.Funds,
		optionalString(request.Price),
		optionalString(request.Shares),
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
