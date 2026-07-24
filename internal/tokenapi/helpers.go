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
	"github.com/useryege/athena/internal/token/projectview"
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
		Decimals:    int32(item.Decimals),
		TotalSupply: formatBigInt(item.TotalSupply),
		WethPair:    formatAddress(item.WethPair),
		UsdtPair:    formatAddress(item.UsdtPair),
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

func mapProjectObservation(item research.ProjectObservation) *v1alpha1.TokenProjectObservation {
	var blockNumber uint64
	if item.BlockNumber != nil {
		blockNumber = *item.BlockNumber
	}
	return &v1alpha1.TokenProjectObservation{
		ObservationID: item.ID,
		ProjectID:     item.ProjectID,
		DataType:      string(item.DataType),
		SchemaVersion: item.SchemaVersion,
		ContentHash:   item.ContentHash.Hex(),
		PayloadJSON:   string(item.Payload),
		BlockNumber:   blockNumber,
		ObservedAt:    formatTime(item.ObservedAt),
		LastCheckedAt: formatTime(item.LastCheckedAt),
		CreatedAt:     formatTime(item.CreatedAt),
	}
}

func mapProjectObservations(items []research.ProjectObservation) []*v1alpha1.TokenProjectObservation {
	result := make([]*v1alpha1.TokenProjectObservation, 0, len(items))
	for _, item := range items {
		result = append(result, mapProjectObservation(item))
	}
	return result
}

func decimalString(value *research.Decimal) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func optionalTimeString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatTime(*value)
}

func mapAveToken(item research.AveTokenV1) *v1alpha1.TokenAveToken {
	result := &v1alpha1.TokenAveToken{
		Address:          formatAddress(item.Address),
		Name:             item.Name,
		Symbol:           item.Symbol,
		Decimals:         int32(item.Decimals),
		TotalSupply:      decimalString(item.TotalSupply),
		CurrentPriceUSD:  decimalString(item.CurrentPriceUSD),
		CurrentPriceETH:  decimalString(item.CurrentPriceETH),
		MarketCap:        decimalString(item.MarketCap),
		FDV:              decimalString(item.FDV),
		TVL:              decimalString(item.TVL),
		MainPairTVL:      decimalString(item.MainPairTVL),
		Holders:          int32(item.Holders),
		RiskLevel:        int32(item.RiskLevel),
		RiskScore:        decimalString(item.RiskScore),
		RiskInfo:         item.RiskInfo,
		HasMintMethod:    item.HasMintMethod,
		IsLPNotLocked:    item.IsLPNotLocked,
		HasNotRenounced:  item.HasNotRenounced,
		HasNotAudited:    item.HasNotAudited,
		HasNotOpenSource: item.HasNotOpenSource,
		IsInBlacklist:    item.IsInBlacklist,
		IsHoneypot:       item.IsHoneypot,
		LaunchAt:         optionalTimeString(item.LaunchAt),
		UpdatedAt:        optionalTimeString(item.UpdatedAt),
	}
	if item.IsMintable != nil {
		result.IsMintableKnown = true
		result.IsMintable = *item.IsMintable
	}
	return result
}

func mapAvePair(item research.AvePairV1) *v1alpha1.TokenAvePair {
	return &v1alpha1.TokenAvePair{
		Pair:          formatAddress(item.Pair),
		ChainID:       item.ChainID,
		AMM:           item.AMM,
		Token0Address: formatAddress(item.Token0Address),
		Token0Symbol:  item.Token0Symbol,
		Token1Address: formatAddress(item.Token1Address),
		Token1Symbol:  item.Token1Symbol,
		Reserve0:      decimalString(item.Reserve0),
		Reserve1:      decimalString(item.Reserve1),
		VolumeUSD:     decimalString(item.VolumeUSD),
		MarketCap:     decimalString(item.MarketCap),
		FDV:           decimalString(item.FDV),
		IsFake:        item.IsFake,
		CreatedAt:     optionalTimeString(item.CreatedAt),
		UpdatedAt:     optionalTimeString(item.UpdatedAt),
	}
}

