package wormtrading

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	utilworm "github.com/useryege/athena/util/worm"
)

const (
	positionCashOutBatchWorkerPollInterval = time.Second
	positionCashOutBatchClaimLease         = 45 * time.Second
	positionCashOutBatchRecoverySize       = int32(100)
	positionCashOutBatchItemPollInterval   = time.Second
	positionCashOutBatchImmediatePoll      = time.Millisecond
	positionCashOutBatchBalancePoll        = 2 * time.Second
	positionCashOutBatchBalanceTimeout     = 2 * time.Minute

	positionCashOutBatchBuildStageWallets   = "WALLET_PREFLIGHT"
	positionCashOutBatchBuildStagePositions = "POSITION_SCAN"

	positionCashOutBatchReasonConnectionChanged  = "WORM_CONNECTION_CHANGED"
	positionCashOutBatchReasonPositionScanFailed = "POSITION_SCAN_FAILED"
	positionCashOutBatchReasonNoPositions        = "NO_OPEN_POSITIONS"
	positionCashOutBatchReasonPositionLimit      = "POSITION_LIMIT_EXCEEDED"
	positionCashOutBatchReasonInvalidResponse    = "INVALID_RESPONSE"
	positionCashOutBatchReasonChildMismatch      = "CHILD_OPERATION_MISMATCH"
	positionCashOutBatchReasonChildFailed        = "POSITION_CASH_OUT_FAILED"
	positionCashOutBatchReasonCloseUnknown       = "CLOSE_OUTCOME_UNKNOWN"
	positionCashOutBatchReasonBalanceUnavailable = "USDC_BALANCE_UNAVAILABLE"
)

type positionCashOutBatchBuildFailure struct {
	stage      string
	reasonCode string
	cause      error
}

func (e *positionCashOutBatchBuildFailure) Error() string {
	if e == nil || e.cause == nil {
		return "position Cash Out batch build failed"
	}
	return e.cause.Error()
}

func (e *positionCashOutBatchBuildFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (s *Service) runPositionCashOutBatchWorker(ctx context.Context) {
	defer s.runWG.Done()
	workerID := uuid.NewString()
	for {
		s.runPositionCashOutBatchRecoveryPass(ctx, workerID)
		timer := time.NewTimer(positionCashOutBatchWorkerPollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-s.positionCashOutBatchWake:
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
		}
	}
}

func (s *Service) runPositionCashOutBatchRecoveryPass(ctx context.Context, workerID string) {
	if ctx.Err() != nil {
		return
	}
	now := timeNowUTC()
	_, err := s.credentialStore.ExpirePositionCashOutBatches(ctx, now, positionCashOutBatchRecoverySize)
	s.recordCredentialStoreResult(err)
	if err != nil {
		log.WithField("error_code", "POSITION_CASH_OUT_BATCH_STORE_UNAVAILABLE").Warn(
			"expire Worm position Cash Out batches",
		)
		return
	}
	recoverable, err := s.credentialStore.ListRecoverablePositionCashOutBatches(
		ctx,
		now,
		positionCashOutBatchRecoverySize,
	)
	s.recordCredentialStoreResult(err)
	if err != nil {
		log.WithField("error_code", "POSITION_CASH_OUT_BATCH_STORE_UNAVAILABLE").Warn(
			"recover Worm position Cash Out batches",
		)
		return
	}
	for index := range recoverable {
		if ctx.Err() != nil {
			return
		}
		claimID := uuid.NewString()
		claimed, claimErr := s.credentialStore.ClaimPositionCashOutBatch(
			ctx,
			wormstore.PositionCashOutBatchClaimRequest{
				BatchID:        recoverable[index].ID,
				ClaimID:        claimID,
				WorkerID:       workerID,
				LeaseExpiresAt: timeNowUTC().Add(positionCashOutBatchClaimLease),
				Now:            timeNowUTC(),
			},
		)
		s.recordCredentialStoreResult(claimErr)
		if claimErr != nil {
			continue
		}
		if err := s.processClaimedPositionCashOutBatch(ctx, claimed, claimID, workerID); err != nil && ctx.Err() == nil {
			log.WithFields(log.Fields{
				"batch_id":   claimed.ID,
				"state":      claimed.State,
				"error_code": positionCashOutBatchWorkerErrorCode(err),
			}).Warn("process Worm position Cash Out batch")
		}
	}
}

func (s *Service) processClaimedPositionCashOutBatch(
	ctx context.Context,
	batch *wormstore.PositionCashOutBatch,
	claimID string,
	workerID string,
) error {
	if batch == nil {
		return errors.New("position Cash Out batch claim is empty")
	}
	switch batch.State {
	case wormstore.PositionCashOutBatchStateBuilding:
		return s.buildClaimedPositionCashOutBatch(ctx, batch, claimID, workerID)
	case wormstore.PositionCashOutBatchStateRunning,
		wormstore.PositionCashOutBatchStatePauseRequested,
		wormstore.PositionCashOutBatchStateTerminateRequested:
		if batch.CurrentItem == nil {
			if batch.State != wormstore.PositionCashOutBatchStateRunning ||
				batch.CompletedCount == batch.PositionCount {
				return s.completePositionCashOutBatchBoundary(ctx, *batch, claimID)
			}
			return s.activateNextPositionCashOutBatchItem(ctx, *batch, claimID)
		}
		return s.processPositionCashOutBatchItem(ctx, *batch, claimID, workerID)
	case wormstore.PositionCashOutBatchStatePaused,
		wormstore.PositionCashOutBatchStateReconciliationRequired:
		if batch.CurrentItem == nil {
			return errors.New("blocked position Cash Out batch has no current item")
		}
		return s.processPositionCashOutBatchItem(ctx, *batch, claimID, workerID)
	default:
		return errors.New("position Cash Out batch claim has an unsupported state")
	}
}

func (s *Service) buildClaimedPositionCashOutBatch(
	ctx context.Context,
	batch *wormstore.PositionCashOutBatch,
	claimID string,
	workerID string,
) error {
	if len(batch.Wallets) == 0 || len(batch.Wallets) > positionCashOutBatchMaximumWallets {
		return s.failPositionCashOutBatchBuild(ctx, *batch, claimID,
			positionCashOutBatchBuildStageWallets, positionCashOutBatchReasonInvalidResponse)
	}
	items := make([]wormstore.PositionCashOutBatchItemInput, 0)
	for walletIndex := range batch.Wallets {
		wallet := batch.Wallets[walletIndex]
		if wallet.Ordinal != int32(walletIndex+1) || wallet.WalletID <= 0 ||
			wallet.Address == "" || wallet.CredentialVersion <= 0 {
			return s.failPositionCashOutBatchBuild(ctx, *batch, claimID,
				positionCashOutBatchBuildStageWallets, positionCashOutBatchReasonInvalidResponse)
		}
		if err := s.renewPositionCashOutBatchClaim(ctx, batch.ID, claimID, workerID); err != nil {
			return err
		}
		client, credentialID, err := s.positionCashOutBatchWalletClient(ctx, wallet)
		if err != nil {
			var failure *positionCashOutBatchBuildFailure
			if errors.As(err, &failure) {
				return s.failPositionCashOutBatchBuild(ctx, *batch, claimID, failure.stage, failure.reasonCode)
			}
			return err
		}
		positions, err := s.scanPositionCashOutBatchWallet(
			ctx,
			client,
			wallet,
			credentialID,
			batch.ID,
			claimID,
			workerID,
		)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			var failure *positionCashOutBatchBuildFailure
			if errors.As(err, &failure) {
				return s.failPositionCashOutBatchBuild(ctx, *batch, claimID, failure.stage, failure.reasonCode)
			}
			return err
		}
		if len(positions) == 0 {
			return s.failPositionCashOutBatchBuild(ctx, *batch, claimID,
				positionCashOutBatchBuildStagePositions, positionCashOutBatchReasonNoPositions)
		}
		if len(items)+len(positions) > positionCashOutBatchMaximumPositions {
			return s.failPositionCashOutBatchBuild(ctx, *batch, claimID,
				positionCashOutBatchBuildStagePositions, positionCashOutBatchReasonPositionLimit)
		}
		items = append(items, positions...)
	}
	now := timeNowUTC()
	_, err := s.credentialStore.CompletePositionCashOutBatchBuild(
		ctx,
		wormstore.CompletePositionCashOutBatchBuildRequest{
			BatchID:                batch.ID,
			ClaimID:                claimID,
			Items:                  items,
			AuthorizationExpiresAt: now.Add(positionCashOutBatchAuthorizationTTL),
			Now:                    now,
		},
	)
	s.recordCredentialStoreResult(err)
	return err
}

