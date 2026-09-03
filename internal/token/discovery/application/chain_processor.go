package application

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/shared"
)

const (
	initialBlockTimeSampleSize = uint64(100)
	maxCandidateBatchSize      = 100
	maxCandidateConcurrency    = 10
)

type ChainProcessorRepository interface {
	GetChainProcessingCheckpoint(context.Context, int64) (*discovery.ChainProcessingCheckpoint, error)
	UpsertChainProcessingCheckpoint(context.Context, discovery.ChainProcessingCheckpoint) (*discovery.ChainProcessingCheckpoint, error)
	UpdateChainProcessingCheckpointStatus(context.Context, int64, discovery.ChainProcessingStatus) (*discovery.ChainProcessingCheckpoint, error)
	StartChainBlockProcessingAttempt(context.Context, int64, uint64) (*discovery.ChainBlockProcessingAttempt, error)
	CompleteChainBlockProcessingAttempt(context.Context, discovery.ChainBlockProcessingAttemptCompletion) (*discovery.ChainBlockProcessingAttempt, error)
	CommitProcessedBlock(context.Context, CommitProcessedBlockCommand) (*CommitProcessedBlockResult, error)
}

type BlockSource interface {
	LatestBlockHeader(context.Context, int64) (discovery.BlockHeader, error)
	BlockHeaderByNumber(context.Context, int64, uint64) (discovery.BlockHeader, error)
	DiscoverProjectBlock(context.Context, int64, uint64) (discovery.ProjectCandidateBlock, error)
}

type CandidateInspector interface {
	InspectCandidates(context.Context, int64, []discovery.ProjectCandidate, int) ([]CandidateInspection, error)
}

type CandidateInspection struct {
	Candidate         discovery.ProjectCandidate
	Accepted          bool
	CodeHash          shared.Hash
	Name              string
	Symbol            string
	Decimals          uint8
	TotalSupply       *big.Int
	WethPair          shared.Address
	UsdtPair          shared.Address
	RelatedWallets    []RelatedWallet
	InitialRecipients []InitialRecipient
}

type RelatedWallet struct {
	Wallet shared.Address
	Role   string
}

type InitialRecipient struct {
	Wallet            shared.Address
	RatioBPS          int64
	RankIndex         int32
	SourceTxHash      shared.Hash
	SourceBlockNumber uint64
}

type CommitProcessedBlockCommand struct {
	Checkpoint  discovery.ChainProcessingCheckpoint
	Block       discovery.BlockHeader
	Inspections []CandidateInspection
}

type CommitProcessedBlockResult struct {
	Checkpoint       discovery.ChainProcessingCheckpoint
	AlreadyProcessed bool
}

type ProcessChainCommand struct {
	ChainID                 int64
	InitialLookbackDuration time.Duration
}

type ProcessResult struct {
	Blocks     uint64
	Candidates int
	Validated  int
	Rejected   int
}

