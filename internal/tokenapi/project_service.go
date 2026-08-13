package tokenapi

import (
	"context"
	"math/big"
	"strings"

	"github.com/useryege/athena/internal/token/projectview"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/selection"
	"github.com/useryege/athena/internal/token/swap"
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
	reportPairKind, err := validateProjectListReportPairKind(req.GetReportPairKind())
	if err != nil {
		return nil, err
	}
	reportPairRemoveLiquidityStates, err := normalizeProjectListReportPairStates(
		"report_pair_remove_liquidity_states",
		req.GetReportPairRemoveLiquidityStates(),
		[]string{"detected", "clear", "no_report", "risk_unavailable"},
	)
	if err != nil {
		return nil, err
	}
	reportPairMintStates, err := normalizeProjectListReportPairStates(
		"report_pair_mint_states",
		req.GetReportPairMintStates(),
		[]string{"detected", "clear", "no_report", "risk_unavailable"},
	)
	if err != nil {
		return nil, err
	}
	reportPairQuoteUSDTMin, err := parseOptionalNonNegativeIntegerField("report_pair_quote_usdt_min", req.GetReportPairQuoteUsdtMin())
	if err != nil {
		return nil, err
	}
	reportPairQuoteUSDTMax, err := parseOptionalNonNegativeIntegerField("report_pair_quote_usdt_max", req.GetReportPairQuoteUsdtMax())
	if err != nil {
		return nil, err
	}
	if reportPairQuoteUSDTMin != nil && reportPairQuoteUSDTMax != nil && reportPairQuoteUSDTMin.Cmp(reportPairQuoteUSDTMax) > 0 {
		return nil, status.Error(codes.InvalidArgument, "report_pair_quote_usdt_min must not exceed report_pair_quote_usdt_max")
	}
	reportPairQuoteMissingStates, err := normalizeProjectListReportPairStates(
		"report_pair_quote_missing_states",
		req.GetReportPairQuoteMissingStates(),
		[]string{"no_report", "risk_unavailable"},
	)
	if err != nil {
		return nil, err
	}
	hasReportPairFilter := len(reportPairRemoveLiquidityStates) > 0 ||
		len(reportPairMintStates) > 0 ||
		reportPairQuoteUSDTMin != nil ||
		reportPairQuoteUSDTMax != nil ||
		len(reportPairQuoteMissingStates) > 0
	if hasReportPairFilter && reportPairKind == "" {
		return nil, status.Error(codes.InvalidArgument, "report_pair_kind is required when a report Pair filter is set")
	}
	page, err := application.ListProjectsPage(ctx, projectview.ProjectListFilter{
		ChainID:                         req.GetChainId(),
		ProjectID:                       req.GetProjectId(),
		CodeHash:                        codeHash,
		Contract:                        contract,
		ResearchStatus:                  research.ProjectResearchStatus(researchStatus),
		ReportState:                     reportState,
		EvaluationStatus:                selection.TaskStatus(evaluationStatus),
		SelectionOutcome:                selection.SelectionOutcome(selectionOutcome),
		ReportPairKind:                  swap.PairKind(reportPairKind),
		ReportPairRemoveLiquidityStates: reportPairRemoveLiquidityStates,
		ReportPairMintStates:            reportPairMintStates,
		ReportPairQuoteUSDTMin:          reportPairQuoteUSDTMin,
		ReportPairQuoteUSDTMax:          reportPairQuoteUSDTMax,
		ReportPairQuoteMissingStates:    reportPairQuoteMissingStates,
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

func validateProjectListReportPairKind(value string) (string, error) {
	value = strings.TrimSpace(value)
	switch value {
	case "", string(swap.PairKindWETH), string(swap.PairKindUSDT):
		return value, nil
	default:
		return "", status.Error(codes.InvalidArgument, "report_pair_kind must be empty, weth, or usdt")
	}
}

func normalizeProjectListReportPairStates(field string, values, allowed []string) ([]string, error) {
	selected := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		valid := false
		for _, allowedValue := range allowed {
			if value == allowedValue {
				valid = true
				break
			}
		}
		if !valid {
			return nil, status.Errorf(codes.InvalidArgument, "%s contains invalid state %q", field, value)
		}
		selected[value] = struct{}{}
	}
	result := make([]string, 0, len(selected))
	for _, value := range allowed {
		if _, ok := selected[value]; ok {
			result = append(result, value)
		}
	}
	return result, nil
}

func parseOptionalNonNegativeIntegerField(field, value string) (*big.Int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return nil, status.Errorf(codes.InvalidArgument, "%s must be a non-negative decimal integer", field)
		}
	}
	result, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "%s must be a non-negative decimal integer", field)
	}
	return result, nil
}
