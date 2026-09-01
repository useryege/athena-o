package wormtrading

import (
	"context"
	"errors"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
)

const (
	positionCashOutBatchAuthorizationTTL = 5 * time.Minute
	positionCashOutBatchMaximumWallets   = 100
	positionCashOutBatchMaximumPositions = 1000
	positionCashOutBatchMaximumPageSize  = int32(100)
)

func (s *Service) CreatePositionCashOutBatch(
	ctx context.Context,
	req *apiclient.CreatePositionCashOutBatchRequest,
) (*apiclient.CreatePositionCashOutBatchResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "position Cash Out Batch request is required")
	}
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	commandID, err := normalizePositionCashOutUUID(req.GetCommandId(), "commandId")
	if err != nil {
		return nil, err
	}
	if len(req.GetWallets()) == 0 || len(req.GetWallets()) > positionCashOutBatchMaximumWallets {
		return nil, status.Errorf(codes.InvalidArgument, "wallets must contain between 1 and %d entries", positionCashOutBatchMaximumWallets)
	}
	wallets := make([]wormstore.PositionCashOutBatchWalletInput, 0, len(req.GetWallets()))
	seenIDs := make(map[int64]struct{}, len(req.GetWallets()))
	seenAddresses := make(map[string]struct{}, len(req.GetWallets()))
	for _, wallet := range req.GetWallets() {
		if wallet == nil {
			return nil, status.Error(codes.InvalidArgument, "position Cash Out Batch wallet is required")
		}
		walletID, address, normalizeErr := normalizeWormWalletReference(wallet.GetWalletId(), wallet.GetAddress())
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		if _, duplicate := seenIDs[walletID]; duplicate {
			return nil, status.Error(codes.InvalidArgument, "position Cash Out Batch wallet IDs must be unique")
		}
		if _, duplicate := seenAddresses[address]; duplicate {
			return nil, status.Error(codes.InvalidArgument, "position Cash Out Batch wallet addresses must be unique")
		}
		seenIDs[walletID] = struct{}{}
		seenAddresses[address] = struct{}{}
		wallets = append(wallets, wormstore.PositionCashOutBatchWalletInput{
			WalletID: walletID, Address: address, Remark: strings.TrimSpace(wallet.GetRemark()),
			AvatarKind: strings.TrimSpace(wallet.GetAvatarKind()), AvatarPresetID: strings.TrimSpace(wallet.GetAvatarPresetId()),
			AvatarURL: strings.TrimSpace(wallet.GetAvatarUrl()),
		})
	}
	now := timeNowUTC()
	batch, err := s.credentialStore.CreatePositionCashOutBatch(ctx, wormstore.CreatePositionCashOutBatchRequest{
		OwnerAccountID: ownerAccountID, CommandID: commandID, Wallets: wallets,
		AuthorizationExpiresAt: now.Add(positionCashOutBatchAuthorizationTTL), Now: now,
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, positionCashOutBatchRPCError(err)
	}
	s.wakePositionCashOutBatchWorker()
	return &apiclient.CreatePositionCashOutBatchResponse{Batch: positionCashOutBatchToProto(batch)}, nil
}

func (s *Service) GetPositionCashOutBatch(
	ctx context.Context,
	req *apiclient.GetPositionCashOutBatchRequest,
) (*apiclient.GetPositionCashOutBatchResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "position Cash Out Batch request is required")
	}
	owner, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	id, err := normalizePositionCashOutUUID(req.GetId(), "position Cash Out Batch ID")
	if err != nil {
		return nil, err
	}
	batch, err := s.credentialStore.GetPositionCashOutBatch(ctx, owner, id)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, positionCashOutBatchRPCError(err)
	}
	return &apiclient.GetPositionCashOutBatchResponse{Batch: positionCashOutBatchToProto(batch)}, nil
}

func (s *Service) GetActivePositionCashOutBatch(
	ctx context.Context,
	req *apiclient.GetActivePositionCashOutBatchRequest,
) (*apiclient.GetActivePositionCashOutBatchResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "active position Cash Out Batch request is required")
	}
	owner, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	batch, err := s.credentialStore.GetActivePositionCashOutBatch(ctx, owner)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, positionCashOutBatchRPCError(err)
	}
	return &apiclient.GetActivePositionCashOutBatchResponse{Batch: positionCashOutBatchToProto(batch)}, nil
}

