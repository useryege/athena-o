package evm

import (
	"context"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/useryege/athena/internal/token/chainregistry"
	"github.com/useryege/athena/internal/token/collection"
	"github.com/useryege/athena/internal/token/shared"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/abi/ERC20"
)

type ProjectStateReader struct {
	registry *chainregistry.Registry
	clients  *ChainClientRegistry
}

func NewProjectStateReader(registry *chainregistry.Registry, clients *ChainClientRegistry) *ProjectStateReader {
	return &ProjectStateReader{registry: registry, clients: clients}
}

func (reader *ProjectStateReader) ReadChainState(ctx context.Context, project collection.ProjectContext) (collection.ChainStateResultV1, uint64, error) {
	client, caller, blockNumber, err := reader.resources(ctx, project.ChainID)
	if err != nil {
		return collection.ChainStateResultV1{}, 0, err
	}
	_ = client
	states, err := caller.ListProjectStates(&bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(blockNumber)}, []common.Address{common.Address(project.Contract)})
	if err != nil {
		reader.clients.Reset(project.ChainID)
		return collection.ChainStateResultV1{}, 0, fmt.Errorf("fetch ATHENA chain state chain_id=%d project_id=%d: %w", project.ChainID, project.ID, err)
	}
	if len(states) != 1 {
		return collection.ChainStateResultV1{}, 0, fmt.Errorf("fetch ATHENA chain state returned %d items", len(states))
	}
	return chainStateResult(states[0]), blockNumber, nil
}

func (reader *ProjectStateReader) ReadWalletAssetState(ctx context.Context, project collection.ProjectContext) (collection.WalletAssetResultV1, uint64, error) {
	_, caller, blockNumber, err := reader.resources(ctx, project.ChainID)
	if err != nil {
		return collection.WalletAssetResultV1{}, 0, err
	}
	wallets := normalizedWallets(project.RelatedWallets)
	if len(wallets) == 0 {
		return collection.WalletAssetResultV1{Items: []collection.WalletAssetStateV1{}}, blockNumber, nil
	}
	items, err := caller.ListWalletAssetStates(&bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(blockNumber)}, wallets)
	if err != nil {
		reader.clients.Reset(project.ChainID)
		return collection.WalletAssetResultV1{}, 0, err
	}
	if len(items) != len(wallets) {
		return collection.WalletAssetResultV1{}, 0, fmt.Errorf("fetch ATHENA wallet asset state returned %d items for %d wallets", len(items), len(wallets))
	}
	states := make([]collection.WalletAssetStateV1, 0, len(items))
	for _, item := range items {
		if item.Wallet == (common.Address{}) {
			continue
		}
		states = append(states, collection.WalletAssetStateV1{ChainID: project.ChainID, Wallet: shared.Address(item.Wallet), WethBalance: cloneBigInt(item.AssetState.WethBalance), UsdtBalance: cloneBigInt(item.AssetState.UsdtBalance), NativeBalance: cloneBigInt(item.AssetState.NativeBalance), TrackedAssetUsdtValue: cloneBigInt(item.AssetState.TotalAssetUsdtValue)})
	}
	sort.Slice(states, func(i, j int) bool { return states[i].Wallet.Hex() < states[j].Wallet.Hex() })
	return collection.WalletAssetResultV1{Items: states}, blockNumber, nil
}

func (reader *ProjectStateReader) ReadSimulationResult(ctx context.Context, project collection.ProjectContext) (collection.SimulationResultV1, uint64, error) {
	client, caller, blockNumber, err := reader.resources(ctx, project.ChainID)
	if err != nil {
		return collection.SimulationResultV1{}, 0, err
	}
	wallets := normalizedWallets(project.RelatedWallets)
	if len(wallets) == 0 || project.WethPair.IsZero() || project.UsdtPair.IsZero() {
		return collection.SimulationResultV1{Items: []collection.WalletSimulationResultV1{}}, blockNumber, nil
	}
	queries := make([]athenacontract.AthenaWalletSimulationStateQuery, 0, len(wallets))
	for _, wallet := range wallets {
		queries = append(queries, athenacontract.AthenaWalletSimulationStateQuery{TokenContract: common.Address(project.Contract), MsgCaller: wallet})
	}
	states, err := caller.ListWalletSimulationStates(&bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(blockNumber)}, queries)
	if err != nil {
		reader.clients.Reset(project.ChainID)
		return collection.SimulationResultV1{}, 0, err
	}
	if len(states) != len(wallets) {
		return collection.SimulationResultV1{}, 0, fmt.Errorf("fetch ATHENA wallet simulation state returned %d items for %d wallets", len(states), len(wallets))
	}
	results, err := simulateProjectWallets(ctx, client.Client(), project, blockNumber, wallets, states)
	if err != nil {
		reader.clients.Reset(project.ChainID)
		return collection.SimulationResultV1{}, 0, err
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Wallet.Hex() < results[j].Wallet.Hex() })
	return collection.SimulationResultV1{Items: results}, blockNumber, nil
}

