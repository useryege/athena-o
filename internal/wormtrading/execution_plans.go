package wormtrading

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxExecutionPlanStepPageSize = 100

func (s *Service) CreateExecutionPlan(
	ctx context.Context,
	req *apiclient.CreateExecutionPlanRequest,
) (*apiclient.CreateExecutionPlanResponse, error) {
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	combinationID, err := normalizeMarketCombinationID(req.GetCombinationId())
	if err != nil {
		return nil, err
	}
	if req.GetExpectedCombinationRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_combination_revision must be positive")
	}
	if req.GetWalletSelectionRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "wallet_selection_revision must be positive")
	}
	wallets, err := executionPlanWalletInputsFromProto(req.GetWallets())
	if err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	selection, err := s.credentialStore.GetWalletSelection(ctx, ownerAccountID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, walletSelectionRPCError(err)
	}
	if !selection.Configured || selection.Revision != req.GetWalletSelectionRevision() {
		return nil, status.Error(codes.FailedPrecondition, "WALLET_SELECTION_CHANGED")
	}
	for _, wallet := range wallets {
		if !walletSelectionContains(selection, wallet.WalletID, wallet.Address) {
			return nil, status.Error(codes.FailedPrecondition, "WALLET_NOT_SELECTED")
		}
	}

	plan, err := s.credentialStore.CreateExecutionPlan(ctx, wormstore.CreateExecutionPlanRequest{
		OwnerAccountID:              ownerAccountID,
		CombinationID:               combinationID,
		ExpectedCombinationRevision: req.GetExpectedCombinationRevision(),
		WalletSelectionRevision:     req.GetWalletSelectionRevision(),
		Wallets:                     wallets,
		Now:                         timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionPlanRPCError(err)
	}
	return &apiclient.CreateExecutionPlanResponse{Plan: executionPlanToProto(plan)}, nil
}

func (s *Service) GetExecutionPlan(
	ctx context.Context,
	req *apiclient.GetExecutionPlanRequest,
) (*apiclient.GetExecutionPlanResponse, error) {
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	planID, err := normalizeMarketCombinationID(req.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}

	plan, err := s.credentialStore.GetExecutionPlan(ctx, ownerAccountID, planID, timeNowUTC())
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionPlanRPCError(err)
	}
	return &apiclient.GetExecutionPlanResponse{Plan: executionPlanToProto(plan)}, nil
}

func (s *Service) ListExecutionPlanSteps(
	ctx context.Context,
	req *apiclient.ListExecutionPlanStepsRequest,
) (*apiclient.ListExecutionPlanStepsResponse, error) {
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	planID, err := normalizeMarketCombinationID(req.GetId())
	if err != nil {
		return nil, err
	}
	if req.GetPage() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "page must be positive")
	}
	if req.GetPageSize() <= 0 || req.GetPageSize() > maxExecutionPlanStepPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "page_size must be between 1 and %d", maxExecutionPlanStepPageSize)
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}

	steps, total, err := s.credentialStore.ListExecutionPlanSteps(
		ctx,
		ownerAccountID,
		planID,
		req.GetPage(),
		req.GetPageSize(),
	)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionPlanRPCError(err)
	}
	items := make([]*apiclient.ExecutionPlanStep, 0, len(steps))
	for _, step := range steps {
		items = append(items, executionPlanStepToProto(step))
	}
	return &apiclient.ListExecutionPlanStepsResponse{
		Items:    items,
		Total:    total,
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	}, nil
}

func executionPlanWalletInputsFromProto(values []*apiclient.ExecutionPlanWalletInput) ([]wormstore.ExecutionPlanWalletInput, error) {
	if len(values) == 0 {
		return nil, status.Error(codes.InvalidArgument, "wallets must contain at least one wallet")
	}
	result := make([]wormstore.ExecutionPlanWalletInput, 0, len(values))
	seenWalletIDs := make(map[int64]struct{}, len(values))
	seenAddresses := make(map[string]struct{}, len(values))
	for index, value := range values {
		if value == nil {
			return nil, status.Errorf(codes.InvalidArgument, "wallet %d is required", index+1)
		}
		walletID, address, err := normalizeWormWalletReference(value.GetWalletId(), value.GetAddress())
		if err != nil {
			return nil, err
		}
		if _, exists := seenWalletIDs[walletID]; exists {
			return nil, status.Error(codes.InvalidArgument, "wallet_id values must be unique")
		}
		if _, exists := seenAddresses[address]; exists {
			return nil, status.Error(codes.InvalidArgument, "wallet address values must be unique")
		}
		remark := strings.TrimSpace(value.GetRemark())
		avatarKind := strings.TrimSpace(value.GetAvatarKind())
		avatarPresetID := strings.TrimSpace(value.GetAvatarPresetId())
		avatarURL := strings.TrimSpace(value.GetAvatarUrl())
		if remark != value.GetRemark() || avatarKind != value.GetAvatarKind() ||
			avatarPresetID != value.GetAvatarPresetId() || avatarURL != value.GetAvatarUrl() {
			return nil, status.Errorf(codes.InvalidArgument, "wallet %d presentation snapshot is not canonical", index+1)
		}
		seenWalletIDs[walletID] = struct{}{}
		seenAddresses[address] = struct{}{}
		result = append(result, wormstore.ExecutionPlanWalletInput{
			WalletID:       walletID,
			Address:        address,
			Remark:         remark,
			AvatarKind:     avatarKind,
			AvatarPresetID: avatarPresetID,
			AvatarURL:      avatarURL,
		})
	}
	return result, nil
}

