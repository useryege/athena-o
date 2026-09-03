package profile

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/useryege/athena/internal/token/collection"
	"github.com/useryege/athena/internal/token/shared"
)

// Build validates the six-result barrier and assembles the one immutable
// profile for a project. Failed collection tasks produce an incomplete profile;
// malformed or inconsistent successful results are build failures.
func Build(input BuildInput, builtAt time.Time) (ProjectProfile, error) {
	if builtAt.IsZero() {
		return ProjectProfile{}, fmt.Errorf("project profile requires a build time")
	}
	builtAt = builtAt.UTC()
	indexed, err := validateBuildInput(input)
	if err != nil {
		return ProjectProfile{}, err
	}

	value := ProjectProfileV1{
		SchemaVersion:   SchemaVersionV1,
		ProjectID:       input.Project.ID,
		BuiltAt:         builtAt,
		FailedDataTypes: append([]collection.DataType(nil), indexed.failedDataTypes...),
		Pairs: PairProfilesV1{
			WrappedNative: &PairProfileV1{Kind: PairKindWrappedNative, Address: input.Project.WrappedNativePair},
			USDT:          &PairProfileV1{Kind: PairKindUSDT, Address: input.Project.USDTPair},
		},
		Wallets:  []WalletProfileV1{},
		Evidence: indexed.evidence,
	}
	if len(indexed.failedDataTypes) == 0 {
		value.CompletenessStatus = CompletenessStatusComplete
	} else {
		value.CompletenessStatus = CompletenessStatusIncomplete
	}

	if result := indexed.results[collection.DataTypeChainState]; result != nil {
		if err := applyChainState(&value, input.Project, *result); err != nil {
			return ProjectProfile{}, err
		}
	}
	if result := indexed.results[collection.DataTypeAve]; result != nil {
		if err := applyAve(&value, input.Project, *result); err != nil {
			return ProjectProfile{}, err
		}
	}
	if result := indexed.results[collection.DataTypeContractCodeSource]; result != nil {
		if err := applyContractSource(&value, input.Project, *result); err != nil {
			return ProjectProfile{}, err
		}
	}
	if err := applyWallets(&value, input.Project, indexed.results); err != nil {
		return ProjectProfile{}, err
	}
	if result := indexed.results[collection.DataTypeWalletNormalTransactions]; result != nil {
		if err := applyTransactions(&value, *result); err != nil {
			return ProjectProfile{}, err
		}
	}

	canonical, err := json.Marshal(value)
	if err != nil {
		return ProjectProfile{}, fmt.Errorf("marshal canonical project profile: %w", err)
	}
	digest := sha256.Sum256(canonical)
	contentHash := shared.BytesToHash(digest[:])
	projection := buildProjection(value)

	return ProjectProfile{
		ProjectID:          input.Project.ID,
		SchemaVersion:      SchemaVersionV1,
		CompletenessStatus: value.CompletenessStatus,
		FailedDataTypes:    append([]collection.DataType(nil), value.FailedDataTypes...),
		ContentHash:        contentHash,
		Profile:            value,
		Projection:         projection,
		BuiltAt:            builtAt,
	}, nil
}

type indexedBuildInput struct {
	results         map[collection.DataType]*collection.Result
	failedDataTypes []collection.DataType
	evidence        []CollectionEvidenceV1
}

