package tokenapi

import (
	"context"
	"strings"

	"github.com/useryege/athena/internal/tokenapi/apiclient"
)

func (s *Service) ListProjectResearchStates(ctx context.Context, req *apiclient.ListProjectResearchStatesRequest) (*apiclient.ListProjectResearchStatesResponse, error) {
	store, e := requiredStore(s.tokenStore())
	if e != nil {
		return nil, e
	}
	if e = validateNonNegativeInt64Field("project_id", req.GetProjectId()); e != nil {
		return nil, e
	}
	if e = validateNonNegativeInt64Field("chain_id", req.GetChainId()); e != nil {
		return nil, e
	}
	statusValue := strings.TrimSpace(req.GetStatus())
	if e = validateProjectResearchStatus(statusValue); e != nil {
		return nil, e
	}
	page, e := store.ListProjectResearchStatesPage(ctx, req.GetChainId(), req.GetProjectId(), statusValue, req.GetPage(), req.GetPageSize())
	if e != nil {
		return nil, wrapStoreError("list project research states", e)
	}
	return &apiclient.ListProjectResearchStatesResponse{ResearchStates: mapProjectResearchStates(page.Items), Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}
func (s *Service) ListProjectReportRevisions(ctx context.Context, req *apiclient.ListProjectReportRevisionsRequest) (*apiclient.ListProjectReportRevisionsResponse, error) {
	store, e := requiredStore(s.tokenStore())
	if e != nil {
		return nil, e
	}
	if e = validateNonNegativeInt64Field("project_id", req.GetProjectId()); e != nil {
		return nil, e
	}
	if e = validateNonNegativeInt64Field("chain_id", req.GetChainId()); e != nil {
		return nil, e
	}
	page, e := store.ListProjectReportRevisionsPage(ctx, req.GetChainId(), req.GetProjectId(), req.GetPage(), req.GetPageSize())
	if e != nil {
		return nil, wrapStoreError("list project report revisions", e)
	}
	return &apiclient.ListProjectReportRevisionsResponse{ReportRevisions: mapProjectReportRevisions(page.Items), Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}
func (s *Service) ListProjectSelections(ctx context.Context, req *apiclient.ListProjectSelectionsRequest) (*apiclient.ListProjectSelectionsResponse, error) {
	store, e := requiredStore(s.tokenStore())
	if e != nil {
		return nil, e
	}
	if e = validateNonNegativeInt64Field("project_id", req.GetProjectId()); e != nil {
		return nil, e
	}
	if e = validateNonNegativeInt64Field("chain_id", req.GetChainId()); e != nil {
		return nil, e
	}
	outcome := strings.TrimSpace(req.GetOutcome())
	if e = validateProjectSelectionOutcome(outcome); e != nil {
		return nil, e
	}
	page, e := store.ListProjectSelectionsPage(ctx, req.GetChainId(), req.GetProjectId(), outcome, req.GetPage(), req.GetPageSize())
	if e != nil {
		return nil, wrapStoreError("list project selections", e)
	}
	return &apiclient.ListProjectSelectionsResponse{Selections: mapProjectSelections(page.Items), Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}
