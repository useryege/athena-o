package application

import (
	"context"
	"fmt"

	"github.com/useryege/athena/internal/token/discovery"
)

const maxBlocksPerScanBatch = uint64(100)

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
	LatestBlockNumber(context.Context, int64) (uint64, error)
	DiscoverProjectCandidates(context.Context, int64, uint64, uint64, int) ([]discovery.ProjectCandidate, error)
}

type ScanChainCommand struct {
	ChainID               int64
	BlockFetchConcurrency int
}

type ScanResult struct {
	Batches    int
	Blocks     uint64
	Candidates int
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
	latest, err := scanner.blocks.LatestBlockNumber(ctx, command.ChainID)
	if err != nil {
		return ScanResult{}, err
	}
	next := uint64(1)
	if checkpoint.CursorBlockNumber > 0 {
		next = checkpoint.CursorBlockNumber + 1
	}
	result := ScanResult{}
	for next <= latest {
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
		if batchEnd > latest {
			batchEnd = latest
		}
		candidates, err := scanner.blocks.DiscoverProjectCandidates(ctx, command.ChainID, next, batchEnd, command.BlockFetchConcurrency)
		if err != nil {
			return result, err
		}
		if _, err = scanner.repository.IngestProjectCandidateBatch(ctx, discovery.ChainIngestCheckpoint{ChainID: command.ChainID, CursorBlockNumber: batchEnd, Status: discovery.ChainIngestStatusRunning}, candidates); err != nil {
			return result, err
		}
		result.Batches++
		result.Blocks += batchEnd - next + 1
		result.Candidates += len(candidates)
		next = batchEnd + 1
	}
	return result, nil
}
