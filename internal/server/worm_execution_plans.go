package server

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"math/big"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/internal/walletsecret"
	wormtradingapiclient "github.com/useryege/athena/internal/wormtrading/apiclient"
	applicationv1alpha1 "github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

const (
	wormExecutionPlanCollectionPath   = "/api/v1/worm-trading/execution-plans"
	wormExecutionPlanResourcePath     = wormExecutionPlanCollectionPath + "/{planId}"
	wormExecutionPlanStepsPath        = wormExecutionPlanResourcePath + "/steps"
	wormExecutionPlanDefaultPageSize  = int32(50)
	wormExecutionPlanMaximumPageSize  = int32(100)
	wormExecutionPlanMaximumBodyBytes = int64(1024 * 1024)
	wormExecutionWalletLookupLimit    = 8
	wormExecutionDecimalMaximumLength = 128
)

type wormExecutionPlanInput struct {
	CombinationID               string  `json:"combinationId"`
	ExpectedCombinationRevision int64   `json:"expectedCombinationRevision"`
	WalletIDs                   []int64 `json:"walletIds"`
}

type wormExecutionPlanResponse struct {
	ID                 string                       `json:"id"`
	Combination        wormExecutionPlanCombination `json:"combination"`
	State              string                       `json:"state"`
	BuildStage         string                       `json:"buildStage"`
	FailureCode        string                       `json:"failureCode"`
	UsabilityCode      string                       `json:"usabilityCode"`
	WalletCount        int64                        `json:"walletCount"`
	ItemCount          int64                        `json:"itemCount"`
	TotalStepCount     int64                        `json:"totalStepCount"`
	CompletedStepCount int64                        `json:"completedStepCount"`
	ReadyStepCount     int64                        `json:"readyStepCount"`
	SkippedStepCount   int64                        `json:"skippedStepCount"`
	ReasonCounts       map[string]int64             `json:"reasonCounts"`
	MaximumCollateral  string                       `json:"maximumCollateral"`
	OpeningFeeEstimate string                       `json:"openingFeeEstimate"`
	TotalUSDCNeeded    string                       `json:"totalUSDCNeeded"`
	RequestedAt        int64                        `json:"requestedAt"`
	CompletedAt        int64                        `json:"completedAt"`
	ExpiresAt          int64                        `json:"expiresAt"`
	RetentionUntil     int64                        `json:"retentionUntil"`
	CreatedAt          int64                        `json:"createdAt"`
	UpdatedAt          int64                        `json:"updatedAt"`
	Wallets            []wormExecutionPlanWallet    `json:"wallets"`
	Items              []wormExecutionPlanItem      `json:"items"`
}

type wormExecutionPlanCombination struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Revision int64  `json:"revision"`
}

type wormExecutionPlanWallet struct {
	Ordinal    int32                             `json:"ordinal"`
	Wallet     wormConnectionInventoryWallet     `json:"wallet"`
	Connection wormConnectionInventoryConnection `json:"connection"`
	SOL        wormExecutionPlanBalance          `json:"sol"`
	USDC       wormExecutionPlanBalance          `json:"usdc"`
	Status     string                            `json:"status"`
	ReasonCode string                            `json:"reasonCode"`
}

type wormExecutionPlanBalance struct {
	Mint              string `json:"mint,omitempty"`
	AtomicAmount      string `json:"atomicAmount"`
	Amount            string `json:"amount"`
	Decimals          int32  `json:"decimals"`
	ObservedSlot      string `json:"observedSlot"`
	Availability      string `json:"availability"`
	ErrorCode         string `json:"errorCode"`
	TokenAccountCount int32  `json:"tokenAccountCount,omitempty"`
}

type wormExecutionPlanItem struct {
	Ordinal           int32                      `json:"ordinal"`
	EventConditionID  string                     `json:"eventConditionId"`
	EventTitle        string                     `json:"eventTitle"`
	EventLogo         string                     `json:"eventLogo"`
	MarketConditionID string                     `json:"marketConditionId"`
	MarketTitle       string                     `json:"marketTitle"`
	MarketLogo        string                     `json:"marketLogo"`
	Side              string                     `json:"side"`
	OutcomeLabel      string                     `json:"outcomeLabel"`
	Backend           string                     `json:"backend"`
	Funds             string                     `json:"funds"`
	Leverage          string                     `json:"leverage"`
	State             string                     `json:"state"`
	ReasonCode        string                     `json:"reasonCode"`
	Estimate          *wormExecutionPlanEstimate `json:"estimate"`
}

