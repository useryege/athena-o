package tokenapi

import (
	"context"
	"math/big"
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
	wethPairFilter, err := parseProjectListPairFilter(
		"weth_pair",
		req.GetWethPairRemoveLiquidityStates(),
		req.GetWethPairMintStates(),
		req.GetWethPairQuoteUsdtMin(),
		req.GetWethPairQuoteUsdtMax(),
		req.GetWethPairQuoteMissingStates(),
	)
	if err != nil {
		return nil, err
	}
	usdtPairFilter, err := parseProjectListPairFilter(
		"usdt_pair",
		req.GetUsdtPairRemoveLiquidityStates(),
		req.GetUsdtPairMintStates(),
		req.GetUsdtPairQuoteUsdtMin(),
		req.GetUsdtPairQuoteUsdtMax(),
		req.GetUsdtPairQuoteMissingStates(),
	)
	if err != nil {
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
		WethPair:         wethPairFilter,
		UsdtPair:         usdtPairFilter,
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

func parseProjectListPairFilter(
	prefix string,
	removeLiquidityStates, mintStates []string,
	quoteUSDTMinValue, quoteUSDTMaxValue string,
	quoteMissingStates []string,
) (projectview.ProjectListPairFilter, error) {
	removeLiquidity, err := normalizeProjectListReportPairStates(
		prefix+"_remove_liquidity_states",
		removeLiquidityStates,
		[]string{"detected", "clear", "no_report", "risk_unavailable"},
	)
	if err != nil {
		return projectview.ProjectListPairFilter{}, err
	}
	mint, err := normalizeProjectListReportPairStates(
		prefix+"_mint_states",
		mintStates,
		[]string{"detected", "clear", "no_report", "risk_unavailable"},
	)
	if err != nil {
		return projectview.ProjectListPairFilter{}, err
	}
	quoteUSDTMin, err := parseOptionalNonNegativeIntegerField(prefix+"_quote_usdt_min", quoteUSDTMinValue)
	if err != nil {
		return projectview.ProjectListPairFilter{}, err
	}
	quoteUSDTMax, err := parseOptionalNonNegativeIntegerField(prefix+"_quote_usdt_max", quoteUSDTMaxValue)
	if err != nil {
		return projectview.ProjectListPairFilter{}, err
	}
	if quoteUSDTMin != nil && quoteUSDTMax != nil && quoteUSDTMin.Cmp(quoteUSDTMax) > 0 {
		return projectview.ProjectListPairFilter{}, status.Errorf(codes.InvalidArgument, "%s_quote_usdt_min must not exceed %s_quote_usdt_max", prefix, prefix)
	}
	quoteMissing, err := normalizeProjectListReportPairStates(
		prefix+"_quote_missing_states",
		quoteMissingStates,
		[]string{"no_report", "risk_unavailable"},
	)
	if err != nil {
		return projectview.ProjectListPairFilter{}, err
	}
	return projectview.ProjectListPairFilter{
		RemoveLiquidityStates: removeLiquidity,
		MintStates:            mint,
		QuoteUSDTMin:          quoteUSDTMin,
		QuoteUSDTMax:          quoteUSDTMax,
		QuoteMissingStates:    quoteMissing,
	}, nil
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
