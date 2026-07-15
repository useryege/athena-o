package application

import (
	"context"

	"github.com/useryege/athena/internal/token/selection"
)

type ReadRepository interface {
	ListProjectSelectionsPage(context.Context, int64, int64, string, int32, int32) (*selection.Page, error)
}

type Queries struct{ repository ReadRepository }

func NewQueries(repository ReadRepository) *Queries { return &Queries{repository: repository} }

func (queries *Queries) ListProjectSelectionsPage(ctx context.Context, chainID, projectID int64, outcome string, page, size int32) (*selection.Page, error) {
	return queries.repository.ListProjectSelectionsPage(ctx, chainID, projectID, outcome, page, size)
}
