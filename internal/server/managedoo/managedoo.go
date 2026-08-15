package managedoo

import (
	"context"

	managedooapiclient "github.com/useryege/athena/internal/managedoo/apiclient"
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
	closer, client, err := s.managedOOClientset.NewManagedOOServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetManagedOOStatus(ctx, &managedooapiclient.GetManagedOOStatusRequest{})
	if err != nil {
		return nil, err
	}

	return &managedoopkg.GetManagedOOStatusResponse{
		Started: resp.GetStarted(),
		Status:  resp.GetStatus(),
	}, nil
}

func (s *Server) ScanManagedOOBlock(ctx context.Context, req *managedoopkg.ScanManagedOOBlockRequest) (*managedoopkg.ScanManagedOOBlockResponse, error) {
	closer, client, err := s.managedOOClientset.NewManagedOOServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ScanManagedOOBlock(ctx, &managedooapiclient.ScanManagedOOBlockRequest{BlockNumber: req.GetBlockNumber()})
	if err != nil {
		return nil, err
	}

	return &managedoopkg.ScanManagedOOBlockResponse{
		BlockNumber:   resp.GetBlockNumber(),
		ProposalCount: resp.GetProposalCount(),
		DisputeCount:  resp.GetDisputeCount(),
	}, nil
}

func (s *Server) ListManagedOOProposals(ctx context.Context, req *managedoopkg.ListManagedOOProposalsRequest) (*managedoopkg.ListManagedOOProposalsResponse, error) {
	closer, client, err := s.managedOOClientset.NewManagedOOServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListManagedOOProposals(ctx, &managedooapiclient.ListManagedOOProposalsRequest{
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
	closer, client, err := s.managedOOClientset.NewManagedOOServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListManagedOODisputes(ctx, &managedooapiclient.ListManagedOODisputesRequest{
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
