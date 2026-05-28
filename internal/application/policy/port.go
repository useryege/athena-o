package policy

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/model"
)

type Engine interface {
	EvaluateProject(ctx context.Context, contract common.Address) (model.ProjectReport, error)
}
