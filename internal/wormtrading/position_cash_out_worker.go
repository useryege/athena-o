package wormtrading

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	utilworm "github.com/useryege/athena/util/worm"
)

const (
	positionCashOutWorkerPollInterval = time.Second
	positionCashOutClaimLease         = 45 * time.Second
	positionCashOutRecoverySize       = int32(100)
	positionCashOutPendingPoll        = time.Second

	positionCashOutReasonNotFound          = "POSITION_NOT_FOUND"
	positionCashOutReasonChanged           = "POSITION_CHANGED"
	positionCashOutReasonLiquidated        = "POSITION_LIQUIDATED"
	positionCashOutReasonConnectionChanged = "WORM_CONNECTION_CHANGED"
	positionCashOutReasonCloseRejected     = "CLOSE_REJECTED"
	positionCashOutReasonCloseUnknown      = "CLOSE_OUTCOME_UNKNOWN"
	positionCashOutReasonProviderInvalid   = "INVALID_RESPONSE"
	positionCashOutReasonBatchTerminated   = "BATCH_TERMINATED"
)

func (s *Service) runPositionCashOutWorker(ctx context.Context) {
	defer s.runWG.Done()
	workerID := uuid.NewString()
	for {
		s.runPositionCashOutRecoveryPass(ctx, workerID)
		timer := time.NewTimer(positionCashOutWorkerPollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-s.positionCashOutWake:
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
		}
	}
}

func (s *Service) runPositionCashOutRecoveryPass(ctx context.Context, workerID string) {
	if ctx.Err() != nil {
		return
	}
	now := timeNowUTC()
	_, err := s.credentialStore.ExpirePositionCashOuts(ctx, now, positionCashOutRecoverySize)
	s.recordCredentialStoreResult(err)
	if err != nil {
		log.WithField("error_code", "POSITION_CASH_OUT_STORE_UNAVAILABLE").Warn("expire Worm position Cash Outs")
		return
	}
	recoverable, err := s.credentialStore.ListRecoverablePositionCashOuts(ctx, now, positionCashOutRecoverySize)
	s.recordCredentialStoreResult(err)
	if err != nil {
		log.WithField("error_code", "POSITION_CASH_OUT_STORE_UNAVAILABLE").Warn("recover Worm position Cash Outs")
		return
	}
	for index := range recoverable {
		if ctx.Err() != nil {
			return
		}
		claimID := uuid.NewString()
		claimed, claimErr := s.credentialStore.ClaimPositionCashOut(ctx, wormstore.PositionCashOutClaimRequest{
			CashOutID:      recoverable[index].ID,
			ClaimID:        claimID,
			WorkerID:       workerID,
			LeaseExpiresAt: timeNowUTC().Add(positionCashOutClaimLease),
			Now:            timeNowUTC(),
		})
		s.recordCredentialStoreResult(claimErr)
		if claimErr != nil {
			continue
		}
		if err := s.processClaimedPositionCashOut(ctx, claimed, claimID); err != nil && ctx.Err() == nil {
			log.WithFields(log.Fields{
				"cash_out_id": claimed.ID,
				"state":       claimed.State,
				"error_code":  positionCashOutWorkerErrorCode(err),
			}).Warn("process Worm position Cash Out")
		}
	}
}