func mapAveObservation(item research.AveObservationV1) *v1alpha1.TokenAveObservation {
	pairs := make([]*v1alpha1.TokenAvePair, 0, len(item.Pairs))
	for _, pair := range item.Pairs {
		pairs = append(pairs, mapAvePair(pair))
	}
	return &v1alpha1.TokenAveObservation{
		ChainID:   item.ChainID,
		Token:     mapAveToken(item.Token),
		Pairs:     pairs,
		IsAudited: item.IsAudited,
	}
}

func mapChainToken(item research.ChainTokenV1) *v1alpha1.TokenChainToken {
	return &v1alpha1.TokenChainToken{
		IsValidERC20: item.IsValidERC20,
		Name:         item.Name,
		Symbol:       item.Symbol,
		Decimals:     int32(item.Decimals),
		TotalSupply:  formatBigInt(item.TotalSupply),
		WethPair:     formatAddress(item.WethPair),
		UsdtPair:     formatAddress(item.UsdtPair),
	}
}

func mapChainPair(item research.ChainPairV1, report research.ChainPairReportV1) *v1alpha1.TokenChainPair {
	lastSwap := uint64(item.LastSwapTimestamp)
	return &v1alpha1.TokenChainPair{
		PairContract: formatAddress(item.PairContract),
		IsCreated:    item.IsCreated,
		Liquidity: &v1alpha1.TokenChainPairLiquidity{
			TotalSupply:                    formatBigInt(item.LiquidityState.TotalSupply),
			LockedLiquidity:                formatBigInt(item.LiquidityState.LockedLiquidity),
			FeeAddressHoldLiquidityBalance: formatBigInt(item.LiquidityState.FeeAddressHoldLiquidityBalance),
			FeeAddressHoldLiquidityRatio:   formatBigInt(item.LiquidityState.FeeAddressHoldLiquidityRatio),
		},
		BaseBalance:       formatBigInt(item.BaseBalance),
		QuoteBalance:      formatBigInt(item.QuoteBalance),
		QuoteUsdtValue:    formatBigInt(item.QuoteUsdtValue),
		QuoteUsdtValueInt: formatBigInt(item.QuoteUsdtValueInt),
		LastSwapAt:        formatUnixTime(&lastSwap),
		IsRemoveLiquidity: report.IsRemoveLiquidity,
		IsMint:            report.IsMint,
	}
}

func mapChainStateObservation(item research.ChainStateObservationV1) *v1alpha1.TokenChainStateObservation {
	return &v1alpha1.TokenChainStateObservation{
		TokenContract: formatAddress(item.TokenContract),
		UpdatedAt:     formatBigInt(item.UpdatedAt),
		Token:         mapChainToken(item.Token),
		IsValidERC20:  item.TokenReport.IsValidERC20,
		WethPair:      mapChainPair(item.WethPair, item.WethReport),
		UsdtPair:      mapChainPair(item.UsdtPair, item.UsdtReport),
	}
}

func mapWalletAssetState(item research.WalletAssetStateV1) *v1alpha1.TokenWalletAssetState {
	return &v1alpha1.TokenWalletAssetState{
		ChainID:       item.ChainID,
		Wallet:        formatAddress(item.Wallet),
		WethBalance:   formatBigInt(item.WethBalance),
		UsdtBalance:   formatBigInt(item.UsdtBalance),
		NativeBalance: formatBigInt(item.NativeBalance),
		UsdtValue:     formatBigInt(item.UsdtValue),
	}
}

