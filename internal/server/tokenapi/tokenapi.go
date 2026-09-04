package tokenapi

import (
	"context"

	tokenapiapiclient "github.com/useryege/athena/internal/tokenapi/apiclient"
	tokenapipkg "github.com/useryege/athena/pkg/apiclient/tokenapi"
)

type Server struct {
	tokenapipkg.UnimplementedTokenCatalogServiceServer
	tokenapipkg.UnimplementedTokenCollectionServiceServer
	tokenapipkg.UnimplementedTokenPolicyServiceServer
	tokenapipkg.UnimplementedTokenOperationsServiceServer
	tokenAPIClientSet tokenapiapiclient.Clientset
}

func NewServer(clientset tokenapiapiclient.Clientset) *Server {
	return &Server{tokenAPIClientSet: clientset}
}

func (s *Server) GetRuntimeConfiguration(ctx context.Context, _ *tokenapipkg.GetRuntimeConfigurationRequest) (*tokenapipkg.GetRuntimeConfigurationResponse, error) {
	client := s.tokenAPIClientSet.Operations()
	r, e := client.GetRuntimeConfiguration(ctx, &tokenapiapiclient.GetRuntimeConfigurationRequest{})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetRuntimeConfigurationResponse{Configuration: r.GetConfiguration()}, nil
}
func (s *Server) ListNodeStatuses(ctx context.Context, _ *tokenapipkg.ListNodeStatusesRequest) (*tokenapipkg.ListNodeStatusesResponse, error) {
	client := s.tokenAPIClientSet.Operations()
	r, e := client.ListNodeStatuses(ctx, &tokenapiapiclient.ListNodeStatusesRequest{})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListNodeStatusesResponse{NodeStatuses: r.GetNodeStatuses()}, nil
}

