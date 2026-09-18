package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
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
	wormPositionCashOutBatchCollectionPath = "/api/v1/worm-trading/position-cash-out-batches"
	wormPositionCashOutBatchActivePath     = wormPositionCashOutBatchCollectionPath + "/active"
	wormPositionCashOutBatchResourcePath   = wormPositionCashOutBatchCollectionPath + "/{batchId}"
	wormPositionCashOutBatchItemsPath      = wormPositionCashOutBatchCollectionPath + "/{batchId}/items"
	wormPositionCashOutBatchMutationPath   = wormPositionCashOutBatchCollectionPath + "/{batchMutationResource}"

	wormPositionCashOutBatchMaxWallets = 20
	wormPositionCashOutBatchPageSize   = int32(100)
)

type wormPositionCashOutBatchCreateInput struct {
	CommandID string  `json:"commandId"`
	WalletIDs []int64 `json:"walletIds"`
}

type wormPositionCashOutBatchCommandInput struct {
	CommandID        string `json:"commandId"`
	ExpectedRevision int64  `json:"expectedRevision"`
}

type wormPositionCashOutBatchWalletResponse struct {
	Ordinal        int32  `json:"ordinal"`
	WalletID       int64  `json:"walletId"`
	Address        string `json:"address"`
	Remark         string `json:"remark"`
	AvatarKind     string `json:"avatarKind"`
	AvatarPresetID string `json:"avatarPresetId"`
	AvatarURL      string `json:"avatarUrl"`
	PositionCount  int64  `json:"positionCount"`
	CompletedCount int64  `json:"completedCount"`
}

type wormPositionCashOutBatchBalanceResponse struct {
	Mint         string `json:"mint"`
	Decimals     int32  `json:"decimals"`
	AtomicAmount string `json:"atomicAmount"`
	ObservedSlot string `json:"observedSlot"`
}

type wormPositionCashOutBatchItemResponse struct {
	ID                    string                                   `json:"id"`
	Ordinal               int64                                    `json:"ordinal"`
	WalletOrdinal         int32                                    `json:"walletOrdinal"`
	PositionOrdinal       int32                                    `json:"positionOrdinal"`
	WalletID              int64                                    `json:"walletId"`
	WalletAddress         string                                   `json:"walletAddress"`
	WalletRemark          string                                   `json:"walletRemark"`
	PositionPubkey        string                                   `json:"positionPubkey"`
	PositionRequestPubkey string                                   `json:"positionRequestPubkey"`
	MarketConditionID     string                                   `json:"marketConditionId"`
	MarketTitle           string                                   `json:"marketTitle"`
	IsYes                 bool                                     `json:"isYes"`
	Shares                string                                   `json:"shares"`
	PositionCreatedAt     int64                                    `json:"positionCreatedAt"`
	State                 string                                   `json:"state"`
	ReasonCode            string                                   `json:"reasonCode"`
	ChildOperationID      string                                   `json:"childOperationId"`
	Baseline              *wormPositionCashOutBatchBalanceResponse `json:"baseline"`
	Observed              *wormPositionCashOutBatchBalanceResponse `json:"observed"`
	DeltaAtomicAmount     string                                   `json:"deltaAtomicAmount"`
	BalanceDeadlineAt     int64                                    `json:"balanceDeadlineAt"`
	CompletedAt           int64                                    `json:"completedAt"`
	UpdatedAt             int64                                    `json:"updatedAt"`
}

type wormPositionCashOutBatchResponse struct {
	ID                     string                                   `json:"id"`
	State                  string                                   `json:"state"`
	ReasonCode             string                                   `json:"reasonCode"`
	Revision               int64                                    `json:"revision"`
	WalletCount            int64                                    `json:"walletCount"`
	PositionCount          int64                                    `json:"positionCount"`
	CompletedCount         int64                                    `json:"completedCount"`
	NotExecutedCount       int64                                    `json:"notExecutedCount"`
	CurrentWalletOrdinal   int32                                    `json:"currentWalletOrdinal"`
	CurrentPositionOrdinal int32                                    `json:"currentPositionOrdinal"`
	CurrentItem            *wormPositionCashOutBatchItemResponse    `json:"currentItem"`
	Wallets                []wormPositionCashOutBatchWalletResponse `json:"wallets"`
	AuthorizationExpiresAt int64                                    `json:"authorizationExpiresAt"`
	ExecutionStartedAt     int64                                    `json:"executionStartedAt"`
	CompletedAt            int64                                    `json:"completedAt"`
	CreatedAt              int64                                    `json:"createdAt"`
	UpdatedAt              int64                                    `json:"updatedAt"`
	AllowedActions         []string                                 `json:"allowedActions"`
}