func mapSimulationResult(item research.SimulationResultV1) *v1alpha1.TokenSimulationResult {
	return &v1alpha1.TokenSimulationResult{
		ProjectID:                          item.ProjectID,
		Wallet:                             formatAddress(item.Wallet),
		CanMintFromDeadViaTransferFrom:     item.CanMintFromDeadViaTransferFrom,
		CanMintFromZeroViaTransferFrom:     item.CanMintFromZeroViaTransferFrom,
		CanMintFromWethPairViaTransferFrom: item.CanMintFromWethPairViaTransferFrom,
		CanMintFromUsdtPairViaTransferFrom: item.CanMintFromUsdtPairViaTransferFrom,
		CanMintViaTransferToWethPair:       item.CanMintViaTransferToWethPair,
		CanMintViaTransferToUsdtPair:       item.CanMintViaTransferToUsdtPair,
	}
}

func mapProjectDetail(item projectview.Detail) (*v1alpha1.TokenProjectDetail, error) {
	result := &v1alpha1.TokenProjectDetail{
		Project:             mapProject(item.Project),
		TransactionCount:    item.TransactionCount,
		CurrentObservations: mapProjectObservations(item.CurrentObservations),
		GeneratedAt:         formatTime(time.Now()),
	}
	if item.ResearchState != nil {
		result.ResearchState = mapProjectResearchState(*item.ResearchState)
	}
	if item.CurrentReport != nil {
		result.CurrentReport = mapProjectReportRevision(*item.CurrentReport)
	}
	if item.CurrentSelection != nil {
		result.CurrentSelection = mapProjectSelection(*item.CurrentSelection)
	}
	for _, relatedWallet := range item.RelatedWallets {
		result.RelatedWallets = append(result.RelatedWallets, &v1alpha1.TokenProjectRelatedWallet{
			ProjectID: relatedWallet.ProjectID,
			Wallet:    formatAddress(relatedWallet.Wallet),
			Role:      string(relatedWallet.Role),
			CreatedAt: formatTime(relatedWallet.CreatedAt),
		})
	}
	for _, recipient := range item.InitialRecipients {
		result.InitialRecipients = append(result.InitialRecipients, &v1alpha1.TokenProjectInitialRecipient{
			RecipientID:       recipient.ID,
			ProjectID:         recipient.ProjectID,
			Wallet:            formatAddress(recipient.Wallet),
			RatioBPS:          recipient.RatioBPS,
			RankIndex:         recipient.RankIndex,
			SourceTxHash:      recipient.SourceTxHash.Hex(),
			SourceBlockNumber: recipient.SourceBlockNumber,
			CreatedAt:         formatTime(recipient.CreatedAt),
		})
	}
	for _, schedule := range item.CollectionSchedules {
		result.CollectionSchedules = append(result.CollectionSchedules, &v1alpha1.TokenCollectionSchedule{
			ProjectID:           schedule.ProjectID,
			DataType:            string(schedule.DataType),
			Status:              string(schedule.Status),
			RefreshIntervalSecs: int64(schedule.RefreshInterval / time.Second),
			NextRunAt:           formatTime(schedule.NextRunAt),
			LatestTaskRevision:  schedule.LatestTaskRevision,
			ConsecutiveFailures: schedule.ConsecutiveFailures,
			LastError:           schedule.LastError,
			LastCheckedAt:       formatTime(schedule.LastCheckedAt),
			CreatedAt:           formatTime(schedule.CreatedAt),
			UpdatedAt:           formatTime(schedule.UpdatedAt),
		})
	}
	for _, count := range item.WalletTransactionCounts {
		result.WalletTransactionCounts = append(result.WalletTransactionCounts, &v1alpha1.TokenWalletTransactionCount{
			Wallet:           formatAddress(count.Wallet),
			TransactionCount: count.TransactionCount,
		})
	}
	for _, observation := range item.CurrentObservations {
		if observation.SchemaVersion != research.ObservationSchemaVersionV1 {
			continue
		}
		switch observation.DataType {
		case research.DataCollectionTypeAve:
			var value research.AveObservationV1
			if err := json.Unmarshal(observation.Payload, &value); err != nil {
				return nil, fmt.Errorf("decode current ave observation: %w", err)
			}
			result.Ave = mapAveObservation(value)
		case research.DataCollectionTypeChainState:
			var value research.ChainStateObservationV1
			if err := json.Unmarshal(observation.Payload, &value); err != nil {
				return nil, fmt.Errorf("decode current chain state observation: %w", err)
			}
			result.ChainState = mapChainStateObservation(value)
		case research.DataCollectionTypeWalletAssetState:
			var value research.WalletAssetObservationV1
			if err := json.Unmarshal(observation.Payload, &value); err != nil {
				return nil, fmt.Errorf("decode current wallet asset observation: %w", err)
			}
			for _, wallet := range value.Items {
				result.WalletAssets = append(result.WalletAssets, mapWalletAssetState(wallet))
			}
		case research.DataCollectionTypeSimulationResult:
			var value research.SimulationObservationV1
			if err := json.Unmarshal(observation.Payload, &value); err != nil {
				return nil, fmt.Errorf("decode current simulation observation: %w", err)
			}
			for _, simulation := range value.Items {
				result.Simulations = append(result.Simulations, mapSimulationResult(simulation))
			}
		case research.DataCollectionTypeContractCodeSource:
			var value research.ContractSourceObservationV1
			if err := json.Unmarshal(observation.Payload, &value); err != nil {
				return nil, fmt.Errorf("decode current contract source observation: %w", err)
			}
			result.ContractSource = &v1alpha1.TokenContractSourceObservation{
				CodeHash:        value.CodeHash.Hex(),
				SourceAvailable: value.SourceAvailable,
			}
		}
	}
	return result, nil
}

