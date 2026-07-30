package application

import (
	"context"
	"fmt"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/shared"
	"github.com/useryege/athena/internal/token/swap"
)

type Repository interface {
	GetSourceCursor(context.Context, int64) (uint64, error)
	GetProcessingCheckpoint(context.Context, int64) (*swap.ProcessingCheckpoint, error)
	SetProcessingStatus(context.Context, int64, swap.ProcessingStatus) (*swap.ProcessingCheckpoint, error)
	InitializeProcessingCheckpoint(context.Context, int64, uint64) (*swap.ProcessingCheckpoint, error)
	FindNextCollectingPairStartBlock(context.Context, int64, uint64) (*uint64, error)
	ListCollectingPairsByAddresses(context.Context, int64, uint64, []shared.Address) ([]swap.Pair, error)
	AdvanceProcessingCheckpoint(context.Context, int64, uint64, uint64) (*swap.ProcessingCheckpoint, error)
	CommitProcessedBlock(context.Context, CommitProcessedBlockCommand) (CommitProcessedBlockResult, error)
}

type BlockSource interface {
	FilterSwapLogs(context.Context, int64, uint64) ([]swap.RawLog, error)
	BlockTimestamp(context.Context, int64, uint64) (uint64, error)
	DecodeBlockSwapEvents(context.Context, int64, uint64, []swap.RawLog) (swap.BlockObservation, error)
}

type CommitProcessedBlockCommand struct {
	ChainID            int64
	PreviousCheckpoint uint64
	BlockNumber        uint64
	BlockTime          uint64
	PairBlocks         []swap.PairBlockEvents
}

type CommitProcessedBlockResult struct {
	StoredEvents   int
	CompletedPairs int
	ExpiredPairs   int
}

type ProcessResult struct {
	Blocks         uint64
	TopicLogs      int
	MatchedPairs   int
	StoredEvents   int
	CompletedPairs int
	ExpiredPairs   int
}

type SwapProcessor struct {
	repository Repository
	blocks     BlockSource
}

func NewSwapProcessor(repository Repository, blocks BlockSource) *SwapProcessor {
	return &SwapProcessor{repository: repository, blocks: blocks}
}

func (processor *SwapProcessor) StartChain(ctx context.Context, chainID int64) error {
	if processor == nil || processor.repository == nil {
		return fmt.Errorf("token swap processor repository is not configured")
	}
	_, err := processor.repository.SetProcessingStatus(ctx, chainID, swap.ProcessingStatusRunning)
	return err
}

func (processor *SwapProcessor) StopChain(ctx context.Context, chainID int64) error {
	if processor == nil || processor.repository == nil {
		return fmt.Errorf("token swap processor repository is not configured")
	}
	_, err := processor.repository.SetProcessingStatus(ctx, chainID, swap.ProcessingStatusStopped)
	return err
}

