package tokenapi

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func parseAddressField(name, value string) (common.Address, error) {
	value = strings.TrimSpace(value)
	if !common.IsHexAddress(value) {
		return common.Address{}, status.Errorf(codes.InvalidArgument, "%s must be a valid hex address", name)
	}
	return common.HexToAddress(value), nil
}

func parseOptionalAddressField(name, value string) (common.Address, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return common.Address{}, nil
	}
	return parseAddressField(name, value)
}

func parseHashField(name, value string) (common.Hash, error) {
	value = strings.TrimSpace(value)
	raw := strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X")
	if len(raw) != 64 {
		return common.Hash{}, status.Errorf(codes.InvalidArgument, "%s must be a 32-byte hex hash", name)
	}
	bytes, err := hex.DecodeString(raw)
	if err != nil {
		return common.Hash{}, status.Errorf(codes.InvalidArgument, "%s must be a valid hex hash", name)
	}
	return common.BytesToHash(bytes), nil
}

func parseOptionalHashField(name, value string) (common.Hash, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return common.Hash{}, nil
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
	case tokenstore.ChainIngestStatusRunning, tokenstore.ChainIngestStatusStopped:
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "status must be %q or %q", tokenstore.ChainIngestStatusRunning, tokenstore.ChainIngestStatusStopped)
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
	case tokenstore.ProjectDataCollectionTypeAve,
		tokenstore.ProjectDataCollectionTypeChainState,
		tokenstore.ProjectDataCollectionTypeWalletAssetState,
		tokenstore.ProjectDataCollectionTypeSimulationResult,
		tokenstore.ProjectDataCollectionTypeContractCodeSource:
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "data_type must be one of %q, %q, %q, %q, or %q",
			tokenstore.ProjectDataCollectionTypeAve,
			tokenstore.ProjectDataCollectionTypeChainState,
			tokenstore.ProjectDataCollectionTypeWalletAssetState,
			tokenstore.ProjectDataCollectionTypeSimulationResult,
			tokenstore.ProjectDataCollectionTypeContractCodeSource,
		)
	}
}

func validateProjectDataCollectionStatus(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	switch value {
	case tokenstore.ProjectDataCollectionStatusPending,
		tokenstore.ProjectDataCollectionStatusRunning,
		tokenstore.ProjectDataCollectionStatusSucceeded,
		tokenstore.ProjectDataCollectionStatusFailed:
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "status must be one of %q, %q, %q, or %q",
			tokenstore.ProjectDataCollectionStatusPending,
			tokenstore.ProjectDataCollectionStatusRunning,
			tokenstore.ProjectDataCollectionStatusSucceeded,
			tokenstore.ProjectDataCollectionStatusFailed,
		)
	}
}

func validateProjectReportEvaluationStatus(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || value == tokenstore.ProjectTaskStatusSucceeded {
		return nil
	}
	return status.Errorf(codes.InvalidArgument, "evaluation_status must be empty or %q", tokenstore.ProjectTaskStatusSucceeded)
}

