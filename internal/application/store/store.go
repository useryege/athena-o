package store

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
)

type ProjectMeta struct {
	ProjectID   uuid.UUID
	BlockTime   uint64
	BlockNumber uint64
	Contract    common.Address
	Creator     common.Address
	Tx          *types.Transaction
	TxHash      common.Hash
	TxIndex     uint64
	SourceCode  string
}

type ProjectStore interface {
	SaveProjectMeta(ctx context.Context, meta ProjectMeta) error
	ListProjectMetas(ctx context.Context) ([]ProjectMeta, error)
	UpdateProjectSourceCode(ctx context.Context, projectID uuid.UUID, sourceCode string) error
}

type SourceCodeBlacklistStore interface {
	ListSourceCodeBlacklistFields(ctx context.Context) ([]string, error)
	AddSourceCodeBlacklistField(ctx context.Context, field string) error
	DeleteSourceCodeBlacklistField(ctx context.Context, field string) error
}

type Store interface {
	ProjectStore
	SourceCodeBlacklistStore
}
