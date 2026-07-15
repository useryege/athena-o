package application

import (
	"context"

	"github.com/useryege/athena/internal/token/reporting"
	"github.com/useryege/athena/internal/token/shared"
)

type ReadRepository interface {
	ListProjectReportsPage(context.Context, int64, int64, shared.Address, string, int32, int32) (*reporting.ProjectReportPage, error)
	ListProjectReportRevisionsPage(context.Context, int64, int64, int32, int32) (*reporting.ReportRevisionPage, error)
}

type Queries struct{ repository ReadRepository }

func NewQueries(repository ReadRepository) *Queries { return &Queries{repository: repository} }

func (queries *Queries) ListProjectReportsPage(ctx context.Context, chainID, projectID int64, contract shared.Address, status string, page, size int32) (*reporting.ProjectReportPage, error) {
	return queries.repository.ListProjectReportsPage(ctx, chainID, projectID, contract, status, page, size)
}

func (queries *Queries) ListProjectReportRevisionsPage(ctx context.Context, chainID, projectID int64, page, size int32) (*reporting.ReportRevisionPage, error) {
	return queries.repository.ListProjectReportRevisionsPage(ctx, chainID, projectID, page, size)
}