func validateBuildInput(input BuildInput) (indexedBuildInput, error) {
	if input.Project.ID <= 0 {
		return indexedBuildInput{}, fmt.Errorf("project profile requires a positive project id")
	}
	if input.Project.ChainID <= 0 {
		return indexedBuildInput{}, fmt.Errorf("project %d profile requires a positive chain id", input.Project.ID)
	}
	if input.Project.Contract.IsZero() {
		return indexedBuildInput{}, fmt.Errorf("project %d profile requires a contract", input.Project.ID)
	}

	expectedTypes := collection.AllDataTypes()
	if len(input.Tasks) != len(expectedTypes) {
		return indexedBuildInput{}, fmt.Errorf("project %d profile requires exactly %d collection tasks, got %d", input.Project.ID, len(expectedTypes), len(input.Tasks))
	}
	tasks := make(map[collection.DataType]collection.Task, len(input.Tasks))
	taskIDs := make(map[int64]struct{}, len(input.Tasks))
	for _, task := range input.Tasks {
		if task.ID <= 0 {
			return indexedBuildInput{}, fmt.Errorf("project %d collection task id must be positive", input.Project.ID)
		}
		if task.ProjectID != input.Project.ID {
			return indexedBuildInput{}, fmt.Errorf("collection task %d belongs to project %d, expected %d", task.ID, task.ProjectID, input.Project.ID)
		}
		if _, ok := collection.ParseDataType(string(task.DataType)); !ok {
			return indexedBuildInput{}, fmt.Errorf("collection task %d has unsupported data type %q", task.ID, task.DataType)
		}
		if _, exists := tasks[task.DataType]; exists {
			return indexedBuildInput{}, fmt.Errorf("project %d has duplicate %s collection tasks", input.Project.ID, task.DataType)
		}
		if _, exists := taskIDs[task.ID]; exists {
			return indexedBuildInput{}, fmt.Errorf("project %d has duplicate collection task id %d", input.Project.ID, task.ID)
		}
		if !task.Status.IsTerminal() {
			return indexedBuildInput{}, fmt.Errorf("project %d collection task %s is not terminal", input.Project.ID, task.DataType)
		}
		if task.FailureCount < 0 || task.FailureCount > 3 {
			return indexedBuildInput{}, fmt.Errorf("project %d collection task %s has invalid failure count %d", input.Project.ID, task.DataType, task.FailureCount)
		}
		if task.Status == collection.TaskStatusFailed && task.FailureCount != 3 {
			return indexedBuildInput{}, fmt.Errorf("project %d failed collection task %s must have failure count 3", input.Project.ID, task.DataType)
		}
		tasks[task.DataType] = task
		taskIDs[task.ID] = struct{}{}
	}

	results := make(map[collection.DataType]*collection.Result, len(input.Results))
	for index := range input.Results {
		result := input.Results[index]
		task, exists := tasks[result.DataType]
		if !exists {
			return indexedBuildInput{}, fmt.Errorf("project %d has a result for unknown collection type %q", input.Project.ID, result.DataType)
		}
		if task.Status != collection.TaskStatusSucceeded {
			return indexedBuildInput{}, fmt.Errorf("project %d failed collection task %s must not have a result", input.Project.ID, result.DataType)
		}
		if result.TaskID != task.ID || result.ProjectID != input.Project.ID {
			return indexedBuildInput{}, fmt.Errorf("project %d collection result %s does not match its task", input.Project.ID, result.DataType)
		}
		if result.SchemaVersion != collection.ResultSchemaVersionV1 {
			return indexedBuildInput{}, fmt.Errorf("project %d collection result %s has unsupported schema version %d", input.Project.ID, result.DataType, result.SchemaVersion)
		}
		if _, exists := results[result.DataType]; exists {
			return indexedBuildInput{}, fmt.Errorf("project %d has duplicate %s collection results", input.Project.ID, result.DataType)
		}
		canonical, contentHash, err := collection.NormalizeResult(result.SchemaVersion, result.Payload)
		if err != nil {
			return indexedBuildInput{}, fmt.Errorf("normalize project %d collection result %s: %w", input.Project.ID, result.DataType, err)
		}
		if contentHash != result.ContentHash {
			return indexedBuildInput{}, fmt.Errorf("project %d collection result %s content hash mismatch", input.Project.ID, result.DataType)
		}
		result.Payload = canonical
		result.BlockNumber = cloneUint64(result.BlockNumber)
		results[result.DataType] = &result
	}

	indexed := indexedBuildInput{
		results:  results,
		evidence: make([]CollectionEvidenceV1, 0, len(expectedTypes)),
	}
	for _, dataType := range expectedTypes {
		task, exists := tasks[dataType]
		if !exists {
			return indexedBuildInput{}, fmt.Errorf("project %d is missing %s collection task", input.Project.ID, dataType)
		}
		evidence := CollectionEvidenceV1{
			TaskID: task.ID, DataType: dataType, Status: task.Status,
			FailureCount: task.FailureCount, LastError: strings.TrimSpace(task.LastError),
		}
		switch task.Status {
		case collection.TaskStatusSucceeded:
			result := results[dataType]
			if result == nil {
				return indexedBuildInput{}, fmt.Errorf("project %d succeeded collection task %s is missing its result", input.Project.ID, dataType)
			}
			hash := result.ContentHash
			collectedAt := result.CollectedAt.UTC()
			evidence.ResultSchemaVersion = result.SchemaVersion
			evidence.ResultContentHash = &hash
			evidence.BlockNumber = cloneUint64(result.BlockNumber)
			evidence.CollectedAt = &collectedAt
		case collection.TaskStatusFailed:
			indexed.failedDataTypes = append(indexed.failedDataTypes, dataType)
		default:
			return indexedBuildInput{}, fmt.Errorf("project %d collection task %s has invalid terminal status %q", input.Project.ID, dataType, task.Status)
		}
		indexed.evidence = append(indexed.evidence, evidence)
	}
	return indexed, nil
}

