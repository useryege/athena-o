package tokenapi

import (
	"context"
	"time"

	athenacommon "github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/tokenapi/apiclient"
	applicationv1alpha1 "github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/ethws"
)

func (s *Service) ListNodeStatuses(ctx context.Context, _ *apiclient.ListNodeStatusesRequest) (*apiclient.ListNodeStatusesResponse, error) {
	type chainProbe struct {
		chainID int64
		results []ethws.ProbeResult
		err     error
	}

	chainIDs := []int64{
		athenacommon.ChainIDEthereumMainnet,
		athenacommon.ChainIDBSCMainnet,
	}
	probeCh := make(chan chainProbe, len(chainIDs))
	for _, chainID := range chainIDs {
		endpoints, _ := s.nodeConfig(chainID)
		go func(chainID int64, endpoints []string) {
			results, err := ethws.ProbeEndpoints(ctx, endpoints, chainID, s.nodeWSUseProxy())
			probeCh <- chainProbe{chainID: chainID, results: results, err: err}
		}(chainID, endpoints)
	}

	byChain := make(map[int64][]ethws.ProbeResult, len(chainIDs))
	for range chainIDs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case probe := <-probeCh:
			if probe.err != nil {
				return nil, probe.err
			}
			byChain[probe.chainID] = probe.results
		}
	}

	statuses := make([]*applicationv1alpha1.TokenAPINodeStatus, 0)
	for _, chainID := range chainIDs {
		for _, result := range byChain[chainID] {
			status := &applicationv1alpha1.TokenAPINodeStatus{
				ChainID:              chainID,
				ChainName:            athenacommon.ChainName(chainID),
				Endpoint:             result.Endpoint,
				Available:            result.Available,
				LatencyMS:            result.Latency.Milliseconds(),
				ReportedChainID:      result.ReportedChainID,
				LatestBlockNumber:    result.LatestBlockNumber,
				ReferenceBlockNumber: result.ReferenceBlockNumber,
				BlockLag:             result.BlockLag,
				Syncing:              result.Syncing,
				CheckedAt:            result.CheckedAt.Format(time.RFC3339Nano),
			}
			if !result.LatestBlockTime.IsZero() {
				status.LatestBlockTime = result.LatestBlockTime.Format(time.RFC3339)
			}
			if result.Err != nil {
				status.Error = result.Err.Error()
			}
			statuses = append(statuses, status)
		}
	}
	return &apiclient.ListNodeStatusesResponse{NodeStatuses: statuses}, nil
}