func (reader *ProjectStateReader) resources(ctx context.Context, chainID int64) (*ethclient.Client, *athenacontract.ATHENACaller, uint64, error) {
	client, err := reader.clients.Client(ctx, chainID)
	if err != nil {
		return nil, nil, 0, err
	}
	chain, ok := reader.registry.Chain(chainID)
	if !ok || !common.IsHexAddress(chain.AthenaContract) {
		return nil, nil, 0, fmt.Errorf("token chain %d ATHENA contract is invalid", chainID)
	}
	caller, err := athenacontract.NewATHENACaller(common.HexToAddress(chain.AthenaContract), client)
	if err != nil {
		return nil, nil, 0, err
	}
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		reader.clients.Reset(chainID)
		return nil, nil, 0, err
	}
	return client, caller, blockNumber, nil
}

func normalizedWallets(values []shared.Address) []common.Address {
	seen := make(map[common.Address]struct{}, len(values))
	result := make([]common.Address, 0, len(values))
	for _, value := range values {
		wallet := common.Address(value)
		if wallet == (common.Address{}) {
			continue
		}
		if _, exists := seen[wallet]; exists {
			continue
		}
		seen[wallet] = struct{}{}
		result = append(result, wallet)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Hex() < result[j].Hex() })
	return result
}

func chainStateResult(state athenacontract.AthenaProjectState) collection.ChainStateResultV1 {
	return collection.ChainStateResultV1{
		TokenContract:  shared.Address(state.TokenContract),
		ChainTimestamp: cloneBigInt(state.UpdatedAt),
		Token: collection.ChainTokenV1{
			IsValidERC20: state.Token.IsValidERC20,
			Name:         state.Token.Name,
			Symbol:       state.Token.Symbol,
			Decimals:     state.Token.Decimals,
			TotalSupply:  cloneBigInt(state.Token.TotalSupply),
			WethPair:     shared.Address(state.Token.WethPair),
			UsdtPair:     shared.Address(state.Token.UsdtPair),
		},
		WethPair: chainPair(state.WethPair, state.Token.TotalSupply),
		UsdtPair: chainPair(state.UsdtPair, state.Token.TotalSupply),
	}
}

func chainPair(pair athenacontract.AthenaPair, tokenTotalSupply *big.Int) collection.ChainPairV1 {
	return collection.ChainPairV1{
		PairContract: shared.Address(pair.PairContract),
		IsCreated:    pair.IsCreated,
		LiquidityState: collection.ChainPairLiquidityStateV1{
			TotalSupply:                    cloneBigInt(pair.LiquidityState.TotalSupply),
			LockedLiquidity:                cloneBigInt(pair.LiquidityState.LockedLiquidity),
			FeeAddressHoldLiquidityBalance: cloneBigInt(pair.LiquidityState.FeeAddressHoldLiquidityBalance),
			FeeAddressHoldLiquidityRatio:   cloneBigInt(pair.LiquidityState.FeeAddressHoldLiquidityRatio),
		},
		BaseBalance:                        cloneBigInt(pair.BaseBalance),
		QuoteBalance:                       cloneBigInt(pair.QuoteBalance),
		QuoteUsdtValue:                     cloneBigInt(pair.QuoteUsdtValue),
		QuoteUsdtValueInt:                  cloneBigInt(pair.QuoteUsdtValueInt),
		ReserveUpdatedAt:                   pair.LastSwapTimestamp,
		PairTokenBalanceExceedsTotalSupply: greaterThan(pair.BaseBalance, tokenTotalSupply),
		LPMinimumSupplyOnly:                equalTo(pair.LiquidityState.TotalSupply, big.NewInt(1000)),
		FixedFeeAddressLPShareGte90Percent: ratioAtLeast(pair.LiquidityState.FeeAddressHoldLiquidityBalance, pair.LiquidityState.TotalSupply, 90),
	}
}

func greaterThan(left, right *big.Int) bool {
	return left != nil && right != nil && left.Cmp(right) > 0
}

func equalTo(left, right *big.Int) bool { return left != nil && right != nil && left.Cmp(right) == 0 }

func ratioAtLeast(numerator, denominator *big.Int, percentage int64) bool {
	if numerator == nil || denominator == nil || denominator.Sign() <= 0 {
		return false
	}
	left := new(big.Int).Mul(new(big.Int).Set(numerator), big.NewInt(100))
	right := new(big.Int).Mul(new(big.Int).Set(denominator), big.NewInt(percentage))
	return left.Cmp(right) >= 0
}

var (
	simulationDeadAddress    = common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	simulationZeroAddress    = common.Address{}
	simulationMintBaseAmount = big.NewInt(1000000000000000000)
	simulationMintMultiplier = big.NewInt(2)
)

type simulationState struct {
	deadAllowance, zeroAllowance, wethPairAllowance, usdtPairAllowance, callerBalance *big.Int
}

type simulationEthCallResult struct {
	Data  hexutil.Bytes
	Error error
}

