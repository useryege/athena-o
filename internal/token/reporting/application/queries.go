package application

import (
	"context"

	"github.com/useryege/athena/internal/token/reporting"
)

type ReadRepository interface {
	ListProjectReportRevisionsPage(context.Context, int64, int64, int32, int32) (*reporting.ReportRevisionPage, error)
}

type Queries struct{ repository ReadRepository }

func NewQueries(repository ReadRepository) *Queries { return &Queries{repository: repository} }

func (queries *Queries) ListProjectReportRevisionsPage(ctx context.Context, chainID, projectID int64, page, size int32) (*reporting.ReportRevisionPage, error) {
	return queries.repository.ListProjectReportRevisionsPage(ctx, chainID, projectID, page, size)
}
