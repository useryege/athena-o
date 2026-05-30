package simulation

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/useryege/athena/internal/application/model"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/abi/ERC20"
)

var (
	deadAddress = common.HexToAddress(model.DeadAddress)
	zeroAddress = common.HexToAddress(model.ZeroAddress)

	simulateMintBaseAmount = big.NewInt(1000000000000000000)
	simulateMintMultiplier = big.NewInt(2)
)

type simulateState struct {
	deadAllowance     *big.Int
	zeroAllowance     *big.Int
	wethPairAllowance *big.Int
	usdtPairAllowance *big.Int
	callerBalance     *big.Int
}

type ethCallResult struct {
	Data  hexutil.Bytes
	Error error
}

func (c *Component) simulatePrimary(ctx context.Context, msgCaller common.Address, tokenAddress common.Address, wethPairContract common.Address, usdtPairContract common.Address, state athenacontract.AthenaSimulationState) (model.SimulateResult, error) {
	var result model.SimulateResult

	parsed, err := ERC20.ERC20MetaData.GetAbi()
	if err != nil {
		return result, err
	}
	primaryCalls, err := buildPrimarySimulateCalls(parsed, msgCaller, tokenAddress, wethPairContract, usdtPairContract, simulateState{
		deadAllowance:     state.DeadAllowance,
		zeroAllowance:     state.ZeroAllowance,
		wethPairAllowance: state.WethPairAllowance,
		usdtPairAllowance: state.UsdtPairAllowance,
		callerBalance:     state.CallerBalance,
	})
	if err != nil {
		return result, err
	}
	primaryResults, err := c.batchEthCall(ctx, primaryCalls)
	if err != nil {
		return result, err
	}
	result.CanMintFromDeadViaTransferFrom = primaryResults[0].Error == nil
	result.CanMintFromZeroViaTransferFrom = primaryResults[1].Error == nil
	result.CanMintFromWethPairViaTransferFrom = primaryResults[2].Error == nil
	result.CanMintFromUsdtPairViaTransferFrom = primaryResults[3].Error == nil
	result.CanMintViaTransferToWethPair = primaryResults[4].Error == nil
	result.CanMintViaTransferToUsdtPair = primaryResults[5].Error == nil
	return result, nil
}

func buildPrimarySimulateCalls(parsed *abi.ABI, msgCaller common.Address, tokenAddress common.Address, wethPairContract common.Address, usdtPairContract common.Address, state simulateState) ([]ethereum.CallMsg, error) {
	deadTransferFromData, err := packTransferFromMint(parsed, deadAddress, msgCaller, state.deadAllowance)
	if err != nil {
		return nil, err
	}
	zeroTransferFromData, err := packTransferFromMint(parsed, zeroAddress, msgCaller, state.zeroAllowance)
	if err != nil {
		return nil, err
	}
	wethPairTransferFromData, err := packTransferFromMint(parsed, wethPairContract, msgCaller, state.wethPairAllowance)
	if err != nil {
		return nil, err
	}
	usdtPairTransferFromData, err := packTransferFromMint(parsed, usdtPairContract, msgCaller, state.usdtPairAllowance)
	if err != nil {
		return nil, err
	}
	wethTransferData, err := packTransferMint(parsed, wethPairContract, state.callerBalance)
	if err != nil {
		return nil, err
	}
	usdtTransferData, err := packTransferMint(parsed, usdtPairContract, state.callerBalance)
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

func packTransferFromMint(parsed *abi.ABI, mintFrom common.Address, to common.Address, allowance *big.Int) ([]byte, error) {
	mintNumber := calculateMintNumber(allowance)
	return parsed.Pack("transferFrom", mintFrom, to, mintNumber)
}

func packTransferMint(parsed *abi.ABI, to common.Address, balance *big.Int) ([]byte, error) {
	mintNumber := calculateMintNumber(balance)
	return parsed.Pack("transfer", to, mintNumber)
}

func calculateMintNumber(value *big.Int) *big.Int {
	mintNumber := new(big.Int)
	if value != nil {
		mintNumber.Set(value)
	}
	mintNumber.Add(mintNumber, simulateMintBaseAmount)
	mintNumber.Mul(mintNumber, simulateMintMultiplier)
	return mintNumber
}

func (c *Component) batchEthCall(ctx context.Context, calls []ethereum.CallMsg) ([]ethCallResult, error) {
	batch := make([]rpc.BatchElem, len(calls))
	results := make([]ethCallResult, len(calls))
	for i := range calls {
		batch[i] = rpc.BatchElem{
			Method: "eth_call",
			Args: []any{
				callMsgToRPCArg(calls[i]),
				"latest",
			},
			Result: &results[i].Data,
		}
	}
	if err := c.nodeClient.BatchCallContext(ctx, batch); err != nil {
		return nil, err
	}
	for i := range batch {
		results[i].Error = batch[i].Error
	}
	return results, nil
}

func callMsgToRPCArg(msg ethereum.CallMsg) map[string]any {
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
