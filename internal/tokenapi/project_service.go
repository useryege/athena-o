package tokenapi

import (
	"context"

	"github.com/useryege/athena/internal/tokenapi/apiclient"
)

func (s *Service) ListProjects(ctx context.Context, req *apiclient.ListProjectsRequest) (*apiclient.ListProjectsResponse, error) {
	store, err := s.catalogApplication()
	if err != nil {
		return nil, err
	}
	if err := validateNonNegativeInt64Field("chain_id", req.GetChainId()); err != nil {
		return nil, err
	}
	codeHash, err := parseOptionalHashField("code_hash", req.GetCodeHash())
	if err != nil {
		return nil, err
	}
	contract, err := parseOptionalAddressField("contract", req.GetContract())
	if err != nil {
		return nil, err
	}
	page, err := store.ListProjectsPage(ctx, req.GetChainId(), codeHash, contract, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, wrapStoreError("list projects", err)
	}
	return &apiclient.ListProjectsResponse{
		Projects: mapProjects(page.Items),
		Total:    page.Total,
		Page:     page.Page,
		PageSize: page.PageSize,
	}, nil
}
