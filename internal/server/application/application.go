package application

import (
	"context"
	"fmt"
	"sort"

	applicationapiclient "github.com/useryege/athena/internal/application/apiclient"
	applicationpkg "github.com/useryege/athena/pkg/apiclient/application"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/rbac"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	applicationpkg.UnimplementedApplicationServiceServer
	applicationClientSet applicationapiclient.Clientset
	enf                  *rbac.Enforcer
}

func NewServer(applicationClientSet applicationapiclient.Clientset, enf *rbac.Enforcer) *Server {
	return &Server{
		applicationClientSet: applicationClientSet,
		enf:                  enf,
	}
}

func (s *Server) ListProjects(ctx context.Context, req *applicationpkg.ListProjectsRequest) (*applicationpkg.ListProjectsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListProjects(ctx, &applicationapiclient.ListProjectsRequest{
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(resp.Items, func(i, j int) bool {
		if resp.Items[i].BlockNumber != resp.Items[j].BlockNumber {
			return resp.Items[i].BlockNumber < resp.Items[j].BlockNumber
		}
		if resp.Items[i].TxIndex != resp.Items[j].TxIndex {
			return resp.Items[i].TxIndex > resp.Items[j].TxIndex
		}
		return resp.Items[i].Contract > resp.Items[j].Contract
	})

	return &applicationpkg.ListProjectsResponse{
		Items:    resp.Items,
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}, nil
}

func (s *Server) GetProjectDiscoveryStatus(ctx context.Context, _ *applicationpkg.GetProjectDiscoveryStatusRequest) (*v1alpha1.ProjectDiscoveryStatus, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	return client.GetProjectDiscoveryStatus(ctx, &applicationapiclient.GetProjectDiscoveryStatusRequest{})
}

func (s *Server) StartProjectDiscovery(ctx context.Context, _ *applicationpkg.StartProjectDiscoveryRequest) (*v1alpha1.ProjectDiscoveryStatus, error) {
	if err := s.ensureHasProjectDiscoveryPermission(ctx); err != nil {
		return nil, err
	}
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	return client.StartProjectDiscovery(ctx, &applicationapiclient.StartProjectDiscoveryRequest{})
}

func (s *Server) StopProjectDiscovery(ctx context.Context, _ *applicationpkg.StopProjectDiscoveryRequest) (*v1alpha1.ProjectDiscoveryStatus, error) {
	if err := s.ensureHasProjectDiscoveryPermission(ctx); err != nil {
		return nil, err
	}
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	return client.StopProjectDiscovery(ctx, &applicationapiclient.StopProjectDiscoveryRequest{})
}

func (s *Server) ensureHasProjectDiscoveryPermission(ctx context.Context) error {
	if s.enf == nil {
		return status.Error(codes.PermissionDenied, "permission denied to control application discovery")
	}
	if err := s.enf.EnforceErr(ctx.Value("claims"), rbac.ResourceApplicationDiscovery, rbac.ActionUpdate, "*"); err != nil {
		return fmt.Errorf("permission denied to control application discovery: %w", err)
	}
	return nil
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

func (s *Server) GetProjectBase(ctx context.Context, req *applicationpkg.GetProjectBaseRequest) (*applicationpkg.GetProjectBaseResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	resp, err := client.GetProjectBase(ctx, &applicationapiclient.GetProjectBaseRequest{Contract: req.GetContract()})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.GetProjectBaseResponse{Item: resp.Item}, nil
}

func (s *Server) GetProjectReport(ctx context.Context, req *applicationpkg.GetProjectReportRequest) (*applicationpkg.GetProjectReportResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	resp, err := client.GetProjectReport(ctx, &applicationapiclient.GetProjectReportRequest{Contract: req.GetContract()})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.GetProjectReportResponse{Item: resp.Item}, nil
}

func (s *Server) GetProjectChainState(ctx context.Context, req *applicationpkg.GetProjectChainStateRequest) (*applicationpkg.GetProjectChainStateResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	resp, err := client.GetProjectChainState(ctx, &applicationapiclient.GetProjectChainStateRequest{Contract: req.GetContract()})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.GetProjectChainStateResponse{Item: resp.Item}, nil
}

func (s *Server) GetProjectSimulation(ctx context.Context, req *applicationpkg.GetProjectSimulationRequest) (*applicationpkg.GetProjectSimulationResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	resp, err := client.GetProjectSimulation(ctx, &applicationapiclient.GetProjectSimulationRequest{Contract: req.GetContract()})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.GetProjectSimulationResponse{Item: resp.Item}, nil
}

func (s *Server) GetProjectAveState(ctx context.Context, req *applicationpkg.GetProjectAveStateRequest) (*applicationpkg.GetProjectAveStateResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	resp, err := client.GetProjectAveState(ctx, &applicationapiclient.GetProjectAveStateRequest{Contract: req.GetContract()})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.GetProjectAveStateResponse{Item: resp.Item}, nil
}

func (s *Server) RefreshProjectAveDetail(ctx context.Context, req *applicationpkg.RefreshProjectAveDetailRequest) (*applicationpkg.RefreshProjectAveDetailResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	resp, err := client.RefreshProjectAveDetail(ctx, &applicationapiclient.RefreshProjectAveDetailRequest{Contract: req.GetContract()})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.RefreshProjectAveDetailResponse{Item: resp.Item}, nil
}

func (s *Server) ListProjectGenesisWallets(ctx context.Context, req *applicationpkg.ListProjectGenesisWalletsRequest) (*applicationpkg.ListProjectGenesisWalletsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	resp, err := client.ListProjectGenesisWallets(ctx, &applicationapiclient.ListProjectGenesisWalletsRequest{Contract: req.GetContract()})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.ListProjectGenesisWalletsResponse{Items: resp.Items}, nil
}

func (s *Server) ListProjectCreatorHistoricalProjects(ctx context.Context, req *applicationpkg.ListProjectCreatorHistoricalProjectsRequest) (*applicationpkg.ListProjectCreatorHistoricalProjectsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	resp, err := client.ListProjectCreatorHistoricalProjects(ctx, &applicationapiclient.ListProjectCreatorHistoricalProjectsRequest{Contract: req.GetContract()})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.ListProjectCreatorHistoricalProjectsResponse{Items: resp.Items}, nil
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

func (s *Server) GetContractSourceInfo(ctx context.Context, req *applicationpkg.GetContractSourceInfoRequest) (*v1alpha1.ContractSourceInfo, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetContractSourceInfo(ctx, &applicationapiclient.GetContractSourceInfoRequest{
		ChainId:  req.GetChainId(),
		Contract: req.GetContract(),
	})
}

func (s *Server) ListBytecodes(ctx context.Context, req *applicationpkg.ListBytecodesRequest) (*applicationpkg.ListBytecodesResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListBytecodes(ctx, &applicationapiclient.ListBytecodesRequest{
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
		CodeHash: req.GetCodeHash(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.ListBytecodesResponse{
		Items:    resp.GetItems(),
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
	}, nil
}

func (s *Server) GetBytecode(ctx context.Context, req *applicationpkg.GetBytecodeRequest) (*v1alpha1.BytecodeDetail, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetBytecode(ctx, &applicationapiclient.GetBytecodeRequest{CodeHash: req.GetCodeHash()})
}

func (s *Server) ListBytecodeDeployments(ctx context.Context, req *applicationpkg.ListBytecodeDeploymentsRequest) (*applicationpkg.ListBytecodeDeploymentsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListBytecodeDeployments(ctx, &applicationapiclient.ListBytecodeDeploymentsRequest{
		CodeHash: req.GetCodeHash(),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
		ChainId:  req.GetChainId(),
		Contract: req.GetContract(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.ListBytecodeDeploymentsResponse{
		Items:    resp.GetItems(),
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
	}, nil
}

func (s *Server) ListBytecodeBlacklistEntries(ctx context.Context, _ *applicationpkg.ListBytecodeBlacklistEntriesRequest) (*applicationpkg.ListBytecodeBlacklistEntriesResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListBytecodeBlacklistEntries(ctx, &applicationapiclient.ListBytecodeBlacklistEntriesRequest{})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.ListBytecodeBlacklistEntriesResponse{Items: resp.GetItems()}, nil
}

func (s *Server) AddBytecodeBlacklistEntry(ctx context.Context, req *applicationpkg.AddBytecodeBlacklistEntryRequest) (*applicationpkg.AddBytecodeBlacklistEntryResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.AddBytecodeBlacklistEntry(ctx, &applicationapiclient.AddBytecodeBlacklistEntryRequest{
		CodeHash:       req.GetCodeHash(),
		SourceChainId:  req.GetSourceChainId(),
		SourceContract: req.GetSourceContract(),
		Note:           req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.AddBytecodeBlacklistEntryResponse{Item: resp.GetItem()}, nil
}

func (s *Server) UpdateBytecodeBlacklistNote(ctx context.Context, req *applicationpkg.UpdateBytecodeBlacklistNoteRequest) (*applicationpkg.UpdateBytecodeBlacklistNoteResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateBytecodeBlacklistNote(ctx, &applicationapiclient.UpdateBytecodeBlacklistNoteRequest{
		CodeHash: req.GetCodeHash(),
		Note:     req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.UpdateBytecodeBlacklistNoteResponse{Item: resp.GetItem()}, nil
}

func (s *Server) DeleteBytecodeBlacklist(ctx context.Context, req *applicationpkg.DeleteBytecodeBlacklistRequest) (*applicationpkg.DeleteBytecodeBlacklistResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.DeleteBytecodeBlacklist(ctx, &applicationapiclient.DeleteBytecodeBlacklistRequest{CodeHash: req.GetCodeHash()}); err != nil {
		return nil, err
	}
	return &applicationpkg.DeleteBytecodeBlacklistResponse{}, nil
}

func (s *Server) ListSourceQualityPrompts(ctx context.Context, _ *applicationpkg.ListSourceQualityPromptsRequest) (*applicationpkg.ListSourceQualityPromptsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListSourceQualityPrompts(ctx, &applicationapiclient.ListSourceQualityPromptsRequest{})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.ListSourceQualityPromptsResponse{Items: resp.GetItems()}, nil
}

func (s *Server) GetSourceQualityPrompt(ctx context.Context, req *applicationpkg.GetSourceQualityPromptRequest) (*v1alpha1.SourceQualityPrompt, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetSourceQualityPrompt(ctx, &applicationapiclient.GetSourceQualityPromptRequest{Id: req.GetId()})
}

func (s *Server) CreateSourceQualityPrompt(ctx context.Context, req *applicationpkg.CreateSourceQualityPromptRequest) (*applicationpkg.CreateSourceQualityPromptResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.CreateSourceQualityPrompt(ctx, &applicationapiclient.CreateSourceQualityPromptRequest{
		Name:         req.GetName(),
		SystemPrompt: req.GetSystemPrompt(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.CreateSourceQualityPromptResponse{Item: resp.GetItem()}, nil
}

func (s *Server) UpdateSourceQualityPrompt(ctx context.Context, req *applicationpkg.UpdateSourceQualityPromptRequest) (*applicationpkg.UpdateSourceQualityPromptResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateSourceQualityPrompt(ctx, &applicationapiclient.UpdateSourceQualityPromptRequest{
		Id:           req.GetId(),
		Name:         req.GetName(),
		SystemPrompt: req.GetSystemPrompt(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.UpdateSourceQualityPromptResponse{Item: resp.GetItem()}, nil
}

func (s *Server) ActivateSourceQualityPrompt(ctx context.Context, req *applicationpkg.ActivateSourceQualityPromptRequest) (*applicationpkg.ActivateSourceQualityPromptResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ActivateSourceQualityPrompt(ctx, &applicationapiclient.ActivateSourceQualityPromptRequest{Id: req.GetId()})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.ActivateSourceQualityPromptResponse{Item: resp.GetItem()}, nil
}

func (s *Server) DeleteSourceQualityPrompt(ctx context.Context, req *applicationpkg.DeleteSourceQualityPromptRequest) (*applicationpkg.DeleteSourceQualityPromptResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.DeleteSourceQualityPrompt(ctx, &applicationapiclient.DeleteSourceQualityPromptRequest{Id: req.GetId()}); err != nil {
		return nil, err
	}
	return &applicationpkg.DeleteSourceQualityPromptResponse{}, nil
}