func mapProjectTrends(item projectview.TrendResult) *v1alpha1.TokenProjectTrends {
	result := &v1alpha1.TokenProjectTrends{
		Range:        item.Range,
		ObservedFrom: formatTime(item.ObservedFrom),
		GeneratedAt:  formatTime(item.GeneratedAt),
	}
	for _, series := range item.Series {
		mapped := &v1alpha1.TokenProjectTrendSeries{
			Key:      series.Key,
			Label:    series.Label,
			Unit:     series.Unit,
			DataType: string(series.DataType),
		}
		for _, point := range series.Points {
			mapped.Points = append(mapped.Points, &v1alpha1.TokenProjectTrendPoint{
				ObservedAt: formatTime(point.ObservedAt),
				Value:      point.Value,
			})
		}
		result.Series = append(result.Series, mapped)
	}
	return result
}

func mapWalletNormalTransaction(item research.WalletNormalTransaction) *v1alpha1.TokenWalletNormalTransaction {
	blockTimestamp := item.BlockTimestamp
	return &v1alpha1.TokenWalletNormalTransaction{
		Wallet:           formatAddress(item.Wallet),
		TransactionHash:  item.TransactionHash.Hex(),
		BlockNumber:      item.BlockNumber,
		BlockTimestamp:   formatUnixTime(&blockTimestamp),
		TransactionIndex: item.TransactionIndex,
		Nonce:            item.Nonce,
		FromAddress:      formatAddress(item.FromAddress),
		ToAddress:        formatAddress(item.ToAddress),
		Value:            formatBigInt(item.Value),
		Gas:              item.Gas,
		GasPrice:         formatBigInt(item.GasPrice),
		GasUsed:          item.GasUsed,
		Input:            item.Input,
		MethodID:         item.MethodID,
		FunctionName:     item.FunctionName,
		ReceiptStatus:    string(item.ReceiptStatus),
		IsError:          item.IsError,
		CollectedAt:      formatTime(item.CollectedAt),
	}
}

func mapWalletNormalTransactions(items []research.WalletNormalTransaction) []*v1alpha1.TokenWalletNormalTransaction {
	result := make([]*v1alpha1.TokenWalletNormalTransaction, 0, len(items))
	for _, item := range items {
		result = append(result, mapWalletNormalTransaction(item))
	}
	return result
}

func wrapStoreError(action string, err error) error {
	if err == nil {
		return nil
	}
	return grpcStoreError(fmt.Errorf("%s: %w", action, err))
}
