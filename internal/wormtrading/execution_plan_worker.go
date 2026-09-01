package wormtrading

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	wormmarketsapiclient "github.com/useryege/athena/internal/wormmarkets/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	executionPlanWorkerPollInterval = time.Second
	executionPlanWorkerLease        = 2 * time.Minute
	executionPlanWorkerHeartbeat    = 20 * time.Second
	executionPlanCleanupInterval    = time.Hour
	executionPlanCleanupLimit       = int32(100)

	executionPlanStageCatalogs    = "READING_MARKETS"
	executionPlanStageConnections = "READING_CONNECTIONS"
	executionPlanStageBalances    = "READING_BALANCES"
	executionPlanStagePreview     = "BUILDING_PREVIEW"
	executionPlanStageFinalizing  = "FINALIZING"

	executionPlanFailureStoreUnavailable      = "STORE_UNAVAILABLE"
	executionPlanFailureMarketsUnavailable    = "WORM_MARKETS_UNAVAILABLE"
	executionPlanFailureMarketsInvalid        = "WORM_MARKETS_INVALID_RESPONSE"
	executionPlanFailureWalletNotConnected    = "WALLET_NOT_CONNECTED"
	executionPlanFailureCredentialUnavailable = "WALLET_CREDENTIAL_UNAVAILABLE"
	executionPlanFailureBalanceUnavailable    = "SOLANA_BALANCE_UNAVAILABLE"
	executionPlanFailureUSDCUnavailable       = "USDC_BALANCE_UNAVAILABLE"
	executionPlanFailureWormReadUnavailable   = "WORM_READ_UNAVAILABLE"
	executionPlanFailureWormResponseInvalid   = "WORM_INVALID_RESPONSE"
	executionPlanFailureSourceChanged         = "PLAN_SOURCE_CHANGED"
	executionPlanFailurePreviewInvalid        = "PREVIEW_INVALID"
	executionPlanUnknownBackend               = "unknown"
	executionPlanMarketMissing                = "MARKET_NOT_FOUND"
	executionPlanMarketUnavailable            = "MARKET_UNAVAILABLE"
	executionPlanOutcomeUnavailable           = "OUTCOME_UNAVAILABLE"
)

type executionPlanBuildFailure struct {
	code string
	err  error
}

func (e *executionPlanBuildFailure) Error() string {
	if e == nil || e.err == nil {
		return "execution plan build failed"
	}
	return e.err.Error()
}

func (e *executionPlanBuildFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func failExecutionPlanBuild(code string, err error) error {
	if err == nil {
		err = errors.New("execution plan build failed")
	}
	return &executionPlanBuildFailure{code: code, err: err}
}

type executionPlanCatalogItem struct {
	input       ExecutionPreviewItemInput
	observation wormstore.ExecutionPlanItemObservation
}

type executionPlanWalletContext struct {
	input              ExecutionPreviewWalletInput
	observation        wormstore.ExecutionPlanWalletObservation
	activeCredentialID int64
}

func (s *Service) runExecutionPlanWorker(ctx context.Context) {
	defer s.runWG.Done()
	workerID := "worm-preview-" + uuid.NewString()
	nextCleanup := time.Time{}
	for {
		if ctx.Err() != nil {
			return
		}
		now := timeNowUTC()
		if nextCleanup.IsZero() || !now.Before(nextCleanup) {
			_, cleanupErr := s.credentialStore.DeleteExpiredExecutionPlans(ctx, now, executionPlanCleanupLimit)
			s.recordCredentialStoreResult(cleanupErr)
			if cleanupErr != nil && ctx.Err() == nil {
				log.WithError(cleanupErr).Warn("failed to clean retained Worm execution previews")
			}
			nextCleanup = now.Add(executionPlanCleanupInterval)
		}

		plan, err := s.credentialStore.ClaimExecutionPlan(ctx, workerID, executionPlanWorkerLease, now)
		s.recordCredentialStoreResult(err)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.WithError(err).Warn("failed to claim a Worm execution preview")
			if !waitExecutionPlanWorker(ctx, executionPlanWorkerPollInterval) {
				return
			}
			continue
		}
		if plan == nil {
			if !waitExecutionPlanWorker(ctx, executionPlanWorkerPollInterval) {
				return
			}
			continue
		}
		s.buildClaimedExecutionPlan(ctx, workerID, plan)
	}
}

