package tokenapi

import (
	"context"
	"strings"

	"github.com/useryege/athena/internal/token/domain"
	"github.com/useryege/athena/internal/tokenapi/apiclient"
)

func (s *Service) GetChainIngestCheckpoint(ctx context.Context, req *apiclient.GetChainIngestCheckpointRequest) (*apiclient.GetChainIngestCheckpointResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	if err := validatePositiveInt64Field("chain_id", req.GetChainId()); err != nil {
		return nil, err
	}
	item, err := store.GetChainIngestCheckpoint(ctx, req.GetChainId())
	if err != nil {
		return nil, wrapStoreError("get chain ingest checkpoint", err)
	}
	if item == nil {
		return &apiclient.GetChainIngestCheckpointResponse{}, nil
	}
	return &apiclient.GetChainIngestCheckpointResponse{
		Found:      true,
		Checkpoint: mapChainIngestCheckpoint(*item),
	}, nil
}

func (s *Service) ListChainIngestCheckpoints(ctx context.Context, _ *apiclient.ListChainIngestCheckpointsRequest) (*apiclient.ListChainIngestCheckpointsResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	items, err := store.ListChainIngestCheckpoints(ctx)
	if err != nil {
		return nil, wrapStoreError("list chain ingest checkpoints", err)
	}
	return &apiclient.ListChainIngestCheckpointsResponse{
		Checkpoints: mapChainIngestCheckpoints(items),
	}, nil
}

func (s *Service) UpdateChainIngestCheckpoint(ctx context.Context, req *apiclient.UpdateChainIngestCheckpointRequest) (*apiclient.UpdateChainIngestCheckpointResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	if err := validatePositiveInt64Field("chain_id", req.GetChainId()); err != nil {
		return nil, err
	}
	status := strings.TrimSpace(req.GetStatus())
	if err := validateChainIngestStatus(status); err != nil {
		return nil, err
	}
	item, err := store.UpdateChainIngestCheckpointStatus(ctx, req.GetChainId(), domain.ChainIngestStatus(status))
	if err != nil {
		return nil, wrapStoreError("update chain ingest checkpoint", err)
	}
	if item == nil {
		return &apiclient.UpdateChainIngestCheckpointResponse{}, nil
	}
	return &apiclient.UpdateChainIngestCheckpointResponse{
		Found:      true,
		Checkpoint: mapChainIngestCheckpoint(*item),
	}, nil
}
