package tokenapi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/catalog"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/policy"
	"github.com/useryege/athena/internal/token/reporting"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/selection"
	"github.com/useryege/athena/internal/token/shared"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func parseAddressField(name, value string) (shared.Address, error) {
	value = strings.TrimSpace(value)
	if !common.IsHexAddress(value) {
		return shared.Address{}, status.Errorf(codes.InvalidArgument, "%s must be a valid hex address", name)
	}
	return shared.Address(common.HexToAddress(value)), nil
}

func parseOptionalAddressField(name, value string) (shared.Address, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return shared.Address{}, nil
	}
	return parseAddressField(name, value)
}

func parseHashField(name, value string) (shared.Hash, error) {
	value = strings.TrimSpace(value)
	raw := strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X")
	if len(raw) != 64 {
		return shared.Hash{}, status.Errorf(codes.InvalidArgument, "%s must be a 32-byte hex hash", name)
	}
	bytes, err := hex.DecodeString(raw)
	if err != nil {
		return shared.Hash{}, status.Errorf(codes.InvalidArgument, "%s must be a valid hex hash", name)
	}
	return shared.BytesToHash(bytes), nil
}

func parseOptionalHashField(name, value string) (shared.Hash, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return shared.Hash{}, nil
	}
	return parseHashField(name, value)
}

func validatePositiveInt64Field(name string, value int64) error {
	if value <= 0 {
		return status.Errorf(codes.InvalidArgument, "%s must be positive", name)
	}
	return nil
}

func validateNonNegativeInt64Field(name string, value int64) error {
	if value < 0 {
		return status.Errorf(codes.InvalidArgument, "%s must not be negative", name)
	}
	return nil
}

func validateChainIngestStatus(value string) error {
	value = strings.TrimSpace(value)
	switch value {
	case string(discovery.ChainIngestStatusRunning), string(discovery.ChainIngestStatusStopped):
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "status must be %q or %q", discovery.ChainIngestStatusRunning, discovery.ChainIngestStatusStopped)
	}
}

func validateRequiredProjectDataCollectionType(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return status.Error(codes.InvalidArgument, "data_type must not be empty")
	}
	return validateProjectDataCollectionType(value)
}

func validateProjectDataCollectionType(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	switch value {
	case string(research.DataCollectionTypeAve),
		string(research.DataCollectionTypeChainState),
		string(research.DataCollectionTypeWalletAssetState),
		string(research.DataCollectionTypeSimulationResult),
		string(research.DataCollectionTypeContractCodeSource),
		string(research.DataCollectionTypeWalletNormalTransactions):
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "data_type must be one of %q, %q, %q, %q, %q, or %q",
			research.DataCollectionTypeAve,
			research.DataCollectionTypeChainState,
			research.DataCollectionTypeWalletAssetState,
			research.DataCollectionTypeSimulationResult,
			research.DataCollectionTypeContractCodeSource,
			research.DataCollectionTypeWalletNormalTransactions,
		)
	}
}

func validateProjectDataCollectionStatus(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	switch value {
	case string(research.TaskStatusPending), string(research.TaskStatusRunning), string(research.TaskStatusSucceeded), string(research.TaskStatusFailed):
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "status must be one of %q, %q, %q, or %q",
			research.TaskStatusPending, research.TaskStatusRunning, research.TaskStatusSucceeded, research.TaskStatusFailed,
		)
	}
}

func validateProjectReportEvaluationStatus(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || value == string(research.TaskStatusSucceeded) {
		return nil
	}
	return status.Errorf(codes.InvalidArgument, "evaluation_status must be empty or %q", research.TaskStatusSucceeded)
}

func validateProjectResearchStatus(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	switch value {
	case string(research.ProjectResearchStatusResearching), string(research.ProjectResearchStatusSelected), string(research.ProjectResearchStatusRejected), string(research.ProjectResearchStatusExpired):
		return nil
	}
	return status.Error(codes.InvalidArgument, "invalid research status")
}
func validateProjectSelectionOutcome(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	switch value {
	case string(selection.SelectionOutcomeSelected), string(selection.SelectionOutcomeRejected), string(selection.SelectionOutcomeDeferred):
		return nil
	}
	return status.Error(codes.InvalidArgument, "invalid selection outcome")
}