func waitExecutionPlanWorker(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (s *Service) buildClaimedExecutionPlan(parent context.Context, workerID string, plan *wormstore.ExecutionPlan) {
	buildCtx, cancelBuild := context.WithCancel(parent)
	guard := newExecutionPlanLeaseGuard(s, plan.ID, workerID, cancelBuild)
	guard.start(parent)
	defer cancelBuild()

	readyRequest, buildErr := s.constructExecutionPlanSnapshot(buildCtx, guard, plan)
	leaseErr := guard.stop()
	if leaseErr != nil {
		if errors.Is(leaseErr, wormstore.ErrExecutionPlanBuildLease) {
			return
		}
		buildErr = failExecutionPlanBuild(executionPlanFailureStoreUnavailable, leaseErr)
	}
	if buildErr == nil && parent.Err() == nil {
		_, markErr := s.credentialStore.MarkExecutionPlanReady(parent, *readyRequest)
		s.recordCredentialStoreResult(markErr)
		switch {
		case markErr == nil:
			buildErr = nil
		case errors.Is(markErr, wormstore.ErrExecutionPlanBuildLease):
			return
		case errors.Is(markErr, wormstore.ErrExecutionPlanRevision), errors.Is(markErr, wormstore.ErrExecutionPlanCombinationChanged):
			buildErr = failExecutionPlanBuild(executionPlanFailureSourceChanged, markErr)
		case errors.Is(markErr, wormstore.ErrExecutionPlanWalletConnectionChanged):
			buildErr = failExecutionPlanBuild(executionPlanFailureWalletNotConnected, markErr)
		case errors.Is(markErr, wormstore.ErrExecutionPlanCredentialChanged):
			buildErr = failExecutionPlanBuild(executionPlanFailureCredentialUnavailable, markErr)
		default:
			buildErr = failExecutionPlanBuild(executionPlanFailureStoreUnavailable, markErr)
		}
	}
	if buildErr == nil || parent.Err() != nil {
		return
	}

	failureCode := executionPlanFailureCode(buildErr)
	_, failureErr := s.credentialStore.MarkExecutionPlanFailed(parent, plan.ID, workerID, failureCode, timeNowUTC())
	s.recordCredentialStoreResult(failureErr)
	if failureErr != nil {
		log.WithFields(log.Fields{"plan_id": plan.ID, "failure_code": failureCode}).WithError(failureErr).
			Warn("failed to persist Worm execution preview failure")
		return
	}
	log.WithFields(log.Fields{"plan_id": plan.ID, "failure_code": failureCode}).Warn("Worm execution preview build failed")
}

func executionPlanFailureCode(err error) string {
	var failure *executionPlanBuildFailure
	if errors.As(err, &failure) && failure.code != "" {
		return failure.code
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return executionPlanFailureWormReadUnavailable
	}
	return executionPlanFailurePreviewInvalid
}

func (s *Service) constructExecutionPlanSnapshot(
	ctx context.Context,
	guard *executionPlanLeaseGuard,
	plan *wormstore.ExecutionPlan,
) (*wormstore.MarkExecutionPlanReadyRequest, error) {
	if plan == nil || plan.ID == "" || plan.WalletCount <= 0 || plan.ItemCount <= 0 ||
		int64(len(plan.Wallets)) != plan.WalletCount || int64(len(plan.Items)) != plan.ItemCount ||
		plan.WalletCount > math.MaxInt64/plan.ItemCount || plan.WalletCount*plan.ItemCount != plan.TotalStepCount {
		return nil, failExecutionPlanBuild(executionPlanFailurePreviewInvalid, errors.New("claimed execution plan is incomplete"))
	}
	if err := guard.setStage(ctx, executionPlanStageCatalogs, 0); err != nil {
		return nil, failExecutionPlanBuild(executionPlanFailureStoreUnavailable, err)
	}
	catalogItems, err := s.readExecutionPlanCatalogs(ctx, plan.Items)
	if err != nil {
		return nil, err
	}

	if err := guard.setStage(ctx, executionPlanStageConnections, 0); err != nil {
		return nil, failExecutionPlanBuild(executionPlanFailureStoreUnavailable, err)
	}
	wallets, err := s.readExecutionPlanWalletConnections(ctx, plan.Wallets)
	if err != nil {
		return nil, err
	}

	if err := guard.setStage(ctx, executionPlanStageBalances, 0); err != nil {
		return nil, failExecutionPlanBuild(executionPlanFailureStoreUnavailable, err)
	}
	if err := s.readExecutionPlanWalletBalances(ctx, wallets); err != nil {
		return nil, err
	}

	estimateClient, err := s.wormClientFactory.NewUnauthenticatedClient()
	if err != nil {
		return nil, failExecutionPlanBuild(executionPlanFailurePreviewInvalid, fmt.Errorf("create public Worm estimate client: %w", err))
	}
	builder, err := NewExecutionPreviewBuilder(estimateClient)
	if err != nil {
		return nil, failExecutionPlanBuild(executionPlanFailurePreviewInvalid, err)
	}
	previewInput := ExecutionPreviewInput{
		Wallets: make([]ExecutionPreviewWalletInput, 0, len(wallets)),
		Items:   make([]ExecutionPreviewItemInput, 0, len(catalogItems)),
	}
	for _, wallet := range wallets {
		previewInput.Wallets = append(previewInput.Wallets, wallet.input)
	}
	for _, item := range catalogItems {
		previewInput.Items = append(previewInput.Items, item.input)
	}
	if err := guard.setStage(ctx, executionPlanStagePreview, 0); err != nil {
		return nil, failExecutionPlanBuild(executionPlanFailureStoreUnavailable, err)
	}
	result, err := builder.Build(ctx, previewInput)
	s.wormCapabilities.recordWormResult(err)
	if err != nil {
		failureCode := s.classifyExecutionPreviewBuildFailure(ctx, err, wallets)
		return nil, failExecutionPlanBuild(failureCode, err)
	}
	if result == nil || len(result.Estimates) != len(catalogItems) || int64(len(result.Steps)) != plan.TotalStepCount {
		return nil, failExecutionPlanBuild(executionPlanFailurePreviewInvalid, errors.New("preview builder returned an incomplete result"))
	}

	walletObservations := make([]wormstore.ExecutionPlanWalletObservation, 0, len(wallets))
	for _, wallet := range wallets {
		walletObservations = append(walletObservations, wallet.observation)
	}
	itemObservations, err := executionPlanItemObservations(catalogItems, result.Estimates)
	if err != nil {
		return nil, failExecutionPlanBuild(executionPlanFailurePreviewInvalid, err)
	}
	steps, totalCollateral, totalOpeningFee, totalUserFundsNeeded, err := executionPlanStepsAndTotals(result.Steps)
	if err != nil {
		return nil, failExecutionPlanBuild(executionPlanFailurePreviewInvalid, err)
	}
	if err := guard.setStage(ctx, executionPlanStageFinalizing, int64(len(steps))); err != nil {
		return nil, failExecutionPlanBuild(executionPlanFailureStoreUnavailable, err)
	}
	return &wormstore.MarkExecutionPlanReadyRequest{
		PlanID:               plan.ID,
		WorkerID:             guard.workerID,
		Wallets:              walletObservations,
		Items:                itemObservations,
		Steps:                steps,
		TotalCollateral:      totalCollateral,
		TotalOpeningFee:      totalOpeningFee,
		TotalUserFundsNeeded: totalUserFundsNeeded,
		Now:                  timeNowUTC(),
	}, nil
}

func (s *Service) classifyExecutionPreviewBuildFailure(
	ctx context.Context,
	err error,
	wallets []executionPlanWalletContext,
) string {
	var walletFailure *ExecutionPreviewWalletExposurePhaseError
	if errors.As(err, &walletFailure) {
		if isWormAuthenticationError(walletFailure.Err) {
			index := int(walletFailure.WalletOrdinal) - 1
			if index < 0 || index >= len(wallets) || wallets[index].input.WalletID != walletFailure.WalletID ||
				wallets[index].activeCredentialID <= 0 {
				return executionPlanFailurePreviewInvalid
			}
			wallet := wallets[index]
			s.markWalletReconnectRequired(
				ctx,
				wallet.input.WalletID,
				wallet.input.Address,
				wallet.activeCredentialID,
			)
			return executionPlanFailureCredentialUnavailable
		}
		if classifyWormError(walletFailure.Err) == wormErrorInvalidResponse {
			return executionPlanFailureWormResponseInvalid
		}
		return executionPlanFailureWormReadUnavailable
	}

	var estimateFailure *ExecutionPreviewEstimatePhaseError
	if errors.As(err, &estimateFailure) {
		if classifyWormError(estimateFailure.Err) == wormErrorInvalidResponse {
			return executionPlanFailureWormResponseInvalid
		}
		// Estimate uses an explicitly unauthenticated client. A 401/403 from
		// this phase is a provider-read failure, never evidence that a stored
		// wallet credential needs reconnecting.
		return executionPlanFailureWormReadUnavailable
	}

	return executionPlanFailurePreviewInvalid
}

func (s *Service) readExecutionPlanCatalogs(
	ctx context.Context,
	items []wormstore.ExecutionPlanItem,
) ([]executionPlanCatalogItem, error) {
	if s.wormMarketsClientset == nil || s.wormMarketsClientset.WormMarkets() == nil {
		return nil, failExecutionPlanBuild(executionPlanFailureMarketsUnavailable, errors.New("Worm Markets client is not configured"))
	}
	events := make(map[string]*wormmarketsapiclient.OrderEventCatalog)
	for _, item := range items {
		if _, exists := events[item.EventConditionID]; exists {
			continue
		}
		response, err := s.wormMarketsClientset.WormMarkets().GetOrderEventCatalog(ctx, &wormmarketsapiclient.GetOrderEventCatalogRequest{
			EventConditionId: item.EventConditionID,
		})
		if err != nil {
			failureCode := executionPlanFailureMarketsUnavailable
			if status.Code(err) == codes.InvalidArgument {
				failureCode = executionPlanFailureMarketsInvalid
			}
			return nil, failExecutionPlanBuild(failureCode, fmt.Errorf("read event catalog: %w", err))
		}
		if response == nil || response.GetEvent() == nil || response.GetFetchedAt() <= 0 ||
			response.GetEvent().GetEventConditionId() != item.EventConditionID {
			return nil, failExecutionPlanBuild(executionPlanFailureMarketsInvalid, errors.New("Worm Markets returned an invalid event catalog"))
		}
		events[item.EventConditionID] = response.GetEvent()
	}
	marketIndexes := make(map[string]map[string]*wormmarketsapiclient.OrderEventCatalogMarket, len(events))
	for eventConditionID, catalog := range events {
		markets, err := indexExecutionPlanCatalogMarkets(catalog)
		if err != nil {
			return nil, failExecutionPlanBuild(executionPlanFailureMarketsInvalid, err)
		}
		marketIndexes[eventConditionID] = markets
	}

	result := make([]executionPlanCatalogItem, 0, len(items))
	for _, item := range items {
		market := marketIndexes[item.EventConditionID][item.MarketConditionID]
		if market == nil {
			result = append(result, unavailableExecutionPlanCatalogItem(item, executionPlanMarketMissing))
			continue
		}
		selectedOutcome, err := executionPlanSelectedOutcome(market, item.IsYes)
		if err != nil {
			return nil, failExecutionPlanBuild(executionPlanFailureMarketsInvalid, err)
		}
		backend := strings.ToLower(strings.TrimSpace(market.GetBackend()))
		if backend == "" {
			backend = executionPlanUnknownBackend
		}
		funds, supportedBackend := executionPreviewFundsForBackend(backend)
		if !supportedBackend {
			funds = "0"
		}
		selectable := market.GetUnavailableCode() == "" && selectedOutcome.GetSelectable() && supportedBackend
		unavailableCode := ""
		if selectable {
			maxLeverage, parseErr := parseExecutionPreviewDecimal(selectedOutcome.GetMaxLeverage())
			if parseErr != nil || maxLeverage.Compare(executionPreviewOne()) < 0 {
				return nil, failExecutionPlanBuild(executionPlanFailureMarketsInvalid, errors.New("Worm Markets returned invalid one-times leverage availability"))
			}
		} else {
			unavailableCode = strings.TrimSpace(selectedOutcome.GetUnavailableCode())
			if unavailableCode == "" {
				unavailableCode = strings.TrimSpace(market.GetUnavailableCode())
			}
			if unavailableCode == "" && !supportedBackend {
				unavailableCode = "BACKEND_UNSUPPORTED"
			}
			if unavailableCode == "" {
				unavailableCode = executionPlanMarketUnavailable
			}
			if !isCanonicalNonEmptyString(unavailableCode) || len(unavailableCode) > 100 {
				return nil, failExecutionPlanBuild(executionPlanFailureMarketsInvalid, errors.New("Worm Markets returned an invalid unavailable code"))
			}
		}
		result = append(result, executionPlanCatalogItem{
			input: ExecutionPreviewItemInput{
				EventConditionID:  item.EventConditionID,
				MarketConditionID: item.MarketConditionID,
				IsYes:             item.IsYes,
				Backend:           backend,
				Funds:             funds,
				Selectable:        selectable,
				UnavailableCode:   unavailableCode,
			},
			observation: wormstore.ExecutionPlanItemObservation{
				Ordinal:    item.Ordinal,
				Backend:    backend,
				Funds:      funds,
				Leverage:   "1",
				State:      executionPlanMarketUnavailable,
				ReasonCode: unavailableCode,
			},
		})
	}
	return result, nil
}

func indexExecutionPlanCatalogMarkets(
	catalog *wormmarketsapiclient.OrderEventCatalog,
) (map[string]*wormmarketsapiclient.OrderEventCatalogMarket, error) {
	if catalog == nil || !validExecutionPreviewConditionID(catalog.GetEventConditionId()) {
		return nil, errors.New("Worm Markets returned an invalid event")
	}
	result := make(map[string]*wormmarketsapiclient.OrderEventCatalogMarket, len(catalog.GetMarkets()))
	for _, market := range catalog.GetMarkets() {
		if market == nil || !validExecutionPreviewConditionID(market.GetMarketConditionId()) ||
			market.GetEventConditionId() != catalog.GetEventConditionId() {
			return nil, errors.New("Worm Markets returned an invalid market")
		}
		if _, exists := result[market.GetMarketConditionId()]; exists {
			return nil, errors.New("Worm Markets returned a duplicate market")
		}
		backend := strings.TrimSpace(market.GetBackend())
		if backend != market.GetBackend() || backend != strings.ToLower(backend) {
			return nil, errors.New("Worm Markets returned a non-canonical market backend")
		}
		result[market.GetMarketConditionId()] = market
	}
	return result, nil
}

func executionPlanSelectedOutcome(
	market *wormmarketsapiclient.OrderEventCatalogMarket,
	isYes bool,
) (*wormmarketsapiclient.OrderEventCatalogOutcome, error) {
	if market == nil || len(market.GetOutcomes()) != 2 {
		return nil, errors.New("Worm Markets returned invalid market outcomes")
	}
	var selected *wormmarketsapiclient.OrderEventCatalogOutcome
	seen := map[bool]bool{}
	for _, outcome := range market.GetOutcomes() {
		if outcome == nil || seen[outcome.GetIsYes()] || !isCanonicalNonEmptyString(outcome.GetLabel()) {
			return nil, errors.New("Worm Markets returned invalid market outcomes")
		}
		seen[outcome.GetIsYes()] = true
		if outcome.GetSelectable() == (outcome.GetUnavailableCode() != "") ||
			outcome.GetUnavailableCode() != strings.TrimSpace(outcome.GetUnavailableCode()) {
			return nil, errors.New("Worm Markets returned inconsistent market outcomes")
		}
		if outcome.GetIsYes() == isYes {
			selected = outcome
		}
	}
	if selected == nil || !seen[true] || !seen[false] {
		return nil, errors.New(executionPlanOutcomeUnavailable)
	}
	return selected, nil
}

func unavailableExecutionPlanCatalogItem(item wormstore.ExecutionPlanItem, reasonCode string) executionPlanCatalogItem {
	return executionPlanCatalogItem{
		input: ExecutionPreviewItemInput{
			EventConditionID:  item.EventConditionID,
			MarketConditionID: item.MarketConditionID,
			IsYes:             item.IsYes,
			Backend:           executionPlanUnknownBackend,
			Funds:             "0",
			Selectable:        false,
			UnavailableCode:   reasonCode,
		},
		observation: wormstore.ExecutionPlanItemObservation{
			Ordinal:    item.Ordinal,
			Backend:    executionPlanUnknownBackend,
			Funds:      "0",
			Leverage:   "1",
			State:      executionPlanMarketUnavailable,
			ReasonCode: reasonCode,
		},
	}
}

func (s *Service) readExecutionPlanWalletConnections(
	ctx context.Context,
	wallets []wormstore.ExecutionPlanWallet,
) ([]executionPlanWalletContext, error) {
	refs := make([]wormstore.WalletReference, 0, len(wallets))
	for _, wallet := range wallets {
		refs = append(refs, wormstore.WalletReference{WalletID: wallet.WalletID, Address: wallet.Address})
	}
	snapshots, err := s.credentialStore.ListWalletConnectionSnapshots(ctx, refs)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, failExecutionPlanBuild(executionPlanFailureStoreUnavailable, err)
	}
	if err := validateStoredConnectionSnapshots(refs, snapshots); err != nil {
		return nil, failExecutionPlanBuild(executionPlanFailurePreviewInvalid, err)
	}
	result := make([]executionPlanWalletContext, 0, len(wallets))
	for index, snapshot := range snapshots {
		if snapshot.State != wormstore.ConnectionStateConnected {
			return nil, failExecutionPlanBuild(executionPlanFailureWalletNotConnected, fmt.Errorf("wallet %d is not connected", index+1))
		}
		credential := snapshot.ActiveCredential
		if credential == nil || credential.State != wormstore.CredentialStateActive || credential.Version <= 0 {
			return nil, failExecutionPlanBuild(executionPlanFailureCredentialUnavailable, fmt.Errorf("wallet %d has no active credential", index+1))
		}
		apiKey, apiSecret, err := s.credentialCipher.decrypt(
			credential.WalletID,
			credential.APIKeyCiphertext,
			credential.APISecretCiphertext,
		)
		if err != nil {
			return nil, failExecutionPlanBuild(executionPlanFailureCredentialUnavailable, err)
		}
		client, err := s.wormClientFactory.NewAuthenticatedClient(apiKey, apiSecret)
		if err != nil {
			return nil, failExecutionPlanBuild(executionPlanFailureCredentialUnavailable, err)
		}
		wallet := wallets[index]
		result = append(result, executionPlanWalletContext{
			input: ExecutionPreviewWalletInput{
				WalletID: wallet.WalletID,
				Address:  wallet.Address,
				Client:   client,
			},
			observation: wormstore.ExecutionPlanWalletObservation{
				Ordinal:               wallet.Ordinal,
				ConnectionState:       snapshot.State,
				ConnectionWarningCode: snapshot.WarningCode,
				ConnectedAt:           snapshot.ConnectedAt,
				CredentialVersion:     credential.Version,
			},
			activeCredentialID: credential.ID,
		})
	}
	return result, nil
}

