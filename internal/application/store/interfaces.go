package store

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/model"
	utilave "github.com/useryege/athena/util/ave"
)

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
	SaveProject(ctx context.Context, project ProjectRecord) error
	GetMaxProjectBlockNumber(ctx context.Context, chainID int64) (uint64, bool, error)
	GetProjectByContract(ctx context.Context, chainID int64, contract common.Address) (*ProjectRecord, error)
	ListProjects(ctx context.Context, chainID int64) ([]ProjectRecord, error)
	ListProjectsPage(ctx context.Context, chainID int64, page int32, pageSize int32) ([]ProjectRecord, int64, int32, int32, error)
	ListProjectsByCreatorBefore(ctx context.Context, chainID int64, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectRecord, error)
	ListProjectMetas(ctx context.Context, chainID int64) ([]model.Project, error)
	ListProjectMetasByPairAddresses(ctx context.Context, chainID int64, pairs []common.Address) ([]model.Project, error)
	UpdateProjectCreatorResult(ctx context.Context, chainID int64, contract common.Address, result model.SimulateResult) error
	ListProjectMetasByCreator(ctx context.Context, chainID int64, creator common.Address) ([]model.Project, error)
	ListProjectMetasByCreatorBefore(ctx context.Context, chainID int64, creator common.Address, blockNumber uint64, txIndex uint64) ([]model.Project, error)
	GetProjectMetaByContract(ctx context.Context, chainID int64, contract common.Address) (*model.Project, error)
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

type ProjectComponentStateStore interface {
	UpsertProjectComponentState(ctx context.Context, item ProjectComponentState) error
	GetProjectComponentState(ctx context.Context, chainID int64, contract common.Address, component string) (*ProjectComponentState, error)
}

type ProjectAveDetailStore interface {
	UpsertProjectAveDetail(ctx context.Context, chainID int64, contract common.Address, response *utilave.TokenDetailResponse, fetchedAt time.Time) error
	GetProjectAveDetail(ctx context.Context, chainID int64, contract common.Address) (*model.ProjectAveDetail, error)
	ListProjectAveDetailsByContracts(ctx context.Context, chainID int64, contracts []common.Address) (map[common.Address]model.ProjectAveDetail, error)
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
	UpsertBytecode(ctx context.Context, codeHash common.Hash) error
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

// Store is the aggregate interface for all store operations.
type Store interface {
	ChainStore
	ProjectCollectionStateStore
	OutboxStore
	ProjectIntakeStore
	ProjectStore
	ProjectChainStateStore
	ProjectSimulationStore
	ProjectComponentStateStore
	ProjectAveDetailStore
	ProjectAveRefreshStore
	ProjectGenesisWalletStore
	ProjectCreatorHistoricalProjectStore
	BytecodeStore
	WalletBlacklistStore
}