func simulateProjectWallets(ctx context.Context, client *rpc.Client, project collection.ProjectContext, blockNumber uint64, wallets []common.Address, states []athenacontract.AthenaSimulationState) ([]collection.WalletSimulationResultV1, error) {
	parsed, err := ERC20.ERC20MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	calls := make([]ethereum.CallMsg, 0, len(wallets)*6)
	for index, wallet := range wallets {
		walletCalls, err := buildSimulationCalls(parsed, wallet, common.Address(project.Contract), common.Address(project.WethPair), common.Address(project.UsdtPair), simulationState{deadAllowance: states[index].DeadAllowance, zeroAllowance: states[index].ZeroAllowance, wethPairAllowance: states[index].WethPairAllowance, usdtPairAllowance: states[index].UsdtPairAllowance, callerBalance: states[index].CallerBalance})
		if err != nil {
			return nil, err
		}
		calls = append(calls, walletCalls...)
	}
	callResults, err := batchEthCall(ctx, client, blockNumber, calls)
	if err != nil {
		return nil, err
	}
	results := make([]collection.WalletSimulationResultV1, 0, len(wallets))
	for index, wallet := range wallets {
		offset := index * 6
		results = append(results, collection.WalletSimulationResultV1{
			ProjectID:                                 project.ID,
			Wallet:                                    shared.Address(wallet),
			TransferFromDeadToWalletCallSucceeded:     callResults[offset].Error == nil,
			TransferFromZeroToWalletCallSucceeded:     callResults[offset+1].Error == nil,
			TransferFromWethPairToWalletCallSucceeded: callResults[offset+2].Error == nil,
			TransferFromUsdtPairToWalletCallSucceeded: callResults[offset+3].Error == nil,
			TransferFromWalletToWethPairCallSucceeded: callResults[offset+4].Error == nil,
			TransferFromWalletToUsdtPairCallSucceeded: callResults[offset+5].Error == nil,
		})
	}
	return results, nil
}

func buildSimulationCalls(parsed *abi.ABI, caller, token, wethPair, usdtPair common.Address, state simulationState) ([]ethereum.CallMsg, error) {
	dead, err := parsed.Pack("transferFrom", simulationDeadAddress, caller, calculateSimulationMintNumber(state.deadAllowance))
	if err != nil {
		return nil, err
	}
	zero, err := parsed.Pack("transferFrom", simulationZeroAddress, caller, calculateSimulationMintNumber(state.zeroAllowance))
	if err != nil {
		return nil, err
	}
	wethFrom, err := parsed.Pack("transferFrom", wethPair, caller, calculateSimulationMintNumber(state.wethPairAllowance))
	if err != nil {
		return nil, err
	}
	usdtFrom, err := parsed.Pack("transferFrom", usdtPair, caller, calculateSimulationMintNumber(state.usdtPairAllowance))
	if err != nil {
		return nil, err
	}
	weth, err := parsed.Pack("transfer", wethPair, calculateSimulationMintNumber(state.callerBalance))
	if err != nil {
		return nil, err
	}
	usdt, err := parsed.Pack("transfer", usdtPair, calculateSimulationMintNumber(state.callerBalance))
	if err != nil {
		return nil, err
	}
	return []ethereum.CallMsg{{From: caller, To: &token, Data: dead}, {From: caller, To: &token, Data: zero}, {From: caller, To: &token, Data: wethFrom}, {From: caller, To: &token, Data: usdtFrom}, {From: caller, To: &token, Data: weth}, {From: caller, To: &token, Data: usdt}}, nil
}

func calculateSimulationMintNumber(value *big.Int) *big.Int {
	result := new(big.Int)
	if value != nil {
		result.Set(value)
	}
	result.Add(result, simulationMintBaseAmount)
	return result.Mul(result, simulationMintMultiplier)
}

func batchEthCall(ctx context.Context, client *rpc.Client, blockNumber uint64, calls []ethereum.CallMsg) ([]simulationEthCallResult, error) {
	batch := make([]rpc.BatchElem, len(calls))
	results := make([]simulationEthCallResult, len(calls))
	for index := range calls {
		batch[index] = rpc.BatchElem{Method: "eth_call", Args: []any{simulationCallMsgToRPCArg(calls[index]), hexutil.EncodeUint64(blockNumber)}, Result: &results[index].Data}
	}
	if err := client.BatchCallContext(ctx, batch); err != nil {
		return nil, err
	}
	for index := range batch {
		results[index].Error = batch[index].Error
	}
	return results, nil
}

func simulationCallMsgToRPCArg(message ethereum.CallMsg) map[string]any {
	argument := map[string]any{"data": hexutil.Encode(message.Data)}
	if message.From != (common.Address{}) {
		argument["from"] = message.From.Hex()
	}
	if message.To != nil {
		argument["to"] = message.To.Hex()
	}
	if message.Gas != 0 {
		argument["gas"] = hexutil.Uint64(message.Gas)
	}
	if message.GasPrice != nil {
		argument["gasPrice"] = (*hexutil.Big)(message.GasPrice)
	}
	if message.Value != nil {
		argument["value"] = (*hexutil.Big)(message.Value)
	}
	return argument
}
