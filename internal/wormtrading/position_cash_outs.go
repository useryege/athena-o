package wormtrading

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	utilworm "github.com/useryege/athena/util/worm"
)

const (
	positionCashOutAuthorizationTTL = 5 * time.Minute
	positionCashOutExecutionTTL     = 5 * time.Minute
	positionCashOutCompletionSource = "HMAC_POSITION_OBSERVED"
)

func (s *Service) CreatePositionCashOut(
	ctx context.Context,
	req *apiclient.CreatePositionCashOutRequest,
) (*apiclient.CreatePositionCashOutResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "position Cash Out request is required")
	}
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	commandID, err := normalizePositionCashOutUUID(req.GetCommandId(), "commandId")
	if err != nil {
		return nil, err
	}
	walletID, walletAddress, err := normalizeWormWalletReference(req.GetWalletId(), req.GetWalletAddress())
	if err != nil {
		return nil, err
	}
	positionPubkey, err := normalizePositionCashOutPubkey(req.GetPositionPubkey(), "position pubkey")
	if err != nil {
		return nil, err
	}
	replayed, found, err := s.credentialStore.GetPositionCashOutCreation(ctx, wormstore.GetPositionCashOutCreationRequest{
		OwnerAccountID: ownerAccountID,
		CommandID:      commandID,
		WalletID:       walletID,
		WalletAddress:  walletAddress,
		PositionPubkey: positionPubkey,
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, positionCashOutRPCError(err)
	}
	if found {
		return &apiclient.CreatePositionCashOutResponse{CashOut: positionCashOutToProto(replayed)}, nil
	}
	if err := s.requireCredentialCapability(); err != nil {
		return nil, err
	}
	selection, err := s.credentialStore.GetWalletSelection(ctx, ownerAccountID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, walletSelectionRPCError(err)
	}
	if !walletSelectionContains(selection, walletID, walletAddress) {
		return nil, status.Error(codes.FailedPrecondition, "WALLET_NOT_SELECTED")
	}

	target, credentialVersion, providerState, err := s.readPositionCashOutTarget(
		ctx,
		walletID,
		walletAddress,
		positionPubkey,
	)
	if err != nil {
		return nil, err
	}
	positionRequestPubkey := ""
	if target.PositionRequestPubkey != nil {
		positionRequestPubkey = *target.PositionRequestPubkey
	}
	now := timeNowUTC()
	cashOut, err := s.credentialStore.CreatePositionCashOut(ctx, wormstore.CreatePositionCashOutRequest{
		OwnerAccountID:         ownerAccountID,
		CommandID:              commandID,
		WalletID:               walletID,
		WalletAddress:          walletAddress,
		CredentialVersion:      credentialVersion,
		PositionPubkey:         target.PositionPubkey,
		MarketConditionID:      target.MarketConditionID,
		IsYes:                  target.IsYes,
		PositionCreatedAt:      time.Unix(target.CreatedAt, 0).UTC(),
		PositionRequestPubkey:  positionRequestPubkey,
		Shares:                 target.TotalShares,
		ProviderState:          providerState,
		ProviderIsClosed:       target.IsClosed,
		ProviderIsLiquidated:   target.IsLiquidated,
		AuthorizationExpiresAt: now.Add(positionCashOutAuthorizationTTL),
		Now:                    now,
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, positionCashOutRPCError(err)
	}
	return &apiclient.CreatePositionCashOutResponse{CashOut: positionCashOutToProto(cashOut)}, nil
}

func (s *Service) GetPositionCashOut(
	ctx context.Context,
	req *apiclient.GetPositionCashOutRequest,
) (*apiclient.GetPositionCashOutResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "position Cash Out request is required")
	}
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	cashOutID, err := normalizePositionCashOutUUID(req.GetId(), "position Cash Out ID")
	if err != nil {
		return nil, err
	}
	cashOut, err := s.credentialStore.GetPositionCashOut(ctx, ownerAccountID, cashOutID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, positionCashOutRPCError(err)
	}
	return &apiclient.GetPositionCashOutResponse{CashOut: positionCashOutToProto(cashOut)}, nil
}

