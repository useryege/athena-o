package application

import (
	"context"
	"fmt"

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

func (s *Server) ListChains(ctx context.Context, _ *applicationpkg.ListChainsRequest) (*applicationpkg.ListChainsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListChains(ctx, &applicationapiclient.ListChainsRequest{})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.ListChainsResponse{Items: resp.GetItems()}, nil
}

func (s *Server) GetChainIngestStatus(ctx context.Context, req *applicationpkg.GetChainIngestStatusRequest) (*v1alpha1.ChainIngestStatus, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetChainIngestStatus(ctx, &applicationapiclient.GetChainIngestStatusRequest{ChainId: req.GetChainId()})
}

func (s *Server) StartChainIngest(ctx context.Context, req *applicationpkg.StartChainIngestRequest) (*v1alpha1.ChainIngestStatus, error) {
	if err := s.ensureHasProjectDiscoveryPermission(ctx); err != nil {
		return nil, err
	}
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.StartChainIngest(ctx, &applicationapiclient.StartChainIngestRequest{ChainId: req.GetChainId()})
}

func (s *Server) StopChainIngest(ctx context.Context, req *applicationpkg.StopChainIngestRequest) (*v1alpha1.ChainIngestStatus, error) {
	if err := s.ensureHasProjectDiscoveryPermission(ctx); err != nil {
		return nil, err
	}
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.StopChainIngest(ctx, &applicationapiclient.StopChainIngestRequest{ChainId: req.GetChainId()})
}

func (s *Server) RequestProjectCollection(ctx context.Context, req *applicationpkg.RequestProjectCollectionRequest) (*applicationpkg.RequestProjectCollectionResponse, error) {
	if err := s.ensureHasProjectDiscoveryPermission(ctx); err != nil {
		return nil, err
	}
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.RequestProjectCollection(ctx, &applicationapiclient.RequestProjectCollectionRequest{
		ChainId:  req.GetChainId(),
		Contract: req.GetContract(),
		Reason:   req.GetReason(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.RequestProjectCollectionResponse{Status: resp.GetStatus()}, nil
}

func (s *Server) GetProjectCollectionStatus(ctx context.Context, req *applicationpkg.GetProjectCollectionStatusRequest) (*v1alpha1.ProjectCollectionStatus, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetProjectCollectionStatus(ctx, &applicationapiclient.GetProjectCollectionStatusRequest{
		ChainId:  req.GetChainId(),
		Contract: req.GetContract(),
	})
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
	return &applicationpkg.ListWalletBlacklistEntriesResponse{Items: resp.GetItems()}, nil
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
	return &applicationpkg.AddWalletBlacklistEntryResponse{Item: resp.GetItem()}, nil
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
	return &applicationpkg.UpdateWalletBlacklistEntryNoteResponse{Item: resp.GetItem()}, nil
}

func (s *Server) DeleteWalletBlacklistEntry(ctx context.Context, req *applicationpkg.DeleteWalletBlacklistEntryRequest) (*applicationpkg.DeleteWalletBlacklistEntryResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.DeleteWalletBlacklistEntry(ctx, &applicationapiclient.DeleteWalletBlacklistEntryRequest{Wallet: req.GetWallet()}); err != nil {
		return nil, err
	}
	return &applicationpkg.DeleteWalletBlacklistEntryResponse{}, nil
}