func (s *Service) positionCashOutBatchWalletClient(
	ctx context.Context,
	wallet wormstore.PositionCashOutBatchWallet,
) (WormAPIClient, int64, error) {
	snapshot, err := s.credentialStore.GetWalletConnectionSnapshot(ctx, wallet.WalletID, wallet.Address)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, 0, err
	}
	credential := snapshot.ActiveCredential
	if snapshot.State != wormstore.ConnectionStateConnected || credential == nil ||
		credential.State != wormstore.CredentialStateActive || credential.Version != wallet.CredentialVersion ||
		effectiveConnectionAddress(*snapshot) != wallet.Address {
		return nil, 0, &positionCashOutBatchBuildFailure{
			stage: positionCashOutBatchBuildStageWallets, reasonCode: positionCashOutBatchReasonConnectionChanged,
			cause: errors.New("position Cash Out batch Wallet connection changed"),
		}
	}
	apiKey, apiSecret, err := s.credentialCipher.decrypt(
		credential.WalletID,
		credential.APIKeyCiphertext,
		credential.APISecretCiphertext,
	)
	if err != nil {
		return nil, credential.ID, &positionCashOutBatchBuildFailure{
			stage: positionCashOutBatchBuildStageWallets, reasonCode: positionCashOutBatchReasonConnectionChanged,
			cause: errors.New("position Cash Out batch credential is unavailable"),
		}
	}
	client, err := s.wormClientFactory.NewAuthenticatedClient(apiKey, apiSecret)
	if err != nil {
		return nil, credential.ID, &positionCashOutBatchBuildFailure{
			stage: positionCashOutBatchBuildStageWallets, reasonCode: positionCashOutBatchReasonConnectionChanged,
			cause: errors.New("position Cash Out batch credential is unavailable"),
		}
	}
	return client, credential.ID, nil
}

