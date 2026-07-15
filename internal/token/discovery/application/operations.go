package application

import (
	"context"

	"github.com/useryege/athena/internal/token/discovery"
)

type Operations struct {
	repository discovery.OperationsRepository
	nodes      NodeStatusProvider
}

type NodeStatusProvider interface {
	ListNodeStatuses(context.Context) ([]discovery.NodeStatus, error)
}

func NewOperations(repository discovery.OperationsRepository, nodes NodeStatusProvider) *Operations {
	return &Operations{repository: repository, nodes: nodes}
}

func (o *Operations) ListChains(ctx context.Context) ([]discovery.Chain, error) {
	return o.repository.ListChains(ctx)
}
func (o *Operations) GetChainIngestCheckpoint(ctx context.Context, chainID int64) (*discovery.ChainIngestCheckpoint, error) {
	return o.repository.GetChainIngestCheckpoint(ctx, chainID)
}
func (o *Operations) ListChainIngestCheckpoints(ctx context.Context) ([]discovery.ChainIngestCheckpoint, error) {
	return o.repository.ListChainIngestCheckpoints(ctx)
}
func (o *Operations) UpdateChainIngestCheckpointStatus(ctx context.Context, chainID int64, status discovery.ChainIngestStatus) (*discovery.ChainIngestCheckpoint, error) {
	return o.repository.UpdateChainIngestCheckpointStatus(ctx, chainID, status)
}

func (o *Operations) ListNodeStatuses(ctx context.Context) ([]discovery.NodeStatus, error) {
	return o.nodes.ListNodeStatuses(ctx)
}
