package application

import (
	"context"
	"sort"

	applicationapiclient "github.com/useryege/athena/internal/application/apiclient"
	v1 "github.com/useryege/athena/internal/pkg/proto/v1"
	applicationpkg "github.com/useryege/athena/pkg/apiclient/application"
	"github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	applicationpkg.UnimplementedApplicationServiceServer
	applicationClientSet applicationapiclient.Clientset
}

func NewServer(applicationClientSet applicationapiclient.Clientset) *Server {
	return &Server{
		applicationClientSet: applicationClientSet,
	}
}

func (s *Server) ListProjects(ctx context.Context, req *applicationpkg.ListProjectsRequest) (*applicationpkg.ListProjectsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListProjects(ctx, &applicationapiclient.ListProjectsRequest{
		Scope:    req.GetScope(),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	})
	if err != nil {
		return nil, err
	}

	if req.GetScope() == v1.ProjectScope_PROJECT_SCOPE_UNSPECIFIED || req.GetScope() == v1.ProjectScope_PROJECT_SCOPE_ACTIVE {
		sort.SliceStable(resp.Items, func(i, j int) bool {
			if resp.Items[i].Meta.BlockNumber != resp.Items[j].Meta.BlockNumber {
				return resp.Items[i].Meta.BlockNumber < resp.Items[j].Meta.BlockNumber
			}
			if resp.Items[i].Meta.TxIndex != resp.Items[j].Meta.TxIndex {
				return resp.Items[i].Meta.TxIndex > resp.Items[j].Meta.TxIndex
			}
			return resp.Items[i].Meta.Contract > resp.Items[j].Meta.Contract
		})
	}

	return &applicationpkg.ListProjectsResponse{
		Items:    resp.Items,
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}, nil
}

func (s *Server) GetProject(ctx context.Context, req *applicationpkg.GetProjectRequest) (*applicationpkg.GetProjectResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetProject(ctx, &applicationapiclient.GetProjectRequest{
		Contract: req.GetContract(),
	})
	if err != nil {
		return nil, err
	}

	return &applicationpkg.GetProjectResponse{Item: resp.Item}, nil
}

func (s *Server) ListProjectEventLogs(ctx context.Context, req *applicationpkg.ListProjectEventLogsRequest) (*applicationpkg.ListProjectEventLogsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListProjectEventLogs(ctx, &applicationapiclient.ListProjectEventLogsRequest{
		Contract: req.GetContract(),
	})
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.ProjectEventLog, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, projectEventLogToAPI(item))
	}
	return &applicationpkg.ListProjectEventLogsResponse{Items: items}, nil
}

func (s *Server) AddProjectComment(ctx context.Context, req *applicationpkg.AddProjectCommentRequest) (*applicationpkg.AddProjectCommentResponse, error) {
	username := session.Username(ctx)
	if username == "" {
		return nil, status.Error(codes.Unauthenticated, "login required to add comment")
	}

	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.AddProjectComment(ctx, &applicationapiclient.AddProjectCommentRequest{
		Contract: req.GetContract(),
		Username: username,
		Content:  req.GetContent(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.AddProjectCommentResponse{Item: projectCommentToAPI(resp.Item)}, nil
}

func (s *Server) ListProjectComments(ctx context.Context, req *applicationpkg.ListProjectCommentsRequest) (*applicationpkg.ListProjectCommentsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListProjectComments(ctx, &applicationapiclient.ListProjectCommentsRequest{
		Contract: req.GetContract(),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	})
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.ProjectComment, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, projectCommentToAPI(item))
	}
	return &applicationpkg.ListProjectCommentsResponse{
		Items:    items,
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}, nil
}

func (s *Server) ListSourceCodeBlacklistFields(ctx context.Context, _ *applicationpkg.ListSourceCodeBlacklistFieldsRequest) (*applicationpkg.ListSourceCodeBlacklistFieldsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListSourceCodeBlacklistFields(ctx, &applicationapiclient.ListSourceCodeBlacklistFieldsRequest{})
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.SourceCodeBlacklistField, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, sourceCodeBlacklistFieldToAPI(item))
	}
	return &applicationpkg.ListSourceCodeBlacklistFieldsResponse{Items: items}, nil
}

func (s *Server) AddSourceCodeBlacklistField(ctx context.Context, req *applicationpkg.AddSourceCodeBlacklistFieldRequest) (*applicationpkg.AddSourceCodeBlacklistFieldResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.AddSourceCodeBlacklistField(ctx, &applicationapiclient.AddSourceCodeBlacklistFieldRequest{Field: req.GetField()})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.AddSourceCodeBlacklistFieldResponse{Item: sourceCodeBlacklistFieldToAPI(resp.Item)}, nil
}

func (s *Server) DeleteSourceCodeBlacklistField(ctx context.Context, req *applicationpkg.DeleteSourceCodeBlacklistFieldRequest) (*applicationpkg.DeleteSourceCodeBlacklistFieldResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.DeleteSourceCodeBlacklistField(ctx, &applicationapiclient.DeleteSourceCodeBlacklistFieldRequest{Field: req.GetField()}); err != nil {
		return nil, err
	}
	return &applicationpkg.DeleteSourceCodeBlacklistFieldResponse{}, nil
}

