package application

import (
	"context"

	"github.com/useryege/athena/internal/token/discovery"
)

type Operations struct {
	repository OperationsRepository
	nodes      NodeStatusProvider
}

type OperationsRepository interface {
	ListChains(context.Context) ([]discovery.Chain, error)
	GetChainProcessingCheckpoint(context.Context, int64) (*discovery.ChainProcessingCheckpoint, error)
	ListChainProcessingCheckpoints(context.Context) ([]discovery.ChainProcessingCheckpoint, error)
	UpdateChainProcessingCheckpointStatus(context.Context, int64, discovery.ChainProcessingStatus) (*discovery.ChainProcessingCheckpoint, error)
	GetChainBlockProcessingSummary(context.Context, discovery.ChainBlockProcessingFilter) (*discovery.ChainBlockProcessingSummary, error)
	ListChainBlockProcessingAttemptsPage(context.Context, discovery.ChainBlockProcessingFilter, int32, int32) (*discovery.ChainBlockProcessingAttemptPage, error)
}

type NodeStatusProvider interface {
	ListNodeStatuses(context.Context) ([]discovery.NodeStatus, error)
}

func NewOperations(repository OperationsRepository, nodes NodeStatusProvider) *Operations {
	return &Operations{repository: repository, nodes: nodes}
}

func (o *Operations) ListChains(ctx context.Context) ([]discovery.Chain, error) {
	return o.repository.ListChains(ctx)
}
func (o *Operations) GetChainProcessingCheckpoint(ctx context.Context, chainID int64) (*discovery.ChainProcessingCheckpoint, error) {
	return o.repository.GetChainProcessingCheckpoint(ctx, chainID)
}
func (o *Operations) ListChainProcessingCheckpoints(ctx context.Context) ([]discovery.ChainProcessingCheckpoint, error) {
	return o.repository.ListChainProcessingCheckpoints(ctx)
}
func (o *Operations) UpdateChainProcessingCheckpointStatus(ctx context.Context, chainID int64, status discovery.ChainProcessingStatus) (*discovery.ChainProcessingCheckpoint, error) {
	return o.repository.UpdateChainProcessingCheckpointStatus(ctx, chainID, status)
}
func (o *Operations) GetChainBlockProcessingSummary(ctx context.Context, filter discovery.ChainBlockProcessingFilter) (*discovery.ChainBlockProcessingSummary, error) {
	return o.repository.GetChainBlockProcessingSummary(ctx, filter)
}
func (o *Operations) ListChainBlockProcessingAttemptsPage(ctx context.Context, filter discovery.ChainBlockProcessingFilter, page, pageSize int32) (*discovery.ChainBlockProcessingAttemptPage, error) {
	return o.repository.ListChainBlockProcessingAttemptsPage(ctx, filter, page, pageSize)
}

func (o *Operations) ListNodeStatuses(ctx context.Context) ([]discovery.NodeStatus, error) {
	return o.nodes.ListNodeStatuses(ctx)
}
