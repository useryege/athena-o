package wormtrading

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	executionWorkerPollInterval = time.Second
	executionWorkerClaimLease   = 45 * time.Second
	executionWorkerClaimRenewal = 10 * time.Second
	executionWorkerRecoverySize = int32(100)
	executionWorkerMaxTxBytes   = 64 << 10

	executionWorkerReasonInsufficientSOL         = "INSUFFICIENT_SOL"
	executionWorkerReasonMarketChanged           = "MARKET_CHANGED"
	executionWorkerReasonPlanInvalid             = "EXECUTION_PLAN_INVALID"
	executionWorkerReasonWalletUnavailable       = "WALLET_UNAVAILABLE"
	executionWorkerReasonWalletSignerInvalid     = "WALLET_SIGNER_INVALID_RESPONSE"
	executionWorkerReasonWalletSignerUnavailable = "WALLET_SIGNER_UNAVAILABLE"
	executionWorkerReasonWebAuthRejected         = "WORM_WEB_AUTH_REJECTED"
	executionWorkerReasonWebUnavailable          = "WORM_WEB_UNAVAILABLE"
	executionWorkerReasonWebResponseInvalid      = "WORM_WEB_INVALID_RESPONSE"
	executionWorkerReasonEdgeBlocked             = "WORM_EDGE_BLOCKED"
	executionWorkerReasonRateLimited             = "RATE_LIMITED"
	executionWorkerReasonOpenRejected            = "WORM_OPEN_REJECTED"
	executionWorkerReasonOpenOutcomeUnknown      = "OPEN_OUTCOME_UNKNOWN"
	executionWorkerReasonFinalizeRejected        = "WORM_FINALIZE_REJECTED"
	executionWorkerReasonFinalizeOutcomeUnknown  = "FINALIZE_OUTCOME_UNKNOWN"
	executionWorkerReasonProviderFailed          = "WORM_REQUEST_FAILED"
	executionWorkerReasonTransactionChanged      = "WORM_TRANSACTION_CHANGED"
	executionWorkerReasonPositionEvidenceInvalid = "OPEN_POSITION_EVIDENCE_AMBIGUOUS"
	executionWorkerReasonReconcileInconclusive   = "RECONCILIATION_INCONCLUSIVE"
	executionWorkerReasonReconciledOpenPosition  = "RECONCILED_OPEN_POSITION"
	executionWorkerReasonReconciledFailed        = "RECONCILED_PROVIDER_FAILED"
)

type executionOpenPositionMatch int

const (
	executionOpenPositionAbsent executionOpenPositionMatch = iota
	executionOpenPositionMatched
	executionOpenPositionAmbiguous
)

type executionOpenPositionEvidence struct {
	positionPubkey        string
	positionRequestPubkey string
	positionCreatedAt     time.Time
}

// executionWorkerTask contains only immutable Run snapshots and durable Step
// identity. Raw provider transactions, signatures, and JWTs never enter it.
type executionWorkerTask struct {
	recovery wormstore.RecoverableExecutionStep
	run      wormstore.ExecutionRun
	step     wormstore.ExecutionRunStep
	wallet   wormstore.ExecutionPlanWallet
	item     wormstore.ExecutionPlanItem
	claimID  string
	workerID string
	intent   []byte
}

type executionWorkerFailure struct {
	code  string
	cause error
}

func (e *executionWorkerFailure) Error() string {
	if e == nil || e.code == "" {
		return "execution worker failed"
	}
	return e.code
}

