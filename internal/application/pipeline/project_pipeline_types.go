package pipeline

import (
	"context"

	"github.com/useryege/athena/internal/application/model"
)

type Lifecycle interface {
	Start(ctx context.Context) error
	Stop() error
}

type ProjectDiscoveryIndexer interface {
	Lifecycle
}

type DiscoveryIntake interface {
	IntakeCandidates(ctx context.Context, items []model.DiscoveredProjectCandidate) error
	ScheduleProjects(ctx context.Context, items []model.DiscoveredProjectCandidate) error
}
