package store

import (
	"context"

	"github.com/useryege/athena/internal/token/domain"
)

func (s *SQLStore) ListCurrentProjectObservations(ctx context.Context, projectID int64) ([]domain.ProjectObservation, error) {
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