func (s *Service) scanPositionCashOutBatchWallet(
	ctx context.Context,
	client WormAPIClient,
	wallet wormstore.PositionCashOutBatchWallet,
	credentialID int64,
	batchID string,
	claimID string,
	workerID string,
) ([]wormstore.PositionCashOutBatchItemInput, error) {
	isClosed := false
	cursor := ""
	seenCursors := make(map[string]struct{})
	seenPubkeys := make(map[string]struct{})
	positions := make([]wormstore.PositionCashOutBatchItemInput, 0)
	pageCount := 0
	for {
		pageCount++
		if pageCount > positionCashOutBatchMaximumPositions/wormPositionPageSize+1 {
			return nil, &positionCashOutBatchBuildFailure{
				stage: positionCashOutBatchBuildStagePositions, reasonCode: positionCashOutBatchReasonInvalidResponse,
				cause: errors.New("position Cash Out batch position pagination is unbounded"),
			}
		}
		if err := s.renewPositionCashOutBatchClaim(ctx, batchID, claimID, workerID); err != nil {
			return nil, err
		}
		attemptCtx, cancel := context.WithTimeout(ctx, s.wormAPIAttemptTimeout)
		response, err := client.ListMarginPositions(attemptCtx, utilworm.ListMarginPositionsOptions{
			PageOptions: utilworm.PageOptions{Limit: wormPositionPageSize, Cursor: cursor},
			IsClosed:    &isClosed,
			Sort:        "-created",
		})
		cancel()
		s.wormCapabilities.recordWormResult(err)
		if err != nil {
			if isWormAuthenticationError(err) {
				s.markWalletReconnectRequired(ctx, wallet.WalletID, wallet.Address, credentialID)
			}
			return nil, &positionCashOutBatchBuildFailure{
				stage: positionCashOutBatchBuildStagePositions, reasonCode: positionCashOutBatchReasonPositionScanFailed,
				cause: fmt.Errorf("list position Cash Out batch positions: %w", err),
			}
		}
		if response == nil || len(response.Positions) > wormPositionPageSize {
			return nil, &positionCashOutBatchBuildFailure{
				stage: positionCashOutBatchBuildStagePositions, reasonCode: positionCashOutBatchReasonInvalidResponse,
				cause: errors.New("position Cash Out batch position page is invalid"),
			}
		}
		for providerIndex := range response.Positions {
			provider := response.Positions[providerIndex]
			mapped, mapErr := wormOpenPositionFromProvider(provider)
			target, inspectErr := utilworm.InspectMarginPositionCashOutTarget(&provider)
			if mapErr != nil || inspectErr != nil || provider.IsClosed || provider.IsLiquidated ||
				mapped.GetCreatedAt() <= 0 || mapped.GetMarket() == nil {
				return nil, &positionCashOutBatchBuildFailure{
					stage: positionCashOutBatchBuildStagePositions, reasonCode: positionCashOutBatchReasonInvalidResponse,
					cause: errors.New("position Cash Out batch position is invalid or not closable"),
				}
			}
			if _, duplicate := seenPubkeys[target.PositionPubkey]; duplicate {
				return nil, &positionCashOutBatchBuildFailure{
					stage: positionCashOutBatchBuildStagePositions, reasonCode: positionCashOutBatchReasonInvalidResponse,
					cause: errors.New("position Cash Out batch position pubkey is duplicated"),
				}
			}
			seenPubkeys[target.PositionPubkey] = struct{}{}
			requestPubkey := ""
			if target.PositionRequestPubkey != nil {
				requestPubkey = *target.PositionRequestPubkey
			}
			positions = append(positions, wormstore.PositionCashOutBatchItemInput{
				WalletOrdinal: wallet.Ordinal, WalletID: wallet.WalletID,
				WalletAddress: wallet.Address, WalletRemark: wallet.Remark,
				CredentialVersion: wallet.CredentialVersion,
				PositionPubkey:    target.PositionPubkey, PositionRequestPubkey: requestPubkey,
				MarketConditionID: target.MarketConditionID, MarketTitle: mapped.GetMarket().GetTitle(),
				IsYes: target.IsYes, Shares: target.TotalShares,
				PositionCreatedAt: time.Unix(target.CreatedAt, 0).UTC(), ProviderState: "open",
			})
			if len(positions) > positionCashOutBatchMaximumPositions {
				return nil, &positionCashOutBatchBuildFailure{
					stage: positionCashOutBatchBuildStagePositions, reasonCode: positionCashOutBatchReasonPositionLimit,
					cause: errors.New("position Cash Out batch position count exceeds the limit"),
				}
			}
		}
		nextCursor, done, cursorErr := nextExecutionPreviewCursor(cursor, response.Meta, seenCursors)
		if cursorErr != nil {
			return nil, &positionCashOutBatchBuildFailure{
				stage: positionCashOutBatchBuildStagePositions, reasonCode: positionCashOutBatchReasonInvalidResponse,
				cause: fmt.Errorf("paginate position Cash Out batch positions: %w", cursorErr),
			}
		}
		if done {
			break
		}
		if len(response.Positions) == 0 {
			return nil, &positionCashOutBatchBuildFailure{
				stage: positionCashOutBatchBuildStagePositions, reasonCode: positionCashOutBatchReasonInvalidResponse,
				cause: errors.New("position Cash Out batch empty position page has a continuation cursor"),
			}
		}
		cursor = nextCursor
	}
	sort.Slice(positions, func(left, right int) bool {
		if positions[left].PositionCreatedAt.Equal(positions[right].PositionCreatedAt) {
			return positions[left].PositionPubkey < positions[right].PositionPubkey
		}
		return positions[left].PositionCreatedAt.After(positions[right].PositionCreatedAt)
	})
	for index := range positions {
		positions[index].PositionOrdinal = int32(index + 1)
	}
	return positions, nil
}

func (s *Service) renewPositionCashOutBatchClaim(
	ctx context.Context,
	batchID string,
	claimID string,
	workerID string,
) error {
	now := timeNowUTC()
	_, err := s.credentialStore.RenewPositionCashOutBatchClaim(
		ctx,
		wormstore.PositionCashOutBatchClaimRequest{
			BatchID: batchID, ClaimID: claimID, WorkerID: workerID,
			LeaseExpiresAt: now.Add(positionCashOutBatchClaimLease), Now: now,
		},
	)
	s.recordCredentialStoreResult(err)
	return err
}

