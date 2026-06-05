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

type ProjectBase struct {
	ChainID     int64
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
	ChainID         int64
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
	ChainID         int64
	ProjectContract common.Address
	Result          SimulateResult
	FetchedAt       time.Time
	UpdatedAt       time.Time
}

type ProjectBytecodeFact struct {
	ChainID               int64
	ProjectContract       common.Address
	CodeHash              common.Hash
	IsBytecodeBlacklisted bool
	FetchedAt             time.Time
	UpdatedAt             time.Time
}

type ProjectComponentState struct {
	ChainID         int64
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

	ProjectComponentStatusPending = "pending"
	ProjectComponentStatusRunning = "running"
	ProjectComponentStatusSuccess = "success"
	ProjectComponentStatusFailed  = "failed"
)

const (
	ProjectCollectionStatusRequested    = "requested"
	ProjectCollectionStatusRunning      = "running"
	ProjectCollectionStatusCompleted    = "completed"
	ProjectCollectionStatusFailed       = "failed"
	ProjectCollectionStatusNotRequested = "not_requested"
)

type ProjectGenesisWallet struct {
	ID                int64
	ChainID           int64
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
	ChainID                   int64
	ProjectContract           common.Address
	HistoricalProjectContract common.Address
	RankIndex                 int32
	CreatedAt                 time.Time
}

type ChainInfo struct {
	ID      int64
	Name    string
	Enabled bool
}

type ChainIngestCheckpoint struct {
	ChainID              int64
	ChainName            string
	Enabled              bool
	FinalizedBlockNumber uint64
	FinalizedBlockHash   common.Hash
	CursorBlockNumber    uint64
	CursorBlockHash      common.Hash
	Status               string
	LockedAt             time.Time
	LockedBy             string
	UpdatedAt            time.Time
}

type ProjectCollectionState struct {
	ChainID         int64
	ProjectContract common.Address
	Status          string
	WorkflowID      string
	LastRequestedAt time.Time
	LastStartedAt   time.Time
	LastCompletedAt time.Time
	NextRunAt       time.Time
	LastError       string
	UpdatedAt       time.Time
	ComponentStates []ProjectComponentState
}

type ChainStore interface {
	GetChainIngestCheckpoint(ctx context.Context, chainID int64) (*ChainIngestCheckpoint, error)
	ListChainIngestCheckpoints(ctx context.Context) ([]ChainIngestCheckpoint, error)
	UpsertChainIngestCheckpoint(ctx context.Context, item ChainIngestCheckpoint) (*ChainIngestCheckpoint, error)
}

type ProjectCollectionStateStore interface {
	GetProjectCollectionState(ctx context.Context, chainID int64, contract common.Address) (*ProjectCollectionState, error)
	ListProjectComponentStates(ctx context.Context, chainID int64, contract common.Address) ([]ProjectComponentState, error)
	MarkProjectCollectionRunning(ctx context.Context, chainID int64, contract common.Address, workflowID string, startedAt time.Time) error
	MarkProjectCollectionCompleted(ctx context.Context, chainID int64, contract common.Address, completedAt time.Time) error
	MarkProjectCollectionFailed(ctx context.Context, chainID int64, contract common.Address, nextRunAt time.Time, lastError string) error
}

type ProjectIntakeStore interface {
	UpsertProjectCandidateAndEnqueueQualification(ctx context.Context, candidate model.DiscoveredProjectCandidate) error
	EnqueueProjectCollection(ctx context.Context, ref model.ProjectRef, reason string) error
}

type OutboxStore interface {
	InsertOutboxEvent(ctx context.Context, item CreateOutboxEventParams) (*OutboxEvent, error)
	ClaimOutboxEvents(ctx context.Context, lockedBy string, now time.Time, limit int32) ([]OutboxEvent, error)
	ClaimOutboxEventsByTypes(ctx context.Context, lockedBy string, now time.Time, limit int32, types []string) ([]OutboxEvent, error)
	MarkOutboxEventProcessed(ctx context.Context, id int64) error
	MarkOutboxEventFailed(ctx context.Context, id int64, nextAttemptAt time.Time, lastError string) error
	MarkOutboxEventDiscarded(ctx context.Context, id int64, lastError string) error
}

type ProjectStore interface {
	SaveProjectMeta(ctx context.Context, meta ProjectMeta) error
	GetMaxProjectBlockNumber(ctx context.Context, chainID int64) (uint64, bool, error)
	ListProjectMetas(ctx context.Context, chainID int64) ([]ProjectMeta, error)
	ListAllProjectMetas(ctx context.Context, chainID int64) ([]ProjectMeta, error)
	ListProjectMetasByPairAddresses(ctx context.Context, chainID int64, pairs []common.Address) ([]ProjectMeta, error)
	UpsertProjectAveDetail(ctx context.Context, chainID int64, contract common.Address, detail ProjectAveDetail) error
	UpdateProjectCreatorResult(ctx context.Context, chainID int64, contract common.Address, result SimulateResult) error
	ListProjectMetasByCreator(ctx context.Context, chainID int64, creator common.Address) ([]ProjectMeta, error)
	ListProjectMetasByCreatorBefore(ctx context.Context, chainID int64, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectMeta, error)
	GetProjectMetaByContract(ctx context.Context, chainID int64, contract common.Address) (*ProjectMeta, error)
}

