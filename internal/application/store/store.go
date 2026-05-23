package store

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ProjectMeta struct {
	BlockTime                          uint64
	BlockNumber                        uint64
	Contract                           common.Address
	Creator                            common.Address
	Tx                                 *types.Transaction
	TxHash                             common.Hash
	TxIndex                            uint64
	SourceCode                         string
	SourceCodeHash                     common.Hash
	SourceCodeFetchedAt                time.Time
	CodeBinHash                        common.Hash
	CodeBinHashFetchedAt               time.Time
	SourceQualityReport                string
	SourceQualityReportFetchedAt       time.Time
	CreatorResult                      SimulateResult
	CreatorResultFetchedAt             time.Time
	GenesisWalletsFetchedAt            time.Time
	CreatorHistoricalProjectsFetchedAt time.Time
}

type SimulateResult struct {
	CanMintFromDeadViaTransferFrom     bool
	CanMintFromZeroViaTransferFrom     bool
	CanMintFromWethPairViaTransferFrom bool
	CanMintFromUsdtPairViaTransferFrom bool
	CanMintViaTransferToWethPair       bool
	CanMintViaTransferToUsdtPair       bool
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

type ProjectCreatorHistoricalProject struct {
	ID                        int64
	ProjectContract           common.Address
	HistoricalProjectContract common.Address
	RankIndex                 int32
	CreatedAt                 time.Time
}

type ProjectStore interface {
	SaveProjectMeta(ctx context.Context, meta ProjectMeta) error
	ListProjectMetas(ctx context.Context) ([]ProjectMeta, error)
	ListAllProjectMetas(ctx context.Context) ([]ProjectMeta, error)
	UpdateProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error
	UpdateProjectCodeBinHash(ctx context.Context, contract common.Address, codeBinHash common.Hash) error
	UpdateProjectSourceQualityReport(ctx context.Context, contract common.Address, report string) error
	UpdateProjectCreatorResult(ctx context.Context, contract common.Address, result SimulateResult) error
	ListProjectMetasByCreator(ctx context.Context, creator common.Address) ([]ProjectMeta, error)
	ListProjectMetasByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectMeta, error)
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

type ProjectCreatorHistoricalProjectStore interface {
	ReplaceProjectCreatorHistoricalProjects(ctx context.Context, contract common.Address, items []ProjectCreatorHistoricalProject) error
	ListProjectCreatorHistoricalProjectsByContract(ctx context.Context, contract common.Address) ([]ProjectCreatorHistoricalProject, error)
	ListProjectCreatorHistoricalProjectsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address][]ProjectCreatorHistoricalProject, error)
}

type Store interface {
	ProjectStore
}
