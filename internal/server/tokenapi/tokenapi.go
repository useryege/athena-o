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

func NewServer(clientset tokenapiapiclient.Clientset) *Server {
	return &Server{tokenAPIClientSet: clientset}
}

func (s *Server) GetOptions(ctx context.Context, _ *tokenapipkg.GetOptionsRequest) (*tokenapipkg.GetOptionsResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.GetOptions(ctx, &tokenapiapiclient.GetOptionsRequest{})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetOptionsResponse{Options: r.GetOptions()}, nil
}
func (s *Server) ListNodeStatuses(ctx context.Context, _ *tokenapipkg.ListNodeStatusesRequest) (*tokenapipkg.ListNodeStatusesResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.ListNodeStatuses(ctx, &tokenapiapiclient.ListNodeStatusesRequest{})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListNodeStatusesResponse{NodeStatuses: r.GetNodeStatuses()}, nil
}

func (s *Server) GetContractCodeBlocklistEntry(ctx context.Context, req *tokenapipkg.GetContractCodeBlocklistEntryRequest) (*tokenapipkg.GetContractCodeBlocklistEntryResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.GetContractCodeBlocklistEntry(ctx, &tokenapiapiclient.GetContractCodeBlocklistEntryRequest{CodeHash: req.GetCodeHash()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetContractCodeBlocklistEntryResponse{Found: r.GetFound(), Entry: r.GetEntry()}, nil
}
func (s *Server) ListContractCodeBlocklistEntries(ctx context.Context, _ *tokenapipkg.ListContractCodeBlocklistEntriesRequest) (*tokenapipkg.ListContractCodeBlocklistEntriesResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.ListContractCodeBlocklistEntries(ctx, &tokenapiapiclient.ListContractCodeBlocklistEntriesRequest{})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListContractCodeBlocklistEntriesResponse{Entries: r.GetEntries()}, nil
}
func (s *Server) CreateContractCodeBlocklistEntry(ctx context.Context, req *tokenapipkg.CreateContractCodeBlocklistEntryRequest) (*tokenapipkg.CreateContractCodeBlocklistEntryResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	_, e = client.CreateContractCodeBlocklistEntry(ctx, &tokenapiapiclient.CreateContractCodeBlocklistEntryRequest{Note: req.GetNote(), SourceChainId: req.GetSourceChainId(), SourceContract: req.GetSourceContract()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.CreateContractCodeBlocklistEntryResponse{}, nil
}
func (s *Server) UpdateContractCodeBlocklistEntry(ctx context.Context, req *tokenapipkg.UpdateContractCodeBlocklistEntryRequest) (*tokenapipkg.UpdateContractCodeBlocklistEntryResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.UpdateContractCodeBlocklistEntry(ctx, &tokenapiapiclient.UpdateContractCodeBlocklistEntryRequest{CodeHash: req.GetCodeHash(), Note: req.GetNote()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.UpdateContractCodeBlocklistEntryResponse{UpdatedCount: r.GetUpdatedCount()}, nil
}
func (s *Server) DeleteContractCodeBlocklistEntry(ctx context.Context, req *tokenapipkg.DeleteContractCodeBlocklistEntryRequest) (*tokenapipkg.DeleteContractCodeBlocklistEntryResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.DeleteContractCodeBlocklistEntry(ctx, &tokenapiapiclient.DeleteContractCodeBlocklistEntryRequest{CodeHash: req.GetCodeHash()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.DeleteContractCodeBlocklistEntryResponse{DeletedCount: r.GetDeletedCount()}, nil
}

func (s *Server) GetWalletBlocklistEntry(ctx context.Context, req *tokenapipkg.GetWalletBlocklistEntryRequest) (*tokenapipkg.GetWalletBlocklistEntryResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.GetWalletBlocklistEntry(ctx, &tokenapiapiclient.GetWalletBlocklistEntryRequest{Wallet: req.GetWallet()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetWalletBlocklistEntryResponse{Found: r.GetFound(), Entry: r.GetEntry()}, nil
}
func (s *Server) ListWalletBlocklistEntries(ctx context.Context, _ *tokenapipkg.ListWalletBlocklistEntriesRequest) (*tokenapipkg.ListWalletBlocklistEntriesResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.ListWalletBlocklistEntries(ctx, &tokenapiapiclient.ListWalletBlocklistEntriesRequest{})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListWalletBlocklistEntriesResponse{Entries: r.GetEntries()}, nil
}
func (s *Server) CreateWalletBlocklistEntry(ctx context.Context, req *tokenapipkg.CreateWalletBlocklistEntryRequest) (*tokenapipkg.CreateWalletBlocklistEntryResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	_, e = client.CreateWalletBlocklistEntry(ctx, &tokenapiapiclient.CreateWalletBlocklistEntryRequest{Wallet: req.GetWallet(), Note: req.GetNote()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.CreateWalletBlocklistEntryResponse{}, nil
}
func (s *Server) UpdateWalletBlocklistEntry(ctx context.Context, req *tokenapipkg.UpdateWalletBlocklistEntryRequest) (*tokenapipkg.UpdateWalletBlocklistEntryResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.UpdateWalletBlocklistEntry(ctx, &tokenapiapiclient.UpdateWalletBlocklistEntryRequest{Wallet: req.GetWallet(), Note: req.GetNote()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.UpdateWalletBlocklistEntryResponse{UpdatedCount: r.GetUpdatedCount()}, nil
}
func (s *Server) DeleteWalletBlocklistEntry(ctx context.Context, req *tokenapipkg.DeleteWalletBlocklistEntryRequest) (*tokenapipkg.DeleteWalletBlocklistEntryResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.DeleteWalletBlocklistEntry(ctx, &tokenapiapiclient.DeleteWalletBlocklistEntryRequest{Wallet: req.GetWallet()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.DeleteWalletBlocklistEntryResponse{DeletedCount: r.GetDeletedCount()}, nil
}

func (s *Server) GetChainIngestCheckpoint(ctx context.Context, req *tokenapipkg.GetChainIngestCheckpointRequest) (*tokenapipkg.GetChainIngestCheckpointResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.GetChainIngestCheckpoint(ctx, &tokenapiapiclient.GetChainIngestCheckpointRequest{ChainId: req.GetChainId()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetChainIngestCheckpointResponse{Found: r.GetFound(), Checkpoint: r.GetCheckpoint()}, nil
}
func (s *Server) ListChainIngestCheckpoints(ctx context.Context, _ *tokenapipkg.ListChainIngestCheckpointsRequest) (*tokenapipkg.ListChainIngestCheckpointsResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.ListChainIngestCheckpoints(ctx, &tokenapiapiclient.ListChainIngestCheckpointsRequest{})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListChainIngestCheckpointsResponse{Checkpoints: r.GetCheckpoints()}, nil
}
func (s *Server) UpdateChainIngestCheckpoint(ctx context.Context, req *tokenapipkg.UpdateChainIngestCheckpointRequest) (*tokenapipkg.UpdateChainIngestCheckpointResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.UpdateChainIngestCheckpoint(ctx, &tokenapiapiclient.UpdateChainIngestCheckpointRequest{ChainId: req.GetChainId(), Status: req.GetStatus()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.UpdateChainIngestCheckpointResponse{Found: r.GetFound(), Checkpoint: r.GetCheckpoint()}, nil
}
func (s *Server) GetContractCode(ctx context.Context, req *tokenapipkg.GetContractCodeRequest) (*tokenapipkg.GetContractCodeResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.GetContractCode(ctx, &tokenapiapiclient.GetContractCodeRequest{CodeHash: req.GetCodeHash()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetContractCodeResponse{Found: r.GetFound(), ContractCode: r.GetContractCode()}, nil
}
func (s *Server) ListContractCodes(ctx context.Context, req *tokenapipkg.ListContractCodesRequest) (*tokenapipkg.ListContractCodesResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.ListContractCodes(ctx, &tokenapiapiclient.ListContractCodesRequest{CodeHash: req.GetCodeHash(), Page: req.GetPage(), PageSize: req.GetPageSize(), OrderBy: req.GetOrderBy()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListContractCodesResponse{ContractCodes: r.GetContractCodes(), Total: r.GetTotal(), Page: r.GetPage(), PageSize: r.GetPageSize()}, nil
}
func (s *Server) ListProjects(ctx context.Context, req *tokenapipkg.ListProjectsRequest) (*tokenapipkg.ListProjectsResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.ListProjects(ctx, &tokenapiapiclient.ListProjectsRequest{ChainId: req.GetChainId(), CodeHash: req.GetCodeHash(), Contract: req.GetContract(), Page: req.GetPage(), PageSize: req.GetPageSize()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListProjectsResponse{Projects: r.GetProjects(), Total: r.GetTotal(), Page: r.GetPage(), PageSize: r.GetPageSize()}, nil
}
func (s *Server) ListProjectReports(ctx context.Context, req *tokenapipkg.ListProjectReportsRequest) (*tokenapipkg.ListProjectReportsResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.ListProjectReports(ctx, &tokenapiapiclient.ListProjectReportsRequest{ChainId: req.GetChainId(), ProjectId: req.GetProjectId(), Contract: req.GetContract(), EvaluationStatus: req.GetEvaluationStatus(), Page: req.GetPage(), PageSize: req.GetPageSize()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListProjectReportsResponse{ProjectReports: r.GetProjectReports(), Total: r.GetTotal(), Page: r.GetPage(), PageSize: r.GetPageSize()}, nil
}
func (s *Server) GetProjectDataCollectionTask(ctx context.Context, req *tokenapipkg.GetProjectDataCollectionTaskRequest) (*tokenapipkg.GetProjectDataCollectionTaskResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.GetProjectDataCollectionTask(ctx, &tokenapiapiclient.GetProjectDataCollectionTaskRequest{TaskId: req.GetTaskId()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.GetProjectDataCollectionTaskResponse{Found: r.GetFound(), Task: r.GetTask()}, nil
}
func (s *Server) ListProjectDataCollectionTasks(ctx context.Context, req *tokenapipkg.ListProjectDataCollectionTasksRequest) (*tokenapipkg.ListProjectDataCollectionTasksResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.ListProjectDataCollectionTasks(ctx, &tokenapiapiclient.ListProjectDataCollectionTasksRequest{ProjectId: req.GetProjectId(), DataType: req.GetDataType(), Status: req.GetStatus(), Page: req.GetPage(), PageSize: req.GetPageSize()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListProjectDataCollectionTasksResponse{Tasks: r.GetTasks(), Total: r.GetTotal(), Page: r.GetPage(), PageSize: r.GetPageSize()}, nil
}
func (s *Server) ListProjectResearchStates(ctx context.Context, req *tokenapipkg.ListProjectResearchStatesRequest) (*tokenapipkg.ListProjectResearchStatesResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.ListProjectResearchStates(ctx, &tokenapiapiclient.ListProjectResearchStatesRequest{ProjectId: req.GetProjectId(), ChainId: req.GetChainId(), Status: req.GetStatus(), Page: req.GetPage(), PageSize: req.GetPageSize()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListProjectResearchStatesResponse{ResearchStates: r.GetResearchStates(), Total: r.GetTotal(), Page: r.GetPage(), PageSize: r.GetPageSize()}, nil
}
func (s *Server) ListProjectReportRevisions(ctx context.Context, req *tokenapipkg.ListProjectReportRevisionsRequest) (*tokenapipkg.ListProjectReportRevisionsResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.ListProjectReportRevisions(ctx, &tokenapiapiclient.ListProjectReportRevisionsRequest{ProjectId: req.GetProjectId(), ChainId: req.GetChainId(), Page: req.GetPage(), PageSize: req.GetPageSize()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListProjectReportRevisionsResponse{ReportRevisions: r.GetReportRevisions(), Total: r.GetTotal(), Page: r.GetPage(), PageSize: r.GetPageSize()}, nil
}
func (s *Server) ListProjectSelections(ctx context.Context, req *tokenapipkg.ListProjectSelectionsRequest) (*tokenapipkg.ListProjectSelectionsResponse, error) {
	c, client, e := s.tokenAPIClientSet.NewTokenAPIServiceClient()
	if e != nil {
		return nil, e
	}
	defer c.Close()
	r, e := client.ListProjectSelections(ctx, &tokenapiapiclient.ListProjectSelectionsRequest{ProjectId: req.GetProjectId(), ChainId: req.GetChainId(), Outcome: req.GetOutcome(), Page: req.GetPage(), PageSize: req.GetPageSize()})
	if e != nil {
		return nil, e
	}
	return &tokenapipkg.ListProjectSelectionsResponse{Selections: r.GetSelections(), Total: r.GetTotal(), Page: r.GetPage(), PageSize: r.GetPageSize()}, nil
}
