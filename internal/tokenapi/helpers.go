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
	"github.com/useryege/athena/internal/token/collection"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/policy"
	"github.com/useryege/athena/internal/token/profile"
	"github.com/useryege/athena/internal/token/projectview"
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
	if strings.TrimSpace(value) == "" {
		return shared.Address{}, nil
	}
	return parseAddressField(name, value)
}

func parseHashField(name, value string) (shared.Hash, error) {
	raw := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(value), "0x"), "0X")
	if len(raw) != 64 {
		return shared.Hash{}, status.Errorf(codes.InvalidArgument, "%s must be a 32-byte hex hash", name)
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		return shared.Hash{}, status.Errorf(codes.InvalidArgument, "%s must be a valid hex hash", name)
	}
	return shared.BytesToHash(decoded), nil
}

func parseOptionalHashField(name, value string) (shared.Hash, error) {
	if strings.TrimSpace(value) == "" {
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
	switch strings.TrimSpace(value) {
	case string(discovery.ChainProcessingStatusRunning), string(discovery.ChainProcessingStatusStopped):
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "status must be %q or %q", discovery.ChainProcessingStatusRunning, discovery.ChainProcessingStatusStopped)
	}
}

func validateChainBlockProcessingAttemptStatus(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	switch discovery.ChainBlockProcessingAttemptStatus(value) {
	case discovery.ChainBlockProcessingAttemptStatusRunning,
		discovery.ChainBlockProcessingAttemptStatusSucceeded,
		discovery.ChainBlockProcessingAttemptStatusFailed,
		discovery.ChainBlockProcessingAttemptStatusCancelled,
		discovery.ChainBlockProcessingAttemptStatusInterrupted:
		return nil
	default:
		return status.Error(codes.InvalidArgument, "status must be running, succeeded, failed, cancelled, or interrupted")
	}
}

func normalizeChainProcessingWindow(value int64) (int64, error) {
	if value == 0 {
		return int64(24 * time.Hour / time.Second), nil
	}
	switch value {
	case int64(time.Hour / time.Second), int64(24 * time.Hour / time.Second), int64(72 * time.Hour / time.Second):
		return value, nil
	default:
		return 0, status.Error(codes.InvalidArgument, "window_seconds must be 3600, 86400, or 259200")
	}
}

func validateProjectDataCollectionType(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if _, ok := collection.ParseDataType(value); !ok {
		return status.Error(codes.InvalidArgument, "data_type must be chain_state, wallet_asset_state, simulation_result, ave, contract_code_source, or wallet_normal_transactions")
	}
	return nil
}

func validateProjectDataCollectionStatus(value string) error {
	switch collection.TaskStatus(strings.TrimSpace(value)) {
	case "", collection.TaskStatusPending, collection.TaskStatusRunning, collection.TaskStatusSucceeded, collection.TaskStatusFailed:
		return nil
	default:
		return status.Error(codes.InvalidArgument, "status must be empty, pending, running, succeeded, or failed")
	}
}

func grpcStoreError(err error) error {
	if err == nil {
		return nil
	}
	return status.Error(codes.Internal, err.Error())
}

func wrapStoreError(action string, err error) error {
	if err == nil {
		return nil
	}
	return grpcStoreError(fmt.Errorf("%s: %w", action, err))
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func optionalTimeString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatTime(*value)
}

func formatHash(value shared.Hash) string {
	if value.IsZero() {
		return ""
	}
	return value.Hex()
}

func optionalHashString(value *shared.Hash) string {
	if value == nil {
		return ""
	}
	return formatHash(*value)
}

func formatAddress(value shared.Address) string {
	if value.IsZero() {
		return ""
	}
	return common.Address(value).Hex()
}