func (s *Service) AuthorizePositionCashOut(
	ctx context.Context,
	req *apiclient.AuthorizePositionCashOutRequest,
) (*apiclient.AuthorizePositionCashOutResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "position Cash Out authorization is required")
	}
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	cashOutID, err := normalizePositionCashOutUUID(req.GetId(), "position Cash Out ID")
	if err != nil {
		return nil, err
	}
	commandID, err := normalizePositionCashOutUUID(req.GetCommandId(), "commandId")
	if err != nil {
		return nil, err
	}
	proofKind := strings.TrimSpace(req.GetProofKind())
	if proofKind != "GOOGLE" && proofKind != "PHANTOM" && proofKind != "DEVELOPMENT" {
		return nil, status.Error(codes.InvalidArgument, "position Cash Out proof kind is invalid")
	}
	if req.GetExpectedRevision() <= 0 || len(req.GetSessionJtiDigest()) != 32 || req.GetAccessRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "position Cash Out authorization binding is invalid")
	}
	now := timeNowUTC()
	cashOut, err := s.credentialStore.AuthorizePositionCashOut(ctx, wormstore.AuthorizePositionCashOutRequest{
		PositionCashOutCommandRequest: wormstore.PositionCashOutCommandRequest{
			OwnerAccountID:   ownerAccountID,
			CashOutID:        cashOutID,
			CommandID:        commandID,
			ExpectedRevision: req.GetExpectedRevision(),
			Now:              now,
		},
		ProofKind:          proofKind,
		SessionJTIDigest:   append([]byte(nil), req.GetSessionJtiDigest()...),
		AccessRevision:     req.GetAccessRevision(),
		ExecutionExpiresAt: now.Add(positionCashOutExecutionTTL),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, positionCashOutRPCError(err)
	}
	s.wakePositionCashOutWorker()
	return &apiclient.AuthorizePositionCashOutResponse{CashOut: positionCashOutToProto(cashOut)}, nil
}

func (s *Service) ReconcilePositionCashOut(
	ctx context.Context,
	req *apiclient.ReconcilePositionCashOutRequest,
) (*apiclient.ReconcilePositionCashOutResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "position Cash Out reconciliation is required")
	}
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	cashOutID, err := normalizePositionCashOutUUID(req.GetId(), "position Cash Out ID")
	if err != nil {
		return nil, err
	}
	commandID, err := normalizePositionCashOutUUID(req.GetCommandId(), "commandId")
	if err != nil {
		return nil, err
	}
	if req.GetExpectedRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expectedRevision must be positive")
	}
	cashOut, err := s.credentialStore.RequestPositionCashOutReconciliation(ctx, wormstore.PositionCashOutCommandRequest{
		OwnerAccountID:   ownerAccountID,
		CashOutID:        cashOutID,
		CommandID:        commandID,
		ExpectedRevision: req.GetExpectedRevision(),
		Now:              timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, positionCashOutRPCError(err)
	}
	s.wakePositionCashOutWorker()
	return &apiclient.ReconcilePositionCashOutResponse{CashOut: positionCashOutToProto(cashOut)}, nil
}

