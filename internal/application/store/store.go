package store

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ProjectMeta struct {
	BlockTime               uint64
	BlockNumber             uint64
	Contract                common.Address
	Creator                 common.Address
	Tx                      *types.Transaction
	TxHash                  common.Hash
	TxIndex                 uint64
	SourceCode              string
	SourceCodeHash          common.Hash
	CodeBinHash             common.Hash
	SourceQualityReport     string
	SourceQualityReportedAt time.Time
}

type ProjectEventLog struct {
	ID             int64
	Contract       common.Address
	EventType      int16
	OccurredAt     time.Time
	Message        string
	Payload        string
	IdempotencyKey string
	CreatedAt      time.Time
}

type ProjectComment struct {
	ID        int64
	Contract  common.Address
	Username  string
	Content   string
	CreatedAt time.Time
}

type ProjectGenesisWallet struct {
	ID                int64
	ProjectContract   common.Address
	Wallet            common.Address
	NetAmount         *big.Int
	RatioBPS          int64
	RankIndex         int32
	TotalSupply       *big.Int
	SourceTxHash      common.Hash
	SourceBlockNumber uint64
	CreatedAt         time.Time
}

type ProjectStore interface {
	SaveProjectMeta(ctx context.Context, meta ProjectMeta) error
	ListProjectMetas(ctx context.Context) ([]ProjectMeta, error)
	ListAllProjectMetas(ctx context.Context) ([]ProjectMeta, error)
	UpdateProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error
	UpdateProjectCodeBinHash(ctx context.Context, contract common.Address, codeBinHash common.Hash) error
	UpdateProjectSourceQualityReport(ctx context.Context, contract common.Address, report string) error
	ListProjectMetasByCreator(ctx context.Context, creator common.Address) ([]ProjectMeta, error)
	GetProjectMetaByContract(ctx context.Context, contract common.Address) (*ProjectMeta, error)
}

type ProjectEventLogStore interface {
	AddProjectEventLog(ctx context.Context, item ProjectEventLog) error
	ListProjectEventLogsByContract(ctx context.Context, contract common.Address) ([]ProjectEventLog, error)
}

type ProjectCommentStore interface {
	AddProjectComment(ctx context.Context, item ProjectComment) (ProjectComment, error)
	ListProjectCommentsByContract(ctx context.Context, contract common.Address, page int32, pageSize int32) ([]ProjectComment, int64, int32, int32, error)
}

type ProjectGenesisWalletStore interface {
	ReplaceProjectGenesisWallets(ctx context.Context, contract common.Address, items []ProjectGenesisWallet) error
	ListProjectGenesisWalletsByContract(ctx context.Context, contract common.Address) ([]ProjectGenesisWallet, error)
	ListProjectGenesisWalletsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address][]ProjectGenesisWallet, error)
	ListProjectGenesisWalletsByWallet(ctx context.Context, wallet common.Address) ([]ProjectGenesisWallet, error)
}

type Store interface {
	ProjectStore
}
