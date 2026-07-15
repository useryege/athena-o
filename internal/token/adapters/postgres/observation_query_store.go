package postgres

import (
	"context"

	"github.com/useryege/athena/internal/token/research"
)

func (s *Database) ListCurrentProjectObservations(ctx context.Context, projectID int64) ([]research.ProjectObservation, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	rows, e := q.ListCurrentProjectObservations(ctx, projectID)
	if e != nil {
		return nil, e
	}
	return mapCurrentProjectObservations(rows)
}