func grpcStoreError(err error) error {
	if err == nil {
		return nil
	}
	return status.Error(codes.Internal, err.Error())
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func formatHash(value shared.Hash) string {
	if value.IsZero() {
		return ""
	}
	return value.Hex()
}

func formatAddress(value shared.Address) string {
	if value.IsZero() {
		return ""
	}
	return common.Address(value).Hex()
}

func mapChainIngestCheckpoint(item discovery.ChainIngestCheckpoint) *v1alpha1.TokenChainCheckpoint {
	return &v1alpha1.TokenChainCheckpoint{
		ChainID:           item.ChainID,
		ChainName:         item.ChainName,
		Enabled:           item.Enabled,
		CursorBlockNumber: item.CursorBlockNumber,
		Status:            string(item.Status),
		CreatedAt:         formatTime(item.CreatedAt),
	}
}

func mapChainIngestCheckpoints(items []discovery.ChainIngestCheckpoint) []*v1alpha1.TokenChainCheckpoint {
	results := make([]*v1alpha1.TokenChainCheckpoint, 0, len(items))
	for _, item := range items {
		results = append(results, mapChainIngestCheckpoint(item))
	}
	return results
}

func mapChainOptions(items []discovery.Chain) []v1alpha1.TokenChain {
	results := make([]v1alpha1.TokenChain, 0, len(items))
	for _, item := range items {
		results = append(results, v1alpha1.TokenChain{
			ChainID:   item.ID,
			ChainName: item.Name,
		})
	}
	return results
}

func mapContractCode(item catalog.ContractCode) *v1alpha1.TokenContractCode {
	return &v1alpha1.TokenContractCode{
		CodeHash:            item.CodeHash.Hex(),
		SourceCode:          item.SourceCode,
		SourceCodeFetchedAt: formatTime(item.SourceCodeFetchedAt),
		CreatedAt:           formatTime(item.CreatedAt),
		DeploymentCount:     item.DeploymentCount,
	}
}

func mapContractCodes(items []catalog.ContractCode) []*v1alpha1.TokenContractCode {
	results := make([]*v1alpha1.TokenContractCode, 0, len(items))
	for _, item := range items {
		results = append(results, mapContractCode(item))
	}
	return results
}

func mapProject(item catalog.Project) *v1alpha1.TokenProject {
	return &v1alpha1.TokenProject{
		ProjectID:   item.ID,
		ChainID:     item.ChainID,
		Name:        item.Name,
		Symbol:      item.Symbol,
		Contract:    formatAddress(item.Contract),
		TxSender:    formatAddress(item.TxSender),
		TxHash:      item.TxHash.Hex(),
		TxIndex:     item.TxIndex,
		BlockNumber: item.BlockNumber,
		BlockTime:   item.BlockTime,
		CodeHash:    item.CodeHash.Hex(),
		CreatedAt:   formatTime(item.CreatedAt),
	}
}

func mapProjects(items []catalog.Project) []*v1alpha1.TokenProject {
	results := make([]*v1alpha1.TokenProject, 0, len(items))
	for _, item := range items {
		results = append(results, mapProject(item))
	}
	return results
}

func formatUnixTime(value *uint64) string {
	if value == nil || *value == 0 {
		return ""
	}
	return time.Unix(int64(*value), 0).UTC().Format(time.RFC3339)
}

func formatBigInt(value *big.Int) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func mapProjectReport(item reporting.ProjectReportReadModel) *v1alpha1.TokenProjectReport {
	report := item.Report
	risk := report.Report.RiskSummary
	dataAvailable := risk.WethPairIsCreated != nil &&
		risk.WethPairIsRemoveLiquidity != nil &&
		risk.WethPairIsMint != nil &&
		risk.UsdtPairIsCreated != nil &&
		risk.UsdtPairIsRemoveLiquidity != nil &&
		risk.UsdtPairIsMint != nil
	result := &v1alpha1.TokenProjectReport{
		ProjectID:           report.ProjectID,
		ChainID:             item.ChainID,
		Name:                item.Name,
		Symbol:              item.Symbol,
		Contract:            formatAddress(item.Contract),
		EvaluationStatus:    item.BuildStatus,
		EvaluationAttempts:  item.BuildAttempts,
		EvaluationLastError: item.BuildLastError,
		EvaluationUpdatedAt: formatTime(item.BuildUpdatedAt),
		ReportDataAvailable: dataAvailable,
		SourceUpdatedAt:     formatTime(report.BuiltAt),
		EvaluatedAt:         formatTime(report.BuiltAt),
		CreatedAt:           formatTime(report.CreatedAt),
	}
	if !dataAvailable {
		return result
	}
	result.WethPairIsCreated = *risk.WethPairIsCreated
	result.WethPairIsRemoveLiquidity = *risk.WethPairIsRemoveLiquidity
	result.WethPairIsMint = *risk.WethPairIsMint
	result.WethPairQuoteUsdtValueInt = formatBigInt(risk.WethPairQuoteUsdtValueInt)
	result.WethPairLastSwapAt = formatUnixTime(risk.WethPairLastSwapTimestamp)
	result.UsdtPairIsCreated = *risk.UsdtPairIsCreated
	result.UsdtPairIsRemoveLiquidity = *risk.UsdtPairIsRemoveLiquidity
	result.UsdtPairIsMint = *risk.UsdtPairIsMint
	result.UsdtPairQuoteUsdtValueInt = formatBigInt(risk.UsdtPairQuoteUsdtValueInt)
	result.UsdtPairLastSwapAt = formatUnixTime(risk.UsdtPairLastSwapTimestamp)
	return result
}

func mapProjectReports(items []reporting.ProjectReportReadModel) []*v1alpha1.TokenProjectReport {
	results := make([]*v1alpha1.TokenProjectReport, 0, len(items))
	for _, item := range items {
		results = append(results, mapProjectReport(item))
	}
	return results
}

func mapProjectDataCollectionTask(item research.ProjectDataCollectionTask) *v1alpha1.TokenCollectionTask {
	return &v1alpha1.TokenCollectionTask{
		TaskID:         item.ID,
		ProjectID:      item.ProjectID,
		DataType:       string(item.DataType),
		Status:         string(item.Status),
		Revision:       item.Revision,
		Attempts:       item.Attempts,
		AvailableAt:    formatTime(item.AvailableAt),
		LeaseExpiresAt: formatTime(item.LeaseExpiresAt),
		LastError:      item.LastError,
		CreatedAt:      formatTime(item.CreatedAt),
		UpdatedAt:      formatTime(item.UpdatedAt),
	}
}

func mapProjectDataCollectionTasks(items []research.ProjectDataCollectionTask) []*v1alpha1.TokenCollectionTask {
	results := make([]*v1alpha1.TokenCollectionTask, 0, len(items))
	for _, item := range items {
		results = append(results, mapProjectDataCollectionTask(item))
	}
	return results
}

func mapContractCodeBlocklistEntry(item policy.ContractCodeBlocklistEntry) *v1alpha1.TokenContractCodeBlocklistEntry {
	sourceContract := ""
	if !item.SourceContract.IsZero() {
		sourceContract = formatAddress(item.SourceContract)
	}
	return &v1alpha1.TokenContractCodeBlocklistEntry{
		CodeHash:       item.CodeHash.Hex(),
		Note:           item.Note,
		SourceChainID:  item.SourceChainID,
		SourceContract: sourceContract,
		CreatedAt:      formatTime(item.CreatedAt),
	}
}

func mapContractCodeBlocklistEntries(items []policy.ContractCodeBlocklistEntry) []*v1alpha1.TokenContractCodeBlocklistEntry {
	results := make([]*v1alpha1.TokenContractCodeBlocklistEntry, 0, len(items))
	for _, item := range items {
		results = append(results, mapContractCodeBlocklistEntry(item))
	}
	return results
}

func mapWalletBlocklistEntry(item policy.WalletBlocklistEntry) *v1alpha1.TokenWalletBlocklistEntry {
	return &v1alpha1.TokenWalletBlocklistEntry{
		Wallet:    formatAddress(item.Wallet),
		Note:      item.Note,
		CreatedAt: formatTime(item.CreatedAt),
	}
}

func mapWalletBlocklistEntries(items []policy.WalletBlocklistEntry) []*v1alpha1.TokenWalletBlocklistEntry {
	results := make([]*v1alpha1.TokenWalletBlocklistEntry, 0, len(items))
	for _, item := range items {
		results = append(results, mapWalletBlocklistEntry(item))
	}
	return results
}

func mapProjectResearchState(item research.ProjectResearchState) *v1alpha1.TokenResearchState {
	return &v1alpha1.TokenResearchState{ProjectID: item.ProjectID, ChainID: item.ChainID, Contract: formatAddress(item.Contract), Status: string(item.Status), EvidenceRevision: item.EvidenceRevision, CurrentReportRevision: item.CurrentReportRevision, CurrentSelectionOutcome: string(item.CurrentSelectionOutcome), LastEvaluatedRevision: item.LastEvaluatedReportRevision, LastEvaluatedAt: formatTime(item.LastEvaluatedAt), ExpiresAt: formatTime(item.ExpiresAt), CreatedAt: formatTime(item.CreatedAt), UpdatedAt: formatTime(item.UpdatedAt)}
}
func mapProjectResearchStates(items []research.ProjectResearchState) []*v1alpha1.TokenResearchState {
	out := make([]*v1alpha1.TokenResearchState, 0, len(items))
	for _, item := range items {
		out = append(out, mapProjectResearchState(item))
	}
	return out
}
func mapProjectReportRevision(item reporting.ProjectReportRevision) *v1alpha1.TokenReportRevision {
	var block uint64
	if item.ObservedBlockNumber != nil {
		block = *item.ObservedBlockNumber
	}
	risk := item.Report.RiskSummary
	return &v1alpha1.TokenReportRevision{
		ReportRevisionID:          item.ID,
		ProjectID:                 item.ProjectID,
		ChainID:                   item.ChainID,
		Contract:                  formatAddress(item.Contract),
		Revision:                  item.Revision,
		ContentHash:               item.ContentHash.Hex(),
		CompletenessStatus:        item.CompletenessStatus,
		EvidenceJSON:              jsonString(item.Evidence),
		ReportJSON:                jsonString(item.Report),
		ObservedBlockNumber:       block,
		WethPairIsCreated:         boolValue(risk.WethPairIsCreated),
		WethPairIsRemoveLiquidity: boolValue(risk.WethPairIsRemoveLiquidity),
		WethPairIsMint:            boolValue(risk.WethPairIsMint),
		WethPairQuoteUsdtValueInt: formatBigInt(risk.WethPairQuoteUsdtValueInt),
		WethPairLastSwapAt:        formatUnixTime(risk.WethPairLastSwapTimestamp),
		UsdtPairIsCreated:         boolValue(risk.UsdtPairIsCreated),
		UsdtPairIsRemoveLiquidity: boolValue(risk.UsdtPairIsRemoveLiquidity),
		UsdtPairIsMint:            boolValue(risk.UsdtPairIsMint),
		UsdtPairQuoteUsdtValueInt: formatBigInt(risk.UsdtPairQuoteUsdtValueInt),
		UsdtPairLastSwapAt:        formatUnixTime(risk.UsdtPairLastSwapTimestamp),
		BuiltAt:                   formatTime(item.BuiltAt),
		CreatedAt:                 formatTime(item.CreatedAt),
	}
}

func jsonString(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}
func mapProjectReportRevisions(items []reporting.ProjectReportRevision) []*v1alpha1.TokenReportRevision {
	out := make([]*v1alpha1.TokenReportRevision, 0, len(items))
	for _, item := range items {
		out = append(out, mapProjectReportRevision(item))
	}
	return out
}
func mapProjectSelection(item selection.ProjectSelection) *v1alpha1.TokenSelection {
	return &v1alpha1.TokenSelection{SelectionID: item.ID, ProjectID: item.ProjectID, ChainID: item.ChainID, Contract: formatAddress(item.Contract), Outcome: string(item.Outcome), StrategyKey: item.StrategyKey, StrategyVersion: item.StrategyVersion, ReportRevision: item.ReportRevision, ReasonCodes: item.ReasonCodes, ReasonDetail: item.ReasonDetail, DecidedAt: formatTime(item.DecidedAt), CreatedAt: formatTime(item.CreatedAt)}
}
func mapProjectSelections(items []selection.ProjectSelection) []*v1alpha1.TokenSelection {
	out := make([]*v1alpha1.TokenSelection, 0, len(items))
	for _, item := range items {
		out = append(out, mapProjectSelection(item))
	}
	return out
}

func wrapStoreError(action string, err error) error {
	if err == nil {
		return nil
	}
	return grpcStoreError(fmt.Errorf("%s: %w", action, err))
}