func validateProjectResearchStatus(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	switch value {
	case tokenstore.ProjectResearchStatusResearching, tokenstore.ProjectResearchStatusSelected, tokenstore.ProjectResearchStatusRejected, tokenstore.ProjectResearchStatusExpired:
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
	case tokenstore.ProjectSelectionOutcomeSelected, tokenstore.ProjectSelectionOutcomeRejected, tokenstore.ProjectSelectionOutcomeDeferred:
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

func requiredStore(store *tokenstore.SQLStore) (*tokenstore.SQLStore, error) {
	if store == nil {
		return nil, status.Error(codes.Internal, "token postgres database is not configured")
	}
	return store, nil
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func formatHash(value common.Hash) string {
	if value == (common.Hash{}) {
		return ""
	}
	return value.Hex()
}

func mapChainIngestCheckpoint(item tokenstore.ChainIngestCheckpoint) *v1alpha1.TokenAPIChainIngestCheckpoint {
	return &v1alpha1.TokenAPIChainIngestCheckpoint{
		ChainID:           item.ChainID,
		ChainName:         item.ChainName,
		Enabled:           item.Enabled,
		CursorBlockNumber: item.CursorBlockNumber,
		Status:            item.Status,
		CreatedAt:         formatTime(item.CreatedAt),
	}
}

func mapChainIngestCheckpoints(items []tokenstore.ChainIngestCheckpoint) []*v1alpha1.TokenAPIChainIngestCheckpoint {
	results := make([]*v1alpha1.TokenAPIChainIngestCheckpoint, 0, len(items))
	for _, item := range items {
		results = append(results, mapChainIngestCheckpoint(item))
	}
	return results
}

func mapChainOptions(items []tokenstore.Chain) []v1alpha1.TokenAPIChainOption {
	results := make([]v1alpha1.TokenAPIChainOption, 0, len(items))
	for _, item := range items {
		results = append(results, v1alpha1.TokenAPIChainOption{
			ChainID:   item.ID,
			ChainName: item.Name,
		})
	}
	return results
}

func mapContractCode(item tokenstore.ContractCode) *v1alpha1.TokenAPIContractCode {
	return &v1alpha1.TokenAPIContractCode{
		CodeHash:            item.CodeHash.Hex(),
		SourceCode:          item.SourceCode,
		SourceCodeFetchedAt: formatTime(item.SourceCodeFetchedAt),
		CreatedAt:           formatTime(item.CreatedAt),
		DeploymentCount:     item.DeploymentCount,
	}
}

func mapContractCodes(items []tokenstore.ContractCode) []*v1alpha1.TokenAPIContractCode {
	results := make([]*v1alpha1.TokenAPIContractCode, 0, len(items))
	for _, item := range items {
		results = append(results, mapContractCode(item))
	}
	return results
}

func mapProject(item tokenstore.Project) *v1alpha1.TokenAPIProject {
	return &v1alpha1.TokenAPIProject{
		ProjectID:   item.ID,
		ChainID:     item.ChainID,
		Name:        item.Name,
		Symbol:      item.Symbol,
		Contract:    item.Contract.Hex(),
		TxSender:    item.TxSender.Hex(),
		TxHash:      item.TxHash.Hex(),
		TxIndex:     item.TxIndex,
		BlockNumber: item.BlockNumber,
		BlockTime:   item.BlockTime,
		CodeHash:    item.CodeHash.Hex(),
		CreatedAt:   formatTime(item.CreatedAt),
	}
}

func mapProjects(items []tokenstore.Project) []*v1alpha1.TokenAPIProject {
	results := make([]*v1alpha1.TokenAPIProject, 0, len(items))
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

func mapProjectReport(item tokenstore.ProjectReportListItem) *v1alpha1.TokenAPIProjectReport {
	report := item.Report
	dataAvailable := report.WethPairIsCreated != nil &&
		report.WethPairIsRemoveLiquidity != nil &&
		report.WethPairIsMint != nil &&
		report.UsdtPairIsCreated != nil &&
		report.UsdtPairIsRemoveLiquidity != nil &&
		report.UsdtPairIsMint != nil
	result := &v1alpha1.TokenAPIProjectReport{
		ProjectID:           report.ProjectID,
		ChainID:             item.ChainID,
		Name:                item.Name,
		Symbol:              item.Symbol,
		Contract:            item.Contract.Hex(),
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
	result.WethPairIsCreated = *report.WethPairIsCreated
	result.WethPairIsRemoveLiquidity = *report.WethPairIsRemoveLiquidity
	result.WethPairIsMint = *report.WethPairIsMint
	result.WethPairQuoteUsdtValueInt = formatBigInt(report.WethPairQuoteUsdtValueInt)
	result.WethPairLastSwapAt = formatUnixTime(report.WethPairLastSwapTimestamp)
	result.UsdtPairIsCreated = *report.UsdtPairIsCreated
	result.UsdtPairIsRemoveLiquidity = *report.UsdtPairIsRemoveLiquidity
	result.UsdtPairIsMint = *report.UsdtPairIsMint
	result.UsdtPairQuoteUsdtValueInt = formatBigInt(report.UsdtPairQuoteUsdtValueInt)
	result.UsdtPairLastSwapAt = formatUnixTime(report.UsdtPairLastSwapTimestamp)
	return result
}

func mapProjectReports(items []tokenstore.ProjectReportListItem) []*v1alpha1.TokenAPIProjectReport {
	results := make([]*v1alpha1.TokenAPIProjectReport, 0, len(items))
	for _, item := range items {
		results = append(results, mapProjectReport(item))
	}
	return results
}

func mapProjectDataCollectionTask(item tokenstore.ProjectDataCollectionTask) *v1alpha1.TokenAPIProjectDataCollectionTask {
	return &v1alpha1.TokenAPIProjectDataCollectionTask{
		TaskID:         item.ID,
		ProjectID:      item.ProjectID,
		DataType:       item.DataType,
		Status:         item.Status,
		Revision:       item.Revision,
		Attempts:       item.Attempts,
		AvailableAt:    formatTime(item.AvailableAt),
		LeaseExpiresAt: formatTime(item.LeaseExpiresAt),
		LastError:      item.LastError,
		CreatedAt:      formatTime(item.CreatedAt),
		UpdatedAt:      formatTime(item.UpdatedAt),
	}
}

func mapProjectDataCollectionTasks(items []tokenstore.ProjectDataCollectionTask) []*v1alpha1.TokenAPIProjectDataCollectionTask {
	results := make([]*v1alpha1.TokenAPIProjectDataCollectionTask, 0, len(items))
	for _, item := range items {
		results = append(results, mapProjectDataCollectionTask(item))
	}
	return results
}

func mapContractCodeBlocklistEntry(item tokenstore.ContractCodeBlocklistEntry) *v1alpha1.TokenAPIContractCodeBlocklistEntry {
	sourceContract := ""
	if item.SourceContract != (common.Address{}) {
		sourceContract = item.SourceContract.Hex()
	}
	return &v1alpha1.TokenAPIContractCodeBlocklistEntry{
		CodeHash:       item.CodeHash.Hex(),
		Note:           item.Note,
		SourceChainID:  item.SourceChainID,
		SourceContract: sourceContract,
		CreatedAt:      formatTime(item.CreatedAt),
	}
}

func mapContractCodeBlocklistEntries(items []tokenstore.ContractCodeBlocklistEntry) []*v1alpha1.TokenAPIContractCodeBlocklistEntry {
	results := make([]*v1alpha1.TokenAPIContractCodeBlocklistEntry, 0, len(items))
	for _, item := range items {
		results = append(results, mapContractCodeBlocklistEntry(item))
	}
	return results
}

func mapWalletBlocklistEntry(item tokenstore.WalletBlocklistEntry) *v1alpha1.TokenAPIWalletBlocklistEntry {
	return &v1alpha1.TokenAPIWalletBlocklistEntry{
		Wallet:    item.Wallet.Hex(),
		Note:      item.Note,
		CreatedAt: formatTime(item.CreatedAt),
	}
}

func mapWalletBlocklistEntries(items []tokenstore.WalletBlocklistEntry) []*v1alpha1.TokenAPIWalletBlocklistEntry {
	results := make([]*v1alpha1.TokenAPIWalletBlocklistEntry, 0, len(items))
	for _, item := range items {
		results = append(results, mapWalletBlocklistEntry(item))
	}
	return results
}

func mapProjectResearchState(item tokenstore.ProjectResearchState) *v1alpha1.TokenAPIProjectResearchState {
	return &v1alpha1.TokenAPIProjectResearchState{ProjectID: item.ProjectID, ChainID: item.ChainID, Contract: item.Contract.Hex(), Status: item.Status, EvidenceRevision: item.EvidenceRevision, CurrentReportRevision: item.CurrentReportRevision, CurrentSelectionOutcome: item.CurrentSelectionOutcome, LastEvaluatedRevision: item.LastEvaluatedReportRevision, LastEvaluatedAt: formatTime(item.LastEvaluatedAt), ExpiresAt: formatTime(item.ExpiresAt), CreatedAt: formatTime(item.CreatedAt), UpdatedAt: formatTime(item.UpdatedAt)}
}
func mapProjectResearchStates(items []tokenstore.ProjectResearchState) []*v1alpha1.TokenAPIProjectResearchState {
	out := make([]*v1alpha1.TokenAPIProjectResearchState, 0, len(items))
	for _, item := range items {
		out = append(out, mapProjectResearchState(item))
	}
	return out
}
func mapProjectReportRevision(item tokenstore.ProjectReportRevision) *v1alpha1.TokenAPIProjectReportRevision {
	var block uint64
	if item.ObservedBlockNumber != nil {
		block = *item.ObservedBlockNumber
	}
	return &v1alpha1.TokenAPIProjectReportRevision{
		ReportRevisionID:          item.ID,
		ProjectID:                 item.ProjectID,
		ChainID:                   item.ChainID,
		Contract:                  item.Contract.Hex(),
		Revision:                  item.Revision,
		ContentHash:               item.ContentHash.Hex(),
		CompletenessStatus:        item.CompletenessStatus,
		EvidenceJSON:              string(item.Evidence),
		ReportJSON:                string(item.Report),
		ObservedBlockNumber:       block,
		WethPairIsCreated:         boolValue(item.WethPairIsCreated),
		WethPairIsRemoveLiquidity: boolValue(item.WethPairIsRemoveLiquidity),
		WethPairIsMint:            boolValue(item.WethPairIsMint),
		WethPairQuoteUsdtValueInt: formatBigInt(item.WethPairQuoteUsdtValueInt),
		WethPairLastSwapAt:        formatUnixTime(item.WethPairLastSwapTimestamp),
		UsdtPairIsCreated:         boolValue(item.UsdtPairIsCreated),
		UsdtPairIsRemoveLiquidity: boolValue(item.UsdtPairIsRemoveLiquidity),
		UsdtPairIsMint:            boolValue(item.UsdtPairIsMint),
		UsdtPairQuoteUsdtValueInt: formatBigInt(item.UsdtPairQuoteUsdtValueInt),
		UsdtPairLastSwapAt:        formatUnixTime(item.UsdtPairLastSwapTimestamp),
		BuiltAt:                   formatTime(item.BuiltAt),
		CreatedAt:                 formatTime(item.CreatedAt),
	}
}
func mapProjectReportRevisions(items []tokenstore.ProjectReportRevision) []*v1alpha1.TokenAPIProjectReportRevision {
	out := make([]*v1alpha1.TokenAPIProjectReportRevision, 0, len(items))
	for _, item := range items {
		out = append(out, mapProjectReportRevision(item))
	}
	return out
}
func mapProjectSelection(item tokenstore.ProjectSelection) *v1alpha1.TokenAPIProjectSelection {
	return &v1alpha1.TokenAPIProjectSelection{SelectionID: item.ID, ProjectID: item.ProjectID, ChainID: item.ChainID, Contract: item.Contract.Hex(), Outcome: item.Outcome, StrategyKey: item.StrategyKey, StrategyVersion: item.StrategyVersion, ReportRevision: item.ReportRevision, ReasonCodes: item.ReasonCodes, ReasonDetail: item.ReasonDetail, DecidedAt: formatTime(item.DecidedAt), CreatedAt: formatTime(item.CreatedAt)}
}
func mapProjectSelections(items []tokenstore.ProjectSelection) []*v1alpha1.TokenAPIProjectSelection {
	out := make([]*v1alpha1.TokenAPIProjectSelection, 0, len(items))
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