func (processor *SwapProcessor) RunOnce(ctx context.Context, chainID int64) (ProcessResult, error) {
	if processor == nil || processor.repository == nil || processor.blocks == nil {
		return ProcessResult{}, fmt.Errorf("token swap processor is not configured")
	}
	runStartedAt := time.Now()
	sourceCursor, err := processor.repository.GetSourceCursor(ctx, chainID)
	if err != nil {
		logSwapRunFailure(ctx, chainID, "source_checkpoint", runStartedAt, err)
		return ProcessResult{}, err
	}
	checkpoint, err := processor.repository.GetProcessingCheckpoint(ctx, chainID)
	if err != nil {
		logSwapRunFailure(ctx, chainID, "swap_checkpoint", runStartedAt, err)
		return ProcessResult{}, err
	}
	if checkpoint == nil {
		err = fmt.Errorf("token swap processing checkpoint missing for chain %d", chainID)
		logSwapRunFailure(ctx, chainID, "swap_checkpoint", runStartedAt, err)
		return ProcessResult{}, err
	}
	if checkpoint.Status != swap.ProcessingStatusRunning {
		return ProcessResult{}, nil
	}
	if !checkpoint.Initialized {
		cursor := sourceCursor
		startBlock, err := processor.repository.FindNextCollectingPairStartBlock(ctx, chainID, sourceCursor)
		if err != nil {
			logSwapRunFailure(ctx, chainID, "checkpoint_initial_target", runStartedAt, err)
			return ProcessResult{}, err
		}
		if startBlock != nil {
			if *startBlock == 0 {
				err = fmt.Errorf("token swap pair start block must be positive")
				logSwapRunFailure(ctx, chainID, "checkpoint_initial_target", runStartedAt, err)
				return ProcessResult{}, err
			}
			cursor = *startBlock - 1
		}
		checkpoint, err = processor.repository.InitializeProcessingCheckpoint(ctx, chainID, cursor)
		if err != nil {
			logSwapRunFailure(ctx, chainID, "checkpoint_initialize", runStartedAt, err)
			return ProcessResult{}, err
		}
		log.WithFields(log.Fields{
			"chain_id":                   chainID,
			"cursor_block_number":        checkpoint.CursorBlockNumber,
			"source_cursor_block_number": sourceCursor,
		}).Info("initialized token swap processing checkpoint")
	}
	if checkpoint.CursorBlockNumber >= sourceCursor {
		return ProcessResult{}, nil
	}

	log.WithFields(log.Fields{
		"chain_id":                   chainID,
		"cursor_block_number":        checkpoint.CursorBlockNumber,
		"source_cursor_block_number": sourceCursor,
		"remaining_block_count":      sourceCursor - checkpoint.CursorBlockNumber,
	}).Info("token swap processing range resolved")

	result := ProcessResult{}
	for checkpoint.CursorBlockNumber < sourceCursor {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		checkpoint, err = processor.repository.GetProcessingCheckpoint(ctx, chainID)
		if err != nil {
			logSwapRunFailure(ctx, chainID, "swap_checkpoint", runStartedAt, err)
			return result, err
		}
		if checkpoint == nil || !checkpoint.Initialized || checkpoint.Status != swap.ProcessingStatusRunning {
			return result, nil
		}
		if checkpoint.CursorBlockNumber >= sourceCursor {
			break
		}
		next := checkpoint.CursorBlockNumber + 1
		startBlock, err := processor.repository.FindNextCollectingPairStartBlock(ctx, chainID, sourceCursor)
		if err != nil {
			logSwapBlockFailure(ctx, chainID, next, "collecting_target", time.Now(), err)
			return result, err
		}
		if startBlock == nil {
			advanced, err := processor.repository.AdvanceProcessingCheckpoint(ctx, chainID, checkpoint.CursorBlockNumber, sourceCursor)
			if err != nil {
				logSwapBlockFailure(ctx, chainID, next, "checkpoint_fast_forward", time.Now(), err)
				return result, err
			}
			result.Blocks += advanced.CursorBlockNumber - checkpoint.CursorBlockNumber
			checkpoint = advanced
			continue
		}
		if *startBlock > next {
			target := *startBlock - 1
			if target > sourceCursor {
				target = sourceCursor
			}
			advanced, err := processor.repository.AdvanceProcessingCheckpoint(ctx, chainID, checkpoint.CursorBlockNumber, target)
			if err != nil {
				logSwapBlockFailure(ctx, chainID, next, "checkpoint_gap", time.Now(), err)
				return result, err
			}
			result.Blocks += advanced.CursorBlockNumber - checkpoint.CursorBlockNumber
			checkpoint = advanced
			continue
		}

		blockResult, err := processor.processBlock(ctx, chainID, checkpoint.CursorBlockNumber, next)
		if err != nil {
			return result, err
		}
		result.Blocks++
		result.TopicLogs += blockResult.TopicLogs
		result.MatchedPairs += blockResult.MatchedPairs
		result.StoredEvents += blockResult.StoredEvents
		result.CompletedPairs += blockResult.CompletedPairs
		result.ExpiredPairs += blockResult.ExpiredPairs
		checkpoint.CursorBlockNumber = next
	}
	return result, nil
}

type blockProcessResult struct {
	TopicLogs      int
	MatchedPairs   int
	StoredEvents   int
	CompletedPairs int
	ExpiredPairs   int
}

