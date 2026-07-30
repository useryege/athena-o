package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/shared"
	"github.com/useryege/athena/internal/token/swap"
	swapapp "github.com/useryege/athena/internal/token/swap/application"
)

type SwapRepository struct{ *baseRepository }

func NewSwapRepository(connection *Connection) *SwapRepository {
	return &SwapRepository{newBaseRepository(connection)}
}

func (repository *SwapRepository) GetSourceCursor(ctx context.Context, chainID int64) (uint64, error) {
	queries, err := repository.querier()
	if err != nil {
		return 0, err
	}
	cursor, err := queries.GetChainProcessingCursorForSwap(ctx, chainID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("token chain processing checkpoint missing for chain %d", chainID)
	}
	if err != nil {
		return 0, fmt.Errorf("get token chain processing cursor for Swap: %w", err)
	}
	return int64ToUint64("cursor_block_number", cursor)
}

func (repository *SwapRepository) GetProcessingCheckpoint(ctx context.Context, chainID int64) (*swap.ProcessingCheckpoint, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetChainSwapProcessingCheckpoint(ctx, chainID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get chain Swap processing checkpoint: %w", err)
	}
	return mapSwapProcessingCheckpoint(row)
}

func (repository *SwapRepository) SetProcessingStatus(ctx context.Context, chainID int64, status swap.ProcessingStatus) (*swap.ProcessingCheckpoint, error) {
	if status != swap.ProcessingStatusRunning && status != swap.ProcessingStatusStopped {
		return nil, fmt.Errorf("unsupported token Swap processing status %q", status)
	}
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.SetChainSwapProcessingCheckpointStatus(ctx, tokensqlc.SetChainSwapProcessingCheckpointStatusParams{
		ChainID: chainID,
		Status:  string(status),
	})
	if err != nil {
		return nil, fmt.Errorf("set chain Swap processing checkpoint status: %w", err)
	}
	return mapSwapProcessingCheckpoint(row)
}

