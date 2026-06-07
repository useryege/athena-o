package tokenapi

import (
	"context"

	"github.com/useryege/athena/internal/tokenapi/apiclient"
)

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
	page, err := store.ListContractCodes(ctx, codeHash, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, wrapStoreError("list contract codes", err)
	}
	return &apiclient.ListContractCodesResponse{
		ContractCodes: mapContractCodes(page.Items),
		Total:         page.Total,
		Page:          page.Page,
		PageSize:      page.PageSize,
	}, nil
}

func (s *Service) DeleteContractCode(ctx context.Context, req *apiclient.DeleteContractCodeRequest) (*apiclient.DeleteContractCodeResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	codeHash, err := parseHashField("code_hash", req.GetCodeHash())
	if err != nil {
		return nil, err
	}
	deleted, err := store.DeleteContractCode(ctx, codeHash)
	if err != nil {
		return nil, wrapStoreError("delete contract code", err)
	}
	return &apiclient.DeleteContractCodeResponse{DeletedCount: deleted}, nil
}
