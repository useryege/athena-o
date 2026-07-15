package postgres

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/catalog"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/policy"
	"github.com/useryege/athena/internal/token/reporting"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/selection"
)

type DiscoveryRepository struct{ database *Database }
type ResearchRepository struct{ database *Database }
type ReportingRepository struct{ database *Database }
type SelectionRepository struct{ database *Database }
type CatalogReadRepository struct{ database *Database }
type ResearchReadRepository struct{ database *Database }
type PolicyRepository struct{ database *Database }
type OperationsRepository struct{ database *Database }

func (d *Database) Discovery() *DiscoveryRepository { return &DiscoveryRepository{database: d} }
func (d *Database) Research() *ResearchRepository   { return &ResearchRepository{database: d} }
func (d *Database) Reporting() *ReportingRepository { return &ReportingRepository{database: d} }
func (d *Database) Selection() *SelectionRepository { return &SelectionRepository{database: d} }
func (d *Database) CatalogReadModel() *CatalogReadRepository {
	return &CatalogReadRepository{database: d}
}
func (d *Database) ResearchReadModel() *ResearchReadRepository {
	return &ResearchReadRepository{database: d}
}
func (d *Database) Policy() *PolicyRepository         { return &PolicyRepository{database: d} }
func (d *Database) Operations() *OperationsRepository { return &OperationsRepository{database: d} }

func (r *DiscoveryRepository) GetChainIngestCheckpoint(ctx context.Context, chainID int64) (*discovery.ChainIngestCheckpoint, error) {
	return r.database.GetChainIngestCheckpoint(ctx, chainID)
}
func (r *DiscoveryRepository) UpsertChainIngestCheckpoint(ctx context.Context, item discovery.ChainIngestCheckpoint) (*discovery.ChainIngestCheckpoint, error) {
	return r.database.UpsertChainIngestCheckpoint(ctx, item)
}
func (r *DiscoveryRepository) UpdateChainIngestCheckpointStatus(ctx context.Context, chainID int64, status discovery.ChainIngestStatus) (*discovery.ChainIngestCheckpoint, error) {
	return r.database.UpdateChainIngestCheckpointStatus(ctx, chainID, status)
}
func (r *DiscoveryRepository) IngestProjectCandidateBatch(ctx context.Context, checkpoint discovery.ChainIngestCheckpoint, items []discovery.ProjectCandidate) (*discovery.ChainIngestCheckpoint, error) {
	return r.database.IngestProjectCandidateBatch(ctx, checkpoint, items)
}
func (r *DiscoveryRepository) ClaimProjectCandidateValidations(ctx context.Context, chainID int64, token string, lease time.Duration, limit int32) ([]discovery.ProjectCandidate, error) {
	return r.database.ClaimProjectCandidateValidations(ctx, chainID, token, lease, limit)
}
func (r *DiscoveryRepository) RenewProjectCandidateValidationClaims(ctx context.Context, token string, lease time.Duration) error {
	return r.database.RenewProjectCandidateValidationClaims(ctx, token, lease)
}
func (r *DiscoveryRepository) ReleaseProjectCandidateValidationClaims(ctx context.Context, token string) error {
	return r.database.ReleaseProjectCandidateValidationClaims(ctx, token)
}
func (r *DiscoveryRepository) RejectProjectCandidate(ctx context.Context, item discovery.ProjectCandidate) error {
	return r.database.RejectProjectCandidate(ctx, item)
}
func (r *DiscoveryRepository) ValidateProjectCandidate(ctx context.Context, candidate discovery.ProjectCandidate, codeHash common.Hash, metadata catalog.ProjectTokenMetadata, wethPair, usdtPair common.Address, wallets []catalog.ProjectRelatedWallet, recipients []catalog.ProjectInitialRecipient) (*catalog.Project, error) {
	return r.database.ValidateProjectCandidate(ctx, candidate, codeHash, metadata, wethPair, usdtPair, wallets, recipients)
}

