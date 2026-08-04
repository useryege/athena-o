package tokenapi

import (
	"context"
	"strings"

	"github.com/useryege/athena/internal/token/projectview"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/selection"
	"github.com/useryege/athena/internal/tokenapi/apiclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) ListProjects(ctx context.Context, req *apiclient.ListProjectsRequest) (*apiclient.ListProjectsResponse, error) {
	application, err := s.projectViewApplication()
	if err != nil {
		return nil, err
	}
	if err := validateNonNegativeInt64Field("chain_id", req.GetChainId()); err != nil {
		return nil, err
	}
	if err := validateNonNegativeInt64Field("project_id", req.GetProjectId()); err != nil {
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
	researchStatus := strings.TrimSpace(req.GetResearchStatus())
	if err := validateProjectListResearchStatus(researchStatus); err != nil {
		return nil, err
	}
	reportState := strings.TrimSpace(req.GetReportState())
	if err := validateProjectListReportState(reportState); err != nil {
		return nil, err
	}
	evaluationStatus := strings.TrimSpace(req.GetEvaluationStatus())
	if err := validateProjectListEvaluationStatus(evaluationStatus); err != nil {
		return nil, err
	}
	selectionOutcome := strings.TrimSpace(req.GetSelectionOutcome())
	if err := validateProjectListSelectionOutcome(selectionOutcome); err != nil {
		return nil, err
	}
	page, err := application.ListProjectsPage(ctx, projectview.ProjectListFilter{
		ChainID:          req.GetChainId(),
		ProjectID:        req.GetProjectId(),
		CodeHash:         codeHash,
		Contract:         contract,
		ResearchStatus:   research.ProjectResearchStatus(researchStatus),
		ReportState:      reportState,
		EvaluationStatus: selection.TaskStatus(evaluationStatus),
		SelectionOutcome: selection.SelectionOutcome(selectionOutcome),
	}, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, wrapStoreError("list projects", err)
	}
	return &apiclient.ListProjectsResponse{
		Projects: mapProjectListItems(page.Items),
		Total:    page.Total,
		Page:     page.Page,
		PageSize: page.PageSize,
	}, nil
}

func validateProjectListResearchStatus(value string) error {
	switch value {
	case "",
		string(research.ProjectResearchStatusResearching),
		string(research.ProjectResearchStatusSelected),
		string(research.ProjectResearchStatusRejected),
		string(research.ProjectResearchStatusExpired):
		return nil
	default:
		return status.Error(codes.InvalidArgument, "research_status must be empty, researching, selected, rejected, or expired")
	}
}

func validateProjectListReportState(value string) error {
	switch value {
	case "", "none", "incomplete", "complete":
		return nil
	default:
		return status.Error(codes.InvalidArgument, "report_state must be empty, none, incomplete, or complete")
	}
}

func validateProjectListEvaluationStatus(value string) error {
	switch value {
	case "none",
		string(selection.TaskStatusPending),
		string(selection.TaskStatusRunning),
		string(selection.TaskStatusSucceeded),
		string(selection.TaskStatusFailed):
		return nil
	case "":
		return nil
	default:
		return status.Error(codes.InvalidArgument, "evaluation_status must be empty, none, pending, running, succeeded, or failed")
	}
}

func validateProjectListSelectionOutcome(value string) error {
	switch value {
	case "none",
		string(selection.SelectionOutcomeSelected),
		string(selection.SelectionOutcomeRejected),
		string(selection.SelectionOutcomeDeferred):
		return nil
	case "":
		return nil
	default:
		return status.Error(codes.InvalidArgument, "selection_outcome must be empty, none, selected, rejected, or deferred")
	}
}
