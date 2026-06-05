package api

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const manualCollectionReason = "manual"

func (s *Service) ListChains(ctx context.Context, _ *applicationpkg.ListChainsRequest) (*applicationpkg.ListChainsResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	items, err := s.store.ListChainIngestCheckpoints(ctx)
	if err != nil {
		return nil, err
	}
	resp := &applicationpkg.ListChainsResponse{Items: make([]*v1alpha1.ApplicationChain, 0, len(items))}
	for _, item := range items {
		resp.Items = append(resp.Items, &v1alpha1.ApplicationChain{
			ChainID: item.ChainID,
			Name:    item.ChainName,
			Enabled: item.Enabled,
		})
	}
	return resp, nil
}

func (s *Service) GetChainIngestStatus(ctx context.Context, req *applicationpkg.GetChainIngestStatusRequest) (*v1alpha1.ChainIngestStatus, error) {
	item, err := s.chainIngestCheckpoint(ctx, req.GetChainId())
	if err != nil {
		return nil, err
	}
	return chainIngestStatusToAPI(item), nil
}

func (s *Service) StartChainIngest(ctx context.Context, req *applicationpkg.StartChainIngestRequest) (*v1alpha1.ChainIngestStatus, error) {
	return s.setChainIngestStatus(ctx, req.GetChainId(), appstore.ChainIngestStatusRunning)
}

func (s *Service) StopChainIngest(ctx context.Context, req *applicationpkg.StopChainIngestRequest) (*v1alpha1.ChainIngestStatus, error) {
	return s.setChainIngestStatus(ctx, req.GetChainId(), appstore.ChainIngestStatusStopped)
}

func (s *Service) RequestProjectCollection(ctx context.Context, req *applicationpkg.RequestProjectCollectionRequest) (*applicationpkg.RequestProjectCollectionResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	chainID, contract, err := parseProjectRefRequest(req.GetChainId(), req.GetContract())
	if err != nil {
		return nil, err
	}
	base, err := s.store.GetProjectBaseByContract(ctx, chainID, contract)
	if err != nil {
		return nil, err
	}
	if base == nil {
		return nil, status.Errorf(codes.NotFound, "project %d/%s not found", chainID, contract.Hex())
	}
	reason := strings.TrimSpace(req.GetReason())
	if reason == "" {
		reason = manualCollectionReason
	}
	if err := s.store.EnqueueProjectCollection(ctx, model.ProjectRef{ChainID: chainID, Contract: contract}, reason); err != nil {
		return nil, err
	}
	item, err := s.store.GetProjectCollectionState(ctx, chainID, contract)
	if err != nil {
		return nil, err
	}
	return &applicationpkg.RequestProjectCollectionResponse{Status: projectCollectionStatusToAPI(item, chainID, contract)}, nil
}

func (s *Service) GetProjectCollectionStatus(ctx context.Context, req *applicationpkg.GetProjectCollectionStatusRequest) (*v1alpha1.ProjectCollectionStatus, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	chainID, contract, err := parseProjectRefRequest(req.GetChainId(), req.GetContract())
	if err != nil {
		return nil, err
	}
	base, err := s.store.GetProjectBaseByContract(ctx, chainID, contract)
	if err != nil {
		return nil, err
	}
	if base == nil {
		return nil, status.Errorf(codes.NotFound, "project %d/%s not found", chainID, contract.Hex())
	}
	item, err := s.store.GetProjectCollectionState(ctx, chainID, contract)
	if err != nil {
		return nil, err
	}
	return projectCollectionStatusToAPI(item, chainID, contract), nil
}

func (s *Service) chainIngestCheckpoint(ctx context.Context, chainID int64) (*appstore.ChainIngestCheckpoint, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	if chainID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "chain_id must be positive")
	}
	item, err := s.store.GetChainIngestCheckpoint(ctx, chainID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, status.Errorf(codes.NotFound, "chain %d not found", chainID)
	}
	return item, nil
}