func (s *Service) processClaimedPositionCashOut(
	ctx context.Context,
	cashOut *wormstore.PositionCashOut,
	claimID string,
) error {
	if cashOut == nil {
		return errors.New("position Cash Out claim is empty")
	}
	if cashOut.BatchID != "" {
		defer s.wakePositionCashOutBatchWorker()
	}
	client, credentialID, err := s.positionCashOutClient(ctx, *cashOut)
	if err != nil {
		if cashOut.Attempt != nil && cashOut.Attempt.State != wormstore.PositionCashOutAttemptStatePrepared {
			return s.recordPositionCashOutReconciliation(ctx, *cashOut, claimID,
				positionCashOutReasonConnectionChanged, "", positionCashOutFrozenTarget(*cashOut))
		}
		return s.recordPositionCashOutFailure(ctx, *cashOut, claimID, positionCashOutReasonConnectionChanged, "")
	}

	if cashOut.State == wormstore.PositionCashOutStatePreflighting {
		observed, observeErr := s.observePositionCashOut(ctx, client, *cashOut)
		if observeErr != nil {
			return s.handlePositionCashOutPreflightError(ctx, *cashOut, claimID, credentialID, observeErr)
		}
		providerState := positionCashOutProviderState(observed)
		if cashOut.BatchID != "" && observed.TotalShares != cashOut.Shares {
			return s.recordPositionCashOutFailure(ctx, *cashOut, claimID,
				positionCashOutReasonChanged, providerState)
		}
		switch {
		case observed.IsLiquidated:
			if cashOut.BatchID != "" {
				return s.recordPositionCashOutFailure(ctx, *cashOut, claimID,
					positionCashOutReasonLiquidated, providerState)
			}
			return s.recordPositionCashOutState(ctx, *cashOut, claimID, wormstore.PositionCashOutStateFailed,
				positionCashOutReasonLiquidated, providerState, observed, time.Time{})
		case observed.IsClosed:
			if cashOut.BatchID != "" {
				return s.recordPositionCashOutFailure(ctx, *cashOut, claimID,
					positionCashOutReasonChanged, providerState)
			}
			return s.recordPositionCashOutState(ctx, *cashOut, claimID, wormstore.PositionCashOutStateCompleted,
				"", providerState, observed, time.Time{})
		}
		if timeNowUTC().After(cashOut.ExecutionExpiresAt) {
			return nil
		}
		command, err := utilworm.PrepareMarginPositionCashOut(observed)
		if err != nil {
			return s.recordPositionCashOutFailure(ctx, *cashOut, claimID, positionCashOutReasonProviderInvalid, providerState)
		}
		cashOut, err = s.credentialStore.BeginPositionCashOutClosing(ctx, wormstore.BeginPositionCashOutClosingRequest{
			CashOutID: cashOut.ID, ClaimID: claimID, ProviderState: providerState, Now: timeNowUTC(),
		})
		s.recordCredentialStoreResult(err)
		if err != nil {
			return err
		}
		return s.dispatchPreparedPositionCashOut(ctx, client, *cashOut, claimID, command, credentialID)
	}

	if cashOut.State == wormstore.PositionCashOutStateClosing {
		if cashOut.Attempt == nil || cashOut.Attempt.State == wormstore.PositionCashOutAttemptStatePrepared {
			// A crash may leave CLOSING with no dispatched mutation. Repeat the
			// exact safe GET so the closability check remains adjacent to the
			// eventual Close instead of trusting an older preflight snapshot.
			observed, observeErr := s.observePositionCashOut(ctx, client, *cashOut)
			if observeErr != nil {
				return s.handlePositionCashOutPreflightError(ctx, *cashOut, claimID, credentialID, observeErr)
			}
			providerState := positionCashOutProviderState(observed)
			if cashOut.BatchID != "" && observed.TotalShares != cashOut.Shares {
				return s.recordPositionCashOutFailure(ctx, *cashOut, claimID,
					positionCashOutReasonChanged, providerState)
			}
			switch {
			case observed.IsLiquidated:
				if cashOut.BatchID != "" {
					return s.recordPositionCashOutFailure(ctx, *cashOut, claimID,
						positionCashOutReasonLiquidated, providerState)
				}
				return s.recordPositionCashOutState(ctx, *cashOut, claimID, wormstore.PositionCashOutStateFailed,
					positionCashOutReasonLiquidated, providerState, observed, time.Time{})
			case observed.IsClosed:
				if cashOut.BatchID != "" {
					return s.recordPositionCashOutFailure(ctx, *cashOut, claimID,
						positionCashOutReasonChanged, providerState)
				}
				return s.recordPositionCashOutState(ctx, *cashOut, claimID, wormstore.PositionCashOutStateCompleted,
					"", providerState, observed, time.Time{})
			}
			if !cashOut.ExecutionExpiresAt.After(timeNowUTC()) {
				return nil
			}
			command, prepareErr := utilworm.PrepareMarginPositionCashOut(observed)
			if prepareErr != nil {
				return s.recordPositionCashOutFailure(ctx, *cashOut, claimID, positionCashOutReasonProviderInvalid, providerState)
			}
			if cashOut.Attempt != nil {
				digest := command.RequestSHA256()
				if len(cashOut.Attempt.RequestSHA256) != len(digest) ||
					subtle.ConstantTimeCompare(cashOut.Attempt.RequestSHA256, digest[:]) != 1 {
					return s.recordPositionCashOutReconciliation(ctx, *cashOut, claimID,
						positionCashOutReasonProviderInvalid, providerState, observed)
				}
			}
			return s.dispatchPreparedPositionCashOut(ctx, client, *cashOut, claimID, command, credentialID)
		}
	}

	return s.reconcileClaimedPositionCashOut(ctx, client, *cashOut, claimID, credentialID)
}

