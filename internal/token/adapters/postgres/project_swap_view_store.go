package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/catalog"
	"github.com/useryege/athena/internal/token/chainregistry"
	"github.com/useryege/athena/internal/token/projectview"
	"github.com/useryege/athena/internal/token/shared"
	"github.com/useryege/athena/internal/token/swap"
)

const swapPriceDecimalPlaces = 100

func (repository *ProjectViewRepository) GetProjectSwapActivity(
	ctx context.Context,
	projectID int64,
) (*projectview.SwapActivity, error) {
	if repository == nil || repository.pool == nil {
		return nil, fmt.Errorf("token postgres repository is not configured")
	}
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return nil, fmt.Errorf("begin project Swap activity snapshot: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := tokensqlc.New(tx)
	projectRow, err := queries.GetProject(ctx, projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project for Swap activity: %w", err)
	}
	project, err := mapProject(projectRow)
	if err != nil {
		return nil, fmt.Errorf("map project for Swap activity: %w", err)
	}
	chainAssets, ok := chainregistry.FixedAssets(project.ChainID)
	if !ok {
		return nil, fmt.Errorf("project %d uses unsupported chain %d", project.ID, project.ChainID)
	}

	pairRows, err := queries.ListProjectSwapActivityPairs(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project Swap activity pairs: %w", err)
	}
	if len(pairRows) != 2 {
		return nil, fmt.Errorf("project %d must have exactly two Swap pairs, got %d", project.ID, len(pairRows))
	}

	activity := &projectview.SwapActivity{
		ProjectID: project.ID,
		ChainID:   project.ChainID,
		Pairs:     make([]projectview.SwapPairActivity, 0, len(pairRows)),
	}
	seenKinds := make(map[swap.PairKind]struct{}, len(pairRows))
	for _, pairRow := range pairRows {
		pair, err := mapProjectSwapPair(pairRow)
		if err != nil {
			return nil, fmt.Errorf("map project Swap pair %d: %w", pairRow.ID, err)
		}
		if _, exists := seenKinds[pair.Kind]; exists {
			return nil, fmt.Errorf("project %d has duplicate Swap pair kind %q", project.ID, pair.Kind)
		}
		seenKinds[pair.Kind] = struct{}{}

		baseAsset, quoteAsset, baseTokenIndex, expectedPairAddress, err := projectSwapAssets(project, chainAssets, pair.Kind)
		if err != nil {
			return nil, err
		}
		if pair.ProjectID != project.ID || pair.ChainID != project.ChainID {
			return nil, fmt.Errorf("Swap pair %d does not belong to project %d and chain %d", pair.ID, project.ID, project.ChainID)
		}
		if pair.StartBlockNumber != project.BlockNumber || pair.StartBlockTime != project.BlockTime {
			return nil, fmt.Errorf("Swap pair %d does not start at project %d deployment block", pair.ID, project.ID)
		}
		if pair.Address.IsZero() || pair.Address != expectedPairAddress {
			return nil, fmt.Errorf("Swap pair %d address does not match project %d %s pair", pair.ID, project.ID, pair.Kind)
		}

		totalsRow, err := queries.GetProjectSwapActivityPairTotals(ctx, tokensqlc.GetProjectSwapActivityPairTotalsParams{
			BaseIsToken0:      baseTokenIndex == 0,
			ProjectSwapPairID: pair.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("get project Swap pair %d totals: %w", pair.ID, err)
		}
		totals, err := mapProjectSwapActivityTotals(totalsRow)
		if err != nil {
			return nil, fmt.Errorf("map project Swap pair %d totals: %w", pair.ID, err)
		}

		blockRows, err := queries.ListProjectSwapActivityBlocks(ctx, tokensqlc.ListProjectSwapActivityBlocksParams{
			BaseIsToken0:      baseTokenIndex == 0,
			ProjectSwapPairID: pair.ID,
			BaseDecimals:      int32(baseAsset.Decimals),
			QuoteDecimals:     int32(quoteAsset.Decimals),
		})
		if err != nil {
			return nil, fmt.Errorf("list project Swap pair %d activity blocks: %w", pair.ID, err)
		}
		blocks := make([]projectview.SwapBlockActivity, 0, len(blockRows))
		for _, blockRow := range blockRows {
			block, err := mapProjectSwapActivityBlock(blockRow)
			if err != nil {
				return nil, fmt.Errorf("map project Swap pair %d activity block: %w", pair.ID, err)
			}
			blocks = append(blocks, block)
		}

		item := projectview.SwapPairActivity{
			ID:                      pair.ID,
			Kind:                    pair.Kind,
			Address:                 pair.Address,
			Status:                  pair.Status,
			SwapBlockCount:          pair.SwapBlockCount,
			TargetSwapBlockCount:    swap.TargetSwapBlockCount,
			BaseAsset:               baseAsset,
			QuoteAsset:              quoteAsset,
			BaseTokenIndex:          baseTokenIndex,
			StartBlockNumber:        pair.StartBlockNumber,
			StartBlockTime:          pair.StartBlockTime,
			FirstSwapBlockNumber:    pair.FirstSwapBlockNumber,
			FirstSwapBlockTime:      pair.FirstSwapBlockTime,
			LastSwapBlockNumber:     pair.LastSwapBlockNumber,
			LastSwapBlockTime:       pair.LastSwapBlockTime,
			AbsoluteExpiryBlockTime: pair.AbsoluteExpiryBlockTime,
			NextExpiryBlockTime:     pair.NextExpiryBlockTime,
			CompletedBlockNumber:    pair.CompletedBlockNumber,
			CompletedBlockTime:      pair.CompletedBlockTime,
			ExpiredBlockNumber:      pair.ExpiredBlockNumber,
			ExpiredBlockTime:        pair.ExpiredBlockTime,
			ExpiredReason:           pair.ExpiredReason,
			Totals:                  totals,
			Blocks:                  blocks,
		}
		if err := validateProjectSwapPairActivity(item); err != nil {
			return nil, fmt.Errorf("validate project Swap pair %d activity: %w", pair.ID, err)
		}
		activity.Pairs = append(activity.Pairs, item)
	}
	if _, ok := seenKinds[swap.PairKindWETH]; !ok {
		return nil, fmt.Errorf("project %d has no WETH Swap pair", project.ID)
	}
	if _, ok := seenKinds[swap.PairKindUSDT]; !ok {
		return nil, fmt.Errorf("project %d has no USDT Swap pair", project.ID)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit project Swap activity snapshot: %w", err)
	}
	return activity, nil
}

func (repository *ProjectViewRepository) ListProjectSwapEventsPage(
	ctx context.Context,
	projectID int64,
	pairKind swap.PairKind,
	blockNumber uint64,
	page, pageSize int32,
) (*projectview.SwapEventPage, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	projectRow, err := queries.GetProject(ctx, projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		page, pageSize, _ = normalizePage(page, pageSize)
		return &projectview.SwapEventPage{Items: []projectview.SwapEvent{}, Page: page, PageSize: pageSize}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project for Swap events: %w", err)
	}
	project, err := mapProject(projectRow)
	if err != nil {
		return nil, fmt.Errorf("map project for Swap events: %w", err)
	}
	chainAssets, ok := chainregistry.FixedAssets(project.ChainID)
	if !ok {
		return nil, fmt.Errorf("project %d uses unsupported chain %d", project.ID, project.ChainID)
	}
	baseAsset, quoteAsset, baseTokenIndex, _, err := projectSwapAssets(project, chainAssets, pairKind)
	if err != nil {
		return nil, err
	}
	blockNumberValue, err := uint64ToInt64("block_number", blockNumber)
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	countParams := tokensqlc.CountProjectSwapEventsParams{
		ProjectID:   projectID,
		PairKind:    string(pairKind),
		BlockNumber: blockNumberValue,
	}
	total, err := queries.CountProjectSwapEvents(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count project Swap events: %w", err)
	}
	rows, err := queries.ListProjectSwapEvents(ctx, tokensqlc.ListProjectSwapEventsParams{
		ProjectID:   projectID,
		PairKind:    string(pairKind),
		BlockNumber: blockNumberValue,
		PageLimit:   pageSize,
		PageOffset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list project Swap events: %w", err)
	}
	items := make([]projectview.SwapEvent, 0, len(rows))
	for _, row := range rows {
		item, err := mapProjectSwapEvent(row, baseTokenIndex == 0, baseAsset.Decimals, quoteAsset.Decimals)
		if err != nil {
			return nil, fmt.Errorf("map project Swap event %d: %w", row.ID, err)
		}
		items = append(items, item)
	}
	return &projectview.SwapEventPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func projectSwapAssets(
	project *catalog.Project,
	chainAssets chainregistry.ChainAssets,
	pairKind swap.PairKind,
) (projectview.SwapAsset, projectview.SwapAsset, uint8, shared.Address, error) {
	if project == nil {
		return projectview.SwapAsset{}, projectview.SwapAsset{}, 0, shared.Address{}, fmt.Errorf("project is required")
	}
	baseAsset := projectview.SwapAsset{
		Address:  project.Contract,
		Symbol:   project.Symbol,
		Decimals: project.Decimals,
	}
	var quoteMetadata chainregistry.AssetMetadata
	var expectedPairAddress shared.Address
	switch pairKind {
	case swap.PairKindWETH:
		quoteMetadata = chainAssets.WrappedNative
		expectedPairAddress = project.WethPair
	case swap.PairKindUSDT:
		quoteMetadata = chainAssets.Stable
		expectedPairAddress = project.UsdtPair
	default:
		return projectview.SwapAsset{}, projectview.SwapAsset{}, 0, shared.Address{}, fmt.Errorf("project %d has unsupported Swap pair kind %q", project.ID, pairKind)
	}
	quoteAsset := projectview.SwapAsset{
		Address:  quoteMetadata.Address,
		Symbol:   quoteMetadata.Symbol,
		Decimals: quoteMetadata.Decimals,
	}
	if baseAsset.Address.IsZero() || quoteAsset.Address.IsZero() {
		return projectview.SwapAsset{}, projectview.SwapAsset{}, 0, shared.Address{}, fmt.Errorf("project %d Swap assets must be nonzero", project.ID)
	}
	if baseAsset.Address == quoteAsset.Address {
		return projectview.SwapAsset{}, projectview.SwapAsset{}, 0, shared.Address{}, fmt.Errorf("project %d base and %s quote assets are identical", project.ID, pairKind)
	}
	baseTokenIndex := uint8(1)
	if bytes.Compare(baseAsset.Address[:], quoteAsset.Address[:]) < 0 {
		baseTokenIndex = 0
	}
	return baseAsset, quoteAsset, baseTokenIndex, expectedPairAddress, nil
}

func mapProjectSwapActivityTotals(
	row tokensqlc.GetProjectSwapActivityPairTotalsRow,
) (projectview.SwapActivityTotals, error) {
	totals := projectview.SwapActivityTotals{
		EventCount:             row.EventCount,
		TransactionCount:       row.TransactionCount,
		TransactionOriginCount: row.TransactionOriginCount,
		BuyEventCount:          row.BuyEventCount,
		SellEventCount:         row.SellEventCount,
		ComplexEventCount:      row.ComplexEventCount,
		Flow: projectview.SwapFlow{
			BaseIn:   row.BaseIn,
			BaseOut:  row.BaseOut,
			QuoteIn:  row.QuoteIn,
			QuoteOut: row.QuoteOut,
		},
		BuyQuoteVolume:  row.BuyQuoteVolume,
		SellQuoteVolume: row.SellQuoteVolume,
	}
	if err := validateProjectSwapActivityTotals(totals); err != nil {
		return projectview.SwapActivityTotals{}, err
	}
	return totals, nil
}

func mapProjectSwapActivityBlock(
	row tokensqlc.ListProjectSwapActivityBlocksRow,
) (projectview.SwapBlockActivity, error) {
	if row.SampleIndex < 1 || row.SampleIndex > int32(swap.TargetSwapBlockCount) {
		return projectview.SwapBlockActivity{}, fmt.Errorf("sample_index %d is outside supported range", row.SampleIndex)
	}
	blockNumber, err := int64ToUint64("block_number", row.BlockNumber)
	if err != nil {
		return projectview.SwapBlockActivity{}, err
	}
	blockTime, err := int64ToUint64("block_time", row.BlockTime)
	if err != nil {
		return projectview.SwapBlockActivity{}, err
	}
	var previousBlockGap *uint64
	var previousTimeGapSeconds *uint64
	if row.SampleIndex > 1 {
		previousBlockGap, err = uint64Pointer(row.PreviousBlockGap)
		if err != nil {
			return projectview.SwapBlockActivity{}, fmt.Errorf("previous_block_gap: %w", err)
		}
		previousTimeGapSeconds, err = uint64Pointer(row.PreviousTimeGapSeconds)
		if err != nil {
			return projectview.SwapBlockActivity{}, fmt.Errorf("previous_time_gap_seconds: %w", err)
		}
	}
	totals := projectview.SwapActivityTotals{
		EventCount:             row.EventCount,
		TransactionCount:       row.TransactionCount,
		TransactionOriginCount: row.TransactionOriginCount,
		BuyEventCount:          row.BuyEventCount,
		SellEventCount:         row.SellEventCount,
		ComplexEventCount:      row.ComplexEventCount,
		Flow: projectview.SwapFlow{
			BaseIn:   row.BaseIn,
			BaseOut:  row.BaseOut,
			QuoteIn:  row.QuoteIn,
			QuoteOut: row.QuoteOut,
		},
		BuyQuoteVolume:  row.BuyQuoteVolume,
		SellQuoteVolume: row.SellQuoteVolume,
	}
	if err := validateProjectSwapActivityTotals(totals); err != nil {
		return projectview.SwapBlockActivity{}, err
	}
	block := projectview.SwapBlockActivity{
		SampleIndex:            uint16(row.SampleIndex),
		BlockNumber:            blockNumber,
		BlockTime:              blockTime,
		PreviousBlockGap:       previousBlockGap,
		PreviousTimeGapSeconds: previousTimeGapSeconds,
		Totals:                 totals,
		Price: projectview.SwapPriceSummary{
			Open:  optionalString(row.OpenPrice),
			High:  optionalString(row.HighPrice),
			Low:   optionalString(row.LowPrice),
			Close: optionalString(row.ClosePrice),
			VWAP:  optionalString(row.VwapPrice),
		},
	}
	if err := validateSwapPriceSummary(block.Price, totals.BuyEventCount+totals.SellEventCount); err != nil {
		return projectview.SwapBlockActivity{}, err
	}
	return block, nil
}

func mapProjectSwapEvent(
	row tokensqlc.ListProjectSwapEventsRow,
	baseIsToken0 bool,
	baseDecimals, quoteDecimals uint8,
) (projectview.SwapEvent, error) {
	transactionIndex, err := int64ToUint64("transaction_index", row.TransactionIndex)
	if err != nil {
		return projectview.SwapEvent{}, err
	}
	logIndex, err := int64ToUint64("log_index", row.LogIndex)
	if err != nil {
		return projectview.SwapEvent{}, err
	}
	amount0In, err := parseSwapAmount("amount0_in", row.Amount0In)
	if err != nil {
		return projectview.SwapEvent{}, err
	}
	amount1In, err := parseSwapAmount("amount1_in", row.Amount1In)
	if err != nil {
		return projectview.SwapEvent{}, err
	}
	amount0Out, err := parseSwapAmount("amount0_out", row.Amount0Out)
	if err != nil {
		return projectview.SwapEvent{}, err
	}
	amount1Out, err := parseSwapAmount("amount1_out", row.Amount1Out)
	if err != nil {
		return projectview.SwapEvent{}, err
	}
	flow, direction, effectivePrice := projectSwapSemantics(
		amount0In,
		amount1In,
		amount0Out,
		amount1Out,
		baseIsToken0,
		baseDecimals,
		quoteDecimals,
	)
	return projectview.SwapEvent{
		ID:               row.ID,
		TransactionHash:  bytesToHash(row.TransactionHash),
		TransactionIndex: transactionIndex,
		LogIndex:         logIndex,
		TxFrom:           bytesToAddress(row.TxFrom),
		Sender:           bytesToAddress(row.Sender),
		ToAddress:        bytesToAddress(row.ToAddress),
		Amount0In:        amount0In.String(),
		Amount1In:        amount1In.String(),
		Amount0Out:       amount0Out.String(),
		Amount1Out:       amount1Out.String(),
		Flow:             flow,
		Direction:        direction,
		EffectivePrice:   effectivePrice,
	}, nil
}

func projectSwapSemantics(
	amount0In, amount1In, amount0Out, amount1Out *big.Int,
	baseIsToken0 bool,
	baseDecimals, quoteDecimals uint8,
) (projectview.SwapFlow, projectview.SwapDirection, *string) {
	baseIn, baseOut := amount1In, amount1Out
	quoteIn, quoteOut := amount0In, amount0Out
	if baseIsToken0 {
		baseIn, baseOut = amount0In, amount0Out
		quoteIn, quoteOut = amount1In, amount1Out
	}
	flow := projectview.SwapFlow{
		BaseIn:   baseIn.String(),
		BaseOut:  baseOut.String(),
		QuoteIn:  quoteIn.String(),
		QuoteOut: quoteOut.String(),
	}
	switch {
	case quoteIn.Sign() > 0 && baseOut.Sign() > 0 && baseIn.Sign() == 0 && quoteOut.Sign() == 0:
		price := formatSwapPrice(quoteIn, baseOut, baseDecimals, quoteDecimals)
		return flow, projectview.SwapDirectionBuy, &price
	case baseIn.Sign() > 0 && quoteOut.Sign() > 0 && quoteIn.Sign() == 0 && baseOut.Sign() == 0:
		price := formatSwapPrice(quoteOut, baseIn, baseDecimals, quoteDecimals)
		return flow, projectview.SwapDirectionSell, &price
	default:
		return flow, projectview.SwapDirectionComplex, nil
	}
}

func formatSwapPrice(quoteRaw, baseRaw *big.Int, baseDecimals, quoteDecimals uint8) string {
	numerator := new(big.Int).Mul(
		new(big.Int).Set(quoteRaw),
		new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(baseDecimals)), nil),
	)
	denominator := new(big.Int).Mul(
		new(big.Int).Set(baseRaw),
		new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(quoteDecimals)), nil),
	)
	return formatPositiveDecimalRatio(numerator, denominator, swapPriceDecimalPlaces)
}