func (s *Service) setChainIngestStatus(ctx context.Context, chainID int64, statusText string) (*v1alpha1.ChainIngestStatus, error) {
	current, err := s.chainIngestCheckpoint(ctx, chainID)
	if err != nil {
		return nil, err
	}
	current.Status = statusText
	updated, err := s.store.UpsertChainIngestCheckpoint(ctx, *current)
	if err != nil {
		return nil, err
	}
	return chainIngestStatusToAPI(updated), nil
}

func parseProjectRefRequest(chainID int64, contractValue string) (int64, common.Address, error) {
	if chainID <= 0 {
		return 0, common.Address{}, status.Error(codes.InvalidArgument, "chain_id must be positive")
	}
	if !common.IsHexAddress(contractValue) {
		return 0, common.Address{}, status.Errorf(codes.InvalidArgument, "invalid contract %q", contractValue)
	}
	contract := common.HexToAddress(contractValue)
	if contract == (common.Address{}) {
		return 0, common.Address{}, status.Error(codes.InvalidArgument, "contract cannot be zero address")
	}
	return chainID, contract, nil
}

func chainIngestStatusToAPI(item *appstore.ChainIngestCheckpoint) *v1alpha1.ChainIngestStatus {
	if item == nil {
		return nil
	}
	return &v1alpha1.ChainIngestStatus{
		ChainID:              item.ChainID,
		Name:                 item.ChainName,
		Enabled:              item.Enabled,
		Status:               item.Status,
		FinalizedBlockNumber: item.FinalizedBlockNumber,
		FinalizedBlockHash:   optionalHashHex(item.FinalizedBlockHash),
		CursorBlockNumber:    item.CursorBlockNumber,
		CursorBlockHash:      optionalHashHex(item.CursorBlockHash),
		UpdatedAt:            formatAPITime(item.UpdatedAt),
	}
}

func projectCollectionStatusToAPI(item *appstore.ProjectCollectionState, chainID int64, contract common.Address) *v1alpha1.ProjectCollectionStatus {
	if item == nil {
		return &v1alpha1.ProjectCollectionStatus{
			ChainID:  chainID,
			Contract: contract.Hex(),
			Status:   appstore.ProjectCollectionStatusNotRequested,
		}
	}
	resp := &v1alpha1.ProjectCollectionStatus{
		ChainID:         item.ChainID,
		Contract:        item.ProjectContract.Hex(),
		Status:          item.Status,
		WorkflowID:      item.WorkflowID,
		LastRequestedAt: formatAPITime(item.LastRequestedAt),
		LastStartedAt:   formatAPITime(item.LastStartedAt),
		LastCompletedAt: formatAPITime(item.LastCompletedAt),
		NextRunAt:       formatAPITime(item.NextRunAt),
		LastError:       item.LastError,
		UpdatedAt:       formatAPITime(item.UpdatedAt),
		Components:      make([]*v1alpha1.ProjectComponentStatus, 0, len(item.ComponentStates)),
	}
	if resp.Status == "" {
		resp.Status = appstore.ProjectCollectionStatusNotRequested
	}
	for _, component := range item.ComponentStates {
		resp.Components = append(resp.Components, projectComponentStatusToAPI(component))
	}
	return resp
}

func projectComponentStatusToAPI(item appstore.ProjectComponentState) *v1alpha1.ProjectComponentStatus {
	return &v1alpha1.ProjectComponentStatus{
		Component:     item.Component,
		Status:        item.Status,
		LastAttemptAt: formatAPITime(item.LastAttemptAt),
		LastSuccessAt: formatAPITime(item.LastSuccessAt),
		NextRunAt:     formatAPITime(item.NextRunAt),
		LastError:     item.LastError,
		UpdatedAt:     formatAPITime(item.UpdatedAt),
	}
}

func optionalHashHex(value common.Hash) string {
	if value == (common.Hash{}) {
		return ""
	}
	return value.Hex()
}

func formatAPITime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
