package tokenapi

import (
	"context"
	"strings"

	"github.com/useryege/athena/internal/tokenapi/apiclient"
)

func (s *Service) ListResearchStates(ctx context.Context, req *apiclient.ListResearchStatesRequest) (*apiclient.ListResearchStatesResponse, error) {
	store, e := s.researchApplication()
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
	return &apiclient.ListResearchStatesResponse{ResearchStates: mapProjectResearchStates(page.Items), Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}
func (s *Service) ListReportRevisions(ctx context.Context, req *apiclient.ListReportRevisionsRequest) (*apiclient.ListReportRevisionsResponse, error) {
	store, e := s.researchApplication()
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
	return &apiclient.ListReportRevisionsResponse{ReportRevisions: mapProjectReportRevisions(page.Items), Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}
func (s *Service) ListSelections(ctx context.Context, req *apiclient.ListSelectionsRequest) (*apiclient.ListSelectionsResponse, error) {
	store, e := s.researchApplication()
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
	return &apiclient.ListSelectionsResponse{Selections: mapProjectSelections(page.Items), Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}