func (s *Service) readPositionCashOutTarget(
	ctx context.Context,
	walletID int64,
	walletAddress string,
	positionPubkey string,
) (utilworm.MarginPositionCashOutTarget, int64, string, error) {
	snapshot, err := s.credentialStore.GetWalletConnectionSnapshot(ctx, walletID, walletAddress)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return utilworm.MarginPositionCashOutTarget{}, 0, "", connectionStoreRPCError("load Worm Cash Out wallet", err)
	}
	credential := snapshot.ActiveCredential
	if snapshot.State != wormstore.ConnectionStateConnected || credential == nil ||
		credential.State != wormstore.CredentialStateActive || credential.Version <= 0 {
		return utilworm.MarginPositionCashOutTarget{}, 0, "", status.Error(codes.FailedPrecondition, "Worm Cash Out wallet is not connected")
	}
	apiKey, apiSecret, err := s.credentialCipher.decrypt(
		credential.WalletID,
		credential.APIKeyCiphertext,
		credential.APISecretCiphertext,
	)
	if err != nil {
		return utilworm.MarginPositionCashOutTarget{}, 0, "", status.Error(codes.FailedPrecondition, "Worm Cash Out credential is unavailable")
	}
	client, err := s.wormClientFactory.NewAuthenticatedClient(apiKey, apiSecret)
	if err != nil {
		return utilworm.MarginPositionCashOutTarget{}, 0, "", status.Error(codes.FailedPrecondition, "Worm Cash Out credential is unavailable")
	}
	attemptCtx, cancel := context.WithTimeout(ctx, s.wormAPIAttemptTimeout)
	position, getErr := client.GetMarginPosition(attemptCtx, positionPubkey)
	cancel()
	s.wormCapabilities.recordWormResult(getErr)
	if getErr != nil {
		if isWormAuthenticationError(getErr) {
			s.markWalletReconnectRequired(ctx, walletID, walletAddress, credential.ID)
		}
		if isWormNotFoundError(getErr) {
			return utilworm.MarginPositionCashOutTarget{}, 0, "", status.Error(codes.NotFound, "Worm position not found")
		}
		return utilworm.MarginPositionCashOutTarget{}, 0, "", wormRPCError("read Worm Cash Out position", getErr)
	}
	target, err := utilworm.InspectMarginPositionCashOutTarget(position)
	if err != nil || target.PositionPubkey != positionPubkey {
		return target, 0, "", status.Error(codes.Unavailable, "Worm returned an invalid Cash Out position")
	}
	providerState := "open"
	if target.IsLiquidated {
		providerState = "liquidated"
	} else if target.IsClosed {
		providerState = "closed"
	}
	return target, credential.Version, providerState, nil
}

func normalizePositionCashOutUUID(raw string, label string) (string, error) {
	value := strings.TrimSpace(raw)
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil || parsed.String() != value {
		return "", status.Errorf(codes.InvalidArgument, "%s must be a canonical non-zero UUID", label)
	}
	return value, nil
}

func normalizePositionCashOutPubkey(raw string, label string) (string, error) {
	value := strings.TrimSpace(raw)
	publicKey, err := solana.PublicKeyFromBase58(value)
	if err != nil || publicKey.IsZero() || publicKey.String() != value {
		return "", status.Errorf(codes.InvalidArgument, "%s must be a canonical Solana public key", label)
	}
	return value, nil
}

func positionCashOutToProto(cashOut *wormstore.PositionCashOut) *apiclient.PositionCashOut {
	if cashOut == nil {
		return nil
	}
	proofKind := ""
	if cashOut.Authorization != nil {
		proofKind = cashOut.Authorization.ProofKind
	}
	completionSource := ""
	if cashOut.State == wormstore.PositionCashOutStateCompleted && cashOut.ProviderIsClosed {
		completionSource = positionCashOutCompletionSource
	}
	return &apiclient.PositionCashOut{
		Id:                     cashOut.ID,
		OwnerAccountId:         cashOut.OwnerAccountID,
		WalletId:               cashOut.WalletID,
		WalletAddress:          cashOut.WalletAddress,
		PositionPubkey:         cashOut.PositionPubkey,
		PositionRequestPubkey:  cashOut.PositionRequestPubkey,
		MarketConditionId:      cashOut.MarketConditionID,
		IsYes:                  cashOut.IsYes,
		PositionCreatedAt:      executionPlanUnix(cashOut.PositionCreatedAt),
		State:                  string(cashOut.State),
		Stage:                  positionCashOutStage(cashOut.State),
		ReasonCode:             cashOut.ReasonCode,
		Revision:               cashOut.Revision,
		IntentSha256:           append([]byte(nil), cashOut.IntentDigestSHA256...),
		ProofKind:              proofKind,
		ProviderState:          cashOut.ProviderState,
		ObservedClosed:         cashOut.ProviderIsClosed,
		ObservedLiquidated:     cashOut.ProviderIsLiquidated,
		CompletionSource:       completionSource,
		AuthorizationExpiresAt: executionPlanUnix(cashOut.AuthorizationExpiresAt),
		DispatchExpiresAt:      executionPlanUnix(cashOut.ExecutionExpiresAt),
		RequestedAt:            executionPlanUnix(cashOut.CreatedAt),
		AuthorizedAt:           executionPlanUnix(cashOut.AuthorizedAt),
		DispatchedAt:           positionCashOutDispatchedAt(cashOut.Attempt),
		CompletedAt:            executionPlanUnix(cashOut.CompletedAt),
		UpdatedAt:              executionPlanUnix(cashOut.UpdatedAt),
		AllowedActions:         positionCashOutAllowedActions(cashOut),
		TotalShares:            cashOut.Shares,
		BatchId:                cashOut.BatchID,
		BatchItemId:            cashOut.BatchItemID,
	}
}

