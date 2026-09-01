package wormtrading

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	wormPositionRequestStates = "created,funding_processing,processing,order_placed,refund_processing"
	wormPositionPageSize      = 100
)

var decimalStringPattern = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?$`)

type positionStreamResult struct {
	positions []*apiclient.WormOpenPosition
	truncated bool
	err       error
}

type requestStreamResult struct {
	requests  []*apiclient.WormInFlightPositionRequest
	truncated bool
	err       error
}

func (s *Service) BatchGetWalletPositionSnapshots(
	ctx context.Context,
	req *apiclient.BatchGetWalletPositionSnapshotsRequest,
) (*apiclient.BatchGetWalletPositionSnapshotsResponse, error) {
	if err := s.requireCredentialCapability(); err != nil {
		return nil, err
	}
	if req == nil || len(req.GetRefs()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "refs must contain at least one wallet")
	}
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	if len(req.GetRefs()) > maxWalletBalanceReferences {
		return nil, status.Errorf(codes.InvalidArgument, "refs must contain at most %d wallets", maxWalletBalanceReferences)
	}

	refs := make([]wormstore.WalletReference, 0, len(req.GetRefs()))
	walletIDs := make(map[int64]struct{}, len(req.GetRefs()))
	for _, ref := range req.GetRefs() {
		if ref == nil {
			return nil, status.Error(codes.InvalidArgument, "every ref must identify a wallet")
		}
		walletID, address, err := normalizeWormWalletReference(ref.GetWalletId(), ref.GetAddress())
		if err != nil {
			return nil, err
		}
		if _, exists := walletIDs[walletID]; exists {
			return nil, status.Error(codes.InvalidArgument, "wallet_id values must be unique")
		}
		walletIDs[walletID] = struct{}{}
		refs = append(refs, wormstore.WalletReference{WalletID: walletID, Address: address})
	}

	pageCtx, cancel := context.WithTimeout(ctx, s.wormPositionBudget)
	defer cancel()
	storeSnapshots, err := s.credentialStore.ListWalletConnectionSnapshots(pageCtx, refs)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "Worm credential store is unavailable")
	}
	if err := validateStoredPositionSnapshots(refs, storeSnapshots); err != nil {
		return nil, status.Error(codes.Internal, "Worm credential store returned mismatched wallet connections")
	}

	items := make([]*apiclient.WalletPositionSnapshot, len(refs))
	connected := make([]bool, len(refs))
	positionResults := make([]positionStreamResult, len(refs))
	requestResults := make([]requestStreamResult, len(refs))
	reconnectMarkers := make([]sync.Once, len(refs))
	markReconnectRequired := func(resultIndex int) {
		connection := storeSnapshots[resultIndex]
		if connection.ActiveCredential == nil {
			return
		}
		reconnectMarkers[resultIndex].Do(func() {
			s.markWalletReconnectRequired(
				pageCtx,
				connection.WalletID,
				effectiveConnectionAddress(connection),
				connection.ActiveCredential.ID,
			)
		})
	}

	var workers sync.WaitGroup
	for index := range refs {
		connection := storeSnapshots[index]
		items[index] = unavailableWalletPositionSnapshot(connection)
		if connection.State != wormstore.ConnectionStateConnected {
			continue
		}
		connected[index] = true
		if connection.ActiveCredential == nil {
			continue
		}
		apiKey, apiSecret, decryptErr := s.credentialCipher.decrypt(
			connection.WalletID,
			connection.ActiveCredential.APIKeyCiphertext,
			connection.ActiveCredential.APISecretCiphertext,
		)
		if decryptErr != nil {
			positionResults[index].err = errors.New("stored Worm credential is unavailable")
			requestResults[index].err = errors.New("stored Worm credential is unavailable")
			continue
		}
		client, clientErr := s.wormClientFactory.NewAuthenticatedClient(apiKey, apiSecret)
		if clientErr != nil {
			positionResults[index].err = errors.New("stored Worm credential is unavailable")
			requestResults[index].err = errors.New("stored Worm credential is unavailable")
			continue
		}

		workers.Add(2)
		go func(resultIndex int, client WormAPIClient) {
			defer workers.Done()
			positionResults[resultIndex] = s.fetchOpenPositions(pageCtx, client)
			if isWormAuthenticationError(positionResults[resultIndex].err) {
				markReconnectRequired(resultIndex)
			}
		}(index, client)
		go func(resultIndex int, client WormAPIClient) {
			defer workers.Done()
			requestResults[resultIndex] = s.fetchInFlightRequests(pageCtx, client)
			if isWormAuthenticationError(requestResults[resultIndex].err) {
				markReconnectRequired(resultIndex)
			}
		}(index, client)
	}
	workers.Wait()

	var openPositionCount int32
	var inFlightRequestCount int32
	pageStatus := walletBalanceComplete
	connectedWalletCount := 0
	unavailableWalletCount := 0
	hasPartialWallet := false
	allConnectedStreamsTemporarilyUnavailable := true
	for index := range refs {
		if !connected[index] {
			continue
		}
		connectedWalletCount++
		connection := storeSnapshots[index]
		positionResult := positionResults[index]
		requestResult := requestResults[index]
		if connection.ActiveCredential == nil {
			positionResult.err = errors.New("stored Worm credential is unavailable")
			requestResult.err = errors.New("stored Worm credential is unavailable")
		}
		if isWormAuthenticationError(positionResult.err) || isWormAuthenticationError(requestResult.err) {
			connection.State = wormstore.ConnectionStateReconnectRequired
			connection.WarningCode = reconnectRequiredWarning(connection.WarningCode)
		}
		item := walletPositionSnapshotFromResults(connection, positionResult, requestResult)
		items[index] = item
		openPositionCount += int32(len(item.OpenPositions))
		inFlightRequestCount += int32(len(item.InFlightRequests))
		if item.Status == walletBalanceUnavailable {
			unavailableWalletCount++
		} else if item.Status == walletBalancePartial {
			hasPartialWallet = true
		}
		if item.OpenPositionsStatus.Availability == availabilityAvailable ||
			item.InFlightRequestsStatus.Availability == availabilityAvailable ||
			!isTemporaryWormErrorCode(item.OpenPositionsStatus.ErrorCode) ||
			!isTemporaryWormErrorCode(item.InFlightRequestsStatus.ErrorCode) {
			allConnectedStreamsTemporarilyUnavailable = false
		}
	}
	if connectedWalletCount == 0 {
		pageStatus = walletBalanceComplete
		allConnectedStreamsTemporarilyUnavailable = false
	} else if unavailableWalletCount == connectedWalletCount {
		pageStatus = walletBalanceUnavailable
	} else if unavailableWalletCount > 0 || hasPartialWallet {
		pageStatus = walletBalancePartial
	}
	if ctx.Err() != nil {
		return nil, status.FromContextError(ctx.Err()).Err()
	}
	if allConnectedStreamsTemporarilyUnavailable {
		return nil, status.Error(codes.Unavailable, "Worm position provider is unavailable")
	}
	if err := s.projectPositionCashOutAvailability(pageCtx, ownerAccountID, items); err != nil {
		return nil, err
	}

	return &apiclient.BatchGetWalletPositionSnapshotsResponse{
		Items:                items,
		FetchedAt:            timeNowUTC().Unix(),
		OpenPositionCount:    openPositionCount,
		InFlightRequestCount: inFlightRequestCount,
		Status:               pageStatus,
	}, nil
}

func (s *Service) projectPositionCashOutAvailability(
	ctx context.Context,
	ownerAccountID string,
	items []*apiclient.WalletPositionSnapshot,
) error {
	walletIDs := make([]int64, 0, len(items))
	for _, item := range items {
		if item != nil && len(item.GetOpenPositions()) > 0 {
			walletIDs = append(walletIDs, item.GetWalletId())
		}
	}
	if len(walletIDs) == 0 {
		return nil
	}
	activeCashOuts, err := s.credentialStore.ListActivePositionCashOuts(ctx, ownerAccountID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return status.Error(codes.Unavailable, "Worm position Cash Out store is unavailable")
	}
	executionWalletIDs, err := s.credentialStore.ListActiveExecutionRunWalletIDs(ctx, ownerAccountID, walletIDs)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return status.Error(codes.Unavailable, "Worm execution guard store is unavailable")
	}
	activeBatch, batchErr := s.credentialStore.GetActivePositionCashOutBatch(ctx, ownerAccountID)
	s.recordCredentialStoreResult(batchErr)
	if batchErr != nil && !errors.Is(batchErr, wormstore.ErrPositionCashOutBatchNotFound) {
		return status.Error(codes.Unavailable, "Worm position Cash Out Batch store is unavailable")
	}
	batchWallets := make(map[int64]struct{})
	batchItems := make(map[string]wormstore.PositionCashOutBatchItem)
	if batchErr == nil && activeBatch != nil {
		for _, wallet := range activeBatch.Wallets {
			batchWallets[wallet.WalletID] = struct{}{}
		}
		frozenItems, err := s.listAllPositionCashOutBatchItems(ctx, *activeBatch)
		if err != nil {
			return err
		}
		for _, frozen := range frozenItems {
			batchItems[positionCashOutBatchProjectionKey(frozen.WalletID, frozen.PositionPubkey)] = frozen
		}
	}
	activeByWallet := make(map[int64]wormstore.PositionCashOut, len(activeCashOuts))
	for _, cashOut := range activeCashOuts {
		activeByWallet[cashOut.WalletID] = cashOut
	}
	executionWallets := make(map[int64]struct{}, len(executionWalletIDs))
	for _, walletID := range executionWalletIDs {
		executionWallets[walletID] = struct{}{}
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		cashOut, hasCashOut := activeByWallet[item.GetWalletId()]
		_, executionActive := executionWallets[item.GetWalletId()]
		_, batchActive := batchWallets[item.GetWalletId()]
		for _, position := range item.GetOpenPositions() {
			if position == nil {
				continue
			}
			summary := &apiclient.WormPositionCashOutSummary{AllowedAction: "CASH_OUT"}
			switch {
			case position.GetIsLiquidated():
				summary.ReasonCode = "POSITION_LIQUIDATED"
				summary.AllowedAction = "NONE"
			case position.GetIsClosed():
				summary.ReasonCode = "POSITION_NOT_CLOSABLE"
				summary.AllowedAction = "NONE"
			case batchActive:
				frozen, frozenInBatch := batchItems[positionCashOutBatchProjectionKey(
					item.GetWalletId(),
					position.GetPubkey(),
				)]
				summary = positionCashOutBatchSummaryToProto(activeBatch, frozen, frozenInBatch)
			case hasCashOut && cashOut.PositionPubkey == position.GetPubkey():
				summary = positionCashOutSummaryToProto(cashOut)
			case hasCashOut:
				summary.OperationId = cashOut.ID
				summary.State = string(cashOut.State)
				summary.Revision = cashOut.Revision
				summary.ReasonCode = "WALLET_CASH_OUT_ACTIVE"
				summary.UpdatedAt = cashOut.UpdatedAt.Unix()
				summary.AllowedAction = "NONE"
			case executionActive:
				summary.ReasonCode = "WALLET_EXECUTION_ACTIVE"
				summary.AllowedAction = "NONE"
			}
			position.CashOut = summary
		}
	}
	return nil
}

func (s *Service) listAllPositionCashOutBatchItems(
	ctx context.Context,
	batch wormstore.PositionCashOutBatch,
) ([]wormstore.PositionCashOutBatchItem, error) {
	if batch.PositionCount == 0 {
		return nil, nil
	}
	result := make([]wormstore.PositionCashOutBatchItem, 0, batch.PositionCount)
	for page := int32(1); ; page++ {
		pageItems, total, err := s.credentialStore.ListPositionCashOutBatchItems(
			ctx,
			batch.OwnerAccountID,
			batch.ID,
			page,
			positionCashOutBatchMaximumPageSize,
		)
		s.recordCredentialStoreResult(err)
		if err != nil {
			return nil, status.Error(codes.Unavailable, "Worm position Cash Out Batch store is unavailable")
		}
		if total != batch.PositionCount || len(pageItems) == 0 {
			return nil, status.Error(codes.Internal, "Worm position Cash Out Batch store returned incomplete items")
		}
		result = append(result, pageItems...)
		if int64(len(result)) >= total {
			if int64(len(result)) != total {
				return nil, status.Error(codes.Internal, "Worm position Cash Out Batch store returned invalid items")
			}
			return result, nil
		}
	}
}

func positionCashOutBatchProjectionKey(walletID int64, positionPubkey string) string {
	return fmt.Sprintf("%d\x00%s", walletID, positionPubkey)
}

func positionCashOutBatchSummaryToProto(
	batch *wormstore.PositionCashOutBatch,
	item wormstore.PositionCashOutBatchItem,
	frozen bool,
) *apiclient.WormPositionCashOutSummary {
	if batch == nil {
		return &apiclient.WormPositionCashOutSummary{
			ReasonCode: "WALLET_CASH_OUT_BATCH_ACTIVE", AllowedAction: "NONE",
		}
	}
	reasonCode := batch.ReasonCode
	updatedAt := batch.UpdatedAt
	itemState := ""
	if frozen {
		itemState = string(item.State)
		if item.ReasonCode != "" {
			reasonCode = item.ReasonCode
		}
		if item.UpdatedAt.After(updatedAt) {
			updatedAt = item.UpdatedAt
		}
	}
	if reasonCode == "" {
		reasonCode = "WALLET_CASH_OUT_BATCH_ACTIVE"
	}
	return &apiclient.WormPositionCashOutSummary{
		ReasonCode:          reasonCode,
		AllowedAction:       "NONE",
		Revision:            batch.Revision,
		UpdatedAt:           updatedAt.Unix(),
		BatchId:             batch.ID,
		BatchState:          string(batch.State),
		BatchItemState:      itemState,
		BatchLockReasonCode: "WALLET_CASH_OUT_BATCH_ACTIVE",
	}
}

func positionCashOutSummaryToProto(cashOut wormstore.PositionCashOut) *apiclient.WormPositionCashOutSummary {
	allowedAction := "NONE"
	switch cashOut.State {
	case wormstore.PositionCashOutStateAwaitingAuthorization:
		allowedAction = "AUTHORIZE_CASH_OUT"
	case wormstore.PositionCashOutStateReconciliationRequired:
		if cashOut.ClaimID == "" && cashOut.ReconcileRequestedAt.IsZero() {
			allowedAction = "CHECK_STATUS"
		}
	}
	return &apiclient.WormPositionCashOutSummary{
		OperationId:   cashOut.ID,
		State:         string(cashOut.State),
		ReasonCode:    cashOut.ReasonCode,
		AllowedAction: allowedAction,
		Revision:      cashOut.Revision,
		UpdatedAt:     cashOut.UpdatedAt.Unix(),
	}
}

func (s *Service) fetchOpenPositions(ctx context.Context, client WormAPIClient) positionStreamResult {
	if err := acquireWormPositionSlot(ctx, s.wormPositionSemaphore); err != nil {
		return positionStreamResult{err: err}
	}
	defer releaseWormPositionSlot(s.wormPositionSemaphore)
	attemptCtx, cancel := context.WithTimeout(ctx, s.wormAPIAttemptTimeout)
	defer cancel()
	isClosed := false
	response, err := client.ListMarginPositions(attemptCtx, worm.ListMarginPositionsOptions{
		PageOptions: worm.PageOptions{Limit: wormPositionPageSize},
		IsClosed:    &isClosed,
		Sort:        "-created",
	})
	s.wormCapabilities.recordWormResult(err)
	if err != nil {
		return positionStreamResult{err: err}
	}
	if response == nil {
		return positionStreamResult{err: errors.New("invalid Worm positions response")}
	}
	positions := make([]*apiclient.WormOpenPosition, 0, len(response.Positions))
	for _, position := range response.Positions {
		if position.IsClosed {
			continue
		}
		mapped, err := wormOpenPositionFromProvider(position)
		if err != nil {
			s.wormCapabilities.recordWormResult(err)
			return positionStreamResult{err: err}
		}
		positions = append(positions, mapped)
	}
	sort.SliceStable(positions, func(i, j int) bool { return positions[i].CreatedAt > positions[j].CreatedAt })
	return positionStreamResult{
		positions: positions,
		truncated: response.Meta.NextCursor != nil && strings.TrimSpace(*response.Meta.NextCursor) != "",
	}
}

func (s *Service) fetchInFlightRequests(ctx context.Context, client WormAPIClient) requestStreamResult {
	if err := acquireWormPositionSlot(ctx, s.wormPositionSemaphore); err != nil {
		return requestStreamResult{err: err}
	}
	defer releaseWormPositionSlot(s.wormPositionSemaphore)
	attemptCtx, cancel := context.WithTimeout(ctx, s.wormAPIAttemptTimeout)
	defer cancel()
	response, err := client.ListPositionRequests(attemptCtx, worm.ListPositionRequestsOptions{
		PageOptions: worm.PageOptions{Limit: wormPositionPageSize},
		States:      wormPositionRequestStates,
		Sort:        "-created",
	})
	s.wormCapabilities.recordWormResult(err)
	if err != nil {
		return requestStreamResult{err: err}
	}
	if response == nil {
		return requestStreamResult{err: errors.New("invalid Worm position requests response")}
	}
	requests := make([]*apiclient.WormInFlightPositionRequest, 0, len(response.Requests))
	for _, request := range response.Requests {
		if !isInFlightPositionRequestState(request.State) {
			continue
		}
		mapped, err := wormInFlightRequestFromProvider(request)
		if err != nil {
			s.wormCapabilities.recordWormResult(err)
			return requestStreamResult{err: err}
		}
		requests = append(requests, mapped)
	}
	sort.SliceStable(requests, func(i, j int) bool { return requests[i].CreatedAt > requests[j].CreatedAt })
	return requestStreamResult{
		requests:  requests,
		truncated: response.Meta.NextCursor != nil && strings.TrimSpace(*response.Meta.NextCursor) != "",
	}
}

func walletPositionSnapshotFromResults(
	connection wormstore.WalletConnectionSnapshot,
	positionResult positionStreamResult,
	requestResult requestStreamResult,
) *apiclient.WalletPositionSnapshot {
	positionStatus := wormPositionStreamStatus(positionResult.err, positionResult.truncated)
	requestStatus := wormPositionStreamStatus(requestResult.err, requestResult.truncated)
	positions := positionResult.positions
	requests := requestResult.requests
	if positionResult.err != nil {
		positions = nil
	}
	if requestResult.err != nil {
		requests = nil
	}
	if positionResult.err == nil && requestResult.err == nil {
		requests = suppressPositionBackedRequests(positions, requests)
	}
	itemStatus := walletBalanceUnavailable
	positionsAvailable := positionStatus.Availability == availabilityAvailable
	requestsAvailable := requestStatus.Availability == availabilityAvailable
	if positionsAvailable && requestsAvailable {
		itemStatus = walletBalanceComplete
	} else if positionsAvailable || requestsAvailable {
		itemStatus = walletBalancePartial
	}
	return &apiclient.WalletPositionSnapshot{
		WalletId:               connection.WalletID,
		Address:                effectiveConnectionAddress(connection),
		Connection:             wormWalletConnectionFromStore(connection),
		OpenPositions:          positions,
		OpenPositionsStatus:    positionStatus,
		InFlightRequests:       requests,
		InFlightRequestsStatus: requestStatus,
		Status:                 itemStatus,
	}
}

func unavailableWalletPositionSnapshot(connection wormstore.WalletConnectionSnapshot) *apiclient.WalletPositionSnapshot {
	errorCode := wormErrorCredentialUnavailable
	if connection.State == wormstore.ConnectionStateReconnectRequired {
		errorCode = wormErrorReconnectRequired
	}
	streamStatus := func() *apiclient.WormPositionStreamStatus {
		return &apiclient.WormPositionStreamStatus{Availability: availabilityUnavailable, ErrorCode: errorCode}
	}
	return &apiclient.WalletPositionSnapshot{
		WalletId:               connection.WalletID,
		Address:                effectiveConnectionAddress(connection),
		Connection:             wormWalletConnectionFromStore(connection),
		OpenPositionsStatus:    streamStatus(),
		InFlightRequestsStatus: streamStatus(),
		Status:                 walletBalanceUnavailable,
	}
}

func wormPositionStreamStatus(err error, truncated bool) *apiclient.WormPositionStreamStatus {
	if err == nil {
		return &apiclient.WormPositionStreamStatus{
			Availability: availabilityAvailable,
			Truncated:    truncated,
		}
	}
	code := classifyWormError(err)
	if strings.Contains(err.Error(), "stored Worm credential") {
		code = wormErrorCredentialUnavailable
	}
	return &apiclient.WormPositionStreamStatus{
		Availability: availabilityUnavailable,
		ErrorCode:    code,
	}
}

func suppressPositionBackedRequests(
	positions []*apiclient.WormOpenPosition,
	requests []*apiclient.WormInFlightPositionRequest,
) []*apiclient.WormInFlightPositionRequest {
	positionRequestKeys := make(map[string]struct{}, len(positions))
	for _, position := range positions {
		if position == nil || position.PositionRequestPubkey == nil || strings.TrimSpace(position.PositionRequestPubkey.Value) == "" {
			continue
		}
		positionRequestKeys[position.PositionRequestPubkey.Value] = struct{}{}
	}
	filtered := make([]*apiclient.WormInFlightPositionRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		if _, duplicate := positionRequestKeys[request.Pubkey]; duplicate {
			continue
		}
		filtered = append(filtered, request)
	}
	return filtered
}

func wormOpenPositionFromProvider(position worm.MarginPosition) (*apiclient.WormOpenPosition, error) {
	if !isCanonicalNonEmptyString(position.Pubkey) || !isCanonicalNonEmptyString(position.Market.ConditionID) ||
		!isCanonicalNonEmptyString(position.Market.Title) ||
		!validNonNegativeDecimalStrings(position.Leverage, position.TotalShares, position.AvgEntryPrice,
			position.UserLiquidity, position.TotalLiquidity) ||
		!isDecimalString(position.RealizedPnL) {
		return nil, errors.New("invalid Worm open position response")
	}
	if position.LiquidationPrice != nil && !isNonNegativeDecimalString(*position.LiquidationPrice) {
		return nil, errors.New("invalid Worm open position response")
	}
	if position.UnrealizedPnL != nil && !isDecimalString(*position.UnrealizedPnL) {
		return nil, errors.New("invalid Worm open position response")
	}
	if position.Market.LastTradePrice != nil && !isNonNegativeDecimalString(*position.Market.LastTradePrice) {
		return nil, errors.New("invalid Worm open position response")
	}
	return &apiclient.WormOpenPosition{
		Pubkey:                position.Pubkey,
		PositionRequestPubkey: wormOptionalString(position.PositionRequestPubkey),
		Market:                wormPositionMarketFromProvider(position.Market),
		IsYes:                 position.IsYes,
		Leverage:              position.Leverage,
		TotalShares:           position.TotalShares,
		AverageEntryPrice:     position.AvgEntryPrice,
		UserLiquidity:         position.UserLiquidity,
		TotalLiquidity:        position.TotalLiquidity,
		LiquidationPrice:      wormOptionalString(position.LiquidationPrice),
		UnrealizedPnl:         wormOptionalString(position.UnrealizedPnL),
		RealizedPnl:           position.RealizedPnL,
		IsClosed:              position.IsClosed,
		IsLiquidated:          position.IsLiquidated,
		IsClaimed:             position.IsClaimed,
		CreatedAt:             unixValue(position.Created),
	}, nil
}

func wormInFlightRequestFromProvider(request worm.PositionRequest) (*apiclient.WormInFlightPositionRequest, error) {
	requestType := strings.ToLower(strings.TrimSpace(request.Type))
	if !isCanonicalNonEmptyString(request.Pubkey) || (requestType != "market" && requestType != "limit") ||
		!isCanonicalNonEmptyString(request.State) || request.Market == nil ||
		!isCanonicalNonEmptyString(request.Market.ConditionID) || !isCanonicalNonEmptyString(request.Market.Title) ||
		!validNonNegativeDecimalStrings(request.Leverage, request.Funds) {
		return nil, errors.New("invalid Worm position request response")
	}
	for _, value := range []*string{request.Price, request.Shares} {
		if value != nil && !isNonNegativeDecimalString(*value) {
			return nil, errors.New("invalid Worm position request response")
		}
	}
	if request.Market.LastTradePrice != nil && !isNonNegativeDecimalString(*request.Market.LastTradePrice) {
		return nil, errors.New("invalid Worm position request response")
	}
	return &apiclient.WormInFlightPositionRequest{
		Pubkey:     request.Pubkey,
		Type:       request.Type,
		State:      request.State,
		OrderState: wormOptionalString(request.OrderState),
		Market:     wormPositionMarketFromProvider(*request.Market),
		IsYes:      request.IsYes,
		Leverage:   request.Leverage,
		Funds:      request.Funds,
		Price:      wormOptionalString(request.Price),
		Shares:     wormOptionalString(request.Shares),
		CreatedAt:  unixValue(request.Created),
	}, nil
}

func wormPositionMarketFromProvider(market worm.MarketSummary) *apiclient.WormPositionMarketSummary {
	result := &apiclient.WormPositionMarketSummary{
		ConditionId:      market.ConditionID,
		Title:            market.Title,
		Logo:             wormOptionalString(market.Logo),
		LatestTradePrice: wormOptionalString(market.LastTradePrice),
	}
	if market.Event != nil {
		result.Event = &apiclient.WormEventSummary{
			ConditionId: market.Event.ConditionID,
			Title:       market.Event.Title,
			Logo:        wormOptionalString(market.Event.Logo),
		}
	}
	return result
}

func wormOptionalString(value *string) *apiclient.WormOptionalString {
	if value == nil {
		return nil
	}
	return &apiclient.WormOptionalString{Value: *value}
}

func unixValue(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func validateStoredPositionSnapshots(refs []wormstore.WalletReference, snapshots []wormstore.WalletConnectionSnapshot) error {
	if len(refs) != len(snapshots) {
		return errors.New("wallet connection count mismatch")
	}
	for index := range refs {
		snapshot := snapshots[index]
		if snapshot.WalletID != refs[index].WalletID || snapshot.RequestedAddress != refs[index].Address {
			return errors.New("wallet connection order mismatch")
		}
		if snapshot.StoredAddress != "" && snapshot.StoredAddress != refs[index].Address {
			return errors.New("wallet connection address mismatch")
		}
		if snapshot.ActiveCredential != nil && snapshot.ActiveCredential.WalletID != snapshot.WalletID {
			return errors.New("wallet credential mismatch")
		}
	}
	return nil
}

func effectiveConnectionAddress(snapshot wormstore.WalletConnectionSnapshot) string {
	if snapshot.StoredAddress != "" {
		return snapshot.StoredAddress
	}
	return snapshot.RequestedAddress
}

func (s *Service) markWalletReconnectRequired(ctx context.Context, walletID int64, address string, activeCredentialID int64) {
	err := s.credentialStore.MarkReconnectRequired(ctx, walletID, address, activeCredentialID, wormErrorReconnectRequired, timeNowUTC())
	s.recordCredentialStoreResult(err)
}

func reconnectRequiredWarning(existing string) string {
	switch existing {
	case wormstore.WarningCredentialRevocationRequired,
		wormstore.WarningCredentialRevocationPending:
		return existing
	default:
		return wormErrorReconnectRequired
	}
}

func acquireWormPositionSlot(ctx context.Context, semaphore chan struct{}) error {
	select {
	case semaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseWormPositionSlot(semaphore chan struct{}) {
	<-semaphore
}

func isInFlightPositionRequestState(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "created", "funding_processing", "processing", "order_placed", "refund_processing":
		return true
	default:
		return false
	}
}

func validNonNegativeDecimalStrings(values ...string) bool {
	for _, value := range values {
		if !isNonNegativeDecimalString(value) {
			return false
		}
	}
	return true
}

func isNonNegativeDecimalString(value string) bool {
	return isDecimalString(value) && !strings.HasPrefix(value, "-")
}

func isCanonicalNonEmptyString(value string) bool {
	return value != "" && value == strings.TrimSpace(value) && !containsControlCharacter(value)
}

func isDecimalString(value string) bool {
	return value != "" && value == strings.TrimSpace(value) && decimalStringPattern.MatchString(value)
}