func applyChainState(target *ProjectProfileV1, project ProjectSnapshot, result collection.Result) error {
	state, err := decodeResult[collection.ChainStateResultV1](result)
	if err != nil {
		return err
	}
	if state.TokenContract != project.Contract {
		return fmt.Errorf("project %d chain state token contract does not match project", project.ID)
	}
	if state.Token.WethPair != project.WrappedNativePair || state.WethPair.PairContract != project.WrappedNativePair {
		return fmt.Errorf("project %d chain state wrapped-native pair does not match project", project.ID)
	}
	if state.Token.UsdtPair != project.USDTPair || state.UsdtPair.PairContract != project.USDTPair {
		return fmt.Errorf("project %d chain state USDT pair does not match project", project.ID)
	}
	target.Pairs.WrappedNative.ChainState = mapPairChainState(state.WethPair)
	target.Pairs.USDT.ChainState = mapPairChainState(state.UsdtPair)
	return nil
}

func mapPairChainState(source collection.ChainPairV1) *PairChainStateProfileV1 {
	return &PairChainStateProfileV1{
		IsCreated:         source.IsCreated,
		BaseBalance:       cloneBigInt(source.BaseBalance),
		QuoteBalance:      cloneBigInt(source.QuoteBalance),
		QuoteUsdtValue:    cloneBigInt(source.QuoteUsdtValue),
		QuoteUsdtValueInt: cloneBigInt(source.QuoteUsdtValueInt),
		ReserveUpdatedAt:  uint64(source.ReserveUpdatedAt),
		Liquidity: PairLiquidityV1{
			TotalSupply:            cloneBigInt(source.LiquidityState.TotalSupply),
			LockedLiquidity:        cloneBigInt(source.LiquidityState.LockedLiquidity),
			FixedFeeAddressBalance: cloneBigInt(source.LiquidityState.FeeAddressHoldLiquidityBalance),
			FixedFeeAddressShare:   cloneBigInt(source.LiquidityState.FeeAddressHoldLiquidityRatio),
		},
		Signals: PairSignalsV1{
			PairTokenBalanceExceedsTotalSupply: source.PairTokenBalanceExceedsTotalSupply,
			LPMinimumSupplyOnly:                source.LPMinimumSupplyOnly,
			FixedFeeAddressLPShareGte90Percent: source.FixedFeeAddressLPShareGte90Percent,
		},
	}
}

