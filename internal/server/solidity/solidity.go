package solidity

import (
	"context"

	solidityapiclient "github.com/useryege/athena/internal/solidity/apiclient"
	soliditypkg "github.com/useryege/athena/pkg/apiclient/solidity"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type Server struct {
	soliditypkg.UnimplementedSolidityServiceServer
	solidityClientSet solidityapiclient.Clientset
}

func NewServer(solidityClientSet solidityapiclient.Clientset) *Server {
	return &Server{solidityClientSet: solidityClientSet}
}

func (s *Server) GetSolidityStatus(ctx context.Context, _ *soliditypkg.GetSolidityStatusRequest) (*v1alpha1.SolidityStatus, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetSolidityStatus(ctx, &solidityapiclient.GetSolidityStatusRequest{})
}

func (s *Server) GetContractSourceInfo(ctx context.Context, req *soliditypkg.GetContractSourceInfoRequest) (*v1alpha1.ContractSourceInfo, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetContractSourceInfo(ctx, &solidityapiclient.GetContractSourceInfoRequest{
		ChainId:  req.GetChainId(),
		Contract: req.GetContract(),
	})
}

func (s *Server) ListBytecodes(ctx context.Context, req *soliditypkg.ListBytecodesRequest) (*soliditypkg.ListBytecodesResponse, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListBytecodes(ctx, &solidityapiclient.ListBytecodesRequest{
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
		CodeHash: req.GetCodeHash(),
	})
	if err != nil {
		return nil, err
	}
	return &soliditypkg.ListBytecodesResponse{
		Items:    resp.GetItems(),
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
	}, nil
}

func (s *Server) GetBytecode(ctx context.Context, req *soliditypkg.GetBytecodeRequest) (*v1alpha1.BytecodeDetail, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetBytecode(ctx, &solidityapiclient.GetBytecodeRequest{CodeHash: req.GetCodeHash()})
}

func (s *Server) ListBytecodeDeployments(ctx context.Context, req *soliditypkg.ListBytecodeDeploymentsRequest) (*soliditypkg.ListBytecodeDeploymentsResponse, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListBytecodeDeployments(ctx, &solidityapiclient.ListBytecodeDeploymentsRequest{
		CodeHash: req.GetCodeHash(),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
		ChainId:  req.GetChainId(),
		Contract: req.GetContract(),
	})
	if err != nil {
		return nil, err
	}
	return &soliditypkg.ListBytecodeDeploymentsResponse{
		Items:    resp.GetItems(),
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
	}, nil
}

func (s *Server) ListBytecodeBlacklistEntries(ctx context.Context, _ *soliditypkg.ListBytecodeBlacklistEntriesRequest) (*soliditypkg.ListBytecodeBlacklistEntriesResponse, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListBytecodeBlacklistEntries(ctx, &solidityapiclient.ListBytecodeBlacklistEntriesRequest{})
	if err != nil {
		return nil, err
	}
	return &soliditypkg.ListBytecodeBlacklistEntriesResponse{Items: resp.GetItems()}, nil
}

func (s *Server) AddBytecodeBlacklistEntry(ctx context.Context, req *soliditypkg.AddBytecodeBlacklistEntryRequest) (*soliditypkg.AddBytecodeBlacklistEntryResponse, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.AddBytecodeBlacklistEntry(ctx, &solidityapiclient.AddBytecodeBlacklistEntryRequest{
		CodeHash:       req.GetCodeHash(),
		SourceChainId:  req.GetSourceChainId(),
		SourceContract: req.GetSourceContract(),
		Note:           req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &soliditypkg.AddBytecodeBlacklistEntryResponse{Item: resp.GetItem()}, nil
}

func (s *Server) UpdateBytecodeBlacklistNote(ctx context.Context, req *soliditypkg.UpdateBytecodeBlacklistNoteRequest) (*soliditypkg.UpdateBytecodeBlacklistNoteResponse, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateBytecodeBlacklistNote(ctx, &solidityapiclient.UpdateBytecodeBlacklistNoteRequest{
		CodeHash: req.GetCodeHash(),
		Note:     req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &soliditypkg.UpdateBytecodeBlacklistNoteResponse{Item: resp.GetItem()}, nil
}

func (s *Server) DeleteBytecodeBlacklist(ctx context.Context, req *soliditypkg.DeleteBytecodeBlacklistRequest) (*soliditypkg.DeleteBytecodeBlacklistResponse, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.DeleteBytecodeBlacklist(ctx, &solidityapiclient.DeleteBytecodeBlacklistRequest{CodeHash: req.GetCodeHash()}); err != nil {
		return nil, err
	}
	return &soliditypkg.DeleteBytecodeBlacklistResponse{}, nil
}

func (s *Server) ListSourceQualityPrompts(ctx context.Context, _ *soliditypkg.ListSourceQualityPromptsRequest) (*soliditypkg.ListSourceQualityPromptsResponse, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListSourceQualityPrompts(ctx, &solidityapiclient.ListSourceQualityPromptsRequest{})
	if err != nil {
		return nil, err
	}
	return &soliditypkg.ListSourceQualityPromptsResponse{Items: resp.GetItems()}, nil
}

func (s *Server) GetSourceQualityPrompt(ctx context.Context, req *soliditypkg.GetSourceQualityPromptRequest) (*v1alpha1.SourceQualityPrompt, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetSourceQualityPrompt(ctx, &solidityapiclient.GetSourceQualityPromptRequest{Id: req.GetId()})
}

func (s *Server) CreateSourceQualityPrompt(ctx context.Context, req *soliditypkg.CreateSourceQualityPromptRequest) (*soliditypkg.CreateSourceQualityPromptResponse, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.CreateSourceQualityPrompt(ctx, &solidityapiclient.CreateSourceQualityPromptRequest{
		Name:         req.GetName(),
		SystemPrompt: req.GetSystemPrompt(),
	})
	if err != nil {
		return nil, err
	}
	return &soliditypkg.CreateSourceQualityPromptResponse{Item: resp.GetItem()}, nil
}

func (s *Server) UpdateSourceQualityPrompt(ctx context.Context, req *soliditypkg.UpdateSourceQualityPromptRequest) (*soliditypkg.UpdateSourceQualityPromptResponse, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateSourceQualityPrompt(ctx, &solidityapiclient.UpdateSourceQualityPromptRequest{
		Id:           req.GetId(),
		Name:         req.GetName(),
		SystemPrompt: req.GetSystemPrompt(),
	})
	if err != nil {
		return nil, err
	}
	return &soliditypkg.UpdateSourceQualityPromptResponse{Item: resp.GetItem()}, nil
}

func (s *Server) ActivateSourceQualityPrompt(ctx context.Context, req *soliditypkg.ActivateSourceQualityPromptRequest) (*soliditypkg.ActivateSourceQualityPromptResponse, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ActivateSourceQualityPrompt(ctx, &solidityapiclient.ActivateSourceQualityPromptRequest{Id: req.GetId()})
	if err != nil {
		return nil, err
	}
	return &soliditypkg.ActivateSourceQualityPromptResponse{Item: resp.GetItem()}, nil
}

func (s *Server) DeleteSourceQualityPrompt(ctx context.Context, req *soliditypkg.DeleteSourceQualityPromptRequest) (*soliditypkg.DeleteSourceQualityPromptResponse, error) {
	closer, client, err := s.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.DeleteSourceQualityPrompt(ctx, &solidityapiclient.DeleteSourceQualityPromptRequest{Id: req.GetId()}); err != nil {
		return nil, err
	}
	return &soliditypkg.DeleteSourceQualityPromptResponse{}, nil
}