func (r *ResearchRepository) ApplyResearchPolicy(ctx context.Context, intervals map[research.DataCollectionType]time.Duration, ttl time.Duration) error {
	return r.database.ApplyResearchPolicy(ctx, intervals, ttl)
}
func (r *ResearchRepository) MaintainResearchLifecycle(ctx context.Context) error {
	return r.database.MaintainResearchLifecycle(ctx)
}
func (r *ResearchRepository) ScheduleDueProjectDataCollectionTasks(ctx context.Context, limit int32) (int, error) {
	return r.database.ScheduleDueProjectDataCollectionTasks(ctx, limit)
}
func (r *ResearchRepository) ListDueProjectDataCollectionTasks(ctx context.Context, dataType research.DataCollectionType, chainIDs []int64, limit int32) ([]research.ProjectDataCollectionTaskWithProject, error) {
	return r.database.ListDueProjectDataCollectionTasks(ctx, dataType, chainIDs, limit)
}
func (r *ResearchRepository) MarkProjectDataCollectionTaskFailed(ctx context.Context, task research.ProjectDataCollectionTask, message string) (*research.ProjectDataCollectionTask, bool, error) {
	return r.database.MarkProjectDataCollectionTaskFailed(ctx, task, message)
}
func (r *ResearchRepository) CompleteProjectAveDataCollection(ctx context.Context, task research.ProjectDataCollectionTask, value research.AveObservationV1, checkedAt time.Time) (*research.ProjectObservation, error) {
	return r.database.CompleteProjectAveDataCollection(ctx, task, value, checkedAt)
}
func (r *ResearchRepository) CompleteProjectChainStateCollection(ctx context.Context, task research.ProjectDataCollectionTask, value research.ChainStateObservationV1, block uint64, checkedAt time.Time) (*research.ProjectObservation, error) {
	return r.database.CompleteProjectChainStateCollection(ctx, task, value, block, checkedAt)
}
func (r *ResearchRepository) CompleteProjectWalletAssetStateCollection(ctx context.Context, task research.ProjectDataCollectionTask, value research.WalletAssetObservationV1, block uint64, checkedAt time.Time) error {
	return r.database.CompleteProjectWalletAssetStateCollection(ctx, task, value, block, checkedAt)
}
func (r *ResearchRepository) CompleteProjectSimulationResultCollection(ctx context.Context, task research.ProjectDataCollectionTask, value research.SimulationObservationV1, block uint64, checkedAt time.Time) error {
	return r.database.CompleteProjectSimulationResultCollection(ctx, task, value, block, checkedAt)
}
func (r *ResearchRepository) CompleteProjectContractCodeSourceCollection(ctx context.Context, task research.ProjectDataCollectionTask, hash common.Hash, source string, checkedAt time.Time) error {
	return r.database.CompleteProjectContractCodeSourceCollection(ctx, task, hash, source, checkedAt)
}
func (r *ResearchRepository) GetContractCode(ctx context.Context, hash common.Hash) (*catalog.ContractCode, error) {
	return r.database.GetContractCode(ctx, hash)
}
func (r *ResearchRepository) ListProjectRelatedWalletsByProject(ctx context.Context, projectID int64) ([]catalog.ProjectRelatedWallet, error) {
	return r.database.ListProjectRelatedWalletsByProject(ctx, projectID)
}

func (r *ReportingRepository) ClaimProjectReportBuildTasks(ctx context.Context, limit int32) ([]reporting.ProjectReportBuildTask, error) {
	return r.database.ClaimProjectReportBuildTasks(ctx, limit)
}
func (r *ReportingRepository) ListCurrentProjectObservations(ctx context.Context, projectID int64) ([]research.ProjectObservation, error) {
	return r.database.ListCurrentProjectObservations(ctx, projectID)
}
func (r *ReportingRepository) CompleteProjectReportBuild(ctx context.Context, task reporting.ProjectReportBuildTask, report reporting.ProjectReportRevision) (*reporting.ProjectReportRevision, bool, error) {
	return r.database.CompleteProjectReportBuild(ctx, task, report)
}
func (r *ReportingRepository) MarkProjectReportBuildTaskFailed(ctx context.Context, task reporting.ProjectReportBuildTask, message string) error {
	return r.database.MarkProjectReportBuildTaskFailed(ctx, task, message)
}

func (r *SelectionRepository) ClaimProjectSelectionEvaluationTasks(ctx context.Context, limit int32) ([]selection.ProjectSelectionEvaluationTask, error) {
	return r.database.ClaimProjectSelectionEvaluationTasks(ctx, limit)
}
func (r *SelectionRepository) GetProjectReportRevision(ctx context.Context, projectID, revision int64) (*reporting.ProjectReportRevision, error) {
	return r.database.GetProjectReportRevision(ctx, projectID, revision)
}
func (r *SelectionRepository) CompleteProjectSelectionEvaluation(ctx context.Context, task selection.ProjectSelectionEvaluationTask, item selection.ProjectSelection) (*selection.ProjectSelection, bool, error) {
	return r.database.CompleteProjectSelectionEvaluation(ctx, task, item)
}
func (r *SelectionRepository) MarkProjectSelectionEvaluationTaskFailed(ctx context.Context, task selection.ProjectSelectionEvaluationTask, message string) error {
	return r.database.MarkProjectSelectionEvaluationTaskFailed(ctx, task, message)
}

func (r *CatalogReadRepository) GetContractCode(ctx context.Context, hash common.Hash) (*catalog.ContractCode, error) {
	return r.database.GetContractCode(ctx, hash)
}
func (r *CatalogReadRepository) ListContractCodes(ctx context.Context, hash common.Hash, page, size int32) (*catalog.ContractCodePage, error) {
	return r.database.ListContractCodes(ctx, hash, page, size)
}
func (r *CatalogReadRepository) ListContractCodesByDeploymentCount(ctx context.Context, page, size int32) (*catalog.ContractCodePage, error) {
	return r.database.ListContractCodesByDeploymentCount(ctx, page, size)
}
func (r *CatalogReadRepository) ListProjectsPage(ctx context.Context, chainID int64, hash common.Hash, contract common.Address, page, size int32) (*catalog.ProjectPage, error) {
	return r.database.ListProjectsPage(ctx, chainID, hash, contract, page, size)
}