func applyAve(target *ProjectProfileV1, project ProjectSnapshot, result collection.Result) error {
	value, err := decodeResult[collection.AveResultV1](result)
	if err != nil {
		return err
	}
	if value.ChainID != project.ChainID || value.Token.Address != project.Contract {
		return fmt.Errorf("project %d Ave result identity does not match project", project.ID)
	}
	target.Market = &MarketProfileV1{
		LogoURL:           strings.TrimSpace(value.Token.LogoURL),
		CurrentPriceUSD:   cloneDecimal(value.Token.CurrentPriceUSD),
		CurrentPriceETH:   cloneDecimal(value.Token.CurrentPriceETH),
		MarketCapUSD:      cloneDecimal(value.Token.MarketCap),
		FDVUSD:            cloneDecimal(value.Token.FDV),
		TVLUSD:            cloneDecimal(value.Token.TVL),
		MainPairTVLUSD:    cloneDecimal(value.Token.MainPairTVL),
		Holders:           int64(value.Token.Holders),
		LaunchAt:          utcTimePtr(value.Token.LaunchAt),
		ProviderUpdatedAt: utcTimePtr(value.Token.UpdatedAt),
		AveRisk: AveRiskProfileV1{
			RiskLevel:             value.AveRisk.RiskLevel,
			RiskScore:             cloneDecimal(value.AveRisk.RiskScore),
			RiskInfo:              strings.TrimSpace(value.AveRisk.RiskInfo),
			Audited:               value.AveRisk.IsAudited,
			Mintable:              cloneBool(value.AveRisk.IsMintable),
			HasMintMethod:         value.AveRisk.HasMintMethod,
			LiquidityPoolUnlocked: value.AveRisk.IsLPNotLocked,
			OwnershipNotRenounced: value.AveRisk.HasNotRenounced,
			NotAudited:            value.AveRisk.HasNotAudited,
			NotOpenSource:         value.AveRisk.HasNotOpenSource,
			InBlacklist:           value.AveRisk.IsInBlacklist,
			Honeypot:              value.AveRisk.IsHoneypot,
		},
	}
	if value.Token.Holders < 0 {
		return fmt.Errorf("project %d Ave result has negative holder count", project.ID)
	}

	markets := make(map[shared.Address]PairMarketProfileV1, len(value.Pairs))
	for _, pair := range value.Pairs {
		if pair.ChainID != project.ChainID {
			return fmt.Errorf("project %d Ave pair %s has mismatched chain", project.ID, pair.Pair.Hex())
		}
		if pair.Pair != project.WrappedNativePair && pair.Pair != project.USDTPair {
			return fmt.Errorf("project %d Ave result contains non-canonical pair %s", project.ID, pair.Pair.Hex())
		}
		if _, exists := markets[pair.Pair]; exists {
			return fmt.Errorf("project %d Ave result contains duplicate pair %s", project.ID, pair.Pair.Hex())
		}
		markets[pair.Pair] = PairMarketProfileV1{
			AMM: strings.TrimSpace(pair.AMM), Token0Address: pair.Token0Address,
			Token0Symbol: strings.TrimSpace(pair.Token0Symbol), Token1Address: pair.Token1Address,
			Token1Symbol: strings.TrimSpace(pair.Token1Symbol), Reserve0: cloneDecimal(pair.Reserve0),
			Reserve1: cloneDecimal(pair.Reserve1), VolumeUSD: cloneDecimal(pair.VolumeUSD),
			MarketCapUSD: cloneDecimal(pair.MarketCap), FDVUSD: cloneDecimal(pair.FDV), IsFake: pair.IsFake,
			CreatedAt: utcTimePtr(pair.CreatedAt), UpdatedAt: utcTimePtr(pair.UpdatedAt),
		}
	}
	if market, ok := markets[project.WrappedNativePair]; ok {
		copy := market
		target.Pairs.WrappedNative.Market = &copy
	}
	if market, ok := markets[project.USDTPair]; ok {
		copy := market
		target.Pairs.USDT.Market = &copy
	}
	return nil
}

func applyContractSource(target *ProjectProfileV1, project ProjectSnapshot, result collection.Result) error {
	value, err := decodeResult[collection.ContractSourceResultV1](result)
	if err != nil {
		return err
	}
	if value.CodeHash != project.CodeHash {
		return fmt.Errorf("project %d contract source code hash does not match project", project.ID)
	}
	status := SourceVerificationStatus(value.VerificationStatus)
	switch status {
	case SourceVerificationStatusVerified, SourceVerificationStatusUnverified:
	default:
		return fmt.Errorf("project %d contract source has invalid verification status %q", project.ID, value.VerificationStatus)
	}
	target.ContractSource = &ContractSourceProfileV1{
		CodeHash: project.CodeHash, VerificationStatus: status,
		ArtifactReference: strings.TrimSpace(value.ArtifactReference),
	}
	return nil
}

type walletBuildState struct {
	profile WalletProfileV1
	roles   map[string]struct{}
}