func (s *Service) ListPositionCashOutBatchItems(
	ctx context.Context,
	req *apiclient.ListPositionCashOutBatchItemsRequest,
) (*apiclient.ListPositionCashOutBatchItemsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "position Cash Out Batch item request is required")
	}
	owner, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	id, err := normalizePositionCashOutUUID(req.GetId(), "position Cash Out Batch ID")
	if err != nil {
		return nil, err
	}
	page, pageSize, err := normalizePositionCashOutBatchPage(req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	items, total, err := s.credentialStore.ListPositionCashOutBatchItems(ctx, owner, id, page, pageSize)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, positionCashOutBatchRPCError(err)
	}
	output := make([]*apiclient.PositionCashOutBatchItem, 0, len(items))
	for index := range items {
		output = append(output, positionCashOutBatchItemToProto(items[index]))
	}
	return &apiclient.ListPositionCashOutBatchItemsResponse{Items: output, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) AuthorizePositionCashOutBatch(
	ctx context.Context,
	req *apiclient.AuthorizePositionCashOutBatchRequest,
) (*apiclient.AuthorizePositionCashOutBatchResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "position Cash Out Batch authorization is required")
	}
	owner, batchID, commandID, err := normalizePositionCashOutBatchCommand(req.GetOwnerAccountId(), req.GetId(), req.GetCommandId(), req.GetExpectedRevision())
	if err != nil {
		return nil, err
	}
	proofKind := strings.TrimSpace(req.GetProofKind())
	if proofKind != "GOOGLE" && proofKind != "PHANTOM" && proofKind != "DEVELOPMENT" {
		return nil, status.Error(codes.InvalidArgument, "position Cash Out Batch proof kind is invalid")
	}
	if len(req.GetSessionJtiDigest()) != 32 || req.GetAccessRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "position Cash Out Batch authorization binding is invalid")
	}
	batch, err := s.credentialStore.AuthorizePositionCashOutBatch(ctx, wormstore.AuthorizePositionCashOutBatchRequest{
		OwnerAccountID: owner, BatchID: batchID, CommandID: commandID,
		ExpectedRevision: req.GetExpectedRevision(), ProofKind: proofKind,
		SessionJTIDigest: append([]byte(nil), req.GetSessionJtiDigest()...), AccessRevision: req.GetAccessRevision(), Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, positionCashOutBatchRPCError(err)
	}
	s.wakePositionCashOutBatchWorker()
	return &apiclient.AuthorizePositionCashOutBatchResponse{Batch: positionCashOutBatchToProto(batch)}, nil
}

func (s *Service) CancelPositionCashOutBatch(ctx context.Context, req *apiclient.PositionCashOutBatchCommandRequest) (*apiclient.PositionCashOutBatchCommandResponse, error) {
	return s.controlPositionCashOutBatch(ctx, req, wormstore.PositionCashOutBatchCommandKindCancel)
}

func (s *Service) PausePositionCashOutBatch(ctx context.Context, req *apiclient.PositionCashOutBatchCommandRequest) (*apiclient.PositionCashOutBatchCommandResponse, error) {
	return s.controlPositionCashOutBatch(ctx, req, wormstore.PositionCashOutBatchCommandKindPause)
}

func (s *Service) ContinuePositionCashOutBatch(ctx context.Context, req *apiclient.PositionCashOutBatchCommandRequest) (*apiclient.PositionCashOutBatchCommandResponse, error) {
	return s.controlPositionCashOutBatch(ctx, req, wormstore.PositionCashOutBatchCommandKindContinue)
}

func (s *Service) TerminatePositionCashOutBatch(ctx context.Context, req *apiclient.PositionCashOutBatchCommandRequest) (*apiclient.PositionCashOutBatchCommandResponse, error) {
	return s.controlPositionCashOutBatch(ctx, req, wormstore.PositionCashOutBatchCommandKindTerminate)
}

func (s *Service) CheckPositionCashOutBatchStatus(ctx context.Context, req *apiclient.PositionCashOutBatchCommandRequest) (*apiclient.PositionCashOutBatchCommandResponse, error) {
	return s.controlPositionCashOutBatch(ctx, req, wormstore.PositionCashOutBatchCommandKindCheckStatus)
}

func (s *Service) controlPositionCashOutBatch(
	ctx context.Context,
	req *apiclient.PositionCashOutBatchCommandRequest,
	kind wormstore.PositionCashOutBatchCommandKind,
) (*apiclient.PositionCashOutBatchCommandResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "position Cash Out Batch command is required")
	}
	owner, batchID, commandID, err := normalizePositionCashOutBatchCommand(req.GetOwnerAccountId(), req.GetId(), req.GetCommandId(), req.GetExpectedRevision())
	if err != nil {
		return nil, err
	}
	batch, err := s.credentialStore.ControlPositionCashOutBatch(ctx, wormstore.PositionCashOutBatchCommandRequest{
		OwnerAccountID: owner, BatchID: batchID, CommandID: commandID,
		ExpectedRevision: req.GetExpectedRevision(), Kind: kind, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, positionCashOutBatchRPCError(err)
	}
	s.wakePositionCashOutBatchWorker()
	return &apiclient.PositionCashOutBatchCommandResponse{Batch: positionCashOutBatchToProto(batch)}, nil
}

