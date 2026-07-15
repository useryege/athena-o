package projectdatacollector

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/domain"
)

type Store interface {
	ListDueProjectDataCollectionTasks(context.Context, domain.DataCollectionType, []int64, int32) ([]domain.ProjectDataCollectionTaskWithProject, error)
	MarkProjectDataCollectionTaskFailed(context.Context, domain.ProjectDataCollectionTask, string) (*domain.ProjectDataCollectionTask, bool, error)
	CompleteProjectAveDataCollection(context.Context, domain.ProjectDataCollectionTask, domain.AveObservationV1, time.Time) (*domain.ProjectObservation, error)
	CompleteProjectChainStateCollection(context.Context, domain.ProjectDataCollectionTask, domain.ChainStateObservationV1, uint64, time.Time) (*domain.ProjectObservation, error)
	CompleteProjectWalletAssetStateCollection(context.Context, domain.ProjectDataCollectionTask, domain.WalletAssetObservationV1, uint64, time.Time) error
	CompleteProjectSimulationResultCollection(context.Context, domain.ProjectDataCollectionTask, domain.SimulationObservationV1, uint64, time.Time) error
	CompleteProjectContractCodeSourceCollection(context.Context, domain.ProjectDataCollectionTask, common.Hash, string, time.Time) error
	GetContractCode(context.Context, common.Hash) (*domain.ContractCode, error)
	ListProjectRelatedWalletsByProject(context.Context, int64) ([]domain.ProjectRelatedWallet, error)
}