func (s *Service) readExecutionPlanWalletBalances(ctx context.Context, wallets []executionPlanWalletContext) error {
	refs := make([]WalletBalanceReference, 0, len(wallets))
	for _, wallet := range wallets {
		refs = append(refs, WalletBalanceReference{WalletID: wallet.input.WalletID, Address: wallet.input.Address})
	}
	results, err := s.adapter.BatchGetBalances(ctx, refs)
	if err != nil {
		return failExecutionPlanBuild(executionPlanFailureBalanceUnavailable, err)
	}
	if len(results) != len(wallets) {
		return failExecutionPlanBuild(executionPlanFailureBalanceUnavailable, errors.New("Solana balance provider returned an incomplete batch"))
	}
	for index, balance := range results {
		wallet := &wallets[index]
		if balance.WalletID != wallet.input.WalletID || balance.Address != wallet.input.Address {
			return failExecutionPlanBuild(executionPlanFailureBalanceUnavailable, errors.New("Solana balance provider returned a mismatched wallet"))
		}
		if !validExecutionPlanBalanceObservation(balance.SOL, SolanaDecimals, false) ||
			!validExecutionPlanBalanceObservation(balance.USDC, USDCDecimals, true) ||
			!isCanonicalNonEmptyString(balance.Status) {
			return failExecutionPlanBuild(executionPlanFailureBalanceUnavailable, fmt.Errorf("wallet %d balance snapshot is invalid", index+1))
		}
		if balance.USDC.Availability != availabilityAvailable {
			return failExecutionPlanBuild(executionPlanFailureUSDCUnavailable, fmt.Errorf("wallet %d USDC balance is unavailable", index+1))
		}
		usdc, err := parseExecutionPreviewDecimal(balance.USDC.Amount)
		if err != nil {
			return failExecutionPlanBuild(executionPlanFailureBalanceUnavailable, fmt.Errorf("wallet %d USDC balance is invalid", index+1))
		}
		wallet.input.USDCBalance = usdc.String()
		wallet.input.SOLBalance = "0"
		if balance.SOL.Availability == availabilityAvailable {
			sol, parseErr := parseExecutionPreviewDecimal(balance.SOL.Amount)
			if parseErr != nil {
				return failExecutionPlanBuild(executionPlanFailureBalanceUnavailable, fmt.Errorf("wallet %d SOL balance is invalid", index+1))
			}
			wallet.input.SOLBalance = sol.String()
		}
		wallet.observation.SOLAtomicAmount = balance.SOL.AtomicAmount
		wallet.observation.SOLAmount = balance.SOL.Amount
		wallet.observation.SOLDecimals = balance.SOL.Decimals
		wallet.observation.SOLObservedSlot = balance.SOL.ObservedSlot
		wallet.observation.SOLAvailability = balance.SOL.Availability
		wallet.observation.SOLErrorCode = balance.SOL.ErrorCode
		wallet.observation.USDCMint = SolanaNativeUSDCMint
		wallet.observation.USDCAtomicAmount = balance.USDC.AtomicAmount
		wallet.observation.USDCAmount = balance.USDC.Amount
		wallet.observation.USDCDecimals = balance.USDC.Decimals
		wallet.observation.USDCObservedSlot = balance.USDC.ObservedSlot
		wallet.observation.USDCAvailability = balance.USDC.Availability
		wallet.observation.USDCErrorCode = balance.USDC.ErrorCode
		wallet.observation.USDCTokenAccountCount = balance.USDC.TokenAccountCount
		wallet.observation.Status = balance.Status
	}
	return nil
}

