package managedoo

import (
	"context"
	"strconv"

	managedooapiclient "github.com/useryege/athena/internal/managedoo/apiclient"
	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
	managedoopkg "github.com/useryege/athena/pkg/apiclient/managedoo"
)

type Server struct {
	managedoopkg.UnimplementedManagedOOServiceServer
	managedOOClientset managedooapiclient.Clientset
}

func NewServer(managedOOClientset managedooapiclient.Clientset) *Server {
	return &Server{managedOOClientset: managedOOClientset}
}

func (s *Server) GetManagedOOStatus(ctx context.Context, _ *managedoopkg.GetManagedOOStatusRequest) (*managedoopkg.GetManagedOOStatusResponse, error) {
	resp, err := s.managedOOClientset.ManagedOO().GetManagedOOStatus(ctx, &managedooapiclient.GetManagedOOStatusRequest{})
	if err != nil {
		return nil, err
	}

	return &managedoopkg.GetManagedOOStatusResponse{
		Started: resp.GetStarted(),
		Status:  resp.GetStatus(),
	}, nil
}

func (s *Server) ScanManagedOOBlock(ctx context.Context, req *managedoopkg.ScanManagedOOBlockRequest) (*managedoopkg.ScanManagedOOBlockResponse, error) {
	resp, err := s.managedOOClientset.ManagedOO().ScanManagedOOBlock(ctx, &managedooapiclient.ScanManagedOOBlockRequest{BlockNumber: req.GetBlockNumber()})
	if err != nil {
		return nil, err
	}
	operationlogrecord.CaptureResource(ctx, "block", strconv.FormatUint(resp.GetBlockNumber(), 10))
	operationlogrecord.CaptureUint64(ctx, "blockNumber", resp.GetBlockNumber())
	operationlogrecord.CaptureInt64(ctx, "proposalCount", int64(resp.GetProposalCount()))
	operationlogrecord.CaptureInt64(ctx, "disputeCount", int64(resp.GetDisputeCount()))
	operationlogrecord.Commit(ctx, "MANAGED_OO_BLOCK_SCAN")

	return &managedoopkg.ScanManagedOOBlockResponse{
		BlockNumber:   resp.GetBlockNumber(),
		ProposalCount: resp.GetProposalCount(),
		DisputeCount:  resp.GetDisputeCount(),
	}, nil
}

func (s *Server) ListManagedOOProposals(ctx context.Context, req *managedoopkg.ListManagedOOProposalsRequest) (*managedoopkg.ListManagedOOProposalsResponse, error) {
	resp, err := s.managedOOClientset.ManagedOO().ListManagedOOProposals(ctx, &managedooapiclient.ListManagedOOProposalsRequest{
		Page:        req.GetPage(),
		PageSize:    req.GetPageSize(),
		BlockNumber: req.GetBlockNumber(),
	})
	if err != nil {
		return nil, err
	}

	return &managedoopkg.ListManagedOOProposalsResponse{
		Items:    resp.GetItems(),
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
	}, nil
}

func (s *Server) ListManagedOODisputes(ctx context.Context, req *managedoopkg.ListManagedOODisputesRequest) (*managedoopkg.ListManagedOODisputesResponse, error) {
	resp, err := s.managedOOClientset.ManagedOO().ListManagedOODisputes(ctx, &managedooapiclient.ListManagedOODisputesRequest{
		Page:        req.GetPage(),
		PageSize:    req.GetPageSize(),
		BlockNumber: req.GetBlockNumber(),
	})
	if err != nil {
		return nil, err
	}

	return &managedoopkg.ListManagedOODisputesResponse{
		Items:    resp.GetItems(),
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
	}, nil
}