type wormPositionCashOutBatchItemPage struct {
	Items    []wormPositionCashOutBatchItemResponse `json:"items"`
	Total    int64                                  `json:"total"`
	Page     int32                                  `json:"page"`
	PageSize int32                                  `json:"pageSize"`
}

func registerWormPositionCashOutBatchHandlers(mux *http.ServeMux, server *AthenaServer) {
	if mux == nil || server == nil {
		return
	}
	mux.Handle("POST "+wormPositionCashOutBatchCollectionPath, traceHTTP(http.HandlerFunc(server.createWormPositionCashOutBatch)))
	mux.Handle("GET "+wormPositionCashOutBatchActivePath, traceHTTP(http.HandlerFunc(server.getActiveWormPositionCashOutBatch)))
	mux.Handle("GET "+wormPositionCashOutBatchItemsPath, traceHTTP(http.HandlerFunc(server.listWormPositionCashOutBatchItems)))
	mux.Handle("GET "+wormPositionCashOutBatchResourcePath, traceHTTP(http.HandlerFunc(server.getWormPositionCashOutBatch)))
	mux.Handle("POST "+wormPositionCashOutBatchMutationPath, traceHTTP(http.HandlerFunc(server.mutateWormPositionCashOutBatch)))
}

func (server *AthenaServer) createWormPositionCashOutBatch(w http.ResponseWriter, request *http.Request) {
	ctx, credential, ok := server.authenticateWormExecutionMutation(w, request)
	if !ok {
		return
	}
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	var input wormPositionCashOutBatchCreateInput
	if err := decodeWormExecutionJSON(w, request, &input); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	commandID, err := canonicalWormExecutionID(input.CommandID, "commandId")
	if err != nil || len(input.WalletIDs) == 0 || len(input.WalletIDs) > wormPositionCashOutBatchMaxWallets {
		walletsecret.WriteError(w, status.Error(codes.InvalidArgument, "position Cash Out Batch command or wallet selection is invalid"))
		return
	}
	selected := make(map[int64]struct{}, len(input.WalletIDs))
	for _, walletID := range input.WalletIDs {
		if walletID <= 0 {
			walletsecret.WriteError(w, status.Error(codes.InvalidArgument, "position Cash Out Batch wallet IDs must be positive"))
			return
		}
		if _, duplicate := selected[walletID]; duplicate {
			walletsecret.WriteError(w, status.Error(codes.InvalidArgument, "position Cash Out Batch wallet IDs must be unique"))
			return
		}
		selected[walletID] = struct{}{}
	}
	selection, err := server.loadWormWalletSelection(ctx, credential.AccountID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	orderedWalletIDs := make([]int64, 0, len(selected))
	for _, item := range selection.SelectedItems {
		if _, included := selected[item.WalletID]; included {
			orderedWalletIDs = append(orderedWalletIDs, item.WalletID)
		}
	}
	// New batches use the durable selection order. If the request no longer
	// belongs to the current selection, preserve all caller IDs so the service
	// can still resolve an exact idempotent replay before rejecting a new batch.
	if len(orderedWalletIDs) != len(selected) {
		orderedWalletIDs = append([]int64(nil), input.WalletIDs...)
	}
	wallets, err := server.resolveOwnedPositionCashOutBatchWallets(ctx, credential.AccountID, orderedWalletIDs)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.CreatePositionCashOutBatch(ctx, &wormtradingapiclient.CreatePositionCashOutBatchRequest{
		OwnerAccountId: credential.AccountID,
		CommandId:      commandID,
		Wallets:        wallets,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormPositionCashOutBatchError(err))
		return
	}
	response, err := projectWormPositionCashOutBatchForCredential(result.GetBatch(), credential, "")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	walletIDs := make([]string, 0, len(orderedWalletIDs))
	for _, walletID := range orderedWalletIDs {
		walletIDs = append(walletIDs, strconv.FormatInt(walletID, 10))
	}
	observeWormHTTPMutationStrings(request.Context(), "cash_out_batch", response.ID, "WORM_CASH_OUT_BATCH_CREATE", response.State, true,
		map[string][]string{"walletIds": walletIDs},
		map[string]string{"requestedCount": strconv.Itoa(len(walletIDs)), "batchId": response.ID, "state": response.State})
	w.Header().Set("Location", wormPositionCashOutBatchCollectionPath+"/"+response.ID)
	writeWormCombinationJSON(w, http.StatusAccepted, response)
}

func (server *AthenaServer) getActiveWormPositionCashOutBatch(w http.ResponseWriter, request *http.Request) {
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
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.GetActivePositionCashOutBatch(ctx, &wormtradingapiclient.GetActivePositionCashOutBatchRequest{OwnerAccountId: credential.AccountID})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormPositionCashOutBatchError(err))
		return
	}
	response, err := projectWormPositionCashOutBatchForCredential(result.GetBatch(), credential, "")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) getWormPositionCashOutBatch(w http.ResponseWriter, request *http.Request) {
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
	batchID, err := canonicalWormExecutionID(request.PathValue("batchId"), "position Cash Out Batch ID")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.GetPositionCashOutBatch(ctx, &wormtradingapiclient.GetPositionCashOutBatchRequest{OwnerAccountId: credential.AccountID, Id: batchID})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormPositionCashOutBatchError(err))
		return
	}
	response, err := projectWormPositionCashOutBatchForCredential(result.GetBatch(), credential, batchID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) listWormPositionCashOutBatchItems(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelRead)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	batchID, err := canonicalWormExecutionID(request.PathValue("batchId"), "position Cash Out Batch ID")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	page, pageSize, err := wormPositionCashOutBatchPage(request)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.ListPositionCashOutBatchItems(ctx, &wormtradingapiclient.ListPositionCashOutBatchItemsRequest{
		OwnerAccountId: credential.AccountID, Id: batchID, Page: page, PageSize: pageSize,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormPositionCashOutBatchError(err))
		return
	}
	if result == nil || result.GetPage() != page || result.GetPageSize() != pageSize || result.GetTotal() < 0 || len(result.GetItems()) > int(pageSize) {
		walletsecret.WriteError(w, status.Error(codes.Internal, "Worm Trading returned an invalid Cash Out Batch item page"))
		return
	}
	items := make([]wormPositionCashOutBatchItemResponse, 0, len(result.GetItems()))
	for _, item := range result.GetItems() {
		projected, projectErr := projectWormPositionCashOutBatchItem(item)
		if projectErr != nil {
			walletsecret.WriteError(w, projectErr)
			return
		}
		items = append(items, projected)
	}
	writeWormCombinationJSON(w, http.StatusOK, wormPositionCashOutBatchItemPage{Items: items, Total: result.GetTotal(), Page: page, PageSize: pageSize})
}

