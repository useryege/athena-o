package chainscanner

import (
	"context"

	"github.com/useryege/athena/internal/token/domain"
)

type Store interface {
	GetChainIngestCheckpoint(context.Context, int64) (*domain.ChainIngestCheckpoint, error)
	UpsertChainIngestCheckpoint(context.Context, domain.ChainIngestCheckpoint) (*domain.ChainIngestCheckpoint, error)
	UpdateChainIngestCheckpointStatus(context.Context, int64, domain.ChainIngestStatus) (*domain.ChainIngestCheckpoint, error)
	IngestProjectCandidateBatch(context.Context, domain.ChainIngestCheckpoint, []domain.ProjectCandidate) (*domain.ChainIngestCheckpoint, error)
}