func formatPositiveDecimalRatio(numerator, denominator *big.Int, decimalPlaces int) string {
	if numerator == nil || numerator.Sign() == 0 {
		return "0"
	}
	if denominator == nil || denominator.Sign() <= 0 {
		return ""
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimalPlaces)), nil)
	scaledNumerator := new(big.Int).Mul(new(big.Int).Set(numerator), scale)
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(scaledNumerator, denominator, remainder)
	if new(big.Int).Lsh(remainder, 1).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	digits := quotient.String()
	if decimalPlaces == 0 {
		return digits
	}
	if len(digits) <= decimalPlaces {
		digits = strings.Repeat("0", decimalPlaces+1-len(digits)) + digits
	}
	whole := digits[:len(digits)-decimalPlaces]
	fraction := digits[len(digits)-decimalPlaces:]
	for len(fraction) > 0 && fraction[len(fraction)-1] == '0' {
		fraction = fraction[:len(fraction)-1]
	}
	if fraction == "" {
		return whole
	}
	return whole + "." + fraction
}

func validateProjectSwapPairActivity(pair projectview.SwapPairActivity) error {
	if pair.SwapBlockCount != uint16(len(pair.Blocks)) {
		return fmt.Errorf("swap_block_count %d does not match %d stored blocks", pair.SwapBlockCount, len(pair.Blocks))
	}
	if pair.TargetSwapBlockCount != swap.TargetSwapBlockCount {
		return fmt.Errorf("target Swap block count is %d", pair.TargetSwapBlockCount)
	}
	if pair.AbsoluteExpiryBlockTime <= pair.StartBlockTime {
		return fmt.Errorf("absolute expiry must follow start time")
	}

	sum := zeroProjectSwapActivityTotals()
	for index, block := range pair.Blocks {
		expectedSampleIndex := uint16(index + 1)
		if block.SampleIndex != expectedSampleIndex {
			return fmt.Errorf("sample index %d must be %d", block.SampleIndex, expectedSampleIndex)
		}
		if block.Totals.EventCount <= 0 {
			return fmt.Errorf("sample %d has no Swap events", block.SampleIndex)
		}
		if index == 0 {
			if block.PreviousBlockGap != nil || block.PreviousTimeGapSeconds != nil {
				return fmt.Errorf("first sample has a previous gap")
			}
		} else {
			previous := pair.Blocks[index-1]
			if block.BlockNumber <= previous.BlockNumber {
				return fmt.Errorf("sample %d block number is not increasing", block.SampleIndex)
			}
			if block.BlockTime < previous.BlockTime {
				return fmt.Errorf("sample %d block time is decreasing", block.SampleIndex)
			}
			if block.PreviousBlockGap == nil || *block.PreviousBlockGap != block.BlockNumber-previous.BlockNumber {
				return fmt.Errorf("sample %d block gap is inconsistent", block.SampleIndex)
			}
			if block.PreviousTimeGapSeconds == nil || *block.PreviousTimeGapSeconds != block.BlockTime-previous.BlockTime {
				return fmt.Errorf("sample %d time gap is inconsistent", block.SampleIndex)
			}
		}
		var err error
		sum, err = addProjectSwapActivityTotals(sum, block.Totals)
		if err != nil {
			return fmt.Errorf("sum sample %d totals: %w", block.SampleIndex, err)
		}
	}
	if err := compareProjectSwapActivityTotals(pair.Totals, sum); err != nil {
		return err
	}

	if pair.SwapBlockCount == 0 {
		if pair.FirstSwapBlockNumber != nil || pair.FirstSwapBlockTime != nil ||
			pair.LastSwapBlockNumber != nil || pair.LastSwapBlockTime != nil {
			return fmt.Errorf("empty pair has first or last Swap position")
		}
	} else {
		first := pair.Blocks[0]
		last := pair.Blocks[len(pair.Blocks)-1]
		if first.BlockNumber < pair.StartBlockNumber || first.BlockTime < pair.StartBlockTime {
			return fmt.Errorf("first Swap sample precedes pair start")
		}
		if !uint64PointerEquals(pair.FirstSwapBlockNumber, first.BlockNumber) ||
			!uint64PointerEquals(pair.FirstSwapBlockTime, first.BlockTime) {
			return fmt.Errorf("first Swap position does not match first sample")
		}
		if !uint64PointerEquals(pair.LastSwapBlockNumber, last.BlockNumber) ||
			!uint64PointerEquals(pair.LastSwapBlockTime, last.BlockTime) {
			return fmt.Errorf("last Swap position does not match last sample")
		}
	}

	switch pair.Status {
	case swap.PairStatusCollecting:
		if pair.SwapBlockCount >= swap.TargetSwapBlockCount || pair.NextExpiryBlockTime == nil ||
			pair.CompletedBlockNumber != nil || pair.CompletedBlockTime != nil ||
			pair.ExpiredBlockNumber != nil || pair.ExpiredBlockTime != nil || pair.ExpiredReason != "" {
			return fmt.Errorf("collecting pair has inconsistent state")
		}
		if *pair.NextExpiryBlockTime <= pair.StartBlockTime ||
			*pair.NextExpiryBlockTime > pair.AbsoluteExpiryBlockTime {
			return fmt.Errorf("collecting pair has invalid next expiry")
		}
	case swap.PairStatusCompleted:
		if pair.SwapBlockCount != swap.TargetSwapBlockCount || pair.NextExpiryBlockTime != nil ||
			!sameUint64Pointers(pair.CompletedBlockNumber, pair.LastSwapBlockNumber) ||
			!sameUint64Pointers(pair.CompletedBlockTime, pair.LastSwapBlockTime) ||
			pair.ExpiredBlockNumber != nil || pair.ExpiredBlockTime != nil || pair.ExpiredReason != "" {
			return fmt.Errorf("completed pair has inconsistent state")
		}
	case swap.PairStatusExpired:
		if pair.SwapBlockCount >= swap.TargetSwapBlockCount || pair.NextExpiryBlockTime != nil ||
			pair.CompletedBlockNumber != nil || pair.CompletedBlockTime != nil ||
			pair.ExpiredBlockNumber == nil || pair.ExpiredBlockTime == nil {
			return fmt.Errorf("expired pair has inconsistent state")
		}
		switch pair.ExpiredReason {
		case swap.ExpiredReasonNoSwap:
			if pair.SwapBlockCount != 0 {
				return fmt.Errorf("no_swap pair has observed Swap blocks")
			}
		case swap.ExpiredReasonInactive:
			if pair.SwapBlockCount == 0 {
				return fmt.Errorf("inactive pair has no observed Swap blocks")
			}
		case swap.ExpiredReasonMaxDuration:
		default:
			return fmt.Errorf("expired pair has unsupported reason %q", pair.ExpiredReason)
		}
		minimumBlockNumber := pair.StartBlockNumber
		minimumBlockTime := pair.StartBlockTime
		if pair.LastSwapBlockNumber != nil {
			minimumBlockNumber = *pair.LastSwapBlockNumber
			minimumBlockTime = *pair.LastSwapBlockTime
		}
		if *pair.ExpiredBlockNumber < minimumBlockNumber || *pair.ExpiredBlockTime < minimumBlockTime {
			return fmt.Errorf("expired pair terminal position precedes its observations")
		}
		if pair.ExpiredReason == swap.ExpiredReasonMaxDuration &&
			*pair.ExpiredBlockTime < pair.AbsoluteExpiryBlockTime {
			return fmt.Errorf("max_duration pair expired before its absolute deadline")
		}
	default:
		return fmt.Errorf("unsupported Pair status %q", pair.Status)
	}
	return nil
}