func (server *AthenaServer) mutateWormPositionCashOutBatch(w http.ResponseWriter, request *http.Request) {
	resource := request.PathValue("batchMutationResource")
	batchID, action, ok := strings.Cut(resource, ":")
	if !ok || batchID == "" || strings.Contains(action, ":") {
		walletsecret.WriteError(w, status.Error(codes.NotFound, "position Cash Out Batch action not found"))
		return
	}
	ctx, credential, authenticated := server.authenticateWormExecutionMutation(w, request)
	if !authenticated {
		return
	}
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	batchID, err := canonicalWormExecutionID(batchID, "position Cash Out Batch ID")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	var input wormPositionCashOutBatchCommandInput
	if err := decodeWormExecutionJSON(w, request, &input); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	commandID, err := canonicalWormExecutionID(input.CommandID, "commandId")
	if err != nil || validateWormExecutionExpectedRevision(input.ExpectedRevision) != nil {
		walletsecret.WriteError(w, status.Error(codes.InvalidArgument, "position Cash Out Batch command binding is invalid"))
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	command := &wormtradingapiclient.PositionCashOutBatchCommandRequest{
		OwnerAccountId: credential.AccountID, Id: batchID, CommandId: commandID, ExpectedRevision: input.ExpectedRevision,
	}
	var batch *wormtradingapiclient.PositionCashOutBatch
	if action == "continue" {
		if err := server.requireWormPositionCashOutBatchContinuationAuthorization(
			ctx, client, credential, batchID, input.ExpectedRevision,
		); err != nil {
			walletsecret.WriteError(w, err)
			return
		}
	}
	switch action {
	case "cancel":
		result, callErr := client.CancelPositionCashOutBatch(ctx, command)
		err = callErr
		if result != nil {
			batch = result.GetBatch()
		}
	case "pause":
		result, callErr := client.PausePositionCashOutBatch(ctx, command)
		err = callErr
		if result != nil {
			batch = result.GetBatch()
		}
	case "continue":
		result, callErr := client.ContinuePositionCashOutBatch(ctx, command)
		err = callErr
		if result != nil {
			batch = result.GetBatch()
		}
	case "terminate":
		result, callErr := client.TerminatePositionCashOutBatch(ctx, command)
		err = callErr
		if result != nil {
			batch = result.GetBatch()
		}
	case "check-status":
		result, callErr := client.CheckPositionCashOutBatchStatus(ctx, command)
		err = callErr
		if result != nil {
			batch = result.GetBatch()
		}
	default:
		walletsecret.WriteError(w, status.Error(codes.NotFound, "position Cash Out Batch action not found"))
		return
	}
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormPositionCashOutBatchError(err))
		return
	}
	response, err := projectWormPositionCashOutBatchForCredential(batch, credential, batchID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	effect := "WORM_CASH_OUT_BATCH_" + strings.ToUpper(strings.ReplaceAll(action, "-", "_"))
	observeWormHTTPMutation(request.Context(), "cash_out_batch", response.ID, effect, response.State, false, map[string]string{
		"expectedRevision":  strconv.FormatInt(input.ExpectedRevision, 10),
		"confirmedRevision": strconv.FormatInt(response.Revision, 10),
		"state":             response.State,
	})
	writeWormCombinationJSON(w, http.StatusAccepted, response)
}