type ChainProcessorOptions struct {
	CandidateBatchSize   int
	CandidateConcurrency int
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

type ChainProcessor struct {
	repository ChainProcessorRepository
	blocks     BlockSource
	inspector  CandidateInspector
	options    ChainProcessorOptions
}

type blockAttemptMetrics struct {
	checkpointRead *time.Duration
	discovery      *time.Duration
	validation     *time.Duration
	persistence    *time.Duration
	candidateCount *int32
	validatedCount *int32
	rejectedCount  *int32
}

func NewChainProcessor(repository ChainProcessorRepository, blocks BlockSource, inspector CandidateInspector, options ChainProcessorOptions) *ChainProcessor {
	if options.CandidateBatchSize <= 0 || options.CandidateBatchSize > maxCandidateBatchSize {
		options.CandidateBatchSize = maxCandidateBatchSize
	}
	if options.CandidateConcurrency <= 0 || options.CandidateConcurrency > maxCandidateConcurrency {
		options.CandidateConcurrency = maxCandidateConcurrency
	}
	return &ChainProcessor{repository: repository, blocks: blocks, inspector: inspector, options: options}
}

func (processor *ChainProcessor) StartChain(ctx context.Context, chainID int64) error {
	checkpoint, err := processor.repository.GetChainProcessingCheckpoint(ctx, chainID)
	if err != nil {
		return err
	}
	if checkpoint == nil {
		return fmt.Errorf("token chain processing checkpoint missing for chain %d", chainID)
	}
	checkpoint.Status = discovery.ChainProcessingStatusRunning
	_, err = processor.repository.UpsertChainProcessingCheckpoint(ctx, *checkpoint)
	return err
}

func (processor *ChainProcessor) StopChain(ctx context.Context, chainID int64) error {
	_, err := processor.repository.UpdateChainProcessingCheckpointStatus(ctx, chainID, discovery.ChainProcessingStatusStopped)
	return err
}

func (processor *ChainProcessor) RunOnce(ctx context.Context, command ProcessChainCommand) (ProcessResult, error) {
	if processor == nil || processor.repository == nil || processor.blocks == nil || processor.inspector == nil {
		return ProcessResult{}, fmt.Errorf("token chain processor application is not configured")
	}
	if command.InitialLookbackDuration < time.Second {
		return ProcessResult{}, fmt.Errorf("token chain processor initial lookback duration must be at least 1s")
	}
	runStartedAt := time.Now()
	checkpointReadStartedAt := time.Now()
	checkpoint, err := processor.repository.GetChainProcessingCheckpoint(ctx, command.ChainID)
	checkpointReadDuration := time.Since(checkpointReadStartedAt)
	if err != nil {
		logProcessorRunFailure(ctx, command.ChainID, "checkpoint_read", runStartedAt, checkpointReadDuration, err)
		return ProcessResult{}, err
	}
	if checkpoint == nil {
		err = fmt.Errorf("token chain processing checkpoint missing for chain %d", command.ChainID)
		logProcessorRunFailure(ctx, command.ChainID, "checkpoint_read", runStartedAt, checkpointReadDuration, err)
		return ProcessResult{}, err
	}
	if !checkpoint.Enabled || checkpoint.Status != discovery.ChainProcessingStatusRunning {
		return ProcessResult{}, nil
	}
	latestHeaderStartedAt := time.Now()
	latest, err := processor.blocks.LatestBlockHeader(ctx, command.ChainID)
	latestHeaderDuration := time.Since(latestHeaderStartedAt)
	if err != nil {
		logProcessorRunFailure(ctx, command.ChainID, "latest_header", runStartedAt, latestHeaderDuration, err)
		return ProcessResult{}, err
	}
	if checkpoint.CursorBlockNumber == 0 {
		lookbackSeconds := uint64(command.InitialLookbackDuration / time.Second)
		targetTimestamp := uint64(0)
		if latest.Timestamp > lookbackSeconds {
			targetTimestamp = latest.Timestamp - lookbackSeconds
		}
		initialBlockEstimateStartedAt := time.Now()
		estimate, err := processor.estimateInitialStartBlock(ctx, command.ChainID, latest, lookbackSeconds, targetTimestamp)
		if err != nil {
			logProcessorRunFailure(ctx, command.ChainID, "initial_block_estimate", runStartedAt, time.Since(initialBlockEstimateStartedAt), err)
			return ProcessResult{}, err
		}
		start := estimate.StartBlock
		checkpoint.CursorBlockNumber = start - 1
		checkpointInitializeStartedAt := time.Now()
		checkpoint, err = processor.repository.UpsertChainProcessingCheckpoint(ctx, *checkpoint)
		if err != nil {
			logProcessorRunFailure(ctx, command.ChainID, "checkpoint_initialize", runStartedAt, time.Since(checkpointInitializeStartedAt), err)
			return ProcessResult{}, err
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
		}).Info("initialized token chain processing checkpoint")
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
		}).Info("token chain processing range resolved")
	}
	result := ProcessResult{}
	for next <= latest.Number {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		blockStartedAt := time.Now()
		attempt, err := processor.repository.StartChainBlockProcessingAttempt(ctx, command.ChainID, next)
		if err != nil {
			logProcessorBlockFailure(ctx, command.ChainID, next, "attempt_start", blockStartedAt, time.Since(blockStartedAt), err)
			return result, err
		}
		metrics := blockAttemptMetrics{}
		blockCheckpointReadStartedAt := time.Now()
		checkpoint, err = processor.repository.GetChainProcessingCheckpoint(ctx, command.ChainID)
		blockCheckpointReadDuration := time.Since(blockCheckpointReadStartedAt)
		metrics.checkpointRead = durationPointer(blockCheckpointReadDuration)
		if err != nil {
			err = processor.finishBlockAttempt(ctx, attempt.ID, 0, blockAttemptFailureStatus(ctx), discovery.ChainBlockProcessingStageCheckpointRead, err, metrics, false)
			logProcessorBlockFailure(ctx, command.ChainID, next, "checkpoint_read", blockStartedAt, blockCheckpointReadDuration, err)
			return result, err
		}
		if checkpoint == nil || !checkpoint.Enabled || checkpoint.Status != discovery.ChainProcessingStatusRunning {
			if finishErr := processor.finishBlockAttempt(ctx, attempt.ID, 0, discovery.ChainBlockProcessingAttemptStatusCancelled, discovery.ChainBlockProcessingStageCheckpointRead, nil, metrics, false); finishErr != nil {
				return result, finishErr
			}
			return result, nil
		}
		log.WithFields(log.Fields{
			"block_number":                next,
			"chain_id":                    command.ChainID,
			"latest_block":                latest.Number,
			"remaining_block_count":       latest.Number - next + 1,
			"checkpoint_read_duration_ms": blockCheckpointReadDuration.Milliseconds(),
		}).Info("token chain block processing started")

		discoveryStartedAt := time.Now()
		discoveredBlock, err := processor.blocks.DiscoverProjectBlock(ctx, command.ChainID, next)
		discoveryDuration := time.Since(discoveryStartedAt)
		metrics.discovery = durationPointer(discoveryDuration)
		if err != nil {
			err = processor.finishBlockAttempt(ctx, attempt.ID, 0, blockAttemptFailureStatus(ctx), discovery.ChainBlockProcessingStageCandidateDiscovery, err, metrics, false)
			logProcessorBlockFailure(ctx, command.ChainID, next, "candidate_discovery", blockStartedAt, discoveryDuration, err)
			return result, err
		}
		if discoveredBlock.Header.Number != next {
			err = fmt.Errorf("token project discovery returned block %d for requested block %d", discoveredBlock.Header.Number, next)
			err = processor.finishBlockAttempt(ctx, attempt.ID, discoveredBlock.Header.Timestamp, discovery.ChainBlockProcessingAttemptStatusFailed, discovery.ChainBlockProcessingStageCandidateDiscovery, err, metrics, false)
			logProcessorBlockFailure(ctx, command.ChainID, next, "candidate_discovery", blockStartedAt, discoveryDuration, err)
			return result, err
		}
		candidates := discoveredBlock.Candidates
		candidateCount := int32(len(candidates))
		metrics.candidateCount = &candidateCount
		validationStartedAt := time.Now()
		inspections, err := processor.inspectCandidateBatches(ctx, command.ChainID, candidates)
		validationDuration := time.Since(validationStartedAt)
		metrics.validation = durationPointer(validationDuration)
		if err != nil {
			err = processor.finishBlockAttempt(ctx, attempt.ID, discoveredBlock.Header.Timestamp, blockAttemptFailureStatus(ctx), discovery.ChainBlockProcessingStageCandidateValidation, err, metrics, false)
			logProcessorBlockFailure(ctx, command.ChainID, next, "candidate_validation", blockStartedAt, validationDuration, err)
			return result, err
		}
		validated, rejected := countInspectionOutcomes(inspections)
		validatedCount, rejectedCount := int32(validated), int32(rejected)
		metrics.validatedCount, metrics.rejectedCount = &validatedCount, &rejectedCount
		persistenceStartedAt := time.Now()
		commitResult, err := processor.repository.CommitProcessedBlock(ctx, CommitProcessedBlockCommand{
			Checkpoint: discovery.ChainProcessingCheckpoint{
				ChainID:           command.ChainID,
				CursorBlockNumber: next,
				Status:            discovery.ChainProcessingStatusRunning,
			},
			Block:       discoveredBlock.Header,
			Inspections: inspections,
		})
		persistenceDuration := time.Since(persistenceStartedAt)
		metrics.persistence = durationPointer(persistenceDuration)
		if err != nil {
			err = processor.finishBlockAttempt(ctx, attempt.ID, discoveredBlock.Header.Timestamp, blockAttemptFailureStatus(ctx), discovery.ChainBlockProcessingStagePersistence, err, metrics, false)
			logProcessorBlockFailure(ctx, command.ChainID, next, "persistence", blockStartedAt, persistenceDuration, err)
			return result, err
		}
		if commitResult == nil {
			err = fmt.Errorf("commit token chain block %d returned no result", next)
			err = processor.finishBlockAttempt(ctx, attempt.ID, discoveredBlock.Header.Timestamp, discovery.ChainBlockProcessingAttemptStatusFailed, discovery.ChainBlockProcessingStagePersistence, err, metrics, false)
			return result, err
		}
		if err := processor.finishBlockAttempt(ctx, attempt.ID, discoveredBlock.Header.Timestamp, discovery.ChainBlockProcessingAttemptStatusSucceeded, discovery.ChainBlockProcessingStagePersistence, nil, metrics, true); err != nil {
			logProcessorBlockFailure(ctx, command.ChainID, next, "attempt_completion", blockStartedAt, time.Since(blockStartedAt), err)
			return result, err
		}
		log.WithFields(log.Fields{
			"block_number":                next,
			"block_time":                  discoveredBlock.Header.Timestamp,
			"candidate_count":             len(candidates),
			"validated_count":             validated,
			"rejected_count":              rejected,
			"chain_id":                    command.ChainID,
			"checkpoint_read_duration_ms": blockCheckpointReadDuration.Milliseconds(),
			"discovery_duration_ms":       discoveryDuration.Milliseconds(),
			"validation_duration_ms":      validationDuration.Milliseconds(),
			"persistence_duration_ms":     persistenceDuration.Milliseconds(),
			"duration_ms":                 blockAttemptDuration(metrics).Milliseconds(),
		}).Info("token chain block processing completed")
		if !commitResult.AlreadyProcessed {
			result.Blocks++
			result.Candidates += len(candidates)
			result.Validated += validated
			result.Rejected += rejected
		}
		next = commitResult.Checkpoint.CursorBlockNumber + 1
	}
	return result, nil
}