func formatBigInt(value *big.Int) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func formatDecimal(value *collection.Decimal) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func formatUnixTime(value *uint64) string {
	if value == nil || *value == 0 {
		return ""
	}
	return time.Unix(int64(*value), 0).UTC().Format(time.RFC3339)
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

func copyUint64(value *uint64) *uint64 {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func copyBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func jsonString(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func mapChainProcessingCheckpoint(item discovery.ChainProcessingCheckpoint) *v1alpha1.TokenChainCheckpoint {
	return &v1alpha1.TokenChainCheckpoint{
		ChainID: item.ChainID, ChainName: item.ChainName, Enabled: item.Enabled,
		CursorBlockNumber: item.CursorBlockNumber, Status: string(item.Status),
		CreatedAt: formatTime(item.CreatedAt), UpdatedAt: formatTime(item.UpdatedAt),
	}
}

func mapChainProcessingCheckpoints(items []discovery.ChainProcessingCheckpoint) []*v1alpha1.TokenChainCheckpoint {
	result := make([]*v1alpha1.TokenChainCheckpoint, 0, len(items))
	for _, item := range items {
		result = append(result, mapChainProcessingCheckpoint(item))
	}
	return result
}

func mapChainBlockProcessingAttempt(item discovery.ChainBlockProcessingAttempt) *v1alpha1.TokenChainProcessingAttempt {
	return &v1alpha1.TokenChainProcessingAttempt{
		AttemptID: item.ID, ChainID: item.ChainID, BlockNumber: item.BlockNumber,
		AttemptNumber: item.AttemptNumber, BlockTime: item.BlockTime, Status: string(item.Status),
		TerminalStage: string(item.TerminalStage), ErrorMessage: item.ErrorMessage,
		CheckpointReadDurationUS: item.CheckpointReadDurationUS, DiscoveryDurationUS: item.DiscoveryDurationUS,
		ValidationDurationUS: item.ValidationDurationUS, PersistenceDurationUS: item.PersistenceDurationUS,
		TotalDurationUS: item.TotalDurationUS, CandidateCount: item.CandidateCount,
		ValidatedCount: item.ValidatedCount, RejectedCount: item.RejectedCount,
		TimingComplete: item.TimingComplete, StartedAt: formatTime(item.StartedAt),
		CompletedAt: formatTime(item.CompletedAt), CreatedAt: formatTime(item.CreatedAt), UpdatedAt: formatTime(item.UpdatedAt),
	}
}

func mapChainBlockProcessingAttempts(items []discovery.ChainBlockProcessingAttempt) []*v1alpha1.TokenChainProcessingAttempt {
	result := make([]*v1alpha1.TokenChainProcessingAttempt, 0, len(items))
	for _, item := range items {
		result = append(result, mapChainBlockProcessingAttempt(item))
	}
	return result
}

func mapChainBlockProcessingSummary(item discovery.ChainBlockProcessingSummary) *v1alpha1.TokenChainProcessingSummary {
	return &v1alpha1.TokenChainProcessingSummary{
		ChainID: item.ChainID, RangeStartBlockTime: item.RangeStartBlockTime, RangeEndBlockTime: item.RangeEndBlockTime,
		AttemptCount: item.AttemptCount, RunningCount: item.RunningCount, SucceededCount: item.SucceededCount,
		FailedCount: item.FailedCount, CancelledCount: item.CancelledCount, InterruptedCount: item.InterruptedCount,
		IncompleteSucceededCount: item.IncompleteSucceededCount, MeasuredSucceededCount: item.MeasuredSucceededCount,
		FailureRateBPS: item.FailureRateBPS, AverageDurationUS: item.AverageDurationUS,
		AverageCheckpointReadDurationUS: item.AverageCheckpointReadDurationUS,
		AverageDiscoveryDurationUS:      item.AverageDiscoveryDurationUS,
		AverageValidationDurationUS:     item.AverageValidationDurationUS,
		AveragePersistenceDurationUS:    item.AveragePersistenceDurationUS,
		FastestBlockNumber:              item.FastestBlockNumber, FastestDurationUS: item.FastestDurationUS,
		SlowestBlockNumber: item.SlowestBlockNumber, SlowestDurationUS: item.SlowestDurationUS,
	}
}

func mapChainOptions(items []discovery.Chain) []v1alpha1.TokenChain {
	result := make([]v1alpha1.TokenChain, 0, len(items))
	for _, item := range items {
		result = append(result, v1alpha1.TokenChain{ChainID: item.ID, ChainName: item.Name})
	}
	return result
}

func mapContractCode(item catalog.ContractCode) *v1alpha1.TokenContractCode {
	return &v1alpha1.TokenContractCode{
		CodeHash: item.CodeHash.Hex(), SourceCode: item.SourceCode,
		SourceCodeFetchedAt: formatTime(item.SourceCodeFetchedAt), CreatedAt: formatTime(item.CreatedAt),
		DeploymentCount: item.DeploymentCount,
	}
}

func mapContractCodes(items []catalog.ContractCode) []*v1alpha1.TokenContractCode {
	result := make([]*v1alpha1.TokenContractCode, 0, len(items))
	for _, item := range items {
		result = append(result, mapContractCode(item))
	}
	return result
}

func mapProject(item catalog.Project) *v1alpha1.TokenProject {
	return &v1alpha1.TokenProject{
		ProjectID: item.ID, ChainID: item.ChainID, Name: item.Name, Symbol: item.Symbol,
		Contract: formatAddress(item.Contract), TxSender: formatAddress(item.TxSender), TxHash: formatHash(item.TxHash),
		TxIndex: item.TxIndex, DeploymentNonce: item.DeploymentNonce, BlockNumber: item.BlockNumber, BlockTime: item.BlockTime,
		CodeHash: formatHash(item.CodeHash), CreatedAt: formatTime(item.CreatedAt), Decimals: int32(item.Decimals),
		TotalSupply: formatBigInt(item.TotalSupply), WethPair: formatAddress(item.WethPair), UsdtPair: formatAddress(item.UsdtPair),
	}
}

func mapProjectMarketSummary(item projectview.ProjectMarketSummary) *v1alpha1.TokenProjectMarketSummary {
	result := &v1alpha1.TokenProjectMarketSummary{
		LogoURL: item.LogoURL, CurrentPriceUSD: formatDecimal(item.CurrentPriceUSD),
		MarketCapUSD: formatDecimal(item.MarketCapUSD), FDVUSD: formatDecimal(item.FDVUSD), TVLUSD: formatDecimal(item.TVLUSD),
	}
	if item.Holders != nil {
		result.Holders = *item.Holders
	}
	return result
}

func mapProjectPairProfileSummary(item projectview.ProjectPairProfileSummary) *v1alpha1.TokenProjectPairProfileSummary {
	return &v1alpha1.TokenProjectPairProfileSummary{
		Kind: string(item.Kind), Address: formatAddress(item.Address), IsCreated: item.IsCreated,
		QuoteUsdtValueInt: formatBigInt(item.QuoteUSDTValueInt), ReserveUpdatedAt: item.ReserveUpdatedAt,
		PairTokenBalanceExceedsTotalSupply: item.PairTokenBalanceExceedsTotalSupply,
		LPMinimumSupplyOnly:                item.LPMinimumSupplyOnly,
		FixedFeeAddressLPShareGte90Percent: item.FixedFeeAddressLPShareGte90Percent,
	}
}

func mapProjectListItem(item projectview.ProjectListItem) *v1alpha1.TokenProjectListItem {
	result := &v1alpha1.TokenProjectListItem{
		ProjectID: item.ProjectID, ChainID: item.ChainID, Name: item.Name, Symbol: item.Symbol,
		Contract: formatAddress(item.Contract), CodeHash: formatHash(item.CodeHash), BlockNumber: item.BlockNumber,
		BlockTime: item.BlockTime, TxHash: formatHash(item.TxHash), CreatedAt: formatTime(item.CreatedAt),
		CollectionStatus: string(item.CollectionStatus), CollectionSucceededCount: item.CollectionSucceededCount,
		CollectionTerminalCount: item.CollectionTerminalCount, CollectionTotalCount: item.CollectionTotalCount,
		ProfileState: string(item.ProfileState), CompletenessStatus: string(item.CompletenessStatus),
		ProfileBuiltAt: formatTime(item.ProfileBuiltAt),
	}
	if item.Market != nil {
		result.Market = mapProjectMarketSummary(*item.Market)
	}
	if item.WrappedNativePair != nil {
		result.WrappedNativePair = mapProjectPairProfileSummary(*item.WrappedNativePair)
	}
	if item.USDTPair != nil {
		result.UsdtPair = mapProjectPairProfileSummary(*item.USDTPair)
	}
	return result
}

func mapProjectListItems(items []projectview.ProjectListItem) []*v1alpha1.TokenProjectListItem {
	result := make([]*v1alpha1.TokenProjectListItem, 0, len(items))
	for _, item := range items {
		result = append(result, mapProjectListItem(item))
	}
	return result
}

func mapCollectionResult(item collection.Result) *v1alpha1.TokenCollectionResult {
	return &v1alpha1.TokenCollectionResult{
		TaskID: item.TaskID, ProjectID: item.ProjectID, DataType: string(item.DataType), SchemaVersion: item.SchemaVersion,
		PayloadJSON: string(item.Payload), ContentHash: formatHash(item.ContentHash), BlockNumber: copyUint64(item.BlockNumber),
		CollectedAt: formatTime(item.CollectedAt),
	}
}

func mapCollectionTask(item collection.Task) *v1alpha1.TokenCollectionTask {
	return &v1alpha1.TokenCollectionTask{
		TaskID: item.ID, ProjectID: item.ProjectID, DataType: string(item.DataType), Status: string(item.Status),
		FailureCount: item.FailureCount, AvailableAt: formatTime(item.AvailableAt), ClaimGeneration: item.ClaimGeneration,
		LockedAt: formatTime(item.LockedAt), LeaseExpiresAt: formatTime(item.LeaseExpiresAt), LastError: item.LastError,
		FinishedAt: formatTime(item.FinishedAt), CreatedAt: formatTime(item.CreatedAt), UpdatedAt: formatTime(item.UpdatedAt),
	}
}

func mapCollectionTaskDetail(item collection.TaskDetail) *v1alpha1.TokenCollectionTask {
	result := mapCollectionTask(item.Task)
	if item.Result != nil {
		result.Result = mapCollectionResult(*item.Result)
	}
	return result
}

func mapCollectionTasks(items []collection.Task) []*v1alpha1.TokenCollectionTask {
	result := make([]*v1alpha1.TokenCollectionTask, 0, len(items))
	for _, item := range items {
		result = append(result, mapCollectionTask(item))
	}
	return result
}

func mapProjectProfilePairLiquidity(item profile.PairLiquidityV1) *v1alpha1.TokenProjectProfilePairLiquidity {
	return &v1alpha1.TokenProjectProfilePairLiquidity{
		TotalSupply: formatBigInt(item.TotalSupply), LockedLiquidity: formatBigInt(item.LockedLiquidity),
		FixedFeeAddressBalance: formatBigInt(item.FixedFeeAddressBalance), FixedFeeAddressShare: formatBigInt(item.FixedFeeAddressShare),
	}
}

func mapProjectProfilePairSignals(item profile.PairSignalsV1) *v1alpha1.TokenProjectProfilePairSignals {
	return &v1alpha1.TokenProjectProfilePairSignals{
		PairTokenBalanceExceedsTotalSupply: item.PairTokenBalanceExceedsTotalSupply,
		LPMinimumSupplyOnly:                item.LPMinimumSupplyOnly,
		FixedFeeAddressLPShareGte90Percent: item.FixedFeeAddressLPShareGte90Percent,
	}
}

func mapProjectProfilePairChainState(item *profile.PairChainStateProfileV1) *v1alpha1.TokenProjectProfilePairChainState {
	if item == nil {
		return nil
	}
	return &v1alpha1.TokenProjectProfilePairChainState{
		IsCreated: item.IsCreated, BaseBalance: formatBigInt(item.BaseBalance), QuoteBalance: formatBigInt(item.QuoteBalance),
		QuoteUsdtValue: formatBigInt(item.QuoteUsdtValue), QuoteUsdtValueInt: formatBigInt(item.QuoteUsdtValueInt),
		ReserveUpdatedAt: item.ReserveUpdatedAt, Liquidity: mapProjectProfilePairLiquidity(item.Liquidity),
		Signals: mapProjectProfilePairSignals(item.Signals),
	}
}

func mapProjectProfilePairMarket(item *profile.PairMarketProfileV1) *v1alpha1.TokenProjectProfilePairMarket {
	if item == nil {
		return nil
	}
	return &v1alpha1.TokenProjectProfilePairMarket{
		AMM: item.AMM, Token0Address: formatAddress(item.Token0Address), Token0Symbol: item.Token0Symbol,
		Token1Address: formatAddress(item.Token1Address), Token1Symbol: item.Token1Symbol,
		Reserve0: formatDecimal(item.Reserve0), Reserve1: formatDecimal(item.Reserve1), VolumeUSD: formatDecimal(item.VolumeUSD),
		MarketCapUSD: formatDecimal(item.MarketCapUSD), FDVUSD: formatDecimal(item.FDVUSD), IsFake: item.IsFake,
		CreatedAt: optionalTimeString(item.CreatedAt), UpdatedAt: optionalTimeString(item.UpdatedAt),
	}
}

func mapProfilePair(item *profile.PairProfileV1) *v1alpha1.TokenProjectProfilePair {
	if item == nil {
		return nil
	}
	return &v1alpha1.TokenProjectProfilePair{
		Kind: string(item.Kind), Address: formatAddress(item.Address),
		ChainState: mapProjectProfilePairChainState(item.ChainState), Market: mapProjectProfilePairMarket(item.Market),
	}
}

func mapProjectProfileAveRisk(item profile.AveRiskProfileV1) *v1alpha1.TokenProjectProfileAveRisk {
	return &v1alpha1.TokenProjectProfileAveRisk{
		RiskLevel: int32(item.RiskLevel), RiskScore: formatDecimal(item.RiskScore), RiskInfo: item.RiskInfo,
		Audited: item.Audited, Mintable: copyBool(item.Mintable), HasMintMethod: item.HasMintMethod, LiquidityPoolUnlocked: item.LiquidityPoolUnlocked,
		OwnershipNotRenounced: item.OwnershipNotRenounced, NotAudited: item.NotAudited,
		NotOpenSource: item.NotOpenSource, InBlacklist: item.InBlacklist, Honeypot: item.Honeypot,
	}
}

func mapProjectProfileMarket(item *profile.MarketProfileV1) *v1alpha1.TokenProjectProfileMarket {
	if item == nil {
		return nil
	}
	return &v1alpha1.TokenProjectProfileMarket{
		LogoURL: item.LogoURL, CurrentPriceUSD: formatDecimal(item.CurrentPriceUSD), CurrentPriceETH: formatDecimal(item.CurrentPriceETH),
		MarketCapUSD: formatDecimal(item.MarketCapUSD), FDVUSD: formatDecimal(item.FDVUSD), TVLUSD: formatDecimal(item.TVLUSD),
		MainPairTVLUSD: formatDecimal(item.MainPairTVLUSD), Holders: item.Holders, LaunchAt: optionalTimeString(item.LaunchAt),
		ProviderUpdatedAt: optionalTimeString(item.ProviderUpdatedAt), AveRisk: mapProjectProfileAveRisk(item.AveRisk),
	}
}

func mapProjectProfileSource(item *profile.ContractSourceProfileV1) *v1alpha1.TokenProjectProfileSource {
	if item == nil {
		return nil
	}
	return &v1alpha1.TokenProjectProfileSource{
		CodeHash: formatHash(item.CodeHash), VerificationStatus: string(item.VerificationStatus), ArtifactReference: item.ArtifactReference,
	}
}

func mapProjectProfileWalletSummary(item profile.WalletSummaryV1) *v1alpha1.TokenProjectProfileWalletSummary {
	result := &v1alpha1.TokenProjectProfileWalletSummary{
		WalletCount: item.WalletCount, NativeBalanceTotal: formatBigInt(item.NativeBalanceTotal),
		WrappedNativeBalanceTotal: formatBigInt(item.WrappedNativeBalanceTotal), UsdtBalanceTotal: formatBigInt(item.USDTBalanceTotal),
		TrackedAssetUsdtValueTotal: formatBigInt(item.TrackedAssetUsdtValueTotal), InitialRecipientCount: item.InitialRecipientCount,
		InitialRecipientAllocationBPS: item.InitialRecipientAllocationBPS, WalletsWithSimulationSignals: item.WalletsWithSimulationSignals,
	}
	for _, count := range item.RoleCounts {
		result.RoleCounts = append(result.RoleCounts, &v1alpha1.TokenProjectProfileWalletRoleCount{Role: count.Role, Count: count.Count})
	}
	return result
}

func mapProjectProfileWalletAssets(item *profile.WalletAssetProfileV1) *v1alpha1.TokenProjectProfileWalletAssets {
	if item == nil {
		return nil
	}
	return &v1alpha1.TokenProjectProfileWalletAssets{
		NativeBalance: formatBigInt(item.NativeBalance), WrappedNativeBalance: formatBigInt(item.WrappedNativeBalance),
		UsdtBalance: formatBigInt(item.USDTBalance), TrackedAssetUsdtValue: formatBigInt(item.TrackedAssetUsdtValue),
	}
}

func mapProjectProfileWalletSimulation(item *profile.WalletSimulationProfileV1) *v1alpha1.TokenProjectProfileWalletSimulation {
	if item == nil {
		return nil
	}
	return &v1alpha1.TokenProjectProfileWalletSimulation{
		TransferFromDeadToWalletCallSucceeded:     item.TransferFromDeadToWalletCallSucceeded,
		TransferFromZeroToWalletCallSucceeded:     item.TransferFromZeroToWalletCallSucceeded,
		TransferFromWethPairToWalletCallSucceeded: item.TransferFromWethPairToWalletCallSucceeded,
		TransferFromUsdtPairToWalletCallSucceeded: item.TransferFromUsdtPairToWalletCallSucceeded,
		TransferFromWalletToWethPairCallSucceeded: item.TransferFromWalletToWethPairCallSucceeded,
		TransferFromWalletToUsdtPairCallSucceeded: item.TransferFromWalletToUsdtPairCallSucceeded,
	}
}

func mapProjectProfileWallet(item profile.WalletProfileV1) *v1alpha1.TokenProjectProfileWallet {
	result := &v1alpha1.TokenProjectProfileWallet{
		Address: formatAddress(item.Address), Roles: append([]string(nil), item.Roles...),
		Assets: mapProjectProfileWalletAssets(item.Assets), Simulation: mapProjectProfileWalletSimulation(item.Simulation),
		TransactionSampleCapped: item.TransactionSampleCapped,
	}
	if item.InitialRecipient != nil {
		result.InitialRecipient = &v1alpha1.TokenProjectProfileInitialRecipient{
			Rank: item.InitialRecipient.Rank, RatioBPS: item.InitialRecipient.RatioBPS,
		}
	}
	return result
}

func mapProjectProfileTransactions(item *profile.TransactionSummaryV1) *v1alpha1.TokenProjectProfileTransactions {
	if item == nil {
		return nil
	}
	result := &v1alpha1.TokenProjectProfileTransactions{
		WalletCount: item.WalletCount, TransactionAssociationCount: item.TransactionAssociationCount,
		UniqueTransactionCount: item.UniqueTransactionCount, SucceededTransactionCount: item.SucceededTransactionCount,
		FailedTransactionCount: item.FailedTransactionCount, TotalInflowNativeValue: formatBigInt(item.TotalInflowNativeValue),
		TotalOutflowNativeValue: formatBigInt(item.TotalOutflowNativeValue),
	}
	for _, wallet := range item.CappedWallets {
		result.CappedWallets = append(result.CappedWallets, formatAddress(wallet))
	}
	for _, method := range item.TopMethods {
		result.TopMethods = append(result.TopMethods, &v1alpha1.TokenProjectProfileTransactionMethod{
			MethodID: method.MethodID, FunctionName: method.FunctionName, Count: method.Count,
		})
	}
	for _, counterparty := range item.TopCounterparties {
		result.TopCounterparties = append(result.TopCounterparties, &v1alpha1.TokenProjectProfileTransactionCounterparty{
			Address: formatAddress(counterparty.Address), Count: counterparty.Count,
		})
	}
	return result
}

func mapProjectProfileEvidence(item profile.CollectionEvidenceV1) *v1alpha1.TokenProjectProfileEvidence {
	return &v1alpha1.TokenProjectProfileEvidence{
		TaskID: item.TaskID, DataType: string(item.DataType), Status: string(item.Status), FailureCount: item.FailureCount,
		LastError: item.LastError, ResultSchemaVersion: item.ResultSchemaVersion,
		ResultContentHash: optionalHashString(item.ResultContentHash), BlockNumber: copyUint64(item.BlockNumber), CollectedAt: optionalTimeString(item.CollectedAt),
	}
}

func mapProjectProfile(item profile.ProjectProfile) *v1alpha1.TokenProjectProfile {
	result := &v1alpha1.TokenProjectProfile{
		ProjectID: item.ProjectID, SchemaVersion: item.SchemaVersion, CompletenessStatus: string(item.CompletenessStatus),
		Market: mapProjectProfileMarket(item.Profile.Market), ContractSource: mapProjectProfileSource(item.Profile.ContractSource),
		WrappedNativePair: mapProfilePair(item.Profile.Pairs.WrappedNative), UsdtPair: mapProfilePair(item.Profile.Pairs.USDT),
		WalletSummary: mapProjectProfileWalletSummary(item.Profile.WalletSummary),
		Transactions:  mapProjectProfileTransactions(item.Profile.Transactions), ProfileJSON: jsonString(item.Profile),
		ContentHash: formatHash(item.ContentHash), BuiltAt: formatTime(item.BuiltAt), CreatedAt: formatTime(item.CreatedAt),
	}
	for _, dataType := range item.FailedDataTypes {
		result.FailedDataTypes = append(result.FailedDataTypes, string(dataType))
	}
	for _, evidence := range item.Profile.Evidence {
		result.Evidence = append(result.Evidence, mapProjectProfileEvidence(evidence))
	}
	for _, wallet := range item.Profile.Wallets {
		result.Wallets = append(result.Wallets, mapProjectProfileWallet(wallet))
	}
	return result
}

func mapProjectDetail(item projectview.Detail) (*v1alpha1.TokenProjectDetail, error) {
	result := &v1alpha1.TokenProjectDetail{
		Project: mapProject(item.Project), TransactionCount: item.TransactionCount, GeneratedAt: formatTime(item.GeneratedAt),
	}
	if item.Profile != nil {
		result.Profile = mapProjectProfile(*item.Profile)
	}
	for _, task := range item.CollectionTasks {
		result.CollectionTasks = append(result.CollectionTasks, mapCollectionTaskDetail(task))
	}
	for _, relatedWallet := range item.RelatedWallets {
		result.RelatedWallets = append(result.RelatedWallets, &v1alpha1.TokenProjectRelatedWallet{
			ProjectID: relatedWallet.ProjectID, Wallet: formatAddress(relatedWallet.Wallet), Role: string(relatedWallet.Role),
			CreatedAt: formatTime(relatedWallet.CreatedAt),
		})
	}
	for _, recipient := range item.InitialRecipients {
		result.InitialRecipients = append(result.InitialRecipients, &v1alpha1.TokenProjectInitialRecipient{
			RecipientID: recipient.ID, ProjectID: recipient.ProjectID, Wallet: formatAddress(recipient.Wallet),
			RatioBPS: recipient.RatioBPS, RankIndex: recipient.RankIndex, SourceTxHash: formatHash(recipient.SourceTxHash),
			SourceBlockNumber: recipient.SourceBlockNumber, CreatedAt: formatTime(recipient.CreatedAt),
		})
	}
	for _, count := range item.WalletTransactionCounts {
		result.WalletTransactionCounts = append(result.WalletTransactionCounts, &v1alpha1.TokenWalletTransactionCount{
			Wallet: formatAddress(count.Wallet), TransactionCount: count.TransactionCount,
		})
	}
	return result, nil
}

func mapContractCodeBlocklistEntry(item policy.ContractCodeBlocklistEntry) *v1alpha1.TokenContractCodeBlocklistEntry {
	return &v1alpha1.TokenContractCodeBlocklistEntry{
		CodeHash: item.CodeHash.Hex(), Note: item.Note, SourceChainID: item.SourceChainID,
		SourceContract: formatAddress(item.SourceContract), CreatedAt: formatTime(item.CreatedAt),
	}
}

func mapContractCodeBlocklistEntries(items []policy.ContractCodeBlocklistEntry) []*v1alpha1.TokenContractCodeBlocklistEntry {
	result := make([]*v1alpha1.TokenContractCodeBlocklistEntry, 0, len(items))
	for _, item := range items {
		result = append(result, mapContractCodeBlocklistEntry(item))
	}
	return result
}

func mapWalletBlocklistEntry(item policy.WalletBlocklistEntry) *v1alpha1.TokenWalletBlocklistEntry {
	return &v1alpha1.TokenWalletBlocklistEntry{Wallet: formatAddress(item.Wallet), Note: item.Note, CreatedAt: formatTime(item.CreatedAt)}
}

func mapWalletBlocklistEntries(items []policy.WalletBlocklistEntry) []*v1alpha1.TokenWalletBlocklistEntry {
	result := make([]*v1alpha1.TokenWalletBlocklistEntry, 0, len(items))
	for _, item := range items {
		result = append(result, mapWalletBlocklistEntry(item))
	}
	return result
}

func mapProjectSwapAsset(item projectview.SwapAsset, tokenIndex uint8) *v1alpha1.TokenProjectSwapAsset {
	return &v1alpha1.TokenProjectSwapAsset{
		Address: formatAddress(item.Address), Symbol: item.Symbol, Decimals: int32(item.Decimals), TokenIndex: int32(tokenIndex),
	}
}

func mapProjectSwapBlock(item projectview.SwapBlockActivity) *v1alpha1.TokenProjectSwapBlock {
	return &v1alpha1.TokenProjectSwapBlock{
		SampleIndex: int32(item.SampleIndex), BlockNumber: formatUint64(item.BlockNumber), BlockTime: formatUnixTime(&item.BlockTime),
		EventCount: item.Totals.EventCount, TransactionCount: item.Totals.TransactionCount,
		TransactionOriginCount: item.Totals.TransactionOriginCount, BuyEventCount: item.Totals.BuyEventCount,
		SellEventCount: item.Totals.SellEventCount, ComplexEventCount: item.Totals.ComplexEventCount,
		PreviousBlockGap: optionalUint64String(item.PreviousBlockGap), PreviousTimeGapSeconds: optionalUint64String(item.PreviousTimeGapSeconds),
		BaseAmountInRaw: item.Totals.Flow.BaseIn, BaseAmountOutRaw: item.Totals.Flow.BaseOut,
		QuoteAmountInRaw: item.Totals.Flow.QuoteIn, QuoteAmountOutRaw: item.Totals.Flow.QuoteOut,
		BuyQuoteAmountRaw: item.Totals.BuyQuoteVolume, SellQuoteAmountRaw: item.Totals.SellQuoteVolume,
		OpenPrice: optionalStringValue(item.Price.Open), HighPrice: optionalStringValue(item.Price.High),
		LowPrice: optionalStringValue(item.Price.Low), ClosePrice: optionalStringValue(item.Price.Close), VWAP: optionalStringValue(item.Price.VWAP),
	}
}

func mapProjectSwapPairActivity(item projectview.SwapPairActivity) *v1alpha1.TokenProjectSwapPairActivity {
	result := &v1alpha1.TokenProjectSwapPairActivity{
		PairKind: string(item.Kind), PairAddress: formatAddress(item.Address), Status: string(item.Status),
		SwapBlockCount: int32(item.SwapBlockCount), TargetSwapBlockCount: int32(item.TargetSwapBlockCount),
		StartBlockNumber: formatUint64(item.StartBlockNumber), StartBlockTime: formatUnixTime(&item.StartBlockTime),
		FirstSwapBlockNumber: optionalUint64String(item.FirstSwapBlockNumber), FirstSwapBlockTime: formatUnixTime(item.FirstSwapBlockTime),
		LastSwapBlockNumber: optionalUint64String(item.LastSwapBlockNumber), LastSwapBlockTime: formatUnixTime(item.LastSwapBlockTime),
		AbsoluteExpiryBlockTime: formatUnixTime(&item.AbsoluteExpiryBlockTime), NextExpiryBlockTime: formatUnixTime(item.NextExpiryBlockTime),
		CompletedBlockNumber: optionalUint64String(item.CompletedBlockNumber), CompletedBlockTime: formatUnixTime(item.CompletedBlockTime),
		ExpiredBlockNumber: optionalUint64String(item.ExpiredBlockNumber), ExpiredBlockTime: formatUnixTime(item.ExpiredBlockTime),
		ExpiredReason: string(item.ExpiredReason), BaseAsset: mapProjectSwapAsset(item.BaseAsset, item.BaseTokenIndex),
		QuoteAsset: mapProjectSwapAsset(item.QuoteAsset, 1-item.BaseTokenIndex), EventCount: item.Totals.EventCount,
		TransactionCount: item.Totals.TransactionCount, TransactionOriginCount: item.Totals.TransactionOriginCount,
		BuyEventCount: item.Totals.BuyEventCount, SellEventCount: item.Totals.SellEventCount,
		ComplexEventCount: item.Totals.ComplexEventCount, BaseAmountInRaw: item.Totals.Flow.BaseIn,
		BaseAmountOutRaw: item.Totals.Flow.BaseOut, QuoteAmountInRaw: item.Totals.Flow.QuoteIn,
		QuoteAmountOutRaw: item.Totals.Flow.QuoteOut, BuyQuoteAmountRaw: item.Totals.BuyQuoteVolume,
		SellQuoteAmountRaw: item.Totals.SellQuoteVolume,
	}
	for _, block := range item.Blocks {
		result.Blocks = append(result.Blocks, mapProjectSwapBlock(block))
	}
	return result
}

func mapProjectSwapActivity(item projectview.SwapActivity) *v1alpha1.TokenProjectSwapActivity {
	result := &v1alpha1.TokenProjectSwapActivity{ProjectID: item.ProjectID, ChainID: item.ChainID, GeneratedAt: formatTime(item.GeneratedAt)}
	for _, pair := range item.Pairs {
		result.Pairs = append(result.Pairs, mapProjectSwapPairActivity(pair))
	}
	return result
}

func mapProjectSwapEvent(item projectview.SwapEvent) *v1alpha1.TokenProjectSwapEvent {
	return &v1alpha1.TokenProjectSwapEvent{
		TransactionHash: formatHash(item.TransactionHash), TransactionIndex: formatUint64(item.TransactionIndex),
		LogIndex: formatUint64(item.LogIndex), TxFrom: formatAddress(item.TxFrom), Sender: formatAddress(item.Sender),
		ToAddress: formatAddress(item.ToAddress), Amount0In: item.Amount0In, Amount1In: item.Amount1In,
		Amount0Out: item.Amount0Out, Amount1Out: item.Amount1Out, BaseAmountInRaw: item.Flow.BaseIn,
		BaseAmountOutRaw: item.Flow.BaseOut, QuoteAmountInRaw: item.Flow.QuoteIn, QuoteAmountOutRaw: item.Flow.QuoteOut,
		Direction: string(item.Direction), EffectivePrice: optionalStringValue(item.EffectivePrice),
	}
}

func mapProjectSwapEvents(items []projectview.SwapEvent) []*v1alpha1.TokenProjectSwapEvent {
	result := make([]*v1alpha1.TokenProjectSwapEvent, 0, len(items))
	for _, item := range items {
		result = append(result, mapProjectSwapEvent(item))
	}
	return result
}

func mapWalletNormalTransaction(item collection.WalletNormalTransaction) *v1alpha1.TokenWalletNormalTransaction {
	blockTimestamp := item.BlockTimestamp
	return &v1alpha1.TokenWalletNormalTransaction{
		Wallet: formatAddress(item.Wallet), TransactionHash: formatHash(item.TransactionHash), BlockNumber: item.BlockNumber,
		BlockTimestamp: formatUnixTime(&blockTimestamp), TransactionIndex: item.TransactionIndex, Nonce: item.Nonce,
		FromAddress: formatAddress(item.FromAddress), ToAddress: formatAddress(item.ToAddress), Value: formatBigInt(item.Value),
		Gas: item.Gas, GasPrice: formatBigInt(item.GasPrice), GasUsed: item.GasUsed, Input: item.Input,
		MethodID: item.MethodID, FunctionName: item.FunctionName, ReceiptStatus: string(item.ReceiptStatus),
		IsError: item.IsError, CollectedAt: formatTime(item.CollectedAt),
	}
}

func mapWalletNormalTransactions(items []collection.WalletNormalTransaction) []*v1alpha1.TokenWalletNormalTransaction {
	result := make([]*v1alpha1.TokenWalletNormalTransaction, 0, len(items))
	for _, item := range items {
		result = append(result, mapWalletNormalTransaction(item))
	}
	return result
}