func (server *AthenaServer) resolveOwnedPositionCashOutBatchWallets(
	ctx context.Context,
	ownerAccountID string,
	orderedWalletIDs []int64,
) ([]*wormtradingapiclient.PositionCashOutBatchWalletInput, error) {
	if server.WalletClientset == nil || server.WalletClientset.Wallet() == nil {
		return nil, status.Error(codes.Unavailable, "Wallet is unavailable")
	}
	remaining := make(map[int64]struct{}, len(orderedWalletIDs))
	for _, walletID := range orderedWalletIDs {
		remaining[walletID] = struct{}{}
	}
	resolved := make(map[int64]*wormtradingapiclient.PositionCashOutBatchWalletInput, len(orderedWalletIDs))
	for page := int32(1); ; page++ {
		result, err := server.WalletClientset.Wallet().ListWallets(ctx, &walletapiclient.ListWalletsRequest{
			WalletType: "SOLANA", Page: page, PageSize: wormPositionCashOutBatchPageSize, RequesterAccountId: ownerAccountID,
		})
		if err != nil {
			return nil, sanitizeWormConnectionInventoryDependencyError(err, "Wallet")
		}
		if result == nil || result.GetPage() != page || result.GetPageSize() != wormPositionCashOutBatchPageSize || result.GetTotal() < int64(len(result.GetItems())) || len(result.GetItems()) > int(wormPositionCashOutBatchPageSize) {
			return nil, status.Error(codes.Internal, "Wallet returned an invalid Cash Out Batch wallet page")
		}
		for _, wallet := range result.GetItems() {
			if wallet == nil || wallet.ID <= 0 || wallet.WalletType != "SOLANA" {
				return nil, status.Error(codes.Internal, "Wallet returned an invalid Cash Out Batch wallet")
			}
			if _, wanted := remaining[wallet.ID]; !wanted {
				continue
			}
			address, addressErr := canonicalWormConditionID(wallet.Address, "wallet address")
			if addressErr != nil || address != wallet.Address {
				return nil, status.Error(codes.Internal, "Wallet returned an invalid Cash Out Batch address")
			}
			resolved[wallet.ID] = &wormtradingapiclient.PositionCashOutBatchWalletInput{
				WalletId: wallet.ID, Address: wallet.Address, Remark: wallet.Remark,
				AvatarKind: wallet.AvatarKind, AvatarPresetId: wallet.AvatarPresetID, AvatarUrl: wallet.AvatarURL,
			}
			delete(remaining, wallet.ID)
		}
		if len(remaining) == 0 {
			break
		}
		if int64(page)*int64(wormPositionCashOutBatchPageSize) >= result.GetTotal() || len(result.GetItems()) == 0 {
			return nil, status.Error(codes.NotFound, "one or more selected Wallets were not found")
		}
	}
	output := make([]*wormtradingapiclient.PositionCashOutBatchWalletInput, 0, len(orderedWalletIDs))
	for _, walletID := range orderedWalletIDs {
		wallet := resolved[walletID]
		if wallet == nil {
			return nil, status.Error(codes.Internal, "Wallet returned an incomplete Cash Out Batch wallet inventory")
		}
		output = append(output, wallet)
	}
	return output, nil
}