func validateProjectSwapActivityTotals(totals projectview.SwapActivityTotals) error {
	if totals.EventCount < 0 || totals.TransactionCount < 0 || totals.TransactionOriginCount < 0 ||
		totals.BuyEventCount < 0 || totals.SellEventCount < 0 || totals.ComplexEventCount < 0 {
		return fmt.Errorf("Swap counts must be nonnegative")
	}
	if totals.TransactionCount > totals.EventCount {
		return fmt.Errorf("transaction count exceeds event count")
	}
	if totals.TransactionOriginCount > totals.TransactionCount {
		return fmt.Errorf("transaction origin count exceeds transaction count")
	}
	if totals.BuyEventCount+totals.SellEventCount+totals.ComplexEventCount != totals.EventCount {
		return fmt.Errorf("direction counts do not sum to event count")
	}
	for field, value := range map[string]string{
		"base_in":           totals.Flow.BaseIn,
		"base_out":          totals.Flow.BaseOut,
		"quote_in":          totals.Flow.QuoteIn,
		"quote_out":         totals.Flow.QuoteOut,
		"buy_quote_volume":  totals.BuyQuoteVolume,
		"sell_quote_volume": totals.SellQuoteVolume,
	} {
		if _, err := parseSwapAmount(field, value); err != nil {
			return err
		}
	}
	return nil
}

