package tokenapi

import (
	"context"
	"strings"

	"github.com/useryege/athena/internal/tokenapi/apiclient"
)

func (s *Service) ListProjectReports(ctx context.Context, req *apiclient.ListProjectReportsRequest) (*apiclient.ListProjectReportsResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	if err := validateNonNegativeInt64Field("chain_id", req.GetChainId()); err != nil {
		return nil, err
	}
	if err := validateNonNegativeInt64Field("project_id", req.GetProjectId()); err != nil {
		return nil, err
	}
	contract, err := parseOptionalAddressField("contract", req.GetContract())
	if err != nil {
		return nil, err
	}
	evaluationStatus := strings.TrimSpace(req.GetEvaluationStatus())
	if err := validateProjectReportEvaluationStatus(evaluationStatus); err != nil {
		return nil, err
	}
	page, err := store.ListProjectReportsPage(
		ctx,
		req.GetChainId(),
		req.GetProjectId(),
		contract,
		evaluationStatus,
		req.GetPage(),
		req.GetPageSize(),
	)
	if err != nil {
		return nil, wrapStoreError("list project reports", err)
	}
	return &apiclient.ListProjectReportsResponse{
		ProjectReports: mapProjectReports(page.Items),
		Total:          page.Total,
		Page:           page.Page,
		PageSize:       page.PageSize,
	}, nil
}
