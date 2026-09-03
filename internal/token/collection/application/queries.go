package application

import (
	"context"

	"github.com/useryege/athena/internal/token/collection"
)

type ReadRepository interface {
	GetCollectionTask(context.Context, int64) (*collection.TaskDetail, error)
	ListCollectionTasks(context.Context, int64, string, string, int32, int32) (*collection.TaskPage, error)
}

type Queries struct {
	repository ReadRepository
}

func NewQueries(repository ReadRepository) *Queries { return &Queries{repository: repository} }

func (queries *Queries) GetCollectionTask(ctx context.Context, id int64) (*collection.TaskDetail, error) {
	return queries.repository.GetCollectionTask(ctx, id)
}

func (queries *Queries) ListCollectionTasks(ctx context.Context, projectID int64, dataType, status string, page, size int32) (*collection.TaskPage, error) {
	return queries.repository.ListCollectionTasks(ctx, projectID, dataType, status, page, size)
}
