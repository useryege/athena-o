package wormtrading

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"github.com/useryege/athena/util/worm"
)

const (
	ExecutionPreviewOutcomeMarketPositionExists         = "MARKET_POSITION_EXISTS"
	ExecutionPreviewOutcomeWalletRequestInFlight        = "WALLET_REQUEST_IN_FLIGHT"
	ExecutionPreviewOutcomeMarketUnavailable            = "MARKET_UNAVAILABLE"
	ExecutionPreviewOutcomeEstimateRejected             = "ESTIMATE_REJECTED"
	ExecutionPreviewOutcomeLiquidityInsufficient        = "LIQUIDITY_INSUFFICIENT"
	ExecutionPreviewOutcomeInsufficientUSDC             = "INSUFFICIENT_USDC"
	ExecutionPreviewOutcomeSkippedAfterInsufficientUSDC = "SKIPPED_AFTER_INSUFFICIENT_USDC"
	ExecutionPreviewOutcomeReady                        = "READY"

	executionPreviewExistingPosition   = "POSITION"
	executionPreviewExistingRequest    = "REQUEST"
	executionPreviewBackendPolymarket  = "polymarket"
	executionPreviewBackendHyperliquid = "hyperliquid"
	executionPreviewPolymarketFunds    = "5"
	executionPreviewHyperliquidFunds   = "1"
	executionPreviewLeverage           = 1.0
	executionPreviewPageSize           = 100
	executionPreviewDecimalMaxLength   = 128
)

// ExecutionPreviewReadClient is an authenticated, wallet-bound Worm client.
// Preview construction deliberately exposes only the two read operations it
// needs; draft creation, signing, submission, and cancellation are impossible
// through this boundary.
type ExecutionPreviewReadClient interface {
	ListPositionRequests(context.Context, worm.ListPositionRequestsOptions) (*worm.ListPositionRequestsResponse, error)
	ListMarginPositions(context.Context, worm.ListMarginPositionsOptions) (*worm.ListMarginPositionsResponse, error)
}

// ExecutionPreviewEstimateClient is an unauthenticated Worm client used for
// public margin estimates. It is separate from every wallet-bound client so a
// public estimate can never accidentally carry a wallet credential.
type ExecutionPreviewEstimateClient interface {
	EstimateMarginPosition(context.Context, worm.EstimateMarginPositionOptions) (*worm.MarginPositionEstimate, error)
}

type ExecutionPreviewWalletInput struct {
	WalletID    int64
	Address     string
	USDCBalance string
	SOLBalance  string
	Client      ExecutionPreviewReadClient
}

type ExecutionPreviewItemInput struct {
	EventConditionID  string
	MarketConditionID string
	IsYes             bool
	Backend           string
	Funds             string
	Selectable        bool
	UnavailableCode   string
}

type ExecutionPreviewInput struct {
	Wallets         []ExecutionPreviewWalletInput
	Items           []ExecutionPreviewItemInput
	PreflightChecks wormstore.ExecutionPreflightChecks
}

type ExecutionPreviewMarketEstimate struct {
	MarketOrdinal     int32
	MarketConditionID string
	IsYes             bool
	Funds             string
	AveragePrice      string
	TotalShares       string
	TotalCost         string
	BestAsk           string
	WorstFillPrice    string
	IsFullyFilled     bool
	FeeAmount         string
	UserFundsNeeded   string
	LiquidationPrice  *string
	RejectionCode     string
}

type ExecutionPreviewStep struct {
	Ordinal           int64
	WalletOrdinal     int32
	MarketOrdinal     int32
	WalletID          int64
	MarketConditionID string
	IsYes             bool
	Outcome           string
	ReasonCode        string
	ExistingKind      string
	ExistingPubkey    string
	Funds             string
	UserFundsNeeded   string
	FeeAmount         string
	USDCBalanceBefore string
	USDCBalanceAfter  string
	AdvisoryCodes     []string
}

type ExecutionPreviewResult struct {
	Estimates []ExecutionPreviewMarketEstimate
	Steps     []ExecutionPreviewStep
}

// ExecutionPreviewEstimatePhaseError identifies failures from the public,
// unauthenticated estimate phase. In particular, an HTTP authentication error
// here must not be attributed to any stored wallet credential.
type ExecutionPreviewEstimatePhaseError struct {
	Err error
}

func (e *ExecutionPreviewEstimatePhaseError) Error() string {
	if e == nil || e.Err == nil {
		return "execution preview estimate phase failed"
	}
	return e.Err.Error()
}

func (e *ExecutionPreviewEstimatePhaseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ExecutionPreviewWalletExposurePhaseError identifies the exact wallet whose
// authenticated position or request read failed. The worker uses this identity
// to CAS only that wallet's active credential into reconnect-required state.
type ExecutionPreviewWalletExposurePhaseError struct {
	WalletOrdinal int32
	WalletID      int64
	Err           error
}

func (e *ExecutionPreviewWalletExposurePhaseError) Error() string {
	if e == nil || e.Err == nil {
		return "execution preview wallet exposure phase failed"
	}
	return fmt.Sprintf("read Worm exposure for wallet %d: %v", e.WalletID, e.Err)
}

func (e *ExecutionPreviewWalletExposurePhaseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type ExecutionPreviewBuilder struct {
	estimateClient ExecutionPreviewEstimateClient
}

func NewExecutionPreviewBuilder(estimateClient ExecutionPreviewEstimateClient) (*ExecutionPreviewBuilder, error) {
	if estimateClient == nil {
		return nil, errors.New("execution preview estimate client is required")
	}
	return &ExecutionPreviewBuilder{estimateClient: estimateClient}, nil
}

type normalizedExecutionPreviewWallet struct {
	ExecutionPreviewWalletInput
	usdc decimalAmount
}

type normalizedExecutionPreviewItem struct {
	ExecutionPreviewItemInput
	backend string
	funds   string
}

type executionPreviewExposure struct {
	positions map[bool][]executionPreviewPosition
	requests  map[bool][]executionPreviewRequest
}

type executionPreviewPosition struct {
	pubkey                string
	positionRequestPubkey string
	isYes                 bool
	leverage              string
	createdAt             time.Time
	hasCreatedAt          bool
}

type executionPreviewRequest struct {
	pubkey            string
	marketConditionID string
	isYes             bool
}

// executionPreviewWalletExposure is a complete, wallet-wide HMAC snapshot.
// Markets retain position and request detail for deterministic preview and
// execution matching, while requests provides the mandatory wallet-global
// in-flight guard. Requests already represented by an open position's
// position_request_pubkey are removed before the snapshot is returned.
type executionPreviewWalletExposure struct {
	markets  map[string]*executionPreviewExposure
	requests []executionPreviewRequest
}

func (b *ExecutionPreviewBuilder) Build(ctx context.Context, input ExecutionPreviewInput) (*ExecutionPreviewResult, error) {
	wallets, items, err := normalizeExecutionPreviewInput(input)
	if err != nil {
		return nil, err
	}

	estimates, err := b.fetchEstimates(ctx, items)
	if err != nil {
		return nil, &ExecutionPreviewEstimatePhaseError{Err: err}
	}
	exposures := make([]executionPreviewWalletExposure, len(wallets))
	for index := range wallets {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		exposures[index], err = fetchExecutionPreviewWalletExposure(ctx, wallets[index].Client)
		if err != nil {
			return nil, &ExecutionPreviewWalletExposurePhaseError{
				WalletOrdinal: int32(index + 1),
				WalletID:      wallets[index].WalletID,
				Err:           err,
			}
		}
	}

	steps, err := buildExecutionPreviewSteps(wallets, items, estimates, exposures, input.PreflightChecks)
	if err != nil {
		return nil, err
	}
	return &ExecutionPreviewResult{Estimates: estimates, Steps: steps}, nil
}

func normalizeExecutionPreviewInput(input ExecutionPreviewInput) (
	[]normalizedExecutionPreviewWallet,
	[]normalizedExecutionPreviewItem,
	error,
) {
	if len(input.Wallets) == 0 {
		return nil, nil, errors.New("execution preview requires at least one wallet")
	}
	if len(input.Items) == 0 {
		return nil, nil, errors.New("execution preview requires at least one market")
	}

	wallets := make([]normalizedExecutionPreviewWallet, 0, len(input.Wallets))
	seenWalletIDs := make(map[int64]struct{}, len(input.Wallets))
	seenAddresses := make(map[string]struct{}, len(input.Wallets))
	for index, walletInput := range input.Wallets {
		walletID, address, err := normalizeWormWalletReference(walletInput.WalletID, walletInput.Address)
		if err != nil {
			return nil, nil, fmt.Errorf("wallet %d is invalid: %w", index+1, err)
		}
		if walletInput.Client == nil {
			return nil, nil, fmt.Errorf("wallet %d read client is required", index+1)
		}
		if _, exists := seenWalletIDs[walletID]; exists {
			return nil, nil, fmt.Errorf("wallet %d duplicates wallet_id %d", index+1, walletID)
		}
		if _, exists := seenAddresses[address]; exists {
			return nil, nil, fmt.Errorf("wallet %d duplicates address %q", index+1, address)
		}
		usdc, err := parseExecutionPreviewDecimal(walletInput.USDCBalance)
		if err != nil {
			return nil, nil, fmt.Errorf("wallet %d USDC balance is invalid: %w", index+1, err)
		}
		if _, err := parseExecutionPreviewDecimal(walletInput.SOLBalance); err != nil {
			return nil, nil, fmt.Errorf("wallet %d SOL balance is invalid: %w", index+1, err)
		}
		seenWalletIDs[walletID] = struct{}{}
		seenAddresses[address] = struct{}{}
		walletInput.WalletID = walletID
		walletInput.Address = address
		walletInput.USDCBalance = usdc.String()
		sol, _ := parseExecutionPreviewDecimal(walletInput.SOLBalance)
		walletInput.SOLBalance = sol.String()
		wallets = append(wallets, normalizedExecutionPreviewWallet{ExecutionPreviewWalletInput: walletInput, usdc: usdc})
	}

	items := make([]normalizedExecutionPreviewItem, 0, len(input.Items))
	seenMarkets := make(map[string]struct{}, len(input.Items))
	for index, itemInput := range input.Items {
		if !validExecutionPreviewConditionID(itemInput.EventConditionID) {
			return nil, nil, fmt.Errorf("market %d event condition ID is invalid", index+1)
		}
		if !validExecutionPreviewConditionID(itemInput.MarketConditionID) {
			return nil, nil, fmt.Errorf("market %d condition ID is invalid", index+1)
		}
		if _, exists := seenMarkets[itemInput.MarketConditionID]; exists {
			return nil, nil, fmt.Errorf("market %d duplicates condition ID %q", index+1, itemInput.MarketConditionID)
		}
		backend := strings.ToLower(strings.TrimSpace(itemInput.Backend))
		if backend != itemInput.Backend || (backend != "" && !isCanonicalNonEmptyString(backend)) {
			return nil, nil, fmt.Errorf("market %d backend is invalid", index+1)
		}
		expectedFunds, supportedBackend := executionPreviewFundsForBackend(backend)
		funds, err := parseExecutionPreviewDecimal(itemInput.Funds)
		if err != nil {
			return nil, nil, fmt.Errorf("market %d funds are invalid: %w", index+1, err)
		}
		if itemInput.Selectable {
			if !supportedBackend || funds.String() != expectedFunds || itemInput.UnavailableCode != "" {
				return nil, nil, fmt.Errorf("market %d selectable configuration is invalid", index+1)
			}
		} else {
			if !isCanonicalNonEmptyString(itemInput.UnavailableCode) {
				return nil, nil, fmt.Errorf("market %d unavailable code is required", index+1)
			}
			if supportedBackend {
				if funds.String() != expectedFunds {
					return nil, nil, fmt.Errorf("market %d funds do not match backend", index+1)
				}
			} else if !funds.IsZero() {
				return nil, nil, fmt.Errorf("market %d unsupported backend must use zero funds", index+1)
			}
		}
		seenMarkets[itemInput.MarketConditionID] = struct{}{}
		itemInput.Backend = backend
		itemInput.Funds = funds.String()
		items = append(items, normalizedExecutionPreviewItem{
			ExecutionPreviewItemInput: itemInput,
			backend:                   backend,
			funds:                     funds.String(),
		})
	}
	return wallets, items, nil
}

func (b *ExecutionPreviewBuilder) fetchEstimates(
	ctx context.Context,
	items []normalizedExecutionPreviewItem,
) ([]ExecutionPreviewMarketEstimate, error) {
	result := make([]ExecutionPreviewMarketEstimate, len(items))
	cache := make(map[string]ExecutionPreviewMarketEstimate, len(items))
	for index, item := range items {
		estimate := ExecutionPreviewMarketEstimate{
			MarketOrdinal:     int32(index + 1),
			MarketConditionID: item.MarketConditionID,
			IsYes:             item.IsYes,
			Funds:             item.funds,
		}
		if !item.Selectable {
			result[index] = estimate
			continue
		}
		key := executionPreviewEstimateKey(item)
		if cached, exists := cache[key]; exists {
			cached.MarketOrdinal = int32(index + 1)
			result[index] = cached
			continue
		}
		isYes := item.IsYes
		leverage := executionPreviewLeverage
		providerEstimate, err := b.estimateClient.EstimateMarginPosition(ctx, worm.EstimateMarginPositionOptions{
			MarketConditionID: item.MarketConditionID,
			Funds:             item.funds,
			IsYes:             &isYes,
			Leverage:          &leverage,
		})
		if err != nil {
			if isExecutionPreviewEstimateRejection(err) {
				estimate.RejectionCode = wormErrorRejected
				result[index] = estimate
				cache[key] = estimate
				continue
			}
			return nil, fmt.Errorf("estimate Worm market %s: %w", item.MarketConditionID, err)
		}
		if providerEstimate == nil {
			return nil, fmt.Errorf("estimate Worm market %s: invalid empty response", item.MarketConditionID)
		}
		normalized, err := normalizeExecutionPreviewEstimate(estimate, providerEstimate)
		if err != nil {
			return nil, fmt.Errorf("estimate Worm market %s: %w", item.MarketConditionID, err)
		}
		result[index] = normalized
		cache[key] = normalized
	}
	return result, nil
}

func normalizeExecutionPreviewEstimate(
	base ExecutionPreviewMarketEstimate,
	provider *worm.MarginPositionEstimate,
) (ExecutionPreviewMarketEstimate, error) {
	priceFields := []struct {
		name   string
		value  string
		target *string
	}{
		{name: "average_price", value: provider.AveragePrice, target: &base.AveragePrice},
		{name: "best_ask", value: provider.BestAsk, target: &base.BestAsk},
		{name: "worst_fill_price", value: provider.WorstFillPrice, target: &base.WorstFillPrice},
	}
	for _, field := range priceFields {
		value, err := parseExecutionPreviewDecimal(field.value)
		if err != nil || value.Compare(executionPreviewOne()) > 0 {
			return ExecutionPreviewMarketEstimate{}, fmt.Errorf("%s is invalid", field.name)
		}
		*field.target = value.String()
	}
	nonNegativeFields := []struct {
		name   string
		value  string
		target *string
	}{
		{name: "total_shares", value: provider.TotalShares, target: &base.TotalShares},
		{name: "total_cost", value: provider.TotalCost, target: &base.TotalCost},
		{name: "fee_amount", value: provider.FeeAmount, target: &base.FeeAmount},
		{name: "user_funds_needed", value: provider.UserFundsNeeded, target: &base.UserFundsNeeded},
	}
	for _, field := range nonNegativeFields {
		value, err := parseExecutionPreviewDecimal(field.value)
		if err != nil {
			return ExecutionPreviewMarketEstimate{}, fmt.Errorf("%s is invalid", field.name)
		}
		*field.target = value.String()
	}
	if provider.LiquidationPrice != nil {
		return ExecutionPreviewMarketEstimate{}, errors.New("liquidation_price must be absent for one-times leverage")
	}
	funds, err := parseExecutionPreviewDecimal(base.Funds)
	if err != nil {
		return ExecutionPreviewMarketEstimate{}, errors.New("funds is invalid")
	}
	fee, err := parseExecutionPreviewDecimal(base.FeeAmount)
	if err != nil {
		return ExecutionPreviewMarketEstimate{}, errors.New("fee_amount is invalid")
	}
	userFundsNeeded, err := parseExecutionPreviewDecimal(base.UserFundsNeeded)
	if err != nil {
		return ExecutionPreviewMarketEstimate{}, errors.New("user_funds_needed is invalid")
	}
	if !executionPreviewDecimalSumEquals(funds, fee, userFundsNeeded) {
		return ExecutionPreviewMarketEstimate{}, errors.New("user_funds_needed must equal funds plus fee_amount")
	}
	base.IsFullyFilled = provider.IsFullyFilled
	return base, nil
}

func fetchExecutionPreviewWalletExposure(
	ctx context.Context,
	client ExecutionPreviewReadClient,
) (executionPreviewWalletExposure, error) {
	exposure := executionPreviewWalletExposure{markets: make(map[string]*executionPreviewExposure)}
	positionRequestPubkeys, err := fetchAllExecutionPreviewPositions(ctx, client, &exposure)
	if err != nil {
		return executionPreviewWalletExposure{}, err
	}
	if err := fetchAllExecutionPreviewRequests(ctx, client, &exposure); err != nil {
		return executionPreviewWalletExposure{}, err
	}
	suppressExecutionPreviewPositionBackedRequests(&exposure, positionRequestPubkeys)
	for _, market := range exposure.markets {
		for side := range market.positions {
			sort.Slice(market.positions[side], func(i, j int) bool {
				return market.positions[side][i].pubkey < market.positions[side][j].pubkey
			})
		}
		for side := range market.requests {
			sort.Slice(market.requests[side], func(i, j int) bool {
				return market.requests[side][i].pubkey < market.requests[side][j].pubkey
			})
		}
	}
	sort.Slice(exposure.requests, func(i, j int) bool {
		return exposure.requests[i].pubkey < exposure.requests[j].pubkey
	})
	return exposure, nil
}

func fetchAllExecutionPreviewPositions(
	ctx context.Context,
	client ExecutionPreviewReadClient,
	exposure *executionPreviewWalletExposure,
) (map[string]struct{}, error) {
	isClosed := false
	cursor := ""
	seenCursors := make(map[string]struct{})
	seenPubkeys := make(map[string]struct{})
	positionRequestPubkeys := make(map[string]struct{})
	for {
		response, err := client.ListMarginPositions(ctx, worm.ListMarginPositionsOptions{
			PageOptions: worm.PageOptions{Limit: executionPreviewPageSize, Cursor: cursor},
			IsClosed:    &isClosed,
			Sort:        "-created",
		})
		if err != nil {
			return nil, fmt.Errorf("list open positions: %w", err)
		}
		if response == nil || len(response.Positions) > executionPreviewPageSize {
			return nil, errors.New("list open positions returned an invalid page")
		}
		for _, position := range response.Positions {
			if position.IsClosed {
				return nil, errors.New("list open positions returned a closed position")
			}
			if _, err := wormOpenPositionFromProvider(position); err != nil ||
				!validExecutionPreviewConditionID(position.Pubkey) ||
				!validExecutionPreviewConditionID(position.Market.ConditionID) {
				return nil, errors.New("list open positions returned an invalid position")
			}
			if position.PositionRequestPubkey != nil &&
				!validExecutionPreviewConditionID(*position.PositionRequestPubkey) {
				return nil, errors.New("list open positions returned an invalid position request pubkey")
			}
			if _, exists := seenPubkeys[position.Pubkey]; exists {
				return nil, errors.New("list open positions returned a duplicate pubkey")
			}
			seenPubkeys[position.Pubkey] = struct{}{}
			entry := executionPreviewPosition{
				pubkey:   position.Pubkey,
				isYes:    position.IsYes,
				leverage: position.Leverage,
			}
			if position.PositionRequestPubkey != nil {
				entry.positionRequestPubkey = *position.PositionRequestPubkey
				positionRequestPubkeys[entry.positionRequestPubkey] = struct{}{}
			}
			if position.Created != nil && *position.Created > 0 {
				entry.createdAt = time.Unix(*position.Created, 0).UTC()
				entry.hasCreatedAt = true
			}
			market := ensureExecutionPreviewExposure(exposure, position.Market.ConditionID)
			market.positions[position.IsYes] = append(market.positions[position.IsYes], entry)
		}
		nextCursor, done, err := nextExecutionPreviewCursor(cursor, response.Meta, seenCursors)
		if err != nil {
			return nil, fmt.Errorf("list open positions: %w", err)
		}
		if done {
			return positionRequestPubkeys, nil
		}
		cursor = nextCursor
	}
}

func fetchAllExecutionPreviewRequests(
	ctx context.Context,
	client ExecutionPreviewReadClient,
	exposure *executionPreviewWalletExposure,
) error {
	cursor := ""
	seenCursors := make(map[string]struct{})
	seenPubkeys := make(map[string]struct{})
	for {
		response, err := client.ListPositionRequests(ctx, worm.ListPositionRequestsOptions{
			PageOptions: worm.PageOptions{Limit: executionPreviewPageSize, Cursor: cursor},
			States:      wormPositionRequestStates,
			Sort:        "-created",
		})
		if err != nil {
			return fmt.Errorf("list in-flight position requests: %w", err)
		}
		if response == nil || len(response.Requests) > executionPreviewPageSize {
			return errors.New("list in-flight position requests returned an invalid page")
		}
		for _, request := range response.Requests {
			if !isInFlightPositionRequestState(request.State) {
				return errors.New("list in-flight position requests returned a terminal request")
			}
			if _, err := wormInFlightRequestFromProvider(request); err != nil ||
				!validExecutionPreviewConditionID(request.Pubkey) || request.Market == nil ||
				!validExecutionPreviewConditionID(request.Market.ConditionID) {
				return errors.New("list in-flight position requests returned an invalid request")
			}
			if _, exists := seenPubkeys[request.Pubkey]; exists {
				return errors.New("list in-flight position requests returned a duplicate pubkey")
			}
			seenPubkeys[request.Pubkey] = struct{}{}
			entry := executionPreviewRequest{
				pubkey:            request.Pubkey,
				marketConditionID: request.Market.ConditionID,
				isYes:             request.IsYes,
			}
			market := ensureExecutionPreviewExposure(exposure, request.Market.ConditionID)
			market.requests[request.IsYes] = append(market.requests[request.IsYes], entry)
			exposure.requests = append(exposure.requests, entry)
		}
		nextCursor, done, err := nextExecutionPreviewCursor(cursor, response.Meta, seenCursors)
		if err != nil {
			return fmt.Errorf("list in-flight position requests: %w", err)
		}
		if done {
			return nil
		}
		cursor = nextCursor
	}
}

func suppressExecutionPreviewPositionBackedRequests(
	exposure *executionPreviewWalletExposure,
	positionRequestPubkeys map[string]struct{},
) {
	if exposure == nil || len(positionRequestPubkeys) == 0 || len(exposure.requests) == 0 {
		return
	}
	filtered := make([]executionPreviewRequest, 0, len(exposure.requests))
	for _, request := range exposure.requests {
		if _, backed := positionRequestPubkeys[request.pubkey]; backed {
			continue
		}
		filtered = append(filtered, request)
	}
	exposure.requests = filtered
	for _, market := range exposure.markets {
		market.requests = map[bool][]executionPreviewRequest{false: {}, true: {}}
	}
	for _, request := range exposure.requests {
		market := ensureExecutionPreviewExposure(exposure, request.marketConditionID)
		market.requests[request.isYes] = append(market.requests[request.isYes], request)
	}
}

func buildExecutionPreviewSteps(
	wallets []normalizedExecutionPreviewWallet,
	items []normalizedExecutionPreviewItem,
	estimates []ExecutionPreviewMarketEstimate,
	exposures []executionPreviewWalletExposure,
	checks wormstore.ExecutionPreflightChecks,
) ([]ExecutionPreviewStep, error) {
	if len(estimates) != len(items) || len(exposures) != len(wallets) {
		return nil, errors.New("execution preview builder received mismatched observations")
	}
	stepCount := new(big.Int).Mul(big.NewInt(int64(len(wallets))), big.NewInt(int64(len(items))))
	if !stepCount.IsInt64() {
		return nil, errors.New("execution preview step count exceeds int64")
	}
	steps := make([]ExecutionPreviewStep, 0, stepCount.Int64())
	var ordinal int64
	for walletIndex, wallet := range wallets {
		remaining := wallet.usdc
		blockedForUSDC := false
		blockedForWalletReason := ""
		for itemIndex, item := range items {
			ordinal++
			estimate := estimates[itemIndex]
			step := ExecutionPreviewStep{
				Ordinal:           ordinal,
				WalletOrdinal:     int32(walletIndex + 1),
				MarketOrdinal:     int32(itemIndex + 1),
				WalletID:          wallet.WalletID,
				MarketConditionID: item.MarketConditionID,
				IsYes:             item.IsYes,
				Funds:             item.funds,
				UserFundsNeeded:   estimate.UserFundsNeeded,
				FeeAmount:         estimate.FeeAmount,
				USDCBalanceBefore: remaining.String(),
				USDCBalanceAfter:  remaining.String(),
			}
			marketExposure := exposures[walletIndex].market(item.MarketConditionID)
			step.AdvisoryCodes = executionPreviewIgnoredAdvisoryCodes(
				item,
				estimate,
				checks,
			)
			if blockedForWalletReason != "" {
				step.Outcome = blockedForWalletReason
				step.ReasonCode = blockedForWalletReason
				steps = append(steps, step)
				continue
			}
			if position, exists := firstExecutionPreviewMarketPosition(marketExposure); exists {
				step.ExistingKind = executionPreviewExistingPosition
				step.ExistingPubkey = position.pubkey
				step.Outcome = ExecutionPreviewOutcomeMarketPositionExists
				step.ReasonCode = ExecutionPreviewOutcomeMarketPositionExists
				blockedForWalletReason = ExecutionPreviewOutcomeMarketPositionExists
				steps = append(steps, step)
				continue
			}
			if request, exists := exposures[walletIndex].firstRequest(); exists {
				step.ExistingKind = executionPreviewExistingRequest
				step.ExistingPubkey = request.pubkey
				step.Outcome = ExecutionPreviewOutcomeWalletRequestInFlight
				step.ReasonCode = ExecutionPreviewOutcomeWalletRequestInFlight
				blockedForWalletReason = ExecutionPreviewOutcomeWalletRequestInFlight
				steps = append(steps, step)
				continue
			}
			if blockedForUSDC {
				step.Outcome = ExecutionPreviewOutcomeSkippedAfterInsufficientUSDC
				step.ReasonCode = ExecutionPreviewOutcomeInsufficientUSDC
				steps = append(steps, step)
				continue
			}
			if !item.Selectable {
				step.Outcome = ExecutionPreviewOutcomeMarketUnavailable
				step.ReasonCode = item.UnavailableCode
				steps = append(steps, step)
				continue
			}
			if estimate.RejectionCode != "" {
				step.Outcome = ExecutionPreviewOutcomeEstimateRejected
				step.ReasonCode = estimate.RejectionCode
				steps = append(steps, step)
				continue
			}
			if !estimate.IsFullyFilled {
				if checks.RequireFullLiquidity {
					step.Outcome = ExecutionPreviewOutcomeLiquidityInsufficient
					step.ReasonCode = ExecutionPreviewOutcomeLiquidityInsufficient
					steps = append(steps, step)
					continue
				}
			}
			needed, err := parseExecutionPreviewDecimal(estimate.UserFundsNeeded)
			if err != nil {
				return nil, fmt.Errorf("market %d normalized estimate is invalid", itemIndex+1)
			}
			if remaining.Compare(needed) < 0 {
				step.Outcome = ExecutionPreviewOutcomeInsufficientUSDC
				step.ReasonCode = ExecutionPreviewOutcomeInsufficientUSDC
				blockedForUSDC = true
				steps = append(steps, step)
				continue
			}
			remaining = remaining.Subtract(needed)
			step.Outcome = ExecutionPreviewOutcomeReady
			step.USDCBalanceAfter = remaining.String()
			steps = append(steps, step)
		}
	}
	return steps, nil
}

func executionPreviewIgnoredAdvisoryCodes(
	item normalizedExecutionPreviewItem,
	estimate ExecutionPreviewMarketEstimate,
	checks wormstore.ExecutionPreflightChecks,
) []string {
	var codes []string
	if !checks.RequireFullLiquidity && item.Selectable && estimate.RejectionCode == "" && !estimate.IsFullyFilled {
		codes = appendExecutionPreviewAdvisory(codes, ExecutionPreviewOutcomeLiquidityInsufficient)
	}
	return codes
}

func appendExecutionPreviewAdvisory(codes []string, code string) []string {
	for _, existing := range codes {
		if existing == code {
			return codes
		}
	}
	return append(codes, code)
}

func ensureExecutionPreviewExposure(exposure *executionPreviewWalletExposure, marketConditionID string) *executionPreviewExposure {
	market := exposure.markets[marketConditionID]
	if market == nil {
		market = &executionPreviewExposure{
			positions: map[bool][]executionPreviewPosition{false: {}, true: {}},
			requests:  map[bool][]executionPreviewRequest{false: {}, true: {}},
		}
		exposure.markets[marketConditionID] = market
	}
	return market
}

func (exposure executionPreviewWalletExposure) market(marketConditionID string) *executionPreviewExposure {
	if exposure.markets == nil {
		return nil
	}
	return exposure.markets[marketConditionID]
}

func (exposure executionPreviewWalletExposure) firstRequest() (executionPreviewRequest, bool) {
	if len(exposure.requests) == 0 {
		return executionPreviewRequest{}, false
	}
	return exposure.requests[0], true
}

func firstExecutionPreviewMarketPosition(market *executionPreviewExposure) (executionPreviewPosition, bool) {
	if market == nil {
		return executionPreviewPosition{}, false
	}
	no := market.positions[false]
	yes := market.positions[true]
	if len(no) == 0 && len(yes) == 0 {
		return executionPreviewPosition{}, false
	}
	if len(no) == 0 {
		return yes[0], true
	}
	if len(yes) == 0 || no[0].pubkey < yes[0].pubkey {
		return no[0], true
	}
	return yes[0], true
}

func nextExecutionPreviewCursor(
	current string,
	meta worm.EnvelopeMeta,
	seen map[string]struct{},
) (string, bool, error) {
	if meta.Limit != executionPreviewPageSize {
		return "", false, errors.New("pagination meta.limit is missing or does not match the requested limit")
	}
	if meta.NextCursor == nil || *meta.NextCursor == "" {
		return "", true, nil
	}
	next := *meta.NextCursor
	if !isCanonicalNonEmptyString(next) || len(next) > 4096 || next == current {
		return "", false, errors.New("pagination cursor is invalid")
	}
	if _, exists := seen[next]; exists {
		return "", false, errors.New("pagination cursor did not advance")
	}
	seen[next] = struct{}{}
	return next, false, nil
}

func executionPreviewEstimateKey(item normalizedExecutionPreviewItem) string {
	return fmt.Sprintf("%s\x00%t\x00%s\x00%.1f", item.MarketConditionID, item.IsYes, item.funds, executionPreviewLeverage)
}

func executionPreviewFundsForBackend(backend string) (string, bool) {
	switch backend {
	case executionPreviewBackendPolymarket:
		return executionPreviewPolymarketFunds, true
	case executionPreviewBackendHyperliquid:
		return executionPreviewHyperliquidFunds, true
	default:
		return "0", false
	}
}

func isExecutionPreviewEstimateRejection(err error) bool {
	var wormErr *worm.Error
	if !errors.As(err, &wormErr) {
		return false
	}
	if wormErr.StatusCode < http.StatusBadRequest || wormErr.StatusCode >= http.StatusInternalServerError {
		return false
	}
	switch wormErr.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusRequestTimeout,
		http.StatusTooEarly, http.StatusTooManyRequests:
		return false
	default:
		return true
	}
}

func validExecutionPreviewConditionID(value string) bool {
	if value == "" || value != strings.TrimSpace(value) {
		return false
	}
	publicKey, err := solana.PublicKeyFromBase58(value)
	return err == nil && publicKey.String() == value
}

type decimalAmount struct {
	coefficient *big.Int
	scale       int
}

func parseExecutionPreviewDecimal(value string) (decimalAmount, error) {
	if len(value) > executionPreviewDecimalMaxLength || !isNonNegativeDecimalString(value) {
		return decimalAmount{}, errors.New("must be a canonical non-negative decimal")
	}
	parts := strings.SplitN(value, ".", 2)
	digits := parts[0]
	scale := 0
	if len(parts) == 2 {
		digits += parts[1]
		scale = len(parts[1])
	}
	coefficient := new(big.Int)
	if _, ok := coefficient.SetString(digits, 10); !ok {
		return decimalAmount{}, errors.New("must be a canonical non-negative decimal")
	}
	for scale > 0 {
		quotient, remainder := new(big.Int), new(big.Int)
		quotient.QuoRem(coefficient, big.NewInt(10), remainder)
		if remainder.Sign() != 0 {
			break
		}
		coefficient = quotient
		scale--
	}
	return decimalAmount{coefficient: coefficient, scale: scale}, nil
}

func executionPreviewOne() decimalAmount {
	return decimalAmount{coefficient: big.NewInt(1)}
}

func (d decimalAmount) IsZero() bool {
	return d.coefficient.Sign() == 0
}

func (d decimalAmount) Compare(other decimalAmount) int {
	left, right := executionPreviewAlignedCoefficients(d, other)
	return left.Cmp(right)
}

func executionPreviewDecimalSumEquals(left, right, total decimalAmount) bool {
	scale := executionPreviewMaxInt(executionPreviewMaxInt(left.scale, right.scale), total.scale)
	sum := new(big.Int).Add(
		executionPreviewScaleCoefficient(left, scale),
		executionPreviewScaleCoefficient(right, scale),
	)
	return sum.Cmp(executionPreviewScaleCoefficient(total, scale)) == 0
}

func (d decimalAmount) Subtract(other decimalAmount) decimalAmount {
	left, right := executionPreviewAlignedCoefficients(d, other)
	if left.Cmp(right) < 0 {
		panic("execution preview decimal subtraction underflow")
	}
	result := decimalAmount{coefficient: new(big.Int).Sub(left, right), scale: executionPreviewMaxInt(d.scale, other.scale)}
	normalized, err := parseExecutionPreviewDecimal(result.String())
	if err != nil {
		panic("execution preview decimal normalization failed")
	}
	return normalized
}

func (d decimalAmount) String() string {
	digits := d.coefficient.String()
	if d.scale == 0 {
		return digits
	}
	if len(digits) <= d.scale {
		digits = strings.Repeat("0", d.scale-len(digits)+1) + digits
	}
	index := len(digits) - d.scale
	return digits[:index] + "." + digits[index:]
}

func executionPreviewAlignedCoefficients(left, right decimalAmount) (*big.Int, *big.Int) {
	scale := executionPreviewMaxInt(left.scale, right.scale)
	return executionPreviewScaleCoefficient(left, scale), executionPreviewScaleCoefficient(right, scale)
}

func executionPreviewScaleCoefficient(value decimalAmount, scale int) *big.Int {
	coefficient := new(big.Int).Set(value.coefficient)
	if delta := scale - value.scale; delta > 0 {
		coefficient.Mul(coefficient, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(delta)), nil))
	}
	return coefficient
}

func executionPreviewMaxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
