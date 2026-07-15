package tokenapi

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/catalog"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/policy"
	"github.com/useryege/athena/internal/token/reporting"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/selection"
)

type CatalogApplication interface {
	GetContractCode(context.Context, common.Hash) (*catalog.ContractCode, error)
	ListContractCodes(context.Context, common.Hash, int32, int32) (*catalog.ContractCodePage, error)
	ListContractCodesByDeploymentCount(context.Context, int32, int32) (*catalog.ContractCodePage, error)
	ListProjectsPage(context.Context, int64, common.Hash, common.Address, int32, int32) (*catalog.ProjectPage, error)
}

type ResearchApplication interface {
	GetProjectDataCollectionTask(context.Context, int64) (*research.ProjectDataCollectionTask, error)
	ListProjectDataCollectionTasks(context.Context, int64, string, string, int32, int32) (*research.CollectionTaskPage, error)
	ListProjectResearchStatesPage(context.Context, int64, int64, string, int32, int32) (*research.ResearchStatePage, error)
	ListProjectReportsPage(context.Context, int64, int64, common.Address, string, int32, int32) (*reporting.ProjectReportPage, error)
	ListProjectReportRevisionsPage(context.Context, int64, int64, int32, int32) (*reporting.ReportRevisionPage, error)
	ListProjectSelectionsPage(context.Context, int64, int64, string, int32, int32) (*selection.Page, error)
}

type PolicyApplication interface {
	GetContractCodeBlocklistEntry(context.Context, common.Hash) (*policy.ContractCodeBlocklistEntry, error)
	ListContractCodeBlocklistEntries(context.Context) ([]policy.ContractCodeBlocklistEntry, error)
	CreateContractCodeBlocklistEntry(context.Context, int64, common.Address, string) error
	UpdateContractCodeBlocklistEntryNote(context.Context, common.Hash, string) (int64, error)
	DeleteContractCodeBlocklistEntry(context.Context, common.Hash) (int64, error)
	GetWalletBlocklistEntry(context.Context, common.Address) (*policy.WalletBlocklistEntry, error)
	ListWalletBlocklistEntries(context.Context) ([]policy.WalletBlocklistEntry, error)
	CreateWalletBlocklistEntry(context.Context, policy.WalletBlocklistEntry) error
	UpdateWalletBlocklistEntryNote(context.Context, common.Address, string) (int64, error)
	DeleteWalletBlocklistEntry(context.Context, common.Address) (int64, error)
}

type OperationsApplication interface {
	ListChains(context.Context) ([]discovery.Chain, error)
	GetChainIngestCheckpoint(context.Context, int64) (*discovery.ChainIngestCheckpoint, error)
	ListChainIngestCheckpoints(context.Context) ([]discovery.ChainIngestCheckpoint, error)
	UpdateChainIngestCheckpointStatus(context.Context, int64, discovery.ChainIngestStatus) (*discovery.ChainIngestCheckpoint, error)
	ListNodeStatuses(context.Context) ([]discovery.NodeStatus, error)
}

type Applications struct {
	Catalog    CatalogApplication
	Research   ResearchApplication
	Policy     PolicyApplication
	Operations OperationsApplication
}
