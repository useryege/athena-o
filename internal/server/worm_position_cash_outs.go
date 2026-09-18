package server

import (
	"context"
	"math/big"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/internal/walletsecret"
	wormtradingapiclient "github.com/useryege/athena/internal/wormtrading/apiclient"
)

const (
	wormPositionCashOutCollectionPath = "/api/v1/worm-trading/position-cash-outs"
	wormPositionCashOutResourcePath   = wormPositionCashOutCollectionPath + "/{cashOutId}"
	wormPositionCashOutMutationPath   = wormPositionCashOutCollectionPath + "/{cashOutMutationResource}"
)

type wormPositionCashOutCreateInput struct {
	CommandID      string `json:"commandId"`
	WalletID       int64  `json:"walletId"`
	PositionPubkey string `json:"positionPubkey"`
}

type wormPositionCashOutCommandInput struct {
	CommandID        string `json:"commandId"`
	ExpectedRevision int64  `json:"expectedRevision"`
}

type wormPositionCashOutResponse struct {
	ID                     string   `json:"id"`
	WalletID               int64    `json:"walletId"`
	WalletAddress          string   `json:"walletAddress"`
	PositionPubkey         string   `json:"positionPubkey"`
	PositionRequestPubkey  string   `json:"positionRequestPubkey"`
	MarketConditionID      string   `json:"marketConditionId"`
	IsYes                  bool     `json:"isYes"`
	PositionCreatedAt      int64    `json:"positionCreatedAt"`
	TotalShares            string   `json:"totalShares"`
	State                  string   `json:"state"`
	Stage                  string   `json:"stage"`
	ReasonCode             string   `json:"reasonCode"`
	Revision               int64    `json:"revision"`
	ProofKind              string   `json:"proofKind"`
	ProviderState          string   `json:"providerState"`
	ObservedClosed         bool     `json:"observedClosed"`
	ObservedLiquidated     bool     `json:"observedLiquidated"`
	CompletionSource       string   `json:"completionSource"`
	AuthorizationExpiresAt int64    `json:"authorizationExpiresAt"`
	DispatchExpiresAt      int64    `json:"dispatchExpiresAt"`
	RequestedAt            int64    `json:"requestedAt"`
	AuthorizedAt           int64    `json:"authorizedAt"`
	DispatchedAt           int64    `json:"dispatchedAt"`
	CompletedAt            int64    `json:"completedAt"`
	UpdatedAt              int64    `json:"updatedAt"`
	AllowedActions         []string `json:"allowedActions"`
}

func registerWormPositionCashOutHandlers(mux *http.ServeMux, server *AthenaServer) {
	if mux == nil || server == nil {
		return
	}
	mux.Handle("POST "+wormPositionCashOutCollectionPath, traceHTTP(http.HandlerFunc(server.createWormPositionCashOut)))
	mux.Handle("GET "+wormPositionCashOutResourcePath, traceHTTP(http.HandlerFunc(server.getWormPositionCashOut)))
	mux.Handle("POST "+wormPositionCashOutMutationPath, traceHTTP(http.HandlerFunc(server.mutateWormPositionCashOut)))
}

