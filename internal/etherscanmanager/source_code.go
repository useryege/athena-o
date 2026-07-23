package etherscanmanager

import (
	"context"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/etherscanmanager/apiclient"
	"github.com/useryege/athena/util/etherscanapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) GetSourceCode(
	ctx context.Context,
	req *apiclient.GetSourceCodeRequest,
) (*apiclient.GetSourceCodeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.GetChainId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "chain_id must be positive")
	}
	contractAddress := strings.TrimSpace(req.GetContractAddress())
	if !ethcommon.IsHexAddress(contractAddress) {
		return nil, status.Error(codes.InvalidArgument, "contract_address must be a valid EVM address")
	}

	response, err := s.manager.GetSourceCode(ctx, req.GetChainId(), ethcommon.HexToAddress(contractAddress).Hex())
	if err != nil {
		return nil, grpcErrorFromEtherscanAPI(err)
	}
	if response == nil {
		return nil, status.Error(codes.DataLoss, "source code response is empty")
	}

	items := make([]*apiclient.SourceCode, 0, len(response.Result))
	for _, item := range response.Result {
		items = append(items, sourceCodeToProto(item))
	}
	return &apiclient.GetSourceCodeResponse{Items: items}, nil
}

func sourceCodeToProto(item etherscanapi.SourceCodeResult) *apiclient.SourceCode {
	return &apiclient.SourceCode{
		SourceCode:           item.SourceCode,
		Abi:                  item.ABI,
		ContractName:         item.ContractName,
		CompilerVersion:      item.CompilerVersion,
		OptimizationUsed:     item.OptimizationUsed,
		Runs:                 item.Runs,
		ConstructorArguments: item.ConstructorArguments,
		EvmVersion:           item.EVMVersion,
		Library:              item.Library,
		LicenseType:          item.LicenseType,
		Proxy:                item.Proxy,
		Implementation:       item.Implementation,
		SwarmSource:          item.SwarmSource,
	}
}
