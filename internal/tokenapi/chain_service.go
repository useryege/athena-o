package tokenapi

import (
	"context"
	"strings"

	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/tokenapi/apiclient"
)

func (s *Service) GetChainCheckpoint(ctx context.Context, req *apiclient.GetChainCheckpointRequest) (*apiclient.GetChainCheckpointResponse, error) {
	store, err := s.operationsApplication()
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
		return &apiclient.GetChainCheckpointResponse{}, nil
	}
	return &apiclient.GetChainCheckpointResponse{
		Found:      true,
		Checkpoint: mapChainIngestCheckpoint(*item),
	}, nil
}

func (s *Service) ListChainCheckpoints(ctx context.Context, _ *apiclient.ListChainCheckpointsRequest) (*apiclient.ListChainCheckpointsResponse, error) {
	store, err := s.operationsApplication()
	if err != nil {
		return nil, err
	}
	items, err := store.ListChainIngestCheckpoints(ctx)
	if err != nil {
		return nil, wrapStoreError("list chain ingest checkpoints", err)
	}
	return &apiclient.ListChainCheckpointsResponse{
		Checkpoints: mapChainIngestCheckpoints(items),
	}, nil
}

func (s *Service) UpdateChainCheckpoint(ctx context.Context, req *apiclient.UpdateChainCheckpointRequest) (*apiclient.UpdateChainCheckpointResponse, error) {
	store, err := s.operationsApplication()
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
	item, err := store.UpdateChainIngestCheckpointStatus(ctx, req.GetChainId(), discovery.ChainIngestStatus(status))
	if err != nil {
		return nil, wrapStoreError("update chain ingest checkpoint", err)
	}
	if item == nil {
		return &apiclient.UpdateChainCheckpointResponse{}, nil
	}
	return &apiclient.UpdateChainCheckpointResponse{
		Found:      true,
		Checkpoint: mapChainIngestCheckpoint(*item),
	}, nil
}