type wormExecutionPlanEstimate struct {
	AveragePrice     string `json:"averagePrice"`
	TotalShares      string `json:"totalShares"`
	TotalCost        string `json:"totalCost"`
	BestAsk          string `json:"bestAsk"`
	WorstFillPrice   string `json:"worstFillPrice"`
	IsFullyFilled    bool   `json:"isFullyFilled"`
	FeeAmount        string `json:"feeAmount"`
	UserFundsNeeded  string `json:"userFundsNeeded"`
	LiquidationPrice string `json:"liquidationPrice"`
}

type wormExecutionPlanStep struct {
	Ordinal             int64  `json:"ordinal"`
	WalletOrdinal       int32  `json:"walletOrdinal"`
	ItemOrdinal         int32  `json:"itemOrdinal"`
	Disposition         string `json:"disposition"`
	ReasonCode          string `json:"reasonCode"`
	ProjectedUSDCBefore string `json:"projectedUSDCBefore"`
	ProjectedUSDCAfter  string `json:"projectedUSDCAfter"`
}

type wormExecutionPlanStepPage struct {
	Items    []wormExecutionPlanStep `json:"items"`
	Total    int64                   `json:"total"`
	Page     int32                   `json:"page"`
	PageSize int32                   `json:"pageSize"`
}

func registerWormExecutionPlanHandlers(mux *http.ServeMux, server *AthenaServer) {
	if mux == nil || server == nil {
		return
	}
	mux.Handle("POST "+wormExecutionPlanCollectionPath, traceHTTP(http.HandlerFunc(server.createWormExecutionPlan)))
	mux.Handle("GET "+wormExecutionPlanResourcePath, traceHTTP(http.HandlerFunc(server.getWormExecutionPlan)))
	mux.Handle("GET "+wormExecutionPlanStepsPath, traceHTTP(http.HandlerFunc(server.listWormExecutionPlanSteps)))
}

func (server *AthenaServer) createWormExecutionPlan(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if !server.validWormConnectionOrigin(request) {
		walletsecret.WriteError(w, status.Error(codes.PermissionDenied, "a same-origin Worm Trading request is required"))
		return
	}
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelReadWrite)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	input, err := decodeWormExecutionPlanInput(w, request)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	wallets, err := server.resolveWormExecutionPlanWallets(ctx, credential.AccountID, input.WalletIDs)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.CreateExecutionPlan(ctx, &wormtradingapiclient.CreateExecutionPlanRequest{
		OwnerAccountId:              credential.AccountID,
		CombinationId:               input.CombinationID,
		ExpectedCombinationRevision: input.ExpectedCombinationRevision,
		Wallets:                     wallets,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionPlanStoreError(err))
		return
	}
	response, err := projectWormExecutionPlan(result.GetPlan(), credential.AccountID, "")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	w.Header().Set("Location", wormExecutionPlanCollectionPath+"/"+response.ID)
	writeWormCombinationJSON(w, http.StatusAccepted, response)
}