func validateSwapPriceSummary(summary projectview.SwapPriceSummary, simpleEventCount int64) error {
	values := []*string{summary.Open, summary.High, summary.Low, summary.Close, summary.VWAP}
	if simpleEventCount == 0 {
		for _, value := range values {
			if value != nil {
				return fmt.Errorf("complex-only block has an execution price")
			}
		}
		return nil
	}
	prices := make([]*big.Rat, 0, len(values))
	for _, value := range values {
		if value == nil {
			return fmt.Errorf("block with simple Swap events has an incomplete price summary")
		}
		price, ok := new(big.Rat).SetString(*value)
		if !ok || price.Sign() <= 0 {
			return fmt.Errorf("execution price %q is not a positive decimal", *value)
		}
		prices = append(prices, price)
	}
	open, high, low, close := prices[0], prices[1], prices[2], prices[3]
	if high.Cmp(low) < 0 || open.Cmp(low) < 0 || open.Cmp(high) > 0 ||
		close.Cmp(low) < 0 || close.Cmp(high) > 0 {
		return fmt.Errorf("execution price OHLC values are inconsistent")
	}
	return nil
}

func zeroProjectSwapActivityTotals() projectview.SwapActivityTotals {
	return projectview.SwapActivityTotals{
		Flow: projectview.SwapFlow{
			BaseIn:   "0",
			BaseOut:  "0",
			QuoteIn:  "0",
			QuoteOut: "0",
		},
		BuyQuoteVolume:  "0",
		SellQuoteVolume: "0",
	}
}