func projectWormPositionCashOutBatch(
	batch *wormtradingapiclient.PositionCashOutBatch,
	expectedOwnerAccountID string,
	expectedBatchID string,
) (wormPositionCashOutBatchResponse, error) {
	if batch == nil {
		return wormPositionCashOutBatchResponse{}, status.Error(codes.Internal, "Worm Trading returned an empty position Cash Out Batch")
	}
	ownerAccountID, err := accountcredentials.CanonicalAccountID(batch.GetOwnerAccountId())
	if err != nil || ownerAccountID != batch.GetOwnerAccountId() || ownerAccountID != expectedOwnerAccountID {
		return wormPositionCashOutBatchResponse{}, status.Error(codes.Internal, "Worm Trading returned a mismatched Cash Out Batch owner")
	}
	id, err := canonicalWormExecutionID(batch.GetId(), "position Cash Out Batch ID")
	if err != nil || (expectedBatchID != "" && id != expectedBatchID) {
		return wormPositionCashOutBatchResponse{}, status.Error(codes.Internal, "Worm Trading returned a mismatched position Cash Out Batch")
	}
	validStates := []string{"BUILDING", "AWAITING_AUTHORIZATION", "QUEUED", "RUNNING", "PAUSE_REQUESTED", "PAUSED", "TERMINATE_REQUESTED", "TERMINATED", "COMPLETED", "FAILED", "RECONCILIATION_REQUIRED", "CANCELLED", "EXPIRED"}
	if !slices.Contains(validStates, batch.GetState()) || batch.GetRevision() <= 0 || batch.GetWalletCount() <= 0 || batch.GetWalletCount() > wormPositionCashOutBatchMaxWallets || batch.GetPositionCount() < 0 || batch.GetPositionCount() > 1000 || batch.GetCompletedCount() < 0 || batch.GetNotExecutedCount() < 0 || batch.GetCompletedCount()+batch.GetNotExecutedCount() > batch.GetPositionCount() || batch.GetCreatedAt() <= 0 || batch.GetUpdatedAt() <= 0 || (batch.GetReasonCode() != "" && !validWormExecutionReasonCode(batch.GetReasonCode())) {
		return wormPositionCashOutBatchResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid position Cash Out Batch")
	}
	authorizationDigest := batch.GetAuthorizationSessionJtiDigest()
	authorizationRevision := batch.GetAuthorizationAccessRevision()
	if (len(authorizationDigest) == 0) != (authorizationRevision == 0) ||
		(len(authorizationDigest) != 0 && (len(authorizationDigest) != sha256.Size || authorizationRevision <= 0)) {
		return wormPositionCashOutBatchResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid Cash Out Batch authorization binding")
	}
	wallets := make([]wormPositionCashOutBatchWalletResponse, 0, len(batch.GetWallets()))
	for index, wallet := range batch.GetWallets() {
		if wallet == nil || wallet.GetOrdinal() != int32(index+1) || wallet.GetWalletId() <= 0 || wallet.GetPositionCount() < 0 || wallet.GetCompletedCount() < 0 || wallet.GetCompletedCount() > wallet.GetPositionCount() {
			return wormPositionCashOutBatchResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid Cash Out Batch Wallet")
		}
		address, addressErr := canonicalWormConditionID(wallet.GetAddress(), "wallet address")
		if addressErr != nil || address != wallet.GetAddress() {
			return wormPositionCashOutBatchResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid Cash Out Batch Wallet")
		}
		wallets = append(wallets, wormPositionCashOutBatchWalletResponse{Ordinal: wallet.GetOrdinal(), WalletID: wallet.GetWalletId(), Address: address, Remark: wallet.GetRemark(), AvatarKind: wallet.GetAvatarKind(), AvatarPresetID: wallet.GetAvatarPresetId(), AvatarURL: wallet.GetAvatarUrl(), PositionCount: wallet.GetPositionCount(), CompletedCount: wallet.GetCompletedCount()})
	}
	if int64(len(wallets)) != batch.GetWalletCount() {
		return wormPositionCashOutBatchResponse{}, status.Error(codes.Internal, "Worm Trading returned an incomplete Cash Out Batch Wallet list")
	}
	var currentItem *wormPositionCashOutBatchItemResponse
	if batch.GetCurrentItem() != nil {
		projected, projectErr := projectWormPositionCashOutBatchItem(batch.GetCurrentItem())
		if projectErr != nil {
			return wormPositionCashOutBatchResponse{}, projectErr
		}
		currentItem = &projected
	}
	actions := append([]string{}, batch.GetAllowedActions()...)
	for _, action := range actions {
		if !slices.Contains([]string{"AUTHORIZE_BATCH", "CANCEL", "PAUSE", "CONTINUE", "TERMINATE", "CHECK_STATUS"}, action) {
			return wormPositionCashOutBatchResponse{}, status.Error(codes.Internal, "Worm Trading returned invalid Cash Out Batch actions")
		}
	}
	return wormPositionCashOutBatchResponse{ID: id, State: batch.GetState(), ReasonCode: batch.GetReasonCode(), Revision: batch.GetRevision(), WalletCount: batch.GetWalletCount(), PositionCount: batch.GetPositionCount(), CompletedCount: batch.GetCompletedCount(), NotExecutedCount: batch.GetNotExecutedCount(), CurrentWalletOrdinal: batch.GetCurrentWalletOrdinal(), CurrentPositionOrdinal: batch.GetCurrentPositionOrdinal(), CurrentItem: currentItem, Wallets: wallets, AuthorizationExpiresAt: batch.GetAuthorizationExpiresAt(), ExecutionStartedAt: batch.GetExecutionStartedAt(), CompletedAt: batch.GetCompletedAt(), CreatedAt: batch.GetCreatedAt(), UpdatedAt: batch.GetUpdatedAt(), AllowedActions: actions}, nil
}