func (s *Service) failPositionCashOutBatchBuild(
	ctx context.Context,
	batch wormstore.PositionCashOutBatch,
	claimID string,
	stage string,
	reasonCode string,
) error {
	_, err := s.credentialStore.FailPositionCashOutBatchBuild(
		ctx,
		wormstore.FailPositionCashOutBatchBuildRequest{
			BatchID: batch.ID, ClaimID: claimID, BuildStage: stage,
			ReasonCode: reasonCode, Now: timeNowUTC(),
		},
	)
	s.recordCredentialStoreResult(err)
	return err
}

func (s *Service) completePositionCashOutBatchBoundary(
	ctx context.Context,
	batch wormstore.PositionCashOutBatch,
	claimID string,
) error {
	_, err := s.credentialStore.CompletePositionCashOutBatchBoundary(
		ctx,
		wormstore.CompletePositionCashOutBatchBoundaryRequest{
			BatchID: batch.ID, ClaimID: claimID, Now: timeNowUTC(),
		},
	)
	s.recordCredentialStoreResult(err)
	return err
}

func (s *Service) activateNextPositionCashOutBatchItem(
	ctx context.Context,
	batch wormstore.PositionCashOutBatch,
	claimID string,
) error {
	item, err := s.loadPositionCashOutBatchItemByOrdinal(ctx, batch, batch.NextItemOrdinal)
	if err != nil {
		return err
	}
	if item.State != wormstore.PositionCashOutBatchItemStatePending || item.ChildCashOutID != "" {
		return errors.New("next position Cash Out batch item is not pending")
	}
	wallet, ok := positionCashOutBatchWalletByOrdinal(batch, item.WalletOrdinal)
	if !ok || wallet.WalletID != item.WalletID || wallet.Address != item.WalletAddress ||
		wallet.CredentialVersion != item.CredentialVersion {
		return s.blockPositionCashOutBatch(ctx, batch, item, claimID,
			wormstore.PositionCashOutBatchItemStateReconciliationRequired,
			wormstore.PositionCashOutBatchStatePaused,
			positionCashOutBatchReasonChildMismatch)
	}
	client, credentialID, clientErr := s.positionCashOutBatchWalletClient(ctx, wallet)
	if clientErr != nil {
		var failure *positionCashOutBatchBuildFailure
		if errors.As(clientErr, &failure) {
			return s.blockPositionCashOutBatch(ctx, batch, item, claimID,
				wormstore.PositionCashOutBatchItemStateReconciliationRequired,
				wormstore.PositionCashOutBatchStatePaused,
				failure.reasonCode)
		}
		return clientErr
	}
	providerState := item.ProviderState
	attemptCtx, cancel := context.WithTimeout(ctx, s.wormAPIAttemptTimeout)
	observed, observeErr := utilworm.ObserveMarginPositionCashOut(
		attemptCtx,
		client,
		positionCashOutBatchFrozenTarget(item),
	)
	cancel()
	s.wormCapabilities.recordWormResult(observeErr)
	if observeErr != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if positionCashOutBatchTemporaryPreflightError(observeErr) {
			_, deferErr := s.credentialStore.DeferPositionCashOutBatchPreflight(
				ctx,
				wormstore.DeferPositionCashOutBatchPreflightRequest{
					BatchID: batch.ID, ItemID: item.ID, ClaimID: claimID, Now: timeNowUTC(),
				},
			)
			s.recordCredentialStoreResult(deferErr)
			return deferErr
		}
		reasonCode := positionCashOutReasonChanged
		if isWormAuthenticationError(observeErr) {
			s.markWalletReconnectRequired(ctx, item.WalletID, item.WalletAddress, credentialID)
			reasonCode = positionCashOutBatchReasonConnectionChanged
		} else if isWormNotFoundError(observeErr) {
			reasonCode = positionCashOutReasonNotFound
		}
		return s.blockPositionCashOutBatch(ctx, batch, item, claimID,
			wormstore.PositionCashOutBatchItemStateReconciliationRequired,
			wormstore.PositionCashOutBatchStatePaused,
			reasonCode)
	} else if observed.TotalShares != item.Shares {
		return s.blockPositionCashOutBatch(ctx, batch, item, claimID,
			wormstore.PositionCashOutBatchItemStateReconciliationRequired,
			wormstore.PositionCashOutBatchStatePaused,
			positionCashOutReasonChanged)
	} else if observed.IsLiquidated || observed.IsClosed {
		reasonCode := positionCashOutReasonChanged
		if observed.IsLiquidated {
			reasonCode = positionCashOutReasonLiquidated
		}
		return s.blockPositionCashOutBatch(ctx, batch, item, claimID,
			wormstore.PositionCashOutBatchItemStateReconciliationRequired,
			wormstore.PositionCashOutBatchStatePaused,
			reasonCode)
	}
	providerState = "open"
	now := timeNowUTC()
	_, err = s.credentialStore.ActivatePositionCashOutBatchItem(
		ctx,
		wormstore.ActivatePositionCashOutBatchItemRequest{
			BatchID: batch.ID, ItemID: item.ID, ClaimID: claimID,
			CashOutID: uuid.NewString(), ProviderState: providerState,
			NextPollAt: now.Add(positionCashOutBatchItemPollInterval), Now: now,
		},
	)
	s.recordCredentialStoreResult(err)
	if err == nil {
		s.wakePositionCashOutWorker()
	}
	return err
}

func positionCashOutBatchTemporaryPreflightError(err error) bool {
	switch classifyWormError(err) {
	case wormErrorRateLimited, wormErrorUnavailable, wormErrorTimeout, wormErrorCancelled:
		return true
	default:
		return false
	}
}

