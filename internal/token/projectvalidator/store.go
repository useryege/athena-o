package projectvalidator

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/domain"
)

type Store interface {
	ListPendingProjectCandidates(context.Context, int64, int32) ([]domain.ProjectCandidate, error)
	MarkProjectCandidateStatus(context.Context, int64, domain.ProjectCandidateStatus) (*domain.ProjectCandidate, error)
	ValidateProjectCandidate(context.Context, domain.ProjectCandidate, common.Hash, domain.ProjectTokenMetadata, common.Address, common.Address, []domain.ProjectRelatedWallet, []domain.ProjectInitialRecipient) (*domain.Project, error)
}
