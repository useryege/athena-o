package application

import (
	"context"

	"github.com/useryege/athena/internal/token/research"
)

type ReadRepository interface {
	GetProjectDataCollectionTask(context.Context, int64) (*research.ProjectDataCollectionTask, error)
	ListProjectDataCollectionTasks(context.Context, int64, string, string, int32, int32) (*research.CollectionTaskPage, error)
	ListProjectResearchStatesPage(context.Context, int64, int64, string, int32, int32) (*research.ResearchStatePage, error)
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