func positionCashOutBatchWalletByOrdinal(
	batch wormstore.PositionCashOutBatch,
	ordinal int32,
) (wormstore.PositionCashOutBatchWallet, bool) {
	for index := range batch.Wallets {
		if batch.Wallets[index].Ordinal == ordinal {
			return batch.Wallets[index], true
		}
	}
	return wormstore.PositionCashOutBatchWallet{}, false
}

func positionCashOutBatchFrozenTarget(
	item wormstore.PositionCashOutBatchItem,
) utilworm.MarginPositionCashOutTarget {
	target := utilworm.MarginPositionCashOutTarget{
		PositionPubkey: item.PositionPubkey, MarketConditionID: item.MarketConditionID,
		IsYes: item.IsYes, CreatedAt: item.PositionCreatedAt.Unix(),
		TotalShares: item.Shares,
	}
	if item.PositionRequestPubkey != "" {
		requestPubkey := item.PositionRequestPubkey
		target.PositionRequestPubkey = &requestPubkey
	}
	return target
}

func (s *Service) loadPositionCashOutBatchItemByOrdinal(
	ctx context.Context,
	batch wormstore.PositionCashOutBatch,
	ordinal int64,
) (wormstore.PositionCashOutBatchItem, error) {
	if ordinal <= 0 || ordinal > batch.PositionCount {
		return wormstore.PositionCashOutBatchItem{}, errors.New("position Cash Out batch item ordinal is invalid")
	}
	pageSize := positionCashOutBatchMaximumPageSize
	page := int32((ordinal-1)/int64(pageSize)) + 1
	items, total, err := s.credentialStore.ListPositionCashOutBatchItems(
		ctx,
		batch.OwnerAccountID,
		batch.ID,
		page,
		pageSize,
	)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return wormstore.PositionCashOutBatchItem{}, err
	}
	if total != batch.PositionCount {
		return wormstore.PositionCashOutBatchItem{}, errors.New("position Cash Out batch item count changed")
	}
	for index := range items {
		if items[index].Ordinal == ordinal {
			return items[index], nil
		}
	}
	return wormstore.PositionCashOutBatchItem{}, errors.New("position Cash Out batch item is missing")
}

