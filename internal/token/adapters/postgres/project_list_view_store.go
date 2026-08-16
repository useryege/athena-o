package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/projectview"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/selection"
)

func (repository *ProjectViewRepository) ListProjectsPage(
	ctx context.Context,
	filter projectview.ProjectListFilter,
	page, pageSize int32,
) (*projectview.ProjectListPage, error) {
	if repository == nil || repository.pool == nil {
		return nil, fmt.Errorf("token postgres repository is not configured")
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return nil, fmt.Errorf("begin project list snapshot: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := tokensqlc.New(tx)
	countParams := tokensqlc.CountProjectListItemsParams{
		ChainID:                       filter.ChainID,
		ProjectID:                     filter.ProjectID,
		CodeHash:                      optionalHashBytes(filter.CodeHash),
		Contract:                      optionalAddressBytes(filter.Contract),
		ResearchStatus:                string(filter.ResearchStatus),
		ReportState:                   filter.ReportState,
		EvaluationStatus:              string(filter.EvaluationStatus),
		SelectionOutcome:              string(filter.SelectionOutcome),
		WethPairRemoveLiquidityStates: filter.WethPair.RemoveLiquidityStates,
		WethPairMintStates:            filter.WethPair.MintStates,
		WethPairQuoteUsdtMin:          nullableNumericFromBigInt(filter.WethPair.QuoteUSDTMin),
		WethPairQuoteUsdtMax:          nullableNumericFromBigInt(filter.WethPair.QuoteUSDTMax),
		WethPairQuoteMissingStates:    filter.WethPair.QuoteMissingStates,
		UsdtPairRemoveLiquidityStates: filter.UsdtPair.RemoveLiquidityStates,
		UsdtPairMintStates:            filter.UsdtPair.MintStates,
		UsdtPairQuoteUsdtMin:          nullableNumericFromBigInt(filter.UsdtPair.QuoteUSDTMin),
		UsdtPairQuoteUsdtMax:          nullableNumericFromBigInt(filter.UsdtPair.QuoteUSDTMax),
		UsdtPairQuoteMissingStates:    filter.UsdtPair.QuoteMissingStates,
	}
	total, err := queries.CountProjectListItems(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count project list items: %w", err)
	}
	rows, err := queries.ListProjectListItems(ctx, tokensqlc.ListProjectListItemsParams{
		ChainID:                       countParams.ChainID,
		ProjectID:                     countParams.ProjectID,
		CodeHash:                      countParams.CodeHash,
		Contract:                      countParams.Contract,
		ResearchStatus:                countParams.ResearchStatus,
		ReportState:                   countParams.ReportState,
		EvaluationStatus:              countParams.EvaluationStatus,
		SelectionOutcome:              countParams.SelectionOutcome,
		WethPairRemoveLiquidityStates: countParams.WethPairRemoveLiquidityStates,
		WethPairMintStates:            countParams.WethPairMintStates,
		WethPairQuoteUsdtMin:          countParams.WethPairQuoteUsdtMin,
		WethPairQuoteUsdtMax:          countParams.WethPairQuoteUsdtMax,
		WethPairQuoteMissingStates:    countParams.WethPairQuoteMissingStates,
		UsdtPairRemoveLiquidityStates: countParams.UsdtPairRemoveLiquidityStates,
		UsdtPairMintStates:            countParams.UsdtPairMintStates,
		UsdtPairQuoteUsdtMin:          countParams.UsdtPairQuoteUsdtMin,
		UsdtPairQuoteUsdtMax:          countParams.UsdtPairQuoteUsdtMax,
		UsdtPairQuoteMissingStates:    countParams.UsdtPairQuoteMissingStates,
		Offset:                        offset,
		Limit:                         pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list project list items: %w", err)
	}
	items := make([]projectview.ProjectListItem, 0, len(rows))
	for _, row := range rows {
		item, mapErr := mapProjectListItem(row)
		if mapErr != nil {
			return nil, mapErr
		}
		items = append(items, item)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit project list snapshot: %w", err)
	}
	return &projectview.ProjectListPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func mapProjectListItem(row tokensqlc.ListProjectListItemsRow) (projectview.ProjectListItem, error) {
	blockTime, err := int64ToUint64("block_time", row.BlockTime)
	if err != nil {
		return projectview.ProjectListItem{}, fmt.Errorf("map project list item %d block time: %w", row.ID, err)
	}
	item := projectview.ProjectListItem{
		ProjectID:      row.ID,
		ChainID:        row.ChainID,
		Name:           row.Name,
		Symbol:         row.Symbol,
		BlockTime:      blockTime,
		CreatedAt:      timeValue(row.CreatedAt),
		LogoURL:        row.LogoUrl,
		ResearchStatus: research.ProjectResearchStatus(row.ResearchStatus),
	}
	if !row.ReportRevision.Valid {
		return item, nil
	}

	riskSummary, err := mapProjectListRiskSummary(row)
	if err != nil {
		return projectview.ProjectListItem{}, err
	}
	item.CurrentReport = &projectview.ProjectReportSummary{
		Revision:           row.ReportRevision.Int64,
		CompletenessStatus: row.ReportCompletenessStatus,
		BuiltAt:            timeValue(row.ReportBuiltAt),
		RiskSummary:        riskSummary,
		Evaluation:         mapProjectListEvaluationSummary(row),
	}
	return item, nil
}

func mapProjectListRiskSummary(row tokensqlc.ListProjectListItemsRow) (*projectview.ProjectReportRiskSummary, error) {
	wethPair, err := mapProjectListPairRiskSummary(
		row.ID,
		row.ReportRevision.Int64,
		"weth",
		row.WethPairIsCreated,
		row.WethPairIsRemoveLiquidity,
		row.WethPairIsMint,
		row.WethPairQuoteUsdtValueInt,
		row.WethPairLastSwapTimestamp,
	)
	if err != nil {
		return nil, err
	}
	usdtPair, err := mapProjectListPairRiskSummary(
		row.ID,
		row.ReportRevision.Int64,
		"usdt",
		row.UsdtPairIsCreated,
		row.UsdtPairIsRemoveLiquidity,
		row.UsdtPairIsMint,
		row.UsdtPairQuoteUsdtValueInt,
		row.UsdtPairLastSwapTimestamp,
	)
	if err != nil {
		return nil, err
	}
	if wethPair == nil && usdtPair == nil {
		return nil, nil
	}
	return &projectview.ProjectReportRiskSummary{WethPair: wethPair, UsdtPair: usdtPair}, nil
}

func mapProjectListPairRiskSummary(
	projectID, reportRevision int64,
	pairKind string,
	isCreated, isRemoveLiquidity, isMint pgtype.Bool,
	quoteUSDTValueInt pgtype.Numeric,
	lastSwapTimestamp pgtype.Int8,
) (*projectview.ProjectPairRiskSummary, error) {
	present := 0
	for _, valid := range []bool{
		isCreated.Valid,
		isRemoveLiquidity.Valid,
		isMint.Valid,
		quoteUSDTValueInt.Valid,
		lastSwapTimestamp.Valid,
	} {
		if valid {
			present++
		}
	}
	if present == 0 {
		return nil, nil
	}
	if present != 5 {
		return nil, fmt.Errorf(
			"project %d report revision %d has partially populated %s pair risk projection",
			projectID,
			reportRevision,
			pairKind,
		)
	}
	lastSwap, err := int64ToUint64(pairKind+"_pair_last_swap_timestamp", lastSwapTimestamp.Int64)
	if err != nil {
		return nil, fmt.Errorf("project %d report revision %d risk projection: %w", projectID, reportRevision, err)
	}
	quote, err := exactBigIntPointerFromNumeric(pairKind+"_pair_quote_usdt_value_int", quoteUSDTValueInt)
	if err != nil {
		return nil, fmt.Errorf("project %d report revision %d risk projection: %w", projectID, reportRevision, err)
	}
	return &projectview.ProjectPairRiskSummary{
		IsCreated:         isCreated.Bool,
		IsRemoveLiquidity: isRemoveLiquidity.Bool,
		IsMint:            isMint.Bool,
		QuoteUSDTValueInt: quote,
		LastSwapTimestamp: lastSwap,
	}, nil
}

func mapProjectListEvaluationSummary(row tokensqlc.ListProjectListItemsRow) *projectview.ProjectReportEvaluationSummary {
	if row.EvaluationStatus == "" {
		return nil
	}
	return &projectview.ProjectReportEvaluationSummary{
		Status:         selection.TaskStatus(row.EvaluationStatus),
		FailedAttempts: row.EvaluationFailedAttempts,
		LastError:      row.EvaluationLastError,
		UpdatedAt:      timeValue(row.EvaluationUpdatedAt),
		Outcome:        selection.SelectionOutcome(row.SelectionOutcome),
		EvaluatedAt:    timeValue(row.EvaluatedAt),
	}
}

func mapProjectSelectionEvaluationTask(row tokensqlc.ProjectSelectionEvaluationTask) *projectview.ProjectReportEvaluationSummary {
	return &projectview.ProjectReportEvaluationSummary{
		Status:         selection.TaskStatus(row.Status),
		FailedAttempts: row.Attempts,
		LastError:      textValue(row.LastError),
		UpdatedAt:      timeValue(row.UpdatedAt),
	}
}