func (s *Server) ListBytecodeBlacklistContracts(ctx context.Context, _ *applicationpkg.ListBytecodeBlacklistContractsRequest) (*applicationpkg.ListBytecodeBlacklistContractsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListBytecodeBlacklistContracts(ctx, &applicationapiclient.ListBytecodeBlacklistContractsRequest{})
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.BytecodeBlacklistContract, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, bytecodeBlacklistContractToAPI(item))
	}
	return &applicationpkg.ListBytecodeBlacklistContractsResponse{Items: items}, nil
}

func (s *Server) AddBytecodeBlacklistContract(ctx context.Context, req *applicationpkg.AddBytecodeBlacklistContractRequest) (*applicationpkg.AddBytecodeBlacklistContractResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.AddBytecodeBlacklistContract(ctx, &applicationapiclient.AddBytecodeBlacklistContractRequest{
		Contract: req.GetContract(),
		Note:     req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.AddBytecodeBlacklistContractResponse{Item: bytecodeBlacklistContractToAPI(resp.Item)}, nil
}

func (s *Server) UpdateBytecodeBlacklistContractNote(ctx context.Context, req *applicationpkg.UpdateBytecodeBlacklistContractNoteRequest) (*applicationpkg.UpdateBytecodeBlacklistContractNoteResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateBytecodeBlacklistContractNote(ctx, &applicationapiclient.UpdateBytecodeBlacklistContractNoteRequest{
		Contract: req.GetContract(),
		Note:     req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.UpdateBytecodeBlacklistContractNoteResponse{Item: bytecodeBlacklistContractToAPI(resp.Item)}, nil
}

func (s *Server) DeleteBytecodeBlacklistContract(ctx context.Context, req *applicationpkg.DeleteBytecodeBlacklistContractRequest) (*applicationpkg.DeleteBytecodeBlacklistContractResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.DeleteBytecodeBlacklistContract(ctx, &applicationapiclient.DeleteBytecodeBlacklistContractRequest{
		Contract: req.GetContract(),
	}); err != nil {
		return nil, err
	}
	return &applicationpkg.DeleteBytecodeBlacklistContractResponse{}, nil
}

func (s *Server) ListSourcecodeBlacklistContracts(ctx context.Context, _ *applicationpkg.ListSourcecodeBlacklistContractsRequest) (*applicationpkg.ListSourcecodeBlacklistContractsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListSourcecodeBlacklistContracts(ctx, &applicationapiclient.ListSourcecodeBlacklistContractsRequest{})
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.SourcecodeBlacklistContract, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, sourcecodeBlacklistContractToAPI(item))
	}
	return &applicationpkg.ListSourcecodeBlacklistContractsResponse{Items: items}, nil
}

func (s *Server) AddSourcecodeBlacklistContract(ctx context.Context, req *applicationpkg.AddSourcecodeBlacklistContractRequest) (*applicationpkg.AddSourcecodeBlacklistContractResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.AddSourcecodeBlacklistContract(ctx, &applicationapiclient.AddSourcecodeBlacklistContractRequest{
		Contract: req.GetContract(),
		Note:     req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.AddSourcecodeBlacklistContractResponse{Item: sourcecodeBlacklistContractToAPI(resp.Item)}, nil
}

func (s *Server) UpdateSourcecodeBlacklistContractNote(ctx context.Context, req *applicationpkg.UpdateSourcecodeBlacklistContractNoteRequest) (*applicationpkg.UpdateSourcecodeBlacklistContractNoteResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateSourcecodeBlacklistContractNote(ctx, &applicationapiclient.UpdateSourcecodeBlacklistContractNoteRequest{
		Contract: req.GetContract(),
		Note:     req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.UpdateSourcecodeBlacklistContractNoteResponse{Item: sourcecodeBlacklistContractToAPI(resp.Item)}, nil
}

func (s *Server) DeleteSourcecodeBlacklistContract(ctx context.Context, req *applicationpkg.DeleteSourcecodeBlacklistContractRequest) (*applicationpkg.DeleteSourcecodeBlacklistContractResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.DeleteSourcecodeBlacklistContract(ctx, &applicationapiclient.DeleteSourcecodeBlacklistContractRequest{
		Contract: req.GetContract(),
	}); err != nil {
		return nil, err
	}
	return &applicationpkg.DeleteSourcecodeBlacklistContractResponse{}, nil
}

func (s *Server) ListWalletBlacklistEntries(ctx context.Context, _ *applicationpkg.ListWalletBlacklistEntriesRequest) (*applicationpkg.ListWalletBlacklistEntriesResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListWalletBlacklistEntries(ctx, &applicationapiclient.ListWalletBlacklistEntriesRequest{})
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.WalletBlacklistEntry, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, walletBlacklistEntryToAPI(item))
	}
	return &applicationpkg.ListWalletBlacklistEntriesResponse{Items: items}, nil
}

func (s *Server) AddWalletBlacklistEntry(ctx context.Context, req *applicationpkg.AddWalletBlacklistEntryRequest) (*applicationpkg.AddWalletBlacklistEntryResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.AddWalletBlacklistEntry(ctx, &applicationapiclient.AddWalletBlacklistEntryRequest{
		Wallet: req.GetWallet(),
		Note:   req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.AddWalletBlacklistEntryResponse{Item: walletBlacklistEntryToAPI(resp.Item)}, nil
}

func (s *Server) UpdateWalletBlacklistEntryNote(ctx context.Context, req *applicationpkg.UpdateWalletBlacklistEntryNoteRequest) (*applicationpkg.UpdateWalletBlacklistEntryNoteResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateWalletBlacklistEntryNote(ctx, &applicationapiclient.UpdateWalletBlacklistEntryNoteRequest{
		Wallet: req.GetWallet(),
		Note:   req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.UpdateWalletBlacklistEntryNoteResponse{Item: walletBlacklistEntryToAPI(resp.Item)}, nil
}

func (s *Server) DeleteWalletBlacklistEntry(ctx context.Context, req *applicationpkg.DeleteWalletBlacklistEntryRequest) (*applicationpkg.DeleteWalletBlacklistEntryResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.DeleteWalletBlacklistEntry(ctx, &applicationapiclient.DeleteWalletBlacklistEntryRequest{
		Wallet: req.GetWallet(),
	}); err != nil {
		return nil, err
	}
	return &applicationpkg.DeleteWalletBlacklistEntryResponse{}, nil
}

func (s *Server) GetProjectOptions(ctx context.Context, _ *applicationpkg.GetProjectOptionsRequest) (*applicationpkg.GetProjectOptionsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetProjectOptions(ctx, &applicationapiclient.GetProjectOptionsRequest{})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.GetProjectOptionsResponse{Options: resp.Options}, nil
}

func (s *Server) ArchiveProject(ctx context.Context, req *applicationpkg.ArchiveProjectRequest) (*applicationpkg.ArchiveProjectResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.ArchiveProject(ctx, &applicationapiclient.ArchiveProjectRequest{Contract: req.GetContract()}); err != nil {
		return nil, err
	}
	return &applicationpkg.ArchiveProjectResponse{}, nil
}

func (s *Server) UnarchiveProject(ctx context.Context, req *applicationpkg.UnarchiveProjectRequest) (*applicationpkg.UnarchiveProjectResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.UnarchiveProject(ctx, &applicationapiclient.UnarchiveProjectRequest{Contract: req.GetContract()}); err != nil {
		return nil, err
	}
	return &applicationpkg.UnarchiveProjectResponse{}, nil
}

func sourceCodeBlacklistFieldToAPI(item *applicationapiclient.SourceCodeBlacklistField) *applicationpkg.SourceCodeBlacklistField {
	if item == nil {
		return nil
	}
	return &applicationpkg.SourceCodeBlacklistField{
		Id:    item.Id,
		Field: item.Field,
	}
}

func bytecodeBlacklistContractToAPI(item *applicationapiclient.BytecodeBlacklistContract) *applicationpkg.BytecodeBlacklistContract {
	if item == nil {
		return nil
	}
	return &applicationpkg.BytecodeBlacklistContract{
		Contract:  item.Contract,
		CodeHash:  item.CodeHash,
		Note:      item.Note,
		CreatedAt: item.CreatedAt,
	}
}

func sourcecodeBlacklistContractToAPI(item *applicationapiclient.SourcecodeBlacklistContract) *applicationpkg.SourcecodeBlacklistContract {
	if item == nil {
		return nil
	}
	return &applicationpkg.SourcecodeBlacklistContract{
		Contract:   item.GetContract(),
		SourceHash: item.GetSourceHash(),
		Note:       item.GetNote(),
		CreatedAt:  item.GetCreatedAt(),
	}
}

func walletBlacklistEntryToAPI(item *applicationapiclient.WalletBlacklistEntry) *applicationpkg.WalletBlacklistEntry {
	if item == nil {
		return nil
	}
	return &applicationpkg.WalletBlacklistEntry{
		Wallet:    item.Wallet,
		Note:      item.Note,
		CreatedAt: item.CreatedAt,
	}
}

func projectEventLogToAPI(item *applicationapiclient.ProjectEventLog) *applicationpkg.ProjectEventLog {
	if item == nil {
		return nil
	}
	return &applicationpkg.ProjectEventLog{
		Id:         item.Id,
		Contract:   item.Contract,
		EventType:  item.EventType,
		OccurredAt: item.OccurredAt,
		Message:    item.Message,
		Payload:    item.Payload,
		CreatedAt:  item.CreatedAt,
	}
}

func projectCommentToAPI(item *applicationapiclient.ProjectComment) *applicationpkg.ProjectComment {
	if item == nil {
		return nil
	}
	return &applicationpkg.ProjectComment{
		Id:        item.Id,
		Contract:  item.Contract,
		Username:  item.Username,
		Content:   item.Content,
		CreatedAt: item.CreatedAt,
	}
}
