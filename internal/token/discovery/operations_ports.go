package discovery

import "context"

type OperationsRepository interface {
	ListChains(context.Context) ([]Chain, error)
	GetChainIngestCheckpoint(context.Context, int64) (*ChainIngestCheckpoint, error)
	ListChainIngestCheckpoints(context.Context) ([]ChainIngestCheckpoint, error)
	UpdateChainIngestCheckpointStatus(context.Context, int64, ChainIngestStatus) (*ChainIngestCheckpoint, error)
}