func validExecutionPlanBalanceObservation(observation BalanceObservation, expectedDecimals int32, token bool) bool {
	if observation.Decimals != expectedDecimals || observation.TokenAccountCount < 0 ||
		(!token && observation.TokenAccountCount != 0) {
		return false
	}
	switch observation.Availability {
	case availabilityAvailable:
		if observation.ErrorCode != "" || observation.ObservedSlot == 0 || !executionPlanUnsignedInteger(observation.AtomicAmount) {
			return false
		}
		_, err := parseExecutionPreviewDecimal(observation.Amount)
		return err == nil
	case availabilityUnavailable:
		return isCanonicalNonEmptyString(observation.ErrorCode) && observation.AtomicAmount == "" && observation.Amount == ""
	default:
		return false
	}
}

func executionPlanUnsignedInteger(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func executionPlanItemObservations(
	items []executionPlanCatalogItem,
	estimates []ExecutionPreviewMarketEstimate,
) ([]wormstore.ExecutionPlanItemObservation, error) {
	if len(items) != len(estimates) {
		return nil, errors.New("execution preview estimate count mismatch")
	}
	result := make([]wormstore.ExecutionPlanItemObservation, 0, len(items))
	for index, item := range items {
		estimate := estimates[index]
		if estimate.MarketOrdinal != int32(index+1) || estimate.MarketConditionID != item.input.MarketConditionID ||
			estimate.IsYes != item.input.IsYes || estimate.Funds != item.input.Funds {
			return nil, errors.New("execution preview estimate order mismatch")
		}
		observation := item.observation
		switch {
		case !item.input.Selectable:
			observation.State = executionPlanMarketUnavailable
		case estimate.RejectionCode != "":
			observation.State = ExecutionPreviewOutcomeEstimateRejected
			observation.ReasonCode = ExecutionPreviewOutcomeEstimateRejected
		default:
			observation.State = ExecutionPreviewOutcomeReady
			observation.ReasonCode = ""
		}
		observation.Estimate = wormstore.ExecutionPlanEstimate{
			AveragePrice:    estimate.AveragePrice,
			TotalShares:     estimate.TotalShares,
			TotalCost:       estimate.TotalCost,
			BestAsk:         estimate.BestAsk,
			WorstFillPrice:  estimate.WorstFillPrice,
			IsFullyFilled:   estimate.IsFullyFilled,
			FeeAmount:       estimate.FeeAmount,
			UserFundsNeeded: estimate.UserFundsNeeded,
		}
		if estimate.LiquidationPrice != nil {
			observation.Estimate.LiquidationPrice = *estimate.LiquidationPrice
		}
		result = append(result, observation)
	}
	return result, nil
}

func executionPlanStepsAndTotals(
	previewSteps []ExecutionPreviewStep,
) ([]wormstore.ExecutionPlanStep, string, string, string, error) {
	totalCollateral, _ := parseExecutionPreviewDecimal("0")
	totalOpeningFee, _ := parseExecutionPreviewDecimal("0")
	totalUserFundsNeeded, _ := parseExecutionPreviewDecimal("0")
	steps := make([]wormstore.ExecutionPlanStep, 0, len(previewSteps))
	for index, step := range previewSteps {
		if step.Ordinal != int64(index+1) || step.WalletOrdinal <= 0 || step.MarketOrdinal <= 0 {
			return nil, "", "", "", errors.New("execution preview step order is invalid")
		}
		disposition := wormstore.ExecutionPlanStepDispositionSkipped
		reasonCode := step.Outcome
		if step.Outcome == ExecutionPreviewOutcomeReady {
			disposition = wormstore.ExecutionPlanStepDispositionReady
			reasonCode = ""
			var err error
			totalCollateral, err = addExecutionPlanDecimal(totalCollateral, step.Funds)
			if err != nil {
				return nil, "", "", "", err
			}
			totalOpeningFee, err = addExecutionPlanDecimal(totalOpeningFee, step.FeeAmount)
			if err != nil {
				return nil, "", "", "", err
			}
			totalUserFundsNeeded, err = addExecutionPlanDecimal(totalUserFundsNeeded, step.UserFundsNeeded)
			if err != nil {
				return nil, "", "", "", err
			}
		}
		if disposition == wormstore.ExecutionPlanStepDispositionSkipped && !isCanonicalNonEmptyString(reasonCode) {
			return nil, "", "", "", errors.New("execution preview skipped step has no reason")
		}
		steps = append(steps, wormstore.ExecutionPlanStep{
			Ordinal:             step.Ordinal,
			WalletOrdinal:       step.WalletOrdinal,
			ItemOrdinal:         step.MarketOrdinal,
			Disposition:         disposition,
			ReasonCode:          reasonCode,
			ProjectedUSDCBefore: step.USDCBalanceBefore,
			ProjectedUSDCAfter:  step.USDCBalanceAfter,
		})
	}
	return steps, totalCollateral.String(), totalOpeningFee.String(), totalUserFundsNeeded.String(), nil
}

func addExecutionPlanDecimal(total decimalAmount, raw string) (decimalAmount, error) {
	value, err := parseExecutionPreviewDecimal(raw)
	if err != nil {
		return decimalAmount{}, err
	}
	left, right := executionPreviewAlignedCoefficients(total, value)
	result := decimalAmount{
		coefficient: new(big.Int).Add(left, right),
		scale:       executionPreviewMaxInt(total.scale, value.scale),
	}
	return parseExecutionPreviewDecimal(result.String())
}

type executionPlanLeaseGuard struct {
	service     *Service
	planID      string
	workerID    string
	cancelBuild context.CancelFunc

	stateMu   sync.Mutex
	stage     string
	completed int64
	callMu    sync.Mutex
	errMu     sync.Mutex
	err       error
	stopOnce  sync.Once
	stopFn    context.CancelFunc
	wg        sync.WaitGroup
}

func newExecutionPlanLeaseGuard(
	service *Service,
	planID string,
	workerID string,
	cancelBuild context.CancelFunc,
) *executionPlanLeaseGuard {
	return &executionPlanLeaseGuard{
		service:     service,
		planID:      planID,
		workerID:    workerID,
		cancelBuild: cancelBuild,
		stage:       "CLAIMED",
	}
}

func (g *executionPlanLeaseGuard) start(parent context.Context) {
	heartbeatCtx, cancel := context.WithCancel(parent)
	g.stopFn = cancel
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		ticker := time.NewTicker(executionPlanWorkerHeartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				if err := g.renew(heartbeatCtx); err != nil {
					g.setError(err)
					g.cancelBuild()
					return
				}
			}
		}
	}()
}