func (processor *SwapProcessor) processBlock(ctx context.Context, chainID int64, previousCheckpoint, blockNumber uint64) (blockProcessResult, error) {
	startedAt := time.Now()
	logsStartedAt := time.Now()
	logs, err := processor.blocks.FilterSwapLogs(ctx, chainID, blockNumber)
	logsDuration := time.Since(logsStartedAt)
	if err != nil {
		logSwapBlockFailure(ctx, chainID, blockNumber, "swap_logs", startedAt, err)
		return blockProcessResult{}, err
	}

	matchStartedAt := time.Now()
	addresses := uniqueLogAddresses(logs)
	pairs, err := processor.repository.ListCollectingPairsByAddresses(ctx, chainID, blockNumber, addresses)
	matchDuration := time.Since(matchStartedAt)
	if err != nil {
		logSwapBlockFailure(ctx, chainID, blockNumber, "pair_match", startedAt, err)
		return blockProcessResult{}, err
	}
	pairsByAddress := make(map[shared.Address][]swap.Pair)
	for _, pair := range pairs {
		pairsByAddress[pair.Address] = append(pairsByAddress[pair.Address], pair)
	}
	relevantLogs := make([]swap.RawLog, 0)
	for _, item := range logs {
		if len(pairsByAddress[item.Address]) > 0 {
			relevantLogs = append(relevantLogs, item)
		}
	}
	sort.Slice(relevantLogs, func(i, j int) bool {
		if relevantLogs[i].TransactionIndex == relevantLogs[j].TransactionIndex {
			return relevantLogs[i].LogIndex < relevantLogs[j].LogIndex
		}
		return relevantLogs[i].TransactionIndex < relevantLogs[j].TransactionIndex
	})

	blockReadStartedAt := time.Now()
	var blockTime uint64
	var decoded []swap.DecodedEvent
	var blockRPCDuration time.Duration
	var decodeDuration time.Duration
	if len(relevantLogs) == 0 {
		blockTime, err = processor.blocks.BlockTimestamp(ctx, chainID, blockNumber)
		blockRPCDuration = time.Since(blockReadStartedAt)
	} else {
		var observation swap.BlockObservation
		observation, err = processor.blocks.DecodeBlockSwapEvents(ctx, chainID, blockNumber, relevantLogs)
		blockTime = observation.BlockTime
		decoded = observation.Events
		blockRPCDuration = observation.RPCDuration
		decodeDuration = observation.DecodeDuration
	}
	blockReadDuration := time.Since(blockReadStartedAt)
	if err != nil {
		logSwapBlockFailure(ctx, chainID, blockNumber, "block_observation", startedAt, err)
		return blockProcessResult{}, err
	}

	pairBlocks := fanOutPairEvents(decoded, pairsByAddress)
	persistenceStartedAt := time.Now()
	committed, err := processor.repository.CommitProcessedBlock(ctx, CommitProcessedBlockCommand{
		ChainID:            chainID,
		PreviousCheckpoint: previousCheckpoint,
		BlockNumber:        blockNumber,
		BlockTime:          blockTime,
		PairBlocks:         pairBlocks,
	})
	persistenceDuration := time.Since(persistenceStartedAt)
	if err != nil {
		logSwapBlockFailure(ctx, chainID, blockNumber, "persistence", startedAt, err)
		return blockProcessResult{}, err
	}
	log.WithFields(log.Fields{
		"block_number":            blockNumber,
		"block_time":              blockTime,
		"chain_id":                chainID,
		"topic_log_count":         len(logs),
		"relevant_log_count":      len(relevantLogs),
		"matched_pair_count":      len(pairs),
		"stored_event_count":      committed.StoredEvents,
		"completed_pair_count":    committed.CompletedPairs,
		"expired_pair_count":      committed.ExpiredPairs,
		"swap_logs_duration_ms":   logsDuration.Milliseconds(),
		"pair_match_duration_ms":  matchDuration.Milliseconds(),
		"block_read_duration_ms":  blockReadDuration.Milliseconds(),
		"block_rpc_duration_ms":   blockRPCDuration.Milliseconds(),
		"decode_duration_ms":      decodeDuration.Milliseconds(),
		"persistence_duration_ms": persistenceDuration.Milliseconds(),
		"duration_ms":             time.Since(startedAt).Milliseconds(),
	}).Info("token swap block processing completed")
	return blockProcessResult{
		TopicLogs:      len(logs),
		MatchedPairs:   len(pairs),
		StoredEvents:   committed.StoredEvents,
		CompletedPairs: committed.CompletedPairs,
		ExpiredPairs:   committed.ExpiredPairs,
	}, nil
}

func uniqueLogAddresses(logs []swap.RawLog) []shared.Address {
	set := make(map[shared.Address]struct{}, len(logs))
	for _, item := range logs {
		set[item.Address] = struct{}{}
	}
	result := make([]shared.Address, 0, len(set))
	for address := range set {
		result = append(result, address)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Hex() < result[j].Hex() })
	return result
}

func fanOutPairEvents(decoded []swap.DecodedEvent, pairsByAddress map[shared.Address][]swap.Pair) []swap.PairBlockEvents {
	eventsByPair := make(map[int64][]swap.Event)
	for _, event := range decoded {
		for _, pair := range pairsByAddress[event.PairAddress] {
			eventsByPair[pair.ID] = append(eventsByPair[pair.ID], swap.Event{
				TransactionHash:  event.TransactionHash,
				TransactionIndex: event.TransactionIndex,
				LogIndex:         event.LogIndex,
				TxFrom:           event.TxFrom,
				Sender:           event.Sender,
				ToAddress:        event.ToAddress,
				Amount0In:        event.Amount0In,
				Amount1In:        event.Amount1In,
				Amount0Out:       event.Amount0Out,
				Amount1Out:       event.Amount1Out,
			})
		}
	}
	pairIDs := make([]int64, 0, len(eventsByPair))
	for pairID := range eventsByPair {
		pairIDs = append(pairIDs, pairID)
	}
	sort.Slice(pairIDs, func(i, j int) bool { return pairIDs[i] < pairIDs[j] })
	result := make([]swap.PairBlockEvents, 0, len(pairIDs))
	for _, pairID := range pairIDs {
		result = append(result, swap.PairBlockEvents{PairID: pairID, Events: eventsByPair[pairID]})
	}
	return result
}

func logSwapRunFailure(ctx context.Context, chainID int64, stage string, startedAt time.Time, err error) {
	if ctx.Err() != nil {
		return
	}
	log.WithError(err).WithFields(log.Fields{
		"chain_id":    chainID,
		"stage":       stage,
		"duration_ms": time.Since(startedAt).Milliseconds(),
	}).Error("token swap processing run failed")
}

func logSwapBlockFailure(ctx context.Context, chainID int64, blockNumber uint64, stage string, startedAt time.Time, err error) {
	if ctx.Err() != nil {
		return
	}
	log.WithError(err).WithFields(log.Fields{
		"block_number": blockNumber,
		"chain_id":     chainID,
		"stage":        stage,
		"duration_ms":  time.Since(startedAt).Milliseconds(),
	}).Error("token swap block processing failed")
}