func (s *Service) processPositionCashOutBatchItem(
	ctx context.Context,
	batch wormstore.PositionCashOutBatch,
	claimID string,
	workerID string,
) error {
	item := batch.CurrentItem
	if item == nil {
		return errors.New("position Cash Out batch has no current item")
	}
	if item.ChildCashOutID == "" {
		return s.processBlockedPositionCashOutBatchItem(ctx, batch, *item, claimID)
	}
	child, err := s.credentialStore.GetPositionCashOut(ctx, batch.OwnerAccountID, item.ChildCashOutID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return s.blockPositionCashOutBatch(ctx, batch, *item, claimID,
			wormstore.PositionCashOutBatchItemStateReconciliationRequired,
			positionCashOutBatchPreservedBlockingState(batch, wormstore.PositionCashOutBatchStateReconciliationRequired),
			positionCashOutBatchReasonChildMismatch)
	}
	if !positionCashOutBatchChildMatches(batch, *item, *child) {
		return s.blockPositionCashOutBatch(ctx, batch, *item, claimID,
			wormstore.PositionCashOutBatchItemStateReconciliationRequired,
			positionCashOutBatchPreservedBlockingState(batch, wormstore.PositionCashOutBatchStateReconciliationRequired),
			positionCashOutBatchReasonChildMismatch)
	}
	providerState := strings.TrimSpace(child.ProviderState)
	if batch.State == wormstore.PositionCashOutBatchStateTerminateRequested &&
		!positionCashOutBatchMutationWasDispatched(*child) &&
		(child.State == wormstore.PositionCashOutStateFailed ||
			child.State == wormstore.PositionCashOutStateReconciliationRequired) {
		return s.completePositionCashOutBatchBoundary(ctx, batch, claimID)
	}
	if item.State == wormstore.PositionCashOutBatchItemStateReconciliationRequired &&
		child.State != wormstore.PositionCashOutStateCompleted {
		reasonCode := positionCashOutBatchChildReason(*child, positionCashOutBatchReasonCloseUnknown)
		if child.State == wormstore.PositionCashOutStateReconciliationRequired &&
			(batch.State == wormstore.PositionCashOutBatchStatePaused ||
				batch.State == wormstore.PositionCashOutBatchStateReconciliationRequired ||
				batch.State == wormstore.PositionCashOutBatchStateTerminateRequested) {
			child, err = s.reconcileBlockedPositionCashOutBatchChild(ctx, *child, workerID)
			if err != nil {
				// Restore the consumed parent Check checkpoint. A stale child
				// claim can therefore expire and the same user-requested Check
				// continues safely without another command or Close mutation.
				_, deferErr := s.credentialStore.DeferPositionCashOutBatchCheck(
					ctx,
					wormstore.DeferPositionCashOutBatchCheckRequest{
						BatchID: batch.ID, ClaimID: claimID, Now: timeNowUTC(),
					},
				)
				s.recordCredentialStoreResult(deferErr)
				if deferErr != nil {
					return errors.Join(err, deferErr)
				}
				return err
			}
			providerState = strings.TrimSpace(child.ProviderState)
			if !positionCashOutBatchChildMatches(batch, *item, *child) {
				return s.blockPositionCashOutBatch(ctx, batch, *item, claimID,
					wormstore.PositionCashOutBatchItemStateReconciliationRequired,
					positionCashOutBatchPreservedBlockingState(batch, wormstore.PositionCashOutBatchStateReconciliationRequired),
					positionCashOutBatchReasonChildMismatch)
			}
			if child.State == wormstore.PositionCashOutStateCompleted {
				// Fall through to the normal COMPLETED case below. It validates
				// durable dispatch and baseline evidence, then atomically moves
				// this item into the balance gate. The Store preserves an
				// automatic balance wake while keeping the batch paused after a
				// successful late credit.
				goto childState
			}
			reasonCode = positionCashOutBatchChildReason(*child, reasonCode)
		} else if batch.State == wormstore.PositionCashOutBatchStatePaused ||
			batch.State == wormstore.PositionCashOutBatchStateReconciliationRequired ||
			batch.State == wormstore.PositionCashOutBatchStateTerminateRequested {
			var checkErr error
			reasonCode, checkErr = s.checkBlockedPositionCashOutBatchChild(ctx, *child, reasonCode)
			if checkErr != nil {
				return checkErr
			}
		}
		return s.blockPositionCashOutBatch(ctx, batch, *item, claimID,
			wormstore.PositionCashOutBatchItemStateReconciliationRequired,
			positionCashOutBatchPreservedBlockingState(batch, batch.State),
			reasonCode)
	}

childState:
	switch child.State {
	case wormstore.PositionCashOutStateQueued,
		wormstore.PositionCashOutStatePreflighting,
		wormstore.PositionCashOutStateClosing:
		if item.State == wormstore.PositionCashOutBatchItemStateAwaitingPosition {
			return s.recordPositionCashOutBatchItemProgress(
				ctx, batch, *item, claimID, wormstore.PositionCashOutBatchItemStateAwaitingPosition,
				providerState, time.Time{}, positionCashOutBatchItemPollInterval,
			)
		}
		return s.recordPositionCashOutBatchItemProgress(
			ctx, batch, *item, claimID, wormstore.PositionCashOutBatchItemStateClosing,
			providerState, time.Time{}, positionCashOutBatchItemPollInterval,
		)
	case wormstore.PositionCashOutStateAwaitingCompletion:
		return s.recordPositionCashOutBatchItemProgress(
			ctx, batch, *item, claimID, wormstore.PositionCashOutBatchItemStateAwaitingPosition,
			providerState, time.Time{}, positionCashOutBatchItemPollInterval,
		)
	case wormstore.PositionCashOutStateCompleted:
		if !child.ProviderIsClosed || child.ProviderIsLiquidated || child.Attempt == nil ||
			child.Attempt.State == wormstore.PositionCashOutAttemptStatePrepared || item.Baseline == nil ||
			child.CompletedAt.IsZero() {
			return s.blockPositionCashOutBatch(ctx, batch, *item, claimID,
				wormstore.PositionCashOutBatchItemStateReconciliationRequired,
				positionCashOutBatchPreservedBlockingState(batch, wormstore.PositionCashOutBatchStateReconciliationRequired),
				positionCashOutBatchReasonChildMismatch)
		}
		if item.State == wormstore.PositionCashOutBatchItemStateAwaitingBalance {
			return s.observePositionCashOutBatchBalance(ctx, batch, *item, claimID)
		}
		deadline := child.CompletedAt.UTC().Add(positionCashOutBatchBalanceTimeout)
		pollAfter := positionCashOutBatchBalancePoll
		if !deadline.After(timeNowUTC()) {
			// The two-minute payout window starts at the durable CLOSED
			// observation, not when the parent Worker happens to notice it.
			// An overdue recovered item is scheduled immediately for its one
			// final balance observation and then pauses if no increase exists.
			pollAfter = positionCashOutBatchImmediatePoll
		}
		return s.recordPositionCashOutBatchItemProgress(
			ctx, batch, *item, claimID, wormstore.PositionCashOutBatchItemStateAwaitingBalance,
			"closed", deadline, pollAfter,
		)
	case wormstore.PositionCashOutStateFailed,
		wormstore.PositionCashOutStateExpired:
		if positionCashOutBatchMutationWasDispatched(*child) {
			return s.blockPositionCashOutBatch(ctx, batch, *item, claimID,
				wormstore.PositionCashOutBatchItemStateFailed,
				wormstore.PositionCashOutBatchStateFailed,
				positionCashOutBatchChildReason(*child, positionCashOutBatchReasonChildFailed))
		}
		return s.blockPositionCashOutBatch(ctx, batch, *item, claimID,
			wormstore.PositionCashOutBatchItemStateReconciliationRequired,
			positionCashOutBatchPreservedBlockingState(batch, wormstore.PositionCashOutBatchStatePaused),
			positionCashOutBatchChildReason(*child, positionCashOutBatchReasonChildFailed))
	case wormstore.PositionCashOutStateReconciliationRequired:
		return s.blockPositionCashOutBatch(ctx, batch, *item, claimID,
			wormstore.PositionCashOutBatchItemStateReconciliationRequired,
			positionCashOutBatchPreservedBlockingState(batch, wormstore.PositionCashOutBatchStateReconciliationRequired),
			positionCashOutBatchChildReason(*child, positionCashOutBatchReasonCloseUnknown))
	default:
		return s.blockPositionCashOutBatch(ctx, batch, *item, claimID,
			wormstore.PositionCashOutBatchItemStateReconciliationRequired,
			positionCashOutBatchPreservedBlockingState(batch, wormstore.PositionCashOutBatchStateReconciliationRequired),
			positionCashOutBatchReasonChildMismatch)
	}
}