func (s *Service) positionCashOutClient(
	ctx context.Context,
	cashOut wormstore.PositionCashOut,
) (WormAPIClient, int64, error) {
	snapshot, err := s.credentialStore.GetWalletConnectionSnapshot(ctx, cashOut.WalletID, cashOut.WalletAddress)
	s.recordCredentialStoreResult(err)
	mutationDispatched := cashOut.Attempt != nil && cashOut.Attempt.State != wormstore.PositionCashOutAttemptStatePrepared
	if err != nil || snapshot.State != wormstore.ConnectionStateConnected || snapshot.ActiveCredential == nil ||
		snapshot.ActiveCredential.State != wormstore.CredentialStateActive ||
		(!mutationDispatched && snapshot.ActiveCredential.Version != cashOut.CredentialVersion) {
		return nil, 0, wormstore.ErrPositionCashOutConnectionChanged
	}
	credential := snapshot.ActiveCredential
	apiKey, apiSecret, err := s.credentialCipher.decrypt(
		credential.WalletID,
		credential.APIKeyCiphertext,
		credential.APISecretCiphertext,
	)
	if err != nil {
		return nil, credential.ID, err
	}
	client, err := s.wormClientFactory.NewAuthenticatedClient(apiKey, apiSecret)
	return client, credential.ID, err
}

func (s *Service) observePositionCashOut(
	ctx context.Context,
	client WormAPIClient,
	cashOut wormstore.PositionCashOut,
) (utilworm.MarginPositionCashOutTarget, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, s.wormAPIAttemptTimeout)
	defer cancel()
	observed, err := utilworm.ObserveMarginPositionCashOut(attemptCtx, client, positionCashOutFrozenTarget(cashOut))
	s.wormCapabilities.recordWormResult(err)
	return observed, err
}

func (s *Service) handlePositionCashOutPreflightError(
	ctx context.Context,
	cashOut wormstore.PositionCashOut,
	claimID string,
	credentialID int64,
	err error,
) error {
	if isWormAuthenticationError(err) {
		s.markWalletReconnectRequired(ctx, cashOut.WalletID, cashOut.WalletAddress, credentialID)
		return s.recordPositionCashOutFailure(ctx, cashOut, claimID, positionCashOutReasonConnectionChanged, "")
	}
	if isWormNotFoundError(err) {
		return s.recordPositionCashOutFailure(ctx, cashOut, claimID, positionCashOutReasonNotFound, "")
	}
	switch classifyWormError(err) {
	case wormErrorRateLimited, wormErrorUnavailable, wormErrorTimeout, wormErrorCancelled:
		// No mutation has been sent. Retain PREFLIGHTING and let the bounded
		// claim expire so a later recovery pass can retry only this safe GET.
		return err
	default:
		return s.recordPositionCashOutFailure(ctx, cashOut, claimID, positionCashOutReasonChanged, "")
	}
}

