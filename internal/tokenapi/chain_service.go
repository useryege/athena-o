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
	item, err := store.GetChainProcessingCheckpoint(ctx, req.GetChainId())
	if err != nil {
		return nil, wrapStoreError("get chain processing checkpoint", err)
	}
	if item == nil {
		return &apiclient.GetChainCheckpointResponse{}, nil
	}
	return &apiclient.GetChainCheckpointResponse{
		Found:      true,
		Checkpoint: mapChainProcessingCheckpoint(*item),
	}, nil
}

func (s *Service) ListChainCheckpoints(ctx context.Context, _ *apiclient.ListChainCheckpointsRequest) (*apiclient.ListChainCheckpointsResponse, error) {
	store, err := s.operationsApplication()
	if err != nil {
		return nil, err
	}
	items, err := store.ListChainProcessingCheckpoints(ctx)
	if err != nil {
		return nil, wrapStoreError("list chain processing checkpoints", err)
	}
	return &apiclient.ListChainCheckpointsResponse{
		Checkpoints: mapChainProcessingCheckpoints(items),
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
	if err := validateChainProcessingStatus(status); err != nil {
		return nil, err
	}
	item, err := store.UpdateChainProcessingCheckpointStatus(ctx, req.GetChainId(), discovery.ChainProcessingStatus(status))
	if err != nil {
		return nil, wrapStoreError("update chain processing checkpoint", err)
	}
	if item == nil {
		return &apiclient.UpdateChainCheckpointResponse{}, nil
	}
	return &apiclient.UpdateChainCheckpointResponse{
		Found:      true,
		Checkpoint: mapChainProcessingCheckpoint(*item),
	}, nil
}

func (s *Service) GetChainProcessingSummary(ctx context.Context, req *apiclient.GetChainProcessingSummaryRequest) (*apiclient.GetChainProcessingSummaryResponse, error) {
	store, err := s.operationsApplication()
	if err != nil {
		return nil, err
	}
	if err := validatePositiveInt64Field("chain_id", req.GetChainId()); err != nil {
		return nil, err
	}
	windowSeconds, err := normalizeChainProcessingWindow(req.GetWindowSeconds())
	if err != nil {
		return nil, err
	}
	item, err := store.GetChainBlockProcessingSummary(ctx, discovery.ChainBlockProcessingFilter{
		ChainID:       req.GetChainId(),
		BlockNumber:   req.GetBlockNumber(),
		WindowSeconds: windowSeconds,
	})
	if err != nil {
		return nil, wrapStoreError("get chain block processing summary", err)
	}
	return &apiclient.GetChainProcessingSummaryResponse{Summary: mapChainBlockProcessingSummary(*item)}, nil
}

func (s *Service) ListChainProcessingAttempts(ctx context.Context, req *apiclient.ListChainProcessingAttemptsRequest) (*apiclient.ListChainProcessingAttemptsResponse, error) {
	store, err := s.operationsApplication()
	if err != nil {
		return nil, err
	}
	if err := validatePositiveInt64Field("chain_id", req.GetChainId()); err != nil {
		return nil, err
	}
	windowSeconds, err := normalizeChainProcessingWindow(req.GetWindowSeconds())
	if err != nil {
		return nil, err
	}
	attemptStatus := strings.TrimSpace(req.GetStatus())
	if err := validateChainBlockProcessingAttemptStatus(attemptStatus); err != nil {
		return nil, err
	}
	page, err := store.ListChainBlockProcessingAttemptsPage(ctx, discovery.ChainBlockProcessingFilter{
		ChainID:       req.GetChainId(),
		BlockNumber:   req.GetBlockNumber(),
		WindowSeconds: windowSeconds,
		Status:        discovery.ChainBlockProcessingAttemptStatus(attemptStatus),
	}, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, wrapStoreError("list chain block processing attempts", err)
	}
	return &apiclient.ListChainProcessingAttemptsResponse{
		Attempts: mapChainBlockProcessingAttempts(page.Items),
		Total:    page.Total,
		Page:     page.Page,
		PageSize: page.PageSize,
	}, nil
}