func positionCashOutDispatchedAt(attempt *wormstore.PositionCashOutAttempt) int64 {
	if attempt == nil {
		return 0
	}
	return executionPlanUnix(attempt.DispatchedAt)
}

func positionCashOutStage(state wormstore.PositionCashOutState) string {
	switch state {
	case wormstore.PositionCashOutStateAwaitingAuthorization:
		return "AUTHORIZATION"
	case wormstore.PositionCashOutStateQueued, wormstore.PositionCashOutStatePreflighting:
		return "PREFLIGHT"
	case wormstore.PositionCashOutStateClosing:
		return "CLOSE"
	case wormstore.PositionCashOutStateAwaitingCompletion, wormstore.PositionCashOutStateReconciliationRequired:
		return "OBSERVATION"
	default:
		return "COMPLETE"
	}
}

func positionCashOutAllowedActions(cashOut *wormstore.PositionCashOut) []string {
	if cashOut == nil {
		return []string{}
	}
	switch cashOut.State {
	case wormstore.PositionCashOutStateAwaitingAuthorization:
		return []string{"AUTHORIZE_CASH_OUT"}
	case wormstore.PositionCashOutStateReconciliationRequired:
		if cashOut.ClaimID == "" && cashOut.ReconcileRequestedAt.IsZero() {
			return []string{"CHECK_STATUS"}
		}
	default:
	}
	return []string{}
}

func positionCashOutRPCError(err error) error {
	switch {
	case errors.Is(err, wormstore.ErrPositionCashOutNotFound):
		return status.Error(codes.NotFound, "position Cash Out not found")
	case errors.Is(err, wormstore.ErrPositionCashOutRevision), errors.Is(err, wormstore.ErrPositionCashOutCommandConflict):
		return status.Error(codes.Aborted, "position Cash Out revision or command conflict")
	case errors.Is(err, wormstore.ErrPositionCashOutExecutionActive):
		return status.Error(codes.FailedPrecondition, "WALLET_EXECUTION_ACTIVE")
	case errors.Is(err, wormstore.ErrPositionCashOutWalletActive):
		return status.Error(codes.FailedPrecondition, "WALLET_CASH_OUT_ACTIVE")
	case errors.Is(err, wormstore.ErrPositionCashOutConnectionChanged):
		return status.Error(codes.FailedPrecondition, "WORM_CONNECTION_CHANGED")
	case errors.Is(err, wormstore.ErrWalletNotSelected):
		return status.Error(codes.FailedPrecondition, "WALLET_NOT_SELECTED")
	case errors.Is(err, wormstore.ErrExpired):
		return status.Error(codes.FailedPrecondition, "AUTHORIZATION_EXPIRED")
	case errors.Is(err, wormstore.ErrPositionCashOutClaim):
		return status.Error(codes.Aborted, "position Cash Out worker claim changed")
	case errors.Is(err, wormstore.ErrInvalidPositionCashOut):
		return status.Error(codes.FailedPrecondition, "position Cash Out conflicts with current state")
	case errors.Is(err, wormstore.ErrTransactionOutcomeUnknown):
		return status.Error(codes.Unavailable, "position Cash Out store commit outcome is unknown")
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		return status.Error(codes.Internal, "position Cash Out store operation failed")
	}
}

func (s *Service) wakePositionCashOutWorker() {
	select {
	case s.positionCashOutWake <- struct{}{}:
	default:
	}
}