type ProjectBaseStore interface {
	SaveProjectBase(ctx context.Context, base ProjectBase) error
	GetProjectBaseByContract(ctx context.Context, chainID int64, contract common.Address) (*ProjectBase, error)
	ListProjectBases(ctx context.Context, chainID int64) ([]ProjectBase, error)
	ListProjectBasesPage(ctx context.Context, chainID int64, page int32, pageSize int32) ([]ProjectBase, int64, int32, int32, error)
	ListProjectBasesByCreatorBefore(ctx context.Context, chainID int64, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectBase, error)
	GetMaxProjectBlockNumber(ctx context.Context, chainID int64) (uint64, bool, error)
}

type ProjectChainStateStore interface {
	UpsertProjectChainState(ctx context.Context, item ProjectChainState) error
	GetProjectChainState(ctx context.Context, chainID int64, contract common.Address) (*ProjectChainState, error)
	ListProjectChainStatesByContracts(ctx context.Context, chainID int64, contracts []common.Address) (map[common.Address]ProjectChainState, error)
	ListProjectChainStatesByPairAddresses(ctx context.Context, chainID int64, pairs []common.Address) ([]ProjectChainState, error)
}

type ProjectSimulationStore interface {
	UpsertProjectSimulationResult(ctx context.Context, item ProjectSimulationResult) error
	GetProjectSimulationResult(ctx context.Context, chainID int64, contract common.Address) (*ProjectSimulationResult, error)
}

type ProjectBytecodeFactStore interface {
	UpsertProjectBytecodeFact(ctx context.Context, item ProjectBytecodeFact) error
	GetProjectBytecodeFact(ctx context.Context, chainID int64, contract common.Address) (*ProjectBytecodeFact, error)
}

type ProjectComponentStateStore interface {
	UpsertProjectComponentState(ctx context.Context, item ProjectComponentState) error
	GetProjectComponentState(ctx context.Context, chainID int64, contract common.Address, component string) (*ProjectComponentState, error)
}

type ProjectAveDetailStore interface {
	UpsertProjectAveDetail(ctx context.Context, chainID int64, contract common.Address, detail ProjectAveDetail) error
	GetProjectAveDetail(ctx context.Context, chainID int64, contract common.Address) (*ProjectAveDetail, error)
	ListProjectAveDetailsByContracts(ctx context.Context, chainID int64, contracts []common.Address) (map[common.Address]ProjectAveDetail, error)
}

type ProjectAveRefreshStore interface {
	ListProjectAveRefreshCandidates(ctx context.Context, chainID int64, staleBefore time.Time, now time.Time, limit int32) ([]common.Address, error)
	ScheduleProjectAveRefresh(ctx context.Context, chainID int64, contract common.Address, nextRunAt time.Time) error
	MarkProjectAveRefreshRunning(ctx context.Context, chainID int64, contract common.Address, at time.Time) error
	MarkProjectAveRefreshSuccess(ctx context.Context, chainID int64, contract common.Address, successAt time.Time, nextRunAt time.Time) error
	MarkProjectAveRefreshFailed(ctx context.Context, chainID int64, contract common.Address, attemptAt time.Time, nextRunAt time.Time, lastError string) error
	GetProjectAveComponentState(ctx context.Context, chainID int64, contract common.Address) (*ProjectComponentState, error)
}

type ProjectGenesisWalletStore interface {
	ReplaceProjectGenesisWallets(ctx context.Context, chainID int64, contract common.Address, items []ProjectGenesisWallet) error
	ListProjectGenesisWalletsByContract(ctx context.Context, chainID int64, contract common.Address) ([]ProjectGenesisWallet, error)
	ListProjectGenesisWalletsByContracts(ctx context.Context, chainID int64, contracts []common.Address) (map[common.Address][]ProjectGenesisWallet, error)
	ListProjectGenesisWalletsByWallet(ctx context.Context, chainID int64, wallet common.Address) ([]ProjectGenesisWallet, error)
}

type ProjectCreatorHistoricalProjectStore interface {
	ReplaceProjectCreatorHistoricalProjects(ctx context.Context, chainID int64, contract common.Address, items []ProjectCreatorHistoricalProject) error
	ListProjectCreatorHistoricalProjectsByContract(ctx context.Context, chainID int64, contract common.Address) ([]ProjectCreatorHistoricalProject, error)
	ListProjectCreatorHistoricalProjectsByContracts(ctx context.Context, chainID int64, contracts []common.Address) (map[common.Address][]ProjectCreatorHistoricalProject, error)
}

type BytecodeStore interface {
	UpsertBytecode(ctx context.Context, codeHash common.Hash, runtimeBytecode []byte) error
	UpsertContractBytecodeDeployment(ctx context.Context, item ContractBytecodeDeployment) error
	GetBytecode(ctx context.Context, codeHash common.Hash) (*Bytecode, error)
	UpdateBytecodeSourceCode(ctx context.Context, codeHash common.Hash, sourceCode string, sourceCodeHash common.Hash, origin string) error
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

type Store interface {
	ChainStore
	ProjectCollectionStateStore
	OutboxStore
	ProjectIntakeStore
	ProjectStore
	ProjectBaseStore
	ProjectChainStateStore
	ProjectSimulationStore
	ProjectBytecodeFactStore
	ProjectComponentStateStore
	ProjectAveDetailStore
	ProjectAveRefreshStore
	ProjectGenesisWalletStore
	ProjectCreatorHistoricalProjectStore
	BytecodeStore
	WalletBlacklistStore
}
