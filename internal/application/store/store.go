package store

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/useryege/athena/internal/application/model"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type ProjectMeta = model.ProjectMeta
type ProjectAveDetail = model.ProjectAveDetail
type ProjectAveTokenDetail = model.ProjectAveTokenDetail
type ProjectAvePair = model.ProjectAvePair
type SimulateResult = model.SimulateResult
type ProjectReport = model.ProjectReport

type ProjectBase struct {
	BlockTime   uint64
	BlockNumber uint64
	Contract    common.Address
	Creator     common.Address
	Tx          *types.Transaction
	TxHash      common.Hash
	TxIndex     uint64
	CreatedAt   time.Time
}

type ProjectChainState struct {
	ProjectContract common.Address
	ChainState      athenacontract.AthenaProject
	RawChainState   json.RawMessage
	WethPair        common.Address
	UsdtPair        common.Address
	TokenName       string
	TokenSymbol     string
	FetchedAt       time.Time
	UpdatedAt       time.Time
}

type ProjectSimulationResult struct {
	ProjectContract common.Address
	Result          SimulateResult
	FetchedAt       time.Time
	UpdatedAt       time.Time
}

type ProjectReportState struct {
	ProjectContract common.Address
	Report          ProjectReport
	EvaluatedAt     time.Time
	UpdatedAt       time.Time
}

type ProjectBytecodeFact struct {
	ProjectContract       common.Address
	CodeHash              common.Hash
	IsBytecodeBlacklisted bool
	FetchedAt             time.Time
	UpdatedAt             time.Time
}

type ProjectComponentState struct {
	ProjectContract common.Address
	Component       string
	Status          string
	LastAttemptAt   time.Time
	LastSuccessAt   time.Time
	NextRunAt       time.Time
	LastError       string
	UpdatedAt       time.Time
}

const (
	ProjectComponentInitializer    = "initializer"
	ProjectComponentChainState     = "chain_state"
	ProjectComponentSimulation     = "simulation"
	ProjectComponentGenesisWallet  = "genesis_wallet"
	ProjectComponentCreatorHistory = "creator_history"
	ProjectComponentBytecodeFact   = "bytecode_fact"
	ProjectComponentAveDetail      = "ave_detail"
	ProjectComponentReport         = "report"

	ProjectComponentStatusPending = "pending"
	ProjectComponentStatusRunning = "running"
	ProjectComponentStatusSuccess = "success"
	ProjectComponentStatusFailed  = "failed"
)

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
	GetMaxProjectBlockNumber(ctx context.Context) (uint64, bool, error)
	ListProjectMetas(ctx context.Context) ([]ProjectMeta, error)
	ListAllProjectMetas(ctx context.Context) ([]ProjectMeta, error)
	ListProjectMetasByPairAddresses(ctx context.Context, pairs []common.Address) ([]ProjectMeta, error)
	UpsertProjectAveDetail(ctx context.Context, contract common.Address, detail ProjectAveDetail) error
	UpdateProjectCreatorResult(ctx context.Context, contract common.Address, result SimulateResult) error
	UpdateProjectReport(ctx context.Context, contract common.Address, report ProjectReport) error
	ListProjectMetasByCreator(ctx context.Context, creator common.Address) ([]ProjectMeta, error)
	ListProjectMetasByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectMeta, error)
	GetProjectMetaByContract(ctx context.Context, contract common.Address) (*ProjectMeta, error)
}

type ProjectBaseStore interface {
	SaveProjectBase(ctx context.Context, base ProjectBase) error
	GetProjectBaseByContract(ctx context.Context, contract common.Address) (*ProjectBase, error)
	ListProjectBases(ctx context.Context) ([]ProjectBase, error)
	ListProjectBasesPage(ctx context.Context, page int32, pageSize int32) ([]ProjectBase, int64, int32, int32, error)
	ListProjectBasesByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectBase, error)
	GetMaxProjectBlockNumber(ctx context.Context) (uint64, bool, error)
}

type ProjectChainStateStore interface {
	UpsertProjectChainState(ctx context.Context, item ProjectChainState) error
	GetProjectChainState(ctx context.Context, contract common.Address) (*ProjectChainState, error)
	ListProjectChainStatesByContracts(ctx context.Context, contracts []common.Address) (map[common.Address]ProjectChainState, error)
	ListProjectChainStatesByPairAddresses(ctx context.Context, pairs []common.Address) ([]ProjectChainState, error)
}

type ProjectSimulationStore interface {
	UpsertProjectSimulationResult(ctx context.Context, item ProjectSimulationResult) error
	GetProjectSimulationResult(ctx context.Context, contract common.Address) (*ProjectSimulationResult, error)
}

type ProjectReportStore interface {
	UpsertProjectReportState(ctx context.Context, item ProjectReportState) error
	GetProjectReportState(ctx context.Context, contract common.Address) (*ProjectReportState, error)
	ListProjectReportStatesByContracts(ctx context.Context, contracts []common.Address) (map[common.Address]ProjectReportState, error)
}

type ProjectBytecodeFactStore interface {
	UpsertProjectBytecodeFact(ctx context.Context, item ProjectBytecodeFact) error
	GetProjectBytecodeFact(ctx context.Context, contract common.Address) (*ProjectBytecodeFact, error)
}

