package tokenapi

import (
	"context"

	tokenapiapiclient "github.com/useryege/athena/internal/tokenapi/apiclient"
	tokenapipkg "github.com/useryege/athena/pkg/apiclient/tokenapi"
)

type Server struct {
	tokenapipkg.UnimplementedTokenAPIServiceServer
	tokenAPIClientSet tokenapiapiclient.Clientset
}

func NewServer(tokenAPIClientSet tokenapiapiclient.Clientset) *Server {
	return &Server{tokenAPIClientSet: tokenAPIClientSet}
}

func (s *Server) GetOptions(ctx context.Context, _ *tokenapipkg.GetOptionsRequest) (*tokenapipkg.GetOptionsResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetOptions(ctx, &tokenapiapiclient.GetOptionsRequest{})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.GetOptionsResponse{Options: resp.GetOptions()}, nil
}

func (s *Server) ListNodeStatuses(ctx context.Context, _ *tokenapipkg.ListNodeStatusesRequest) (*tokenapipkg.ListNodeStatusesResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListNodeStatuses(ctx, &tokenapiapiclient.ListNodeStatusesRequest{})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.ListNodeStatusesResponse{NodeStatuses: resp.GetNodeStatuses()}, nil
}

func (s *Server) GetBytecodeBlacklist(ctx context.Context, req *tokenapipkg.GetBytecodeBlacklistRequest) (*tokenapipkg.GetBytecodeBlacklistResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetBytecodeBlacklist(ctx, &tokenapiapiclient.GetBytecodeBlacklistRequest{
		CodeHash: req.GetCodeHash(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.GetBytecodeBlacklistResponse{
		Found:             resp.GetFound(),
		BytecodeBlacklist: resp.GetBytecodeBlacklist(),
	}, nil
}

func (s *Server) ListBytecodeBlacklists(ctx context.Context, _ *tokenapipkg.ListBytecodeBlacklistsRequest) (*tokenapipkg.ListBytecodeBlacklistsResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListBytecodeBlacklists(ctx, &tokenapiapiclient.ListBytecodeBlacklistsRequest{})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.ListBytecodeBlacklistsResponse{BytecodeBlacklists: resp.GetBytecodeBlacklists()}, nil
}

func (s *Server) CreateBytecodeBlacklist(ctx context.Context, req *tokenapipkg.CreateBytecodeBlacklistRequest) (*tokenapipkg.CreateBytecodeBlacklistResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.CreateBytecodeBlacklist(ctx, &tokenapiapiclient.CreateBytecodeBlacklistRequest{
		Note:           req.GetNote(),
		SourceChainId:  req.GetSourceChainId(),
		SourceContract: req.GetSourceContract(),
	}); err != nil {
		return nil, err
	}
	return &tokenapipkg.CreateBytecodeBlacklistResponse{}, nil
}

func (s *Server) UpdateBytecodeBlacklist(ctx context.Context, req *tokenapipkg.UpdateBytecodeBlacklistRequest) (*tokenapipkg.UpdateBytecodeBlacklistResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateBytecodeBlacklist(ctx, &tokenapiapiclient.UpdateBytecodeBlacklistRequest{
		CodeHash: req.GetCodeHash(),
		Note:     req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.UpdateBytecodeBlacklistResponse{UpdatedCount: resp.GetUpdatedCount()}, nil
}

func (s *Server) DeleteBytecodeBlacklist(ctx context.Context, req *tokenapipkg.DeleteBytecodeBlacklistRequest) (*tokenapipkg.DeleteBytecodeBlacklistResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.DeleteBytecodeBlacklist(ctx, &tokenapiapiclient.DeleteBytecodeBlacklistRequest{
		CodeHash: req.GetCodeHash(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.DeleteBytecodeBlacklistResponse{DeletedCount: resp.GetDeletedCount()}, nil
}

func (s *Server) GetWalletBlacklist(ctx context.Context, req *tokenapipkg.GetWalletBlacklistRequest) (*tokenapipkg.GetWalletBlacklistResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetWalletBlacklist(ctx, &tokenapiapiclient.GetWalletBlacklistRequest{
		Wallet: req.GetWallet(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.GetWalletBlacklistResponse{
		Found:           resp.GetFound(),
		WalletBlacklist: resp.GetWalletBlacklist(),
	}, nil
}

func (s *Server) ListWalletBlacklists(ctx context.Context, _ *tokenapipkg.ListWalletBlacklistsRequest) (*tokenapipkg.ListWalletBlacklistsResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListWalletBlacklists(ctx, &tokenapiapiclient.ListWalletBlacklistsRequest{})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.ListWalletBlacklistsResponse{WalletBlacklists: resp.GetWalletBlacklists()}, nil
}

func (s *Server) CreateWalletBlacklist(ctx context.Context, req *tokenapipkg.CreateWalletBlacklistRequest) (*tokenapipkg.CreateWalletBlacklistResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.CreateWalletBlacklist(ctx, &tokenapiapiclient.CreateWalletBlacklistRequest{
		Wallet: req.GetWallet(),
		Note:   req.GetNote(),
	}); err != nil {
		return nil, err
	}
	return &tokenapipkg.CreateWalletBlacklistResponse{}, nil
}

func (s *Server) UpdateWalletBlacklist(ctx context.Context, req *tokenapipkg.UpdateWalletBlacklistRequest) (*tokenapipkg.UpdateWalletBlacklistResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateWalletBlacklist(ctx, &tokenapiapiclient.UpdateWalletBlacklistRequest{
		Wallet: req.GetWallet(),
		Note:   req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.UpdateWalletBlacklistResponse{UpdatedCount: resp.GetUpdatedCount()}, nil
}

func (s *Server) DeleteWalletBlacklist(ctx context.Context, req *tokenapipkg.DeleteWalletBlacklistRequest) (*tokenapipkg.DeleteWalletBlacklistResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.DeleteWalletBlacklist(ctx, &tokenapiapiclient.DeleteWalletBlacklistRequest{
		Wallet: req.GetWallet(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.DeleteWalletBlacklistResponse{DeletedCount: resp.GetDeletedCount()}, nil
}

func (s *Server) GetChainIngestCheckpoint(ctx context.Context, req *tokenapipkg.GetChainIngestCheckpointRequest) (*tokenapipkg.GetChainIngestCheckpointResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetChainIngestCheckpoint(ctx, &tokenapiapiclient.GetChainIngestCheckpointRequest{
		ChainId: req.GetChainId(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.GetChainIngestCheckpointResponse{
		Found:      resp.GetFound(),
		Checkpoint: resp.GetCheckpoint(),
	}, nil
}

func (s *Server) ListChainIngestCheckpoints(ctx context.Context, _ *tokenapipkg.ListChainIngestCheckpointsRequest) (*tokenapipkg.ListChainIngestCheckpointsResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListChainIngestCheckpoints(ctx, &tokenapiapiclient.ListChainIngestCheckpointsRequest{})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.ListChainIngestCheckpointsResponse{Checkpoints: resp.GetCheckpoints()}, nil
}

func (s *Server) UpdateChainIngestCheckpoint(ctx context.Context, req *tokenapipkg.UpdateChainIngestCheckpointRequest) (*tokenapipkg.UpdateChainIngestCheckpointResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateChainIngestCheckpoint(ctx, &tokenapiapiclient.UpdateChainIngestCheckpointRequest{
		ChainId: req.GetChainId(),
		Status:  req.GetStatus(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.UpdateChainIngestCheckpointResponse{
		Found:      resp.GetFound(),
		Checkpoint: resp.GetCheckpoint(),
	}, nil
}

func (s *Server) GetContractCode(ctx context.Context, req *tokenapipkg.GetContractCodeRequest) (*tokenapipkg.GetContractCodeResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetContractCode(ctx, &tokenapiapiclient.GetContractCodeRequest{
		CodeHash: req.GetCodeHash(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.GetContractCodeResponse{
		Found:        resp.GetFound(),
		ContractCode: resp.GetContractCode(),
	}, nil
}

func (s *Server) ListContractCodes(ctx context.Context, req *tokenapipkg.ListContractCodesRequest) (*tokenapipkg.ListContractCodesResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListContractCodes(ctx, &tokenapiapiclient.ListContractCodesRequest{
		CodeHash: req.GetCodeHash(),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
		OrderBy:  req.GetOrderBy(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.ListContractCodesResponse{
		ContractCodes: resp.GetContractCodes(),
		Total:         resp.GetTotal(),
		Page:          resp.GetPage(),
		PageSize:      resp.GetPageSize(),
	}, nil
}

func (s *Server) ListProjects(ctx context.Context, req *tokenapipkg.ListProjectsRequest) (*tokenapipkg.ListProjectsResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListProjects(ctx, &tokenapiapiclient.ListProjectsRequest{
		ChainId:  req.GetChainId(),
		CodeHash: req.GetCodeHash(),
		Contract: req.GetContract(),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.ListProjectsResponse{
		Projects: resp.GetProjects(),
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
	}, nil
}

func (s *Server) ListProjectReports(ctx context.Context, req *tokenapipkg.ListProjectReportsRequest) (*tokenapipkg.ListProjectReportsResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListProjectReports(ctx, &tokenapiapiclient.ListProjectReportsRequest{
		ChainId:          req.GetChainId(),
		ProjectId:        req.GetProjectId(),
		Contract:         req.GetContract(),
		EvaluationStatus: req.GetEvaluationStatus(),
		Page:             req.GetPage(),
		PageSize:         req.GetPageSize(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.ListProjectReportsResponse{
		ProjectReports: resp.GetProjectReports(),
		Total:          resp.GetTotal(),
		Page:           resp.GetPage(),
		PageSize:       resp.GetPageSize(),
	}, nil
}

func (s *Server) GetProjectDataCollectionTask(ctx context.Context, req *tokenapipkg.GetProjectDataCollectionTaskRequest) (*tokenapipkg.GetProjectDataCollectionTaskResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetProjectDataCollectionTask(ctx, &tokenapiapiclient.GetProjectDataCollectionTaskRequest{
		ProjectId: req.GetProjectId(),
		DataType:  req.GetDataType(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.GetProjectDataCollectionTaskResponse{
		Found: resp.GetFound(),
		Task:  resp.GetTask(),
	}, nil
}

func (s *Server) ListProjectDataCollectionTasks(ctx context.Context, req *tokenapipkg.ListProjectDataCollectionTasksRequest) (*tokenapipkg.ListProjectDataCollectionTasksResponse, error) {
	closer, client, err := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListProjectDataCollectionTasks(ctx, &tokenapiapiclient.ListProjectDataCollectionTasksRequest{
		ProjectId: req.GetProjectId(),
		DataType:  req.GetDataType(),
		Status:    req.GetStatus(),
		Page:      req.GetPage(),
		PageSize:  req.GetPageSize(),
	})
	if err != nil {
		return nil, err
	}
	return &tokenapipkg.ListProjectDataCollectionTasksResponse{
		Tasks:    resp.GetTasks(),
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
	}, nil
}