func (processor *ChainProcessor) finishBlockAttempt(
	ctx context.Context,
	attemptID int64,
	blockTime uint64,
	status discovery.ChainBlockProcessingAttemptStatus,
	stage discovery.ChainBlockProcessingStage,
	processingErr error,
	metrics blockAttemptMetrics,
	timingComplete bool,
) error {
	completionCtx := ctx
	cancel := func() {}
	if ctx.Err() != nil {
		completionCtx, cancel = context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	}
	defer cancel()
	errorMessage := ""
	if processingErr != nil {
		errorMessage = processingErr.Error()
	}
	_, completionErr := processor.repository.CompleteChainBlockProcessingAttempt(completionCtx, discovery.ChainBlockProcessingAttemptCompletion{
		AttemptID:              attemptID,
		BlockTime:              blockTime,
		Status:                 status,
		TerminalStage:          stage,
		ErrorMessage:           errorMessage,
		CheckpointReadDuration: metrics.checkpointRead,
		DiscoveryDuration:      metrics.discovery,
		ValidationDuration:     metrics.validation,
		PersistenceDuration:    metrics.persistence,
		CandidateCount:         metrics.candidateCount,
		ValidatedCount:         metrics.validatedCount,
		RejectedCount:          metrics.rejectedCount,
		TimingComplete:         timingComplete,
	})
	if processingErr != nil && completionErr != nil {
		return errors.Join(processingErr, fmt.Errorf("record token chain block processing attempt: %w", completionErr))
	}
	if processingErr != nil {
		return processingErr
	}
	if completionErr != nil {
		return fmt.Errorf("record token chain block processing attempt: %w", completionErr)
	}
	return nil
}

