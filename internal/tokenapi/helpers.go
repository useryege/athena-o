package tokenapi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
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

func validateChainProcessingStatus(value string) error {
	value = strings.TrimSpace(value)
	switch value {
	case string(discovery.ChainProcessingStatusRunning), string(discovery.ChainProcessingStatusStopped):
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "status must be %q or %q", discovery.ChainProcessingStatusRunning, discovery.ChainProcessingStatusStopped)
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

func mapChainProcessingCheckpoint(item discovery.ChainProcessingCheckpoint) *v1alpha1.TokenChainCheckpoint {
	return &v1alpha1.TokenChainCheckpoint{
		ChainID:           item.ChainID,
		ChainName:         item.ChainName,
		Enabled:           item.Enabled,
		CursorBlockNumber: item.CursorBlockNumber,
		Status:            string(item.Status),
		CreatedAt:         formatTime(item.CreatedAt),
	}
}

func mapChainProcessingCheckpoints(items []discovery.ChainProcessingCheckpoint) []*v1alpha1.TokenChainCheckpoint {
	results := make([]*v1alpha1.TokenChainCheckpoint, 0, len(items))
	for _, item := range items {
		results = append(results, mapChainProcessingCheckpoint(item))
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
		ProjectID:       item.ID,
		ChainID:         item.ChainID,
		Name:            item.Name,
		Symbol:          item.Symbol,
		Contract:        formatAddress(item.Contract),
		TxSender:        formatAddress(item.TxSender),
		TxHash:          item.TxHash.Hex(),
		TxIndex:         item.TxIndex,
		DeploymentNonce: item.DeploymentNonce,
		BlockNumber:     item.BlockNumber,
		BlockTime:       item.BlockTime,
		CodeHash:        item.CodeHash.Hex(),
		CreatedAt:       formatTime(item.CreatedAt),
		Decimals:        int32(item.Decimals),
		TotalSupply:     formatBigInt(item.TotalSupply),
		WethPair:        formatAddress(item.WethPair),
		UsdtPair:        formatAddress(item.UsdtPair),
	}
}

func mapProjectPairRiskSummary(item projectview.ProjectPairRiskSummary) *v1alpha1.TokenProjectPairRiskSummary {
	lastSwapTimestamp := item.LastSwapTimestamp
	return &v1alpha1.TokenProjectPairRiskSummary{
		IsCreated:         item.IsCreated,
		IsRemoveLiquidity: item.IsRemoveLiquidity,
		IsMint:            item.IsMint,
		QuoteUsdtValueInt: formatBigInt(item.QuoteUSDTValueInt),
		LastSwapAt:        formatUnixTime(&lastSwapTimestamp),
	}
}

func mapProjectReportRiskSummary(item projectview.ProjectReportRiskSummary) *v1alpha1.TokenProjectReportRiskSummary {
	result := &v1alpha1.TokenProjectReportRiskSummary{}
	if item.WethPair != nil {
		result.WethPair = mapProjectPairRiskSummary(*item.WethPair)
	}
	if item.UsdtPair != nil {
		result.UsdtPair = mapProjectPairRiskSummary(*item.UsdtPair)
	}
	return result
}

func mapProjectReportEvaluationSummary(item projectview.ProjectReportEvaluationSummary) *v1alpha1.TokenProjectReportEvaluationSummary {
	return &v1alpha1.TokenProjectReportEvaluationSummary{
		Status:         string(item.Status),
		FailedAttempts: item.FailedAttempts,
		LastError:      item.LastError,
		UpdatedAt:      formatTime(item.UpdatedAt),
		Outcome:        string(item.Outcome),
		EvaluatedAt:    formatTime(item.EvaluatedAt),
	}
}

func mapProjectReportSummary(item projectview.ProjectReportSummary) *v1alpha1.TokenProjectReportSummary {
	result := &v1alpha1.TokenProjectReportSummary{
		Revision:           item.Revision,
		CompletenessStatus: item.CompletenessStatus,
		BuiltAt:            formatTime(item.BuiltAt),
	}
	if item.RiskSummary != nil {
		result.RiskSummary = mapProjectReportRiskSummary(*item.RiskSummary)
	}
	if item.Evaluation != nil {
		result.Evaluation = mapProjectReportEvaluationSummary(*item.Evaluation)
	}
	return result
}

func mapProjectListItem(item projectview.ProjectListItem) *v1alpha1.TokenProjectListItem {
	result := &v1alpha1.TokenProjectListItem{
		Project:        mapProject(item.Project),
		ResearchStatus: string(item.ResearchStatus),
		LogoURL:        item.LogoURL,
	}
	if item.CurrentReport != nil {
		result.CurrentReport = mapProjectReportSummary(*item.CurrentReport)
	}
	return result
}

func mapProjectListItems(items []projectview.ProjectListItem) []*v1alpha1.TokenProjectListItem {
	results := make([]*v1alpha1.TokenProjectListItem, 0, len(items))
	for _, item := range items {
		results = append(results, mapProjectListItem(item))
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

func mapReportingProjectPairRiskSummary(
	pairKind string,
	isCreated *bool,
	isRemoveLiquidity *bool,
	isMint *bool,
	quoteUSDTValueInt *big.Int,
	lastSwapTimestamp *uint64,
) (*v1alpha1.TokenProjectPairRiskSummary, error) {
	if isCreated == nil && isRemoveLiquidity == nil && isMint == nil && quoteUSDTValueInt == nil && lastSwapTimestamp == nil {
		return nil, nil
	}
	if isCreated == nil || isRemoveLiquidity == nil || isMint == nil || quoteUSDTValueInt == nil || lastSwapTimestamp == nil {
		return nil, fmt.Errorf("%s pair risk summary must be fully populated or absent", pairKind)
	}
	return &v1alpha1.TokenProjectPairRiskSummary{
		IsCreated:         *isCreated,
		IsRemoveLiquidity: *isRemoveLiquidity,
		IsMint:            *isMint,
		QuoteUsdtValueInt: quoteUSDTValueInt.String(),
		LastSwapAt:        formatUnixTime(lastSwapTimestamp),
	}, nil
}

func mapReportingProjectReportRiskSummary(item reporting.ReportRiskSummary) (*v1alpha1.TokenProjectReportRiskSummary, error) {
	wethPair, err := mapReportingProjectPairRiskSummary(
		"weth",
		item.WethPairIsCreated,
		item.WethPairIsRemoveLiquidity,
		item.WethPairIsMint,
		item.WethPairQuoteUsdtValueInt,
		item.WethPairLastSwapTimestamp,
	)
	if err != nil {
		return nil, err
	}
	usdtPair, err := mapReportingProjectPairRiskSummary(
		"usdt",
		item.UsdtPairIsCreated,
		item.UsdtPairIsRemoveLiquidity,
		item.UsdtPairIsMint,
		item.UsdtPairQuoteUsdtValueInt,
		item.UsdtPairLastSwapTimestamp,
	)
	if err != nil {
		return nil, err
	}
	if wethPair == nil && usdtPair == nil {
		return nil, nil
	}
	return &v1alpha1.TokenProjectReportRiskSummary{
		WethPair: wethPair,
		UsdtPair: usdtPair,
	}, nil
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
	var expiredBlockNumber, expiredBlockTime uint64
	if item.ExpiredBlockNumber != nil {
		expiredBlockNumber = *item.ExpiredBlockNumber
	}
	if item.ExpiredBlockTime != nil {
		expiredBlockTime = *item.ExpiredBlockTime
	}
	return &v1alpha1.TokenResearchState{
		ProjectID: item.ProjectID, ChainID: item.ChainID, Contract: formatAddress(item.Contract), Status: string(item.Status),
		EvidenceRevision: item.EvidenceRevision, CurrentReportRevision: item.CurrentReportRevision, CurrentSelectionOutcome: string(item.CurrentSelectionOutcome),
		LastEvaluatedRevision: item.LastEvaluatedReportRevision, LastEvaluatedAt: formatTime(item.LastEvaluatedAt),
		AttentionStartBlockNumber: item.AttentionStartBlockNumber, AttentionStartBlockTime: item.AttentionStartBlockTime,
		AttentionExpiryBlockTime: item.AttentionExpiryBlockTime, ExpiredBlockNumber: expiredBlockNumber, ExpiredBlockTime: expiredBlockTime,
		CreatedAt: formatTime(item.CreatedAt), UpdatedAt: formatTime(item.UpdatedAt),
	}
}
func mapProjectResearchStates(items []research.ProjectResearchState) []*v1alpha1.TokenResearchState {
	out := make([]*v1alpha1.TokenResearchState, 0, len(items))
	for _, item := range items {
		out = append(out, mapProjectResearchState(item))
	}
	return out
}
func mapProjectReportRevision(item reporting.ProjectReportRevision) (*v1alpha1.TokenReportRevision, error) {
	var block uint64
	if item.ObservedBlockNumber != nil {
		block = *item.ObservedBlockNumber
	}
	riskSummary, err := mapReportingProjectReportRiskSummary(item.Report.RiskSummary)
	if err != nil {
		return nil, fmt.Errorf("map report revision %d risk summary: %w", item.ID, err)
	}
	return &v1alpha1.TokenReportRevision{
		ReportRevisionID:    item.ID,
		ProjectID:           item.ProjectID,
		ChainID:             item.ChainID,
		Contract:            formatAddress(item.Contract),
		Revision:            item.Revision,
		ContentHash:         item.ContentHash.Hex(),
		CompletenessStatus:  item.CompletenessStatus,
		EvidenceJSON:        jsonString(item.Evidence),
		ReportJSON:          jsonString(item.Report),
		ObservedBlockNumber: block,
		RiskSummary:         riskSummary,
		BuiltAt:             formatTime(item.BuiltAt),
		CreatedAt:           formatTime(item.CreatedAt),
	}, nil
}

func jsonString(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}
func mapProjectReportRevisions(items []reporting.ProjectReportRevision) ([]*v1alpha1.TokenReportRevision, error) {
	out := make([]*v1alpha1.TokenReportRevision, 0, len(items))
	for _, item := range items {
		mapped, err := mapProjectReportRevision(item)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
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
		LogoURL:          item.LogoURL,
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
		ChainID:             item.ChainID,
		Wallet:              formatAddress(item.Wallet),
		WethBalance:         formatBigInt(item.WethBalance),
		UsdtBalance:         formatBigInt(item.UsdtBalance),
		NativeBalance:       formatBigInt(item.NativeBalance),
		TotalAssetUsdtValue: formatBigInt(item.TotalAssetUsdtValue),
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
		currentReport, err := mapProjectReportRevision(*item.CurrentReport)
		if err != nil {
			return nil, err
		}
		result.CurrentReport = currentReport
	}
	if item.CurrentReportEvaluation != nil {
		result.CurrentReportEvaluation = mapProjectReportEvaluationSummary(*item.CurrentReportEvaluation)
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
			RetryIntervalSecs:   int64(schedule.RetryInterval / time.Second),
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

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func formatUint64(value uint64) string {
	return strconv.FormatUint(value, 10)
}

func optionalUint64String(value *uint64) string {
	if value == nil {
		return ""
	}
	return formatUint64(*value)
}

func mapProjectSwapAsset(item projectview.SwapAsset, tokenIndex uint8) *v1alpha1.TokenProjectSwapAsset {
	return &v1alpha1.TokenProjectSwapAsset{
		Address:    formatAddress(item.Address),
		Symbol:     item.Symbol,
		Decimals:   int32(item.Decimals),
		TokenIndex: int32(tokenIndex),
	}
}

func mapProjectSwapBlock(item projectview.SwapBlockActivity) *v1alpha1.TokenProjectSwapBlock {
	return &v1alpha1.TokenProjectSwapBlock{
		SampleIndex:            int32(item.SampleIndex),
		BlockNumber:            formatUint64(item.BlockNumber),
		BlockTime:              formatUnixTime(&item.BlockTime),
		EventCount:             item.Totals.EventCount,
		TransactionCount:       item.Totals.TransactionCount,
		TransactionOriginCount: item.Totals.TransactionOriginCount,
		BuyEventCount:          item.Totals.BuyEventCount,
		SellEventCount:         item.Totals.SellEventCount,
		ComplexEventCount:      item.Totals.ComplexEventCount,
		PreviousBlockGap:       optionalUint64String(item.PreviousBlockGap),
		PreviousTimeGapSeconds: optionalUint64String(item.PreviousTimeGapSeconds),
		BaseAmountInRaw:        item.Totals.Flow.BaseIn,
		BaseAmountOutRaw:       item.Totals.Flow.BaseOut,
		QuoteAmountInRaw:       item.Totals.Flow.QuoteIn,
		QuoteAmountOutRaw:      item.Totals.Flow.QuoteOut,
		BuyQuoteAmountRaw:      item.Totals.BuyQuoteVolume,
		SellQuoteAmountRaw:     item.Totals.SellQuoteVolume,
		OpenPrice:              optionalStringValue(item.Price.Open),
		HighPrice:              optionalStringValue(item.Price.High),
		LowPrice:               optionalStringValue(item.Price.Low),
		ClosePrice:             optionalStringValue(item.Price.Close),
		VWAP:                   optionalStringValue(item.Price.VWAP),
	}
}

func mapProjectSwapPairActivity(item projectview.SwapPairActivity) *v1alpha1.TokenProjectSwapPairActivity {
	blocks := make([]*v1alpha1.TokenProjectSwapBlock, 0, len(item.Blocks))
	for _, block := range item.Blocks {
		blocks = append(blocks, mapProjectSwapBlock(block))
	}
	return &v1alpha1.TokenProjectSwapPairActivity{
		PairKind:                string(item.Kind),
		PairAddress:             formatAddress(item.Address),
		Status:                  string(item.Status),
		SwapBlockCount:          int32(item.SwapBlockCount),
		TargetSwapBlockCount:    int32(item.TargetSwapBlockCount),
		StartBlockNumber:        formatUint64(item.StartBlockNumber),
		StartBlockTime:          formatUnixTime(&item.StartBlockTime),
		FirstSwapBlockNumber:    optionalUint64String(item.FirstSwapBlockNumber),
		FirstSwapBlockTime:      formatUnixTime(item.FirstSwapBlockTime),
		LastSwapBlockNumber:     optionalUint64String(item.LastSwapBlockNumber),
		LastSwapBlockTime:       formatUnixTime(item.LastSwapBlockTime),
		AbsoluteExpiryBlockTime: formatUnixTime(&item.AbsoluteExpiryBlockTime),
		NextExpiryBlockTime:     formatUnixTime(item.NextExpiryBlockTime),
		CompletedBlockNumber:    optionalUint64String(item.CompletedBlockNumber),
		CompletedBlockTime:      formatUnixTime(item.CompletedBlockTime),
		ExpiredBlockNumber:      optionalUint64String(item.ExpiredBlockNumber),
		ExpiredBlockTime:        formatUnixTime(item.ExpiredBlockTime),
		ExpiredReason:           string(item.ExpiredReason),
		BaseAsset:               mapProjectSwapAsset(item.BaseAsset, item.BaseTokenIndex),
		QuoteAsset:              mapProjectSwapAsset(item.QuoteAsset, 1-item.BaseTokenIndex),
		EventCount:              item.Totals.EventCount,
		TransactionCount:        item.Totals.TransactionCount,
		TransactionOriginCount:  item.Totals.TransactionOriginCount,
		BuyEventCount:           item.Totals.BuyEventCount,
		SellEventCount:          item.Totals.SellEventCount,
		ComplexEventCount:       item.Totals.ComplexEventCount,
		BaseAmountInRaw:         item.Totals.Flow.BaseIn,
		BaseAmountOutRaw:        item.Totals.Flow.BaseOut,
		QuoteAmountInRaw:        item.Totals.Flow.QuoteIn,
		QuoteAmountOutRaw:       item.Totals.Flow.QuoteOut,
		BuyQuoteAmountRaw:       item.Totals.BuyQuoteVolume,
		SellQuoteAmountRaw:      item.Totals.SellQuoteVolume,
		Blocks:                  blocks,
	}
}

func mapProjectSwapActivity(item projectview.SwapActivity) *v1alpha1.TokenProjectSwapActivity {
	result := &v1alpha1.TokenProjectSwapActivity{
		ProjectID:   item.ProjectID,
		ChainID:     item.ChainID,
		GeneratedAt: formatTime(item.GeneratedAt),
	}
	for _, pair := range item.Pairs {
		result.Pairs = append(result.Pairs, mapProjectSwapPairActivity(pair))
	}
	return result
}

func mapProjectSwapEvent(item projectview.SwapEvent) *v1alpha1.TokenProjectSwapEvent {
	return &v1alpha1.TokenProjectSwapEvent{
		TransactionHash:   formatHash(item.TransactionHash),
		TransactionIndex:  formatUint64(item.TransactionIndex),
		LogIndex:          formatUint64(item.LogIndex),
		TxFrom:            formatAddress(item.TxFrom),
		Sender:            formatAddress(item.Sender),
		ToAddress:         formatAddress(item.ToAddress),
		Amount0In:         item.Amount0In,
		Amount1In:         item.Amount1In,
		Amount0Out:        item.Amount0Out,
		Amount1Out:        item.Amount1Out,
		BaseAmountInRaw:   item.Flow.BaseIn,
		BaseAmountOutRaw:  item.Flow.BaseOut,
		QuoteAmountInRaw:  item.Flow.QuoteIn,
		QuoteAmountOutRaw: item.Flow.QuoteOut,
		Direction:         string(item.Direction),
		EffectivePrice:    optionalStringValue(item.EffectivePrice),
	}
}

func mapProjectSwapEvents(items []projectview.SwapEvent) []*v1alpha1.TokenProjectSwapEvent {
	result := make([]*v1alpha1.TokenProjectSwapEvent, 0, len(items))
	for _, item := range items {
		result = append(result, mapProjectSwapEvent(item))
	}
	return result
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