func projectWormPositionCashOutBatchForCredential(
	batch *wormtradingapiclient.PositionCashOutBatch,
	credential accountcredentials.AuthenticatedCredential,
	expectedBatchID string,
) (wormPositionCashOutBatchResponse, error) {
	projection, err := projectWormPositionCashOutBatch(batch, credential.AccountID, expectedBatchID)
	if err != nil {
		return wormPositionCashOutBatchResponse{}, err
	}
	if batch.GetState() != "PAUSED" || batch.GetCurrentItem() != nil ||
		!slices.Contains(projection.AllowedActions, "CONTINUE") {
		return projection, nil
	}
	digest, accessRevision, bindingErr := wormExecutionCredentialBinding(credential)
	if bindingErr == nil && wormPositionCashOutBatchAuthorizationMatches(batch, digest, accessRevision) {
		return projection, nil
	}
	projection.AllowedActions = []string{"AUTHORIZE_BATCH", "TERMINATE"}
	return projection, nil
}

func wormPositionCashOutBatchAuthorizationMatches(
	batch *wormtradingapiclient.PositionCashOutBatch,
	sessionJTIDigest []byte,
	accessRevision int64,
) bool {
	return batch != nil && len(sessionJTIDigest) == sha256.Size && accessRevision > 0 &&
		len(batch.GetAuthorizationSessionJtiDigest()) == sha256.Size &&
		batch.GetAuthorizationAccessRevision() == accessRevision &&
		subtle.ConstantTimeCompare(batch.GetAuthorizationSessionJtiDigest(), sessionJTIDigest) == 1
}