func blockAttemptFailureStatus(ctx context.Context) discovery.ChainBlockProcessingAttemptStatus {
	if ctx.Err() != nil {
		return discovery.ChainBlockProcessingAttemptStatusCancelled
	}
	return discovery.ChainBlockProcessingAttemptStatusFailed
}

func durationPointer(value time.Duration) *time.Duration {
	result := value
	return &result
}

func blockAttemptDuration(metrics blockAttemptMetrics) time.Duration {
	var result time.Duration
	for _, value := range []*time.Duration{metrics.checkpointRead, metrics.discovery, metrics.validation, metrics.persistence} {
		if value != nil {
			result += *value
		}
	}
	return result
}

func (processor *ChainProcessor) inspectCandidateBatches(ctx context.Context, chainID int64, candidates []discovery.ProjectCandidate) ([]CandidateInspection, error) {
	if len(candidates) == 0 {
		return nil, nil
	}
	result := make([]CandidateInspection, 0, len(candidates))
	for start := 0; start < len(candidates); start += processor.options.CandidateBatchSize {
		end := start + processor.options.CandidateBatchSize
		if end > len(candidates) {
			end = len(candidates)
		}
		items, err := processor.inspector.InspectCandidates(ctx, chainID, candidates[start:end], processor.options.CandidateConcurrency)
		if err != nil {
			return nil, err
		}
		if len(items) != end-start {
			return nil, fmt.Errorf("inspect candidates returned %d results for %d candidates", len(items), end-start)
		}
		for index := range items {
			expected := candidates[start+index]
			if items[index].Candidate != expected {
				return nil, fmt.Errorf("inspect candidates returned an out-of-order result at index %d", start+index)
			}
		}
		result = append(result, items...)
	}
	return result, nil
}

