package tokenapi

import (
	"context"
	"math/big"
	"strings"

	"github.com/useryege/athena/internal/token/projectview"
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
	collectionStatus := strings.TrimSpace(req.GetCollectionStatus())
	if err := validateProjectCollectionStatus(collectionStatus); err != nil {
		return nil, err
	}
	profileState := strings.TrimSpace(req.GetProfileState())
	if err := validateProjectProfileState(profileState); err != nil {
		return nil, err
	}
	wrappedNativePairFilter, err := parseProjectListPairFilter(
		"wrapped_native_pair",
		req.GetWrappedNativePairBalanceSupplyStates(),
		req.GetWrappedNativePairMinimumLpStates(),
		req.GetWrappedNativePairFeeLpShareStates(),
		req.GetWrappedNativePairQuoteUsdtMin(),
		req.GetWrappedNativePairQuoteUsdtMax(),
		req.GetWrappedNativePairQuoteMissingStates(),
	)
	if err != nil {
		return nil, err
	}
	usdtPairFilter, err := parseProjectListPairFilter(
		"usdt_pair",
		req.GetUsdtPairBalanceSupplyStates(),
		req.GetUsdtPairMinimumLpStates(),
		req.GetUsdtPairFeeLpShareStates(),
		req.GetUsdtPairQuoteUsdtMin(),
		req.GetUsdtPairQuoteUsdtMax(),
		req.GetUsdtPairQuoteMissingStates(),
	)
	if err != nil {
		return nil, err
	}
	page, err := application.ListProjectsPage(ctx, projectview.ProjectListFilter{
		ChainID:           req.GetChainId(),
		ProjectID:         req.GetProjectId(),
		CodeHash:          codeHash,
		Contract:          contract,
		CollectionStatus:  projectview.ProjectCollectionStatus(collectionStatus),
		ProfileState:      projectview.ProjectProfileState(profileState),
		WrappedNativePair: wrappedNativePairFilter,
		USDTPair:          usdtPairFilter,
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

func validateProjectCollectionStatus(value string) error {
	switch projectview.ProjectCollectionStatus(value) {
	case "",
		projectview.ProjectCollectionStatusQueued,
		projectview.ProjectCollectionStatusCollecting,
		projectview.ProjectCollectionStatusComplete,
		projectview.ProjectCollectionStatusNeedsAttention:
		return nil
	default:
		return status.Error(codes.InvalidArgument, "collection_status must be empty, queued, collecting, complete, or needs_attention")
	}
}

func validateProjectProfileState(value string) error {
	switch projectview.ProjectProfileState(value) {
	case "",
		projectview.ProjectProfileStatePending,
		projectview.ProjectProfileStateComplete,
		projectview.ProjectProfileStateIncomplete,
		projectview.ProjectProfileStateFailed:
		return nil
	default:
		return status.Error(codes.InvalidArgument, "profile_state must be empty, pending, complete, incomplete, or failed")
	}
}

func parseProjectListPairFilter(
	prefix string,
	balanceSupplyStates, minimumLPStates, feeLPShareStates []string,
	quoteUSDTMinValue, quoteUSDTMaxValue string,
	quoteMissingStates []string,
) (projectview.ProjectListPairFilter, error) {
	balanceSupply, err := normalizeProjectListPairStates(
		prefix+"_balance_supply_states",
		balanceSupplyStates,
		[]string{"detected", "clear", "no_profile", "signal_unavailable"},
	)
	if err != nil {
		return projectview.ProjectListPairFilter{}, err
	}
	minimumLP, err := normalizeProjectListPairStates(
		prefix+"_minimum_lp_states",
		minimumLPStates,
		[]string{"detected", "clear", "no_profile", "signal_unavailable"},
	)
	if err != nil {
		return projectview.ProjectListPairFilter{}, err
	}
	feeLPShare, err := normalizeProjectListPairStates(
		prefix+"_fee_lp_share_states",
		feeLPShareStates,
		[]string{"detected", "clear", "no_profile", "signal_unavailable"},
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
	quoteMissing, err := normalizeProjectListPairStates(
		prefix+"_quote_missing_states",
		quoteMissingStates,
		[]string{"no_profile", "value_unavailable"},
	)
	if err != nil {
		return projectview.ProjectListPairFilter{}, err
	}
	return projectview.ProjectListPairFilter{
		PairTokenBalanceExceedsTotalSupplyStates: balanceSupply,
		LPMinimumSupplyOnlyStates:                minimumLP,
		FixedFeeAddressLPShareGte90PercentStates: feeLPShare,
		QuoteUSDTMin:                             quoteUSDTMin,
		QuoteUSDTMax:                             quoteUSDTMax,
		QuoteMissingStates:                       quoteMissing,
	}, nil
}

func normalizeProjectListPairStates(field string, values, allowed []string) ([]string, error) {
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
