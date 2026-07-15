package projectvalidator

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/token/domain"
)

type Store interface {
	ClaimProjectCandidateValidations(context.Context, int64, uuid.UUID, time.Duration, int32) ([]domain.ProjectCandidate, error)
	RenewProjectCandidateValidationClaims(context.Context, uuid.UUID, time.Duration) error
	ReleaseProjectCandidateValidationClaims(context.Context, uuid.UUID) error
	RejectProjectCandidate(context.Context, domain.ProjectCandidate) error
	ValidateProjectCandidate(context.Context, domain.ProjectCandidate, common.Hash, domain.ProjectTokenMetadata, common.Address, common.Address, []domain.ProjectRelatedWallet, []domain.ProjectInitialRecipient) (*domain.Project, error)
}