func normalizePositionCashOutBatchCommand(ownerRaw, batchRaw, commandRaw string, revision int64) (string, string, string, error) {
	owner, err := normalizeMarketCombinationAccountID(ownerRaw)
	if err != nil {
		return "", "", "", err
	}
	batchID, err := normalizePositionCashOutUUID(batchRaw, "position Cash Out Batch ID")
	if err != nil {
		return "", "", "", err
	}
	commandID, err := normalizePositionCashOutUUID(commandRaw, "commandId")
	if err != nil {
		return "", "", "", err
	}
	if revision <= 0 {
		return "", "", "", status.Error(codes.InvalidArgument, "expectedRevision must be positive")
	}
	return owner, batchID, commandID, nil
}

func normalizePositionCashOutBatchPage(rawPage, rawPageSize int32) (int32, int32, error) {
	page := rawPage
	if page <= 0 {
		page = 1
	}
	pageSize := rawPageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > positionCashOutBatchMaximumPageSize {
		return 0, 0, status.Errorf(codes.InvalidArgument, "page_size must be at most %d", positionCashOutBatchMaximumPageSize)
	}
	return page, pageSize, nil
}

func positionCashOutBatchToProto(batch *wormstore.PositionCashOutBatch) *apiclient.PositionCashOutBatch {
	if batch == nil {
		return nil
	}
	wallets := make([]*apiclient.PositionCashOutBatchWallet, 0, len(batch.Wallets))
	for _, wallet := range batch.Wallets {
		wallets = append(wallets, &apiclient.PositionCashOutBatchWallet{
			Ordinal: wallet.Ordinal, WalletId: wallet.WalletID, Address: wallet.Address,
			Remark: wallet.Remark, AvatarKind: wallet.AvatarKind, AvatarPresetId: wallet.AvatarPresetID,
			AvatarUrl: wallet.AvatarURL, PositionCount: wallet.PositionCount, CompletedCount: wallet.CompletedCount,
		})
	}
	proofKind := ""
	authorizationSessionJTIDigest := []byte(nil)
	authorizationAccessRevision := int64(0)
	if batch.Authorization != nil {
		proofKind = batch.Authorization.ProofKind
		authorizationSessionJTIDigest = append([]byte(nil), batch.Authorization.SessionJTIDigest...)
		authorizationAccessRevision = batch.Authorization.AccessRevision
	}
	currentWalletOrdinal := int32(0)
	currentPositionOrdinal := int32(0)
	if batch.CurrentItem != nil {
		currentWalletOrdinal = batch.CurrentItem.WalletOrdinal
		currentPositionOrdinal = batch.CurrentItem.PositionOrdinal
	}
	return &apiclient.PositionCashOutBatch{
		Id: batch.ID, OwnerAccountId: batch.OwnerAccountID, State: string(batch.State), ReasonCode: batch.ReasonCode,
		Revision: batch.Revision, IntentSha256: append([]byte(nil), batch.IntentDigestSHA256...), ProofKind: proofKind,
		WalletCount: batch.WalletCount, PositionCount: batch.PositionCount, CompletedCount: batch.CompletedCount,
		NotExecutedCount: batch.NotExecutedCount, CurrentWalletOrdinal: currentWalletOrdinal,
		CurrentPositionOrdinal: currentPositionOrdinal, CurrentItem: positionCashOutBatchItemToProtoPointer(batch.CurrentItem),
		Wallets: wallets, AuthorizationExpiresAt: executionPlanUnix(batch.AuthorizationExpiresAt),
		ExecutionStartedAt: executionPlanUnix(batch.ExecutionStartedAt), CompletedAt: executionPlanUnix(batch.CompletedAt),
		CreatedAt: executionPlanUnix(batch.CreatedAt), UpdatedAt: executionPlanUnix(batch.UpdatedAt),
		AllowedActions:                positionCashOutBatchAllowedActions(batch),
		AuthorizationSessionJtiDigest: authorizationSessionJTIDigest,
		AuthorizationAccessRevision:   authorizationAccessRevision,
	}
}

func positionCashOutBatchItemToProtoPointer(item *wormstore.PositionCashOutBatchItem) *apiclient.PositionCashOutBatchItem {
	if item == nil {
		return nil
	}
	return positionCashOutBatchItemToProto(*item)
}