func (e *executionWorkerFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

type executionWebError struct {
	code           string
	httpStatus     int32
	providerCode   int32
	providerSlug   string
	structured401  bool
	definiteClient bool
	temporary      bool
	ambiguous      bool
}

type executionSignedPayload struct {
	mode               utilworm.WebFinalizeMode
	value              string
	transactionVersion string
	requiredSignatures int32
	signerIndex        int32
}

func (s *Service) wakeExecutionWorker() {
	if s == nil || s.executionWorkerWake == nil {
		return
	}
	select {
	case s.executionWorkerWake <- struct{}{}:
	default:
	}
}

func (s *Service) runExecutionWorker(ctx context.Context) {
	defer s.runWG.Done()
	workerID := "worm-execution-" + uuid.NewString()
	for {
		if ctx.Err() != nil {
			return
		}
		s.processRecoverableExecutionSteps(ctx, workerID)
		timer := time.NewTimer(executionWorkerPollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-s.executionWorkerWake:
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
		}
	}
}

func (s *Service) processRecoverableExecutionSteps(ctx context.Context, workerID string) {
	recoverable, err := s.credentialStore.ListRecoverableExecutionSteps(
		ctx,
		timeNowUTC(),
		executionWorkerRecoverySize,
	)
	s.recordCredentialStoreResult(err)
	if err != nil {
		if ctx.Err() == nil {
			log.WithField("error_code", executionPlanFailureStoreUnavailable).
				Warn("failed to list recoverable Worm execution steps")
		}
		return
	}
	for index := range recoverable {
		if ctx.Err() != nil {
			return
		}
		s.processRecoverableExecutionStep(ctx, workerID, recoverable[index])
	}
}

func (s *Service) processRecoverableExecutionStep(
	ctx context.Context,
	workerID string,
	recovery wormstore.RecoverableExecutionStep,
) {
	now := timeNowUTC()
	claimID := uuid.NewString()
	step, err := s.credentialStore.RecoverExecutionStep(ctx, wormstore.RecoverExecutionStepRequest{
		RunID:          recovery.RunID,
		StepOrdinal:    recovery.Step.Ordinal,
		ClaimID:        claimID,
		WorkerID:       workerID,
		LeaseExpiresAt: now.Add(executionWorkerClaimLease),
		Now:            now,
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		if !errors.Is(err, wormstore.ErrExecutionRunConflict) && ctx.Err() == nil {
			logExecutionWorkerFailure(recovery.RunID, recovery.Step.Ordinal, recovery.Step.State, executionPlanFailureStoreUnavailable)
		}
		return
	}
	if step == nil {
		logExecutionWorkerFailure(recovery.RunID, recovery.Step.Ordinal, recovery.Step.State, executionWorkerReasonPlanInvalid)
		return
	}

	guard := newExecutionWorkerClaimGuard(s, ctx, recovery.RunID, step.Ordinal, claimID, workerID)
	guard.start()
	task, taskErr := s.loadExecutionWorkerTask(guard.context(), recovery, *step, claimID, workerID)
	if taskErr == nil {
		taskErr = s.executeClaimedExecutionStep(guard, task)
	}
	guardErr := guard.stop()
	if taskErr == nil {
		taskErr = guardErr
	}
	if taskErr != nil && ctx.Err() == nil && !errors.Is(taskErr, context.Canceled) &&
		!errors.Is(taskErr, wormstore.ErrExecutionRunConflict) {
		code := executionWorkerFailureCode(taskErr)
		logExecutionWorkerFailure(recovery.RunID, step.Ordinal, step.State, code)
	}
	s.clearExecutionWebJWTIfRunTerminal(ctx, recovery.OwnerAccountID, recovery.RunID)
}

func (s *Service) loadExecutionWorkerTask(
	ctx context.Context,
	recovery wormstore.RecoverableExecutionStep,
	step wormstore.ExecutionRunStep,
	claimID string,
	workerID string,
) (*executionWorkerTask, error) {
	run, err := s.credentialStore.GetExecutionRun(ctx, recovery.OwnerAccountID, recovery.RunID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, &executionWorkerFailure{code: executionPlanFailureStoreUnavailable, cause: err}
	}
	if run == nil || run.ID != recovery.RunID || run.OwnerAccountID != recovery.OwnerAccountID ||
		run.CurrentStepOrdinal != step.Ordinal || run.CurrentStep == nil ||
		run.CurrentStep.ID != step.ID || run.CurrentStep.Ordinal != step.Ordinal ||
		step.ClaimID != claimID || step.ClaimOwner != workerID ||
		step.WalletOrdinal <= 0 || int(step.WalletOrdinal) > len(run.Wallets) ||
		step.ItemOrdinal <= 0 || int(step.ItemOrdinal) > len(run.Items) {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	wallet := run.Wallets[step.WalletOrdinal-1]
	item := run.Items[step.ItemOrdinal-1]
	if wallet.Ordinal != step.WalletOrdinal || item.Ordinal != step.ItemOrdinal ||
		wallet.WalletID <= 0 || !validExecutionPreviewConditionID(item.MarketConditionID) ||
		!validExecutionPreviewConditionID(item.EventConditionID) {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	intent, err := executionWorkerIntentDigest(*run, step, wallet, item)
	if err != nil {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid, cause: err}
	}
	return &executionWorkerTask{
		recovery: recovery,
		run:      run.Clone(),
		step:     step.Clone(),
		wallet:   wallet,
		item:     item,
		claimID:  claimID,
		workerID: workerID,
		intent:   intent,
	}, nil
}

func executionWorkerIntentDigest(
	run wormstore.ExecutionRun,
	step wormstore.ExecutionRunStep,
	wallet wormstore.ExecutionPlanWallet,
	item wormstore.ExecutionPlanItem,
) ([]byte, error) {
	payload := struct {
		Version           int64  `json:"version"`
		RunID             string `json:"run_id"`
		StepID            string `json:"step_id"`
		PlanVersion       int64  `json:"plan_version"`
		PlanDigestSHA256  []byte `json:"plan_digest_sha256"`
		WalletID          int64  `json:"wallet_id"`
		WalletAddress     string `json:"wallet_address"`
		MarketConditionID string `json:"market_condition_id"`
		IsYes             bool   `json:"is_yes"`
		Backend           string `json:"backend"`
		Funds             string `json:"funds"`
		Leverage          string `json:"leverage"`
	}{
		Version: 1, RunID: run.ID, StepID: step.ID, PlanVersion: run.PlanVersion,
		PlanDigestSHA256: append([]byte(nil), run.PlanDigestSHA256...),
		WalletID:         wallet.WalletID, WalletAddress: wallet.Address,
		MarketConditionID: item.MarketConditionID, IsYes: item.IsYes,
		Backend: item.Backend, Funds: item.Funds, Leverage: item.Leverage,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(encoded)
	return digest[:], nil
}

func (s *Service) executeClaimedExecutionStep(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
) error {
	for guard.context().Err() == nil {
		var (
			next *wormstore.ExecutionRunStep
			err  error
		)
		switch task.step.State {
		case wormstore.ExecutionStepStatePreflighting:
			next, err = s.executeFreshPreflight(guard, task)
		case wormstore.ExecutionStepStateOpening:
			next, err = s.executeWormOpen(guard, task)
		case wormstore.ExecutionStepStateOpened, wormstore.ExecutionStepStateSigning:
			next, err = s.executeWormSigning(guard, task)
		case wormstore.ExecutionStepStateFinalizing:
			next, err = s.executeWormFinalize(guard, task)
		case wormstore.ExecutionStepStateAwaitingCompletion:
			next, err = s.pollWormPositionRequest(guard, task)
		case wormstore.ExecutionStepStateOutcomeUnknown:
			next, err = s.reconcileWormExecutionStep(guard, task)
		default:
			return nil
		}
		if err != nil {
			return err
		}
		if next == nil {
			return nil
		}
		task.step = next.Clone()
		switch task.step.State {
		case wormstore.ExecutionStepStateOpening, wormstore.ExecutionStepStateOpened,
			wormstore.ExecutionStepStateSigning, wormstore.ExecutionStepStateFinalizing:
			// These phases retain the same durable claim and may safely continue
			// inside this worker turn.
		default:
			// Awaiting completion has a durable next_poll_at and no active claim;
			// outcome unknown requires a new explicit reconcile command; terminal
			// states must return control to the browser before another Step.
			return nil
		}
	}
	return guard.context().Err()
}

func logExecutionWorkerFailure(runID string, stepOrdinal int64, phase wormstore.ExecutionStepState, code string) {
	log.WithFields(log.Fields{
		"run_id":       runID,
		"step_ordinal": stepOrdinal,
		"phase":        phase,
		"error_code":   code,
	}).Warn("Worm execution step worker stopped")
}

func executionWorkerFailureCode(err error) string {
	var failure *executionWorkerFailure
	if errors.As(err, &failure) && failure.code != "" {
		return failure.code
	}
	if errors.Is(err, context.Canceled) {
		return wormErrorCancelled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return wormErrorTimeout
	}
	return executionWorkerReasonWebUnavailable
}

type executionWorkerClaimGuard struct {
	service      *Service
	runID        string
	stepOrdinal  int64
	claimID      string
	workerID     string
	ctx          context.Context
	cancel       context.CancelFunc
	stopOnce     sync.Once
	stopCh       chan struct{}
	doneCh       chan struct{}
	mu           sync.Mutex
	renewalError error
}

func newExecutionWorkerClaimGuard(
	service *Service,
	parent context.Context,
	runID string,
	stepOrdinal int64,
	claimID string,
	workerID string,
) *executionWorkerClaimGuard {
	ctx, cancel := context.WithCancel(parent)
	return &executionWorkerClaimGuard{
		service: service, runID: runID, stepOrdinal: stepOrdinal,
		claimID: claimID, workerID: workerID, ctx: ctx, cancel: cancel,
		stopCh: make(chan struct{}), doneCh: make(chan struct{}),
	}
}

func (g *executionWorkerClaimGuard) context() context.Context { return g.ctx }

func (g *executionWorkerClaimGuard) start() {
	go func() {
		defer close(g.doneCh)
		ticker := time.NewTicker(executionWorkerClaimRenewal)
		defer ticker.Stop()
		for {
			select {
			case <-g.stopCh:
				return
			case <-g.ctx.Done():
				return
			case <-ticker.C:
				now := timeNowUTC()
				_, err := g.service.credentialStore.RenewExecutionStepClaim(g.ctx, wormstore.RenewExecutionStepClaimRequest{
					RunID: g.runID, StepOrdinal: g.stepOrdinal, ClaimID: g.claimID,
					WorkerID: g.workerID, LeaseExpiresAt: now.Add(executionWorkerClaimLease), Now: now,
				})
				g.service.recordCredentialStoreResult(err)
				if err != nil {
					g.mu.Lock()
					g.renewalError = err
					g.mu.Unlock()
					g.cancel()
					return
				}
			}
		}
	}()
}

func (g *executionWorkerClaimGuard) stopRenewal() error {
	g.stopOnce.Do(func() { close(g.stopCh) })
	<-g.doneCh
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.renewalError
}

func (g *executionWorkerClaimGuard) stop() error {
	err := g.stopRenewal()
	g.cancel()
	return err
}

func (g *executionWorkerClaimGuard) expireSoon() {
	now := timeNowUTC()
	_, err := g.service.credentialStore.RenewExecutionStepClaim(g.ctx, wormstore.RenewExecutionStepClaimRequest{
		RunID: g.runID, StepOrdinal: g.stepOrdinal, ClaimID: g.claimID,
		WorkerID: g.workerID, LeaseExpiresAt: now.Add(time.Second), Now: now,
	})
	g.service.recordCredentialStoreResult(err)
}

func (s *Service) executeFreshPreflight(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
) (*wormstore.ExecutionRunStep, error) {
	ctx := guard.context()
	if task.item.State != ExecutionPreviewOutcomeReady || task.item.Leverage != "1" {
		return s.completeExecutionPreflight(guard, task, wormstore.ExecutionStepStateFailed,
			executionWorkerReasonPlanInvalid, wormstore.ExecutionStepScopeCurrent)
	}
	expectedFunds, supportedBackend := executionPreviewFundsForBackend(task.item.Backend)
	funds, fundsErr := parseExecutionPreviewDecimal(task.item.Funds)
	maximumFunds, _ := parseExecutionPreviewDecimal("10")
	if !supportedBackend || task.item.Funds != expectedFunds || fundsErr != nil || funds.IsZero() ||
		funds.Compare(maximumFunds) > 0 {
		return s.completeExecutionPreflight(guard, task, wormstore.ExecutionStepStateFailed,
			executionWorkerReasonPlanInvalid, wormstore.ExecutionStepScopeCurrent)
	}

	catalogs, err := s.readExecutionPlanCatalogs(ctx, []wormstore.ExecutionPlanItem{task.item})
	if err != nil {
		switch executionPlanFailureCode(err) {
		case executionPlanFailureMarketsInvalid:
			return s.completeExecutionPreflight(guard, task, wormstore.ExecutionStepStateSkipped,
				executionWorkerReasonMarketChanged, wormstore.ExecutionStepScopeRemainingMarket)
		default:
			return s.pauseClaimedExecution(guard, task, executionPlanFailureMarketsUnavailable, err)
		}
	}
	if len(catalogs) != 1 || catalogs[0].input.MarketConditionID != task.item.MarketConditionID ||
		catalogs[0].input.IsYes != task.item.IsYes || catalogs[0].input.Backend != task.item.Backend ||
		catalogs[0].input.Funds != task.item.Funds {
		return s.completeExecutionPreflight(guard, task, wormstore.ExecutionStepStateSkipped,
			executionWorkerReasonMarketChanged, wormstore.ExecutionStepScopeRemainingMarket)
	}
	if !catalogs[0].input.Selectable {
		reason := strings.TrimSpace(catalogs[0].input.UnavailableCode)
		if reason == "" {
			reason = ExecutionPreviewOutcomeMarketUnavailable
		}
		return s.completeExecutionPreflight(guard, task, wormstore.ExecutionStepStateSkipped,
			reason, wormstore.ExecutionStepScopeRemainingMarket)
	}

	wallets, err := s.readExecutionPlanWalletConnections(ctx, []wormstore.ExecutionPlanWallet{task.wallet})
	if err != nil {
		switch executionPlanFailureCode(err) {
		case executionPlanFailureWalletNotConnected, executionPlanFailureCredentialUnavailable:
			return s.completeExecutionPreflight(guard, task, wormstore.ExecutionStepStateSkipped,
				executionWorkerReasonWalletUnavailable, wormstore.ExecutionStepScopeRemainingWallet)
		default:
			return s.pauseClaimedExecution(guard, task, executionPlanFailureStoreUnavailable, err)
		}
	}
	if len(wallets) != 1 || wallets[0].input.WalletID != task.wallet.WalletID ||
		wallets[0].input.Address != task.wallet.Address {
		return s.completeExecutionPreflight(guard, task, wormstore.ExecutionStepStateFailed,
			executionWorkerReasonPlanInvalid, wormstore.ExecutionStepScopeCurrent)
	}
	if err := s.readExecutionPlanWalletBalances(ctx, wallets); err != nil {
		return s.pauseClaimedExecution(guard, task, executionPlanFailureBalanceUnavailable, err)
	}
	sol, solErr := parseExecutionPreviewDecimal(wallets[0].input.SOLBalance)
	if wallets[0].observation.SOLAvailability != availabilityAvailable || solErr != nil {
		return s.pauseClaimedExecution(guard, task, executionPlanFailureBalanceUnavailable,
			errors.New("fresh SOL balance is unavailable"))
	}
	if sol.IsZero() {
		return s.completeExecutionPreflight(guard, task, wormstore.ExecutionStepStateSkipped,
			executionWorkerReasonInsufficientSOL, wormstore.ExecutionStepScopeRemainingWallet)
	}

	estimateClient, err := s.wormClientFactory.NewUnauthenticatedClient()
	if err != nil {
		return s.pauseClaimedExecution(guard, task, executionPlanFailureWormReadUnavailable, err)
	}
	builder, err := NewExecutionPreviewBuilder(estimateClient)
	if err != nil {
		return s.pauseClaimedExecution(guard, task, executionPlanFailurePreviewInvalid, err)
	}
	result, err := builder.Build(ctx, ExecutionPreviewInput{
		Wallets:         []ExecutionPreviewWalletInput{wallets[0].input},
		PreflightChecks: task.run.PreflightChecks,
		Items: []ExecutionPreviewItemInput{{
			EventConditionID: task.item.EventConditionID, MarketConditionID: task.item.MarketConditionID,
			IsYes: task.item.IsYes, Backend: task.item.Backend, Funds: task.item.Funds,
			Selectable: true,
		}},
	})
	s.wormCapabilities.recordWormResult(err)
	if err != nil {
		return s.pauseClaimedExecution(guard, task, s.classifyExecutionPreviewBuildFailure(ctx, err, wallets), err)
	}
	if result == nil || len(result.Steps) != 1 || len(result.Estimates) != 1 {
		return s.pauseClaimedExecution(guard, task, executionPlanFailureWormResponseInvalid,
			errors.New("fresh preflight returned an incomplete result"))
	}
	previewStep := result.Steps[0]
	if previewStep.WalletID != task.wallet.WalletID ||
		previewStep.MarketConditionID != task.item.MarketConditionID || previewStep.IsYes != task.item.IsYes ||
		previewStep.Funds != task.item.Funds {
		return s.pauseClaimedExecution(guard, task, executionPlanFailureWormResponseInvalid,
			errors.New("fresh preflight returned a mismatched result"))
	}
	switch previewStep.Outcome {
	case ExecutionPreviewOutcomeMarketPositionExists, ExecutionPreviewOutcomeWalletRequestInFlight:
		return s.completeExecutionPreflight(guard, task, wormstore.ExecutionStepStateSkipped,
			previewStep.Outcome, wormstore.ExecutionStepScopeRemainingWallet, previewStep.AdvisoryCodes...)
	case ExecutionPreviewOutcomeLiquidityInsufficient:
		return s.completeExecutionPreflight(guard, task, wormstore.ExecutionStepStateSkipped,
			ExecutionPreviewOutcomeLiquidityInsufficient, wormstore.ExecutionStepScopeCurrent, previewStep.AdvisoryCodes...)
	case ExecutionPreviewOutcomeMarketUnavailable, ExecutionPreviewOutcomeEstimateRejected:
		reason := previewStep.ReasonCode
		if reason == "" {
			reason = previewStep.Outcome
		}
		return s.completeExecutionPreflight(guard, task, wormstore.ExecutionStepStateSkipped,
			reason, wormstore.ExecutionStepScopeRemainingMarket, previewStep.AdvisoryCodes...)
	case ExecutionPreviewOutcomeInsufficientUSDC, ExecutionPreviewOutcomeSkippedAfterInsufficientUSDC:
		return s.completeExecutionPreflight(guard, task, wormstore.ExecutionStepStateSkipped,
			ExecutionPreviewOutcomeInsufficientUSDC, wormstore.ExecutionStepScopeRemainingWallet, previewStep.AdvisoryCodes...)
	case ExecutionPreviewOutcomeReady:
	default:
		return s.pauseClaimedExecution(guard, task, executionPlanFailureWormResponseInvalid,
			errors.New("fresh preflight returned an unsupported outcome"))
	}

	// Establishing the non-durable Web session before the OPENING transition
	// lets a definite sign-in rejection remain a preflight Wallet result. On
	// restart, an OPENING Step may log in again, but it is never allowed to
	// replay a dispatched Open attempt.
	if _, err := s.executionWebJWT(ctx, task); err != nil {
		code := executionWorkerFailureCode(err)
		if executionWorkerFailureIsDeterministic(err) {
			return s.completeExecutionPreflight(guard, task, wormstore.ExecutionStepStateSkipped,
				code, wormstore.ExecutionStepScopeRemainingWallet, previewStep.AdvisoryCodes...)
		}
		return s.pauseClaimedExecution(guard, task, code, err)
	}
	// The preview snapshot above protects review consistency. This second,
	// authoritative read is deliberately after every other preflight operation
	// and Web login, immediately before the Step can enter OPENING.
	guardResult, err := fetchExecutionPreviewWalletExposure(ctx, wallets[0].input.Client)
	s.wormCapabilities.recordWormResult(err)
	if err != nil {
		return s.pauseClaimedExecution(guard, task, executionPlanFailureWormReadUnavailable, err)
	}
	if reason, blocked := executionPreOpenGuard(guardResult, task.item.MarketConditionID); blocked {
		return s.completeExecutionPreflight(
			guard,
			task,
			wormstore.ExecutionStepStateSkipped,
			reason,
			wormstore.ExecutionStepScopeRemainingWallet,
			previewStep.AdvisoryCodes...,
		)
	}
	return s.completeExecutionPreflight(
		guard,
		task,
		wormstore.ExecutionStepStateOpening,
		"",
		"",
		previewStep.AdvisoryCodes...,
	)
}

func executionPreOpenGuard(
	exposure executionPreviewWalletExposure,
	marketConditionID string,
) (string, bool) {
	if _, exists := firstExecutionPreviewMarketPosition(exposure.market(marketConditionID)); exists {
		return ExecutionPreviewOutcomeMarketPositionExists, true
	}
	if _, exists := exposure.firstRequest(); exists {
		return ExecutionPreviewOutcomeWalletRequestInFlight, true
	}
	return "", false
}

func (s *Service) readExecutionWalletExposure(
	ctx context.Context,
	task *executionWorkerTask,
) (executionPreviewWalletExposure, error) {
	wallets, err := s.readExecutionPlanWalletConnections(
		ctx,
		[]wormstore.ExecutionPlanWallet{task.wallet},
	)
	if err != nil {
		return executionPreviewWalletExposure{}, err
	}
	if len(wallets) != 1 || wallets[0].input.WalletID != task.wallet.WalletID ||
		wallets[0].input.Address != task.wallet.Address {
		return executionPreviewWalletExposure{}, errors.New("execution wallet exposure identity is invalid")
	}
	exposure, err := fetchExecutionPreviewWalletExposure(ctx, wallets[0].input.Client)
	s.wormCapabilities.recordWormResult(err)
	if err != nil {
		return executionPreviewWalletExposure{}, err
	}
	return exposure, nil
}

func (s *Service) recordExecutionPreOpenSkip(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	reason string,
) (*wormstore.ExecutionRunStep, error) {
	if reason != ExecutionPreviewOutcomeMarketPositionExists &&
		reason != ExecutionPreviewOutcomeWalletRequestInFlight {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	if err := guard.stopRenewal(); err != nil {
		return nil, err
	}
	step, err := s.credentialStore.RecordExecutionProviderObservation(
		guard.context(),
		wormstore.RecordExecutionProviderObservationRequest{
			RunID: task.run.ID, StepOrdinal: task.step.Ordinal, CommandID: task.recovery.CommandID,
			ClaimID: task.claimID, ExpectedState: wormstore.ExecutionStepStateOpening,
			NextState: wormstore.ExecutionStepStateSkipped, ReasonCode: reason,
			SkipScope: wormstore.ExecutionStepScopeRemainingWallet, Now: timeNowUTC(),
		},
	)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	return step, nil
}

func (s *Service) completeExecutionPreflight(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	nextState wormstore.ExecutionStepState,
	reason string,
	scope wormstore.ExecutionStepScope,
	advisoryCodes ...string,
) (*wormstore.ExecutionRunStep, error) {
	if nextState != wormstore.ExecutionStepStateOpening {
		if err := guard.stopRenewal(); err != nil {
			return nil, err
		}
	}
	step, err := s.credentialStore.CompleteExecutionPreflight(guard.context(), wormstore.CompleteExecutionPreflightRequest{
		RunID: task.run.ID, StepOrdinal: task.step.Ordinal, CommandID: task.recovery.CommandID,
		ClaimID: task.claimID, NextState: nextState, ReasonCode: reason, SkipScope: scope,
		AdvisoryCodes: append([]string(nil), advisoryCodes...), Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	return step, nil
}

func (s *Service) pauseClaimedExecution(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	code string,
	cause error,
) (*wormstore.ExecutionRunStep, error) {
	if code == "" {
		code = executionWorkerReasonWebUnavailable
	}
	if err := guard.stopRenewal(); err != nil {
		return nil, err
	}
	_, err := s.credentialStore.PauseExecutionRunForFailure(guard.context(), wormstore.PauseExecutionRunForFailureRequest{
		RunID: task.run.ID, PauseCode: code, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil && !errors.Is(err, wormstore.ErrExecutionRunConflict) {
		return nil, err
	}
	guard.expireSoon()
	return nil, &executionWorkerFailure{code: code, cause: cause}
}

func executionWorkerFailureIsDeterministic(err error) bool {
	var webFailure *executionWorkerFailure
	if errors.As(err, &webFailure) {
		switch webFailure.code {
		case executionWorkerReasonWebAuthRejected, executionWorkerReasonWalletUnavailable:
			return true
		}
	}
	return false
}

func (s *Service) executionWebJWT(ctx context.Context, task *executionWorkerTask) (string, error) {
	key := executionWebJWTKey(task.run.ID, task.wallet.WalletID)
	s.wormWebJWTMu.Lock()
	cached := strings.TrimSpace(s.wormWebJWT[key])
	s.wormWebJWTMu.Unlock()
	if cached != "" {
		return cached, nil
	}

	challenge, err := s.wormWebClient.GetSignInChallenge(ctx, task.wallet.Address)
	if err != nil {
		return "", executionWorkerWebFailure(err)
	}
	if challenge == nil || strings.TrimSpace(challenge.Nonce) == "" {
		return "", &executionWorkerFailure{code: executionWorkerReasonWebResponseInvalid}
	}
	message := utilworm.BuildWebSignInMessage(task.wallet.Address, challenge.Nonce, timeNowUTC())
	messageDigest := sha256.Sum256([]byte(message))
	signerCtx, cancel := context.WithTimeout(ctx, s.wormPositionBudget)
	signed, err := s.walletSignerClientset.Signer().SignWormWebSignInMessage(
		signerCtx,
		&walletapiclient.SignWormWebSignInMessageRequest{
			Id: task.wallet.WalletID, RequesterAccountId: task.run.OwnerAccountID,
			ExpectedAddress: task.wallet.Address, Nonce: challenge.Nonce, Message: message,
			ExpectedMessageSha256: messageDigest[:], ExecutionRunId: task.run.ID,
			ExecutionStepId: task.step.ID, IntentSha256: append([]byte(nil), task.intent...),
		},
	)
	cancel()
	if err != nil {
		return "", executionWorkerSignerFailure(err)
	}
	if signed == nil || strings.TrimSpace(signed.GetSignature()) == "" ||
		!bytes.Equal(signed.GetMessageSha256(), messageDigest[:]) ||
		signed.GetExecutionRunId() != task.run.ID || signed.GetExecutionStepId() != task.step.ID ||
		!bytes.Equal(signed.GetIntentSha256(), task.intent) {
		return "", &executionWorkerFailure{code: executionWorkerReasonWalletSignerInvalid}
	}
	response, err := s.wormWebClient.SignIn(ctx, utilworm.WebSignInRequest{
		Message: message, Signature: signed.GetSignature(), Address: task.wallet.Address,
		Nonce: challenge.Nonce,
	})
	if err != nil {
		return "", executionWorkerWebFailure(err)
	}
	if response == nil || strings.TrimSpace(response.AccessToken) == "" {
		return "", &executionWorkerFailure{code: executionWorkerReasonWebResponseInvalid}
	}
	token := strings.TrimSpace(response.AccessToken)
	s.wormWebJWTMu.Lock()
	if existing := strings.TrimSpace(s.wormWebJWT[key]); existing != "" {
		token = existing
	} else {
		s.wormWebJWT[key] = token
	}
	s.wormWebJWTMu.Unlock()
	return token, nil
}

func executionWebJWTKey(runID string, walletID int64) string {
	return fmt.Sprintf("%s:%d", runID, walletID)
}

func (s *Service) clearExecutionWebJWT(task *executionWorkerTask, token string) {
	key := executionWebJWTKey(task.run.ID, task.wallet.WalletID)
	s.wormWebJWTMu.Lock()
	if strings.TrimSpace(s.wormWebJWT[key]) == strings.TrimSpace(token) {
		delete(s.wormWebJWT, key)
	}
	s.wormWebJWTMu.Unlock()
}

func (s *Service) clearExecutionWebJWTRun(runID string) {
	prefix := strings.TrimSpace(runID) + ":"
	if prefix == ":" {
		return
	}
	s.wormWebJWTMu.Lock()
	for key := range s.wormWebJWT {
		if strings.HasPrefix(key, prefix) {
			delete(s.wormWebJWT, key)
		}
	}
	s.wormWebJWTMu.Unlock()
}

func (s *Service) clearExecutionWebJWTIfRunTerminal(ctx context.Context, ownerAccountID string, runID string) {
	run, err := s.credentialStore.GetExecutionRun(ctx, ownerAccountID, runID)
	s.recordCredentialStoreResult(err)
	if err != nil || run == nil {
		return
	}
	switch run.State {
	case wormstore.ExecutionRunStateCompleted,
		wormstore.ExecutionRunStateTerminated,
		wormstore.ExecutionRunStateFailed:
		s.clearExecutionWebJWTRun(run.ID)
	}
}

func executionWorkerSignerFailure(err error) error {
	code := status.Code(err)
	switch code {
	case codes.InvalidArgument:
		return &executionWorkerFailure{code: executionWorkerReasonWalletSignerInvalid, cause: err}
	case codes.NotFound, codes.PermissionDenied, codes.FailedPrecondition:
		return &executionWorkerFailure{code: executionWorkerReasonWalletUnavailable, cause: err}
	case codes.OK:
		return &executionWorkerFailure{code: executionWorkerReasonWalletSignerInvalid, cause: err}
	default:
		return &executionWorkerFailure{code: executionWorkerReasonWalletSignerUnavailable, cause: err}
	}
}

func executionWorkerWebFailure(err error) error {
	metadata := classifyExecutionWebError(err)
	return &executionWorkerFailure{code: metadata.code, cause: err}
}

func classifyExecutionWebError(err error) executionWebError {
	if err == nil {
		return executionWebError{}
	}
	if errors.Is(err, context.Canceled) {
		return executionWebError{code: wormErrorCancelled, temporary: true, ambiguous: true}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return executionWebError{code: wormErrorTimeout, temporary: true, ambiguous: true}
	}
	var edge *utilworm.WebEdgeBlockedError
	if errors.As(err, &edge) {
		return executionWebError{
			code: executionWorkerReasonEdgeBlocked, httpStatus: int32(edge.StatusCode),
			temporary: true, ambiguous: true,
		}
	}
	var api *utilworm.WebAPIError
	if errors.As(err, &api) {
		metadata := executionWebError{
			httpStatus: int32(api.StatusCode), providerCode: int32(api.Code),
			providerSlug:  api.Slug,
			structured401: api.StructuredJSON && api.StatusCode == http.StatusUnauthorized,
		}
		switch {
		case api.StatusCode == http.StatusUnauthorized || api.StatusCode == http.StatusForbidden:
			metadata.code = executionWorkerReasonWebAuthRejected
			metadata.definiteClient = api.StructuredJSON
		case api.StatusCode == http.StatusTooManyRequests:
			metadata.code = executionWorkerReasonRateLimited
			metadata.temporary = true
			metadata.ambiguous = true
		case api.StatusCode == http.StatusRequestTimeout || api.StatusCode == http.StatusConflict ||
			api.StatusCode == http.StatusTooEarly || api.StatusCode >= http.StatusInternalServerError:
			metadata.code = executionWorkerReasonWebUnavailable
			metadata.temporary = true
			metadata.ambiguous = true
		case api.StatusCode >= http.StatusBadRequest:
			metadata.code = executionWorkerReasonOpenRejected
			metadata.definiteClient = true
		default:
			metadata.code = executionWorkerReasonWebResponseInvalid
			metadata.ambiguous = true
		}
		return metadata
	}
	var transport *utilworm.WebTransportError
	if errors.As(err, &transport) {
		return executionWebError{code: executionWorkerReasonWebUnavailable, temporary: true, ambiguous: true}
	}
	var response *utilworm.WebResponseError
	if errors.As(err, &response) {
		return executionWebError{code: executionWorkerReasonWebResponseInvalid, temporary: true, ambiguous: true}
	}
	return executionWebError{code: executionWorkerReasonWebUnavailable, temporary: true, ambiguous: true}
}

func (s *Service) executeWormOpen(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
) (*wormstore.ExecutionRunStep, error) {
	request := utilworm.WebMarketPositionOpenRequest{
		MarketConditionID: task.item.MarketConditionID,
		Funds:             task.item.Funds,
		IsYes:             task.item.IsYes,
		Leverage:          1,
	}
	requestDigest, err := executionWorkerDigest(request)
	if err != nil {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid, cause: err}
	}
	attempt := executionMutationAttempt(task.step, wormstore.ExecutionMutationKindOpen)
	if attempt != nil && !bytes.Equal(attempt.RequestSHA256, requestDigest) {
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonPlanInvalid,
			errors.New("durable open request digest changed"))
	}
	if attempt != nil {
		switch attempt.State {
		case wormstore.ExecutionMutationStateDispatched:
			metadata := executionWebError{code: executionWorkerReasonOpenOutcomeUnknown}
			resolved, resolveErr := s.resolveExecutionMutation(
				guard.context(), *attempt, wormstore.ExecutionMutationStateOutcomeUnknown,
				attempt.PositionRequestID, metadata, metadata.code,
			)
			if resolveErr != nil {
				return nil, resolveErr
			}
			if step, handled, observeErr := s.observeExecutionOpenPosition(
				guard, task, resolved, task.step.ProviderState, task.step.ProviderOrderState,
				task.step.FundingTxID, task.step.RefundTxID,
			); handled {
				return step, observeErr
			}
			return s.markExecutionOutcomeUnknown(guard, task, *resolved,
				executionWorkerReasonOpenOutcomeUnknown, metadata)
		case wormstore.ExecutionMutationStateOutcomeUnknown:
			if step, handled, observeErr := s.observeExecutionOpenPosition(
				guard, task, attempt, task.step.ProviderState, task.step.ProviderOrderState,
				task.step.FundingTxID, task.step.RefundTxID,
			); handled {
				return step, observeErr
			}
			return s.markExecutionOutcomeUnknown(guard, task, *attempt,
				executionWorkerReasonOpenOutcomeUnknown, executionWebError{code: executionWorkerReasonOpenOutcomeUnknown})
		case wormstore.ExecutionMutationStateDefiniteFailure:
			return s.recoverDefiniteOpen(guard, task, *attempt)
		case wormstore.ExecutionMutationStateSucceeded:
			return s.recoverSuccessfulOpen(guard, task, *attempt)
		case wormstore.ExecutionMutationStatePrepared:
		default:
			return s.pauseClaimedExecution(guard, task, executionWorkerReasonPlanInvalid,
				errors.New("open attempt has an invalid state"))
		}
	}

	token, err := s.executionWebJWT(guard.context(), task)
	if err != nil {
		return s.pauseClaimedExecution(guard, task, executionWorkerFailureCode(err), err)
	}
	// OPENING may be recovered after a process restart. Repeat the mandatory
	// HMAC guard for every still-undispatched Open so a stale preflight can never
	// authorize a provider mutation.
	exposure, err := s.readExecutionWalletExposure(guard.context(), task)
	if err != nil {
		return s.pauseClaimedExecution(guard, task, executionPlanFailureWormReadUnavailable, err)
	}
	if reason, blocked := executionPreOpenGuard(exposure, task.item.MarketConditionID); blocked {
		return s.recordExecutionPreOpenSkip(guard, task, reason)
	}
	if attempt == nil {
		prepared, prepareErr := s.credentialStore.PrepareExecutionMutation(guard.context(), wormstore.PrepareExecutionMutationRequest{
			AttemptID: uuid.NewString(), RunID: task.run.ID, StepOrdinal: task.step.Ordinal,
			CommandID: task.recovery.CommandID, ClaimID: task.claimID,
			Kind: wormstore.ExecutionMutationKindOpen, RequestSHA256: requestDigest, Now: timeNowUTC(),
		})
		s.recordCredentialStoreResult(prepareErr)
		if prepareErr != nil {
			return nil, prepareErr
		}
		if prepared == nil || prepared.Kind != wormstore.ExecutionMutationKindOpen ||
			prepared.State != wormstore.ExecutionMutationStatePrepared ||
			!bytes.Equal(prepared.RequestSHA256, requestDigest) {
			return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
		}
		attempt = prepared
	}
	dispatched, err := s.credentialStore.DispatchExecutionMutation(guard.context(), wormstore.DispatchExecutionMutationRequest{
		AttemptID: attempt.ID, RunID: task.run.ID, StepOrdinal: task.step.Ordinal,
		ClaimID: task.claimID, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	if dispatched == nil || dispatched.State != wormstore.ExecutionMutationStateDispatched {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	attempt = dispatched

	opened, openErr := s.wormWebClient.OpenMarketPosition(guard.context(), token, request)
	if openErr != nil {
		metadata := classifyExecutionWebError(openErr)
		if metadata.structured401 {
			s.clearExecutionWebJWT(task, token)
		}
		if metadata.definiteClient && !metadata.temporary {
			resolved, resolveErr := s.resolveExecutionMutation(guard.context(), *attempt,
				wormstore.ExecutionMutationStateDefiniteFailure, 0, metadata, metadata.code)
			if resolveErr != nil {
				return nil, resolveErr
			}
			if step, handled, observeErr := s.observeExecutionOpenPosition(
				guard, task, resolved, task.step.ProviderState, task.step.ProviderOrderState,
				task.step.FundingTxID, task.step.RefundTxID,
			); handled {
				return step, observeErr
			}
			scope := wormstore.ExecutionStepScopeRemainingMarket
			if metadata.code == executionWorkerReasonWebAuthRejected {
				scope = wormstore.ExecutionStepScopeRemainingWallet
			}
			return s.recordExecutionProviderFailure(guard, task, task.step.State, *resolved,
				metadata.code, scope, nil)
		}
		resolved, resolveErr := s.resolveExecutionMutation(guard.context(), *attempt,
			wormstore.ExecutionMutationStateOutcomeUnknown, 0, metadata, metadata.code)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, resolved, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
		return s.markExecutionOutcomeUnknown(guard, task, *resolved,
			executionWorkerReasonOpenOutcomeUnknown, metadata)
	}
	if opened == nil || int64(opened.ID) <= 0 {
		metadata := executionWebError{code: executionWorkerReasonWebResponseInvalid, ambiguous: true}
		resolved, resolveErr := s.resolveExecutionMutation(guard.context(), *attempt,
			wormstore.ExecutionMutationStateOutcomeUnknown, 0, metadata, metadata.code)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, resolved, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
		return s.markExecutionOutcomeUnknown(guard, task, *resolved,
			executionWorkerReasonOpenOutcomeUnknown, metadata)
	}
	providerState, orderState, fundingTxID, refundTxID, responseErr := normalizedExecutionProviderRequest(opened)
	if responseErr != nil {
		metadata := executionWebError{code: executionWorkerReasonWebResponseInvalid, ambiguous: true}
		resolved, resolveErr := s.resolveExecutionMutation(guard.context(), *attempt,
			wormstore.ExecutionMutationStateOutcomeUnknown, int64(opened.ID), metadata, metadata.code)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, resolved, providerState, orderState, fundingTxID, refundTxID,
		); handled {
			return step, observeErr
		}
		return s.markExecutionOutcomeUnknownWithProvider(guard, task, *resolved,
			executionWorkerReasonOpenOutcomeUnknown, metadata, providerState, orderState, fundingTxID, refundTxID)
	}
	if executionProviderTerminalFailure(providerState) {
		metadata := executionWebError{code: executionWorkerReasonProviderFailed, httpStatus: http.StatusOK}
		resolved, resolveErr := s.resolveExecutionMutation(guard.context(), *attempt,
			wormstore.ExecutionMutationStateDefiniteFailure, int64(opened.ID), metadata, metadata.code)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, resolved, providerState, orderState, fundingTxID, refundTxID,
		); handled {
			return step, observeErr
		}
		return s.recordExecutionProviderFailure(guard, task, task.step.State, *resolved,
			executionWorkerReasonProviderFailed, wormstore.ExecutionStepScopeCurrent, opened)
	}
	transactionDigest, transactionErr := executionTransactionDigest(opened.Message)
	if transactionErr != nil {
		metadata := executionWebError{code: executionWorkerReasonWebResponseInvalid, httpStatus: http.StatusOK, ambiguous: true}
		resolved, resolveErr := s.resolveExecutionMutation(guard.context(), *attempt,
			wormstore.ExecutionMutationStateOutcomeUnknown, int64(opened.ID), metadata, metadata.code)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, resolved, providerState, orderState, fundingTxID, refundTxID,
		); handled {
			return step, observeErr
		}
		return s.markExecutionOutcomeUnknownWithProvider(guard, task, *resolved,
			executionWorkerReasonOpenOutcomeUnknown, metadata, providerState, orderState, fundingTxID, refundTxID)
	}
	// RecordExecutionStepOpened resolves the dispatched Open attempt and stores
	// its request ID and transaction digest in one database transaction.
	step, err := s.credentialStore.RecordExecutionStepOpened(guard.context(), wormstore.RecordExecutionStepOpenedRequest{
		AttemptID: attempt.ID, RunID: task.run.ID, StepOrdinal: task.step.Ordinal, ClaimID: task.claimID,
		PositionRequestID: int64(opened.ID), TransactionMessageSHA256: transactionDigest,
		ProviderState: providerState, ProviderOrderState: orderState, HTTPStatus: http.StatusOK,
		Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	if step == nil {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	task.step = step.Clone()
	openedAttempt := executionMutationAttempt(task.step, wormstore.ExecutionMutationKindOpen)
	if openedAttempt == nil || openedAttempt.State != wormstore.ExecutionMutationStateSucceeded ||
		openedAttempt.PositionRequestID != int64(opened.ID) {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	if observed, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, openedAttempt, providerState, orderState, fundingTxID, refundTxID,
	); handled {
		return observed, observeErr
	}
	if executionProviderCompleted(providerState) {
		return s.recordExecutionAwaiting(
			guard, task, task.step.State, providerState, orderState, fundingTxID, refundTxID,
		)
	}
	return step, nil
}

func (s *Service) recoverSuccessfulOpen(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	attempt wormstore.ExecutionMutationAttempt,
) (*wormstore.ExecutionRunStep, error) {
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, &attempt, task.step.ProviderState, task.step.ProviderOrderState,
		task.step.FundingTxID, task.step.RefundTxID,
	); handled {
		return step, observeErr
	}
	if attempt.PositionRequestID <= 0 {
		return s.markExecutionOutcomeUnknown(
			guard, task, attempt, executionWorkerReasonOpenOutcomeUnknown,
			executionWebError{code: executionWorkerReasonOpenOutcomeUnknown},
		)
	}
	request, err := s.getExecutionPositionRequest(guard.context(), task, attempt.PositionRequestID)
	if err != nil {
		return s.pauseClaimedExecution(guard, task, executionWorkerFailureCode(err), err)
	}
	providerState, orderState, fundingTxID, refundTxID, normalizeErr := normalizedExecutionProviderRequest(request)
	if normalizeErr != nil {
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonWebResponseInvalid, normalizeErr)
	}
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, &attempt, providerState, orderState, fundingTxID, refundTxID,
	); handled {
		return step, observeErr
	}
	if executionProviderTerminalFailure(providerState) {
		return s.recordExecutionRecoveredOpenObservation(
			guard, task, attempt.PositionRequestID, executionWorkerReasonProviderFailed,
			providerState, orderState, fundingTxID, refundTxID,
		)
	}
	transactionDigest, digestErr := executionTransactionDigest(request.Message)
	if digestErr != nil {
		if executionProviderCompleted(providerState) {
			return s.markExecutionOutcomeUnknownWithProvider(
				guard, task, attempt, executionWorkerReasonOpenOutcomeUnknown,
				executionWebError{code: executionWorkerReasonOpenOutcomeUnknown},
				providerState, orderState, fundingTxID, refundTxID,
			)
		}
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonWebResponseInvalid, digestErr)
	}
	step, err := s.credentialStore.RecordExecutionStepOpened(guard.context(), wormstore.RecordExecutionStepOpenedRequest{
		AttemptID: attempt.ID, RunID: task.run.ID, StepOrdinal: task.step.Ordinal, ClaimID: task.claimID,
		PositionRequestID: attempt.PositionRequestID, TransactionMessageSHA256: transactionDigest,
		ProviderState: providerState, ProviderOrderState: orderState,
		HTTPStatus: executionHTTPStatusOrOK(attempt.HTTPStatus), ProviderCode: attempt.ProviderCode,
		ProviderSlug: attempt.ProviderSlug, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	return step, err
}

func (s *Service) recoverDefiniteOpen(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	attempt wormstore.ExecutionMutationAttempt,
) (*wormstore.ExecutionRunStep, error) {
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, &attempt, task.step.ProviderState, task.step.ProviderOrderState,
		task.step.FundingTxID, task.step.RefundTxID,
	); handled {
		return step, observeErr
	}
	if attempt.PositionRequestID <= 0 {
		return s.recordExecutionProviderFailure(
			guard, task, task.step.State, attempt, executionWorkerReasonOpenRejected,
			wormstore.ExecutionStepScopeCurrent, nil,
		)
	}
	request, err := s.getExecutionPositionRequest(guard.context(), task, attempt.PositionRequestID)
	if err != nil {
		return s.pauseClaimedExecution(guard, task, executionWorkerFailureCode(err), err)
	}
	providerState, orderState, fundingTxID, refundTxID, normalizeErr := normalizedExecutionProviderRequest(request)
	if normalizeErr != nil {
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonWebResponseInvalid, normalizeErr)
	}
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, &attempt, providerState, orderState, fundingTxID, refundTxID,
	); handled {
		return step, observeErr
	}
	if executionProviderTerminalFailure(providerState) {
		return s.recordExecutionRecoveredOpenObservation(
			guard, task, attempt.PositionRequestID, executionWorkerReasonProviderFailed,
			providerState, orderState, fundingTxID, refundTxID,
		)
	}
	if executionProviderCompleted(providerState) {
		return s.markExecutionOutcomeUnknownWithProvider(
			guard, task, attempt, executionWorkerReasonOpenOutcomeUnknown,
			executionWebError{code: executionWorkerReasonOpenOutcomeUnknown},
			providerState, orderState, fundingTxID, refundTxID,
		)
	}
	return s.pauseClaimedExecution(guard, task, executionWorkerReasonWebResponseInvalid,
		errors.New("known Open failure is not terminal in the authoritative response"))
}

func executionMutationAttempt(
	step wormstore.ExecutionRunStep,
	kind wormstore.ExecutionMutationKind,
) *wormstore.ExecutionMutationAttempt {
	for index := range step.Attempts {
		if step.Attempts[index].Kind == kind {
			attempt := step.Attempts[index]
			return &attempt
		}
	}
	return nil
}

func executionWorkerDigest(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(encoded)
	return digest[:], nil
}

func executionTransactionDigest(transactionHex string) ([]byte, error) {
	encoded := strings.TrimSpace(transactionHex)
	if strings.HasPrefix(encoded, "0x") || strings.HasPrefix(encoded, "0X") {
		encoded = encoded[2:]
	}
	if encoded == "" || len(encoded) > executionWorkerMaxTxBytes*2 {
		return nil, errors.New("Worm transaction is missing or too large")
	}
	decoded, err := hex.DecodeString(encoded)
	if err != nil || len(decoded) == 0 {
		return nil, errors.New("Worm transaction is not valid hexadecimal")
	}
	defer clear(decoded)
	digest := sha256.Sum256(decoded)
	return digest[:], nil
}

func executionHTTPStatusOrOK(value int32) int32 {
	if value >= http.StatusOK && value < http.StatusMultipleChoices {
		return value
	}
	return http.StatusOK
}

func (s *Service) resolveExecutionMutation(
	ctx context.Context,
	attempt wormstore.ExecutionMutationAttempt,
	state wormstore.ExecutionMutationState,
	positionRequestID int64,
	metadata executionWebError,
	errorCode string,
) (*wormstore.ExecutionMutationAttempt, error) {
	resolved, err := s.credentialStore.ResolveExecutionMutation(ctx, wormstore.ResolveExecutionMutationRequest{
		AttemptID: attempt.ID, State: state, PositionRequestID: positionRequestID,
		HTTPStatus: metadata.httpStatus, ProviderCode: metadata.providerCode,
		ProviderSlug: metadata.providerSlug, ErrorCode: errorCode, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	return resolved, nil
}

func (s *Service) recordExecutionProviderFailure(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	expectedState wormstore.ExecutionStepState,
	attempt wormstore.ExecutionMutationAttempt,
	reason string,
	scope wormstore.ExecutionStepScope,
	provider *utilworm.WebPositionRequest,
) (*wormstore.ExecutionRunStep, error) {
	if attempt.State != wormstore.ExecutionMutationStateDefiniteFailure {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	providerState := task.step.ProviderState
	orderState := task.step.ProviderOrderState
	fundingTxID := task.step.FundingTxID
	refundTxID := task.step.RefundTxID
	positionRequestID := int64(0)
	if provider != nil {
		var err error
		providerState, orderState, fundingTxID, refundTxID, err = normalizedExecutionProviderRequest(provider)
		if err != nil {
			return nil, &executionWorkerFailure{code: executionWorkerReasonWebResponseInvalid, cause: err}
		}
		if expectedState == wormstore.ExecutionStepStateOpening &&
			executionProviderTerminalFailure(providerState) && attempt.PositionRequestID > 0 {
			positionRequestID = attempt.PositionRequestID
		}
	}
	if err := guard.stopRenewal(); err != nil {
		return nil, err
	}
	step, err := s.credentialStore.RecordExecutionProviderObservation(guard.context(), wormstore.RecordExecutionProviderObservationRequest{
		RunID: task.run.ID, StepOrdinal: task.step.Ordinal, CommandID: task.recovery.CommandID,
		ClaimID: task.claimID, ExpectedState: expectedState, NextState: wormstore.ExecutionStepStateFailed,
		ReasonCode: reason, PositionRequestID: positionRequestID,
		ProviderState: providerState, ProviderOrderState: orderState,
		FundingTxID: fundingTxID, RefundTxID: refundTxID, SkipScope: scope, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	return step, nil
}

func (s *Service) markExecutionOutcomeUnknown(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	attempt wormstore.ExecutionMutationAttempt,
	reason string,
	metadata executionWebError,
) (*wormstore.ExecutionRunStep, error) {
	return s.markExecutionOutcomeUnknownWithProvider(
		guard, task, attempt, reason, metadata,
		task.step.ProviderState, task.step.ProviderOrderState,
		task.step.FundingTxID, task.step.RefundTxID,
	)
}

func (s *Service) markExecutionOutcomeUnknownWithProvider(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	attempt wormstore.ExecutionMutationAttempt,
	reason string,
	metadata executionWebError,
	providerState string,
	providerOrderState string,
	fundingTxID string,
	refundTxID string,
) (*wormstore.ExecutionRunStep, error) {
	if attempt.State == wormstore.ExecutionMutationStateDispatched {
		resolved, err := s.resolveExecutionMutation(guard.context(), attempt,
			wormstore.ExecutionMutationStateOutcomeUnknown, attempt.PositionRequestID, metadata, reason)
		if err != nil {
			return nil, err
		}
		attempt = *resolved
	}
	if attempt.State != wormstore.ExecutionMutationStateOutcomeUnknown &&
		attempt.State != wormstore.ExecutionMutationStateSucceeded &&
		attempt.State != wormstore.ExecutionMutationStateDefiniteFailure {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	if err := guard.stopRenewal(); err != nil {
		return nil, err
	}
	positionRequestID := int64(0)
	if task.step.State == wormstore.ExecutionStepStateOpening && task.step.PositionRequestID == 0 {
		positionRequestID = attempt.PositionRequestID
	}
	step, err := s.credentialStore.RecordExecutionProviderObservation(guard.context(), wormstore.RecordExecutionProviderObservationRequest{
		RunID: task.run.ID, StepOrdinal: task.step.Ordinal, CommandID: task.recovery.CommandID,
		ClaimID: task.claimID, ExpectedState: task.step.State, NextState: wormstore.ExecutionStepStateOutcomeUnknown,
		ReasonCode: reason, ProviderState: providerState, ProviderOrderState: providerOrderState,
		FundingTxID: fundingTxID, RefundTxID: refundTxID,
		PositionRequestID: positionRequestID,
		IsolationID:       uuid.NewString(), AttemptID: attempt.ID, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	return step, nil
}

func normalizedExecutionProviderRequest(
	request *utilworm.WebPositionRequest,
) (string, string, string, string, error) {
	if request == nil || int64(request.ID) <= 0 {
		return "", "", "", "", errors.New("provider request identity is invalid")
	}
	state := strings.ToLower(strings.TrimSpace(request.State))
	orderState := strings.ToLower(strings.TrimSpace(request.OrderState))
	if len(state) > 100 || len(orderState) > 100 || containsExecutionControl(state) || containsExecutionControl(orderState) {
		return "", "", "", "", errors.New("provider request state is invalid")
	}
	fundingTxID, err := normalizedExecutionProviderTxID(request.FundingTxID)
	if err != nil {
		return "", "", "", "", err
	}
	refundTxID, err := normalizedExecutionProviderTxID(request.RefundTxID)
	if err != nil {
		return "", "", "", "", err
	}
	return state, orderState, fundingTxID, refundTxID, nil
}

func normalizedExecutionProviderTxID(value *string) (string, error) {
	if value == nil {
		return "", nil
	}
	normalized := strings.TrimSpace(*value)
	if len(normalized) > 200 || containsExecutionControl(normalized) {
		return "", errors.New("provider transaction identity is invalid")
	}
	return normalized, nil
}

func containsExecutionControl(value string) bool {
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return true
		}
	}
	return false
}

func executionProviderTerminalFailure(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "failed", "cancelled":
		return true
	default:
		return false
	}
}

func executionProviderCompleted(state string) bool {
	return strings.EqualFold(strings.TrimSpace(state), "completed")
}

func (s *Service) observeExecutionOpenPosition(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	openAttempt *wormstore.ExecutionMutationAttempt,
	providerState string,
	providerOrderState string,
	fundingTxID string,
	refundTxID string,
) (*wormstore.ExecutionRunStep, bool, error) {
	exposure, err := s.readExecutionWalletExposure(guard.context(), task)
	if err != nil {
		if task.step.State == wormstore.ExecutionStepStateAwaitingCompletion {
			if providerState == "" {
				providerState = task.step.ProviderState
			}
			if providerState == "" {
				providerState = "unknown"
			}
			step, persistErr := s.recordExecutionAwaitingAfterProviderFailure(
				guard, task, task.step.State, providerState, providerOrderState,
				fundingTxID, refundTxID, executionPlanFailureWormReadUnavailable,
			)
			return step, true, persistErr
		}
		if task.step.State == wormstore.ExecutionStepStateOutcomeUnknown {
			step, persistErr := s.recordExecutionReconciliation(
				guard, task, wormstore.ExecutionStepStateOutcomeUnknown,
				executionWorkerReasonReconcileInconclusive,
				providerState, providerOrderState, fundingTxID, refundTxID, "",
			)
			return step, true, persistErr
		}
		step, pauseErr := s.pauseClaimedExecution(
			guard, task, executionPlanFailureWormReadUnavailable, err,
		)
		return step, true, pauseErr
	}
	match, evidence := matchExecutionOpenPosition(exposure, task, openAttempt)
	switch match {
	case executionOpenPositionAbsent:
		return nil, false, nil
	case executionOpenPositionMatched:
		step, persistErr := s.recordExecutionOpenPositionCompletion(
			guard, task, openAttempt, evidence,
			providerState, providerOrderState, fundingTxID, refundTxID,
		)
		return step, true, persistErr
	case executionOpenPositionAmbiguous:
		if task.step.State == wormstore.ExecutionStepStateOutcomeUnknown {
			step, persistErr := s.recordExecutionReconciliation(
				guard, task, wormstore.ExecutionStepStateOutcomeUnknown,
				executionWorkerReasonPositionEvidenceInvalid,
				providerState, providerOrderState, fundingTxID, refundTxID, "",
			)
			return step, true, persistErr
		}
		step, persistErr := s.recordExecutionPositionAmbiguity(
			guard, task, openAttempt, providerState, providerOrderState, fundingTxID, refundTxID,
		)
		return step, true, persistErr
	default:
		return nil, true, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
}

func matchExecutionOpenPosition(
	exposure executionPreviewWalletExposure,
	task *executionWorkerTask,
	openAttempt *wormstore.ExecutionMutationAttempt,
) (executionOpenPositionMatch, *executionOpenPositionEvidence) {
	market := exposure.market(task.item.MarketConditionID)
	if market == nil || (len(market.positions[false]) == 0 && len(market.positions[true]) == 0) {
		return executionOpenPositionAbsent, nil
	}
	positions := make([]executionPreviewPosition, 0, len(market.positions[false])+len(market.positions[true]))
	positions = append(positions, market.positions[false]...)
	positions = append(positions, market.positions[true]...)
	if len(positions) != 1 {
		return executionOpenPositionAmbiguous, nil
	}
	if openAttempt == nil {
		openAttempt = executionMutationAttempt(task.step, wormstore.ExecutionMutationKindOpen)
	}
	position := positions[0]
	leverage, leverageErr := parseExecutionPreviewDecimal(position.leverage)
	if openAttempt == nil || openAttempt.Kind != wormstore.ExecutionMutationKindOpen ||
		openAttempt.DispatchedAt.IsZero() || position.isYes != task.item.IsYes ||
		leverageErr != nil || leverage.Compare(executionPreviewOne()) != 0 ||
		!position.hasCreatedAt || !validExecutionPreviewConditionID(position.pubkey) ||
		(position.positionRequestPubkey != "" &&
			!validExecutionPreviewConditionID(position.positionRequestPubkey)) {
		return executionOpenPositionAmbiguous, nil
	}
	dispatchedAtSecond := time.Unix(openAttempt.DispatchedAt.Unix(), 0).UTC()
	if position.createdAt.Before(dispatchedAtSecond) {
		return executionOpenPositionAmbiguous, nil
	}
	return executionOpenPositionMatched, &executionOpenPositionEvidence{
		positionPubkey:        position.pubkey,
		positionRequestPubkey: position.positionRequestPubkey,
		positionCreatedAt:     position.createdAt,
	}
}

func (s *Service) recordExecutionOpenPositionCompletion(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	openAttempt *wormstore.ExecutionMutationAttempt,
	evidence *executionOpenPositionEvidence,
	providerState string,
	providerOrderState string,
	fundingTxID string,
	refundTxID string,
) (*wormstore.ExecutionRunStep, error) {
	if evidence == nil || !validExecutionPreviewConditionID(evidence.positionPubkey) ||
		evidence.positionCreatedAt.IsZero() {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	if evidence.positionRequestPubkey != "" &&
		!validExecutionPreviewConditionID(evidence.positionRequestPubkey) {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	if openAttempt == nil {
		openAttempt = executionMutationAttempt(task.step, wormstore.ExecutionMutationKindOpen)
	}
	if openAttempt == nil || openAttempt.ID == "" || openAttempt.DispatchedAt.IsZero() {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	if task.step.State == wormstore.ExecutionStepStateOutcomeUnknown &&
		(task.step.Isolation == nil || task.step.Isolation.ID == "") {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	if err := guard.stopRenewal(); err != nil {
		return nil, err
	}
	request := wormstore.RecordExecutionProviderObservationRequest{
		RunID: task.run.ID, StepOrdinal: task.step.Ordinal, CommandID: task.recovery.CommandID,
		ClaimID: task.claimID, ExpectedState: task.step.State,
		NextState:     wormstore.ExecutionStepStateCompleted,
		ProviderState: providerState, ProviderOrderState: providerOrderState,
		FundingTxID: fundingTxID, RefundTxID: refundTxID,
		CompletionSource:                wormstore.ExecutionCompletionSourceOpenPosition,
		CompletionPositionPubkey:        evidence.positionPubkey,
		CompletionPositionRequestPubkey: evidence.positionRequestPubkey,
		CompletionPositionCreatedAt:     evidence.positionCreatedAt,
		Now:                             timeNowUTC(),
	}
	if task.step.State == wormstore.ExecutionStepStateOpening && task.step.PositionRequestID == 0 &&
		openAttempt.PositionRequestID > 0 {
		request.PositionRequestID = openAttempt.PositionRequestID
	}
	if task.step.State == wormstore.ExecutionStepStateOutcomeUnknown {
		request.ResolveIsolationID = task.step.Isolation.ID
		request.IsolationResolutionCode = executionWorkerReasonReconciledOpenPosition
	}
	step, err := s.credentialStore.RecordExecutionProviderObservation(guard.context(), request)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	return step, nil
}

func (s *Service) recordExecutionPositionAmbiguity(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	openAttempt *wormstore.ExecutionMutationAttempt,
	providerState string,
	providerOrderState string,
	fundingTxID string,
	refundTxID string,
) (*wormstore.ExecutionRunStep, error) {
	if openAttempt == nil {
		openAttempt = executionMutationAttempt(task.step, wormstore.ExecutionMutationKindOpen)
	}
	if openAttempt == nil || openAttempt.ID == "" || openAttempt.DispatchedAt.IsZero() {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	if err := guard.stopRenewal(); err != nil {
		return nil, err
	}
	request := wormstore.RecordExecutionProviderObservationRequest{
		RunID: task.run.ID, StepOrdinal: task.step.Ordinal, CommandID: task.recovery.CommandID,
		ClaimID: task.claimID, ExpectedState: task.step.State,
		NextState:     wormstore.ExecutionStepStateOutcomeUnknown,
		ReasonCode:    executionWorkerReasonPositionEvidenceInvalid,
		ProviderState: providerState, ProviderOrderState: providerOrderState,
		FundingTxID: fundingTxID, RefundTxID: refundTxID,
		IsolationID: uuid.NewString(), AttemptID: openAttempt.ID, Now: timeNowUTC(),
	}
	if task.step.State == wormstore.ExecutionStepStateOpening && task.step.PositionRequestID == 0 &&
		openAttempt.PositionRequestID > 0 {
		request.PositionRequestID = openAttempt.PositionRequestID
	}
	step, err := s.credentialStore.RecordExecutionProviderObservation(guard.context(), request)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	return step, nil
}

func (s *Service) getExecutionPositionRequest(
	ctx context.Context,
	task *executionWorkerTask,
	requestID int64,
) (*utilworm.WebPositionRequest, error) {
	if requestID <= 0 {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	token, err := s.executionWebJWT(ctx, task)
	if err != nil {
		return nil, err
	}
	request, err := s.wormWebClient.GetPositionRequest(ctx, token, requestID)
	if err != nil {
		metadata := classifyExecutionWebError(err)
		if metadata.structured401 {
			s.clearExecutionWebJWT(task, token)
			token, err = s.executionWebJWT(ctx, task)
			if err != nil {
				return nil, err
			}
			request, err = s.wormWebClient.GetPositionRequest(ctx, token, requestID)
		}
	}
	if err != nil {
		return nil, executionWorkerWebFailure(err)
	}
	if request == nil || int64(request.ID) != requestID {
		return nil, &executionWorkerFailure{code: executionWorkerReasonWebResponseInvalid}
	}
	return request, nil
}

func (s *Service) executeWormSigning(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
) (*wormstore.ExecutionRunStep, error) {
	openAttempt := executionMutationAttempt(task.step, wormstore.ExecutionMutationKindOpen)
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
		task.step.FundingTxID, task.step.RefundTxID,
	); handled {
		return step, observeErr
	}
	if task.step.PositionRequestID <= 0 || len(task.step.TransactionMessageSHA256) != sha256.Size {
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonPlanInvalid,
			errors.New("opened step has incomplete durable transaction metadata"))
	}
	provider, err := s.getExecutionPositionRequest(guard.context(), task, task.step.PositionRequestID)
	if err != nil {
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
		return s.pauseClaimedExecution(guard, task, executionWorkerFailureCode(err), err)
	}
	providerState, orderState, fundingTxID, refundTxID, err := normalizedExecutionProviderRequest(provider)
	if err != nil {
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonWebResponseInvalid, err)
	}
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, openAttempt, providerState, orderState, fundingTxID, refundTxID,
	); handled {
		return step, observeErr
	}
	if executionProviderCompleted(providerState) {
		return s.recordExecutionAwaiting(
			guard, task, task.step.State, providerState, orderState, fundingTxID, refundTxID,
		)
	}
	if executionProviderTerminalFailure(providerState) {
		return s.recordExecutionDirectObservation(
			guard, task, task.step.State, wormstore.ExecutionStepStateFailed,
			executionWorkerReasonProviderFailed, wormstore.ExecutionStepScopeCurrent,
			providerState, orderState, fundingTxID, refundTxID,
		)
	}
	transactionDigest, err := executionTransactionDigest(provider.Message)
	if err != nil || !bytes.Equal(transactionDigest, task.step.TransactionMessageSHA256) {
		attempt := executionMutationAttempt(task.step, wormstore.ExecutionMutationKindOpen)
		if attempt == nil || attempt.State != wormstore.ExecutionMutationStateSucceeded {
			return s.pauseClaimedExecution(guard, task, executionWorkerReasonTransactionChanged, err)
		}
		return s.markExecutionOutcomeUnknownWithProvider(
			guard, task, *attempt, executionWorkerReasonTransactionChanged,
			executionWebError{code: executionWorkerReasonTransactionChanged},
			providerState, orderState, fundingTxID, refundTxID,
		)
	}
	if task.step.State == wormstore.ExecutionStepStateOpened {
		advanced, advanceErr := s.credentialStore.MarkExecutionStepSigning(guard.context(), wormstore.AdvanceExecutionStepRequest{
			RunID: task.run.ID, StepOrdinal: task.step.Ordinal, ClaimID: task.claimID, Now: timeNowUTC(),
		})
		s.recordCredentialStoreResult(advanceErr)
		if advanceErr != nil {
			return nil, advanceErr
		}
		if advanced == nil {
			return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
		}
		task.step = advanced.Clone()
	}
	signed, err := s.signExecutionPositionTransaction(guard.context(), task, provider.Message)
	if err != nil {
		if executionWorkerFailureIsDeterministic(err) {
			return s.recordExecutionDirectObservation(
				guard, task, task.step.State, wormstore.ExecutionStepStateFailed,
				executionWorkerFailureCode(err), wormstore.ExecutionStepScopeRemainingWallet,
				providerState, orderState, fundingTxID, refundTxID,
			)
		}
		return s.pauseClaimedExecution(guard, task, executionWorkerFailureCode(err), err)
	}
	step, err := s.credentialStore.RecordExecutionStepSigned(guard.context(), wormstore.RecordExecutionStepSignedRequest{
		AdvanceExecutionStepRequest: wormstore.AdvanceExecutionStepRequest{
			RunID: task.run.ID, StepOrdinal: task.step.Ordinal, ClaimID: task.claimID, Now: timeNowUTC(),
		},
		FinalizeMode: string(signed.mode), TransactionVersion: signed.transactionVersion,
		RequiredSignatureCount: signed.requiredSignatures, WalletSignerIndex: signed.signerIndex,
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	return step, nil
}

func (s *Service) signExecutionPositionTransaction(
	ctx context.Context,
	task *executionWorkerTask,
	transactionHex string,
) (*executionSignedPayload, error) {
	transactionDigest, err := executionTransactionDigest(transactionHex)
	if err != nil || !bytes.Equal(transactionDigest, task.step.TransactionMessageSHA256) {
		return nil, &executionWorkerFailure{code: executionWorkerReasonTransactionChanged, cause: err}
	}
	signerCtx, cancel := context.WithTimeout(ctx, s.wormPositionBudget)
	response, err := s.walletSignerClientset.Signer().SignWormPositionRequestTransaction(
		signerCtx,
		&walletapiclient.SignWormPositionRequestTransactionRequest{
			Id: task.wallet.WalletID, RequesterAccountId: task.run.OwnerAccountID,
			ExpectedAddress: task.wallet.Address, PositionRequestId: task.step.PositionRequestID,
			TransactionHex: transactionHex, ExpectedTransactionSha256: transactionDigest,
			ExecutionRunId: task.run.ID, ExecutionStepId: task.step.ID,
			IntentSha256: append([]byte(nil), task.intent...),
		},
	)
	cancel()
	if err != nil {
		return nil, executionWorkerSignerFailure(err)
	}
	if response == nil || !bytes.Equal(response.GetTransactionSha256(), transactionDigest) ||
		response.GetExecutionRunId() != task.run.ID || response.GetExecutionStepId() != task.step.ID ||
		!bytes.Equal(response.GetIntentSha256(), task.intent) ||
		response.GetPositionRequestId() != task.step.PositionRequestID ||
		strings.TrimSpace(response.GetTransactionVersion()) == "" ||
		response.GetRequiredSignatures() <= 0 || response.GetSignerIndex() < 0 ||
		response.GetSignerIndex() >= response.GetRequiredSignatures() {
		return nil, &executionWorkerFailure{code: executionWorkerReasonWalletSignerInvalid}
	}
	result := &executionSignedPayload{
		transactionVersion: strings.TrimSpace(response.GetTransactionVersion()),
		requiredSignatures: response.GetRequiredSignatures(), signerIndex: response.GetSignerIndex(),
	}
	switch payload := response.GetFinalizePayload().(type) {
	case *walletapiclient.SignWormPositionRequestTransactionResponse_Signature:
		result.mode = utilworm.WebFinalizeModeSignature
		result.value = strings.TrimSpace(payload.Signature)
	case *walletapiclient.SignWormPositionRequestTransactionResponse_SignedTransaction:
		result.mode = utilworm.WebFinalizeModeSignedTransaction
		result.value = strings.TrimSpace(payload.SignedTransaction)
	default:
		return nil, &executionWorkerFailure{code: executionWorkerReasonWalletSignerInvalid}
	}
	if result.value == "" || containsExecutionControl(result.value) {
		return nil, &executionWorkerFailure{code: executionWorkerReasonWalletSignerInvalid}
	}
	return result, nil
}

func (s *Service) recordExecutionDirectObservation(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	expectedState wormstore.ExecutionStepState,
	nextState wormstore.ExecutionStepState,
	reason string,
	scope wormstore.ExecutionStepScope,
	providerState string,
	providerOrderState string,
	fundingTxID string,
	refundTxID string,
) (*wormstore.ExecutionRunStep, error) {
	if err := guard.stopRenewal(); err != nil {
		return nil, err
	}
	step, err := s.credentialStore.RecordExecutionProviderObservation(guard.context(), wormstore.RecordExecutionProviderObservationRequest{
		RunID: task.run.ID, StepOrdinal: task.step.Ordinal, CommandID: task.recovery.CommandID,
		ClaimID: task.claimID, ExpectedState: expectedState, NextState: nextState,
		ReasonCode: reason, ProviderState: providerState, ProviderOrderState: providerOrderState,
		FundingTxID: fundingTxID, RefundTxID: refundTxID, SkipScope: scope, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	return step, nil
}

func (s *Service) recordExecutionRecoveredOpenObservation(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	positionRequestID int64,
	reason string,
	providerState string,
	providerOrderState string,
	fundingTxID string,
	refundTxID string,
) (*wormstore.ExecutionRunStep, error) {
	if positionRequestID <= 0 || reason == "" {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	if err := guard.stopRenewal(); err != nil {
		return nil, err
	}
	step, err := s.credentialStore.RecordExecutionProviderObservation(guard.context(), wormstore.RecordExecutionProviderObservationRequest{
		RunID: task.run.ID, StepOrdinal: task.step.Ordinal, CommandID: task.recovery.CommandID,
		ClaimID: task.claimID, ExpectedState: wormstore.ExecutionStepStateOpening,
		NextState:  wormstore.ExecutionStepStateFailed,
		ReasonCode: reason, PositionRequestID: positionRequestID,
		ProviderState: providerState, ProviderOrderState: providerOrderState,
		FundingTxID: fundingTxID, RefundTxID: refundTxID,
		SkipScope: wormstore.ExecutionStepScopeCurrent, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	return step, nil
}

func (s *Service) executeWormFinalize(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
) (*wormstore.ExecutionRunStep, error) {
	attempt := executionMutationAttempt(task.step, wormstore.ExecutionMutationKindFinalize)
	if attempt != nil {
		switch attempt.State {
		case wormstore.ExecutionMutationStateDispatched,
			wormstore.ExecutionMutationStateSucceeded,
			wormstore.ExecutionMutationStateOutcomeUnknown,
			wormstore.ExecutionMutationStateDefiniteFailure:
			return s.recoverFinalizingExecution(guard, task, *attempt)
		case wormstore.ExecutionMutationStatePrepared:
		default:
			return s.pauseClaimedExecution(guard, task, executionWorkerReasonPlanInvalid,
				errors.New("finalize attempt has an invalid state"))
		}
	}
	openAttempt := executionMutationAttempt(task.step, wormstore.ExecutionMutationKindOpen)
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
		task.step.FundingTxID, task.step.RefundTxID,
	); handled {
		return step, observeErr
	}
	if task.step.PositionRequestID <= 0 || len(task.step.TransactionMessageSHA256) != sha256.Size ||
		(task.step.FinalizeMode != string(utilworm.WebFinalizeModeSignature) &&
			task.step.FinalizeMode != string(utilworm.WebFinalizeModeSignedTransaction)) {
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonPlanInvalid,
			errors.New("finalizing step has incomplete durable metadata"))
	}

	provider, err := s.getExecutionPositionRequest(guard.context(), task, task.step.PositionRequestID)
	if err != nil {
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
		return s.pauseClaimedExecution(guard, task, executionWorkerFailureCode(err), err)
	}
	providerState, orderState, fundingTxID, refundTxID, err := normalizedExecutionProviderRequest(provider)
	if err != nil {
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonWebResponseInvalid, err)
	}
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, openAttempt, providerState, orderState, fundingTxID, refundTxID,
	); handled {
		return step, observeErr
	}
	if executionProviderCompleted(providerState) || executionProviderTerminalFailure(providerState) {
		if executionProviderCompleted(providerState) {
			return s.recordExecutionAwaiting(
				guard, task, task.step.State, providerState, orderState, fundingTxID, refundTxID,
			)
		}
		return s.recordExecutionDirectObservation(
			guard, task, task.step.State, wormstore.ExecutionStepStateFailed,
			executionWorkerReasonProviderFailed, wormstore.ExecutionStepScopeCurrent,
			providerState, orderState, fundingTxID, refundTxID,
		)
	}
	transactionDigest, err := executionTransactionDigest(provider.Message)
	if err != nil || !bytes.Equal(transactionDigest, task.step.TransactionMessageSHA256) {
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonTransactionChanged, err)
	}
	signed, err := s.signExecutionPositionTransaction(guard.context(), task, provider.Message)
	if err != nil {
		if executionWorkerFailureIsDeterministic(err) {
			return s.recordExecutionDirectObservation(
				guard, task, task.step.State, wormstore.ExecutionStepStateFailed,
				executionWorkerFailureCode(err), wormstore.ExecutionStepScopeRemainingWallet,
				providerState, orderState, fundingTxID, refundTxID,
			)
		}
		return s.pauseClaimedExecution(guard, task, executionWorkerFailureCode(err), err)
	}
	if string(signed.mode) != task.step.FinalizeMode ||
		signed.transactionVersion != task.step.TransactionVersion ||
		signed.requiredSignatures != task.step.RequiredSignatureCount ||
		signed.signerIndex != task.step.WalletSignerIndex {
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonWalletSignerInvalid,
			errors.New("Wallet signer metadata changed before finalize"))
	}
	request := utilworm.WebPositionFinalizeRequest{
		PositionRequestID: task.step.PositionRequestID,
		Payload:           utilworm.WebPositionFinalizePayload{Mode: signed.mode, Value: signed.value},
	}
	requestDigest, err := executionWorkerDigest(request)
	if err != nil {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid, cause: err}
	}
	if attempt != nil && !bytes.Equal(attempt.RequestSHA256, requestDigest) {
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonTransactionChanged,
			errors.New("durable finalize request digest changed"))
	}
	if attempt == nil {
		prepared, prepareErr := s.credentialStore.PrepareExecutionMutation(guard.context(), wormstore.PrepareExecutionMutationRequest{
			AttemptID: uuid.NewString(), RunID: task.run.ID, StepOrdinal: task.step.Ordinal,
			CommandID: task.recovery.CommandID, ClaimID: task.claimID,
			Kind: wormstore.ExecutionMutationKindFinalize, RequestSHA256: requestDigest,
			PositionRequestID: task.step.PositionRequestID, Now: timeNowUTC(),
		})
		s.recordCredentialStoreResult(prepareErr)
		if prepareErr != nil {
			return nil, prepareErr
		}
		if prepared == nil || prepared.State != wormstore.ExecutionMutationStatePrepared ||
			!bytes.Equal(prepared.RequestSHA256, requestDigest) {
			return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
		}
		attempt = prepared
	}
	token, err := s.executionWebJWT(guard.context(), task)
	if err != nil {
		return s.pauseClaimedExecution(guard, task, executionWorkerFailureCode(err), err)
	}
	dispatched, err := s.credentialStore.DispatchExecutionMutation(guard.context(), wormstore.DispatchExecutionMutationRequest{
		AttemptID: attempt.ID, RunID: task.run.ID, StepOrdinal: task.step.Ordinal,
		ClaimID: task.claimID, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	if dispatched == nil || dispatched.State != wormstore.ExecutionMutationStateDispatched {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	attempt = dispatched
	finalized, finalizeErr := s.wormWebClient.FinalizePosition(guard.context(), token, request)
	if finalizeErr != nil {
		metadata := classifyExecutionWebError(finalizeErr)
		if metadata.code == executionWorkerReasonOpenRejected {
			metadata.code = executionWorkerReasonFinalizeRejected
		}
		if metadata.structured401 {
			s.clearExecutionWebJWT(task, token)
		}
		return s.reconcileAmbiguousFinalize(guard, task, *attempt, metadata)
	}
	providerState, orderState, fundingTxID, refundTxID, err = normalizedExecutionProviderRequest(finalized)
	if err != nil || int64(finalized.ID) != task.step.PositionRequestID {
		return s.reconcileAmbiguousFinalize(guard, task, *attempt,
			executionWebError{code: executionWorkerReasonWebResponseInvalid, httpStatus: http.StatusOK, ambiguous: true})
	}
	if executionProviderTerminalFailure(providerState) {
		metadata := executionWebError{code: executionWorkerReasonProviderFailed, httpStatus: http.StatusOK}
		resolved, resolveErr := s.resolveExecutionMutation(guard.context(), *attempt,
			wormstore.ExecutionMutationStateDefiniteFailure, task.step.PositionRequestID, metadata, metadata.code)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, providerState, orderState, fundingTxID, refundTxID,
		); handled {
			return step, observeErr
		}
		return s.recordExecutionProviderFailure(guard, task, task.step.State, *resolved,
			executionWorkerReasonProviderFailed, wormstore.ExecutionStepScopeCurrent, finalized)
	}
	if providerState == "" {
		return s.reconcileAmbiguousFinalize(guard, task, *attempt,
			executionWebError{code: executionWorkerReasonWebResponseInvalid, httpStatus: http.StatusOK, ambiguous: true})
	}
	resolved, err := s.resolveExecutionMutation(guard.context(), *attempt,
		wormstore.ExecutionMutationStateSucceeded, task.step.PositionRequestID,
		executionWebError{httpStatus: http.StatusOK}, "")
	if err != nil {
		return nil, err
	}
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, openAttempt, providerState, orderState, fundingTxID, refundTxID,
	); handled {
		return step, observeErr
	}
	_ = resolved
	return s.recordExecutionAwaiting(guard, task, task.step.State,
		providerState, orderState, fundingTxID, refundTxID)
}

func (s *Service) recoverFinalizingExecution(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	attempt wormstore.ExecutionMutationAttempt,
) (*wormstore.ExecutionRunStep, error) {
	openAttempt := executionMutationAttempt(task.step, wormstore.ExecutionMutationKindOpen)
	if attempt.State != wormstore.ExecutionMutationStateDispatched {
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
	}
	if attempt.State == wormstore.ExecutionMutationStateDefiniteFailure {
		return s.recordExecutionProviderFailure(guard, task, task.step.State, attempt,
			executionWorkerReasonFinalizeRejected, wormstore.ExecutionStepScopeCurrent, nil)
	}
	provider, err := s.getExecutionPositionRequest(guard.context(), task, task.step.PositionRequestID)
	if err != nil {
		if attempt.State == wormstore.ExecutionMutationStateDispatched {
			metadata := classifyExecutionWebError(errors.Unwrap(err))
			if metadata.code == "" {
				metadata = executionWebError{code: executionWorkerFailureCode(err), ambiguous: true}
			}
			resolved, resolveErr := s.resolveExecutionMutation(guard.context(), attempt,
				wormstore.ExecutionMutationStateOutcomeUnknown, task.step.PositionRequestID,
				metadata, executionWorkerReasonFinalizeOutcomeUnknown)
			if resolveErr != nil {
				return nil, resolveErr
			}
			attempt = *resolved
		}
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
		providerState := task.step.ProviderState
		if providerState == "" {
			providerState = "unknown"
		}
		return s.recordExecutionAwaitingAfterProviderFailure(
			guard, task, task.step.State, providerState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID, executionWorkerFailureCode(err),
		)
	}
	providerState, orderState, fundingTxID, refundTxID, normalizeErr := normalizedExecutionProviderRequest(provider)
	if normalizeErr != nil {
		if attempt.State == wormstore.ExecutionMutationStateDispatched {
			metadata := executionWebError{code: executionWorkerReasonWebResponseInvalid, ambiguous: true}
			resolved, resolveErr := s.resolveExecutionMutation(
				guard.context(), attempt, wormstore.ExecutionMutationStateOutcomeUnknown,
				task.step.PositionRequestID, metadata, metadata.code,
			)
			if resolveErr != nil {
				return nil, resolveErr
			}
			attempt = *resolved
		}
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonWebResponseInvalid, normalizeErr)
	}
	if attempt.State == wormstore.ExecutionMutationStateDispatched {
		if executionProviderTerminalFailure(providerState) {
			metadata := executionWebError{code: executionWorkerReasonProviderFailed}
			resolved, resolveErr := s.resolveExecutionMutation(guard.context(), attempt,
				wormstore.ExecutionMutationStateDefiniteFailure, task.step.PositionRequestID, metadata, metadata.code)
			if resolveErr != nil {
				return nil, resolveErr
			}
			if step, handled, observeErr := s.observeExecutionOpenPosition(
				guard, task, openAttempt, providerState, orderState, fundingTxID, refundTxID,
			); handled {
				return step, observeErr
			}
			return s.recordExecutionProviderFailure(guard, task, task.step.State, *resolved,
				executionWorkerReasonProviderFailed, wormstore.ExecutionStepScopeCurrent, provider)
		}
		resolved, resolveErr := s.resolveExecutionMutation(guard.context(), attempt,
			wormstore.ExecutionMutationStateOutcomeUnknown, task.step.PositionRequestID,
			executionWebError{}, executionWorkerReasonFinalizeOutcomeUnknown)
		if resolveErr != nil {
			return nil, resolveErr
		}
		attempt = *resolved
	}
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, openAttempt, providerState, orderState, fundingTxID, refundTxID,
	); handled {
		return step, observeErr
	}
	if executionProviderCompleted(providerState) {
		return s.recordExecutionAwaiting(
			guard, task, task.step.State, providerState, orderState, fundingTxID, refundTxID,
		)
	}
	if executionProviderTerminalFailure(providerState) {
		return s.recordExecutionDirectObservation(
			guard, task, task.step.State, wormstore.ExecutionStepStateFailed,
			executionWorkerReasonProviderFailed, wormstore.ExecutionStepScopeCurrent,
			providerState, orderState, fundingTxID, refundTxID,
		)
	}
	if providerState != "" {
		return s.recordExecutionAwaiting(guard, task, task.step.State,
			providerState, orderState, fundingTxID, refundTxID)
	}
	return s.recordExecutionAwaitingAfterProviderFailure(
		guard, task, task.step.State, "unknown", orderState, fundingTxID, refundTxID,
		executionWorkerReasonWebResponseInvalid,
	)
}

func (s *Service) reconcileAmbiguousFinalize(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	attempt wormstore.ExecutionMutationAttempt,
	metadata executionWebError,
) (*wormstore.ExecutionRunStep, error) {
	openAttempt := executionMutationAttempt(task.step, wormstore.ExecutionMutationKindOpen)
	if attempt.State != wormstore.ExecutionMutationStateDispatched {
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
	}
	provider, getErr := s.getExecutionPositionRequest(guard.context(), task, task.step.PositionRequestID)
	if getErr != nil {
		if metadata.definiteClient && !metadata.temporary {
			resolved, resolveErr := s.resolveExecutionMutation(guard.context(), attempt,
				wormstore.ExecutionMutationStateDefiniteFailure, task.step.PositionRequestID,
				metadata, executionWorkerReasonFinalizeRejected)
			if resolveErr != nil {
				return nil, resolveErr
			}
			if step, handled, observeErr := s.observeExecutionOpenPosition(
				guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
				task.step.FundingTxID, task.step.RefundTxID,
			); handled {
				return step, observeErr
			}
			return s.recordExecutionProviderFailure(
				guard, task, task.step.State, *resolved,
				executionWorkerReasonFinalizeRejected, wormstore.ExecutionStepScopeCurrent, nil,
			)
		}
		resolved, resolveErr := s.resolveExecutionMutation(guard.context(), attempt,
			wormstore.ExecutionMutationStateOutcomeUnknown, task.step.PositionRequestID,
			metadata, executionWorkerReasonFinalizeOutcomeUnknown)
		if resolveErr != nil {
			return nil, resolveErr
		}
		attempt = *resolved
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
		providerState := task.step.ProviderState
		if providerState == "" {
			providerState = "unknown"
		}
		return s.recordExecutionAwaitingAfterProviderFailure(
			guard, task, task.step.State, providerState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID, metadata.code,
		)
	}
	providerState, orderState, fundingTxID, refundTxID, normalizeErr := normalizedExecutionProviderRequest(provider)
	if normalizeErr != nil {
		if metadata.definiteClient && !metadata.temporary {
			resolved, resolveErr := s.resolveExecutionMutation(guard.context(), attempt,
				wormstore.ExecutionMutationStateDefiniteFailure, task.step.PositionRequestID,
				metadata, executionWorkerReasonFinalizeRejected)
			if resolveErr != nil {
				return nil, resolveErr
			}
			if step, handled, observeErr := s.observeExecutionOpenPosition(
				guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
				task.step.FundingTxID, task.step.RefundTxID,
			); handled {
				return step, observeErr
			}
			return s.recordExecutionProviderFailure(
				guard, task, task.step.State, *resolved,
				executionWorkerReasonFinalizeRejected, wormstore.ExecutionStepScopeCurrent, nil,
			)
		}
		resolved, resolveErr := s.resolveExecutionMutation(guard.context(), attempt,
			wormstore.ExecutionMutationStateOutcomeUnknown, task.step.PositionRequestID,
			metadata, executionWorkerReasonFinalizeOutcomeUnknown)
		if resolveErr != nil {
			return nil, resolveErr
		}
		attempt = *resolved
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
		providerState := task.step.ProviderState
		if providerState == "" {
			providerState = "unknown"
		}
		return s.recordExecutionAwaitingAfterProviderFailure(
			guard, task, task.step.State, providerState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID, executionWorkerReasonWebResponseInvalid,
		)
	}
	if executionProviderTerminalFailure(providerState) {
		resolved, resolveErr := s.resolveExecutionMutation(guard.context(), attempt,
			wormstore.ExecutionMutationStateDefiniteFailure, task.step.PositionRequestID,
			metadata, executionWorkerReasonProviderFailed)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, providerState, orderState, fundingTxID, refundTxID,
		); handled {
			return step, observeErr
		}
		return s.recordExecutionProviderFailure(guard, task, task.step.State, *resolved,
			executionWorkerReasonProviderFailed, wormstore.ExecutionStepScopeCurrent, provider)
	}
	if metadata.definiteClient && !metadata.temporary && !executionProviderCompleted(providerState) {
		resolved, resolveErr := s.resolveExecutionMutation(guard.context(), attempt,
			wormstore.ExecutionMutationStateDefiniteFailure, task.step.PositionRequestID,
			metadata, executionWorkerReasonFinalizeRejected)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, providerState, orderState, fundingTxID, refundTxID,
		); handled {
			return step, observeErr
		}
		return s.recordExecutionProviderFailure(
			guard, task, task.step.State, *resolved,
			executionWorkerReasonFinalizeRejected, wormstore.ExecutionStepScopeCurrent, provider,
		)
	}
	_, resolveErr := s.resolveExecutionMutation(guard.context(), attempt,
		wormstore.ExecutionMutationStateOutcomeUnknown, task.step.PositionRequestID,
		metadata, executionWorkerReasonFinalizeOutcomeUnknown)
	if resolveErr != nil {
		return nil, resolveErr
	}
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, openAttempt, providerState, orderState, fundingTxID, refundTxID,
	); handled {
		return step, observeErr
	}
	if executionProviderCompleted(providerState) {
		return s.recordExecutionAwaiting(
			guard, task, task.step.State, providerState, orderState, fundingTxID, refundTxID,
		)
	}
	if providerState == "" {
		providerState = "unknown"
	}
	if metadata.temporary || metadata.ambiguous {
		return s.recordExecutionAwaitingAfterProviderFailure(
			guard, task, task.step.State, providerState, orderState,
			fundingTxID, refundTxID, metadata.code,
		)
	}
	return s.recordExecutionAwaiting(
		guard, task, task.step.State, providerState, orderState, fundingTxID, refundTxID,
	)
}

func (s *Service) recordExecutionAwaiting(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	expectedState wormstore.ExecutionStepState,
	providerState string,
	providerOrderState string,
	fundingTxID string,
	refundTxID string,
) (*wormstore.ExecutionRunStep, error) {
	if providerState == "" {
		return nil, &executionWorkerFailure{code: executionWorkerReasonWebResponseInvalid}
	}
	if err := guard.stopRenewal(); err != nil {
		return nil, err
	}
	now := timeNowUTC()
	step, err := s.credentialStore.RecordExecutionProviderObservation(guard.context(), wormstore.RecordExecutionProviderObservationRequest{
		RunID: task.run.ID, StepOrdinal: task.step.Ordinal, CommandID: task.recovery.CommandID,
		ClaimID: task.claimID, ExpectedState: expectedState,
		NextState:     wormstore.ExecutionStepStateAwaitingCompletion,
		ProviderState: providerState, ProviderOrderState: providerOrderState,
		FundingTxID: fundingTxID, RefundTxID: refundTxID,
		NextPollAt: now.Add(executionPollDelay(task.step.PollCount)), Now: now,
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	return step, nil
}

func (s *Service) recordExecutionAwaitingAfterProviderFailure(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	expectedState wormstore.ExecutionStepState,
	providerState string,
	providerOrderState string,
	fundingTxID string,
	refundTxID string,
	pauseCode string,
) (*wormstore.ExecutionRunStep, error) {
	if pauseCode == "" {
		pauseCode = executionWorkerReasonWebUnavailable
	}
	step, err := s.recordExecutionAwaiting(
		guard, task, expectedState, providerState, providerOrderState, fundingTxID, refundTxID,
	)
	if err != nil {
		return nil, err
	}
	_, pauseErr := s.credentialStore.PauseExecutionRunForFailure(guard.context(), wormstore.PauseExecutionRunForFailureRequest{
		RunID: task.run.ID, PauseCode: pauseCode, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(pauseErr)
	if pauseErr != nil && !errors.Is(pauseErr, wormstore.ErrExecutionRunConflict) {
		return nil, pauseErr
	}
	return step, nil
}

func executionPollDelay(pollCount int32) time.Duration {
	if pollCount < 10 {
		return time.Second
	}
	shift := pollCount - 10
	if shift > 6 {
		shift = 6
	}
	delay := time.Second * time.Duration(1<<shift)
	if delay > time.Minute {
		return time.Minute
	}
	return delay
}

func (s *Service) pollWormPositionRequest(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
) (*wormstore.ExecutionRunStep, error) {
	openAttempt := executionMutationAttempt(task.step, wormstore.ExecutionMutationKindOpen)
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
		task.step.FundingTxID, task.step.RefundTxID,
	); handled {
		return step, observeErr
	}
	if task.step.PositionRequestID <= 0 {
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonPlanInvalid,
			errors.New("awaiting step has no request id"))
	}
	provider, err := s.getExecutionPositionRequest(guard.context(), task, task.step.PositionRequestID)
	if err != nil {
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
		metadata := classifyExecutionWebError(errors.Unwrap(err))
		if metadata.definiteClient && metadata.httpStatus != http.StatusNotFound &&
			metadata.httpStatus != http.StatusUnauthorized && metadata.httpStatus != http.StatusForbidden {
			return s.recordExecutionDirectObservation(
				guard, task, task.step.State, wormstore.ExecutionStepStateFailed,
				executionWorkerFailureCode(err), wormstore.ExecutionStepScopeCurrent,
				task.step.ProviderState, task.step.ProviderOrderState,
				task.step.FundingTxID, task.step.RefundTxID,
			)
		}
		providerState := task.step.ProviderState
		if providerState == "" {
			providerState = "unknown"
		}
		step, persistErr := s.recordExecutionAwaiting(
			guard, task, task.step.State, providerState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		)
		if persistErr != nil {
			return nil, persistErr
		}
		_, pauseErr := s.credentialStore.PauseExecutionRunForFailure(guard.context(), wormstore.PauseExecutionRunForFailureRequest{
			RunID: task.run.ID, PauseCode: executionWorkerFailureCode(err), Now: timeNowUTC(),
		})
		s.recordCredentialStoreResult(pauseErr)
		if pauseErr != nil && !errors.Is(pauseErr, wormstore.ErrExecutionRunConflict) {
			return nil, pauseErr
		}
		return step, nil
	}
	providerState, orderState, fundingTxID, refundTxID, err := normalizedExecutionProviderRequest(provider)
	if err != nil || providerState == "" {
		if step, handled, observeErr := s.observeExecutionOpenPosition(
			guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID,
		); handled {
			return step, observeErr
		}
		return s.pauseClaimedExecution(guard, task, executionWorkerReasonWebResponseInvalid, err)
	}
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, openAttempt, providerState, orderState, fundingTxID, refundTxID,
	); handled {
		return step, observeErr
	}
	if executionProviderCompleted(providerState) {
		return s.recordExecutionAwaiting(
			guard, task, task.step.State, providerState, orderState, fundingTxID, refundTxID,
		)
	}
	if executionProviderTerminalFailure(providerState) {
		return s.recordExecutionDirectObservation(
			guard, task, task.step.State, wormstore.ExecutionStepStateFailed,
			executionWorkerReasonProviderFailed, wormstore.ExecutionStepScopeCurrent,
			providerState, orderState, fundingTxID, refundTxID,
		)
	}
	return s.recordExecutionAwaiting(
		guard, task, task.step.State, providerState, orderState, fundingTxID, refundTxID,
	)
}

func (s *Service) reconcileWormExecutionStep(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
) (*wormstore.ExecutionRunStep, error) {
	if task.step.Isolation == nil || task.step.Isolation.ID == "" ||
		task.step.ReconcileRequestedAt.IsZero() {
		return nil, &executionWorkerFailure{code: executionWorkerReasonPlanInvalid}
	}
	openAttempt := executionMutationAttempt(task.step, wormstore.ExecutionMutationKindOpen)
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, openAttempt, task.step.ProviderState, task.step.ProviderOrderState,
		task.step.FundingTxID, task.step.RefundTxID,
	); handled {
		return step, observeErr
	}
	if task.step.PositionRequestID <= 0 {
		return s.recordExecutionReconciliation(
			guard, task, wormstore.ExecutionStepStateOutcomeUnknown,
			executionWorkerReasonReconcileInconclusive,
			task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID, "",
		)
	}
	provider, err := s.getExecutionPositionRequest(guard.context(), task, task.step.PositionRequestID)
	if err != nil {
		return s.recordExecutionReconciliation(
			guard, task, wormstore.ExecutionStepStateOutcomeUnknown,
			executionWorkerReasonReconcileInconclusive,
			task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID, "",
		)
	}
	providerState, orderState, fundingTxID, refundTxID, normalizeErr := normalizedExecutionProviderRequest(provider)
	if normalizeErr != nil {
		return s.recordExecutionReconciliation(
			guard, task, wormstore.ExecutionStepStateOutcomeUnknown,
			executionWorkerReasonReconcileInconclusive,
			task.step.ProviderState, task.step.ProviderOrderState,
			task.step.FundingTxID, task.step.RefundTxID, "",
		)
	}
	if step, handled, observeErr := s.observeExecutionOpenPosition(
		guard, task, openAttempt, providerState, orderState, fundingTxID, refundTxID,
	); handled {
		return step, observeErr
	}
	if executionProviderTerminalFailure(providerState) {
		return s.recordExecutionReconciliation(
			guard, task, wormstore.ExecutionStepStateFailed,
			executionWorkerReasonProviderFailed, providerState, orderState,
			fundingTxID, refundTxID, executionWorkerReasonReconciledFailed,
		)
	}
	return s.recordExecutionReconciliation(
		guard, task, wormstore.ExecutionStepStateOutcomeUnknown,
		executionWorkerReasonReconcileInconclusive, providerState, orderState,
		fundingTxID, refundTxID, "",
	)
}

func (s *Service) recordExecutionReconciliation(
	guard *executionWorkerClaimGuard,
	task *executionWorkerTask,
	nextState wormstore.ExecutionStepState,
	reason string,
	providerState string,
	providerOrderState string,
	fundingTxID string,
	refundTxID string,
	resolutionCode string,
) (*wormstore.ExecutionRunStep, error) {
	if err := guard.stopRenewal(); err != nil {
		return nil, err
	}
	request := wormstore.RecordExecutionProviderObservationRequest{
		RunID: task.run.ID, StepOrdinal: task.step.Ordinal, CommandID: task.recovery.CommandID,
		ClaimID: task.claimID, ExpectedState: wormstore.ExecutionStepStateOutcomeUnknown,
		NextState: nextState, ReasonCode: reason,
		ProviderState: providerState, ProviderOrderState: providerOrderState,
		FundingTxID: fundingTxID, RefundTxID: refundTxID, Now: timeNowUTC(),
	}
	if nextState != wormstore.ExecutionStepStateOutcomeUnknown {
		request.ResolveIsolationID = task.step.Isolation.ID
		request.IsolationResolutionCode = resolutionCode
	}
	step, err := s.credentialStore.RecordExecutionProviderObservation(guard.context(), request)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	return step, nil
}
