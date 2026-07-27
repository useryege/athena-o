package application

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/discovery"
)

const (
	initialBlockTimeSampleSize = uint64(100)
)

type ScannerRepository interface {
	GetChainIngestCheckpoint(context.Context, int64) (*discovery.ChainIngestCheckpoint, error)
	UpsertChainIngestCheckpoint(context.Context, discovery.ChainIngestCheckpoint) (*discovery.ChainIngestCheckpoint, error)
	UpdateChainIngestCheckpointStatus(context.Context, int64, discovery.ChainIngestStatus) (*discovery.ChainIngestCheckpoint, error)
	IngestProjectCandidateBlock(context.Context, discovery.ChainIngestCheckpoint, []discovery.ProjectCandidate) (*discovery.ChainIngestCheckpoint, error)
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
	DiscoverProjectCandidates(context.Context, int64, uint64) ([]discovery.ProjectCandidate, error)
}

type ScanChainCommand struct {
	ChainID                 int64
	InitialLookbackDuration time.Duration
}

type ScanResult struct {
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
	runStartedAt := time.Now()
	checkpointReadStartedAt := time.Now()
	checkpoint, err := scanner.repository.GetChainIngestCheckpoint(ctx, command.ChainID)
	checkpointReadDuration := time.Since(checkpointReadStartedAt)
	if err != nil {
		logScannerRunFailure(ctx, command.ChainID, "checkpoint_read", runStartedAt, checkpointReadDuration, err)
		return ScanResult{}, err
	}
	if checkpoint == nil {
		err = fmt.Errorf("token chain scanner checkpoint missing for chain %d", command.ChainID)
		logScannerRunFailure(ctx, command.ChainID, "checkpoint_read", runStartedAt, checkpointReadDuration, err)
		return ScanResult{}, err
	}
	if !checkpoint.Enabled || checkpoint.Status != discovery.ChainIngestStatusRunning {
		return ScanResult{}, nil
	}
	latestHeaderStartedAt := time.Now()
	latest, err := scanner.blocks.LatestBlockHeader(ctx, command.ChainID)
	latestHeaderDuration := time.Since(latestHeaderStartedAt)
	if err != nil {
		logScannerRunFailure(ctx, command.ChainID, "latest_header", runStartedAt, latestHeaderDuration, err)
		return ScanResult{}, err
	}
	if checkpoint.CursorBlockNumber == 0 {
		lookbackSeconds := uint64(command.InitialLookbackDuration / time.Second)
		targetTimestamp := uint64(0)
		if latest.Timestamp > lookbackSeconds {
			targetTimestamp = latest.Timestamp - lookbackSeconds
		}
		initialBlockEstimateStartedAt := time.Now()
		estimate, err := scanner.estimateInitialStartBlock(ctx, command.ChainID, latest, lookbackSeconds, targetTimestamp)
		if err != nil {
			logScannerRunFailure(ctx, command.ChainID, "initial_block_estimate", runStartedAt, time.Since(initialBlockEstimateStartedAt), err)
			return ScanResult{}, err
		}
		start := estimate.StartBlock
		checkpoint.CursorBlockNumber = start - 1
		checkpointInitializeStartedAt := time.Now()
		checkpoint, err = scanner.repository.UpsertChainIngestCheckpoint(ctx, *checkpoint)
		if err != nil {
			logScannerRunFailure(ctx, command.ChainID, "checkpoint_initialize", runStartedAt, time.Since(checkpointInitializeStartedAt), err)
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
	if next <= latest.Number {
		log.WithFields(log.Fields{
			"chain_id":                    command.ChainID,
			"cursor_block_number":         checkpoint.CursorBlockNumber,
			"start_block":                 next,
			"latest_block":                latest.Number,
			"remaining_block_count":       latest.Number - next + 1,
			"checkpoint_read_duration_ms": checkpointReadDuration.Milliseconds(),
			"latest_header_duration_ms":   latestHeaderDuration.Milliseconds(),
			"duration_ms":                 time.Since(runStartedAt).Milliseconds(),
		}).Info("token scanner scan range resolved")
	}
	result := ScanResult{}
	for next <= latest.Number {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		blockStartedAt := time.Now()
		blockCheckpointReadStartedAt := time.Now()
		checkpoint, err = scanner.repository.GetChainIngestCheckpoint(ctx, command.ChainID)
		blockCheckpointReadDuration := time.Since(blockCheckpointReadStartedAt)
		if err != nil {
			logScannerBlockFailure(ctx, command.ChainID, next, "checkpoint_read", blockStartedAt, blockCheckpointReadDuration, err)
			return result, err
		}
		if checkpoint == nil || !checkpoint.Enabled || checkpoint.Status != discovery.ChainIngestStatusRunning {
			return result, nil
		}
		log.WithFields(log.Fields{
			"block_number":                next,
			"chain_id":                    command.ChainID,
			"latest_block":                latest.Number,
			"remaining_block_count":       latest.Number - next + 1,
			"checkpoint_read_duration_ms": blockCheckpointReadDuration.Milliseconds(),
		}).Info("token scanner block started")

		discoveryStartedAt := time.Now()
		candidates, err := scanner.blocks.DiscoverProjectCandidates(ctx, command.ChainID, next)
		discoveryDuration := time.Since(discoveryStartedAt)
		if err != nil {
			logScannerBlockFailure(ctx, command.ChainID, next, "candidate_discovery", blockStartedAt, discoveryDuration, err)
			return result, err
		}
		persistenceStartedAt := time.Now()
		if _, err = scanner.repository.IngestProjectCandidateBlock(ctx, discovery.ChainIngestCheckpoint{ChainID: command.ChainID, CursorBlockNumber: next, Status: discovery.ChainIngestStatusRunning}, candidates); err != nil {
			logScannerBlockFailure(ctx, command.ChainID, next, "persistence", blockStartedAt, time.Since(persistenceStartedAt), err)
			return result, err
		}
		persistenceDuration := time.Since(persistenceStartedAt)
		log.WithFields(log.Fields{
			"block_number":                next,
			"candidate_count":             len(candidates),
			"chain_id":                    command.ChainID,
			"checkpoint_read_duration_ms": blockCheckpointReadDuration.Milliseconds(),
			"discovery_duration_ms":       discoveryDuration.Milliseconds(),
			"persistence_duration_ms":     persistenceDuration.Milliseconds(),
			"duration_ms":                 time.Since(blockStartedAt).Milliseconds(),
		}).Info("token scanner block completed")
		result.Blocks++
		result.Candidates += len(candidates)
		next++
	}
	return result, nil
}

func logScannerRunFailure(
	ctx context.Context,
	chainID int64,
	stage string,
	startedAt time.Time,
	phaseDuration time.Duration,
	err error,
) {
	if ctx.Err() != nil {
		return
	}
	log.WithError(err).WithFields(log.Fields{
		"chain_id":          chainID,
		"stage":             stage,
		"phase_duration_ms": phaseDuration.Milliseconds(),
		"duration_ms":       time.Since(startedAt).Milliseconds(),
	}).Error("token scanner run failed")
}

func logScannerBlockFailure(
	ctx context.Context,
	chainID int64,
	blockNumber uint64,
	stage string,
	startedAt time.Time,
	phaseDuration time.Duration,
	err error,
) {
	if ctx.Err() != nil {
		return
	}
	log.WithError(err).WithFields(log.Fields{
		"block_number":      blockNumber,
		"chain_id":          chainID,
		"stage":             stage,
		"phase_duration_ms": phaseDuration.Milliseconds(),
		"duration_ms":       time.Since(startedAt).Milliseconds(),
	}).Error("token scanner block failed")
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