func (repository *SwapRepository) InitializeProcessingCheckpoint(ctx context.Context, chainID int64, cursorBlockNumber uint64) (*swap.ProcessingCheckpoint, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	cursor, err := uint64ToInt64("cursor_block_number", cursorBlockNumber)
	if err != nil {
		return nil, err
	}
	row, err := queries.InitializeChainSwapProcessingCheckpoint(ctx, tokensqlc.InitializeChainSwapProcessingCheckpointParams{
		ChainID:           chainID,
		CursorBlockNumber: cursor,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize chain Swap processing checkpoint: %w", err)
	}
	return mapSwapProcessingCheckpoint(row)
}

func (repository *SwapRepository) FindNextCollectingPairStartBlock(ctx context.Context, chainID int64, sourceCursorBlockNumber uint64) (*uint64, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	sourceCursor, err := uint64ToInt64("source_cursor_block_number", sourceCursorBlockNumber)
	if err != nil {
		return nil, err
	}
	blockNumber, err := queries.FindNextCollectingProjectSwapPairStartBlock(ctx, tokensqlc.FindNextCollectingProjectSwapPairStartBlockParams{
		ChainID:                 chainID,
		SourceCursorBlockNumber: sourceCursor,
	})
	if err != nil {
		return nil, fmt.Errorf("find next collecting project Swap pair start block: %w", err)
	}
	if blockNumber == 0 {
		return nil, nil
	}
	value, err := int64ToUint64("start_block_number", blockNumber)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (repository *SwapRepository) ListCollectingPairsByAddresses(ctx context.Context, chainID int64, blockNumber uint64, addresses []shared.Address) ([]swap.Pair, error) {
	if len(addresses) == 0 {
		return nil, nil
	}
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	number, err := uint64ToInt64("block_number", blockNumber)
	if err != nil {
		return nil, err
	}
	pairAddresses := make([][]byte, 0, len(addresses))
	for _, address := range addresses {
		if address.IsZero() {
			continue
		}
		pairAddresses = append(pairAddresses, address.Bytes())
	}
	if len(pairAddresses) == 0 {
		return nil, nil
	}
	rows, err := queries.ListMatchingCollectingProjectSwapPairs(ctx, tokensqlc.ListMatchingCollectingProjectSwapPairsParams{
		ChainID:       chainID,
		BlockNumber:   number,
		PairAddresses: pairAddresses,
	})
	if err != nil {
		return nil, fmt.Errorf("list matching collecting project Swap pairs: %w", err)
	}
	return mapProjectSwapPairs(rows)
}

func (repository *SwapRepository) AdvanceProcessingCheckpoint(ctx context.Context, chainID int64, expectedCursorBlockNumber, cursorBlockNumber uint64) (*swap.ProcessingCheckpoint, error) {
	if cursorBlockNumber < expectedCursorBlockNumber {
		return nil, fmt.Errorf("token Swap checkpoint cannot move backward from %d to %d", expectedCursorBlockNumber, cursorBlockNumber)
	}
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	expected, err := uint64ToInt64("expected_cursor_block_number", expectedCursorBlockNumber)
	if err != nil {
		return nil, err
	}
	cursor, err := uint64ToInt64("cursor_block_number", cursorBlockNumber)
	if err != nil {
		return nil, err
	}
	row, err := queries.AdvanceChainSwapProcessingCheckpoint(ctx, tokensqlc.AdvanceChainSwapProcessingCheckpointParams{
		CursorBlockNumber:         cursor,
		ChainID:                   chainID,
		ExpectedCursorBlockNumber: expected,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("chain %d Swap checkpoint is not running at expected cursor %d", chainID, expectedCursorBlockNumber)
	}
	if err != nil {
		return nil, fmt.Errorf("advance chain Swap processing checkpoint: %w", err)
	}
	return mapSwapProcessingCheckpoint(row)
}

func (repository *SwapRepository) CommitProcessedBlock(ctx context.Context, command swapapp.CommitProcessedBlockCommand) (swapapp.CommitProcessedBlockResult, error) {
	if repository == nil || repository.pool == nil {
		return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("token Swap repository is not configured")
	}
	if command.PreviousCheckpoint == math.MaxUint64 || command.BlockNumber != command.PreviousCheckpoint+1 {
		return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("token Swap block %d does not follow checkpoint %d", command.BlockNumber, command.PreviousCheckpoint)
	}
	blockNumber, err := uint64ToInt64("block_number", command.BlockNumber)
	if err != nil {
		return swapapp.CommitProcessedBlockResult{}, err
	}
	blockTime, err := uint64ToInt64("block_time", command.BlockTime)
	if err != nil {
		return swapapp.CommitProcessedBlockResult{}, err
	}
	expectedCursor, err := uint64ToInt64("expected_cursor_block_number", command.PreviousCheckpoint)
	if err != nil {
		return swapapp.CommitProcessedBlockResult{}, err
	}

	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("begin token Swap block transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	queries := tokensqlc.New(tx)
	checkpoint, err := queries.GetChainSwapProcessingCheckpoint(ctx, command.ChainID)
	if err != nil {
		return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("get chain Swap checkpoint for block commit: %w", err)
	}
	if !checkpoint.Initialized || checkpoint.Status != string(swap.ProcessingStatusRunning) || checkpoint.CursorBlockNumber != expectedCursor {
		return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("chain %d Swap checkpoint is not running at expected cursor %d", command.ChainID, command.PreviousCheckpoint)
	}

	result := swapapp.CommitProcessedBlockResult{}
	seenPairs := make(map[int64]struct{}, len(command.PairBlocks))
	for _, pairBlock := range command.PairBlocks {
		if pairBlock.PairID <= 0 || len(pairBlock.Events) == 0 {
			return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("token Swap pair block must contain a valid pair and at least one event")
		}
		if _, exists := seenPairs[pairBlock.PairID]; exists {
			return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("token Swap pair %d appears more than once in block %d", pairBlock.PairID, command.BlockNumber)
		}
		seenPairs[pairBlock.PairID] = struct{}{}
		pair, err := queries.GetProjectSwapPairForUpdate(ctx, pairBlock.PairID)
		if err != nil {
			return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("lock project Swap pair %d: %w", pairBlock.PairID, err)
		}
		if pair.ChainID != command.ChainID || pair.Status != string(swap.PairStatusCollecting) || pair.StartBlockNumber > blockNumber || pair.SwapBlockCount < 0 || pair.SwapBlockCount >= int32(swap.TargetSwapBlockCount) {
			return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("project Swap pair %d is not eligible for block %d", pairBlock.PairID, command.BlockNumber)
		}
		swapBlock, err := queries.CreateProjectSwapBlock(ctx, tokensqlc.CreateProjectSwapBlockParams{
			BlockNumber:       blockNumber,
			BlockTime:         blockTime,
			ProjectSwapPairID: pairBlock.PairID,
		})
		if err != nil {
			return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("create project Swap block for pair %d: %w", pairBlock.PairID, err)
		}
		if swapBlock.SampleIndex != pair.SwapBlockCount+1 {
			return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("project Swap pair %d sample index changed while committing block %d", pairBlock.PairID, command.BlockNumber)
		}
		for _, event := range pairBlock.Events {
			params, err := prepareProjectSwapEvent(swapBlock.ID, pairBlock.PairID, event)
			if err != nil {
				return swapapp.CommitProcessedBlockResult{}, err
			}
			if _, err := queries.CreateProjectSwapEvent(ctx, params); err != nil {
				return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("create project Swap event for pair %d transaction %s log %d: %w", pairBlock.PairID, event.TransactionHash, event.LogIndex, err)
			}
			result.StoredEvents++
		}
		if pair.SwapBlockCount == int32(swap.TargetSwapBlockCount)-1 {
			if _, err := queries.CompleteProjectSwapPair(ctx, tokensqlc.CompleteProjectSwapPairParams{
				BlockNumber:       blockNumber,
				BlockTime:         blockTime,
				ProjectSwapPairID: pairBlock.PairID,
			}); err != nil {
				return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("complete project Swap pair %d: %w", pairBlock.PairID, err)
			}
			result.CompletedPairs++
			continue
		}
		nextExpiry, err := nextSwapExpiryBlockTime(command.BlockTime, pair.AbsoluteExpiryBlockTime)
		if err != nil {
			return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("calculate next expiry for project Swap pair %d: %w", pairBlock.PairID, err)
		}
		if _, err := queries.ObserveProjectSwapPairBlock(ctx, tokensqlc.ObserveProjectSwapPairBlockParams{
			BlockNumber:         blockNumber,
			BlockTime:           blockTime,
			NextExpiryBlockTime: nextExpiry,
			ProjectSwapPairID:   pairBlock.PairID,
		}); err != nil {
			return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("observe project Swap pair %d block: %w", pairBlock.PairID, err)
		}
	}

	duePairs, err := queries.ListDueCollectingProjectSwapPairsForUpdate(ctx, tokensqlc.ListDueCollectingProjectSwapPairsForUpdateParams{
		ChainID:     command.ChainID,
		BlockNumber: blockNumber,
		BlockTime:   blockTime,
	})
	if err != nil {
		return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("list due project Swap pairs: %w", err)
	}
	for _, pair := range duePairs {
		reason := swap.ExpiredReasonInactive
		if command.BlockTime >= uint64(pair.AbsoluteExpiryBlockTime) {
			reason = swap.ExpiredReasonMaxDuration
		} else if pair.SwapBlockCount == 0 {
			reason = swap.ExpiredReasonNoSwap
		}
		affected, err := queries.ExpireProjectSwapPair(ctx, tokensqlc.ExpireProjectSwapPairParams{
			ExpiredBlockNumber: blockNumber,
			ExpiredBlockTime:   blockTime,
			ExpiredReason:      string(reason),
			ProjectSwapPairID:  pair.ID,
		})
		if err != nil {
			return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("expire project Swap pair %d: %w", pair.ID, err)
		}
		if affected != 1 {
			return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("expire project Swap pair %d affected %d rows", pair.ID, affected)
		}
		result.ExpiredPairs++
	}

	if _, err := queries.AdvanceChainSwapProcessingCheckpoint(ctx, tokensqlc.AdvanceChainSwapProcessingCheckpointParams{
		CursorBlockNumber:         blockNumber,
		ChainID:                   command.ChainID,
		ExpectedCursorBlockNumber: expectedCursor,
	}); err != nil {
		return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("advance chain Swap checkpoint for block %d: %w", command.BlockNumber, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return swapapp.CommitProcessedBlockResult{}, fmt.Errorf("commit token Swap block %d transaction: %w", command.BlockNumber, err)
	}
	return result, nil
}

func prepareProjectSwapEvent(swapBlockID, pairID int64, event swap.Event) (tokensqlc.CreateProjectSwapEventParams, error) {
	if event.TransactionHash.IsZero() {
		return tokensqlc.CreateProjectSwapEventParams{}, fmt.Errorf("project Swap pair %d event has an empty transaction hash", pairID)
	}
	transactionIndex, err := uint64ToInt64("transaction_index", event.TransactionIndex)
	if err != nil {
		return tokensqlc.CreateProjectSwapEventParams{}, err
	}
	logIndex, err := uint64ToInt64("log_index", event.LogIndex)
	if err != nil {
		return tokensqlc.CreateProjectSwapEventParams{}, err
	}
	amounts := []*big.Int{event.Amount0In, event.Amount1In, event.Amount0Out, event.Amount1Out}
	for index, amount := range amounts {
		if amount == nil || amount.Sign() < 0 || amount.BitLen() > 256 {
			return tokensqlc.CreateProjectSwapEventParams{}, fmt.Errorf("project Swap pair %d event amount %d is invalid", pairID, index)
		}
	}
	return tokensqlc.CreateProjectSwapEventParams{
		ProjectSwapPairID:  pairID,
		ProjectSwapBlockID: swapBlockID,
		TransactionHash:    event.TransactionHash.Bytes(),
		TransactionIndex:   transactionIndex,
		LogIndex:           logIndex,
		TxFrom:             event.TxFrom.Bytes(),
		Sender:             event.Sender.Bytes(),
		ToAddress:          event.ToAddress.Bytes(),
		Amount0In:          numericFromBigInt(event.Amount0In),
		Amount1In:          numericFromBigInt(event.Amount1In),
		Amount0Out:         numericFromBigInt(event.Amount0Out),
		Amount1Out:         numericFromBigInt(event.Amount1Out),
	}, nil
}

func nextSwapExpiryBlockTime(blockTime uint64, absoluteExpiryBlockTime int64) (int64, error) {
	absolute, err := int64ToUint64("absolute_expiry_block_time", absoluteExpiryBlockTime)
	if err != nil {
		return 0, err
	}
	idleSeconds := uint64(swap.IdleSwapTimeout / time.Second)
	next := blockTime + idleSeconds
	if next < blockTime {
		return 0, fmt.Errorf("idle expiry block time overflow")
	}
	if next > absolute {
		next = absolute
	}
	return uint64ToInt64("next_expiry_block_time", next)
}

func addSwapObservationDuration(blockTime uint64, duration time.Duration) (int64, error) {
	if duration <= 0 || duration%time.Second != 0 {
		return 0, fmt.Errorf("Swap observation duration must be a positive whole number of seconds")
	}
	seconds := uint64(duration / time.Second)
	result := blockTime + seconds
	if result < blockTime {
		return 0, fmt.Errorf("Swap observation block time overflow")
	}
	return uint64ToInt64("swap_expiry_block_time", result)
}

func mapSwapProcessingCheckpoint(row tokensqlc.ChainSwapProcessingCheckpoint) (*swap.ProcessingCheckpoint, error) {
	cursor, err := int64ToUint64("cursor_block_number", row.CursorBlockNumber)
	if err != nil {
		return nil, err
	}
	status := swap.ProcessingStatus(row.Status)
	if status != swap.ProcessingStatusRunning && status != swap.ProcessingStatusStopped {
		return nil, fmt.Errorf("unsupported token Swap processing status %q", row.Status)
	}
	return &swap.ProcessingCheckpoint{
		ChainID:           row.ChainID,
		CursorBlockNumber: cursor,
		Initialized:       row.Initialized,
		Status:            status,
		CreatedAt:         timeValue(row.CreatedAt),
		UpdatedAt:         timeValue(row.UpdatedAt),
	}, nil
}

func mapProjectSwapPairs(rows []tokensqlc.ProjectSwapPair) ([]swap.Pair, error) {
	result := make([]swap.Pair, 0, len(rows))
	for _, row := range rows {
		item, err := mapProjectSwapPair(row)
		if err != nil {
			return nil, err
		}
		result = append(result, *item)
	}
	return result, nil
}

func mapProjectSwapPair(row tokensqlc.ProjectSwapPair) (*swap.Pair, error) {
	startBlockNumber, err := int64ToUint64("start_block_number", row.StartBlockNumber)
	if err != nil {
		return nil, err
	}
	startBlockTime, err := int64ToUint64("start_block_time", row.StartBlockTime)
	if err != nil {
		return nil, err
	}
	if row.SwapBlockCount < 0 || row.SwapBlockCount > int32(swap.TargetSwapBlockCount) {
		return nil, fmt.Errorf("swap_block_count %d is outside supported range", row.SwapBlockCount)
	}
	absoluteExpiryBlockTime, err := int64ToUint64("absolute_expiry_block_time", row.AbsoluteExpiryBlockTime)
	if err != nil {
		return nil, err
	}
	firstSwapBlockNumber, err := uint64PointerFromInt64("first_swap_block_number", row.FirstSwapBlockNumber)
	if err != nil {
		return nil, err
	}
	firstSwapBlockTime, err := uint64PointerFromInt64("first_swap_block_time", row.FirstSwapBlockTime)
	if err != nil {
		return nil, err
	}
	lastSwapBlockNumber, err := uint64PointerFromInt64("last_swap_block_number", row.LastSwapBlockNumber)
	if err != nil {
		return nil, err
	}
	lastSwapBlockTime, err := uint64PointerFromInt64("last_swap_block_time", row.LastSwapBlockTime)
	if err != nil {
		return nil, err
	}
	nextExpiryBlockTime, err := uint64PointerFromInt64("next_expiry_block_time", row.NextExpiryBlockTime)
	if err != nil {
		return nil, err
	}
	completedBlockNumber, err := uint64PointerFromInt64("completed_block_number", row.CompletedBlockNumber)
	if err != nil {
		return nil, err
	}
	completedBlockTime, err := uint64PointerFromInt64("completed_block_time", row.CompletedBlockTime)
	if err != nil {
		return nil, err
	}
	expiredBlockNumber, err := uint64PointerFromInt64("expired_block_number", row.ExpiredBlockNumber)
	if err != nil {
		return nil, err
	}
	expiredBlockTime, err := uint64PointerFromInt64("expired_block_time", row.ExpiredBlockTime)
	if err != nil {
		return nil, err
	}
	return &swap.Pair{
		ID:                      row.ID,
		ProjectID:               row.ProjectID,
		ChainID:                 row.ChainID,
		Kind:                    swap.PairKind(row.PairKind),
		Address:                 bytesToAddress(row.PairAddress),
		StartBlockNumber:        startBlockNumber,
		StartBlockTime:          startBlockTime,
		SwapBlockCount:          uint16(row.SwapBlockCount),
		Status:                  swap.PairStatus(row.Status),
		FirstSwapBlockNumber:    firstSwapBlockNumber,
		FirstSwapBlockTime:      firstSwapBlockTime,
		LastSwapBlockNumber:     lastSwapBlockNumber,
		LastSwapBlockTime:       lastSwapBlockTime,
		AbsoluteExpiryBlockTime: absoluteExpiryBlockTime,
		NextExpiryBlockTime:     nextExpiryBlockTime,
		CompletedBlockNumber:    completedBlockNumber,
		CompletedBlockTime:      completedBlockTime,
		ExpiredBlockNumber:      expiredBlockNumber,
		ExpiredBlockTime:        expiredBlockTime,
		ExpiredReason:           swap.ExpiredReason(textValue(row.ExpiredReason)),
		CreatedAt:               timeValue(row.CreatedAt),
		UpdatedAt:               timeValue(row.UpdatedAt),
	}, nil
}