func (r *ResearchReadRepository) GetProjectDataCollectionTask(ctx context.Context, id int64) (*research.ProjectDataCollectionTask, error) {
	return r.database.GetProjectDataCollectionTask(ctx, id)
}
func (r *ResearchReadRepository) ListProjectDataCollectionTasks(ctx context.Context, projectID int64, dataType, status string, page, size int32) (*research.CollectionTaskPage, error) {
	return r.database.ListProjectDataCollectionTasks(ctx, projectID, dataType, status, page, size)
}
func (r *ResearchReadRepository) ListProjectResearchStatesPage(ctx context.Context, chainID, projectID int64, status string, page, size int32) (*research.ResearchStatePage, error) {
	return r.database.ListProjectResearchStatesPage(ctx, chainID, projectID, status, page, size)
}
func (r *ResearchReadRepository) ListProjectReportsPage(ctx context.Context, chainID, projectID int64, contract common.Address, status string, page, size int32) (*reporting.ProjectReportPage, error) {
	return r.database.ListProjectReportsPage(ctx, chainID, projectID, contract, status, page, size)
}
func (r *ResearchReadRepository) ListProjectReportRevisionsPage(ctx context.Context, chainID, projectID int64, page, size int32) (*reporting.ReportRevisionPage, error) {
	return r.database.ListProjectReportRevisionsPage(ctx, chainID, projectID, page, size)
}
func (r *ResearchReadRepository) ListProjectSelectionsPage(ctx context.Context, chainID, projectID int64, outcome string, page, size int32) (*selection.Page, error) {
	return r.database.ListProjectSelectionsPage(ctx, chainID, projectID, outcome, page, size)
}

func (r *PolicyRepository) GetContractCodeBlocklistEntry(ctx context.Context, hash common.Hash) (*policy.ContractCodeBlocklistEntry, error) {
	return r.database.GetContractCodeBlocklistEntry(ctx, hash)
}
func (r *PolicyRepository) ListContractCodeBlocklistEntries(ctx context.Context) ([]policy.ContractCodeBlocklistEntry, error) {
	return r.database.ListContractCodeBlocklistEntries(ctx)
}
func (r *PolicyRepository) CreateContractCodeBlocklistEntry(ctx context.Context, item policy.ContractCodeBlocklistEntry) error {
	return r.database.CreateContractCodeBlocklistEntry(ctx, item)
}
func (r *PolicyRepository) UpdateContractCodeBlocklistEntryNote(ctx context.Context, hash common.Hash, note string) (int64, error) {
	return r.database.UpdateContractCodeBlocklistEntryNote(ctx, hash, note)
}
func (r *PolicyRepository) DeleteContractCodeBlocklistEntry(ctx context.Context, hash common.Hash) (int64, error) {
	return r.database.DeleteContractCodeBlocklistEntry(ctx, hash)
}
func (r *PolicyRepository) GetWalletBlocklistEntry(ctx context.Context, wallet common.Address) (*policy.WalletBlocklistEntry, error) {
	return r.database.GetWalletBlocklistEntry(ctx, wallet)
}
func (r *PolicyRepository) ListWalletBlocklistEntries(ctx context.Context) ([]policy.WalletBlocklistEntry, error) {
	return r.database.ListWalletBlocklistEntries(ctx)
}
func (r *PolicyRepository) CreateWalletBlocklistEntry(ctx context.Context, item policy.WalletBlocklistEntry) error {
	return r.database.CreateWalletBlocklistEntry(ctx, item)
}
func (r *PolicyRepository) UpdateWalletBlocklistEntryNote(ctx context.Context, wallet common.Address, note string) (int64, error) {
	return r.database.UpdateWalletBlocklistEntryNote(ctx, wallet, note)
}
func (r *PolicyRepository) DeleteWalletBlocklistEntry(ctx context.Context, wallet common.Address) (int64, error) {
	return r.database.DeleteWalletBlocklistEntry(ctx, wallet)
}

func (r *OperationsRepository) ListChains(ctx context.Context) ([]discovery.Chain, error) {
	return r.database.ListChains(ctx)
}
func (r *OperationsRepository) GetChainIngestCheckpoint(ctx context.Context, id int64) (*discovery.ChainIngestCheckpoint, error) {
	return r.database.GetChainIngestCheckpoint(ctx, id)
}
func (r *OperationsRepository) ListChainIngestCheckpoints(ctx context.Context) ([]discovery.ChainIngestCheckpoint, error) {
	return r.database.ListChainIngestCheckpoints(ctx)
}
func (r *OperationsRepository) UpdateChainIngestCheckpointStatus(ctx context.Context, id int64, status discovery.ChainIngestStatus) (*discovery.ChainIngestCheckpoint, error) {
	return r.database.UpdateChainIngestCheckpointStatus(ctx, id, status)
}