func (s *Server) GetContractCodeBlocklistEntry(ctx context.Context, req *tokenapipkg.GetContractCodeBlocklistEntryRequest) (*tokenapipkg.GetContractCodeBlocklistEntryResponse, error) {
	client := s.tokenAPIClientSet.Policy()
	r, e := client.GetContractCodeBlocklistEntry(ctx, &tokenapiapiclient.GetContractCodeBlocklistEntryRequest{CodeHash: req.GetCodeHash()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetContractCodeBlocklistEntryResponse{Found: r.GetFound(), Entry: r.GetEntry()}, nil
}
func (s *Server) ListContractCodeBlocklistEntries(ctx context.Context, _ *tokenapipkg.ListContractCodeBlocklistEntriesRequest) (*tokenapipkg.ListContractCodeBlocklistEntriesResponse, error) {
	client := s.tokenAPIClientSet.Policy()
	r, e := client.ListContractCodeBlocklistEntries(ctx, &tokenapiapiclient.ListContractCodeBlocklistEntriesRequest{})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListContractCodeBlocklistEntriesResponse{Entries: r.GetEntries()}, nil
}
func (s *Server) CreateContractCodeBlocklistEntry(ctx context.Context, req *tokenapipkg.CreateContractCodeBlocklistEntryRequest) (*tokenapipkg.CreateContractCodeBlocklistEntryResponse, error) {
	client := s.tokenAPIClientSet.Policy()
	_, e := client.CreateContractCodeBlocklistEntry(ctx, &tokenapiapiclient.CreateContractCodeBlocklistEntryRequest{Note: req.GetNote(), SourceChainId: req.GetSourceChainId(), SourceContract: req.GetSourceContract()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.CreateContractCodeBlocklistEntryResponse{}, nil
}
func (s *Server) UpdateContractCodeBlocklistEntry(ctx context.Context, req *tokenapipkg.UpdateContractCodeBlocklistEntryRequest) (*tokenapipkg.UpdateContractCodeBlocklistEntryResponse, error) {
	client := s.tokenAPIClientSet.Policy()
	r, e := client.UpdateContractCodeBlocklistEntry(ctx, &tokenapiapiclient.UpdateContractCodeBlocklistEntryRequest{CodeHash: req.GetCodeHash(), Note: req.GetNote()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.UpdateContractCodeBlocklistEntryResponse{UpdatedCount: r.GetUpdatedCount()}, nil
}
func (s *Server) DeleteContractCodeBlocklistEntry(ctx context.Context, req *tokenapipkg.DeleteContractCodeBlocklistEntryRequest) (*tokenapipkg.DeleteContractCodeBlocklistEntryResponse, error) {
	client := s.tokenAPIClientSet.Policy()
	r, e := client.DeleteContractCodeBlocklistEntry(ctx, &tokenapiapiclient.DeleteContractCodeBlocklistEntryRequest{CodeHash: req.GetCodeHash()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.DeleteContractCodeBlocklistEntryResponse{DeletedCount: r.GetDeletedCount()}, nil
}

func (s *Server) GetWalletBlocklistEntry(ctx context.Context, req *tokenapipkg.GetWalletBlocklistEntryRequest) (*tokenapipkg.GetWalletBlocklistEntryResponse, error) {
	client := s.tokenAPIClientSet.Policy()
	r, e := client.GetWalletBlocklistEntry(ctx, &tokenapiapiclient.GetWalletBlocklistEntryRequest{Wallet: req.GetWallet()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetWalletBlocklistEntryResponse{Found: r.GetFound(), Entry: r.GetEntry()}, nil
}
func (s *Server) ListWalletBlocklistEntries(ctx context.Context, _ *tokenapipkg.ListWalletBlocklistEntriesRequest) (*tokenapipkg.ListWalletBlocklistEntriesResponse, error) {
	client := s.tokenAPIClientSet.Policy()
	r, e := client.ListWalletBlocklistEntries(ctx, &tokenapiapiclient.ListWalletBlocklistEntriesRequest{})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListWalletBlocklistEntriesResponse{Entries: r.GetEntries()}, nil
}
func (s *Server) CreateWalletBlocklistEntry(ctx context.Context, req *tokenapipkg.CreateWalletBlocklistEntryRequest) (*tokenapipkg.CreateWalletBlocklistEntryResponse, error) {
	client := s.tokenAPIClientSet.Policy()
	_, e := client.CreateWalletBlocklistEntry(ctx, &tokenapiapiclient.CreateWalletBlocklistEntryRequest{Wallet: req.GetWallet(), Note: req.GetNote()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.CreateWalletBlocklistEntryResponse{}, nil
}
func (s *Server) UpdateWalletBlocklistEntry(ctx context.Context, req *tokenapipkg.UpdateWalletBlocklistEntryRequest) (*tokenapipkg.UpdateWalletBlocklistEntryResponse, error) {
	client := s.tokenAPIClientSet.Policy()
	r, e := client.UpdateWalletBlocklistEntry(ctx, &tokenapiapiclient.UpdateWalletBlocklistEntryRequest{Wallet: req.GetWallet(), Note: req.GetNote()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.UpdateWalletBlocklistEntryResponse{UpdatedCount: r.GetUpdatedCount()}, nil
}
func (s *Server) DeleteWalletBlocklistEntry(ctx context.Context, req *tokenapipkg.DeleteWalletBlocklistEntryRequest) (*tokenapipkg.DeleteWalletBlocklistEntryResponse, error) {
	client := s.tokenAPIClientSet.Policy()
	r, e := client.DeleteWalletBlocklistEntry(ctx, &tokenapiapiclient.DeleteWalletBlocklistEntryRequest{Wallet: req.GetWallet()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.DeleteWalletBlocklistEntryResponse{DeletedCount: r.GetDeletedCount()}, nil
}

func (s *Server) GetChainCheckpoint(ctx context.Context, req *tokenapipkg.GetChainCheckpointRequest) (*tokenapipkg.GetChainCheckpointResponse, error) {
	client := s.tokenAPIClientSet.Operations()
	r, e := client.GetChainCheckpoint(ctx, &tokenapiapiclient.GetChainCheckpointRequest{ChainId: req.GetChainId()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetChainCheckpointResponse{Found: r.GetFound(), Checkpoint: r.GetCheckpoint()}, nil
}
func (s *Server) ListChainCheckpoints(ctx context.Context, _ *tokenapipkg.ListChainCheckpointsRequest) (*tokenapipkg.ListChainCheckpointsResponse, error) {
	client := s.tokenAPIClientSet.Operations()
	r, e := client.ListChainCheckpoints(ctx, &tokenapiapiclient.ListChainCheckpointsRequest{})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListChainCheckpointsResponse{Checkpoints: r.GetCheckpoints()}, nil
}
func (s *Server) UpdateChainCheckpoint(ctx context.Context, req *tokenapipkg.UpdateChainCheckpointRequest) (*tokenapipkg.UpdateChainCheckpointResponse, error) {
	client := s.tokenAPIClientSet.Operations()
	r, e := client.UpdateChainCheckpoint(ctx, &tokenapiapiclient.UpdateChainCheckpointRequest{ChainId: req.GetChainId(), Status: req.GetStatus()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.UpdateChainCheckpointResponse{Found: r.GetFound(), Checkpoint: r.GetCheckpoint()}, nil
}
func (s *Server) GetChainProcessingSummary(ctx context.Context, req *tokenapipkg.GetChainProcessingSummaryRequest) (*tokenapipkg.GetChainProcessingSummaryResponse, error) {
	client := s.tokenAPIClientSet.Operations()
	r, e := client.GetChainProcessingSummary(ctx, &tokenapiapiclient.GetChainProcessingSummaryRequest{
		ChainId:       req.GetChainId(),
		BlockNumber:   req.GetBlockNumber(),
		WindowSeconds: req.GetWindowSeconds(),
	})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetChainProcessingSummaryResponse{Summary: r.GetSummary()}, nil
}
func (s *Server) ListChainProcessingAttempts(ctx context.Context, req *tokenapipkg.ListChainProcessingAttemptsRequest) (*tokenapipkg.ListChainProcessingAttemptsResponse, error) {
	client := s.tokenAPIClientSet.Operations()
	r, e := client.ListChainProcessingAttempts(ctx, &tokenapiapiclient.ListChainProcessingAttemptsRequest{
		ChainId:       req.GetChainId(),
		BlockNumber:   req.GetBlockNumber(),
		WindowSeconds: req.GetWindowSeconds(),
		Status:        req.GetStatus(),
		Page:          req.GetPage(),
		PageSize:      req.GetPageSize(),
	})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListChainProcessingAttemptsResponse{Attempts: r.GetAttempts(), Total: r.GetTotal(), Page: r.GetPage(), PageSize: r.GetPageSize()}, nil
}
func (s *Server) GetContractCode(ctx context.Context, req *tokenapipkg.GetContractCodeRequest) (*tokenapipkg.GetContractCodeResponse, error) {
	client := s.tokenAPIClientSet.Catalog()
	r, e := client.GetContractCode(ctx, &tokenapiapiclient.GetContractCodeRequest{CodeHash: req.GetCodeHash()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetContractCodeResponse{Found: r.GetFound(), ContractCode: r.GetContractCode()}, nil
}
func (s *Server) ListContractCodes(ctx context.Context, req *tokenapipkg.ListContractCodesRequest) (*tokenapipkg.ListContractCodesResponse, error) {
	client := s.tokenAPIClientSet.Catalog()
	r, e := client.ListContractCodes(ctx, &tokenapiapiclient.ListContractCodesRequest{CodeHash: req.GetCodeHash(), Page: req.GetPage(), PageSize: req.GetPageSize(), OrderBy: req.GetOrderBy()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListContractCodesResponse{ContractCodes: r.GetContractCodes(), Total: r.GetTotal(), Page: r.GetPage(), PageSize: r.GetPageSize()}, nil
}
func (s *Server) ListProjects(ctx context.Context, req *tokenapipkg.ListProjectsRequest) (*tokenapipkg.ListProjectsResponse, error) {
	client := s.tokenAPIClientSet.Catalog()
	r, e := client.ListProjects(ctx, &tokenapiapiclient.ListProjectsRequest{
		CodeHash:                req.GetCodeHash(),
		Contract:                req.GetContract(),
		Page:                    req.GetPage(),
		PageSize:                req.GetPageSize(),
		CollectionStatus:        req.GetCollectionStatus(),
		ProfileState:            req.GetProfileState(),
		PairBalanceSupplyStates: req.GetPairBalanceSupplyStates(),
		PairMinimumLpStates:     req.GetPairMinimumLpStates(),
		PairFeeLpShareStates:    req.GetPairFeeLpShareStates(),
		PairQuoteUsdtMin:        req.GetPairQuoteUsdtMin(),
		PairQuoteUsdtMax:        req.GetPairQuoteUsdtMax(),
	})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListProjectsResponse{Projects: r.GetProjects(), Total: r.GetTotal(), Page: r.GetPage(), PageSize: r.GetPageSize()}, nil
}
func (s *Server) GetProjectProfile(ctx context.Context, req *tokenapipkg.GetProjectProfileRequest) (*tokenapipkg.GetProjectProfileResponse, error) {
	client := s.tokenAPIClientSet.Catalog()
	r, e := client.GetProjectProfile(ctx, &tokenapiapiclient.GetProjectProfileRequest{ProjectId: req.GetProjectId()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetProjectProfileResponse{Found: r.GetFound(), Profile: r.GetProfile()}, nil
}
func (s *Server) GetProjectDetail(ctx context.Context, req *tokenapipkg.GetProjectDetailRequest) (*tokenapipkg.GetProjectDetailResponse, error) {
	client := s.tokenAPIClientSet.Catalog()
	r, e := client.GetProjectDetail(ctx, &tokenapiapiclient.GetProjectDetailRequest{ProjectId: req.GetProjectId()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetProjectDetailResponse{Found: r.GetFound(), Detail: r.GetDetail()}, nil
}
func (s *Server) ListProjectWalletNormalTransactions(ctx context.Context, req *tokenapipkg.ListProjectWalletNormalTransactionsRequest) (*tokenapipkg.ListProjectWalletNormalTransactionsResponse, error) {
	client := s.tokenAPIClientSet.Catalog()
	r, e := client.ListProjectWalletNormalTransactions(ctx, &tokenapiapiclient.ListProjectWalletNormalTransactionsRequest{ProjectId: req.GetProjectId(), Wallet: req.GetWallet(), ReceiptStatus: req.GetReceiptStatus(), MethodId: req.GetMethodId(), Page: req.GetPage(), PageSize: req.GetPageSize()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListProjectWalletNormalTransactionsResponse{Transactions: r.GetTransactions(), Total: r.GetTotal(), Page: r.GetPage(), PageSize: r.GetPageSize()}, nil
}
func (s *Server) GetCollectionTask(ctx context.Context, req *tokenapipkg.GetCollectionTaskRequest) (*tokenapipkg.GetCollectionTaskResponse, error) {
	client := s.tokenAPIClientSet.Collection()
	r, e := client.GetCollectionTask(ctx, &tokenapiapiclient.GetCollectionTaskRequest{TaskId: req.GetTaskId()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetCollectionTaskResponse{Found: r.GetFound(), Task: r.GetTask()}, nil
}
func (s *Server) ListCollectionTasks(ctx context.Context, req *tokenapipkg.ListCollectionTasksRequest) (*tokenapipkg.ListCollectionTasksResponse, error) {
	client := s.tokenAPIClientSet.Collection()
	r, e := client.ListCollectionTasks(ctx, &tokenapiapiclient.ListCollectionTasksRequest{ProjectId: req.GetProjectId(), DataType: req.GetDataType(), Status: req.GetStatus(), Page: req.GetPage(), PageSize: req.GetPageSize()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListCollectionTasksResponse{Tasks: r.GetTasks(), Total: r.GetTotal(), Page: r.GetPage(), PageSize: r.GetPageSize()}, nil
}
