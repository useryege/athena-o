package application

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/reporting"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/selection"
)

type ReadRepository interface {
	GetProjectDataCollectionTask(context.Context, int64) (*research.ProjectDataCollectionTask, error)
	ListProjectDataCollectionTasks(context.Context, int64, string, string, int32, int32) (*research.CollectionTaskPage, error)
	ListProjectResearchStatesPage(context.Context, int64, int64, string, int32, int32) (*research.ResearchStatePage, error)
	ListProjectReportsPage(context.Context, int64, int64, common.Address, string, int32, int32) (*reporting.ProjectReportPage, error)
	ListProjectReportRevisionsPage(context.Context, int64, int64, int32, int32) (*reporting.ReportRevisionPage, error)
	ListProjectSelectionsPage(context.Context, int64, int64, string, int32, int32) (*selection.Page, error)
}

type Queries struct {
	repository ReadRepository
}

func NewQueries(repository ReadRepository) *Queries {
	return &Queries{repository: repository}
}

func (q *Queries) GetProjectDataCollectionTask(ctx context.Context, id int64) (*research.ProjectDataCollectionTask, error) {
	return q.repository.GetProjectDataCollectionTask(ctx, id)
}

func (q *Queries) ListProjectDataCollectionTasks(ctx context.Context, projectID int64, dataType, status string, page, size int32) (*research.CollectionTaskPage, error) {
	return q.repository.ListProjectDataCollectionTasks(ctx, projectID, dataType, status, page, size)
}

func (q *Queries) ListProjectResearchStatesPage(ctx context.Context, chainID, projectID int64, status string, page, size int32) (*research.ResearchStatePage, error) {
	return q.repository.ListProjectResearchStatesPage(ctx, chainID, projectID, status, page, size)
}

func (q *Queries) ListProjectReportsPage(ctx context.Context, chainID, projectID int64, contract common.Address, status string, page, size int32) (*reporting.ProjectReportPage, error) {
	return q.repository.ListProjectReportsPage(ctx, chainID, projectID, contract, status, page, size)
}

func (q *Queries) ListProjectReportRevisionsPage(ctx context.Context, chainID, projectID int64, page, size int32) (*reporting.ReportRevisionPage, error) {
	return q.repository.ListProjectReportRevisionsPage(ctx, chainID, projectID, page, size)
}

func (q *Queries) ListProjectSelectionsPage(ctx context.Context, chainID, projectID int64, outcome string, page, size int32) (*selection.Page, error) {
	return q.repository.ListProjectSelectionsPage(ctx, chainID, projectID, outcome, page, size)
}