func (server *AthenaServer) requireWormPositionCashOutBatchContinuationAuthorization(
	ctx context.Context,
	client wormtradingapiclient.WormTradingServiceClient,
	credential accountcredentials.AuthenticatedCredential,
	batchID string,
	expectedRevision int64,
) error {
	result, err := client.GetPositionCashOutBatch(ctx, &wormtradingapiclient.GetPositionCashOutBatchRequest{
		OwnerAccountId: credential.AccountID,
		Id:             batchID,
	})
	if err != nil {
		return sanitizeWormPositionCashOutBatchError(err)
	}
	batch := result.GetBatch()
	if batch == nil || batch.GetId() != batchID || batch.GetOwnerAccountId() != credential.AccountID ||
		batch.GetRevision() != expectedRevision || batch.GetState() != "PAUSED" || batch.GetCurrentItem() != nil {
		return status.Error(codes.Aborted, "position Cash Out Batch revision or state changed")
	}
	digest, accessRevision, err := wormExecutionCredentialBinding(credential)
	if err != nil || !wormPositionCashOutBatchAuthorizationMatches(batch, digest, accessRevision) {
		return status.Error(codes.FailedPrecondition, "WORM_POSITION_CASH_OUT_BATCH_AUTHORIZATION_REQUIRED")
	}
	return nil
}

