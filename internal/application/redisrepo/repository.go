package redisrepo

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appstore "github.com/useryege/athena/internal/application/store"
)

type ProjectCacheRepository interface {
	SetBase(ctx context.Context, item appstore.ProjectBase) error
	GetBase(ctx context.Context, contract common.Address) (*appstore.ProjectBase, bool, error)
	ListBasePage(ctx context.Context, page int32, pageSize int32) ([]appstore.ProjectBase, int64, int32, int32, error)
	ListBasesByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]appstore.ProjectBase, error)
	GetMaxBaseBlockNumber(ctx context.Context) (uint64, bool, error)

	SetChainState(ctx context.Context, item appstore.ProjectChainState) error
	GetChainState(ctx context.Context, contract common.Address) (*appstore.ProjectChainState, bool, error)
	ListChainStatesByPairAddresses(ctx context.Context, pairs []common.Address) ([]appstore.ProjectChainState, error)

	SetSimulation(ctx context.Context, item appstore.ProjectSimulationResult) error
	GetSimulation(ctx context.Context, contract common.Address) (*appstore.ProjectSimulationResult, bool, error)

	SetReport(ctx context.Context, item appstore.ProjectReportState) error
	GetReport(ctx context.Context, contract common.Address) (*appstore.ProjectReportState, bool, error)

	SetBytecodeFact(ctx context.Context, item appstore.ProjectBytecodeFact) error
	GetBytecodeFact(ctx context.Context, contract common.Address) (*appstore.ProjectBytecodeFact, bool, error)

	SetAveDetail(ctx context.Context, contract common.Address, item appstore.ProjectAveDetail) error
	GetAveDetail(ctx context.Context, contract common.Address) (*appstore.ProjectAveDetail, bool, error)
	SetGenesisWallets(ctx context.Context, contract common.Address, items []appstore.ProjectGenesisWallet) error
	GetGenesisWallets(ctx context.Context, contract common.Address) ([]appstore.ProjectGenesisWallet, bool, error)
	SetCreatorHistory(ctx context.Context, contract common.Address, items []appstore.ProjectCreatorHistoricalProject) error
	GetCreatorHistory(ctx context.Context, contract common.Address) ([]appstore.ProjectCreatorHistoricalProject, bool, error)

	SetComponentState(ctx context.Context, item appstore.ProjectComponentState) error
	GetComponentState(ctx context.Context, contract common.Address, component string) (*appstore.ProjectComponentState, bool, error)
	ListComponentStatesByNextRun(ctx context.Context, component string, now time.Time, limit int32) ([]appstore.ProjectComponentState, error)
}

type ComponentStreamRepository interface {
	Publish(ctx context.Context, payload string) error
	ReadGroup(ctx context.Context, group string, consumer string, stream string, count int64, block time.Duration) ([]map[string]any, error)
}