func executionPlanToProto(plan *wormstore.ExecutionPlan) *apiclient.ExecutionPlan {
	if plan == nil {
		return nil
	}
	wallets := make([]*apiclient.ExecutionPlanWallet, 0, len(plan.Wallets))
	for _, wallet := range plan.Wallets {
		wallets = append(wallets, executionPlanWalletToProto(wallet))
	}
	items := make([]*apiclient.ExecutionPlanItem, 0, len(plan.Items))
	for _, item := range plan.Items {
		items = append(items, executionPlanItemToProto(item))
	}
	reasonCounts := make([]*apiclient.ExecutionPlanReasonCount, 0, len(plan.ReasonCounts))
	for _, count := range plan.ReasonCounts {
		reasonCounts = append(reasonCounts, &apiclient.ExecutionPlanReasonCount{
			ReasonCode: count.ReasonCode,
			Count:      count.Count,
		})
	}
	return &apiclient.ExecutionPlan{
		Id:                      plan.ID,
		OwnerAccountId:          plan.OwnerAccountID,
		CombinationId:           plan.CombinationID,
		CombinationName:         plan.CombinationName,
		CombinationRevision:     plan.CombinationRevision,
		WalletSelectionRevision: plan.WalletSelectionRevision,
		State:                   string(plan.State),
		BuildStage:              plan.BuildStage,
		FailureCode:             plan.FailureCode,
		WalletCount:             plan.WalletCount,
		ItemCount:               plan.ItemCount,
		TotalStepCount:          plan.TotalStepCount,
		CompletedStepCount:      plan.CompletedStepCount,
		ReadyStepCount:          plan.ReadyStepCount,
		SkippedStepCount:        plan.SkippedStepCount,
		TotalCollateral:         plan.TotalCollateral,
		TotalOpeningFee:         plan.TotalOpeningFee,
		TotalUserFundsNeeded:    plan.TotalUserFundsNeeded,
		RequestedAt:             executionPlanUnix(plan.RequestedAt),
		CompletedAt:             executionPlanUnix(plan.CompletedAt),
		ExpiresAt:               executionPlanUnix(plan.ExpiresAt),
		RetentionUntil:          executionPlanUnix(plan.RetentionUntil),
		CreatedAt:               executionPlanUnix(plan.CreatedAt),
		UpdatedAt:               executionPlanUnix(plan.UpdatedAt),
		Wallets:                 wallets,
		Items:                   items,
		UsabilityCode:           plan.UsabilityCode,
		ReasonCounts:            reasonCounts,
	}
}

func executionPlanWalletToProto(wallet wormstore.ExecutionPlanWallet) *apiclient.ExecutionPlanWallet {
	result := &apiclient.ExecutionPlanWallet{
		Ordinal:        wallet.Ordinal,
		WalletId:       wallet.WalletID,
		Address:        wallet.Address,
		Remark:         wallet.Remark,
		AvatarKind:     wallet.AvatarKind,
		AvatarPresetId: wallet.AvatarPresetID,
		AvatarUrl:      wallet.AvatarURL,
		Status:         wallet.Status,
		ReasonCode:     wallet.ReasonCode,
	}
	if wallet.ConnectionState != "" {
		result.Connection = &apiclient.WormWalletConnection{
			WalletId:    wallet.WalletID,
			Address:     wallet.Address,
			State:       string(wallet.ConnectionState),
			WarningCode: wallet.ConnectionWarningCode,
			ConnectedAt: executionPlanUnix(wallet.ConnectedAt),
		}
	}
	if wallet.SOLAvailability != "" {
		result.Sol = &apiclient.AssetBalance{
			AtomicAmount: wallet.SOLAtomicAmount,
			Amount:       wallet.SOLAmount,
			Decimals:     wallet.SOLDecimals,
			ObservedSlot: wallet.SOLObservedSlot,
			Availability: wallet.SOLAvailability,
			ErrorCode:    wallet.SOLErrorCode,
		}
	}
	if wallet.USDCAvailability != "" {
		result.Usdc = &apiclient.TokenBalance{
			Mint:              wallet.USDCMint,
			AtomicAmount:      wallet.USDCAtomicAmount,
			Amount:            wallet.USDCAmount,
			Decimals:          wallet.USDCDecimals,
			ObservedSlot:      wallet.USDCObservedSlot,
			Availability:      wallet.USDCAvailability,
			ErrorCode:         wallet.USDCErrorCode,
			TokenAccountCount: wallet.USDCTokenAccountCount,
		}
	}
	return result
}