func addProjectSwapActivityTotals(
	left, right projectview.SwapActivityTotals,
) (projectview.SwapActivityTotals, error) {
	result := projectview.SwapActivityTotals{
		EventCount:             left.EventCount + right.EventCount,
		TransactionCount:       left.TransactionCount + right.TransactionCount,
		TransactionOriginCount: left.TransactionOriginCount,
		BuyEventCount:          left.BuyEventCount + right.BuyEventCount,
		SellEventCount:         left.SellEventCount + right.SellEventCount,
		ComplexEventCount:      left.ComplexEventCount + right.ComplexEventCount,
	}
	var err error
	result.Flow.BaseIn, err = addSwapAmountStrings(left.Flow.BaseIn, right.Flow.BaseIn)
	if err != nil {
		return projectview.SwapActivityTotals{}, err
	}
	result.Flow.BaseOut, err = addSwapAmountStrings(left.Flow.BaseOut, right.Flow.BaseOut)
	if err != nil {
		return projectview.SwapActivityTotals{}, err
	}
	result.Flow.QuoteIn, err = addSwapAmountStrings(left.Flow.QuoteIn, right.Flow.QuoteIn)
	if err != nil {
		return projectview.SwapActivityTotals{}, err
	}
	result.Flow.QuoteOut, err = addSwapAmountStrings(left.Flow.QuoteOut, right.Flow.QuoteOut)
	if err != nil {
		return projectview.SwapActivityTotals{}, err
	}
	result.BuyQuoteVolume, err = addSwapAmountStrings(left.BuyQuoteVolume, right.BuyQuoteVolume)
	if err != nil {
		return projectview.SwapActivityTotals{}, err
	}
	result.SellQuoteVolume, err = addSwapAmountStrings(left.SellQuoteVolume, right.SellQuoteVolume)
	if err != nil {
		return projectview.SwapActivityTotals{}, err
	}
	return result, nil
}