func countInspectionOutcomes(inspections []CandidateInspection) (validated, rejected int) {
	for _, inspection := range inspections {
		if inspection.Accepted {
			validated++
		} else {
			rejected++
		}
	}
	return validated, rejected
}

func logProcessorRunFailure(ctx context.Context, chainID int64, stage string, startedAt time.Time, phaseDuration time.Duration, err error) {
	if ctx.Err() != nil {
		return
	}
	log.WithError(err).WithFields(log.Fields{
		"chain_id":          chainID,
		"stage":             stage,
		"phase_duration_ms": phaseDuration.Milliseconds(),
		"duration_ms":       time.Since(startedAt).Milliseconds(),
	}).Error("token chain processor run failed")
}

func logProcessorBlockFailure(ctx context.Context, chainID int64, blockNumber uint64, stage string, startedAt time.Time, phaseDuration time.Duration, err error) {
	if ctx.Err() != nil {
		return
	}
	log.WithError(err).WithFields(log.Fields{
		"block_number":      blockNumber,
		"chain_id":          chainID,
		"stage":             stage,
		"phase_duration_ms": phaseDuration.Milliseconds(),
		"duration_ms":       time.Since(startedAt).Milliseconds(),
	}).Error("token chain block processing failed")
}

func (processor *ChainProcessor) estimateInitialStartBlock(ctx context.Context, chainID int64, latest discovery.BlockHeader, lookbackSeconds, targetTimestamp uint64) (initialBlockEstimate, error) {
	estimate := initialBlockEstimate{StartBlock: 1, EstimatedLookbackBlocks: latest.Number}
	if latest.Number <= initialBlockTimeSampleSize || targetTimestamp == 0 {
		return estimate, nil
	}

	sampleBlock := latest.Number - initialBlockTimeSampleSize
	sample, err := processor.blocks.BlockHeaderByNumber(ctx, chainID, sampleBlock)
	if err != nil {
		return initialBlockEstimate{}, err
	}
	if sample.Number != sampleBlock {
		return initialBlockEstimate{}, fmt.Errorf("token chain processor block-time sample returned block %d for requested block %d", sample.Number, sampleBlock)
	}
	if sample.Timestamp >= latest.Timestamp {
		return initialBlockEstimate{}, fmt.Errorf("token chain processor block-time sample must have an earlier timestamp: sample=%d latest=%d", sample.Timestamp, latest.Timestamp)
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