func applyWallets(target *ProjectProfileV1, project ProjectSnapshot, results map[collection.DataType]*collection.Result) error {
	wallets := make(map[shared.Address]*walletBuildState)
	for _, related := range project.RelatedWallets {
		role := strings.TrimSpace(related.Role)
		if role == "" {
			return fmt.Errorf("project %d contains an empty related-wallet role", project.ID)
		}
		state := wallets[related.Address]
		if state == nil {
			state = &walletBuildState{profile: WalletProfileV1{Address: related.Address}, roles: make(map[string]struct{})}
			wallets[related.Address] = state
		}
		state.roles[role] = struct{}{}
	}
	seenRecipients := make(map[shared.Address]struct{}, len(project.InitialRecipients))
	for _, recipient := range project.InitialRecipients {
		state := wallets[recipient.Address]
		if state == nil {
			return fmt.Errorf("project %d initial recipient %s is not a related wallet", project.ID, recipient.Address.Hex())
		}
		if _, exists := seenRecipients[recipient.Address]; exists {
			return fmt.Errorf("project %d contains duplicate initial recipient %s", project.ID, recipient.Address.Hex())
		}
		seenRecipients[recipient.Address] = struct{}{}
		state.profile.InitialRecipient = &InitialRecipientProfileV1{Rank: recipient.Rank, RatioBPS: recipient.RatioBPS}
	}

	if result := results[collection.DataTypeWalletAssetState]; result != nil {
		value, err := decodeResult[collection.WalletAssetResultV1](*result)
		if err != nil {
			return err
		}
		seen := make(map[shared.Address]struct{}, len(value.Items))
		for _, item := range value.Items {
			state := wallets[item.Wallet]
			if state == nil || item.ChainID != project.ChainID {
				return fmt.Errorf("project %d wallet asset result contains an unknown wallet or chain", project.ID)
			}
			if _, exists := seen[item.Wallet]; exists {
				return fmt.Errorf("project %d wallet asset result contains duplicate wallet %s", project.ID, item.Wallet.Hex())
			}
			seen[item.Wallet] = struct{}{}
			state.profile.Assets = &WalletAssetProfileV1{
				NativeBalance: cloneBigInt(item.NativeBalance), WrappedNativeBalance: cloneBigInt(item.WethBalance),
				USDTBalance: cloneBigInt(item.UsdtBalance), TrackedAssetUsdtValue: cloneBigInt(item.TrackedAssetUsdtValue),
			}
		}
		if len(seen) != len(wallets) {
			return fmt.Errorf("project %d wallet asset result covers %d of %d related wallets", project.ID, len(seen), len(wallets))
		}
	}

	if result := results[collection.DataTypeSimulationResult]; result != nil {
		value, err := decodeResult[collection.SimulationResultV1](*result)
		if err != nil {
			return err
		}
		seen := make(map[shared.Address]struct{}, len(value.Items))
		for _, item := range value.Items {
			state := wallets[item.Wallet]
			if state == nil || item.ProjectID != project.ID {
				return fmt.Errorf("project %d simulation result contains an unknown wallet or project", project.ID)
			}
			if _, exists := seen[item.Wallet]; exists {
				return fmt.Errorf("project %d simulation result contains duplicate wallet %s", project.ID, item.Wallet.Hex())
			}
			seen[item.Wallet] = struct{}{}
			state.profile.Simulation = &WalletSimulationProfileV1{
				TransferFromDeadToWalletCallSucceeded:     item.TransferFromDeadToWalletCallSucceeded,
				TransferFromZeroToWalletCallSucceeded:     item.TransferFromZeroToWalletCallSucceeded,
				TransferFromWethPairToWalletCallSucceeded: item.TransferFromWethPairToWalletCallSucceeded,
				TransferFromUsdtPairToWalletCallSucceeded: item.TransferFromUsdtPairToWalletCallSucceeded,
				TransferFromWalletToWethPairCallSucceeded: item.TransferFromWalletToWethPairCallSucceeded,
				TransferFromWalletToUsdtPairCallSucceeded: item.TransferFromWalletToUsdtPairCallSucceeded,
			}
		}
		if len(seen) != len(wallets) {
			return fmt.Errorf("project %d simulation result covers %d of %d related wallets", project.ID, len(seen), len(wallets))
		}
	}

	if result := results[collection.DataTypeWalletNormalTransactions]; result != nil {
		value, err := decodeResult[collection.WalletNormalTransactionsResultV1](*result)
		if err != nil {
			return err
		}
		for _, wallet := range value.CappedWallets {
			state := wallets[wallet]
			if state == nil {
				return fmt.Errorf("project %d transaction result contains unknown capped wallet %s", project.ID, wallet.Hex())
			}
			state.profile.TransactionSampleCapped = true
		}
	}

	addresses := make([]shared.Address, 0, len(wallets))
	for address := range wallets {
		addresses = append(addresses, address)
	}
	sort.Slice(addresses, func(i, j int) bool { return bytes.Compare(addresses[i][:], addresses[j][:]) < 0 })
	roleCounts := make(map[string]int32)
	assetsAvailable := results[collection.DataTypeWalletAssetState] != nil
	var nativeTotal, wrappedTotal, usdtTotal, trackedTotal *big.Int
	if assetsAvailable {
		nativeTotal, wrappedTotal, usdtTotal, trackedTotal = new(big.Int), new(big.Int), new(big.Int), new(big.Int)
	}
	var allocationBPS uint64
	var initialCount, signalCount int32
	for _, address := range addresses {
		state := wallets[address]
		state.profile.Roles = make([]string, 0, len(state.roles))
		for role := range state.roles {
			state.profile.Roles = append(state.profile.Roles, role)
			roleCounts[role]++
		}
		sort.Strings(state.profile.Roles)
		if state.profile.InitialRecipient != nil {
			if ^uint64(0)-allocationBPS < state.profile.InitialRecipient.RatioBPS {
				return fmt.Errorf("project %d initial-recipient allocation overflows", project.ID)
			}
			allocationBPS += state.profile.InitialRecipient.RatioBPS
			initialCount++
		}
		if state.profile.Assets != nil {
			addBigInt(nativeTotal, state.profile.Assets.NativeBalance)
			addBigInt(wrappedTotal, state.profile.Assets.WrappedNativeBalance)
			addBigInt(usdtTotal, state.profile.Assets.USDTBalance)
			addBigInt(trackedTotal, state.profile.Assets.TrackedAssetUsdtValue)
		}
		if state.profile.Simulation != nil && state.profile.Simulation.HasSignal() {
			signalCount++
		}
		target.Wallets = append(target.Wallets, state.profile)
	}
	roles := make([]string, 0, len(roleCounts))
	for role := range roleCounts {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	roleItems := make([]WalletRoleCountV1, 0, len(roles))
	for _, role := range roles {
		roleItems = append(roleItems, WalletRoleCountV1{Role: role, Count: roleCounts[role]})
	}
	target.WalletSummary = WalletSummaryV1{
		WalletCount: int32(len(wallets)), RoleCounts: roleItems,
		NativeBalanceTotal: nativeTotal, WrappedNativeBalanceTotal: wrappedTotal,
		USDTBalanceTotal: usdtTotal, TrackedAssetUsdtValueTotal: trackedTotal,
		InitialRecipientCount: initialCount, InitialRecipientAllocationBPS: allocationBPS,
		WalletsWithSimulationSignals: signalCount,
	}
	return nil
}

func applyTransactions(target *ProjectProfileV1, result collection.Result) error {
	value, err := decodeResult[collection.WalletNormalTransactionsResultV1](result)
	if err != nil {
		return err
	}
	if value.WalletCount < 0 || value.TransactionAssociationCount < 0 || value.UniqueTransactionCount < 0 ||
		value.SucceededTransactionCount < 0 || value.FailedTransactionCount < 0 {
		return fmt.Errorf("project %d transaction result contains a negative count", result.ProjectID)
	}
	if value.WalletCount != int(target.WalletSummary.WalletCount) {
		return fmt.Errorf("project %d transaction result covers %d of %d related wallets", result.ProjectID, value.WalletCount, target.WalletSummary.WalletCount)
	}
	methods := make([]TransactionMethodCountV1, 0, len(value.TopMethods))
	for _, method := range value.TopMethods {
		if method.Count < 0 {
			return fmt.Errorf("project %d transaction result contains a negative method count", result.ProjectID)
		}
		methods = append(methods, TransactionMethodCountV1{
			MethodID: strings.TrimSpace(method.MethodID), FunctionName: strings.TrimSpace(method.FunctionName), Count: int64(method.Count),
		})
	}
	sort.Slice(methods, func(i, j int) bool {
		if methods[i].Count != methods[j].Count {
			return methods[i].Count > methods[j].Count
		}
		if methods[i].MethodID != methods[j].MethodID {
			return methods[i].MethodID < methods[j].MethodID
		}
		return methods[i].FunctionName < methods[j].FunctionName
	})
	counterparties := make([]TransactionCounterpartyV1, 0, len(value.TopCounterparties))
	for _, counterparty := range value.TopCounterparties {
		if counterparty.Count < 0 {
			return fmt.Errorf("project %d transaction result contains a negative counterparty count", result.ProjectID)
		}
		counterparties = append(counterparties, TransactionCounterpartyV1{Address: counterparty.Address, Count: int64(counterparty.Count)})
	}
	sort.Slice(counterparties, func(i, j int) bool {
		if counterparties[i].Count != counterparties[j].Count {
			return counterparties[i].Count > counterparties[j].Count
		}
		return bytes.Compare(counterparties[i].Address[:], counterparties[j].Address[:]) < 0
	})
	capped := append(make([]shared.Address, 0, len(value.CappedWallets)), value.CappedWallets...)
	sort.Slice(capped, func(i, j int) bool { return bytes.Compare(capped[i][:], capped[j][:]) < 0 })
	capped = deduplicateAddresses(capped)
	target.Transactions = &TransactionSummaryV1{
		WalletCount: int32(value.WalletCount), TransactionAssociationCount: int64(value.TransactionAssociationCount),
		UniqueTransactionCount: int64(value.UniqueTransactionCount), SucceededTransactionCount: int64(value.SucceededTransactionCount),
		FailedTransactionCount: int64(value.FailedTransactionCount), TotalInflowNativeValue: cloneBigInt(value.TotalInflowNativeValue),
		TotalOutflowNativeValue: cloneBigInt(value.TotalOutflowNativeValue), TopMethods: methods,
		TopCounterparties: counterparties, CappedWallets: capped,
	}
	return nil
}

func buildProjection(value ProjectProfileV1) Projection {
	projection := Projection{}
	if value.Market != nil {
		projection.LogoURL = value.Market.LogoURL
		projection.CurrentPriceUSD = cloneDecimal(value.Market.CurrentPriceUSD)
		projection.MarketCapUSD = cloneDecimal(value.Market.MarketCapUSD)
		projection.FDVUSD = cloneDecimal(value.Market.FDVUSD)
		projection.TVLUSD = cloneDecimal(value.Market.TVLUSD)
		holders := value.Market.Holders
		projection.Holders = &holders
	}
	if value.ContractSource != nil {
		status := value.ContractSource.VerificationStatus
		projection.ContractSourceStatus = &status
	}
	projection.WrappedNativePair = pairProjection(value.Pairs.WrappedNative)
	projection.USDTPair = pairProjection(value.Pairs.USDT)
	return projection
}

func pairProjection(value *PairProfileV1) *PairProjection {
	if value == nil || value.ChainState == nil {
		return nil
	}
	state := value.ChainState
	return &PairProjection{
		IsCreated: state.IsCreated, PairTokenBalanceExceedsTotalSupply: state.Signals.PairTokenBalanceExceedsTotalSupply,
		LPMinimumSupplyOnly:                state.Signals.LPMinimumSupplyOnly,
		FixedFeeAddressLPShareGte90Percent: state.Signals.FixedFeeAddressLPShareGte90Percent,
		QuoteUsdtValueInt:                  cloneBigInt(state.QuoteUsdtValueInt), ReserveUpdatedAt: state.ReserveUpdatedAt,
	}
}

func decodeResult[T any](result collection.Result) (T, error) {
	var value T
	if err := json.Unmarshal(result.Payload, &value); err != nil {
		return value, fmt.Errorf("decode project %d %s result schema version %d: %w", result.ProjectID, result.DataType, result.SchemaVersion, err)
	}
	return value, nil
}

func cloneBigInt(value *big.Int) *big.Int {
	if value == nil {
		return nil
	}
	return new(big.Int).Set(value)
}

func addBigInt(total, value *big.Int) {
	if total != nil && value != nil {
		total.Add(total, value)
	}
}

func cloneDecimal(value *collection.Decimal) *collection.Decimal {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneUint64(value *uint64) *uint64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func utcTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func deduplicateAddresses(values []shared.Address) []shared.Address {
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}