func compareProjectSwapActivityTotals(expected, actual projectview.SwapActivityTotals) error {
	if expected.EventCount != actual.EventCount ||
		expected.TransactionCount != actual.TransactionCount ||
		expected.BuyEventCount != actual.BuyEventCount ||
		expected.SellEventCount != actual.SellEventCount ||
		expected.ComplexEventCount != actual.ComplexEventCount ||
		expected.Flow != actual.Flow ||
		expected.BuyQuoteVolume != actual.BuyQuoteVolume ||
		expected.SellQuoteVolume != actual.SellQuoteVolume {
		return fmt.Errorf("pair totals do not match block totals")
	}
	return nil
}

func parseSwapAmount(field, value string) (*big.Int, error) {
	amount, ok := new(big.Int).SetString(value, 10)
	if !ok || amount.Sign() < 0 {
		return nil, fmt.Errorf("%s is not a nonnegative base-10 integer", field)
	}
	return amount, nil
}

func addSwapAmountStrings(left, right string) (string, error) {
	leftAmount, err := parseSwapAmount("left Swap amount", left)
	if err != nil {
		return "", err
	}
	rightAmount, err := parseSwapAmount("right Swap amount", right)
	if err != nil {
		return "", err
	}
	return new(big.Int).Add(leftAmount, rightAmount).String(), nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func uint64Pointer(value int64) (*uint64, error) {
	mapped, err := int64ToUint64("value", value)
	if err != nil {
		return nil, err
	}
	return &mapped, nil
}

func uint64PointerEquals(pointer *uint64, value uint64) bool {
	return pointer != nil && *pointer == value
}

func sameUint64Pointers(left, right *uint64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