// reconcileBlockedPositionCashOutBatchChild consumes one persisted batch
// Check checkpoint. Batch RECONCILIATION_REQUIRED children are deliberately
// excluded from the generic single-operation recovery scan, so this explicit
// claim is the only caller that may perform the safe status GET. The shared
// single-position worker path is reused because it already enforces exact
// frozen-target matching and cannot dispatch a Close from this state.
func (s *Service) reconcileBlockedPositionCashOutBatchChild(
	ctx context.Context,
	child wormstore.PositionCashOut,
	workerID string,
) (*wormstore.PositionCashOut, error) {
	if child.BatchID == "" || child.BatchItemID == "" ||
		child.State != wormstore.PositionCashOutStateReconciliationRequired {
		return nil, errors.New("blocked batch child is not reconcilable")
	}
	claimID := uuid.NewString()
	now := timeNowUTC()
	claimed, err := s.credentialStore.ClaimPositionCashOut(ctx, wormstore.PositionCashOutClaimRequest{
		CashOutID: child.ID, ClaimID: claimID, WorkerID: workerID,
		LeaseExpiresAt: now.Add(positionCashOutClaimLease), Now: now,
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	if err := s.processClaimedPositionCashOut(ctx, claimed, claimID); err != nil {
		return nil, err
	}
	updated, err := s.credentialStore.GetPositionCashOut(ctx, child.OwnerAccountID, child.ID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) checkBlockedPositionCashOutBatchChild(
	ctx context.Context,
	child wormstore.PositionCashOut,
	fallbackReason string,
) (string, error) {
	client, credentialID, err := s.positionCashOutClient(ctx, child)
	if err != nil {
		return positionCashOutBatchReasonConnectionChanged, nil
	}
	observed, observeErr := s.observePositionCashOut(ctx, client, child)
	if ctx.Err() != nil {
		return fallbackReason, ctx.Err()
	}
	if observeErr != nil {
		if isWormAuthenticationError(observeErr) {
			s.markWalletReconnectRequired(ctx, child.WalletID, child.WalletAddress, credentialID)
			return positionCashOutBatchReasonConnectionChanged, nil
		}
		if isWormNotFoundError(observeErr) {
			return positionCashOutReasonNotFound, nil
		}
		return fallbackReason, nil
	}
	if observed.IsLiquidated {
		return positionCashOutReasonLiquidated, nil
	}
	if positionCashOutBatchMutationWasDispatched(child) {
		if observed.IsClosed {
			s.wakePositionCashOutWorker()
		}
		return positionCashOutBatchReasonCloseUnknown, nil
	}
	return positionCashOutReasonChanged, nil
}

func (s *Service) processBlockedPositionCashOutBatchItem(
	ctx context.Context,
	batch wormstore.PositionCashOutBatch,
	item wormstore.PositionCashOutBatchItem,
	claimID string,
) error {
	if batch.State == wormstore.PositionCashOutBatchStateTerminateRequested {
		return s.completePositionCashOutBatchBoundary(ctx, batch, claimID)
	}
	if item.State != wormstore.PositionCashOutBatchItemStateReconciliationRequired ||
		(batch.State != wormstore.PositionCashOutBatchStatePaused &&
			batch.State != wormstore.PositionCashOutBatchStateReconciliationRequired) {
		return errors.New("position Cash Out batch child operation is missing")
	}
	wallet, ok := positionCashOutBatchWalletByOrdinal(batch, item.WalletOrdinal)
	if !ok {
		return s.blockPositionCashOutBatch(ctx, batch, item, claimID,
			wormstore.PositionCashOutBatchItemStateReconciliationRequired,
			batch.State,
			positionCashOutBatchReasonChildMismatch)
	}
	client, credentialID, err := s.positionCashOutBatchWalletClient(ctx, wallet)
	if err != nil {
		var failure *positionCashOutBatchBuildFailure
		if errors.As(err, &failure) {
			return s.blockPositionCashOutBatch(ctx, batch, item, claimID,
				wormstore.PositionCashOutBatchItemStateReconciliationRequired,
				batch.State,
				failure.reasonCode)
		}
		return err
	}
	attemptCtx, cancel := context.WithTimeout(ctx, s.wormAPIAttemptTimeout)
	observed, observeErr := utilworm.ObserveMarginPositionCashOut(
		attemptCtx,
		client,
		positionCashOutBatchFrozenTarget(item),
	)
	cancel()
	s.wormCapabilities.recordWormResult(observeErr)
	reasonCode := item.ReasonCode
	if reasonCode == "" {
		reasonCode = positionCashOutReasonChanged
	}
	switch {
	case ctx.Err() != nil:
		return ctx.Err()
	case isWormAuthenticationError(observeErr):
		s.markWalletReconnectRequired(ctx, item.WalletID, item.WalletAddress, credentialID)
		reasonCode = positionCashOutBatchReasonConnectionChanged
	case isWormNotFoundError(observeErr):
		reasonCode = positionCashOutReasonNotFound
	case observeErr != nil:
		// A manual Check status performs exactly this one GET. Temporary
		// unavailability retains the previous blocking evidence and never
		// schedules an automatic follow-up while the batch is paused.
	case observed.IsLiquidated:
		reasonCode = positionCashOutReasonLiquidated
	case observed.IsClosed:
		reasonCode = positionCashOutReasonChanged
	default:
		// Even if the target becomes open again, a frozen-batch identity
		// conflict is never auto-resumed. The user must terminate and build a
		// new authoritative batch.
		reasonCode = positionCashOutReasonChanged
	}
	return s.blockPositionCashOutBatch(ctx, batch, item, claimID,
		wormstore.PositionCashOutBatchItemStateReconciliationRequired,
		batch.State,
		reasonCode)
}

func positionCashOutBatchPreservedBlockingState(
	batch wormstore.PositionCashOutBatch,
	fallback wormstore.PositionCashOutBatchState,
) wormstore.PositionCashOutBatchState {
	if batch.State == wormstore.PositionCashOutBatchStateTerminateRequested {
		return wormstore.PositionCashOutBatchStateTerminateRequested
	}
	if batch.State == wormstore.PositionCashOutBatchStatePaused ||
		batch.State == wormstore.PositionCashOutBatchStateReconciliationRequired {
		return batch.State
	}
	return fallback
}

func positionCashOutBatchChildMatches(
	batch wormstore.PositionCashOutBatch,
	item wormstore.PositionCashOutBatchItem,
	child wormstore.PositionCashOut,
) bool {
	return child.ID == item.ChildCashOutID && child.BatchID == batch.ID && child.BatchItemID == item.ID &&
		child.OwnerAccountID == batch.OwnerAccountID && child.WalletID == item.WalletID &&
		child.WalletAddress == item.WalletAddress && child.CredentialVersion == item.CredentialVersion &&
		child.PositionPubkey == item.PositionPubkey && child.PositionRequestPubkey == item.PositionRequestPubkey &&
		child.MarketConditionID == item.MarketConditionID && child.IsYes == item.IsYes &&
		child.PositionCreatedAt.Equal(item.PositionCreatedAt) && child.Shares == item.Shares
}

func positionCashOutBatchMutationWasDispatched(child wormstore.PositionCashOut) bool {
	return child.Attempt != nil && child.Attempt.State != wormstore.PositionCashOutAttemptStatePrepared
}

func positionCashOutBatchChildReason(child wormstore.PositionCashOut, fallback string) string {
	reason := strings.TrimSpace(child.ReasonCode)
	if reason == "" {
		return fallback
	}
	return reason
}

func (s *Service) recordPositionCashOutBatchItemProgress(
	ctx context.Context,
	batch wormstore.PositionCashOutBatch,
	item wormstore.PositionCashOutBatchItem,
	claimID string,
	nextState wormstore.PositionCashOutBatchItemState,
	providerState string,
	balanceDeadline time.Time,
	pollAfter time.Duration,
) error {
	expectedState := item.State
	if expectedState == nextState && expectedState == wormstore.PositionCashOutBatchItemStatePreflighting {
		nextState = wormstore.PositionCashOutBatchItemStateClosing
	}
	now := timeNowUTC()
	_, err := s.credentialStore.RecordPositionCashOutBatchItemState(
		ctx,
		wormstore.RecordPositionCashOutBatchItemStateRequest{
			BatchID: batch.ID, ItemID: item.ID, ClaimID: claimID,
			ExpectedState: expectedState, NextState: nextState,
			ProviderState: providerState, BalanceDeadlineAt: balanceDeadline,
			NextPollAt: now.Add(pollAfter), Now: now,
		},
	)
	s.recordCredentialStoreResult(err)
	return err
}

func (s *Service) observePositionCashOutBatchBalance(
	ctx context.Context,
	batch wormstore.PositionCashOutBatch,
	item wormstore.PositionCashOutBatchItem,
	claimID string,
) error {
	now := timeNowUTC()
	timedOut := !item.BalanceDeadlineAt.IsZero() && !item.BalanceDeadlineAt.After(now)
	observed, err := s.readPositionCashOutBatchBalance(ctx, item.WalletID, item.WalletAddress)
	if err != nil {
		_, deferErr := s.credentialStore.DeferPositionCashOutBatchBalance(
			ctx,
			wormstore.DeferPositionCashOutBatchBalanceRequest{
				BatchID: batch.ID, ItemID: item.ID, ClaimID: claimID,
				TimedOut: timedOut, Now: now,
			},
		)
		s.recordCredentialStoreResult(deferErr)
		if deferErr != nil {
			return deferErr
		}
		if timedOut {
			return nil
		}
		return fmt.Errorf("%s: %w", positionCashOutBatchReasonBalanceUnavailable, err)
	}
	_, err = s.credentialStore.RecordPositionCashOutBatchBalance(
		ctx,
		wormstore.RecordPositionCashOutBatchBalanceRequest{
			BatchID: batch.ID, ItemID: item.ID, ClaimID: claimID,
			Observed: observed, TimedOut: timedOut, Now: now,
		},
	)
	s.recordCredentialStoreResult(err)
	return err
}

func (s *Service) blockPositionCashOutBatch(
	ctx context.Context,
	batch wormstore.PositionCashOutBatch,
	item wormstore.PositionCashOutBatchItem,
	claimID string,
	itemState wormstore.PositionCashOutBatchItemState,
	batchState wormstore.PositionCashOutBatchState,
	reasonCode string,
) error {
	_, err := s.credentialStore.RecordPositionCashOutBatchBlocking(
		ctx,
		wormstore.RecordPositionCashOutBatchBlockingRequest{
			BatchID: batch.ID, ItemID: item.ID, ClaimID: claimID,
			ItemState: itemState, BatchState: batchState,
			ReasonCode: reasonCode, Now: timeNowUTC(),
		},
	)
	s.recordCredentialStoreResult(err)
	return err
}

func positionCashOutBatchWorkerErrorCode(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, wormstore.ErrPositionCashOutBatchClaim):
		return "CLAIM_CHANGED"
	case errors.Is(err, wormstore.ErrTransactionOutcomeUnknown):
		return "STORE_OUTCOME_UNKNOWN"
	case errors.Is(err, context.Canceled):
		return "CANCELLED"
	case errors.Is(err, context.DeadlineExceeded):
		return "TIMEOUT"
	}
	code := classifyWormError(err)
	if code != "" {
		return code
	}
	return "POSITION_CASH_OUT_BATCH_FAILED"
}
