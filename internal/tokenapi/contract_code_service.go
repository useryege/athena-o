package tokenapi

import (
	"context"
	"strings"

	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/internal/tokenapi/apiclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const contractCodeOrderByDeploymentCount = "deployment_count"

func (s *Service) GetContractCode(ctx context.Context, req *apiclient.GetContractCodeRequest) (*apiclient.GetContractCodeResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	codeHash, err := parseHashField("code_hash", req.GetCodeHash())
	if err != nil {
		return nil, err
	}
	item, err := store.GetContractCode(ctx, codeHash)
	if err != nil {
		return nil, wrapStoreError("get contract code", err)
	}
	if item == nil {
		return &apiclient.GetContractCodeResponse{}, nil
	}
	return &apiclient.GetContractCodeResponse{
		Found:        true,
		ContractCode: mapContractCode(*item),
	}, nil
}

func (s *Service) ListContractCodes(ctx context.Context, req *apiclient.ListContractCodesRequest) (*apiclient.ListContractCodesResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	codeHash, err := parseOptionalHashField("code_hash", req.GetCodeHash())
	if err != nil {
		return nil, err
	}
	orderBy := strings.TrimSpace(req.GetOrderBy())
	var page *tokenstore.ContractCodePage
	switch orderBy {
	case "":
		defaultPage, err := store.ListContractCodes(ctx, codeHash, req.GetPage(), req.GetPageSize())
		if err != nil {
			return nil, wrapStoreError("list contract codes", err)
		}
		page = defaultPage
	case contractCodeOrderByDeploymentCount:
		if req.GetCodeHash() != "" {
			return nil, status.Error(codes.InvalidArgument, "code_hash cannot be used with order_by deployment_count")
		}
		deploymentCountPage, err := store.ListContractCodesByDeploymentCount(ctx, req.GetPage(), req.GetPageSize())
		if err != nil {
			return nil, wrapStoreError("list contract codes by deployment count", err)
		}
		page = deploymentCountPage
	default:
		return nil, status.Errorf(codes.InvalidArgument, "order_by must be empty or %q", contractCodeOrderByDeploymentCount)
	}
	return &apiclient.ListContractCodesResponse{
		ContractCodes: mapContractCodes(page.Items),
		Total:         page.Total,
		Page:          page.Page,
		PageSize:      page.PageSize,
	}, nil
}
