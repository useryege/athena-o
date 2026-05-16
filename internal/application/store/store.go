package store

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ProjectMeta struct {
	BlockTime   uint64
	BlockNumber uint64
	Contract    common.Address
	Creator     common.Address
	Tx          *types.Transaction
	TxHash      common.Hash
	TxIndex     uint64
	SourceCode  string
	IsArchived  bool
	ArchivedAt  time.Time
}

type ProjectStore interface {
	SaveProjectMeta(ctx context.Context, meta ProjectMeta) error
	ListProjectMetas(ctx context.Context) ([]ProjectMeta, error)
	ListAllProjectMetas(ctx context.Context) ([]ProjectMeta, error)
	UpdateProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error
	ArchiveProjectByContract(ctx context.Context, contract common.Address) error
	UnarchiveProjectByContract(ctx context.Context, contract common.Address) error
	ListArchivedProjectMetas(ctx context.Context, page int32, pageSize int32) ([]ProjectMeta, int64, int32, int32, error)
	GetArchivedProjectMetaByContract(ctx context.Context, contract common.Address) (*ProjectMeta, error)
	GetProjectMetaByContract(ctx context.Context, contract common.Address) (*ProjectMeta, error)
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
