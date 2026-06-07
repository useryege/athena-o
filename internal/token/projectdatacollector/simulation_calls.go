package projectdatacollector

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	tokenstore "github.com/useryege/athena/internal/token/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/abi/ERC20"
)

var (
	simulationDeadAddress = common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	simulationZeroAddress = common.Address{}

	simulationMintBaseAmount = big.NewInt(1000000000000000000)
	simulationMintMultiplier = big.NewInt(2)
)

type simulationState struct {
	deadAllowance     *big.Int
	zeroAllowance     *big.Int
	wethPairAllowance *big.Int
	usdtPairAllowance *big.Int
	callerBalance     *big.Int
}

type simulationEthCallResult struct {
	Data  hexutil.Bytes
	Error error
}

func simulateProjectWallets(ctx context.Context, rpcClient *rpc.Client, project tokenstore.Project, wallets []common.Address, states []athenacontract.AthenaSimulationState) ([]tokenstore.ProjectSimulationResult, error) {
	parsed, err := ERC20.ERC20MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	calls := make([]ethereum.CallMsg, 0, len(wallets)*6)
	for i, wallet := range wallets {
		walletCalls, err := buildSimulationCalls(parsed, wallet, project.Contract, project.WethPair, project.UsdtPair, simulationState{
			deadAllowance:     states[i].DeadAllowance,
			zeroAllowance:     states[i].ZeroAllowance,
			wethPairAllowance: states[i].WethPairAllowance,
			usdtPairAllowance: states[i].UsdtPairAllowance,
			callerBalance:     states[i].CallerBalance,
		})
		if err != nil {
			return nil, err
		}
		calls = append(calls, walletCalls...)
	}
	callResults, err := batchEthCall(ctx, rpcClient, calls)
	if err != nil {
		return nil, err
	}
	results := make([]tokenstore.ProjectSimulationResult, 0, len(wallets))
	for i, wallet := range wallets {
		offset := i * 6
		results = append(results, tokenstore.ProjectSimulationResult{
			ProjectID:                          project.ID,
			Wallet:                             wallet,
			CanMintFromDeadViaTransferFrom:     callResults[offset].Error == nil,
			CanMintFromZeroViaTransferFrom:     callResults[offset+1].Error == nil,
			CanMintFromWethPairViaTransferFrom: callResults[offset+2].Error == nil,
			CanMintFromUsdtPairViaTransferFrom: callResults[offset+3].Error == nil,
			CanMintViaTransferToWethPair:       callResults[offset+4].Error == nil,
			CanMintViaTransferToUsdtPair:       callResults[offset+5].Error == nil,
		})
	}
	return results, nil
}

func buildSimulationCalls(parsed *abi.ABI, msgCaller common.Address, tokenAddress common.Address, wethPairContract common.Address, usdtPairContract common.Address, state simulationState) ([]ethereum.CallMsg, error) {
	deadTransferFromData, err := packSimulationTransferFromMint(parsed, simulationDeadAddress, msgCaller, state.deadAllowance)
	if err != nil {
		return nil, err
	}
	zeroTransferFromData, err := packSimulationTransferFromMint(parsed, simulationZeroAddress, msgCaller, state.zeroAllowance)
	if err != nil {
		return nil, err
	}
	wethPairTransferFromData, err := packSimulationTransferFromMint(parsed, wethPairContract, msgCaller, state.wethPairAllowance)
	if err != nil {
		return nil, err
	}
	usdtPairTransferFromData, err := packSimulationTransferFromMint(parsed, usdtPairContract, msgCaller, state.usdtPairAllowance)
	if err != nil {
		return nil, err
	}
	wethTransferData, err := packSimulationTransferMint(parsed, wethPairContract, state.callerBalance)
	if err != nil {
		return nil, err
	}
	usdtTransferData, err := packSimulationTransferMint(parsed, usdtPairContract, state.callerBalance)
	if err != nil {
		return nil, err
	}
	return []ethereum.CallMsg{
		{From: msgCaller, To: &tokenAddress, Data: deadTransferFromData},
		{From: msgCaller, To: &tokenAddress, Data: zeroTransferFromData},
		{From: msgCaller, To: &tokenAddress, Data: wethPairTransferFromData},
		{From: msgCaller, To: &tokenAddress, Data: usdtPairTransferFromData},
		{From: msgCaller, To: &tokenAddress, Data: wethTransferData},
		{From: msgCaller, To: &tokenAddress, Data: usdtTransferData},
	}, nil
}

func packSimulationTransferFromMint(parsed *abi.ABI, mintFrom common.Address, to common.Address, allowance *big.Int) ([]byte, error) {
	return parsed.Pack("transferFrom", mintFrom, to, calculateSimulationMintNumber(allowance))
}

func packSimulationTransferMint(parsed *abi.ABI, to common.Address, balance *big.Int) ([]byte, error) {
	return parsed.Pack("transfer", to, calculateSimulationMintNumber(balance))
}

func calculateSimulationMintNumber(value *big.Int) *big.Int {
	mintNumber := new(big.Int)
	if value != nil {
		mintNumber.Set(value)
	}
	mintNumber.Add(mintNumber, simulationMintBaseAmount)
	mintNumber.Mul(mintNumber, simulationMintMultiplier)
	return mintNumber
}

func batchEthCall(ctx context.Context, rpcClient *rpc.Client, calls []ethereum.CallMsg) ([]simulationEthCallResult, error) {
	batch := make([]rpc.BatchElem, len(calls))
	results := make([]simulationEthCallResult, len(calls))
	for i := range calls {
		batch[i] = rpc.BatchElem{
			Method: "eth_call",
			Args: []any{
				simulationCallMsgToRPCArg(calls[i]),
				"latest",
			},
			Result: &results[i].Data,
		}
	}
	if err := rpcClient.BatchCallContext(ctx, batch); err != nil {
		return nil, err
	}
	for i := range batch {
		results[i].Error = batch[i].Error
	}
	return results, nil
}

func simulationCallMsgToRPCArg(msg ethereum.CallMsg) map[string]any {
	arg := map[string]any{
		"data": hexutil.Encode(msg.Data),
	}
	if msg.From != (common.Address{}) {
		arg["from"] = msg.From.Hex()
	}
	if msg.To != nil {
		arg["to"] = msg.To.Hex()
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	return arg
}