func (s *Service) dispatchPreparedPositionCashOut(
	ctx context.Context,
	client WormAPIClient,
	cashOut wormstore.PositionCashOut,
	claimID string,
	command *utilworm.MarginPositionCashOutCommand,
	credentialID int64,
) error {
	attemptID := ""
	if cashOut.Attempt == nil {
		attemptID = uuid.NewString()
		digest := command.RequestSHA256()
		attempt, err := s.credentialStore.PreparePositionCashOutAttempt(ctx, wormstore.PreparePositionCashOutAttemptRequest{
			AttemptID: attemptID, CashOutID: cashOut.ID, ClaimID: claimID,
			RequestSHA256: append([]byte(nil), digest[:]...), Now: timeNowUTC(),
		})
		s.recordCredentialStoreResult(err)
		if err != nil {
			return err
		}
		cashOut.Attempt = attempt
	} else {
		attemptID = cashOut.Attempt.ID
	}
	var dispatched *wormstore.PositionCashOutAttempt
	var err error
	if cashOut.BatchID != "" {
		baseline, baselineErr := s.readPositionCashOutBatchBalance(ctx, cashOut.WalletID, cashOut.WalletAddress)
		if baselineErr != nil {
			// No mutation has been sent. Retain the PREPARED attempt so a later
			// recovery pass can repeat only the safe balance read and exact
			// position GET before attempting the same durable Close.
			return baselineErr
		}
		now := timeNowUTC()
		dispatched, err = s.credentialStore.DispatchPositionCashOutBatchAttempt(ctx,
			wormstore.DispatchPositionCashOutBatchAttemptRequest{
				BatchID: cashOut.BatchID, ItemID: cashOut.BatchItemID,
				AttemptID: attemptID, CashOutID: cashOut.ID, ClaimID: claimID,
				Baseline: baseline, NextPollAt: now, Now: now,
			})
	} else {
		dispatched, err = s.credentialStore.DispatchPositionCashOutAttempt(ctx, wormstore.DispatchPositionCashOutAttemptRequest{
			AttemptID: attemptID, CashOutID: cashOut.ID, ClaimID: claimID, Now: timeNowUTC(),
		})
	}
	s.recordCredentialStoreResult(err)
	if err != nil {
		if cashOut.BatchID != "" && errors.Is(err, wormstore.ErrPositionCashOutBatchClaim) {
			// The parent batch rejected this still-PREPARED mutation (most
			// importantly after TERMINATE_REQUESTED). The dispatch transaction
			// did not commit, so it is safe to make the child terminal and let
			// the parent mark this and all remaining items NOT_EXECUTED.
			return s.recordPositionCashOutFailure(
				ctx, cashOut, claimID, positionCashOutReasonBatchTerminated, cashOut.ProviderState,
			)
		}
		// The transaction outcome may be unknown. Never compensate or send the
		// Close here; recovery must inspect the database's durable attempt state.
		return err
	}
	cashOut.Attempt = dispatched

	attemptCtx, cancel := context.WithTimeout(ctx, s.wormAPIAttemptTimeout)
	observation, dispatchErr := utilworm.DispatchMarginPositionCashOut(attemptCtx, client, command)
	cancel()
	s.wormCapabilities.recordWormResult(dispatchErr)
	if isWormAuthenticationError(dispatchErr) {
		s.markWalletReconnectRequired(ctx, cashOut.WalletID, cashOut.WalletAddress, credentialID)
	}
	metadata := classifyPositionCashOutDispatch(dispatchErr)
	attemptState := wormstore.PositionCashOutAttemptStateAcknowledged
	if dispatchErr != nil {
		if metadata.explicitlyRejected {
			attemptState = wormstore.PositionCashOutAttemptStateRejected
		} else {
			attemptState = wormstore.PositionCashOutAttemptStateOutcomeUnknown
		}
	}
	providerState := ""
	if observation != nil {
		if observation.IsClosed {
			providerState = "closed"
		} else {
			providerState = "closing"
		}
	}
	if dispatchErr == nil && observation != nil && observation.PositionPubkey == cashOut.PositionPubkey && observation.IsClosed {
		_, completeErr := s.credentialStore.CompletePositionCashOutDispatch(ctx, wormstore.CompletePositionCashOutDispatchRequest{
			AttemptID: attemptID, CashOutID: cashOut.ID, ClaimID: claimID,
			HTTPStatus: metadata.httpStatus, ProviderCode: metadata.providerCode,
			ProviderSlug: metadata.providerSlug, ProviderState: "closed",
			ErrorCode: metadata.errorCode, Now: timeNowUTC(),
		})
		s.recordCredentialStoreResult(completeErr)
		return completeErr
	}
	resolved, resolveErr := s.credentialStore.ResolvePositionCashOutAttempt(ctx, wormstore.ResolvePositionCashOutAttemptRequest{
		AttemptID: attemptID, CashOutID: cashOut.ID, State: attemptState,
		HTTPStatus: metadata.httpStatus, ProviderCode: metadata.providerCode,
		ProviderSlug: metadata.providerSlug, ProviderState: providerState,
		ErrorCode: metadata.errorCode, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(resolveErr)
	if resolveErr != nil {
		return resolveErr
	}
	cashOut.Attempt = resolved
	return s.reconcileClaimedPositionCashOut(ctx, client, cashOut, claimID, credentialID)
}

func (s *Service) readPositionCashOutBatchBalance(
	ctx context.Context,
	walletID int64,
	walletAddress string,
) (wormstore.PositionCashOutBatchBalanceEvidence, error) {
	results, err := s.adapter.BatchGetBalances(ctx, []WalletBalanceReference{{
		WalletID: walletID,
		Address:  walletAddress,
	}})
	if err != nil {
		return wormstore.PositionCashOutBatchBalanceEvidence{}, fmt.Errorf("read batch Cash Out USDC baseline: %w", err)
	}
	if len(results) != 1 || results[0].WalletID != walletID || results[0].Address != walletAddress {
		return wormstore.PositionCashOutBatchBalanceEvidence{}, errors.New("batch Cash Out balance response does not match the Wallet")
	}
	observation := results[0].USDC
	if observation.Availability != availabilityAvailable || observation.Decimals != USDCDecimals ||
		observation.ObservedSlot == 0 {
		return wormstore.PositionCashOutBatchBalanceEvidence{}, errors.New("batch Cash Out confirmed USDC balance is unavailable")
	}
	atomicAmount, ok := canonicalNonNegativeAtomicAmount(observation.AtomicAmount)
	if !ok {
		return wormstore.PositionCashOutBatchBalanceEvidence{}, errors.New("batch Cash Out confirmed USDC balance is invalid")
	}
	return wormstore.PositionCashOutBatchBalanceEvidence{
		Mint:         SolanaNativeUSDCMint,
		Decimals:     USDCDecimals,
		AtomicAmount: atomicAmount,
		ObservedSlot: observation.ObservedSlot,
	}, nil
}

func canonicalNonNegativeAtomicAmount(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" || value != raw || (len(value) > 1 && value[0] == '0') {
		return "", false
	}
	parsed, ok := new(big.Int).SetString(value, 10)
	if !ok || parsed.Sign() < 0 || parsed.String() != value {
		return "", false
	}
	return value, true
}

func (s *Service) reconcileClaimedPositionCashOut(
	ctx context.Context,
	client WormAPIClient,
	cashOut wormstore.PositionCashOut,
	claimID string,
	credentialID int64,
) error {
	observed, err := s.observePositionCashOut(ctx, client, cashOut)
	if err != nil {
		if isWormAuthenticationError(err) {
			s.markWalletReconnectRequired(ctx, cashOut.WalletID, cashOut.WalletAddress, credentialID)
		}
		reason := positionCashOutReasonCloseUnknown
		if isWormNotFoundError(err) {
			reason = positionCashOutReasonNotFound
		}
		return s.recordPositionCashOutReconciliation(
			ctx, cashOut, claimID, reason, "", positionCashOutFrozenTarget(cashOut),
		)
	}
	providerState := positionCashOutProviderState(observed)
	if observed.IsLiquidated {
		if positionCashOutBatchMutationNotDispatched(cashOut) {
			return s.recordPositionCashOutFailure(ctx, cashOut, claimID,
				positionCashOutReasonLiquidated, providerState)
		}
		return s.recordPositionCashOutState(ctx, cashOut, claimID, wormstore.PositionCashOutStateFailed,
			positionCashOutReasonLiquidated, providerState, observed, time.Time{})
	}
	if observed.IsClosed {
		if positionCashOutBatchMutationNotDispatched(cashOut) {
			return s.recordPositionCashOutFailure(ctx, cashOut, claimID,
				positionCashOutReasonChanged, providerState)
		}
		return s.recordPositionCashOutState(ctx, cashOut, claimID, wormstore.PositionCashOutStateCompleted,
			"", providerState, observed, time.Time{})
	}
	if cashOut.Attempt != nil && cashOut.Attempt.State == wormstore.PositionCashOutAttemptStateRejected {
		return s.recordPositionCashOutState(ctx, cashOut, claimID, wormstore.PositionCashOutStateFailed,
			positionCashOutReasonCloseRejected, providerState, observed, time.Time{})
	}
	if cashOut.State == wormstore.PositionCashOutStateReconciliationRequired ||
		(cashOut.Attempt != nil && cashOut.Attempt.State == wormstore.PositionCashOutAttemptStateOutcomeUnknown) ||
		(cashOut.Attempt != nil && cashOut.Attempt.State == wormstore.PositionCashOutAttemptStateDispatched) {
		return s.recordPositionCashOutReconciliation(ctx, cashOut, claimID,
			positionCashOutReasonCloseUnknown, providerState, observed)
	}
	return s.recordPositionCashOutState(ctx, cashOut, claimID, wormstore.PositionCashOutStateAwaitingCompletion,
		"", providerState, observed, timeNowUTC().Add(positionCashOutPendingPoll))
}

func positionCashOutBatchMutationNotDispatched(cashOut wormstore.PositionCashOut) bool {
	return cashOut.BatchID != "" &&
		(cashOut.Attempt == nil || cashOut.Attempt.State == wormstore.PositionCashOutAttemptStatePrepared)
}

func (s *Service) recordPositionCashOutFailure(
	ctx context.Context,
	cashOut wormstore.PositionCashOut,
	claimID string,
	reasonCode string,
	providerState string,
) error {
	return s.recordPositionCashOutState(ctx, cashOut, claimID, wormstore.PositionCashOutStateFailed,
		reasonCode, providerState, positionCashOutFrozenTarget(cashOut), time.Time{})
}

func (s *Service) recordPositionCashOutReconciliation(
	ctx context.Context,
	cashOut wormstore.PositionCashOut,
	claimID string,
	reasonCode string,
	providerState string,
	observed utilworm.MarginPositionCashOutTarget,
) error {
	return s.recordPositionCashOutState(ctx, cashOut, claimID, wormstore.PositionCashOutStateReconciliationRequired,
		reasonCode, providerState, observed, timeNowUTC().Add(positionCashOutPendingPoll))
}

func (s *Service) recordPositionCashOutState(
	ctx context.Context,
	cashOut wormstore.PositionCashOut,
	claimID string,
	nextState wormstore.PositionCashOutState,
	reasonCode string,
	providerState string,
	observed utilworm.MarginPositionCashOutTarget,
	nextPollAt time.Time,
) error {
	_, err := s.credentialStore.RecordPositionCashOutObservation(ctx, wormstore.RecordPositionCashOutObservationRequest{
		CashOutID: cashOut.ID, ClaimID: claimID, ExpectedState: cashOut.State, NextState: nextState,
		ReasonCode: reasonCode, ProviderState: providerState,
		ProviderIsClosed: observed.IsClosed, ProviderIsLiquidated: observed.IsLiquidated,
		NextPollAt: nextPollAt, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	return err
}

func positionCashOutFrozenTarget(cashOut wormstore.PositionCashOut) utilworm.MarginPositionCashOutTarget {
	target := utilworm.MarginPositionCashOutTarget{
		PositionPubkey: cashOut.PositionPubkey, MarketConditionID: cashOut.MarketConditionID,
		IsYes: cashOut.IsYes, TotalShares: cashOut.Shares,
		CreatedAt: cashOut.PositionCreatedAt.Unix(), IsClosed: cashOut.ProviderIsClosed,
		IsLiquidated: cashOut.ProviderIsLiquidated,
	}
	if cashOut.PositionRequestPubkey != "" {
		value := cashOut.PositionRequestPubkey
		target.PositionRequestPubkey = &value
	}
	return target
}

func positionCashOutProviderState(target utilworm.MarginPositionCashOutTarget) string {
	switch {
	case target.IsLiquidated:
		return "liquidated"
	case target.IsClosed:
		return "closed"
	default:
		return "open"
	}
}

type positionCashOutDispatchMetadata struct {
	httpStatus         int32
	providerCode       int32
	providerSlug       string
	errorCode          string
	explicitlyRejected bool
}

func classifyPositionCashOutDispatch(err error) positionCashOutDispatchMetadata {
	if err == nil {
		return positionCashOutDispatchMetadata{}
	}
	metadata := positionCashOutDispatchMetadata{errorCode: classifyWormError(err)}
	var providerErr *utilworm.Error
	if !errors.As(err, &providerErr) {
		return metadata
	}
	metadata.httpStatus = int32(providerErr.StatusCode)
	metadata.providerCode = int32(providerErr.Code)
	metadata.providerSlug = providerErr.Slug
	if providerErr.StatusCode == 0 ||
		(providerErr.StatusCode >= http.StatusBadRequest && providerErr.StatusCode < http.StatusInternalServerError &&
			providerErr.StatusCode != http.StatusRequestTimeout && providerErr.StatusCode != http.StatusTooManyRequests) {
		metadata.explicitlyRejected = true
	}
	return metadata
}

func positionCashOutWorkerErrorCode(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, wormstore.ErrPositionCashOutClaim) {
		return "CLAIM_CHANGED"
	}
	if errors.Is(err, wormstore.ErrTransactionOutcomeUnknown) {
		return "STORE_OUTCOME_UNKNOWN"
	}
	code := classifyWormError(err)
	if code != "" {
		return code
	}
	return "POSITION_CASH_OUT_FAILED"
}
