package tokenapi

import (
	"context"
	"time"

	"github.com/useryege/athena/internal/tokenapi/apiclient"
	applicationv1alpha1 "github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

func (s *Service) ListNodeStatuses(ctx context.Context, _ *apiclient.ListNodeStatusesRequest) (*apiclient.ListNodeStatusesResponse, error) {
	operations, err := s.operationsApplication()
	if err != nil {
		return nil, err
	}
	items, err := operations.ListNodeStatuses(ctx)
	if err != nil {
		return nil, err
	}
	statuses := make([]*applicationv1alpha1.TokenNodeStatus, 0, len(items))
	for _, item := range items {
		status := &applicationv1alpha1.TokenNodeStatus{ChainID: item.ChainID, ChainName: item.ChainName, Endpoint: item.Endpoint, Available: item.Available, LatencyMS: item.Latency.Milliseconds(), ReportedChainID: item.ReportedChainID, LatestBlockNumber: item.LatestBlockNumber, ReferenceBlockNumber: item.ReferenceBlockNumber, BlockLag: item.BlockLag, Syncing: item.Syncing, CheckedAt: item.CheckedAt.Format(time.RFC3339Nano), Error: item.Error}
		if !item.LatestBlockTime.IsZero() {
			status.LatestBlockTime = item.LatestBlockTime.Format(time.RFC3339)
		}
		statuses = append(statuses, status)
	}
	return &apiclient.ListNodeStatusesResponse{NodeStatuses: statuses}, nil
}
