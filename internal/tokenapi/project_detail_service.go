package tokenapi

import (
	"context"
	"strconv"
	"strings"

	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/swap"
	"github.com/useryege/athena/internal/tokenapi/apiclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) GetProjectDetail(ctx context.Context, req *apiclient.GetProjectDetailRequest) (*apiclient.GetProjectDetailResponse, error) {
	if err := validatePositiveInt64Field("project_id", req.GetProjectId()); err != nil {
		return nil, err
	}
	application, err := s.projectViewApplication()
	if err != nil {
		return nil, err
	}
	item, err := application.GetProjectDetail(ctx, req.GetProjectId())
	if err != nil {
		return nil, wrapStoreError("get project detail", err)
	}
	if item == nil {
		return &apiclient.GetProjectDetailResponse{}, nil
	}
	mapped, err := mapProjectDetail(*item)
	if err != nil {
		return nil, wrapStoreError("map project detail", err)
	}
	return &apiclient.GetProjectDetailResponse{Found: true, Detail: mapped}, nil
}

func (s *Service) GetProjectSwapActivity(ctx context.Context, req *apiclient.GetProjectSwapActivityRequest) (*apiclient.GetProjectSwapActivityResponse, error) {
	if err := validatePositiveInt64Field("project_id", req.GetProjectId()); err != nil {
		return nil, err
	}
	application, err := s.projectViewApplication()
	if err != nil {
		return nil, err
	}
	item, err := application.GetProjectSwapActivity(ctx, req.GetProjectId())
	if err != nil {
		return nil, wrapStoreError("get project Swap activity", err)
	}
	if item == nil {
		return &apiclient.GetProjectSwapActivityResponse{}, nil
	}
	return &apiclient.GetProjectSwapActivityResponse{
		Found:    true,
		Activity: mapProjectSwapActivity(*item),
	}, nil
}

func (s *Service) ListProjectSwapEvents(ctx context.Context, req *apiclient.ListProjectSwapEventsRequest) (*apiclient.ListProjectSwapEventsResponse, error) {
	if err := validatePositiveInt64Field("project_id", req.GetProjectId()); err != nil {
		return nil, err
	}
	pairKind := swap.PairKind(strings.TrimSpace(req.GetPairKind()))
	switch pairKind {
	case swap.PairKindWETH, swap.PairKindUSDT:
	default:
		return nil, status.Error(codes.InvalidArgument, "pair_kind must be weth or usdt")
	}
	blockNumber, err := strconv.ParseUint(strings.TrimSpace(req.GetBlockNumber()), 10, 64)
	if err != nil || blockNumber == 0 {
		return nil, status.Error(codes.InvalidArgument, "block_number must be positive")
	}
	application, err := s.projectViewApplication()
	if err != nil {
		return nil, err
	}
	page, err := application.ListProjectSwapEventsPage(
		ctx,
		req.GetProjectId(),
		pairKind,
		blockNumber,
		req.GetPage(),
		req.GetPageSize(),
	)
	if err != nil {
		return nil, wrapStoreError("list project Swap events", err)
	}
	return &apiclient.ListProjectSwapEventsResponse{
		Events:   mapProjectSwapEvents(page.Items),
		Total:    page.Total,
		Page:     page.Page,
		PageSize: page.PageSize,
	}, nil
}

func (s *Service) ListProjectTrends(ctx context.Context, req *apiclient.ListProjectTrendsRequest) (*apiclient.ListProjectTrendsResponse, error) {
	if err := validatePositiveInt64Field("project_id", req.GetProjectId()); err != nil {
		return nil, err
	}
	rangeValue := strings.TrimSpace(req.GetRange())
	if rangeValue == "" {
		rangeValue = "24h"
	}
	switch rangeValue {
	case "1h", "6h", "24h", "7d":
	default:
		return nil, status.Error(codes.InvalidArgument, "range must be one of 1h, 6h, 24h, or 7d")
	}
	application, err := s.projectViewApplication()
	if err != nil {
		return nil, err
	}
	result, err := application.ListProjectTrends(ctx, req.GetProjectId(), rangeValue)
	if err != nil {
		return nil, wrapStoreError("list project trends", err)
	}
	return &apiclient.ListProjectTrendsResponse{Trends: mapProjectTrends(*result)}, nil
}

func (s *Service) ListProjectObservations(ctx context.Context, req *apiclient.ListProjectObservationsRequest) (*apiclient.ListProjectObservationsResponse, error) {
	if err := validatePositiveInt64Field("project_id", req.GetProjectId()); err != nil {
		return nil, err
	}
	dataType := strings.TrimSpace(req.GetDataType())
	if err := validateProjectObservationType(dataType); err != nil {
		return nil, err
	}
	application, err := s.projectViewApplication()
	if err != nil {
		return nil, err
	}
	page, err := application.ListProjectObservationsPage(ctx, req.GetProjectId(), dataType, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, wrapStoreError("list project observations", err)
	}
	return &apiclient.ListProjectObservationsResponse{
		Observations: mapProjectObservations(page.Items),
		Total:        page.Total,
		Page:         page.Page,
		PageSize:     page.PageSize,
	}, nil
}

func (s *Service) ListProjectWalletNormalTransactions(
	ctx context.Context,
	req *apiclient.ListProjectWalletNormalTransactionsRequest,
) (*apiclient.ListProjectWalletNormalTransactionsResponse, error) {
	if err := validatePositiveInt64Field("project_id", req.GetProjectId()); err != nil {
		return nil, err
	}
	wallet, err := parseOptionalAddressField("wallet", req.GetWallet())
	if err != nil {
		return nil, err
	}
	receiptStatus := strings.TrimSpace(req.GetReceiptStatus())
	switch receiptStatus {
	case "", string(research.NormalTransactionReceiptStatusUnspecified), string(research.NormalTransactionReceiptStatusFailed), string(research.NormalTransactionReceiptStatusSuccess):
	default:
		return nil, status.Error(codes.InvalidArgument, "receipt_status must be unspecified, failed, or success")
	}
	application, err := s.projectViewApplication()
	if err != nil {
		return nil, err
	}
	page, err := application.ListProjectWalletNormalTransactionsPage(
		ctx,
		req.GetProjectId(),
		wallet,
		receiptStatus,
		strings.TrimSpace(req.GetMethodId()),
		req.GetPage(),
		req.GetPageSize(),
	)
	if err != nil {
		return nil, wrapStoreError("list project wallet normal transactions", err)
	}
	return &apiclient.ListProjectWalletNormalTransactionsResponse{
		Transactions: mapWalletNormalTransactions(page.Items),
		Total:        page.Total,
		Page:         page.Page,
		PageSize:     page.PageSize,
	}, nil
}

func validateProjectObservationType(value string) error {
	switch value {
	case "",
		string(research.DataCollectionTypeAve),
		string(research.DataCollectionTypeChainState),
		string(research.DataCollectionTypeWalletAssetState),
		string(research.DataCollectionTypeSimulationResult),
		string(research.DataCollectionTypeContractCodeSource):
		return nil
	default:
		return status.Error(codes.InvalidArgument, "data_type must be ave, chain_state, wallet_asset_state, simulation_result, or contract_code_source")
	}
}