func (server *AthenaServer) getWormExecutionPlan(w http.ResponseWriter, request *http.Request) {
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
	planID, err := canonicalWormCombinationID(request.PathValue("planId"))
	if err != nil {
		walletsecret.WriteError(w, status.Error(codes.InvalidArgument, "execution plan ID must be a canonical non-zero UUID"))
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.GetExecutionPlan(ctx, &wormtradingapiclient.GetExecutionPlanRequest{OwnerAccountId: credential.AccountID, Id: planID})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionPlanStoreError(err))
		return
	}
	response, err := projectWormExecutionPlan(result.GetPlan(), credential.AccountID, planID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) listWormExecutionPlanSteps(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelRead)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	planID, err := canonicalWormCombinationID(request.PathValue("planId"))
	if err != nil {
		walletsecret.WriteError(w, status.Error(codes.InvalidArgument, "execution plan ID must be a canonical non-zero UUID"))
		return
	}
	page, pageSize, err := wormExecutionPlanPagination(request)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.ListExecutionPlanSteps(ctx, &wormtradingapiclient.ListExecutionPlanStepsRequest{
		OwnerAccountId: credential.AccountID,
		Id:             planID,
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionPlanStoreError(err))
		return
	}
	response, err := projectWormExecutionPlanStepPage(result, page, pageSize)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func decodeWormExecutionPlanInput(w http.ResponseWriter, request *http.Request) (wormExecutionPlanInput, error) {
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return wormExecutionPlanInput{}, status.Error(codes.InvalidArgument, "Content-Type must be application/json")
	}
	request.Body = http.MaxBytesReader(w, request.Body, wormExecutionPlanMaximumBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var input wormExecutionPlanInput
	if err := decoder.Decode(&input); err != nil {
		return wormExecutionPlanInput{}, status.Error(codes.InvalidArgument, "request body must be one valid JSON object")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return wormExecutionPlanInput{}, status.Error(codes.InvalidArgument, "request body must contain exactly one JSON object")
	}
	combinationID, err := canonicalWormCombinationID(input.CombinationID)
	if err != nil {
		return wormExecutionPlanInput{}, err
	}
	if input.ExpectedCombinationRevision <= 0 || input.ExpectedCombinationRevision == math.MaxInt64 {
		return wormExecutionPlanInput{}, status.Error(codes.InvalidArgument, "expectedCombinationRevision must be a positive incrementable integer")
	}
	if len(input.WalletIDs) == 0 {
		return wormExecutionPlanInput{}, status.Error(codes.InvalidArgument, "walletIds must contain at least one wallet")
	}
	seen := make(map[int64]struct{}, len(input.WalletIDs))
	for _, walletID := range input.WalletIDs {
		if walletID <= 0 {
			return wormExecutionPlanInput{}, status.Error(codes.InvalidArgument, "walletIds must contain only positive integers")
		}
		if _, exists := seen[walletID]; exists {
			return wormExecutionPlanInput{}, status.Error(codes.InvalidArgument, "walletIds must not contain duplicates")
		}
		seen[walletID] = struct{}{}
	}
	input.CombinationID = combinationID
	return input, nil
}

