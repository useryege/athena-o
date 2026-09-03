package application

import (
	"context"
	"fmt"

	"github.com/useryege/athena/internal/token/profile"
)

type ReadRepository interface {
	GetProjectProfile(context.Context, int64) (*profile.ProjectProfile, error)
}

type Queries struct {
	repository ReadRepository
}

func NewQueries(repository ReadRepository) *Queries {
	return &Queries{repository: repository}
}

func (queries *Queries) GetProjectProfile(ctx context.Context, projectID int64) (*profile.ProjectProfile, error) {
	if queries == nil || queries.repository == nil {
		return nil, fmt.Errorf("token profile queries are not configured")
	}
	if projectID <= 0 {
		return nil, fmt.Errorf("token profile project id must be positive")
	}
	return queries.repository.GetProjectProfile(ctx, projectID)
}
