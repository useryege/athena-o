package application

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/discovery"
)

const (
	maxBlocksPerScanBatch      = uint64(100)
	initialBlockTimeSampleSize = uint64(100)
)

type ScannerRepository interface {
	GetChainIngestCheckpoint(context.Context, int64) (*discovery.ChainIngestCheckpoint, error)
	UpsertChainIngestCheckpoint(context.Context, discovery.ChainIngestCheckpoint) (*discovery.ChainIngestCheckpoint, error)
	UpdateChainIngestCheckpointStatus(context.Context, int64, discovery.ChainIngestStatus) (*discovery.ChainIngestCheckpoint, error)
	IngestProjectCandidateBatch(context.Context, discovery.ChainIngestCheckpoint, []discovery.ProjectCandidate) (*discovery.ChainIngestCheckpoint, error)
}

func (scanner *Scanner) StartChain(ctx context.Context, chainID int64) error {
	checkpoint, err := scanner.repository.GetChainIngestCheckpoint(ctx, chainID)
	if err != nil {
		return err
	}
	if checkpoint == nil {
		return fmt.Errorf("token chain scanner checkpoint missing for chain %d", chainID)
	}
	checkpoint.Status = discovery.ChainIngestStatusRunning
	_, err = scanner.repository.UpsertChainIngestCheckpoint(ctx, *checkpoint)
	return err
}

func (scanner *Scanner) StopChain(ctx context.Context, chainID int64) error {
	_, err := scanner.repository.UpdateChainIngestCheckpointStatus(ctx, chainID, discovery.ChainIngestStatusStopped)
	return err
}

type BlockSource interface {
	LatestBlockHeader(context.Context, int64) (discovery.BlockHeader, error)
	BlockHeaderByNumber(context.Context, int64, uint64) (discovery.BlockHeader, error)
	DiscoverProjectCandidates(context.Context, int64, uint64, uint64, int) ([]discovery.ProjectCandidate, error)
}

type ScanChainCommand struct {
	ChainID                 int64
	InitialLookbackDuration time.Duration
	BlockFetchConcurrency   int
}

type ScanResult struct {
	Batches    int
	Blocks     uint64
	Candidates int
}

type initialBlockEstimate struct {
	StartBlock              uint64
	EstimatedLookbackBlocks uint64
	SampleBlock             uint64
	SampleTimestamp         uint64
	SampleBlockCount        uint64
	SampleElapsedSeconds    uint64
	AverageBlockTimeMillis  uint64
}

type Scanner struct {
	repository ScannerRepository
	blocks     BlockSource
}

func NewScanner(repository ScannerRepository, blocks BlockSource) *Scanner {
	return &Scanner{repository: repository, blocks: blocks}
}

