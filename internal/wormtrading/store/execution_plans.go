package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	wormtradingsqlc "github.com/useryege/athena/internal/wormtrading/store/sqlc"
)

const (
	executionPlanReadyTTL          = 15 * time.Minute
	executionPlanRetention         = 7 * 24 * time.Hour
	executionPlanMinAddressLength  = 32
	executionPlanSOLDecimals       = int32(9)
	executionPlanUSDCDecimals      = int32(6)
	maxExecutionPlanStepPageSize   = 100
	maxExecutionPlanCleanupLimit   = 100
	maxExecutionPlanWorkerIDLength = 200
	maxExecutionPlanCodeLength     = 100
	executionPlanAdvisoryLiquidity = "LIQUIDITY_INSUFFICIENT"
)

var executionPlanAdvisoryOrder = []string{
	executionPlanAdvisoryLiquidity,
}

func (s *SQLStore) CreateExecutionPlan(
	ctx context.Context,
	req CreateExecutionPlanRequest,
) (*ExecutionPlan, error) {
	ownerUUID, ownerAccountID, err := marketCombinationUUID(req.OwnerAccountID, "owner account ID")
	if err != nil {
		return nil, invalidExecutionPlan(err)
	}
	combinationID, _, err := marketCombinationUUID(req.CombinationID, "market combination ID")
	if err != nil {
		return nil, invalidExecutionPlan(err)
	}
	if req.ExpectedCombinationRevision <= 0 {
		return nil, invalidExecutionPlan(fmt.Errorf("expected combination revision must be positive"))
	}
	wallets, err := normalizeExecutionPlanWallets(req.Wallets)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}

	now := canonicalNow(req.Now)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin create execution plan transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	combinationRow, err := queries.GetMarketCombinationForUpdate(ctx, wormtradingsqlc.GetMarketCombinationForUpdateParams{
		ID:             combinationID,
		OwnerAccountID: ownerUUID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMarketCombinationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock execution plan market combination: %w", err)
	}
	if combinationRow.Revision != req.ExpectedCombinationRevision {
		return nil, ErrExecutionPlanRevision
	}
	combination, err := loadMarketCombination(ctx, queries, combinationRow)
	if err != nil {
		return nil, err
	}
	if len(combination.Items) == 0 {
		return nil, invalidExecutionPlan(fmt.Errorf("market combination has no items"))
	}
	if int64(len(wallets)) > math.MaxInt64/int64(len(combination.Items)) {
		return nil, invalidExecutionPlan(fmt.Errorf("execution plan step count overflows"))
	}
	totalSteps := int64(len(wallets)) * int64(len(combination.Items))
	planID := uuid.New()
	planUUID := pgtype.UUID{Bytes: [16]byte(planID), Valid: true}
	row, err := queries.CreateExecutionPlan(ctx, wormtradingsqlc.CreateExecutionPlanParams{
		ID:                   planUUID,
		OwnerAccountID:       ownerUUID,
		CombinationID:        combinationID,
		CombinationName:      combination.Name,
		CombinationRevision:  combination.Revision,
		WalletCount:          int64(len(wallets)),
		ItemCount:            int64(len(combination.Items)),
		TotalStepCount:       totalSteps,
		RequireFullLiquidity: req.PreflightChecks.RequireFullLiquidity,
		Now:                  timestampParam(now),
		RetentionUntil:       timestampParam(now.Add(executionPlanRetention)),
	})
	if err != nil {
		return nil, fmt.Errorf("create execution plan: %w", err)
	}
	for index, wallet := range wallets {
		if err := queries.CreateExecutionPlanWallet(ctx, wormtradingsqlc.CreateExecutionPlanWalletParams{
			PlanID:         planUUID,
			Ordinal:        int32(index + 1),
			WalletID:       wallet.WalletID,
			Address:        wallet.Address,
			Remark:         wallet.Remark,
			AvatarKind:     wallet.AvatarKind,
			AvatarPresetID: wallet.AvatarPresetID,
			AvatarUrl:      wallet.AvatarURL,
		}); err != nil {
			return nil, fmt.Errorf("create execution plan wallet %d: %w", index+1, err)
		}
	}
	for _, item := range combination.Items {
		if err := queries.CreateExecutionPlanItem(ctx, wormtradingsqlc.CreateExecutionPlanItemParams{
			PlanID:            planUUID,
			Ordinal:           item.Ordinal,
			EventConditionID:  item.EventConditionID,
			EventTitle:        item.EventTitle,
			EventLogo:         item.EventLogo,
			MarketConditionID: item.MarketConditionID,
			MarketTitle:       item.MarketTitle,
			MarketLogo:        item.MarketLogo,
			IsYes:             item.IsYes,
			OutcomeLabel:      item.OutcomeLabel,
		}); err != nil {
			return nil, fmt.Errorf("create execution plan item %d: %w", item.Ordinal, err)
		}
	}
	plan, err := loadExecutionPlan(ctx, queries, row, now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create execution plan: %w", err)
	}
	plan.OwnerAccountID = ownerAccountID
	return &plan, nil
}

func (s *SQLStore) GetExecutionPlan(
	ctx context.Context,
	ownerAccountID string,
	planID string,
	now time.Time,
) (*ExecutionPlan, error) {
	ownerUUID, ownerAccountID, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return nil, invalidExecutionPlan(err)
	}
	id, _, err := marketCombinationUUID(planID, "execution plan ID")
	if err != nil {
		return nil, invalidExecutionPlan(err)
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin get execution plan transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.GetExecutionPlan(ctx, wormtradingsqlc.GetExecutionPlanParams{ID: id, OwnerAccountID: ownerUUID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionPlanNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get execution plan: %w", err)
	}
	plan, err := loadExecutionPlan(ctx, queries, row, canonicalNow(now))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit get execution plan: %w", err)
	}
	plan.OwnerAccountID = ownerAccountID
	return &plan, nil
}

