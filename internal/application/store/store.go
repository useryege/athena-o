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
	TxIndex     uint64
}

type ProjectStore interface {
	SaveProjectMeta(ctx context.Context, meta ProjectMeta) error
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
