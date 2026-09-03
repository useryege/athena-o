package application

import (
	"context"
	"fmt"
	"time"

	"github.com/useryege/athena/internal/token/projectview"
	"github.com/useryege/athena/internal/token/shared"
	"github.com/useryege/athena/internal/token/swap"
)

type ReadRepository interface {
	ListProjectsPage(context.Context, projectview.ProjectListFilter, int32, int32) (*projectview.ProjectListPage, error)
	GetProjectDetail(context.Context, int64) (*projectview.Detail, error)
	GetProjectSwapActivity(context.Context, int64) (*projectview.SwapActivity, error)
	ListProjectSwapEventsPage(context.Context, int64, swap.PairKind, uint64, int32, int32) (*projectview.SwapEventPage, error)
	ListProjectWalletNormalTransactionsPage(context.Context, int64, shared.Address, string, string, int32, int32) (*projectview.WalletNormalTransactionPage, error)
}

type Queries struct {
	repository ReadRepository
	now        func() time.Time
}

func NewQueries(repository ReadRepository) *Queries {
	return &Queries{repository: repository, now: time.Now}
}

func (queries *Queries) ListProjectsPage(ctx context.Context, filter projectview.ProjectListFilter, page, pageSize int32) (*projectview.ProjectListPage, error) {
	return queries.repository.ListProjectsPage(ctx, filter, page, pageSize)
}

func (queries *Queries) GetProjectDetail(ctx context.Context, projectID int64) (*projectview.Detail, error) {
	detail, err := queries.repository.GetProjectDetail(ctx, projectID)
	if err != nil || detail == nil {
		return detail, err
	}
	detail.GeneratedAt = queries.now().UTC()
	return detail, nil
}

func (queries *Queries) GetProjectSwapActivity(ctx context.Context, projectID int64) (*projectview.SwapActivity, error) {
	activity, err := queries.repository.GetProjectSwapActivity(ctx, projectID)
	if err != nil || activity == nil {
		return activity, err
	}
	activity.GeneratedAt = queries.now().UTC()
	return activity, nil
}

func (queries *Queries) ListProjectSwapEventsPage(
	ctx context.Context,
	projectID int64,
	pairKind swap.PairKind,
	blockNumber uint64,
	page, pageSize int32,
) (*projectview.SwapEventPage, error) {
	switch pairKind {
	case swap.PairKindWETH, swap.PairKindUSDT:
	default:
		return nil, fmt.Errorf("pair kind must be weth or usdt")
	}
	if blockNumber == 0 {
		return nil, fmt.Errorf("block number must be positive")
	}
	return queries.repository.ListProjectSwapEventsPage(ctx, projectID, pairKind, blockNumber, page, pageSize)
}

func (queries *Queries) ListProjectWalletNormalTransactionsPage(
	ctx context.Context,
	projectID int64,
	wallet shared.Address,
	receiptStatus string,
	methodID string,
	page, pageSize int32,
) (*projectview.WalletNormalTransactionPage, error) {
	return queries.repository.ListProjectWalletNormalTransactionsPage(ctx, projectID, wallet, receiptStatus, methodID, page, pageSize)
}