func (s *SQLStore) ListExecutionPlanSteps(
	ctx context.Context,
	ownerAccountID string,
	planID string,
	page int32,
	pageSize int32,
) ([]ExecutionPlanStep, int64, error) {
	ownerUUID, _, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return nil, 0, invalidExecutionPlan(err)
	}
	id, _, err := marketCombinationUUID(planID, "execution plan ID")
	if err != nil {
		return nil, 0, invalidExecutionPlan(err)
	}
	if page <= 0 || pageSize <= 0 || pageSize > maxExecutionPlanStepPageSize {
		return nil, 0, invalidExecutionPlan(fmt.Errorf("page must be positive and page size must be between 1 and %d", maxExecutionPlanStepPageSize))
	}
	offset := int64(page-1) * int64(pageSize)
	if offset < 0 {
		return nil, 0, invalidExecutionPlan(fmt.Errorf("page offset is invalid"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, 0, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, 0, fmt.Errorf("begin list execution plan steps transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	if _, err := queries.GetExecutionPlan(ctx, wormtradingsqlc.GetExecutionPlanParams{ID: id, OwnerAccountID: ownerUUID}); errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, ErrExecutionPlanNotFound
	} else if err != nil {
		return nil, 0, fmt.Errorf("get execution plan for steps: %w", err)
	}
	total, err := queries.CountExecutionPlanSteps(ctx, id)
	if err != nil {
		return nil, 0, fmt.Errorf("count execution plan steps: %w", err)
	}
	rows, err := queries.ListExecutionPlanSteps(ctx, wormtradingsqlc.ListExecutionPlanStepsParams{
		PlanID:     id,
		PageSize:   pageSize,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list execution plan steps: %w", err)
	}
	steps := make([]ExecutionPlanStep, 0, len(rows))
	for _, row := range rows {
		steps = append(steps, mapExecutionPlanStep(row))
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("commit list execution plan steps: %w", err)
	}
	return steps, total, nil
}

func (s *SQLStore) ClaimExecutionPlan(
	ctx context.Context,
	workerID string,
	leaseTimeout time.Duration,
	now time.Time,
) (*ExecutionPlan, error) {
	workerID, err := normalizeExecutionPlanWorkerID(workerID)
	if err != nil {
		return nil, invalidExecutionPlan(err)
	}
	if leaseTimeout <= 0 {
		return nil, invalidExecutionPlan(fmt.Errorf("execution plan lease timeout must be positive"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	now = canonicalNow(now)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin claim execution plan transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.ClaimNextExecutionPlan(ctx, wormtradingsqlc.ClaimNextExecutionPlanParams{
		Now:            timestampParam(now),
		WorkerID:       workerID,
		LeaseExpiresAt: timestampParam(now.Add(leaseTimeout)),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit empty execution plan claim: %w", err)
		}
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim execution plan: %w", err)
	}
	plan, err := loadExecutionPlan(ctx, queries, row, now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim execution plan: %w", err)
	}
	return &plan, nil
}

func (s *SQLStore) UpdateExecutionPlanBuildProgress(
	ctx context.Context,
	progress ExecutionPlanBuildProgress,
) (*ExecutionPlan, error) {
	id, _, err := marketCombinationUUID(progress.PlanID, "execution plan ID")
	if err != nil {
		return nil, invalidExecutionPlan(err)
	}
	workerID, err := normalizeExecutionPlanWorkerID(progress.WorkerID)
	if err != nil {
		return nil, invalidExecutionPlan(err)
	}
	stage, err := normalizeExecutionPlanCode(progress.BuildStage, "build stage")
	if err != nil {
		return nil, invalidExecutionPlan(err)
	}
	if progress.CompletedStepCount < 0 || progress.LeaseExpiresAt.IsZero() {
		return nil, invalidExecutionPlan(fmt.Errorf("execution plan progress is invalid"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	now := canonicalNow(progress.Now)
	if !progress.LeaseExpiresAt.After(now) {
		return nil, invalidExecutionPlan(fmt.Errorf("execution plan lease must end in the future"))
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin execution plan progress transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.UpdateExecutionPlanBuildProgress(ctx, wormtradingsqlc.UpdateExecutionPlanBuildProgressParams{
		ID:                 id,
		WorkerID:           workerID,
		BuildStage:         stage,
		CompletedStepCount: progress.CompletedStepCount,
		LeaseExpiresAt:     timestampParam(progress.LeaseExpiresAt),
		Now:                timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionPlanBuildLease
	}
	if err != nil {
		return nil, fmt.Errorf("update execution plan build progress: %w", err)
	}
	plan := mapExecutionPlan(row)
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit execution plan progress: %w", err)
	}
	return &plan, nil
}

func (s *SQLStore) MarkExecutionPlanReady(
	ctx context.Context,
	req MarkExecutionPlanReadyRequest,
) (*ExecutionPlan, error) {
	id, _, err := marketCombinationUUID(req.PlanID, "execution plan ID")
	if err != nil {
		return nil, invalidExecutionPlan(err)
	}
	workerID, err := normalizeExecutionPlanWorkerID(req.WorkerID)
	if err != nil {
		return nil, invalidExecutionPlan(err)
	}
	if err := validateExecutionPlanTotals(req.TotalCollateral, req.TotalOpeningFee, req.TotalUserFundsNeeded); err != nil {
		return nil, invalidExecutionPlan(err)
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	now := canonicalNow(req.Now)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin mark execution plan ready transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.GetExecutionPlanForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionPlanNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock execution plan: %w", err)
	}
	if ExecutionPlanState(row.State) != ExecutionPlanStateBuilding || row.WorkerID != workerID || !timestampValue(row.LeaseExpiresAt).After(now) {
		return nil, ErrExecutionPlanBuildLease
	}
	if err := validateExecutionPlanReadyRequest(row, req); err != nil {
		return nil, err
	}
	combinationRow, err := queries.GetMarketCombinationForUpdate(ctx, wormtradingsqlc.GetMarketCombinationForUpdateParams{
		ID:             row.CombinationID,
		OwnerAccountID: row.OwnerAccountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionPlanCombinationChanged
	}
	if err != nil {
		return nil, fmt.Errorf("lock execution plan market combination: %w", err)
	}
	if combinationRow.Revision != row.CombinationRevision {
		return nil, ErrExecutionPlanCombinationChanged
	}
	walletRows, err := queries.ListExecutionPlanWallets(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list execution plan wallets for ready: %w", err)
	}
	if err := validateAndPersistExecutionPlanWalletObservations(ctx, queries, id, walletRows, req.Wallets); err != nil {
		return nil, err
	}
	itemRows, err := queries.ListExecutionPlanItems(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list execution plan items for ready: %w", err)
	}
	if err := validateAndPersistExecutionPlanItemObservations(ctx, queries, id, itemRows, req.Items); err != nil {
		return nil, err
	}
	if err := persistExecutionPlanSteps(ctx, tx, queries, id, walletRows, itemRows, req.Steps); err != nil {
		return nil, err
	}
	readyCount, skippedCount := executionPlanStepCounts(req.Steps)
	completedAt := now
	readyRow, err := queries.MarkExecutionPlanReady(ctx, wormtradingsqlc.MarkExecutionPlanReadyParams{
		ID:                   id,
		WorkerID:             workerID,
		CompletedStepCount:   int64(len(req.Steps)),
		ReadyStepCount:       readyCount,
		SkippedStepCount:     skippedCount,
		TotalCollateral:      req.TotalCollateral,
		TotalOpeningFee:      req.TotalOpeningFee,
		TotalUserFundsNeeded: req.TotalUserFundsNeeded,
		CompletedAt:          timestampParam(completedAt),
		ExpiresAt:            timestampParam(completedAt.Add(executionPlanReadyTTL)),
		RetentionUntil:       timestampParam(completedAt.Add(executionPlanRetention)),
		Now:                  timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionPlanBuildLease
	}
	if err != nil {
		return nil, fmt.Errorf("mark execution plan ready: %w", err)
	}
	plan, err := loadExecutionPlan(ctx, queries, readyRow, now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit mark execution plan ready: %w", err)
	}
	return &plan, nil
}

func (s *SQLStore) MarkExecutionPlanFailed(
	ctx context.Context,
	planID string,
	workerID string,
	failureCode string,
	now time.Time,
) (*ExecutionPlan, error) {
	id, _, err := marketCombinationUUID(planID, "execution plan ID")
	if err != nil {
		return nil, invalidExecutionPlan(err)
	}
	workerID, err = normalizeExecutionPlanWorkerID(workerID)
	if err != nil {
		return nil, invalidExecutionPlan(err)
	}
	failureCode, err = normalizeExecutionPlanCode(failureCode, "failure code")
	if err != nil || failureCode == "" {
		return nil, invalidExecutionPlan(fmt.Errorf("failure code is invalid"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	now = canonicalNow(now)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin mark execution plan failed transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.MarkExecutionPlanFailed(ctx, wormtradingsqlc.MarkExecutionPlanFailedParams{
		ID:             id,
		WorkerID:       workerID,
		FailureCode:    failureCode,
		CompletedAt:    timestampParam(now),
		RetentionUntil: timestampParam(now.Add(executionPlanRetention)),
		Now:            timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionPlanBuildLease
	}
	if err != nil {
		return nil, fmt.Errorf("mark execution plan failed: %w", err)
	}
	plan, err := loadExecutionPlan(ctx, queries, row, now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit mark execution plan failed: %w", err)
	}
	return &plan, nil
}

func (s *SQLStore) DeleteExpiredExecutionPlans(ctx context.Context, now time.Time, limit int32) (int64, error) {
	if limit <= 0 || limit > maxExecutionPlanCleanupLimit {
		return 0, invalidExecutionPlan(fmt.Errorf("cleanup limit must be between 1 and %d", maxExecutionPlanCleanupLimit))
	}
	if err := s.requireDatabase(); err != nil {
		return 0, err
	}
	rows, err := s.queries.DeleteExpiredExecutionPlans(ctx, wormtradingsqlc.DeleteExpiredExecutionPlansParams{
		Now:          timestampParam(canonicalNow(now)),
		CleanupLimit: limit,
	})
	if err != nil {
		return 0, fmt.Errorf("delete expired execution plans: %w", err)
	}
	return int64(len(rows)), nil
}

func loadExecutionPlan(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	row wormtradingsqlc.WormExecutionPlan,
	now time.Time,
) (ExecutionPlan, error) {
	walletRows, err := queries.ListExecutionPlanWallets(ctx, row.ID)
	if err != nil {
		return ExecutionPlan{}, fmt.Errorf("list execution plan wallets: %w", err)
	}
	itemRows, err := queries.ListExecutionPlanItems(ctx, row.ID)
	if err != nil {
		return ExecutionPlan{}, fmt.Errorf("list execution plan items: %w", err)
	}
	plan := mapExecutionPlan(row)
	plan.Wallets = make([]ExecutionPlanWallet, 0, len(walletRows))
	for _, walletRow := range walletRows {
		plan.Wallets = append(plan.Wallets, mapExecutionPlanWallet(walletRow))
	}
	plan.Items = make([]ExecutionPlanItem, 0, len(itemRows))
	for _, itemRow := range itemRows {
		plan.Items = append(plan.Items, mapExecutionPlanItem(itemRow))
	}
	if plan.State == ExecutionPlanStateReady {
		reasonRows, err := queries.ListExecutionPlanReasonCounts(ctx, row.ID)
		if err != nil {
			return ExecutionPlan{}, fmt.Errorf("list execution plan reason counts: %w", err)
		}
		plan.ReasonCounts = make([]ExecutionPlanReasonCount, 0, len(reasonRows))
		for _, reasonRow := range reasonRows {
			if reasonRow.ReasonCode == "" || reasonRow.Count <= 0 {
				return ExecutionPlan{}, fmt.Errorf("execution plan reason count is invalid")
			}
			plan.ReasonCounts = append(plan.ReasonCounts, ExecutionPlanReasonCount{
				ReasonCode: reasonRow.ReasonCode,
				Count:      reasonRow.Count,
			})
		}
		advisoryRows, err := queries.ListExecutionPlanAdvisoryCounts(ctx, row.ID)
		if err != nil {
			return ExecutionPlan{}, fmt.Errorf("list execution plan advisory counts: %w", err)
		}
		plan.AdvisoryCounts = make([]ExecutionPlanReasonCount, 0, len(advisoryRows))
		for _, advisoryRow := range advisoryRows {
			if advisoryRow.ReasonCode == "" || advisoryRow.Count <= 0 {
				return ExecutionPlan{}, fmt.Errorf("execution plan advisory count is invalid")
			}
			plan.AdvisoryCounts = append(plan.AdvisoryCounts, ExecutionPlanReasonCount{
				ReasonCode: advisoryRow.ReasonCode,
				Count:      advisoryRow.Count,
			})
		}
		plan.UsabilityCode = executionPlanUsability(ctx, queries, row, canonicalNow(now))
	}
	return plan, nil
}

func executionPlanUsability(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	row wormtradingsqlc.WormExecutionPlan,
	now time.Time,
) string {
	if expiresAt := timestampValue(row.ExpiresAt); !expiresAt.IsZero() && !expiresAt.After(now) {
		return ExecutionPlanUsabilityExpired
	}
	if row.ReadyStepCount == 0 {
		return ExecutionPlanUsabilityNoActionableSteps
	}
	combination, err := queries.GetMarketCombination(ctx, wormtradingsqlc.GetMarketCombinationParams{
		ID:             row.CombinationID,
		OwnerAccountID: row.OwnerAccountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ExecutionPlanUsabilityCombinationDeleted
	}
	if err == nil && combination.Revision != row.CombinationRevision {
		return ExecutionPlanUsabilityCombinationChanged
	}
	// A database read failure must never make a preview appear consumable. The
	// API Server projects this non-secret code as a refreshable availability
	// failure instead of treating it as a market skip.
	if err != nil {
		return "COMBINATION_UNAVAILABLE"
	}
	return ""
}

func mapExecutionPlan(row wormtradingsqlc.WormExecutionPlan) ExecutionPlan {
	return ExecutionPlan{
		ID:                   uuidValue(row.ID),
		OwnerAccountID:       uuidValue(row.OwnerAccountID),
		CombinationID:        uuidValue(row.CombinationID),
		CombinationName:      row.CombinationName,
		CombinationRevision:  row.CombinationRevision,
		State:                ExecutionPlanState(row.State),
		BuildStage:           row.BuildStage,
		FailureCode:          row.FailureCode,
		WorkerID:             row.WorkerID,
		LockedAt:             timestampValue(row.LockedAt),
		LeaseExpiresAt:       timestampValue(row.LeaseExpiresAt),
		WalletCount:          row.WalletCount,
		ItemCount:            row.ItemCount,
		TotalStepCount:       row.TotalStepCount,
		CompletedStepCount:   row.CompletedStepCount,
		ReadyStepCount:       row.ReadyStepCount,
		SkippedStepCount:     row.SkippedStepCount,
		TotalCollateral:      row.TotalCollateral,
		TotalOpeningFee:      row.TotalOpeningFee,
		TotalUserFundsNeeded: row.TotalUserFundsNeeded,
		RequestedAt:          timestampValue(row.RequestedAt),
		CompletedAt:          timestampValue(row.CompletedAt),
		ExpiresAt:            timestampValue(row.ExpiresAt),
		RetentionUntil:       timestampValue(row.RetentionUntil),
		CreatedAt:            timestampValue(row.CreatedAt),
		UpdatedAt:            timestampValue(row.UpdatedAt),
		PreflightChecks:      ExecutionPreflightChecks{RequireFullLiquidity: row.RequireFullLiquidity},
	}
}

func mapExecutionPlanWallet(row wormtradingsqlc.WormExecutionPlanWallet) ExecutionPlanWallet {
	return ExecutionPlanWallet{
		Ordinal: row.Ordinal,
		ExecutionPlanWalletInput: ExecutionPlanWalletInput{
			WalletID:       row.WalletID,
			Address:        row.Address,
			Remark:         row.Remark,
			AvatarKind:     row.AvatarKind,
			AvatarPresetID: row.AvatarPresetID,
			AvatarURL:      row.AvatarUrl,
		},
		ConnectionState:       ConnectionState(row.ConnectionState),
		ConnectionWarningCode: row.ConnectionWarningCode,
		ConnectedAt:           timestampValue(row.ConnectedAt),
		CredentialVersion:     row.CredentialVersion,
		SOLAtomicAmount:       row.SolAtomicAmount,
		SOLAmount:             row.SolAmount,
		SOLDecimals:           row.SolDecimals,
		SOLObservedSlot:       uint64(row.SolObservedSlot),
		SOLAvailability:       row.SolAvailability,
		SOLErrorCode:          row.SolErrorCode,
		USDCMint:              row.UsdcMint,
		USDCAtomicAmount:      row.UsdcAtomicAmount,
		USDCAmount:            row.UsdcAmount,
		USDCDecimals:          row.UsdcDecimals,
		USDCObservedSlot:      uint64(row.UsdcObservedSlot),
		USDCAvailability:      row.UsdcAvailability,
		USDCErrorCode:         row.UsdcErrorCode,
		USDCTokenAccountCount: row.UsdcTokenAccountCount,
		Status:                row.Status,
		ReasonCode:            row.ReasonCode,
	}
}

func mapExecutionPlanItem(row wormtradingsqlc.WormExecutionPlanItem) ExecutionPlanItem {
	return ExecutionPlanItem{
		Ordinal: row.Ordinal,
		MarketCombinationItemInput: MarketCombinationItemInput{
			EventConditionID:  row.EventConditionID,
			EventTitle:        row.EventTitle,
			EventLogo:         row.EventLogo,
			MarketConditionID: row.MarketConditionID,
			MarketTitle:       row.MarketTitle,
			MarketLogo:        row.MarketLogo,
			IsYes:             row.IsYes,
			OutcomeLabel:      row.OutcomeLabel,
		},
		Backend:    row.Backend,
		Funds:      row.Funds,
		Leverage:   row.Leverage,
		State:      row.State,
		ReasonCode: row.ReasonCode,
		Estimate: ExecutionPlanEstimate{
			AveragePrice:     row.EstimateAveragePrice,
			TotalShares:      row.EstimateTotalShares,
			TotalCost:        row.EstimateTotalCost,
			BestAsk:          row.EstimateBestAsk,
			WorstFillPrice:   row.EstimateWorstFillPrice,
			IsFullyFilled:    row.EstimateIsFullyFilled,
			FeeAmount:        row.EstimateFeeAmount,
			UserFundsNeeded:  row.EstimateUserFundsNeeded,
			LiquidationPrice: row.EstimateLiquidationPrice,
		},
	}
}

func mapExecutionPlanStep(row wormtradingsqlc.WormExecutionPlanStep) ExecutionPlanStep {
	return ExecutionPlanStep{
		Ordinal:             row.Ordinal,
		WalletOrdinal:       row.WalletOrdinal,
		ItemOrdinal:         row.ItemOrdinal,
		Disposition:         ExecutionPlanStepDisposition(row.Disposition),
		ReasonCode:          row.ReasonCode,
		ProjectedUSDCBefore: row.ProjectedUsdcBefore,
		ProjectedUSDCAfter:  row.ProjectedUsdcAfter,
		AdvisoryCodes:       append([]string(nil), row.AdvisoryCodes...),
	}
}

func normalizeExecutionPlanWallets(values []ExecutionPlanWalletInput) ([]ExecutionPlanWalletInput, error) {
	if len(values) == 0 || len(values) > math.MaxInt32 {
		return nil, invalidExecutionPlan(fmt.Errorf("at least one wallet is required"))
	}
	result := make([]ExecutionPlanWalletInput, 0, len(values))
	walletIDs := make(map[int64]struct{}, len(values))
	addresses := make(map[string]struct{}, len(values))
	for index, value := range values {
		value.Address = strings.TrimSpace(value.Address)
		value.Remark = strings.TrimSpace(value.Remark)
		value.AvatarKind = strings.TrimSpace(value.AvatarKind)
		value.AvatarPresetID = strings.TrimSpace(value.AvatarPresetID)
		value.AvatarURL = strings.TrimSpace(value.AvatarURL)
		if err := validateWalletReference(value.WalletID, value.Address); err != nil {
			return nil, invalidExecutionPlan(fmt.Errorf("wallet %d: %w", index+1, err))
		}
		if utf8.RuneCountInString(value.Remark) > 50 || utf8.RuneCountInString(value.AvatarKind) > 40 ||
			utf8.RuneCountInString(value.AvatarPresetID) > 100 || utf8.RuneCountInString(value.AvatarURL) > 4096 {
			return nil, invalidExecutionPlan(fmt.Errorf("wallet %d presentation snapshot is invalid", index+1))
		}
		if _, exists := walletIDs[value.WalletID]; exists {
			return nil, invalidExecutionPlan(fmt.Errorf("wallet ID %d is duplicated", value.WalletID))
		}
		if _, exists := addresses[value.Address]; exists {
			return nil, invalidExecutionPlan(fmt.Errorf("wallet address %q is duplicated", value.Address))
		}
		walletIDs[value.WalletID] = struct{}{}
		addresses[value.Address] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func normalizeExecutionPlanWorkerID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxExecutionPlanWorkerIDLength {
		return "", fmt.Errorf("worker ID is invalid")
	}
	return value, nil
}

func normalizeExecutionPlanCode(value string, label string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) > maxExecutionPlanCodeLength {
		return "", fmt.Errorf("%s is invalid", label)
	}
	return value, nil
}

func invalidExecutionPlan(err error) error {
	return fmt.Errorf("%w: %v", ErrInvalidExecutionPlan, err)
}

func validateExecutionPlanTotals(values ...string) error {
	for _, value := range values {
		if !validExecutionPlanDecimal(value) {
			return fmt.Errorf("execution plan total is not a non-negative decimal")
		}
	}
	return nil
}

func validExecutionPlanDecimal(value string) bool {
	if value == "" || value != strings.TrimSpace(value) {
		return false
	}
	whole, fraction, hasFraction := strings.Cut(value, ".")
	if whole == "" || (hasFraction && fraction == "") {
		return false
	}
	if len(whole) > 1 && whole[0] == '0' {
		return false
	}
	for _, character := range whole {
		if character < '0' || character > '9' {
			return false
		}
	}
	for _, character := range fraction {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func validateExecutionPlanReadyRequest(
	plan wormtradingsqlc.WormExecutionPlan,
	req MarkExecutionPlanReadyRequest,
) error {
	if int64(len(req.Wallets)) != plan.WalletCount || int64(len(req.Items)) != plan.ItemCount ||
		int64(len(req.Steps)) != plan.TotalStepCount {
		return invalidExecutionPlan(fmt.Errorf("execution plan ready snapshot count does not match frozen plan"))
	}
	if req.TotalCollateral == "" || req.TotalOpeningFee == "" || req.TotalUserFundsNeeded == "" {
		return invalidExecutionPlan(fmt.Errorf("execution plan totals are required"))
	}
	return nil
}

func validateAndPersistExecutionPlanWalletObservations(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	planID pgtype.UUID,
	stored []wormtradingsqlc.WormExecutionPlanWallet,
	observations []ExecutionPlanWalletObservation,
) error {
	if len(stored) != len(observations) {
		return invalidExecutionPlan(fmt.Errorf("wallet observation count does not match frozen plan"))
	}
	byOrdinal := make(map[int32]ExecutionPlanWalletObservation, len(observations))
	for _, observation := range observations {
		if observation.Ordinal <= 0 {
			return invalidExecutionPlan(fmt.Errorf("wallet observation ordinal is invalid"))
		}
		if _, exists := byOrdinal[observation.Ordinal]; exists {
			return invalidExecutionPlan(fmt.Errorf("wallet observation ordinal %d is duplicated", observation.Ordinal))
		}
		if err := validateExecutionPlanWalletObservation(observation); err != nil {
			return invalidExecutionPlan(err)
		}
		byOrdinal[observation.Ordinal] = observation
	}
	// Lock shared wallet state in one global order. Execution plans preserve the
	// caller's wallet order, so two plans can legitimately contain the same
	// wallets in opposite ordinal order. Taking connection and credential locks
	// by wallet ID prevents those finalizers from deadlocking each other.
	lockOrder := append([]wormtradingsqlc.WormExecutionPlanWallet(nil), stored...)
	sort.Slice(lockOrder, func(i, j int) bool {
		if lockOrder[i].WalletID == lockOrder[j].WalletID {
			return lockOrder[i].Ordinal < lockOrder[j].Ordinal
		}
		return lockOrder[i].WalletID < lockOrder[j].WalletID
	})
	for _, wallet := range lockOrder {
		observation, exists := byOrdinal[wallet.Ordinal]
		if !exists {
			return invalidExecutionPlan(fmt.Errorf("wallet observation %d is missing", wallet.Ordinal))
		}
		connection, err := queries.GetWalletConnectionForUpdate(ctx, wallet.WalletID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrExecutionPlanWalletConnectionChanged
		}
		if err != nil {
			return fmt.Errorf("lock execution plan wallet connection: %w", err)
		}
		if connection.Address != wallet.Address || ConnectionState(connection.State) != ConnectionStateConnected ||
			observation.ConnectionState != ConnectionStateConnected {
			return ErrExecutionPlanWalletConnectionChanged
		}
		credential, err := queries.GetActiveCredentialForUpdate(ctx, wallet.WalletID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrExecutionPlanCredentialChanged
		}
		if err != nil {
			return fmt.Errorf("lock execution plan wallet credential: %w", err)
		}
		if observation.CredentialVersion <= 0 || credential.Version != observation.CredentialVersion {
			return ErrExecutionPlanCredentialChanged
		}
	}
	// Persist the immutable preview snapshot in the user-selected ordinal order
	// after every connection and active credential has passed the final CAS.
	for _, wallet := range stored {
		observation := byOrdinal[wallet.Ordinal]
		if err := queries.UpdateExecutionPlanWalletObservation(ctx, executionPlanWalletObservationParams(planID, observation)); err != nil {
			return fmt.Errorf("persist execution plan wallet observation %d: %w", observation.Ordinal, err)
		}
	}
	return nil
}

func validateExecutionPlanWalletObservation(value ExecutionPlanWalletObservation) error {
	if !validConnectionState(value.ConnectionState) || value.CredentialVersion < 0 || value.SOLDecimals < 0 ||
		value.USDCDecimals < 0 || value.USDCTokenAccountCount < 0 ||
		value.SOLObservedSlot > math.MaxInt64 || value.USDCObservedSlot > math.MaxInt64 {
		return fmt.Errorf("wallet observation is invalid")
	}
	for label, code := range map[string]string{
		"connection warning": value.ConnectionWarningCode,
		"SOL availability":   value.SOLAvailability,
		"SOL error":          value.SOLErrorCode,
		"USDC availability":  value.USDCAvailability,
		"USDC error":         value.USDCErrorCode,
		"status":             value.Status,
		"reason":             value.ReasonCode,
	} {
		if _, err := normalizeExecutionPlanCode(code, label); err != nil {
			return err
		}
	}
	if len(value.USDCMint) > maxWalletAddressLength || value.USDCMint != strings.TrimSpace(value.USDCMint) {
		return fmt.Errorf("USDC mint is invalid")
	}
	if value.ConnectionState != ConnectionStateConnected || value.CredentialVersion <= 0 || value.ConnectedAt.IsZero() {
		return fmt.Errorf("wallet connection observation is invalid")
	}
	if err := validateExecutionPlanBalanceSnapshot(
		value.SOLAtomicAmount,
		value.SOLAmount,
		value.SOLDecimals,
		value.SOLObservedSlot,
		value.SOLAvailability,
		value.SOLErrorCode,
		executionPlanSOLDecimals,
		0,
	); err != nil {
		return fmt.Errorf("SOL observation is invalid: %w", err)
	}
	if err := validateExecutionPlanBalanceSnapshot(
		value.USDCAtomicAmount,
		value.USDCAmount,
		value.USDCDecimals,
		value.USDCObservedSlot,
		value.USDCAvailability,
		value.USDCErrorCode,
		executionPlanUSDCDecimals,
		value.USDCTokenAccountCount,
	); err != nil {
		return fmt.Errorf("USDC observation is invalid: %w", err)
	}
	if value.USDCAvailability != "AVAILABLE" || len(value.USDCMint) < executionPlanMinAddressLength {
		return fmt.Errorf("USDC observation is unavailable")
	}
	expectedStatus := "PARTIAL"
	if value.SOLAvailability == "AVAILABLE" {
		expectedStatus = "COMPLETE"
	}
	if value.Status != expectedStatus {
		return fmt.Errorf("wallet balance status is inconsistent")
	}
	return nil
}

func validateExecutionPlanBalanceSnapshot(
	atomicAmount string,
	amount string,
	decimals int32,
	observedSlot uint64,
	availability string,
	errorCode string,
	expectedDecimals int32,
	tokenAccountCount int32,
) error {
	if decimals != expectedDecimals || tokenAccountCount < 0 {
		return fmt.Errorf("decimal metadata is inconsistent")
	}
	switch availability {
	case "AVAILABLE":
		if errorCode != "" || observedSlot == 0 || !executionPlanUnsignedInteger(atomicAmount) || !validExecutionPlanDecimal(amount) {
			return fmt.Errorf("available balance fields are inconsistent")
		}
	case "UNAVAILABLE":
		if errorCode == "" || observedSlot != 0 || atomicAmount != "" || amount != "" || tokenAccountCount != 0 {
			return fmt.Errorf("unavailable balance fields are inconsistent")
		}
	default:
		return fmt.Errorf("availability is invalid")
	}
	return nil
}

func executionPlanUnsignedInteger(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func executionPlanWalletObservationParams(
	planID pgtype.UUID,
	value ExecutionPlanWalletObservation,
) wormtradingsqlc.UpdateExecutionPlanWalletObservationParams {
	return wormtradingsqlc.UpdateExecutionPlanWalletObservationParams{
		PlanID:                planID,
		Ordinal:               value.Ordinal,
		ConnectionState:       string(value.ConnectionState),
		ConnectionWarningCode: value.ConnectionWarningCode,
		ConnectedAt:           timestampParam(value.ConnectedAt),
		CredentialVersion:     value.CredentialVersion,
		SolAtomicAmount:       value.SOLAtomicAmount,
		SolAmount:             value.SOLAmount,
		SolDecimals:           value.SOLDecimals,
		SolObservedSlot:       int64(value.SOLObservedSlot),
		SolAvailability:       value.SOLAvailability,
		SolErrorCode:          value.SOLErrorCode,
		UsdcMint:              value.USDCMint,
		UsdcAtomicAmount:      value.USDCAtomicAmount,
		UsdcAmount:            value.USDCAmount,
		UsdcDecimals:          value.USDCDecimals,
		UsdcObservedSlot:      int64(value.USDCObservedSlot),
		UsdcAvailability:      value.USDCAvailability,
		UsdcErrorCode:         value.USDCErrorCode,
		UsdcTokenAccountCount: value.USDCTokenAccountCount,
		Status:                value.Status,
		ReasonCode:            value.ReasonCode,
	}
}

func validateAndPersistExecutionPlanItemObservations(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	planID pgtype.UUID,
	stored []wormtradingsqlc.WormExecutionPlanItem,
	observations []ExecutionPlanItemObservation,
) error {
	if len(stored) != len(observations) {
		return invalidExecutionPlan(fmt.Errorf("market observation count does not match frozen plan"))
	}
	byOrdinal := make(map[int32]ExecutionPlanItemObservation, len(observations))
	for _, observation := range observations {
		if observation.Ordinal <= 0 {
			return invalidExecutionPlan(fmt.Errorf("market observation ordinal is invalid"))
		}
		if _, exists := byOrdinal[observation.Ordinal]; exists {
			return invalidExecutionPlan(fmt.Errorf("market observation ordinal %d is duplicated", observation.Ordinal))
		}
		if err := validateExecutionPlanItemObservation(observation); err != nil {
			return invalidExecutionPlan(err)
		}
		byOrdinal[observation.Ordinal] = observation
	}
	for _, item := range stored {
		observation, exists := byOrdinal[item.Ordinal]
		if !exists {
			return invalidExecutionPlan(fmt.Errorf("market observation %d is missing", item.Ordinal))
		}
		if err := queries.UpdateExecutionPlanItemObservation(ctx, executionPlanItemObservationParams(planID, observation)); err != nil {
			return fmt.Errorf("persist execution plan item observation %d: %w", observation.Ordinal, err)
		}
	}
	return nil
}

func validateExecutionPlanItemObservation(value ExecutionPlanItemObservation) error {
	for label, code := range map[string]string{
		"backend": value.Backend,
		"state":   value.State,
		"reason":  value.ReasonCode,
	} {
		if _, err := normalizeExecutionPlanCode(code, label); err != nil {
			return err
		}
	}
	if value.Backend == "" || value.State == "" || !validExecutionPlanDecimal(value.Funds) || !validExecutionPlanDecimal(value.Leverage) {
		return fmt.Errorf("market observation is invalid")
	}
	for _, decimal := range []string{
		value.Estimate.AveragePrice,
		value.Estimate.TotalShares,
		value.Estimate.TotalCost,
		value.Estimate.BestAsk,
		value.Estimate.WorstFillPrice,
		value.Estimate.FeeAmount,
		value.Estimate.UserFundsNeeded,
	} {
		if decimal != "" && !validExecutionPlanDecimal(decimal) {
			return fmt.Errorf("estimate decimal is invalid")
		}
	}
	if value.Estimate.LiquidationPrice != "" && !validExecutionPlanDecimal(value.Estimate.LiquidationPrice) {
		return fmt.Errorf("estimate liquidation price is invalid")
	}
	return nil
}

func executionPlanItemObservationParams(
	planID pgtype.UUID,
	value ExecutionPlanItemObservation,
) wormtradingsqlc.UpdateExecutionPlanItemObservationParams {
	return wormtradingsqlc.UpdateExecutionPlanItemObservationParams{
		PlanID:                   planID,
		Ordinal:                  value.Ordinal,
		Backend:                  value.Backend,
		Funds:                    value.Funds,
		Leverage:                 value.Leverage,
		State:                    value.State,
		ReasonCode:               value.ReasonCode,
		EstimateAveragePrice:     value.Estimate.AveragePrice,
		EstimateTotalShares:      value.Estimate.TotalShares,
		EstimateTotalCost:        value.Estimate.TotalCost,
		EstimateBestAsk:          value.Estimate.BestAsk,
		EstimateWorstFillPrice:   value.Estimate.WorstFillPrice,
		EstimateIsFullyFilled:    value.Estimate.IsFullyFilled,
		EstimateFeeAmount:        value.Estimate.FeeAmount,
		EstimateUserFundsNeeded:  value.Estimate.UserFundsNeeded,
		EstimateLiquidationPrice: value.Estimate.LiquidationPrice,
	}
}

func persistExecutionPlanSteps(
	ctx context.Context,
	tx pgx.Tx,
	queries *wormtradingsqlc.Queries,
	planID pgtype.UUID,
	wallets []wormtradingsqlc.WormExecutionPlanWallet,
	items []wormtradingsqlc.WormExecutionPlanItem,
	steps []ExecutionPlanStep,
) error {
	expected := int64(len(wallets)) * int64(len(items))
	if int64(len(steps)) != expected {
		return invalidExecutionPlan(fmt.Errorf("step count does not match wallet-major plan"))
	}
	for index, step := range steps {
		ordinal := int64(index + 1)
		walletIndex := index / len(items)
		itemIndex := index % len(items)
		if step.Ordinal != ordinal || step.WalletOrdinal != wallets[walletIndex].Ordinal || step.ItemOrdinal != items[itemIndex].Ordinal {
			return invalidExecutionPlan(fmt.Errorf("step %d is not wallet-major", index+1))
		}
		if step.Disposition != ExecutionPlanStepDispositionReady && step.Disposition != ExecutionPlanStepDispositionSkipped {
			return invalidExecutionPlan(fmt.Errorf("step %d disposition is invalid", index+1))
		}
		reasonCode, err := normalizeExecutionPlanCode(step.ReasonCode, "step reason")
		if err != nil {
			return invalidExecutionPlan(err)
		}
		if (step.Disposition == ExecutionPlanStepDispositionReady && reasonCode != "") ||
			(step.Disposition == ExecutionPlanStepDispositionSkipped && reasonCode == "") {
			return invalidExecutionPlan(fmt.Errorf("step %d disposition and reason are inconsistent", index+1))
		}
		if !validExecutionPlanDecimal(step.ProjectedUSDCBefore) || !validExecutionPlanDecimal(step.ProjectedUSDCAfter) {
			return invalidExecutionPlan(fmt.Errorf("step %d projected USDC is invalid", index+1))
		}
		if _, err := normalizeExecutionPlanAdvisoryCodes(step.AdvisoryCodes); err != nil {
			return invalidExecutionPlan(fmt.Errorf("step %d advisories are invalid: %w", index+1, err))
		}
	}
	if err := queries.DeleteExecutionPlanSteps(ctx, planID); err != nil {
		return fmt.Errorf("delete stale execution plan steps: %w", err)
	}
	written, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"worm_execution_plan_steps"},
		[]string{
			"plan_id",
			"ordinal",
			"wallet_ordinal",
			"item_ordinal",
			"disposition",
			"reason_code",
			"projected_usdc_before",
			"projected_usdc_after",
			"advisory_codes",
		},
		pgx.CopyFromSlice(len(steps), func(index int) ([]any, error) {
			step := steps[index]
			reasonCode, err := normalizeExecutionPlanCode(step.ReasonCode, "step reason")
			if err != nil {
				return nil, err
			}
			advisoryCodes, err := normalizeExecutionPlanAdvisoryCodes(step.AdvisoryCodes)
			if err != nil {
				return nil, err
			}
			return []any{
				planID,
				step.Ordinal,
				step.WalletOrdinal,
				step.ItemOrdinal,
				string(step.Disposition),
				reasonCode,
				step.ProjectedUSDCBefore,
				step.ProjectedUSDCAfter,
				advisoryCodes,
			}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("copy execution plan steps: %w", err)
	}
	if written != int64(len(steps)) {
		return fmt.Errorf("copy execution plan steps: wrote %d of %d rows", written, len(steps))
	}
	return nil
}

func normalizeExecutionPlanAdvisoryCodes(values ...[]string) ([]string, error) {
	seen := make(map[string]struct{}, len(executionPlanAdvisoryOrder))
	for _, group := range values {
		for _, value := range group {
			code, err := normalizeExecutionPlanCode(value, "advisory code")
			if err != nil || code == "" {
				return nil, fmt.Errorf("advisory code is invalid")
			}
			switch code {
			case executionPlanAdvisoryLiquidity:
				seen[code] = struct{}{}
			default:
				return nil, fmt.Errorf("advisory code %q is not supported", code)
			}
		}
	}
	result := make([]string, 0, len(seen))
	for _, code := range executionPlanAdvisoryOrder {
		if _, exists := seen[code]; exists {
			result = append(result, code)
		}
	}
	return result, nil
}

func executionPlanStepCounts(steps []ExecutionPlanStep) (int64, int64) {
	var ready, skipped int64
	for _, step := range steps {
		switch step.Disposition {
		case ExecutionPlanStepDispositionReady:
			ready++
		case ExecutionPlanStepDispositionSkipped:
			skipped++
		}
	}
	return ready, skipped
}
