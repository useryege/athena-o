package research

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/catalog"
)

type CollectorRepository interface {
	ListDueProjectDataCollectionTasks(context.Context, DataCollectionType, []int64, int32) ([]ProjectDataCollectionTaskWithProject, error)
	MarkProjectDataCollectionTaskFailed(context.Context, ProjectDataCollectionTask, string) (*ProjectDataCollectionTask, bool, error)
	CompleteProjectAveDataCollection(context.Context, ProjectDataCollectionTask, AveObservationV1, time.Time) (*ProjectObservation, error)
	CompleteProjectChainStateCollection(context.Context, ProjectDataCollectionTask, ChainStateObservationV1, uint64, time.Time) (*ProjectObservation, error)
	CompleteProjectWalletAssetStateCollection(context.Context, ProjectDataCollectionTask, WalletAssetObservationV1, uint64, time.Time) error
	CompleteProjectSimulationResultCollection(context.Context, ProjectDataCollectionTask, SimulationObservationV1, uint64, time.Time) error
	CompleteProjectContractCodeSourceCollection(context.Context, ProjectDataCollectionTask, common.Hash, string, time.Time) error
	GetContractCode(context.Context, common.Hash) (*catalog.ContractCode, error)
	ListProjectRelatedWalletsByProject(context.Context, int64) ([]catalog.ProjectRelatedWallet, error)
}

type MarketDataProvider interface {
	GetMarketData(context.Context, int64, common.Address) (AveObservationV1, error)
}

type SourceCodeProvider interface {
	GetSourceCode(context.Context, int64, common.Address) (string, error)
}