func (scanner *Scanner) RunOnce(ctx context.Context, command ScanChainCommand) (ScanResult, error) {
	if scanner == nil || scanner.repository == nil || scanner.blocks == nil {
		return ScanResult{}, fmt.Errorf("token scanner application is not configured")
	}
	if command.InitialLookbackDuration < time.Second {
		return ScanResult{}, fmt.Errorf("token scanner initial lookback duration must be at least 1s")
	}
	checkpoint, err := scanner.repository.GetChainIngestCheckpoint(ctx, command.ChainID)
	if err != nil {
		return ScanResult{}, err
	}
	if checkpoint == nil {
		return ScanResult{}, fmt.Errorf("token chain scanner checkpoint missing for chain %d", command.ChainID)
	}
	if !checkpoint.Enabled || checkpoint.Status != discovery.ChainIngestStatusRunning {
		return ScanResult{}, nil
	}
	latest, err := scanner.blocks.LatestBlockHeader(ctx, command.ChainID)
	if err != nil {
		return ScanResult{}, err
	}
	if checkpoint.CursorBlockNumber == 0 {
		lookbackSeconds := uint64(command.InitialLookbackDuration / time.Second)
		targetTimestamp := uint64(0)
		if latest.Timestamp > lookbackSeconds {
			targetTimestamp = latest.Timestamp - lookbackSeconds
		}
		estimate, err := scanner.estimateInitialStartBlock(ctx, command.ChainID, latest, lookbackSeconds, targetTimestamp)
		if err != nil {
			return ScanResult{}, err
		}
		start := estimate.StartBlock
		checkpoint.CursorBlockNumber = start - 1
		checkpoint, err = scanner.repository.UpsertChainIngestCheckpoint(ctx, *checkpoint)
		if err != nil {
			return ScanResult{}, err
		}
		log.WithFields(log.Fields{
			"chain_id":                  command.ChainID,
			"latest_block":              latest.Number,
			"latest_timestamp":          latest.Timestamp,
			"lookback_duration":         command.InitialLookbackDuration.String(),
			"start_block":               start,
			"target_timestamp":          targetTimestamp,
			"estimated_lookback_blocks": estimate.EstimatedLookbackBlocks,
			"sample_block":              estimate.SampleBlock,
			"sample_timestamp":          estimate.SampleTimestamp,
			"sample_block_count":        estimate.SampleBlockCount,
			"sample_elapsed_seconds":    estimate.SampleElapsedSeconds,
			"average_block_time_ms":     estimate.AverageBlockTimeMillis,
		}).Info("initialized token chain scanner checkpoint")
	}
	next := uint64(1)
	if checkpoint.CursorBlockNumber > 0 {
		next = checkpoint.CursorBlockNumber + 1
	}
	result := ScanResult{}
	for next <= latest.Number {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		checkpoint, err = scanner.repository.GetChainIngestCheckpoint(ctx, command.ChainID)
		if err != nil {
			return result, err
		}
		if checkpoint == nil || !checkpoint.Enabled || checkpoint.Status != discovery.ChainIngestStatusRunning {
			return result, nil
		}
		batchEnd := next + maxBlocksPerScanBatch - 1
		if batchEnd > latest.Number {
			batchEnd = latest.Number
		}
		batchStartedAt := time.Now()
		candidates, err := scanner.blocks.DiscoverProjectCandidates(ctx, command.ChainID, next, batchEnd, command.BlockFetchConcurrency)
		if err != nil {
			return result, err
		}
		if _, err = scanner.repository.IngestProjectCandidateBatch(ctx, discovery.ChainIngestCheckpoint{ChainID: command.ChainID, CursorBlockNumber: batchEnd, Status: discovery.ChainIngestStatusRunning}, candidates); err != nil {
			return result, err
		}
		blockCount := batchEnd - next + 1
		log.WithFields(log.Fields{
			"block_count":     blockCount,
			"candidate_count": len(candidates),
			"chain_id":        command.ChainID,
			"duration_ms":     time.Since(batchStartedAt).Milliseconds(),
			"end_block":       batchEnd,
			"start_block":     next,
		}).Info("token scanner batch completed")
		result.Batches++
		result.Blocks += blockCount
		result.Candidates += len(candidates)
		next = batchEnd + 1
	}
	return result, nil
}

func (scanner *Scanner) estimateInitialStartBlock(
	ctx context.Context,
	chainID int64,
	latest discovery.BlockHeader,
	lookbackSeconds uint64,
	targetTimestamp uint64,
) (initialBlockEstimate, error) {
	estimate := initialBlockEstimate{
		StartBlock:              1,
		EstimatedLookbackBlocks: latest.Number,
	}
	if latest.Number <= initialBlockTimeSampleSize || targetTimestamp == 0 {
		return estimate, nil
	}

	sampleBlock := latest.Number - initialBlockTimeSampleSize
	sample, err := scanner.blocks.BlockHeaderByNumber(ctx, chainID, sampleBlock)
	if err != nil {
		return initialBlockEstimate{}, err
	}
	if sample.Number != sampleBlock {
		return initialBlockEstimate{}, fmt.Errorf(
			"token scanner block-time sample returned block %d for requested block %d",
			sample.Number,
			sampleBlock,
		)
	}
	if sample.Timestamp >= latest.Timestamp {
		return initialBlockEstimate{}, fmt.Errorf(
			"token scanner block-time sample must have an earlier timestamp: sample=%d latest=%d",
			sample.Timestamp,
			latest.Timestamp,
		)
	}

	sampleElapsedSeconds := latest.Timestamp - sample.Timestamp
	numerator := lookbackSeconds * initialBlockTimeSampleSize
	estimatedLookbackBlocks := numerator / sampleElapsedSeconds
	remainder := numerator % sampleElapsedSeconds
	if remainder >= sampleElapsedSeconds-remainder {
		estimatedLookbackBlocks++
	}

	startBlock := uint64(1)
	if estimatedLookbackBlocks < latest.Number {
		startBlock = latest.Number - estimatedLookbackBlocks
	}
	return initialBlockEstimate{
		StartBlock:              startBlock,
		EstimatedLookbackBlocks: estimatedLookbackBlocks,
		SampleBlock:             sample.Number,
		SampleTimestamp:         sample.Timestamp,
		SampleBlockCount:        initialBlockTimeSampleSize,
		SampleElapsedSeconds:    sampleElapsedSeconds,
		AverageBlockTimeMillis:  sampleElapsedSeconds * 1000 / initialBlockTimeSampleSize,
	}, nil
}