func (g *executionPlanLeaseGuard) setStage(ctx context.Context, stage string, completed int64) error {
	g.stateMu.Lock()
	g.stage = stage
	g.completed = completed
	g.stateMu.Unlock()
	if err := g.renew(ctx); err != nil {
		g.setError(err)
		g.cancelBuild()
		return err
	}
	return nil
}

func (g *executionPlanLeaseGuard) renew(ctx context.Context) error {
	g.callMu.Lock()
	defer g.callMu.Unlock()
	g.stateMu.Lock()
	stage := g.stage
	completed := g.completed
	g.stateMu.Unlock()
	now := timeNowUTC()
	_, err := g.service.credentialStore.UpdateExecutionPlanBuildProgress(ctx, wormstore.ExecutionPlanBuildProgress{
		PlanID:             g.planID,
		WorkerID:           g.workerID,
		BuildStage:         stage,
		CompletedStepCount: completed,
		LeaseExpiresAt:     now.Add(executionPlanWorkerLease),
		Now:                now,
	})
	g.service.recordCredentialStoreResult(err)
	return err
}

func (g *executionPlanLeaseGuard) setError(err error) {
	if err == nil {
		return
	}
	g.errMu.Lock()
	if g.err == nil {
		g.err = err
	}
	g.errMu.Unlock()
}

func (g *executionPlanLeaseGuard) stop() error {
	g.stopOnce.Do(func() {
		if g.stopFn != nil {
			g.stopFn()
		}
		g.wg.Wait()
	})
	g.errMu.Lock()
	defer g.errMu.Unlock()
	return g.err
}