func positionCashOutBatchItemToProto(item wormstore.PositionCashOutBatchItem) *apiclient.PositionCashOutBatchItem {
	return &apiclient.PositionCashOutBatchItem{
		Id: item.ID, Ordinal: item.Ordinal, WalletOrdinal: item.WalletOrdinal, PositionOrdinal: item.PositionOrdinal,
		WalletId: item.WalletID, WalletAddress: item.WalletAddress, WalletRemark: item.WalletRemark,
		PositionPubkey: item.PositionPubkey, PositionRequestPubkey: item.PositionRequestPubkey,
		MarketConditionId: item.MarketConditionID, MarketTitle: item.MarketTitle, IsYes: item.IsYes,
		Shares: item.Shares, PositionCreatedAt: executionPlanUnix(item.PositionCreatedAt), State: string(item.State),
		ReasonCode: item.ReasonCode, ChildOperationId: item.ChildCashOutID,
		Baseline: positionCashOutBatchBalanceToProto(item.Baseline), Observed: positionCashOutBatchBalanceToProto(item.Observed),
		DeltaAtomicAmount: item.DeltaUSDCAtomicAmount, BalanceDeadlineAt: executionPlanUnix(item.BalanceDeadlineAt),
		CompletedAt: executionPlanUnix(item.CompletedAt), UpdatedAt: executionPlanUnix(item.UpdatedAt),
	}
}

func positionCashOutBatchBalanceToProto(value *wormstore.PositionCashOutBatchBalanceEvidence) *apiclient.PositionCashOutBatchBalanceEvidence {
	if value == nil {
		return nil
	}
	return &apiclient.PositionCashOutBatchBalanceEvidence{Mint: value.Mint, Decimals: value.Decimals, AtomicAmount: value.AtomicAmount, ObservedSlot: value.ObservedSlot}
}

func positionCashOutBatchAllowedActions(batch *wormstore.PositionCashOutBatch) []string {
	if batch == nil || batch.ClaimID != "" {
		return []string{}
	}
	switch batch.State {
	case wormstore.PositionCashOutBatchStateBuilding:
		return []string{"CANCEL"}
	case wormstore.PositionCashOutBatchStateAwaitingAuthorization:
		return []string{"AUTHORIZE_BATCH", "CANCEL"}
	case wormstore.PositionCashOutBatchStateQueued, wormstore.PositionCashOutBatchStateRunning:
		return []string{"PAUSE", "TERMINATE"}
	case wormstore.PositionCashOutBatchStatePauseRequested:
		return []string{"TERMINATE"}
	case wormstore.PositionCashOutBatchStatePaused:
		if batch.CurrentItem != nil {
			return []string{"TERMINATE", "CHECK_STATUS"}
		}
		return []string{"CONTINUE", "TERMINATE"}
	case wormstore.PositionCashOutBatchStateReconciliationRequired:
		return []string{"TERMINATE", "CHECK_STATUS"}
	case wormstore.PositionCashOutBatchStateTerminateRequested:
		if batch.CurrentItem != nil {
			return []string{"CHECK_STATUS"}
		}
		return []string{}
	default:
		return []string{}
	}
}

func positionCashOutBatchRPCError(err error) error {
	switch {
	case errors.Is(err, wormstore.ErrPositionCashOutBatchNotFound):
		return status.Error(codes.NotFound, "position Cash Out Batch not found")
	case errors.Is(err, wormstore.ErrPositionCashOutBatchRevision), errors.Is(err, wormstore.ErrPositionCashOutBatchCommandConflict):
		return status.Error(codes.Aborted, "position Cash Out Batch revision or command conflict")
	case errors.Is(err, wormstore.ErrPositionCashOutBatchOwnerActive):
		return status.Error(codes.FailedPrecondition, "POSITION_CASH_OUT_BATCH_ACTIVE")
	case errors.Is(err, wormstore.ErrPositionCashOutBatchExecutionActive):
		return status.Error(codes.FailedPrecondition, "WALLET_EXECUTION_ACTIVE")
	case errors.Is(err, wormstore.ErrPositionCashOutBatchCashOutActive):
		return status.Error(codes.FailedPrecondition, "WALLET_CASH_OUT_ACTIVE")
	case errors.Is(err, wormstore.ErrPositionCashOutBatchWalletActive):
		return status.Error(codes.FailedPrecondition, "WALLET_BATCH_CASH_OUT_ACTIVE")
	case errors.Is(err, wormstore.ErrPositionCashOutBatchClaim):
		return status.Error(codes.Aborted, "position Cash Out Batch worker claim changed")
	case errors.Is(err, wormstore.ErrExpired):
		return status.Error(codes.FailedPrecondition, "AUTHORIZATION_EXPIRED")
	case errors.Is(err, wormstore.ErrInvalidPositionCashOutBatch):
		return status.Error(codes.FailedPrecondition, "position Cash Out Batch conflicts with current state")
	case errors.Is(err, wormstore.ErrTransactionOutcomeUnknown):
		return status.Error(codes.Unavailable, "position Cash Out Batch store commit outcome is unknown")
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		return status.Error(codes.Internal, "position Cash Out Batch store operation failed")
	}
}

func (s *Service) wakePositionCashOutBatchWorker() {
	select {
	case s.positionCashOutBatchWake <- struct{}{}:
	default:
	}
}