type ProjectComponentStateStore interface {
	UpsertProjectComponentState(ctx context.Context, item ProjectComponentState) error
	GetProjectComponentState(ctx context.Context, contract common.Address, component string) (*ProjectComponentState, error)
}

type ProjectAveDetailStore interface {
	UpsertProjectAveDetail(ctx context.Context, contract common.Address, detail ProjectAveDetail) error
	GetProjectAveDetail(ctx context.Context, contract common.Address) (*ProjectAveDetail, error)
	ListProjectAveDetailsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address]ProjectAveDetail, error)
}

type ProjectAveRefreshStore interface {
	ListProjectAveRefreshCandidates(ctx context.Context, staleBefore time.Time, now time.Time, limit int32) ([]common.Address, error)
	ScheduleProjectAveRefresh(ctx context.Context, contract common.Address, nextRunAt time.Time) error
	MarkProjectAveRefreshRunning(ctx context.Context, contract common.Address, at time.Time) error
	MarkProjectAveRefreshSuccess(ctx context.Context, contract common.Address, successAt time.Time, nextRunAt time.Time) error
	MarkProjectAveRefreshFailed(ctx context.Context, contract common.Address, attemptAt time.Time, nextRunAt time.Time, lastError string) error
	GetProjectAveComponentState(ctx context.Context, contract common.Address) (*ProjectComponentState, error)
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

type BytecodeStore interface {
	UpsertBytecode(ctx context.Context, codeHash common.Hash, runtimeBytecode []byte) error
	UpsertContractBytecodeDeployment(ctx context.Context, item ContractBytecodeDeployment) error
	GetBytecode(ctx context.Context, codeHash common.Hash) (*Bytecode, error)
	UpdateBytecodeSourceCode(ctx context.Context, codeHash common.Hash, sourceCode string, sourceCodeHash common.Hash, origin string) error
	UpdateBytecodeSourceQualityReport(ctx context.Context, codeHash common.Hash, report string, origin string, promptVersion int64) error
	ListBytecodes(ctx context.Context, codeHash *common.Hash, limit, offset int64) ([]BytecodeListRecord, int64, error)
	GetBytecodeDetail(ctx context.Context, codeHash common.Hash) (*BytecodeDetailRecord, error)
	ListBytecodeDeployments(ctx context.Context, codeHash common.Hash, chainID int64, contract *common.Address, limit, offset int64) ([]BytecodeDeploymentRecord, int64, error)
	IsBytecodeBlacklisted(ctx context.Context, codeHash common.Hash) (bool, error)
	ListBytecodeBlacklistEntries(ctx context.Context) ([]BytecodeBlacklistEntry, error)
	AddBytecodeBlacklistEntry(ctx context.Context, item BytecodeBlacklistEntry) error
	UpdateBytecodeBlacklistNote(ctx context.Context, codeHash common.Hash, note string) error
	DeleteBytecodeBlacklist(ctx context.Context, codeHash common.Hash) error
	GetBytecodeBlacklistEntry(ctx context.Context, codeHash common.Hash) (*BytecodeBlacklistEntry, error)
}

type WalletBlacklistStore interface {
	ListWalletBlacklistEntries(ctx context.Context) ([]WalletBlacklistEntry, error)
	AddWalletBlacklistEntry(ctx context.Context, item WalletBlacklistEntry) error
	UpdateWalletBlacklistEntryNote(ctx context.Context, wallet common.Address, note string) error
	DeleteWalletBlacklistEntry(ctx context.Context, wallet common.Address) error
	GetWalletBlacklistEntry(ctx context.Context, wallet common.Address) (*WalletBlacklistEntry, error)
}

type SourceQualityPromptStore interface {
	EnsureDefaultSourceQualityPrompt(ctx context.Context, name, systemPrompt string) (*SourceQualityPrompt, error)
	GetActiveSourceQualityPrompt(ctx context.Context) (*SourceQualityPrompt, error)
	ListSourceQualityPrompts(ctx context.Context) ([]SourceQualityPrompt, error)
	GetSourceQualityPrompt(ctx context.Context, id int64) (*SourceQualityPrompt, error)
	CreateSourceQualityPrompt(ctx context.Context, name, systemPrompt string) (*SourceQualityPrompt, error)
	UpdateSourceQualityPrompt(ctx context.Context, id int64, name, systemPrompt string) (*SourceQualityPrompt, error)
	ActivateSourceQualityPrompt(ctx context.Context, id int64) (*SourceQualityPrompt, error)
	DeleteSourceQualityPrompt(ctx context.Context, id int64) error
}

type Store interface {
	ProjectStore
	ProjectBaseStore
	ProjectChainStateStore
	ProjectSimulationStore
	ProjectReportStore
	ProjectBytecodeFactStore
	ProjectComponentStateStore
	ProjectAveDetailStore
	ProjectAveRefreshStore
	ProjectGenesisWalletStore
	ProjectCreatorHistoricalProjectStore
	BytecodeStore
	WalletBlacklistStore
	SourceQualityPromptStore
}
