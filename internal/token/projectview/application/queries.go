package application

import (
	"context"
	"time"

	"github.com/useryege/athena/internal/token/projectview"
	"github.com/useryege/athena/internal/token/shared"
)

type ReadRepository interface {
	ListProjectsPage(context.Context, projectview.ProjectListFilter, int32, int32) (*projectview.ProjectListPage, error)
	GetProjectDetail(context.Context, int64) (*projectview.Detail, error)
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
