package tokenapi

import (
	"context"
	"github.com/useryege/athena/internal/token/catalog"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/policy"
	"github.com/useryege/athena/internal/token/projectview"
	"github.com/useryege/athena/internal/token/reporting"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/selection"
	"github.com/useryege/athena/internal/token/shared"
	"github.com/useryege/athena/internal/token/swap"
)

type CatalogApplication interface {
	GetContractCode(context.Context, shared.Hash) (*catalog.ContractCode, error)
	ListContractCodes(context.Context, shared.Hash, int32, int32) (*catalog.ContractCodePage, error)
	ListContractCodesByDeploymentCount(context.Context, int32, int32) (*catalog.ContractCodePage, error)
	ListProjectsPage(context.Context, int64, shared.Hash, shared.Address, int32, int32) (*catalog.ProjectPage, error)
}

type ResearchApplication interface {
	GetProjectDataCollectionTask(context.Context, int64) (*research.ProjectDataCollectionTask, error)
	ListProjectDataCollectionTasks(context.Context, int64, string, string, int32, int32) (*research.CollectionTaskPage, error)
	ListProjectResearchStatesPage(context.Context, int64, int64, string, int32, int32) (*research.ResearchStatePage, error)
}

type ReportingApplication interface {
	ListProjectReportsPage(context.Context, int64, int64, shared.Address, string, int32, int32) (*reporting.ProjectReportPage, error)
	ListProjectReportRevisionsPage(context.Context, int64, int64, int32, int32) (*reporting.ReportRevisionPage, error)
}

type SelectionApplication interface {
	ListProjectSelectionsPage(context.Context, int64, int64, string, int32, int32) (*selection.Page, error)
}

type ProjectViewApplication interface {
	GetProjectDetail(context.Context, int64) (*projectview.Detail, error)
	GetProjectSwapActivity(context.Context, int64) (*projectview.SwapActivity, error)
	ListProjectSwapEventsPage(context.Context, int64, swap.PairKind, uint64, int32, int32) (*projectview.SwapEventPage, error)
	ListProjectObservationsPage(context.Context, int64, string, int32, int32) (*projectview.ObservationPage, error)
	ListProjectTrends(context.Context, int64, string) (*projectview.TrendResult, error)
	ListProjectWalletNormalTransactionsPage(context.Context, int64, shared.Address, string, string, int32, int32) (*projectview.WalletNormalTransactionPage, error)
}

type PolicyApplication interface {
	GetContractCodeBlocklistEntry(context.Context, shared.Hash) (*policy.ContractCodeBlocklistEntry, error)
	ListContractCodeBlocklistEntries(context.Context) ([]policy.ContractCodeBlocklistEntry, error)
	CreateContractCodeBlocklistEntry(context.Context, int64, shared.Address, string) error
	UpdateContractCodeBlocklistEntryNote(context.Context, shared.Hash, string) (int64, error)
	DeleteContractCodeBlocklistEntry(context.Context, shared.Hash) (int64, error)
	GetWalletBlocklistEntry(context.Context, shared.Address) (*policy.WalletBlocklistEntry, error)
	ListWalletBlocklistEntries(context.Context) ([]policy.WalletBlocklistEntry, error)
	CreateWalletBlocklistEntry(context.Context, policy.WalletBlocklistEntry) error
	UpdateWalletBlocklistEntryNote(context.Context, shared.Address, string) (int64, error)
	DeleteWalletBlocklistEntry(context.Context, shared.Address) (int64, error)
}

type OperationsApplication interface {
	ListChains(context.Context) ([]discovery.Chain, error)
	GetChainProcessingCheckpoint(context.Context, int64) (*discovery.ChainProcessingCheckpoint, error)
	ListChainProcessingCheckpoints(context.Context) ([]discovery.ChainProcessingCheckpoint, error)
	UpdateChainProcessingCheckpointStatus(context.Context, int64, discovery.ChainProcessingStatus) (*discovery.ChainProcessingCheckpoint, error)
	ListNodeStatuses(context.Context) ([]discovery.NodeStatus, error)
}

type Applications struct {
	Catalog     CatalogApplication
	Research    ResearchApplication
	Reporting   ReportingApplication
	Selection   SelectionApplication
	ProjectView ProjectViewApplication
	Policy      PolicyApplication
	Operations  OperationsApplication
}
