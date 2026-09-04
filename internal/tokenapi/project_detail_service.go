package tokenapi

import (
	"context"
	"strings"

	"github.com/useryege/athena/internal/token/collection"
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

func (s *Service) GetProjectProfile(ctx context.Context, req *apiclient.GetProjectProfileRequest) (*apiclient.GetProjectProfileResponse, error) {
	if err := validatePositiveInt64Field("project_id", req.GetProjectId()); err != nil {
		return nil, err
	}
	application, err := s.profileApplication()
	if err != nil {
		return nil, err
	}
	item, err := application.GetProjectProfile(ctx, req.GetProjectId())
	if err != nil {
		return nil, wrapStoreError("get project profile", err)
	}
	if item == nil {
		return &apiclient.GetProjectProfileResponse{}, nil
	}
	return &apiclient.GetProjectProfileResponse{
		Found:   true,
		Profile: mapProjectProfile(*item),
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
	case "", string(collection.NormalTransactionReceiptStatusUnspecified), string(collection.NormalTransactionReceiptStatusFailed), string(collection.NormalTransactionReceiptStatusSuccess):
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
