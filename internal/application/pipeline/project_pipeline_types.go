package pipeline

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/model"
)

type Lifecycle interface {
	Start(ctx context.Context) error
	Stop() error
}

type ProjectDiscoveryIndexer interface {
	Lifecycle
}

type ProjectStateReconciler interface {
	Lifecycle
	InitProject(ctx context.Context, candidates []model.DiscoveredProjectCandidate) error
	ScheduleProjects(ctx context.Context, candidates []model.DiscoveredProjectCandidate) error
}

type ProjectPolicyEngine interface {
	EvaluateProject(ctx context.Context, contract common.Address) (model.ProjectReport, error)
}

type DiscoveryIntake interface {
	IntakeCandidates(ctx context.Context, items []model.DiscoveredProjectCandidate) error
	ScheduleProjects(ctx context.Context, items []model.DiscoveredProjectCandidate) error
}