func (server *AthenaServer) createWormPositionCashOut(w http.ResponseWriter, request *http.Request) {
	ctx, credential, ok := server.authenticateWormExecutionMutation(w, request)
	if !ok {
		return
	}
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	var input wormPositionCashOutCreateInput
	if err := decodeWormExecutionJSON(w, request, &input); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	commandID, err := canonicalWormExecutionID(input.CommandID, "commandId")
	if err != nil || input.WalletID <= 0 {
		walletsecret.WriteError(w, status.Error(codes.InvalidArgument, "position Cash Out command or wallet binding is invalid"))
		return
	}
	positionPubkey, err := canonicalWormConditionID(input.PositionPubkey, "position pubkey")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	walletAddress, err := server.resolveOwnedPositionCashOutWallet(ctx, credential.AccountID, input.WalletID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.CreatePositionCashOut(ctx, &wormtradingapiclient.CreatePositionCashOutRequest{
		OwnerAccountId: credential.AccountID,
		CommandId:      commandID,
		WalletId:       input.WalletID,
		WalletAddress:  walletAddress,
		PositionPubkey: positionPubkey,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormPositionCashOutError(err))
		return
	}
	response, err := projectWormPositionCashOut(result.GetCashOut(), credential.AccountID, "")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	observeWormHTTPMutation(request.Context(), "cash_out", response.ID, "WORM_CASH_OUT_CREATE", response.State, true, map[string]string{
		"walletId":  strconv.FormatInt(response.WalletID, 10),
		"cashOutId": response.ID,
		"state":     response.State,
		"stage":     response.Stage,
	})
	w.Header().Set("Location", wormPositionCashOutCollectionPath+"/"+response.ID)
	writeWormCombinationJSON(w, http.StatusAccepted, response)
}

func (server *AthenaServer) getWormPositionCashOut(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelRead)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	cashOutID, err := canonicalWormExecutionID(request.PathValue("cashOutId"), "position Cash Out ID")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.GetPositionCashOut(ctx, &wormtradingapiclient.GetPositionCashOutRequest{
		OwnerAccountId: credential.AccountID,
		Id:             cashOutID,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormPositionCashOutError(err))
		return
	}
	response, err := projectWormPositionCashOut(result.GetCashOut(), credential.AccountID, cashOutID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) mutateWormPositionCashOut(w http.ResponseWriter, request *http.Request) {
	resource := request.PathValue("cashOutMutationResource")
	cashOutID, action, ok := strings.Cut(resource, ":")
	if !ok || cashOutID == "" || action != "reconcile" || strings.Contains(action, ":") {
		walletsecret.WriteError(w, status.Error(codes.NotFound, "position Cash Out action not found"))
		return
	}
	request.SetPathValue("cashOutId", cashOutID)
	server.reconcileWormPositionCashOut(w, request)
}

func (server *AthenaServer) reconcileWormPositionCashOut(w http.ResponseWriter, request *http.Request) {
	ctx, credential, ok := server.authenticateWormExecutionMutation(w, request)
	if !ok {
		return
	}
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	cashOutID, err := canonicalWormExecutionID(request.PathValue("cashOutId"), "position Cash Out ID")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	var input wormPositionCashOutCommandInput
	if err := decodeWormExecutionJSON(w, request, &input); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	commandID, err := canonicalWormExecutionID(input.CommandID, "commandId")
	if err != nil || validateWormExecutionExpectedRevision(input.ExpectedRevision) != nil {
		walletsecret.WriteError(w, status.Error(codes.InvalidArgument, "position Cash Out reconciliation binding is invalid"))
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.ReconcilePositionCashOut(ctx, &wormtradingapiclient.ReconcilePositionCashOutRequest{
		OwnerAccountId:   credential.AccountID,
		Id:               cashOutID,
		CommandId:        commandID,
		ExpectedRevision: input.ExpectedRevision,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormPositionCashOutError(err))
		return
	}
	response, err := projectWormPositionCashOut(result.GetCashOut(), credential.AccountID, cashOutID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	observeWormHTTPMutation(request.Context(), "cash_out", response.ID, "WORM_CASH_OUT_RECONCILE", response.State, false, map[string]string{
		"expectedRevision":  strconv.FormatInt(input.ExpectedRevision, 10),
		"confirmedRevision": strconv.FormatInt(response.Revision, 10),
		"state":             response.State,
		"stage":             response.Stage,
	})
	writeWormCombinationJSON(w, http.StatusAccepted, response)
}

func (server *AthenaServer) resolveOwnedPositionCashOutWallet(
	ctx context.Context,
	ownerAccountID string,
	walletID int64,
) (string, error) {
	if server.WalletClientset == nil || server.WalletClientset.Wallet() == nil {
		return "", status.Error(codes.Unavailable, "Wallet is unavailable")
	}
	result, err := server.WalletClientset.Wallet().GetWallet(ctx, &walletapiclient.GetWalletRequest{
		Id: walletID, RequesterAccountId: ownerAccountID,
	})
	if err != nil {
		return "", sanitizeWormConnectionInventoryDependencyError(err, "Wallet")
	}
	wallet := result.GetItem()
	if wallet == nil || wallet.ID != walletID || wallet.WalletType != "SOLANA" {
		return "", status.Error(codes.FailedPrecondition, "position Cash Out requires an owned Solana Wallet")
	}
	address, err := canonicalWormConditionID(wallet.Address, "wallet address")
	if err != nil || address != wallet.Address {
		return "", status.Error(codes.Internal, "Wallet returned an invalid Cash Out wallet")
	}
	return address, nil
}

func projectWormPositionCashOut(
	cashOut *wormtradingapiclient.PositionCashOut,
	expectedOwnerAccountID string,
	expectedCashOutID string,
) (wormPositionCashOutResponse, error) {
	if cashOut == nil {
		return wormPositionCashOutResponse{}, status.Error(codes.Internal, "Worm Trading returned an empty position Cash Out")
	}
	ownerAccountID, err := accountcredentials.CanonicalAccountID(cashOut.GetOwnerAccountId())
	if err != nil || ownerAccountID != cashOut.GetOwnerAccountId() || ownerAccountID != expectedOwnerAccountID {
		return wormPositionCashOutResponse{}, status.Error(codes.Internal, "Worm Trading returned a mismatched Cash Out owner")
	}
	id, err := canonicalWormExecutionID(cashOut.GetId(), "position Cash Out ID")
	if err != nil || (expectedCashOutID != "" && id != expectedCashOutID) {
		return wormPositionCashOutResponse{}, status.Error(codes.Internal, "Worm Trading returned a mismatched position Cash Out")
	}
	walletAddress, walletErr := canonicalWormConditionID(cashOut.GetWalletAddress(), "wallet address")
	positionPubkey, positionErr := canonicalWormConditionID(cashOut.GetPositionPubkey(), "position pubkey")
	marketID, marketErr := canonicalWormConditionID(cashOut.GetMarketConditionId(), "market condition ID")
	positionRequestPubkey := cashOut.GetPositionRequestPubkey()
	var positionRequestErr error
	if positionRequestPubkey != "" {
		_, positionRequestErr = canonicalWormConditionID(positionRequestPubkey, "position request pubkey")
	}
	state := cashOut.GetState()
	validStates := []string{
		"AWAITING_AUTHORIZATION", "QUEUED", "PREFLIGHTING", "CLOSING", "AWAITING_COMPLETION",
		"COMPLETED", "FAILED", "RECONCILIATION_REQUIRED", "EXPIRED",
	}
	expectedStage := map[string]string{
		"AWAITING_AUTHORIZATION": "AUTHORIZATION",
		"QUEUED":                 "PREFLIGHT", "PREFLIGHTING": "PREFLIGHT", "CLOSING": "CLOSE",
		"AWAITING_COMPLETION": "OBSERVATION", "RECONCILIATION_REQUIRED": "OBSERVATION",
		"COMPLETED": "COMPLETE", "FAILED": "COMPLETE", "EXPIRED": "COMPLETE",
	}[state]
	reasonCode := cashOut.GetReasonCode()
	providerState := cashOut.GetProviderState()
	proofKind := cashOut.GetProofKind()
	completionSource := cashOut.GetCompletionSource()
	sharesValue, sharesOK := new(big.Rat).SetString(cashOut.GetTotalShares())
	if cashOut.GetWalletId() <= 0 || walletErr != nil || positionErr != nil || marketErr != nil ||
		walletAddress != cashOut.GetWalletAddress() || positionPubkey != cashOut.GetPositionPubkey() || marketID != cashOut.GetMarketConditionId() ||
		positionRequestErr != nil || cashOut.GetPositionCreatedAt() <= 0 ||
		!validWormExecutionDecimal(cashOut.GetTotalShares(), false) || !sharesOK || sharesValue.Sign() <= 0 ||
		!slices.Contains(validStates, state) || expectedStage == "" || cashOut.GetStage() != expectedStage ||
		cashOut.GetRevision() <= 0 || cashOut.GetRequestedAt() <= 0 || cashOut.GetUpdatedAt() <= 0 ||
		(reasonCode != "" && !validWormExecutionReasonCode(reasonCode)) ||
		providerState != strings.TrimSpace(providerState) || len(providerState) > 100 ||
		(proofKind != "" && proofKind != "GOOGLE" && proofKind != "PHANTOM" && proofKind != "DEVELOPMENT") ||
		(completionSource != "" && completionSource != "HMAC_POSITION_OBSERVED") ||
		(state == "COMPLETED" && (!cashOut.GetObservedClosed() || cashOut.GetObservedLiquidated())) {
		return wormPositionCashOutResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid position Cash Out")
	}
	allowedActions := append([]string{}, cashOut.GetAllowedActions()...)
	validActions := len(allowedActions) == 0
	if state == "AWAITING_AUTHORIZATION" {
		validActions = slices.Equal(allowedActions, []string{"AUTHORIZE_CASH_OUT"})
	} else if state == "RECONCILIATION_REQUIRED" {
		validActions = len(allowedActions) == 0 || slices.Equal(allowedActions, []string{"CHECK_STATUS"})
	}
	if !validActions {
		return wormPositionCashOutResponse{}, status.Error(codes.Internal, "Worm Trading returned invalid Cash Out actions")
	}
	return wormPositionCashOutResponse{
		ID: id, WalletID: cashOut.GetWalletId(), WalletAddress: walletAddress,
		PositionPubkey: positionPubkey, PositionRequestPubkey: positionRequestPubkey,
		MarketConditionID: marketID, IsYes: cashOut.GetIsYes(), PositionCreatedAt: cashOut.GetPositionCreatedAt(),
		TotalShares: cashOut.GetTotalShares(), State: state, Stage: expectedStage,
		ReasonCode: reasonCode, Revision: cashOut.GetRevision(), ProofKind: proofKind,
		ProviderState: providerState, ObservedClosed: cashOut.GetObservedClosed(),
		ObservedLiquidated: cashOut.GetObservedLiquidated(), CompletionSource: completionSource,
		AuthorizationExpiresAt: cashOut.GetAuthorizationExpiresAt(), DispatchExpiresAt: cashOut.GetDispatchExpiresAt(),
		RequestedAt: cashOut.GetRequestedAt(), AuthorizedAt: cashOut.GetAuthorizedAt(),
		DispatchedAt: cashOut.GetDispatchedAt(), CompletedAt: cashOut.GetCompletedAt(),
		UpdatedAt: cashOut.GetUpdatedAt(), AllowedActions: allowedActions,
	}, nil
}

func sanitizeWormPositionCashOutError(err error) error {
	return sanitizeWormExecutionStoreError(err)
}
