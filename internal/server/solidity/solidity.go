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
