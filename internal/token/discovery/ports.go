package discovery

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/catalog"
)

type ScannerRepository interface {
	GetChainIngestCheckpoint(context.Context, int64) (*ChainIngestCheckpoint, error)
	UpsertChainIngestCheckpoint(context.Context, ChainIngestCheckpoint) (*ChainIngestCheckpoint, error)
	UpdateChainIngestCheckpointStatus(context.Context, int64, ChainIngestStatus) (*ChainIngestCheckpoint, error)
	IngestProjectCandidateBatch(context.Context, ChainIngestCheckpoint, []ProjectCandidate) (*ChainIngestCheckpoint, error)
}

type ValidatorRepository interface {
	ClaimProjectCandidateValidations(context.Context, int64, string, time.Duration, int32) ([]ProjectCandidate, error)
	RenewProjectCandidateValidationClaims(context.Context, string, time.Duration) error
	ReleaseProjectCandidateValidationClaims(context.Context, string) error
	RejectProjectCandidate(context.Context, ProjectCandidate) error
	ValidateProjectCandidate(context.Context, ProjectCandidate, common.Hash, catalog.ProjectTokenMetadata, common.Address, common.Address, []catalog.ProjectRelatedWallet, []catalog.ProjectInitialRecipient) (*catalog.Project, error)
}
