package postgres

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/profile"
	"github.com/useryege/athena/internal/token/projectview"
	"github.com/useryege/athena/internal/token/shared"
)

func (repository *ProjectViewRepository) ListProjectsPage(
	ctx context.Context,
	filter projectview.ProjectListFilter,
	page, pageSize int32,
) (*projectview.ProjectListPage, error) {
	if repository == nil || repository.pool == nil {
		return nil, fmt.Errorf("token project view repository is not configured")
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin project list snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)
	countParams := projectListCountParams(filter)
	total, err := queries.CountProjectListItems(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count project list items: %w", err)
	}
	rows, err := queries.ListProjectListItems(ctx, projectListParams(countParams, offset, pageSize))
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
	return &projectview.ProjectListPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func projectListCountParams(filter projectview.ProjectListFilter) tokensqlc.CountProjectListItemsParams {
	return tokensqlc.CountProjectListItemsParams{
		CodeHash: optionalHashBytes(filter.CodeHash), Contract: optionalAddressBytes(filter.Contract),
		CollectionStatus: string(filter.CollectionStatus), ProfileState: string(filter.ProfileState),
		PairBalanceSupplyStates: filter.Pair.PairTokenBalanceExceedsTotalSupplyStates,
		PairMinimumLpStates:     filter.Pair.LPMinimumSupplyOnlyStates,
		PairFeeLpShareStates:    filter.Pair.FixedFeeAddressLPShareGte90PercentStates,
		PairQuoteUsdtMin:        nullableNumericFromBigInt(filter.Pair.QuoteUSDTMin),
		PairQuoteUsdtMax:        nullableNumericFromBigInt(filter.Pair.QuoteUSDTMax),
	}
}

func projectListParams(source tokensqlc.CountProjectListItemsParams, offset, limit int32) tokensqlc.ListProjectListItemsParams {
	return tokensqlc.ListProjectListItemsParams{
		CodeHash: source.CodeHash, Contract: source.Contract,
		CollectionStatus: source.CollectionStatus, ProfileState: source.ProfileState,
		PairBalanceSupplyStates: source.PairBalanceSupplyStates,
		PairMinimumLpStates:     source.PairMinimumLpStates,
		PairFeeLpShareStates:    source.PairFeeLpShareStates,
		PairQuoteUsdtMin:        source.PairQuoteUsdtMin,
		PairQuoteUsdtMax:        source.PairQuoteUsdtMax,
		Offset:                  offset, Limit: limit,
	}
}

func mapProjectListItem(row tokensqlc.ListProjectListItemsRow) (projectview.ProjectListItem, error) {
	blockNumber, err := int64ToUint64("block_number", row.BlockNumber)
	if err != nil {
		return projectview.ProjectListItem{}, fmt.Errorf("map project list item %d: %w", row.ID, err)
	}
	blockTime, err := int64ToUint64("block_time", row.BlockTime)
	if err != nil {
		return projectview.ProjectListItem{}, fmt.Errorf("map project list item %d: %w", row.ID, err)
	}
	taskCount, err := boundedInt32("collection_task_count", row.CollectionTaskCount)
	if err != nil {
		return projectview.ProjectListItem{}, fmt.Errorf("map project list item %d: %w", row.ID, err)
	}
	terminalCount, err := boundedInt32("collection_terminal_count", row.CollectionTerminalCount)
	if err != nil {
		return projectview.ProjectListItem{}, fmt.Errorf("map project list item %d: %w", row.ID, err)
	}
	succeededCount, err := boundedInt32("collection_succeeded_count", row.CollectionSucceededCount)
	if err != nil {
		return projectview.ProjectListItem{}, fmt.Errorf("map project list item %d: %w", row.ID, err)
	}
	failedCount, err := boundedInt32("collection_failed_count", row.CollectionFailedCount)
	if err != nil {
		return projectview.ProjectListItem{}, fmt.Errorf("map project list item %d: %w", row.ID, err)
	}
	if taskCount > int32(6) || terminalCount > taskCount || succeededCount+failedCount != terminalCount {
		return projectview.ProjectListItem{}, fmt.Errorf("project %d has inconsistent collection counts", row.ID)
	}
	collectionStatus := projectview.ProjectCollectionStatus(row.CollectionStatus)
	switch collectionStatus {
	case projectview.ProjectCollectionStatusQueued, projectview.ProjectCollectionStatusCollecting,
		projectview.ProjectCollectionStatusComplete, projectview.ProjectCollectionStatusNeedsAttention:
	default:
		return projectview.ProjectListItem{}, fmt.Errorf("project %d has unsupported collection status %q", row.ID, row.CollectionStatus)
	}
	profileState := projectview.ProjectProfileState(row.ProfileState)
	switch profileState {
	case projectview.ProjectProfileStatePending, projectview.ProjectProfileStateComplete,
		projectview.ProjectProfileStateIncomplete, projectview.ProjectProfileStateFailed:
	default:
		return projectview.ProjectListItem{}, fmt.Errorf("project %d has unsupported profile state %q", row.ID, row.ProfileState)
	}
	item := projectview.ProjectListItem{
		ProjectID: row.ID, ChainID: row.ChainID, Name: row.Name, Symbol: row.Symbol,
		Contract: bytesToAddress(row.Contract), CodeHash: bytesToHash(row.CodeHash), TxHash: bytesToHash(row.TxHash),
		BlockNumber: blockNumber, BlockTime: blockTime, CreatedAt: timeValue(row.CreatedAt),
		CollectionStatus: collectionStatus, CollectionTotalCount: taskCount,
		CollectionTerminalCount: terminalCount, CollectionSucceededCount: succeededCount,
		CollectionFailedCount: failedCount, ProfileState: profileState,
	}
	if !row.ProfileProjectID.Valid {
		return item, nil
	}
	if row.ProfileProjectID.Int64 != row.ID || !row.CompletenessStatus.Valid || !row.ProfileBuiltAt.Valid {
		return projectview.ProjectListItem{}, fmt.Errorf("project %d has an incomplete profile projection", row.ID)
	}
	item.CompletenessStatus = profile.CompletenessStatus(row.CompletenessStatus.String)
	if item.CompletenessStatus != profile.CompletenessStatusComplete && item.CompletenessStatus != profile.CompletenessStatusIncomplete {
		return projectview.ProjectListItem{}, fmt.Errorf("project %d has unsupported completeness status %q", row.ID, row.CompletenessStatus.String)
	}
	item.ProfileBuiltAt = timeValue(row.ProfileBuiltAt)
	market, err := mapProjectListMarket(row)
	if err != nil {
		return projectview.ProjectListItem{}, fmt.Errorf("map project %d market summary: %w", row.ID, err)
	}
	item.Market = market
	wrapped, err := mapProjectListPair(
		profile.PairKindWrappedNative, bytesToAddress(row.WethPair), row.WethPairIsCreated,
		row.WethPairTokenBalanceExceedsTotalSupply, row.WethPairLpMinimumSupplyOnly,
		row.WethPairFixedFeeAddressLpShareGte90Percent, row.WethPairQuoteUsdtValueInt,
		row.WethPairReserveUpdatedAt,
	)
	if err != nil {
		return projectview.ProjectListItem{}, fmt.Errorf("map project %d wrapped-native pair summary: %w", row.ID, err)
	}
	item.WrappedNativePair = wrapped
	usdt, err := mapProjectListPair(
		profile.PairKindUSDT, bytesToAddress(row.UsdtPair), row.UsdtPairIsCreated,
		row.UsdtPairTokenBalanceExceedsTotalSupply, row.UsdtPairLpMinimumSupplyOnly,
		row.UsdtPairFixedFeeAddressLpShareGte90Percent, row.UsdtPairQuoteUsdtValueInt,
		row.UsdtPairReserveUpdatedAt,
	)
	if err != nil {
		return projectview.ProjectListItem{}, fmt.Errorf("map project %d USDT pair summary: %w", row.ID, err)
	}
	item.USDTPair = usdt
	return item, nil
}

func mapProjectListMarket(row tokensqlc.ListProjectListItemsRow) (*projectview.ProjectMarketSummary, error) {
	currentPrice, err := decimalFromNumeric("current_price_usd", row.CurrentPriceUsd)
	if err != nil {
		return nil, err
	}
	marketCap, err := decimalFromNumeric("market_cap_usd", row.MarketCapUsd)
	if err != nil {
		return nil, err
	}
	fdv, err := decimalFromNumeric("fdv_usd", row.FdvUsd)
	if err != nil {
		return nil, err
	}
	tvl, err := decimalFromNumeric("tvl_usd", row.TvlUsd)
	if err != nil {
		return nil, err
	}
	var holders *int64
	if row.Holders.Valid {
		value := row.Holders.Int64
		holders = &value
	}
	if row.LogoUrl == "" && currentPrice == nil && marketCap == nil && fdv == nil && tvl == nil && holders == nil {
		return nil, nil
	}
	return &projectview.ProjectMarketSummary{
		LogoURL: row.LogoUrl, CurrentPriceUSD: currentPrice, MarketCapUSD: marketCap,
		FDVUSD: fdv, TVLUSD: tvl, Holders: holders,
	}, nil
}

func mapProjectListPair(
	kind profile.PairKind,
	address shared.Address,
	created, balance, minimum, share pgtype.Bool,
	quote pgtype.Numeric,
	reserve pgtype.Int8,
) (*projectview.ProjectPairProfileSummary, error) {
	projection, err := mapPairProjection(string(kind), created, balance, minimum, share, quote, reserve)
	if err != nil || projection == nil {
		return nil, err
	}
	return &projectview.ProjectPairProfileSummary{
		Kind: kind, Address: address, IsCreated: projection.IsCreated,
		QuoteUSDTValueInt: projection.QuoteUsdtValueInt, ReserveUpdatedAt: projection.ReserveUpdatedAt,
		PairTokenBalanceExceedsTotalSupply: projection.PairTokenBalanceExceedsTotalSupply,
		LPMinimumSupplyOnly:                projection.LPMinimumSupplyOnly,
		FixedFeeAddressLPShareGte90Percent: projection.FixedFeeAddressLPShareGte90Percent,
	}, nil
}

func boundedInt32(field string, value int64) (int32, error) {
	if value < 0 || value > math.MaxInt32 {
		return 0, fmt.Errorf("%s is outside int32 range", field)
	}
	return int32(value), nil
}