func projectWormPositionCashOutBatchItem(item *wormtradingapiclient.PositionCashOutBatchItem) (wormPositionCashOutBatchItemResponse, error) {
	if item == nil {
		return wormPositionCashOutBatchItemResponse{}, status.Error(codes.Internal, "Worm Trading returned an empty Cash Out Batch item")
	}
	id, idErr := canonicalWormExecutionID(item.GetId(), "Cash Out Batch item ID")
	walletAddress, walletErr := canonicalWormConditionID(item.GetWalletAddress(), "wallet address")
	positionPubkey, positionErr := canonicalWormConditionID(item.GetPositionPubkey(), "position pubkey")
	marketID, marketErr := canonicalWormConditionID(item.GetMarketConditionId(), "market condition ID")
	shares, sharesOK := new(big.Rat).SetString(item.GetShares())
	validStates := []string{"PENDING", "PREFLIGHTING", "CLOSING", "AWAITING_POSITION", "AWAITING_BALANCE", "COMPLETED", "FAILED", "RECONCILIATION_REQUIRED", "NOT_EXECUTED"}
	if idErr != nil || item.GetOrdinal() <= 0 || item.GetWalletOrdinal() <= 0 || item.GetPositionOrdinal() <= 0 || item.GetWalletId() <= 0 || walletErr != nil || positionErr != nil || marketErr != nil || !validWormExecutionDecimal(item.GetShares(), false) || !sharesOK || shares.Sign() <= 0 || item.GetPositionCreatedAt() <= 0 || !slices.Contains(validStates, item.GetState()) || (item.GetReasonCode() != "" && !validWormExecutionReasonCode(item.GetReasonCode())) || item.GetUpdatedAt() <= 0 {
		return wormPositionCashOutBatchItemResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid Cash Out Batch item")
	}
	positionRequestPubkey := item.GetPositionRequestPubkey()
	if positionRequestPubkey != "" {
		if _, err := canonicalWormConditionID(positionRequestPubkey, "position request pubkey"); err != nil {
			return wormPositionCashOutBatchItemResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid Cash Out Batch item")
		}
	}
	childOperationID := item.GetChildOperationId()
	if childOperationID != "" {
		if _, err := canonicalWormExecutionID(childOperationID, "child operation ID"); err != nil {
			return wormPositionCashOutBatchItemResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid Cash Out Batch item")
		}
	}
	baseline, err := projectWormPositionCashOutBatchBalance(item.GetBaseline())
	if err != nil {
		return wormPositionCashOutBatchItemResponse{}, err
	}
	observed, err := projectWormPositionCashOutBatchBalance(item.GetObserved())
	if err != nil {
		return wormPositionCashOutBatchItemResponse{}, err
	}
	if item.GetDeltaAtomicAmount() != "" {
		delta, ok := new(big.Int).SetString(item.GetDeltaAtomicAmount(), 10)
		if !ok || delta.Sign() <= 0 || delta.String() != item.GetDeltaAtomicAmount() {
			return wormPositionCashOutBatchItemResponse{}, status.Error(codes.Internal, "Worm Trading returned invalid Cash Out Batch balance evidence")
		}
	}
	return wormPositionCashOutBatchItemResponse{ID: id, Ordinal: item.GetOrdinal(), WalletOrdinal: item.GetWalletOrdinal(), PositionOrdinal: item.GetPositionOrdinal(), WalletID: item.GetWalletId(), WalletAddress: walletAddress, WalletRemark: item.GetWalletRemark(), PositionPubkey: positionPubkey, PositionRequestPubkey: positionRequestPubkey, MarketConditionID: marketID, MarketTitle: item.GetMarketTitle(), IsYes: item.GetIsYes(), Shares: item.GetShares(), PositionCreatedAt: item.GetPositionCreatedAt(), State: item.GetState(), ReasonCode: item.GetReasonCode(), ChildOperationID: childOperationID, Baseline: baseline, Observed: observed, DeltaAtomicAmount: item.GetDeltaAtomicAmount(), BalanceDeadlineAt: item.GetBalanceDeadlineAt(), CompletedAt: item.GetCompletedAt(), UpdatedAt: item.GetUpdatedAt()}, nil
}

func projectWormPositionCashOutBatchBalance(value *wormtradingapiclient.PositionCashOutBatchBalanceEvidence) (*wormPositionCashOutBatchBalanceResponse, error) {
	if value == nil {
		return nil, nil
	}
	amount, ok := new(big.Int).SetString(value.GetAtomicAmount(), 10)
	if value.GetMint() != "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v" || value.GetDecimals() != 6 || !ok || amount.Sign() < 0 || amount.String() != value.GetAtomicAmount() || value.GetObservedSlot() == 0 {
		return nil, status.Error(codes.Internal, "Worm Trading returned invalid Cash Out Batch balance evidence")
	}
	return &wormPositionCashOutBatchBalanceResponse{Mint: value.GetMint(), Decimals: value.GetDecimals(), AtomicAmount: value.GetAtomicAmount(), ObservedSlot: strconv.FormatUint(value.GetObservedSlot(), 10)}, nil
}

func wormPositionCashOutBatchPage(request *http.Request) (int32, int32, error) {
	query := request.URL.Query()
	for key := range query {
		if key != "page" && key != "pageSize" {
			return 0, 0, status.Error(codes.InvalidArgument, "unexpected query parameter")
		}
	}
	page, err := positiveWormConnectionQueryValue(query["page"], "page", 1, 1<<30)
	if err != nil {
		return 0, 0, err
	}
	pageSize, err := positiveWormConnectionQueryValue(query["pageSize"], "pageSize", 50, 100)
	if err != nil {
		return 0, 0, err
	}
	return page, pageSize, nil
}

func sanitizeWormPositionCashOutBatchError(err error) error {
	return sanitizeWormExecutionStoreError(err)
}