func (server *AthenaServer) resolveWormExecutionPlanWallets(
	ctx context.Context,
	ownerAccountID string,
	walletIDs []int64,
) ([]*wormtradingapiclient.ExecutionPlanWalletInput, error) {
	if server.WalletClientset == nil || server.WalletClientset.Wallet() == nil {
		return nil, status.Error(codes.Unavailable, "Wallet is unavailable")
	}
	ownerAccountID, err := accountcredentials.CanonicalAccountID(ownerAccountID)
	if err != nil {
		return nil, status.Error(codes.Internal, "current account is invalid")
	}
	items := make([]*applicationv1alpha1.WalletItem, len(walletIDs))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(wormExecutionWalletLookupLimit)
	for index, walletID := range walletIDs {
		index, walletID := index, walletID
		group.Go(func() error {
			result, err := server.WalletClientset.Wallet().GetWallet(groupCtx, &apiclient.GetWalletRequest{
				Id:                 walletID,
				RequesterAccountId: ownerAccountID,
			})
			if err != nil {
				return sanitizeWormConnectionInventoryDependencyError(err, "Wallet")
			}
			if result == nil || result.GetItem() == nil {
				return status.Error(codes.Internal, "Wallet returned an incomplete execution plan wallet")
			}
			items[index] = result.GetItem()
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}

	result := make([]*wormtradingapiclient.ExecutionPlanWalletInput, len(items))
	seenAddresses := make(map[string]struct{}, len(items))
	for index, wallet := range items {
		if wallet == nil || wallet.ID != walletIDs[index] || wallet.WalletType != "SOLANA" {
			return nil, status.Error(codes.FailedPrecondition, "every execution plan wallet must be an owned Solana wallet")
		}
		address, addressErr := canonicalWormConditionID(wallet.Address, "wallet address")
		remark := strings.TrimSpace(wallet.Remark)
		avatarKind := strings.TrimSpace(wallet.AvatarKind)
		avatarPresetID := strings.TrimSpace(wallet.AvatarPresetID)
		avatarURL := strings.TrimSpace(wallet.AvatarURL)
		if addressErr != nil || address != wallet.Address || remark != wallet.Remark ||
			avatarKind != wallet.AvatarKind || avatarPresetID != wallet.AvatarPresetID || avatarURL != wallet.AvatarURL {
			return nil, status.Error(codes.Internal, "Wallet returned an invalid execution plan wallet")
		}
		if _, exists := seenAddresses[address]; exists {
			return nil, status.Error(codes.Internal, "Wallet returned duplicate execution plan addresses")
		}
		seenAddresses[address] = struct{}{}
		result[index] = &wormtradingapiclient.ExecutionPlanWalletInput{
			WalletId:       wallet.ID,
			Address:        address,
			Remark:         remark,
			AvatarKind:     avatarKind,
			AvatarPresetId: avatarPresetID,
			AvatarUrl:      avatarURL,
		}
	}
	return result, nil
}

func projectWormExecutionPlan(
	plan *wormtradingapiclient.ExecutionPlan,
	expectedOwnerAccountID string,
	expectedPlanID string,
) (wormExecutionPlanResponse, error) {
	if plan == nil {
		return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned an empty execution plan")
	}
	ownerAccountID, err := accountcredentials.CanonicalAccountID(plan.GetOwnerAccountId())
	if err != nil || ownerAccountID != plan.GetOwnerAccountId() || ownerAccountID != expectedOwnerAccountID {
		return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned a mismatched execution plan owner")
	}
	planID, err := canonicalWormCombinationID(plan.GetId())
	if err != nil || planID != plan.GetId() || (expectedPlanID != "" && planID != expectedPlanID) {
		return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan ID")
	}
	combinationID, err := canonicalWormCombinationID(plan.GetCombinationId())
	combinationName := strings.TrimSpace(plan.GetCombinationName())
	if err != nil || combinationID != plan.GetCombinationId() || validateWormCombinationName(combinationName) != nil ||
		combinationName != plan.GetCombinationName() || plan.GetCombinationRevision() <= 0 {
		return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan combination")
	}
	state := strings.TrimSpace(plan.GetState())
	buildStage := strings.TrimSpace(plan.GetBuildStage())
	failureCode := strings.TrimSpace(plan.GetFailureCode())
	usabilityCode := strings.TrimSpace(plan.GetUsabilityCode())
	if state != plan.GetState() || buildStage != plan.GetBuildStage() || failureCode != plan.GetFailureCode() || usabilityCode != plan.GetUsabilityCode() ||
		(state != "BUILDING" && state != "READY" && state != "FAILED") {
		return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan state")
	}
	if plan.GetWalletCount() <= 0 || plan.GetItemCount() <= 0 || plan.GetWalletCount() > math.MaxInt64/plan.GetItemCount() ||
		plan.GetTotalStepCount() != plan.GetWalletCount()*plan.GetItemCount() || plan.GetCompletedStepCount() < 0 ||
		plan.GetReadyStepCount() < 0 || plan.GetSkippedStepCount() < 0 || plan.GetCompletedStepCount() > plan.GetTotalStepCount() ||
		plan.GetReadyStepCount()+plan.GetSkippedStepCount() > plan.GetCompletedStepCount() {
		return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned invalid execution plan counts")
	}
	for _, amount := range []string{plan.GetTotalCollateral(), plan.GetTotalOpeningFee(), plan.GetTotalUserFundsNeeded()} {
		if !validWormExecutionDecimal(amount, false) {
			return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan total")
		}
	}
	if plan.GetRequestedAt() <= 0 || plan.GetCreatedAt() <= 0 || plan.GetUpdatedAt() < plan.GetCreatedAt() || plan.GetRetentionUntil() < plan.GetCreatedAt() {
		return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned invalid execution plan timestamps")
	}
	switch state {
	case "BUILDING":
		if plan.GetCompletedAt() != 0 || plan.GetExpiresAt() != 0 || failureCode != "" {
			return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid building execution plan")
		}
	case "READY":
		if plan.GetCompletedAt() <= 0 || plan.GetExpiresAt() <= plan.GetCompletedAt() || failureCode != "" || plan.GetCompletedStepCount() != plan.GetTotalStepCount() {
			return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid ready execution plan")
		}
		if time.Now().Unix() >= plan.GetExpiresAt() {
			state = "EXPIRED"
			if usabilityCode == "" {
				usabilityCode = "EXPIRED"
			}
		}
	case "FAILED":
		if plan.GetCompletedAt() <= 0 || failureCode == "" {
			return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid failed execution plan")
		}
	}
	if int64(len(plan.GetWallets())) != plan.GetWalletCount() || int64(len(plan.GetItems())) != plan.GetItemCount() {
		return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned incomplete execution plan snapshots")
	}
	response := wormExecutionPlanResponse{
		ID:                 planID,
		Combination:        wormExecutionPlanCombination{ID: combinationID, Name: combinationName, Revision: plan.GetCombinationRevision()},
		State:              state,
		BuildStage:         buildStage,
		FailureCode:        failureCode,
		UsabilityCode:      usabilityCode,
		WalletCount:        plan.GetWalletCount(),
		ItemCount:          plan.GetItemCount(),
		TotalStepCount:     plan.GetTotalStepCount(),
		CompletedStepCount: plan.GetCompletedStepCount(),
		ReadyStepCount:     plan.GetReadyStepCount(),
		SkippedStepCount:   plan.GetSkippedStepCount(),
		ReasonCounts:       make(map[string]int64, len(plan.GetReasonCounts())),
		MaximumCollateral:  plan.GetTotalCollateral(),
		OpeningFeeEstimate: plan.GetTotalOpeningFee(),
		TotalUSDCNeeded:    plan.GetTotalUserFundsNeeded(),
		RequestedAt:        plan.GetRequestedAt(),
		CompletedAt:        plan.GetCompletedAt(),
		ExpiresAt:          plan.GetExpiresAt(),
		RetentionUntil:     plan.GetRetentionUntil(),
		CreatedAt:          plan.GetCreatedAt(),
		UpdatedAt:          plan.GetUpdatedAt(),
		Wallets:            make([]wormExecutionPlanWallet, 0, len(plan.GetWallets())),
		Items:              make([]wormExecutionPlanItem, 0, len(plan.GetItems())),
	}
	var reasonCountTotal int64
	var skippedReasonCount int64
	previousReasonCode := ""
	for _, count := range plan.GetReasonCounts() {
		if count == nil || !validWormExecutionReasonCode(count.GetReasonCode()) || count.GetCount() <= 0 ||
			(previousReasonCode != "" && count.GetReasonCode() <= previousReasonCode) ||
			reasonCountTotal > math.MaxInt64-count.GetCount() {
			return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan reason count")
		}
		previousReasonCode = count.GetReasonCode()
		reasonCountTotal += count.GetCount()
		response.ReasonCounts[count.GetReasonCode()] = count.GetCount()
		if count.GetReasonCode() == "READY" {
			if count.GetCount() != plan.GetReadyStepCount() {
				return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned an inconsistent execution plan ready count")
			}
			continue
		}
		skippedReasonCount += count.GetCount()
	}
	if state == "READY" || state == "EXPIRED" {
		if reasonCountTotal != plan.GetTotalStepCount() || skippedReasonCount != plan.GetSkippedStepCount() {
			return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned incomplete execution plan reason counts")
		}
	} else if len(plan.GetReasonCounts()) != 0 {
		return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned premature execution plan reason counts")
	}
	seenWalletIDs := make(map[int64]struct{}, len(plan.GetWallets()))
	seenAddresses := make(map[string]struct{}, len(plan.GetWallets()))
	for index, wallet := range plan.GetWallets() {
		projected, err := projectWormExecutionPlanWallet(wallet, int32(index+1), state == "BUILDING" || state == "FAILED")
		if err != nil {
			return wormExecutionPlanResponse{}, err
		}
		if _, exists := seenWalletIDs[projected.Wallet.WalletID]; exists {
			return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned duplicate execution plan wallets")
		}
		if _, exists := seenAddresses[projected.Wallet.Address]; exists {
			return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned duplicate execution plan addresses")
		}
		seenWalletIDs[projected.Wallet.WalletID] = struct{}{}
		seenAddresses[projected.Wallet.Address] = struct{}{}
		response.Wallets = append(response.Wallets, projected)
	}
	seenMarkets := make(map[string]struct{}, len(plan.GetItems()))
	for index, item := range plan.GetItems() {
		projected, err := projectWormExecutionPlanItem(item, int32(index+1), state == "BUILDING" || state == "FAILED")
		if err != nil {
			return wormExecutionPlanResponse{}, err
		}
		if _, exists := seenMarkets[projected.MarketConditionID]; exists {
			return wormExecutionPlanResponse{}, status.Error(codes.Internal, "Worm Trading returned duplicate execution plan markets")
		}
		seenMarkets[projected.MarketConditionID] = struct{}{}
		response.Items = append(response.Items, projected)
	}
	return response, nil
}

func projectWormExecutionPlanWallet(wallet *wormtradingapiclient.ExecutionPlanWallet, expectedOrdinal int32, incompleteAllowed bool) (wormExecutionPlanWallet, error) {
	if wallet == nil || wallet.GetOrdinal() != expectedOrdinal || wallet.GetWalletId() <= 0 {
		return wormExecutionPlanWallet{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan wallet order")
	}
	address, err := canonicalWormConditionID(wallet.GetAddress(), "wallet address")
	remark := strings.TrimSpace(wallet.GetRemark())
	if err != nil || address != wallet.GetAddress() || remark != wallet.GetRemark() {
		return wormExecutionPlanWallet{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan wallet")
	}
	connection := wormConnectionInventoryConnection{}
	if wallet.GetConnection() != nil {
		if !validWormConnectionInventoryProjection(wallet.GetConnection(), wallet.GetWalletId(), address) {
			return wormExecutionPlanWallet{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan connection")
		}
		connection = wormConnectionInventoryConnection{
			State:       wallet.GetConnection().GetState(),
			WarningCode: wallet.GetConnection().GetWarningCode(),
			ConnectedAt: wallet.GetConnection().GetConnectedAt(),
		}
	} else if !incompleteAllowed {
		return wormExecutionPlanWallet{}, status.Error(codes.Internal, "Worm Trading omitted an execution plan connection")
	}
	projected := wormExecutionPlanWallet{
		Ordinal: wallet.GetOrdinal(),
		Wallet: wormConnectionInventoryWallet{
			WalletID:       wallet.GetWalletId(),
			Address:        address,
			Remark:         remark,
			AvatarKind:     strings.TrimSpace(wallet.GetAvatarKind()),
			AvatarPresetID: strings.TrimSpace(wallet.GetAvatarPresetId()),
			AvatarURL:      strings.TrimSpace(wallet.GetAvatarUrl()),
		},
		Connection: connection,
		Status:     strings.TrimSpace(wallet.GetStatus()),
		ReasonCode: strings.TrimSpace(wallet.GetReasonCode()),
	}
	if wallet.GetSol() != nil {
		projected.SOL = executionAssetBalance(wallet.GetSol())
	}
	if wallet.GetUsdc() != nil {
		projected.USDC = executionTokenBalance(wallet.GetUsdc())
	}
	if (!incompleteAllowed && (projected.Connection.State == "" || !validWormExecutionBalance(projected.SOL) ||
		!validWormExecutionBalance(projected.USDC) || projected.USDC.Mint == "")) ||
		projected.Status != wallet.GetStatus() || projected.ReasonCode != wallet.GetReasonCode() {
		return wormExecutionPlanWallet{}, status.Error(codes.Internal, "Worm Trading returned an incomplete execution plan wallet snapshot")
	}
	return projected, nil
}

func executionAssetBalance(balance *wormtradingapiclient.AssetBalance) wormExecutionPlanBalance {
	return wormExecutionPlanBalance{
		AtomicAmount: balance.GetAtomicAmount(), Amount: balance.GetAmount(), Decimals: balance.GetDecimals(),
		ObservedSlot: strconv.FormatUint(balance.GetObservedSlot(), 10), Availability: balance.GetAvailability(), ErrorCode: balance.GetErrorCode(),
	}
}

func executionTokenBalance(balance *wormtradingapiclient.TokenBalance) wormExecutionPlanBalance {
	return wormExecutionPlanBalance{
		Mint: balance.GetMint(), AtomicAmount: balance.GetAtomicAmount(), Amount: balance.GetAmount(), Decimals: balance.GetDecimals(),
		ObservedSlot: strconv.FormatUint(balance.GetObservedSlot(), 10), Availability: balance.GetAvailability(), ErrorCode: balance.GetErrorCode(),
		TokenAccountCount: balance.GetTokenAccountCount(),
	}
}

func validWormExecutionBalance(balance wormExecutionPlanBalance) bool {
	if balance.Availability == "AVAILABLE" {
		return balance.ErrorCode == "" && validWormExecutionObservedSlot(balance.ObservedSlot) && balance.Decimals >= 0 &&
			validWormExecutionDecimal(balance.AtomicAmount, false) && validWormExecutionDecimal(balance.Amount, false)
	}
	return balance.Availability == "UNAVAILABLE" && balance.ErrorCode != "" && balance.AtomicAmount == "" && balance.Amount == "" &&
		balance.ObservedSlot == "0" && balance.Decimals >= 0 && balance.TokenAccountCount == 0
}

func projectWormExecutionPlanItem(item *wormtradingapiclient.ExecutionPlanItem, expectedOrdinal int32, incompleteAllowed bool) (wormExecutionPlanItem, error) {
	if item == nil || item.GetOrdinal() != expectedOrdinal {
		return wormExecutionPlanItem{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan item order")
	}
	eventID, eventErr := canonicalWormConditionID(item.GetEventConditionId(), "event condition ID")
	marketID, marketErr := canonicalWormConditionID(item.GetMarketConditionId(), "market condition ID")
	eventTitle := strings.TrimSpace(item.GetEventTitle())
	marketTitle := strings.TrimSpace(item.GetMarketTitle())
	outcomeLabel := strings.TrimSpace(item.GetOutcomeLabel())
	if eventErr != nil || marketErr != nil || eventID != item.GetEventConditionId() || marketID != item.GetMarketConditionId() ||
		eventTitle == "" || marketTitle == "" || outcomeLabel == "" {
		return wormExecutionPlanItem{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan item")
	}
	backend := strings.TrimSpace(item.GetBackend())
	funds := strings.TrimSpace(item.GetFunds())
	leverage := strings.TrimSpace(item.GetLeverage())
	itemState := strings.TrimSpace(item.GetState())
	reasonCode := strings.TrimSpace(item.GetReasonCode())
	projected := wormExecutionPlanItem{
		Ordinal: item.GetOrdinal(), EventConditionID: eventID, EventTitle: eventTitle, EventLogo: strings.TrimSpace(item.GetEventLogo()),
		MarketConditionID: marketID, MarketTitle: marketTitle, MarketLogo: strings.TrimSpace(item.GetMarketLogo()),
		Side: "NO", OutcomeLabel: outcomeLabel, Backend: backend, Funds: funds, Leverage: leverage, State: itemState, ReasonCode: reasonCode,
	}
	if item.GetIsYes() {
		projected.Side = "YES"
	}
	if item.GetEstimate() != nil {
		estimate := item.GetEstimate()
		values := []string{estimate.GetAveragePrice(), estimate.GetTotalShares(), estimate.GetTotalCost(), estimate.GetBestAsk(), estimate.GetWorstFillPrice(), estimate.GetFeeAmount(), estimate.GetUserFundsNeeded()}
		for _, value := range values {
			if !validWormExecutionDecimal(value, false) {
				return wormExecutionPlanItem{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan estimate")
			}
		}
		liquidationPrice := ""
		if estimate.GetLiquidationPrice() != nil {
			liquidationPrice = estimate.GetLiquidationPrice().GetValue()
			if !validWormExecutionDecimal(liquidationPrice, false) {
				return wormExecutionPlanItem{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan liquidation price")
			}
		}
		projected.Estimate = &wormExecutionPlanEstimate{
			AveragePrice: estimate.GetAveragePrice(), TotalShares: estimate.GetTotalShares(), TotalCost: estimate.GetTotalCost(),
			BestAsk: estimate.GetBestAsk(), WorstFillPrice: estimate.GetWorstFillPrice(), IsFullyFilled: estimate.GetIsFullyFilled(),
			FeeAmount: estimate.GetFeeAmount(), UserFundsNeeded: estimate.GetUserFundsNeeded(), LiquidationPrice: liquidationPrice,
		}
		if leverage == "1" && (liquidationPrice != "" || !wormExecutionEstimateFundsMatch(funds, projected.Estimate.FeeAmount, projected.Estimate.UserFundsNeeded)) {
			return wormExecutionPlanItem{}, status.Error(codes.Internal, "Worm Trading returned an inconsistent execution plan estimate")
		}
	}
	if (!incompleteAllowed && (backend == "" || !validWormExecutionDecimal(funds, false) || leverage != "1" || itemState == "")) ||
		backend != item.GetBackend() || funds != item.GetFunds() || leverage != item.GetLeverage() || itemState != item.GetState() || reasonCode != item.GetReasonCode() {
		return wormExecutionPlanItem{}, status.Error(codes.Internal, "Worm Trading returned an incomplete execution plan item snapshot")
	}
	return projected, nil
}

func projectWormExecutionPlanStepPage(result *wormtradingapiclient.ListExecutionPlanStepsResponse, expectedPage, expectedPageSize int32) (wormExecutionPlanStepPage, error) {
	if result == nil || result.GetPage() != expectedPage || result.GetPageSize() != expectedPageSize || result.GetTotal() < 0 || len(result.GetItems()) > int(expectedPageSize) {
		return wormExecutionPlanStepPage{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan step page")
	}
	offset := int64(expectedPage-1) * int64(expectedPageSize)
	remaining := result.GetTotal() - offset
	if remaining < 0 {
		remaining = 0
	}
	expectedCount := int64(expectedPageSize)
	if remaining < expectedCount {
		expectedCount = remaining
	}
	if int64(len(result.GetItems())) != expectedCount {
		return wormExecutionPlanStepPage{}, status.Error(codes.Internal, "Worm Trading returned an inconsistent execution plan step page")
	}
	response := wormExecutionPlanStepPage{Items: make([]wormExecutionPlanStep, 0, len(result.GetItems())), Total: result.GetTotal(), Page: expectedPage, PageSize: expectedPageSize}
	for index, step := range result.GetItems() {
		if step == nil || step.GetOrdinal() != offset+int64(index)+1 || step.GetWalletOrdinal() <= 0 || step.GetItemOrdinal() <= 0 ||
			(step.GetDisposition() != "READY" && step.GetDisposition() != "SKIPPED") {
			return wormExecutionPlanStepPage{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan step")
		}
		reasonCode := strings.TrimSpace(step.GetReasonCode())
		if reasonCode != step.GetReasonCode() || (step.GetDisposition() == "READY" && reasonCode != "") || (step.GetDisposition() == "SKIPPED" && reasonCode == "") ||
			!validWormExecutionDecimal(step.GetProjectedUsdcBefore(), true) || !validWormExecutionDecimal(step.GetProjectedUsdcAfter(), true) {
			return wormExecutionPlanStepPage{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan step result")
		}
		response.Items = append(response.Items, wormExecutionPlanStep{
			Ordinal: step.GetOrdinal(), WalletOrdinal: step.GetWalletOrdinal(), ItemOrdinal: step.GetItemOrdinal(),
			Disposition: step.GetDisposition(), ReasonCode: reasonCode,
			ProjectedUSDCBefore: step.GetProjectedUsdcBefore(), ProjectedUSDCAfter: step.GetProjectedUsdcAfter(),
		})
	}
	return response, nil
}

func wormExecutionPlanPagination(request *http.Request) (int32, int32, error) {
	query := request.URL.Query()
	for key := range query {
		if key != "page" && key != "pageSize" {
			return 0, 0, status.Errorf(codes.InvalidArgument, "unsupported query parameter %q", key)
		}
	}
	page, err := positiveWormConnectionQueryValue(query["page"], "page", 1, 0)
	if err != nil {
		return 0, 0, err
	}
	pageSize, err := positiveWormConnectionQueryValue(query["pageSize"], "pageSize", wormExecutionPlanDefaultPageSize, wormExecutionPlanMaximumPageSize)
	if err != nil {
		return 0, 0, err
	}
	return page, pageSize, nil
}

func validWormExecutionDecimal(value string, allowEmpty bool) bool {
	if value == "" {
		return allowEmpty
	}
	if len(value) > wormExecutionDecimalMaximumLength || value != strings.TrimSpace(value) {
		return false
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" || (len(parts) == 2 && parts[1] == "") {
		return false
	}
	for _, part := range parts {
		for _, character := range part {
			if character < '0' || character > '9' {
				return false
			}
		}
	}
	parsed, ok := new(big.Rat).SetString(value)
	return ok && parsed.Sign() >= 0
}

func wormExecutionEstimateFundsMatch(funds, fee, userFundsNeeded string) bool {
	fundsValue, fundsOK := new(big.Rat).SetString(funds)
	feeValue, feeOK := new(big.Rat).SetString(fee)
	userFundsValue, userFundsOK := new(big.Rat).SetString(userFundsNeeded)
	if !fundsOK || !feeOK || !userFundsOK || fundsValue.Sign() < 0 || feeValue.Sign() < 0 || userFundsValue.Sign() < 0 {
		return false
	}
	return new(big.Rat).Add(fundsValue, feeValue).Cmp(userFundsValue) == 0
}

func validWormExecutionReasonCode(value string) bool {
	if value == "" || len(value) > 100 || value != strings.TrimSpace(value) {
		return false
	}
	for index, character := range value {
		if (character >= 'A' && character <= 'Z') || (index > 0 && character >= '0' && character <= '9') ||
			(index > 0 && character == '_') {
			continue
		}
		return false
	}
	return true
}

func validWormExecutionObservedSlot(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	parsed, ok := new(big.Int).SetString(value, 10)
	return ok && parsed.Sign() > 0
}

func sanitizeWormExecutionPlanStoreError(err error) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.Aborted, codes.FailedPrecondition, codes.ResourceExhausted:
		return err
	case codes.Canceled, codes.DeadlineExceeded, codes.Unavailable, codes.Unauthenticated, codes.PermissionDenied:
		return status.Error(codes.Unavailable, "Worm Trading execution preview is unavailable")
	default:
		return status.Error(codes.Internal, "Worm Trading execution preview request failed")
	}
}