func executionPlanItemToProto(item wormstore.ExecutionPlanItem) *apiclient.ExecutionPlanItem {
	result := &apiclient.ExecutionPlanItem{
		Ordinal:           item.Ordinal,
		EventConditionId:  item.EventConditionID,
		EventTitle:        item.EventTitle,
		EventLogo:         item.EventLogo,
		MarketConditionId: item.MarketConditionID,
		MarketTitle:       item.MarketTitle,
		MarketLogo:        item.MarketLogo,
		IsYes:             item.IsYes,
		OutcomeLabel:      item.OutcomeLabel,
		Backend:           item.Backend,
		Funds:             item.Funds,
		Leverage:          item.Leverage,
		State:             item.State,
		ReasonCode:        item.ReasonCode,
	}
	if executionPlanEstimateObserved(item.Estimate) {
		result.Estimate = executionPlanEstimateToProto(item.Estimate)
	}
	return result
}

func executionPlanEstimateObserved(estimate wormstore.ExecutionPlanEstimate) bool {
	return estimate.AveragePrice != "" || estimate.TotalShares != "" || estimate.TotalCost != "" ||
		estimate.BestAsk != "" || estimate.WorstFillPrice != "" || estimate.FeeAmount != "" ||
		estimate.UserFundsNeeded != "" || estimate.LiquidationPrice != ""
}

func executionPlanEstimateToProto(estimate wormstore.ExecutionPlanEstimate) *apiclient.ExecutionPlanEstimate {
	result := &apiclient.ExecutionPlanEstimate{
		AveragePrice:    estimate.AveragePrice,
		TotalShares:     estimate.TotalShares,
		TotalCost:       estimate.TotalCost,
		BestAsk:         estimate.BestAsk,
		WorstFillPrice:  estimate.WorstFillPrice,
		IsFullyFilled:   estimate.IsFullyFilled,
		FeeAmount:       estimate.FeeAmount,
		UserFundsNeeded: estimate.UserFundsNeeded,
	}
	if estimate.LiquidationPrice != "" {
		result.LiquidationPrice = &apiclient.WormOptionalString{Value: estimate.LiquidationPrice}
	}
	return result
}

func executionPlanStepToProto(step wormstore.ExecutionPlanStep) *apiclient.ExecutionPlanStep {
	return &apiclient.ExecutionPlanStep{
		Ordinal:             step.Ordinal,
		WalletOrdinal:       step.WalletOrdinal,
		ItemOrdinal:         step.ItemOrdinal,
		Disposition:         string(step.Disposition),
		ReasonCode:          step.ReasonCode,
		ProjectedUsdcBefore: step.ProjectedUSDCBefore,
		ProjectedUsdcAfter:  step.ProjectedUSDCAfter,
	}
}

func executionPlanUnix(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.Unix()
}

func executionPlanRPCError(err error) error {
	switch {
	case errors.Is(err, wormstore.ErrExecutionPlanNotFound), errors.Is(err, wormstore.ErrMarketCombinationNotFound):
		return status.Error(codes.NotFound, "execution plan not found")
	case errors.Is(err, wormstore.ErrExecutionPlanRevision), errors.Is(err, wormstore.ErrExecutionPlanCombinationChanged):
		return status.Error(codes.Aborted, "market combination revision conflict")
	case errors.Is(err, wormstore.ErrExecutionPlanWalletConnectionChanged):
		return status.Error(codes.FailedPrecondition, "execution plan wallet connection changed")
	case errors.Is(err, wormstore.ErrExecutionPlanCredentialChanged):
		return status.Error(codes.FailedPrecondition, "execution plan wallet credential changed")
	case errors.Is(err, wormstore.ErrExecutionPlanWalletSelectionChanged):
		return status.Error(codes.FailedPrecondition, "WALLET_SELECTION_CHANGED")
	case errors.Is(err, wormstore.ErrExecutionPlanBuildLease):
		return status.Error(codes.Aborted, "execution plan build lease was lost")
	case errors.Is(err, wormstore.ErrInvalidExecutionPlan):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		return status.Error(codes.Internal, "execution plan store operation failed")
	}
}
